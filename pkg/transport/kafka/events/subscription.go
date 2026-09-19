package events

const (
	// SubscriptionCreated — пользователь подписался на чужой профиль.
	SubscriptionCreated = "subscription.created"
)

type SubscriptionEvent struct {
	Envelop
	Payload SubscriptionPayload `json:"payload"`
}

// SubscriptionPayload — состоявшаяся подписка.
//
// Отписка отдельным событием не публикуется: уведомлять о ней некого, а
// потребителям (уведомления, геймификация) факт отмены ничего не меняет
// задним числом.
type SubscriptionPayload struct {
	// SubscriberID — кто подписался, TargetID — на кого. Уведомление
	// адресуется TargetID: это он получает нового подписчика.
	SubscriberID uint `json:"subscriberId"`
	TargetID     uint `json:"targetId"`

	// SubscriberName — имя подписавшегося для текста уведомления.
	SubscriberName string `json:"subscriberName,omitempty"`
}

func NewSubscriptionCreated(payload SubscriptionPayload) SubscriptionEvent {
	return SubscriptionEvent{
		Envelop: NewEnvelop(SubscriptionCreated),
		Payload: payload,
	}
}
