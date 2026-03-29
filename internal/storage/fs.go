package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSStore implements Store using the local filesystem.
type FSStore struct {
	root      string // absolute root directory for stored files
	publicURL string // URL prefix, e.g. "/api/v1/media"
}

// NewFS creates a new FSStore rooted at the given directory.
func NewFS(root, publicURL string) (*FSStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("storage: fs abs: %w", err)
	}
	if err := os.MkdirAll(abs, 0750); err != nil {
		return nil, fmt.Errorf("storage: fs init: %w", err)
	}
	if publicURL == "" {
		publicURL = "/api/v1/media"
	}
	return &FSStore{root: abs, publicURL: publicURL}, nil
}

// safePath resolves a key to an absolute path and ensures it stays under root.
func (f *FSStore) safePath(key string) (string, error) {
	// Reject absolute paths and clean the key first
	clean := filepath.FromSlash(key)
	if filepath.IsAbs(clean) {
		return "", errors.New("storage: absolute path rejected")
	}
	p := filepath.Clean(filepath.Join(f.root, clean))
	if !strings.HasPrefix(p, f.root+string(os.PathSeparator)) {
		return "", errors.New("storage: path traversal rejected")
	}
	return p, nil
}

func (f *FSStore) Put(_ context.Context, key, contentType string, data []byte) (string, error) {
	path, err := f.safePath(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", fmt.Errorf("storage: fs mkdir %s: %w", key, err)
	}
	if err := os.WriteFile(path, data, 0640); err != nil {
		return "", fmt.Errorf("storage: fs put %s: %w", key, err)
	}
	return f.URL(key), nil
}

func (f *FSStore) Get(_ context.Context, key string) ([]byte, error) {
	path, err := f.safePath(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("storage: fs get %s: %w", key, err)
	}
	return data, nil
}

func (f *FSStore) URL(key string) string {
	return f.publicURL + "/" + key
}
