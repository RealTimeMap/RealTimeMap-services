package personal

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	Update(ctx context.Context, obj *Model) error
	List(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]Model, error)
	Delete(ctx context.Context, markID uint) error
	GetByID(ctx context.Context, markID, userID uint) (Model, error)
	// SetRevision проставляет ревизию без загрузки модели: нужен при удалении,
	// чтобы soft-deleted строка доехала до клиента через ListChanges.
	SetRevision(ctx context.Context, markID, revision uint) error
	// ReplaceGroups перезаписывает состав many2many personal_marks_groups.
	ReplaceGroups(ctx context.Context, obj *Model, groups []*Group) error
}

type RevisionRepository interface {
	UpsertRevision(ctx context.Context, userID uint) (uint, error)
	GetRevision(ctx context.Context, userID uint) (uint, error)
}

type GroupRepository interface {
	Create(ctx context.Context, obj *Group) error
	Update(ctx context.Context, obj *Group) error
	List(ctx context.Context, userID uint, params pagination.Params) ([]*Group, int64, error)
	Delete(ctx context.Context, obj *Group) error
	GetBatch(ctx context.Context, userID uint, ids []uint) ([]*Group, error)
	GetByID(ctx context.Context, groupID, userID uint) (Group, error)
	Get(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]Group, error)
	// CountMarks считает живые метки, привязанные к группе: удаление
	// непустой группы осиротило бы их, поэтому запрещено.
	CountMarks(ctx context.Context, groupID uint) (int64, error)
	// SetRevision проставляет ревизию без загрузки модели — нужен при удалении,
	// чтобы soft-deleted строка доехала до клиента через ListChanges.
	SetRevision(ctx context.Context, groupID, revision uint) error
}
