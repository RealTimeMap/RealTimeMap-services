package profile

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"

type CreateProfileInput struct {
	UserID   uint
	Username string

	// IsAdmin приезжает из события auth-сервиса. Через HTTP не выставляется
	// ни при каких условиях: поле принадлежит auth, а не пользователю.
	IsAdmin bool
}

// SyncAdminInput — применение признака администратора, пришедшего из auth.
type SyncAdminInput struct {
	UserID  uint
	IsAdmin bool
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
