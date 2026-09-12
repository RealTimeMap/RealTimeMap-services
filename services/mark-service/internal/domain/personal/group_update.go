package personal

import (
	"context"

	"go.uber.org/zap"
)

// UpdateGroupParams — частичное обновление: nil-поле означает "не трогать".
type UpdateGroupParams struct {
	Name        *string
	Description *string
}

func (s *GroupService) UpdateGroup(ctx context.Context, params UpdateGroupParams, groupID, userID uint) (*Group, error) {
	s.logger.Info("start UpdateGroup", zap.String("layer", "domain service"))

	obj, err := s.repo.GetByID(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		revision, err := s.revisionRepo.UpsertRevision(txCtx, userID)
		if err != nil {
			return err
		}

		if params.Name != nil {
			obj.Name = *params.Name
		}
		if params.Description != nil {
			obj.Description = params.Description
		}
		obj.Revision = revision

		return s.repo.Update(txCtx, &obj)
	})
	if err != nil {
		return nil, err
	}

	return &obj, nil
}
