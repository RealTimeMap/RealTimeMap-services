package comment_interaction

// ReactionResult — состояние лайка комментария для клиента.
type ReactionResult struct {
	Liked      bool
	LikesCount int64
}

func toReactionResult(likesCount int64, liked bool) ReactionResult {
	return ReactionResult{
		Liked:      liked,
		LikesCount: likesCount,
	}
}
