package stats

import (
	"context"

	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/stats"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (h *Handler) GetUserMarksMonthlyActivity(ctx context.Context, req *markstat.MarksMonthlyActivityRequest) (*markstat.UserMarksActivityResponse, error) {
	activities, err := h.service.GetUserMonthlyActivity(ctx, uint(req.GetUserId()), int(req.GetYear()))
	if err != nil {
		h.logger.Error("GetUserMarksMonthlyActivity error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toActivityResponse(activities), nil
}

func (h *Handler) GetUserMarksHeatMap(ctx context.Context, req *markstat.MarksHeatMapRequest) (*markstat.MarksHeatMapResponse, error) {
	activities, err := h.service.GetCountsForPeriod(ctx, uint(req.GetUserId()), req.GetStartDate().AsTime(), req.GetEndDate().AsTime())
	if err != nil {
		h.logger.Error("GetUserMarksHeatMap error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toDayActivity(activities), nil
}

func toDayActivity(data []model.DayActivity) *markstat.MarksHeatMapResponse {
	result := make([]*markstat.MarkHeatMapResponse, 0, len(data))
	for _, d := range data {
		result = append(result, &markstat.MarkHeatMapResponse{
			Day:   timestamppb.New(d.Day),
			Count: d.Count,
		})
	}
	return &markstat.MarksHeatMapResponse{
		Activity: result,
	}
}

func toActivityResponse(data []model.MonthlyActivity) *markstat.UserMarksActivityResponse {
	results := make([]*markstat.MarkMonthResponse, 0, len(data))
	for _, item := range data {
		results = append(results, &markstat.MarkMonthResponse{
			Month: item.Month,
			Count: item.Count,
		})
	}
	return &markstat.UserMarksActivityResponse{
		Activities: results,
	}
}

func toResponse(count int64) *markstat.MarksCountResponse {
	return &markstat.MarksCountResponse{
		Count: count,
	}
}
