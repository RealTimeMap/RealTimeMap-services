// Package user — клиент UserService, реализованного auth-сервисом на Python.
//
// Сервис слушает порт 50052 (см. run_grpc.py и docker-compose backend).
package user

import (
	"context"
	"fmt"
	"time"

	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address string
	Timeout time.Duration
}

type Client struct {
	conn    *grpc.ClientConn
	api     pb.UserServiceClient
	timeout time.Duration
}

func NewClient(cfg *Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to user service: %w", err)
	}
	return &Client{
		conn:    conn,
		api:     pb.NewUserServiceClient(conn),
		timeout: cfg.Timeout,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// GetUserByID возвращает пользователя вместе с его email.
func (c *Client) GetUserByID(ctx context.Context, id int64) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.api.GetUserById(ctx, &pb.UserRequest{Id: id})
	if err != nil {
		return nil, wrapErr(err)
	}

	return &User{
		ID:          resp.GetId(),
		Username:    resp.GetUsername(),
		Email:       resp.GetEmail(),
		IsSuperuser: resp.GetIsSuperuser(),
	}, nil
}

// wrapErr переводит grpc-коды в доменные ошибки пакета.
//
// Различие нужно вызывающему для решения об offset: недоступность сервиса —
// повод перечитать сообщение позже, отсутствие пользователя — повод пропустить
// его навсегда.
func wrapErr(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	case codes.NotFound:
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	default:
		return err
	}
}
