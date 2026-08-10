package mailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/wneessen/go-mail"
)

func classOf(t *testing.T, err error) string {
	t.Helper()
	switch {
	case err == nil:
		return "nil"
	case errors.Is(err, email.ErrTransient):
		return "transient"
	case errors.Is(err, email.ErrPermanent):
		return "permanent"
	default:
		return "unclassified"
	}
}

// Сетевые отказы обязаны быть временными. Отдельная проверка нужна потому,
// что go-mail определяет временность по первому символу текста ошибки — у
// "dial tcp: i/o timeout" там не '4', и библиотека сочла бы её постоянной,
// то есть письмо потерялось бы при первом же моргании сети.
func TestClassifyNetworkErrorsAreTransient(t *testing.T) {
	cases := map[string]error{
		"op error":       &net.OpError{Op: "dial", Err: errors.New("connection refused")},
		"dns error":      &net.DNSError{Err: "no such host", Name: "smtp.example.com"},
		"eof":            io.EOF,
		"unexpected eof": io.ErrUnexpectedEOF,
		"deadline":       context.DeadlineExceeded,
		"net closed":     net.ErrClosed,
		"io timeout":     errors.New("dial tcp 1.2.3.4:465: i/o timeout"),
		"conn reset":     errors.New("read tcp: connection reset by peer"),
		"broken pipe":    errors.New("write tcp: broken pipe"),
		"unreachable":    errors.New("dial tcp: network is unreachable"),
		"tls timeout":    errors.New("net/http: TLS handshake timeout"),
		"wrapped":        fmt.Errorf("send failed: %w", io.EOF),
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			if got := classOf(t, classify(err)); got != "transient" {
				t.Errorf("classified as %s, want transient", got)
			}
		})
	}
}

// Ошибки настройки повторять бессмысленно: пока конфиг не поправят, каждая
// попытка провалится точно так же.
func TestClassifyConfigErrorsArePermanent(t *testing.T) {
	cases := map[string]error{
		"auth failed":      errors.New("535 authentication failed"),
		"auth unsupported": errors.New("unsupported SMTP AUTH type"),
		"bad certificate":  errors.New("x509: certificate signed by unknown authority"),
		"cert expired":     errors.New("tls: failed to verify certificate: expired"),
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			if got := classOf(t, classify(err)); got != "permanent" {
				t.Errorf("classified as %s, want permanent", got)
			}
		})
	}
}

// RSET уходит уже после того, как сервер принял письмо. Ошибка на этом шаге
// означает, что доставка состоялась, а сорвалась только очистка сессии —
// повторять такое письмо нельзя, адресат получит его дважды.
func TestClassifyResetAfterDeliveryIsNotRetried(t *testing.T) {
	err := classify(&mail.SendError{Reason: mail.ErrSMTPReset})

	if got := classOf(t, err); got != "permanent" {
		t.Errorf("classified as %s, want permanent (retry would duplicate the email)", got)
	}
	if !strings.Contains(err.Error(), "delivered") {
		t.Errorf("error should state that the message was delivered: %q", err.Error())
	}
}

// Отказы, случившиеся до приёма письма, повторять нужно: доставки не было.
func TestClassifyPreDeliveryFailuresAreTransient(t *testing.T) {
	cases := map[string]mail.SendErrReason{
		"connection check": mail.ErrConnCheck,
		"data close":       mail.ErrSMTPDataClose,
		"write content":    mail.ErrWriteContent,
	}

	for name, reason := range cases {
		t.Run(name, func(t *testing.T) {
			if got := classOf(t, classify(&mail.SendError{Reason: reason})); got != "transient" {
				t.Errorf("classified as %s, want transient", got)
			}
		})
	}
}

// Письмо составлено неправильно — сервер ни при чём, повтор не поможет.
func TestClassifyMalformedMessageIsPermanent(t *testing.T) {
	cases := map[string]mail.SendErrReason{
		"bad sender":     mail.ErrGetSender,
		"bad recipients": mail.ErrGetRcpts,
		"no unencoded":   mail.ErrNoUnencoded,
	}

	for name, reason := range cases {
		t.Run(name, func(t *testing.T) {
			if got := classOf(t, classify(&mail.SendError{Reason: reason})); got != "permanent" {
				t.Errorf("classified as %s, want permanent", got)
			}
		})
	}
}

func TestClassifyNil(t *testing.T) {
	if err := classify(nil); err != nil {
		t.Errorf("classify(nil) = %v, want nil", err)
	}
}

// Неизвестная ошибка считается временной: потерянное письмо хуже лишней
// попытки, а потолок max_attempt всё равно ограничивает повторы.
func TestClassifyUnknownDefaultsToTransient(t *testing.T) {
	if got := classOf(t, classify(errors.New("something odd happened"))); got != "transient" {
		t.Errorf("classified as %s, want transient", got)
	}
}

// Исходная ошибка должна оставаться в тексте: без неё в логе будет только
// «transient», без кода ответа сервера и причины.
func TestClassifyPreservesOriginalError(t *testing.T) {
	original := errors.New("550 5.1.1 no such user")
	got := classify(original)

	if !errors.Is(got, email.ErrPermanent) && !errors.Is(got, email.ErrTransient) {
		t.Fatalf("error lost its class: %v", got)
	}
	if !strings.Contains(got.Error(), "no such user") {
		t.Errorf("original error text lost: %q", got.Error())
	}
}
