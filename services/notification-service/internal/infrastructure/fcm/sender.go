package fcm

import (
	"context"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
)

type FirebaseSender struct {
	cli *messaging.Client

	logger *zap.Logger
}

var androidTTL = 1 * time.Hour

func NewFirebaseSender(ctx context.Context, projectID string, logger *zap.Logger) (notification.Sender, error) {
	config := &firebase.Config{ProjectID: projectID}

	sdk, err := firebase.NewApp(ctx, config)
	if err != nil {
		return nil, err
	}

	cli, err := sdk.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebaseSender{
		cli:    cli,
		logger: logger,
	}, nil
}

func (s *FirebaseSender) Send(ctx context.Context, mess notification.Message) error {
	message := &messaging.Message{
		Data: mess.Data,
		Notification: &messaging.Notification{
			Title: mess.Title,
			Body:  mess.Content,
		},
		Android: &messaging.AndroidConfig{
			TTL: &androidTTL,
			Notification: &messaging.AndroidNotification{
				Color: "#973bff",
			},
		},
		Token: mess.Token,
	}
	response, err := s.cli.Send(ctx, message)
	if err != nil {
		err = classify(err)
		s.logger.Error("firebase sending via error", zap.Error(err))
		return err
	}

	s.logger.Info("send notification", zap.String("result", response))
	return nil
}
