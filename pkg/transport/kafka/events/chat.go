package events

const (
	// ChatMessageCreated — новое сообщение в чате.
	//
	// Событие дублирует realtime-доставку через Socket.IO, а не заменяет её:
	// сокет обслуживает открытое приложение, Kafka — push тем, у кого оно
	// закрыто. Сокет для этого не годится — у отключённого клиента комнаты
	// нет, и событие просто некому доставить.
	ChatMessageCreated = "chat.message.created"
)

type ChatMessageEvent struct {
	Envelop
	Payload ChatMessagePayload `json:"payload"`
}

// ChatMessagePayload — сообщение чата на момент отправки.
//
// RecipientIDs едет в событии, а не резолвится потребителем: состав участников
// знает только social-service, а gRPC-ручки «участники чата» в proto нет.
// Автор в список не входит — отправитель не уведомляется о собственном
// сообщении.
type ChatMessagePayload struct {
	MessageID uint `json:"messageId"`
	ChatID    uint `json:"chatId"`
	SenderID  uint `json:"senderId"`

	// SenderName — имя для заголовка уведомления. Без него потребителю
	// пришлось бы ходить за профилем на каждое сообщение.
	SenderName string `json:"senderName,omitempty"`

	// ChatTitle заполняется только для групповых чатов: в direct заголовком
	// служит имя отправителя.
	ChatTitle string `json:"chatTitle,omitempty"`
	IsGroup   bool   `json:"isGroup"`

	// Preview — усечённый текст сообщения для тела уведомления. Полный текст
	// в топик не кладётся: переписка не должна лежать там по retention.
	Preview string `json:"preview,omitempty"`

	RecipientIDs []uint `json:"recipientIds"`
}

func NewChatMessageCreated(payload ChatMessagePayload) ChatMessageEvent {
	return ChatMessageEvent{
		Envelop: NewEnvelop(ChatMessageCreated),
		Payload: payload,
	}
}
