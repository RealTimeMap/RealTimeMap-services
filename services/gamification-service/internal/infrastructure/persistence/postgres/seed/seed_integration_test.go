package seed

import (
	"context"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// dsn указывает на ту же базу, что и config.yaml сервиса.
const dsn = "host=localhost port=5432 user=postgres password=admin dbname=rtm_gamification sslmode=disable"

// TestSeedRun проверяет сидер против живой базы: что он заводит строки и что
// повторный запуск ничего не дублирует и не меняет.
func TestSeedRun(t *testing.T) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres недоступен: %v", err)
	}

	require.NoError(t, db.AutoMigrate(
		&model.Level{}, &model.UserProgress{}, &model.Achievement{},
		&model.UserAchievement{}, &model.XPReward{}, &model.EventRule{},
		&model.XPOperation{}, &model.UserAchievementCount{},
	))

	ctx := context.Background()
	log := zap.NewNop()

	require.NoError(t, Run(ctx, db, log))

	var rule model.EventRule
	require.NoError(t, db.Where("event_type = ?", "comment.created").First(&rule).Error)
	require.True(t, rule.IsActive)
	require.NotNil(t, rule.DailyLimit)
	require.Equal(t, commentDailyLimit, *rule.DailyLimit)

	var reward model.XPReward
	require.NoError(t, db.First(&reward, rule.RewardID).Error)
	require.Equal(t, uint(5), reward.Amount)

	var achievements []model.Achievement
	require.NoError(t, db.Where("trigger_event_type = ?", "comment.created").
		Order("threshold ASC").Find(&achievements).Error)
	require.Len(t, achievements, len(commentAchievements))

	// Цепочка связана: каждая ступень, кроме последней, указывает на следующую.
	for i := 0; i < len(achievements)-1; i++ {
		require.NotNil(t, achievements[i].NextID, "ступень %s без next", achievements[i].Code)
		require.Equal(t, achievements[i+1].ID, *achievements[i].NextID)
	}
	require.Nil(t, achievements[len(achievements)-1].NextID)

	// Повторный запуск не должен ничего добавить.
	require.NoError(t, Run(ctx, db, log))

	var countAfter int64
	require.NoError(t, db.Model(&model.Achievement{}).
		Where("trigger_event_type = ?", "comment.created").Count(&countAfter).Error)
	require.Equal(t, int64(len(commentAchievements)), countAfter)

	var rewardCount int64
	require.NoError(t, db.Model(&model.XPReward{}).
		Where("code IN ?", []string{"comment_created", "ach_first_comment", "ach_commenter_10", "ach_commenter_50", "ach_commenter_100"}).
		Count(&rewardCount).Error)
	require.Equal(t, int64(len(commentRewards)), rewardCount)
}
