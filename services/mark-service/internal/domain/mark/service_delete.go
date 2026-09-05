package mark

import (
	"context"
)

// DeleteMark удаление метки
func (s *Service) DeleteMark(ctx context.Context, userID, markID uint) error {
	obj, err := s.markRepo.GetByID(ctx, markID)
	if err != nil {
		return err
	}

	if err := s.checkOwnerShip(obj, userID); err != nil {
		return err
	}

	return s.markRepo.Delete(ctx, markID)
}
