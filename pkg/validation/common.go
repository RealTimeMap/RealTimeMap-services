package validation

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Location []any
type ErrorResponse struct {
	Detail []ValidationError `json:"detail"`
}

type ValidationError struct {
	Loc     Location `json:"loc"`
	Msg     string   `json:"msg"`
	ErrType string   `json:"type"`
	Input   any      `json:"input,omitempty"`
}

// New создаёт одну ошибку валидации
func New(loc Location, msg, errType string, input any) ValidationError {
	return ValidationError{
		Loc:     loc,
		Msg:     msg,
		ErrType: errType,
		Input:   input,
	}
}

// NewFieldError Конструктор для создания ошибок определенных типов. Создает ошибку для полей в body
func NewFieldError(field, msd, errType string, input any) ValidationError {
	return New(Location{"body", field}, msd, errType, input)
}

// NewQueryError Конструктор для создания ошибок определенных типов. Создает ошибку для query параметров
func NewQueryError(param, msd, errType string, input any) ValidationError {
	return New(Location{"query", param}, msd, errType, input)
}

// NewPathError Конструктор для создания ошибок определенных типов. Создает ошибку для path параметров
func NewPathError(param, msd, errType string, input any) ValidationError {
	return New(Location{"path", param}, msd, errType, input)
}

func Response(errs ...ValidationError) ErrorResponse {
	return ErrorResponse{Detail: errs}
}

func Abort(c *gin.Context, errs ...ValidationError) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, Response(errs...))
}
