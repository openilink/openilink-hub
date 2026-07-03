package sink

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
	"github.com/openilink/openilink-hub/internal/supamemory"
)

func TestCalculateUsageUnitsV2(t *testing.T) {
	t.Run("short text costs one", func(t *testing.T) {
		units := calculateUsageUnitsV2("你好", []provider.MessageItem{{Type: "text", Text: "你好"}}, 180)
		if units != 1 {
			t.Fatalf("units=%d, want=1", units)
		}
	})

	t.Run("long text scales by char length", func(t *testing.T) {
		longText := strings.Repeat("a", 361)
		units := calculateUsageUnitsV2(longText, []provider.MessageItem{{Type: "text", Text: longText}}, 180)
		if units != 3 {
			t.Fatalf("units=%d, want=3", units)
		}
	})

	t.Run("file baseline overrides short text", func(t *testing.T) {
		units := calculateUsageUnitsV2("ok", []provider.MessageItem{{Type: "video"}}, 180)
		if units != 4 {
			t.Fatalf("units=%d, want=4", units)
		}
	})
}

func TestResolveUsageBillingConfig_FallbackEnv(t *testing.T) {
	aiSink := &AI{
		UsageBillingV2Enabled:    true,
		UsageBillingCharsPerUnit: 200,
	}
	enabled, chars, source := aiSink.resolveUsageBillingConfig(context.Background())
	if !enabled {
		t.Fatal("enabled should be true")
	}
	if chars != 200 {
		t.Fatalf("chars=%d", chars)
	}
	if source != "env_fallback" {
		t.Fatalf("source=%q", source)
	}
}

func TestResolveUsageBillingConfig_FromSupabase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_dict_items"):
			_, _ = w.Write([]byte(`[{"item_value":"true","ext":{"text_chars_per_unit":240}}]`))
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

	aiSink := &AI{
		SupaMemory:               client,
		UsageBillingV2Enabled:    false,
		UsageBillingCharsPerUnit: 180,
	}
	enabled, chars, source := aiSink.resolveUsageBillingConfig(context.Background())
	if !enabled {
		t.Fatal("enabled should be true")
	}
	if chars != 240 {
		t.Fatalf("chars=%d", chars)
	}
	if source != "supabase_dict_items" {
		t.Fatalf("source=%q", source)
	}
}

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

func TestResolveLanguageModel(t *testing.T) {
	cfg := store.AIConfig{
		Model:         "ai21/jamba-large-1.7",
		ModelZH:       "deepseek/deepseek-v3.2",
		ModelNonZH:    "ai21/jamba-large-1.7",
		FallbackModel: "openai/gpt-4o-mini",
	}

	t.Run("zh routes to deepseek", func(t *testing.T) {
		model, fallback, bucket := resolveLanguageModel(cfg, "你好，今天聊角色设定", false)
		if model != "deepseek/deepseek-v3.2" {
			t.Fatalf("model=%q", model)
		}
		if fallback != "openai/gpt-4o-mini" {
			t.Fatalf("fallback=%q", fallback)
		}
		if bucket != "zh-CN" {
			t.Fatalf("bucket=%q", bucket)
		}
	})

	t.Run("non-zh routes to jamba", func(t *testing.T) {
		model, _, bucket := resolveLanguageModel(cfg, "Can you help me with this character?", false)
		if model != "ai21/jamba-large-1.7" {
			t.Fatalf("model=%q", model)
		}
		if bucket != "default" {
			t.Fatalf("bucket=%q", bucket)
		}
	})

	t.Run("bot override keeps selected model", func(t *testing.T) {
		overrideCfg := cfg
		overrideCfg.Model = "anthropic/claude-3.5-sonnet"
		model, _, bucket := resolveLanguageModel(overrideCfg, "你好", true)
		if model != "anthropic/claude-3.5-sonnet" {
			t.Fatalf("model=%q", model)
		}
		if bucket != "bot_override" {
			t.Fatalf("bucket=%q", bucket)
		}
	})
}

func TestResolveCompletionTimeout(t *testing.T) {
	t.Run("default timeout when unset", func(t *testing.T) {
		got := resolveCompletionTimeout(store.AIConfig{})
		if got != defaultCompletionTimeout {
			t.Fatalf("timeout=%v", got)
		}
	})

	t.Run("clamp low value", func(t *testing.T) {
		got := resolveCompletionTimeout(store.AIConfig{CompletionTimeoutSec: 1})
		if got != 5*time.Second {
			t.Fatalf("timeout=%v", got)
		}
	})

	t.Run("clamp high value", func(t *testing.T) {
		got := resolveCompletionTimeout(store.AIConfig{CompletionTimeoutSec: 360})
		if got != 120*time.Second {
			t.Fatalf("timeout=%v", got)
		}
	})

	t.Run("use configured value", func(t *testing.T) {
		got := resolveCompletionTimeout(store.AIConfig{CompletionTimeoutSec: 42})
		if got != 42*time.Second {
			t.Fatalf("timeout=%v", got)
		}
	})
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
	if cfg1.SystemPrompt != "sys" {
		t.Fatalf("first call prompt=%q", cfg1.SystemPrompt)
	}
	if meta1.UserPrompt != "usr" {
		t.Fatalf("first call user_prompt=%q", meta1.UserPrompt)
	}
	if meta1.Source != "supabase_rpc" {
		t.Fatalf("first call source=%q", meta1.Source)
	}

	cfg2, meta2 := aiSink.resolveRuntimePrompt(context.Background(), base, "bot-local-1", "provider-bot-1", "", "wx-1")
	if cfg2.SystemPrompt != "sys" {
		t.Fatalf("second call prompt=%q", cfg2.SystemPrompt)
	}
	if meta2.UserPrompt != "usr" {
		t.Fatalf("second call user_prompt=%q", meta2.UserPrompt)
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
	if cfg.SystemPrompt != "sys" {
		t.Fatalf("prompt=%q", cfg.SystemPrompt)
	}
	if meta.UserPrompt != "usr" {
		t.Fatalf("user_prompt=%q", meta.UserPrompt)
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

func TestResolveRuntimePrompt_ComposeWhenFullPromptMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_tool_bindings"):
			_, _ = w.Write([]byte(`[{"id":"bind-1","user_id":"1001","external_account_id":"provider-bot-1","external_chat_id":"ctx-1","binding_status":"active"}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_role_tool_routes"):
			_, _ = w.Write([]byte(`[{"id":"route-1","user_id":"1001","role_id":"2002","tool_binding_id":"bind-1"}]`))
		case r.URL.Path == "/rest/v1/rpc/get_effective_full_prompt":
			_, _ = w.Write([]byte(`[{"full_prompt":"","system_prompt":"sys-only","user_prompt":"usr-only","prompt_version":3001}]`))
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
	if cfg.SystemPrompt != "sys-only" {
		t.Fatalf("prompt=%q", cfg.SystemPrompt)
	}
	if meta.UserPrompt != "usr-only" {
		t.Fatalf("user_prompt=%q", meta.UserPrompt)
	}
	if meta.Source != "supabase_rpc" {
		t.Fatalf("source=%q", meta.Source)
	}
	if meta.Version != 3001 {
		t.Fatalf("version=%d", meta.Version)
	}
}

func TestResolveEmojiDecision(t *testing.T) {
	now := time.Now().Unix()
	base := struct {
		EmojiEnabled             bool
		Text                     string
		LatestConversationEmojiS int64
		UserEmojiCountInWindow   int
		LatestUserEmojiURL       string
		CandidateEmojiURL        string
		NowSec                   int64
		TurnsSinceLastEmoji      int
	}{
		EmojiEnabled:             true,
		Text:                     "来个表情包",
		LatestConversationEmojiS: 0,
		UserEmojiCountInWindow:   0,
		LatestUserEmojiURL:       "",
		CandidateEmojiURL:        "https://cdn.example.com/a.webp",
		NowSec:                   now,
		TurnsSinceLastEmoji:      10,
	}

	decision := resolveEmojiDecision(base)
	if !decision.IncludeEmoji || decision.Reason != "force_trigger" {
		t.Fatalf("decision=%+v", decision)
	}

	for _, text := range []string{"发一个", "来一个吧", "都可以，随便发"} {
		forceInput := base
		forceInput.Text = text
		decision = resolveEmojiDecision(forceInput)
		if !decision.IncludeEmoji || decision.Reason != "force_trigger" {
			t.Fatalf("force text=%q decision=%+v", text, decision)
		}
	}

	suppressed := base
	suppressed.Text = "我想退款订单，别发表情包"
	decision = resolveEmojiDecision(suppressed)
	if decision.Reason != "suppressed" || decision.IncludeEmoji {
		t.Fatalf("suppressed decision=%+v", decision)
	}

	throttled := base
	throttled.LatestConversationEmojiS = now - 60
	decision = resolveEmojiDecision(throttled)
	if decision.Reason != "throttled_conversation" || decision.IncludeEmoji {
		t.Fatalf("throttled decision=%+v", decision)
	}

	dedup := base
	dedup.LatestUserEmojiURL = "https://cdn.example.com/a.webp"
	decision = resolveEmojiDecision(dedup)
	if decision.Reason != "dedup_recent_asset" || decision.IncludeEmoji {
		t.Fatalf("dedup decision=%+v", decision)
	}
}

func TestResolveEmojiReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_bots"):
			_, _ = w.Write([]byte(`[{"emoji_reply_enabled":true}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_dict_items"):
			_, _ = w.Write([]byte(`[
				{"item_name":"Happy","item_value":"https://cdn.example.com/happy.webp","ext":{"enabled":true,"lang":"zh-CN","desc":"开心图"}}
			]`))
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

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer db.Close()

	aiSink := &AI{
		Store:      db,
		SupaMemory: client,
	}
	delivery := Delivery{
		BotDBID: "bot-1",
		Message: provider.InboundMessage{
			Sender: "wx-user-1",
		},
	}
	msgItem, _ := json.Marshal([]map[string]any{{
		"type": "image",
		"url":  "https://cdn.example.com/old.webp",
	}})
	_, _ = db.SaveMessage(&store.Message{
		BotID:       "bot-1",
		Direction:   "outbound",
		ToUserID:    "wx-user-1",
		MessageType: 2,
		ItemList:    msgItem,
		CreatedAt:   time.Now().Unix() - int64((emojiConversationCooldown/time.Second)+60),
		CreateTimeMs: func() *int64 {
			v := time.Now().Add(-(emojiConversationCooldown + time.Minute)).UnixMilli()
			return &v
		}(),
	})

	asset, info := aiSink.resolveEmojiReply(context.Background(), delivery, runtimePromptMeta{
		UserID: "1001",
		RoleID: "2002",
	}, "来个表情包")
	if asset == nil {
		t.Fatalf("asset should not be nil, info=%+v", info)
	}
	if info.Reason != "force_trigger" {
		t.Fatalf("reason=%s", info.Reason)
	}
	if asset.URL != "https://cdn.example.com/happy.webp" {
		t.Fatalf("url=%s", asset.URL)
	}
}

func TestResolveEmojiReply_DedupRecentAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_bots"):
			_, _ = w.Write([]byte(`[{"emoji_reply_enabled":true}]`))
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_dict_items"):
			_, _ = w.Write([]byte(`[
				{"item_name":"Happy","item_value":"https://cdn.example.com/happy.webp","ext":{"enabled":true,"lang":"zh-CN","desc":"开心图"}}
			]`))
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

	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer db.Close()

	aiSink := &AI{
		Store:      db,
		SupaMemory: client,
	}
	delivery := Delivery{
		BotDBID: "bot-1",
		Message: provider.InboundMessage{
			Sender: "wx-user-1",
		},
	}
	msgItem, _ := json.Marshal([]map[string]any{{
		"type": "image",
		"url":  "https://cdn.example.com/happy.webp",
	}})
	_, _ = db.SaveMessage(&store.Message{
		BotID:       "bot-1",
		Direction:   "outbound",
		ToUserID:    "wx-user-1",
		MessageType: 2,
		ItemList:    msgItem,
		CreateTimeMs: func() *int64 {
			v := time.Now().Add(-(emojiConversationCooldown + time.Minute)).UnixMilli()
			return &v
		}(),
	})

	asset, info := aiSink.resolveEmojiReply(context.Background(), delivery, runtimePromptMeta{
		UserID: "1001",
		RoleID: "2002",
	}, "来个表情包")
	if asset != nil {
		t.Fatalf("asset should be nil, info=%+v", info)
	}
	if info.Reason != "dedup_recent_asset" {
		t.Fatalf("reason=%s", info.Reason)
	}
}

func TestResolveConversationContext_FallbackWhenSupabaseReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/v1/bl_platform_messages"):
			_, _ = w.Write([]byte(`[]`))
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
	delivery := Delivery{
		Message: provider.InboundMessage{
			ExternalID:   "evt-1",
			ContextToken: "ctx-1",
			Sender:       "wx-user-1",
		},
	}
	convID, turnID := aiSink.resolveConversationContext(context.Background(), runtimePromptMeta{
		UserID: "u-1",
		RoleID: "r-1",
	}, delivery)

	if convID == "" {
		t.Fatal("conversation id should fallback, got empty")
	}
	if !strings.HasPrefix(convID, "fallback_u-1_r-1_") {
		t.Fatalf("unexpected fallback conversation id: %q", convID)
	}
	if turnID != "evt-1" {
		t.Fatalf("turnID=%q", turnID)
	}
}

func TestTrimMemoriesForPhase2(t *testing.T) {
	rows := []supamemory.MemoryRow{
		{Source: "openilink_user", Content: "u1"},
		{Source: "openilink_user", Content: "u2"},
		{Source: "openilink_assistant", Content: "a1"},
		{Source: "role_profile", Content: "r1"},
		{Source: "global", Content: "g1"},
		{Source: "global", Content: "g2"},
	}
	got := trimMemoriesForPhase2(rows, 4)
	if len(got) != 4 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Content != "u1" || got[1].Content != "u2" {
		t.Fatalf("session priority broken: %#v", got)
	}
}

func TestTrimMemoriesForPhase2DefaultLimit(t *testing.T) {
	rows := make([]supamemory.MemoryRow, 0, 12)
	for i := 0; i < 12; i++ {
		rows = append(rows, supamemory.MemoryRow{Source: "global", Content: "g"})
	}
	got := trimMemoriesForPhase2(rows, 0)
	if len(got) != memoryPromptMaxRows {
		t.Fatalf("len=%d, want=%d", len(got), memoryPromptMaxRows)
	}
}

func TestMergeRollingSummary(t *testing.T) {
	got := mergeRollingSummary("之前摘要", "用户问了一个很长很长的问题", "助手给了一个很长很长的回答")
	if !strings.Contains(got, "之前摘要") {
		t.Fatalf("missing prev: %q", got)
	}
	if !strings.Contains(got, "用户:") || !strings.Contains(got, "助手:") {
		t.Fatalf("missing merged parts: %q", got)
	}
}

func TestMergeRollingSummaryAllowsLongerSummary(t *testing.T) {
	got := mergeRollingSummary(strings.Repeat("旧", 760), "用户希望以后先给结论", "我会按这个偏好回复")
	runes := len([]rune(got))
	if runes > rollingSummaryMaxRunes {
		t.Fatalf("summary runes=%d, max=%d", runes, rollingSummaryMaxRunes)
	}
	if runes <= 420 {
		t.Fatalf("summary should exceed old 420 rune cap, got=%d", runes)
	}
}

func TestBuildMemoryQueryFromMessagesIncludesRecentContext(t *testing.T) {
	query := buildMemoryQueryFromMessages([]ai.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "我的偏好是以后先给结论"},
		{Role: "assistant", Content: "我会记住这个偏好"},
		{Role: "user", Content: "继续"},
	}, "继续")
	if !strings.Contains(query, "我的偏好是以后先给结论") {
		t.Fatalf("missing history context: %q", query)
	}
	if !strings.Contains(query, "user: 继续") {
		t.Fatalf("missing current text: %q", query)
	}
	if strings.Count(query, "user: 继续") != 1 {
		t.Fatalf("current text duplicated: %q", query)
	}
}

func TestShouldRecordLongTermMemory(t *testing.T) {
	cases := []struct {
		name   string
		source string
		text   string
		want   bool
	}{
		{name: "drops greeting", source: "openilink_user", text: "你好", want: false},
		{name: "drops command", source: "openilink_user", text: "/help", want: false},
		{name: "keeps user preference", source: "openilink_user", text: "我的偏好是以后先给结论再给步骤", want: true},
		{name: "drops assistant error", source: "openilink_assistant", text: "工具调用失败 timeout error", want: false},
		{name: "keeps assistant commitment", source: "openilink_assistant", text: "我会记住你希望以后先给结论再给步骤推进", want: true},
		// 场景/叙事/情感类信号
		{name: "keeps scene location keyword", source: "openilink_user", text: "我们坐在海边小木屋的窗边看雨", want: true},
		{name: "keeps weather scene", source: "openilink_user", text: "窗外下雨，气氛很安静", want: true},
		{name: "keeps emotional action", source: "openilink_user", text: "她微笑着递给我一杯热茶", want: true},
		{name: "keeps memory recall", source: "openilink_user", text: "我想起了小时候在老家的那段日子", want: true},
		{name: "keeps long narrative no keyword", source: "openilink_user", text: "今天天色渐暗，街道上行人寥寥，我一个人走着，心里说不清是什么感受，只是觉得有些空荡荡的", want: true},
		{name: "drops short plain text no keyword", source: "openilink_user", text: "嗯嗯", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRecordLongTermMemory(tc.source, tc.text)
			if got != tc.want {
				t.Fatalf("got=%v, want=%v", got, tc.want)
			}
		})
	}
}

func TestResolveSceneState(t *testing.T) {
	t.Run("empty payload returns zero", func(t *testing.T) {
		text, remaining := resolveSceneState(nil)
		if text != "" || remaining != 0 {
			t.Fatalf("got text=%q remaining=%d", text, remaining)
		}
	})
	t.Run("reads text and sticky from payload", func(t *testing.T) {
		payload := map[string]any{
			"scene_state": map[string]any{
				"text":                   "夕阳下两人坐在阳台",
				"sticky_turns_remaining": float64(15),
			},
		}
		text, remaining := resolveSceneState(payload)
		if text != "夕阳下两人坐在阳台" {
			t.Fatalf("text=%q", text)
		}
		if remaining != 15 {
			t.Fatalf("remaining=%d", remaining)
		}
	})
	t.Run("missing scene_state key returns zero", func(t *testing.T) {
		text, remaining := resolveSceneState(map[string]any{"rolling_summary": "foo"})
		if text != "" || remaining != 0 {
			t.Fatalf("got text=%q remaining=%d", text, remaining)
		}
	})
}

func TestComputeNextSceneStateFields(t *testing.T) {
	t.Run("decrements sticky when remaining > 0", func(t *testing.T) {
		prev := map[string]any{
			"scene_state": map[string]any{
				"text":                   "场景A",
				"sticky_turns_remaining": float64(10),
			},
		}
		next := computeNextSceneStateFields(prev, "场景B（新）")
		if next["text"] != "场景A" {
			t.Fatalf("text should stay, got=%v", next["text"])
		}
		if next["sticky_turns_remaining"] != 9 {
			t.Fatalf("should decrement to 9, got=%v", next["sticky_turns_remaining"])
		}
	})
	t.Run("resets with new summary when sticky exhausted", func(t *testing.T) {
		prev := map[string]any{
			"scene_state": map[string]any{
				"text":                   "旧场景",
				"sticky_turns_remaining": float64(0),
			},
		}
		next := computeNextSceneStateFields(prev, "新场景摘要内容")
		if next["text"] != "新场景摘要内容" {
			t.Fatalf("text should update, got=%v", next["text"])
		}
		if next["sticky_turns_remaining"] != sceneStateStickyDefault {
			t.Fatalf("should reset to default=%d, got=%v", sceneStateStickyDefault, next["sticky_turns_remaining"])
		}
	})
	t.Run("keeps old text when new summary empty", func(t *testing.T) {
		prev := map[string]any{
			"scene_state": map[string]any{
				"text":                   "保留旧场景",
				"sticky_turns_remaining": float64(0),
			},
		}
		next := computeNextSceneStateFields(prev, "")
		if next["text"] != "保留旧场景" {
			t.Fatalf("should keep old text, got=%v", next["text"])
		}
	})
	t.Run("nil payload with new summary initializes", func(t *testing.T) {
		next := computeNextSceneStateFields(nil, "初次场景")
		if next["text"] != "初次场景" {
			t.Fatalf("text=%v", next["text"])
		}
		if next["sticky_turns_remaining"] != sceneStateStickyDefault {
			t.Fatalf("sticky=%v", next["sticky_turns_remaining"])
		}
	})
}

func TestExtractSceneSummaryFromMemories(t *testing.T) {
	t.Run("returns empty when no scene_summary", func(t *testing.T) {
		rows := []supamemory.MemoryRow{
			{Source: "summary", Content: "CronSummary@...\n摘要内容"},
			{Source: "openilink_user", Content: "普通消息"},
		}
		got := extractSceneSummaryFromMemories(rows)
		if got != "" {
			t.Fatalf("expected empty, got=%q", got)
		}
	})
	t.Run("strips SceneSummary@ prefix", func(t *testing.T) {
		rows := []supamemory.MemoryRow{
			{Source: "scene_summary", Content: "SceneSummary@2026-05-20T00:00:00Z\n两人走进咖啡馆，窗外下雨"},
		}
		got := extractSceneSummaryFromMemories(rows)
		if got != "两人走进咖啡馆，窗外下雨" {
			t.Fatalf("got=%q", got)
		}
	})
	t.Run("returns first scene_summary in list", func(t *testing.T) {
		rows := []supamemory.MemoryRow{
			{Source: "openilink_user", Content: "消息"},
			{Source: "scene_summary", Content: "第一条场景摘要"},
			{Source: "scene_summary", Content: "第二条场景摘要"},
		}
		got := extractSceneSummaryFromMemories(rows)
		if got != "第一条场景摘要" {
			t.Fatalf("got=%q", got)
		}
	})
}

func TestSeparatePlatformMessageRows(t *testing.T) {
	rows := []supamemory.MemoryRow{
		{Source: "platform_message:user", Content: "user: 历史消息A"},
		{Source: "platform_message:assistant", Content: "assistant: 历史回复A"},
		{Source: "summary", Content: "CronSummary@...\n摘要"},
		{Source: "openilink_user", Content: "用户偏好"},
	}
	platform, other := separatePlatformMessageRows(rows)
	if len(platform) != 2 {
		t.Fatalf("platform count=%d, want 2", len(platform))
	}
	if len(other) != 2 {
		t.Fatalf("other count=%d, want 2", len(other))
	}
	for _, r := range platform {
		if !strings.HasPrefix(r.Source, "platform_message:") {
			t.Fatalf("unexpected source in platform: %q", r.Source)
		}
	}
}

func TestInjectRAGMessagesBlock(t *testing.T) {
	t.Run("empty platform rows returns original", func(t *testing.T) {
		msgs := []ai.Message{
			{Role: "system", Content: "你是助手"},
			{Role: "user", Content: "你好"},
		}
		got := injectRAGMessagesBlock(msgs, nil, ragMessagesMaxCount)
		if len(got) != 2 {
			t.Fatalf("len=%d want 2", len(got))
		}
	})
	t.Run("injects RAG block between older and recent messages", func(t *testing.T) {
		msgs := []ai.Message{
			{Role: "system", Content: "你是助手"},
			{Role: "user", Content: "消息1"},
			{Role: "assistant", Content: "回复1"},
			{Role: "user", Content: "消息2"},
			{Role: "assistant", Content: "回复2"},
			{Role: "user", Content: "消息3"},
			{Role: "assistant", Content: "回复3"},
			{Role: "user", Content: "当前消息"},
		}
		platform := []supamemory.MemoryRow{
			{Source: "platform_message:user", Content: "user: 历史相关消息X"},
		}
		got := injectRAGMessagesBlock(msgs, platform, ragMessagesMaxCount)
		// 验证有 system 分隔符
		var systemContents []string
		for _, m := range got {
			if m.Role == "system" {
				if c, ok := m.Content.(string); ok {
					systemContents = append(systemContents, c)
				}
			}
		}
		hasRAGHeader := false
		for _, c := range systemContents {
			if strings.Contains(c, "历史对话片段") {
				hasRAGHeader = true
			}
		}
		if !hasRAGHeader {
			t.Fatalf("RAG header not found in system messages: %v", systemContents)
		}
		// 最后一条消息仍为 "当前消息"
		last := got[len(got)-1]
		if c, ok := last.Content.(string); !ok || c != "当前消息" {
			t.Fatalf("last message should be current, got=%v", last.Content)
		}
	})
	t.Run("respects maxCount limit", func(t *testing.T) {
		msgs := []ai.Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "cur"}}
		platform := make([]supamemory.MemoryRow, 10)
		for i := range platform {
			platform[i] = supamemory.MemoryRow{Source: "platform_message:user", Content: "user: msg"}
		}
		got := injectRAGMessagesBlock(msgs, platform, 3)
		// 最多 3 条 RAG 消息 + 2 system 分隔 + 原始 2 条
		ragCount := 0
		for _, m := range got {
			if m.Role == "user" {
				if c, ok := m.Content.(string); ok && c == "msg" {
					ragCount++
				}
			}
		}
		if ragCount > 3 {
			t.Fatalf("RAG injected %d messages, want <= 3", ragCount)
		}
	})
}

func TestBuildMemoryPromptKeepsLongerContent(t *testing.T) {
	content := strings.Repeat("记", 200)
	got := buildMemoryPrompt([]supamemory.MemoryRow{{Source: "global", Content: content}})
	if !strings.Contains(got, strings.Repeat("记", 180)) {
		t.Fatalf("content was truncated too aggressively: %q", got)
	}
}

func TestDeriveEmotionPolicy(t *testing.T) {
	serious := deriveEmotionPolicy("这个退款和账号安全怎么处理", "calm", "友好清晰")
	if serious.State != "serious" || serious.AllowEmoji {
		t.Fatalf("serious policy=%+v", serious)
	}
	casual := deriveEmotionPolicy("哈哈这个太有趣了", "calm", "友好清晰")
	if casual.State != "excited" || !casual.AllowEmoji {
		t.Fatalf("casual policy=%+v", casual)
	}
}

func TestComposeSystemWithEmotionPolicy(t *testing.T) {
	base := "你是一个助手"
	p := emotionPolicy{
		State:      "serious",
		ToneTarget: "严谨克制",
		AllowEmoji: false,
		Reason:     "serious_topic",
	}
	got := composeSystemWithEmotionPolicy(base, p)
	if !strings.Contains(got, "情绪与语气策略") {
		t.Fatalf("missing policy block: %q", got)
	}
	if !strings.Contains(got, "表情许可: deny") {
		t.Fatalf("missing deny: %q", got)
	}
}

func TestIsBackchannelMessage(t *testing.T) {
	positives := []string{
		// Chinese
		"嗯", "嗯嗯", "嗯呢", "哦", "噢", "好", "好的", "好呀", "好吧", "好嘞", "好哒", "好滴",
		"行", "行吧", "对", "对的", "是的", "知道了", "了解", "收到", "明白", "可以", "没问题",
		"哈哈", "哈哈哈", "呵呵", "嘿嘿", "嘻嘻", "呜呜", "啊", "哇", "唉", "额", "卧槽",
		"然后呢", "是吗", "真的吗", "所以呢", "为什么", "怎么说", "接着呢", "后来呢",
		"算了", "随便", "无所谓", "都行", "不知道", "没想法", "再说吧", "随意",
		// English
		"yeah", "yep", "sure", "ok", "okay", "got it", "i see", "fine",
		"wow", "haha", "hahaha", "lol", "omg", "nice", "cool", "emmm", "emm", "umm",
		"and then?", "really?", "go on", "no way", "for real?",
		"whatever", "idk", "dunno", "nah", "meh",
		// Japanese
		"うん", "うんうん", "ええ", "はい", "そう", "なるほど", "わかった", "そっか",
		"すごい", "マジ", "へー", "笑",
		"それで", "ほんと?", "なんで?",
		"まあ", "別に", "知らない", "わからない", "いいや",
	}
	for _, p := range positives {
		if !isBackchannelMessage(p) {
			t.Errorf("expected backchannel for %q", p)
		}
	}
	negatives := []string{
		"今天天气怎么样",
		"我想去吃火锅",
		"帮我写一段代码",
		"这个项目进展如何",
		"",
		"   ",
		"嗯嗯嗯嗯嗯嗯嗯嗯嗯",
		"Can you help me with this?",
		"I want to learn programming",
		"今日の天気はどうですか",
	}
	for _, n := range negatives {
		if isBackchannelMessage(n) {
			t.Errorf("expected NOT backchannel for %q", n)
		}
	}
}

func TestBuildMemoryQueryFromMessages_Backchannel(t *testing.T) {
	messages := []ai.Message{
		{Role: "user", Content: "我最近在学吉他"},
		{Role: "assistant", Content: "学吉他很有意思啊，你学了多久了？指法练习是比较枯燥的阶段"},
		{Role: "user", Content: "嗯"},
	}
	query := buildMemoryQueryFromMessages(messages, "嗯")
	if !strings.Contains(query, "assistant:") {
		t.Errorf("backchannel query should include last assistant content, got: %s", query)
	}
	if strings.HasSuffix(strings.TrimSpace(query), "user: 嗯") {
		t.Errorf("backchannel query should NOT end with user backchannel text, got: %s", query)
	}
}

func TestLastAssistantContent(t *testing.T) {
	messages := []ai.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "你好呀"},
		{Role: "user", Content: "嗯"},
	}
	got := lastAssistantContent(messages)
	if got != "你好呀" {
		t.Errorf("expected '你好呀', got %q", got)
	}
}
