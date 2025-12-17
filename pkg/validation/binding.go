package validation

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Мапинг для автоматеческого определения типа ошибки
var typeMapping = map[string]string{
	// Базовые валидации
	"required":         "value_error.missing",
	"required_with":    "value_error.missing",
	"required_without": "value_error.missing",
	"required_unless":  "value_error.missing",

	// Строки
	"email":       "value_error.email",
	"len":         "value_error.any_str.length",
	"min":         "value_error.any_str.min_length",
	"max":         "value_error.any_str.max_length",
	"alpha":       "value_error.str.regex",
	"alphanum":    "value_error.str.regex",
	"numeric":     "value_error.str.regex",
	"contains":    "value_error.str.contains",
	"url":         "value_error.url",
	"uri":         "value_error.url",
	"uuid":        "value_error.uuid",
	"uuid4":       "value_error.uuid",
	"uuid3":       "value_error.uuid",
	"uuid5":       "value_error.uuid",
	"isbn10":      "value_error.isbn",
	"isbn13":      "value_error.isbn",
	"hexadecimal": "value_error.str.regex",
	"json":        "value_error.json",
	"jwt":         "value_error.jwt",
	"base64":      "value_error.base64",
	"ascii":       "value_error.str.ascii",
	"lowercase":   "value_error.str.lowercase",
	"uppercase":   "value_error.str.uppercase",

	// Числа
	"number_min": "value_error.number.not_ge",
	"number_max": "value_error.number.not_le",
	"eq":         "value_error.number.not_equal",
	"ne":         "value_error.number.equal",
	"gt":         "value_error.number.not_gt",
	"gte":        "value_error.number.not_ge",
	"lt":         "value_error.number.not_lt",
	"lte":        "value_error.number.not_le",
	"oneof":      "value_error.const",

	// Файлы
	"file":  "value_error.file",
	"image": "value_error.image",
	"mime":  "value_error.mime_type",

	// Дата и время
	"datetime": "value_error.datetime",
	"timezone": "value_error.timezone",

	// Массивы
	"dive":      "value_error.list.items",
	"unique":    "value_error.list.unique",
	"array_min": "value_error.list.min_items",
	"array_max": "value_error.list.max_items",

	// Специальные
	"ip":          "value_error.ip_address",
	"ipv4":        "value_error.ipv4_address",
	"ipv6":        "value_error.ipv6_address",
	"mac":         "value_error.mac_address",
	"credit_card": "value_error.credit_card",
	"ssn":         "value_error.ssn",
	"latitude":    "value_error.latitude",
	"longitude":   "value_error.longitude",

	// Кастомные
	"custom": "value_error.custom",
}

func FromBindingError(err error) []ValidationError {
	var result []ValidationError

	if validationErr, ok := err.(validator.ValidationErrors); ok {
		for _, v := range validationErr {
			errType := typeMapping[v.Tag()]
			if errType == "" {
				errType = "value_error." + v.Tag()
			}

			result = append(result, ValidationError{
				Loc:     Location{"body", v.Field()},
				Msg:     buildMessage(v),
				ErrType: errType,
				Input:   v.Value(),
			})
		}
	} else {
		result = append(result, ValidationError{
			Loc:     Location{"body"},
			Msg:     err.Error(),
			ErrType: "value_error.jsonencode",
			Input:   "",
		})
	}
	return result
}

func AbortWithBindingError(c *gin.Context, err error) {
	Abort(c, FromBindingError(err)...)
}

func buildMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "field required"
	case "email":
		return "value is not a valid email address"
	case "min":
		return "ensure this value is greater than or equal to " + e.Param()
	case "max":
		return "ensure this value is less than or equal to " + e.Param()
	default:
		return "invalid value"
	}
}
