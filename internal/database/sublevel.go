package database

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/google/uuid"
)

type Sublevel struct {
	ID         string
	UserID     string
	BotDBID    string
	Name       string
	APIKey     string
	FilterRule string
	Enabled    bool
	CreatedAt  int64
	UpdatedAt  int64
}

func generateAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (db *DB) CreateSublevel(userID, botDBID, name string) (*Sublevel, error) {
	id := uuid.New().String()
	apiKey := generateAPIKey()
	_, err := db.Exec(
		"INSERT INTO sublevels (id, user_id, bot_db_id, name, api_key) VALUES (?, ?, ?, ?, ?)",
		id, userID, botDBID, name, apiKey,
	)
	if err != nil {
		return nil, err
	}
	return &Sublevel{ID: id, UserID: userID, BotDBID: botDBID, Name: name, APIKey: apiKey, Enabled: true}, nil
}

func (db *DB) ListSublevelsByUser(userID string) ([]Sublevel, error) {
	rows, err := db.Query(
		"SELECT id, user_id, bot_db_id, name, api_key, filter_rule, enabled, created_at, updated_at FROM sublevels WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Sublevel
	for rows.Next() {
		var s Sublevel
		if err := rows.Scan(&s.ID, &s.UserID, &s.BotDBID, &s.Name, &s.APIKey, &s.FilterRule, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (db *DB) ListSublevelsByBot(botDBID string) ([]Sublevel, error) {
	rows, err := db.Query(
		"SELECT id, user_id, bot_db_id, name, api_key, filter_rule, enabled, created_at, updated_at FROM sublevels WHERE bot_db_id = ? AND enabled = 1",
		botDBID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Sublevel
	for rows.Next() {
		var s Sublevel
		if err := rows.Scan(&s.ID, &s.UserID, &s.BotDBID, &s.Name, &s.APIKey, &s.FilterRule, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (db *DB) GetSublevelByAPIKey(apiKey string) (*Sublevel, error) {
	s := &Sublevel{}
	err := db.QueryRow(
		"SELECT id, user_id, bot_db_id, name, api_key, filter_rule, enabled, created_at, updated_at FROM sublevels WHERE api_key = ?",
		apiKey,
	).Scan(&s.ID, &s.UserID, &s.BotDBID, &s.Name, &s.APIKey, &s.FilterRule, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) DeleteSublevel(id string) error {
	_, err := db.Exec("DELETE FROM sublevels WHERE id = ?", id)
	return err
}

func (db *DB) RotateSublevelKey(id string) (string, error) {
	newKey := generateAPIKey()
	_, err := db.Exec("UPDATE sublevels SET api_key = ?, updated_at = unixepoch() WHERE id = ?", newKey, id)
	return newKey, err
}
