package dto

import "github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"

// CreateTokenRequest — регистрация токена устройства.
type CreateTokenRequest struct {
	Token string `form:"token" json:"token" binding:"required,max=4096"`
}

func (r CreateTokenRequest) ToParams(userID uint) token.CreateTokenParams {
	return token.CreateTokenParams{
		UserID: userID,
		Token:  r.Token,
	}
}
