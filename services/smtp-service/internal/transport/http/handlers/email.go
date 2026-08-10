// Package handlers — HTTP-вход для админ-панели.
//
// Ручной путь отправки. В очередь письма попадают через тот же
// email.Service, что и события из Kafka, поэтому валидация, рендер и
// дедупликация работают одинаково независимо от источника.
package handlers

import (
	nethttp "net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EmailHandlerDeps struct {
	Emails  email.Repository
	Events  email.EventRepository
	Emailer *email.Service

	Logger *zap.Logger
}

type EmailHandler struct {
	emails  email.Repository
	events  email.EventRepository
	emailer *email.Service

	logger *zap.Logger
}

func NewEmailHandler(g *gin.RouterGroup, deps EmailHandlerDeps) {
	h := &EmailHandler{
		emails:  deps.Emails,
		events:  deps.Events,
		emailer: deps.Emailer,
		logger:  deps.Logger,
	}

	r := g.Group("/emails")
	{
		// Отправка писем с домена проекта доступна только администраторам:
		// открытая ручка — это открытый релей.
		r.POST("", auth.AdminOnly(), h.Send)
		r.GET("", auth.AdminOnly(), h.List)
		r.GET("/:id", auth.AdminOnly(), h.Get)
		r.GET("/:id/events", auth.AdminOnly(), h.Events)
	}
}

// Send ставит письмо в очередь.
//
// Отвечает 202, а не 200: письмо принято, но ещё не отправлено. Ошибки
// шаблона и адреса при этом возвращаются синхронно — рендер происходит здесь
// же, поэтому админка узнаёт о них сразу, а не из молчаливого failed через час.
func (h *EmailHandler) Send(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.emailer.Enqueue(c.Request.Context(), email.EnqueueInput{
		TemplateID:      req.TemplateID,
		TemplateVersion: req.TemplateVersion,
		ToEmail:         req.To,
		Data:            req.Data,
		Priority:        req.Priority,
		ScheduledAt:     req.ScheduledAt,
		IdempotencyKey:  req.IdempotencyKey,
		TraceID:         helper.GetTraceID(c),
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(nethttp.StatusAccepted, SendEmailResponse{
		EmailID:   res.EmailID.String(),
		Duplicate: res.Duplicate,
	})
}

// Get отдаёт состояние письма — по нему админка отвечает на вопрос
// «дошло ли».
func (h *EmailHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid email id"})
		return
	}

	found, err := h.emails.GetByID(c.Request.Context(), id)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(nethttp.StatusOK, toEmailResponse(*found))
}

// List отдаёт письма с фильтрами.
func (h *EmailHandler) List(c *gin.Context) {
	var req ListEmailsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	req.Params.ApplyDefaults(pagination.DefaultConfig)

	filter := email.Filter{
		From:   req.From,
		To:     req.Till,
		Limit:  req.Params.PageSize,
		Offset: (req.Params.Page - 1) * req.Params.PageSize,
	}
	if req.Status != nil {
		status := email.Status(*req.Status)
		filter.Status = &status
	}
	if req.To != nil {
		filter.ToEmail = *req.To
	}

	items, total, err := h.emails.List(c.Request.Context(), filter)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(nethttp.StatusOK, ListEmailsResponse{
		Items: toEmailResponses(items),
		Total: total,
	})
}

// Events отдаёт историю переходов письма: по ней видно, сколько было попыток
// и почему письмо не ушло.
func (h *EmailHandler) Events(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid email id"})
		return
	}

	list, err := h.events.ListByEmailID(c.Request.Context(), id)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{"items": toEventResponses(list)})
}
