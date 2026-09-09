package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

// Linker — часть домена, которой пользуется таск-менеджер: привязка бага
// к задаче и перенос статуса задачи обратно на баг.
type Linker interface {
	Link(ctx context.Context, params bug.LinkParams) (*bug.Model, error)
	Unlink(ctx context.Context, taskID uint) (*bug.Model, error)
	UnlinkBug(ctx context.Context, bugID uint) (*bug.Model, error)
	SyncStatus(ctx context.Context, params bug.SyncParams) (*bug.Model, error)
}

// LinkBugHandler обслуживает все три операции связи с задачей.
//
// Один обработчик, а не три: у них общий порт и общий вызывающий —
// таск-менеджер, — и разносить их значило бы трижды повторить один
// конструктор.
type LinkBugHandler struct {
	linker Linker

	logger *zap.Logger
}

func NewLinkBugHandler(linker Linker, logger *zap.Logger) *LinkBugHandler {
	return &LinkBugHandler{linker: linker, logger: logger}
}

type LinkBugCommand struct {
	BugID  uint
	TaskID uint
}

type UnlinkBugCommand struct {
	TaskID uint
}

// UnlinkByBugCommand снимает привязку с конкретного бага. Нужна, когда
// задача меняет баг: прежний ищется по своему идентификатору.
type UnlinkByBugCommand struct {
	BugID uint
}

type SyncBugCommand struct {
	TaskID uint
	Status string
}

func (h *LinkBugHandler) Link(ctx context.Context, cmd LinkBugCommand) (BugResult, error) {
	obj, err := h.linker.Link(ctx, bug.LinkParams{BugID: cmd.BugID, TaskID: cmd.TaskID})
	if err != nil {
		h.logger.Error("link bug", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, err
	}
	return toBugResult(*obj), nil
}

// Unlink снимает привязку. Задача без бага — обычный случай, поэтому
// «ничего не изменилось» сообщается флагом, а не ошибкой.
func (h *LinkBugHandler) Unlink(ctx context.Context, cmd UnlinkBugCommand) (BugResult, bool, error) {
	obj, err := h.linker.Unlink(ctx, cmd.TaskID)
	if err != nil {
		h.logger.Error("unlink bug", zap.Uint("task_id", cmd.TaskID), zap.Error(err))
		return BugResult{}, false, err
	}
	if obj == nil {
		return BugResult{}, false, nil
	}
	return toBugResult(*obj), true, nil
}

func (h *LinkBugHandler) Sync(ctx context.Context, cmd SyncBugCommand) (BugResult, bool, error) {
	obj, err := h.linker.SyncStatus(ctx, bug.SyncParams{
		TaskID: cmd.TaskID,
		Status: bug.Status(cmd.Status),
	})
	if err != nil {
		h.logger.Error("sync bug status", zap.Uint("task_id", cmd.TaskID), zap.Error(err))
		return BugResult{}, false, err
	}
	if obj == nil {
		return BugResult{}, false, nil
	}
	return toBugResult(*obj), true, nil
}

// UnlinkBug снимает привязку с конкретного бага.
func (h *LinkBugHandler) UnlinkBug(ctx context.Context, cmd UnlinkByBugCommand) (BugResult, bool, error) {
	obj, err := h.linker.UnlinkBug(ctx, cmd.BugID)
	if err != nil {
		h.logger.Error("unlink bug by id", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, false, err
	}
	if obj == nil {
		return BugResult{}, false, nil
	}
	return toBugResult(*obj), true, nil
}
