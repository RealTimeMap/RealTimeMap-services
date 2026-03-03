package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *model.Profile) (*model.Profile, error)
}
