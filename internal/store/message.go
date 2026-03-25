package store

import (
	"encoding/json"
	"time"
)

type Message struct {
	ID        int64   `json:"id"`
	BotID     string  `json:"bot_id"`
	ChannelID *string `json:"channel_id,omitempty"`
	Direction string  `json:"direction"`

	Seq          *int64          `json:"seq,omitempty"`
	MessageID    *int64          `json:"message_id,omitempty"`
	FromUserID   string          `json:"from_user_id"`
	ToUserID     string          `json:"to_user_id,omitempty"`
	ClientID     string          `json:"client_id,omitempty"`
	CreateTimeMs *int64          `json:"create_time_ms,omitempty"`
	UpdateTimeMs *int64          `json:"update_time_ms,omitempty"`
	DeleteTimeMs *int64          `json:"delete_time_ms,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	GroupID      string          `json:"group_id,omitempty"`
	MessageType  int             `json:"message_type,omitempty"`
	MessageState int             `json:"message_state,omitempty"`
	ItemList     json.RawMessage `json:"item_list"`
	ContextToken string          `json:"context_token,omitempty"`

	MediaStatus string           `json:"media_status,omitempty"`
	MediaKeys   json.RawMessage  `json:"media_keys,omitempty"`
	Raw         *json.RawMessage `json:"raw,omitempty"`

	CreatedAt int64 `json:"created_at"`
}

// SaveResult holds the result of a SaveMessage upsert.
type SaveResult struct {
	ID       int64
	Inserted bool
}

type MessageStore interface {
	SaveMessage(m *Message) (SaveResult, error)
	GetMessage(id int64) (*Message, error)
	ListMessages(botID string, limit int, beforeID int64) ([]Message, error)
	ListMessagesBySender(botID, sender string, limit int) ([]Message, error)
	ListChannelMessages(channelID, sender string, limit int) ([]Message, error)
	GetMessagesSince(botID string, afterSeq int64, limit int) ([]Message, error)
	GetLatestContextToken(botID string) string
	HasFreshContextToken(botID string, maxAge time.Duration) bool
	BatchHasFreshContextToken(botIDs []string, maxAge time.Duration) map[string]bool
	UpdateMediaStatus(botID, status string, keys json.RawMessage) error
	UpdateMediaStatusByID(id int64, status string, keys json.RawMessage) error
	UpdateMessagePayload(id int64, payload json.RawMessage) error
	UpdateMediaPayloads(botID, eqp string, newPayload json.RawMessage) error
	MarkProcessed(id int64) error
	GetUnprocessedMessages(botID string, limit int) ([]Message, error)
	PruneMessages(maxAgeDays int) (int64, error)
}
