package personal

import (
	"strconv"
	"time"

	"github.com/google/uuid"
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

	// Информация для UI
	Color string
	Icon  string
}

func (Model) TableName() string {
	return "personal_marks"
}

func (m Model) GetID() uint       { return m.ID }
func (m Model) GetSyncID() string { return strconv.FormatUint(uint64(m.ID), 10) }
func (m Model) GetRevision() uint { return m.Revision }
func (m Model) IsDeleted() bool   { return m.DeletedAt.Valid }

type Revision struct {
	UserID   uint `gorm:"primaryKey;autoIncrement:false"`
	Revision uint `gorm:"not null;default:0"`
}

// Group — список персональных меток.
//
// Идентификатор — UUID, а не автоинкремент: группу заводят офлайн на клиенте,
// и её id должен существовать до того, как о ней узнает сервер. Иначе метки,
// созданные офлайн, нечем связать с группой до первой удачной синхронизации.
type Group struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Name        string `gorm:"not null;size:255;uniqueIndex:idx_group_user_name,priority:2"`
	Description *string
	Revision    uint `gorm:"not null;index:idx_group_owner_rev,priority:2"`
	UserID      uint `gorm:"not null;index:idx_group_owner_rev,priority:1;uniqueIndex:idx_group_user_name,priority:1"`
	Color       string
	Icon        string
}

func (Group) TableName() string {
	return "groups"
}

func (g Group) GetID() uuid.UUID  { return g.ID }
func (g Group) GetSyncID() string { return g.ID.String() }
func (g Group) GetRevision() uint { return g.Revision }
func (g Group) IsDeleted() bool   { return g.DeletedAt.Valid }

// Changes — окно изменений для синхронизации.
//
// Removed — строки, потому что секции синхронизации разнотипны по ключу:
// у меток он uint, у групп — UUID. Числовой id метки в JSON выглядит как
// "12", и клиент разбирает его тем же способом, что и раньше.
type Changes[T any] struct {
	Upserted []T      `json:"upserted"`
	Removed  []string `json:"removed"`
	Cursor   uint     `json:"cursor"`
	HasMore  bool     `json:"hasMore"`
}
