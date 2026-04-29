package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedProfileServiceServer

	service *profile.Service
	logger  *zap.Logger
}

func NewHandler(service *profile.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) GetUserProfileByID(ctx context.Context, req *pb.ProfileRequest) (*pb.ProfileResponse, error) {
	p, err := h.service.GetProfile(ctx, uint(req.GetId()))
	if err != nil {
		var notFound *apperror.NotFoundError
		if errors.As(err, &notFound) {
			return nil, status.Errorf(codes.NotFound, "profile %d not found", req.GetId())
		}
		h.logger.Error("GetUserProfileByID failed", zap.Error(err), zap.Uint64("id", req.GetId()))
		return nil, status.Error(codes.Internal, "internal error")
	}
	return toResponse(p), nil
}

func (h *Handler) GetUserProfileByIDs(ctx context.Context, req *pb.MultipleProfileRequest) (*pb.MultipleProfileResponse, error) {
	ids := make([]uint, 0, len(req.GetIds()))
	for _, r := range req.GetIds() {
		ids = append(ids, uint(r.GetId()))
	}

	profiles, err := h.service.GetProfilesByIDs(ctx, ids)
	if err != nil {
		h.logger.Error("GetUserProfileByIDs failed", zap.Error(err), zap.Int("ids_count", len(ids)))
		return nil, status.Error(codes.Internal, "internal error")
	}

	out := make([]*pb.ProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, toResponse(p))
	}
	return &pb.MultipleProfileResponse{Profiles: out}, nil
}

func toResponse(p *model.Profile) *pb.ProfileResponse {
	fmt.Println(p)
	return &pb.ProfileResponse{
		Id:       uint64(p.UserID),
		Username: p.Username,
		Tag:      p.Tag,
		Avatar:   p.Avatar.URL,
	}
}
