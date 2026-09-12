package personal

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
)

// UpdatePersonalMarkParams — частичное обновление: nil-поле означает "не трогать".
// Скалярные поля приходят указателями именно поэтому: пустой Title и снятый
// IsVisible иначе неотличимы от отсутствующего поля.
type UpdatePersonalMarkParams struct {
	Geom *types.Point

	Title       *string
	Description *string
	Category    *string
	Color       *string
	Icon        *string

	IsVisible *bool

	// GroupsIds заменяет состав групп целиком, если передан непустым.
	GroupsIds []uint

	// PhotosToDelete — URL существующих фото, ссылки на которые надо убрать.
	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput
}

func (s *Service) Update(ctx context.Context, params UpdatePersonalMarkParams, markID, userID uint) (*Model, error) {
	obj, err := s.repo.GetByID(ctx, markID, userID)
	if err != nil {
		return nil, err
	}

	if err = s.checkOwnerShip(obj, userID); err != nil {
		return nil, err
	}

	// Фото обрабатываются до транзакции: загрузка в S3 — сетевой вызов.
	photos, err := s.updatePhotos(ctx, obj.Photos, params.Photos, params.PhotosToDelete)
	if err != nil {
		return nil, err
	}

	validIds := utils.UniqueValues(params.GroupsIds)

	err = s.tx.WithTx(ctx, func(txCtx context.Context) error {
		var groups []*Group
		if len(validIds) > 0 {
			groups, err = s.groupRepo.GetBatch(txCtx, userID, validIds)
			if err != nil {
				return err
			}
		}

		revision, err := s.revisionRepo.UpsertRevision(txCtx, userID)
		if err != nil {
			return err
		}

		applyUpdates(&obj, params)
		obj.Photos = photos
		obj.Revision = revision

		if err := s.repo.Update(txCtx, &obj); err != nil {
			return err
		}

		if groups != nil {
			if err := s.repo.ReplaceGroups(txCtx, &obj, groups); err != nil {
				return err
			}
			obj.Groups = groups
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func applyUpdates(obj *Model, params UpdatePersonalMarkParams) {
	if params.Geom != nil {
		obj.Geom = *params.Geom
	}
	if params.Title != nil {
		obj.Title = *params.Title
	}
	if params.Description != nil {
		obj.Description = params.Description
	}
	if params.Category != nil {
		obj.Category = *params.Category
	}
	if params.Color != nil {
		obj.Color = *params.Color
	}
	if params.Icon != nil {
		obj.Icon = *params.Icon
	}
	if params.IsVisible != nil {
		obj.IsVisible = *params.IsVisible
	}
}

// updatePhotos убирает ссылки на удаляемые фото и добавляет новые.
//
// Байты из бакета не удаляются: ключ content-addressed, поэтому одно и то же
// фото в двух метках имеет один ключ, и удаление здесь стёрло бы файл во
// второй метке.
func (s *Service) updatePhotos(
	ctx context.Context,
	current types.Photos,
	newPhotos []mediavalidator.PhotoInput,
	photosToDelete []string,
) (types.Photos, error) {
	deleteMap := make(map[string]struct{}, len(photosToDelete))
	for _, url := range photosToDelete {
		deleteMap[url] = struct{}{}
	}

	kept := make(types.Photos, 0, len(current))
	for _, photo := range current {
		if _, drop := deleteMap[photo.URL]; !drop {
			kept = append(kept, photo)
		}
	}

	if len(kept)+len(newPhotos) > maxPhotos {
		return nil, ErrPhotosLimit(len(kept) + len(newPhotos))
	}

	if len(newPhotos) == 0 {
		return kept, nil
	}

	uploaded, err := s.uploadPhotos(ctx, newPhotos)
	if err != nil {
		return nil, ErrStorageOperation("upload photos", err)
	}

	return append(kept, uploaded...), nil
}
