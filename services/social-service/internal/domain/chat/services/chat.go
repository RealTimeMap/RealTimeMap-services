package services

import (
	"context"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

type ChatService struct {
	chatRepo    chat.Repository
	partRepo    chat.ParticipantRepository
	blockedRepo repository.BlockedUserRepository
	txm         *txmanager.TxManager
	logger      *zap.Logger
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

func NewChatService(chatRepo chat.Repository, partRepo chat.ParticipantRepository, blockedRepo repository.BlockedUserRepository, txm *txmanager.TxManager, logger *zap.Logger) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		partRepo:    partRepo,
		blockedRepo: blockedRepo,
		txm:         txm,
		logger:      logger,
	}
}

func (s *ChatService) OpenDirectChat(ctx context.Context, params ChatCreateParams) (*chat.Chat, error) {
	if err := checkID(params.UserID, params.PeerID); err != nil {
		return nil, err
	}
	if err := s.checkNotBlocked(ctx, params.UserID, params.PeerID); err != nil {
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

// Leave помечает пользователя вышедшим из чата (soft-leave через left_at). Только
// групповые чаты: из direct-чата выйти нельзя (нет продуктового смысла — чат на
// двоих). Пользователь должен быть активным участником. Идемпотентно: повторный
// выход не ошибка (Remove не заденет уже вышедшего).
func (s *ChatService) Leave(ctx context.Context, chatID, userID uint) error {
	part, err := s.partRepo.Get(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if part == nil || part.LeftAt != nil {
		return chat.ErrNotParticipant()
	}

	obj, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	if obj.Type == chat.DirectType {
		return chat.ErrCantLeaveDirect()
	}

	return s.partRepo.Remove(ctx, chatID, userID)
}

// DeleteResult — итог удаления чата.
type DeleteResult struct {
	// ForEveryone — чат удалён целиком у всех участников. false — только
	// история очищена у удалившего.
	ForEveryone bool
	// ParticipantIDs — активные участники на момент удаления; при
	// ForEveryone их нужно уведомить и вывести из комнаты чата.
	ParticipantIDs []uint
}

// Delete удаляет чат по запросу участника.
//
// Direct-чат удаляется на выбор:
//   - forEveryone = false — «у себя»: история до текущего последнего сообщения
//     скрывается от удалившего, чат пропадает из его списка и вернётся с
//     первым новым сообщением. Собеседник ничего не теряет.
//   - forEveryone = true — у обоих: чат и вся история удаляются физически.
//
// Групповой чат удаляется только целиком и только владельцем; forEveryone для
// группы не влияет на результат. Остальные участники выходят через Leave.
func (s *ChatService) Delete(ctx context.Context, chatID, userID uint, forEveryone bool) (*DeleteResult, error) {
	obj, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	part := activeParticipant(obj, userID)
	if part == nil {
		return nil, chat.ErrNotParticipant()
	}

	if obj.Type == chat.GroupType {
		if part.Role != chat.OwnerParticipantType {
			return nil, chat.ErrNotChatOwner()
		}
		forEveryone = true
	}

	if !forEveryone {
		var upTo uint
		if obj.LastMessageID != nil {
			upTo = *obj.LastMessageID
		}
		if err := s.partRepo.ClearHistory(ctx, chatID, userID, upTo); err != nil {
			return nil, err
		}
		return &DeleteResult{ParticipantIDs: []uint{userID}}, nil
	}

	participants := activeParticipantIDs(obj)
	if err := s.chatRepo.Delete(ctx, chatID); err != nil {
		return nil, err
	}
	return &DeleteResult{ForEveryone: true, ParticipantIDs: participants}, nil
}

func activeParticipant(obj *chat.Chat, userID uint) *chat.ChatParticipant {
	for i := range obj.Participants {
		p := &obj.Participants[i]
		if p.UserID == userID && p.LeftAt == nil {
			return p
		}
	}
	return nil
}

func activeParticipantIDs(obj *chat.Chat) []uint {
	ids := make([]uint, 0, len(obj.Participants))
	for _, p := range obj.Participants {
		if p.LeftAt == nil {
			ids = append(ids, p.UserID)
		}
	}
	return ids
}

// checkNotBlocked запрещает операцию, если между пользователями есть блок в любую
// сторону. Открыть direct-чат или писать в него с заблокированным нельзя.
func (s *ChatService) checkNotBlocked(ctx context.Context, userID, otherID uint) error {
	blocked, err := s.blockedRepo.ExistsBetween(ctx, userID, otherID)
	if err != nil {
		return err
	}
	if blocked {
		return chat.ErrBlocked()
	}
	return nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID uint) (*chat.Chat, error) {
	return s.chatRepo.GetByID(ctx, chatID)
}

func (s *ChatService) GetUsersChats(ctx context.Context, userID uint) ([]*chat.ChatListItem, error) {
	return s.chatRepo.ListByUser(ctx, userID)
}

// MarkRead отмечает чат прочитанным для пользователя: двигает курсор
// last_read_message_id участника на последнее сообщение чата. Пользователь
// должен быть активным участником. Если сообщений в чате ещё нет — no-op.
// Курсор монотонный (см. UpdateLastRead), поэтому повторный/устаревший вызов
// безопасен.
//
// Возвращает lastReadMessageID — id сообщения, на которое встал курсор, — и
// moved: было ли что двигать. moved=false означает, что в чате нет сообщений
// (курсор не тронут); вызывающий useCase по этому флагу решает, публиковать ли
// realtime-событие chat.read.
func (s *ChatService) MarkRead(ctx context.Context, chatID, userID uint) (lastReadMessageID uint, moved bool, err error) {
	part, err := s.partRepo.Get(ctx, chatID, userID)
	if err != nil {
		return 0, false, err
	}
	if part == nil || part.LeftAt != nil {
		return 0, false, chat.ErrNotParticipant()
	}

	obj, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return 0, false, err
	}
	// Нечего отмечать: в чате ещё нет сообщений.
	if obj.LastMessageID == nil {
		return 0, false, nil
	}

	if err := s.partRepo.UpdateLastRead(ctx, chatID, userID, *obj.LastMessageID); err != nil {
		return 0, false, err
	}
	return *obj.LastMessageID, true, nil
}
