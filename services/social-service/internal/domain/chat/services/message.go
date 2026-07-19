package services

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"go.uber.org/zap"
)

const (
	messageLimit      = 15
	maxMessageContent = 4000
)

type MessageService struct {
	messageRepo message.Repository
	chatRepo    chat.Repository
	partRepo    chat.ParticipantRepository
	txm         *txmanager.TxManager
	logger      *zap.Logger
}

func NewMessageService(messageRepo message.Repository,
	chatRepo chat.Repository, partRepo chat.ParticipantRepository, txm *txmanager.TxManager, logger *zap.Logger) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		partRepo:    partRepo,
		txm:         txm,
		logger:      logger,
	}
}

type MessageCreateParams struct {
	ChatID   uint
	SenderID uint
	Content  string
}

type MessageGetParams struct {
	ChatID        uint
	UserID        uint
	LastMessageID *uint
}

func (s *MessageService) SendMessage(ctx context.Context, params MessageCreateParams) (*message.Message, error) {
	if err := s.validateParticipant(ctx, params.ChatID, params.SenderID); err != nil {
		return nil, err
	}

	content, err := validateContent(params.Content)
	if err != nil {
		return nil, err
	}

	obj := &message.Message{
		ChatID:   params.ChatID,
		SenderID: params.SenderID,
		Content:  content,
		Type:     message.TextType,
	}
	err = s.txm.WithTx(ctx, func(ctx context.Context) error {
		if err := s.messageRepo.Create(ctx, obj); err != nil {
			return err
		}
		if err := s.chatRepo.UpdateLastMessage(ctx, params.ChatID, obj.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// History метод используется для получения истории чата
func (s *MessageService) History(ctx context.Context, params MessageGetParams) ([]*message.Message, error) {
	if err := s.validateParticipant(ctx, params.ChatID, params.UserID); err != nil {
		return nil, err
	}

	return s.messageRepo.GetMessages(ctx, message.Filter{
		ChatID:        params.ChatID,
		LastMessageID: params.LastMessageID,
		Limit:         messageLimit,
	})
}

// validateParticipant вспомогательная функция для проверки, что участник находится в чате
func (s *MessageService) validateParticipant(ctx context.Context, chatID, userID uint) error {
	obj, err := s.partRepo.Get(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if obj == nil || obj.LeftAt != nil {
		return chat.ErrNotParticipant()
	}
	return nil
}

// validateContent нормализует и проверяет текст сообщения: обрезает крайние
// пробелы, запрещает пустое и слишком длинное
func validateContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", chat.ErrEmptyMessage()
	}
	if utf8.RuneCountInString(content) > maxMessageContent {
		return "", chat.ErrMessageTooLong(maxMessageContent, content)
	}
	return content, nil
}
