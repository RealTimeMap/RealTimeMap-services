package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/paulmach/orb"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MarkRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewMarkRepository(db *gorm.DB, logger *zap.Logger) repository.MarkRepository {
	return &MarkRepository{
		db:    db,
		log:   logger,
		layer: "mark_repository",
	}
}

func (r *MarkRepository) Create(ctx context.Context, data *model.Mark) (*model.Mark, error) {
	r.log.Info("create mark in: ", sl.String("layer", r.layer))

	// Создаем запись
	err := r.db.WithContext(ctx).Create(data).Error
	if err != nil {
		r.log.Error("create mark err: ", sl.String("layer", r.layer), zap.Error(err))
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

func (r *MarkRepository) TodayCreated(ctx context.Context, userID int) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Mark{}).Where("user_id = ? AND DATE(created_at) = CURRENT_DATE", userID).Count(&count).Error
	if err != nil {
		r.log.Error("failed to get mark count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *MarkRepository) GetMarksInArea(ctx context.Context, filter repository.Filter) ([]*model.Mark, error) {
	var marks []*model.Mark
	bbox := filter.BoundingBox
	err := r.db.WithContext(ctx).Model(&model.Mark{}).
		Joins("Category").
		Where("geom && ST_MakeEnvelope(?, ?, ?, ?, 4326)", bbox.LeftTop.Lon, bbox.RightBottom.Lat, bbox.RightBottom.Lon, bbox.LeftTop.Lat).
		//Where("geohash IN (?)", filter.GeoHashes()).
		Where("start_at >= ?", filter.StartAt).
		Where("(end_at) >= ?", filter.EndAt).
		Find(&marks).Error
	if err != nil {
		r.log.Error("failed to get marks in area", zap.Error(err))
		return nil, err
	}

	return marks, nil
}

func (r *MarkRepository) GetMarksInCluster(ctx context.Context, filter repository.Filter) ([]*model.Cluster, error) {
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
              AND start_at >= ?
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

	err := r.db.WithContext(ctx).Raw(query, 0.01, 1, bbox.LeftTop.Lon, bbox.RightBottom.Lat, bbox.RightBottom.Lon, bbox.LeftTop.Lat, filter.StartAt, filter.EndAt).Scan(&results).Error
	if err != nil {
		r.log.Error("failed to get marks in cluster", zap.Error(err))
		return nil, err
	}
	clusters := make([]*model.Cluster, len(results))
	for i, result := range results {
		clusters[i] = &model.Cluster{
			Center: types.Point{
				Point: orb.Point{result.CenterLon, result.CenterLat},
			},
			Count: result.Count,
		}
	}
	return clusters, nil
}
