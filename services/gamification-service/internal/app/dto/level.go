package dto

import "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"

type LevelResponse struct {
	Level      uint   `json:"level"`
	Title      string `json:"title"`
	XpRequired uint   `json:"xpRequired"`
}

func NewLevelResponse(l *model.Level) LevelResponse {
	return LevelResponse{
		Level:      l.Level,
		Title:      l.Title,
		XpRequired: l.XPRequired,
	}
}

func NewMultiResponse(l []*model.Level) []LevelResponse {
	res := make([]LevelResponse, 0, len(l))
	for _, level := range l {
		res = append(res, NewLevelResponse(level))
	}
	return res
}
