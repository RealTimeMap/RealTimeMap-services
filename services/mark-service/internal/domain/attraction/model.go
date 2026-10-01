package attraction

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	}
	return false
}

type BlockType string

const (
	BlockTypeAttribute BlockType = "attribute"
	BlockTypeStatic    BlockType = "static"
)

func (t BlockType) Valid() bool {
	switch t {
	case BlockTypeAttribute, BlockTypeStatic:
		return true
	}
	return false
}

type Model struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Slug     string      `gorm:"not null;size:255;uniqueIndex:idx_attractions_slug,where:deleted_at IS NULL"`
	Name     string      `gorm:"not null;size:255"`
	Geom     types.Point `gorm:"type:geometry(POINT,4326);not null;index:idx_attractions_geom,type:gist"`
	City     string      `gorm:"not null;size:128;index:idx_attractions_city_status,priority:1,where:deleted_at IS NULL"`
	Category string      `gorm:"not null;size:128"`
	Status   Status      `gorm:"not null;size:16;default:'draft';index:idx_attractions_city_status,priority:2;check:chk_attractions_status,status IN ('draft','published','archived')"`

	Slides []Slide `gorm:"foreignKey:AttractionID;constraint:OnDelete:CASCADE"`
}

func (Model) TableName() string {
	return "attractions"
}

func (m *Model) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type Slide struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	AttractionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_attraction_slides_position,priority:1"`
	Position     uint      `gorm:"not null;uniqueIndex:idx_attraction_slides_position,priority:2"`
	DurationMs   uint      `gorm:"not null;check:chk_attraction_slides_duration,duration_ms > 0"`

	Blocks []SlideBlock `gorm:"foreignKey:SlideID;constraint:OnDelete:CASCADE"`
}

func (Slide) TableName() string {
	return "attraction_slides"
}

func (s *Slide) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type SlideBlock struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	SlideID   uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_attraction_slide_blocks_position,priority:1"`
	Position  uint           `gorm:"not null;uniqueIndex:idx_attraction_slide_blocks_position,priority:2"`
	BlockType BlockType      `gorm:"not null;size:16;check:chk_attraction_slide_blocks_type,block_type IN ('attribute','static')"`
	Content   map[string]any `gorm:"not null;serializer:json;type:jsonb"`
}

func (SlideBlock) TableName() string {
	return "attraction_slide_blocks"
}

func (b *SlideBlock) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
