package progress

type UserExpProgress struct {
	UserID          uint
	CurrentLevel    uint64
	CurrentXP       uint64
	XPForNextLevel  uint64
	ProgressPercent float64
}
