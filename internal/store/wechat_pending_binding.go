package store

import "time"

const (
	WechatPendingBindingStatusPending   = "pending_finalize"
	WechatPendingBindingStatusFinalized = "finalized"
	WechatPendingBindingStatusExpired   = "expired"
	WechatPendingBindingStatusFailed    = "failed"
)

type WechatPendingBinding struct {
	ID                  int64  `json:"id"`
	EventID             string `json:"event_id"`
	ProviderBotID       string `json:"provider_bot_id"`
	BotID               string `json:"bot_id"`
	BindingID           string `json:"binding_id"`
	RoleID              string `json:"role_id"`
	SessionID           string `json:"session_id"`
	Status              string `json:"status"`
	ExternalChatID      string `json:"external_chat_id,omitempty"`
	LastFinalizeEventID string `json:"last_finalize_event_id,omitempty"`
	LastError           string `json:"last_error,omitempty"`
	CreatedAt           int64  `json:"created_at"`
	ExpiresAt           int64  `json:"expires_at"`
	FinalizedAt         int64  `json:"finalized_at,omitempty"`
	UpdatedAt           int64  `json:"updated_at"`
}

type WechatPendingBindingCreateInput struct {
	EventID       string
	ProviderBotID string
	BotID         string
	BindingID     string
	RoleID        string
	SessionID     string
	Status        string
	ExpiresAt     time.Time
}

type WechatPendingBindingStore interface {
	CreateWechatPendingBinding(in WechatPendingBindingCreateInput) (*WechatPendingBinding, bool, error)
	GetLatestPendingWechatBinding(botID, providerBotID string, now time.Time) (*WechatPendingBinding, error)
	FinalizeWechatPendingBinding(id int64, externalChatID, finalizeEventID string, finalizedAt time.Time) (bool, error)
	MarkWechatPendingBindingRetry(id int64, finalizeEventID, lastError string, now time.Time) error
}

