package profile

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

const topN int = 3

type MarkStatGetter interface {
	GetUserMarksCount(ctx context.Context, userID uint) (int64, error)
	GetUserMarksMonthlyActivity(ctx context.Context, userID uint, year int) ([]*mark.MonthlyActivity, error)
	GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*mark.HeatMapItem, error)
	GetPopularUserCategories(ctx context.Context, userID uint, topN int) ([]*mark.PopularCategory, error)
}

type StatService struct {
	markStat   MarkStatGetter
	friendRepo repository.FriendShipRepository
	logger     *zap.Logger
}

func NewStatService(markStat MarkStatGetter, friendRepo repository.FriendShipRepository, logger *zap.Logger) *StatService {
	return &StatService{
		markStat:   markStat,
		friendRepo: friendRepo,
		logger:     logger,
	}
}

// GetProfileSummaryStat Формирует Summary статистику для отображения в профиле
func (s *StatService) GetProfileSummaryStat(ctx context.Context, userID uint) (int64, int64, int64, error) {
	s.logger.Info("StatService.GetProfileSummaryStat", zap.Uint("user_id", userID))

	marksCount, err := s.markStat.GetUserMarksCount(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get marks count")
	}

	friendCount, subsCount, err := s.friendRepo.CountFriendAndSubs(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get marks count")
	}

	return marksCount, friendCount, subsCount, nil
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

// GetUserMarksHeatMap делает запрос к gRPC сервису для получения данных используемых для формирования Тепловой карты активности
func (s *StatService) GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*mark.HeatMapItem, error) {
	s.logger.Info("StatService.GetUserMarksHeatMap", zap.Uint("user_id", userID))

	if err := validateDateRange(start, end); err != nil {
		return nil, err
	}

	marks, err := s.markStat.GetUserMarksHeatMap(ctx, userID, start, end)
	if err != nil {
		s.logger.Warn("failed to get marks heat map")
		return nil, err
	}
	return marks, nil
}

func (s *StatService) GetPopularCategories(ctx context.Context, userID uint) ([]*mark.PopularCategory, error) {
	s.logger.Info("StatService.GetPopularCategories", zap.Uint("user_id", userID))

	categories, err := s.markStat.GetPopularUserCategories(ctx, userID, topN)
	if err != nil {
		s.logger.Warn("failed to get marks categories")
		return nil, err
	}
	return categories, nil
}

func validateDateRange(start, end time.Time) error {
	if start.After(end) {
		return domainerrors.DateValidationErr("start", "must be before end", start)
	}
	if end.After(time.Now().AddDate(0, 0, 1)) {
		return domainerrors.DateValidationErr("end", "cannot be greater than the current date", end)
	}
	return nil
}
