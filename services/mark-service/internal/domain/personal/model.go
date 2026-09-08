package personal

import (
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
)

type Model struct {
	// Системная информация
	gorm.Model
	Revision uint        `gorm:"not null;index:idx_marks_owner_rev,priority:2"`
	UserID   uint        `gorm:"not null;index:idx_marks_owner_rev,priority:1"`
	Geom     types.Point `gorm:"type:geometry(POINT,4326);not null"`

	// Списки к которому присвоена метка
	Groups []*group.Model `gorm:"many2many:personal_marks_groups"`

	// Флаг для метки который показывает возможность делиться ею
	IsShare   bool `gorm:"default:false"`
	IsVisible bool `gorm:"default:true"`

	// Информация о самой метке
	Title       string
	Description *string
	Photos      types.Photos `gorm:"type:jsonb"`
	Category    string

	// Информация для UI
	Color string
	Icon  string
}

func (Model) TableName() string {
	return "personal_marks"
}

type Revision struct {
	UserID   uint `gorm:"primaryKey;autoIncrement:false"`
	Revision uint `gorm:"not null;default:0"`
}

type Changes struct {
	Marks   []Model
	Removed []uint
	Cursor  uint
	HasMore bool
}
