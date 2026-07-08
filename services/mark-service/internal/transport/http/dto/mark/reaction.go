package mark

import "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_interaction"

// ReactionResponse — состояние реакций метки для клиента.
// @name ReactionResponse
type ReactionResponse struct {
	Count   int64 `json:"count"`
	IsLiked bool  `json:"isLiked"`
	CanLike bool  `json:"canLike"`
}

func NewReactionResponse(result mark_interaction.ReactionResult) ReactionResponse {
	return ReactionResponse{
		Count:   result.Count,
		IsLiked: result.IsLiked,
		CanLike: result.CanLike,
	}
}

// MarkStatResponse — агрегированная статистика метки.
// likes/shares — форматированные строки (напр. "1.2 K") для отображения.
// @name MarkStatResponse
type MarkStatResponse struct {
	Likes   string `json:"likes"`
	Shares  string `json:"shares"`
	IsLiked bool   `json:"isLiked"`
	CanLike bool   `json:"canLike"`
}

func NewMarkStatResponse(result mark_interaction.MarkStatResult) MarkStatResponse {
	return MarkStatResponse{
		Likes:   result.Likes,
		Shares:  result.Shares,
		IsLiked: result.IsLiked,
		CanLike: result.CanLike,
	}
}
