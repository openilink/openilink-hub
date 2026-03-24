package database

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TraceSpan represents a single processing step in a message trace.
type TraceSpan struct {
	Type      string `json:"type"`                // receive, match_handle, match_command, match_event, deliver_app, deliver_webhook, reply, ai, store
	Name      string `json:"name"`                // human-readable description
	Status    string `json:"status"`              // ok, error, skip, timeout, pending
	Detail    string `json:"detail,omitempty"`     // response body, error message, etc.
	DurationMs int   `json:"duration_ms,omitempty"`
	Timestamp int64  `json:"timestamp"`           // unix milliseconds
}

// MessageTrace is a complete processing record for one inbound message.
type MessageTrace struct {
	ID        int64       `json:"id"`
	BotID     string      `json:"bot_id"`
	TraceID   string      `json:"trace_id"`
	MessageID int64       `json:"message_id"`
	Sender    string      `json:"sender"`
	Content   string      `json:"content"`
	MsgType   string      `json:"msg_type"`
	Spans     []TraceSpan `json:"spans"`
	CreatedAt int64       `json:"created_at"`
}

// TraceBuilder collects spans during message processing, then flushes to DB.
type TraceBuilder struct {
	mu      sync.Mutex
	db      *DB
	botID   string
	traceID string
	msgID   int64
	sender  string
	content string
	msgType string
	spans   []TraceSpan
}

// NewTraceBuilder creates a trace for a new inbound message.
func NewTraceBuilder(db *DB, botID, sender, content, msgType string) *TraceBuilder {
	return &TraceBuilder{
		db:      db,
		botID:   botID,
		traceID: "tr_" + uuid.New().String(),
		sender:  sender,
		content: truncateStr(content, 200),
		msgType: msgType,
	}
}

func (t *TraceBuilder) TraceID() string { return t.traceID }

func (t *TraceBuilder) SetMessageID(id int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.msgID = id
}

// Add appends a completed span.
func (t *TraceBuilder) Add(spanType, name, status, detail string, durationMs int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = append(t.spans, TraceSpan{
		Type:       spanType,
		Name:       name,
		Status:     status,
		Detail:     truncateStr(detail, 512),
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	})
}

// StartTimer returns a function that, when called, adds the span with elapsed duration.
func (t *TraceBuilder) StartTimer(spanType, name string) func(status, detail string) {
	start := time.Now()
	return func(status, detail string) {
		t.Add(spanType, name, status, detail, int(time.Since(start).Milliseconds()))
	}
}

// Flush writes the trace to the database. Call once after all processing is done.
func (t *TraceBuilder) Flush() {
	t.mu.Lock()
	spans := make([]TraceSpan, len(t.spans))
	copy(spans, t.spans)
	t.mu.Unlock()

	spansJSON, _ := json.Marshal(spans)
	_, _ = t.db.Exec(`INSERT INTO message_traces (bot_id, trace_id, message_id, sender, content, msg_type, spans)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.botID, t.traceID, t.msgID, t.sender, t.content, t.msgType, spansJSON)
}

// ListTraces returns recent message traces for a bot.
func (db *DB) ListTraces(botID string, limit int) ([]MessageTrace, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT id, bot_id, trace_id, message_id, sender, content, msg_type, spans,
		EXTRACT(EPOCH FROM created_at)::BIGINT
		FROM message_traces WHERE bot_id = $1
		ORDER BY id DESC LIMIT %d`, limit), botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var traces []MessageTrace
	for rows.Next() {
		var t MessageTrace
		var spansJSON json.RawMessage
		if err := rows.Scan(&t.ID, &t.BotID, &t.TraceID, &t.MessageID, &t.Sender, &t.Content, &t.MsgType, &spansJSON, &t.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(spansJSON, &t.Spans)
		traces = append(traces, t)
	}
	return traces, rows.Err()
}
