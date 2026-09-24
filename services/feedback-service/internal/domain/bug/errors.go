package bug

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

var (
	ErrBugTagUnavailable = func(tag string) error {
		return apperror.NewFieldValidationError("tag", "tag is not allowed", "value_error", tag)
	}
	ErrBugStatusUnavailable = func(status string) error {
		return apperror.NewFieldValidationError("status", "status is not allowed", "value_error", status)
	}
	ErrRejectReasonUnavailable = func(reason string) error {
		return apperror.NewFieldValidationError("reason", "reason is not allowed", "value_error", reason)
	}

	ErrBugNotFound = func(id uint) error {
		return apperror.NewNotFoundErrorByID("bug", id)
	}

	// ErrBugAlreadyLinked защищает от увода бага из чужой задачи:
	// привязка одна, и переписывать её молча нельзя — иначе первая
	// задача осталась бы без бага, о чём никто бы не узнал.
	ErrBugAlreadyLinked = func(id, taskID uint) error {
		return apperror.NewConflictError(
			"taskId",
			fmt.Sprintf("bug %d is already linked to task %d", id, taskID),
			taskID,
		)
	}

	// ErrBugClosed не даёт привязать к задаче баг, работа над которым
	// уже завершена или отменена.
	ErrBugClosed = func(id uint, status string) error {
		return apperror.NewConflictError(
			"status",
			fmt.Sprintf("bug %d is %s and cannot be taken into work", id, status),
			status,
		)
	}

	// ErrBugNotConfirmed не даёт взять в задачу баг, который разработчик
	// ещё не воспроизвёл: большинство отчётов не подтверждается, и
	// задачи по ним были бы пустой работой.
	ErrBugNotConfirmed = func(id uint) error {
		return apperror.NewConflictError(
			"status",
			fmt.Sprintf("bug %d is not confirmed yet", id),
			string(New),
		)
	}

	// ErrBugReviewForbidden — решение по багу нельзя принять в его
	// текущем статусе: например, подтвердить уже отклонённый или
	// отклонить тот, над которым идёт работа.
	ErrBugReviewForbidden = func(id uint, status Status, action string) error {
		return apperror.NewConflictError(
			"status",
			fmt.Sprintf("bug %d is %s and cannot be %s", id, status, action),
			string(status),
		)
	}
)
