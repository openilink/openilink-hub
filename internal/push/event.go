package push

import "encoding/json"

// Event types pushed to browser clients.
const (
	EventTraceCompleted = "trace_completed"
	EventMessageNew     = "message_new"
	EventWebhookLog     = "webhook_log"
	EventBotStatus      = "bot_status"
)

// Envelope is the JSON frame sent over the push WebSocket.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// BotEvent is the common payload for bot-scoped push events.
type BotEvent struct {
	BotID   string `json:"bot_id"`
	TraceID string `json:"trace_id,omitempty"`
}

// NewEnvelope builds an Envelope from a type and any marshalable data.
func NewEnvelope(typ string, data any) Envelope {
	raw, _ := json.Marshal(data)
	return Envelope{Type: typ, Data: raw}
}

// Subscribe / Unsubscribe are client→server messages.
type Subscribe struct {
	BotIDs []string `json:"bot_ids"`
}

type Unsubscribe struct {
	BotIDs []string `json:"bot_ids"`
}
