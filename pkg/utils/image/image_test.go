package image_test

import (
	"bytes"
	stdimage "image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pkgimage "github.com/PhantomX7/athleton/pkg/utils/image"
)

// noisyImage renders a deterministic noisy RGBA image. Pixel values vary per
// coordinate so encoders cannot collapse the image.
func noisyImage(width, height int) *stdimage.RGBA {
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test data, not crypto
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rng.Intn(256)),
				G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)),
				A: 255,
			})
		}
	}
	return img
}

// noisyPNG encodes a deterministic noisy PNG (same pattern as libs/s3 tests).
func noisyPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, noisyImage(width, height)))
	return buf.Bytes()
}

// noisyJPEG encodes a deterministic noisy JPEG.
func noisyJPEG(t *testing.T, width, height int) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, noisyImage(width, height), &jpeg.Options{Quality: 95}))
	return buf.Bytes()
}

func TestNewImageCompressorClampsQuality(t *testing.T) {
	t.Parallel()

	require.Equal(t, 75, pkgimage.NewImageCompressor(10).Quality)
	require.Equal(t, 75, pkgimage.NewImageCompressor(75).Quality)
	require.Equal(t, 90, pkgimage.NewImageCompressor(90).Quality)
	require.Equal(t, 100, pkgimage.NewImageCompressor(100).Quality)
	require.Equal(t, 100, pkgimage.NewImageCompressor(150).Quality)

	ic := pkgimage.NewImageCompressor(85)
	require.Equal(t, 2560, ic.MaxWidth)
	require.Equal(t, 2560, ic.MaxHeight)
}

func TestCompressImagePNGProducesWebP(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)
	original := noisyPNG(t, 400, 400)

	result, err := ic.CompressImage(bytes.NewReader(original), "photo.png")
	require.NoError(t, err)
	require.NotNil(t, result)

	// PNG input always takes the WebP path.
	require.Equal(t, "webp", result.Format)
	require.Equal(t, "image/webp", result.ContentType)
	require.Equal(t, 400, result.Width)
	require.Equal(t, 400, result.Height)
	require.Greater(t, result.Size, int64(0))
	require.Equal(t, int64(result.Data.Len()), result.Size)

	// Output must actually be WebP: RIFF....WEBP container header.
	data := result.Data.Bytes()
	require.GreaterOrEqual(t, len(data), 12)
	require.Equal(t, "RIFF", string(data[0:4]))
	require.Equal(t, "WEBP", string(data[8:12]))
}

func TestCompressImageLargeJPEGProducesJPEG(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)
	// 800x800 = 640k pixels: above the small-image and low-color-count
	// thresholds, opaque, so the photo path (JPEG) is taken.
	original := noisyJPEG(t, 800, 800)

	result, err := ic.CompressImage(bytes.NewReader(original), "photo.jpg")
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, "jpeg", result.Format)
	require.Equal(t, "image/jpeg", result.ContentType)
	require.Equal(t, 800, result.Width)
	require.Equal(t, 800, result.Height)
	require.Greater(t, result.Size, int64(0))
	require.Equal(t, int64(result.Data.Len()), result.Size)

	// The output must decode as a valid JPEG with the same dimensions.
	decoded, format, err := stdimage.Decode(bytes.NewReader(result.Data.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, 800, decoded.Bounds().Dx())
	require.Equal(t, 800, decoded.Bounds().Dy())
}

func TestCompressImageResizesOversizedImages(t *testing.T) {
	t.Parallel()

	// Exported fields allow shrinking the limits so the test stays fast.
	ic := &pkgimage.ImageCompressor{Quality: 85, MaxWidth: 100, MaxHeight: 100}
	original := noisyPNG(t, 400, 200)

	result, err := ic.CompressImage(bytes.NewReader(original), "wide.png")
	require.NoError(t, err)

	// imaging.Fit preserves aspect ratio within the 100x100 box.
	require.Equal(t, 100, result.Width)
	require.Equal(t, 50, result.Height)
	require.Equal(t, "webp", result.Format)
}

func TestCompressImageRejectsNonImageInput(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)

	result, err := ic.CompressImage(strings.NewReader("definitely not an image"), "junk.png")
	require.Nil(t, result)
	require.ErrorContains(t, err, "failed to decode image")
}

func TestCreateThumbnailSmallUsesWebP(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)
	original := noisyPNG(t, 400, 400)

	result, err := ic.CreateThumbnail(bytes.NewReader(original), 100, 100)
	require.NoError(t, err)

	require.Equal(t, "webp", result.Format)
	require.Equal(t, "image/webp", result.ContentType)
	require.Equal(t, 100, result.Width)
	require.Equal(t, 100, result.Height)
	require.Greater(t, result.Size, int64(0))
	require.Equal(t, int64(result.Data.Len()), result.Size)
}

func TestCreateThumbnailLargeFallsBackToJPEG(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)
	original := noisyPNG(t, 800, 800)

	result, err := ic.CreateThumbnail(bytes.NewReader(original), 400, 400)
	require.NoError(t, err)

	require.Equal(t, "jpeg", result.Format)
	require.Equal(t, "image/jpeg", result.ContentType)
	require.Equal(t, 400, result.Width)
	require.Equal(t, 400, result.Height)
	require.Greater(t, result.Size, int64(0))
}

func TestCreateThumbnailRejectsNonImageInput(t *testing.T) {
	t.Parallel()

	ic := pkgimage.NewImageCompressor(85)

	result, err := ic.CreateThumbnail(strings.NewReader("not an image"), 100, 100)
	require.Nil(t, result)
	require.Error(t, err)
}
