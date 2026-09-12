package events

import "time"

const (
	UserRegistered = "user.registered"
	UserCreated    = "user.created"
	UserUpdated    = "user.updated"
	UserDeleted    = "user.deleted"
)

type UserEvent struct {
	Envelop
	Payload UserPayload `json:"payload"`
}

type UserPayload struct {
	UserID int64 `json:"user_id"`
}

func NewUserCreated(userID int64) UserEvent {
	return UserEvent{
		Envelop: NewEnvelop(UserCreated),
		Payload: UserPayload{
			UserID: userID,
		},
	}
}

// UserRegisteredEvent — регистрация пользователя, публикуется auth-сервисом.
type UserRegisteredEvent struct {
	Envelop
	Payload UserRegisteredPayload `json:"payload"`
}

// UserRegisteredPayload несёт адрес получателя прямо в событии: сервиса, у
// которого его можно спросить, пока нет (proto/user/service.proto без
// реализации). Когда UserService появится, email перестанет ходить через
// Kafka — это персональные данные, живущие в топике по retention.
type UserRegisteredPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	Phone        *string   `json:"phone,omitempty"`
	IsVerified   bool      `json:"is_verified"`
	OAuth        bool      `json:"oauth"`
	RegisteredAt time.Time `json:"registered_at"`
}
