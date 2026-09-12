package personal

import (
	"context"
)

// Delete помечает метку удалённой. Ревизия инкрементится в той же транзакции:
// без этого soft-deleted строка не попадёт в окно ListChanges и клиент никогда
// не узнает об удалении.
func (s *Service) Delete(ctx context.Context, markID, userID uint) error {
	obj, err := s.repo.GetByID(ctx, markID, userID)
	if err != nil {
		return err
	}

	if err = s.checkOwnerShip(obj, userID); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context) error {
		revision, err := s.revisionRepo.UpsertRevision(txCtx, userID)
		if err != nil {
			return err
		}

		if err := s.repo.SetRevision(txCtx, markID, revision); err != nil {
			return err
		}

		return s.repo.Delete(txCtx, markID)
	})
}

func (s *Service) checkOwnerShip(obj Model, userID uint) error {
	if obj.UserID != userID {
		return ErrOwnerShip()
	}
	return nil
}
