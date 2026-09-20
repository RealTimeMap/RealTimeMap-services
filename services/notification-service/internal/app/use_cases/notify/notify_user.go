package notify

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
)

type TokenGetter interface {
	GetUserDevices(ctx context.Context, userID uint) ([]token.Model, error)
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

	// Kind — тип уведомления, от которого устройство могло отписаться. Пустой
	// означает системную отправку (тест, административная рассылка): такая
	// проходит на все устройства, где уведомления не выключены целиком.
	Kind token.Kind
}

// Handle рассылает уведомление на устройства пользователя, спрашивая каждое,
// принимает ли оно такой тип.
//
// Настройки проверяются поштучно, а не одним условием на пользователя: в этом
// вся суть их привязки к устройству — выключенные уведомления на ПК не мешают
// тому же уведомлению прийти на телефон.
func (h *UserNotifyHanlder) Handle(ctx context.Context, cmd NotifyUserCommand) error {
	h.logger.Info("start Handle", zap.String("use_case", "notify use case"))

	devices, err := h.getter.GetUserDevices(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	var skipped int
	for _, obj := range devices {
		if !obj.Allows(cmd.Kind) {
			skipped++
			continue
		}

		err := h.fcm.Send(ctx, notification.Message{
			Token:   obj.Token,
			Title:   cmd.Title,
			Content: cmd.Content,
		})
		if err != nil {
			h.logger.Warn("failed send message", zap.Error(err))
		}
	}

	if skipped > 0 {
		h.logger.Debug("notification muted on some devices",
			zap.Uint("user_id", cmd.UserID),
			zap.String("kind", string(cmd.Kind)),
			zap.Int("skipped", skipped),
			zap.Int("total", len(devices)),
		)
	}
	return nil
}
