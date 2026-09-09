package personal

import (
	"context"

	personalsrv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

// typedChangeSource описывает доменный сервис, который возвращает изменения
// конкретного типа T. Go-дженерики инвариантны, поэтому Changes[Model] и
// Changes[Group] не являются Changes[any] и не могут напрямую удовлетворить
// ChangeSource.
type typedChangeSource[T any] interface {
	Name() string
	ListChanges(ctx context.Context, userID uint, since *uint, upTo uint, limit int) (personalsrv.Changes[T], error)
}

// changeSourceAdapter стирает параметр типа, приводя Changes[T] к Changes[any]
// на границе domain -> use_cases.
type changeSourceAdapter[T any] struct {
	src typedChangeSource[T]
}

// NewChangeSource оборачивает типизированный доменный сервис в ChangeSource.
func NewChangeSource[T any](src typedChangeSource[T]) ChangeSource {
	return changeSourceAdapter[T]{src: src}
}

func (a changeSourceAdapter[T]) Name() string { return a.src.Name() }

func (a changeSourceAdapter[T]) ListChanges(
	ctx context.Context,
	userID uint,
	since *uint,
	upTo uint,
	limit int,
) (personalsrv.Changes[any], error) {
	ch, err := a.src.ListChanges(ctx, userID, since, upTo, limit)
	if err != nil {
		return personalsrv.Changes[any]{}, err
	}

	upserted := make([]any, 0, len(ch.Upserted))
	for _, v := range ch.Upserted {
		upserted = append(upserted, v)
	}

	removed := ch.Removed
	if removed == nil {
		removed = []uint{}
	}

	return personalsrv.Changes[any]{
		Upserted: upserted,
		Removed:  removed,
		Cursor:   ch.Cursor,
		HasMore:  ch.HasMore,
	}, nil
}
