package handlers

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
)

// SendEmailRequest — запрос на отправку письма из админки.
type SendEmailRequest struct {
	TemplateID string         `json:"templateId" binding:"required"`
	To         string         `json:"to" binding:"required"`
	Data       map[string]any `json:"data" binding:"omitempty"`

	// Priority поднимает письмо в очереди. Пригодится, когда появятся письма,
	// которых пользователь ждёт прямо сейчас (код подтверждения, сброс пароля).
	Priority int `json:"priority" binding:"omitempty"`

	// ScheduledAt откладывает отправку. Время в прошлом означает «сейчас».
	ScheduledAt *time.Time `json:"scheduledAt" binding:"omitempty"`

	// IdempotencyKey задаёт смысл «то же самое письмо» явно. Без него ключ
	// считается по содержимому, и повторный запрос с теми же данными в
	// пределах окна дедупликации вернёт уже созданное письмо.
	IdempotencyKey string `json:"idempotencyKey" binding:"omitempty"`

	// TemplateVersion зарезервировано под фазу 2, когда шаблоны переедут в БД
	// и админка сможет выбирать версию явно.
	TemplateVersion *uint `json:"templateVersion" binding:"omitempty"`
}

// SendEmailResponse — результат постановки письма в очередь.
type SendEmailResponse struct {
	EmailID string `json:"emailId"`

	// Duplicate означает, что такое письмо уже стояло в очереди и повторно не
	// создавалось. Это успех: возвращается идентификатор существующего.
	Duplicate bool `json:"duplicate"`
}

// EmailResponse — состояние письма.
//
// Тело письма наружу не отдаётся: в нём персональные данные, а для отладки
// достаточно статуса и причины отказа.
type EmailResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`

	To         string `json:"to"`
	Subject    string `json:"subject"`
	TemplateID string `json:"templateId"`

	Attempt    uint `json:"attempt"`
	MaxAttempt uint `json:"max_attempt"`
	Priority   int  `json:"priority"`

	LastError string `json:"lastError,omitempty"`
	TraceID   string `json:"traceId,omitempty"`

	ScheduledAt time.Time  `json:"scheduledAt"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ListEmailsRequest — фильтры списка писем.
type ListEmailsRequest struct {
	pagination.Params

	Status *string    `form:"status" binding:"omitempty"`
	To     *string    `form:"to" binding:"omitempty"`
	From   *time.Time `form:"from" binding:"omitempty" time_format:"2006-01-02"`
	Till   *time.Time `form:"till" binding:"omitempty" time_format:"2006-01-02"`
}

// ListEmailsResponse — страница писем.
type ListEmailsResponse struct {
	Items []EmailResponse `json:"items"`
	Total int64           `json:"total"`
}

// EventResponse — запись в истории письма.
type EventResponse struct {
	EventType    string         `json:"event_type"`
	OccurredTime time.Time      `json:"occurred_time"`
	Attempt      uint           `json:"attempt"`
	WorkerID     string         `json:"workerId,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

func toEmailResponse(e email.Email) EmailResponse {
	return EmailResponse{
		ID:          e.ID.String(),
		Status:      string(e.Status),
		To:          e.ToEmail,
		Subject:     e.Subject,
		TemplateID:  e.TemplateID,
		Attempt:     e.Attempt,
		MaxAttempt:  e.MaxAttempt,
		Priority:    e.Priority,
		LastError:   e.LastError,
		TraceID:     e.TraceID,
		ScheduledAt: e.ScheduledAt,
		SentAt:      e.SentAt,
		CreatedAt:   e.CreatedAt,
	}
}

func toEmailResponses(list []email.Email) []EmailResponse {
	out := make([]EmailResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEmailResponse(e))
	}
	return out
}

func toEventResponses(list []email.Event) []EventResponse {
	out := make([]EventResponse, 0, len(list))
	for _, e := range list {
		out = append(out, EventResponse{
			EventType:    string(e.EventType),
			OccurredTime: e.OccurredTime,
			Attempt:      e.Attempt,
			WorkerID:     e.WorkerID,
			Details:      e.Details,
		})
	}
	return out
}

type ApiKeyCreateRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expiresAt" binding:"omitempty"`
}
