package main

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/handlers"
	"github.com/gin-contrib/cors"
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

	container := app.MustContainer(cfg, db, log)

	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://real-time-map-frontend.vercel.app", "https://trip-scheduler.ru", "https://realtimemap.ru", "http://localhost:5173", "http://localhost:1420"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-User-Name", "X-User-Ban", "X-User-Admin", "X-Trace-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.HandleMethodNotAllowed = true

	// Handle OPTIONS for all routes (including non-existent ones)
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.JSON(404, gin.H{"error": "Not found"})
	})

	router.NoMethod(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.JSON(405, gin.H{"error": "Method not allowed"})
	})

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))
	router.POST("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))
	apiV1 := router.Group("/api/v2")

	handlers.InitCategoryHandler(apiV1, container.CategoryService, log)
	handlers.InitMarkHandler(apiV1, container.MarkService, log)
	handlers.InitAdminMarkHandler(apiV1, container.AdminMarkService, log)
	router.Static("./store", "./store")

	// Health check endpoint for Docker/K8s
	healthHandler := func(c *gin.Context) {
		// Проверка подключения к БД
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{
				"status":   "unhealthy",
				"database": "down",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "mark-service",
			"env":     cfg.Env,
		})
	}

	// Support both GET and HEAD methods for health check
	apiV1.GET("/health", healthHandler)
	apiV1.HEAD("/health", healthHandler)

	router.Run(":8080")
}
