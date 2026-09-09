package personal

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
)

type CreatePersonalMarkParams struct {
	UserID uint
	Geom   types.Point

	Title       string
	Description *string
	Category    string
	Color       string
	Icon        string // Иконка из iconfy

	IsVisible bool

	GroupsIds []uint
	Photos    []mediavalidator.PhotoInput
}

func (s *Service) Create(ctx context.Context, params CreatePersonalMarkParams) (*Model, error) {
	if err := s.validateData(params); err != nil {
		return nil, err
	}

	validIds := utils.UniqueValues(params.GroupsIds)
	if len(validIds) < 1 {
		return nil, ErrGroupsRequired(validIds)
	}

	// Фото грузятся до транзакции: это сетевой вызов в S3
	var photos types.Photos
	if len(params.Photos) > 0 {
		uploaded, err := s.uploadPhotos(ctx, params.Photos)
		if err != nil {
			return nil, ErrStorageOperation("upload photos", err)
		}
		photos = uploaded
	}

	var obj *Model
	err := s.tx.WithTx(ctx, func(txCtx context.Context) error {
		groups, err := s.groupRepo.GetBatch(txCtx, params.UserID, validIds)
		if err != nil {
			return err
		}
		revision, err := s.revisionRepo.UpsertRevision(txCtx, params.UserID)
		if err != nil {
			return err
		}

		payload := &Model{
			UserID:      params.UserID,
			Geom:        params.Geom,
			Groups:      groups,
			IsShare:     false,
			IsVisible:   params.IsVisible,
			Title:       params.Title,
			Description: params.Description,
			Category:    params.Category,
			Color:       params.Color,
			Icon:        params.Icon,
			Photos:      photos,
			Revision:    revision,
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

func (s *Service) validateData(params CreatePersonalMarkParams) error {
	if len(params.Photos) > maxPhotos {
		return ErrPhotosLimit(len(params.Photos))
	}
	return nil
}
