package dto

import (
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
