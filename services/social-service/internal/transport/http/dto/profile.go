package dto

import "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"

type BaseProfileResponse struct {
	UserID    uint   `json:"userId"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
	IsPrivate bool   `json:"isPrivate"`
}

func NewBaseProfileResponse(data *model.Profile) BaseProfileResponse {
	return BaseProfileResponse{
		UserID:    data.UserID,
		Username:  data.Username,
		Avatar:    data.Avatar.URL,
		IsPrivate: data.IsPrivate,
	}
}

type UserSettings struct {
	ShowInSearch bool `json:"showInSearch"`
}

func NewUserSettings(data model.PrivacySettings) UserSettings {
	return UserSettings{
		ShowInSearch: data.ShowInSearch,
	}
}

type PersonalProfileResponse struct {
	BaseProfileResponse
	Settings UserSettings `json:"settings"`
}

func NewPersonalProfileResponse(data *model.Profile) *PersonalProfileResponse {
	response := &PersonalProfileResponse{
		BaseProfileResponse: NewBaseProfileResponse(data),
		Settings:            NewUserSettings(data.PrivacySettings),
	}
	return response
}

// Поиск профилей

type SearchProfileRequest struct {
	Query    string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type SearchProfileItem struct {
	BaseProfileResponse
}

func NewSearchProfileItem(p *model.Profile) SearchProfileItem {
	return SearchProfileItem{
		BaseProfileResponse: NewBaseProfileResponse(p),
	}
}

func NewSearchProfileItems(profiles []*model.Profile) []SearchProfileItem {
	items := make([]SearchProfileItem, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, NewSearchProfileItem(p))
	}
	return items
}
