package date

import (
	"time"
)

type Resolved struct {
	period Period
	now    time.Time
	loc    *time.Location

	currentStart *time.Time
	currentEnd   *time.Time
	prevStart    *time.Time
	prevEnd      *time.Time
}

func Resolve(period Period, now time.Time, loc *time.Location) (Resolved, error) {
	if loc == nil {
		loc = time.Local
	}

	r := Resolved{period: period, now: now, loc: loc}

	if period == AllTime {
		return r, nil
	}

	var curStart, curEnd, prevStart time.Time
	nowLocal := now.In(loc)

	switch period {
	case Week:
		curStart = startOfWeek(nowLocal, loc)
		curEnd = curStart.AddDate(0, 0, 7)
		prevStart = curStart.AddDate(0, 0, -7)
	case Month:
		curStart = startOfMonth(nowLocal, loc)
		curEnd = curStart.AddDate(0, 1, 0)
		prevStart = curStart.AddDate(0, -1, 0)
	case Year:
		curStart = startOfYear(nowLocal, loc)
		curEnd = curStart.AddDate(1, 0, 0)
		prevStart = curStart.AddDate(-1, 0, 0)
	default:
		return Resolved{}, ErrInvalidPeriod
	}

	r.currentStart = &curStart
	r.currentEnd = &curEnd
	r.prevStart = &prevStart
	prevEnd := curStart
	r.prevEnd = &prevEnd

	return r, nil
}

func (r Resolved) IsAllTime() bool { return r.period == AllTime }

func (r Resolved) Period() Period { return r.period }

// Current возвращает диапазон текущего периода [start, end).
// Для AllTime — (nil, nil): означает "без границ".
func (r Resolved) Current() (start, end *time.Time) {
	return r.currentStart, r.currentEnd
}

// Previous возвращает диапазон предыдущего периода той же длины.
// Для AllTime — (nil, nil).
func (r Resolved) Previous() (start, end *time.Time) {
	return r.prevStart, r.prevEnd
}
