package http

// Map универсальный сборщик массива объектов
// Принимает сами объекты и функцию сборки для одного экземпляра
func Map[S, T any](src []S, conv func(S) T) []T {
	if src == nil {
		return []T{}
	}
	out := make([]T, len(src))
	for i := range src {
		out[i] = conv(src[i])
	}
	return out
}
