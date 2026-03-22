package relay

import "encoding/json"

// Envelope is the JSON frame exchanged over WebSocket.
type Envelope struct {
	Type  string          `json:"type"`
	ReqID string          `json:"req_id,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// --- Server → Client ---

type MessageData struct {
	MessageID  int64          `json:"message_id"`
	FromUserID string         `json:"from_user_id"`
	Timestamp  int64          `json:"timestamp"`
	Items      []MessageItem  `json:"items"`
	ContextToken string       `json:"context_token,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
}

type MessageItem struct {
	Type     string `json:"type"` // text, image, voice, file, video
	Text     string `json:"text,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

type BotStatusData struct {
	BotID  string `json:"bot_id"`
	Status string `json:"status"`
}

type SendAckData struct {
	ReqID    string `json:"req_id"`
	Success  bool   `json:"success"`
	ClientID string `json:"client_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Client → Server ---

type SendTextData struct {
	ToUserID string `json:"to_user_id"`
	Text     string `json:"text"`
}

type SendTypingData struct {
	ToUserID string `json:"to_user_id"`
	Status   string `json:"status"` // "typing" or "cancel"
}

// Helpers to build envelopes

func NewEnvelope(typ string, data any) Envelope {
	raw, _ := json.Marshal(data)
	return Envelope{Type: typ, Data: raw}
}

func NewAck(reqID string, success bool, clientID, errMsg string) Envelope {
	return NewEnvelope("send_ack", SendAckData{
		ReqID: reqID, Success: success, ClientID: clientID, Error: errMsg,
	})
}
