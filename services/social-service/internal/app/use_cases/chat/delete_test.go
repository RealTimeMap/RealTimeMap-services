package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDeleter struct {
	res *services.DeleteResult
	err error
}

func (f *fakeDeleter) Delete(context.Context, uint, uint, bool) (*services.DeleteResult, error) {
	return f.res, f.err
}

type recordingRooms struct {
	published []ChatEvent
	left      [][]uint
}

func (r *recordingRooms) Publish(_ context.Context, e ChatEvent) error {
	r.published = append(r.published, e)
	return nil
}

func (r *recordingRooms) JoinUsers(context.Context, uint, []uint) error { return nil }

func (r *recordingRooms) LeaveUsers(_ context.Context, _ uint, ids []uint) error {
	r.left = append(r.left, ids)
	return nil
}

func TestDeleteHandler(t *testing.T) {
	t.Run("у себя — событие только своим устройствам, комната сохраняется", func(t *testing.T) {
		rooms := &recordingRooms{}
		h := NewDeleteHandler(&fakeDeleter{res: &services.DeleteResult{ParticipantIDs: []uint{1}}}, rooms, zap.NewNop())

		require.NoError(t, h.Handle(context.Background(), DeleteCommand{ChatID: 10, UserID: 1}))

		require.Len(t, rooms.published, 1)
		e := rooms.published[0]
		assert.Equal(t, EventChatDeleted, e.Type)
		assert.Equal(t, []uint{1}, e.RecipientIDs, "собеседник об удалении у себя не узнаёт")
		assert.Equal(t, DeletedResult{ChatID: 10, DeletedBy: 1, ForEveryone: false}, e.Payload)
		assert.Empty(t, rooms.left, "удаливший остаётся в комнате: новое сообщение вернёт чат")
	})

	t.Run("у всех — событие всем участникам и выход из комнаты", func(t *testing.T) {
		rooms := &recordingRooms{}
		h := NewDeleteHandler(&fakeDeleter{res: &services.DeleteResult{ForEveryone: true, ParticipantIDs: []uint{1, 2}}}, rooms, zap.NewNop())

		require.NoError(t, h.Handle(context.Background(), DeleteCommand{ChatID: 10, UserID: 1, ForEveryone: true}))

		require.Len(t, rooms.published, 1)
		assert.Equal(t, []uint{1, 2}, rooms.published[0].RecipientIDs)
		assert.Equal(t, DeletedResult{ChatID: 10, DeletedBy: 1, ForEveryone: true}, rooms.published[0].Payload)
		assert.Equal(t, [][]uint{{1, 2}}, rooms.left)
	})

	t.Run("ошибка домена — ничего не рассылается", func(t *testing.T) {
		rooms := &recordingRooms{}
		h := NewDeleteHandler(&fakeDeleter{err: errors.New("forbidden")}, rooms, zap.NewNop())

		require.Error(t, h.Handle(context.Background(), DeleteCommand{ChatID: 10, UserID: 1}))
		assert.Empty(t, rooms.published)
		assert.Empty(t, rooms.left)
	})
}
