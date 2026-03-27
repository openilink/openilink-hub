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
// Only ListChannelMessages is used by ai.Complete; rest are stubs.
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

func TestComplete_TextReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "cmd.list_prs" {
			t.Errorf("tool name = %q", req.Tools[0].Function.Name)
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
	json.Unmarshal(tc.Arguments, &args)
	if args["repo"] != "openilink/hub" {
		t.Errorf("args.repo = %q", args["repo"])
	}
}

func TestContinueWithToolResults(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// Return tool_call
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
			// Verify tool result message is present
			hasToolMsg := false
			for _, msg := range req.Messages {
				if msg.Role == "tool" && msg.ToolCallID == "call_abc" {
					hasToolMsg = true
					content, _ := msg.Content.(string)
					if content != "Sunny, 25°C" {
						t.Errorf("tool result = %v, want %q", msg.Content, "Sunny, 25°C")
					}
				}
			}
			if !hasToolMsg {
				t.Error("expected tool message in continuation")
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

	// Step 1: initial call returns tool_call
	result, err := Complete(context.Background(), cfg, &mockMessageStore{}, "ch1", "user1", "weather?", tools)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected tool_call, got text: %q", result.Content)
	}

	// Step 2: feed tool result
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
}

func TestComplete_MultiRoundToolCalls(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch n {
		case 1:
			// Round 1: call tool A
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "tool_calls": []map[string]any{
						{"id": "call_1", "type": "function", "function": map[string]any{"name": "cmd.list", "arguments": `{}`}},
					}}, "finish_reason": "tool_calls"},
				},
			})
		case 2:
			// Round 2: call tool B based on result of A
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"role": "assistant", "tool_calls": []map[string]any{
						{"id": "call_2", "type": "function", "function": map[string]any{"name": "cmd.detail", "arguments": `{"id":"pr-42"}`}},
					}}, "finish_reason": "tool_calls"},
				},
			})
		case 3:
			// Final: text reply
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

	// Simulate the full loop that AI sink does
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
}

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
