package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/key"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgApiKeyRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPgApiKeyRepository(db *gorm.DB, logger *zap.Logger) key.Repository {
	return &PgApiKeyRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgApiKeyRepository) Create(ctx context.Context, obj *key.Model) error {
	if obj.ID == uuid.Nil {
		obj.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(obj).Error
}
func (r *PgApiKeyRepository) Update(ctx context.Context, obj *key.Model) error {
	return r.db.WithContext(ctx).Save(obj).Error
}
func (r *PgApiKeyRepository) Get(ctx context.Context, token string) (*key.Model, error) {
	var obj *key.Model

	err := r.db.WithContext(ctx).First(&obj, "key_hash = ?", token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, key.ApiKeyNotFound()
		}
		return nil, err
	}
	return obj, nil
}
