package chatsocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Парсинг chatId — единственное место, где socket-слой доверяет данным клиента,
// поэтому проверяем и валидные формы (JSON отдаёт числа как float64), и мусор.
func TestParseChatID(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want uint
		ok   bool
	}{
		{
			name: "объект с числовым chatId — обычный путь socket.io",
			args: []any{map[string]any{"chatId": float64(42)}},
			want: 42,
			ok:   true,
		},
		{
			name: "объект со строковым chatId",
			args: []any{map[string]any{"chatId": "42"}},
			want: 42,
			ok:   true,
		},
		{
			name: "голое число вместо объекта",
			args: []any{float64(7)},
			want: 7,
			ok:   true,
		},
		{
			name: "нет аргументов",
			args: nil,
			ok:   false,
		},
		{
			name: "объект без поля chatId",
			args: []any{map[string]any{"foo": float64(1)}},
			ok:   false,
		},
		{
			name: "нулевой chatId отбраковывается — chat:0 не существует",
			args: []any{map[string]any{"chatId": float64(0)}},
			ok:   false,
		},
		{
			name: "отрицательный chatId не должен обернуться в огромный uint",
			args: []any{map[string]any{"chatId": float64(-1)}},
			ok:   false,
		},
		{
			name: "нечисловая строка",
			args: []any{map[string]any{"chatId": "abc"}},
			ok:   false,
		},
		{
			name: "неподдерживаемый тип",
			args: []any{map[string]any{"chatId": true}},
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseChatID(tt.args)

			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Непрерывный набор шлёт typing.start повторно — таймер должен сдвигаться, а не
// плодиться, иначе автосброс сработал бы от первого start посреди набора.
func TestTypingTrackerArmResetsExistingTimer(t *testing.T) {
	tracker := newTypingTracker()

	var fired int
	var mu sync.Mutex
	onExpire := func() {
		mu.Lock()
		fired++
		mu.Unlock()
	}

	tracker.arm(1, onExpire)
	tracker.arm(1, onExpire)
	tracker.arm(1, onExpire)

	tracker.mu.Lock()
	timerCount := len(tracker.timers)
	tracker.mu.Unlock()

	assert.Equal(t, 1, timerCount, "повторный arm не должен создавать второй таймер")

	mu.Lock()
	defer mu.Unlock()
	assert.Zero(t, fired, "автосброс не должен срабатывать сразу")
}

// disarm возвращает false, если гасить нечего: клиент может слать stop повторно,
// и ретранслировать его во второй раз не нужно.
func TestTypingTrackerDisarm(t *testing.T) {
	tracker := newTypingTracker()
	tracker.arm(1, func() {})

	assert.True(t, tracker.disarm(1), "первый stop гасит зажжённый индикатор")
	assert.False(t, tracker.disarm(1), "повторный stop гасить нечего")
	assert.False(t, tracker.disarm(999), "stop для чата, где не печатали")
}

// Таймеры независимы по чатам: пользователь может печатать в нескольких чатах.
func TestTypingTrackerStopAllReturnsActiveChats(t *testing.T) {
	tracker := newTypingTracker()
	tracker.arm(1, func() {})
	tracker.arm(2, func() {})
	tracker.disarm(1)
	tracker.arm(3, func() {})

	active := tracker.stopAll()

	assert.ElementsMatch(t, []uint{2, 3}, active,
		"на disconnect гасим только те чаты, где индикатор ещё горит")

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	assert.Empty(t, tracker.timers)
}

// После disconnect таймер не должен выстрелить в закрытый сокет.
func TestTypingTrackerStopAllPreventsLateExpiry(t *testing.T) {
	tracker := newTypingTracker()

	var fired bool
	var mu sync.Mutex
	tracker.arm(1, func() {
		mu.Lock()
		fired = true
		mu.Unlock()
	})

	tracker.stopAll()

	// Ждём заведомо дольше, чем прожил бы таймер, если бы его не сняли.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.False(t, fired, "снятый таймер не должен вызывать колбэк")
}

// arm после stopAll — сокет уже закрыт, новых таймеров заводить нельзя.
func TestTypingTrackerArmAfterStopIsNoop(t *testing.T) {
	tracker := newTypingTracker()
	tracker.stopAll()

	tracker.arm(1, func() {})

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	require.Empty(t, tracker.timers, "закрытый трекер не принимает новые таймеры")
}
