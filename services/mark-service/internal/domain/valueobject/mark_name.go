package valueobject

import "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"

type MarkName struct {
	value string
}

func NewMarkName(value string) (MarkName, error) {
	if value == "" {
		return MarkName{}, domainerrors.ErrMarkNameRequired()
	}
	if len(value) < 3 {
		return MarkName{}, domainerrors.ErrMarkNameTooShort(value)
	}
	if len(value) > 100 {
		return MarkName{}, domainerrors.ErrMarkNameTooLong(value)
	}
	return MarkName{value: value}, nil
}

func (m MarkName) String() string {
	return m.value
}
