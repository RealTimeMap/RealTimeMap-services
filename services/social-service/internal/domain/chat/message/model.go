package message

import (
	"time"

	"gorm.io/gorm"
)

type MessageType string

const (
	TextType   MessageType = "text"
	ImageType  MessageType = "image"
	SystemType MessageType = "system"
)

type Message struct {
	ID     uint `gorm:"primaryKey;index:idx_messages_chat_id_id,priority:2"`
	ChatID uint `gorm:"not null;index:idx_messages_chat_id_id,priority:1"`

	SenderID uint        `gorm:"not null;index"`
	Type     MessageType `gorm:"type:varchar(16);not null;default:'text'"`
	Content  string      `gorm:"type:text"`

	// ReplyToID — ответ на другое сообщение внутри этого же чата (опц.)
	ReplyToID *uint `gorm:"index"`

	// EditedAt проставляется при редактировании; отличается от UpdatedAt,
	// которое меняется при любом апдейте строки.
	EditedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // soft-delete: «сообщение удалено»
}
