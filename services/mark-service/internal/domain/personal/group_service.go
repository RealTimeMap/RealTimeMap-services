package personal

import (
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
)

type GroupService struct {
	repo         GroupRepository
	revisionRepo RevisionRepository
	tx           txmanager.TxManager
	logger       *zap.Logger
}

func NewGroupService(repo GroupRepository, revisionRepo RevisionRepository, tx txmanager.TxManager, logger *zap.Logger) *GroupService {
	return &GroupService{
		repo:         repo,
		revisionRepo: revisionRepo,
		tx:           tx,
		logger:       logger,
	}
}
