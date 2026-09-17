// Package kafka публикует доменные события social-service в шину.
package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"go.uber.org/zap"
)

type ProfilePublisher struct {
	producer *producer.Producer
	logger   *zap.Logger
}

func NewProfilePublisher(p *producer.Producer, logger *zap.Logger) profile.EventPublisher {
	return &ProfilePublisher{producer: p, logger: logger}
}

// PublishProfileUpdated отправляет актуальное состояние профиля.
//
// Ключ сообщения — user_id (его кладёт PublishWithMeta из меты): события
// одного пользователя попадают в одну партицию и приходят потребителю в
// порядке публикации. Без этого два быстрых редактирования подряд могли бы
// примениться в auth задом наперёд, оставив там старый username.
func (p *ProfilePublisher) PublishProfileUpdated(ctx context.Context, prof *model.Profile) error {
	payload := events.ProfilePayload{
		UserID:   prof.UserID,
		Username: prof.Username,
		Tag:      prof.Tag,
		Avatar:   prof.Avatar.URL,
	}

	userID := strconv.FormatUint(uint64(prof.UserID), 10)
	meta := producer.EventMeta{
		EventType: events.ProfileUpdated,
		UserID:    userID,
		SourceID:  userID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := p.producer.PublishWithMeta(ctx, meta, events.NewProfileUpdated(payload)); err != nil {
		p.logger.Error("failed to publish profile event",
			zap.String("event_type", events.ProfileUpdated),
			zap.Uint("user_id", prof.UserID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published profile event",
		zap.String("event_type", events.ProfileUpdated),
		zap.Uint("user_id", prof.UserID),
	)
	return nil
}

func (p *ProfilePublisher) Close() error {
	return p.producer.Close()
}
