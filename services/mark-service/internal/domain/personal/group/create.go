package group

import (
	"context"

	"go.uber.org/zap"
)

type CreateGroupParams struct {
	Name        string
	Description *string
	UserID      uint
}

func (s *Service) CreateGroup(ctx context.Context, param CreateGroupParams) (*Model, error) {
	s.logger.Info("start CreateGroup", zap.String("layer", "domain service"))

	payload := &Model{
		Name:        param.Name,
		Description: param.Description,
		UserID:      param.UserID,
	}

	err := s.repo.Create(ctx, payload)
	if err != nil {
		return nil, err
	}
	return payload, err
}
