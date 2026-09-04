package comment

import "gorm.io/gorm"

type EntityType string

const (
	EntityMark EntityType = "mark_action"
)

const (
	OwnerDeletedContent     string = "Content removed by owner"
	ModeratorDeletedContent string = "Content removed by moderator"
)

type CommentStatus string

const (
	CommentActive  CommentStatus = "active"
	CommentDeleted CommentStatus = "deleted"
)

const MaxDepth uint = 2 // Константа для обозначения максимальной глубины вложенности комментариев

type Comment struct {
	gorm.Model
	UserID   uint // ID Юзера
	Username string
	Content  string

	ParentID     *uint     `gorm:"index"`
	Parent       *Comment  `gorm:"foreignKey:ParentID"`
	Replies      []Comment `gorm:"foreignKey:ParentID"`
	RepliesCount int64     `gorm:"->" json:"-"`

	EntityType EntityType `gorm:"size:32;not null;index:idx_entity"`
	EntityID   uint       `gorm:"not null;index:idx_entity"`

	Status CommentStatus `gorm:"type:varchar(20);not null;default:'active'"`
	// LikesCount вычисляется подзапросом из таблицы reactions при выборке
	// (как и RepliesCount). Колонка в comments не поддерживалась в актуальном
	// состоянии при лайке, поэтому источником истины считаем сами реакции.
	LikesCount uint `gorm:"->"`

	Depth uint `gorm:"not null; default:0"`

	Author *UserProfile `gorm:"-" json:"-"`
}

func (c *Comment) IsDeleted() bool {
	return c.Status == CommentDeleted
}
