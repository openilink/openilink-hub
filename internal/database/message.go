package database

type Message struct {
	ID          int64
	BotDBID     string
	Direction   string // "inbound" or "outbound"
	ILinkUserID string
	MessageType int
	Content     string
	SublevelID  *string
	CreatedAt   int64
}

func (db *DB) SaveMessage(botDBID, direction, ilinkUserID string, msgType int, content string, sublevelID *string) error {
	_, err := db.Exec(
		"INSERT INTO messages (bot_db_id, direction, ilink_user_id, message_type, content, sublevel_id) VALUES (?, ?, ?, ?, ?, ?)",
		botDBID, direction, ilinkUserID, msgType, content, sublevelID,
	)
	return err
}

func (db *DB) ListMessages(botDBID string, limit int, beforeID int64) ([]Message, error) {
	query := "SELECT id, bot_db_id, direction, ilink_user_id, message_type, content, sublevel_id, created_at FROM messages WHERE bot_db_id = ?"
	args := []any{botDBID}
	if beforeID > 0 {
		query += " AND id < ?"
		args = append(args, beforeID)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.BotDBID, &m.Direction, &m.ILinkUserID, &m.MessageType, &m.Content, &m.SublevelID, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
