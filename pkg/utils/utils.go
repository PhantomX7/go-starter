package utils

import (
	"fmt"

	"github.com/PhantomX7/athleton/pkg/config"
)

// Map applies a function to each element in a slice and returns a new slice with the results.
// A is the input slice element type, B is the output slice element type.
func Map[T any, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// Generate s3 public URL
func GenerateS3PublicURL(key string) string {
	if config.Get().S3.CdnURL != "" {
		return fmt.Sprintf("%s/%s", config.Get().S3.CdnURL, key)
	}
	if config.Get().S3.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", config.Get().S3.Endpoint, config.Get().S3.Bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		config.Get().S3.Bucket,
		config.Get().S3.Region,
		key,
	)
}
