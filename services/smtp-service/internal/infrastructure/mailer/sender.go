// Package mailer отправляет письма через SMTP.
//
// За интерфейсом email.Sender: переезд на SES/SendGrid не затрагивает ни
// очередь, ни воркеров.
package mailer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

// Config — параметры подключения к SMTP-серверу.
type Config struct {
	Host     string
	Port     int
	UseSSL   bool
	Username string
	Password string

	// From — адрес в заголовке письма, может отличаться от Username.
	From     string
	FromName string

	// AllowInsecure разрешает соединение без шифрования. По умолчанию
	// требуется TLS: письма несут персональные данные, и открытый SMTP в
	// продакшене недопустим. Включается только для локального релея без TLS
	// (MailHog, тесты).
	AllowInsecure bool

	// Timeout ограничивает и установку соединения, и SMTP-диалог.
	Timeout time.Duration

	// IdleTimeout — после какого простоя соединение считается протухшим и
	// переоткрывается. Серверы закрывают неактивные сессии молча, и попытка
	// писать в такое соединение проваливается уже после отправки команды.
	IdleTimeout time.Duration
}

const (
	defaultTimeout     = 30 * time.Second
	defaultIdleTimeout = 60 * time.Second
)

// smtpSender отправляет письма, удерживая соединение между вызовами.
//
// DialAndSend на каждое письмо стоит полного круга TCP + TLS + AUTH — сотни
// миллисекунд на письмо и прямая дорога к «421 too many connections».
// Соединение открывается лениво при первой отправке и переоткрывается, если
// протухло или разорвано.
//
// Экземпляр рассчитан на одного воркера: клиент go-mail сериализует отправку
// внутренним мьютексом, поэтому общий sender превратил бы пул воркеров в
// очередь из одного. Каждый воркер держит свой.
type smtpSender struct {
	cfg    Config
	logger *zap.Logger

	mu       sync.Mutex
	client   *mail.Client
	lastUsed time.Time
}

// NewSender собирает отправителя по SMTP.
func NewSender(cfg Config, logger *zap.Logger) (email.Sender, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.From == "" {
		cfg.From = cfg.Username
	}

	if cfg.Host == "" {
		return nil, errors.New("smtp host is empty")
	}
	if cfg.From == "" {
		return nil, errors.New("smtp from address is empty")
	}

	return &smtpSender{
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Send отправляет письмо, переиспользуя открытое соединение.
//
// Ошибки классифицированы: email.ErrTransient — письмо вернётся в очередь,
// email.ErrPermanent — уйдёт в failed без повторов.
func (s *smtpSender) Send(ctx context.Context, msg email.OutgoingMessage) error {
	m, err := s.buildMessage(msg)
	if err != nil {
		// Письмо не собралось — повтор не поможет.
		return fmt.Errorf("%w: %v", email.ErrPermanent, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.send(ctx, m); err != nil {
		// Соединение могло разорваться на середине диалога. Пробуем ещё раз
		// на свежем: сервер закрывает неактивные сессии молча, и узнать об
		// этом можно только по неудачной команде.
		if errors.Is(err, email.ErrTransient) {
			s.closeLocked()

			if retryErr := s.send(ctx, m); retryErr != nil {
				return retryErr
			}

			s.logger.Debug("email sent after reconnect")
			return nil
		}
		return err
	}

	return nil
}

// send выполняет одну попытку: при необходимости открывает соединение.
// Вызывается под s.mu.
func (s *smtpSender) send(ctx context.Context, m *mail.Msg) error {
	if err := s.ensureConnLocked(ctx); err != nil {
		return err
	}

	if err := s.client.Send(m); err != nil {
		return classify(err)
	}

	s.lastUsed = time.Now()
	return nil
}

// ensureConnLocked открывает соединение, если его нет или оно простаивало
// дольше IdleTimeout. Вызывается под s.mu.
func (s *smtpSender) ensureConnLocked(ctx context.Context) error {
	if s.client != nil && time.Since(s.lastUsed) > s.cfg.IdleTimeout {
		s.closeLocked()
	}

	if s.client != nil {
		return nil
	}

	client, err := s.newClient()
	if err != nil {
		// Клиент не создаётся из-за конфигурации, а не из-за сети.
		return fmt.Errorf("%w: create smtp client: %v", email.ErrPermanent, err)
	}

	dialCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	if err := client.DialWithContext(dialCtx); err != nil {
		return classify(err)
	}

	s.client = client
	s.lastUsed = time.Now()

	return nil
}

func (s *smtpSender) newClient() (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithPort(s.cfg.Port),
		mail.WithTimeout(s.cfg.Timeout),
	}

	switch {
	case s.cfg.UseSSL:
		// Implicit TLS: шифрование с первого байта (порт 465).
		opts = append(opts, mail.WithSSL())
	case s.cfg.AllowInsecure:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}

	// Пустой пароль означает локальный релей без авторизации (dev, MailHog):
	// навязывать ему AUTH нельзя, диалог оборвётся.
	if s.cfg.Username != "" && s.cfg.Password != "" {
		opts = append(opts,
			mail.WithUsername(s.cfg.Username),
			mail.WithPassword(s.cfg.Password),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
		)
	}

	return mail.NewClient(s.cfg.Host, opts...)
}

func (s *smtpSender) buildMessage(msg email.OutgoingMessage) (*mail.Msg, error) {
	m := mail.NewMsg()

	// Отображаемое имя вместо голого адреса: письмо от «RealTimeMap» вызывает
	// меньше подозрений у получателя и у спам-фильтров, чем от строки вида
	// persproj@yandex.ru.
	if s.cfg.FromName != "" {
		if err := m.FromFormat(s.cfg.FromName, s.cfg.From); err != nil {
			return nil, fmt.Errorf("set from: %w", err)
		}
	} else if err := m.From(s.cfg.From); err != nil {
		return nil, fmt.Errorf("set from: %w", err)
	}

	if err := m.To(msg.To); err != nil {
		return nil, fmt.Errorf("set to: %w", err)
	}

	m.Subject(msg.Subject)
	m.SetBodyString(mail.TypeTextHTML, msg.HTML)

	return m, nil
}

// Close закрывает соединение. Вызывается при остановке воркера.
func (s *smtpSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
	return nil
}

func (s *smtpSender) closeLocked() {
	if s.client == nil {
		return
	}
	if err := s.client.Close(); err != nil {
		s.logger.Debug("smtp connection close failed", zap.Error(err))
	}
	s.client = nil
}
