package storage

import (
	"context"
)

// Store is the storage interface for media files.
// Implementations: S3Store (MinIO/S3), FSStore (local filesystem).
type Store interface {
	Put(ctx context.Context, key, contentType string, data []byte) (string, error)
	Get(ctx context.Context, key string) ([]byte, error)
	URL(key string) string
}
