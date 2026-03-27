package store

import (
	"encoding/json"
	"time"
)

type Bot struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Name           string          `json:"name"`
	DisplayName    string          `json:"display_name"`
	Provider       string          `json:"provider"`
	ProviderID     string          `json:"provider_id,omitempty"`
	Status         string          `json:"status"`
	Credentials    json.RawMessage `json:"credentials,omitempty"`
	SyncState      json.RawMessage `json:"-"`
	MsgCount       int64           `json:"msg_count"`
	LastMsgAt      *int64          `json:"last_msg_at,omitempty"`
	ReminderHours  int             `json:"reminder_hours"`
	LastRemindedAt *int64          `json:"last_reminded_at,omitempty"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
	AIEnabled      bool            `json:"ai_enabled"`
	AIModel        string          `json:"ai_model"`
}

type BotStats struct {
	TotalBots     int   `json:"total_bots"`
	OnlineBots    int   `json:"online_bots"`
	TotalChannels int   `json:"total_channels"`
	TotalMessages int64 `json:"total_messages"`
	ConnectedWS   int   `json:"connected_ws"`
}

type AdminStats struct {
	TotalUsers       int   `json:"total_users"`
	ActiveUsers      int   `json:"active_users"`
	TotalBots        int   `json:"total_bots"`
	OnlineBots       int   `json:"online_bots"`
	ExpiredBots      int   `json:"expired_bots"`
	TotalChannels    int   `json:"total_channels"`
	TotalMessages    int64 `json:"total_messages"`
	InboundMessages  int64 `json:"inbound_messages"`
	OutboundMessages int64 `json:"outbound_messages"`
	ConnectedWS      int   `json:"connected_ws"`
}

type RecentContact struct {
	UserID    string `json:"user_id"`
	LastMsgAt int64  `json:"last_msg_at"`
	MsgCount  int    `json:"msg_count"`
}

type BotStore interface {
	CreateBot(userID, name, provider, providerID string, credentials json.RawMessage) (*Bot, error)
	GetBot(id string) (*Bot, error)
	ListBotsByUser(userID string) ([]Bot, error)
	GetAllBots() ([]Bot, error)
	FindBotByProviderID(provider, providerID string) (*Bot, error)
	FindBotByCredential(key, value string) (*Bot, error)
	UpdateBotCredentials(id, providerID string, credentials json.RawMessage) error
	UpdateBotName(id, name string) error
	UpdateBotDisplayName(id, displayName string) error
	UpdateBotStatus(id, status string) error
	UpdateBotSyncState(id string, syncState json.RawMessage) error
	IncrBotMsgCount(id string) error
	UpdateBotReminder(id string, hours int) error
	MarkBotReminded(id string) error
	GetBotsNeedingReminder() ([]Bot, error)
	DeleteBot(id string) error
	CountBotsByUser(userID string) (int, error)
	GetAdminStats() (*AdminStats, error)
	GetBotStats(userID string) (*BotStats, error)
	ListRecentContacts(botID string, limit int) ([]RecentContact, error)
	UpdateBotAIEnabled(id string, enabled bool) error
	UpdateBotAIModel(id, model string) error
	LastActivityAt(userID string) *time.Time
}
