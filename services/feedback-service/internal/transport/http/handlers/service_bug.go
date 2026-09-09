package handlers

import (
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ServiceBugHandler обслуживает межсервисные вызовы таск-менеджера.
//
// Отдельный обработчик, а не флаг в BugHandler: у этих маршрутов другая
// аутентификация (ключ сервиса вместо заголовков пользователя) и другой
// потребитель, и смешивать их в одном дереве значило бы каждый раз
// выяснять, кто именно пришёл.
type ServiceBugHandler struct {
	useCase *bug.Application

	logger *zap.Logger
}

type ServiceBugHandlerDeps struct {
	UseCase *bug.Application

	Logger *zap.Logger
}

func NewServiceBugHandler(g *gin.RouterGroup, deps ServiceBugHandlerDeps) {
	h := &ServiceBugHandler{
		useCase: deps.UseCase,
		logger:  deps.Logger,
	}

	r := g.Group("/bugs")
	{
		// Перечень багов, которые можно взять в задачу.
		r.GET("", h.ListOpen)
		// Один баг целиком — с логами и обстановкой воспроизведения.
		r.GET("/:id", h.Get)
		// Привязка бага к задаче и снятие привязки.
		r.PUT("/:id/task", h.Link)
		r.DELETE("/:id/task", h.UnlinkBug)
		r.DELETE("/task/:taskId", h.Unlink)
		// Обратная синхронизация: статус задачи переносится на баг.
		r.PATCH("/task/:taskId/status", h.SyncStatus)
	}
}

// ServiceBugListParams — параметры перечня для таск-менеджера.
//
// По умолчанию отдаются только открытые и ещё не занятые баги: именно
// их предлагают привязать к задаче. Флаги позволяют снять сужение —
// например, чтобы показать баг, уже привязанный к текущей задаче.
type ServiceBugListParams struct {
	pagination.Params
	Tag *string `form:"tag" binding:"omitempty"`

	IncludeClosed bool `form:"includeClosed"`
	IncludeLinked bool `form:"includeLinked"`
}

func (h *ServiceBugHandler) ListOpen(c *gin.Context) {
	var req ServiceBugListParams
	if err := c.ShouldBindQuery(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCase.List.Handle(c.Request.Context(), bug.ListBugCommand{
		Tag:          req.Tag,
		Pagination:   req.Params,
		OnlyOpen:     !req.IncludeClosed,
		OnlyUnlinked: !req.IncludeLinked,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, mapToServiceListResponse(res))
}

// Get отдаёт баг со всеми подробностями отчёта.
func (h *ServiceBugHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	res, err := h.useCase.Get.Handle(c.Request.Context(), bug.GetBugCommand{BugID: id})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, mapToServiceDetail(res))
}

type LinkBugRequest struct {
	TaskID uint `json:"taskId" binding:"required"`
}

func (h *ServiceBugHandler) Link(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	var req LinkBugRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCase.Link.Link(c.Request.Context(), bug.LinkBugCommand{
		BugID:  id,
		TaskID: req.TaskID,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, mapToServiceItem(res))
}

// UnlinkBug снимает привязку с конкретного бага.
//
// Отдельно от Unlink по задаче: когда задача меняет баг, прежний уже не
// найти по её идентификатору — привязка задачи указывает на новый.
func (h *ServiceBugHandler) UnlinkBug(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	res, changed, err := h.useCase.Link.UnlinkBug(c.Request.Context(), bug.UnlinkByBugCommand{
		BugID: id,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	if !changed {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, mapToServiceItem(res))
}

func (h *ServiceBugHandler) Unlink(c *gin.Context) {
	taskID, err := parseUintParam(c, "taskId")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	res, changed, err := h.useCase.Link.Unlink(c.Request.Context(), bug.UnlinkBugCommand{
		TaskID: taskID,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	// У задачи не было бага — менять нечего. Это штатный исход, и
	// вызывающему достаточно знать, что тела нет.
	if !changed {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, mapToServiceItem(res))
}

// SyncBugStatusRequest — новый статус бага, снятый с задачи.
//
// Список допустимых значений здесь не перечислен: статус «in work»
// содержит пробел, а oneof разделяет варианты как раз пробелом. Проверку
// делает домен — он и так единственный источник истины по статусам.
type SyncBugStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *ServiceBugHandler) SyncStatus(c *gin.Context) {
	taskID, err := parseUintParam(c, "taskId")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	var req SyncBugStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, changed, err := h.useCase.Link.Sync(c.Request.Context(), bug.SyncBugCommand{
		TaskID: taskID,
		Status: req.Status,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	if !changed {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, mapToServiceItem(res))
}

// parseUintParam читает положительный числовой параметр пути.
func parseUintParam(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, apperror.NewFieldValidationError(
			name, "must be a positive integer", "value_error", raw,
		)
	}
	return uint(value), nil
}
