package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const (
	PromptProfileStatusActive   = "active"
	PromptProfileStatusInactive = "inactive"
)

type PromptProfile struct {
	ID              int64  `json:"id"`
	BotID           string `json:"bot_id"`
	SenderUserID    string `json:"sender_user_id"`
	BindingID       string `json:"binding_id"`
	SystemPrompt    string `json:"system_prompt,omitempty"`
	UserPrompt      string `json:"user_prompt,omitempty"`
	FullPrompt      string `json:"full_prompt,omitempty"`
	FullPromptHash  string `json:"full_prompt_hash,omitempty"`
	PromptVersion   int64  `json:"prompt_version"`
	SourceUpdatedAt int64  `json:"source_updated_at"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type PromptProfileUpsertInput struct {
	BotID           string
	SenderUserID    string
	BindingID       string
	SystemPrompt    string
	UserPrompt      string
	FullPrompt      string
	PromptVersion   int64
	SourceUpdatedAt int64
	Status          string
}

// TruncatedPromptResult indicates whether prompt truncation was applied.
type TruncatedPromptResult struct {
	Value     string
	Truncated bool
}

func NormalizePromptProfileStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PromptProfileStatusInactive:
		return PromptProfileStatusInactive
	default:
		return PromptProfileStatusActive
	}
}

func IsValidPromptVersion(nextVersion, nextSourceUpdatedAt, currVersion, currSourceUpdatedAt int64) bool {
	if nextVersion > currVersion {
		return true
	}
	if nextVersion < currVersion {
		return false
	}
	return nextSourceUpdatedAt > currSourceUpdatedAt
}

func NormalizePromptByMaxBytes(prompt string, maxBytes int) TruncatedPromptResult {
	if maxBytes <= 0 {
		maxBytes = 8192
	}
	b := []byte(prompt)
	if len(b) <= maxBytes {
		return TruncatedPromptResult{Value: prompt}
	}
	return TruncatedPromptResult{Value: string(b[:maxBytes]), Truncated: true}
}

func IsBlankPrompt(prompt string) bool {
	return strings.TrimSpace(prompt) == ""
}

func HashPrompt(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

func HashPrefix(hash string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(hash) <= n {
		return hash
	}
	return hash[:n]
}

func ToUnixSeconds(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

type PromptProfileStore interface {
	UpsertPromptProfile(in PromptProfileUpsertInput) (*PromptProfile, bool, error)
	GetActivePromptProfile(botID, senderUserID string) (*PromptProfile, error)
	InvalidatePromptProfile(botID, senderUserID, bindingID string) (bool, error)
	GetPromptProfile(botID, senderUserID, bindingID string) (*PromptProfile, error)
}
