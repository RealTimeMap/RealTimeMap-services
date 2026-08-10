package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
	segmentio "github.com/segmentio/kafka-go"
)

type recordingEnqueuer struct {
	mu     sync.Mutex
	calls  []email.EnqueueInput
	result *email.EnqueueResult
	err    error
}

func (r *recordingEnqueuer) Enqueue(_ context.Context, in email.EnqueueInput) (*email.EnqueueResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, in)

	if r.err != nil {
		return nil, r.err
	}
	if r.result != nil {
		return r.result, nil
	}
	return &email.EnqueueResult{EmailID: uuid.New()}, nil
}

func (r *recordingEnqueuer) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recordingEnqueuer) last() email.EnqueueInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[len(r.calls)-1]
}

func registeredMessage(t *testing.T, mutate func(map[string]any)) segmentio.Message {
	t.Helper()

	payload := map[string]any{
		"event_type":    "user.registered",
		"user_id":       80,
		"username":      "TestUser",
		"email":         "TestUser@yandex.com",
		"phone":         nil,
		"is_verified":   false,
		"oauth":         false,
		"registered_at": "2026-08-08T13:24:54.071805+00:00",
	}
	if mutate != nil {
		mutate(payload)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return segmentio.Message{Topic: "user.registered", Value: body}
}

func newHandler(enq *recordingEnqueuer) *Handler {
	return NewHandler(enq, logger.NewNop())
}

func TestHandlerQueuesWelcomeEmail(t *testing.T) {
	enq := &recordingEnqueuer{}

	if err := newHandler(enq).HandleMessage(context.Background(), registeredMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if enq.count() != 1 {
		t.Fatalf("enqueued %d emails, want 1", enq.count())
	}

	in := enq.last()
	if in.TemplateID != "welcome" {
		t.Errorf("template = %q, want welcome", in.TemplateID)
	}
	if in.ToEmail != "TestUser@yandex.com" {
		t.Errorf("to = %q", in.ToEmail)
	}
	if in.Data["username"] != "TestUser" {
		t.Errorf("username = %v", in.Data["username"])
	}
	if in.UserID == nil || *in.UserID != 80 {
		t.Errorf("user_id = %v", in.UserID)
	}
	// Ключ построен на user_id, а не на содержимом: смысл «одно приветственное
	// письмо на пользователя» не должен зависеть от полей события.
	if in.IdempotencyKey != "user.registered:80" {
		t.Errorf("idempotency key = %q", in.IdempotencyKey)
	}
}

// Тот же пользователь — тот же ключ, даже если остальные поля события
// изменились (например, продюсер начал слать другое registered_at).
func TestHandlerKeyIgnoresVolatileFields(t *testing.T) {
	enq := &recordingEnqueuer{}
	h := newHandler(enq)

	first := registeredMessage(t, nil)
	second := registeredMessage(t, func(p map[string]any) {
		p["registered_at"] = "2026-09-01T10:00:00+00:00"
		p["is_verified"] = true
	})

	for _, msg := range []segmentio.Message{first, second} {
		if err := h.HandleMessage(context.Background(), msg); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	if enq.calls[0].IdempotencyKey != enq.calls[1].IdempotencyKey {
		t.Errorf("keys differ: %q vs %q", enq.calls[0].IdempotencyKey, enq.calls[1].IdempotencyKey)
	}
}

// В MVP письмо одно и то же независимо от способа регистрации: ветвление
// требует токена подтверждения, которого в событии нет.
func TestHandlerIgnoresOAuthAndVerifiedFlags(t *testing.T) {
	for _, oauth := range []bool{true, false} {
		enq := &recordingEnqueuer{}
		msg := registeredMessage(t, func(p map[string]any) {
			p["oauth"] = oauth
			p["is_verified"] = oauth
		})

		if err := newHandler(enq).HandleMessage(context.Background(), msg); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if got := enq.last().TemplateID; got != "welcome" {
			t.Errorf("oauth=%v produced template %q, want welcome", oauth, got)
		}
	}
}

// Дубль — успех: offset коммитится, повторная доставка события не создаёт
// второго письма и не роняет обработку.
func TestHandlerAcceptsDuplicate(t *testing.T) {
	enq := &recordingEnqueuer{
		result: &email.EnqueueResult{EmailID: uuid.New(), Duplicate: true},
	}

	if err := newHandler(enq).HandleMessage(context.Background(), registeredMessage(t, nil)); err != nil {
		t.Errorf("duplicate reported as failure: %v", err)
	}
}

// События, до которых сервису нет дела, пропускаются молча: топик может быть
// общим.
func TestHandlerIgnoresUnknownEvent(t *testing.T) {
	enq := &recordingEnqueuer{}
	msg := registeredMessage(t, func(p map[string]any) {
		p["event_type"] = "user.deleted"
	})

	if err := newHandler(enq).HandleMessage(context.Background(), msg); err != nil {
		t.Errorf("unknown event returned error: %v", err)
	}
	if enq.count() != 0 {
		t.Errorf("enqueued %d emails for an unrelated event", enq.count())
	}
}

// Битое сообщение пропускается с коммитом: перечитывание не исправит JSON.
func TestHandlerSkipsMalformedMessage(t *testing.T) {
	enq := &recordingEnqueuer{}
	msg := segmentio.Message{Topic: "user.registered", Value: []byte("{not json")}

	err := newHandler(enq).HandleMessage(context.Background(), msg)
	if !errors.Is(err, consumer.ErrSkip) {
		t.Errorf("error = %v, want skip", err)
	}
	if enq.count() != 0 {
		t.Error("malformed message reached the queue")
	}
}

// Ошибка данных не должна останавливать партицию: сообщение в топике не
// изменится, и Retryable заклинил бы обработку навсегда.
func TestHandlerSkipsUnprocessableEvent(t *testing.T) {
	cases := map[string]error{
		"invalid email":    apperror.NewInvalidFormatError("to", "email", "bad"),
		"missing field":    apperror.NewRequiredError("UserName"),
		"no such template": apperror.NewNotFoundErrorByID("template", "welcome"),
	}

	for name, cause := range cases {
		t.Run(name, func(t *testing.T) {
			enq := &recordingEnqueuer{err: cause}

			err := newHandler(enq).HandleMessage(context.Background(), registeredMessage(t, nil))
			if !errors.Is(err, consumer.ErrSkip) {
				t.Errorf("error = %v, want skip", err)
			}
			if errors.Is(err, consumer.ErrRetryable) {
				t.Error("data error marked retryable — partition would stall forever")
			}
		})
	}
}

// Недоступность БД, наоборот, обязана быть retryable: письмо нельзя терять
// из-за временного сбоя.
func TestHandlerRetriesOnInfrastructureFailure(t *testing.T) {
	cases := map[string]error{
		"plain error":         errors.New("connection refused"),
		"internal error":      apperror.NewInternalError("db", errors.New("timeout")),
		"service unavailable": apperror.NewServiceUnavailableError("postgres", errors.New("down")),
	}

	for name, cause := range cases {
		t.Run(name, func(t *testing.T) {
			enq := &recordingEnqueuer{err: cause}

			err := newHandler(enq).HandleMessage(context.Background(), registeredMessage(t, nil))
			if !errors.Is(err, consumer.ErrRetryable) {
				t.Errorf("error = %v, want retryable", err)
			}
		})
	}
}

// Тип события может приезжать только в заголовке — часть продюсеров в проекте
// кладёт его именно туда.
func TestHandlerReadsEventTypeFromHeader(t *testing.T) {
	enq := &recordingEnqueuer{}

	msg := registeredMessage(t, func(p map[string]any) {
		delete(p, "event_type")
	})
	msg.Headers = []segmentio.Header{{Key: "event_type", Value: []byte("user.registered")}}

	if err := newHandler(enq).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if enq.count() != 1 {
		t.Errorf("enqueued %d emails, want 1", enq.count())
	}
}

// Трассировка должна пережить переход через очередь: иначе связь между
// событием и отправкой через несколько минут теряется.
func TestHandlerPropagatesTraceID(t *testing.T) {
	enq := &recordingEnqueuer{}

	msg := registeredMessage(t, nil)
	msg.Headers = []segmentio.Header{{Key: "X-Trace-Id", Value: []byte("trace-42")}}

	if err := newHandler(enq).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got := enq.last().TraceID; got != "trace-42" {
		t.Errorf("trace_id = %q, want trace-42", got)
	}
}
