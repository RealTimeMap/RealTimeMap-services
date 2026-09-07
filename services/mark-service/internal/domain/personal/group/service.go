package group

import (
	"go.uber.org/zap"
)

type Service struct {
	repo Repository

	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}
