package stats

import (
	"context"
	"errors"

	pkgmark "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
)

type Adapter struct {
	client *pkgmark.Client
}

func NewAdapter(client *pkgmark.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) GetUserMarksCount(ctx context.Context, userID uint) (int64, error) {
	count, err := a.client.GetUserMarksCount(ctx, userID)
	if err != nil {
		return 0, mapError(err)
	}
	return count, nil
}

func mapError(err error) error {
	if errors.Is(err, pkgmark.ErrServiceUnavailable) {
		return domainerrors.ProgressServiceUnavailable(err)
	}
	return err
}
