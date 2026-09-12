package comment_test

import (
	"context"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	pgpersistence "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const dsn = "host=localhost port=5432 user=postgres password=admin dbname=rtm_comments sslmode=disable"

// TestCreateReplySetsParentWithoutDuplicating проверяет, что проставленный
// вручную Comment.Parent доезжает до вызывающего и при этом GORM не вставляет
// родителя повторно как новую строку.
func TestCreateReplySetsParentWithoutDuplicating(t *testing.T) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("postgres недоступен: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&comment.Comment{}))

	log := zap.NewNop()
	repo := pgpersistence.NewPgCommentRepository(db, log)
	svc := comment.NewService(repo, nil, log)
	ctx := context.Background()

	var before int64
	require.NoError(t, db.Model(&comment.Comment{}).Count(&before).Error)

	root, err := svc.Create(ctx, comment.CreateParams{
		Content:    "родительский комментарий",
		EntityType: string(comment.EntityMark),
		EntityID:   777,
	}, 100, "author_root")
	require.NoError(t, err)
	require.Nil(t, root.Parent)

	reply, err := svc.Create(ctx, comment.CreateParams{
		Content:    "ответ",
		EntityType: string(comment.EntityMark),
		EntityID:   777,
		ParentID:   &root.ID,
	}, 200, "author_reply")
	require.NoError(t, err)

	// Родитель доступен вызывающему — из него берётся адресат уведомления.
	require.NotNil(t, reply.Parent, "Parent должен быть проставлен для ответа")
	require.Equal(t, uint(100), reply.Parent.UserID)
	require.Equal(t, root.ID, reply.Parent.ID)

	// Ровно две новые строки: родитель и ответ. Третья означала бы, что GORM
	// вставил родителя повторно через ассоциацию.
	var after int64
	require.NoError(t, db.Model(&comment.Comment{}).Count(&after).Error)
	require.Equal(t, before+2, after, "родитель не должен вставляться повторно")

	t.Cleanup(func() {
		db.Unscoped().Delete(&comment.Comment{}, "entity_id = ?", 777)
	})
}
