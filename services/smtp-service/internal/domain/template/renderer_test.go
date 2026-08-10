package template

import (
	"context"
	"strings"
	"testing"
)

// stubProvider отдаёт заранее заданные шаблоны: тесты рендера не должны
// зависеть от того, что лежит в embed.FS.
type stubProvider struct {
	templates map[string]Template
	calls     int
}

func (s *stubProvider) Get(_ context.Context, id string, _ *uint) (*Template, error) {
	s.calls++
	t, ok := s.templates[id]
	if !ok {
		return nil, ErrNotFound(id)
	}
	return &t, nil
}

func newStub(t Template) *stubProvider {
	return &stubProvider{templates: map[string]Template{t.ID: t}}
}

func TestRenderSubstitutesData(t *testing.T) {
	provider := newStub(Template{
		ID:           "welcome",
		Version:      1,
		Subject:      "Добро пожаловать, {{.UserName}}!",
		HTML:         `<p>Здравствуйте, {{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	})

	got, err := NewRenderer(provider).Render(context.Background(), "welcome", nil, map[string]any{
		"UserName": "Вася",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if got.Subject != "Добро пожаловать, Вася!" {
		t.Errorf("subject = %q", got.Subject)
	}
	if !strings.Contains(got.HTML, "Здравствуйте, Вася") {
		t.Errorf("html = %q", got.HTML)
	}
	if got.TemplateID != "welcome" || got.Version != 1 {
		t.Errorf("template id/version = %s@%d", got.TemplateID, got.Version)
	}
}

// Отсутствующее поле html/template подставляет как пустую строку. Для письма
// это недопустимо: «Здравствуйте, !» уходит адресату необратимо, в отличие от
// опечатки на странице, которую можно поправить.
func TestRenderRejectsMissingRequiredData(t *testing.T) {
	provider := newStub(Template{
		ID:           "welcome",
		Subject:      "Привет, {{.UserName}}",
		HTML:         `<p>{{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	})
	renderer := NewRenderer(provider)

	cases := map[string]map[string]any{
		"field absent": {},
		"field nil":    {"UserName": nil},
		"field empty":  {"UserName": ""},
		"field blank":  {"UserName": "   "},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := renderer.Render(context.Background(), "welcome", nil, data); err == nil {
				t.Error("render succeeded, want error about missing UserName")
			}
		})
	}
}

func TestRenderAllowsOptionalDataBeyondContract(t *testing.T) {
	provider := newStub(Template{
		ID:           "welcome",
		Subject:      "Привет",
		HTML:         `<p>{{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	})

	_, err := NewRenderer(provider).Render(context.Background(), "welcome", nil, map[string]any{
		"UserName": "Вася",
		"Extra":    "не объявлено, но и не мешает",
	})
	if err != nil {
		t.Errorf("render: %v", err)
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	provider := newStub(Template{ID: "welcome", Subject: "x", HTML: "x"})

	if _, err := NewRenderer(provider).Render(context.Background(), "nope", nil, nil); err == nil {
		t.Error("render of unknown template succeeded")
	}
}

// html/template экранирует по контексту: имя пользователя не должно
// превращаться в разметку письма.
func TestRenderEscapesUserData(t *testing.T) {
	provider := newStub(Template{
		ID:           "welcome",
		Subject:      "Привет, {{.UserName}}",
		HTML:         `<p>{{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	})

	got, err := NewRenderer(provider).Render(context.Background(), "welcome", nil, map[string]any{
		"UserName": `<script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(got.HTML, "<script>") {
		t.Errorf("script tag survived escaping: %q", got.HTML)
	}
}

func TestRenderBrokenTemplateFails(t *testing.T) {
	provider := newStub(Template{
		ID:      "broken",
		Subject: "ok",
		HTML:    `<p>{{.Unclosed</p>`,
	})

	if _, err := NewRenderer(provider).Render(context.Background(), "broken", nil, map[string]any{}); err == nil {
		t.Error("broken template rendered without error")
	}
}

// Разбор шаблона на каждое письмо — лишняя работа на горячем пути; провайдер
// при этом опрашивается всегда, потому что в фазе 2 он станет источником
// правок из админки.
func TestRenderCachesCompiledTemplates(t *testing.T) {
	provider := newStub(Template{
		ID:           "welcome",
		Version:      1,
		Subject:      "Привет, {{.UserName}}",
		HTML:         `<p>{{.UserName}}</p>`,
		RequiredData: []string{"UserName"},
	})
	renderer := NewRenderer(provider)

	for i := 0; i < 3; i++ {
		if _, err := renderer.Render(context.Background(), "welcome", nil, map[string]any{"UserName": "Вася"}); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}

	renderer.mu.RLock()
	cached := len(renderer.cache)
	renderer.mu.RUnlock()

	if cached != 1 {
		t.Errorf("cache holds %d entries, want 1", cached)
	}
	if provider.calls != 3 {
		t.Errorf("provider called %d times, want 3 (cache must not hide template updates)", provider.calls)
	}
}

func TestRenderCachesVersionsSeparately(t *testing.T) {
	provider := &stubProvider{templates: map[string]Template{}}
	renderer := NewRenderer(provider)

	for _, v := range []uint{1, 2} {
		provider.templates["welcome"] = Template{
			ID:      "welcome",
			Version: v,
			Subject: "Привет",
			HTML:    `<p>version {{.V}}</p>`,
		}
		if _, err := renderer.Render(context.Background(), "welcome", nil, map[string]any{"V": v}); err != nil {
			t.Fatalf("render v%d: %v", v, err)
		}
	}

	renderer.mu.RLock()
	cached := len(renderer.cache)
	renderer.mu.RUnlock()

	// Иначе правка шаблона в фазе 2 не доедет до отправки: новая версия
	// вытеснялась бы в кэше под тем же ключом.
	if cached != 2 {
		t.Errorf("cache holds %d entries, want 2 (one per version)", cached)
	}
}
