package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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

	repo := postgres.NewCategoryRepository(db, log)
	markRepo := postgres.NewMarkRepository(db, log)
	store, _ := storage.NewLocalStorage("../../store", "http://localhost:8080/store", log) // TODO переести в контейнер DI
	categoryService := service.NewCategoryService(repo, store)
	markService := service.NewMarkService(markRepo, repo, store)
	router := gin.Default()

	handlers.InitCategoryHandler(router.Group("/"), categoryService, log)
	handlers.InitMarkHandler(router.Group("/"), markService, log)
	router.Static("./store", "./store")
	router.Run(":8080")
}
