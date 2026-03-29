package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Store implements Store using MinIO / S3 compatible object storage.
type S3Store struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// S3Config holds S3/MinIO configuration.
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

// NewS3 creates a new S3Store and ensures the bucket exists.
func NewS3(cfg S3Config) (*S3Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: s3 connect: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: s3 check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("storage: s3 create bucket: %w", err)
		}
	}

	publicURL := cfg.PublicURL
	if publicURL == "" {
		publicURL = "/api/v1/media"
	}

	return &S3Store{client: client, bucket: cfg.Bucket, publicURL: publicURL}, nil
}

func (s *S3Store) Put(ctx context.Context, key, contentType string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("storage: s3 put %s: %w", key, err)
	}
	return s.URL(key), nil
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: s3 get %s: %w", key, err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("storage: s3 read %s: %w", key, err)
	}
	return data, nil
}

func (s *S3Store) URL(key string) string {
	return s.publicURL + "/" + key
}
