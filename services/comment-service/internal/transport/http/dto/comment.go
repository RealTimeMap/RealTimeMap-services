package dto

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
)

type CommentParams struct {
	Entity string `form:"entity" binding:"required"`
	Sort   string `form:"sort"`
	Limit  int    `form:"limit"`
	Cursor *uint  `form:"cursor"`
}

func (p CommentParams) ToFilter(entityID uint, parentID *uint) model.CommentFilter {
	limit := p.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	sort := model.SortNewest
	if p.Sort == "oldest" {
		sort = model.SortOldest
	}

	return model.CommentFilter{
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

type Meta struct {
	CanReply     bool  `json:"canReply"`
	HaveReplies  bool  `json:"haveReplies"`
	RepliesCount int64 `json:"repliesCount"`
}

func NewMeta(c *model.Comment) Meta {

	return Meta{
		CanReply:     c.Depth <= model.MaxDepth,
		HaveReplies:  c.RepliesCount > 0,
		RepliesCount: c.RepliesCount,
	}

}

type CommentResponse struct {
	ID       uint   `json:"id"`
	Content  string `json:"content"`
	Likes    uint   `json:"likes"`
	Dislikes uint   `json:"dislikes"`
	Meta     Meta   `json:"meta"`
}

func NewCommentResponse(comment *model.Comment) CommentResponse {
	return CommentResponse{
		ID:       comment.ID,
		Content:  comment.Content,
		Likes:    comment.LikesCount,
		Dislikes: comment.DislikesCount,
		Meta:     NewMeta(comment),
	}
}

func NewMultipCommentResponse(comments []*model.Comment) []CommentResponse {
	res := make([]CommentResponse, 0, len(comments))
	for _, comment := range comments {
		res = append(res, NewCommentResponse(comment))
	}
	return res
}

type CursorPaginateResponse struct {
	Items   []CommentResponse `json:"items"`
	HasMore bool              `json:"hasMore"`
}

func NewCursorPaginateResponse(items []*model.Comment, hasMore bool) CursorPaginateResponse {
	res := NewMultipCommentResponse(items)
	return CursorPaginateResponse{
		Items:   res,
		HasMore: hasMore,
	}
}
