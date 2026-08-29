package comment_stat

// StatResult — статистика комментариев пользователя за период.
type StatResult struct {
	Current  int64
	Previous int64
}

func toStatResult(current, previous int64) StatResult {
	return StatResult{Current: current, Previous: previous}
}
