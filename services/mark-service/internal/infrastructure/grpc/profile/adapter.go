package profile

import (
	"context"
	"errors"

	pkgprofile "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
)

type Adapter struct {
	client *pkgprofile.Client
}

func NewAdapter(client *pkgprofile.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) GetUserProfileByID(ctx context.Context, id uint) (*model.UserProfile, error) {
	p, err := a.client.GetUserProfileByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return toProfile(p), nil
}

func (a *Adapter) GetUserProfileByIDs(ctx context.Context, ids []uint) ([]*model.UserProfile, error) {
	ps, err := a.client.GetUserProfileByIDs(ctx, ids)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*model.UserProfile, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProfile(p))
	}
	return out, nil
}

func mapError(err error) error {
	if errors.Is(err, pkgprofile.ErrUnavailable) {
		return domainerrors.ProfileUnavailable(err)
	}
	return err
}

func toProfile(p *pkgprofile.UserProfile) *model.UserProfile {
	return &model.UserProfile{
		ID:       p.ID,
		Username: p.Username,
		Tag:      p.Tag,
		Avatar:   p.Avatar,
	}
}
