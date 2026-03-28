package sqlite

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func generateAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const channelSelectCols = `id, bot_id, name, handle, ai_config, webhook_config,
	api_key, filter_rule, enabled, last_seq,
	created_at, updated_at`

func scanChannel(scanner interface{ Scan(...any) error }) (*store.Channel, error) {
	c := &store.Channel{}
	var filterJSON, aiJSON, webhookJSON []byte
	err := scanner.Scan(&c.ID, &c.BotID, &c.Name, &c.Handle, &aiJSON, &webhookJSON,
		&c.APIKey, &filterJSON, &c.Enabled, &c.LastSeq, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(filterJSON, &c.FilterRule)
	_ = json.Unmarshal(aiJSON, &c.AIConfig)
	_ = json.Unmarshal(webhookJSON, &c.WebhookConfig)
	return c, nil
}

func (db *DB) CreateChannel(botID, name, handle string, filter *store.FilterRule, ai *store.AIConfig) (*store.Channel, error) {
	id := uuid.New().String()
	apiKey := generateAPIKey()
	if filter == nil {
		filter = &store.FilterRule{}
	}
	if ai == nil {
		ai = &store.AIConfig{}
	}
	filterJSON, _ := json.Marshal(filter)
	aiJSON, _ := json.Marshal(ai)
	_, err := db.Exec(
		"INSERT INTO channels (id, bot_id, name, handle, ai_config, api_key, filter_rule) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, botID, name, handle, aiJSON, apiKey, filterJSON,
	)
	if err != nil {
		return nil, err
	}
	return &store.Channel{ID: id, BotID: botID, Name: name, Handle: handle, AIConfig: *ai,
		APIKey: apiKey, FilterRule: *filter, Enabled: true}, nil
}

func (db *DB) GetChannel(id string) (*store.Channel, error) {
	return scanChannel(db.QueryRow("SELECT "+channelSelectCols+" FROM channels WHERE id = ?", id))
}

func (db *DB) GetChannelByAPIKey(apiKey string) (*store.Channel, error) {
	return scanChannel(db.QueryRow("SELECT "+channelSelectCols+" FROM channels WHERE api_key = ?", apiKey))
}

func (db *DB) ListChannelsByBot(botID string) ([]store.Channel, error) {
	rows, err := db.Query("SELECT "+channelSelectCols+" FROM channels WHERE bot_id = ? AND enabled = 1", botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chs []store.Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		chs = append(chs, *c)
	}
	return chs, rows.Err()
}

func (db *DB) ListChannelsByBotIDs(botIDs []string) ([]store.Channel, error) {
	if len(botIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(botIDs))
	args := make([]any, len(botIDs))
	for i, id := range botIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("SELECT %s FROM channels WHERE bot_id IN (%s) ORDER BY created_at",
		channelSelectCols, strings.Join(placeholders, ", "))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chs []store.Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		chs = append(chs, *c)
	}
	return chs, rows.Err()
}

func (db *DB) UpdateChannel(id, name, handle string, filter *store.FilterRule, ai *store.AIConfig, webhook *store.WebhookConfig, enabled bool) error {
	filterJSON, _ := json.Marshal(filter)
	aiJSON, _ := json.Marshal(ai)
	webhookJSON, _ := json.Marshal(webhook)
	_, err := db.Exec(
		`UPDATE channels SET name = ?, handle = ?, filter_rule = ?, ai_config = ?,
		 webhook_config = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		name, handle, filterJSON, aiJSON, webhookJSON, enabled, db.now(), id,
	)
	return err
}

func (db *DB) DeleteChannel(id string) error {
	_, err := db.Exec("DELETE FROM channels WHERE id = ?", id)
	return err
}

func (db *DB) RotateChannelKey(id string) (string, error) {
	newKey := generateAPIKey()
	_, err := db.Exec("UPDATE channels SET api_key = ?, updated_at = ? WHERE id = ?", newKey, db.now(), id)
	return newKey, err
}

func (db *DB) UpdateChannelLastSeq(channelID string, seq int64) error {
	_, err := db.Exec("UPDATE channels SET last_seq = ?, updated_at = ? WHERE id = ?", seq, db.now(), channelID)
	return err
}

func (db *DB) CountChannelsByBot(botID string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM channels WHERE bot_id = ?", botID).Scan(&count)
	return count, err
}
