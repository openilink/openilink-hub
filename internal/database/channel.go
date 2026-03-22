package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type FilterRule struct {
	UserIDs      []string `json:"user_ids,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	MessageTypes []string `json:"message_types,omitempty"` // "text","image","voice","file","video"
}

// AIConfig holds optional AI auto-reply configuration for a channel.
type AIConfig struct {
	Enabled      bool   `json:"enabled"`
	Source       string `json:"source,omitempty"`         // "builtin" (use global config) or "custom"
	BaseURL      string `json:"base_url,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	Model        string `json:"model,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	MaxHistory   int    `json:"max_history,omitempty"`    // context messages, default: 20
}

type Channel struct {
	ID         string     `json:"id"`
	BotID      string     `json:"bot_id"`
	Name       string     `json:"name"`
	Handle     string     `json:"handle"`
	AIConfig   AIConfig   `json:"ai_config"`
	APIKey     string     `json:"api_key"`
	FilterRule FilterRule `json:"filter_rule"`
	Enabled    bool       `json:"enabled"`
	LastSeq    int64      `json:"last_seq"`
	CreatedAt  int64      `json:"created_at"`
	UpdatedAt  int64      `json:"updated_at"`
}

func generateAPIKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const channelSelectCols = `id, bot_id, name, handle, ai_config, api_key, filter_rule, enabled, last_seq,
	EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT`

func scanChannel(scanner interface{ Scan(...any) error }) (*Channel, error) {
	c := &Channel{}
	var filterJSON, aiJSON []byte
	err := scanner.Scan(&c.ID, &c.BotID, &c.Name, &c.Handle, &aiJSON, &c.APIKey,
		&filterJSON, &c.Enabled, &c.LastSeq, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(filterJSON, &c.FilterRule)
	_ = json.Unmarshal(aiJSON, &c.AIConfig)
	return c, nil
}

func (db *DB) CreateChannel(botID, name, handle string, filter *FilterRule, ai *AIConfig) (*Channel, error) {
	id := uuid.New().String()
	apiKey := generateAPIKey()
	if filter == nil {
		filter = &FilterRule{}
	}
	if ai == nil {
		ai = &AIConfig{}
	}
	filterJSON, _ := json.Marshal(filter)
	aiJSON, _ := json.Marshal(ai)
	_, err := db.Exec(
		"INSERT INTO channels (id, bot_id, name, handle, ai_config, api_key, filter_rule) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		id, botID, name, handle, aiJSON, apiKey, filterJSON,
	)
	if err != nil {
		return nil, err
	}
	return &Channel{ID: id, BotID: botID, Name: name, Handle: handle, AIConfig: *ai,
		APIKey: apiKey, FilterRule: *filter, Enabled: true}, nil
}

func (db *DB) GetChannel(id string) (*Channel, error) {
	return scanChannel(db.QueryRow("SELECT "+channelSelectCols+" FROM channels WHERE id = $1", id))
}

func (db *DB) GetChannelByAPIKey(apiKey string) (*Channel, error) {
	return scanChannel(db.QueryRow("SELECT "+channelSelectCols+" FROM channels WHERE api_key = $1", apiKey))
}

func (db *DB) ListChannelsByBot(botID string) ([]Channel, error) {
	rows, err := db.Query("SELECT "+channelSelectCols+" FROM channels WHERE bot_id = $1 AND enabled = TRUE", botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chs []Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		chs = append(chs, *c)
	}
	return chs, rows.Err()
}

func (db *DB) ListChannelsByBotIDs(botIDs []string) ([]Channel, error) {
	if len(botIDs) == 0 {
		return nil, nil
	}
	// Build query with IN ($1, $2, ...) since database/sql doesn't support ANY with slices
	query := "SELECT " + channelSelectCols + " FROM channels WHERE bot_id IN ("
	args := make([]any, len(botIDs))
	for i, id := range botIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query += ") ORDER BY created_at"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chs []Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		chs = append(chs, *c)
	}
	return chs, rows.Err()
}

func (db *DB) UpdateChannel(id, name, handle string, filter *FilterRule, ai *AIConfig, enabled bool) error {
	filterJSON, _ := json.Marshal(filter)
	aiJSON, _ := json.Marshal(ai)
	_, err := db.Exec(
		"UPDATE channels SET name = $1, handle = $2, filter_rule = $3, ai_config = $4, enabled = $5, updated_at = NOW() WHERE id = $6",
		name, handle, filterJSON, aiJSON, enabled, id,
	)
	return err
}

func (db *DB) DeleteChannel(id string) error {
	_, err := db.Exec("DELETE FROM channels WHERE id = $1", id)
	return err
}

func (db *DB) RotateChannelKey(id string) (string, error) {
	newKey := generateAPIKey()
	_, err := db.Exec("UPDATE channels SET api_key = $1, updated_at = NOW() WHERE id = $2", newKey, id)
	return newKey, err
}

func (db *DB) UpdateChannelLastSeq(channelID string, seq int64) error {
	_, err := db.Exec("UPDATE channels SET last_seq = $1, updated_at = NOW() WHERE id = $2", seq, channelID)
	return err
}

func (db *DB) CountChannelsByBot(botID string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM channels WHERE bot_id = $1", botID).Scan(&count)
	return count, err
}
