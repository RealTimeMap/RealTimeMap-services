package dto

type UserResponse struct {
	ID       int     `json:"id"`
	UserName string  `json:"username"`
	Avatar   *string `json:"avatar,omitempty"`
}

func NewUserResponse(id int, userName string, avatar *string) *UserResponse {
	return &UserResponse{
		ID:       id,
		UserName: userName,
		Avatar:   avatar,
	}
}
