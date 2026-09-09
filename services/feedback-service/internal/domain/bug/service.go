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

	// OnlyOpen отбирает баги, над которыми ещё идёт или может пойти
	// работа. Перечень для привязки к задаче показывает только их.
	OnlyOpen bool

	// OnlyUnlinked отбрасывает баги, уже взятые в другую задачу.
	OnlyUnlinked bool
}

// LinkParams — привязка бага к задаче таск-менеджера.
type LinkParams struct {
	BugID  uint
	TaskID uint
}

// SyncParams — обратная синхронизация: статус задачи переносится на баг.
type SyncParams struct {
	TaskID uint
	Status Status
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
		Pagination:   filter.Pagination,
		Status:       &status,
		Tag:          &tag,
		OnlyUnlinked: filter.OnlyUnlinked,
	}
	// Явный статус точнее набора «открытых»: если его спросили, он и
	// применяется, а OnlyOpen лишь сужает выборку до незакрытых.
	if filter.OnlyOpen && status == "" {
		f.Statuses = OpenStatuses()
	}
	f.Pagination.Defaults()

	return s.repo.GetList(ctx, f)
}

// GetByID отдаёт баг целиком — со всеми подробностями отчёта.
func (s *Service) GetByID(ctx context.Context, id uint) (*Model, error) {
	return s.repo.GetByID(ctx, id)
}

// Link привязывает баг к задаче и переводит его в работу.
//
// Привязка и статус меняются вместе: баг, взятый в задачу, по
// определению в работе, а отдельный вызов «поменяйте ещё и статус»
// оставлял бы окно, в котором баг занят, но выглядит новым.
func (s *Service) Link(ctx context.Context, params LinkParams) (*Model, error) {
	obj, err := s.repo.GetByID(ctx, params.BugID)
	if err != nil {
		return nil, err
	}

	// Повторная привязка к той же задаче — не ошибка: клиент мог
	// повторить запрос, и результат от этого не меняется.
	if obj.IsLinked() && !obj.IsLinkedTo(params.TaskID) {
		return nil, ErrBugAlreadyLinked(obj.ID, *obj.TaskID)
	}
	if !obj.Status.IsOpen() {
		return nil, ErrBugClosed(obj.ID, string(obj.Status))
	}

	obj.TaskID = &params.TaskID
	obj.Status = InWork

	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, err
	}

	s.logger.Info("bug linked to task",
		zap.Uint("bug_id", obj.ID),
		zap.Uint("task_id", params.TaskID),
	)
	return obj, nil
}

// UnlinkBug снимает привязку с конкретного бага.
//
// Нужен, когда задача меняет баг: прежний баг ищется по своему
// идентификатору, а не по задаче, — искать по задаче уже поздно, её
// привязка указывает на новый баг.
func (s *Service) UnlinkBug(ctx context.Context, bugID uint) (*Model, error) {
	obj, err := s.repo.GetByID(ctx, bugID)
	if err != nil {
		return nil, err
	}

	return s.release(ctx, obj)
}

// Unlink снимает привязку и возвращает баг в перечень свободных.
//
// Отсутствие привязки не ошибка: задачу могли удалить, так и не
// связав её с багом, и отказ здесь заставил бы вызывающего разбирать
// заведомо безобидный случай.
func (s *Service) Unlink(ctx context.Context, taskID uint) (*Model, error) {
	obj, err := s.repo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return s.release(ctx, obj)
}

// release снимает привязку и возвращает баг в перечень свободных.
func (s *Service) release(ctx context.Context, obj *Model) (*Model, error) {
	if obj == nil || !obj.IsLinked() {
		return nil, nil
	}

	previous := *obj.TaskID
	obj.TaskID = nil
	// Закрытый баг открывать обратно незачем: работа над ним завершена,
	// и отвязка от задачи этого не отменяет.
	if obj.Status.IsOpen() {
		obj.Status = New
	}

	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, err
	}

	s.logger.Info("bug unlinked from task",
		zap.Uint("bug_id", obj.ID),
		zap.Uint("task_id", previous),
	)
	return obj, nil
}

// SyncStatus переносит статус задачи на привязанный баг.
//
// Задача — источник истины для бага, который в ней ведут: пока она в
// работе, баг в работе, а её завершение закрывает баг. Задача без
// привязки просто ничего не меняет.
func (s *Service) SyncStatus(ctx context.Context, params SyncParams) (*Model, error) {
	if !params.Status.IsValid() {
		return nil, ErrBugStatusUnavailable(string(params.Status))
	}

	obj, err := s.repo.GetByTaskID(ctx, params.TaskID)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	if obj.Status == params.Status {
		return obj, nil
	}

	obj.Status = params.Status
	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, err
	}

	s.logger.Info("bug status synced from task",
		zap.Uint("bug_id", obj.ID),
		zap.Uint("task_id", params.TaskID),
		zap.String("status", string(params.Status)),
	)
	return obj, nil
}
