package socket

import (
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/doquangtan/socketio/v4"
	"go.uber.org/zap"
)

type SocketServer struct {
	io     *socketio.Io
	logger *zap.Logger

	markService *service.UserMarkService
}

func New(logger *zap.Logger, markService *service.UserMarkService) *SocketServer {
	io := socketio.New()

	server := &SocketServer{
		logger:      logger,
		markService: markService,
		io:          io,
	}

	InitMarkNamespace(server)

	return server
}

func (s *SocketServer) HttpHandler() http.Handler {
	return s.io.HttpHandler()
}
