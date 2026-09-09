package handlers

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
)

// ServiceBugResponse — представление бага для таск-менеджера.
//
// Шире, чем BugListItemResponse админки: по этим полям задача
// заполняется при создании, поэтому нужно описание, а не только
// признак наличия логов.
type ServiceBugResponse struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Title     string    `json:"title"`
	Desc      string    `json:"desc"`
	Tag       string    `json:"tag"`
	Status    string    `json:"status"`
	Platform  string    `json:"platform"`
	Build     string    `json:"build"`
	HasLogs   bool      `json:"hasLogs"`
	UserID    *uint     `json:"userId"`

	// Обстановка, в которой баг воспроизвёлся.
	OS         string   `json:"os,omitempty"`
	Resolution string   `json:"resolution,omitempty"`
	Battery    *float64 `json:"battery,omitempty"`

	// TaskID — задача, в которой баг ведут. Пусто, пока баг свободен.
	TaskID *uint `json:"taskId"`
}

func mapToServiceItem(m bug.BugResult) ServiceBugResponse {
	return ServiceBugResponse{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		Title:     m.Title,
		Desc:      m.Desc,
		Tag:       m.Tag,
		Status:    m.Status,
		Platform:  m.Platform,
		Build:     m.Build,
		HasLogs:   len(m.Logs) > 0,
		UserID:    m.UserID,
		TaskID:    m.TaskID,

		OS:         m.OS,
		Resolution: m.Resolution,
		Battery:    m.Battery,
	}
}

// ServiceBugDetailResponse — баг целиком, вместе с логами.
//
// Отдельный тип от карточки перечня: логи нужны только тому, кто
// открыл конкретный баг, и в списке они были бы лишним весом.
type ServiceBugDetailResponse struct {
	ServiceBugResponse

	// Logs — журнал приложения на момент отправки отчёта.
	Logs []string `json:"logs"`

	// Width и Height разобраны из разрешения: клиенту не нужно знать,
	// что оно хранится строкой.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

func mapToServiceDetail(m bug.BugResult) ServiceBugDetailResponse {
	return ServiceBugDetailResponse{
		ServiceBugResponse: mapToServiceItem(m),
		// Пустой срез, а не nil: клиенту удобнее получить [] и не
		// разбирать отдельно случай «логов нет».
		Logs:   append([]string{}, m.Logs...),
		Width:  m.Width,
		Height: m.Height,
	}
}

// ServiceBugListResponse — перечень багов. Обёрнут в объект, а не отдан
// голым массивом: так к нему можно добавить метаданные, не ломая клиента.
type ServiceBugListResponse struct {
	Items []ServiceBugResponse `json:"items"`
}

func mapToServiceListResponse(models []bug.BugResult) ServiceBugListResponse {
	items := make([]ServiceBugResponse, 0, len(models))
	for _, m := range models {
		items = append(items, mapToServiceItem(m))
	}
	return ServiceBugListResponse{Items: items}
}
