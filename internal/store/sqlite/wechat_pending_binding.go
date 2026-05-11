package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const wechatPendingBindingSelectCols = `id, event_id, provider_bot_id, bot_id, binding_id, role_id, session_id, status,
	external_chat_id, last_finalize_event_id, last_error, created_at, expires_at, finalized_at, updated_at`

func scanWechatPendingBinding(scanner interface{ Scan(...any) error }) (*store.WechatPendingBinding, error) {
	row := &store.WechatPendingBinding{}
	var externalChatID sql.NullString
	var lastFinalizeEventID sql.NullString
	var lastError sql.NullString
	var finalizedAt sql.NullInt64
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
	if finalizedAt.Valid {
		row.FinalizedAt = finalizedAt.Int64
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
	expiresAt := in.ExpiresAt.Unix()
	now := db.now()
	res, err := db.Exec(`INSERT INTO wechat_pending_bindings (
		event_id, provider_bot_id, bot_id, binding_id, role_id, session_id, status, expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(event_id) DO NOTHING`,
		in.EventID,
		in.ProviderBotID,
		in.BotID,
		in.BindingID,
		in.RoleID,
		in.SessionID,
		status,
		expiresAt,
		now,
		now,
	)
	if err != nil {
		return nil, false, err
	}
	affected, _ := res.RowsAffected()
	created := affected > 0
	row, err := scanWechatPendingBinding(db.QueryRow(
		"SELECT "+wechatPendingBindingSelectCols+" FROM wechat_pending_bindings WHERE event_id = ? LIMIT 1",
		in.EventID,
	))
	if err != nil {
		return nil, created, err
	}
	return row, created, nil
}

func (db *DB) GetLatestPendingWechatBinding(botID, providerBotID string, now time.Time) (*store.WechatPendingBinding, error) {
	row, err := scanWechatPendingBinding(db.QueryRow(
		"SELECT "+wechatPendingBindingSelectCols+" FROM wechat_pending_bindings WHERE bot_id = ? AND provider_bot_id = ? AND status = ? AND expires_at >= ? ORDER BY created_at DESC LIMIT 1",
		botID, providerBotID, store.WechatPendingBindingStatusPending, now.Unix(),
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
	now := db.now()
	res, err := db.Exec(`UPDATE wechat_pending_bindings
		SET status = ?, external_chat_id = ?, last_finalize_event_id = ?, finalized_at = ?, updated_at = ?, last_error = ''
		WHERE id = ? AND status = ?`,
		store.WechatPendingBindingStatusFinalized,
		externalChatID,
		finalizeEventID,
		finalizedAt.Unix(),
		now,
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
		SET last_finalize_event_id = ?, last_error = ?, updated_at = ?
		WHERE id = ?`,
		finalizeEventID,
		lastError,
		now.Unix(),
		id,
	)
	return err
}

