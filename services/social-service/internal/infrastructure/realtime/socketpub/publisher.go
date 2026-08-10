// Package socketpub реализует порт chat.EventPublisher поверх Socket.IO.
// Событие рассылается в комнаты user:<id> получателей; межинстансную доставку
// обеспечивает Redis adapter, подключённый в chatsocket.
package socketpub

import (
	"context"

	chatuc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app/use_cases/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	chatsocket "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/socket"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

// PresenceAnnouncer — узкий порт socket-слоя, позволяющий досылать онлайн-статусы
// в только что созданный чат. Реализуется chatsocket.SocketServer.
type PresenceAnnouncer interface {
	AnnounceInChat(ctx context.Context, chatID uint, userIDs []uint)
}

type Publisher struct {
	ns socket.Namespace
	// presence опционален: без него чат работает как раньше, просто участники
	// нового чата узнают статусы друг друга только после реконнекта.
	presence PresenceAnnouncer
	logger   *zap.Logger
}

func NewPublisher(ns socket.Namespace, presence PresenceAnnouncer, logger *zap.Logger) *Publisher {
	return &Publisher{
		ns:       ns,
		presence: presence,
		logger:   logger,
	}
}

// Publish доставляет событие получателям. Способ адресации задаётся событием:
//   - RecipientIDs заданы → шлём каждому в его комнату user:<id> (message.new).
//   - иначе, ChatID>0 → шлём в комнату чата chat:<id>: одним emit покрываем все
//     сокеты всех участников, включая другие устройства инициатора (chat.read —
//     прочтение синхронизируется между девайсами).
//
// Ошибка emit не прерывает остальных; вызывающий useCase трактует публикацию
// как best-effort.
func (p *Publisher) Publish(_ context.Context, e chatuc.ChatEvent) error {
	// Конвертируем useCase-payload в транспортный DTO, чтобы socket-событие
	// отдавало те же camelCase-поля, что и HTTP-ответ.
	payload := toSocketPayload(e)

	if len(e.RecipientIDs) > 0 {
		for _, uid := range e.RecipientIDs {
			if err := p.ns.To(chatsocket.UserRoom(uid)).Emit(string(e.Type), payload); err != nil {
				p.logger.Warn("failed to emit chat event",
					zap.Error(err),
					zap.String("event", string(e.Type)),
					zap.Uint("recipient_id", uid))
			}
		}
		return nil
	}

	if e.ChatID > 0 {
		if err := p.ns.To(chatsocket.ChatRoom(e.ChatID)).Emit(string(e.Type), payload); err != nil {
			p.logger.Warn("failed to emit chat event to chat room",
				zap.Error(err),
				zap.String("event", string(e.Type)),
				zap.Uint("chat_id", e.ChatID))
		}
	}
	return nil
}

// JoinUsers заводит сокеты каждого пользователя в комнату чата chat:<id>. Целим
// по комнате user:<id> — так покрываются все устройства/вкладки пользователя на
// всех инстансах (Redis adapter выполняет SocketsJoin удалённо). Best-effort.
//
// Сразу после джойна досылаем в новую комнату онлайн-статусы участников:
// presence.online рассылается по чатам, известным на момент подключения, поэтому
// без этого собеседники в свежесозданном чате отображались бы офлайн до реконнекта.
func (p *Publisher) JoinUsers(ctx context.Context, chatID uint, userIDs []uint) error {
	room := chatsocket.ChatRoom(chatID)
	for _, uid := range userIDs {
		p.ns.In(chatsocket.UserRoom(uid)).SocketsJoin(room)
	}

	if p.presence != nil {
		p.presence.AnnounceInChat(ctx, chatID, userIDs)
	}
	return nil
}

// LeaveUsers выводит сокеты каждого пользователя из комнаты чата chat:<id> — после
// выхода/исключения участник немедленно перестаёт получать события чата, без
// реконнекта. Best-effort.
func (p *Publisher) LeaveUsers(_ context.Context, chatID uint, userIDs []uint) error {
	room := chatsocket.ChatRoom(chatID)
	for _, uid := range userIDs {
		p.ns.In(chatsocket.UserRoom(uid)).SocketsLeave(room)
	}
	return nil
}

// toSocketPayload приводит payload события к внешнему DTO-контракту по типу
// события. Неизвестный/несматчившийся payload отдаётся как есть.
func toSocketPayload(e chatuc.ChatEvent) any {
	switch e.Type {
	case chatuc.EventMessageNew:
		if m, ok := e.Payload.(chatuc.MessageResult); ok {
			return dto.NewMessageResponse(m)
		}
	case chatuc.EventChatRead:
		if r, ok := e.Payload.(chatuc.ReadResult); ok {
			return dto.NewReadResponse(r)
		}
	}
	return e.Payload
}
