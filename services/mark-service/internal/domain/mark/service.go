package mark

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
)

const (
	maxPhotosPerMark     = 10  // Максимум 10 фото
	maxStartAtPastDays   = 1   // Не более 1 дня назад
	maxStartAtFutureDays = 30  // Не более 30 дней вперед
	maxMarksPerDay       = 100 // Лимит на создание меток для пользователя TODO уменьшить для production версии
)

type CreateMarkParams struct {
	MarkName       string
	AdditionalInfo *string
	CategoryId     uint
	StartAt        time.Time
	EndAt          *time.Time
	Geom           types.Point
	Geohash        string
	Photos         []mediavalidator.PhotoInput // Чистые данные: []byte + filename
}

type UpdateMarkParams struct {
	MarkName       string
	AdditionalInfo *string
	EndAt          *time.Time

	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput
}

type Service struct {
	markRepo     Repository
	categoryRepo category.Repository

	store  storage.Storage
	logger *zap.Logger
}

func NewService(
	markRepo Repository, categoryRepo category.Repository,
	store storage.Storage, logger *zap.Logger,
) *Service {
	return &Service{
		markRepo:     markRepo,
		categoryRepo: categoryRepo,
		store:        store,
		logger:       logger,
	}
}
