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

type Publisher struct {
	ns     socket.Namespace
	logger *zap.Logger
}

func NewPublisher(ns socket.Namespace, logger *zap.Logger) *Publisher {
	return &Publisher{
		ns:     ns,
		logger: logger,
	}
}

// Publish доставляет событие каждому получателю в его комнату user:<id>.
// Ошибка одного emit не прерывает остальных; вызывающий useCase трактует
// публикацию как best-effort.
func (p *Publisher) Publish(_ context.Context, e chatuc.ChatEvent) error {
	// Конвертируем useCase-payload в транспортный DTO, чтобы socket-событие
	// отдавало те же camelCase-поля, что и HTTP-ответ.
	payload := toSocketPayload(e)

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

// toSocketPayload приводит payload события к внешнему DTO-контракту по типу
// события. Неизвестный/несматчившийся payload отдаётся как есть.
func toSocketPayload(e chatuc.ChatEvent) any {
	switch e.Type {
	case chatuc.EventMessageNew:
		if m, ok := e.Payload.(chatuc.MessageResult); ok {
			return dto.NewMessageResponse(m)
		}
	}
	return e.Payload
}
