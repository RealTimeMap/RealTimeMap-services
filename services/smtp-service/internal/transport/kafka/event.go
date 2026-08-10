package kafka

import "time"

// Типы событий, на которые сервис отправляет письма.
const (
	EventUserRegistered = "user.registered"
)

// UserRegistered — событие регистрации пользователя.
//
// Приезжает из auth-service. Адрес получателя лежит прямо в событии: сервиса,
// у которого его можно спросить, пока нет (proto/user/service.proto без
// реализации). Когда UserService появится, email перестанет ходить через
// Kafka — это персональные данные, живущие в топике по retention.
type UserRegistered struct {
	EventType string `json:"event_type"`
	UserID    uint64 `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`

	// Поля ниже сервис пока не использует: письмо на регистрацию одно и то же
	// независимо от способа входа. Ветвление (подтверждение почты для обычной
	// регистрации, приветствие сразу для OAuth) требует токена подтверждения,
	// которого в событии нет.
	Phone        *string   `json:"phone"`
	IsVerified   bool      `json:"is_verified"`
	OAuth        bool      `json:"oauth"`
	RegisteredAt time.Time `json:"registered_at"`
}
