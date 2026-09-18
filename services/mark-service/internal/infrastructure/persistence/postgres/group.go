package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type PgGroupRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgGroupRepository(db *gorm.DB, logger *zap.Logger) personal.GroupRepository {
	return &PgGroupRepository{
		db:     db,
		logger: logger,
	}
}

// dbCtx возвращает транзакцию из контекста (если сервис обернул вызов в
// txmanager.WithTx) либо собственный пул.
func (r *PgGroupRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

func (r *PgGroupRepository) Create(ctx context.Context, obj *personal.Group) error {
	r.logger.Info("start Create", zap.String("layer", "postgres repo"))
	err := r.dbCtx(ctx).Create(obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			// Дубль ловится и здесь, а не только предварительной проверкой в
			// сервисе: два параллельных запроса с одним UUID проходят её оба,
			// и разойтись их заставляет уже уникальность первичного ключа.
			// Ошибку нужно различать — иначе занятый id отдаётся клиенту как
			// конфликт имени, и тот безуспешно переименовывает группу.
			if isPrimaryKeyViolation(err) {
				return personal.ErrGroupIDTaken(obj.ID)
			}
			return personal.ErrAlreadyExistGroup(obj.Name)
		}
		return err
	}
	return nil
}

// groupPKConstraint — имя ограничения первичного ключа, которое Postgres
// подставляет по умолчанию для таблицы groups.
const groupPKConstraint = "groups_pkey"

// isPrimaryKeyViolation отличает конфликт по первичному ключу от конфликта по
// уникальному индексу (user_id, name).
func isPrimaryKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName == groupPKConstraint
	}
	return false
}

func (r *PgGroupRepository) Update(ctx context.Context, obj *personal.Group) error {
	r.logger.Info("start Update", zap.String("layer", "postgres repo"))

	// Select по именам колонок: Updates со структурой пропустил бы nil-поля,
	// а description сбрасывается в null осознанно.
	err := r.dbCtx(ctx).
		Model(obj).
		Select("name", "description", "revision", "color", "icon").
		Updates(obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return personal.ErrAlreadyExistGroup(obj.Name)
		}
		return err
	}
	return nil
}

func (r *PgGroupRepository) GetByID(ctx context.Context, groupID uuid.UUID, userID uint) (personal.Group, error) {
	r.logger.Info("start GetByID", zap.String("layer", "postgres repo"))

	var obj personal.Group

	err := r.dbCtx(ctx).
		Where("id = ?", groupID).
		Where("user_id = ?", userID).
		First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return personal.Group{}, personal.ErrNotFoundGroup(groupID)
		}
		return personal.Group{}, err
	}

	return obj, nil
}

// ExistsByID сообщает, занят ли идентификатор. Unscoped и без фильтра по
// владельцу: PK остаётся занятым и после soft-delete, а чужая группа для
// вызывающего — такой же конфликт, как своя.
func (r *PgGroupRepository) ExistsByID(ctx context.Context, groupID uuid.UUID) (bool, error) {
	r.logger.Info("start ExistsByID", zap.String("layer", "postgres repo"))

	var count int64

	err := r.dbCtx(ctx).
		Unscoped().
		Model(&personal.Group{}).
		Where("id = ?", groupID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CountMarks считает живые метки в группе через таблицу связи many2many.
func (r *PgGroupRepository) CountMarks(ctx context.Context, groupID uuid.UUID) (int64, error) {
	r.logger.Info("start CountMarks", zap.String("layer", "postgres repo"))

	var count int64

	err := r.dbCtx(ctx).
		Table("personal_marks_groups AS pmg").
		Joins("JOIN personal_marks AS pm ON pm.id = pmg.model_id").
		Where("pmg.group_id = ?", groupID).
		Where("pm.deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// SetRevision обновляет только revision. Unscoped, чтобы вызов после
// soft-delete тоже находил строку.
func (r *PgGroupRepository) SetRevision(ctx context.Context, groupID uuid.UUID, revision uint) error {
	r.logger.Info("start SetRevision", zap.String("layer", "postgres repo"))

	return r.dbCtx(ctx).
		Unscoped().
		Model(&personal.Group{}).
		Where("id = ?", groupID).
		Update("revision", revision).Error
}

func (r *PgGroupRepository) List(ctx context.Context, userID uint, params pagination.Params) ([]*personal.Group, int64, error) {
	r.logger.Info("start List", zap.String("layer", "postgres repo"))
	var count int64
	var objs []*personal.Group
	err := r.dbCtx(ctx).
		Model(&personal.Group{}).
		Where("user_id = ?", userID).
		Offset(params.Offset()).
		Limit(params.Limit()).
		Find(&objs).
		Count(&count).
		Error
	if err != nil {
		return nil, 0, err
	}
	return objs, count, nil
}

func (r *PgGroupRepository) Delete(ctx context.Context, obj *personal.Group) error {
	r.logger.Info("start Delete", zap.String("layer", "postgres repo"))

	return r.dbCtx(ctx).Delete(obj).Error
}

func (r *PgGroupRepository) GetBatch(ctx context.Context, userID uint, ids []uuid.UUID) ([]*personal.Group, error) {
	r.logger.Info("start GetBatch", zap.String("layer", "postgres repo"), zap.Any("ids", ids))
	var objs []*personal.Group

	err := r.dbCtx(ctx).Where("user_id = ? AND id IN ?", userID, ids).Find(&objs).Error
	if err != nil {
		return nil, err
	}

	if len(objs) != len(ids) {
		return nil, personal.ErrNotFoundGroup(missing(ids, objs))
	}

	return objs, nil
}

func missing(ids []uuid.UUID, rows []*personal.Group) []uuid.UUID {
	found := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		found[rows[i].ID] = struct{}{}
	}
	out := make([]uuid.UUID, 0, len(ids)-len(rows))
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

func (r *PgGroupRepository) Get(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]personal.Group, error) {
	r.logger.Info("start List", zap.String("layer", "postgres repo"))

	q := r.db.WithContext(ctx).
		Model(&personal.Group{}).
		Unscoped().
		Where("user_id = ?", userID).
		Where("revision <= ?", upTo).
		Order("revision ASC").
		Limit(limit + 1)

	if since != nil {
		q = q.Where("revision > ?", *since)
	} else {
		q = q.Where("deleted_at IS NULL")
	}

	var objs []personal.Group

	if err := q.Find(&objs).Error; err != nil {
		return nil, err
	}

	return objs, nil
}
