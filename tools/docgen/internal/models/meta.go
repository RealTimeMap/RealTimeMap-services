package models

// meta.yaml
// Это файл описывающий сам сервис, название, описание, для чего сделан и на чем сделан.
// Техническая информация

type Meta struct {
	Name           string         `yaml:"name"`
	DisplayName    string         `yaml:"displayName"`
	Description    string         `yaml:"description"`
	Version        string         `yaml:"version"`
	Struct         Struct         `yaml:"struct"`
	Infrastructure Infrastructure `yaml:"infrastructure"`
	Tags           []string       `yaml:"tags"`
}

type Struct struct {
	Language  string   `yaml:"language"`
	Protocols []string `yaml:"protocols"`
}

type Infrastructure struct {
	Type        []string `yaml:"type"`
	Description []string `yaml:"description"`
}
