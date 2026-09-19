package app

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	pkgredis "github.com/RealTimeMap/RealTimeMap-backend/pkg/redis"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app/use_cases/notify"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/notification"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
	collapseredis "github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/infrastructure/collapse/redis"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/infrastructure/fcm"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/infrastructure/persistence/postgres"
)

type Container struct {
	TokenService *token.Service
	Fcm          notification.Sender

	// CollapseLimiter отдаётся наружу ради обхода сводок: он работает с тем
	// же состоянием серий, что и хендлер Kafka.
	CollapseLimiter collapse.Limiter

	NotifyUseCase *notify.Application
	DB            *gorm.DB
	Redis         *redis.Client
	Logger        *zap.Logger
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	// REPO

	tokenRepo := postgres.NewPgUserTokenRepository(db, logger)

	// Infra
	fcm, err := fcm.NewFirebaseSender(context.Background(), cfg.Firebase.ProjectID, logger)
	if err != nil {
		logger.Fatal("container build failed", zap.Error(err))
	}

	rdb := pkgredis.NewRedisCli(cfg.Redis)
	limiter := collapseredis.NewLimiter(rdb)

	// Services

	tokenSrv := token.NewService(tokenRepo, logger)

	// UseCases
	notifyUser := notify.NewUserNotifyHanlder(tokenSrv, fcm, logger)
	notifyUseCases := &notify.Application{
		NotifyUser:  notifyUser,
		NotifyEvent: notify.NewEventNotifyHandler(notifyUser, limiter, cfg.Collapse.Window, logger),
	}

	return &Container{
		TokenService:    tokenSrv,
		Fcm:             fcm,
		CollapseLimiter: limiter,
		NotifyUseCase:   notifyUseCases,
		DB:              db,
		Redis:           rdb,
		Logger:          logger,
	}
}
