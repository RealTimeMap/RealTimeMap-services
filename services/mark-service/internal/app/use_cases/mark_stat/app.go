package mark_stat

type Application struct {
	GetCategories    *MarkStatCategoryHandler
	GetMonthActivity *MarkMonthHandler
	GetHeatMap       *MarkHeatMapHandler
	GetMarkCount     *MarkCountHandler
}
