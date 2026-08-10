package postgres

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/google/uuid"
)

// Сквозная проверка постановки в очередь на настоящей БД. Юнит-тесты Enqueue
// работают с фейковым репозиторием, который лишь имитирует уникальный индекс —
// здесь проверяется, что дедупликация опирается на реальное ограничение схемы.

type fixedProvider struct{}

func (fixedProvider) Get(_ context.Context, id string, _ *uint) (*template.Template, error) {
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

func setupEmailService(t *testing.T) (*email.Service, email.Repository, string) {
	t.Helper()

	repo, events, _, run := setupRepo(t)
	log := logger.NewNop()
	svc := email.NewService(repo, events, template.NewRenderer(fixedProvider{}), 5, log)

	return svc, repo, run
}

func TestIntegrationEnqueueStoresRenderedEmail(t *testing.T) {
	svc, repo, run := setupEmailService(t)
	ctx := context.Background()

	res, err := svc.Enqueue(ctx, email.EnqueueInput{
		TemplateID:     "welcome",
		ToEmail:        "queue-test@example.com",
		Data:           map[string]any{"UserName": "Вася"},
		IdempotencyKey: "test:" + run + ":enqueue",
		TraceID:        "trace-" + run,
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	stored, err := repo.GetByID(ctx, res.EmailID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if stored.Status != email.StatusQueued {
		t.Errorf("status = %s, want queued", stored.Status)
	}
	// Тело хранится отрендеренным: воркеру шаблоны уже не нужны.
	if !strings.Contains(stored.HTML, "Вася") {
		t.Errorf("body not rendered in storage: %q", stored.HTML)
	}
	if !strings.Contains(stored.Subject, "Вася") {
		t.Errorf("subject not rendered in storage: %q", stored.Subject)
	}
	if stored.TraceID != "trace-"+run {
		t.Errorf("trace_id = %q", stored.TraceID)
	}
}

// Главное: дубликат отсекается ограничением БД, а не проверкой в коде —
// иначе две параллельные доставки одного события прошли бы обе.
func TestIntegrationEnqueueConcurrentDeduplication(t *testing.T) {
	svc, _, run := setupEmailService(t)
	ctx := context.Background()

	const attempts = 8
	ids := make([]uuid.UUID, attempts)
	duplicates := make([]bool, attempts)
	errs := make([]error, attempts)

	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := svc.Enqueue(ctx, email.EnqueueInput{
				TemplateID:     "welcome",
				ToEmail:        "queue-test@example.com",
				Data:           map[string]any{"UserName": "Вася"},
				IdempotencyKey: "test:" + run + ":concurrent",
			})
			if err != nil {
				errs[i] = err
				return
			}
			ids[i] = res.EmailID
			duplicates[i] = res.Duplicate
		}(i)
	}
	wg.Wait()

	created := 0
	for i := 0; i < attempts; i++ {
		if errs[i] != nil {
			t.Errorf("attempt %d failed: %v", i, errs[i])
			continue
		}
		if !duplicates[i] {
			created++
		}
		// Все вызовы обязаны вернуть идентификатор одного и того же письма,
		// иначе вызывающий не сможет отследить его статус.
		if ids[i] != ids[0] {
			t.Errorf("attempt %d returned id %s, want %s", i, ids[i], ids[0])
		}
	}

	if created != 1 {
		t.Errorf("%d emails created out of %d concurrent attempts, want exactly 1", created, attempts)
	}
}

// Без явного ключа дедупликация опирается на хеш содержимого — он должен
// оставаться стабильным между вызовами, несмотря на неупорядоченность map.
func TestIntegrationEnqueueContentDeduplication(t *testing.T) {
	svc, _, run := setupEmailService(t)
	ctx := context.Background()

	input := email.EnqueueInput{
		TemplateID: "welcome",
		ToEmail:    fmt.Sprintf("content-%s@example.com", run),
		Data:       map[string]any{"UserName": "Вася", "City": "Москва", "Age": 30},
	}

	first, err := svc.Enqueue(ctx, input)
	if err != nil {
		t.Fatalf("first enqueue: %v", err)
	}

	second, err := svc.Enqueue(ctx, input)
	if err != nil {
		t.Fatalf("second enqueue: %v", err)
	}

	if !second.Duplicate {
		t.Error("identical email was stored twice")
	}
	if second.EmailID != first.EmailID {
		t.Errorf("duplicate id %s != original %s", second.EmailID, first.EmailID)
	}
}

func TestIntegrationEnqueueRejectsBadInput(t *testing.T) {
	svc, _, run := setupEmailService(t)
	ctx := context.Background()

	cases := map[string]email.EnqueueInput{
		"invalid email": {
			TemplateID:     "welcome",
			ToEmail:        "not-an-email",
			Data:           map[string]any{"UserName": "Вася"},
			IdempotencyKey: "test:" + run + ":bad-email",
		},
		"missing required data": {
			TemplateID:     "welcome",
			ToEmail:        "queue-test@example.com",
			Data:           map[string]any{},
			IdempotencyKey: "test:" + run + ":bad-data",
		},
		"unknown template": {
			TemplateID:     "nope",
			ToEmail:        "queue-test@example.com",
			IdempotencyKey: "test:" + run + ":bad-template",
		},
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			// Отказ синхронный: вызывающий узнаёт о проблеме сразу, и заведомо
			// провальное письмо не занимает воркера.
			if _, err := svc.Enqueue(ctx, in); err == nil {
				t.Error("enqueue accepted invalid input")
			}
		})
	}
}
