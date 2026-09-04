package app

import (
	pkgprofile "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	producer2 "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_interaction"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_stat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/stat"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/grpc/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	CommentUseCases     *comment_action.Application
	InteractionUseCases *comment_interaction.Application
	StatUseCases        *comment_stat.Application

	publisher      comment_action.EventPublisher
	profileClient  *pkgprofile.Client
	profileAdapter *profilegrpc.Adapter

	Logger *zap.Logger
	DB     *gorm.DB
}

func MustContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	// Репозитории
	commentRepo := postgres.NewPgCommentRepository(db, logger)
	reactionRepo := postgres.NewPgReactionRepository(db, logger)
	statRepo := postgres.NewPgStatisticRepositoryRepository(db, logger)

	// Kafka producer (только если включён)
	var publisher comment_action.EventPublisher
	if cfg.Kafka.Enabled {
		p := producer2.New(
			producer2.DefaultConfig().
				WithBrokers(cfg.Kafka.Brokers[0]).
				WithTopic(cfg.Kafka.ProducerTopic),
			producer2.WithLogger(logger),
		)
		publisher = kafka.NewCommentPublisher(p, logger)
		logger.Info("Kafka event publisher initialized")
	} else {
		publisher = comment_action.NoOpEventPublisher{}
		logger.Info("Using NoOp event publisher (Kafka disabled)")
	}

	// Profile gRPC client + адаптер
	profileClient, err := pkgprofile.NewClient(&pkgprofile.Config{
		Address: cfg.Profile.Address,
		Timeout: cfg.Profile.Timeout,
	})
	if err != nil {
		logger.Fatal("failed to init profile gRPC client", zap.Error(err))
	}
	logger.Info("Profile gRPC client initialized", zap.String("address", cfg.Profile.Address))
	profileAdapter := profilegrpc.NewAdapter(profileClient)

	// Доменные сервисы
	commentService := comment.NewService(commentRepo, reactionRepo, logger)
	statService := stat.NewService(statRepo, logger)

	// Use cases
	commentUseCases := &comment_action.Application{
		CreateComment: comment_action.NewCreateCommentHandler(commentService, profileAdapter, publisher, logger),
		GetComments:   comment_action.NewGetCommentsHandler(commentService, profileAdapter, reactionRepo, logger),
		UpdateComment: comment_action.NewUpdateCommentHandler(commentService, profileAdapter, logger),
		DeleteComment: comment_action.NewDeleteCommentHandler(commentService, logger),
	}

	interactionUseCases := &comment_interaction.Application{
		LikeComment:   comment_interaction.NewLikeCommentHandler(commentService, logger),
		UnlikeComment: comment_interaction.NewUnlikeCommentHandler(commentService, logger),
	}

	statUseCases := &comment_stat.Application{
		GetStat: comment_stat.NewGetStatHandler(statService, logger),
	}

	return &Container{
		CommentUseCases:     commentUseCases,
		InteractionUseCases: interactionUseCases,
		StatUseCases:        statUseCases,
		publisher:           publisher,
		profileClient:       profileClient,
		profileAdapter:      profileAdapter,
		Logger:              logger,
		DB:                  db,
	}
}

func (c *Container) Close() error {
	if c.profileClient != nil {
		if err := c.profileClient.Close(); err != nil {
			c.Logger.Warn("profile gRPC client close failed", zap.Error(err))
		}
	}
	if closer, ok := c.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
