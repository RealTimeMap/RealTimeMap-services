package events

// ProfileUpdated — профиль изменён в social-service.
//
// Публикуется в топик social-service.events, чтобы auth-сервис держал свою
// копию username в актуальном состоянии: профиль редактируется здесь, но
// username участвует в выдаче auth (заголовок X-User-Name на gateway), и без
// события там навсегда осталось бы значение с момента регистрации.
const ProfileUpdated = "profile.updated"

type ProfileEvent struct {
	Envelop
	Payload ProfilePayload `json:"payload"`
}

// ProfilePayload — состояние профиля на момент изменения.
//
// Едет полное состояние изменяемых полей, а не дельта: потребитель не обязан
// хранить предыдущую версию, а повтор события (Kafka at-least-once) при полном
// состоянии идемпотентен — повторное применение тех же значений ничего не портит.
//
// IsAdmin сюда не входит намеренно: признак принадлежит auth-сервису, social
// его только зеркалит. Отправка своей копии обратно закольцевала бы поле —
// устаревшее значение из social перезаписало бы актуальное в auth.
type ProfilePayload struct {
	UserID uint `json:"user_id"`

	// Username и Tag — omitempty не ставится: пустая строка здесь означает
	// «поле не менялось» только вместе с отсутствием ключа, а различать эти
	// два случая потребителю не нужно — едет актуальное состояние целиком.
	Username string `json:"username"`
	Tag      string `json:"tag"`
	Avatar   string `json:"avatar"`
}

func NewProfileUpdated(payload ProfilePayload) ProfileEvent {
	return ProfileEvent{
		Envelop: NewEnvelop(ProfileUpdated),
		Payload: payload,
	}
}
