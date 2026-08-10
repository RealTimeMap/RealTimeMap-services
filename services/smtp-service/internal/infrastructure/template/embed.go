// Package template содержит embed-реализацию источника шаблонов.
//
// Шаблоны лежат рядом с кодом и катятся вместе с деплоем: правка письма
// проходит через ревью и git blame. В фазе 2 источником станет Postgres с
// редактированием через админку, а этот пакет останется способом засеять
// пустое окружение дефолтным набором.
package template

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"

	domain "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// FS отдаёт встроенные шаблоны наружу. Нужен фазе 2: сидинг наполняет пустую
// БД тем же набором, что зашит в бинарь.
func FS() fs.FS { return templateFS }

// meta — то, что нельзя выразить в самом HTML: тема письма и контракт на
// входные данные.
type meta struct {
	subject      string
	requiredData []string
}

// registry — набор писем, которые сервис умеет отправлять.
//
// Отдельная таблица, а не соглашение об именах файлов: RequiredData — это
// контракт с сервисами-продюсерами, и он должен быть виден в одном месте.
var registry = map[string]meta{
	"welcome": {
		subject:      "Добро пожаловать в RealTimeMap, {{.username}}!",
		requiredData: []string{"username"},
	},
}

// Provider отдаёт шаблоны, встроенные в бинарь.
type Provider struct {
	templates map[string]domain.Template
}

// NewProvider читает встроенные шаблоны и проверяет, что они разбираются.
//
// Проверка при старте, а не при отправке: бинарь со сломанным письмом не
// должен подниматься — иначе поломка всплывёт на первом же адресате.
// Разобранные шаблоны кэширует Renderer, здесь они только валидируются.
func NewProvider() (*Provider, error) {
	p := &Provider{
		templates: make(map[string]domain.Template, len(registry)),
	}

	for id, m := range registry {
		file := path.Join("templates", id+".html")

		raw, err := templateFS.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read template %q: %w", id, err)
		}

		if _, err := template.New(id).Parse(string(raw)); err != nil {
			return nil, fmt.Errorf("parse template %q: %w", id, err)
		}
		if _, err := template.New(id + ":subject").Parse(m.subject); err != nil {
			return nil, fmt.Errorf("parse subject of %q: %w", id, err)
		}

		p.templates[id] = domain.Template{
			ID:           id,
			Version:      1,
			Subject:      m.subject,
			HTML:         string(raw),
			RequiredData: m.requiredData,
		}
	}

	return p, nil
}

// Get возвращает описание шаблона.
//
// version игнорируется: встроенные шаблоны версионируются вместе с бинарём.
func (p *Provider) Get(_ context.Context, templateID string, _ *uint) (*domain.Template, error) {
	t, ok := p.templates[templateID]
	if !ok {
		return nil, domain.ErrNotFound(templateID)
	}
	return &t, nil
}

// IDs перечисляет доступные шаблоны. Полезно для админки и диагностики.
func (p *Provider) IDs() []string {
	ids := make([]string, 0, len(p.templates))
	for id := range p.templates {
		ids = append(ids, id)
	}
	return ids
}
