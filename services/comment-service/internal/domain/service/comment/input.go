package comment

import "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"

type CreateInput struct {
	Content    string
	EntityType string
	EntityID   uint

	ParentID *uint
}

type UpdateInput struct {
	Content string
}

type ToggleReactionInput struct {
	Type model.ReactionType
}
