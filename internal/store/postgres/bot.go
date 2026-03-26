package postgres

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

const botSelectCols = `id, user_id, name, provider, provider_id, status, credentials, sync_state,
	msg_count, EXTRACT(EPOCH FROM last_msg_at)::BIGINT,
	reminder_hours, EXTRACT(EPOCH FROM last_reminded_at)::BIGINT,
	EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT,
	ai_enabled`

func scanBot(scanner interface{ Scan(...any) error }) (*store.Bot, error) {
	b := &store.Bot{}
	err := scanner.Scan(&b.ID, &b.UserID, &b.Name, &b.Provider, &b.ProviderID, &b.Status,
		&b.Credentials, &b.SyncState, &b.MsgCount, &b.LastMsgAt,
		&b.ReminderHours, &b.LastRemindedAt,
		&b.CreatedAt, &b.UpdatedAt, &b.AIEnabled)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (db *DB) CreateBot(userID, name, provider, providerID string, credentials json.RawMessage) (*store.Bot, error) {
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
	return &store.Bot{
		ID: id, UserID: userID, Name: name, Provider: provider, ProviderID: providerID,
		Status: "connected", Credentials: credentials,
	}, nil
}

func (db *DB) GetBot(id string) (*store.Bot, error) {
	return scanBot(db.QueryRow("SELECT "+botSelectCols+" FROM bots WHERE id = $1", id))
}

func (db *DB) ListBotsByUser(userID string) ([]store.Bot, error) {
	rows, err := db.Query("SELECT "+botSelectCols+" FROM bots WHERE user_id = $1 ORDER BY created_at", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []store.Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

func (db *DB) GetAllBots() ([]store.Bot, error) {
	rows, err := db.Query("SELECT " + botSelectCols + " FROM bots")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []store.Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

func (db *DB) FindBotByProviderID(provider, providerID string) (*store.Bot, error) {
	return scanBot(db.QueryRow(
		"SELECT "+botSelectCols+" FROM bots WHERE provider = $1 AND provider_id = $2", provider, providerID))
}

func (db *DB) FindBotByCredential(key, value string) (*store.Bot, error) {
	return scanBot(db.QueryRow(
		"SELECT "+botSelectCols+" FROM bots WHERE credentials->>$1 = $2", key, value))
}

func (db *DB) UpdateBotCredentials(id, providerID string, credentials json.RawMessage) error {
	_, err := db.Exec(
		"UPDATE bots SET credentials = $1, provider_id = $2, status = 'connected', sync_state = '{}', updated_at = NOW() WHERE id = $3",
		credentials, providerID, id)
	if err != nil {
		return err
	}
	db.Exec("UPDATE messages SET context_token = '' WHERE bot_id = $1 AND context_token != '' AND created_at > NOW() - INTERVAL '1 day'", id)
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

func (db *DB) GetBotsNeedingReminder() ([]store.Bot, error) {
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
	var bots []store.Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, err
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

func (db *DB) UpdateBotAIEnabled(id string, enabled bool) error {
	_, err := db.Exec("UPDATE bots SET ai_enabled = $1, updated_at = NOW() WHERE id = $2", enabled, id)
	return err
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

func (db *DB) GetAdminStats() (*store.AdminStats, error) {
	s := &store.AdminStats{}
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active') FROM users`).
		Scan(&s.TotalUsers, &s.ActiveUsers)
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'connected'), COUNT(*) FILTER (WHERE status = 'session_expired') FROM bots`).
		Scan(&s.TotalBots, &s.OnlineBots, &s.ExpiredBots)
	db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&s.TotalChannels)
	db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE direction = 'inbound'), COUNT(*) FILTER (WHERE direction = 'outbound') FROM messages`).
		Scan(&s.TotalMessages, &s.InboundMessages, &s.OutboundMessages)
	return s, nil
}

func (db *DB) GetBotStats(userID string) (*store.BotStats, error) {
	s := &store.BotStats{}
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
	db.QueryRow(`SELECT COUNT(*) FROM channels WHERE bot_id IN (SELECT id FROM bots WHERE user_id = $1)`, userID).Scan(&s.TotalChannels)
	return s, nil
}

func (db *DB) ListRecentContacts(botID string, limit int) ([]store.RecentContact, error) {
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
	var contacts []store.RecentContact
	for rows.Next() {
		var c store.RecentContact
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

// BatchHasFreshContextToken checks multiple bots at once.
func (db *DB) BatchHasFreshContextToken(botIDs []string, maxAge time.Duration) map[string]bool {
	if len(botIDs) == 0 {
		return map[string]bool{}
	}
	result := make(map[string]bool, len(botIDs))
	secs := int(maxAge.Seconds())

	placeholders := make([]string, len(botIDs))
	args := make([]any, 0, len(botIDs)+1)
	args = append(args, secs)
	for i, id := range botIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	rows, err := db.Query(
		"SELECT DISTINCT bot_id FROM messages WHERE bot_id IN ("+strings.Join(placeholders, ",")+") AND context_token != '' AND created_at > NOW() - $1 * INTERVAL '1 second'",
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
