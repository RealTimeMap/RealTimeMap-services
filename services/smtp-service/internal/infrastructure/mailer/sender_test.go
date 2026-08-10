package mailer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
)

// fakeSMTP — минимальный SMTP-сервер для тестов: без TLS и авторизации,
// отвечает ровно настолько, чтобы go-mail довёл диалог до конца.
//
// Нужен, потому что главные свойства sender'а — удержание соединения между
// письмами и восстановление после разрыва — видны только на стороне сервера.
type fakeSMTP struct {
	listener net.Listener

	mu          sync.Mutex
	connections int
	delivered   []string
	// replyOnRcpt подменяет ответ на RCPT TO, чтобы имитировать отказ сервера.
	replyOnRcpt string
	accepted    int
	active      []net.Conn
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &fakeSMTP{listener: ln}
	go s.serve()
	t.Cleanup(func() { ln.Close() })

	return s
}

func (s *fakeSMTP) addr() (string, int) {
	a := s.listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port
}

func (s *fakeSMTP) stats() (conns int, delivered []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connections, append([]string(nil), s.delivered...)
}

func (s *fakeSMTP) setRcptReply(reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replyOnRcpt = reply
}

// dropConnections обрывает все открытые сессии, не предупреждая клиента —
// так ведут себя провайдеры, закрывающие неактивные соединения.
func (s *fakeSMTP) dropConnections() {
	s.mu.Lock()
	conns := s.active
	s.active = nil
	s.mu.Unlock()

	for _, c := range conns {
		c.Close()
	}
}

func (s *fakeSMTP) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.connections++
		s.active = append(s.active, conn)
		s.mu.Unlock()
		go s.handle(conn)
	}
}

func (s *fakeSMTP) handle(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	write := func(format string, args ...any) error {
		if _, err := fmt.Fprintf(w, format+"\r\n", args...); err != nil {
			return err
		}
		return w.Flush()
	}

	if err := write("220 fake ESMTP"); err != nil {
		return
	}

	var inData bool
	var body strings.Builder

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false

				s.mu.Lock()
				s.delivered = append(s.delivered, body.String())
				s.accepted++
				s.mu.Unlock()

				body.Reset()

				if err := write("250 2.0.0 OK"); err != nil {
					return
				}
				continue
			}
			body.WriteString(line)
			body.WriteString("\n")
			continue
		}

		verb := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(verb, "EHLO"), strings.HasPrefix(verb, "HELO"):
			fmt.Fprintf(w, "250-fake greets you\r\n")
			fmt.Fprintf(w, "250 SIZE 35882577\r\n")
			w.Flush()
		case strings.HasPrefix(verb, "MAIL FROM"):
			write("250 2.1.0 OK")
		case strings.HasPrefix(verb, "RCPT TO"):
			s.mu.Lock()
			reply := s.replyOnRcpt
			s.mu.Unlock()
			if reply == "" {
				reply = "250 2.1.5 OK"
			}
			write(reply)
		case verb == "DATA":
			write("354 send it")
			inData = true
		case verb == "RSET":
			write("250 2.0.0 OK")
		case verb == "NOOP":
			write("250 2.0.0 OK")
		case verb == "QUIT":
			write("221 2.0.0 bye")
			return
		default:
			write("500 5.5.2 unrecognized")
		}
	}
}

func newTestSender(t *testing.T, s *fakeSMTP, tune func(*Config)) email.Sender {
	t.Helper()

	host, port := s.addr()
	cfg := Config{
		Host:     host,
		Port:     port,
		UseSSL:   false,
		From:     "noreply@realtimemap.ru",
		FromName: "RealTimeMap",
		Timeout:  3 * time.Second,
		// Фейковый сервер не умеет TLS; в бою шифрование обязательно.
		AllowInsecure: true,
	}
	if tune != nil {
		tune(&cfg)
	}

	sender, err := NewSender(cfg, logger.NewNop())
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}
	t.Cleanup(func() {
		if c, ok := sender.(interface{ Close() error }); ok {
			c.Close()
		}
	})

	return sender
}

func testMessage(to string) email.OutgoingMessage {
	return email.OutgoingMessage{
		To:      to,
		Subject: "Добро пожаловать",
		HTML:    "<p>Здравствуйте!</p>",
	}
}

func TestSenderDelivers(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	if err := sender.Send(context.Background(), testMessage("user@example.com")); err != nil {
		t.Fatalf("send: %v", err)
	}

	conns, delivered := srv.stats()
	if conns != 1 {
		t.Errorf("opened %d connections, want 1", conns)
	}
	if len(delivered) != 1 {
		t.Fatalf("delivered %d messages, want 1", len(delivered))
	}

	msg := delivered[0]

	// Кириллица уезжает закодированной (quoted-printable в теле,
	// RFC 2047 в заголовках), поэтому проверяем по структуре, а не по тексту.
	if !strings.Contains(strings.ToLower(msg), "text/html") {
		t.Error("content type is not HTML")
	}
	if !strings.Contains(msg, "<p>") {
		t.Error("HTML body not delivered")
	}
	// Письмо от имени, а не от голого адреса: так меньше шансов попасть в спам.
	if !strings.Contains(msg, "RealTimeMap") {
		t.Error("From display name missing")
	}
	if !strings.Contains(msg, "noreply@realtimemap.ru") {
		t.Error("From address missing")
	}
	if !strings.Contains(msg, "user@example.com") {
		t.Error("To address missing")
	}
	if !strings.Contains(strings.ToLower(msg), "subject:") {
		t.Error("subject header missing")
	}
}

// Главное свойство: несколько писем подряд идут по одному соединению.
// DialAndSend на каждое письмо стоит полного круга TCP+TLS+AUTH и упирается
// в лимит одновременных подключений у провайдера.
func TestSenderReusesConnection(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	for i := 0; i < 5; i++ {
		if err := sender.Send(context.Background(), testMessage(fmt.Sprintf("user%d@example.com", i))); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	conns, delivered := srv.stats()
	if conns != 1 {
		t.Errorf("opened %d connections for 5 emails, want 1", conns)
	}
	if len(delivered) != 5 {
		t.Errorf("delivered %d messages, want 5", len(delivered))
	}
}

// Сервер молча закрывает протухшую сессию, и узнать об этом можно только по
// неудачной команде. Письмо при этом терять нельзя — sender переоткрывает
// соединение и повторяет отправку на нём.
func TestSenderReconnectsAfterServerDrop(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	// Первое письмо проходит и оставляет живое соединение в кэше sender'а.
	if err := sender.Send(context.Background(), testMessage("first@example.com")); err != nil {
		t.Fatalf("first send: %v", err)
	}

	// Сервер обрывает все текущие сессии: с точки зрения sender'а соединение
	// всё ещё «открыто», и провал обнаружится только на следующей команде.
	srv.dropConnections()

	if err := sender.Send(context.Background(), testMessage("second@example.com")); err != nil {
		t.Fatalf("send after drop: %v", err)
	}

	conns, delivered := srv.stats()
	if conns < 2 {
		t.Errorf("opened %d connections, want at least 2 (reconnect expected)", conns)
	}
	if len(delivered) < 2 {
		t.Errorf("delivered %d messages, want 2 (the second one must survive the drop)", len(delivered))
	}
}

// Простой дольше IdleTimeout — соединение считается протухшим заранее, не
// дожидаясь провала команды.
func TestSenderReopensIdleConnection(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, func(c *Config) {
		c.IdleTimeout = 50 * time.Millisecond
	})

	if err := sender.Send(context.Background(), testMessage("first@example.com")); err != nil {
		t.Fatalf("first send: %v", err)
	}

	time.Sleep(120 * time.Millisecond)

	if err := sender.Send(context.Background(), testMessage("second@example.com")); err != nil {
		t.Fatalf("second send: %v", err)
	}

	conns, _ := srv.stats()
	if conns != 2 {
		t.Errorf("opened %d connections, want 2 (idle one must be reopened)", conns)
	}
}

// 5xx означает «этого адреса не существует». Повторять такое вредно: попытки
// доставки на несуществующие адреса портят репутацию отправителя.
func TestSenderPermanentRejection(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	srv.setRcptReply("550 5.1.1 no such user here")

	err := sender.Send(context.Background(), testMessage("ghost@example.com"))
	if err == nil {
		t.Fatal("send succeeded despite 550")
	}
	if !errors.Is(err, email.ErrPermanent) {
		t.Errorf("error is %v, want permanent", err)
	}
}

// 4xx — «попробуй позже»: письмо должно вернуться в очередь, а не сгореть.
func TestSenderTransientRejection(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	srv.setRcptReply("451 4.3.0 mailbox busy, try again")

	err := sender.Send(context.Background(), testMessage("busy@example.com"))
	if err == nil {
		t.Fatal("send succeeded despite 451")
	}
	if !errors.Is(err, email.ErrTransient) {
		t.Errorf("error is %v, want transient", err)
	}
}

// Недоступный сервер — временная беда: письмо ждёт следующей попытки.
func TestSenderUnreachableServerIsTransient(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // порт заведомо закрыт

	sender, err := NewSender(Config{
		Host:          "127.0.0.1",
		Port:          port,
		From:          "noreply@realtimemap.ru",
		Timeout:       time.Second,
		AllowInsecure: true,
	}, logger.NewNop())
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	sendErr := sender.Send(context.Background(), testMessage("user@example.com"))
	if sendErr == nil {
		t.Fatal("send succeeded against closed port")
	}
	if !errors.Is(sendErr, email.ErrTransient) {
		t.Errorf("error is %v, want transient", sendErr)
	}
}

// Битый адрес получателя ловится до выхода в сеть.
func TestSenderInvalidRecipientIsPermanent(t *testing.T) {
	srv := newFakeSMTP(t)
	sender := newTestSender(t, srv, nil)

	err := sender.Send(context.Background(), testMessage("not-an-email"))
	if err == nil {
		t.Fatal("send succeeded with malformed recipient")
	}
	if !errors.Is(err, email.ErrPermanent) {
		t.Errorf("error is %v, want permanent", err)
	}

	if conns, _ := srv.stats(); conns != 0 {
		t.Errorf("opened %d connections for an invalid address, want 0", conns)
	}
}

func TestNewSenderValidatesConfig(t *testing.T) {
	if _, err := NewSender(Config{From: "a@b.c"}, logger.NewNop()); err == nil {
		t.Error("empty host accepted")
	}
	if _, err := NewSender(Config{Host: "smtp.example.com"}, logger.NewNop()); err == nil {
		t.Error("empty from address accepted")
	}
}

// From по умолчанию совпадает с учётной записью: в MVP это один и тот же ящик,
// расходятся они только после переезда на собственный домен.
func TestNewSenderDefaultsFromToUsername(t *testing.T) {
	srv := newFakeSMTP(t)
	host, port := srv.addr()

	sender, err := NewSender(Config{
		Host:          host,
		Port:          port,
		Username:      "service@yandex.ru",
		Timeout:       3 * time.Second,
		AllowInsecure: true,
	}, logger.NewNop())
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	if err := sender.Send(context.Background(), testMessage("user@example.com")); err != nil {
		t.Fatalf("send: %v", err)
	}

	_, delivered := srv.stats()
	if len(delivered) != 1 || !strings.Contains(delivered[0], "service@yandex.ru") {
		t.Error("From did not fall back to username")
	}
}
