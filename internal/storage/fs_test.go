package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFSStore(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFS(dir, "/api/v1/media")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Put
	url, err := fs.Put(ctx, "bot1/2026/03/29/img_0.jpg", "image/jpeg", []byte("fake-jpeg"))
	if err != nil {
		t.Fatal(err)
	}
	if url != "/api/v1/media/bot1/2026/03/29/img_0.jpg" {
		t.Fatalf("unexpected url: %s", url)
	}

	// File exists on disk
	data, err := os.ReadFile(filepath.Join(dir, "bot1/2026/03/29/img_0.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-jpeg" {
		t.Fatalf("unexpected data: %s", data)
	}

	// Get
	got, err := fs.Get(ctx, "bot1/2026/03/29/img_0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fake-jpeg" {
		t.Fatalf("unexpected get data: %s", got)
	}

	// Get non-existent
	_, err = fs.Get(ctx, "missing/key")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestFSStoreNestedDirs(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFS(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = fs.Put(ctx, "a/b/c/d/file.txt", "text/plain", []byte("nested"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := fs.Get(ctx, "a/b/c/d/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "nested" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestFSStorePathTraversal(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFS(dir, "/api/v1/media")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	malicious := []string{
		"../../etc/passwd",
		"../../../etc/shadow",
		"bot1/../../etc/passwd",
		"bot1/../../../etc/passwd",
		"/etc/passwd",
		"bot1/2026/../../../../../../etc/passwd",
	}

	for _, key := range malicious {
		t.Run("Put_"+key, func(t *testing.T) {
			_, err := fs.Put(ctx, key, "text/plain", []byte("hack"))
			if err == nil {
				t.Fatalf("expected path traversal rejection for key %q", key)
			}
		})
		t.Run("Get_"+key, func(t *testing.T) {
			_, err := fs.Get(ctx, key)
			if err == nil {
				t.Fatalf("expected path traversal rejection for key %q", key)
			}
		})
	}
}
