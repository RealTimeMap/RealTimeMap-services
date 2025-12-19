package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/socket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	// Репозитории
	CategoryRepo repository.CategoryRepository
	MarkRepo     repository.MarkRepository

	// Сервисы для пользовательский кейсов
	MarkService     *service.UserMarkService
	CategoryService *service.CategoryService

	// Сервисы для админских кейсов
	AdminMarkService *service.AdminMarkService

	// Сокет

	Socket *socket.SocketServer
}

func MustContainer(cfg *config.Config, db *gorm.DB, log *zap.Logger) *Container {

	// создание репозиториев
	categoryRepo := postgres.NewCategoryRepository(db, log)
	markRepo := postgres.NewMarkRepository(db, log)

	// Создание вспомогательных компонентов
	imageValidator := mediavalidator.NewPhotoValidator()
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

	// Создание сервисов
	categoryService := service.NewCategoryService(categoryRepo, store)
	markService := service.NewUserMarkService(markRepo, categoryRepo, store, p, imageValidator)

	// админские сервисы
	adminMarkService := service.NewAdminMarkService(markRepo, categoryRepo, store, p, imageValidator)

	// Сокеты
	socketServer := socket.New(log, markService)

	// добавление
	return &Container{

		CategoryRepo: categoryRepo,
		MarkRepo:     markRepo,

		MarkService:     markService,
		CategoryService: categoryService,

		AdminMarkService: adminMarkService,

		Socket: socketServer,
	}

}
