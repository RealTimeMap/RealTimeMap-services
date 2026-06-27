package utils

import "testing"

func TestThreshold(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		want float64
	}{
		{"меньше порога", 500, 500.0},
		{"ноль", 0, 0.0},
		{"отрицательное", -200, -200.0},
		{"ровно 1000 (граница, не > 1000)", 1000, 1000.0},
		{"чуть больше порога", 1001, 1.001},
		{"большое значение", 5000, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Threshold(tt.val)
			if got != tt.want {
				t.Errorf("Threshold(%d) = %v; want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		name        string
		part, total int64
		want        float64
	}{
		{"половина", 1, 2, 50.0},
		{"треть округляется вниз", 1, 3, 33.33},
		{"две трети округляются вверх", 2, 3, 66.67},
		{"ровно 100%", 50, 50, 100.0},
		{"ноль в числителе", 0, 100, 0.0},
		{"part больше total", 3, 2, 150.0},
		{"одна восьмая", 1, 8, 12.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Percent(tt.part, tt.total)
			if got != tt.want {
				t.Errorf("Percent(%d, %d) = %v; want %v", tt.part, tt.total, got, tt.want)
			}
		})
	}
}
