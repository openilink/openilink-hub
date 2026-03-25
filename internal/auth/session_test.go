package auth

import (
	"os"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
)

func testDB(t *testing.T) store.Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSessionCreateValidateDelete(t *testing.T) {
	db := testDB(t)

	user, err := db.CreateUser("test_session_user", "Session User")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { db.DeleteUser(user.ID) })

	token, err := CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	// Validate
	userID, err := ValidateSession(db, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if userID != user.ID {
		t.Errorf("user_id = %q, want %q", userID, user.ID)
	}

	// Invalid token
	_, err = ValidateSession(db, "nonexistent-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}

	// Delete
	DeleteSession(db, token)
	_, err = ValidateSession(db, token)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSessionStoreSetGet(t *testing.T) {
	store := NewSessionStore()

	store.Set("key1", nil)
	data := store.Get("key1")
	if data != nil {
		t.Error("expected nil data")
	}

	// Get consumes the entry
	data = store.Get("key1")
	if data != nil {
		t.Error("second get should return nil (consumed)")
	}

	// Non-existent key
	data = store.Get("nonexistent")
	if data != nil {
		t.Error("expected nil for missing key")
	}
}
