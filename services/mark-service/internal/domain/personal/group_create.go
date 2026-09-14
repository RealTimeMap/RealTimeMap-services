package personal

import (
	"context"

	"go.uber.org/zap"
)

type CreateGroupParams struct {
	Name        string
	Description *string
	UserID      uint
	Color       string
	Icon        string // Иконка из iconfy
}

func (s *GroupService) CreateGroup(ctx context.Context, params CreateGroupParams) (*Group, error) {
	s.logger.Info("start CreateGroup", zap.String("layer", "domain service"))

	if err := validateGroupColor(params.Color); err != nil {
		return nil, err
	}
	if err := validateGroupIcon(params.Icon); err != nil {
		return nil, err
	}

	var obj *Group
	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		revision, err := s.revisionRepo.UpsertRevision(txCtx, params.UserID)
		if err != nil {
			return err
		}
		payload := &Group{
			Name:        params.Name,
			Description: params.Description,
			UserID:      params.UserID,
			Revision:    revision,
			Color:       params.Color,
			Icon:        params.Icon,
		}
		if err := s.repo.Create(txCtx, payload); err != nil {
			return err
		}

		obj = payload
		return nil
	})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
