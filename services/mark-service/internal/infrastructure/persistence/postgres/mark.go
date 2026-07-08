package postgres

import (
	"context"
	"errors"
	"math"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/paulmach/orb"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const clusterPixelThreshold = 60.0

type MarkRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewMarkRepositoryV2(db *gorm.DB, logger *zap.Logger) mark.Repository {
	return &MarkRepository{
		db:    db,
		log:   logger,
		layer: "mark_repository",
	}
}

func (r *MarkRepository) Create(ctx context.Context, data *mark.Mark) (*mark.Mark, error) {
	r.log.Info("create mark_action in: ", sl.String("layer", r.layer))

	// Создаем запись
	err := r.db.WithContext(ctx).Create(data).Error
	if err != nil {
		r.log.Error("create mark_action err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	// Загружаем связанную Category для возврата полного объекта
	err = r.db.WithContext(ctx).Preload("Category").First(data, data.ID).Error
	if err != nil {
		r.log.Error("failed to preload category: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	return data, nil
}

func (r *MarkRepository) Update(ctx context.Context, id uint, obj *mark.Mark) (*mark.Mark, error) {
	r.log.Info("MarkRepository.Update", zap.Uint("id", id))

	err := r.db.WithContext(ctx).Model(&mark.Mark{}).Where("id = ?", id).Save(obj).Error
	if err != nil {
		r.log.Error("update_mark_by_id err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return obj, nil
}

func (r *MarkRepository) GetByID(ctx context.Context, id uint) (*mark.Mark, error) {
	r.log.Info("get_mark_by_id", sl.String("layer", r.layer))
	var obj *mark.Mark
	err := r.db.WithContext(ctx).Model(&mark.Mark{}).Preload("Category").Where("id = ? AND deleted_at IS NULL", id).First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mark.ErrMarkNotFound(id)
		}
		r.log.Error("get_mark_by_id err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return obj, nil
}

func (r *MarkRepository) Delete(ctx context.Context, id uint) error {
	r.log.Info("delete_mark_by_id", sl.String("layer", r.layer))

	result := r.db.WithContext(ctx).Delete(&mark.Mark{}, id)
	if result.Error != nil {
		r.log.Error("delete_mark_by_id err: ", sl.String("layer", r.layer), zap.Error(result.Error))
		return result.Error
	}

	// Проверка, что запись существовала
	if result.RowsAffected == 0 {
		return mark.ErrMarkNotFound(id)
	}

	return nil
}

func (r *MarkRepository) TodayCreated(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&mark.Mark{}).Where("user_id = ? AND DATE(created_at) = CURRENT_DATE", userID).Count(&count).Error
	if err != nil {
		r.log.Error("failed to get mark_action count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *MarkRepository) GetMarksInArea(ctx context.Context, filter mark.Filter) ([]*mark.Mark, error) {
	var marks []*mark.Mark
	bbox := filter.BoundingBox
	err := r.db.WithContext(ctx).Model(&mark.Mark{}).
		Joins("Category").
		Where("geom && ST_MakeEnvelope(?, ?, ?, ?, 4326)", bbox.LeftTop.Lon, bbox.RightBottom.Lat, bbox.RightBottom.Lon, bbox.LeftTop.Lat).
		Where("start_at <= ? AND end_at >= ?", filter.EndAt, filter.StartAt).
		Where("deleted_at IS NULL").
		Find(&marks).Error
	if err != nil {
		r.log.Error("error MarkRepository.GetMarksInArea", zap.Error(err))
		return nil, err
	}

	return marks, nil
}

func (r *MarkRepository) GetMarksInCluster(ctx context.Context, filter mark.Filter) ([]*mark.Cluster, error) {
	type clusterResult struct {
		ClusterID int     `gorm:"column:cluster_id"`
		CenterLon float64 `gorm:"column:center_lon"`
		CenterLat float64 `gorm:"column:center_lat"`
		Count     int     `gorm:"column:count"`
	}

	var results []clusterResult
	bbox := filter.BoundingBox
	query := `
        WITH clustered_marks AS (
            SELECT
                id,
                geom,
                ST_ClusterDBSCAN(geom, eps := ?, minpoints := ?) OVER (
                    ORDER BY id
                ) AS cluster_id
            FROM marks
            WHERE geom && ST_MakeEnvelope(?, ?, ?, ?, 4326)
              AND start_at <= ?
              AND end_at >= ?
              AND deleted_at IS NULL
        )
        SELECT
            cluster_id,
            ST_X(ST_Centroid(ST_Collect(geom))) AS center_lon,
            ST_Y(ST_Centroid(ST_Collect(geom))) AS center_lat,
            COUNT(*) AS count
        FROM clustered_marks
        WHERE cluster_id IS NOT NULL
        GROUP BY cluster_id
    `

	eps := clusterPixelThreshold * 360.0 / (256.0 * math.Pow(2, filter.ZoomLevel))

	err := r.db.WithContext(ctx).Raw(query, eps, 1, bbox.LeftTop.Lon, bbox.RightBottom.Lat, bbox.RightBottom.Lon, bbox.LeftTop.Lat, filter.EndAt, filter.StartAt).Scan(&results).Error
	if err != nil {
		r.log.Error("failed to get marks in cluster", zap.Error(err))
		return nil, err
	}
	clusters := make([]*mark.Cluster, len(results))
	for i, result := range results {
		clusters[i] = &mark.Cluster{
			Center: types.Point{
				Point: orb.Point{result.CenterLon, result.CenterLat},
			},
			Count: result.Count,
		}
	}
	return clusters, nil
}

func (r *MarkRepository) GetUserMarks(ctx context.Context, userID uint, params pagination.Params) ([]*mark.Mark, int64, error) {
	r.log.Info("GetUserMarks", zap.Uint("user_id", userID))
	var marks []*mark.Mark
	var count int64
	err := r.db.WithContext(ctx).Model(&mark.Mark{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(params.Limit()).
		Offset(params.Offset()).
		Find(&marks).
		Count(&count).Error
	return marks, count, err
}

func (r *MarkRepository) Exist(ctx context.Context, id uint) (bool, error) {
	r.log.Info("check_exist_mark_by_id", sl.String("layer", r.layer))
	var exists bool
	err := r.db.WithContext(ctx).
		Model(&mark.Mark{}).
		Select("1").
		Where("id = ?", id).
		Limit(1).
		Find(&exists).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, mark.ErrMarkNotFound(id)
		}
		r.log.Error("check_exist_mark_by_id err: ", sl.String("layer", r.layer), zap.Error(err))
		return false, err
	}
	return exists, nil
}

func (r *MarkRepository) IncShare(ctx context.Context, markID uint) (int64, error) {
	var obj mark.Mark

	err := r.db.WithContext(ctx).
		Model(&obj).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "shared_count"}}}).
		Where("id = ?", markID).
		Update("shared_count", gorm.Expr("shared_count + 1")).Error
	if err != nil {
		return 0, err
	}

	return obj.SharedCount, nil
}
