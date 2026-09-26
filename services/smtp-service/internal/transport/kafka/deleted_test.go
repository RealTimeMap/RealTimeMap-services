package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	segmentio "github.com/segmentio/kafka-go"
)

// recordingEraser запоминает адреса, письма на которые просили удалить.
type recordingEraser struct {
	emails []string
	err    error
}

func (r *recordingEraser) DeleteByRecipient(_ context.Context, toEmail string) (int64, error) {
	r.emails = append(r.emails, toEmail)
	return 1, r.err
}

func TestHandlerUserDeleted(t *testing.T) {
	deleted := segmentio.Message{Value: []byte(
		`{"type":"user.deleted","payload":{"user_id":42,"email":"User@Example.com"}}`)}

	t.Run("удаляет письма на адрес из события", func(t *testing.T) {
		enq, eraser := &recordingEnqueuer{}, &recordingEraser{}
		h := NewHandler(enq, &stubUsers{}, eraser, "https://realtimemap.ru", logger.NewNop())

		if err := h.HandleMessage(context.Background(), deleted); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if len(eraser.emails) != 1 || eraser.emails[0] != "User@Example.com" {
			t.Fatalf("erased %v, want [User@Example.com]", eraser.emails)
		}
		if enq.count() != 0 {
			t.Fatalf("удаление аккаунта не должно ставить писем, поставлено %d", enq.count())
		}
	})

	t.Run("сбой БД — повтор", func(t *testing.T) {
		eraser := &recordingEraser{err: errors.New("db down")}
		h := NewHandler(&recordingEnqueuer{}, &stubUsers{}, eraser, "https://realtimemap.ru", logger.NewNop())

		if err := h.HandleMessage(context.Background(), deleted); !errors.Is(err, consumer.ErrRetryable) {
			t.Fatalf("err = %v, want retryable", err)
		}
	})

	t.Run("без адреса — пропуск", func(t *testing.T) {
		eraser := &recordingEraser{}
		h := NewHandler(&recordingEnqueuer{}, &stubUsers{}, eraser, "https://realtimemap.ru", logger.NewNop())

		err := h.HandleMessage(context.Background(), segmentio.Message{Value: []byte(
			`{"type":"user.deleted","payload":{"user_id":42}}`)})
		if !errors.Is(err, consumer.ErrSkip) {
			t.Fatalf("err = %v, want skip", err)
		}
		if len(eraser.emails) != 0 {
			t.Fatalf("erased %v, want nothing", eraser.emails)
		}
	})
}
