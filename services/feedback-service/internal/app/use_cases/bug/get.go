package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

// Getter — часть домена, отдающая один баг целиком.
type Getter interface {
	GetByID(ctx context.Context, id uint) (*bug.Model, error)
}

// GetBugHandler отдаёт баг со всеми подробностями: описанием, логами и
// обстановкой, в которой он воспроизвёлся.
//
// Отдельно от перечня: логи весят прилично, и таскать их в списке из
// полусотни багов значило бы качать мегабайты ради трёх строк заголовка.
type GetBugHandler struct {
	getter Getter

	logger *zap.Logger
}

func NewGetBugHandler(getter Getter, logger *zap.Logger) *GetBugHandler {
	return &GetBugHandler{getter: getter, logger: logger}
}

type GetBugCommand struct {
	BugID uint
}

func (h *GetBugHandler) Handle(ctx context.Context, cmd GetBugCommand) (BugResult, error) {
	obj, err := h.getter.GetByID(ctx, cmd.BugID)
	if err != nil {
		h.logger.Error("get bug", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, err
	}
	return toBugResult(*obj), nil
}
