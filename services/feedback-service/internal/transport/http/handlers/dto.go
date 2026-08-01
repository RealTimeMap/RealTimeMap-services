package handlers

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
)

type BugListItemResponse struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Title     string    `json:"title"`
	Tag       string    `json:"tag"`
	Status    string    `json:"status"`
	Platform  string    `json:"platform"` // Из DeviceInfo
	Build     string    `json:"build"`    // Из AppInfo
	HasLogs   bool      `json:"hasLogs"`  // Просто флаг, есть ли логи
	UserID    *uint     `json:"userId"`   // Идеально в будущем отдавать тут Email или Имя
}

func mapToListResponse(models []bug.BugResult) []BugListItemResponse {
	res := make([]BugListItemResponse, len(models))
	for i, m := range models {
		res[i] = BugListItemResponse{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			Title:     m.Title,
			Tag:       m.Tag,
			Status:    m.Status,
			Platform:  m.Platform,
			Build:     m.Build,
			HasLogs:   len(m.Logs) > 0,
			UserID:    m.UserID,
		}
	}
	return res
}
