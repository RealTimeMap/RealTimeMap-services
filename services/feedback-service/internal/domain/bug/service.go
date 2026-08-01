package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"go.uber.org/zap"
)

type Service struct {
	repo Repository

	logger *zap.Logger
}

type ApplicationInfoParams struct {
	Build string
	Logs  []string
}

type DeviceInfoParams struct {
	OS         string
	Platform   string
	Resolution string
	Battery    *float64
}
type CreateBugParams struct {
	Title, Desc string
	Tag         string
	IP          string
	UserID      *uint
	App         ApplicationInfoParams
	Device      DeviceInfoParams
}

type GetBugParams struct {
	Pagination pagination.Params
	Tag        *string
	Status     *string
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, data CreateBugParams) error {
	if valid := Tag.IsValid(Tag(data.Tag)); !valid {
		return ErrBugTagUnavailable(data.Tag)
	}

	payload := &Model{
		Title:  data.Title,
		Desc:   data.Desc,
		Tag:    Tag(data.Tag),
		IP:     data.IP,
		UserID: data.UserID,
		App:    AppInfo{Build: data.App.Build, Logs: data.App.Logs},
		Device: DeviceInfo{
			OS:         data.Device.OS,
			Platform:   data.Device.Platform,
			Resolution: data.Device.Resolution,
		},
	}
	if err := s.repo.Create(ctx, payload); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetList(ctx context.Context, filter GetBugParams) ([]Model, error) {
	var tag Tag
	if filter.Tag != nil {
		tag = Tag(*filter.Tag)
		if !tag.IsValid() {
			return nil, ErrBugTagUnavailable(*filter.Tag)
		}
	}

	var status Status
	if filter.Status != nil {
		status = Status(*filter.Status)
		if !status.IsValid() {
			return nil, ErrBugStatusUnavailable(*filter.Status)
		}
	}
	f := Filter{
		Pagination: filter.Pagination,
		Status:     &status,
		Tag:        &tag,
	}
	f.Pagination.Defaults()

	return s.repo.GetList(ctx, f)
}
