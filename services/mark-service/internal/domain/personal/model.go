package personal

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
	"gorm.io/gorm"
)

type Model struct {
	// Системная информация
	gorm.Model
	UserID uint
	Geom   types.Point `gorm:"type:geometry(POINT,4326);not null"`

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
