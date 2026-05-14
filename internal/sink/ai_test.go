package sink

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/supamemory"
)

func TestParseCustomHeaders_ObjectFormat(t *testing.T) {
	m := parseCustomHeaders(`{"HTTP-Referer":"https://openclaw.ai","X-Title":"OpenClaw"}`)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["HTTP-Referer"] != "https://openclaw.ai" {
		t.Errorf("HTTP-Referer = %q", m["HTTP-Referer"])
	}
	if m["X-Title"] != "OpenClaw" {
		t.Errorf("X-Title = %q", m["X-Title"])
	}
}

func TestParseCustomHeaders_ArrayFormat(t *testing.T) {
	m := parseCustomHeaders(`[["HTTP-Referer","https://openclaw.ai"],["X-Title","OpenClaw"]]`)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["HTTP-Referer"] != "https://openclaw.ai" {
		t.Errorf("HTTP-Referer = %q", m["HTTP-Referer"])
	}
}

func TestParseCustomHeaders_EmptyKeyFiltered(t *testing.T) {
	m := parseCustomHeaders(`[["","value"],["X-Good","ok"]]`)
	if _, ok := m[""]; ok {
		t.Error("empty key should be filtered")
	}
	if m["X-Good"] != "ok" {
		t.Errorf("X-Good = %q", m["X-Good"])
	}
}

func TestParseCustomHeaders_InvalidJSON(t *testing.T) {
	m := parseCustomHeaders(`not json`)
	if m != nil {
		t.Errorf("expected nil for invalid JSON, got %v", m)
	}
}

func TestParseCustomHeaders_Empty(t *testing.T) {
	m := parseCustomHeaders(`{}`)
	if m != nil {
		t.Errorf("expected nil for empty object, got %v", m)
	}
	m = parseCustomHeaders(`[]`)
	if m != nil {
		t.Errorf("expected nil for empty array, got %v", m)
	}
}

func TestResolveRuntimePrompt_RPCAndCache(t *testing.T) {
	var promptRPCCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			_, _ = w.Write([]byte(`[{"id":"bind-1","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"wx-1","binding_status":"active"}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			_, _ = w.Write([]byte(`[{"id":"route-1","user_id":"1001","role_id":"2002","tool_binding_id":"bind-1"}]`))
		case r.URL.Path == "/rest/v1/rpc/get_effective_full_prompt":
			atomic.AddInt32(&promptRPCCount, 1)
			_, _ = w.Write([]byte(`[{"full_prompt":"rpc-full-prompt","system_prompt":"sys","user_prompt":"usr","prompt_version":1715}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := supamemory.NewClient(supamemory.Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	aiSink := &AI{SupaMemory: client}
	base := store.AIConfig{SystemPrompt: "global-system-prompt"}

	cfg1, meta1 := aiSink.resolveRuntimePrompt(context.Background(), base, "bot-local-1", "provider-bot-1", "", "wx-1")
	if cfg1.SystemPrompt != "rpc-full-prompt" {
		t.Fatalf("first call prompt=%q", cfg1.SystemPrompt)
	}
	if meta1.Source != "supabase_rpc" {
		t.Fatalf("first call source=%q", meta1.Source)
	}

	cfg2, meta2 := aiSink.resolveRuntimePrompt(context.Background(), base, "bot-local-1", "provider-bot-1", "", "wx-1")
	if cfg2.SystemPrompt != "rpc-full-prompt" {
		t.Fatalf("second call prompt=%q", cfg2.SystemPrompt)
	}
	if meta2.Source != "cache" {
		t.Fatalf("second call source=%q", meta2.Source)
	}
	if atomic.LoadInt32(&promptRPCCount) != 1 {
		t.Fatalf("prompt rpc should be called once, got %d", promptRPCCount)
	}
}

func TestResolveRuntimePrompt_RPCFailureFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			_, _ = w.Write([]byte(`[{"id":"bind-1","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"wx-1","binding_status":"active"}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			_, _ = w.Write([]byte(`[{"id":"route-1","user_id":"1001","role_id":"2002","tool_binding_id":"bind-1"}]`))
		case r.URL.Path == "/rest/v1/rpc/get_effective_full_prompt":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"rpc error"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := supamemory.NewClient(supamemory.Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	aiSink := &AI{SupaMemory: client}
	base := store.AIConfig{SystemPrompt: "global-system-prompt"}

	cfg, meta := aiSink.resolveRuntimePrompt(context.Background(), base, "bot-local-1", "provider-bot-1", "", "wx-1")
	if cfg.SystemPrompt != "global-system-prompt" {
		t.Fatalf("fallback prompt=%q", cfg.SystemPrompt)
	}
	if meta.Source != "global_fallback" {
		t.Fatalf("fallback source=%q", meta.Source)
	}
}

func TestResolveRuntimePrompt_ContextTokenPriority(t *testing.T) {
	var promptRPCCount int32
	var routeHitByBindingID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			q := r.URL.Query()
			externalChat := q.Get("external_chat_id")
			if externalChat == "eq.ctx-1" {
				_, _ = w.Write([]byte(`[{"id":"bind-ctx","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"ctx-1","binding_status":"active"}]`))
				return
			}
			if externalChat == "eq.wx-1" {
				_, _ = w.Write([]byte(`[{"id":"bind-wx","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"wx-1","binding_status":"active"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			q := r.URL.Query()
			toolBindingID := q.Get("tool_binding_id")
			routeHitByBindingID = toolBindingID
			if toolBindingID == "eq.bind-ctx" {
				_, _ = w.Write([]byte(`[{"id":"route-ctx","user_id":"1001","role_id":"2002","tool_binding_id":"bind-ctx"}]`))
				return
			}
			if toolBindingID == "eq.bind-wx" {
				_, _ = w.Write([]byte(`[{"id":"route-wx","user_id":"1001","role_id":"2999","tool_binding_id":"bind-wx"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.URL.Path == "/rest/v1/rpc/get_effective_full_prompt":
			atomic.AddInt32(&promptRPCCount, 1)
			_, _ = w.Write([]byte(`[{"full_prompt":"rpc-from-context-token","system_prompt":"sys","user_prompt":"usr","prompt_version":2001}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := supamemory.NewClient(supamemory.Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	aiSink := &AI{SupaMemory: client}
	base := store.AIConfig{SystemPrompt: "global-system-prompt"}

	cfg, meta := aiSink.resolveRuntimePrompt(context.Background(), base, "bot-local-1", "provider-bot-1", "ctx-1", "wx-1")
	if cfg.SystemPrompt != "rpc-from-context-token" {
		t.Fatalf("prompt=%q", cfg.SystemPrompt)
	}
	if meta.Source != "supabase_rpc" {
		t.Fatalf("source=%q", meta.Source)
	}
	if meta.RoleID != "2002" {
		t.Fatalf("role_id=%q", meta.RoleID)
	}
	if routeHitByBindingID != "eq.bind-ctx" {
		t.Fatalf("expected route hit by context-token binding, got %q", routeHitByBindingID)
	}
	if atomic.LoadInt32(&promptRPCCount) != 1 {
		t.Fatalf("prompt rpc should be called once, got %d", promptRPCCount)
	}
}
