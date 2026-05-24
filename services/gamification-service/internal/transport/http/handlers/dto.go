package handlers

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
)

type NearestAchievementResponse struct {
	Achievement *AchievementResponse `json:"achievement"`
	Current     uint                 `json:"current"`
	Threshold   uint                 `json:"threshold"`
	Progress    float64              `json:"progress"`
}

func ToNearestAchievementResponse(n repository.NearestAchievement) *NearestAchievementResponse {
	var progress float64
	if n.Threshold > 0 {
		progress = utils.Percent(int64(n.Current), int64(n.Threshold))
	}
	return &NearestAchievementResponse{
		Achievement: ToAchievementResponse(&n.Achievement),
		Current:     n.Current,
		Threshold:   n.Threshold,
		Progress:    progress,
	}
}

func ToNearestAchievementResponseList(items []repository.NearestAchievement) []*NearestAchievementResponse {
	out := make([]*NearestAchievementResponse, 0, len(items))
	for _, n := range items {
		out = append(out, ToNearestAchievementResponse(n))
	}
	return out
}

type UserAchievementResponse struct {
	Achievement *AchievementResponse `json:"achievement"`
	UnlockedAt  time.Time            `json:"unlockedAt"`
}

func ToUserAchievementResponse(ua *model.UserAchievement) *UserAchievementResponse {
	if ua == nil {
		return nil
	}
	return &UserAchievementResponse{
		Achievement: ToAchievementResponse(&ua.Achievement),
		UnlockedAt:  ua.UnlockedAt,
	}
}

func ToUserAchievementResponseList(items []*model.UserAchievement) []*UserAchievementResponse {
	out := make([]*UserAchievementResponse, 0, len(items))
	for _, ua := range items {
		out = append(out, ToUserAchievementResponse(ua))
	}
	return out
}

type AchievementResponse struct {
	ID               uint                 `json:"id"`
	Code             string               `json:"code"`
	Title            string               `json:"title"`
	Desc             string               `json:"desc"`
	TriggerEventType string               `json:"triggerEventType"`
	Threshold        uint                 `json:"threshold"`
	Icon             string               `json:"icon"`
	Reward           *XPRewardResponse    `json:"reward,omitempty"`
	Next             *AchievementResponse `json:"next,omitempty"`
}

type XPRewardResponse struct {
	ID     uint   `json:"id"`
	Code   string `json:"code"`
	Amount uint   `json:"amount"`
}

func ToAchievementResponse(a *model.Achievement) *AchievementResponse {
	if a == nil {
		return nil
	}
	resp := &AchievementResponse{
		ID:               a.ID,
		Code:             a.Code,
		Title:            a.Title,
		Desc:             a.Desc,
		TriggerEventType: a.TriggerEventType,
		Threshold:        a.Threshold,
		Icon:             a.Icon.URL,
		Reward:           toXPRewardResponse(&a.Reward),
		Next:             ToAchievementResponse(a.Next),
	}
	return resp
}

func ToAchievementResponseList(items []*model.Achievement) []*AchievementResponse {
	out := make([]*AchievementResponse, 0, len(items))
	for _, a := range items {
		out = append(out, ToAchievementResponse(a))
	}
	return out
}

func toXPRewardResponse(r *model.XPReward) *XPRewardResponse {
	if r == nil || r.ID == 0 {
		return nil
	}
	return &XPRewardResponse{
		ID:     r.ID,
		Code:   r.Code,
		Amount: r.Amount,
	}
}
