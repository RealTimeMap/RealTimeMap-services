package worker

import (
	"testing"
	"time"
)

func TestBackoffFollowsSchedule(t *testing.T) {
	schedule := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}

	for attempt, want := range map[uint]time.Duration{
		1: time.Minute,
		2: 5 * time.Minute,
		3: 15 * time.Minute,
	} {
		got := backoff(schedule, attempt)

		// Точное совпадение невозможно: задержка намеренно размывается.
		low := time.Duration(float64(want) * (1 - jitterFraction - 0.01))
		high := time.Duration(float64(want) * (1 + jitterFraction + 0.01))

		if got < low || got > high {
			t.Errorf("attempt %d: got %v, want ~%v", attempt, got, want)
		}
	}
}

// Расписание задаёт нарастание задержки, а число повторов ограничивает
// MaxAttempt — поэтому попытки сверх расписания используют последний интервал,
// а не обнуляют паузу.
func TestBackoffClampsBeyondSchedule(t *testing.T) {
	schedule := []time.Duration{time.Minute, time.Hour}

	got := backoff(schedule, 99)
	if got < 30*time.Minute {
		t.Errorf("got %v for attempt beyond schedule, want ~1h", got)
	}
}

func TestBackoffHandlesEdgeCases(t *testing.T) {
	if got := backoff(nil, 1); got <= 0 {
		t.Errorf("empty schedule produced %v, want a positive fallback", got)
	}
	if got := backoff([]time.Duration{time.Minute}, 0); got <= 0 {
		t.Errorf("zero attempt produced %v, want a positive delay", got)
	}
}

// Без разброса вся очередь, накопившаяся за время недоступности провайдера,
// уходит одним залпом при восстановлении — и провайдер отвечает отказом снова.
func TestBackoffJitterSpreadsRetries(t *testing.T) {
	schedule := []time.Duration{time.Hour}

	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		seen[backoff(schedule, 1)] = true
	}

	if len(seen) < 10 {
		t.Errorf("only %d distinct delays out of 50 — retries are not spread", len(seen))
	}
}
