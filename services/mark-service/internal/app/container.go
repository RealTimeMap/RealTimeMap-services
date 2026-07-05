package app

import (
	pkgprofile "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/category"
	category2 "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/accrual"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/stats"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	grpcstat "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/grpc/stats"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/socket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	DB *gorm.DB

	// Репозитории
	CategoryRepo repository.CategoryRepository
	MarkRepo     repository.MarkRepository
	AccrualRepo  accrual.Repository

	// Сервисы для пользовательский кейсов
	MarkStatsService *stats.MarkStatsService
	AccrualService   *accrual.Service

	// Сокет

	Socket *socket.SocketServer

	MarkUseCases     *mark_action.Application
	CategoryUseCases *category.Application

	// grpc
	MarkStatServer *grpcstat.Handler
	Logger         *zap.Logger
}

func MustContainer(cfg *config.Config, db *gorm.DB, log *zap.Logger) *Container {

	// создание репозиториев
	markRepo := postgres.NewMarkRepository(db, log)
	markStatRepo := postgres.NewMarkStatRepository(db, log)
	accrualRepo := postgres.NewPgAccrualRepository(db, log)

	// Создание вспомогательных компонентов
	//imageValidator := mediavalidator.NewPhotoValidator()
	store, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.Storage.BaseURL, log)
	if err != nil {
		panic(err)
	}

	// Kafka producer (только если включен)
	var p *producer.Producer
	if cfg.Kafka.Enabled {
		p = producer.New(
			producer.DefaultConfig().WithBrokers(cfg.Kafka.Brokers[0]).WithTopic(cfg.Kafka.ProducerTopic),
			producer.WithLogger(log),
		)
		log.Info("Kafka producer initialized", zap.String("topic", cfg.Kafka.ProducerTopic))
	} else {
		log.Info("Kafka producer disabled")
	}

	log.Debug("kafka producer initialized", zap.Any("topic", p))

	profileGrpcHandler, err := pkgprofile.NewClient(&pkgprofile.Config{
		Address: cfg.Profile.Address,
		Timeout: cfg.Profile.Timeout,
	})
	if err != nil {
		log.Fatal("Profile client initialization failed", zap.Error(err))
	}
	// Создание сервисов
	markStatService := stats.NewMarkStatsService(markStatRepo, log)
	accrualService := accrual.NewService(markRepo, accrualRepo, log)
	// админские сервисы

	// TODO После завершения переноса переименовать
	v2MarkRepo := postgres.NewMarkRepositoryV2(db, log)
	categoryRepo := postgres.NewCategoryRepositoryV2(db, log)

	markService := mark.NewService(v2MarkRepo, categoryRepo, store, log)
	categoryService := category2.NewService(categoryRepo)
	// USE CASE

	markUseCases := &mark_action.Application{
		CreateMark:  mark_action.NewCreateMarkHandler(markService, log),
		GetMark:     mark_action.NewMarkGetterHandler(markService, log),
		GetDetail:   mark_action.NewDetailMarkHandler(markService, profileGrpcHandler, log),
		DeleteMark:  mark_action.NewRemoverMarkHandler(markService, log),
		GetUserMark: mark_action.NewUserMarkGetterHandler(markService, log),
		UpdateMark:  mark_action.NewUpdateMarkHandler(markService, log),
	}

	categoryUseCases := &category.Application{
		Create: category.NewCreateCategoryCommand(categoryService, log),
		Get:    category.NewGetterCategoryHandler(categoryService, log),
	}
	// Сокеты
	socketServer := socket.New(socket.Deps{MarkUseCases: markUseCases, Logger: log})

	// grpc
	markStatGrpc := grpcstat.NewHandler(markStatService, log)

	// добавление
	return &Container{
		DB: db,

		AccrualRepo: accrualRepo,

		MarkStatsService: markStatService,
		AccrualService:   accrualService,

		Socket: socketServer,

		MarkStatServer: markStatGrpc,

		MarkUseCases:     markUseCases,
		CategoryUseCases: categoryUseCases,

		Logger: log,
	}

}
