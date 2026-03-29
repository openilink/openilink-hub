package postgres

import (
	"encoding/json"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const msgSelectCols = `id, bot_id, channel_id, direction,
	seq, message_id, from_user_id, to_user_id, client_id,
	create_time_ms, update_time_ms, delete_time_ms,
	session_id, group_id, message_type, message_state, item_list, context_token,
	media_status, media_keys, raw,
	EXTRACT(EPOCH FROM created_at)::BIGINT`

func scanMessage(scanner interface{ Scan(...any) error }) (*store.Message, error) {
	m := &store.Message{}
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

func (db *DB) SaveMessage(m *store.Message) (store.SaveResult, error) {
	if m.ItemList == nil {
		m.ItemList = json.RawMessage(`[]`)
	}
	if m.MediaKeys == nil {
		m.MediaKeys = json.RawMessage(`{}`)
	}
	var processedAt any
	if m.Direction == "outbound" {
		processedAt = db.now()
	}

	var r store.SaveResult
	err := db.QueryRow(`
		INSERT INTO messages (bot_id, channel_id, direction,
			seq, message_id, from_user_id, to_user_id, client_id,
			create_time_ms, update_time_ms, delete_time_ms,
			session_id, group_id, message_type, message_state, item_list, context_token,
			media_status, media_keys, raw, processed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
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
		m.MediaStatus, m.MediaKeys, m.Raw, processedAt,
	).Scan(&r.ID, &r.Inserted)
	return r, err
}

func (db *DB) GetMessage(id int64) (*store.Message, error) {
	return scanMessage(db.QueryRow("SELECT "+msgSelectCols+" FROM messages WHERE id = $1", id))
}

func (db *DB) ListMessages(botID string, limit int, beforeID int64) ([]store.Message, error) {
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

func (db *DB) ListMessagesBySender(botID, sender string, limit int) ([]store.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND (from_user_id = $2 OR to_user_id = $2) ORDER BY id DESC LIMIT $3",
		botID, sender, limit,
	)
}

func (db *DB) ListChannelMessages(channelID, sender string, limit int) ([]store.Message, error) {
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

func (db *DB) GetMessagesSince(botID string, afterSeq int64, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND id > $2 ORDER BY id ASC LIMIT $3",
		botID, afterSeq, limit,
	)
}

func (db *DB) GetLatestContextToken(botID string) string {
	var token string
	db.QueryRow(
		"SELECT context_token FROM messages WHERE bot_id = $1 AND context_token != '' ORDER BY id DESC LIMIT 1",
		botID,
	).Scan(&token)
	return token
}

func (db *DB) HasFreshContextToken(botID string, maxAge time.Duration) bool {
	var exists bool
	db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM messages WHERE bot_id = $1 AND context_token != '' AND created_at > $2::timestamptz - ($3 * INTERVAL '1 second'))",
		botID, db.now(), int(maxAge.Seconds()),
	).Scan(&exists)
	return exists
}

func (db *DB) UpdateMediaStatus(botID, status string, keys json.RawMessage) error {
	if keys == nil {
		keys = json.RawMessage(`{}`)
	}
	_, err := db.Exec(`UPDATE messages SET media_status = $1, media_keys = $2
		WHERE bot_id = $3 AND media_status = 'downloading'`,
		status, keys, botID)
	return err
}

func (db *DB) UpdateMediaStatusByID(id int64, status string, keys json.RawMessage) error {
	if keys == nil {
		keys = json.RawMessage(`{}`)
	}
	_, err := db.Exec(`UPDATE messages SET media_status = $1, media_keys = $2 WHERE id = $3`,
		status, keys, id)
	return err
}

func (db *DB) UpdateMessagePayload(id int64, payload json.RawMessage) error {
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

func (db *DB) MarkProcessed(id int64) error {
	_, err := db.Exec("UPDATE messages SET processed_at = $1::timestamptz WHERE id = $2", db.now(), id)
	return err
}

func (db *DB) GetUnprocessedMessages(botID string, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = $1 AND direction = 'inbound' AND processed_at IS NULL AND created_at > ($2::timestamptz - INTERVAL '1 day') ORDER BY id ASC LIMIT $3",
		botID, db.now(), limit,
	)
}

func (db *DB) PruneMessages(maxAgeDays int) (int64, error) {
	result, err := db.Exec("DELETE FROM messages WHERE created_at < $1::timestamptz - (INTERVAL '1 day' * $2)", db.now(), maxAgeDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanMessages(db *DB, query string, args ...any) ([]store.Message, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []store.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, *m)
	}
	return msgs, rows.Err()
}
