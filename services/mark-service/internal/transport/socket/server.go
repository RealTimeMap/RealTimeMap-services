package socket

import (
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	"github.com/doquangtan/socketio/v4"
	"go.uber.org/zap"
)

type SocketServer struct {
	io      *socketio.Io
	useCase *mark_action.Application
	logger  *zap.Logger
}

type Deps struct {
	MarkUseCases *mark_action.Application

	Logger *zap.Logger
}

func New(deps Deps) *SocketServer {
	io := socketio.New()

	server := &SocketServer{
		io:      io,
		useCase: deps.MarkUseCases,
		logger:  deps.Logger,
	}

	InitMarkNamespace(server)

	return server
}

func (s *SocketServer) HttpHandler() http.Handler {
	return s.io.HttpHandler()
}
