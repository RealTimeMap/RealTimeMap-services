package model

type Progress struct {
	CurrentLevel    uint64
	CurrentXP       uint64
	XPForNextLevel  uint64
	ProgressPercent float64
}
