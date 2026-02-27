package models

// meta.yaml
// Это файл описывающий сам сервис, название, описание, для чего сделан и на чем сделан.
// Техническая информация

type Meta struct {
	Name           string         `yaml:"name" json:"name"`
	DisplayName    string         `yaml:"displayName" json:"displayName"`
	Description    string         `yaml:"description" json:"description"`
	Version        string         `yaml:"version" json:"version"`
	Struct         Struct         `yaml:"struct" json:"struct"`
	Infrastructure Infrastructure `yaml:"infrastructure" json:"infrastructure"`
	Tags           []string       `yaml:"tags" json:"tags"`
}

type Struct struct {
	Language  string   `yaml:"language" json:"language"`
	Protocols []string `yaml:"protocols" json:"protocols"`
}

type Infrastructure struct {
	Type        []string `yaml:"type" json:"type"`
	Description []string `yaml:"description" json:"description"`
}
