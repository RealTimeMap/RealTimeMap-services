package chat

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

// PROFILES

type ProfileResult struct {
	ID       uint
	Username string
	Avatar   string
}

// toProfileResult безопасно маппит профиль. Если профиль не подгрузился
// (nil), возвращается пустой результат — обогащение best-effort и не должно
// ронять весь ответ.
func toProfileResult(p *model.Profile) ProfileResult {
	if p == nil {
		return ProfileResult{}
	}
	return ProfileResult{
		ID:       p.UserID,
		Username: p.Username,
		Avatar:   p.Avatar.URL,
	}
}

// MESSAGES

type MessageResult struct {
	ID        uint
	Type      string
	Content   string
	CreatedAt time.Time
	ChatID    uint
	Sender    ProfileResult
}

func toMessageResult(m *message.Message, p *model.Profile) MessageResult {
	sender := toProfileResult(p)
	// Даже если профиль не подгрузился, автор сообщения известен из самого
	// сообщения — отдаём хотя бы его id, чтобы клиент мог сматчить.
	if sender.ID == 0 {
		sender.ID = m.SenderID
	}
	return MessageResult{
		ID:        m.ID,
		Type:      string(m.Type),
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		ChatID:    m.ChatID,
		Sender:    sender,
	}
}

func toMultiMessageResult(messages []*message.Message, profiles map[uint]*model.Profile) []MessageResult {
	results := make([]MessageResult, 0, len(messages))
	for _, m := range messages {
		results = append(results, toMessageResult(m, profiles[m.SenderID]))
	}
	return results
}

type MessageHistoryResult struct {
	Messages      []MessageResult
	LastMessageID *uint
}

func toMessageHistoryResult(messages []*message.Message, profiles map[uint]*model.Profile, lastID *uint) MessageHistoryResult {
	return MessageHistoryResult{
		Messages:      toMultiMessageResult(messages, profiles),
		LastMessageID: lastID,
	}

}

// CHATS

type DirectChatResult struct {
	Peer      ProfileResult
	ChatID    uint
	CreatedAt time.Time
}

func toDirectChatResult(obj *chat.Chat, peer *model.Profile) DirectChatResult {
	return DirectChatResult{
		ChatID:    obj.ID,
		CreatedAt: obj.CreatedAt,
		Peer:      toProfileResult(peer),
	}
}

type GroupChatResult struct {
	ChatID    uint
	Title     string
	CreatedAt time.Time
}

func toGroupChatResult(obj *chat.Chat) GroupChatResult {
	var title string
	if obj.Title != nil {
		title = *obj.Title
	}
	return GroupChatResult{
		ChatID:    obj.ID,
		CreatedAt: obj.CreatedAt,
		Title:     title,
	}
}

type LastMessagePreview struct {
	Username  string
	Content   string
	MessageID uint
}

type ChatListItemResult struct {
	ChatID      uint
	Type        string
	Title       string
	Avatar      string
	LastMessage *LastMessagePreview
	UnreadCount int
	UpdatedAt   time.Time
}

// toChatListItemResult маппит один элемент списка чатов.
func toChatListItemResult(item *chat.ChatListItem, requesterID uint, profiles map[uint]*model.Profile) ChatListItemResult {
	res := ChatListItemResult{
		ChatID:      item.Chat.ID,
		Type:        string(item.Chat.Type),
		UnreadCount: item.UnreadCount,
		UpdatedAt:   item.Chat.UpdatedAt,
	}

	switch item.Chat.Type {
	case chat.DirectType:
		if peer := directPeerProfile(item.Chat, requesterID, profiles); peer != nil {
			res.Title = peer.Username
			res.Avatar = peer.Avatar.URL
		}
	default: // group
		if item.Chat.Title != nil {
			res.Title = *item.Chat.Title
		}
	}

	res.LastMessage = toLastMessagePreview(item.LastMessage, profiles)
	return res
}

// directPeerProfile находит профиль собеседника в direct-чате — участника,
func directPeerProfile(c chat.Chat, requesterID uint, profiles map[uint]*model.Profile) *model.Profile {
	for _, p := range c.Participants {
		if p.UserID != requesterID {
			return profiles[p.UserID]
		}
	}
	return nil
}

// toLastMessagePreview собирает превью последнего сообщения с именем автора из
func toLastMessagePreview(m *message.Message, profiles map[uint]*model.Profile) *LastMessagePreview {
	if m == nil {
		return nil
	}
	var username string
	if p := profiles[m.SenderID]; p != nil {
		username = p.Username
	}
	return &LastMessagePreview{
		MessageID: m.ID,
		Content:   m.Content,
		Username:  username,
	}
}

func toChatListResult(items []*chat.ChatListItem, requesterID uint, profiles map[uint]*model.Profile) []ChatListItemResult {
	results := make([]ChatListItemResult, 0, len(items))
	for _, item := range items {
		results = append(results, toChatListItemResult(item, requesterID, profiles))
	}
	return results
}
