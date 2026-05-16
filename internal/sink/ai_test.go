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
	}{
		EmojiEnabled:             true,
		Text:                     "来个表情包",
		LatestConversationEmojiS: 0,
		UserEmojiCountInWindow:   0,
		LatestUserEmojiURL:       "",
		CandidateEmojiURL:        "https://cdn.example.com/a.webp",
		NowSec:                   now,
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

func TestMergeRollingSummary(t *testing.T) {
	got := mergeRollingSummary("之前摘要", "用户问了一个很长很长的问题", "助手给了一个很长很长的回答")
	if !strings.Contains(got, "之前摘要") {
		t.Fatalf("missing prev: %q", got)
	}
	if !strings.Contains(got, "用户:") || !strings.Contains(got, "助手:") {
		t.Fatalf("missing merged parts: %q", got)
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
