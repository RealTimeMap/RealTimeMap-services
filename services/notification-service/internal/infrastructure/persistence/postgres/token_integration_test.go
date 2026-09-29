package postgres

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
)

//	go test ./services/notification-service/... -run Integration -v

func TestIntegrationDeleteByDevice(t *testing.T) {
	db := dbtest.Open(t, "rtm_notifications")
	require.NoError(t, db.AutoMigrate(&token.Model{}))
	repo := NewPgUserTokenRepository(db, zap.NewNop())
	ctx := context.Background()

	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	me, other := base+1, base+2
	t.Cleanup(func() { db.Where("user_id IN ?", []uint{me, other}).Delete(&token.Model{}) })

	register := func(userID uint, deviceID, tok string) {
		require.NoError(t, repo.Register(ctx, &token.Model{
			UserID: userID, DeviceID: deviceID, Token: tok,
			Platform: token.PlatformAndroid, Settings: token.DefaultSettings(),
		}))
	}
	register(me, "phone", "tok-me-phone-"+t.Name())
	register(me, "laptop", "tok-me-laptop-"+t.Name())
	register(other, "phone", "tok-other-phone-"+t.Name())

	deleted, err := repo.DeleteByDevice(ctx, me, "phone")
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	mine, err := repo.GetByUser(ctx, me)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, "laptop", mine[0].DeviceID, "остальные устройства пользователя на месте")

	others, err := repo.GetByUser(ctx, other)
	require.NoError(t, err)
	assert.Len(t, others, 1, "устройство с тем же deviceId у другого пользователя не тронуто")

	deleted, err = repo.DeleteByDevice(ctx, me, "phone")
	require.NoError(t, err, "повторное удаление — не ошибка")
	assert.Zero(t, deleted)
}
