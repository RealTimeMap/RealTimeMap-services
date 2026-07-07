package app

import (
	pkgprofile "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/category"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_stat"
	category2 "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	grpcstat "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/grpc/stats"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/socket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	DB *gorm.DB

	// Сокет

	Socket *socket.SocketServer

	MarkUseCases     *mark_action.Application
	CategoryUseCases *category.Application

	// grpc
	MarkStatServer *grpcstat.Handler
	Logger         *zap.Logger
}

func MustContainer(cfg *config.Config, db *gorm.DB, log *zap.Logger) *Container {
	// Создание вспомогательных компонентов
	//imageValidator := mediavalidator.NewPhotoValidator()
	store, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.Storage.BaseURL, log)
	if err != nil {
		panic(err)
	}

	// Kafka producer (только если включен).
	// eventPublisher остаётся nil-интерфейсом, если Kafka выключена —
	// use case это допускает и просто не публикует событие.
	var p *producer.Producer
	var eventPublisher mark_action.EventPublisher
	if cfg.Kafka.Enabled {
		p = producer.New(
			producer.DefaultConfig().WithBrokers(cfg.Kafka.Brokers[0]).WithTopic(cfg.Kafka.ProducerTopic),
			producer.WithLogger(log),
		)

		eventPublisher = p
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

	// Создание доменных репозиториев
	statRepo := postgres.NewPgMarkStatRepository(db, log)
	markRepo := postgres.NewMarkRepositoryV2(db, log)
	categoryRepo := postgres.NewCategoryRepositoryV2(db, log)

	// Создание доменных сервисов
	statService := mark.NewStatService(statRepo, log)
	markService := mark.NewService(markRepo, categoryRepo, store, log)
	categoryService := category2.NewService(categoryRepo)

	// USE CASE

	markUseCases := &mark_action.Application{
		CreateMark:  mark_action.NewCreateMarkHandler(markService, eventPublisher, log),
		GetMark:     mark_action.NewMarkGetterHandler(markService, log),
		GetDetail:   mark_action.NewDetailMarkHandler(markService, profileGrpcHandler, log),
		DeleteMark:  mark_action.NewRemoverMarkHandler(markService, log),
		GetUserMark: mark_action.NewUserMarkGetterHandler(markService, log),
		UpdateMark:  mark_action.NewUpdateMarkHandler(markService, log),
	}

	markStatUseCases := &mark_stat.Application{
		GetCategories:    mark_stat.NewMarkStatCategoryHandler(statService, log),
		GetMonthActivity: mark_stat.NewMarkMonthHandler(statService, log),
		GetHeatMap:       mark_stat.NewMarkHeatMapHandler(statService, log),
		GetMarkCount:     mark_stat.NewMarkCountHandler(statService, log),
	}

	categoryUseCases := &category.Application{
		Create: category.NewCreateCategoryCommand(categoryService, log),
		Get:    category.NewGetterCategoryHandler(categoryService, log),
	}
	// Сокеты
	socketServer := socket.New(socket.Deps{MarkUseCases: markUseCases, Logger: log})

	// grpc
	markStatGrpc := grpcstat.NewHandler(markStatUseCases, log)

	// добавление
	return &Container{
		DB: db,

		Socket: socketServer,

		MarkStatServer: markStatGrpc,

		MarkUseCases:     markUseCases,
		CategoryUseCases: categoryUseCases,

		Logger: log,
	}

}
