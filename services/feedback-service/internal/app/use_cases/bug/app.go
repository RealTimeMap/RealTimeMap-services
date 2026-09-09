package bug

type Application struct {
	Create *CreatorBugHandler
	List   *ListBugHandler

	// Get отдаёт один баг со всеми подробностями — их показывает
	// карточка задачи, в которой этот баг ведут.
	Get *GetBugHandler

	// Link обслуживает связь бага с задачей таск-менеджера:
	// привязку, отвязку и перенос статуса задачи на баг.
	Link *LinkBugHandler
}
