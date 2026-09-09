package personal

import (
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Model struct {
	// Системная информация
	gorm.Model
	Revision uint        `gorm:"not null;index:idx_marks_owner_rev,priority:2"`
	UserID   uint        `gorm:"not null;index:idx_marks_owner_rev,priority:1"`
	Geom     types.Point `gorm:"type:geometry(POINT,4326);not null"`

	// Списки к которому присвоена метка
	Groups []*Group `gorm:"many2many:personal_marks_groups"`

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

func (m Model) GetID() uint       { return m.ID }
func (m Model) GetRevision() uint { return m.Revision }
func (m Model) IsDeleted() bool   { return m.DeletedAt.Valid }

type Revision struct {
	UserID   uint `gorm:"primaryKey;autoIncrement:false"`
	Revision uint `gorm:"not null;default:0"`
}

type Group struct {
	gorm.Model
	Name        string `gorm:"not null;size:255;uniqueIndex:idx_group_user_name,priority:2"`
	Description *string
	Revision    uint `gorm:"not null;index:idx_group_owner_rev,priority:2"`
	UserID      uint `gorm:"not null;index:idx_group_owner_rev,priority:1;uniqueIndex:idx_group_user_name,priority:1"`
}

func (Group) TableName() string {
	return "groups"
}

func (g Group) GetID() uint       { return g.ID }
func (g Group) GetRevision() uint { return g.Revision }
func (g Group) IsDeleted() bool   { return g.DeletedAt.Valid }

// Changes GENERIC для формирования ответов для синхронизации
type Changes[T any] struct {
	Upserted []T
	Removed  []uint
	Cursor   uint
	HasMore  bool
}
