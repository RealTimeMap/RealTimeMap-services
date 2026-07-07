package mark_action

import (
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	ctxHelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type MarkCreateCommand struct {
	User           ctxHelper.UserInput
	MarkName       string
	AdditionalInfo *string
	CategoryId     uint
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
	marks     MarkCreator
	publisher EventPublisher

	logger *zap.Logger
}

func NewCreateMarkHandler(marks MarkCreator, publisher EventPublisher, logger *zap.Logger) *CreateMarkHandler {
	return &CreateMarkHandler{
		marks:     marks,
		publisher: publisher,
		logger:    logger,
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

	h.publishCreated(ctx, obj, cmd.User.UserID)

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

// publishCreated публикует событие создания метки.
// Ошибка публикации не влияет на результат операции — метка уже создана,
// поэтому сбой шины только логируется.
func (h *CreateMarkHandler) publishCreated(ctx context.Context, obj *mark.Mark, userID int) {
	if h.publisher == nil {
		return
	}

	payload := events.MarkPayload{
		MarkID:         int(obj.ID),
		CategoryID:     int(obj.CategoryID),
		OwnerID:        userID,
		MarkName:       obj.MarkName,
		AdditionalInfo: obj.AdditionalInfo,
		IsEnded:        obj.IsEnded,
	}

	event := events.NewMarkCreate(payload)

	// key = ownerId для партиционирования: события одной метки/владельца в одну партицию.
	if err := h.publisher.PublishWithMeta(ctx, producer.EventMeta{
		EventType: "mark.created",
		UserID:    strconv.Itoa(int(obj.UserID)),
		SourceID:  strconv.Itoa(int(obj.ID)),
		Timestamp: time.Now().Format(time.RFC3339)},
		event); err != nil {
		h.logger.Error("publish markCreated event failed", zap.Error(err))
	}
}
