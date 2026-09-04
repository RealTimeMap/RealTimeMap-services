package comment

type SortOrder string

const (
	SortNewest SortOrder = "newest"
	SortOldest SortOrder = "oldest"
)

func (s SortOrder) OrderClause() string {
	if s == SortNewest {
		return "id DESC"
	}
	return "id ASC"
}

func (s SortOrder) CursorCondition() string {
	if s == SortOldest {
		return "id > ?"
	}
	return "id < ?"
}

type CommentFilter struct {
	Cursor   *uint
	Limit    int
	Entity   string
	EntityID uint
	Sort     SortOrder
	ParentID *uint

	// ViewerID — пользователь, от лица которого читается лента.
	// nil для гостя: в этом случае флаги isLiked/canLike всегда false.
	ViewerID *uint
}
