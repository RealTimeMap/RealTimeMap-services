package xp

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/level"
	"go.uber.org/zap"
)

// XPOperator олператор начисления опыта
type XPOperator struct {
	operationRepo repository.XPOperationRepository
	progressRepo  repository.UserProgressRepository

	levelService *level.Service
	tx           txmanager.TxManager

	logger *zap.Logger
}

func NewXPOperator(operationRepo repository.XPOperationRepository, progressRepo repository.UserProgressRepository, levelService *level.Service, tx txmanager.TxManager, logger *zap.Logger) *XPOperator {
	return &XPOperator{
		operationRepo: operationRepo,
		progressRepo:  progressRepo,
		levelService:  levelService,
		logger:        logger,
		tx:            tx,
	}
}

// Credit Функция начисления опыта
func (o *XPOperator) Credit(ctx context.Context, input CreditInput) (*CreditResult, error) {
	o.logger.Info("XPOperator.Credit", zap.Any("input", input))
	var result *CreditResult

	err := o.tx.WithTx(ctx, func(txCtx context.Context) error {
		op, err := o.operationRepo.Create(txCtx, &model.XPOperation{
			UserID:     input.UserID,
			Amount:     input.Amount,
			SourceType: input.SourceType,
			SourceID:   input.SourceID,
			Reason:     input.Reason,
		})
		if err != nil {
			return err
		}

		progress, err := o.progressRepo.GetOrCreate(txCtx, input.UserID)
		if err != nil {
			return err
		}
		progress.CurrentXP = uint(int(progress.CurrentXP) + input.Amount)

		levelUp, err := o.levelService.RecalculateLevel(txCtx, progress)
		if err != nil {
			return err
		}

		progress, err = o.progressRepo.Update(txCtx, progress)
		if err != nil {
			return err
		}

		result = &CreditResult{
			LeveledUp: levelUp,
			Operation: op,
			Progress:  progress,
		}
		return nil
	})
	return result, err
}
