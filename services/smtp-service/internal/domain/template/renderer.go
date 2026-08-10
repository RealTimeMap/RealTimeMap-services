package template

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"sync"
)

// Rendered — готовое к отправке содержимое письма.
type Rendered struct {
	TemplateID string
	Version    uint
	Subject    string
	HTML       string
}

// Renderer подставляет данные в шаблон.
//
// Живёт в домене и работает с любым Provider: и со встроенными шаблонами, и с
// теми, что в фазе 2 приедут из БД. Единственная точка рендера в сервисе —
// поэтому превью в админке гарантированно покажет то же, что уйдёт адресату.
type Renderer struct {
	provider Provider

	// cache хранит разобранные шаблоны. Разбор на каждое письмо — лишняя
	// работа на горячем пути; ключ включает версию, поэтому в фазе 2, когда
	// админка правит шаблон, новая версия попадёт в кэш как отдельная запись,
	// а не вытеснит старую под тем же ключом.
	mu    sync.RWMutex
	cache map[string]*compiled
}

type compiled struct {
	subject *template.Template
	body    *template.Template
}

func NewRenderer(provider Provider) *Renderer {
	return &Renderer{
		provider: provider,
		cache:    make(map[string]*compiled),
	}
}

// Render проверяет контракт шаблона и подставляет данные.
//
// Ошибки рендера терминальные: повторная попытка отправки их не исправит.
func (r *Renderer) Render(ctx context.Context, templateID string, version *uint, data map[string]any) (*Rendered, error) {
	tmpl, err := r.provider.Get(ctx, templateID, version)
	if err != nil {
		return nil, err
	}

	if err := validateData(tmpl, data); err != nil {
		return nil, err
	}

	c, err := r.compile(tmpl)
	if err != nil {
		return nil, err
	}

	subject, err := execute(c.subject, data)
	if err != nil {
		return nil, fmt.Errorf("render subject of %q: %w", tmpl.ID, err)
	}

	body, err := execute(c.body, data)
	if err != nil {
		return nil, fmt.Errorf("render body of %q: %w", tmpl.ID, err)
	}

	return &Rendered{
		TemplateID: tmpl.ID,
		Version:    tmpl.Version,
		Subject:    subject,
		HTML:       body,
	}, nil
}

// validateData требует, чтобы каждое объявленное шаблоном поле присутствовало
// и было непустым.
//
// Без этой проверки html/template молча подставляет пустую строку, и письмо
// уходит с «Здравствуйте, !» — необратимо, в отличие от опечатки на странице.
func validateData(tmpl *Template, data map[string]any) error {
	for _, field := range tmpl.RequiredData {
		value, ok := data[field]
		if !ok || value == nil {
			return ErrMissingData(field)
		}
		if s, isString := value.(string); isString && strings.TrimSpace(s) == "" {
			return ErrMissingData(field)
		}
	}
	return nil
}

// compile возвращает разобранный шаблон, разбирая его при первом обращении.
func (r *Renderer) compile(tmpl *Template) (*compiled, error) {
	key := fmt.Sprintf("%s@%d", tmpl.ID, tmpl.Version)

	r.mu.RLock()
	c, ok := r.cache[key]
	r.mu.RUnlock()
	if ok {
		return c, nil
	}

	// html/template, а не text/template: он экранирует подставляемые данные с
	// учётом контекста, поэтому имя пользователя вида "<script>" попадёт в
	// письмо как текст, а не как разметка.
	subject, err := template.New(key + ":subject").Parse(tmpl.Subject)
	if err != nil {
		return nil, fmt.Errorf("parse subject of %q: %w", tmpl.ID, err)
	}

	body, err := template.New(key).Parse(tmpl.HTML)
	if err != nil {
		return nil, fmt.Errorf("parse body of %q: %w", tmpl.ID, err)
	}

	c = &compiled{subject: subject, body: body}

	r.mu.Lock()
	r.cache[key] = c
	r.mu.Unlock()

	return c, nil
}

func execute(tmpl *template.Template, data map[string]any) (string, error) {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
