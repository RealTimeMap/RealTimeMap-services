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

type AvatarUpload struct {
	Data     []byte
	FileName string
}

type UpdateProfileInput struct {
	UserID   uint
	Username *string
	Tag      *string
	Avatar   *AvatarUpload
}
