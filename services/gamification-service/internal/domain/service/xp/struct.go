package xp

import "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"

type CreditInput struct {
	UserID     uint
	Amount     int
	SourceType model.SourceType
	SourceID   uint
	Reason     *string
}

type CreditResult struct {
	Operation *model.XPOperation
	Progress  *model.UserProgress
	LeveledUp bool
}
