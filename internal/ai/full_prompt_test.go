package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

type historyStore struct {
	messages []store.Message
}

func (h *historyStore) ListChannelMessages(channelID, sender string, limit int) ([]store.Message, error) {
	if len(h.messages) > limit {
		return h.messages[:limit], nil
	}
	return h.messages, nil
}
func (h *historyStore) SaveMessage(*store.Message) (store.SaveResult, error) {
	return store.SaveResult{}, nil
}
func (h *historyStore) GetMessage(int64) (*store.Message, error) { return nil, nil }
func (h *historyStore) ListMessages(string, int, int64) ([]store.Message, error) {
	return nil, nil
}
func (h *historyStore) ListMessagesBySender(string, string, int) ([]store.Message, error) {
	return nil, nil
}
func (h *historyStore) GetMessagesSince(string, int64, int) ([]store.Message, error) { return nil, nil }
func (h *historyStore) GetLatestContextToken(string) string                          { return "" }
func (h *historyStore) HasFreshContextToken(string, time.Duration) bool              { return false }
func (h *historyStore) BatchHasFreshContextToken([]string, time.Duration) map[string]bool {
	return nil
}
func (h *historyStore) UpdateMediaStatus(string, string, json.RawMessage) error { return nil }
func (h *historyStore) UpdateMediaStatusByID(int64, string, json.RawMessage) error {
	return nil
}
func (h *historyStore) UpdateMessagePayload(int64, json.RawMessage) error { return nil }
func (h *historyStore) UpdateMediaPayloads(string, string, json.RawMessage) error {
	return nil
}
func (h *historyStore) MarkProcessed(int64) error { return nil }
func (h *historyStore) GetUnprocessedMessages(string, int) ([]store.Message, error) {
	return nil, nil
}
func (h *historyStore) PruneMessages(int) (int64, error) { return 0, nil }

func TestBuildMessages_UsesFinalSystemPromptOnly(t *testing.T) {
	hs := &historyStore{}
	cfg := store.AIConfig{SystemPrompt: "FULL_PROMPT_FINAL"}
	msgs := BuildMessages(context.Background(), cfg, hs, "ch1", "wx1", "hello", nil, nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Fatalf("first role=%s, want system", msgs[0].Role)
	}
	content, _ := msgs[0].Content.(string)
	if content != "FULL_PROMPT_FINAL" {
		t.Fatalf("system prompt=%q, want FULL_PROMPT_FINAL", content)
	}
}

func TestBuildMessages_HistoryAndCurrentInput(t *testing.T) {
	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": "history user"}})
	hs := &historyStore{messages: []store.Message{{Direction: "inbound", ItemList: itemList}}}
	cfg := store.AIConfig{SystemPrompt: "SP"}
	msgs := BuildMessages(context.Background(), cfg, hs, "ch1", "wx1", "current", nil, nil)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[1].Role != "user" || msgs[2].Role != "user" {
		t.Fatalf("unexpected roles: %#v", msgs)
	}
}
