package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

// ProfileProvider — порт к profile-сервису для обогащения комментариев авторами.
type ProfileProvider interface {
	GetUserProfileByIDs(ctx context.Context, ids []uint) ([]*comment.UserProfile, error)
}

// attachAuthors проставляет автора каждому комментарию через profile-сервис.
// При недоступности сервиса — мягкая деградация на локальные данные комментария.
func attachAuthors(ctx context.Context, provider ProfileProvider, logger *zap.Logger, comments []*comment.Comment) {
	if len(comments) == 0 {
		return
	}

	ids := uniqueUserIDs(comments)
	profiles, err := provider.GetUserProfileByIDs(ctx, ids)

	byID := make(map[uint]*comment.UserProfile, len(profiles))
	if err != nil {
		logger.Warn("profile-service degraded, using local author fallback", zap.Error(err))
	} else {
		for _, p := range profiles {
			byID[p.ID] = p
		}
	}

	for _, c := range comments {
		if p, ok := byID[c.UserID]; ok {
			c.Author = p
		} else {
			c.Author = localFallback(c)
		}
	}
}

func uniqueUserIDs(comments []*comment.Comment) []uint {
	ids := make([]uint, len(comments))
	for i, c := range comments {
		ids[i] = c.UserID
	}
	return utils.UniqueValues(ids)
}

func localFallback(c *comment.Comment) *comment.UserProfile {
	return &comment.UserProfile{
		ID:       c.UserID,
		Username: c.Username,
	}
}
