package personal

import (
	"context"

	"go.uber.org/zap"

	personalsrv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type RevisionGetter interface {
	ActualRevision(ctx context.Context, userID uint) (uint, error)
}

type SyncMarkHandler struct {
	sources  []ChangeSource
	revision RevisionGetter
	logger   *zap.Logger
}

func NewSyncMarkHandler(revision RevisionGetter, logger *zap.Logger, sources ...ChangeSource) *SyncMarkHandler {
	return &SyncMarkHandler{
		sources:  sources,
		revision: revision,
		logger:   logger,
	}
}

type SyncCommand struct {
	Since *uint
	Limit int
}

type SyncResult struct {
	Sections map[string]personalsrv.Changes[any] `json:"sections"`
	Cursor   uint                                `json:"cursor"`
	HasMore  bool                                `json:"hasMore"`
	UpdateTo uint                                `json:"updateTo"`
}

type ChangeSource interface {
	Name() string
	ListChanges(ctx context.Context, userID uint, since *uint, upTo uint, limit int) (personalsrv.Changes[any], error)
}

func (h *SyncMarkHandler) Handle(ctx context.Context, userID uint, cmd SyncCommand) (SyncResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Sync"))

	upTo, err := h.revision.ActualRevision(ctx, userID)
	if err != nil {
		return SyncResult{}, err
	}

	result := SyncResult{
		Cursor:   upTo,
		UpdateTo: upTo,
		Sections: map[string]personalsrv.Changes[any]{},
	}

	for _, s := range h.sources {
		ch, err := s.ListChanges(ctx, userID, cmd.Since, upTo, cmd.Limit)
		if err != nil {
			return SyncResult{}, err
		}

		result.Sections[s.Name()] = ch
		result.HasMore = result.HasMore || ch.HasMore
		if ch.Cursor < result.Cursor {
			result.Cursor = ch.Cursor
		}
	}
	return result, nil
}
