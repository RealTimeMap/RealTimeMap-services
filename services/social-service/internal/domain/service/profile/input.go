package profile

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"

type CreateProfileInput struct {
	UserID   uint
	Username string
}

type SearchProfilesInput struct {
	Username   string
	Pagination pagination.Params
}
