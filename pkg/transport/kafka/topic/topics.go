package topic

// Топики именуются по владельцу-издателю: в топик пишет ровно один сервис,
// читают сколько угодно. Потребителю не нужно знать, какие ещё события туда
// приедут позже — он разбирает знакомые типы и игнорирует прочие.
const (
	// UserEvents — события auth-сервиса (Python): регистрация, изменение
	// профиля. Историческое имя без суффикса .events — менять его значит
	// одновременно переключить продюсер и всех потребителей.
	UserEvents = "user-service"

	// CommentEvents — события comment-service.
	CommentEvents = "comment-service.events"

	// MarkEvents — события mark-service.
	MarkEvents = "mark_action-service.events"
)
