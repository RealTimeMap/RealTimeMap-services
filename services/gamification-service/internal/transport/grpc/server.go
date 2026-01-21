package grpc

import (
	"fmt"
	"net"

	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/proto/gamification"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *zap.Logger
}

func NewServer(handler *Handler, port int, logger *zap.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()

	// Register the ProgressService
	pb.RegisterProgressServiceServer(grpcServer, handler)

	//// Enable reflection for debugging tools like grpcurl
	//reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		listener:   listener,
		logger:     logger,
	}, nil
}

// Run starts the gRPC server (blocking)
func (s *Server) Run() error {
	s.logger.Info("gRPC server starting", zap.String("address", s.listener.Addr().String()))
	return s.grpcServer.Serve(s.listener)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
