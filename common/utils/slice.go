package utils

func RemoveIndex[T any](s []T, index int) []T {
	return append(s[:index], s[index+1:]...)
}
