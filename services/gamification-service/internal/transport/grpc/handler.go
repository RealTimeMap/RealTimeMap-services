package grpc

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/proto/gamification"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelservice"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements pb.ProgressServiceServer interface
type Handler struct {
	pb.UnimplementedProgressServiceServer

	progressRepo repository.UserProgressRepository
	levelService *levelservice.LevelService
	logger       *zap.Logger
}

func NewHandler(
	progressRepo repository.UserProgressRepository,
	levelService *levelservice.LevelService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		progressRepo: progressRepo,
		levelService: levelService,
		logger:       logger,
	}
}

// GetUserProgress получение прогресса пользователя
func (h *Handler) GetUserProgress(ctx context.Context, req *pb.GetUserProgressRequest) (*pb.UserProgressResponse, error) {
	h.logger.Info("GetUserProgress called", zap.Uint64("user_id", req.GetUserId()))

	// Получаем прогресс из бд
	progress, err := h.progressRepo.GetOrCreate(ctx, uint(req.GetUserId()))
	if err != nil {
		var notFoundErr *apperror.NotFoundError
		if errors.As(err, &notFoundErr) { // Если не найдено
			return nil, status.Errorf(codes.NotFound, "user progress not found for user_id: %d", req.GetUserId())
		}
		h.logger.Error("failed to get user progress", zap.Error(err)) // Логируем все остальные ошибки
		return nil, status.Errorf(codes.Internal, "failed to get user progress: %v", err)
	}

	nextLevel, xpForNext, percent, err := h.calculateLevelMeta(ctx, progress)
	if err != nil {
		h.logger.Error("failed to get user progress meta", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get user progress meta: %v", err)
	}

	// Собираем ответ
	return &pb.UserProgressResponse{
		UserId:          req.GetUserId(),
		CurrentLevel:    uint64(progress.CurrentLevel),
		CurrentXp:       uint64(progress.CurrentXP),
		NextLevel:       nextLevel,
		XpForNextLevel:  xpForNext,
		ProgressPercent: percent,
	}, nil
}

// calculateLevelMeta вычисляет информацию для следующего уровня
func (h *Handler) calculateLevelMeta(ctx context.Context, progress *model.UserProgress) (nextLevel, XpForNextLevel uint64, percent float64, err error) {
	nLevel, err := h.levelService.GetNextLevel(ctx, progress.CurrentLevel)
	if err != nil {
		h.logger.Error("failed to get next level", zap.Error(err))
		return 0, 0, 0, err
	}

	progressPercent := nLevel.Percent(float64(progress.CurrentXP))

	return uint64(nLevel.Level), uint64(nLevel.XPRequired), progressPercent, nil
}
