package services

import (
	"context"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"go.uber.org/zap"
)

type ChatService struct {
	chatRepo chat.Repository
	partRepo chat.ParticipantRepository
	txm      *txmanager.TxManager
	logger   *zap.Logger
}

type ChatCreateParams struct {
	UserID uint
	PeerID uint
}

type GroupChatCreateParams struct {
	OwnerID  uint
	PeersIds []uint
	Title    *string
}

func (g *GroupChatCreateParams) getTitle() string {
	if g.Title != nil && *g.Title != "" {
		return *g.Title
	}
	return fmt.Sprintf("Group chat with %d user's", len(g.PeersIds)+1)
}

// minGroupMembers — минимальное итоговое число участников группы (owner + peers).
const minGroupMembers = 3

func NewChatService(chatRepo chat.Repository, partRepo chat.ParticipantRepository, txm *txmanager.TxManager, logger *zap.Logger) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
		partRepo: partRepo,
		txm:      txm,
		logger:   logger,
	}
}

func (s *ChatService) OpenDirectChat(ctx context.Context, params ChatCreateParams) (*chat.Chat, error) {
	if err := checkID(params.UserID, params.PeerID); err != nil {
		return nil, err
	}
	obj, err := s.chatRepo.GetOrCreateDirect(ctx, params.UserID, params.PeerID)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// OpenGroupChat создаёт групповой чат вместе с участниками атомарно: вставка
// чата и участников выполняется в одной транзакции через txmanager, поэтому
// пустой группы (чат без участников) при сбое возникнуть не может. Создатель
// гарантированно входит в состав с ролью owner.
func (s *ChatService) OpenGroupChat(ctx context.Context, param GroupChatCreateParams) (*chat.Chat, error) {
	members := buildGroupMembers(param.OwnerID, param.PeersIds)
	if len(members) < minGroupMembers {
		return nil, chat.ErrLowMembers(minGroupMembers)
	}

	title := param.getTitle()
	obj := &chat.Chat{
		CreatedBy: param.OwnerID,
		Type:      chat.GroupType,
		Title:     &title,
	}

	err := s.txm.WithTx(ctx, func(ctx context.Context) error {
		if err := s.chatRepo.Create(ctx, obj); err != nil {
			return err
		}

		participants := make([]*chat.ChatParticipant, 0, len(members))
		for userID, role := range members {
			participants = append(participants, &chat.ChatParticipant{
				ChatID: obj.ID,
				UserID: userID,
				Role:   role,
			})
		}
		return s.partRepo.BulkAdd(ctx, participants)
	})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// buildGroupMembers формирует итоговый состав группы: создатель с ролью owner
// плюс уникальные участники-члены. Дубликаты и повтор owner в списке peers удаляются
func buildGroupMembers(ownerID uint, peers []uint) map[uint]chat.ParticipantRole {
	members := make(map[uint]chat.ParticipantRole, len(peers)+1)
	members[ownerID] = chat.OwnerParticipantType
	for _, id := range peers {
		if id == ownerID {
			continue
		}
		members[id] = chat.MemberParticipantType
	}
	return members
}

func checkID(userA, userB uint) error {
	if userA == userB {
		return chat.ErrSelfDirectChat(userA)
	}
	return nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID uint) (*chat.Chat, error) {
	return s.chatRepo.GetByID(ctx, chatID)
}

func (s *ChatService) GetUsersChats(ctx context.Context, userID uint) ([]*chat.ChatListItem, error) {
	return s.chatRepo.ListByUser(ctx, userID)
}
