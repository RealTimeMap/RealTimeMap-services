package utils

import (
	"fmt"
	"math"
)

func Threshold(val int64) float64 {
	if val > 1000 {
		return float64(val) / 1000.0
	}
	return float64(val)
}

func Percent(part, total int64) float64 {
	return math.Round(float64(part)/float64(total)*10000) / 100
}

func FormatNumber(val int64) string {
	// Для отрицательных чисел
	if val < 0 {
		return "-" + FormatNumber(-val)
	}

	switch {
	case val >= 1_000_000:
		return fmt.Sprintf("%.1f M", float64(val)/1_000_000.0)

	case val >= 1000:
		return fmt.Sprintf("%.1f G", float64(val)/1000.0)
	default:
		return fmt.Sprintf("%d", val)
	}
}
