package models

import "gopkg.in/yaml.v3"

// API — корневая структура api.yaml.
type API struct {
	HTTP     *HTTPSpec     `yaml:"http,omitempty"     json:"http,omitempty"`
	SocketIO *SocketIOSpec `yaml:"socketio,omitempty" json:"socketio,omitempty"`
	GRPC     *GRPCSpec     `yaml:"grpc,omitempty"     json:"grpc,omitempty"`
}

// HTTPSpec описывает HTTP-раздел api.yaml.
type HTTPSpec struct {
	BasePath       string           `yaml:"base_path"       json:"base_path,omitempty"`
	DefaultHeaders map[string]Field `yaml:"default_headers" json:"default_headers,omitempty"`
	Groups         []HTTPGroup      `yaml:"groups"          json:"groups,omitempty"`
}

type HTTPGroup struct {
	Name        string         `yaml:"name"        json:"name"`
	Description string         `yaml:"description" json:"description,omitempty"`
	Endpoints   []HTTPEndpoint `yaml:"endpoints"   json:"endpoints,omitempty"`
}

type HTTPEndpoint struct {
	Path        string                  `yaml:"path"        json:"path"`
	Method      string                  `yaml:"method"      json:"method"`
	Summary     string                  `yaml:"summary"     json:"summary,omitempty"`
	Description string                  `yaml:"description" json:"description,omitempty"`
	Headers     interface{}             `yaml:"headers"     json:"headers,omitempty"` // "none" или map[string]Field
	Parameters  []HTTPParameter         `yaml:"parameters"  json:"parameters,omitempty"`
	Request     *HTTPRequest            `yaml:"request"     json:"request,omitempty"`
	Responses   map[string]HTTPResponse `yaml:"responses" json:"responses,omitempty"`
}

type HTTPParameter struct {
	Name        string `yaml:"name"        json:"name"`
	In          string `yaml:"in"          json:"in"`
	Type        string `yaml:"type"        json:"type,omitempty"`
	Required    bool   `yaml:"required"    json:"required,omitempty"`
	Description string `yaml:"description" json:"description,omitempty"`
	Default     any    `yaml:"default"     json:"default,omitempty"`
	Enum        []any  `yaml:"enum"        json:"enum,omitempty"`
	Format      string `yaml:"format"      json:"format,omitempty"`
}

type HTTPRequest struct {
	ContentType string     `yaml:"content_type" json:"content_type,omitempty"`
	Body        BodyFields `yaml:"body"        json:"body,omitempty"`
}

type HTTPResponse struct {
	Description string     `yaml:"description"  json:"description,omitempty"`
	ContentType string     `yaml:"content_type" json:"content_type,omitempty"`
	Body        BodyFields `yaml:"body"        json:"body,omitempty"`
}

// BodyFields — map[string]Field с кастомным UnmarshalYAML.
// Поддерживает два формата:
//  1. Одиночная ссылка: `$ref: ModelName` → {"$ref": Field{Ref: "ModelName"}}
//  2. Инлайн-карта полей: обычный map[string]Field
//  3. Одиночное поле с type/items (например, type: array + items.$ref)
type BodyFields map[string]Field

// isSingleFieldNode возвращает true, если mapping node описывает одиночное поле Field,
// а не карту именованных полей. Признаки: ключи $ref, type, items, description, nullable и т.д.
func isSingleFieldNode(node *yaml.Node) bool {
	singleFieldKeys := map[string]bool{
		"$ref": true, "type": true, "items": true, "description": true,
		"nullable": true, "required": true, "example": true, "default": true,
		"format": true, "enum": true, "fields": true, "deprecated": true,
		"min_length": true, "max_length": true, "min": true, "max": true,
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if !singleFieldKeys[key] {
			return false
		}
	}
	return true
}

func (b *BodyFields) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return nil
	}

	result := make(BodyFields)

	// Если узел описывает одиночное поле (type/items/$ref/...) — сохраняем под ключом "$self"
	if isSingleFieldNode(value) {
		var f Field
		if err := value.Decode(&f); err != nil {
			return err
		}
		result["$self"] = f
		*b = result
		return nil
	}

	// Инлайн-карта: ключи — имена полей, значения — Field
	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		valNode := value.Content[i+1]

		var f Field
		if err := valNode.Decode(&f); err != nil {
			return err
		}
		result[key] = f
	}

	*b = result
	return nil
}

// SocketIOSpec описывает socketio-раздел api.yaml.
type SocketIOSpec struct {
	Namespace   string              `yaml:"namespace"   json:"namespace,omitempty"`
	Description string              `yaml:"description" json:"description,omitempty"`
	Connection  *SocketIOConnection `yaml:"connection"  json:"connection,omitempty"`
	Events      []SocketIOEvent     `yaml:"events"      json:"events,omitempty"`
}

type SocketIOConnection struct {
	Headers     map[string]Field `yaml:"headers"     json:"headers,omitempty"`
	Description string           `yaml:"description" json:"description,omitempty"`
}

type SocketIOEvent struct {
	Name        string                 `yaml:"name"        json:"name"`
	Direction   string                 `yaml:"direction"   json:"direction"`
	Summary     string                 `yaml:"summary"     json:"summary,omitempty"`
	Description string                 `yaml:"description" json:"description,omitempty"`
	Request     BodyFields             `yaml:"request"     json:"request,omitempty"`
	Response    *SocketIOEventResponse `yaml:"response"    json:"response,omitempty"`
	Payload     BodyFields             `yaml:"payload"     json:"payload,omitempty"`
}

type SocketIOEventResponse struct {
	Event   string     `yaml:"event"   json:"event,omitempty"`
	Payload BodyFields `yaml:"payload" json:"payload,omitempty"`
}

// GRPCSpec описывает grpc-раздел api.yaml (на будущее).
type GRPCSpec struct {
	Package   string        `yaml:"package"    json:"package,omitempty"`
	ProtoFile string        `yaml:"proto_file" json:"proto_file,omitempty"`
	Services  []GRPCService `yaml:"services"   json:"services,omitempty"`
}

type GRPCService struct {
	Name        string       `yaml:"name"        json:"name"`
	Description string       `yaml:"description" json:"description,omitempty"`
	Methods     []GRPCMethod `yaml:"methods"     json:"methods,omitempty"`
}

type GRPCMethod struct {
	Name        string       `yaml:"name"        json:"name"`
	Description string       `yaml:"description" json:"description,omitempty"`
	Request     *GRPCMessage `yaml:"request"     json:"request,omitempty"`
	Response    *GRPCMessage `yaml:"response"    json:"response,omitempty"`
	Errors      []GRPCError  `yaml:"errors"      json:"errors,omitempty"`
}

type GRPCMessage struct {
	Message string           `yaml:"message" json:"message,omitempty"`
	Fields  map[string]Field `yaml:"fields"  json:"fields,omitempty"`
}

type GRPCError struct {
	Code        string `yaml:"code"        json:"code"`
	Description string `yaml:"description" json:"description,omitempty"`
}
