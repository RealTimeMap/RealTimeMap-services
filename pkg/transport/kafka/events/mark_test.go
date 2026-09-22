package events

import "testing"

// TestMarkTimeOfDayEvent фиксирует границы окон: они полуинтервальные
// [From, To), поэтому 5 часов — уже утро, а не ночь, и в 8 никакого события
// не публикуется. Ошибка на границе дала бы пользователю не то достижение,
// и заметить это по логам было бы почти нельзя.
func TestMarkTimeOfDayEvent(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, MarkCreatedAtNight},
		{3, MarkCreatedAtNight},
		{4, MarkCreatedAtNight},
		{5, MarkCreatedAtEarlyMorning},
		{7, MarkCreatedAtEarlyMorning},
		{8, ""},
		{12, ""},
		{23, ""},
	}

	for _, tt := range tests {
		if got := MarkTimeOfDayEvent(tt.hour); got != tt.want {
			t.Errorf("MarkTimeOfDayEvent(%d) = %q, want %q", tt.hour, got, tt.want)
		}
	}
}

// TestMarkTimeOfDayWindowsDoNotOverlap проверяет, что каждый час суток даёт
// не больше одного события: пересекись окна, метка открывала бы оба
// достижения разом.
func TestMarkTimeOfDayWindowsDoNotOverlap(t *testing.T) {
	night, morning := 0, 0

	for hour := 0; hour < 24; hour++ {
		switch MarkTimeOfDayEvent(hour) {
		case MarkCreatedAtNight:
			night++
		case MarkCreatedAtEarlyMorning:
			morning++
		}
	}

	if night != NightHourTo-NightHourFrom {
		t.Errorf("ночное окно покрывает %d часов, want %d", night, NightHourTo-NightHourFrom)
	}
	if morning != EarlyMorningHourTo-EarlyMorningHourFrom {
		t.Errorf("утреннее окно покрывает %d часов, want %d", morning, EarlyMorningHourTo-EarlyMorningHourFrom)
	}
}
