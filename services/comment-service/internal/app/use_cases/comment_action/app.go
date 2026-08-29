package comment_action

type Application struct {
	CreateComment *CreateCommentHandler
	GetComments   *GetCommentsHandler
	UpdateComment *UpdateCommentHandler
	DeleteComment *DeleteCommentHandler
}
