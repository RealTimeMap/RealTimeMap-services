package notify

type Application struct {
	NotifyUser *UserNotifyHanlder

	// NotifyEvent — вход для доменных событий из Kafka: та же отправка, но
	// через схлопывание серий.
	NotifyEvent *EventNotifyHandler
}
