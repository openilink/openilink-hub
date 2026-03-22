package database

import "github.com/google/uuid"

type Bot struct {
	ID          string
	UserID      string
	BotID       string // ilink_bot_id
	BotToken    string
	BaseURL     string
	ILinkUserID string
	SyncBuf     string
	Status      string
	CreatedAt   int64
	UpdatedAt   int64
}

func (db *DB) CreateBot(userID, botID, botToken, baseURL, ilinkUserID string) (*Bot, error) {
	id := uuid.New().String()
	_, err := db.Exec(
		`INSERT INTO bots (id, user_id, bot_id, bot_token, base_url, ilink_user_id, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'connected')`,
		id, userID, botID, botToken, baseURL, ilinkUserID,
	)
	if err != nil {
		return nil, err
	}
	return &Bot{ID: id, UserID: userID, BotID: botID, BotToken: botToken, BaseURL: baseURL, ILinkUserID: ilinkUserID, Status: "connected"}, nil
}

func (db *DB) ListBotsByUser(userID string) ([]Bot, error) {
	rows, err := db.Query(
		"SELECT id, user_id, bot_id, bot_token, base_url, ilink_user_id, sync_buf, status, created_at, updated_at FROM bots WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []Bot
	for rows.Next() {
		var b Bot
		if err := rows.Scan(&b.ID, &b.UserID, &b.BotID, &b.BotToken, &b.BaseURL, &b.ILinkUserID, &b.SyncBuf, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bots = append(bots, b)
	}
	return bots, rows.Err()
}

func (db *DB) GetBot(id string) (*Bot, error) {
	b := &Bot{}
	err := db.QueryRow(
		"SELECT id, user_id, bot_id, bot_token, base_url, ilink_user_id, sync_buf, status, created_at, updated_at FROM bots WHERE id = ?", id,
	).Scan(&b.ID, &b.UserID, &b.BotID, &b.BotToken, &b.BaseURL, &b.ILinkUserID, &b.SyncBuf, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (db *DB) GetAllBots() ([]Bot, error) {
	rows, err := db.Query(
		"SELECT id, user_id, bot_id, bot_token, base_url, ilink_user_id, sync_buf, status, created_at, updated_at FROM bots",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []Bot
	for rows.Next() {
		var b Bot
		if err := rows.Scan(&b.ID, &b.UserID, &b.BotID, &b.BotToken, &b.BaseURL, &b.ILinkUserID, &b.SyncBuf, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		bots = append(bots, b)
	}
	return bots, rows.Err()
}

func (db *DB) UpdateBotStatus(id, status string) error {
	_, err := db.Exec("UPDATE bots SET status = ?, updated_at = unixepoch() WHERE id = ?", status, id)
	return err
}

func (db *DB) UpdateBotSyncBuf(id, syncBuf string) error {
	_, err := db.Exec("UPDATE bots SET sync_buf = ?, updated_at = unixepoch() WHERE id = ?", syncBuf, id)
	return err
}

func (db *DB) DeleteBot(id string) error {
	_, err := db.Exec("DELETE FROM bots WHERE id = ?", id)
	return err
}
