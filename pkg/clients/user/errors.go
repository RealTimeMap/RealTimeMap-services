package user

import "errors"

var (
	// ErrUnavailable — сервис недоступен или не ответил в срок. Вызывающему
	// стоит повторить попытку позже.
	ErrUnavailable = errors.New("user-service unavailable")

	// ErrNotFound — пользователя с таким id нет. Повтор не поможет.
	ErrNotFound = errors.New("user not found")
)
