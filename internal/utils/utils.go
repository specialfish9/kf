package utils

func MapFromSlice[T comparable, E any](source []E, keyExtractor func(E) T) map[T]E {
	m := make(map[T]E, len(source))
	for _, val := range source {
		m[keyExtractor(val)] = val
	}
	return m
}

func Map[T any, U any](slice []T, functor func(T) U) []U {
	result := make([]U, 0, len(slice))
	for _, v := range slice {
		result = append(result, functor(v))
	}
	return result
}
