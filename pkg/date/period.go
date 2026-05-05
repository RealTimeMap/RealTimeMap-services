package date

type Period string

const (
	Week    Period = "week"
	Month   Period = "month"
	Year    Period = "year"
	AllTime Period = "allTime"
)

type Params struct {
	Period Period
}

func ParsePeriod(s string) (Period, error) {
	switch Period(s) {
	case Week, Month, Year, AllTime:
		return Period(s), nil
	default:
		return "", ErrInvalidPeriod
	}
}

func (p Period) IsAllTime() bool {
	return p == AllTime
}
