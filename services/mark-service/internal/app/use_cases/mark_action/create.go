package mark_action

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	ctxHelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type MarkCreateCommand struct {
	User           ctxHelper.UserInput
	MarkName       string
	AdditionalInfo *string
	CategoryId     int
	StartAt        time.Time
	EndAt          *time.Time
	Geom           types.Point
	Geohash        string
	Photos         []mediavalidator.PhotoInput
}

func (c MarkCreateCommand) Validate() error {
	if c.MarkName == "" {
		return apperror.NewRequiredError("markName")
	}
	if c.CategoryId < 0 {
		return apperror.NewFieldValidationError("category_id", "categoryID must be greater than 0", "value_error", c.CategoryId)
	}
	return nil
}

type MarkCreator interface {
	Create(ctx context.Context, user ctxHelper.UserInput, cmd mark.CreateMarkParams) (*mark.Mark, error)
}

type CreateMarkHandler struct {
	marks MarkCreator

	logger *zap.Logger
}

func NewCreateMarkHandler(marks MarkCreator, logger *zap.Logger) *CreateMarkHandler {
	return &CreateMarkHandler{
		marks:  marks,
		logger: logger,
	}
}

func (h *CreateMarkHandler) Handle(ctx context.Context, cmd MarkCreateCommand) (MarkResult, error) {
	h.logger.Info("start create mark UseCase")
	if err := cmd.Validate(); err != nil {
		return MarkResult{}, err
	}

	obj, err := h.marks.Create(ctx, cmd.User, mark.CreateMarkParams{
		MarkName:       cmd.MarkName,
		AdditionalInfo: cmd.AdditionalInfo,
		CategoryId:     cmd.CategoryId,
		StartAt:        cmd.StartAt,
		EndAt:          cmd.EndAt,
		Geom:           cmd.Geom,
		Geohash:        cmd.Geohash,
		Photos:         cmd.Photos,
	})

	if err != nil {
		return MarkResult{}, err
	}

	photos := make([]string, 0, len(obj.Photos))
	for _, photo := range obj.Photos {
		photos = append(photos, photo.URL)
	}

	return MarkResult{
		ID:       obj.ID,
		MarkName: obj.MarkName,
		Geom:     obj.Geom,
		Photos:   photos,
	}, nil
}
