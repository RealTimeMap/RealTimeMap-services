package personal

import (
	"context"

	"go.uber.org/zap"
)

// DeleteGroup помечает группу удалённой.
//
// Непустая группа не удаляется: метка обязана принадлежать хотя бы одной группе
// (см. ErrGroupsRequired в Create), поэтому удаление последней группы метки
// оставило бы её висеть без привязки.
func (s *GroupService) DeleteGroup(ctx context.Context, groupID, userID uint) error {
	s.logger.Info("start DeleteGroup", zap.String("layer", "domain service"))

	obj, err := s.repo.GetByID(ctx, groupID, userID)
	if err != nil {
		return err
	}

	count, err := s.repo.CountMarks(ctx, groupID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrGroupNotEmpty(count)
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		revision, err := s.revisionRepo.UpsertRevision(txCtx, userID)
		if err != nil {
			return err
		}

		if err := s.repo.SetRevision(txCtx, groupID, revision); err != nil {
			return err
		}

		return s.repo.Delete(txCtx, &obj)
	})
}
