package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

// Reviewer — часть домена, через которую разработчик выносит решение по
// отчёту: подтверждает, отклоняет или возвращает на повторную проверку.
type Reviewer interface {
	Confirm(ctx context.Context, params bug.ConfirmParams) (*bug.Model, bool, error)
	Reject(ctx context.Context, params bug.RejectParams) (*bug.Model, error)
	Reopen(ctx context.Context, bugID uint) (*bug.Model, error)
}

// ReviewBugHandler обслуживает проверку бага разработчиком.
//
// Проверка отделена от привязки к задаче: большинство отчётов не
// подтверждается, и решение по ним принимается до того, как по багу
// заводят задачу.
type ReviewBugHandler struct {
	reviewer  Reviewer
	publisher EventPublisher

	logger *zap.Logger
}

// NewReviewBugHandler собирает обработчик. nil вместо publisher означает
// выключенную шину: события просто не публикуются.
func NewReviewBugHandler(reviewer Reviewer, publisher EventPublisher, logger *zap.Logger) *ReviewBugHandler {
	if publisher == nil {
		publisher = NoOpEventPublisher{}
	}
	return &ReviewBugHandler{reviewer: reviewer, publisher: publisher, logger: logger}
}

type ConfirmBugCommand struct {
	BugID   uint
	Comment string
}

type RejectBugCommand struct {
	BugID   uint
	Reason  string
	Comment string
}

type ReopenBugCommand struct {
	BugID uint
}

func (h *ReviewBugHandler) Confirm(ctx context.Context, cmd ConfirmBugCommand) (BugResult, error) {
	obj, first, err := h.reviewer.Confirm(ctx, bug.ConfirmParams{BugID: cmd.BugID, Comment: cmd.Comment})
	if err != nil {
		h.logger.Error("confirm bug", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, err
	}

	// Награда положена автору отчёта один раз — за первое подтверждение.
	// Анонимный отчёт засчитать некому, поэтому без UserID событие не
	// уходит вовсе.
	if first && obj.UserID != nil {
		confirmed := *obj
		publishAsync(h.logger, events.BugConfirmed, func(ctx context.Context) error {
			return h.publisher.PublishBugConfirmed(ctx, &confirmed)
		})
	}

	return toBugResult(*obj), nil
}

func (h *ReviewBugHandler) Reject(ctx context.Context, cmd RejectBugCommand) (BugResult, error) {
	obj, err := h.reviewer.Reject(ctx, bug.RejectParams{
		BugID:   cmd.BugID,
		Reason:  bug.RejectReason(cmd.Reason),
		Comment: cmd.Comment,
	})
	if err != nil {
		h.logger.Error("reject bug", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, err
	}
	return toBugResult(*obj), nil
}

func (h *ReviewBugHandler) Reopen(ctx context.Context, cmd ReopenBugCommand) (BugResult, error) {
	obj, err := h.reviewer.Reopen(ctx, cmd.BugID)
	if err != nil {
		h.logger.Error("reopen bug", zap.Uint("bug_id", cmd.BugID), zap.Error(err))
		return BugResult{}, err
	}
	return toBugResult(*obj), nil
}
