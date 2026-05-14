package supamemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveBindingContext_RoleIDAsNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			_, _ = w.Write([]byte(`[{"id":"bind-1","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"ctx-1","binding_status":"active"}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			_, _ = w.Write([]byte(`[{"id":"route-1","user_id":"1001","role_id":2002,"tool_binding_id":"bind-1"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.ResolveBindingContext(context.Background(), "provider-bot-1", "ctx-1")
	if err != nil {
		t.Fatalf("ResolveBindingContext: %v", err)
	}
	if got == nil {
		t.Fatal("binding context should not be nil")
	}
	if got.UserID != "1001" {
		t.Fatalf("user_id=%q", got.UserID)
	}
	if got.RoleID != "2002" {
		t.Fatalf("role_id=%q", got.RoleID)
	}
}

