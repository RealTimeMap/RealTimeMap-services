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
	groups, err := s.groupRepo.GetBatch(ctx, params.UserID, validIds)
	if err != nil {
		return nil, err
	}

	var photos types.Photos
	if len(params.Photos) > 0 {
		uploadedPhotos, err := s.uploadPhotos(ctx, params.Photos)
		if err != nil {
			return nil, ErrStorageOperation("upload photos", err)
		}
		photos = uploadedPhotos
	}

	payload := &Model{
		UserID:      params.UserID,
		Geom:        params.Geom,
		Groups:      groups,
		IsVisible:   params.IsVisible,
		Title:       params.Title,
		Description: params.Description,
		Category:    params.Category,
		Photos:      photos,
	}

	err = s.repo.Create(ctx, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (s *Service) validateData(params CreatePersonalMarkParams) error {
	if len(params.Photos) > maxPhotos {
		return ErrPhotosLimit(len(params.Photos))
	}
	return nil
}
