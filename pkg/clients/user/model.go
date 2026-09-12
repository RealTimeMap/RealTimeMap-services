package user

// User — то, что UserService отдаёт о пользователе.
//
// Email здесь единственная причина существования клиента: сервисы, которым
// нужно написать пользователю, не должны получать адрес из Kafka-события —
// в топике он живёт по retention, а это персональные данные.
type User struct {
	ID          int64
	Username    string
	Email       string
	IsSuperuser bool
}
