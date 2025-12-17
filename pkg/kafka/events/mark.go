package events

const (
	MarkCreated = "markCreated"
	MarkUpdated = "markUpdated"
	MarkDeleted = "markDeleted"
)

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
