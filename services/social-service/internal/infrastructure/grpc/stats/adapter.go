package stats

import (
	"context"
	"errors"
	"time"

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

func (a *Adapter) GetUserMarksMonthlyActivity(ctx context.Context, userID uint, year int) ([]*pkgmark.MonthlyActivity, error) {
	result, err := a.client.GetUserMarksMonthlyActivity(ctx, userID, year)
	if err != nil {
		return nil, mapError(err)
	}
	return result, nil
}

func (a *Adapter) GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*pkgmark.HeatMapItem, error) {
	result, err := a.client.GetUserMarksHeatMap(ctx, userID, start, end)
	if err != nil {
		return nil, mapError(err)
	}
	return result, nil
}

func mapError(err error) error {
	if errors.Is(err, pkgmark.ErrServiceUnavailable) {
		return domainerrors.MarkServiceUnavailable(err)
	}
	return err
}
