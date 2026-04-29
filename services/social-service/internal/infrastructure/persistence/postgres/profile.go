package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgProfileRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgProfileRepository(db *gorm.DB, logger *zap.Logger) repository.ProfileRepository {
	return &PgProfileRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgProfileRepository) Create(ctx context.Context, profile *model.Profile) (*model.Profile, error) {
	err := r.db.WithContext(ctx).Create(profile).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *PgProfileRepository) GetProfile(ctx context.Context, userID uint) (*model.Profile, error) {
	var profile *model.Profile
	err := r.db.WithContext(ctx).First(&profile, "user_id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ProfileNotFound(userID)
		}
		return nil, err
	}
	return profile, nil
}

func (r *PgProfileRepository) GetProfiles(ctx context.Context, search string, params pagination.Params) ([]*model.Profile, int64, error) {
	var profiles []*model.Profile
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Profile{}).
		Where("is_private = false AND privacy_settings->>'showInSearch' = 'true'")

	if search != "" {
		query = query.Where("username ILIKE ?", "%"+search+"%")
	}

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	q := query.Offset(params.Offset()).Limit(params.Limit())
	if search == "" {
		q = q.Order("RANDOM()")
	} else {
		q = q.Order("username")
	}

	err = q.Find(&profiles).Error
	if err != nil {
		return nil, 0, err
	}
	return profiles, count, nil
}

func (r *PgProfileRepository) Update(ctx context.Context, userID uint, fields map[string]any) (*model.Profile, error) {
	if len(fields) == 0 {
		return r.GetProfile(ctx, userID)
	}

	res := r.db.WithContext(ctx).
		Model(&model.Profile{}).
		Where("user_id = ?", userID).
		Updates(fields)

	if err := res.Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if tag, ok := fields["tag"].(string); ok {
				return nil, domainerrors.TagAlreadyTaken(tag)
			}
			return nil, fmt.Errorf("unique violation: %w", err)
		}
		return nil, err
	}

	if res.RowsAffected == 0 {
		return nil, domainerrors.ProfileNotFound(userID)
	}

	return r.GetProfile(ctx, userID)
}

func (r *PgProfileRepository) GetProfilesByIDs(ctx context.Context, ids []uint) ([]*model.Profile, error) {
	if len(ids) == 0 {
		return []*model.Profile{}, nil
	}

	var profiles []*model.Profile

	err := r.db.WithContext(ctx).Where("user_id IN (?)", ids).Find(&profiles).Error
	if err != nil {
		return nil, err
	}

	return profiles, nil
}
