package events

import "time"

const (
	UserRegistered = "user.registered"
	UserCreated    = "user.created"
	UserUpdated    = "user.updated"
	UserDeleted    = "user.deleted"

	// События, по которым auth-сервис просит отправить письмо.
	//
	// Токены и ссылки едут в payload: сгенерировать их на стороне
	// smtp-service нельзя — они подписаны секретом auth-сервиса.
	UserVerifyRequested   = "user.verify_requested"
	UserPasswordForgotten = "user.password_forgotten"
	UserPasswordChanged   = "user.password_changed"
	UserLoggedIn          = "user.logged_in"
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

// UserVerifyRequestedPayload — запрос подтверждения адреса.
type UserVerifyRequestedPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	// VerifyURL содержит одноразовый токен, подписанный auth-сервисом.
	VerifyURL string `json:"verify_url"`

	// Code — короткий код для ввода в приложении. Пустой, если auth его не
	// выдаёт: шаблон показывает блок с кодом только когда он есть.
	Code string `json:"code,omitempty"`

	// TTLMinutes — сколько ссылка и код действительны.
	TTLMinutes uint `json:"ttl_minutes"`
}

// UserPasswordForgottenPayload — запрос на сброс пароля.
type UserPasswordForgottenPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	ResetURL   string `json:"reset_url"`
	TTLMinutes uint   `json:"ttl_minutes"`

	// Контекст запроса: по нему пользователь отличает свой запрос от чужого.
	RequestedAt time.Time `json:"requested_at"`
	Device      string    `json:"device,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// UserPasswordChangedPayload — пароль уже изменён.
//
// Отдельное событие от UserPasswordForgotten: то присылает ссылку для смены,
// это сообщает о свершившемся факте и служит сигналом безопасности.
type UserPasswordChangedPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	ChangedAt time.Time `json:"changed_at"`
	Device    string    `json:"device,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
}

// UserLoggedInPayload — вход в аккаунт.
type UserLoggedInPayload struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	SignedInAt time.Time `json:"signed_in_at"`
	Device     string    `json:"device,omitempty"`
	Location   string    `json:"location,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}
