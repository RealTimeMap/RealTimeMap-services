package personal

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CreateGroupParams struct {
	// ID — идентификатор, сгенерированный клиентом для группы, созданной
	// офлайн. Нулевое значение означает "клиент не указал" — тогда UUID
	// выдаёт сервер.
	ID          uuid.UUID
	Name        string
	Description *string
	UserID      uint
	Color       string
	Icon        string // Иконка из iconfy
}

func (s *GroupService) CreateGroup(ctx context.Context, params CreateGroupParams) (*Group, error) {
	s.logger.Info("start CreateGroup", zap.String("layer", "domain service"))

	if err := validateGroupColor(params.Color); err != nil {
		return nil, err
	}
	if err := validateGroupIcon(params.Icon); err != nil {
		return nil, err
	}

	id := params.ID
	if id == uuid.Nil {
		generated, err := uuid.NewRandom()
		if err != nil {
			return nil, ErrGroupIDGeneration(err)
		}
		id = generated
	}

	var obj *Group
	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		// Занятый идентификатор — конфликт, даже если владелец чужой: иначе
		// ответ раскрыл бы существование чужой группы либо, того хуже, привязал
		// к ней метки. Проверка внутри транзакции, но гонку окончательно
		// снимает уникальность первичного ключа в Create.
		exists, err := s.repo.ExistsByID(txCtx, id)
		if err != nil {
			return err
		}
		if exists {
			return ErrGroupIDTaken(id)
		}

		revision, err := s.revisionRepo.UpsertRevision(txCtx, params.UserID)
		if err != nil {
			return err
		}
		payload := &Group{
			ID:          id,
			Name:        params.Name,
			Description: params.Description,
			UserID:      params.UserID,
			Revision:    revision,
			Color:       params.Color,
			Icon:        params.Icon,
		}
		if err := s.repo.Create(txCtx, payload); err != nil {
			return err
		}

		obj = payload
		return nil
	})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
