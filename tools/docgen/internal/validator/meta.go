package validator

import (
	"docgen/internal/models"
	"fmt"
	"regexp"
)

var (
	// name: kebab-case, например "mark-service", "auth-gateway"
	serviceNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

	// version: semver (упрощённый), например "1.0.0", "2.1.3"
	semverRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

	availableLanguages = []string{
		"go", "golang", "python", "java", "typescript", "rust", "csharp",
	}

	availableProtocols = []string{
		"http", "grpc", "graphql", "websocket", "socketio", "kafka", "nats", "amqp",
	}

	availableInfraTypes = []string{
		"postgres", "mysql", "mongodb", "redis", "kafka", "nats",
		"rabbitmq", "s3", "local-storage", "elasticsearch",
	}
)

// ValidateMeta валидирует структуру Meta (содержимое meta.yaml).
func ValidateMeta(meta models.Meta) []error {
	var errs []error

	// Обязательные поля
	if meta.Name == "" {
		errs = append(errs, fmt.Errorf("meta: поле name обязательно"))
	} else if !serviceNameRe.MatchString(meta.Name) {
		errs = append(errs, fmt.Errorf("meta: name %q — ожидается kebab-case (например mark-service)", meta.Name))
	}

	if meta.DisplayName == "" {
		errs = append(errs, fmt.Errorf("meta: поле displayName обязательно"))
	}

	if meta.Description == "" {
		errs = append(errs, fmt.Errorf("meta: поле description обязательно"))
	}

	if meta.Version == "" {
		errs = append(errs, fmt.Errorf("meta: поле version обязательно"))
	} else if !semverRe.MatchString(meta.Version) {
		errs = append(errs, fmt.Errorf("meta: version %q — ожидается формат semver (например 1.0.0)", meta.Version))
	}

	// Struct
	errs = append(errs, validateStruct(meta.Struct)...)

	// Infrastructure
	errs = append(errs, validateInfrastructure(meta.Infrastructure)...)

	return errs
}

func validateStruct(s models.Struct) []error {
	var errs []error

	if s.Language == "" {
		errs = append(errs, fmt.Errorf("meta.struct: поле language обязательно"))
	} else if !contains(availableLanguages, s.Language) {
		errs = append(errs, fmt.Errorf(
			"meta.struct: неизвестный language %q (допустимые: %v)",
			s.Language, availableLanguages,
		))
	}

	if len(s.Protocols) == 0 {
		errs = append(errs, fmt.Errorf("meta.struct: необходимо указать хотя бы один protocol"))
	}
	for _, p := range s.Protocols {
		if !contains(availableProtocols, p) {
			errs = append(errs, fmt.Errorf(
				"meta.struct: неизвестный protocol %q (допустимые: %v)",
				p, availableProtocols,
			))
		}
	}

	return errs
}

func validateInfrastructure(infra models.Infrastructure) []error {
	var errs []error

	if len(infra.Type) == 0 {
		errs = append(errs, fmt.Errorf("meta.infrastructure: необходимо указать хотя бы один type"))
	}

	for _, t := range infra.Type {
		if !contains(availableInfraTypes, t) {
			errs = append(errs, fmt.Errorf(
				"meta.infrastructure: неизвестный type %q (допустимые: %v)",
				t, availableInfraTypes,
			))
		}
	}

	if len(infra.Type) != len(infra.Description) {
		errs = append(errs, fmt.Errorf(
			"meta.infrastructure: количество type (%d) и description (%d) должно совпадать",
			len(infra.Type), len(infra.Description),
		))
	}

	return errs
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
