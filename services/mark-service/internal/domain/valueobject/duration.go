package valueobject

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
)

type Duration struct {
	hours int
}

func NewDuration(value int) (Duration, error) {
	for _, allowed := range model.AllowedDuration {
		if value == allowed {
			return Duration{hours: value}, nil
		}
	}
	return Duration{}, domainerrors.ErrInvalidDuration(value)

}

func (d Duration) Int() int {
	return d.hours
}
