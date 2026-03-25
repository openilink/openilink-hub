package database

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Bot struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Name           string          `json:"name"`
	Provider       string          `json:"provider"`
	ProviderID     string          `json:"provider_id,omitempty"` // provider-specific bot ID (e.g. iLink bot_id)
	Status         string          `json:"status"`
	Credentials    json.RawMessage `json:"credentials,omitempty"`
	SyncState      json.RawMessage `json:"-"`
	MsgCount       int64           `json:"msg_count"`
	LastMsgAt      *int64          `json:"last_msg_at,omitempty"`
	ReminderHours  int             `json:"reminder_hours"`
	LastRemindedAt *int64          `json:"last_reminded_at,omitempty"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
}

const botSelectCols = `id, user_id, name, provider, provider_id, status, credentials, sync_state,
	msg_count, EXTRACT(EPOCH FROM last_msg_at)::BIGINT,
	reminder_hours, EXTRACT(EPOCH FROM last_reminded_at)::BIGINT,
	EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT`

func scanBot(scanner interface{ Scan(...any) error }) (*Bot, error) {
	b := &Bot{}
	err := scanner.Scan(&b.ID, &b.UserID, &b.Name, &b.Provider, &b.ProviderID, &b.Status,
		&b.Credentials, &b.SyncState, &b.MsgCount, &b.LastMsgAt,
		&b.ReminderHours, &b.LastRemindedAt,
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (db *DB) CreateBot(userID, name, provider, providerID string, credentials json.RawMessage) (*Bot, error) {
	id := uuid.New().String()
	if name == "" {
		name = "Bot-" + id[:8]
	}
	if credentials == nil {
		credentials = json.RawMessage(`{}`)
	}
	_, err := db.Exec(
		`INSERT INTO bots (id, user_id, name, provider, provider_id, status, credentials)
		 VALUES ($1, $2, $3, $4, $5, 'connected', $6)`,
		id, userID, name, provider, providerID, credentials,
	)
	if err != nil {
		return nil, err
	}
	return &Bot{
		ID: id, UserID: userID, Name: name, Provider: provider, ProviderID: providerID,
		Status: "connected", Credentials: credentials,
	}, nil
}

func (db *DB) GetBot(id string) (*Bot, error) {
	return scanBot(db.QueryRow("SELECT "+botSelectCols+" FROM bots WHERE id = $1", id))
}

func (db *DB) ListBotsByUser(userID string) ([]Bot, error) {
	rows, err := db.Query("SELECT "+botSelectCols+" FROM bots WHERE user_id = $1 ORDER BY created_at", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

func (db *DB) GetAllBots() ([]Bot, error) {
	rows, err := db.Query("SELECT " + botSelectCols + " FROM bots")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

// FindBotByProviderID finds a bot by its provider-specific ID.
func (db *DB) FindBotByProviderID(provider, providerID string) (*Bot, error) {
	return scanBot(db.QueryRow(
		"SELECT "+botSelectCols+" FROM bots WHERE provider = $1 AND provider_id = $2", provider, providerID))
}

// FindBotByCredential finds a bot by a JSON credential field (e.g. ilink_user_id).
func (db *DB) FindBotByCredential(key, value string) (*Bot, error) {
	return scanBot(db.QueryRow(
		"SELECT "+botSelectCols+" FROM bots WHERE credentials->>$1 = $2", key, value))
}

// UpdateBotCredentials updates credentials, provider_id, and resets status to connected.
// Also clears old context_tokens since they become invalid after rebind.
func (db *DB) UpdateBotCredentials(id, providerID string, credentials json.RawMessage) error {
	_, err := db.Exec(
		"UPDATE bots SET credentials = $1, provider_id = $2, status = 'connected', sync_state = '{}', updated_at = NOW() WHERE id = $3",
		credentials, providerID, id)
	if err != nil {
		return err
	}
	// Invalidate old context_tokens — they belong to the previous session
	db.Exec("UPDATE messages SET context_token = '' WHERE bot_id = $1 AND context_token != ''", id)
	return nil
}

func (db *DB) UpdateBotName(id, name string) error {
	_, err := db.Exec("UPDATE bots SET name = $1, updated_at = NOW() WHERE id = $2", name, id)
	return err
}

func (db *DB) UpdateBotStatus(id, status string) error {
	_, err := db.Exec("UPDATE bots SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}

func (db *DB) UpdateBotSyncState(id string, syncState json.RawMessage) error {
	_, err := db.Exec("UPDATE bots SET sync_state = $1, updated_at = NOW() WHERE id = $2", syncState, id)
	return err
}

func (db *DB) IncrBotMsgCount(id string) error {
	_, err := db.Exec("UPDATE bots SET msg_count = msg_count + 1, last_msg_at = NOW(), updated_at = NOW() WHERE id = $1", id)
	return err
}

func (db *DB) UpdateBotReminder(id string, hours int) error {
	_, err := db.Exec("UPDATE bots SET reminder_hours = $1, updated_at = NOW() WHERE id = $2", hours, id)
	return err
}

func (db *DB) MarkBotReminded(id string) error {
	_, err := db.Exec("UPDATE bots SET last_reminded_at = NOW() WHERE id = $1", id)
	return err
}

// GetBotsNeedingReminder returns connected bots where:
// - reminder_hours > 0
// - last_msg_at is older than reminder_hours
// - last_reminded_at is null or older than 1 hour (avoid spam)
func (db *DB) GetBotsNeedingReminder() ([]Bot, error) {
	rows, err := db.Query(`SELECT `+botSelectCols+` FROM bots
		WHERE status = 'connected'
		AND reminder_hours > 0
		AND last_msg_at IS NOT NULL
		AND last_msg_at < NOW() - INTERVAL '1 hour' * reminder_hours
		AND (last_reminded_at IS NULL OR last_reminded_at < NOW() - INTERVAL '1 hour')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

func (db *DB) DeleteBot(id string) error {
	_, err := db.Exec("DELETE FROM bots WHERE id = $1", id)
	return err
}

func (db *DB) CountBotsByUser(userID string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM bots WHERE user_id = $1", userID).Scan(&count)
	return count, err
}

// BotStats returns aggregated stats for a user's bots.
type BotStats struct {
	TotalBots     int   `json:"total_bots"`
	OnlineBots    int   `json:"online_bots"`
	TotalChannels int   `json:"total_channels"`
	TotalMessages int64 `json:"total_messages"`
	ConnectedWS   int   `json:"connected_ws"` // set by API layer
}

// AdminStats returns system-wide stats for the admin dashboard.
type AdminStats struct {
	TotalUsers       int   `json:"total_users"`
	ActiveUsers      int   `json:"active_users"`
	TotalBots        int   `json:"total_bots"`
	OnlineBots       int   `json:"online_bots"`
	ExpiredBots      int   `json:"expired_bots"`
	TotalChannels    int   `json:"total_channels"`
	TotalMessages    int64 `json:"total_messages"`
	InboundMessages  int64 `json:"inbound_messages"`
	OutboundMessages int64 `json:"outbound_messages"`
	ConnectedWS      int   `json:"connected_ws"` // set by API layer
}

func (db *DB) GetAdminStats() (*AdminStats, error) {
	s := &AdminStats{}
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active') FROM users`).
		Scan(&s.TotalUsers, &s.ActiveUsers)
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'connected'), COUNT(*) FILTER (WHERE status = 'session_expired') FROM bots`).
		Scan(&s.TotalBots, &s.OnlineBots, &s.ExpiredBots)
	db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&s.TotalChannels)
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE direction = 'inbound'), COUNT(*) FILTER (WHERE direction = 'outbound') FROM messages`).
		Scan(&s.TotalMessages, &s.InboundMessages, &s.OutboundMessages)
	return s, nil
}

func (db *DB) GetBotStats(userID string) (*BotStats, error) {
	s := &BotStats{}
	err := db.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'connected'),
			COALESCE(SUM(msg_count), 0)
		FROM bots WHERE user_id = $1`, userID,
	).Scan(&s.TotalBots, &s.OnlineBots, &s.TotalMessages)
	if err != nil {
		return nil, err
	}
	// Count channels for user's bots
	db.QueryRow(`SELECT COUNT(*) FROM channels WHERE bot_id IN (SELECT id FROM bots WHERE user_id = $1)`, userID).Scan(&s.TotalChannels)
	return s, nil
}

// RecentContact tracks external users that have communicated through a bot.
type RecentContact struct {
	UserID    string `json:"user_id"`
	LastMsgAt int64  `json:"last_msg_at"`
	MsgCount  int    `json:"msg_count"`
}

func (db *DB) ListRecentContacts(botID string, limit int) ([]RecentContact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT from_user_id, EXTRACT(EPOCH FROM MAX(created_at))::BIGINT, COUNT(*)
		FROM messages WHERE bot_id = $1 AND direction = 'inbound' AND from_user_id != ''
		GROUP BY from_user_id ORDER BY MAX(created_at) DESC LIMIT $2`,
		botID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var contacts []RecentContact
	for rows.Next() {
		var c RecentContact
		if err := rows.Scan(&c.UserID, &c.LastMsgAt, &c.MsgCount); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (db *DB) LastActivityAt(userID string) *time.Time {
	var t *time.Time
	db.QueryRow(`SELECT MAX(last_msg_at) FROM bots WHERE user_id = $1`, userID).Scan(&t)
	return t
}
