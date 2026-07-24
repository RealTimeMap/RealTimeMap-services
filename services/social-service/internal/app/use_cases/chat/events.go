package chat

import "context"

// ChatEventType — тип realtime-события чата, доставляемого клиенту через сокет.
type ChatEventType string

const (
	// EventMessageNew — новое сообщение в чате. Payload = MessageResult.
	EventMessageNew ChatEventType = "message.new"
	// EventChatRead — участник продвинул курсор прочтения. Payload = ReadResult.
	// Адресуется в комнату чата (ChatID), чтобы дошло до всех участников, включая
	// другие устройства самого читающего (обнуление unread на всех девайсах).
	EventChatRead ChatEventType = "chat.read"
)

// ChatEvent — доменное событие, которое useCase публикует после изменения
// состояния чата. Задаётся ровно один способ адресации:
//   - RecipientIDs — доставить перечисленным пользователям в их комнаты user:<id>
//     (например, message.new — всем участникам поimённо).
//   - ChatID (>0, при пустом RecipientIDs) — доставить в комнату чата chat:<id>,
//     т.е. всем сокетам всех участников, включая другие устройства инициатора
//     (например, chat.read — прочтение должно синхронизироваться между девайсами).
//
// Payload — уже готовый DTO ответа (тот же контракт, что уходит по HTTP).
type ChatEvent struct {
	Type         ChatEventType
	RecipientIDs []uint
	ChatID       uint
	Payload      any
}

// EventPublisher — порт realtime-эффектов чата. useCase зависит только от него и
// ничего не знает про Socket.IO/Redis. Помимо доставки событий порт синхронизирует
// членство сокетов в комнатах chat:<id> при изменении состава чата (создание,
// выход) — членство в комнате = участие в чате, на нём строится адресация событий.
type EventPublisher interface {
	// Publish доставляет событие получателям в их комнаты user:<id>.
	Publish(ctx context.Context, e ChatEvent) error
	// JoinUsers заводит сокеты указанных пользователей в комнату чата chat:<id>.
	// Вызывается после добавления участников (создание direct/group). Best-effort.
	JoinUsers(ctx context.Context, chatID uint, userIDs []uint) error
	// LeaveUsers выводит сокеты указанных пользователей из комнаты чата chat:<id>.
	// Вызывается после выхода/исключения участника. Best-effort.
	LeaveUsers(ctx context.Context, chatID uint, userIDs []uint) error
}
