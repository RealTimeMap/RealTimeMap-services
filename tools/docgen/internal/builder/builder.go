package builder

import (
	"docgen/internal/models"
	"docgen/internal/scaner"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Builder собирает итоговый JSON из просканированных сервисов.
type Builder struct {
	services []scaner.ServiceDocs

	// Все модели всех сервисов: "service-name" → { "ModelName" → Model }
	serviceModels map[string]map[string]models.Model

	// Shared-модели из docs/shared/models.yaml
	sharedModels map[string]models.Model
}

func New(services []scaner.ServiceDocs) *Builder {
	return &Builder{
		services:      services,
		serviceModels: make(map[string]map[string]models.Model),
	}
}

// Build выполняет полную сборку: загрузка → резолв $ref → формирование output.
func (b *Builder) Build(sharedModelsPath string) (*models.BuildOutput, error) {
	// 1. Загрузить shared-модели
	if err := b.loadSharedModels(sharedModelsPath); err != nil {
		return nil, fmt.Errorf("shared model: %w", err)
	}

	// 2. Загрузить модели всех сервисов (для межсервисных $ref)
	for _, svc := range b.services {
		if svc.ModelsPath == "" {
			continue
		}
		m, err := loadModelsFile(svc.ModelsPath)
		if err != nil {
			return nil, fmt.Errorf("сервис %s model: %w", svc.Name, err)
		}
		b.serviceModels[svc.Name] = m
	}

	// 3. Собрать каждый сервис
	output := &models.BuildOutput{}

	for _, svc := range b.services {
		svcOut, err := b.buildService(svc)
		if err != nil {
			return nil, fmt.Errorf("сервис %s: %w", svc.Name, err)
		}
		output.Services = append(output.Services, *svcOut)
	}

	return output, nil
}

func (b *Builder) buildService(svc scaner.ServiceDocs) (*models.ServiceOutput, error) {
	out := &models.ServiceOutput{}

	// Meta
	if svc.MetaPath != "" {
		meta, err := loadMetaFile(svc.MetaPath)
		if err != nil {
			return nil, fmt.Errorf("meta: %w", err)
		}
		out.Name = meta.Name
		out.DisplayName = meta.DisplayName
		out.Description = meta.Description
		out.Version = meta.Version
		out.Struct = meta.Struct
		out.Infrastructure = meta.Infrastructure
		out.Tags = meta.Tags
	}

	// Models — загрузить и зарезолвить $ref
	if svc.ModelsPath != "" {
		m := b.serviceModels[svc.Name]
		resolved, err := b.resolveModels(m)
		if err != nil {
			return nil, fmt.Errorf("model resolve: %w", err)
		}
		out.Models = resolved
	}

	// API
	if svc.APIPath != "" {
		localModels := b.serviceModels[svc.Name]
		api, err := b.loadAndResolveAPI(svc.APIPath, localModels)
		if err != nil {
			return nil, fmt.Errorf("api: %w", err)
		}
		out.API = api
	}

	return out, nil
}

// resolveModels резолвит все $ref внутри карты моделей.
func (b *Builder) resolveModels(m map[string]models.Model) (map[string]models.Model, error) {
	result := make(map[string]models.Model, len(m))

	for name, model := range m {
		resolved, err := b.resolveModel(model, m)
		if err != nil {
			return nil, fmt.Errorf("модель %s: %w", name, err)
		}
		result[name] = resolved
	}

	return result, nil
}

func (b *Builder) resolveModel(model models.Model, localModels map[string]models.Model) (models.Model, error) {
	resolved := models.Model{
		Description: model.Description,
		Fields:      make(map[string]models.Field, len(model.Fields)),
	}

	for fieldName, field := range model.Fields {
		rf, err := b.resolveField(field, localModels)
		if err != nil {
			return resolved, fmt.Errorf("поле %s: %w", fieldName, err)
		}
		resolved.Fields[fieldName] = rf
	}

	return resolved, nil
}

func (b *Builder) resolveField(f models.Field, localModels map[string]models.Model) (models.Field, error) {
	// Резолвим items рекурсивно
	if f.Items != nil {
		resolved, err := b.resolveField(*f.Items, localModels)
		if err != nil {
			return f, fmt.Errorf("items: %w", err)
		}
		f.Items = &resolved
	}

	// Если нет $ref — возвращаем как есть
	if f.Ref == "" {
		return f, nil
	}

	// Резолвим $ref → находим модель и подставляем как inline object
	refModel, refModels, err := b.lookupRefWithContext(f.Ref, localModels)
	if err != nil {
		return f, err
	}

	// Рекурсивно резолвим поля ссылочной модели в её контексте
	resolvedModel, err := b.resolveModel(*refModel, refModels)
	if err != nil {
		return f, fmt.Errorf("$ref %s: %w", f.Ref, err)
	}

	// Заменяем $ref на inline-описание: type=object + fields
	resolved := models.Field{
		Type:        "object",
		Description: f.Description,
		Nullable:    f.Nullable,
		Required:    f.Required,
		Fields:      resolvedModel.Fields,
	}

	if resolved.Description == "" {
		resolved.Description = refModel.Description
	}

	return resolved, nil
}

// lookupRef находит модель по ссылке (использует localModels как контекст резолва).
func (b *Builder) lookupRef(ref string, localModels map[string]models.Model) (*models.Model, error) {
	m, _, err := b.lookupRefWithContext(ref, localModels)
	return m, err
}

// lookupRefWithContext находит модель по ссылке и возвращает контекст (набор моделей),
// в котором нужно резолвить внутренние $ref найденной модели.
func (b *Builder) lookupRefWithContext(ref string, localModels map[string]models.Model) (*models.Model, map[string]models.Model, error) {
	// Межсервисная ссылка: service-name.ModelName
	if parts := strings.SplitN(ref, ".", 2); len(parts) == 2 {
		source := parts[0]
		modelName := parts[1]

		// shared.ModelName — резолвить внутри shared контекста
		if source == "shared" {
			if m, ok := b.sharedModels[modelName]; ok {
				return &m, b.sharedModels, nil
			}
			return nil, nil, fmt.Errorf("$ref %q: модель %q не найдена в shared", ref, modelName)
		}

		// service-name.ModelName
		if svcModels, ok := b.serviceModels[source]; ok {
			if m, ok := svcModels[modelName]; ok {
				return &m, svcModels, nil
			}
			return nil, nil, fmt.Errorf("$ref %q: модель %q не найдена в сервисе %q", ref, modelName, source)
		}

		return nil, nil, fmt.Errorf("$ref %q: сервис %q не найден", ref, source)
	}

	// Локальная ссылка: ModelName
	if m, ok := localModels[ref]; ok {
		return &m, localModels, nil
	}

	return nil, nil, fmt.Errorf("$ref %q: модель не найдена", ref)
}

// loadAndResolveAPI читает api.yaml и резолвит все $ref.
func (b *Builder) loadAndResolveAPI(path string, localModels map[string]models.Model) (*models.API, error) {
	api, err := loadAPIFile(path)
	if err != nil {
		return nil, err
	}

	if localModels == nil {
		localModels = make(map[string]models.Model)
	}

	if err := b.resolveAPI(api, localModels); err != nil {
		return nil, err
	}

	return api, nil
}

func loadAPIFile(path string) (*models.API, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var api models.API
	if err := yaml.Unmarshal(data, &api); err != nil {
		return nil, err
	}
	return &api, nil
}

// resolveAPI резолвит все $ref в структуре API.
func (b *Builder) resolveAPI(api *models.API, localModels map[string]models.Model) error {
	if api.HTTP != nil {
		if err := b.resolveHTTP(api.HTTP, localModels); err != nil {
			return fmt.Errorf("http: %w", err)
		}
	}

	if api.SocketIO != nil {
		if err := b.resolveSocketIO(api.SocketIO, localModels); err != nil {
			return fmt.Errorf("socketio: %w", err)
		}
	}

	return nil
}

func (b *Builder) resolveHTTP(http *models.HTTPSpec, localModels map[string]models.Model) error {
	for gi := range http.Groups {
		for ei := range http.Groups[gi].Endpoints {
			ep := &http.Groups[gi].Endpoints[ei]

			if ep.Request != nil {
				resolved, err := b.resolveBodyFields(ep.Request.Body, localModels)
				if err != nil {
					return fmt.Errorf("group[%d].endpoint[%d].request.body: %w", gi, ei, err)
				}
				ep.Request.Body = resolved
			}

			for code, resp := range ep.Responses {
				resolved, err := b.resolveBodyFields(resp.Body, localModels)
				if err != nil {
					return fmt.Errorf("group[%d].endpoint[%d].responses[%s].body: %w", gi, ei, code, err)
				}
				resp.Body = resolved
				ep.Responses[code] = resp
			}
		}
	}
	return nil
}

func (b *Builder) resolveSocketIO(sio *models.SocketIOSpec, localModels map[string]models.Model) error {
	for i := range sio.Events {
		ev := &sio.Events[i]

		if ev.Request != nil {
			resolved, err := b.resolveBodyFields(ev.Request, localModels)
			if err != nil {
				return fmt.Errorf("events[%d].request: %w", i, err)
			}
			ev.Request = resolved
		}

		if ev.Payload != nil {
			resolved, err := b.resolveBodyFields(ev.Payload, localModels)
			if err != nil {
				return fmt.Errorf("events[%d].payload: %w", i, err)
			}
			ev.Payload = resolved
		}

		if ev.Response != nil && ev.Response.Payload != nil {
			resolved, err := b.resolveBodyFields(ev.Response.Payload, localModels)
			if err != nil {
				return fmt.Errorf("events[%d].response.payload: %w", i, err)
			}
			ev.Response.Payload = resolved
		}
	}
	return nil
}

// resolveBodyFields резолвит BodyFields:
// - "$self" — одиночное Field (type: array, $ref и т.д.) — резолвится через resolveField.
// - Инлайн-карта — каждое поле резолвится отдельно.
func (b *Builder) resolveBodyFields(body models.BodyFields, localModels map[string]models.Model) (models.BodyFields, error) {
	if len(body) == 0 {
		return body, nil
	}

	// Одиночное поле (было распарсено как $self)
	if selfField, ok := body["$self"]; ok && len(body) == 1 {
		resolved, err := b.resolveField(selfField, localModels)
		if err != nil {
			return nil, fmt.Errorf("$self: %w", err)
		}
		return models.BodyFields{"$self": resolved}, nil
	}

	// Инлайн-карта: резолвим каждое поле
	result := make(models.BodyFields, len(body))
	for name, field := range body {
		resolved, err := b.resolveField(field, localModels)
		if err != nil {
			return nil, fmt.Errorf("поле %s: %w", name, err)
		}
		result[name] = resolved
	}
	return result, nil
}

func loadModelsFile(path string) (map[string]models.Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]models.Model
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func loadMetaFile(path string) (*models.Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta models.Meta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (b *Builder) loadSharedModels(path string) error {
	if path == "" {
		b.sharedModels = make(map[string]models.Model)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Shared не обязателен
		b.sharedModels = make(map[string]models.Model)
		return nil
	}

	var m map[string]models.Model
	if err := yaml.Unmarshal(data, &m); err != nil {
		return err
	}
	b.sharedModels = m
	return nil
}
