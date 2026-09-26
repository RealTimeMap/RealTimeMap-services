package repository

import "context"

// AccountRepository стирает игровые данные пользователя, удалившего аккаунт:
// прогресс, историю начислений и открытые достижения.
//
// Повторный вызов для того же пользователя безопасен и ничего не меняет.
type AccountRepository interface {
	EraseUser(ctx context.Context, userID uint) error
}
