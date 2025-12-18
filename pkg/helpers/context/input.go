package context

type UserInput struct {
	UserID   int
	UserName string
	IsAdmin  bool
}

func NewUserInput(userID int, userName string, isAdmin bool) UserInput {
	return UserInput{
		UserID:   userID,
		UserName: userName,
		IsAdmin:  isAdmin,
	}
}
