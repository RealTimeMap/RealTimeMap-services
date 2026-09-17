package kafka

import (
	"encoding/json"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeEvent разбирает два формата сразу: конверт и плоское тело, которым
// auth слал регистрацию раньше. Тип и payload из них достаются по-разному,
// поэтому проверяются оба, плюс откат на headers.
func TestDecodeEvent(t *testing.T) {
	tests := []struct {
		name      string
		msg       kafka.Message
		wantType  string
		wantAdmin bool
	}{
		{
			name: "конверт: тип из type, payload из вложенного объекта",
			msg: kafka.Message{Value: []byte(
				`{"type":"user.updated","payload":{"user_id":7,"is_admin":true}}`)},
			wantType:  events.UserUpdated,
			wantAdmin: true,
		},
		{
			name: "плоский формат: тип из event_type, payload — само тело",
			msg: kafka.Message{Value: []byte(
				`{"event_type":"user.updated","user_id":7,"is_admin":true}`)},
			wantType:  events.UserUpdated,
			wantAdmin: true,
		},
		{
			name: "тип только в headers — тело без него остаётся разбираемым",
			msg: kafka.Message{
				Value:   []byte(`{"user_id":7,"is_admin":true}`),
				Headers: []kafka.Header{{Key: "event_type", Value: []byte("user.updated")}},
			},
			wantType:  events.UserUpdated,
			wantAdmin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, payload, err := decodeEvent(tt.msg)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, eventType)

			user, err := decodeUpdated(payload)
			require.NoError(t, err)
			require.NotNil(t, user.IsAdmin)
			assert.Equal(t, uint(7), user.UserID)
			assert.Equal(t, tt.wantAdmin, *user.IsAdmin)
		})
	}
}

// Отсутствующий is_admin и явный false — разные вещи: первый означает, что
// событие не про права и трогать профиль нельзя, второй снимает админку.
func TestDecodeUpdatedDistinguishesMissingFromFalse(t *testing.T) {
	missing, err := decodeUpdated(json.RawMessage(`{"user_id":7,"username":"bob"}`))
	require.NoError(t, err)
	assert.Nil(t, missing.IsAdmin, "ключа нет — применять нечего")

	explicit, err := decodeUpdated(json.RawMessage(`{"user_id":7,"is_admin":false}`))
	require.NoError(t, err)
	require.NotNil(t, explicit.IsAdmin, "явный false — это снятие админки")
	assert.False(t, *explicit.IsAdmin)
}

// Профиль с нулевым идентификатором не принадлежит никому: такое событие
// должно отбраковываться, а не заводить и не править чужую запись.
func TestDecodeRejectsMissingUserID(t *testing.T) {
	for _, payload := range []string{`{"is_admin":true}`, `{"user_id":0,"is_admin":true}`} {
		_, err := decodeUpdated(json.RawMessage(payload))
		assert.Error(t, err, payload)

		_, err = decodeRegistered(json.RawMessage(payload))
		assert.Error(t, err, payload)
	}
}

// Регистрация несёт признак администратора: обычный пользователь приезжает с
// false, заранее заведённый админ — с true.
func TestDecodeRegisteredCarriesAdminFlag(t *testing.T) {
	plain, err := decodeRegistered(json.RawMessage(`{"user_id":7,"username":"bob"}`))
	require.NoError(t, err)
	assert.False(t, plain.IsAdmin)
	assert.Equal(t, "bob", plain.Username)

	admin, err := decodeRegistered(json.RawMessage(`{"user_id":8,"username":"root","is_admin":true}`))
	require.NoError(t, err)
	assert.True(t, admin.IsAdmin)
}
