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
		return apperror.NewFieldValidationError("tag", "status is not allowed", "value_error", status)
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
)
