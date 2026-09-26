package message

type Filter struct {
	ChatID uint

	// LastMessageID — курсор keyset-пагинации: выбираем сообщения с id < LastMessageID.
	// nil на первой странице (самые свежие).
	LastMessageID *uint

	// AfterMessageID — нижняя граница: выбираем сообщения с id > AfterMessageID.
	// Задаётся курсором очистки истории участника; nil — без ограничения.
	AfterMessageID *uint

	// Limit — размер страницы. Обязателен, иначе вернётся вся история чата.
	Limit int
}
