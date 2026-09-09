package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Filter struct {
	Pagination pagination.Params
	Status     *Status
	Tag        *Tag

	// Statuses отбирает баги по набору статусов. Нужен перечню открытых
	// багов: «new или in work» одним условием не выразить через Status.
	Statuses []Status

	// OnlyUnlinked отбирает баги, не привязанные ни к одной задаче.
	// Перечень для привязки показывает только их: занятый баг предлагать
	// второй раз незачем.
	OnlyUnlinked bool
}

type Repository interface {
	Create(ctx context.Context, data *Model) error
	GetByID(ctx context.Context, id uint) (*Model, error)
	GetList(ctx context.Context, filter Filter) ([]Model, error)

	// Update сохраняет изменённый баг. Используется привязкой к задаче
	// и обратной синхронизацией статуса.
	Update(ctx context.Context, data *Model) error

	// GetByTaskID находит баг, привязанный к задаче. Отсутствие привязки —
	// не ошибка: возвращается nil.
	GetByTaskID(ctx context.Context, taskID uint) (*Model, error)
}
