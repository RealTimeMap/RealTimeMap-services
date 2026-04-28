package app

import (
	pkgprofile "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	producer2 "github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service/comment"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/grpc/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	CommentService *comment.Service
	EventPublisher service.EventPublisher
	ProfileAdapter *profilegrpc.Adapter

	profileClient *pkgprofile.Client

	Logger *zap.Logger
	DB     *gorm.DB
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {

	// Транзакции
	txManager := postgres.NewTxManager(db)

	// Репозитории
	commentRepo := postgres.NewPgCommentRepository(db, logger)
	reactionRepo := postgres.NewPgReactionRepository(db, logger)

	// Kafka producer (только если включен)
	var publisher service.EventPublisher
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
		publisher = &service.NoOpEventPublisher{}
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

	// Сервисы
	commentService := comment.NewCommentService(commentRepo, reactionRepo, publisher, txManager, profileAdapter, logger)

	return &Container{
		CommentService: commentService,
		EventPublisher: publisher,
		ProfileAdapter: profileAdapter,
		profileClient:  profileClient,
		DB:             db,
		Logger:         logger,
	}
}

func (c *Container) Close() error {
	if c.profileClient != nil {
		if err := c.profileClient.Close(); err != nil {
			c.Logger.Warn("profile gRPC client close failed", zap.Error(err))
		}
	}
	if closer, ok := c.EventPublisher.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
