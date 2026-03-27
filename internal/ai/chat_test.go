package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
)

// mockMessageStore implements store.MessageStore for testing.
type mockMessageStore struct{}

func (m *mockMessageStore) ListChannelMessages(channelID, sender string, limit int) ([]store.Message, error) {
	return nil, nil
}
func (m *mockMessageStore) SaveMessage(_ *store.Message) (store.SaveResult, error) {
	return store.SaveResult{}, nil
}
func (m *mockMessageStore) GetMessage(_ int64) (*store.Message, error)     { return nil, nil }
func (m *mockMessageStore) ListMessages(_ string, _ int, _ int64) ([]store.Message, error) {
	return nil, nil
}
func (m *mockMessageStore) ListMessagesBySender(_, _ string, _ int) ([]store.Message, error) {
	return nil, nil
}
func (m *mockMessageStore) GetMessagesSince(_ string, _ int64, _ int) ([]store.Message, error) {
	return nil, nil
}
func (m *mockMessageStore) GetLatestContextToken(_ string) string                        { return "" }
func (m *mockMessageStore) HasFreshContextToken(_ string, _ time.Duration) bool          { return false }
func (m *mockMessageStore) BatchHasFreshContextToken(_ []string, _ time.Duration) map[string]bool {
	return nil
}
func (m *mockMessageStore) UpdateMediaStatus(_, _ string, _ json.RawMessage) error   { return nil }
func (m *mockMessageStore) UpdateMediaStatusByID(_ int64, _ string, _ json.RawMessage) error {
	return nil
}
func (m *mockMessageStore) UpdateMessagePayload(_ int64, _ json.RawMessage) error    { return nil }
func (m *mockMessageStore) UpdateMediaPayloads(_, _ string, _ json.RawMessage) error { return nil }
func (m *mockMessageStore) MarkProcessed(_ int64) error                              { return nil }
func (m *mockMessageStore) GetUnprocessedMessages(_ string, _ int) ([]store.Message, error) {
	return nil, nil
}
func (m *mockMessageStore) PruneMessages(_ int) (int64, error) { return 0, nil }

// ==================== Tests ====================

func TestComplete_TextReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Content != "Hello!" {
		t.Errorf("content = %q, want %q", result.Content, "Hello!")
	}
	if len(result.ToolCalls) != 0 {
		t.Errorf("tool_calls = %d, want 0", len(result.ToolCalls))
	}
}

func TestComplete_ToolCall(t *testing.T) {
	var gotToolCount int
	var gotToolName string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		gotToolCount = len(req.Tools)
		if len(req.Tools) > 0 {
			gotToolName = req.Tools[0].Function.Name
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{
							{
								"id":   "call_123",
								"type": "function",
								"function": map[string]any{
									"name":      "cmd.list_prs",
									"arguments": `{"repo":"openilink/hub","state":"open"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	tools := []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "cmd.list_prs",
				Description: "List pull requests",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"repo":{"type":"string"},"state":{"type":"string"}}}`),
			},
		},
	}

	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "show PRs", tools)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Verify tools were sent
	if gotToolCount != 1 {
		t.Errorf("tools sent = %d, want 1", gotToolCount)
	}
	if gotToolName != "cmd.list_prs" {
		t.Errorf("tool name sent = %q, want %q", gotToolName, "cmd.list_prs")
	}

	// Verify result
	if result.Content != "" {
		t.Errorf("content should be empty, got %q", result.Content)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(result.ToolCalls))
	}
	tc := result.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("id = %q, want %q", tc.ID, "call_123")
	}
	if tc.Name != "cmd.list_prs" {
		t.Errorf("name = %q, want %q", tc.Name, "cmd.list_prs")
	}
	var args map[string]string
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["repo"] != "openilink/hub" {
		t.Errorf("args.repo = %q, want %q", args["repo"], "openilink/hub")
	}
	if args["state"] != "open" {
		t.Errorf("args.state = %q, want %q", args["state"], "open")
	}
}

func TestContinueWithToolResults(t *testing.T) {
	var callCount atomic.Int32
	var hasAssistantToolCalls, hasToolResult bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{"id": "call_abc", "type": "function", "function": map[string]any{"name": "cmd.weather", "arguments": `{"city":"Tokyo"}`}},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		} else {
			// Verify assistant tool_calls message + tool result message
			for _, msg := range req.Messages {
				if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						if tc.Function.Name == "cmd.weather" && tc.ID == "call_abc" {
							hasAssistantToolCalls = true
						}
					}
				}
				if msg.Role == "tool" && msg.ToolCallID == "call_abc" {
					content, _ := msg.Content.(string)
					if content == "Sunny, 25°C" {
						hasToolResult = true
					}
				}
			}

			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "Tokyo is sunny, 25°C."}, "finish_reason": "stop"},
				},
			})
		}
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	tools := []Tool{{Type: "function", Function: ToolFunction{Name: "cmd.weather", Description: "Get weather"}}}

	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "weather?", tools)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected tool_call, got text: %q", result.Content)
	}

	messages := BuildMessages(cfg, &mockMessageStore{}, "ch1", "user1", "weather?")
	messages = AppendAssistantToolCalls(messages, result.ToolCalls)
	result2, _, err := ContinueWithToolResults(context.Background(), cfg, messages, []ToolCallResult{
		{ID: "call_abc", Name: "cmd.weather", Content: "Sunny, 25°C"},
	}, tools)
	if err != nil {
		t.Fatalf("ContinueWithToolResults: %v", err)
	}
	if result2.Content != "Tokyo is sunny, 25°C." {
		t.Errorf("final = %q", result2.Content)
	}

	// Verify message chain was correct
	if !hasAssistantToolCalls {
		t.Error("continuation request missing assistant tool_calls message")
	}
	if !hasToolResult {
		t.Error("continuation request missing tool result message")
	}
}

func TestComplete_MultiRoundToolCalls(t *testing.T) {
	var callCount atomic.Int32
	var round2HasCall1Result, round3HasCall2Result bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch n {
		case 1:
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "tool_calls": []map[string]any{
						{"id": "call_1", "type": "function", "function": map[string]any{"name": "cmd.list", "arguments": `{}`}},
					}}, "finish_reason": "tool_calls"},
				},
			})
		case 2:
			// Verify call_1 result is present
			for _, msg := range req.Messages {
				if msg.Role == "tool" && msg.ToolCallID == "call_1" {
					round2HasCall1Result = true
				}
			}
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "tool_calls": []map[string]any{
						{"id": "call_2", "type": "function", "function": map[string]any{"name": "cmd.detail", "arguments": `{"id":"pr-42"}`}},
					}}, "finish_reason": "tool_calls"},
				},
			})
		case 3:
			// Verify call_2 result is present
			for _, msg := range req.Messages {
				if msg.Role == "tool" && msg.ToolCallID == "call_2" {
					round3HasCall2Result = true
				}
			}
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "content": "PR #42 is a bug fix."}, "finish_reason": "stop"},
				},
			})
		}
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	tools := []Tool{
		{Type: "function", Function: ToolFunction{Name: "cmd.list", Description: "List items"}},
		{Type: "function", Function: ToolFunction{Name: "cmd.detail", Description: "Get detail"}},
	}

	messages := BuildMessages(cfg, &mockMessageStore{}, "ch1", "user1", "tell me about the latest PR")
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "tell me about the latest PR", tools)
	if err != nil {
		t.Fatalf("round 1: %v", err)
	}

	for round := 0; round < MaxToolRounds && len(result.ToolCalls) > 0; round++ {
		messages = AppendAssistantToolCalls(messages, result.ToolCalls)
		var results []ToolCallResult
		for _, tc := range result.ToolCalls {
			results = append(results, ToolCallResult{ID: tc.ID, Name: tc.Name, Content: "mock result for " + tc.Name})
		}
		result, messages, err = ContinueWithToolResults(context.Background(), cfg, messages, results, tools)
		if err != nil {
			t.Fatalf("round %d: %v", round+1, err)
		}
	}

	if result.Content != "PR #42 is a bug fix." {
		t.Errorf("final = %q", result.Content)
	}
	if callCount.Load() != 3 {
		t.Errorf("api calls = %d, want 3", callCount.Load())
	}
	if !round2HasCall1Result {
		t.Error("round 2 missing call_1 tool result")
	}
	if !round3HasCall2Result {
		t.Error("round 3 missing call_2 tool result")
	}
}

func TestComplete_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":{"message":"internal server error"}}`))
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	_, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestComplete_ErrorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "rate limit exceeded"},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	_, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err == nil {
		t.Fatal("expected error for error JSON response")
	}
}

func TestComplete_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	_, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestComplete_MaxToolRoundsExceeded(t *testing.T) {
	// Mock that always returns tool_calls, never text
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "tool_calls": []map[string]any{
					{"id": "call_" + string(rune('0'+n)), "type": "function", "function": map[string]any{"name": "cmd.loop", "arguments": `{}`}},
				}}, "finish_reason": "tool_calls"},
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	tools := []Tool{{Type: "function", Function: ToolFunction{Name: "cmd.loop", Description: "Loop forever"}}}

	messages := BuildMessages(cfg, &mockMessageStore{}, "ch1", "user1", "loop")
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "loop", tools)
	if err != nil {
		t.Fatalf("initial: %v", err)
	}

	rounds := 0
	for rounds < MaxToolRounds && len(result.ToolCalls) > 0 {
		messages = AppendAssistantToolCalls(messages, result.ToolCalls)
		var results []ToolCallResult
		for _, tc := range result.ToolCalls {
			results = append(results, ToolCallResult{ID: tc.ID, Name: tc.Name, Content: "still looping"})
		}
		result, messages, err = ContinueWithToolResults(context.Background(), cfg, messages, results, tools)
		if err != nil {
			t.Fatalf("round %d: %v", rounds+1, err)
		}
		rounds++
	}

	// Should stop at MaxToolRounds even though LLM keeps returning tool_calls
	if rounds != MaxToolRounds {
		t.Errorf("rounds = %d, want %d", rounds, MaxToolRounds)
	}
	// Total API calls: 1 initial + MaxToolRounds continuations
	if int(callCount.Load()) != MaxToolRounds+1 {
		t.Errorf("api calls = %d, want %d", callCount.Load(), MaxToolRounds+1)
	}
}

func TestComplete_UsageParsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hi!"}, "finish_reason": "stop"},
			},
			"usage": map[string]any{
				"prompt_tokens":     100,
				"completion_tokens": 50,
				"total_tokens":      150,
				"prompt_tokens_details": map[string]any{
					"cached_tokens": 30,
				},
				"completion_tokens_details": map[string]any{
					"reasoning_tokens": 20,
				},
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if result.Usage.PromptTokens != 100 {
		t.Errorf("prompt_tokens = %d, want 100", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 50 {
		t.Errorf("completion_tokens = %d, want 50", result.Usage.CompletionTokens)
	}
	if result.Usage.TotalTokens != 150 {
		t.Errorf("total_tokens = %d, want 150", result.Usage.TotalTokens)
	}
	if result.Usage.CachedTokens != 30 {
		t.Errorf("cached_tokens = %d, want 30", result.Usage.CachedTokens)
	}
	if result.Usage.ReasoningTokens != 20 {
		t.Errorf("reasoning_tokens = %d, want 20", result.Usage.ReasoningTokens)
	}
}

func TestComplete_UsageNilWhenMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hi!"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Usage != nil {
		t.Errorf("expected usage to be nil when not in response, got %+v", result.Usage)
	}
}

func TestComplete_UsagePartialDetails(t *testing.T) {
	// Most real-world models return usage without the detail sub-objects
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hi!"}, "finish_reason": "stop"},
			},
			"usage": map[string]any{
				"prompt_tokens":     80,
				"completion_tokens": 20,
				"total_tokens":      100,
				// no prompt_tokens_details or completion_tokens_details
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if result.Usage.PromptTokens != 80 {
		t.Errorf("prompt_tokens = %d, want 80", result.Usage.PromptTokens)
	}
	if result.Usage.TotalTokens != 100 {
		t.Errorf("total_tokens = %d, want 100", result.Usage.TotalTokens)
	}
	if result.Usage.CachedTokens != 0 {
		t.Errorf("cached_tokens = %d, want 0 (no details)", result.Usage.CachedTokens)
	}
	if result.Usage.ReasoningTokens != 0 {
		t.Errorf("reasoning_tokens = %d, want 0 (no details)", result.Usage.ReasoningTokens)
	}
}

func TestComplete_UsageOnToolCallResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{
							{"id": "call_1", "type": "function", "function": map[string]any{"name": "cmd.foo", "arguments": `{}`}},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     60,
				"completion_tokens": 10,
				"total_tokens":      70,
			},
		})
	}))
	defer srv.Close()

	cfg := store.AIConfig{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "Hi", nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(result.ToolCalls) == 0 {
		t.Fatal("expected tool_calls")
	}
	if result.Usage == nil {
		t.Fatal("expected usage to be populated on tool-call response")
	}
	if result.Usage.TotalTokens != 70 {
		t.Errorf("total_tokens = %d, want 70", result.Usage.TotalTokens)
	}
}

// ==================== Real API test (skipped unless env vars set) ====================

func TestCompleteWithRealAPI(t *testing.T) {
	baseURL := os.Getenv("TEST_AI_BASE_URL")
	apiKey := os.Getenv("TEST_AI_API_KEY")
	if baseURL == "" || apiKey == "" {
		t.Skip("TEST_AI_BASE_URL and TEST_AI_API_KEY not set")
	}

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

	result, err := Complete(context.Background(), cfg, db, "nonexistent-channel", "test-sender", "Hello, what is 1+1?", nil)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if result.Content == "" {
		t.Fatal("got empty reply")
	}
	t.Logf("AI reply: %s", result.Content)
}
