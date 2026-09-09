package personal

type Application struct {
	Create *CreatePersonalHandler
	Sync   *SyncMarkHandler
}
