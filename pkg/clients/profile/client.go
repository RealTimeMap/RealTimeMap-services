package profile

import (
	"context"
	"fmt"
	"time"

	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/profile"
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
	api     pb.ProfileServiceClient
	timeout time.Duration
}

func NewClient(cfg *Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to profile service: %w", err)
	}
	return &Client{
		conn:    conn,
		api:     pb.NewProfileServiceClient(conn),
		timeout: cfg.Timeout,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetUserProfileByID(ctx context.Context, id uint) (*UserProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.api.GetUserProfileByID(ctx, &pb.ProfileRequest{Id: uint64(id)})
	if err != nil {
		return nil, wrapErr(err)
	}
	return toProfile(resp), nil
}

func (c *Client) GetUserProfileByIDs(ctx context.Context, ids []uint) ([]*UserProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &pb.MultipleProfileRequest{Ids: make([]*pb.ProfileRequest, 0, len(ids))}
	for _, id := range ids {
		req.Ids = append(req.Ids, &pb.ProfileRequest{Id: uint64(id)})
	}

	resp, err := c.api.GetUserProfileByIDs(ctx, req)
	if err != nil {
		return nil, wrapErr(err)
	}

	out := make([]*UserProfile, 0, len(resp.GetProfiles()))
	for _, p := range resp.GetProfiles() {
		out = append(out, toProfile(p))
	}
	return out, nil
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

func toProfile(p *pb.ProfileResponse) *UserProfile {
	return &UserProfile{
		ID:       uint(p.GetId()),
		Username: p.GetUsername(),
		Tag:      p.GetTag(),
		Avatar:   p.GetAvatar(),
	}
}
