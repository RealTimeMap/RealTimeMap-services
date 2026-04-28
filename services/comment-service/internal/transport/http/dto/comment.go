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

type ReactionRequest struct {
	Type string `json:"type" binding:"required,oneof=like dislike"`
}

type ReactionResponse struct {
	UserReaction  *string `json:"userReaction"`
	LikesCount    uint    `json:"likesCount"`
	DislikesCount uint    `json:"dislikesCount"`
}

func NewReactionResponse(result *model.ToggleResult) ReactionResponse {
	var userReaction *string
	if result.Reaction != nil {
		t := string(result.Reaction.Type)
		userReaction = &t
	}
	return ReactionResponse{
		UserReaction:  userReaction,
		LikesCount:    result.LikesCount,
		DislikesCount: result.DislikesCount,
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

type AuthorResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Tag      string `json:"tag"`
	Avatar   string `json:"avatar"`
}

func NewAuthorResponse(p *model.UserProfile) AuthorResponse {
	if p == nil {
		return AuthorResponse{}
	}
	return AuthorResponse{
		ID:       p.ID,
		Username: p.Username,
		Tag:      p.Tag,
		Avatar:   p.Avatar,
	}
}

type CommentResponse struct {
	ID       uint           `json:"id"`
	Content  string         `json:"content"`
	Author   AuthorResponse `json:"author"`
	Likes    uint           `json:"likes"`
	Dislikes uint           `json:"dislikes"`
	Meta     Meta           `json:"meta"`
}

func NewCommentResponse(comment *model.Comment) CommentResponse {
	return CommentResponse{
		ID:       comment.ID,
		Content:  comment.Content,
		Author:   NewAuthorResponse(comment.Author),
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
