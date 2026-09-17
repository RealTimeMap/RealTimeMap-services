package comment

type UserProfile struct {
	ID       uint
	Username string
	Tag      string
	Avatar   string

	// IsAdmin — бейдж администратора у автора комментария. Ведётся
	// social-service как копия признака из auth; правами здесь не управляет.
	// При деградации profile-сервиса остаётся false: localFallback знает
	// только имя автора, сохранённое вместе с комментарием.
	IsAdmin bool
}
