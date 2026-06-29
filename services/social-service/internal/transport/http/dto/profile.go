package dto

import "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"

type BaseProfileResponse struct {
	UserID    uint   `json:"userId"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
	Tag       string `json:"tag"`
	IsPrivate bool   `json:"isPrivate"`
}

func NewBaseProfileResponse(data *model.Profile) BaseProfileResponse {
	return BaseProfileResponse{
		UserID:    data.UserID,
		Username:  data.Username,
		Avatar:    data.Avatar.URL,
		IsPrivate: data.IsPrivate,
		Tag:       data.Tag,
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

type GamificationResponse struct {
	CurrentLevel     uint64            `json:"currentLevel"`
	CurrentLevelName string            `json:"currentLevelName"`
	CurrentXP        uint64            `json:"currentXp"`
	XPForNextLevel   uint64            `json:"xpForNextLevel"`
	ProgressPercent  float64           `json:"progressPercent"`
	NextLevel        NextLevelResponse `json:"nextLevel"`
}

type NextLevelResponse struct {
	Level     uint64 `json:"level"`
	LevelName string `json:"levelName"`
}

func NewGamificationResponse(p *model.Progress) *GamificationResponse {
	if p == nil {
		return nil
	}
	return &GamificationResponse{
		CurrentLevel:     p.CurrentLevel,
		CurrentLevelName: p.CurrentLevelName,
		CurrentXP:        p.CurrentXP,
		XPForNextLevel:   p.XPForNextLevel,
		ProgressPercent:  p.ProgressPercent,
		NextLevel:        NextLevelResponse{Level: p.NextLevel.Level, LevelName: p.NextLevel.LevelName},
	}
}

type PersonalProfileResponse struct {
	BaseProfileResponse
	Settings     UserSettings          `json:"settings"`
	Gamification *GamificationResponse `json:"gamification,omitempty"`
}

func NewPersonalProfileResponse(data *model.Profile) *PersonalProfileResponse {
	response := &PersonalProfileResponse{
		BaseProfileResponse: NewBaseProfileResponse(data),
		Settings:            NewUserSettings(data.PrivacySettings),
	}
	return response
}

func NewPersonalProfileResponseWithGamification(data *model.Profile, progress *model.Progress) *PersonalProfileResponse {
	r := NewPersonalProfileResponse(data)
	r.Gamification = NewGamificationResponse(progress)
	return r
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

// Обновление профиля

type ProfileUpdateRequest struct {
	Username *string `form:"username" binding:"omitempty,min=2,max=32"`
	Tag      *string `form:"tag" binding:"omitempty"`
}
