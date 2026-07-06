package category

type Category struct {
	ID           uint   `gorm:"primaryKey, autoIncrementIncrement"`
	CategoryName string `gorm:"unique,not null"`
	Color        string `gorm:"not null"`
	IsActive     bool   `gorm:"default:true"`
	Icon         string `gorm:"size:128"`
}

type CategoryStat struct {
	CategoryName string
	Count        int64
	Percent      float64
}
