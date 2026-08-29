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

func toMetaResult(c *comment.Comment) MetaResult {
	return MetaResult{
		CanReply:     c.Depth <= comment.MaxDepth,
		HaveReplies:  c.RepliesCount > 0,
		RepliesCount: c.RepliesCount,
		Status:       c.Status,
	}
}

func toCommentResult(c *comment.Comment) CommentResult {
	return CommentResult{
		ID:      c.ID,
		Content: c.Content,
		Author:  toAuthorResult(c.Author),
		Likes:   c.LikesCount,
		Meta:    toMetaResult(c),
	}
}

func toMultiCommentResult(comments []*comment.Comment) []CommentResult {
	res := make([]CommentResult, 0, len(comments))
	for _, c := range comments {
		res = append(res, toCommentResult(c))
	}
	return res
}
