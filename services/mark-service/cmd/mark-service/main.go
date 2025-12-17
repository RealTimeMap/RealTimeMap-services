package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/docs"
)

// @title           Your API
// @version         1.0
// @description     Описание вашего API
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "mark-service")
	defer log.Sync()

	log.Info("Starting Mark Service", zap.String("env", cfg.Env))

	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)

	db.AutoMigrate(&model.Mark{}, &model.Category{})

	// Инициализация Kafka Producer

	p := producer.New(producer.DefaultConfig().WithBrokers(cfg.Kafka.Brokers[0]).WithTopic("marks"), producer.WithLogger(log))
	imageValidator := mediavalidator.NewPhotoValidator()
	repo := postgres.NewCategoryRepository(db, log)
	markRepo := postgres.NewMarkRepository(db, log)
	store, _ := storage.NewLocalStorage("../../store", "http://localhost:8080/store", log) // TODO переести в контейнер DI

	categoryService := service.NewCategoryService(repo, store)
	markService := service.NewMarkService(markRepo, repo, store, p, imageValidator) // ← Передаём Kafka producer
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	apiV1 := router.Group("/api/v1")

	handlers.InitCategoryHandler(apiV1, categoryService, log)
	handlers.InitMarkHandler(apiV1, markService, log)
	router.Static("./store", "./store")
	router.Run(":8080")
}
