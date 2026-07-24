package dto

import (
	"time"

	chatuc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app/use_cases/chat"
)

// DTO-слой чата. Отдельные транспортные структуры с camelCase json-тегами,
// чтобы не тащить теги в useCase. Мапперы принимают useCase-результаты
// (chatuc.*Result) и превращают их в стабильный внешний контракт.
//
// MessageResponse используется и HTTP-хендлерами, и socket-publisher-ом
// (событие message.new), поэтому имена полей должны совпадать в обоих каналах.

// SenderResponse — краткий профиль автора/участника.
type SenderResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func NewSenderResponse(p chatuc.ProfileResult) SenderResponse {
	return SenderResponse{
		ID:       p.ID,
		Username: p.Username,
		Avatar:   p.Avatar,
	}
}

// MessageResponse — сообщение чата. Единый контракт для HTTP-ответа и socket-события.
type MessageResponse struct {
	ID   uint   `json:"id"`
	Type string `json:"type"`
	// SenderID — id автора на верхнем уровне (дублирует sender.id) для удобного UI:
	// клиент сразу решает «своё/чужое» без разворачивания sender.
	SenderID uint   `json:"senderId"`
	Content  string `json:"content"`
	// ClientMessageID — UUID, присланный клиентом при отправке (null, если не было).
	// Клиент по нему матчит оптимистичное сообщение с серверным и дедупит эхо.
	ClientMessageID *string        `json:"clientMessageId"`
	CreatedAt       time.Time      `json:"createdAt"`
	ChatID          uint           `json:"chatId"`
	Sender          SenderResponse `json:"sender"`
}

func NewMessageResponse(m chatuc.MessageResult) MessageResponse {
	return MessageResponse{
		ID:              m.ID,
		Type:            m.Type,
		SenderID:        m.SenderID,
		Content:         m.Content,
		ClientMessageID: m.ClientMessageID,
		CreatedAt:       m.CreatedAt,
		ChatID:          m.ChatID,
		Sender:          NewSenderResponse(m.Sender),
	}
}

// ReadResponse — событие chat.read: участник продвинул курсор прочтения. Уходит
// только через socket (у HTTP /read тело ответа пустое, 204).
type ReadResponse struct {
	ChatID            uint `json:"chatId"`
	UserID            uint `json:"userId"`
	LastReadMessageID uint `json:"lastReadMessageId"`
}

func NewReadResponse(r chatuc.ReadResult) ReadResponse {
	return ReadResponse{
		ChatID:            r.ChatID,
		UserID:            r.UserID,
		LastReadMessageID: r.LastReadMessageID,
	}
}

// DirectChatResponse — ответ создания/получения личного чата.
type DirectChatResponse struct {
	ChatID    uint           `json:"chatId"`
	Peer      SenderResponse `json:"peer"`
	CreatedAt time.Time      `json:"createdAt"`
}

func NewDirectChatResponse(r chatuc.DirectChatResult) DirectChatResponse {
	return DirectChatResponse{
		ChatID:    r.ChatID,
		Peer:      NewSenderResponse(r.Peer),
		CreatedAt: r.CreatedAt,
	}
}

// GroupChatResponse — ответ создания группового чата.
type GroupChatResponse struct {
	ChatID    uint      `json:"chatId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewGroupChatResponse(r chatuc.GroupChatResult) GroupChatResponse {
	return GroupChatResponse{
		ChatID:    r.ChatID,
		Title:     r.Title,
		CreatedAt: r.CreatedAt,
	}
}

// MessageHistoryResponse — страница истории с курсором для keyset-пагинации.
type MessageHistoryResponse struct {
	Messages      []MessageResponse `json:"messages"`
	LastMessageID *uint             `json:"lastMessageId"`
}

func NewMessageHistoryResponse(r chatuc.MessageHistoryResult) MessageHistoryResponse {
	messages := make([]MessageResponse, 0, len(r.Messages))
	for _, m := range r.Messages {
		messages = append(messages, NewMessageResponse(m))
	}
	return MessageHistoryResponse{
		Messages:      messages,
		LastMessageID: r.LastMessageID,
	}
}

// LastMessagePreviewResponse — превью последнего сообщения в списке чатов.
type LastMessagePreviewResponse struct {
	MessageID uint   `json:"messageId"`
	Username  string `json:"username"`
	Content   string `json:"content"`
}

// ChatListItemResponse — элемент списка чатов пользователя. Полиморфен по Type
// (direct/group): для direct title/avatar — данные собеседника, для group — чата.
type ChatListItemResponse struct {
	ChatID      uint                        `json:"chatId"`
	Type        string                      `json:"type"`
	Title       string                      `json:"title"`
	Avatar      string                      `json:"avatar"`
	LastMessage *LastMessagePreviewResponse `json:"lastMessage"`
	UnreadCount int                         `json:"unreadCount"`
	UpdatedAt   time.Time                   `json:"updatedAt"`
}

func NewChatListItemResponse(item chatuc.ChatListItemResult) ChatListItemResponse {
	res := ChatListItemResponse{
		ChatID:      item.ChatID,
		Type:        item.Type,
		Title:       item.Title,
		Avatar:      item.Avatar,
		UnreadCount: item.UnreadCount,
		UpdatedAt:   item.UpdatedAt,
	}
	if item.LastMessage != nil {
		res.LastMessage = &LastMessagePreviewResponse{
			MessageID: item.LastMessage.MessageID,
			Username:  item.LastMessage.Username,
			Content:   item.LastMessage.Content,
		}
	}
	return res
}

func NewChatListResponse(items []chatuc.ChatListItemResult) []ChatListItemResponse {
	res := make([]ChatListItemResponse, 0, len(items))
	for _, item := range items {
		res = append(res, NewChatListItemResponse(item))
	}
	return res
}
