package personal

import (
	"context"
)

// Get возвращает одну персональную метку владельца вместе с группами.
func (s *Service) Get(ctx context.Context, markID, userID uint) (Model, error) {
	obj, err := s.repo.GetByID(ctx, markID, userID)
	if err != nil {
		return Model{}, err
	}

	if err = s.checkOwnerShip(obj, userID); err != nil {
		return Model{}, err
	}

	return obj, nil
}
