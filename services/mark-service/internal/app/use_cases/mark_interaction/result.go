package mark_interaction

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
)

type ShareResult struct {
	Count int64
}

func toShareResult(count int64) ShareResult {
	return ShareResult{
		Count: count,
	}
}

// ReactionResult — состояние реакций метки для фронтенда.
type ReactionResult struct {
	Count   int64
	IsLiked bool
	CanLike bool
}

func toReactionResult(state mark.ReactionState) ReactionResult {
	return ReactionResult{
		Count:   state.Count,
		IsLiked: state.IsLiked,
		CanLike: !state.IsLiked,
	}
}

// MarkStatResult — статистика метки для клиента.
// Likes/Shares — форматированные строки (utils.FormatNumber) для отображения больших чисел.
type MarkStatResult struct {
	Likes   string
	Shares  string
	IsLiked bool
	CanLike bool
}

func toMarkStatResult(stat mark.MarkStat) MarkStatResult {
	return MarkStatResult{
		Likes:   utils.FormatNumber(stat.Likes),
		Shares:  utils.FormatNumber(stat.Shares),
		IsLiked: stat.IsLiked,
		CanLike: !stat.IsLiked,
	}
}
