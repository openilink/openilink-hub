package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMiddlewareRejectsNoSession(t *testing.T) {
	db := testDB(t)

	handler := Middleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	db := testDB(t)

	handler := Middleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "bad-token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddlewareAcceptsValidSession(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db := testDB(t)

	user, _ := db.CreateUser("test_mw_user", "MW User")
	t.Cleanup(func() { db.DeleteUser(user.ID) })

	token, _ := CreateSession(db, user.ID)

	var gotUserID string
	handler := Middleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if gotUserID != user.ID {
		t.Errorf("user_id = %q, want %q", gotUserID, user.ID)
	}
}

func TestUserIDFromContextEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	uid := UserIDFromContext(req.Context())
	if uid != "" {
		t.Errorf("expected empty, got %q", uid)
	}
}
