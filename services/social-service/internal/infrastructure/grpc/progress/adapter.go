package progress

import (
	"context"
	"errors"

	pkgprogress "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/progress"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type Adapter struct {
	client *pkgprogress.Client
}

func NewAdapter(client *pkgprogress.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) GetUserProgress(ctx context.Context, userID uint) (*model.Progress, error) {
	up, err := a.client.GetUserProgress(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProgress(up), nil
}

func mapError(err error) error {
	if errors.Is(err, pkgprogress.ErrUnavailable) {
		return domainerrors.ProgressServiceUnavailable(err)
	}
	return err
}

func toProgress(p *pkgprogress.UserExpProgress) *model.Progress {
	return &model.Progress{
		CurrentLevel:     p.CurrentLevel,
		CurrentLevelName: p.CurrentLevelName,
		CurrentXP:        p.CurrentXP,
		XPForNextLevel:   p.XPForNextLevel,
		ProgressPercent:  p.ProgressPercent,
		NextLevel: model.NextLevel{
			Level:     p.NextLevel.Level,
			LevelName: p.NextLevel.LevelName,
		},
	}
}
