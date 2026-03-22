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
	SeqID        int64         `json:"seq_id"`
	ExternalID   string        `json:"external_id,omitempty"`
	Sender       string        `json:"sender"`
	Recipient    string        `json:"recipient,omitempty"`
	GroupID      string        `json:"group_id,omitempty"`
	Timestamp    int64         `json:"timestamp"`
	MessageState int           `json:"message_state,omitempty"` // 0=new, 1=generating, 2=finish
	Items        []MessageItem `json:"items"`
	ContextToken string        `json:"context_token,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
}

type InitData struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	BotID       string `json:"bot_id"`
	BotStatus   string `json:"bot_status"`
}

type MessageItem struct {
	Type     string   `json:"type"` // text, image, voice, file, video
	Text     string   `json:"text,omitempty"`
	FileName string   `json:"file_name,omitempty"`
	Media    *Media   `json:"media,omitempty"`
	RefMsg   *RefMsg  `json:"ref_msg,omitempty"`
}

type Media struct {
	URL         string `json:"url,omitempty"`
	AESKey      string `json:"aes_key,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	PlayTime    int    `json:"play_time,omitempty"`
	PlayLength  int    `json:"play_length,omitempty"`
	ThumbWidth  int    `json:"thumb_width,omitempty"`
	ThumbHeight int    `json:"thumb_height,omitempty"`
}

type RefMsg struct {
	Title string      `json:"title,omitempty"`
	Item  MessageItem `json:"item"`
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
	Recipient string `json:"recipient"`
	Text      string `json:"text"`
}

type SendTypingData struct {
	Ticket string `json:"ticket"`
	Status string `json:"status"` // "typing" or "cancel"
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
