package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiskSinkWritesNestedKeys(t *testing.T) {
	dir := t.TempDir()
	data := []byte("webp bytes")

	location, err := (DiskSink{Dir: dir}).Put(context.Background(), "profile/overview.webp", data)
	require.NoError(t, err)

	expected := filepath.Join(dir, "profile", "overview.webp")
	require.Equal(t, expected, location)
	require.FileExists(t, expected)
	require.Equal(t, data, mustReadFile(t, expected))
}

func TestS3SinkPutUsesPrefixMetadataAndPublicURL(t *testing.T) {
	requests := make(chan s3PutRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests <- s3PutRequest{
			Method:       r.Method,
			Path:         r.URL.Path,
			ContentType:  r.Header.Get("Content-Type"),
			CacheControl: r.Header.Get("Cache-Control"),
			ACL:          r.Header.Get("X-Amz-Acl"),
			Body:         string(body),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("S3_BUCKET_NAME", "widgets-bucket")
	t.Setenv("S3_PREFIX", "/profiles/caian/")
	t.Setenv("S3_ENDPOINT_URL", server.URL)
	t.Setenv("S3_PUBLIC_BASE_URL", "https://cdn.example.test/widgets/")
	t.Setenv("S3_ACL", "public-read")

	sink, err := NewS3Sink(context.Background())
	require.NoError(t, err)

	location, err := sink.Put(context.Background(), "overview.webp", []byte("webp payload"))
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.test/widgets/profiles/caian/overview.webp", location)

	req := <-requests
	require.Equal(t, http.MethodPut, req.Method)
	require.Equal(t, "/widgets-bucket/profiles/caian/overview.webp", req.Path)
	require.Equal(t, "image/webp", req.ContentType)
	require.Equal(t, "max-age=0", req.CacheControl)
	require.Equal(t, "public-read", req.ACL)
	require.Equal(t, "webp payload", req.Body)
}

type s3PutRequest struct {
	Method       string
	Path         string
	ContentType  string
	CacheControl string
	ACL          string
	Body         string
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
