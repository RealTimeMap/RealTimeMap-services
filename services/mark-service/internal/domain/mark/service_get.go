package mark

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

// GetMarksInArea получение меток в области карты
func (s *Service) GetMarksInArea(ctx context.Context, filter Filter) ([]*Mark, error) {
	marks, err := s.markRepo.GetMarksInArea(ctx, filter)
	if err != nil {
		return nil, err
	}
	return marks, nil

}

// GetMarksInCluster получение сгруппированных меток по кластерам для отображения при большой области карты
func (s *Service) GetMarksInCluster(ctx context.Context, filter Filter) ([]*Cluster, error) {
	clusters, err := s.markRepo.GetMarksInCluster(ctx, filter)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (s *Service) GetDetailMark(ctx context.Context, markID uint) (*Mark, error) {
	mark, err := s.markRepo.GetByID(ctx, markID)
	if err != nil {
		return nil, err
	}
	return mark, nil
}

func (s *Service) GetUserMarks(ctx context.Context, userID uint, paginationParams pagination.Params) ([]*Mark, int64, error) {
	paginationParams.Defaults()
	objs, count, err := s.markRepo.GetUserMarks(ctx, userID, paginationParams)
	if err != nil {
		return nil, 0, err
	}
	return objs, count, nil
}
