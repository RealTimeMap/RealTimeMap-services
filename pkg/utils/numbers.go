package utils

import "math"

func Threshold(val int64) float64 {
	if val > 1000 {
		return float64(val) / 1000.0
	}
	return float64(val)
}

func Percent(part, total int64) float64 {
	return math.Round(float64(part)/float64(total)*10000) / 100
}
