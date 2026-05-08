package utils

func Threshold(val int64) float64 {
	if val > 1000 {
		return float64(val) / 1000.0
	}
	return float64(val)
}
