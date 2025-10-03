package utils

// Map applies a function to each element in a slice and returns a new slice with the results.
// A is the input slice element type, B is the output slice element type.
func Map[T any, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}