package template

import (
	"context"
	"strings"
	"testing"

	domain "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
)

// Проверяет реальные встроенные шаблоны, а не подставные: смысл в том, чтобы
// сломанное письмо не доехало до продакшена.
func TestProviderLoadsEmbeddedTemplates(t *testing.T) {
	p, err := NewProvider()
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	ids := p.IDs()
	if len(ids) == 0 {
		t.Fatal("no templates registered")
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			tmpl, err := p.Get(context.Background(), id, nil)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if strings.TrimSpace(tmpl.HTML) == "" {
				t.Error("template body is empty")
			}
			if strings.TrimSpace(tmpl.Subject) == "" {
				t.Error("subject is empty")
			}
			if tmpl.Version == 0 {
				t.Error("version is zero")
			}
		})
	}
}

func TestProviderUnknownTemplate(t *testing.T) {
	p, err := NewProvider()
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	if _, err := p.Get(context.Background(), "does-not-exist", nil); err == nil {
		t.Error("unknown template returned no error")
	}
}

// Контракт RequiredData существует, чтобы письмо не ушло с пустотой вместо
// имени. Если поле объявлено, но в шаблоне не используется — контракт врёт
// продюсерам; если используется, но не объявлено — проверка его пропустит.
func TestWelcomeContractMatchesTemplate(t *testing.T) {
	p, err := NewProvider()
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	tmpl, err := p.Get(context.Background(), "welcome", nil)
	if err != nil {
		t.Fatalf("get welcome: %v", err)
	}

	if len(tmpl.RequiredData) == 0 {
		t.Fatal("welcome declares no required data")
	}

	for _, field := range tmpl.RequiredData {
		if !usesField(tmpl.HTML, field) && !usesField(tmpl.Subject, field) {
			t.Errorf("field %q is declared required but never used", field)
		}
	}
}

// Сквозная проверка: встроенный шаблон проходит через доменный рендерер и даёт
// пригодное к отправке письмо.
func TestWelcomeRendersEndToEnd(t *testing.T) {
	p, err := NewProvider()
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	got, err := domain.NewRenderer(p).Render(context.Background(), "welcome", nil, map[string]any{
		"username":                "Вася",
		"mapUrl":                  "https://realtimemap.ru/map",
		"friendsUrl":              "https://realtimemap.ru/profile/friends",
		"notificationSettingsUrl": "https://realtimemap.ru/settings/notifications",
		"unsubscribeUrl":          "https://realtimemap.ru/unsubscribe",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(got.Subject, "Вася") {
		t.Errorf("subject has no user name: %q", got.Subject)
	}
	if !strings.Contains(got.HTML, "Вася") {
		t.Error("body has no user name")
	}
	if strings.Contains(got.HTML, "{{") {
		t.Error("unrendered placeholders left in body")
	}

	// Почтовые клиенты не поддерживают современный CSS-layout, поэтому вёрстка
	// обязана быть табличной.
	if !strings.Contains(got.HTML, "<table") {
		t.Error("body is not table-based, will break in Outlook")
	}

	// Чужой шаблонный синтаксис (Jinja/Django) Go не обрабатывает: такой текст
	// уедет получателю как есть.
	if strings.Contains(got.HTML, "{%") {
		t.Error("body contains Jinja-style tags that Go templates ignore")
	}

	if _, err := domain.NewRenderer(p).Render(context.Background(), "welcome", nil, map[string]any{}); err == nil {
		t.Error("welcome rendered without UserName")
	}
}

// usesField ищет обращение к полю в шаблоне.
//
// Go допускает пробелы внутри действия, и в шаблонах проекта принята форма
// "{{ .field }}". Сравнение с одной жёсткой строкой объявляло бы используемое
// поле неиспользуемым, поэтому проверяются обе формы.
func usesField(tmpl, field string) bool {
	return strings.Contains(tmpl, "{{."+field+"}}") ||
		strings.Contains(tmpl, "{{ ."+field+" }}")
}
