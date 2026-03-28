package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const msgSelectCols = `id, bot_id, channel_id, direction,
	seq, message_id, from_user_id, to_user_id, client_id,
	create_time_ms, update_time_ms, delete_time_ms,
	session_id, group_id, message_type, message_state, item_list, context_token,
	media_status, media_keys, raw,
	created_at`

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

	tx, err := db.Begin()
	if err != nil {
		return store.SaveResult{}, err
	}
	defer tx.Rollback()

	// Check for existing row
	var existingID int64
	err = tx.QueryRow("SELECT id FROM messages WHERE bot_id = ? AND message_id = ? AND message_id IS NOT NULL",
		m.BotID, m.MessageID).Scan(&existingID)
	if err == nil {
		// Update existing
		_, err = tx.Exec(`UPDATE messages SET message_state=?, item_list=?, update_time_ms=?, context_token=?, raw=? WHERE id=?`,
			m.MessageState, m.ItemList, m.UpdateTimeMs, m.ContextToken, m.Raw, existingID)
		if err != nil {
			return store.SaveResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return store.SaveResult{}, err
		}
		return store.SaveResult{ID: existingID, Inserted: false}, nil
	}

	// Insert new
	processedAt := sql.NullInt64{}
	if m.Direction == "outbound" {
		processedAt = sql.NullInt64{Int64: db.now(), Valid: true}
	}
	result, err := tx.Exec(`INSERT INTO messages (bot_id, channel_id, direction,
		seq, message_id, from_user_id, to_user_id, client_id,
		create_time_ms, update_time_ms, delete_time_ms,
		session_id, group_id, message_type, message_state, item_list, context_token,
		media_status, media_keys, raw, processed_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.BotID, m.ChannelID, m.Direction,
		m.Seq, m.MessageID, m.FromUserID, m.ToUserID, m.ClientID,
		m.CreateTimeMs, m.UpdateTimeMs, m.DeleteTimeMs,
		m.SessionID, m.GroupID, m.MessageType, m.MessageState, m.ItemList, m.ContextToken,
		m.MediaStatus, m.MediaKeys, m.Raw, processedAt,
	)
	if err != nil {
		return store.SaveResult{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return store.SaveResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.SaveResult{}, err
	}
	return store.SaveResult{ID: id, Inserted: true}, nil
}

func (db *DB) GetMessage(id int64) (*store.Message, error) {
	return scanMessage(db.QueryRow("SELECT "+msgSelectCols+" FROM messages WHERE id = ?", id))
}

func (db *DB) ListMessages(botID string, limit int, beforeID int64) ([]store.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var query string
	var args []any
	if beforeID > 0 {
		query = "SELECT " + msgSelectCols + " FROM messages WHERE bot_id = ? AND id < ? ORDER BY id DESC LIMIT ?"
		args = []any{botID, beforeID, limit}
	} else {
		query = "SELECT " + msgSelectCols + " FROM messages WHERE bot_id = ? ORDER BY id DESC LIMIT ?"
		args = []any{botID, limit}
	}
	return scanMessages(db, query, args...)
}

func (db *DB) ListMessagesBySender(botID, sender string, limit int) ([]store.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = ? AND (from_user_id = ? OR to_user_id = ?) ORDER BY id DESC LIMIT ?",
		botID, sender, sender, limit,
	)
}

func (db *DB) ListChannelMessages(channelID, sender string, limit int) ([]store.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var botID string
	if err := db.QueryRow("SELECT bot_id FROM channels WHERE id = ?", channelID).Scan(&botID); err != nil {
		return nil, err
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = ? AND (from_user_id = ? OR to_user_id = ?) ORDER BY id DESC LIMIT ?",
		botID, sender, sender, limit,
	)
}

func (db *DB) GetMessagesSince(botID string, afterSeq int64, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = ? AND id > ? ORDER BY id ASC LIMIT ?",
		botID, afterSeq, limit,
	)
}

func (db *DB) GetLatestContextToken(botID string) string {
	var token string
	db.QueryRow(
		"SELECT context_token FROM messages WHERE bot_id = ? AND context_token != '' ORDER BY id DESC LIMIT 1",
		botID,
	).Scan(&token)
	return token
}

func (db *DB) HasFreshContextToken(botID string, maxAge time.Duration) bool {
	var count int
	db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE bot_id = ? AND context_token != '' AND created_at > ? - ? LIMIT 1",
		botID, db.now(), int(maxAge.Seconds()),
	).Scan(&count)
	return count > 0
}

func (db *DB) UpdateMediaStatus(botID, status string, keys json.RawMessage) error {
	if keys == nil {
		keys = json.RawMessage(`{}`)
	}
	_, err := db.Exec(`UPDATE messages SET media_status = ?, media_keys = ?
		WHERE bot_id = ? AND media_status = 'downloading'`,
		status, keys, botID)
	return err
}

func (db *DB) UpdateMediaStatusByID(id int64, status string, keys json.RawMessage) error {
	if keys == nil {
		keys = json.RawMessage(`{}`)
	}
	_, err := db.Exec(`UPDATE messages SET media_status = ?, media_keys = ? WHERE id = ?`,
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
	_, err := db.Exec("UPDATE messages SET media_status = ?, media_keys = ? WHERE id = ?",
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
	_, err := db.Exec(`UPDATE messages SET media_status = ?, media_keys = ?
		WHERE bot_id = ? AND media_status = 'downloading'`,
		status, keys, botID)
	return err
}

func (db *DB) MarkProcessed(id int64) error {
	_, err := db.Exec("UPDATE messages SET processed_at = ? WHERE id = ?", db.now(), id)
	return err
}

func (db *DB) GetUnprocessedMessages(botID string, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	return scanMessages(db,
		"SELECT "+msgSelectCols+" FROM messages WHERE bot_id = ? AND direction = 'inbound' AND processed_at IS NULL AND created_at > ? - 86400 ORDER BY id ASC LIMIT ?",
		botID, db.now(), limit,
	)
}

func (db *DB) PruneMessages(maxAgeDays int) (int64, error) {
	result, err := db.Exec("DELETE FROM messages WHERE created_at < ? - 86400 * ?", db.now(), maxAgeDays)
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
