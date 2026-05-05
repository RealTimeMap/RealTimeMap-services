package dto

type SummaryProfileStat struct {
	MarkCount        int64 `json:"markCount"`
	FriendsCount     int64 `json:"friendsCount"`
	SubscribersCount int64 `json:"subscribersCount"`
}

func NewSummaryProfileStat(marks, friends, subs int64) SummaryProfileStat {
	return SummaryProfileStat{
		MarkCount:        marks,
		FriendsCount:     friends,
		SubscribersCount: subs,
	}
}
