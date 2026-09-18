package personal

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Get возвращает одну группу владельца.
func (s *GroupService) Get(ctx context.Context, groupID uuid.UUID, userID uint) (Group, error) {
	s.logger.Info("start Get", zap.String("layer", "domain service"))

	return s.repo.GetByID(ctx, groupID, userID)
}
