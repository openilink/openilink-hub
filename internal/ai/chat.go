package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openilink/openilink-hub/internal/store"
)

const defaultBaseURL = "https://api.openai.com/v1"
const defaultModel = "gpt-4o-mini"
const defaultMaxHistory = 20

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete calls the OpenAI-compatible chat completion API.
// It builds context from recent message history for the given sender.
func Complete(ctx context.Context, cfg store.AIConfig, s store.MessageStore, channelID, sender, text string) (string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	maxHistory := cfg.MaxHistory
	if maxHistory <= 0 {
		maxHistory = defaultMaxHistory
	}

	// Build conversation from message history
	var messages []chatMessage

	if cfg.SystemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: cfg.SystemPrompt})
	}

	// Load recent history scoped to this channel
	history, _ := s.ListChannelMessages(channelID, sender, maxHistory)
	// history is DESC order, reverse it
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		content := extractTextFromItems(m.ItemList)
		if content == "" {
			continue
		}
		if m.Direction == "inbound" {
			messages = append(messages, chatMessage{Role: "user", Content: content})
		} else {
			messages = append(messages, chatMessage{Role: "assistant", Content: content})
		}
	}

	// Append current message
	messages = append(messages, chatMessage{Role: "user", Content: text})

	// Call API
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	reqBody, _ := json.Marshal(chatRequest{Model: model, Messages: messages})
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ai api returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("ai response parse failed (not JSON): %s", truncate(string(body), 200))
	}

	if result.Error != nil {
		return "", fmt.Errorf("ai error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("ai returned empty response")
	}

	return result.Choices[0].Message.Content, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func extractTextFromItems(itemList json.RawMessage) string {
	var items []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	json.Unmarshal(itemList, &items)
	for _, item := range items {
		if item.Text != "" {
			return item.Text
		}
	}
	return ""
}
