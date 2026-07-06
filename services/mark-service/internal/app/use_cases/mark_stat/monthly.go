package mark_stat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type MarkMonthGetter interface {
	GetUserMonthlyActivity(ctx context.Context, userID uint, year int) ([]mark.MonthlyActivity, error)
}

type MarkMonthHandler struct {
	getter MarkMonthGetter
	logger *zap.Logger
}

func NewMarkMonthHandler(getter MarkMonthGetter, logger *zap.Logger) *MarkMonthHandler {
	return &MarkMonthHandler{
		getter: getter,
		logger: logger,
	}
}

func (s *MarkMonthHandler) Handle(ctx context.Context, userID uint, year int) ([]MonthlyActivityResult, error) {
	objs, err := s.getter.GetUserMonthlyActivity(ctx, userID, year)
	if err != nil {
		return []MonthlyActivityResult{}, err
	}
	return toMultiMonthlyActivityResult(objs), nil
}
