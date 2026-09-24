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

	// Review — проверка отчёта разработчиком: подтверждение, отклонение
	// и возврат на повторную проверку. В задачу берут только
	// подтверждённые баги.
	Review *ReviewBugHandler
}
