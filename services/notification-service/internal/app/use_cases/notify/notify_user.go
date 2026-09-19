package notify

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
)

type TokenGetter interface {
	GetUserTokens(ctx context.Context, userID uint) ([]token.Model, error)
}

type FCM interface {
	Send(ctx context.Context, mess notification.Message) error
}

type UserNotifyHanlder struct {
	getter TokenGetter
	fcm    FCM

	logger *zap.Logger
}

func NewUserNotifyHanlder(getter TokenGetter, fcm FCM, logger *zap.Logger) *UserNotifyHanlder {
	return &UserNotifyHanlder{
		getter: getter,
		fcm:    fcm,
		logger: logger,
	}
}

type NotifyUserCommand struct {
	UserID  uint
	Title   string
	Content string
}

func (h *UserNotifyHanlder) Handle(ctx context.Context, cmd NotifyUserCommand) error {
	h.logger.Info("start Handle", zap.String("use_case", "notify use case"))

	tokens, err := h.getter.GetUserTokens(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	for _, obj := range tokens {
		err := h.fcm.Send(ctx, notification.Message{
			Token:   obj.Token,
			Title:   cmd.Title,
			Content: cmd.Content,
		})
		if err != nil {
			h.logger.Warn("failed send message", zap.Error(err))
		}
	}
	return nil
}
