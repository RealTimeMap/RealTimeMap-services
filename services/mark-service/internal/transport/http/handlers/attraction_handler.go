package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	httputils "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/attraction"
)

type AttractionDeps struct {
	UseCases *attraction.Application
	Logger   *zap.Logger
}

type attractionHandler struct {
	useCases *attraction.Application
	logger   *zap.Logger
}

func InitAttractionHandler(g *gin.RouterGroup, deps AttractionDeps) {
	h := &attractionHandler{
		useCases: deps.UseCases,
		logger:   deps.Logger,
	}
	r := g.Group("/attraction")
	{
		r.POST("/create", auth.AdminOnly(), h.Create)
		r.GET("/list", h.ListByCity)
		r.GET("/:attractionID", h.Get)
		r.PATCH("/:attractionID", auth.AdminOnly(), h.Update)
		r.DELETE("/:attractionID", auth.AdminOnly(), h.Delete)
	}
}

type (
	ListAttractionRequest struct {
		City string `form:"city" binding:"required,max=128"`
	}
	AttractionShortResponse struct {
		ID        uuid.UUID `json:"id"`
		Slug      string    `json:"slug"`
		Name      string    `json:"name"`
		Category  string    `json:"category"`
		Longitude float64   `json:"longitude"`
		Latitude  float64   `json:"latitude"`
	}
)

func ToAttractionShortResponse(obj attraction.AttractionShortResult) AttractionShortResponse {
	return AttractionShortResponse{
		ID:        obj.ID,
		Slug:      obj.Slug,
		Name:      obj.Name,
		Category:  obj.Category,
		Longitude: obj.Geom.Lon(),
		Latitude:  obj.Geom.Lat(),
	}
}

func parseAttractionID(c *gin.Context) (uuid.UUID, error) {
	raw := c.Param("attractionID")

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperror.NewFieldValidationError(
			"attractionID",
			"attractionID must be a uuid",
			"value_error.uuid",
			raw,
		)
	}
	return id, nil
}

func (h *attractionHandler) ListByCity(c *gin.Context) {
	var req ListAttractionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	objs, err := h.useCases.ListByCity.Handle(c.Request.Context(), attraction.ListByCityQuery{City: req.City})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, httputils.Map(objs, ToAttractionShortResponse))
}

func (h *attractionHandler) Get(c *gin.Context) {
	id, err := parseAttractionID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	obj, err := h.useCases.Get.Handle(c.Request.Context(), attraction.GetAttractionQuery{ID: id})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, ToAttractionResponse(obj))
}

type (
	CreateSlideBlockRequest struct {
		Position  uint           `json:"position"`
		BlockType string         `json:"blockType" binding:"required,oneof=attribute static"`
		Content   map[string]any `json:"content" binding:"required"`
	}
	CreateSlideRequest struct {
		Position uint                      `json:"position"`
		Duration uint                      `json:"duration" binding:"required,gt=0"`
		Blocks   []CreateSlideBlockRequest `json:"blocks" binding:"dive"`
	}
	CreateAttractionRequest struct {
		Slug     string               `json:"slug" binding:"required,max=255"`
		Name     string               `json:"name" binding:"required,max=255"`
		City     string               `json:"city" binding:"required,max=128"`
		Category string               `json:"category" binding:"required,max=128"`
		Publish  bool                 `json:"publish"`
		Lon      *float64             `json:"longitude" binding:"required,longitude"`
		Lat      *float64             `json:"latitude" binding:"required,latitude"`
		Slides   []CreateSlideRequest `json:"slides" binding:"dive"`
	}
)

type (
	SlideBlockResponse struct {
		ID        uuid.UUID      `json:"id"`
		Position  uint           `json:"position"`
		BlockType string         `json:"blockType"`
		Content   map[string]any `json:"content"`
	}
	SlideResponse struct {
		ID       uuid.UUID            `json:"id"`
		Position uint                 `json:"position"`
		Duration uint                 `json:"duration"`
		Blocks   []SlideBlockResponse `json:"blocks"`
	}
	AttractionResponse struct {
		ID        uuid.UUID       `json:"id"`
		Slug      string          `json:"slug"`
		Name      string          `json:"name"`
		City      string          `json:"city"`
		Category  string          `json:"category"`
		Status    string          `json:"status"`
		Longitude float64         `json:"longitude"`
		Latitude  float64         `json:"latitude"`
		Slides    []SlideResponse `json:"slides"`
		CreatedAt time.Time       `json:"createdAt"`
		UpdatedAt time.Time       `json:"updatedAt"`
	}
)

func toSlideBlockResponse(b attraction.SlideBlockResult) SlideBlockResponse {
	return SlideBlockResponse{
		ID:        b.ID,
		Position:  b.Position,
		BlockType: b.BlockType,
		Content:   b.Content,
	}
}

func toSlideResponse(s attraction.SlideResult) SlideResponse {
	return SlideResponse{
		ID:       s.ID,
		Position: s.Position,
		Duration: s.Duration,
		Blocks:   httputils.Map(s.Blocks, toSlideBlockResponse),
	}
}

func ToAttractionResponse(obj attraction.AttractionResult) AttractionResponse {
	return AttractionResponse{
		ID:        obj.ID,
		Slug:      obj.Slug,
		Name:      obj.Name,
		City:      obj.City,
		Category:  obj.Category,
		Status:    obj.Status,
		Longitude: obj.Geom.Lon(),
		Latitude:  obj.Geom.Lat(),
		Slides:    httputils.Map(obj.Slides, toSlideResponse),
		CreatedAt: obj.CreatedAt,
		UpdatedAt: obj.UpdatedAt,
	}
}

func (h *attractionHandler) Create(c *gin.Context) {
	var req CreateAttractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	obj, err := h.useCases.Create.Handle(c.Request.Context(), attraction.CreateAttractionCommand{
		Slug:     req.Slug,
		Name:     req.Name,
		City:     req.City,
		Category: req.Category,
		Publish:  req.Publish,
		Geom:     types.Point{Point: orb.Point{*req.Lon, *req.Lat}},
		Slides:   toSlideCommands(req.Slides),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusCreated, ToAttractionResponse(obj))
}

func toSlideCommands(reqs []CreateSlideRequest) []attraction.CreateSlideCommand {
	slides := make([]attraction.CreateSlideCommand, 0, len(reqs))
	for _, s := range reqs {
		blocks := make([]attraction.CreateSlideBlockCommand, 0, len(s.Blocks))
		for _, b := range s.Blocks {
			blocks = append(blocks, attraction.CreateSlideBlockCommand{
				Position:  b.Position,
				BlockType: b.BlockType,
				Content:   b.Content,
			})
		}
		slides = append(slides, attraction.CreateSlideCommand{
			Position: s.Position,
			Duration: s.Duration,
			Blocks:   blocks,
		})
	}
	return slides
}

// UpdateAttractionRequest — частичное обновление: отсутствующее поле не меняется.
// slides заменяет все слайды целиком, [] удаляет их.
type UpdateAttractionRequest struct {
	Slug     *string               `json:"slug" binding:"omitempty,max=255"`
	Name     *string               `json:"name" binding:"omitempty,max=255"`
	City     *string               `json:"city" binding:"omitempty,max=128"`
	Category *string               `json:"category" binding:"omitempty,max=128"`
	Status   *string               `json:"status" binding:"omitempty,oneof=draft published archived"`
	Lon      *float64              `json:"longitude" binding:"omitempty,longitude"`
	Lat      *float64              `json:"latitude" binding:"omitempty,latitude"`
	Slides   *[]CreateSlideRequest `json:"slides" binding:"omitempty,dive"`
}

var ErrCoordsPairRequired = apperror.NewFieldValidationError(
	"longitude",
	"longitude and latitude must be passed together",
	"value_error.missing",
	nil,
)

func (h *attractionHandler) Update(c *gin.Context) {
	id, err := parseAttractionID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req UpdateAttractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	cmd := attraction.UpdateAttractionCommand{
		ID:       id,
		Slug:     req.Slug,
		Name:     req.Name,
		City:     req.City,
		Category: req.Category,
		Status:   req.Status,
	}
	if (req.Lon == nil) != (req.Lat == nil) {
		errorhandler.HandleError(c, ErrCoordsPairRequired, h.logger)
		return
	}
	if req.Lon != nil {
		cmd.Geom = &types.Point{Point: orb.Point{*req.Lon, *req.Lat}}
	}
	if req.Slides != nil {
		slides := toSlideCommands(*req.Slides)
		cmd.Slides = &slides
	}

	obj, err := h.useCases.Update.Handle(c.Request.Context(), cmd)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, ToAttractionResponse(obj))
}

func (h *attractionHandler) Delete(c *gin.Context) {
	id, err := parseAttractionID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := h.useCases.Delete.Handle(c.Request.Context(), attraction.DeleteAttractionCommand{ID: id}); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}
