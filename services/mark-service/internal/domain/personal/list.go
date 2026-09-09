package personal

import "context"

type SyncParams struct {
	Since *uint
	Limit int
}

func (s *Service) ListChanges(ctx context.Context, userID uint, since *uint, upTo uint, limit int) (Changes[Model], error) {
	objs, err := s.repo.List(ctx, userID, since, upTo, limit)
	if err != nil {
		return Changes[Model]{}, err
	}

	return toChanges(objs, upTo, limit), nil
}

func (s *Service) Name() string {
	return "personalMarks"
}
