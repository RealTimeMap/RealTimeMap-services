package mark_action

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
)

type MapViewMode string

const (
	ModeCluster MapViewMode = "cluster"
	ModeMarks   MapViewMode = "marks"
)

type ClusterResult struct {
	Center types.Point
	Count  int
}

func toClusterResult(clusters []*mark.Cluster) []ClusterResult {
	results := make([]ClusterResult, 0, len(clusters))
	for _, cluster := range clusters {
		results = append(results, ClusterResult{
			Center: cluster.Center,
			Count:  cluster.Count,
		})
	}
	return results
}

type UserResult struct {
	ID       uint
	Username string
	Avatar   string
	Tag      string
}

type CategoryResult struct {
	ID           uint
	CategoryName string
	Color        string
	Icon         string
}

func toCategoryResult(obj *category.Category) CategoryResult {
	if obj == nil {
		return CategoryResult{}
	}
	return CategoryResult{
		ID:           obj.ID,
		CategoryName: obj.CategoryName,
		Color:        obj.Color,
		Icon:         obj.Icon,
	}
}

type DateResult struct {
	StartAt         time.Time
	EndAt           time.Time
	ProgressPercent float64
	DaysPassed      int
	DaysLeft        int
}

type MetaResult struct {
	Status   string
	MarkType string
}

type DetailMarkResult struct {
	ID             uint
	MarkName       string
	AdditionalInfo *string
	StartAt        time.Time
	EndAt          time.Time
	Photos         []string
	Geom           types.Point
	Owner          UserResult
	Category       CategoryResult
	Type           string
	Meta           MetaResult
	Date           DateResult
}

func toDetailMarkResult(m *mark.Mark, owner UserResult) DetailMarkResult {
	return DetailMarkResult{
		ID:             m.ID,
		MarkName:       m.MarkName,
		AdditionalInfo: m.AdditionalInfo,
		Geom:           m.Geom,
		Photos:         m.GetMarkPhotos(),
		Category: CategoryResult{
			ID:           m.Category.ID,
			CategoryName: m.Category.CategoryName,
			Color:        m.Category.Color,
			Icon:         m.Category.Icon,
		},
		Owner: owner,
		Date: DateResult{
			StartAt:         m.StartAt,
			EndAt:           m.EndAt,
			ProgressPercent: m.ProgressPercent(),
			DaysPassed:      m.DaysSinceStart(),
			DaysLeft:        m.DaysLeft(),
		},
		Meta: MetaResult{
			Status:   m.Status(),
			MarkType: string(m.GetMarkType()),
		},
	}
}

type MarkResult struct {
	ID       uint
	MarkName string
	Geom     types.Point
	Photos   []string
	Total    int64
}

func toMarkResult(mark *mark.Mark) MarkResult {
	if mark == nil {
		return MarkResult{}
	}

	photos := make([]string, 0, len(mark.Photos))
	for _, photo := range mark.Photos {
		photos = append(photos, photo.URL)
	}

	return MarkResult{
		ID:       uint(mark.ID),
		MarkName: mark.MarkName,
		Geom:     mark.Geom,
		Photos:   photos,
	}
}

func toMultiMarkResult(marks []*mark.Mark) []MarkResult {
	results := make([]MarkResult, 0, len(marks))
	for _, item := range marks {
		results = append(results, toMarkResult(item))
	}
	return results
}

type MapViewResult struct {
	Mode     MapViewMode
	Clusters []ClusterResult
	Marks    []MarkResult
}
