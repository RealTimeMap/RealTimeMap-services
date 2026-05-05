package date

import "errors"

var ErrInvalidPeriod = errors.New("invalid period (expected week|month|year|all)")
