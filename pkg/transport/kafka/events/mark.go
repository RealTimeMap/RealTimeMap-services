package events

const (
	MarkCreated = "markCreated"
	MarkUpdated = "markUpdated"
	MarkDeleted = "markDeleted"

	// MarkCreatedAtNight и MarkCreatedAtEarlyMorning — метка, созданная в
	// ночные и ранние утренние часы.
	// Публикуются дополнительно к MarkCreated, не вместо него: опыт за
	// создание метки начисляется по нему и от часа не зависит.
	MarkCreatedAtNight        = "markCreatedAtNight"
	MarkCreatedAtEarlyMorning = "markCreatedAtEarlyMorning"
)

// Границы ночного и раннего утреннего окна, в часах [From, To).
const (
	NightHourFrom = 0
	NightHourTo   = 5

	EarlyMorningHourFrom = 5
	EarlyMorningHourTo   = 8
)

// MarkTimeOfDayEvent возвращает тип события для часа создания метки.
func MarkTimeOfDayEvent(hour int) string {
	switch {
	case hour >= NightHourFrom && hour < NightHourTo:
		return MarkCreatedAtNight
	case hour >= EarlyMorningHourFrom && hour < EarlyMorningHourTo:
		return MarkCreatedAtEarlyMorning
	default:
		return ""
	}
}

type MarkEvent struct {
	Envelop
	Payload MarkPayload `json:"payload"`
}

type MarkPayload struct {
	MarkID         int     `json:"id"`
	CategoryID     int     `json:"categoryId"`
	OwnerID        int     `json:"ownerId"`
	MarkName       string  `json:"markName"`
	AdditionalInfo *string `json:"additionalInfo"`
	IsEnded        bool    `json:"isEnded"`
}

func NewMarkPayload(markID int, categoryID int, ownerID int, markName string, additionalInfo *string) MarkPayload {
	return MarkPayload{
		MarkID:         markID,
		CategoryID:     categoryID,
		OwnerID:        ownerID,
		MarkName:       markName,
		AdditionalInfo: additionalInfo,
		IsEnded:        false,
	}
}

func NewMarkCreate(payload MarkPayload) MarkEvent {
	return MarkEvent{
		Envelop: NewEnvelop(MarkCreated),
		Payload: payload,
	}
}

// NewMarkTimeOfDayEvent собирает событие о метке, созданной в отмеченный час.
// eventType берётся из MarkTimeOfDayEvent.
func NewMarkTimeOfDayEvent(eventType string, payload MarkPayload) MarkEvent {
	return MarkEvent{
		Envelop: NewEnvelop(eventType),
		Payload: payload,
	}
}
