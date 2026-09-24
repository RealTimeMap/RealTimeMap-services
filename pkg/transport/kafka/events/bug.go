package events

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
