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

// Config конфиг для env/yaml заточенный под работу с HTTP
type Config struct {
	Port             int           `env:"PORT" yaml:"port"`
	AllowOrigins     []string      `env:"ALLOW_ORIGINS" yaml:"allow_origins" env-separator:","`
	AllowMethods     []string      `env:"ALLOW_METHODS" yaml:"allow_methods" env-separator:","`
	AllowHeaders     []string      `env:"ALLOW_HEADERS" yaml:"allow_headers" env-separator:","`
	AllowCredentials bool          `env:"ALLOW_CREDENTIALS" yaml:"allow_credentials" env-default:"true"`
	MaxAge           time.Duration `env:"MAX_AGE" yaml:"max_age" env-default:"12h"`
}

var (
	defaultAllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	defaultAllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-User-Name", "X-User-Ban", "X-User-Admin", "X-Trace-Id"}
)

type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	logger     *zap.Logger
}

// NewServer Создает настроенный экземпляр сервера для HTTP поверх GIN
func NewServer(cfg Config, logger *zap.Logger) *Server {
	methods := cfg.AllowMethods
	if len(methods) == 0 {
		methods = defaultAllowMethods
	}
	headers := cfg.AllowHeaders
	if len(headers) == 0 {
		headers = defaultAllowHeaders
	}

	router := gin.Default()
	router.HandleMethodNotAllowed = true
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     methods,
		AllowHeaders:     headers,
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}))

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
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
