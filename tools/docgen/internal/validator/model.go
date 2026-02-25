package validator

import (
	"docgen/internal/models"
	"fmt"
	"regexp"
	"strings"
)

var availableTypes = []string{
	"string",
	"integer",
	"float",
	"number",
	"boolean",
	"datetime",
	"object",
	"array",
	"file",
}

// localRefRe — локальная ссылка
var localRefRe = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

// externalRefRe — межсервисная ссылка
var externalRefRe = regexp.MustCompile(`^[a-z][a-z0-9-]*\.[A-Z][a-zA-Z0-9]*$`)

// ValidateModels валидирует модели.
// Возвращает слайс ошибок — пустой, если всё корректно.
func ValidateModels(modelsByName map[string]models.Model) []error {
	var errs []error

	for name, model := range modelsByName {
		errs = append(errs, validateModel(name, model, modelsByName)...)
	}

	return errs
}

func validateModel(name string, model models.Model, allModels map[string]models.Model) []error {
	var errs []error

	if len(model.Fields) == 0 {
		errs = append(errs, fmt.Errorf("model %q: должно быть хотя бы одно поле", name))
		return errs
	}

	for fieldName, field := range model.Fields {
		path := fmt.Sprintf("model %q → field %q", name, fieldName)
		errs = append(errs, validateField(path, field, allModels)...)
	}

	return errs
}

func validateField(path string, f models.Field, allModels map[string]models.Model) []error {
	var errs []error

	hasType := f.Type != ""
	hasRef := f.Ref != ""

	// Поле должно иметь либо type либо $ref
	if !hasType && !hasRef {
		errs = append(errs, fmt.Errorf("%s: необходимо указать type или $ref", path))
		return errs
	}

	if hasType && hasRef {
		errs = append(errs, fmt.Errorf("%s: нельзя указывать type и $ref одновременно", path))
		return errs
	}

	// Валидация $ref
	if hasRef {
		errs = append(errs, validateRef(path, f.Ref, allModels)...)
		return errs
	}

	// Проверяем допустимость типа
	if !isValidType(f.Type) {
		errs = append(errs, fmt.Errorf(
			"%s: недопустимый тип %q (допустимые: %s)",
			path, f.Type, strings.Join(availableTypes, ", "),
		))
	}

	// Для array обязательно наличие items
	if f.Type == "array" {
		if f.Items == nil {
			errs = append(errs, fmt.Errorf("%s: тип array требует описание items", path))
		} else {
			itemsPath := path + " → items"
			errs = append(errs, validateField(itemsPath, *f.Items, allModels)...)
		}
	}

	// min_length / max_length — только для string
	if f.MinLength != nil || f.MaxLength != nil {
		if f.Type != "string" {
			errs = append(errs, fmt.Errorf("%s: min_length/max_length применимы только к типу string", path))
		}
		if f.MinLength != nil && *f.MinLength < 0 {
			errs = append(errs, fmt.Errorf("%s: min_length не может быть отрицательным", path))
		}
		if f.MaxLength != nil && *f.MaxLength < 0 {
			errs = append(errs, fmt.Errorf("%s: max_length не может быть отрицательным", path))
		}
		if f.MinLength != nil && f.MaxLength != nil && *f.MinLength > *f.MaxLength {
			errs = append(errs, fmt.Errorf("%s: min_length (%d) > max_length (%d)", path, *f.MinLength, *f.MaxLength))
		}
	}

	// min / max — только для числовых типов
	if f.Min != nil || f.Max != nil {
		if f.Type != "integer" && f.Type != "float" && f.Type != "number" {
			errs = append(errs, fmt.Errorf("%s: min/max применимы только к числовым типам", path))
		}
		if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
			errs = append(errs, fmt.Errorf("%s: min (%v) > max (%v)", path, *f.Min, *f.Max))
		}
	}

	return errs
}

func validateRef(path string, ref string, allModels map[string]models.Model) []error {
	var errs []error

	switch {
	case localRefRe.MatchString(ref):
		// Локальная ссылка — проверяем существование модели в текущей карте
		if _, exists := allModels[ref]; !exists {
			errs = append(errs, fmt.Errorf("%s: $ref %q — модель не найдена в текущем файле", path, ref))
		}
	case externalRefRe.MatchString(ref):
		// Межсервисная ссылка (service-name.ModelName) — формат корректен,
	default:
		errs = append(errs, fmt.Errorf(
			"%s: $ref %q — неверный формат (ожидается ModelName или service-name.ModelName)",
			path, ref,
		))
	}

	return errs
}

func isValidType(t string) bool {
	for _, valid := range availableTypes {
		if t == valid {
			return true
		}
	}
	return false
}
