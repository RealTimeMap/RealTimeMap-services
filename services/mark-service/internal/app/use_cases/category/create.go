package category

import (
	"context"

	cat "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
)

type Creator interface {
	Create(ctx context.Context, category cat.CreateCategoryParams) (*cat.Category, error)
}

type CreateCategoryCommand struct {
	CategoryName string
	Color        string
	Icon         string
}

func (c CreateCategoryCommand) Validate() error {
	return nil
}

type CreateCategoryHandler struct {
	creator Creator

	logger *zap.Logger
}

func NewCreateCategoryCommand(creator Creator, logger *zap.Logger) *CreateCategoryHandler {
	return &CreateCategoryHandler{
		creator: creator,
		logger:  logger,
	}
}

func (h *CreateCategoryHandler) Handle(ctx context.Context, cmd CreateCategoryCommand) (CategoryResult, error) {
	if err := cmd.Validate(); err != nil {
		return CategoryResult{}, err
	}

	res, err := h.creator.Create(ctx, cat.CreateCategoryParams{
		CategoryName: cmd.CategoryName,
		Color:        cmd.Color,
		Icon:         cmd.Icon,
	})

	if err != nil {
		return CategoryResult{}, err
	}

	return CategoryResult{
		ID:           res.ID,
		CategoryName: res.CategoryName,
		Color:        res.Color,
		Icon:         cmd.Icon,
	}, nil
}
