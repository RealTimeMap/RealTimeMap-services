package date

import "time"

type Query struct {
	Period string `form:"period" binding:"required,oneof=week month year allTime"`
}

func (q Query) Resolve(now time.Time, loc *time.Location) (Resolved, error) {
	period, err := ParsePeriod(q.Period)
	if err != nil {
		return Resolved{}, err
	}
	return Resolve(period, now, loc)
}
