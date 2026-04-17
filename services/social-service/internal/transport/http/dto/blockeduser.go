package dto

type BlockedUserRequest struct {
	UserID uint `json:"UserId" binding:"required" validate:"required"`
}

type BlockedSearchParams struct {
	Query    string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
