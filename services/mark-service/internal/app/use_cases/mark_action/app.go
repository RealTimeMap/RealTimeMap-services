package mark_action

type Application struct {
	CreateMark  *CreateMarkHandler
	GetMark     *GetterMarkHandler
	GetDetail   *DetailMarkHandler
	DeleteMark  *RemoverMarkHandler
	GetUserMark *UserMarkGetterHandler
	UpdateMark  *UpdateMarkHandler
}
