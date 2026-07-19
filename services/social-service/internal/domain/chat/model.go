package chat

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"gorm.io/gorm"
)

type ChatType string
type ParticipantRole string

const (
	DirectType ChatType = "direct"
	GroupType  ChatType = "group"

	OwnerParticipantType  ParticipantRole = "owner"
	AdminParticipantType  ParticipantRole = "admin"
	MemberParticipantType ParticipantRole = "member"
)

type Chat struct {
	gorm.Model
	Type  ChatType `gorm:"type:varchar(16);not null;default:'direct';index"`
	Title *string  `gorm:"size:255"`

	// DirectKey заполняется только для direct-чатов в формате min(uid)_max(uid)
	// и обеспечивает дедупликацию на уровне БД + защиту от гонки. Для group — NULL.
	DirectKey *string `gorm:"size:64;uniqueIndex"`

	CreatedBy uint `gorm:"not null;index"`

	// LastMessageID — денормализация для сортировки списка чатов и превью
	// последнего сообщения. Без жёсткого FK, чтобы избежать цикла Chat<->Message.
	LastMessageID *uint `gorm:"index"`

	Participants []ChatParticipant `gorm:"foreignKey:ChatID"`
}

type ChatParticipant struct {
	ChatID uint `gorm:"primaryKey;autoincrement:false"`
	UserID uint `gorm:"primaryKey;autoincrement:false;index"`

	Role ParticipantRole `gorm:"type:varchar(16);not null;default:'member'"`

	LastReadMessageID *uint

	JoinedAt time.Time  `gorm:"not null;default:now()"`
	LeftAt   *time.Time `gorm:"index"`

	MutedUntil *time.Time

	Chat Chat `gorm:"foreignKey:ChatID"`
}

type ChatListItem struct {
	Chat        Chat
	LastMessage *message.Message
	UnreadCount int
}
