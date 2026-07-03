package store

import (
	"encoding/json"
	"time"
)

const (
	OutboxEventMessageInbound      = "message_inbound"
	OutboxEventMessageOutbound     = "message_outbound"
	OutboxEventPromptProfileChange = "prompt_profile_changed"
	OutboxEventBindingInvalidated  = "binding_invalidated"
)

const (
	OutboxStatusPending    = "pending"
	OutboxStatusProcessing = "processing"
	OutboxStatusSent       = "sent"
	OutboxStatusDead       = "dead"
)

type SyncOutboxEvent struct {
	ID           int64           `json:"id"`
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	PartitionKey string          `json:"partition_key"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	RetryCount   int             `json:"retry_count"`
	NextRetryAt  int64           `json:"next_retry_at"`
	LastError    string          `json:"last_error,omitempty"`
	CreatedAt    int64           `json:"created_at"`
	UpdatedAt    int64           `json:"updated_at"`
	SentAt       int64           `json:"sent_at,omitempty"`
}

type EnqueueOutboxInput struct {
	EventID      string
	EventType    string
	PartitionKey string
	Payload      json.RawMessage
}

type ClaimOutboxOptions struct {
	Limit int
	Now   time.Time
}

type RetryOutboxInput struct {
	EventID     string
	RetryCount  int
	NextRetryAt time.Time
	LastError   string
	Status      string
}

type SyncOutboxStore interface {
	EnqueueSyncOutboxEvent(in EnqueueOutboxInput) (*SyncOutboxEvent, bool, error)
	ClaimPendingSyncOutboxEvents(opt ClaimOutboxOptions) ([]SyncOutboxEvent, error)
	MarkSyncOutboxEventSent(eventID string, sentAt time.Time) error
	MarkSyncOutboxEventRetry(in RetryOutboxInput) error
	MarkSyncOutboxEventDead(eventID, lastError string) error
}
