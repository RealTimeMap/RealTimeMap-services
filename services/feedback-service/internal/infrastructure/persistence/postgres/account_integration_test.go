package postgres

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Интеграционный тест обезличивания отчётов на настоящем Postgres.
//
//	go test ./services/feedback-service/... -run Integration -v

func TestIntegrationEraseUser(t *testing.T) {
	db := dbtest.Open(t, "rtm_feedback")
	require.NoError(t, db.AutoMigrate(&bug.Model{}))
	ctx := context.Background()

	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	victim, other := base+1, base+2
	title := "erase-test-" + t.Name()

	newBug := func(owner uint) *bug.Model {
		b := &bug.Model{UserID: &owner, IP: "10.0.0.1", Title: title, Desc: "шаги",
			App: bug.AppInfo{Build: "1.0", Logs: []string{"lat=55.7 lon=37.6"}}}
		require.NoError(t, db.Create(b).Error)
		return b
	}
	victimBug, otherBug := newBug(victim), newBug(other)
	t.Cleanup(func() { db.Unscoped().Where("title = ?", title).Delete(&bug.Model{}) })

	repo := NewPgAccountRepository(db)
	require.NoError(t, repo.EraseUser(ctx, victim))

	var got bug.Model
	require.NoError(t, db.First(&got, victimBug.ID).Error)
	assert.Nil(t, got.UserID)
	assert.Empty(t, got.IP)
	assert.Empty(t, got.App.Logs)
	assert.Equal(t, "шаги", got.Desc, "текст отчёта остаётся")
	assert.Equal(t, "1.0", got.App.Build)

	var untouched bug.Model
	require.NoError(t, db.First(&untouched, otherBug.ID).Error)
	require.NotNil(t, untouched.UserID)
	assert.Equal(t, other, *untouched.UserID, "чужой отчёт не тронут")
	assert.Equal(t, "10.0.0.1", untouched.IP)
	assert.Len(t, untouched.App.Logs, 1)

	require.NoError(t, repo.EraseUser(ctx, victim), "повторная доставка события")
}
