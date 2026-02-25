package models

type Field struct {
	// Базовый тип: string, integer, float, boolean, datetime, object, etc.
	Type        string   `yaml:"type,omitempty"      json:"type,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Format      string   `yaml:"format,omitempty"    json:"format,omitempty"`
	Required    bool     `yaml:"required,omitempty"  json:"required,omitempty"`
	Nullable    bool     `yaml:"nullable,omitempty"  json:"nullable,omitempty"`
	Example     any      `yaml:"example,omitempty"   json:"example,omitempty"`
	Default     any      `yaml:"default,omitempty"   json:"default,omitempty"`
	Enum        []any    `yaml:"enum,omitempty"      json:"enum,omitempty"`
	MinLength   *int     `yaml:"min_length,omitempty" json:"min_length,omitempty"`
	MaxLength   *int     `yaml:"max_length,omitempty" json:"max_length,omitempty"`
	Min         *float64 `yaml:"min,omitempty"       json:"min,omitempty"`
	Max         *float64 `yaml:"max,omitempty"       json:"max,omitempty"`
	Deprecated  bool     `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`

	// Ссылка на другую модель
	Ref string `yaml:"$ref,omitempty" json:"_ref,omitempty"`

	// Для type: array — описание элементов
	Items *Field `yaml:"items,omitempty" json:"items,omitempty"`

	// Для type: object или resolved $ref — вложенные поля
	Fields map[string]Field `yaml:"fields,omitempty" json:"fields,omitempty"`
}

type Model struct {
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Fields      map[string]Field `yaml:"fields,omitempty" json:"fields,omitempty"`
}
