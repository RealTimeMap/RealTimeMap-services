package events

import "time"

// BugConfirmed — разработчик подтвердил баг из отчёта пользователя.
//
// Публикуется только для отчётов с автором: анонимный отчёт некому
// засчитать, а геймификация без user_id событие всё равно отбросит.
// Отправляется один раз на баг — при первом подтверждении: возврат на
// проверку и повторное подтверждение не должны давать второе начисление.
const BugConfirmed = "bug.confirmed"

type BugEvent struct {
	Envelop
	Payload BugPayload `json:"payload"`
}

// BugPayload — подтверждённый баг и его автор.
type BugPayload struct {
	BugID  uint   `json:"bugId"`
	UserID uint   `json:"userId"`
	Tag    string `json:"tag"`
}

func NewBugConfirmed(payload BugPayload) BugEvent {
	return BugEvent{
		Envelop: NewEnvelop(BugConfirmed),
		Payload: payload,
	}
}

// BugCreated — пользователь отправил новый баг-репорт. По нему smtp-service
// уведомляет разработчиков.
const BugCreated = "bug.created"

type BugCreatedEvent struct {
	Envelop
	Payload BugCreatedPayload `json:"payload"`
}

// BugCreatedPayload — содержимое отчёта. UserID пуст у анонимного отчёта.
type BugCreatedPayload struct {
	BugID      uint      `json:"bugId"`
	UserID     *uint     `json:"userId,omitempty"`
	Title      string    `json:"title"`
	Desc       string    `json:"desc"`
	Tag        string    `json:"tag"`
	Platform   string    `json:"platform"`
	OS         string    `json:"os"`
	Resolution string    `json:"resolution"`
	Build      string    `json:"build"`
	CreatedAt  time.Time `json:"createdAt"`
}

func NewBugCreated(payload BugCreatedPayload) BugCreatedEvent {
	return BugCreatedEvent{
		Envelop: NewEnvelop(BugCreated),
		Payload: payload,
	}
}
