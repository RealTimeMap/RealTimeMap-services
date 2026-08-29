package comment_interaction

type Application struct {
	LikeComment   *LikeCommentHandler
	UnlikeComment *UnlikeCommentHandler
}
