package kafka

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app/use_cases/chat"
)

// previewLimit — сколько символов сообщения уезжает в событие.
//
// Обрезка здесь, а не у потребителя: текст переписки живёт в топике по
// retention, и класть туда сообщение целиком незачем — в пуш всё равно
// попадает только начало.
const previewLimit = 160

// ChatPublisher публикует события чата в Kafka.
//
// Дублирует realtime-доставку через Socket.IO, а не заменяет её: сокет
// обслуживает открытое приложение, Kafka — push тем, у кого оно закрыто.
type ChatPublisher struct {
	producer *producer.Producer
	topic    string
	logger   *zap.Logger
}

func NewChatPublisher(p *producer.Producer, topic string, logger *zap.Logger) chat.KafkaPublisher {
	return &ChatPublisher{producer: p, topic: topic, logger: logger}
}

// PublishMessageCreated отправляет событие о новом сообщении.
//
// Ключ сообщения — id отправителя (его кладёт PublishWithMeta из меты):
// события одного пользователя попадают в одну партицию и приходят потребителю
// в порядке публикации.
func (p *ChatPublisher) PublishMessageCreated(ctx context.Context, in chat.MessageEventInput) error {
	recipients := withoutSender(in.RecipientIDs, in.SenderID)
	if len(recipients) == 0 {
		// Чат с самим собой или пустой список: публиковать нечего, и пустое
		// событие только заставило бы потребителя разбирать его впустую.
		return nil
	}

	payload := events.ChatMessagePayload{
		MessageID:    in.MessageID,
		ChatID:       in.ChatID,
		SenderID:     in.SenderID,
		SenderName:   in.SenderName,
		ChatTitle:    in.ChatTitle,
		IsGroup:      in.IsGroup,
		Preview:      truncate(in.Content, previewLimit),
		RecipientIDs: recipients,
	}

	senderID := strconv.FormatUint(uint64(in.SenderID), 10)
	meta := producer.EventMeta{
		EventType: events.ChatMessageCreated,
		UserID:    senderID,
		SourceID:  strconv.FormatUint(uint64(in.ChatID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := p.producer.PublishWithMeta(ctx, meta, events.NewChatMessageCreated(payload)); err != nil {
		p.logger.Error("failed to publish chat message event",
			zap.String("event_type", events.ChatMessageCreated),
			zap.Uint("chat_id", in.ChatID),
			zap.Uint("message_id", in.MessageID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published chat message event",
		zap.String("event_type", events.ChatMessageCreated),
		zap.String("topic", p.topic),
		zap.Uint("chat_id", in.ChatID),
		zap.Int("recipients", len(recipients)),
	)
	return nil
}

// withoutSender убирает отправителя из списка получателей.
//
// RecipientIDs в social включает его намеренно — сокету нужно эхо на другие
// устройства отправителя. Для пуша это лишнее: уведомление о собственном
// сообщении не нужно никому.
func withoutSender(ids []uint, senderID uint) []uint {
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id != senderID {
			out = append(out, id)
		}
	}
	return out
}

// truncate режет текст по рунам, не разваливая многобайтные символы.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}
