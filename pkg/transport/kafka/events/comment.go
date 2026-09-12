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

// CommentPayload — состояние комментария на момент события.
//
// Content не заполняется для comment.deleted: текст удалённого комментария
// не должен лежать в топике по retention, а потребителям он там не нужен —
// геймификация считает события, уведомления берут текст из своей копии.
type CommentPayload struct {
	CommentID  uint   `json:"commentId"`
	UserID     uint   `json:"userId"`
	Username   string `json:"username,omitempty"`
	EntityType string `json:"entityType"`
	EntityID   uint   `json:"entityId"`
	ParentID   *uint  `json:"parentId,omitempty"`
	Content    string `json:"content,omitempty"`

	// ParentUserID — автор комментария, на который отвечают.
	//
	// Едет в событии, а не резолвится потребителем: comment-service знает его
	// в момент создания ответа, а у потребителей нет способа спросить —
	// сервис не поднимает gRPC, и ручки «комментарий по id» в proto нет.
	// Без этого поля уведомление об ответе построить не из чего.
	ParentUserID *uint `json:"parentUserId,omitempty"`
}

// NewCommentPayload собирает payload из полей комментария.
//
// parentUserID передаётся отдельно, а не берётся из parentID: это разные
// сущности — id родительского комментария и id его автора.
func NewCommentPayload(commentID, userID, entityID uint, entityType, username string, parentID, parentUserID *uint, content string) CommentPayload {
	return CommentPayload{
		CommentID:    commentID,
		UserID:       userID,
		Username:     username,
		EntityType:   entityType,
		EntityID:     entityID,
		ParentID:     parentID,
		ParentUserID: parentUserID,
		Content:      content,
	}
}

func NewCommentCreated(payload CommentPayload) CommentEvent {
	return CommentEvent{
		Envelop: NewEnvelop(CommentCreated),
		Payload: payload,
	}
}

func NewCommentUpdated(payload CommentPayload) CommentEvent {
	return CommentEvent{
		Envelop: NewEnvelop(CommentUpdated),
		Payload: payload,
	}
}

// NewCommentDeleted собирает событие удаления, вычищая текст комментария.
func NewCommentDeleted(payload CommentPayload) CommentEvent {
	payload.Content = ""
	return CommentEvent{
		Envelop: NewEnvelop(CommentDeleted),
		Payload: payload,
	}
}
