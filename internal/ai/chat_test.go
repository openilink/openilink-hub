package ai

import (
	"context"
	"os"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
)

func TestCompleteWithRealAPI(t *testing.T) {
	baseURL := os.Getenv("TEST_AI_BASE_URL")
	apiKey := os.Getenv("TEST_AI_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("TEST_AI_BASE_URL and TEST_AI_API_KEY not set")
	}

	// We need a DB for history lookup, but for this test we can use a real one
	// or just test the raw HTTP call directly
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://openilink:openilink@localhost:15432/openilink_test?sslmode=disable"
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Skipf("skip: database unavailable: %v", err)
	}
	defer db.Close()

	cfg := store.AIConfig{
		Enabled:      true,
		BaseURL:      baseURL,
		APIKey:       apiKey,
		Model:        os.Getenv("TEST_AI_MODEL"),
		SystemPrompt: "You are a helpful assistant. Reply in one short sentence.",
		MaxHistory:   5,
	}

	reply, err := Complete(context.Background(), cfg, db, "nonexistent-channel", "test-sender", "Hello, what is 1+1?")
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if reply == "" {
		t.Fatal("got empty reply")
	}
	t.Logf("AI reply: %s", reply)
}
