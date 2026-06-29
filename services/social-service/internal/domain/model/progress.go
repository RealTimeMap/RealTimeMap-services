package model

type Progress struct {
	CurrentLevel     uint64
	CurrentLevelName string
	CurrentXP        uint64
	XPForNextLevel   uint64
	ProgressPercent  float64
	NextLevel        NextLevel
}

type NextLevel struct {
	Level     uint64
	LevelName string
}
