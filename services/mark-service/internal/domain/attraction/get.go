package attraction

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) ListByCity(ctx context.Context, city string) ([]Model, error) {
	s.logger.Info("start ListByCity", zap.String("layer", "domain service"), zap.String("city", city))

	city = strings.TrimSpace(city)
	if city == "" {
		return nil, ErrRequired("city")
	}
	if len([]rune(city)) > maxCityLen {
		return nil, ErrTooLong("city", maxCityLen, city)
	}
	return s.repo.ListByCity(ctx, city)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Model, error) {
	s.logger.Info("start Get", zap.String("layer", "domain service"), zap.String("id", id.String()))
	return s.repo.GetPublished(ctx, id)
}
