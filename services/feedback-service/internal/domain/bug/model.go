package bug

import (
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type Tag string

const (
	TagFeature Tag = "feature"
	TagUI      Tag = "ui"
	TagLogic   Tag = "logic"
)

type Status string

const (
	New      Status = "new"
	InWork   Status = "in work"
	Closed   Status = "closed"
	Canceled Status = "canceled"
)

func (t Status) IsValid() bool {
	switch t {
	case New, InWork, Closed, Canceled:
		return true
	}
	return false
}

// IsOpen сообщает, что баг ещё в работе и его можно взять в задачу.
// Закрытые и отменённые баги в перечень для привязки не попадают:
// работать над ними уже не нужно.
func (t Status) IsOpen() bool {
	return t == New || t == InWork
}

// OpenStatuses — статусы, при которых баг считается открытым.
// Используется фильтром перечня: одно место вместо перечисления
// в каждом запросе.
func OpenStatuses() []Status {
	return []Status{New, InWork}
}

func (t Tag) IsValid() bool {
	switch t {
	case TagFeature, TagUI, TagLogic:
		return true
	}
	return false
}

type Model struct {
	gorm.Model
	UserID *uint
	IP     string
	Title  string
	Desc   string
	Tag    Tag        `gorm:"type:varchar(15);default:feature"`
	Status Status     `gorm:"type:varchar(15);default:new"`
	Device DeviceInfo `gorm:"embedded;embeddedPrefix:device_"`
	App    AppInfo    `gorm:"embedded;embeddedPrefix:app_"`

	// Привязка к задаче в таск-менеджере. Заполняется, когда баг берут
	// в работу: по нему видно, что багом уже занимаются, и через него
	// идёт обратная синхронизация статуса.
	//
	// Идентификатор внешний — таблицы задач в этой базе нет, поэтому
	// внешнего ключа тоже нет, только индекс для поиска привязки.
	TaskID *uint `gorm:"index"`
}

// IsLinked сообщает, что баг уже привязан к задаче.
func (m Model) IsLinked() bool { return m.TaskID != nil }

// IsLinkedTo сообщает, что баг привязан именно к этой задаче.
func (m Model) IsLinkedTo(taskID uint) bool {
	return m.TaskID != nil && *m.TaskID == taskID
}

func (m Model) TableName() string {
	return "bugs"
}

type DeviceInfo struct {
	Platform   string
	OS         string // OS + OS Version
	Resolution string
	Battery    *float64
}

func (d DeviceInfo) Width() int {
	parts := strings.Split(d.Resolution, "x")
	if len(parts) < 2 {
		return 0
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return width
}

func (d DeviceInfo) Height() int {
	parts := strings.Split(d.Resolution, "x")
	if len(parts) < 2 {
		return 0
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return height
}

type AppInfo struct {
	Build string
	Logs  []string `gorm:"type:text;serializer:json"`
}
