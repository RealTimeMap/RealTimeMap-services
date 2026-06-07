package utils

import "testing"

func TestSliceString(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		limit int
		want  string
	}{
		{"limit меньше длины", "hello", 3, "hel"},
		{"limit равен длине", "hello", 5, "hello"},
		{"limit больше длины", "hello", 10, "hello"},
		{"limit ноль", "hello", 0, ""},
		{"пустая строка", "", 5, ""},
		{"пустая строка и limit ноль", "", 0, ""},
		{"кириллица режется по рунам", "привет", 3, "при"},
		{"кириллица limit равен длине", "привет", 6, "привет"},
		{"эмодзи режется по рунам", "👍🔥😀🎉", 2, "👍🔥"},
		{"символы с диакритикой", "café", 3, "caf"},
		{"limit ровно на границе", "ab", 2, "ab"},
		{"один символ из многих", "test", 1, "t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SliceString(tt.src, tt.limit)
			if got != tt.want {
				t.Errorf("SliceString() = %v, want %v", got, tt.want)
			}
		})
	}
}
