package bug

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Tag string

const (
	TagFeature Tag = "feature"
	TagUI      Tag = "ui"
	TagLogic   Tag = "logic"
)

type Status string

// Жизненный цикл бага:
//
//	new ──confirm──▶ confirmed ──link──▶ in work ──sync──▶ closed / canceled
//	 │                  │
//	 └─────reject───────┴──▶ rejected ──reopen──▶ new
//
// Большая часть отчётов не подтверждается, поэтому между «пришёл отчёт»
// и «берём в работу» стоит проверка разработчиком: в задачу попадает
// только то, что он воспроизвёл.
const (
	// New — отчёт пришёл и ждёт проверки разработчиком.
	New Status = "new"
	// Confirmed — разработчик воспроизвёл баг; его можно брать в задачу.
	Confirmed Status = "confirmed"
	// Rejected — проверка не подтвердила баг. Причина — в RejectReason.
	Rejected Status = "rejected"
	InWork   Status = "in work"
	Closed   Status = "closed"
	Canceled Status = "canceled"
)

func (t Status) IsValid() bool {
	switch t {
	case New, Confirmed, Rejected, InWork, Closed, Canceled:
		return true
	}
	return false
}

// IsOpen сообщает, что баг подтверждён и работа над ним ещё идёт или
// может пойти — только такой баг можно взять в задачу. Непроверенные,
// отклонённые, закрытые и отменённые баги в перечень для привязки не
// попадают.
func (t Status) IsOpen() bool {
	return t == Confirmed || t == InWork
}

// OpenStatuses — статусы, при которых баг считается открытым.
// Используется фильтром перечня: одно место вместо перечисления
// в каждом запросе.
func OpenStatuses() []Status {
	return []Status{Confirmed, InWork}
}

// RejectReason — почему проверка не подтвердила баг. Коды фиксированы,
// чтобы по ним можно было считать, какие отчёты чаще всего пустые.
type RejectReason string

const (
	ReasonNotReproducible  RejectReason = "not_reproducible"
	ReasonNotABug          RejectReason = "not_a_bug"
	ReasonDuplicate        RejectReason = "duplicate"
	ReasonInsufficientInfo RejectReason = "insufficient_info"
	ReasonSpam             RejectReason = "spam"
)

func (r RejectReason) IsValid() bool {
	switch r {
	case ReasonNotReproducible, ReasonNotABug, ReasonDuplicate, ReasonInsufficientInfo, ReasonSpam:
		return true
	}
	return false
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

	// Итог проверки разработчиком. Пусто, пока баг не проверен или
	// возвращён на повторную проверку.
	Review Review `gorm:"embedded;embeddedPrefix:review_"`

	// FirstConfirmedAt — когда баг подтвердили впервые. В отличие от
	// Review, возврат на проверку его не стирает: по нему автор отчёта
	// получает награду ровно один раз, сколько бы раз решение ни меняли.
	FirstConfirmedAt *time.Time
}

// Review — решение разработчика по отчёту.
type Review struct {
	At *time.Time
	// RejectReason заполнен только у отклонённого бага.
	RejectReason RejectReason `gorm:"type:varchar(32)"`
	// Comment — пояснение разработчика: шаги, которыми баг воспроизвёлся,
	// или почему он отклонён.
	Comment string
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
