package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "social-service")
	defer log.Sync()

	log.Info("Starting social service", zap.String("env", cfg.Env))

	// Database
	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)

	db.AutoMigrate(&model.Profile{}, &model.Friendship{}, &model.BlockedUser{})

	container := app.NewContainer(db, log)

	httpServer := httptransport.NewServer(cfg.Http.Port, log)
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

	g, gCtx := errgroup.WithContext(ctx)

	// HTTP server
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

	log.Info("Social Service stopped")

}
