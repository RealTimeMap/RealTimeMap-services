package events

const (
	CommentCreated = "comment.created"
	CommentUpdated = "comment.updated"
	CommentDeleted = "comment.deleted"
)

type CommentEvent struct {
	Envelop
	Payload CommentPayload `json:"payload"`
}

type CommentPayload struct {
	CommentID  uint   `json:"commentId"`
	UserID     uint   `json:"userId"`
	EntityType string `json:"entityType"`
	EntityID   uint   `json:"entityId"`
	ParentID   *uint  `json:"parentId,omitempty"`
	Content    string `json:"content"`
}

func NewCommentPayload(commentID, userID, entityID uint, entityType string, parentID *uint, content string) CommentPayload {
	return CommentPayload{
		CommentID:  commentID,
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		ParentID:   parentID,
		Content:    content,
	}
}

func NewCommentCreated(payload CommentPayload) CommentEvent {
	return CommentEvent{
		Envelop: NewEnvelop(CommentCreated),
		Payload: payload,
	}
}
