package profile

import (
	"context"
	"math/rand"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"go.uber.org/zap"
)

type MarkStatGetter interface {
	GetUserMarksCount(ctx context.Context, userID uint) (int64, error)
	GetUserMarksMonthlyActivity(ctx context.Context, userID uint, year int) ([]*mark.MonthlyActivity, error)
	GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*mark.HeatMapItem, error)
}

type StatService struct {
	markStat MarkStatGetter

	logger *zap.Logger
}

func NewStatService(markStat MarkStatGetter, logger *zap.Logger) *StatService {
	return &StatService{
		markStat: markStat,
		logger:   logger,
	}
}

// GetProfileSummaryStat Формирует Summary статистику для отображения в профиле
func (s *StatService) GetProfileSummaryStat(ctx context.Context, userID uint) (int64, int64, int64, error) {
	s.logger.Info("StatService.GetProfileSummaryStat", zap.Uint("user_id", userID))

	marksCount, err := s.markStat.GetUserMarksCount(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get marks count")
	}
	return marksCount, rand.Int63(), rand.Int63(), nil

}

// GetUserMonthlyActivity Формирует данные для предоставления графика активности по месяцам в течении текущего года
func (s *StatService) GetUserMonthlyActivity(ctx context.Context, userID uint) ([]*mark.MonthlyActivity, error) {
	s.logger.Info("StatService.GetUserMonthlyActivity", zap.Uint("user_id", userID))
	year := time.Now().Year()
	activities, err := s.markStat.GetUserMarksMonthlyActivity(ctx, userID, year)
	if err != nil {
		s.logger.Warn("failed to get marks monthly activity")
		return nil, err
	}
	return activities, nil
}

// GetUserMarksHeatMap делает запрос к gRPC сервису для получение данных используемых для формирования Тепловой карты активности
func (s *StatService) GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*mark.HeatMapItem, error) {
	s.logger.Info("StatService.GetUserMarksHeatMap", zap.Uint("user_id", userID))
	marks, err := s.markStat.GetUserMarksHeatMap(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn("failed to get marks heat map")
		return nil, err
	}
	return marks, nil
}
