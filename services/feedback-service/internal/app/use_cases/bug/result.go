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
	Desc      string
	Build     string
	Logs      []string
	UserID    *uint

	// Обстановка, в которой баг воспроизвёлся. Разработчику она нужна
	// целиком: без версии ОС и разрешения половина отчётов
	// невоспроизводима.
	Platform   string
	OS         string
	Resolution string
	Width      int
	Height     int
	Battery    *float64

	// TaskID — задача, в которой баг ведут. Пусто, пока баг свободен.
	TaskID *uint
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
		Desc:      obj.Desc,
		Logs:      obj.App.Logs,
		Build:     obj.App.Build,
		TaskID:    obj.TaskID,

		OS:         obj.Device.OS,
		Resolution: obj.Device.Resolution,
		// Ширина и высота разобраны доменом: клиенту не нужно знать,
		// что разрешение хранится строкой вида «1080x2400».
		Width:   obj.Device.Width(),
		Height:  obj.Device.Height(),
		Battery: obj.Device.Battery,
	}
}

func toMultiBugResult(objs []bug.Model) []BugResult {
	result := make([]BugResult, 0, len(objs))
	for _, obj := range objs {
		result = append(result, toBugResult(obj))
	}
	return result
}
