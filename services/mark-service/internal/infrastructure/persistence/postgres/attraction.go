package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type PgAttractionRepostiroty struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgAttractionRepostiroty(db *gorm.DB, logger *zap.Logger) attraction.Repository {
	return &PgAttractionRepostiroty{
		db:     db,
		logger: logger,
	}
}

func (r *PgAttractionRepostiroty) Create(ctx context.Context, obj *attraction.Model) error {
	err := r.db.WithContext(ctx).Create(obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return attraction.ErrAlreadyExist(obj.Slug)
		}
		return err
	}
	return nil
}

func (r *PgAttractionRepostiroty) ListByCity(ctx context.Context, city string) ([]attraction.Model, error) {
	var objs []attraction.Model
	err := r.db.WithContext(ctx).
		Select("id", "slug", "name", "category", "geom").
		Where("city = ? AND status = ?", city, attraction.StatusPublished).
		Order("name").
		Find(&objs).Error
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (r *PgAttractionRepostiroty) GetPublished(ctx context.Context, id uuid.UUID) (attraction.Model, error) {
	return r.getWithSlides(ctx, id, r.db.Where("status = ?", attraction.StatusPublished))
}

func (r *PgAttractionRepostiroty) GetByID(ctx context.Context, id uuid.UUID) (attraction.Model, error) {
	return r.getWithSlides(ctx, id, r.db)
}

func (r *PgAttractionRepostiroty) getWithSlides(ctx context.Context, id uuid.UUID, q *gorm.DB) (attraction.Model, error) {
	var obj attraction.Model
	err := q.WithContext(ctx).
		Preload("Slides", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Preload("Slides.Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Where("id = ?", id).
		First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return attraction.Model{}, attraction.ErrNotFound(id)
		}
		return attraction.Model{}, err
	}
	return obj, nil
}

func (r *PgAttractionRepostiroty) Update(ctx context.Context, obj *attraction.Model, replaceSlides bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(obj).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return attraction.ErrAlreadyExist(obj.Slug)
			}
			return err
		}
		if !replaceSlides {
			return nil
		}

		// Блоки удаляются каскадом по FK
		if err := tx.Where("attraction_id = ?", obj.ID).Delete(&attraction.Slide{}).Error; err != nil {
			return err
		}
		if len(obj.Slides) == 0 {
			return nil
		}
		for i := range obj.Slides {
			obj.Slides[i].AttractionID = obj.ID
		}
		return tx.Create(&obj.Slides).Error
	})
}

func (r *PgAttractionRepostiroty) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&attraction.Model{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return attraction.ErrNotFound(id)
	}
	return nil
}
