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
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (c *Client) GetUserMarksMonthlyActivity(ctx context.Context, userID uint, year int) ([]*MonthlyActivity, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.api.GetUserMarksMonthlyActivity(ctx, &markstat.MarksMonthlyActivityRequest{UserId: uint64(userID), Year: int64(year)})
	if err != nil {
		return nil, wrapErr(err)
	}
	return toMonthlyActivityResponse(res), nil
}

func (c *Client) GetUserMarksHeatMap(ctx context.Context, userID uint, start, end time.Time) ([]*HeatMapItem, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.api.GetUserMarksHeatMap(ctx, &markstat.MarksHeatMapRequest{UserId: uint64(userID), StartDate: timestamppb.New(start), EndDate: timestamppb.New(end)})
	if err != nil {
		return nil, wrapErr(err)
	}
	return toHeatMapResponse(res), nil
}

func (c *Client) GetPopularUserCategories(ctx context.Context, userID uint, topN int) ([]*PopularCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.api.GetPopularUserCategories(ctx, &markstat.PopularCategoriesRequest{UserId: uint64(userID), TopN: int64(topN)})
	if err != nil {
		return nil, wrapErr(err)
	}
	return toPopularCategoriesResponse(res), nil
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

func toHeatMapResponse(data *markstat.MarksHeatMapResponse) []*HeatMapItem {
	if data == nil {
		return nil
	}
	res := make([]*HeatMapItem, 0, len(data.GetActivity()))
	for _, activity := range data.GetActivity() {
		res = append(res, &HeatMapItem{
			Day:   activity.Day.AsTime(),
			Count: activity.Count,
		})
	}
	return res
}

func toMonthlyActivityResponse(data *markstat.UserMarksActivityResponse) []*MonthlyActivity {
	res := make([]*MonthlyActivity, 0, len(data.GetActivities()))
	for _, a := range data.GetActivities() {
		res = append(res, &MonthlyActivity{
			Month: a.GetMonth(),
			Count: a.GetCount(),
		})
	}
	return res

}

func toPopularCategoriesResponse(data *markstat.PopularCategoriesResponse) []*PopularCategory {
	res := make([]*PopularCategory, 0, len(data.GetCategories()))
	for _, category := range data.GetCategories() {
		res = append(res, &PopularCategory{
			CategoryName: category.GetCategoryName(),
			Count:        category.GetCount(),
			Percent:      category.GetPercent(),
		})
	}
	return res
}
