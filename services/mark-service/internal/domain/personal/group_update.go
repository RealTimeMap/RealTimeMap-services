package personal

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UpdateGroupParams — частичное обновление: nil-поле означает "не трогать".
type UpdateGroupParams struct {
	Name        *string
	Description *string
	Color       *string
	Icon        *string
}

func (s *GroupService) UpdateGroup(ctx context.Context, params UpdateGroupParams, groupID uuid.UUID, userID uint) (*Group, error) {
	s.logger.Info("start UpdateGroup", zap.String("layer", "domain service"))

	if params.Color != nil {
		if err := validateGroupColor(*params.Color); err != nil {
			return nil, err
		}
	}
	if params.Icon != nil {
		if err := validateGroupIcon(*params.Icon); err != nil {
			return nil, err
		}
	}

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
		if params.Color != nil {
			obj.Color = *params.Color
		}
		if params.Icon != nil {
			obj.Icon = *params.Icon
		}
		obj.Revision = revision

		return s.repo.Update(txCtx, &obj)
	})
	if err != nil {
		return nil, err
	}

	return &obj, nil
}
