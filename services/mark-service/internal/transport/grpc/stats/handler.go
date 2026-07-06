package stats

import (
	"context"

	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_stat"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	markstat.MarkStatsServiceServer

	useCase *mark_stat.Application
	logger  *zap.Logger
}

func NewHandler(useCase *mark_stat.Application, logger *zap.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) GetUserMarksCount(ctx context.Context, req *markstat.MarksCountRequest) (*markstat.MarksCountResponse, error) {
	count, err := h.useCase.GetMarkCount.Handle(ctx, uint(req.GetUserId()))
	if err != nil {
		h.logger.Error("GetUserMarksCount error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toCountResponse(count), nil
}

func (h *Handler) GetUserMarksMonthlyActivity(ctx context.Context, req *markstat.MarksMonthlyActivityRequest) (*markstat.UserMarksActivityResponse, error) {
	activities, err := h.useCase.GetMonthActivity.Handle(ctx, uint(req.GetUserId()), int(req.GetYear()))
	if err != nil {
		h.logger.Error("GetUserMarksMonthlyActivity error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toActivityResponse(activities), nil
}

func (h *Handler) GetUserMarksHeatMap(ctx context.Context, req *markstat.MarksHeatMapRequest) (*markstat.MarksHeatMapResponse, error) {
	activities, err := h.useCase.GetHeatMap.Handle(ctx, uint(req.GetUserId()), req.GetStartDate().AsTime(), req.GetEndDate().AsTime())
	if err != nil {
		h.logger.Error("GetUserMarksHeatMap error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toDayActivity(activities), nil
}

func (h *Handler) GetPopularUserCategories(ctx context.Context, req *markstat.PopularCategoriesRequest) (*markstat.PopularCategoriesResponse, error) {
	categories, err := h.useCase.GetCategories.Handle(ctx, uint(req.GetUserId()), int(req.GetTopN()))
	if err != nil {
		h.logger.Error("GetPopularUserCategories error", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal err")
	}
	return toPopularCategoriesResponse(categories), nil
}

func toDayActivity(data []mark_stat.DayActivityResult) *markstat.MarksHeatMapResponse {
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

func toActivityResponse(data []mark_stat.MonthlyActivityResult) *markstat.UserMarksActivityResponse {
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

func toPopularCategoriesResponse(data []mark_stat.CategoryStatResult) *markstat.PopularCategoriesResponse {
	results := make([]*markstat.PopularCategoriesItem, 0, len(data))
	for _, item := range data {
		results = append(results, &markstat.PopularCategoriesItem{
			CategoryName: item.CategoryName,
			Percent:      item.Percent,
			Count:        item.Count,
		})
	}
	return &markstat.PopularCategoriesResponse{
		Categories: results,
	}
}

func toCountResponse(count int64) *markstat.MarksCountResponse {
	return &markstat.MarksCountResponse{
		Count: count,
	}
}
