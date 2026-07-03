package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
)

// testS3Config is the S3 configuration every test client is built with.
func testS3Config() config.S3Config {
	return config.S3Config{
		Bucket:    "test-bucket",
		Region:    "us-east-1",
		Endpoint:  "http://storage.local",
		UploadACL: "public-read",
	}
}

func setTestEnv(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() { logger.Log = prev })
}

// newTestClient builds an s3Client whose SDK client talks to the given
// handler instead of AWS. Path-style addressing keeps the fake bucket out of
// the host name, and a single attempt keeps failure tests fast.
func newTestClient(t *testing.T, handler http.Handler) *s3Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	awsCfg := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
		BaseEndpoint:     aws.String(srv.URL),
		RetryMaxAttempts: 1,
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) { o.UsePathStyle = true })

	return &s3Client{
		client:       client,
		uploader:     transfermanager.New(client),
		clientConfig: awsCfg,
		s3Cfg:        testS3Config(),
	}
}

// makeFileHeader builds a real *multipart.FileHeader whose Open() serves the
// given content, the same way Gin produces one from a form upload.
func makeFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	form, err := multipart.NewReader(&buf, w.Boundary()).ReadForm(int64(len(content)) + 1<<20)
	require.NoError(t, err)
	t.Cleanup(func() { _ = form.RemoveAll() })

	files := form.File["file"]
	require.Len(t, files, 1)
	return files[0]
}

// noisyPNG renders a PNG large enough (>200KB) to take the compression path.
// Pixel values vary per coordinate so the encoder cannot collapse the image.
func noisyPNG(t *testing.T) []byte {
	t.Helper()

	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test data, not crypto
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rng.Intn(256)),
				G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	require.Greater(t, buf.Len(), 200*1024, "test image must exceed the compression threshold")
	return buf.Bytes()
}

func TestGenerateS3Key(t *testing.T) {
	c := &s3Client{}

	key := c.generateS3Key("photo.png", "avatars")
	require.True(t, strings.HasPrefix(key, "avatars/"))
	require.True(t, strings.HasSuffix(key, ".png"))

	key = c.generateS3Key("doc.pdf", "")
	require.False(t, strings.HasPrefix(key, "/"))
	require.True(t, strings.HasSuffix(key, ".pdf"))
}

func TestGenerateS3KeyWithExtension(t *testing.T) {
	c := &s3Client{}

	require.True(t, strings.HasSuffix(c.generateS3KeyWithExtension("img", "webp"), ".webp"))
	// jpeg maps to the conventional .jpg extension
	require.True(t, strings.HasSuffix(c.generateS3KeyWithExtension("img", "jpeg"), ".jpg"))
	require.True(t, strings.HasPrefix(c.generateS3KeyWithExtension("img", "webp"), "img/"))
}

func TestIsImageFile(t *testing.T) {
	c := &s3Client{}

	for _, name := range []string{"a.jpg", "b.JPEG", "c.png", "d.webp", "e.gif"} {
		require.True(t, c.isImageFile(name), name)
	}
	for _, name := range []string{"a.pdf", "b.txt", "noext", "c.svg"} {
		require.False(t, c.isImageFile(name), name)
	}
}

func TestCalculateOptimalQuality(t *testing.T) {
	c := &s3Client{}

	require.Equal(t, 95, c.calculateOptimalQuality(100*1024))
	require.Equal(t, 92, c.calculateOptimalQuality(1*1024*1024))
	require.Equal(t, 90, c.calculateOptimalQuality(3*1024*1024))
	require.Equal(t, 88, c.calculateOptimalQuality(10*1024*1024))
}

func TestShouldCompressImage(t *testing.T) {
	c := &s3Client{}

	small := makeFileHeader(t, "small.jpg", bytes.Repeat([]byte("x"), 1024))
	require.False(t, c.shouldCompressImage(small))

	big := makeFileHeader(t, "big.jpg", bytes.Repeat([]byte("x"), 300*1024))
	require.True(t, c.shouldCompressImage(big))

	// Small-ish WebP is already compressed — leave it alone.
	webp := makeFileHeader(t, "img.webp", bytes.Repeat([]byte("x"), 300*1024))
	require.False(t, c.shouldCompressImage(webp))

	hugeWebp := makeFileHeader(t, "img.webp", bytes.Repeat([]byte("x"), 2*1024*1024))
	require.True(t, c.shouldCompressImage(hugeWebp))
}

func TestDetectContentType(t *testing.T) {
	c := &s3Client{}

	fh := makeFileHeader(t, "img.png", noisyPNG(t))
	contentType, err := c.detectContentType(fh)
	require.NoError(t, err)
	require.Equal(t, "image/png", contentType)
}

func TestUploadACL(t *testing.T) {
	setTestEnv(t)

	cases := []struct {
		configured string
		want       tmtypes.ObjectCannedACL
	}{
		{"public-read", tmtypes.ObjectCannedACLPublicRead},
		{"private", tmtypes.ObjectCannedACLPrivate},
		{"authenticated-read", tmtypes.ObjectCannedACLAuthenticatedRead},
		// Unknown or unset values must fall back to private, never public.
		{"public-read-writ", tmtypes.ObjectCannedACLPrivate},
		{"", tmtypes.ObjectCannedACLPrivate},
	}

	for _, tc := range cases {
		c := &s3Client{s3Cfg: config.S3Config{UploadACL: tc.configured}}
		require.Equal(t, tc.want, c.uploadACL(), "configured ACL %q", tc.configured)
	}
}

func TestSanitizeMetadataValue(t *testing.T) {
	// Plain ASCII passes through untouched.
	require.Equal(t, "photo (1).jpg", sanitizeMetadataValue("photo (1).jpg"))
	// Non-ASCII is percent-encoded so S3 metadata stays ASCII-only.
	require.Equal(t, "caf%C3%A9.png", sanitizeMetadataValue("café.png"))
	// Control characters are encoded too.
	require.NotContains(t, sanitizeMetadataValue("a\nb.txt"), "\n")
}

func TestUploadImageWithoutCompression(t *testing.T) {
	setTestEnv(t)

	var gotPath, gotACL, gotFilename atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		gotPath.Store(r.URL.Path)
		gotACL.Store(r.Header.Get("x-amz-acl"))
		gotFilename.Store(r.Header.Get("x-amz-meta-original-filename"))
		w.Header().Set("ETag", `"test-etag"`)
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(t, handler)

	content := []byte("plain text payload")
	fh := makeFileHeader(t, "notes.txt", content)

	result, err := c.UploadImage(context.Background(), fh, "docs")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, strings.HasPrefix(result.Key, "docs/"))
	require.True(t, strings.HasSuffix(result.Key, ".txt"))
	require.Equal(t, "test-bucket", result.Bucket)
	require.Equal(t, `"test-etag"`, result.ETag)
	require.Equal(t, int64(len(content)), result.Size)
	require.Equal(t, "txt", result.Format)
	require.Equal(t, "http://storage.local/test-bucket/"+result.Key, result.URL)

	require.Equal(t, "/test-bucket/"+result.Key, gotPath.Load())
	require.Equal(t, "public-read", gotACL.Load())
	require.Equal(t, "notes.txt", gotFilename.Load())
}

func TestUploadImageCompressesLargeImages(t *testing.T) {
	setTestEnv(t)

	var uploadedBytes atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		uploadedBytes.Add(n)
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(t, handler)

	original := noisyPNG(t)
	fh := makeFileHeader(t, "photo.png", original)

	result, err := c.UploadImage(context.Background(), fh, "photos")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Format)
	require.True(t, strings.HasPrefix(result.Key, "photos/"))
	require.Greater(t, result.Size, int64(0))
	require.Greater(t, uploadedBytes.Load(), int64(0))
}

func TestUploadImageReturnsServerError(t *testing.T) {
	setTestEnv(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	c := newTestClient(t, handler)

	fh := makeFileHeader(t, "notes.txt", []byte("payload"))

	result, err := c.UploadImage(context.Background(), fh, "docs")

	require.Nil(t, result)
	require.ErrorContains(t, err, "failed to upload to S3")
}

func TestDeleteImageSkipsEmptyKey(t *testing.T) {
	setTestEnv(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent for an empty key")
	})
	c := newTestClient(t, handler)

	require.NoError(t, c.DeleteImage(context.Background(), ""))
}

func TestDeleteImageSuccess(t *testing.T) {
	setTestEnv(t)

	var gotPath atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		gotPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	c := newTestClient(t, handler)

	require.NoError(t, c.DeleteImage(context.Background(), "photos/a.png"))
	require.Equal(t, "/test-bucket/photos/a.png", gotPath.Load())
}

func TestDeleteImageReturnsServerError(t *testing.T) {
	setTestEnv(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	c := newTestClient(t, handler)

	err := c.DeleteImage(context.Background(), "photos/a.png")
	require.ErrorContains(t, err, "failed to delete image")
}

type deleteResult struct {
	XMLName xml.Name      `xml:"DeleteResult"`
	Deleted []deletedItem `xml:"Deleted"`
	Errors  []deleteError `xml:"Error"`
}

type deletedItem struct {
	Key string `xml:"Key"`
}

type deleteError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func TestDeleteImagesFiltersAndDeletes(t *testing.T) {
	setTestEnv(t)

	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/xml")
		out := deleteResult{
			Deleted: []deletedItem{{Key: "a.png"}},
			// A missing key in the batch is logged but must not fail the call.
			Errors: []deleteError{{Key: "b.png", Code: "NoSuchKey", Message: "gone"}},
		}
		body, err := xml.Marshal(out)
		require.NoError(t, err)
		_, _ = w.Write(body)
	})
	c := newTestClient(t, handler)

	err := c.DeleteImages(context.Background(), []string{"a.png", "", "example.jpg", "b.png"})

	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
}

func TestDeleteImagesNoValidKeys(t *testing.T) {
	setTestEnv(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent when every key is filtered out")
	})
	c := newTestClient(t, handler)

	require.NoError(t, c.DeleteImages(context.Background(), nil))
	require.NoError(t, c.DeleteImages(context.Background(), []string{"", ""}))
}

func TestUploadImagesParallel(t *testing.T) {
	setTestEnv(t)

	var uploads atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploads.Add(1)
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(t, handler)

	files := []*multipart.FileHeader{
		makeFileHeader(t, "a.txt", []byte("first")),
		makeFileHeader(t, "b.txt", []byte("second")),
		makeFileHeader(t, "c.txt", []byte("third")),
	}

	results, err := c.UploadImagesParallel(context.Background(), files, "docs", 2)

	require.NoError(t, err)
	require.Len(t, results, 3)
	for _, r := range results {
		require.NotNil(t, r)
		require.True(t, strings.HasPrefix(r.Key, "docs/"))
	}
	require.Equal(t, int32(3), uploads.Load())
}

func TestUploadImagesParallelEmptyInput(t *testing.T) {
	setTestEnv(t)

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected")
	}))

	results, err := c.UploadImagesParallel(context.Background(), nil, "docs", 2)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestUploadImagesParallelPropagatesFailures(t *testing.T) {
	setTestEnv(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	c := newTestClient(t, handler)

	files := []*multipart.FileHeader{
		makeFileHeader(t, "a.txt", []byte("first")),
		makeFileHeader(t, "b.txt", []byte("second")),
	}

	results, err := c.UploadImagesParallel(context.Background(), files, "docs", 2)
	require.ErrorContains(t, err, "upload failed for 2 files")
	require.Nil(t, results, "a failed batch must not hand back partial results")
}

func TestUploadImagesParallelCleansUpPartialUploads(t *testing.T) {
	setTestEnv(t)

	var uploadedKey, deleteBody atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			// Fail only the second file so the first one gets orphaned.
			if r.Header.Get("x-amz-meta-original-filename") == "b.txt" {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			uploadedKey.Store(strings.TrimPrefix(r.URL.Path, "/test-bucket/"))
			w.Header().Set("ETag", `"etag"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			// DeleteObjects (batch delete) for the rollback.
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			deleteBody.Store(string(body))

			w.Header().Set("Content-Type", "application/xml")
			out, err := xml.Marshal(deleteResult{Deleted: []deletedItem{{Key: "cleaned"}}})
			require.NoError(t, err)
			_, _ = w.Write(out)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	c := newTestClient(t, handler)

	files := []*multipart.FileHeader{
		makeFileHeader(t, "a.txt", []byte("first")),
		makeFileHeader(t, "b.txt", []byte("second")),
	}

	results, err := c.UploadImagesParallel(context.Background(), files, "docs", 2)

	require.ErrorContains(t, err, "upload failed for 1 files")
	require.Nil(t, results)

	key, ok := uploadedKey.Load().(string)
	require.True(t, ok, "the first file should have been uploaded")
	body, ok := deleteBody.Load().(string)
	require.True(t, ok, "the orphaned object should have been deleted")
	require.Contains(t, body, key, "cleanup must target the successfully uploaded key")
}

func TestUploadImageSanitizesNonASCIIFilenameMetadata(t *testing.T) {
	setTestEnv(t)

	var gotFilename atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilename.Store(r.Header.Get("x-amz-meta-original-filename"))
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	})
	c := newTestClient(t, handler)

	fh := makeFileHeader(t, "café photo.txt", []byte("payload"))

	_, err := c.UploadImage(context.Background(), fh, "docs")

	require.NoError(t, err)
	require.Equal(t, "caf%C3%A9%20photo.txt", gotFilename.Load())
}
