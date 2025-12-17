package apperror

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
)

type DomainError interface {
	error
	HTTPStatus() int
	ToValidation() []validation.ValidationError
}

type FieldValidationError struct {
	field   string
	message string
	errType string
	value   interface{}
}

func (e *FieldValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.field, e.message)
}

func (e *FieldValidationError) HTTPStatus() int {
	return 422
}

func (e *FieldValidationError) ToValidation() []validation.ValidationError {
	return []validation.ValidationError{
		validation.NewFieldError(e.field, e.message, e.errType, e.value),
	}
}

// NotFoundError - ресурс не найден (404)
type NotFoundError struct {
	Resource string
	Field    string // Имя поля для validation error (например, "category_id")
	ID       interface{}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %v", e.Resource, e.ID)
}

func (e *NotFoundError) HTTPStatus() int {
	return 404
}

func (e *NotFoundError) ToValidation() []validation.ValidationError {
	field := e.Field
	if field == "" {
		field = "id" // Дефолтное значение для обратной совместимости
	}
	return []validation.ValidationError{
		validation.NewFieldError(
			field,
			fmt.Sprintf("%s not found", e.Resource),
			"value_error.not_found",
			e.ID,
		),
	}
}

// ConflictError - конфликт бизнес-правил (409)
type ConflictError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConflictError) Error() string {
	return e.Message
}

func (e *ConflictError) HTTPStatus() int {
	return 409
}

func (e *ConflictError) ToValidation() []validation.ValidationError {
	return []validation.ValidationError{
		validation.NewFieldError(e.Field, e.Message, "value_error.conflict", e.Value),
	}
}

// MultipleValidationErrors - несколько ошибок валидации сразу
type MultipleValidationErrors struct {
	Errors []DomainError
}

func (e *MultipleValidationErrors) Error() string {
	return fmt.Sprintf("multiple validation domainerrors: %d", len(e.Errors))
}

func (e *MultipleValidationErrors) HTTPStatus() int {
	return 422
}

func (e *MultipleValidationErrors) ToValidation() []validation.ValidationError {
	var result []validation.ValidationError
	for _, err := range e.Errors {
		result = append(result, err.ToValidation()...)
	}
	return result
}

// ForbiddenError - ошибка прав доступа
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return "forbidden"
}
func (e *ForbiddenError) HTTPStatus() int {
	return 403
}

func (e *ForbiddenError) ToValidation() []validation.ValidationError {
	return nil
}

// InternalError - внутренняя ошибка (500)
type InternalError struct {
	Message string
	Cause   error
}

func (e *InternalError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *InternalError) HTTPStatus() int {
	return 500
}

func (e *InternalError) ToValidation() []validation.ValidationError {
	return nil
}

func (e *InternalError) Unwrap() error {
	return e.Cause
}
