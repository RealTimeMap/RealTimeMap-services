package mark

import "time"

type MarksCount struct {
	Count int64
}

type MonthlyActivity struct {
	Month string
	Count int64
}

type HeatMapItem struct {
	Day   time.Time
	Count int64
}
