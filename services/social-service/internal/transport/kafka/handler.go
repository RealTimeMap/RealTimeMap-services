package kafka

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Handler struct {
	service *profile.Service

	logger *zap.Logger
}

func NewHandler(service *profile.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type User struct {
	Username string `json:"username"`
	UserID   uint   `json:"user_id"`
}

func (h *Handler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	var user User
	if err := json.Unmarshal(msg.Value, &user); err != nil {
		h.logger.Error("Error unmarshalling message", zap.Error(err))
		return consumer.Skip(err)
	}

	h.logger.Info("Message received", zap.Any("user", user))

	_, err := h.service.CreateProfile(ctx, profile.CreateProfileInput{
		Username: user.Username,
		UserID:   user.UserID,
	})
	if err != nil {
		var conflictErr *apperror.ConflictError
		if errors.As(err, &conflictErr) {
			h.logger.Info("profile already exists, skipping",
				zap.Uint("user_id", user.UserID))
			return consumer.Skip(err)
		}

		h.logger.Error("Error creating profile", zap.Error(err))
		return consumer.Retryable(err)
	}

	return nil
}
