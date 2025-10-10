package utils

// SafeFetch safely fetches data. If v = nil, a fallback value is returned.
func SafeFetch[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}
	return *v
}
