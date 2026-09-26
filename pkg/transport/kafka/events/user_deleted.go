package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// UserDeletedPayload — пользователь удалил аккаунт, публикуется auth-сервисом.
//
// По событию каждый сервис стирает или обезличивает свои данные этого
// пользователя. Обработка обязана быть идемпотентной: at-least-once доставка
// привезёт событие повторно после ребаланса, а удалять к тому моменту уже
// нечего.
type UserDeletedPayload struct {
	UserID uint `json:"user_id"`

	// Email нужен smtp-service: письма там хранятся по адресу получателя, а
	// спросить адрес уже удалённого пользователя не у кого.
	Email string `json:"email"`

	DeletedAt time.Time `json:"deleted_at"`
}

// ParseUserDeleted разбирает сообщение топика user-service.
//
// ok = false — событие другого типа, его нужно пропустить без ошибки: в тот же
// топик едут регистрация, входы и смена пароля. Ошибка — событие
// user.deleted, которое не удалось разобрать; повтор его не исправит.
//
// user_id обязателен: удаление «по нулевому пользователю» в сервисах, где
// 0 означает обезличенного автора, задело бы чужие данные.
func ParseUserDeleted(value []byte) (payload UserDeletedPayload, ok bool, err error) {
	var raw RawEvent
	if err := json.Unmarshal(value, &raw); err != nil {
		return UserDeletedPayload{}, false, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if raw.Type != UserDeleted {
		return UserDeletedPayload{}, false, nil
	}

	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return UserDeletedPayload{}, true, fmt.Errorf("unmarshal user.deleted payload: %w", err)
	}
	if payload.UserID == 0 {
		return UserDeletedPayload{}, true, errors.New("user.deleted: user_id is missing")
	}

	return payload, true, nil
}
