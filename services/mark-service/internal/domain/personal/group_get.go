package personal

import (
	"context"

	"go.uber.org/zap"
)

// Get возвращает одну группу владельца.
func (s *GroupService) Get(ctx context.Context, groupID, userID uint) (Group, error) {
	s.logger.Info("start Get", zap.String("layer", "domain service"))

	return s.repo.GetByID(ctx, groupID, userID)
}
