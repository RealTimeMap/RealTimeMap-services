package stats

import (
	"context"

	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/stats"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	markstat.MarkStatsServiceServer

	service *stats.MarkStatsService
	logger  *zap.Logger
}

func NewHandler(service *stats.MarkStatsService, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) GetUserMarksCount(ctx context.Context, req *markstat.MarksCountRequest) (*markstat.MarksCountResponse, error) {
	count, err := h.service.GetMarkCount(ctx, uint(req.GetUserId()))
	if err != nil {
		h.logger.Error("GetUserMarksCount error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toResponse(count), nil
}

func toResponse(count int64) *markstat.MarksCountResponse {
	return &markstat.MarksCountResponse{
		Count: count,
	}
}
