package apperror

import "fmt"

func NewFieldValidationError(field, message, errType string, value interface{}) *FieldValidationError {
	return &FieldValidationError{
		field:   field,
		message: message,
		errType: errType,
		value:   value,
	}
}
func NewRequiredError(field string) *FieldValidationError {
	return NewFieldValidationError(field, "field is required", "value_error.missing", nil)
}

func NewTooLongError(field string, maxLength int, value string) *FieldValidationError {
	return NewFieldValidationError(
		field,
		fmt.Sprintf("must not exceed %d characters", maxLength),
		"value_error.any_str.max_length",
		value,
	)
}

func NewInvalidFormatError(field, format string, value interface{}) *FieldValidationError {
	return NewFieldValidationError(
		field,
		fmt.Sprintf("must be in %s format", format),
		"value_error.str.regex",
		value,
	)
}

func NewInvalidMimeTypeError(field string, allowed []string, actual string) *FieldValidationError {
	return NewFieldValidationError(
		field,
		fmt.Sprintf("must be one of: %v", allowed),
		"value_error.mime_type",
		actual,
	)
}

func NewInvalidImageError(field string) *FieldValidationError {
	return NewFieldValidationError(field, "file is not a valid image", "value_error.image", nil)
}

// Not Found constructors

func NewNotFoundError(resource string, field string, id interface{}) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Field:    field,
		ID:       id,
	}
}

// NewNotFoundErrorByID создает NotFoundError с дефолтным полем "id" (для обратной совместимости)
func NewNotFoundErrorByID(resource string, id interface{}) *NotFoundError {
	return NewNotFoundError(resource, "id", id)
}

// Conflict constructors

func NewAlreadyExistsError(field string, value interface{}) *ConflictError {
	return &ConflictError{
		Field:   field,
		Message: fmt.Sprintf("%s already exists", field),
		Value:   value,
	}
}

func NewConflictError(field, message string, value interface{}) *ConflictError {
	return &ConflictError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// Multiple domainerrors

func NewMultipleErrors(errors ...DomainError) *MultipleValidationErrors {
	return &MultipleValidationErrors{Errors: errors}
}

// Internal domainerrors

func NewInternalError(message string, cause error) *InternalError {
	return &InternalError{
		Message: message,
		Cause:   cause,
	}
}

func WrapInternalError(message string, err error) *InternalError {
	return NewInternalError(message, err)
}
