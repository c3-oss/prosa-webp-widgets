package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Sink stores one rendered widget and returns the resulting location.
type Sink interface {
	Put(ctx context.Context, key string, data []byte) (string, error)
}

type DiskSink struct {
	Dir string
}

func (s DiskSink) Put(_ context.Context, key string, data []byte) (string, error) {
	if strings.TrimSpace(s.Dir) == "" {
		return "", fmt.Errorf("output directory is required")
	}
	target := filepath.Join(s.Dir, filepath.FromSlash(key))
	// Rendered widgets are public web assets; world-readable output is intentional.
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301
		return "", fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil { // #nosec G306
		return "", fmt.Errorf("write %s: %w", target, err)
	}
	return target, nil
}

type S3Sink struct {
	client    *s3.Client
	bucket    string
	prefix    string
	publicURL string
	acl       types.ObjectCannedACL
}

func NewS3Sink(ctx context.Context) (*S3Sink, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET_NAME"))
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME is required")
	}
	region := envDefault("AWS_REGION", "us-east-1")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if accessKey != "" || secretKey != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	}
	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT_URL"))
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = endpoint != ""
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	acl := types.ObjectCannedACL(os.Getenv("S3_ACL"))
	return &S3Sink{
		client:    client,
		bucket:    bucket,
		prefix:    strings.Trim(strings.TrimSpace(os.Getenv("S3_PREFIX")), "/"),
		publicURL: strings.TrimRight(strings.TrimSpace(os.Getenv("S3_PUBLIC_BASE_URL")), "/"),
		acl:       acl,
	}, nil
}

func (s *S3Sink) Put(ctx context.Context, key string, data []byte) (string, error) {
	objectKey := key
	if s.prefix != "" {
		objectKey = path.Join(s.prefix, key)
	}
	input := &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(objectKey),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("image/webp"),
		CacheControl: aws.String("max-age=0"),
	}
	if s.acl != "" {
		input.ACL = s.acl
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("put s3://%s/%s: %w", s.bucket, objectKey, err)
	}
	if s.publicURL != "" {
		u, err := url.JoinPath(s.publicURL, objectKey)
		if err == nil {
			return u, nil
		}
	}
	return "s3://" + s.bucket + "/" + objectKey, nil
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
