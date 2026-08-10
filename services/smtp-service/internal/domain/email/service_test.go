package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/google/uuid"
)

// fakeRepo хранит очередь в памяти и воспроизводит единственное поведение,
// существенное для Enqueue: уникальность пары (dedup_key, dedup_bucket).
type fakeRepo struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*Email
	byKey  map[string]*Email
	create func(*Email) error // подмена для проверки ошибок БД
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:  make(map[uuid.UUID]*Email),
		byKey: make(map[string]*Email),
	}
}

// Повторяет уникальный индекс (dedup_key, dedup_bucket) из схемы: именно он,
// а не проверка перед вставкой, отсекает дубликаты в реальной БД.
func dedupIndex(e *Email) string {
	return fmt.Sprintf("%s\x00%d", e.DedupKey, e.DedupBucket)
}

func (r *fakeRepo) Create(_ context.Context, e *Email) (bool, error) {
	if r.create != nil {
		if err := r.create(e); err != nil {
			return false, err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := dedupIndex(e)
	if existing, ok := r.byKey[key]; ok {
		*e = *existing
		return false, nil
	}

	stored := *e
	r.byKey[key] = &stored
	r.byID[e.ID] = &stored

	return true, nil
}

func (r *fakeRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}

func (r *fakeRepo) get(id uuid.UUID) *Email {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byID[id]
}

func (r *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (*Email, error) {
	if e := r.get(id); e != nil {
		return e, nil
	}
	return nil, ErrNotFound(id.String())
}

func (r *fakeRepo) List(context.Context, Filter) ([]Email, int64, error) { return nil, 0, nil }
func (r *fakeRepo) Claim(context.Context, string, int, time.Duration) ([]Email, error) {
	return nil, nil
}
func (r *fakeRepo) MarkSent(context.Context, uuid.UUID, time.Time) error           { return nil }
func (r *fakeRepo) MarkFailed(context.Context, uuid.UUID, string) error            { return nil }
func (r *fakeRepo) Reschedule(context.Context, uuid.UUID, time.Time, string) error { return nil }
func (r *fakeRepo) ReleaseExpired(context.Context, time.Time) (int64, int64, error) {
	return 0, 0, nil
}
func (r *fakeRepo) CountSentSince(context.Context, time.Time) (int64, error) { return 0, nil }
func (r *fakeRepo) OldestQueuedAge(context.Context, time.Time) (time.Duration, error) {
	return 0, nil
}

type fakeEvents struct {
	mu     sync.Mutex
	events []Event
	err    error
}

func (f *fakeEvents) Append(_ context.Context, e *Event) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, *e)
	return nil
}

func (f *fakeEvents) ListByEmailID(_ context.Context, id uuid.UUID) ([]Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Event
	for _, e := range f.events {
		if e.EmailID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeEvents) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

type stubTemplates struct {
	templates map[string]template.Template
}

func (s stubTemplates) Get(_ context.Context, id string, _ *uint) (*template.Template, error) {
	t, ok := s.templates[id]
	if !ok {
		return nil, template.ErrNotFound(id)
	}
	return &t, nil
}

func newService(t *testing.T) (*Service, *fakeRepo, *fakeEvents) {
	t.Helper()

	repo := newFakeRepo()
	events := &fakeEvents{}
	provider := stubTemplates{templates: map[string]template.Template{
		"welcome": {
			ID:           "welcome",
			Version:      1,
			Subject:      "Добро пожаловать, {{.UserName}}!",
			HTML:         `<p>Привет, {{.UserName}}</p>`,
			RequiredData: []string{"UserName"},
		},
	}}

	return NewService(repo, events, template.NewRenderer(provider), 5, logger.NewNop()), repo, events
}

func welcomeInput() EnqueueInput {
	return EnqueueInput{
		TemplateID: "welcome",
		ToEmail:    "user@example.com",
		Data:       map[string]any{"UserName": "Вася"},
	}
}

func TestEnqueueRendersAndStores(t *testing.T) {
	svc, repo, events := newService(t)

	res, err := svc.Enqueue(context.Background(), welcomeInput())
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if res.Duplicate {
		t.Error("first enqueue reported as duplicate")
	}

	stored := repo.get(res.EmailID)
	if stored == nil {
		t.Fatal("email not stored")
	}

	// Тело рендерится при постановке: письмо неизменяемо с этого момента, и
	// повторная попытка отправит ровно согласованное содержимое.
	if !strings.Contains(stored.HTML, "Вася") {
		t.Errorf("body not rendered: %q", stored.HTML)
	}
	if !strings.Contains(stored.Subject, "Вася") {
		t.Errorf("subject not rendered: %q", stored.Subject)
	}
	if stored.Status != StatusQueued {
		t.Errorf("status = %s, want queued", stored.Status)
	}
	if stored.MaxAttempt != 5 {
		t.Errorf("max_attempt = %d, want 5", stored.MaxAttempt)
	}
	if stored.TemplateVersion != 1 {
		t.Errorf("template_version = %d, want 1", stored.TemplateVersion)
	}
	if events.count() != 1 {
		t.Errorf("recorded %d events, want 1 (created)", events.count())
	}
}

// Невалидный адрес не должен занимать воркера и накручивать попытки.
func TestEnqueueRejectsInvalidEmail(t *testing.T) {
	svc, repo, _ := newService(t)

	cases := map[string]string{
		"empty":         "",
		"blank":         "   ",
		"no at":         "not-an-email",
		"no domain":     "user@",
		"no local part": "@example.com",
		"local only":    "user@localhost",
		"garbage":       "a b c",
	}

	for name, address := range cases {
		t.Run(name, func(t *testing.T) {
			in := welcomeInput()
			in.ToEmail = address

			if _, err := svc.Enqueue(context.Background(), in); err == nil {
				t.Error("enqueue accepted invalid address")
			}
		})
	}

	if repo.count() != 0 {
		t.Errorf("%d emails stored, want 0", repo.count())
	}
}

func TestEnqueueNormalizesAddress(t *testing.T) {
	svc, repo, _ := newService(t)

	in := welcomeInput()
	in.ToEmail = "  Вася <user@example.com>  "

	res, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// В очередь кладётся голый адрес: иначе отображаемое имя попало бы в
	// dedup-ключ, и то же письмо с другим написанием имени прошло бы дважды.
	if got := repo.get(res.EmailID).ToEmail; got != "user@example.com" {
		t.Errorf("stored address = %q, want bare address", got)
	}
}

// Контракт шаблона проверяется синхронно: вызывающий узнаёт о пропущенном
// поле сразу, а не из молчаливого failed через час.
func TestEnqueueRejectsMissingTemplateData(t *testing.T) {
	svc, repo, _ := newService(t)

	in := welcomeInput()
	in.Data = map[string]any{}

	if _, err := svc.Enqueue(context.Background(), in); err == nil {
		t.Error("enqueue accepted data without required field")
	}
	if repo.count() != 0 {
		t.Errorf("%d emails stored, want 0", repo.count())
	}
}

func TestEnqueueRejectsUnknownTemplate(t *testing.T) {
	svc, repo, _ := newService(t)

	in := welcomeInput()
	in.TemplateID = "does-not-exist"

	if _, err := svc.Enqueue(context.Background(), in); err == nil {
		t.Error("enqueue accepted unknown template")
	}
	if repo.count() != 0 {
		t.Errorf("%d emails stored, want 0", repo.count())
	}
}

// Повторная доставка одного Kafka-события не должна давать второе письмо.
func TestEnqueueDeduplicatesByExplicitKey(t *testing.T) {
	svc, repo, events := newService(t)

	in := welcomeInput()
	in.IdempotencyKey = "user.registered:80"

	first, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("first enqueue: %v", err)
	}

	second, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("second enqueue: %v", err)
	}

	if !second.Duplicate {
		t.Error("duplicate not detected")
	}
	if second.EmailID != first.EmailID {
		t.Errorf("duplicate returned id %s, want existing %s", second.EmailID, first.EmailID)
	}
	if repo.count() != 1 {
		t.Errorf("%d emails stored, want 1", repo.count())
	}
	// История принадлежит письму, а не попытке его создать.
	if events.count() != 1 {
		t.Errorf("recorded %d events, want 1", events.count())
	}
}

// Без явного ключа дедупликация работает по содержимому.
func TestEnqueueDeduplicatesByContent(t *testing.T) {
	svc, repo, _ := newService(t)

	if _, err := svc.Enqueue(context.Background(), welcomeInput()); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	res, err := svc.Enqueue(context.Background(), welcomeInput())
	if err != nil {
		t.Fatalf("second enqueue: %v", err)
	}

	if !res.Duplicate {
		t.Error("identical email was not deduplicated")
	}
	if repo.count() != 1 {
		t.Errorf("%d emails stored, want 1", repo.count())
	}
}

// Обход map в Go неупорядочен: без канонической сериализации одно и то же
// письмо давало бы разные ключи и дедупликация не срабатывала бы.
func TestEnqueueContentKeyIsStable(t *testing.T) {
	base := map[string]any{"UserName": "Вася", "City": "Москва", "Age": 30}
	reordered := map[string]any{"Age": 30, "City": "Москва", "UserName": "Вася"}

	first := contentDedupKey("welcome", "user@example.com", base)
	for i := 0; i < 50; i++ {
		if got := contentDedupKey("welcome", "user@example.com", reordered); got != first {
			t.Fatalf("key changed between runs: %s != %s", got, first)
		}
	}
}

func TestEnqueueContentKeyDistinguishesInputs(t *testing.T) {
	key := func(template, to string, data map[string]any) string {
		return contentDedupKey(template, to, data)
	}
	base := key("welcome", "user@example.com", map[string]any{"UserName": "Вася"})

	cases := map[string]string{
		"other recipient": key("welcome", "other@example.com", map[string]any{"UserName": "Вася"}),
		"other template":  key("reset", "user@example.com", map[string]any{"UserName": "Вася"}),
		"other data":      key("welcome", "user@example.com", map[string]any{"UserName": "Петя"}),
		"extra field":     key("welcome", "user@example.com", map[string]any{"UserName": "Вася", "X": 1}),
	}

	for name, other := range cases {
		t.Run(name, func(t *testing.T) {
			if other == base {
				t.Error("different input produced the same dedup key")
			}
		})
	}
}

// Окна фиксированные, а не скользящие: bucket = unix / 300, то есть границы
// привязаны к абсолютному времени, а не к моменту первого письма.
//
// Практическое следствие: два одинаковых письма схлопнутся, только если попали
// в одно окно. Повторная доставка Kafka-события приходит через секунды и
// покрывается всегда; повтор через 4 минуты может уже не покрыться, если
// первое письмо было у самой границы окна.
//
// Это осознанный размен: скользящее окно требует чтения перед вставкой, а
// значит гонки между параллельными доставками — ровно того, ради защиты от
// чего дедупликация и существует.
func TestDedupBucketIsFixedWindow(t *testing.T) {
	// Привязываемся к началу окна, а не к time.Now(): иначе тест зависит от
	// того, в какой момент суток он запущен.
	windowStart := time.Unix((time.Now().Unix()/int64(DedupWindow.Seconds()))*int64(DedupWindow.Seconds()), 0)

	if dedupBucket(windowStart) != dedupBucket(windowStart.Add(DedupWindow-time.Second)) {
		t.Error("bucket changed inside a single window — duplicates would slip through")
	}

	if dedupBucket(windowStart) == dedupBucket(windowStart.Add(DedupWindow)) {
		t.Error("bucket unchanged across windows — legitimate resend would be dropped forever")
	}
}

// Разные письма одного пользователя не должны схлопываться.
func TestEnqueueDifferentRecipientsCoexist(t *testing.T) {
	svc, repo, _ := newService(t)

	for _, addr := range []string{"a@example.com", "b@example.com"} {
		in := welcomeInput()
		in.ToEmail = addr
		if _, err := svc.Enqueue(context.Background(), in); err != nil {
			t.Fatalf("enqueue %s: %v", addr, err)
		}
	}

	if repo.count() != 2 {
		t.Errorf("%d emails stored, want 2", repo.count())
	}
}

func TestEnqueueSchedulesForLater(t *testing.T) {
	svc, repo, _ := newService(t)

	later := time.Now().Add(2 * time.Hour)
	in := welcomeInput()
	in.ScheduledAt = &later

	res, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if got := repo.get(res.EmailID).ScheduledAt; !got.After(time.Now().Add(time.Hour)) {
		t.Errorf("scheduled_at = %v, want ~2h ahead", got)
	}
}

// Время в прошлом означает «отправить сейчас», а не «просрочено».
func TestEnqueueIgnoresPastSchedule(t *testing.T) {
	svc, repo, _ := newService(t)

	past := time.Now().Add(-time.Hour)
	in := welcomeInput()
	in.ScheduledAt = &past

	res, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if got := repo.get(res.EmailID).ScheduledAt; got.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("scheduled_at = %v, want now", got)
	}
}

func TestEnqueuePropagatesTraceAndPriority(t *testing.T) {
	svc, repo, _ := newService(t)

	in := welcomeInput()
	in.TraceID = "trace-abc"
	in.Priority = 10

	res, err := svc.Enqueue(context.Background(), in)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	stored := repo.get(res.EmailID)
	if stored.TraceID != "trace-abc" {
		t.Errorf("trace_id = %q", stored.TraceID)
	}
	if stored.Priority != 10 {
		t.Errorf("priority = %d, want 10", stored.Priority)
	}
}

func TestEnqueueReportsRepositoryFailure(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.create = func(*Email) error { return errors.New("database is down") }

	if _, err := svc.Enqueue(context.Background(), welcomeInput()); err == nil {
		t.Error("enqueue hid a database failure")
	}
}

// Журнал не важнее письма: сбой записи истории не должен отменять отправку.
func TestEnqueueSurvivesEventFailure(t *testing.T) {
	repo := newFakeRepo()
	events := &fakeEvents{err: errors.New("events table is gone")}
	provider := stubTemplates{templates: map[string]template.Template{
		"welcome": {ID: "welcome", Version: 1, Subject: "Привет", HTML: "<p>hi</p>"},
	}}
	svc := NewService(repo, events, template.NewRenderer(provider), 5, logger.NewNop())

	if _, err := svc.Enqueue(context.Background(), welcomeInput()); err != nil {
		t.Errorf("enqueue failed because of event logging: %v", err)
	}
	if repo.count() != 1 {
		t.Error("email was not stored")
	}
}

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"user@example.com":    "u***r@example.com",
		"ab@example.com":      "a*@example.com",
		"a@example.com":       "*@example.com",
		"TestUser@yandex.com": "T***r@yandex.com",
		"broken":              "***",
	}

	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := MaskEmail(input); got != want {
				t.Errorf("MaskEmail(%q) = %q, want %q", input, got, want)
			}
		})
	}
}
