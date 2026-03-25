package store

import "encoding/json"

type WebhookLog struct {
	ID             int64           `json:"id"`
	BotID          string          `json:"bot_id"`
	ChannelID      string          `json:"channel_id"`
	MessageID      *int64          `json:"message_id,omitempty"`
	PluginID       string          `json:"plugin_id,omitempty"`
	PluginVersion  string          `json:"plugin_version,omitempty"`
	Status         string          `json:"status"`
	RequestURL     string          `json:"request_url,omitempty"`
	RequestMethod  string          `json:"request_method,omitempty"`
	RequestBody    string          `json:"request_body,omitempty"`
	ResponseStatus int             `json:"response_status,omitempty"`
	ResponseBody   string          `json:"response_body,omitempty"`
	ScriptError    string          `json:"script_error,omitempty"`
	Replies        json.RawMessage `json:"replies"`
	DurationMs     int             `json:"duration_ms"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
}

type WebhookLogStore interface {
	CreateWebhookLog(log *WebhookLog) (int64, error)
	UpdateWebhookLogRequest(id int64, status, url, method, body string) error
	UpdateWebhookLogResponse(id int64, status string, respStatus int, respBody string, durationMs int) error
	UpdateWebhookLogResult(id int64, status, scriptError string, replies []string) error
	ListWebhookLogs(botID, channelID string, limit int) ([]WebhookLog, error)
	CleanOldWebhookLogs(days int) error
}
