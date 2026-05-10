package dto

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
)

type SummaryProfileStat struct {
	MarkCount        float64 `json:"markCount"`
	FriendsCount     float64 `json:"friendsCount"`
	SubscribersCount float64 `json:"subscribersCount"`
}

func NewSummaryProfileStat(marks, friends, subs int64) SummaryProfileStat {
	return SummaryProfileStat{
		MarkCount:        utils.Threshold(marks),
		FriendsCount:     utils.Threshold(friends),
		SubscribersCount: utils.Threshold(subs),
	}
}

type MonthActivity struct {
	Month      string `json:"month"`
	ShortMonth string `json:"shortMonth"`
	Count      int64  `json:"count"`
}

func NewMonthActivity(data *mark.MonthlyActivity) MonthActivity {
	if data == nil {
		return MonthActivity{}
	}
	return MonthActivity{
		Month:      data.Month,
		ShortMonth: utils.SliceString(data.Month, 3),
		Count:      data.Count,
	}
}

func NewMultipleMonthlyActivity(data []*mark.MonthlyActivity) []MonthActivity {
	if len(data) == 0 {
		return []MonthActivity{}
	}
	res := make([]MonthActivity, 0, len(data))
	for _, m := range data {
		res = append(res, NewMonthActivity(m))
	}
	return res
}

type HeatmapItem struct {
	Day   time.Time `json:"day"`
	Count int64     `json:"count"`
}

func NewHeatmapItem(item *mark.HeatMapItem) HeatmapItem {
	if item == nil {
		return HeatmapItem{}
	}
	return HeatmapItem{
		Day:   item.Day,
		Count: item.Count,
	}
}

type Range struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func NewRange(start, end time.Time) Range {
	return Range{
		Start: start,
		End:   end,
	}
}

type HeatmapResponse struct {
	Items []HeatmapItem `json:"items"`
	Range Range         `json:"range"`
}

func NewHeatmapResponse(items []*mark.HeatMapItem, start, end time.Time) HeatmapResponse {
	res := make([]HeatmapItem, 0, len(items))
	for _, item := range items {
		res = append(res, NewHeatmapItem(item))
	}
	return HeatmapResponse{
		Items: res,
		Range: NewRange(start, end),
	}
}

type PopularCategoryResponse struct {
	CategoryName string  `json:"categoryName"`
	Count        int64   `json:"count"`
	Percent      float64 `json:"percent"`
}

func NewPopularCategoryResponse(item *mark.PopularCategory) PopularCategoryResponse {
	if item == nil {
		return PopularCategoryResponse{}
	}
	return PopularCategoryResponse{
		CategoryName: item.CategoryName,
		Count:        item.Count,
		Percent:      item.Percent,
	}
}

func NewMultiplePopularCategoryResponse(items []*mark.PopularCategory) []PopularCategoryResponse {
	if len(items) == 0 {
		return []PopularCategoryResponse{}
	}
	res := make([]PopularCategoryResponse, 0, len(items))
	for _, item := range items {
		res = append(res, NewPopularCategoryResponse(item))
	}
	return res
}

type DateRangeParam struct {
	Start time.Time `json:"start" form:"start" query:"start" binding:"required"`
	End   time.Time `json:"end" form:"end" query:"end"`
}

func (p *DateRangeParam) Defaults() {
	if p.End.IsZero() {
		p.End = time.Now()
	}
}
