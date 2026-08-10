package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() { gin.SetMode(gin.TestMode) }

// memoryRepo — очередь в памяти, ровно настолько, насколько её видит
// HTTP-слой.
type memoryRepo struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*email.Email
	byKey  map[string]*email.Email
	listed []email.Email
	err    error
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		byID:  make(map[uuid.UUID]*email.Email),
		byKey: make(map[string]*email.Email),
	}
}

func (r *memoryRepo) Create(_ context.Context, e *email.Email) (bool, error) {
	if r.err != nil {
		return false, r.err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := e.DedupKey
	if existing, ok := r.byKey[key]; ok {
		*e = *existing
		return false, nil
	}

	stored := *e
	r.byID[e.ID] = &stored
	r.byKey[key] = &stored
	r.listed = append(r.listed, stored)

	return true, nil
}

func (r *memoryRepo) GetByID(_ context.Context, id uuid.UUID) (*email.Email, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.byID[id]; ok {
		return e, nil
	}
	return nil, email.ErrNotFound(id.String())
}

func (r *memoryRepo) List(_ context.Context, f email.Filter) ([]email.Email, int64, error) {
	if r.err != nil {
		return nil, 0, r.err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var out []email.Email
	for _, e := range r.listed {
		if f.Status != nil && e.Status != *f.Status {
			continue
		}
		if f.ToEmail != "" && e.ToEmail != f.ToEmail {
			continue
		}
		out = append(out, e)
	}

	return out, int64(len(out)), nil
}

func (r *memoryRepo) Claim(context.Context, string, int, time.Duration) ([]email.Email, error) {
	return nil, nil
}
func (r *memoryRepo) MarkSent(context.Context, uuid.UUID, time.Time) error { return nil }
func (r *memoryRepo) MarkFailed(context.Context, uuid.UUID, string) error  { return nil }
func (r *memoryRepo) Reschedule(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}
func (r *memoryRepo) ReleaseExpired(context.Context, time.Time) (int64, int64, error) {
	return 0, 0, nil
}
func (r *memoryRepo) CountSentSince(context.Context, time.Time) (int64, error) { return 0, nil }
func (r *memoryRepo) OldestQueuedAge(context.Context, time.Time) (time.Duration, error) {
	return 0, nil
}

type memoryEvents struct {
	mu     sync.Mutex
	events []email.Event
}

func (m *memoryEvents) Append(_ context.Context, e *email.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, *e)
	return nil
}

func (m *memoryEvents) ListByEmailID(_ context.Context, id uuid.UUID) ([]email.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []email.Event
	for _, e := range m.events {
		if e.EmailID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

type staticTemplates struct{}

func (staticTemplates) Get(_ context.Context, id string, _ *uint) (*template.Template, error) {
	if id != "welcome" {
		return nil, template.ErrNotFound(id)
	}
	return &template.Template{
		ID:           "welcome",
		Version:      1,
		Subject:      "Добро пожаловать, {{.UserName}}!",
		HTML:         `<p>Привет, {{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	}, nil
}

// newRouter собирает приложение без авторизации: она проверяется отдельно,
// а здесь мешала бы каждому запросу.
func newRouter(t *testing.T) (*gin.Engine, *memoryRepo, *memoryEvents) {
	t.Helper()

	repo := newMemoryRepo()
	events := &memoryEvents{}
	log := logger.NewNop()
	svc := email.NewService(repo, events, template.NewRenderer(staticTemplates{}), 5, log)

	h := &EmailHandler{emails: repo, events: events, emailer: svc, logger: log}

	r := gin.New()
	g := r.Group("/api/v2/emails")
	g.POST("", h.Send)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/events", h.Events)

	return r, repo, events
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	return rec
}

func TestSendQueuesEmail(t *testing.T) {
	r, repo, _ := newRouter(t)

	rec := doJSON(t, r, http.MethodPost, "/api/v2/emails", SendEmailRequest{
		TemplateID: "welcome",
		To:         "user@example.com",
		Data:       map[string]any{"UserName": "Вася"},
	})

	// 202, а не 200: письмо принято в очередь, но ещё не отправлено.
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rec.Code, rec.Body.String())
	}

	var res SendEmailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.EmailID == "" {
		t.Error("email_id is empty")
	}
	if res.Duplicate {
		t.Error("first request reported as duplicate")
	}

	id := uuid.MustParse(res.EmailID)
	stored, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("stored email not found: %v", err)
	}
	if !strings.Contains(stored.HTML, "Вася") {
		t.Error("body was not rendered at enqueue time")
	}
}

// Ошибки шаблона и адреса возвращаются синхронно: админка узнаёт о проблеме
// сразу, а не из молчаливого failed через час.
func TestSendRejectsBadRequests(t *testing.T) {
	cases := map[string]SendEmailRequest{
		"invalid email": {
			TemplateID: "welcome",
			To:         "not-an-email",
			Data:       map[string]any{"UserName": "Вася"},
		},
		"missing required data": {
			TemplateID: "welcome",
			To:         "user@example.com",
			Data:       map[string]any{},
		},
		"unknown template": {
			TemplateID: "nope",
			To:         "user@example.com",
		},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			r, repo, _ := newRouter(t)

			rec := doJSON(t, r, http.MethodPost, "/api/v2/emails", req)
			if rec.Code < 400 || rec.Code >= 500 {
				t.Errorf("status = %d, want a 4xx: %s", rec.Code, rec.Body.String())
			}

			items, _, _ := repo.List(context.Background(), email.Filter{})
			if len(items) != 0 {
				t.Errorf("%d emails queued despite the error", len(items))
			}
		})
	}
}

func TestSendValidatesPayload(t *testing.T) {
	r, repo, _ := newRouter(t)

	// template_id и to обязательны. Код 422 — конвенция проекта для ошибок
	// валидации (pkg/validation.AbortWithBindingError).
	rec := doJSON(t, r, http.MethodPost, "/api/v2/emails", map[string]any{"data": map[string]any{}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422: %s", rec.Code, rec.Body.String())
	}

	items, _, _ := repo.List(context.Background(), email.Filter{})
	if len(items) != 0 {
		t.Errorf("%d emails queued from an invalid payload", len(items))
	}
}

// Повторный запрос с тем же ключом возвращает уже созданное письмо: для
// вызывающего дубль — успех, а не ошибка.
func TestSendIsIdempotent(t *testing.T) {
	r, repo, _ := newRouter(t)

	req := SendEmailRequest{
		TemplateID:     "welcome",
		To:             "user@example.com",
		Data:           map[string]any{"UserName": "Вася"},
		IdempotencyKey: "manual:42",
	}

	first := doJSON(t, r, http.MethodPost, "/api/v2/emails", req)
	second := doJSON(t, r, http.MethodPost, "/api/v2/emails", req)

	if second.Code != http.StatusAccepted {
		t.Fatalf("second request status = %d, want 202", second.Code)
	}

	var a, b SendEmailResponse
	json.Unmarshal(first.Body.Bytes(), &a)
	json.Unmarshal(second.Body.Bytes(), &b)

	if !b.Duplicate {
		t.Error("second request not marked as duplicate")
	}
	if a.EmailID != b.EmailID {
		t.Errorf("ids differ: %s vs %s", a.EmailID, b.EmailID)
	}

	items, _, _ := repo.List(context.Background(), email.Filter{})
	if len(items) != 1 {
		t.Errorf("%d emails queued, want 1", len(items))
	}
}

func TestGetReturnsStatus(t *testing.T) {
	r, _, _ := newRouter(t)

	send := doJSON(t, r, http.MethodPost, "/api/v2/emails", SendEmailRequest{
		TemplateID: "welcome",
		To:         "user@example.com",
		Data:       map[string]any{"UserName": "Вася"},
	})
	var created SendEmailResponse
	json.Unmarshal(send.Body.Bytes(), &created)

	rec := doJSON(t, r, http.MethodGet, "/api/v2/emails/"+created.EmailID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var res EmailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Status != string(email.StatusQueued) {
		t.Errorf("status = %q, want queued", res.Status)
	}
	if res.TemplateID != "welcome" {
		t.Errorf("template = %q", res.TemplateID)
	}

	// Тело письма наружу не отдаётся: в нём персональные данные, а для
	// диагностики хватает статуса и причины отказа.
	if strings.Contains(rec.Body.String(), "<p>") {
		t.Error("response exposes the rendered body")
	}
}

func TestGetRejectsBadID(t *testing.T) {
	r, _, _ := newRouter(t)

	if rec := doJSON(t, r, http.MethodGet, "/api/v2/emails/not-a-uuid", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGetUnknownEmail(t *testing.T) {
	r, _, _ := newRouter(t)

	rec := doJSON(t, r, http.MethodGet, "/api/v2/emails/"+uuid.New().String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestListFiltersByStatus(t *testing.T) {
	r, _, _ := newRouter(t)

	for _, addr := range []string{"a@example.com", "b@example.com"} {
		doJSON(t, r, http.MethodPost, "/api/v2/emails", SendEmailRequest{
			TemplateID: "welcome",
			To:         addr,
			Data:       map[string]any{"UserName": "Вася"},
		})
	}

	rec := doJSON(t, r, http.MethodGet, "/api/v2/emails?status=queued", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var res ListEmailsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Total != 2 || len(res.Items) != 2 {
		t.Errorf("total=%d items=%d, want 2", res.Total, len(res.Items))
	}

	filtered := doJSON(t, r, http.MethodGet, "/api/v2/emails?to=a@example.com", nil)
	json.Unmarshal(filtered.Body.Bytes(), &res)
	if res.Total != 1 {
		t.Errorf("filtered total = %d, want 1", res.Total)
	}
}

// История переходов — то, по чему видно, сколько было попыток и почему письмо
// не ушло.
func TestEventsReturnsHistory(t *testing.T) {
	r, _, _ := newRouter(t)

	send := doJSON(t, r, http.MethodPost, "/api/v2/emails", SendEmailRequest{
		TemplateID: "welcome",
		To:         "user@example.com",
		Data:       map[string]any{"UserName": "Вася"},
	})
	var created SendEmailResponse
	json.Unmarshal(send.Body.Bytes(), &created)

	rec := doJSON(t, r, http.MethodGet, "/api/v2/emails/"+created.EmailID+"/events", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Items []EventResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].EventType != string(email.EventCreated) {
		t.Errorf("history = %+v, want one created event", res.Items)
	}
}
