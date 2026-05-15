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

func TestResolveBindingContext_FallbackBySenderAsExternalAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			// First query by external_account_id=provider-bot-1 and external_chat_id=ctx-2 misses.
			// Fallback query by external_account_id=sender should hit.
			if strings.Contains(r.URL.RawQuery, "external_account_id=eq.provider-bot-1") {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			if strings.Contains(r.URL.RawQuery, "external_account_id=eq.o9cq80lrKImcs-Px9shB1do0FAWk%40im.wechat") {
				_, _ = w.Write([]byte(`[{"id":"bind-sender","user_id":"1001","external_account_id":"o9cq80lrKImcs-Px9shB1do0FAWk@im.wechat","external_chat_id":"AARz-token-1","binding_status":"active"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			_, _ = w.Write([]byte(`[{"id":"route-1","user_id":"1001","role_id":18,"tool_binding_id":"bind-sender"}]`))
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

	got, err := client.ResolveBindingContext(context.Background(), "provider-bot-1", "o9cq80lrKImcs-Px9shB1do0FAWk@im.wechat")
	if err != nil {
		t.Fatalf("ResolveBindingContext: %v", err)
	}
	if got == nil {
		t.Fatal("binding context should not be nil")
	}
	if got.RoleID != "18" {
		t.Fatalf("role_id=%q", got.RoleID)
	}
}

func TestWriteAuditLog_DefaultTable(t *testing.T) {
	var gotPath string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.WriteAuditLog(context.Background(), AuditLogInput{
		EventType: "unit.audit.default_table",
		SessionID: "sess-1",
		TraceID:   "trace-1",
		Detail: map[string]any{
			"hello": "world",
		},
	})
	if err != nil {
		t.Fatalf("WriteAuditLog: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/rest/v1/bl_platform_audit_logs" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestWriteUsageEvent_DefaultTable(t *testing.T) {
	var gotPath string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.WriteUsageEvent(context.Background(), UsageEventInput{
		UserID:      "1001",
		PeriodMonth: "2026-05",
		Delta:       3,
		Source:      "openilink_hub_ai",
		SessionID:   "sess-1",
		TraceID:     "trace-1",
		Detail: map[string]any{
			"usage_units": 3,
		},
	})
	if err != nil {
		t.Fatalf("WriteUsageEvent: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/rest/v1/bl_usage_events" {
		t.Fatalf("path=%q", gotPath)
	}
}
