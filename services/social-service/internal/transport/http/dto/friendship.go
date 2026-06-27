package dto

type FriendRequest struct {
	UserID uint `json:"UserId" binding:"required" validate:"required"`
}

type FriendsSearchParams struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}
