package events

const (
	UserCreated = "user.created"
	UserUpdated = "user.updated"
	UserDeleted = "user.deleted"
)

type UserEvent struct {
	Envelop
	Payload UserPayload `json:"payload"`
}

type UserPayload struct {
	UserID int64 `json:"user_id"`
	//UserName    string `json:"username"`
	//Phone       string `json:"phone,omitempty"`
	//Avatar      string `json:"avatar"`
	//IsActive    bool   `json:"is_active"`
	//IsSuperuser bool   `json:"is_superuser"`
	//IsVerified  bool   `json:"is_verified"`
}

func NewUserCreated(userID int64) UserEvent {
	return UserEvent{
		Envelop: NewEnvelop(UserCreated),
		Payload: UserPayload{
			UserID: userID,
		},
	}
}
