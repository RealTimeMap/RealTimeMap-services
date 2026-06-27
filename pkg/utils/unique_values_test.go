package utils

import (
	"reflect"
	"testing"
)

func TestUniqueValuesInt(t *testing.T) {
	tests := []struct {
		name string
		data []int
		want []int
	}{
		{"без дубликатов", []int{1, 2, 3}, []int{1, 2, 3}},
		{"дубликаты с сохранением порядка", []int{3, 1, 2, 1, 3}, []int{3, 1, 2}},
		{"все одинаковые", []int{5, 5, 5}, []int{5}},
		{"один элемент", []int{1}, []int{1}},
		{"подряд идущие дубли", []int{1, 1, 2, 2, 3, 3}, []int{1, 2, 3}},
		{"пустой слайс", []int{}, nil},
		{"nil на входе", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueValues(tt.data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueValuesStruct(t *testing.T) {
	type point struct{ X, Y int }
	tests := []struct {
		name string
		data []point
		want []point
	}{
		{"одинаковые структуры", []point{{1, 2}, {1, 2}, {3, 4}}, []point{{1, 2}, {3, 4}}},
		{"отличие в одном поле", []point{{1, 2}, {1, 3}}, []point{{1, 2}, {1, 3}}},
		{"все дубликаты", []point{{0, 0}, {0, 0}, {0, 0}}, []point{{0, 0}}},
		{"пустой слайс", []point{}, nil},
		{"один элемент", []point{{5, 5}}, []point{{5, 5}}},
		{"нет дубликатов", []point{{1, 1}, {2, 2}, {3, 3}, {4, 4}}, []point{{1, 1}, {2, 2}, {3, 3}, {4, 4}}},
		{"дубликаты в разном порядке", []point{{1, 2}, {3, 4}, {1, 2}, {5, 6}, {3, 4}}, []point{{1, 2}, {3, 4}, {5, 6}}},
		{"с отрицательными координатами", []point{{-1, -2}, {-1, -2}, {1, 2}, {-3, 4}}, []point{{-1, -2}, {1, 2}, {-3, 4}}},
		{"с нулевыми значениями", []point{{0, 0}, {0, 5}, {0, 0}, {5, 0}}, []point{{0, 0}, {0, 5}, {5, 0}}},
		{"много дубликатов с перемешиванием", []point{{1, 1}, {2, 2}, {1, 1}, {2, 2}, {3, 3}, {1, 1}}, []point{{1, 1}, {2, 2}, {3, 3}}},
		{"одинаковые X, разные Y", []point{{1, 0}, {1, 1}, {1, 0}, {1, 2}}, []point{{1, 0}, {1, 1}, {1, 2}}},
		{"одинаковые Y, разные X", []point{{0, 1}, {1, 1}, {0, 1}, {2, 1}}, []point{{0, 1}, {1, 1}, {2, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UniqueValues(tt.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueValuesFloat(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want []float64
	}{
		{"без дубликатов", []float64{1, 2, 3}, []float64{1, 2, 3}},
		{"с дубликатами", []float64{1, 1, 1}, []float64{1}},
		{"с дубликатами", []float64{1, 2, 2, 3, 3, 3}, []float64{1, 2, 3}},
		{"пустой слайс", []float64{}, nil},
		{"один элемент", []float64{42}, []float64{42}},
		{"все дубликаты", []float64{5, 5, 5, 5, 5}, []float64{5}},
		{"с отрицательными числами", []float64{-1, -2, -2, -3, -1}, []float64{-1, -2, -3}},
		{"с нулями", []float64{0, 0, 1, 0, 2}, []float64{0, 1, 2}},
		{"с дробными числами", []float64{1.1, 1.2, 1.1, 1.3, 1.2}, []float64{1.1, 1.2, 1.3}},
		{"неотсортированные данные", []float64{3, 1, 2, 1, 3, 2}, []float64{3, 1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueValues(tt.data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
