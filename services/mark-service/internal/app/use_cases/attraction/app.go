package attraction

type Application struct {
	Create     *CreateAttractionHandler
	ListByCity *ListByCityHandler
	Get        *GetAttractionHandler
	Update     *UpdateAttractionHandler
	Delete     *DeleteAttractionHandler
}
