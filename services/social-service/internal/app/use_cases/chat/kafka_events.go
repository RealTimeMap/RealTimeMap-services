package chat

import (
	"context"

	"go.uber.org/zap"
)

// MessageEventInput — то, что нужно шине от отправленного сообщения.
//
// Отдельный тип, а не MessageResult: тот собран для ответа клиенту и несёт
// поля realtime-контракта (ClientMessageID для дедупа эха), которые в топике
// не нужны. Зато нужны данные чата — групповой он и как называется, — которых
// в MessageResult нет.
type MessageEventInput struct {
	MessageID uint
	ChatID    uint
	SenderID  uint

	SenderName string
	ChatTitle  string
	IsGroup    bool

	Content string

	// RecipientIDs — участники чата, включая отправителя: чистит список уже
	// публикатор, он же решает, кому пуш не нужен.
	RecipientIDs []uint
}

// KafkaPublisher — порт publish-в-шину для событий чата.
//
// Отдельный от EventPublisher: тот доставляет realtime в сокеты открытого
// приложения, этот — межсервисные события для тех, у кого приложение закрыто.
// У них разные получатели, разная семантика доставки и разные последствия
// сбоя, поэтому и порты разные.
type KafkaPublisher interface {
	PublishMessageCreated(ctx context.Context, in MessageEventInput) error
}

// NoOpKafkaPublisher — заглушка на случай выключенного продюсера.
//
// Kafka не должна быть обязательной для работы чата: без неё сообщения всё
// так же отправляются и доходят по сокету, теряются только push-уведомления.
type NoOpKafkaPublisher struct{}

func (NoOpKafkaPublisher) PublishMessageCreated(context.Context, MessageEventInput) error {
	return nil
}

// ChatInfoGetter отдаёт данные чата для текста уведомления.
type ChatInfoGetter interface {
	ChatInfo(ctx context.Context, chatID uint) (title string, isGroup bool, err error)
}

// noopChatInfo используется, когда источник данных о чате не задан.
type noopChatInfo struct{}

func (noopChatInfo) ChatInfo(context.Context, uint) (string, bool, error) {
	return "", false, nil
}

// logKafkaFailure логирует сбой публикации.
//
// Отдельная функция, потому что реакция всюду одна: событие в шину —
// best-effort, сообщение уже сохранено и доставлено по сокету. Ронять из-за
// этого HTTP-ответ нельзя — клиент ретраит и создаёт дубликат.
func logKafkaFailure(logger *zap.Logger, chatID uint, err error) {
	logger.Warn("failed to publish chat event to kafka",
		zap.Uint("chat_id", chatID),
		zap.Error(err),
	)
}
