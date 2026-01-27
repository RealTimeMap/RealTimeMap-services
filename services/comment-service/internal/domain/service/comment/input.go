package comment

type CreateInput struct {
	Content    string
	EntityType string
	EntityID   uint

	ParentID *uint
}
