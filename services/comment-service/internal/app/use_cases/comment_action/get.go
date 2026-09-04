package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentGetter interface {
	GetComments(ctx context.Context, filters comment.CommentFilter) ([]*comment.Comment, bool, error)
}

// ReactionChecker отдаёт лайки читателя по странице комментариев одним запросом.
type ReactionChecker interface {
	LikedByUser(ctx context.Context, commentIDs []uint, userID uint) (map[uint]bool, error)
}

type GetCommentsHandler struct {
	getter    CommentGetter
	provider  ProfileProvider
	reactions ReactionChecker

	logger *zap.Logger
}

func NewGetCommentsHandler(getter CommentGetter, provider ProfileProvider, reactions ReactionChecker, logger *zap.Logger) *GetCommentsHandler {
	return &GetCommentsHandler{
		getter:    getter,
		provider:  provider,
		reactions: reactions,
		logger:    logger,
	}
}

func (h *GetCommentsHandler) Handle(ctx context.Context, filter comment.CommentFilter) (CursorPage, error) {
	h.logger.Info("start commentUseCases.GetCommentsHandler.Handle")

	comments, hasMore, err := h.getter.GetComments(ctx, filter)
	if err != nil {
		return CursorPage{}, err
	}

	attachAuthors(ctx, h.provider, h.logger, comments)

	viewer := h.resolveViewer(ctx, filter.ViewerID, comments)
	return CursorPage{Items: toMultiCommentResult(comments, viewer), HasMore: hasMore}, nil
}

// resolveViewer собирает состояние читателя. Для гостя возвращает нулевое
// значение, не обращаясь к БД. Ошибка выборки лайков не роняет выдачу
// комментариев — флаги просто останутся false.
func (h *GetCommentsHandler) resolveViewer(ctx context.Context, viewerID *uint, comments []*comment.Comment) viewerState {
	if viewerID == nil || *viewerID == 0 {
		return viewerState{}
	}

	viewer := viewerState{authorized: true, liked: map[uint]bool{}}
	if h.reactions == nil || len(comments) == 0 {
		return viewer
	}

	ids := make([]uint, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.ID)
	}

	liked, err := h.reactions.LikedByUser(ctx, ids, *viewerID)
	if err != nil {
		h.logger.Error("LikedByUser failed", zap.Error(err), zap.Uint("viewer_id", *viewerID))
		return viewer
	}
	viewer.liked = liked
	return viewer
}
