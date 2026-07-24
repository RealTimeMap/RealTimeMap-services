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
	ChatID uint `gorm:"not null;index:idx_messages_chat_id_id,priority:1;uniqueIndex:idx_messages_client_dedup,priority:1"`

	SenderID uint        `gorm:"not null;index;uniqueIndex:idx_messages_client_dedup,priority:2"`
	Type     MessageType `gorm:"type:varchar(16);not null;default:'text'"`
	Content  string      `gorm:"type:text"`

	// ClientMessageID — идемпотентный ключ отправки от клиента (UUID). Уникален в
	// пределах (chat_id, sender_id): повторная отправка с тем же ключом ловится на
	// уровне БД. NULL (не *"") при отсутствии ключа — NULL в Postgres не нарушает
	// UNIQUE, поэтому сообщения без дедупа не конфликтуют между собой.
	ClientMessageID *string `gorm:"size:64;uniqueIndex:idx_messages_client_dedup,priority:3"`

	// ReplyToID — ответ на другое сообщение внутри этого же чата (опц.)
	ReplyToID *uint `gorm:"index"`

	// EditedAt проставляется при редактировании; отличается от UpdatedAt,
	// которое меняется при любом апдейте строки.
	EditedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // soft-delete: «сообщение удалено»
}
