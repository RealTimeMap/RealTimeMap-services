package group

import (
	"time"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
)

type GroupResult struct {
	ID          uint
	UserID      uint
	Name        string
	Description *string
	CreatedAt   time.Time
}

func toGroupResult(obj *srv.Model) GroupResult {
	return GroupResult{
		ID:          obj.ID,
		UserID:      obj.UserID,
		Name:        obj.Name,
		Description: obj.Description,
		CreatedAt:   obj.CreatedAt,
	}
}

func toListGroupResult(objs []*srv.Model) []GroupResult {
	res := make([]GroupResult, 0, len(objs))
	for _, item := range objs {
		res = append(res, toGroupResult(item))
	}
	return res
}
