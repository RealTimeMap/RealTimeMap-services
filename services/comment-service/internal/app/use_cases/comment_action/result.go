package comment_action

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
)

type AuthorResult struct {
	ID       uint
	Username string
	Tag      string
	Avatar   string
}

type MetaResult struct {
	CanReply     bool
	HaveReplies  bool
	RepliesCount int64
	Status       comment.CommentStatus

	// IsLiked — лайкнул ли текущий пользователь этот комментарий.
	// CanLike — может ли поставить лайк: авторизован и ещё не лайкал.
	// У гостя оба поля всегда false.
	IsLiked bool
	CanLike bool
}

type CommentResult struct {
	ID      uint
	Content string
	Author  AuthorResult
	Likes   uint
	Meta    MetaResult
}

type CursorPage struct {
	Items   []CommentResult
	HasMore bool
}

func toAuthorResult(p *comment.UserProfile) AuthorResult {
	if p == nil {
		return AuthorResult{}
	}
	return AuthorResult{
		ID:       p.ID,
		Username: p.Username,
		Tag:      p.Tag,
		Avatar:   p.Avatar,
	}
}

// viewerState описывает читателя ленты: авторизован ли он и что уже лайкнул.
// Нулевое значение — гость, для него isLiked/canLike остаются false.
type viewerState struct {
	authorized bool
	liked      map[uint]bool
}

func toMetaResult(c *comment.Comment, v viewerState) MetaResult {
	isLiked := v.liked[c.ID]
	return MetaResult{
		CanReply:     c.Depth <= comment.MaxDepth,
		HaveReplies:  c.RepliesCount > 0,
		RepliesCount: c.RepliesCount,
		Status:       c.Status,
		IsLiked:      isLiked,
		CanLike:      v.authorized && !isLiked,
	}
}

func toCommentResult(c *comment.Comment) CommentResult {
	return toCommentResultFor(c, viewerState{})
}

func toCommentResultFor(c *comment.Comment, v viewerState) CommentResult {
	return CommentResult{
		ID:      c.ID,
		Content: c.Content,
		Author:  toAuthorResult(c.Author),
		Likes:   c.LikesCount,
		Meta:    toMetaResult(c, v),
	}
}

func toMultiCommentResult(comments []*comment.Comment, v viewerState) []CommentResult {
	res := make([]CommentResult, 0, len(comments))
	for _, c := range comments {
		res = append(res, toCommentResultFor(c, v))
	}
	return res
}
