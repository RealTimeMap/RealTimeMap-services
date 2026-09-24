package kafka

import (
	"encoding/json"
	"testing"

	segmentio "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
)

// TestExtractMetaBugConfirmed проверяет разбор события feedback-service без
// брокера: автор отчёта и баг берутся из тела, заголовки не нужны.
func TestExtractMetaBugConfirmed(t *testing.T) {
	body, err := json.Marshal(events.NewBugConfirmed(events.BugPayload{
		BugID:  9005,
		UserID: 1005,
		Tag:    "ui",
	}))
	require.NoError(t, err)

	meta, err := extractMeta(segmentio.Message{Value: body})
	require.NoError(t, err)

	assert.Equal(t, events.BugConfirmed, meta.EventType)
	assert.Equal(t, uint(1005), meta.UserID)
	require.NotNil(t, meta.SourceID, "bugId из тела становится источником события")
	assert.Equal(t, uint(9005), *meta.SourceID)
}
