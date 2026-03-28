package sqlite

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

const botSelectCols = `id, user_id, name, display_name, provider, provider_id, status, credentials, sync_state,
	msg_count, last_msg_at,
	reminder_hours, last_reminded_at,
	created_at, updated_at, ai_enabled, ai_model`

func scanBot(scanner interface{ Scan(...any) error }) (*store.Bot, error) {
	b := &store.Bot{}
	var credStr, syncStr string
	err := scanner.Scan(&b.ID, &b.UserID, &b.Name, &b.DisplayName, &b.Provider, &b.ProviderID, &b.Status,
		&credStr, &syncStr, &b.MsgCount, &b.LastMsgAt,
		&b.ReminderHours, &b.LastRemindedAt,
		&b.CreatedAt, &b.UpdatedAt, &b.AIEnabled, &b.AIModel)
	if err != nil {
		return nil, err
	}
	b.Credentials = json.RawMessage(credStr)
	b.SyncState = json.RawMessage(syncStr)
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
		 VALUES (?, ?, ?, ?, ?, 'connected', ?)`,
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
	return scanBot(db.QueryRow("SELECT "+botSelectCols+" FROM bots WHERE id = ?", id))
}

func (db *DB) ListBotsByUser(userID string) ([]store.Bot, error) {
	rows, err := db.Query("SELECT "+botSelectCols+" FROM bots WHERE user_id = ? ORDER BY created_at", userID)
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
		"SELECT "+botSelectCols+" FROM bots WHERE provider = ? AND provider_id = ?", provider, providerID))
}

func (db *DB) FindBotByCredential(key, value string) (*store.Bot, error) {
	return scanBot(db.QueryRow(
		"SELECT "+botSelectCols+" FROM bots WHERE json_extract(credentials, '$.' || ?) = ?", key, value))
}

func (db *DB) UpdateBotCredentials(id, providerID string, credentials json.RawMessage) error {
	now := db.now()
	_, err := db.Exec(
		"UPDATE bots SET credentials = ?, provider_id = ?, status = 'connected', sync_state = '{}', updated_at = ? WHERE id = ?",
		credentials, providerID, now, id)
	if err != nil {
		return err
	}
	db.Exec("UPDATE messages SET context_token = '' WHERE bot_id = ? AND context_token != '' AND created_at > ? - 86400", id, now)
	return nil
}

func (db *DB) UpdateBotName(id, name string) error {
	_, err := db.Exec("UPDATE bots SET name = ?, updated_at = ? WHERE id = ?", name, db.now(), id)
	return err
}

func (db *DB) UpdateBotDisplayName(id, displayName string) error {
	_, err := db.Exec("UPDATE bots SET display_name = ?, updated_at = ? WHERE id = ?", displayName, db.now(), id)
	return err
}

func (db *DB) UpdateBotStatus(id, status string) error {
	_, err := db.Exec("UPDATE bots SET status = ?, updated_at = ? WHERE id = ?", status, db.now(), id)
	return err
}

func (db *DB) UpdateBotSyncState(id string, syncState json.RawMessage) error {
	_, err := db.Exec("UPDATE bots SET sync_state = ?, updated_at = ? WHERE id = ?", syncState, db.now(), id)
	return err
}

func (db *DB) IncrBotMsgCount(id string) error {
	now := db.now()
	_, err := db.Exec("UPDATE bots SET msg_count = msg_count + 1, last_msg_at = ?, updated_at = ? WHERE id = ?", now, now, id)
	return err
}

func (db *DB) UpdateBotReminder(id string, hours int) error {
	_, err := db.Exec("UPDATE bots SET reminder_hours = ?, updated_at = ? WHERE id = ?", hours, db.now(), id)
	return err
}

func (db *DB) MarkBotReminded(id string) error {
	_, err := db.Exec("UPDATE bots SET last_reminded_at = ? WHERE id = ?", db.now(), id)
	return err
}

func (db *DB) GetBotsNeedingReminder() ([]store.Bot, error) {
	now := db.now()
	rows, err := db.Query(`SELECT `+botSelectCols+` FROM bots
		WHERE status = 'connected'
		AND reminder_hours > 0
		AND last_msg_at IS NOT NULL
		AND last_msg_at < ? - 3600 * reminder_hours
		AND (last_reminded_at IS NULL OR last_reminded_at < ? - 3600)`, now, now)
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
	_, err := db.Exec("UPDATE bots SET ai_enabled = ?, updated_at = ? WHERE id = ?", enabled, db.now(), id)
	return err
}

func (db *DB) UpdateBotAIModel(id, model string) error {
	_, err := db.Exec("UPDATE bots SET ai_model = ?, updated_at = ? WHERE id = ?", model, db.now(), id)
	return err
}

func (db *DB) DeleteBot(id string) error {
	_, err := db.Exec("DELETE FROM bots WHERE id = ?", id)
	return err
}

func (db *DB) CountBotsByUser(userID string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM bots WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (db *DB) GetAdminStats() (*store.AdminStats, error) {
	s := &store.AdminStats{}
	db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) FROM users`).
		Scan(&s.TotalUsers, &s.ActiveUsers)
	db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN status = 'connected' THEN 1 ELSE 0 END), SUM(CASE WHEN status = 'session_expired' THEN 1 ELSE 0 END) FROM bots`).
		Scan(&s.TotalBots, &s.OnlineBots, &s.ExpiredBots)
	db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&s.TotalChannels)
	db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN direction = 'inbound' THEN 1 ELSE 0 END), SUM(CASE WHEN direction = 'outbound' THEN 1 ELSE 0 END) FROM messages`).
		Scan(&s.TotalMessages, &s.InboundMessages, &s.OutboundMessages)
	return s, nil
}

func (db *DB) GetBotStats(userID string) (*store.BotStats, error) {
	s := &store.BotStats{}
	err := db.QueryRow(`
		SELECT
			COUNT(*),
			SUM(CASE WHEN status = 'connected' THEN 1 ELSE 0 END),
			COALESCE(SUM(msg_count), 0)
		FROM bots WHERE user_id = ?`, userID,
	).Scan(&s.TotalBots, &s.OnlineBots, &s.TotalMessages)
	if err != nil {
		return nil, err
	}
	db.QueryRow(`SELECT COUNT(*) FROM channels WHERE bot_id IN (SELECT id FROM bots WHERE user_id = ?)`, userID).Scan(&s.TotalChannels)
	return s, nil
}

func (db *DB) ListRecentContacts(botID string, limit int) ([]store.RecentContact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT from_user_id, MAX(created_at), COUNT(*)
		FROM messages WHERE bot_id = ? AND direction = 'inbound' AND from_user_id != ''
		GROUP BY from_user_id ORDER BY MAX(created_at) DESC LIMIT ?`,
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
	var epoch *int64
	db.QueryRow(`SELECT MAX(last_msg_at) FROM bots WHERE user_id = ?`, userID).Scan(&epoch)
	if epoch == nil {
		return nil
	}
	t := time.Unix(*epoch, 0)
	return &t
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
		placeholders[i] = "?"
		args = append(args, id)
	}

	rows, err := db.Query(
		"SELECT DISTINCT bot_id FROM messages WHERE bot_id IN ("+strings.Join(placeholders, ",")+") AND context_token != '' AND created_at > ? - ?",
		append(append(args[1:], db.now()), args[0])...,
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
