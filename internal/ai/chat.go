package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/openilink/openilink-hub/internal/store"
)

// ImageData holds image bytes for multimodal content.
type ImageData struct {
	Data        []byte
	ContentType string // e.g. "image/jpeg"
}

// MediaResolver reads image data by storage key.
type MediaResolver func(ctx context.Context, key string) ([]byte, error)

// contentPart is an OpenAI multimodal content part.
type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

const maxImageBytes = 20 * 1024 * 1024 // 20MB max for base64 encoding

const defaultBaseURL = "https://api.openai.com/v1"
const defaultModel = "gpt-4o-mini"
const defaultMaxHistory = 20
const MaxToolRounds = 5

// Message supports text, tool_calls, and tool results.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`     // string or null
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`  // assistant response
	ToolCallID string     `json:"tool_call_id,omitempty"` // tool result
	Name       string     `json:"name,omitempty"`        // tool result function name
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool describes an OpenAI-compatible function tool.
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool        `json:"tools,omitempty"`
}

type chatUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	PromptTokensDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details,omitempty"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   *chatUsage   `json:"usage,omitempty"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type chatChoice struct {
	Message      chatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

type chatResponseMessage struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content"`
	Thinking         *string    `json:"thinking,omitempty"`          // DeepSeek / some providers
	ReasoningContent *string    `json:"reasoning_content,omitempty"` // OpenAI o1/o3 compatible
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
}

// ToolCallRequest is returned when the LLM wants to call a tool.
type ToolCallRequest struct {
	ID        string          // tool_call ID for correlation
	Name      string          // function name
	Arguments json.RawMessage // parsed arguments
}

// ToolCallResult is the result of executing a tool call.
type ToolCallResult struct {
	ID      string      // matches ToolCallRequest.ID
	Name    string      // function name
	Content string      // result text to feed back to LLM
	Images  []ImageData // optional images to include as multimodal content
	Async   bool        // true = result will be pushed to user later, skip LLM continuation
}

// Usage holds token usage statistics from the API response.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CachedTokens     int
	ReasoningTokens  int
}

// CompletionResult holds the outcome of a completion call.
type CompletionResult struct {
	Content   string            // text reply (empty if tool_calls)
	Thinking  string            // chain-of-thought / reasoning content (may be empty)
	ToolCalls []ToolCallRequest // tool calls to execute (empty if text reply)
	Usage     *Usage            // token usage (nil if not provided by API)
}

// Complete calls the OpenAI-compatible chat completion API.
// It builds context from recent message history for the given sender.
// currentImages are pre-downloaded images for the current message.
// resolver reads image data from storage for history messages (may be nil).
// Returns text content or tool call requests.
func Complete(ctx context.Context, cfg store.AIConfig, s store.MessageStore, channelID, sender, text string, tools []Tool, currentImages []ImageData, resolver MediaResolver) (*CompletionResult, error) {
	messages := BuildMessages(ctx, cfg, s, channelID, sender, text, currentImages, resolver)
	return CompleteMessages(ctx, cfg, messages, tools)
}

// CompleteMessages calls the LLM with pre-built messages. Use this when you
// already have a messages slice (e.g. from BuildMessages) and want to avoid
// rebuilding it.
func CompleteMessages(ctx context.Context, cfg store.AIConfig, messages []Message, tools []Tool) (*CompletionResult, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	return callAPI(ctx, baseURL, cfg.APIKey, model, messages, tools, cfg.CustomHeaders)
}

// ContinueWithToolResults feeds tool results back to the LLM and gets the next response.
func ContinueWithToolResults(ctx context.Context, cfg store.AIConfig, messages []Message, results []ToolCallResult, tools []Tool) (*CompletionResult, []Message, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	// Append tool results as messages.
	// OpenAI tool messages only support text content, so images are sent
	// in a follow-up user message.
	var toolImages []ImageData
	for _, r := range results {
		content := r.Content
		if content == "" {
			content = "OK"
		}
		messages = append(messages, Message{
			Role:       "tool",
			ToolCallID: r.ID,
			Name:       r.Name,
			Content:    content,
		})
		toolImages = append(toolImages, r.Images...)
	}
	if len(toolImages) > 0 {
		messages = append(messages, Message{
			Role:    "user",
			Content: buildCurrentContent("The tool returned the following image(s). Please describe or discuss them.", toolImages),
		})
	}

	result, err := callAPI(ctx, baseURL, cfg.APIKey, model, messages, tools, cfg.CustomHeaders)
	return result, messages, err
}

// BuildMessages builds the conversation message list from history and the current message.
func BuildMessages(ctx context.Context, cfg store.AIConfig, s store.MessageStore, channelID, sender, text string, currentImages []ImageData, resolver MediaResolver) []Message {
	maxHistory := cfg.MaxHistory
	if maxHistory <= 0 {
		maxHistory = defaultMaxHistory
	}

	var messages []Message
	if cfg.SystemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: cfg.SystemPrompt})
	}

	history, _ := s.ListChannelMessages(channelID, sender, maxHistory)
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Direction == "inbound" {
			content := buildHistoryContent(ctx, m.ItemList, m.MediaKeys, resolver)
			if content == nil {
				continue
			}
			messages = append(messages, Message{Role: "user", Content: content})
		} else {
			text := extractTextFromItems(m.ItemList)
			if text == "" {
				continue
			}
			messages = append(messages, Message{Role: "assistant", Content: text})
		}
	}

	// Append current message (with optional images)
	messages = append(messages, Message{Role: "user", Content: buildCurrentContent(text, currentImages)})
	return messages
}

// AppendAssistantToolCalls appends the assistant's tool_calls message to the conversation.
func AppendAssistantToolCalls(messages []Message, calls []ToolCallRequest) []Message {
	var tcs []toolCall
	for _, c := range calls {
		tcs = append(tcs, toolCall{
			ID:   c.ID,
			Type: "function",
			Function: functionCall{
				Name:      c.Name,
				Arguments: string(c.Arguments),
			},
		})
	}
	return append(messages, Message{
		Role:      "assistant",
		ToolCalls: tcs,
	})
}

func callAPI(ctx context.Context, baseURL, apiKey, model string, messages []Message, tools []Tool, customHeaders map[string]string) (*CompletionResult, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	req := chatRequest{Model: model, Messages: messages}
	if len(tools) > 0 {
		req.Tools = tools
	}

	reqBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	for k, v := range customHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ai api returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ai response parse failed: %s", truncate(string(body), 200))
	}

	if result.Error != nil {
		return nil, fmt.Errorf("ai error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("ai returned empty response")
	}

	choice := result.Choices[0]

	// Extract thinking/reasoning content (varies by provider)
	thinking := ""
	if choice.Message.Thinking != nil && *choice.Message.Thinking != "" {
		thinking = *choice.Message.Thinking
	} else if choice.Message.ReasoningContent != nil && *choice.Message.ReasoningContent != "" {
		thinking = *choice.Message.ReasoningContent
	}

	// Convert usage if present
	var usage *Usage
	if result.Usage != nil {
		usage = &Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
		if result.Usage.PromptTokensDetails != nil {
			usage.CachedTokens = result.Usage.PromptTokensDetails.CachedTokens
		}
		if result.Usage.CompletionTokensDetails != nil {
			usage.ReasoningTokens = result.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}

	// Tool calls
	if len(choice.Message.ToolCalls) > 0 {
		var calls []ToolCallRequest
		for _, tc := range choice.Message.ToolCalls {
			calls = append(calls, ToolCallRequest{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			})
		}
		return &CompletionResult{Thinking: thinking, ToolCalls: calls, Usage: usage}, nil
	}

	// Text reply
	content := ""
	if choice.Message.Content != nil {
		content = *choice.Message.Content
	}
	return &CompletionResult{Content: content, Thinking: thinking, Usage: usage}, nil
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

// buildHistoryContent builds content for a history message.
// Returns a string for text-only, []contentPart for multimodal, or nil if empty.
func buildHistoryContent(ctx context.Context, itemList, mediaKeys json.RawMessage, resolver MediaResolver) any {
	var items []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}
	json.Unmarshal(itemList, &items)

	var keys map[string]string
	if len(mediaKeys) > 2 {
		json.Unmarshal(mediaKeys, &keys)
	}

	var text string
	var imageParts []contentPart

	for i, item := range items {
		if item.Text != "" {
			text = item.Text
		}
		if item.Type == "image" && resolver != nil && keys != nil {
			key := keys[strconv.Itoa(i)]
			if key == "" {
				continue
			}
			data, err := resolver(ctx, key)
			if err != nil || len(data) == 0 {
				continue
			}
			if len(data) > maxImageBytes {
				continue
			}
			ct := http.DetectContentType(data)
			if !strings.HasPrefix(ct, "image/") {
				continue
			}
			imageParts = append(imageParts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: imageDataURI(data)},
			})
		}
	}

	if text == "" && len(imageParts) == 0 {
		return nil
	}
	if len(imageParts) == 0 {
		return text
	}

	var parts []contentPart
	if text != "" {
		parts = append(parts, contentPart{Type: "text", Text: text})
	}
	return append(parts, imageParts...)
}

// buildCurrentContent builds content for the current message with optional images.
func buildCurrentContent(text string, images []ImageData) any {
	if len(images) == 0 {
		return text
	}
	var parts []contentPart
	if text != "" {
		parts = append(parts, contentPart{Type: "text", Text: text})
	}
	for _, img := range images {
		if len(img.Data) > maxImageBytes {
			continue
		}
		ct := img.ContentType
		if ct == "" {
			ct = http.DetectContentType(img.Data)
		}
		if !strings.HasPrefix(ct, "image/") {
			continue
		}
		parts = append(parts, contentPart{
			Type:     "image_url",
			ImageURL: &imageURL{URL: "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(img.Data)},
		})
	}
	if len(parts) == 0 {
		return text
	}
	return parts
}

func imageDataURI(data []byte) string {
	ct := http.DetectContentType(data)
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
}
