package personal

import "context"

type SyncParams struct {
	Since *uint
	UpTo  uint
	Limit int
}

func (s *Service) Sync(ctx context.Context, userID uint, params SyncParams) (Changes, error) {
	obj, err := s.repo.List(ctx, userID, params.Since, params.UpTo, params.Limit)
	if err != nil {
		return Changes{}, err
	}

	return obj, nil
}
