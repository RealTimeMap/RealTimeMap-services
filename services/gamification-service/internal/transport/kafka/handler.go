package kafka

import (
	"context"
	"strconv"

	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/achievement"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/event"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Handler struct {
	service    *event.Service
	achService achievement.Service
	logger     *zap.Logger
}

func NewHandler(service *event.Service, achService achievement.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service:    service,
		achService: achService,
		logger:     logger,
	}
}

// HandleMessage обрабатывает входящее Kafka сообщение.
func (h *Handler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	meta := pkgkafka.ExtractMeta(msg)

	h.logger.Debug("received kafka message",
		zap.String("event_type", meta.EventType),
		zap.String("user_id", meta.UserID),
		zap.String("source_id", meta.SourceID),
	)

	if meta.UserID == "" {
		h.logger.Warn("missing user_id in message headers")
		return consumer.Skip(nil)
	}

	if meta.EventType == "" {
		h.logger.Warn("missing event_type in message headers")
		return consumer.Skip(nil)
	}

	userID, err := strconv.ParseUint(meta.UserID, 10, 64)
	if err != nil {
		h.logger.Warn("invalid user_id format", zap.String("user_id", meta.UserID), zap.Error(err))
		return consumer.Skip(err)
	}

	sourceID, err := parseOptionalUint(meta.SourceID)
	if err != nil {
		h.logger.Warn("invalid source_id format", zap.String("source_id", meta.SourceID), zap.Error(err))
		return consumer.Skip(err)
	}

	err = h.service.GreatUserExp(ctx, uint(userID), meta.EventType, sourceID)
	if err != nil {
		h.logger.Warn("error in gamification service", zap.Error(err))
		return consumer.Skip(err)
	}

	h.achService.OnEvent(ctx, uint(userID), meta.EventType)

	return nil
}

// parseOptionalUint парсит строку в *uint.
func parseOptionalUint(s string) (*uint, error) {
	if s == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, err
	}
	id := uint(parsed)
	return &id, nil
}
