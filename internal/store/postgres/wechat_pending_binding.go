package postgres

import (
	"database/sql"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const wechatPendingBindingSelectCols = `id, event_id, provider_bot_id, bot_id, binding_id, role_id, session_id, status,
	external_chat_id, last_finalize_event_id, last_error,
	EXTRACT(EPOCH FROM created_at)::BIGINT,
	EXTRACT(EPOCH FROM expires_at)::BIGINT,
	COALESCE(EXTRACT(EPOCH FROM finalized_at)::BIGINT, 0),
	EXTRACT(EPOCH FROM updated_at)::BIGINT`

func scanWechatPendingBinding(scanner interface{ Scan(...any) error }) (*store.WechatPendingBinding, error) {
	row := &store.WechatPendingBinding{}
	var externalChatID sql.NullString
	var lastFinalizeEventID sql.NullString
	var lastError sql.NullString
	var finalizedAt int64
	if err := scanner.Scan(
		&row.ID,
		&row.EventID,
		&row.ProviderBotID,
		&row.BotID,
		&row.BindingID,
		&row.RoleID,
		&row.SessionID,
		&row.Status,
		&externalChatID,
		&lastFinalizeEventID,
		&lastError,
		&row.CreatedAt,
		&row.ExpiresAt,
		&finalizedAt,
		&row.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if externalChatID.Valid {
		row.ExternalChatID = externalChatID.String
	}
	if lastFinalizeEventID.Valid {
		row.LastFinalizeEventID = lastFinalizeEventID.String
	}
	if lastError.Valid {
		row.LastError = lastError.String
	}
	if finalizedAt > 0 {
		row.FinalizedAt = finalizedAt
	}
	return row, nil
}

func normalizeWechatPendingStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case store.WechatPendingBindingStatusFinalized:
		return store.WechatPendingBindingStatusFinalized
	case store.WechatPendingBindingStatusExpired:
		return store.WechatPendingBindingStatusExpired
	case store.WechatPendingBindingStatusFailed:
		return store.WechatPendingBindingStatusFailed
	default:
		return store.WechatPendingBindingStatusPending
	}
}

func (db *DB) CreateWechatPendingBinding(in store.WechatPendingBindingCreateInput) (*store.WechatPendingBinding, bool, error) {
	status := normalizeWechatPendingStatus(in.Status)
	expiresAt := in.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = db.now().Add(10 * time.Minute)
	}
	var insertedID int64
	// Insert through CTE so we can detect if row was inserted or dedup hit.
	err := db.QueryRow(`
WITH ins AS (
	INSERT INTO wechat_pending_bindings (
		event_id, provider_bot_id, bot_id, binding_id, role_id, session_id, status, expires_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	ON CONFLICT(event_id) DO NOTHING
	RETURNING id
)
SELECT COALESCE((SELECT id FROM ins), 0)
`,
		in.EventID,
		in.ProviderBotID,
		in.BotID,
		in.BindingID,
		in.RoleID,
		in.SessionID,
		status,
		expiresAt,
	).Scan(&insertedID)
	if err != nil {
		return nil, false, err
	}
	row, err := scanWechatPendingBinding(db.QueryRow(
		"SELECT "+wechatPendingBindingSelectCols+" FROM wechat_pending_bindings WHERE event_id = $1 LIMIT 1",
		in.EventID,
	))
	if err != nil {
		return nil, false, err
	}
	return row, insertedID > 0, nil
}

func (db *DB) GetLatestPendingWechatBinding(botID, providerBotID string, now time.Time) (*store.WechatPendingBinding, error) {
	row, err := scanWechatPendingBinding(db.QueryRow(
		"SELECT "+wechatPendingBindingSelectCols+" FROM wechat_pending_bindings WHERE bot_id = $1 AND provider_bot_id = $2 AND status = $3 AND expires_at >= $4 ORDER BY created_at DESC LIMIT 1",
		botID, providerBotID, store.WechatPendingBindingStatusPending, now,
	))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (db *DB) FinalizeWechatPendingBinding(id int64, externalChatID, finalizeEventID string, finalizedAt time.Time) (bool, error) {
	res, err := db.Exec(`UPDATE wechat_pending_bindings
		SET status = $1, external_chat_id = $2, last_finalize_event_id = $3, finalized_at = $4, updated_at = $5, last_error = ''
		WHERE id = $6 AND status = $7`,
		store.WechatPendingBindingStatusFinalized,
		externalChatID,
		finalizeEventID,
		finalizedAt,
		db.now(),
		id,
		store.WechatPendingBindingStatusPending,
	)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (db *DB) MarkWechatPendingBindingRetry(id int64, finalizeEventID, lastError string, now time.Time) error {
	_, err := db.Exec(`UPDATE wechat_pending_bindings
		SET last_finalize_event_id = $1, last_error = $2, updated_at = $3
		WHERE id = $4`,
		finalizeEventID,
		lastError,
		now,
		id,
	)
	return err
}

