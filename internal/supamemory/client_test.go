package supamemory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeMemoryRows_BotIDAsNumber(t *testing.T) {
	body := []byte(`[
		{"id":"m1","user_id":"u1","bot_id":18,"content":"hello","source":"openilink_user","created_at":"2026-05-16T00:00:00Z"}
	]`)
	rows, err := decodeMemoryRows(body)
	if err != nil {
		t.Fatalf("decodeMemoryRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len=%d", len(rows))
	}
	if string(rows[0].RoleID) != "18" {
		t.Fatalf("role_id=%q", rows[0].RoleID)
	}
}

func TestDecodeMemoryRows_BotIDAsString(t *testing.T) {
	body := []byte(`[
		{"id":"m1","user_id":"u1","bot_id":"18","content":"hello","source":"openilink_user","created_at":"2026-05-16T00:00:00Z"}
	]`)
	rows, err := decodeMemoryRows(body)
	if err != nil {
		t.Fatalf("decodeMemoryRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len=%d", len(rows))
	}
	if string(rows[0].RoleID) != "18" {
		t.Fatalf("role_id=%q", rows[0].RoleID)
	}
}

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

func TestBumpUsageLedger_UsesUnifiedRPC(t *testing.T) {
	var gotPath string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"user_id":"00000000-0000-0000-0000-000000000001","period_month":"2026-05","message_used":3,"updated_at":"2026-05-15T00:00:00Z","event_id":9}]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.BumpUsageLedger(context.Background(), UsageLedgerInput{
		UserID:     "1001",
		Delta:      3,
		Source:     "openilink_hub_ai",
		SessionID:  "sess-1",
		TraceID:    "trace-1",
		WriteEvent: true,
		Detail: map[string]any{
			"usage_units": 3,
		},
	})
	if err != nil {
		t.Fatalf("BumpUsageLedger: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/rest/v1/rpc/bump_usage_ledger" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestGetUsageBillingConfig_FromDictItems(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"item_value":"true","ext":{"text_chars_per_unit":220}}]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cfg, err := client.GetUsageBillingConfig(context.Background())
	if err != nil {
		t.Fatalf("GetUsageBillingConfig: %v", err)
	}
	if gotPath != "/rest/v1/bl_dict_items" {
		t.Fatalf("path=%q", gotPath)
	}
	if cfg == nil {
		t.Fatal("config should not be nil")
	}
	if !cfg.Enabled {
		t.Fatal("enabled should be true")
	}
	if cfg.TextCharsPerUnit != 220 {
		t.Fatalf("chars_per_unit=%d", cfg.TextCharsPerUnit)
	}
}

func TestIsEmojiReplyEnabled(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"emoji_reply_enabled":true}]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	enabled, err := client.IsEmojiReplyEnabled(context.Background(), "1001", "2002")
	if err != nil {
		t.Fatalf("IsEmojiReplyEnabled: %v", err)
	}
	if gotPath != "/rest/v1/bl_bots" {
		t.Fatalf("path=%q", gotPath)
	}
	if !enabled {
		t.Fatal("enabled should be true")
	}
}

func TestListEmojiAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"item_name":"Happy","item_value":"https://cdn.example.com/happy.webp","ext":{"enabled":true,"lang":"zh-CN","desc":"开心"}},
			{"item_name":"Off","item_value":"https://cdn.example.com/off.webp","ext":{"enabled":false,"lang":"default","desc":"关闭"}},
			{"item_name":"Default","item_value":"https://cdn.example.com/default.webp","ext":{"lang":"default","desc":"默认"}}
		]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	assets, err := client.ListEmojiAssets(context.Background())
	if err != nil {
		t.Fatalf("ListEmojiAssets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("assets len=%d", len(assets))
	}
	if assets[0].Lang != "zh-CN" || assets[0].Desc != "开心" {
		t.Fatalf("first asset = %#v", assets[0])
	}
	if assets[1].Lang != "default" || assets[1].URL != "https://cdn.example.com/default.webp" {
		t.Fatalf("second asset = %#v", assets[1])
	}
}

func TestResolveConversationID_ByContextToken(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"conversation_id":"11111111-1111-1111-1111-111111111111"}]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	id, err := client.ResolveConversationID(context.Background(), "1001", "2002", "ctx-1", "wx-user-1")
	if err != nil {
		t.Fatalf("ResolveConversationID: %v", err)
	}
	if gotPath != "/rest/v1/bl_platform_messages" {
		t.Fatalf("path=%q", gotPath)
	}
	if id != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("conversation_id=%q", id)
	}
}

func TestUpsertConversationState_DefaultTable(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotPrefer string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotPrefer = r.Header.Get("Prefer")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
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

	err = client.UpsertConversationState(context.Background(), ConversationStateInput{
		ConversationID: "11111111-1111-1111-1111-111111111111",
		Stage:          "task_active",
		ActiveFlow:     "qa",
		StatePayload: map[string]any{
			"planner": "ok",
		},
		Version: 2,
	})
	if err != nil {
		t.Fatalf("UpsertConversationState: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/rest/v1/bl_conversation_states" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotPrefer != "resolution=merge-duplicates,return=minimal" {
		t.Fatalf("prefer=%q", gotPrefer)
	}
	if gotBody["stage"] != "task_active" {
		t.Fatalf("stage=%v", gotBody["stage"])
	}
}

func TestAppendDialogueEvent_DefaultTable(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
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

	err = client.AppendDialogueEvent(context.Background(), DialogueEventInput{
		ConversationID: "11111111-1111-1111-1111-111111111111",
		TurnID:         "turn-1",
		EventID:        "evt-1",
		EventType:      "dialogue.guard.relevance",
		IdempotencyKey: "idem-1",
		EventPayload: map[string]any{
			"score": 90,
		},
	})
	if err != nil {
		t.Fatalf("AppendDialogueEvent: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q", gotMethod)
	}
	if gotPath != "/rest/v1/bl_dialogue_events" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["event_type"] != "dialogue.guard.relevance" {
		t.Fatalf("event_type=%v", gotBody["event_type"])
	}
}

func TestGetDialogueRuntimeFlags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"item_code":"planner_only","item_value":"true"},
			{"item_code":"guard_soft_mode","item_value":"false"},
			{"item_code":"fallback_fast_path","item_value":"1"}
		]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	flags, err := client.GetDialogueRuntimeFlags(context.Background())
	if err != nil {
		t.Fatalf("GetDialogueRuntimeFlags: %v", err)
	}
	if flags == nil {
		t.Fatal("flags should not be nil")
	}
	if !flags.PlannerOnly {
		t.Fatal("planner_only should be true")
	}
	if flags.GuardSoftMode {
		t.Fatal("guard_soft_mode should be false")
	}
	if !flags.FallbackFastPath {
		t.Fatal("fallback_fast_path should be true")
	}
}

func TestGetConversationState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"conversation_id":"11111111-1111-1111-1111-111111111111",
			"stage":"idle",
			"active_flow":"free_chat",
			"state_payload":{"rolling_summary":"foo bar"},
			"version":"7"
		}]`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:        srv.URL,
		ServiceRoleKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	row, err := client.GetConversationState(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("GetConversationState: %v", err)
	}
	if row == nil {
		t.Fatal("row should not be nil")
	}
	if row.Stage != "idle" {
		t.Fatalf("stage=%q", row.Stage)
	}
	if row.Version != 7 {
		t.Fatalf("version=%d", row.Version)
	}
	if row.StatePayload["rolling_summary"] != "foo bar" {
		t.Fatalf("summary=%v", row.StatePayload["rolling_summary"])
	}
}
