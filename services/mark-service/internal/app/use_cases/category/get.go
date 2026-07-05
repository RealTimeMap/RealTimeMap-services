package category

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
)

type Getter interface {
	GetAll(ctx context.Context) ([]*category.Category, error)
}

type GetterCategoryHandler struct {
	getter Getter

	logger *zap.Logger
}

func NewGetterCategoryHandler(getter Getter, logger *zap.Logger) *GetterCategoryHandler {
	return &GetterCategoryHandler{getter: getter, logger: logger}
}

func (h *GetterCategoryHandler) Handle(ctx context.Context) ([]CategoryResult, error) {
	objs, err := h.getter.GetAll(ctx)

	if err != nil {
		return nil, err
	}

	return toMultiCategoryResult(objs), nil
}
