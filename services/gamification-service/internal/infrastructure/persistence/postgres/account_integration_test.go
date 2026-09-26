package postgres

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Интеграционный тест стирания игровых данных на настоящем Postgres.
// Требует засеянных уровней и достижений (их заводит старт сервиса).
//
//	go test ./services/gamification-service/... -run Integration -v

func TestIntegrationEraseUser(t *testing.T) {
	db := dbtest.Open(t, "rtm_gamification")
	ctx := context.Background()

	var ach model.Achievement
	if err := db.First(&ach).Error; err != nil {
		t.Skipf("no seeded achievements: %v", err)
	}

	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	victim, other := base+1, base+2
	users := []uint{victim, other}
	t.Cleanup(func() {
		for _, m := range userModels() {
			db.Unscoped().Where("user_id IN ?", users).Delete(m)
		}
	})

	for _, id := range users {
		require.NoError(t, db.Create(&model.UserProgress{UserID: id, CurrentLevel: 1}).Error)
		require.NoError(t, db.Create(&model.XPOperation{UserID: id, Amount: 10}).Error)
		require.NoError(t, db.Create(&model.UserAchievementCount{UserID: id, EventType: "mark.created", Count: 3}).Error)
		require.NoError(t, db.Create(&model.UserAchievement{UserID: id, AchievementID: ach.ID, UnlockedAt: time.Now()}).Error)
	}

	repo := NewPgAccountRepository(db)
	require.NoError(t, repo.EraseUser(ctx, victim))

	for _, m := range userModels() {
		assert.Zero(t, countUser(t, db, m, victim), "%T удалённого пользователя", m)
		assert.EqualValues(t, 1, countUser(t, db, m, other), "%T чужого пользователя", m)
	}

	// Повторная доставка события — не ошибка.
	require.NoError(t, repo.EraseUser(ctx, victim))
}

func userModels() []any {
	return []any{&model.XPOperation{}, &model.UserAchievement{}, &model.UserAchievementCount{}, &model.UserProgress{}}
}

func countUser(t *testing.T, db *gorm.DB, m any, userID uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(m).Where("user_id = ?", userID).Count(&n).Error)
	return n
}
