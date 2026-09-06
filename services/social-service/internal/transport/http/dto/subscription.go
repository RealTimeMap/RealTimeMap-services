package dto

type SubscriptionRequest struct {
	UserID uint `json:"userId" binding:"required" validate:"required"`
}

type SubscriptionSearchParams struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

type SubscriptionStatusResponse struct {
	Subscribed bool `json:"subscribed"`
}

func NewSubscriptionStatusResponse(subscribed bool) SubscriptionStatusResponse {
	return SubscriptionStatusResponse{Subscribed: subscribed}
}
