package database

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Message mirrors WeChat's WeixinMessage structure + Hub operational fields.
type Message struct {
	// Internal
	ID        int64   `json:"id"`
	BotID     string  `json:"bot_id"`
	ChannelID *string `json:"channel_id,omitempty"`
	Direction string  `json:"direction"` // "inbound" / "outbound"

	// WeChat 1:1 fields
	Seq          *int64          `json:"seq,omitempty"`
	MessageID    *int64          `json:"message_id,omitempty"`
	FromUserID   string          `json:"from_user_id"`
	ToUserID     string          `json:"to_user_id,omitempty"`
	ClientID     string          `json:"client_id,omitempty"`
	CreateTimeMs *int64          `json:"create_time_ms,omitempty"`
	UpdateTimeMs *int64          `json:"update_time_ms,omitempty"`
	DeleteTimeMs *int64          `json:"delete_time_ms,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	GroupID      string          `json:"group_id,omitempty"`
	MessageType  int             `json:"message_type,omitempty"`  // 1=USER, 2=BOT
	MessageState int             `json:"message_state,omitempty"` // 0=NEW, 1=GENERATING, 2=FINISH
	ItemList     json.RawMessage `json:"item_list"`
	ContextToken string          `json:"context_token,omitempty"`

	// Operational
	MediaStatus string           `json:"media_status,omitempty"` // downloading/ready/failed
	MediaKeys   json.RawMessage  `json:"media_keys,omitempty"`   // {"0":"key.jpg","0_thumb":"key_thumb.jpg"}
	Raw         *json.RawMessage `json:"raw,omitempty"`

	CreatedAt int64 `json:"created_at"`
}

const msgSelectCols = `id, bot_id, channel_id, direction,
	seq, message_id, from_user_id, to_user_id, client_id,
	create_time_ms, update_time_ms, delete_time_ms,
	session_id, group_id, message_type, message_state, item_list, context_token,
	media_status, media_keys, raw,
	EXTRACT(EPOCH FROM created_at)::BIGINT`

func scanMessage(scanner interface{ Scan(...any) error }) (*Message, error) {
	m := &Message{}
	err := scanner.Scan(
		&m.ID, &m.BotID, &m.ChannelID, &m.Direction,
		&m.Seq, &m.MessageID, &m.FromUserID, &m.ToUserID, &m.ClientID,
		&m.CreateTimeMs, &m.UpdateTimeMs, &m.DeleteTimeMs,
		&m.SessionID, &m.GroupID, &m.MessageType, &m.MessageState, &m.ItemList, &m.ContextToken,
		&m.MediaStatus, &m.MediaKeys, &m.Raw,
		&m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// SaveResult holds the result of a SaveMessage upsert.
type SaveResult struct {
	ID       int64
	Inserted bool // true = new row, false = updated existing (duplicate message_id)
}

func (db *DB) SaveMessage(m *Message) (SaveResult, error) {
	if m.ItemList == nil {
		m.ItemList = json.RawMessage(`[]`)
	}
	if m.MediaKeys == nil {
		m.MediaKeys = json.RawMessage(`{}`)
	}
	// Inbound messages start unprocessed (processed_at = NULL);
	// outbound messages are immediately considered processed.
	processedExpr := "NULL"
	if m.Direction == "outbound" {
		processedExpr = "NOW()"
	}

	var r SaveResult
	// xmax = 0 means the row was freshly inserted; non-zero means it was updated
	// via ON CONFLICT (PostgreSQL internal: xmax holds the updating transaction ID).
	err := db.QueryRow(`
		INSERT INTO messages (bot_id, channel_id, direction,
			seq, message_id, from_user_id, to_user_id, client_id,
			create_time_ms, update_time_ms, delete_time_ms,
			session_id, group_id, message_type, message_state, item_list, context_token,
			media_status, media_keys, raw, processed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20, `+processedExpr+`)
		ON CONFLICT (bot_id, message_id) WHERE message_id IS NOT NULL DO UPDATE SET
			message_state  = EXCLUDED.message_state,
			item_list      = EXCLUDED.item_list,
			update_time_ms = EXCLUDED.update_time_ms,
			context_token  = EXCLUDED.context_token,
			raw            = EXCLUDED.raw
		RETURNING id, (xmax = 0)`,
		m.BotID, m.ChannelID, m.Direction,
		m.Seq, m.MessageID, m.FromUserID, m.ToUserID, m.ClientID,
		m.CreateTimeMs, m.UpdateTimeMs, m.DeleteTimeMs,
		m.SessionID, m.GroupID, m.MessageType, m.MessageState, m.ItemList, m.ContextToken,
		m.MediaStatus, m.MediaKeys, m.Raw,
	).Scan(&r.ID, &r.Inserted)
	return r, err
}

func (db *DB) GetMessage(id int64) (*Message, error) {
	return scanMessage(db.QueryRow("SELECT "+msgSelectCols+" FROM messages WHERE id = $1", id))
}

func (db *DB) ListMessages(botID string, limit int, beforeID int64) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var query string
	var args []any
	if beforeID > 0 {
		query = "SELECT " + msgSelectCols + " FROM messages WHERE bot_id = $1 AND id < $2 ORDER BY id DESC LIMIT $3"
		args = []any{botID, beforeID, limit}
	} else {
		query = "SELECT " + msgSelectCols + " FROM messages WHERE bot_id = $1 ORDER BY id DESC LIMIT $2"
		args = []any{botID, limit}
	}
	return scanMessages(db, query, args...)
}

func (db *DB) ListMessagesBySender(botID, sender string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND (from_user_id = $2 OR to_user_id = $2) ORDER BY id DESC LIMIT $3",
		botID, sender, limit,
	)
}

// ListChannelMessages returns messages for a channel's bot + sender.
// All messages (inbound + outbound) are stored globally with channel_id IS NULL.
func (db *DB) ListChannelMessages(channelID, sender string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var botID string
	if err := db.QueryRow("SELECT bot_id FROM channels WHERE id = $1", channelID).Scan(&botID); err != nil {
		return nil, err
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND (from_user_id = $2 OR to_user_id = $2) ORDER BY id DESC LIMIT $3",
		botID, sender, limit,
	)
}

func (db *DB) GetMessagesSince(botID string, afterSeq int64, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND id > $2 ORDER BY id ASC LIMIT $3",
		botID, afterSeq, limit,
	)
}

// GetLatestContextToken returns the most recent non-empty context_token for a bot.
func (db *DB) GetLatestContextToken(botID string) string {
	var token string
	db.QueryRow(
		"SELECT context_token FROM messages WHERE bot_id = $1 AND context_token != '' ORDER BY id DESC LIMIT 1",
		botID,
	).Scan(&token)
	return token
}

// HasFreshContextToken checks if the bot has a context_token from a message
// received within the given duration. WeChat context_tokens expire after ~24h.
func (db *DB) HasFreshContextToken(botID string, maxAge time.Duration) bool {
	var exists bool
	db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM messages WHERE bot_id = $1 AND context_token != '' AND created_at > NOW() - $2::INTERVAL)",
		botID, fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
	).Scan(&exists)
	return exists
}

// BatchHasFreshContextToken checks multiple bots at once, returning a set of bot IDs that have fresh tokens.
func (db *DB) BatchHasFreshContextToken(botIDs []string, maxAge time.Duration) map[string]bool {
	if len(botIDs) == 0 {
		return nil
	}
	result := make(map[string]bool, len(botIDs))
	interval := fmt.Sprintf("%d seconds", int(maxAge.Seconds()))

	// Build placeholders: $2, $3, ...
	placeholders := make([]string, len(botIDs))
	args := make([]any, 0, len(botIDs)+1)
	args = append(args, interval)
	for i, id := range botIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	rows, err := db.Query(
		"SELECT DISTINCT bot_id FROM messages WHERE bot_id IN ("+strings.Join(placeholders, ",")+") AND context_token != '' AND created_at > NOW() - $1::INTERVAL",
		args...,
	)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		result[id] = true
	}
	return result
}

// UpdateMediaStatus updates media_status and media_keys for all downloading messages of a bot.
func (db *DB) UpdateMediaStatus(botID, status string, keys json.RawMessage) error {
	if keys == nil {
		keys = json.RawMessage(`{}`)
	}
	_, err := db.Exec(`UPDATE messages SET media_status = $1, media_keys = $2
		WHERE bot_id = $3 AND media_status = 'downloading'`,
		status, keys, botID)
	return err
}

func (db *DB) UpdateMessagePayload(id int64, payload json.RawMessage) error {
	// Legacy: updates media_status and media_keys from old-style payload
	var p map[string]any
	json.Unmarshal(payload, &p)
	status, _ := p["media_status"].(string)
	keys := json.RawMessage(`{}`)
	if k, ok := p["media_key"].(string); ok {
		keys, _ = json.Marshal(map[string]string{"0": k})
	}
	_, err := db.Exec("UPDATE messages SET media_status = $1, media_keys = $2 WHERE id = $3",
		status, keys, id)
	return err
}

func (db *DB) UpdateMediaPayloads(botID, eqp string, newPayload json.RawMessage) error {
	// Extract media_status and media_keys from the payload
	var p map[string]any
	json.Unmarshal(newPayload, &p)
	status, _ := p["media_status"].(string)
	keys := json.RawMessage(`{}`)
	if k, ok := p["media_key"].(string); ok {
		keys, _ = json.Marshal(map[string]string{"0": k})
	}
	_, err := db.Exec(`UPDATE messages SET media_status = $1, media_keys = $2
		WHERE bot_id = $3 AND media_status = 'downloading'`,
		status, keys, botID)
	return err
}

// MarkProcessed sets processed_at = NOW() for a message.
func (db *DB) MarkProcessed(id int64) error {
	_, err := db.Exec("UPDATE messages SET processed_at = NOW() WHERE id = $1", id)
	return err
}

// GetUnprocessedMessages returns inbound messages that were never fully processed
// (processed_at IS NULL), typically due to a crash or restart. Limited to recent
// messages (last 24h) to avoid replaying ancient data.
func (db *DB) GetUnprocessedMessages(botID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND direction = 'inbound' AND processed_at IS NULL AND created_at > NOW() - INTERVAL '1 day' ORDER BY id ASC LIMIT $2",
		botID, limit,
	)
}

func (db *DB) PruneMessages(maxAgeDays int) (int64, error) {
	result, err := db.Exec("DELETE FROM messages WHERE created_at < NOW() - INTERVAL '1 day' * $1", maxAgeDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanMessages(db *DB, query string, args ...any) ([]Message, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, *m)
	}
	return msgs, rows.Err()
}
