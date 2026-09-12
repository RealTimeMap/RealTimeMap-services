package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	userclient "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/user"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	domaintemplate "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	embedtemplate "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/infrastructure/template"
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

// stubUsers подменяет UserService: отдаёт заранее заданных пользователей и
// умеет изображать недоступность сервиса.
type stubUsers struct {
	byID map[int64]*userclient.User
	err  error

	mu    sync.Mutex
	calls []int64
}

func (s *stubUsers) GetUserByID(_ context.Context, id int64) (*userclient.User, error) {
	s.mu.Lock()
	s.calls = append(s.calls, id)
	s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	u, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", userclient.ErrNotFound, id)
	}
	return u, nil
}

func newHandler(enq *recordingEnqueuer) *Handler {
	return NewHandler(enq, &stubUsers{}, "https://realtimemap.ru", logger.NewNop())
}

func newHandlerWithUsers(enq *recordingEnqueuer, users UserResolver) *Handler {
	return NewHandler(enq, users, "https://realtimemap.ru", logger.NewNop())
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

// --- события комментариев ---

// commentMessage собирает сообщение comment.created в общем конверте.
func commentMessage(t *testing.T, mutate func(*events.CommentPayload)) segmentio.Message {
	t.Helper()

	parentID := uint(10)
	parentUserID := uint(100)
	payload := events.CommentPayload{
		CommentID:    77,
		UserID:       200,
		Username:     "replier",
		EntityType:   "mark_action",
		EntityID:     42,
		ParentID:     &parentID,
		ParentUserID: &parentUserID,
		Content:      "Согласен, отличное место",
	}
	if mutate != nil {
		mutate(&payload)
	}

	body, err := json.Marshal(events.NewCommentCreated(payload))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return segmentio.Message{Topic: "comment-service.events", Value: body}
}

func usersWith(id int64, username, mail string) *stubUsers {
	return &stubUsers{byID: map[int64]*userclient.User{
		id: {ID: id, Username: username, Email: mail},
	}}
}

// Ответ на комментарий: письмо уходит автору родителя, а не автору ответа.
func TestHandlerQueuesCommentReply(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := usersWith(100, "parentAuthor", "parent@example.com")

	if err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), commentMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if enq.count() != 1 {
		t.Fatalf("enqueued %d emails, want 1", enq.count())
	}

	in := enq.last()
	if in.TemplateID != "commentReply" {
		t.Errorf("template = %q, want commentReply", in.TemplateID)
	}
	if in.ToEmail != "parent@example.com" {
		t.Errorf("to = %q, want parent@example.com", in.ToEmail)
	}
	if in.Data["username"] != "parentAuthor" {
		t.Errorf("username = %v, want parentAuthor", in.Data["username"])
	}
	if in.Data["authorName"] != "replier" {
		t.Errorf("authorName = %v, want replier", in.Data["authorName"])
	}
	if in.Data["commentUrl"] != "https://realtimemap.ru/marks/42#comment-77" {
		t.Errorf("commentUrl = %v", in.Data["commentUrl"])
	}
	// Ключ по комментарию-ответу: повторная доставка события не создаёт второго письма.
	if in.IdempotencyKey != "comment.created:77" {
		t.Errorf("idempotency key = %q", in.IdempotencyKey)
	}
}

// Комментарий верхнего уровня уведомлять некого: у сущности нет владельца,
// известного этому сервису.
func TestHandlerSkipsTopLevelComment(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := usersWith(100, "someone", "someone@example.com")

	msg := commentMessage(t, func(p *events.CommentPayload) {
		p.ParentID = nil
		p.ParentUserID = nil
	})

	if err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if enq.count() != 0 {
		t.Errorf("enqueued %d emails for a top-level comment", enq.count())
	}
}

// Ответ самому себе не должен порождать письмо.
func TestHandlerSkipsSelfReply(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := usersWith(200, "self", "self@example.com")

	msg := commentMessage(t, func(p *events.CommentPayload) {
		same := uint(200)
		p.ParentUserID = &same // совпадает с UserID автора ответа
	})

	if err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if enq.count() != 0 {
		t.Errorf("enqueued %d emails for a self-reply", enq.count())
	}
}

// Нет такого пользователя — повтор не поможет, сообщение коммитится.
func TestHandlerSkipsWhenRecipientNotFound(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := &stubUsers{byID: map[int64]*userclient.User{}}

	err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), commentMessage(t, nil))
	if !errors.Is(err, consumer.ErrSkip) {
		t.Errorf("error = %v, want skip", err)
	}
	if errors.Is(err, consumer.ErrRetryable) {
		t.Error("missing user marked retryable — partition would stall forever")
	}
}

// UserService недоступен — сообщение обязано перечитаться, письмо терять нельзя.
func TestHandlerRetriesWhenUserServiceUnavailable(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := &stubUsers{err: fmt.Errorf("%w: dial tcp", userclient.ErrUnavailable)}

	err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), commentMessage(t, nil))
	if !errors.Is(err, consumer.ErrRetryable) {
		t.Errorf("error = %v, want retryable", err)
	}
	if enq.count() != 0 {
		t.Error("email queued despite unresolved recipient")
	}
}

// Продюсер может не прислать имя автора; письмо всё равно должно уйти —
// шаблон требует authorName непустым.
func TestHandlerFallsBackToDefaultAuthorName(t *testing.T) {
	enq := &recordingEnqueuer{}
	users := usersWith(100, "parentAuthor", "parent@example.com")

	msg := commentMessage(t, func(p *events.CommentPayload) {
		p.Username = "   "
	})

	if err := newHandlerWithUsers(enq, users).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got := enq.last().Data["authorName"]; got != "Участник" {
		t.Errorf("authorName = %v, want fallback", got)
	}
}

// --- совместимость форматов ---

// Регистрация в общем конверте {type, payload} должна читаться так же, как
// старый плоский формат.
func TestHandlerReadsRegisteredInEnvelope(t *testing.T) {
	enq := &recordingEnqueuer{}

	event := events.UserRegisteredEvent{
		Envelop: events.NewEnvelop(events.UserRegistered),
		Payload: events.UserRegisteredPayload{
			UserID:   80,
			Username: "TestUser",
			Email:    "TestUser@yandex.com",
		},
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	msg := segmentio.Message{Topic: "user-service", Value: body}
	if err := newHandler(enq).HandleMessage(context.Background(), msg); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if enq.count() != 1 {
		t.Fatalf("enqueued %d emails, want 1", enq.count())
	}
	in := enq.last()
	if in.ToEmail != "TestUser@yandex.com" || in.Data["username"] != "TestUser" {
		t.Errorf("payload not read from envelope: to=%q username=%v", in.ToEmail, in.Data["username"])
	}
	// Ключ тот же, что у плоского формата: переход на конверт не должен
	// приводить к повторной отправке приветствия тем, кто его уже получил.
	if in.IdempotencyKey != "user.registered:80" {
		t.Errorf("idempotency key = %q", in.IdempotencyKey)
	}
}

// --- согласованность с контрактами шаблонов ---

// renderingEnqueuer прогоняет данные письма через настоящий рендерер.
//
// Остальные тесты используют мок и потому не заметят, что хендлер перестал
// слать поле, объявленное шаблоном обязательным: письмо упало бы только в
// рантайме, на живом пользователе.
type renderingEnqueuer struct {
	renderer *domaintemplate.Renderer
	rendered []*domaintemplate.Rendered
}

func (r *renderingEnqueuer) Enqueue(ctx context.Context, in email.EnqueueInput) (*email.EnqueueResult, error) {
	out, err := r.renderer.Render(ctx, in.TemplateID, in.TemplateVersion, in.Data)
	if err != nil {
		return nil, err
	}
	r.rendered = append(r.rendered, out)
	return &email.EnqueueResult{EmailID: uuid.New()}, nil
}

func newRenderingHandler(t *testing.T, users UserResolver) (*Handler, *renderingEnqueuer) {
	t.Helper()

	provider, err := embedtemplate.NewProvider()
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	enq := &renderingEnqueuer{renderer: domaintemplate.NewRenderer(provider)}
	return NewHandler(enq, users, "https://realtimemap.ru", logger.NewNop()), enq
}

// Данные, которые хендлер собирает для приветствия, должны покрывать контракт
// шаблона целиком.
func TestWelcomeDataSatisfiesTemplate(t *testing.T) {
	h, enq := newRenderingHandler(t, &stubUsers{})

	if err := h.HandleMessage(context.Background(), registeredMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(enq.rendered) != 1 {
		t.Fatalf("rendered %d emails, want 1", len(enq.rendered))
	}
	if !strings.Contains(enq.rendered[0].HTML, "TestUser") {
		t.Error("welcome body has no user name")
	}
	if !strings.Contains(enq.rendered[0].HTML, "https://realtimemap.ru/map") {
		t.Error("welcome body has no map link built from config")
	}
}

// То же для письма об ответе на комментарий.
func TestCommentReplyDataSatisfiesTemplate(t *testing.T) {
	h, enq := newRenderingHandler(t, usersWith(100, "parentAuthor", "parent@example.com"))

	if err := h.HandleMessage(context.Background(), commentMessage(t, nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(enq.rendered) != 1 {
		t.Fatalf("rendered %d emails, want 1", len(enq.rendered))
	}
	if !strings.Contains(enq.rendered[0].HTML, "Согласен, отличное место") {
		t.Error("reply body has no comment text")
	}
}
