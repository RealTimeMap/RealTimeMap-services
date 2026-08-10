package worker

import (
	"math/rand"
	"time"
)

// jitterFraction — доля задержки, на которую она случайно смещается.
//
// Без разброса вся очередь, накопившаяся за время недоступности провайдера,
// уходит одним залпом в момент восстановления — и провайдер снова отвечает
// отказом. Разброс размазывает повторные попытки по времени.
const jitterFraction = 0.2

// backoff возвращает задержку перед следующей попыткой.
//
// attempt — номер уже сделанной попытки (1 после первой). Если попыток больше,
// чем задано интервалов, используется последний: расписание задаёт нарастание,
// а не жёсткое число повторов — их ограничивает MaxAttempt.
func backoff(schedule []time.Duration, attempt uint) time.Duration {
	if len(schedule) == 0 {
		return time.Minute
	}

	idx := int(attempt) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(schedule) {
		idx = len(schedule) - 1
	}

	return withJitter(schedule[idx])
}

func withJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}

	spread := float64(d) * jitterFraction
	// Смещение в обе стороны: половина писем уйдёт раньше расчётного момента,
	// половина позже, вместо общего сдвига всей очереди вперёд.
	delta := (rand.Float64()*2 - 1) * spread

	result := time.Duration(float64(d) + delta)
	if result < 0 {
		return 0
	}

	return result
}
