package models

// ServiceOutput — итоговое представление сервиса в JSON.
type ServiceOutput struct {
	Name           string           `json:"name"`
	DisplayName    string           `json:"displayName"`
	Description    string           `json:"description"`
	Version        string           `json:"version"`
	Struct         Struct           `json:"struct"`
	Infrastructure Infrastructure   `json:"infrastructure"`
	Tags           []string         `json:"tags"`
	Models         map[string]Model `json:"models,omitempty"`
	API            API              `json:"api,omitempty"`
}

// BuildOutput — корневая структура итогового JSON.
type BuildOutput struct {
	Services []ServiceOutput `json:"services"`
}
