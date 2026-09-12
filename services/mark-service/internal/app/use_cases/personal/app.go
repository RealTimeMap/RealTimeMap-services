package personal

type Application struct {
	Create *CreatePersonalHandler
	Get    *GetPersonalHandler
	Update *UpdatePersonalHandler
	Delete *DeletePersonalHandler
	Sync   *SyncMarkHandler
}
