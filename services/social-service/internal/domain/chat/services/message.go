package services

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
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
	blockedRepo repository.BlockedUserRepository
	txm         *txmanager.TxManager
	logger      *zap.Logger
}

func NewMessageService(messageRepo message.Repository,
	chatRepo chat.Repository, partRepo chat.ParticipantRepository, blockedRepo repository.BlockedUserRepository, txm *txmanager.TxManager, logger *zap.Logger) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		partRepo:    partRepo,
		blockedRepo: blockedRepo,
		txm:         txm,
		logger:      logger,
	}
}

type MessageCreateParams struct {
	ChatID   uint
	SenderID uint
	Content  string

	// ClientMessageID — идемпотентный ключ от клиента (UUID). Повторная отправка с
	// тем же ключом в том же чате от того же отправителя не создаёт дубль, а
	// возвращает исходное сообщение. Пусто → без дедупа (обратная совместимость).
	ClientMessageID string
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

	if err := s.checkNotBlockedInDirect(ctx, params.ChatID, params.SenderID); err != nil {
		return nil, err
	}

	content, err := validateContent(params.Content)
	if err != nil {
		return nil, err
	}

	obj := &message.Message{
		ChatID:          params.ChatID,
		SenderID:        params.SenderID,
		Content:         content,
		Type:            message.TextType,
		ClientMessageID: normalizeClientMessageID(params.ClientMessageID),
	}
	// created сообщает, было ли сообщение реально вставлено. При повторе с тем же
	// ClientMessageID возвращается исходная строка (created=false) — тогда чат не
	// трогаем (last_message уже актуален) и событие message.new повторно не шлём.
	var created bool
	err = s.txm.WithTx(ctx, func(ctx context.Context) error {
		var err error
		created, err = s.messageRepo.CreateIdempotent(ctx, obj)
		if err != nil {
			return err
		}
		if !created {
			return nil
		}
		return s.chatRepo.UpdateLastMessage(ctx, params.ChatID, obj.ID)
	})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// checkNotBlockedInDirect применяет блок-проверку только к direct-чатам: писать
// собеседнику, с которым стоит блок (в любую сторону), нельзя. В группах блок не
// применяется — участников много и семантика блока к ним не относится.
func (s *MessageService) checkNotBlockedInDirect(ctx context.Context, chatID, senderID uint) error {
	obj, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	if obj == nil || obj.Type != chat.DirectType {
		return nil
	}

	participants, err := s.partRepo.ListByChat(ctx, chatID)
	if err != nil {
		return err
	}
	for _, p := range participants {
		if p.UserID == senderID {
			continue
		}
		blocked, err := s.blockedRepo.ExistsBetween(ctx, senderID, p.UserID)
		if err != nil {
			return err
		}
		if blocked {
			return chat.ErrBlocked()
		}
	}
	return nil
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

// RecipientIDs возвращает id всех активных участников чата, кому нужно доставить
// realtime-событие о новом сообщении, — включая самого отправителя. Отправитель
// получает эхо на свои другие устройства/вкладки; девайс-инициатор дедупит
// входящее по серверному id (он уже показал сообщение по HTTP-ответу).
func (s *MessageService) RecipientIDs(ctx context.Context, chatID uint) ([]uint, error) {
	participants, err := s.partRepo.ListByChat(ctx, chatID)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(participants))
	for _, p := range participants {
		ids = append(ids, p.UserID)
	}
	return ids, nil
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

// normalizeClientMessageID приводит клиентский ключ идемпотентности к каноничному
// виду. Пустой ключ → nil: в БД это NULL, который не участвует в UNIQUE, поэтому
// сообщения без дедупа не конфликтуют друг с другом.
func normalizeClientMessageID(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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
