package utils

func SliceString(src string, limit int) string {
	tmp := []rune(src)

	if len(tmp) > limit {
		return string(tmp[:limit])
	}
	return string(tmp)
}
