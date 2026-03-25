package store

type FilterRule struct {
	UserIDs      []string `json:"user_ids,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	MessageTypes []string `json:"message_types,omitempty"`
}

type AIConfig struct {
	Enabled      bool   `json:"enabled"`
	Source       string `json:"source,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	Model        string `json:"model,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	MaxHistory   int    `json:"max_history,omitempty"`
}

type WebhookConfig struct {
	URL       string       `json:"url,omitempty"`
	Auth      *WebhookAuth `json:"auth,omitempty"`
	PluginID  string       `json:"plugin_id,omitempty"`
	VersionID string       `json:"version_id,omitempty"`
	Script    string       `json:"script,omitempty"`
}

type WebhookAuth struct {
	Type   string `json:"type"`
	Token  string `json:"token,omitempty"`
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Secret string `json:"secret,omitempty"`
}

type Channel struct {
	ID            string        `json:"id"`
	BotID         string        `json:"bot_id"`
	Name          string        `json:"name"`
	Handle        string        `json:"handle"`
	AIConfig      AIConfig      `json:"ai_config"`
	WebhookConfig WebhookConfig `json:"webhook_config"`
	APIKey        string        `json:"api_key"`
	FilterRule    FilterRule    `json:"filter_rule"`
	Enabled       bool          `json:"enabled"`
	LastSeq       int64         `json:"last_seq"`
	CreatedAt     int64         `json:"created_at"`
	UpdatedAt     int64         `json:"updated_at"`
}

type ChannelStore interface {
	CreateChannel(botID, name, handle string, filter *FilterRule, ai *AIConfig) (*Channel, error)
	GetChannel(id string) (*Channel, error)
	GetChannelByAPIKey(apiKey string) (*Channel, error)
	ListChannelsByBot(botID string) ([]Channel, error)
	ListChannelsByBotIDs(botIDs []string) ([]Channel, error)
	UpdateChannel(id, name, handle string, filter *FilterRule, ai *AIConfig, webhook *WebhookConfig, enabled bool) error
	DeleteChannel(id string) error
	RotateChannelKey(id string) (string, error)
	UpdateChannelLastSeq(channelID string, seq int64) error
	CountChannelsByBot(botID string) (int, error)
}
