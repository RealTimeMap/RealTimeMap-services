package accrual

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	markRepo    repository.MarkRepository
	accrualRepo repository.AccrualRepository

	logger *zap.Logger
}

func NewService(markRepo repository.MarkRepository, accrualRepo repository.AccrualRepository, logger *zap.Logger) *Service {
	return &Service{
		markRepo:    markRepo,
		accrualRepo: accrualRepo,
		logger:      logger,
	}
}

func (s *Service) IncreaseShare(ctx context.Context, markID uint) (int64, error) {
	s.logger.Info("IncreaseShare", zap.Uint("markID", markID))
	if err := s.checkMarkExist(ctx, markID); err != nil {
		return 0, err
	}

	count, err := s.accrualRepo.IncShare(ctx, markID)
	if err != nil {
		s.logger.Error("IncreaseShare. Accrual error", zap.Uint("markID", markID))
		return 0, err
	}
	return count, nil
}

func (s *Service) SetLike(ctx context.Context, markID, userID uint) error {
	if err := s.checkMarkExist(ctx, markID); err != nil {
		return err
	}
	err := s.accrualRepo.Like(ctx, markID, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) checkMarkExist(ctx context.Context, markID uint) error {
	exist, err := s.markRepo.Exist(ctx, int(markID))
	if err != nil {
		s.logger.Error("IncreaseShare. Mark error", zap.Uint("markID", markID))
		return err
	}
	if !exist {
		s.logger.Warn("IncreaseShare. Mark not exist", zap.Uint("markID", markID))
		return domainerrors.ErrMarkNotFound(int(markID))
	}
	return nil
}
