package input

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
)

// MarkInput - чистые данные для создания метки
type MarkInput struct {
	MarkName       valueobject.MarkName
	AdditionalInfo *string
	CategoryId     int
	StartAt        time.Time
	Duration       valueobject.Duration
	Geom           types.Point
	Geohash        string
	Photos         []mediavalidator.PhotoInput // Чистые данные: []byte + filename
	context.UserInput
}

// MarkUpdateInput - чистые данные для обновления метки
type MarkUpdateInput struct {
	MarkID         int
	MarkName       *valueobject.MarkName
	AdditionalInfo *string
	CategoryId     *int
	Duration       *valueobject.Duration

	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput // Чистые данные: []byte + filename

	context.UserInput // TODO Что это вообще за хуйня?!
}
