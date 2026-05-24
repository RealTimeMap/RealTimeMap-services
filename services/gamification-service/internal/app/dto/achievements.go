package dto

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type AchievementItem struct {
	ID         uint      `json:"id"`
	Title      string    `json:"title"`
	Desc       string    `json:"desc"`
	Icon       string    `json:"icon"`
	Threshold  uint      `json:"threshold"`
	Reward     uint      `json:"reward"`
	UnlockedAt time.Time `json:"unlockedAt"`
}

func newAchievementItem(uAch model.UserAchievement) AchievementItem {
	return AchievementItem{
		ID:         uAch.Achievement.ID,
		Title:      uAch.Achievement.Title,
		Desc:       uAch.Achievement.Desc,
		Icon:       uAch.Achievement.Icon.URL,
		Threshold:  uAch.Achievement.Threshold,
		Reward:     uAch.Achievement.Reward.Amount,
		UnlockedAt: uAch.UnlockedAt,
	}
}

type AchievementResponse struct {
	AchievementItem
	Next *AchievementItem `json:"next"`
}
