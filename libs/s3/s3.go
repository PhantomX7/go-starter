// Package s3 provides the application's S3-compatible object-storage integration.
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"
	"github.com/PhantomX7/athleton/pkg/utils/image"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// Client exposes the S3 operations used by the application.
type Client interface {
	UploadImage(ctx context.Context, file *multipart.FileHeader, folder string) (*S3UploadResult, error)
	UploadImagesParallel(ctx context.Context, files []*multipart.FileHeader, folder string, maxConcurrency int) ([]*S3UploadResult, error)
	DeleteImage(ctx context.Context, key string) error
	DeleteImages(ctx context.Context, keys []string) error
}

type s3Client struct {
	client       *s3.Client
	uploader     *manager.Uploader
	clientConfig aws.Config
}

// S3UploadResult contains the metadata returned after a successful upload.
type S3UploadResult struct {
	Key      string `json:"key"`
	URL      string `json:"url"`
	Location string `json:"location"`
	Bucket   string `json:"bucket"`
	ETag     string `json:"etag"`
	Size     int64  `json:"size"`
	Format   string `json:"format"` // Added: webp or jpeg
}

// NewS3Client constructs the shared S3 client and uploader.
func NewS3Client() (Client, error) {
	awsConfig, err := s3config.LoadDefaultConfig(context.TODO(),
		s3config.WithRegion(config.Get().S3.Region),
		s3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.Get().S3.AccessKeyID,
			config.Get().S3.SecretAccessKey,
			"",
		)),
		s3config.WithBaseEndpoint(config.Get().S3.Endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig)
	uploader := manager.NewUploader(client)

	logger.Info("S3 client initialized successfully",
		zap.String("region", config.Get().S3.Region),
		zap.String("bucket", config.Get().S3.Bucket),
	)

	return &s3Client{
		client:       client,
		uploader:     uploader,
		clientConfig: awsConfig,
	}, nil
}

// UploadImage uploads and compresses images intelligently
func (s3c *s3Client) UploadImage(ctx context.Context, file *multipart.FileHeader, folder string) (*S3UploadResult, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Starting image upload",
		zap.String("request_id", requestID),
		zap.String("filename", file.Filename),
		zap.Int64("size", file.Size),
		zap.String("folder", folder),
	)

	src, err := file.Open()
	if err != nil {
		logger.Error("Failed to open file",
			zap.String("request_id", requestID),
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = src.Close()
	}()

	var uploadBody io.Reader
	var contentType string
	var fileSize int64
	var outputFormat string
	var key string

	// Compress images intelligently
	if s3c.isImageFile(file.Filename) && s3c.shouldCompressImage(file) {
		quality := s3c.calculateOptimalQuality(file.Size)
		compressor := image.NewImageCompressor(quality)

		compressedImage, err := compressor.CompressImage(src, file.Filename)
		if err != nil {
			logger.Error("Failed to compress image",
				zap.String("request_id", requestID),
				zap.String("filename", file.Filename),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to compress image: %w", err)
		}

		uploadBody = bytes.NewReader(compressedImage.Data.Bytes())
		contentType = compressedImage.ContentType
		fileSize = compressedImage.Size
		outputFormat = compressedImage.Format

		// Generate key with correct extension based on output format
		key = s3c.generateS3KeyWithExtension(folder, outputFormat)

		logger.Debug("Image compressed",
			zap.String("request_id", requestID),
			zap.String("original_filename", file.Filename),
			zap.String("output_format", outputFormat),
			zap.Int64("original_size", file.Size),
			zap.Int64("compressed_size", fileSize),
			zap.Int("quality", quality),
			zap.Int("width", compressedImage.Width),
			zap.Int("height", compressedImage.Height),
			zap.Float64("compression_ratio", float64(file.Size)/float64(fileSize)),
		)
	} else {
		// Upload without compression
		uploadBody = src
		contentType, err = s3c.detectContentType(file)
		if err != nil {
			contentType = "application/octet-stream"
		}
		fileSize = file.Size
		outputFormat = strings.TrimPrefix(filepath.Ext(file.Filename), ".")
		key = s3c.generateS3Key(file.Filename, folder)

		logger.Debug("Uploading without compression",
			zap.String("request_id", requestID),
			zap.String("filename", file.Filename),
			zap.Int64("size", fileSize),
		)
	}

	// Upload to S3
	result, err := s3c.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(config.Get().S3.Bucket),
		Key:         aws.String(key),
		Body:        uploadBody,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
		Metadata: map[string]string{
			"original-filename": file.Filename,
			"original-size":     fmt.Sprintf("%d", file.Size),
			"output-format":     outputFormat,
			"compressed-size":   fmt.Sprintf("%d", fileSize),
			"uploaded-at":       time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		logger.Error("Failed to upload to S3",
			zap.String("request_id", requestID),
			zap.String("filename", file.Filename),
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	uploadResult := &S3UploadResult{
		Key:      key,
		URL:      utils.GenerateS3PublicURL(key),
		Location: result.Location,
		Bucket:   config.Get().S3.Bucket,
		ETag:     aws.ToString(result.ETag),
		Size:     fileSize,
		Format:   outputFormat,
	}

	logger.Info("Image uploaded successfully",
		zap.String("request_id", requestID),
		zap.String("original_filename", file.Filename),
		zap.String("key", key),
		zap.String("url", uploadResult.URL),
		zap.String("format", outputFormat),
		zap.Int64("size", fileSize),
	)

	return uploadResult, nil
}

// generateS3Key generates key with original extension
func (s3c *s3Client) generateS3Key(filename, folder string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	timestamp := time.Now().Format("20060102")

	if folder != "" {
		return fmt.Sprintf("%s/%s/%s%s", folder, timestamp, uniqueID, ext)
	}
	return fmt.Sprintf("%s/%s%s", timestamp, uniqueID, ext)
}

// generateS3KeyWithExtension generates key with specified extension (for format conversion)
func (s3c *s3Client) generateS3KeyWithExtension(folder, format string) string {
	uniqueID := uuid.New().String()
	timestamp := time.Now().Format("20060102")

	// Map format to extension
	ext := "." + format
	if format == "jpeg" {
		ext = ".jpg"
	}

	if folder != "" {
		return fmt.Sprintf("%s/%s/%s%s", folder, timestamp, uniqueID, ext)
	}
	return fmt.Sprintf("%s/%s%s", timestamp, uniqueID, ext)
}

// detectContentType detects content type from file
func (s3c *s3Client) detectContentType(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = src.Close()
	}()

	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return http.DetectContentType(buffer[:n]), nil
}

// isImageFile checks if file is an image
func (s3c *s3Client) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff"}
	return slices.Contains(imageExtensions, ext)
}

// calculateOptimalQuality returns quality based on file size
func (s3c *s3Client) calculateOptimalQuality(fileSize int64) int {
	if fileSize < 500*1024 { // < 500KB
		return 95
	}
	if fileSize < 2*1024*1024 { // < 2MB
		return 92
	}
	if fileSize < 5*1024*1024 { // < 5MB
		return 90
	}
	return 88
}

// shouldCompressImage determines if compression should be applied
func (s3c *s3Client) shouldCompressImage(file *multipart.FileHeader) bool {
	// Always compress images > 200KB
	if file.Size < 200*1024 {
		return false
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))

	// Skip compression for already-compressed WebP if small
	if ext == ".webp" && file.Size < 1*1024*1024 {
		return false
	}

	return true
}

// DeleteImage deletes a single image from S3
func (s3c *s3Client) DeleteImage(ctx context.Context, key string) error {
	if key == "" || key == "example.jpg" {
		return nil
	}

	requestID := utils.GetRequestIDFromContext(ctx)

	_, err := s3c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(config.Get().S3.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var noSuchKey *types.NoSuchKey
		var notFound *types.NotFound

		if errors.As(err, &noSuchKey) || errors.As(err, &notFound) {
			logger.Warn("Image not found, treating as already deleted",
				zap.String("request_id", requestID),
				zap.String("key", key),
			)
			return nil
		}

		logger.Error("Failed to delete image from S3",
			zap.String("request_id", requestID),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete image: %w", err)
	}

	logger.Info("Image deleted successfully",
		zap.String("request_id", requestID),
		zap.String("key", key),
	)

	return nil
}

// UploadImagesParallel uploads multiple images concurrently
func (s3c *s3Client) UploadImagesParallel(
	ctx context.Context,
	files []*multipart.FileHeader,
	folder string,
	maxConcurrency int,
) ([]*S3UploadResult, error) {
	if len(files) == 0 {
		return []*S3UploadResult{}, nil
	}

	requestID := utils.GetRequestIDFromContext(ctx)

	// Set default concurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	if maxConcurrency > len(files) {
		maxConcurrency = len(files)
	}

	logger.Info("Starting parallel upload",
		zap.String("request_id", requestID),
		zap.Int("file_count", len(files)),
		zap.Int("concurrency", maxConcurrency),
	)

	// Create worker pool
	pool, err := ants.NewPool(maxConcurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}
	defer pool.Release()

	var wg sync.WaitGroup
	var mu sync.Mutex

	results := make([]*S3UploadResult, len(files))
	var uploadErrors []error

	// Submit upload tasks
	for i, file := range files {
		wg.Add(1)
		idx := i
		currentFile := file

		err := pool.Submit(func() {
			defer wg.Done()

			// Check context cancellation
			if ctx.Err() != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Errorf("upload cancelled for file %d", idx))
				mu.Unlock()
				return
			}

			// Upload file
			result, err := s3c.UploadImage(ctx, currentFile, folder)

			mu.Lock()
			if err != nil {
				uploadErrors = append(uploadErrors, fmt.Errorf("file %d: %w", idx, err))
			} else {
				results[idx] = result
			}
			mu.Unlock()
		})

		if err != nil {
			wg.Done()
			mu.Lock()
			uploadErrors = append(uploadErrors, fmt.Errorf("failed to submit file %d: %w", i, err))
			mu.Unlock()
		}
	}

	wg.Wait()

	// Handle errors
	if len(uploadErrors) > 0 {
		successCount := len(files) - len(uploadErrors)
		logger.Error("Parallel upload completed with errors",
			zap.String("request_id", requestID),
			zap.Int("total", len(files)),
			zap.Int("success", successCount),
			zap.Int("failed", len(uploadErrors)),
		)
		return results, fmt.Errorf("upload failed for %d files", len(uploadErrors))
	}

	logger.Info("Parallel upload completed successfully",
		zap.String("request_id", requestID),
		zap.Int("file_count", len(files)),
	)

	return results, nil
}

// DeleteImages deletes multiple images from S3
func (s3c *s3Client) DeleteImages(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	requestID := utils.GetRequestIDFromContext(ctx)

	// Prepare delete request, filtering out "example.jpg" and empty strings
	var objects []types.ObjectIdentifier
	for _, key := range keys {
		if key == "" || key == "example.jpg" {
			continue
		}
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(key)})
	}

	// If no keys remain after filtering, return early
	if len(objects) == 0 {
		logger.Debug("No valid images to delete after filtering",
			zap.String("request_id", requestID),
		)
		return nil
	}

	// Delete objects
	output, err := s3c.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(config.Get().S3.Bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	})

	if err != nil {
		logger.Error("Failed to delete images",
			zap.String("request_id", requestID),
			zap.Int("key_count", len(objects)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete images: %w", err)
	}

	// Log results
	if len(output.Deleted) > 0 {
		logger.Info("Images deleted successfully",
			zap.String("request_id", requestID),
			zap.Int("deleted_count", len(output.Deleted)),
		)
	}

	// Log errors (but don't fail)
	for _, delErr := range output.Errors {
		code := aws.ToString(delErr.Code)
		if code == "NoSuchKey" || code == "NotFound" {
			logger.Debug("Image already deleted",
				zap.String("key", aws.ToString(delErr.Key)),
			)
		} else {
			logger.Warn("Failed to delete image",
				zap.String("key", aws.ToString(delErr.Key)),
				zap.String("code", code),
				zap.String("message", aws.ToString(delErr.Message)),
			)
		}
	}

	return nil
}
