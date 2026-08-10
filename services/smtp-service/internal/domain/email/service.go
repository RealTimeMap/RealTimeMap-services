package email

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DedupWindow — окно, внутри которого одинаковые письма считаются одним.
//
// Глобальная уникальность не годится: юзер, дважды запросивший код
// подтверждения, должен получить его дважды. Окно отсекает повторы, вызванные
// повторной доставкой одного события, не мешая легитимным.
//
// Окна фиксированные (bucket = unix / DedupWindow), а не скользящие: граница
// привязана к абсолютному времени, а не к моменту первого письма. Поэтому
// фактическая защита длится от 0 до DedupWindow — письмо, попавшее к концу
// окна, повторится уже в следующем. Для повторной доставки Kafka-события,
// приходящей через секунды, этого достаточно, а скользящее окно потребовало бы
// чтения перед вставкой и вернуло бы гонку, ради устранения которой
// дедупликация и вводилась.
const DedupWindow = 5 * time.Minute

// EnqueueInput — запрос на отправку письма.
type EnqueueInput struct {
	TemplateID      string
	TemplateVersion *uint

	ToEmail string
	UserID  *uint64

	Data     map[string]any
	Priority int

	// IdempotencyKey задаёт смысл «то же самое письмо» явно. Если пуст,
	// считается хеш содержимого — работает, но ломается, когда в данных есть
	// уникальное поле вроде времени регистрации.
	IdempotencyKey string

	// ScheduledAt откладывает отправку. Пустое значение — отправить сразу.
	ScheduledAt *time.Time

	TraceID string
}

// EnqueueResult — что произошло с запросом.
type EnqueueResult struct {
	EmailID uuid.UUID

	// Duplicate означает, что письмо уже стояло в очереди и повторно не
	// создавалось. Для вызывающего это успех, а не ошибка.
	Duplicate bool
}

// Service ставит письма в очередь.
//
// Единственный вход в очередь: и Kafka, и HTTP проходят через него, поэтому
// валидация, рендер и дедупликация работают одинаково независимо от источника.
type Service struct {
	emails   Repository
	events   EventRepository
	renderer *template.Renderer

	maxAttempt uint
	logger     *zap.Logger
}

func NewService(
	emails Repository,
	events EventRepository,
	renderer *template.Renderer,
	maxAttempt uint,
	logger *zap.Logger,
) *Service {
	if maxAttempt == 0 {
		maxAttempt = 5
	}
	return &Service{
		emails:     emails,
		events:     events,
		renderer:   renderer,
		maxAttempt: maxAttempt,
		logger:     logger,
	}
}

// Enqueue проверяет запрос, рендерит письмо и ставит его в очередь.
//
// Рендер здесь, а не в воркере: ошибка шаблона возвращается вызывающему
// сразу — админка показывает «нет обязательного поля» вместо молчаливого
// failed через час. Письмо становится неизменяемым с этого момента, поэтому
// повторная попытка отправит ровно то, что было согласовано.
func (s *Service) Enqueue(ctx context.Context, in EnqueueInput) (*EnqueueResult, error) {
	to, err := normalizeEmail(in.ToEmail)
	if err != nil {
		return nil, err
	}

	rendered, err := s.renderer.Render(ctx, in.TemplateID, in.TemplateVersion, in.Data)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	scheduledAt := now
	if in.ScheduledAt != nil && in.ScheduledAt.After(now) {
		scheduledAt = *in.ScheduledAt
	}

	dedupKey := in.IdempotencyKey
	if dedupKey == "" {
		dedupKey = contentDedupKey(in.TemplateID, to, in.Data)
	}

	record := &Email{
		ID:              uuid.New(),
		Status:          StatusQueued,
		Priority:        in.Priority,
		ToEmail:         to,
		Subject:         rendered.Subject,
		HTML:            rendered.HTML,
		TemplateID:      rendered.TemplateID,
		TemplateVersion: rendered.Version,
		DedupKey:        dedupKey,
		DedupBucket:     dedupBucket(now),
		ScheduledAt:     scheduledAt,
		MaxAttempt:      s.maxAttempt,
		TraceID:         in.TraceID,
	}

	created, err := s.emails.Create(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("enqueue email: %w", err)
	}

	if !created {
		s.logger.Debug("email already queued, skipping duplicate",
			zap.String("email_id", record.ID.String()),
			zap.String("template", in.TemplateID),
			zap.String("dedup_key", dedupKey),
		)
		return &EnqueueResult{EmailID: record.ID, Duplicate: true}, nil
	}

	s.appendEvent(ctx, record, EventCreated, nil)

	s.logger.Info("email queued",
		zap.String("email_id", record.ID.String()),
		zap.String("template", record.TemplateID),
		zap.String("to", MaskEmail(record.ToEmail)),
		zap.String("trace_id", record.TraceID),
	)

	return &EnqueueResult{EmailID: record.ID}, nil
}

// appendEvent записывает переход в историю.
//
// Сбой записи истории не отменяет саму операцию: письмо важнее журнала.
func (s *Service) appendEvent(ctx context.Context, e *Email, eventType EventType, details map[string]any) {
	event := &Event{
		EmailID:         e.ID,
		EventType:       eventType,
		WorkerID:        e.WorkerID,
		OccurredTime:    time.Now(),
		TemplateVersion: e.TemplateVersion,
		Details:         details,
		Attempt:         e.Attempt,
		MaxAttempt:      e.MaxAttempt,
	}

	if err := s.events.Append(ctx, event); err != nil {
		s.logger.Warn("failed to append email event",
			zap.String("email_id", e.ID.String()),
			zap.String("event_type", string(eventType)),
			zap.Error(err),
		)
	}
}

// normalizeEmail проверяет адрес и приводит его к каноничному виду.
//
// Проверка на входе, а не при отправке: заведомо непригодный адрес не должен
// занимать воркера и накручивать попытки.
func normalizeEmail(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrInvalidEmail(value)
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmail(value)
	}

	// ParseAddress принимает форму "Имя <a@b.c>" — в очередь кладём только
	// сам адрес, иначе он попадёт в dedup-ключ вместе с именем.
	if _, err := parseDomain(addr.Address); err != nil {
		return "", ErrInvalidEmail(value)
	}

	return addr.Address, nil
}

func parseDomain(address string) (string, error) {
	at := strings.LastIndex(address, "@")
	if at <= 0 || at == len(address)-1 {
		return "", ErrInvalidEmail(address)
	}
	domain := address[at+1:]
	// Домен без точки — это локальное имя вроде "localhost": для писем,
	// уходящих во внешний мир, такой адрес недостижим.
	if !strings.Contains(domain, ".") {
		return "", ErrInvalidEmail(address)
	}
	return domain, nil
}

// dedupBucket — номер временного окна.
func dedupBucket(at time.Time) int64 {
	return at.Unix() / int64(DedupWindow.Seconds())
}

// contentDedupKey строит ключ по содержимому письма.
//
// Запасной вариант для отправителей, не передавших ключ явно. Данные
// сериализуются с сортировкой ключей: обход map в Go неупорядочен, и без
// этого одно и то же письмо давало бы разные хеши.
func contentDedupKey(templateID, to string, data map[string]any) string {
	h := sha256.New()
	h.Write([]byte(templateID))
	h.Write([]byte{0})
	h.Write([]byte(to))
	h.Write([]byte{0})
	h.Write([]byte(canonicalJSON(data)))

	return "content:" + hex.EncodeToString(h.Sum(nil))
}

func canonicalJSON(data map[string]any) string {
	if len(data) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(k)
		b.Write(key)
		b.WriteByte(':')

		value, err := json.Marshal(data[k])
		if err != nil {
			// Незасериализуемое значение не должно ронять отправку: оно лишь
			// делает ключ менее точным.
			value = []byte(fmt.Sprintf("%q", fmt.Sprint(data[k])))
		}
		b.Write(value)
	}
	b.WriteByte('}')

	return b.String()
}

// MaskEmail прячет адрес для логов.
//
// Логи уезжают в Loki и живут там по retention: адреса в открытом виде
// вычистить задним числом уже нельзя.
func MaskEmail(address string) string {
	at := strings.LastIndex(address, "@")
	if at <= 0 {
		return "***"
	}

	local, domain := address[:at], address[at:]
	switch len(local) {
	case 1:
		return "*" + domain
	case 2:
		return local[:1] + "*" + domain
	default:
		return local[:1] + strings.Repeat("*", 3) + local[len(local)-1:] + domain
	}
}
