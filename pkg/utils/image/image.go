// image/compressor.go
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
)

type ImageCompressor struct {
	Quality   int
	MaxWidth  int
	MaxHeight int
}

type CompressedImage struct {
	Data        *bytes.Buffer
	ContentType string
	Size        int64
	Format      string
	Width       int
	Height      int
}

func NewImageCompressor(quality int) *ImageCompressor {
	if quality < 75 {
		quality = 75
	}
	if quality > 100 {
		quality = 100
	}

	return &ImageCompressor{
		Quality:   quality,
		MaxWidth:  2560,
		MaxHeight: 2560,
	}
}

// CompressImage intelligently compresses images:
// - Lossless WebP for: PNG, graphics, transparent images, small images
// - JPEG for: photos and complex images
func (ic *ImageCompressor) CompressImage(reader io.Reader, filename string) (*CompressedImage, error) {
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if needed
	img = ic.resizeIfNeeded(img)
	bounds := img.Bounds()

	// Decide format: WebP or JPEG
	useWebP := ic.shouldUseWebP(img, format)

	buf := new(bytes.Buffer)
	var contentType, outputFormat string

	if useWebP {
		if err = nativewebp.Encode(buf, img, nil); err != nil {
			return nil, fmt.Errorf("failed to encode to WebP: %w", err)
		}
		contentType = "image/webp"
		outputFormat = "webp"
	} else {
		quality := ic.Quality
		if quality < 85 {
			quality = 85
		}
		if err = jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("failed to encode to JPEG: %w", err)
		}
		contentType = "image/jpeg"
		outputFormat = "jpeg"
	}

	return &CompressedImage{
		Data:        buf,
		ContentType: contentType,
		Size:        int64(buf.Len()),
		Format:      outputFormat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	}, nil
}

// resizeIfNeeded resizes image if it exceeds max dimensions
func (ic *ImageCompressor) resizeIfNeeded(img image.Image) image.Image {
	bounds := img.Bounds()
	if bounds.Dx() > ic.MaxWidth || bounds.Dy() > ic.MaxHeight {
		return imaging.Fit(img, ic.MaxWidth, ic.MaxHeight, imaging.Lanczos)
	}
	return img
}

// shouldUseWebP determines format based on image characteristics
func (ic *ImageCompressor) shouldUseWebP(img image.Image, format string) bool {
	// Always use WebP for PNG (usually graphics)
	if strings.EqualFold(format, "png") {
		return true
	}

	bounds := img.Bounds()
	pixels := bounds.Dx() * bounds.Dy()

	// Use WebP for small images (likely icons/graphics)
	if pixels < 150000 { // ~387x387
		return true
	}

	// Check for transparency
	if ic.hasTransparency(img) {
		return true
	}

	// Check color complexity (graphics have fewer colors)
	if pixels < 500000 && ic.hasLowColorCount(img, 2048) {
		return true
	}

	// Default to JPEG for photos
	return false
}

// hasTransparency checks if image has actual transparency
func (ic *ImageCompressor) hasTransparency(img image.Image) bool {
	// Only check alpha-capable formats
	switch img.(type) {
	case *image.RGBA, *image.RGBA64, *image.NRGBA, *image.NRGBA64:
		bounds := img.Bounds()
		step := max(bounds.Dx()/20, bounds.Dy()/20, 1)

		for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
			for x := bounds.Min.X; x < bounds.Max.X; x += step {
				_, _, _, a := img.At(x, y).RGBA()
				if a < 65535 {
					return true // Found transparency
				}
			}
		}
	}
	return false
}

// hasLowColorCount checks if image has limited colors (indicates graphic)
func (ic *ImageCompressor) hasLowColorCount(img image.Image, threshold int) bool {
	bounds := img.Bounds()
	colorSet := make(map[uint32]struct{})

	step := max(bounds.Dx()/50, bounds.Dy()/50, 1)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, a := img.At(x, y).RGBA()

			// Combine RGBA into single uint32 for faster comparison
			color := (r&0xFF00)<<16 | (g&0xFF00)<<8 | (b & 0xFF00) | (a >> 8)
			colorSet[color] = struct{}{}

			// Early exit if too many colors
			if len(colorSet) > threshold {
				return false
			}
		}
	}

	return true
}

// CreateThumbnail creates a thumbnail (WebP for small, JPEG for larger)
func (ic *ImageCompressor) CreateThumbnail(reader io.Reader, width, height int) (*CompressedImage, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	thumbnail := imaging.Thumbnail(img, width, height, imaging.Lanczos)
	bounds := thumbnail.Bounds()
	buf := new(bytes.Buffer)

	// Try WebP for small thumbnails (better compression)
	var contentType, format string
	if width <= 300 && height <= 300 {
		if err = nativewebp.Encode(buf, thumbnail, nil); err == nil {
			contentType = "image/webp"
			format = "webp"
		}
	}

	// Fallback to JPEG
	if format == "" {
		if err = jpeg.Encode(buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
		contentType = "image/jpeg"
		format = "jpeg"
	}

	return &CompressedImage{
		Data:        buf,
		ContentType: contentType,
		Size:        int64(buf.Len()),
		Format:      format,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	}, nil
}

// Helper function for max
func max(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
