package progress

import (
	"context"
	"fmt"
	"time"

	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/proto/gamification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address string        `yaml:"address" env:"GAMIFICATION_GRPC_ADDRESS"`
	Timeout time.Duration `yaml:"timeout" env:"GAMIFICATION_GRPC_TIMEOUT" env-default:"300ms"`
}

type Client struct {
	conn    *grpc.ClientConn
	api     pb.ProgressServiceClient
	timeout time.Duration
}

func NewClient(cfg *Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to profile service: %w", err)
	}
	return &Client{
		conn:    conn,
		api:     pb.NewProgressServiceClient(conn),
		timeout: cfg.Timeout,
	}, nil

}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetUserProgress(ctx context.Context, id uint) (*UserExpProgress, error) {

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.api.GetUserProgress(ctx, &pb.GetUserProgressRequest{UserId: uint64(id)})
	if err != nil {
		return nil, wrapErr(err)
	}
	return toProgress(resp), nil
}

func wrapErr(err error) error {
	if isUnavailable(err) {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return err
}

func isUnavailable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func toProgress(p *pb.UserProgressResponse) *UserExpProgress {
	return &UserExpProgress{
		UserID:          uint(p.GetUserId()),
		CurrentLevel:    p.GetCurrentLevel(),
		CurrentXP:       p.GetCurrentXp(),
		ProgressPercent: p.GetProgressPercent(),
		XPForNextLevel:  p.GetXpForNextLevel(),
	}
}
