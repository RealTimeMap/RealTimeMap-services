package dto

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_interaction"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
)

// ---- Requests ----

type CommentParams struct {
	Entity string `form:"entity" binding:"required"`
	Sort   string `form:"sort"`
	Limit  int    `form:"limit"`
	Cursor *uint  `form:"cursor"`
}

func (p CommentParams) ToFilter(entityID uint, parentID *uint) comment.CommentFilter {
	limit := p.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	sort := comment.SortNewest
	if p.Sort == "oldest" {
		sort = comment.SortOldest
	}

	return comment.CommentFilter{
		Limit:    limit,
		Sort:     sort,
		Cursor:   p.Cursor,
		Entity:   p.Entity,
		EntityID: entityID,
		ParentID: parentID,
	}
}

type CommentRequest struct {
	Content  string `form:"content" json:"content" binding:"required"`
	EntityID uint   `form:"entityId" json:"entityId" binding:"required"`
	Entity   string `form:"entity" json:"entity" binding:"required"`
	ParentID *uint  `form:"parentId" json:"parentId"`
}

type CommentUpdateRequest struct {
	Content string `form:"content" json:"content" binding:"required"`
}

// ---- Responses (мапятся из Result use case) ----

type Meta struct {
	CanReply     bool                  `json:"canReply"`
	HaveReplies  bool                  `json:"haveReplies"`
	RepliesCount int64                 `json:"repliesCount"`
	Status       comment.CommentStatus `json:"status"`
}

type AuthorResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Tag      string `json:"tag"`
	Avatar   string `json:"avatar"`
}

type CommentResponse struct {
	ID      uint           `json:"id"`
	Content string         `json:"content"`
	Author  AuthorResponse `json:"author"`
	Likes   uint           `json:"likes"`
	Meta    Meta           `json:"meta"`
}

func NewCommentResponse(res comment_action.CommentResult) CommentResponse {
	return CommentResponse{
		ID:      res.ID,
		Content: res.Content,
		Author: AuthorResponse{
			ID:       res.Author.ID,
			Username: res.Author.Username,
			Tag:      res.Author.Tag,
			Avatar:   res.Author.Avatar,
		},
		Likes: res.Likes,
		Meta: Meta{
			CanReply:     res.Meta.CanReply,
			HaveReplies:  res.Meta.HaveReplies,
			RepliesCount: res.Meta.RepliesCount,
			Status:       res.Meta.Status,
		},
	}
}

type CursorPaginateResponse struct {
	Items   []CommentResponse `json:"items"`
	HasMore bool              `json:"hasMore"`
}

func NewCursorPaginateResponse(page comment_action.CursorPage) CursorPaginateResponse {
	items := make([]CommentResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, NewCommentResponse(item))
	}
	return CursorPaginateResponse{
		Items:   items,
		HasMore: page.HasMore,
	}
}

// ---- Reaction ----

type ReactionResponse struct {
	Liked      bool  `json:"liked"`
	LikesCount int64 `json:"likesCount"`
}

func NewReactionResponse(res comment_interaction.ReactionResult) ReactionResponse {
	return ReactionResponse{
		Liked:      res.Liked,
		LikesCount: res.LikesCount,
	}
}
