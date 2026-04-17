package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	logger     *zap.Logger
}

func NewServer(port int, logger *zap.Logger) *Server {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://real-time-map-frontend.vercel.app", "https://trip-scheduler.ru", "https://realtimemap.ru", "http://localhost:5174", "http://localhost:1420"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-User-Name", "X-User-Ban", "X-User-Admin", "X-Trace-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.HandleMethodNotAllowed = true
	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
		router: router,
		logger: logger,
	}
}

// Router возвращает gin.Engine для регистрации роутов
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Run запускает HTTP сервер
func (s *Server) Run() error {
	s.logger.Info("HTTP server starting", zap.String("address", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server stopping")
	return s.httpServer.Shutdown(ctx)
}
