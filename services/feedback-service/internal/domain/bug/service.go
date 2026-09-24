package bug

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"go.uber.org/zap"
)

type Service struct {
	repo Repository

	// now — источник времени решения по багу; подменяется в тестах.
	now func() time.Time

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

// ConfirmParams — разработчик воспроизвёл баг.
type ConfirmParams struct {
	BugID   uint
	Comment string
}

// RejectParams — проверка не подтвердила баг.
type RejectParams struct {
	BugID   uint
	Reason  RejectReason
	Comment string
}

// SyncParams — обратная синхронизация: статус задачи переносится на баг.
type SyncParams struct {
	TaskID uint
	Status Status
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, now: time.Now, logger: logger}
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
			Battery:    data.Device.Battery,
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
	if obj.Status == New {
		return nil, ErrBugNotConfirmed(obj.ID)
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
	// Баг возвращается подтверждённым, а не новым: проверку он уже прошёл,
	// и снятие с задачи этого не отменяет. Закрытый баг открывать обратно
	// тоже незачем — работа над ним завершена.
	if obj.Status == InWork {
		obj.Status = Confirmed
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
//
// Статусы проверки задача не выставляет: отклонение — отдельное решение
// разработчика с причиной (Reject). Возврат задачи в «new» переносится
// на баг как «confirmed»: привязанный баг проверку уже прошёл, и
// отправлять его на повторную проверку задача не вправе.
func (s *Service) SyncStatus(ctx context.Context, params SyncParams) (*Model, error) {
	switch params.Status {
	case New:
		params.Status = Confirmed
	case Confirmed, InWork, Closed, Canceled:
	default:
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

// Confirm фиксирует, что разработчик воспроизвёл баг: с этого момента
// его можно брать в задачу.
//
// Повторное подтверждение — не ошибка: клиент мог повторить запрос.
// Подтвердить можно только непроверенный баг; отклонённый сначала
// возвращается на проверку (Reopen), чтобы смена решения была явной.
//
// Второе значение сообщает, что баг подтверждён впервые за всю его
// историю, — только тогда автору отчёта положена награда.
func (s *Service) Confirm(ctx context.Context, params ConfirmParams) (*Model, bool, error) {
	obj, err := s.repo.GetByID(ctx, params.BugID)
	if err != nil {
		return nil, false, err
	}

	switch obj.Status {
	case Confirmed:
		return obj, false, nil
	case New:
	default:
		return nil, false, ErrBugReviewForbidden(obj.ID, obj.Status, "confirmed")
	}

	now := s.stamp()
	obj.Status = Confirmed
	obj.Review = Review{At: now, Comment: params.Comment}

	first := obj.FirstConfirmedAt == nil
	if first {
		obj.FirstConfirmedAt = now
	}

	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, false, err
	}

	s.logger.Info("bug confirmed",
		zap.Uint("bug_id", obj.ID),
		zap.Bool("first_confirmation", first),
	)
	return obj, first, nil
}

// Reject фиксирует, что проверка не подтвердила баг.
//
// Причина обязательна и берётся из фиксированного набора: по ней видно,
// какие отчёты оказываются пустыми. Отклонить можно непроверенный или
// подтверждённый, но ещё не взятый в задачу баг: у бага в работе
// источник истины — задача, и снимать его нужно через неё.
func (s *Service) Reject(ctx context.Context, params RejectParams) (*Model, error) {
	if !params.Reason.IsValid() {
		return nil, ErrRejectReasonUnavailable(string(params.Reason))
	}

	obj, err := s.repo.GetByID(ctx, params.BugID)
	if err != nil {
		return nil, err
	}

	switch obj.Status {
	case Rejected:
		return obj, nil
	case New, Confirmed:
		if obj.IsLinked() {
			return nil, ErrBugAlreadyLinked(obj.ID, *obj.TaskID)
		}
	default:
		return nil, ErrBugReviewForbidden(obj.ID, obj.Status, "rejected")
	}

	obj.Status = Rejected
	obj.Review = Review{At: s.stamp(), RejectReason: params.Reason, Comment: params.Comment}

	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, err
	}

	s.logger.Info("bug rejected",
		zap.Uint("bug_id", obj.ID),
		zap.String("reason", string(params.Reason)),
	)
	return obj, nil
}

// Reopen возвращает баг на повторную проверку и стирает прежнее решение.
//
// Нужен, когда решение оказалось ошибочным: отклонённый баг всё-таки
// воспроизвёлся или подтверждение было поспешным. Баг в работе или уже
// закрытый так не вернуть — им управляет задача.
func (s *Service) Reopen(ctx context.Context, bugID uint) (*Model, error) {
	obj, err := s.repo.GetByID(ctx, bugID)
	if err != nil {
		return nil, err
	}

	switch obj.Status {
	case New:
		return obj, nil
	case Rejected, Confirmed:
		if obj.IsLinked() {
			return nil, ErrBugAlreadyLinked(obj.ID, *obj.TaskID)
		}
	default:
		return nil, ErrBugReviewForbidden(obj.ID, obj.Status, "reopened")
	}

	obj.Status = New
	obj.Review = Review{}

	if err := s.repo.Update(ctx, obj); err != nil {
		return nil, err
	}

	s.logger.Info("bug reopened for review", zap.Uint("bug_id", obj.ID))
	return obj, nil
}

func (s *Service) stamp() *time.Time {
	t := s.now().UTC()
	return &t
}
