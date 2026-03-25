package sqlite

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

func genSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (db *DB) InsertSpan(traceID, spanID, parentSpanID, name, kind, statusCode, statusMessage string,
	startTime, endTime int64, attrsJSON, eventsJSON []byte, botID string) error {
	_, err := db.Exec(`INSERT INTO trace_spans
		(trace_id, span_id, parent_span_id, name, kind, status_code, status_message, start_time, end_time, attributes, events, bot_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		traceID, spanID, parentSpanID, name, kind,
		statusCode, statusMessage, startTime, endTime,
		attrsJSON, eventsJSON, botID)
	return err
}

func (db *DB) AppendSpan(traceID, botID, name, kind, statusCode, statusMessage string, attrs map[string]any) error {
	var parentSpanID string
	_ = db.QueryRow("SELECT span_id FROM trace_spans WHERE trace_id=? AND parent_span_id='' LIMIT 1", traceID).Scan(&parentSpanID)

	attrsJSON, _ := json.Marshal(attrs)
	now := time.Now().UnixMilli()
	_, err := db.Exec(`INSERT INTO trace_spans
		(trace_id, span_id, parent_span_id, name, kind, status_code, status_message, start_time, end_time, attributes, events, bot_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,'[]',?)`,
		traceID, genSpanID(), parentSpanID, name, kind, statusCode, statusMessage, now, now, attrsJSON, botID)
	return err
}

func (db *DB) ListRootSpans(botID string, limit int) ([]store.TraceSpan, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT id, trace_id, span_id, parent_span_id, name, kind,
		status_code, status_message, start_time, end_time, attributes, events, bot_id,
		created_at
		FROM trace_spans WHERE bot_id = ? AND parent_span_id = ''
		ORDER BY id DESC LIMIT %d`, limit), botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSpans(rows)
}

func (db *DB) ListSpansByTrace(traceID string) ([]store.TraceSpan, error) {
	rows, err := db.Query(`SELECT id, trace_id, span_id, parent_span_id, name, kind,
		status_code, status_message, start_time, end_time, attributes, events, bot_id,
		created_at
		FROM trace_spans WHERE trace_id = ?
		ORDER BY start_time`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSpans(rows)
}

func scanSpans(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]store.TraceSpan, error) {
	var spans []store.TraceSpan
	for rows.Next() {
		var s store.TraceSpan
		var attrsJSON, eventsJSON json.RawMessage
		if err := rows.Scan(&s.ID, &s.TraceID, &s.SpanID, &s.ParentSpanID, &s.Name, &s.Kind,
			&s.StatusCode, &s.StatusMessage, &s.StartTime, &s.EndTime,
			&attrsJSON, &eventsJSON, &s.BotID, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(attrsJSON, &s.Attributes)
		_ = json.Unmarshal(eventsJSON, &s.Events)
		spans = append(spans, s)
	}
	return spans, rows.Err()
}
