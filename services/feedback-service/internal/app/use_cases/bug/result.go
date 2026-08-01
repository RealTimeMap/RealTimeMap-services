package bug

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
)

type BugResult struct {
	ID        uint
	CreatedAt time.Time
	Status    string
	Tag       string
	Title     string
	Platform  string
	Build     string
	Logs      []string
	UserID    *uint
}

func toBugResult(obj bug.Model) BugResult {
	return BugResult{
		ID:        obj.ID,
		Tag:       string(obj.Tag),
		CreatedAt: obj.CreatedAt,
		Status:    string(obj.Status),
		UserID:    obj.UserID,
		Platform:  obj.Device.Platform,
		Title:     obj.Title,
		Logs:      obj.App.Logs,
		Build:     obj.App.Build,
	}
}

func toMultiBugResult(objs []bug.Model) []BugResult {
	result := make([]BugResult, 0, len(objs))
	for _, obj := range objs {
		result = append(result, toBugResult(obj))
	}
	return result
}
