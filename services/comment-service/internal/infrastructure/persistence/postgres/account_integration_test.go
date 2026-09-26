package postgres

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/reaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Интеграционный тест обезличивания на настоящем Postgres.
//
//	go test ./services/comment-service/... -run Integration -v

func TestIntegrationEraseUser(t *testing.T) {
	db := dbtest.Open(t, "rtm_comments")
	require.NoError(t, db.AutoMigrate(&comment.Comment{}, &reaction.Reaction{}))
	ctx := context.Background()

	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	victim, other, entity := base+1, base+2, base+3

	parent := comment.Comment{UserID: victim, Username: "victim", Content: "вопрос", EntityType: "mark_action", EntityID: entity}
	require.NoError(t, db.Create(&parent).Error)
	reply := comment.Comment{UserID: other, Username: "other", Content: "ответ", EntityType: "mark_action", EntityID: entity, ParentID: &parent.ID, Depth: 1}
	require.NoError(t, db.Create(&reply).Error)
	softDeleted := comment.Comment{UserID: victim, Username: "victim", Content: "стёрто", EntityType: "mark_action", EntityID: entity}
	require.NoError(t, db.Create(&softDeleted).Error)
	require.NoError(t, db.Delete(&softDeleted).Error)
	require.NoError(t, db.Create(&[]reaction.Reaction{
		{CommentID: reply.ID, UserID: victim},
		{CommentID: parent.ID, UserID: other},
	}).Error)
	t.Cleanup(func() {
		db.Where("comment_id IN ?", []uint{parent.ID, reply.ID}).Delete(&reaction.Reaction{})
		db.Unscoped().Where("entity_id = ?", entity).Delete(&comment.Comment{})
	})

	repo := NewPgAccountRepository(db)
	require.NoError(t, repo.EraseUser(ctx, victim))

	var got []comment.Comment
	require.NoError(t, db.Unscoped().Where("entity_id = ?", entity).Order("id").Find(&got).Error)
	require.Len(t, got, 3, "комментарии не удаляются — ветка ответов должна уцелеть")

	assert.Zero(t, got[0].UserID)
	assert.Empty(t, got[0].Username)
	assert.Equal(t, "вопрос", got[0].Content)
	assert.Equal(t, other, got[1].UserID, "чужой ответ не тронут")
	assert.Equal(t, "other", got[1].Username)
	assert.Zero(t, got[2].UserID, "мягко удалённый тоже обезличен")
	assert.Empty(t, got[2].Username)

	var victimLikes, otherLikes int64
	db.Model(&reaction.Reaction{}).Where("user_id = ?", victim).Count(&victimLikes)
	db.Model(&reaction.Reaction{}).Where("user_id = ?", other).Count(&otherLikes)
	assert.Zero(t, victimLikes)
	assert.EqualValues(t, 1, otherLikes)

	require.NoError(t, repo.EraseUser(ctx, victim), "повторная доставка события")
}
