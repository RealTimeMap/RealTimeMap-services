package postgres

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/like"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Интеграционный тест стирания меток на настоящем Postgres.
//
//	go test ./services/mark-service/... -run Integration -v

func TestIntegrationEraseUser(t *testing.T) {
	db := dbtest.Open(t, "rtm_marks")
	require.NoError(t, db.AutoMigrate(&like.Reaction{}, &mark.Mark{}, &category.Category{},
		&personal.Group{}, &personal.Model{}, &personal.Revision{}))
	ctx := context.Background()

	var cat category.Category
	if err := db.First(&cat).Error; err != nil {
		t.Skipf("no categories: %v", err)
	}

	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	victim, other := base+1, base+2
	users := []uint{victim, other}
	t.Cleanup(func() { cleanupMarks(db, users) })

	point := types.Point{Point: orb.Point{37.6, 55.7}}
	now := time.Now()
	newMark := func(owner uint) *mark.Mark {
		m := &mark.Mark{MarkName: "test", UserID: owner, UserName: "u", CategoryID: cat.ID,
			StartAt: now, EndAt: now.Add(time.Hour), Geom: point, Geohash: "ucfv0"}
		require.NoError(t, db.Create(m).Error)
		return m
	}
	victimMark, softDeleted, otherMark := newMark(victim), newMark(victim), newMark(other)
	require.NoError(t, db.Delete(softDeleted).Error)

	require.NoError(t, db.Create(&[]like.Reaction{
		{MarkID: victimMark.ID, UserID: other}, // чужой лайк на метке удалённого
		{MarkID: otherMark.ID, UserID: victim}, // лайк удалённого на чужой метке
	}).Error)

	for _, owner := range users {
		g := personal.Group{ID: uuid.New(), Name: "list", UserID: owner}
		require.NoError(t, db.Create(&g).Error)
		pm := personal.Model{UserID: owner, Geom: point, Title: "home", Groups: []*personal.Group{&g}}
		require.NoError(t, db.Create(&pm).Error)
		require.NoError(t, db.Create(&personal.Revision{UserID: owner, Revision: 3}).Error)
	}

	repo := NewPgAccountRepository(db)
	require.NoError(t, repo.EraseUser(ctx, victim))

	for _, m := range []any{&mark.Mark{}, &personal.Model{}, &personal.Group{}, &personal.Revision{}} {
		assert.Zero(t, countWhere(t, db, m, "user_id = ?", victim), "%T удалённого", m)
		assert.EqualValues(t, 1, countWhere(t, db, m, "user_id = ?", other), "%T чужой", m)
	}
	assert.Zero(t, countWhere(t, db, &like.Reaction{}, "user_id = ? OR mark_id = ?", victim, victimMark.ID))
	assert.Zero(t, joinRows(t, db, victim))
	assert.EqualValues(t, 1, joinRows(t, db, other), "связь чужой личной метки со списком цела")

	require.NoError(t, repo.EraseUser(ctx, victim), "повторная доставка события")
}

func countWhere(t *testing.T, db *gorm.DB, m any, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(m).Where(where, args...).Count(&n).Error)
	return n
}

func joinRows(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM personal_marks_groups pmg
		JOIN groups g ON g.id = pmg.group_id WHERE g.user_id = ?`, userID).Scan(&n).Error)
	return n
}

func cleanupMarks(db *gorm.DB, users []uint) {
	db.Exec(`DELETE FROM personal_marks_groups WHERE group_id IN (SELECT id FROM groups WHERE user_id IN ?)`, users)
	db.Where("user_id IN ?", users).Delete(&like.Reaction{})
	for _, m := range []any{&mark.Mark{}, &personal.Model{}, &personal.Group{}, &personal.Revision{}} {
		db.Unscoped().Where("user_id IN ?", users).Delete(m)
	}
}
