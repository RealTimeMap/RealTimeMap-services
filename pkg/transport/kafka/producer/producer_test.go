package producer

import (
	"errors"
	"fmt"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// TestIsTopicNotReady проверяет, какие ошибки записи повторяются: только те,
// что брокер отдаёт, пока новый топик ещё создаётся.
func TestIsTopicNotReady(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"лидер ещё не выбран", kafka.LeaderNotAvailable, true},
		{"топика ещё нет", kafka.UnknownTopicOrPartition, true},
		{"обёрнутая ошибка", fmt.Errorf("write: %w", kafka.LeaderNotAvailable), true},
		{"внутри WriteErrors", kafka.WriteErrors{kafka.UnknownTopicOrPartition}, true},
		{"слишком большое сообщение", kafka.MessageSizeTooLarge, false},
		{"посторонняя ошибка", errors.New("connection refused"), false},
		{"WriteErrors без ошибки создания", kafka.WriteErrors{nil, kafka.MessageSizeTooLarge}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isTopicNotReady(tt.err))
		})
	}
}
