package group

type Application struct {
	Create *CreateGroupHandler
	Get    *GetGroupHandler
	List   *ListGroupHandler
	Update *UpdateGroupHandler
	Delete *DeleteGroupHandler
}
