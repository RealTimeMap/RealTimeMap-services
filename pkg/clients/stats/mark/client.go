package mark

import (
	"context"
	"fmt"
	"time"

	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address string        `yaml:"address" env:"MARK_STATS_ADDRESS"`
	Timeout time.Duration `yaml:"timeout" env:"MARK_STATS_TIMEOUT"`
}

type Client struct {
	conn    *grpc.ClientConn
	api     markstat.MarkStatsServiceClient
	timeout time.Duration
}

func NewClient(cfg Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to MarkStatsService: %w", err)
	}
	return &Client{
		conn:    conn,
		api:     markstat.NewMarkStatsServiceClient(conn),
		timeout: cfg.Timeout,
	}, err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetUserMarksCount(ctx context.Context, userID uint) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.api.GetUserMarksCount(ctx, &markstat.MarksCountRequest{UserId: uint64(userID)})
	if err != nil {
		return 0, wrapErr(err)
	}
	return res.GetCount(), nil
}

func wrapErr(err error) error {
	if isUnavailable(err) {
		return fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
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
