package util

func MapIsEmpty[k comparable, v any](m map[k]v) bool {
	return len(m) == 0
}

func KeyExistsInMap[K comparable, V any](key K, m map[K]V) bool {
	_, ok := m[key]
	return ok
}


