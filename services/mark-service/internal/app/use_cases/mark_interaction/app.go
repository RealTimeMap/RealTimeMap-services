package mark_interaction

type Application struct {
	CreateShare *ShareCreatorHandler
	LikeMark    *LikeMarkHandler
	UnlikeMark  *UnlikeMarkHandler
	GetStat     *GetStatHandler
}
