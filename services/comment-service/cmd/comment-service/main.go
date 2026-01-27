package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "comment-service")
	defer log.Sync()

	log.Info("Starting Comment Service", zap.String("env", cfg.Env))

	// Database
	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)
	db.AutoMigrate(&model.Comment{})

	container := app.NewContainer(cfg, db, log)
	if container.Producer != nil {
		defer container.Producer.Close()
	}

	httpServer := httptransport.NewServer(cfg.HTTP.Port, log)
	httptransport.RegisterRoutes(httpServer.Router(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info("Shutdown signal received")
		cancel()
	}()

	// Run all servers with errgroup
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return httpServer.Run()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		log.Error("Server error", zap.Error(err))
	}

	log.Info("Comment Service stopped")
}
