package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID uint) (*model.Profile, error)
	GetProfiles(ctx context.Context, search string, params pagination.Params) ([]*model.Profile, int64, error)
	GetProfilesByIDs(ctx context.Context, ids []uint) ([]*model.Profile, error)
	Create(ctx context.Context, profile *model.Profile) (*model.Profile, error)
	Update(ctx context.Context, userID uint, fields map[string]any) (*model.Profile, error)
}
