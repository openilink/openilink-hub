package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const syncOutboxSelectCols = `id, event_id, event_type, partition_key, payload,
	status, retry_count, next_retry_at, last_error,
	created_at, updated_at, COALESCE(sent_at, 0)`

func scanSyncOutbox(scanner interface{ Scan(...any) error }) (*store.SyncOutboxEvent, error) {
	e := &store.SyncOutboxEvent{}
	var payload string
	if err := scanner.Scan(
		&e.ID,
		&e.EventID,
		&e.EventType,
		&e.PartitionKey,
		&payload,
		&e.Status,
		&e.RetryCount,
		&e.NextRetryAt,
		&e.LastError,
		&e.CreatedAt,
		&e.UpdatedAt,
		&e.SentAt,
	); err != nil {
		return nil, err
	}
	if payload == "" {
		e.Payload = json.RawMessage(`{}`)
	} else {
		e.Payload = json.RawMessage(payload)
	}
	return e, nil
}

func (db *DB) EnqueueSyncOutboxEvent(in store.EnqueueOutboxInput) (*store.SyncOutboxEvent, bool, error) {
	if len(in.Payload) == 0 {
		in.Payload = json.RawMessage(`{}`)
	}
	result, err := db.Exec(`INSERT INTO sync_outbox (event_id, event_type, partition_key, payload, status, next_retry_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?)
		ON CONFLICT(event_id) DO NOTHING`,
		in.EventID, in.EventType, in.PartitionKey, string(in.Payload), db.now(), db.now(),
	)
	if err != nil {
		return nil, false, err
	}
	rows, _ := result.RowsAffected()
	e, err := scanSyncOutbox(db.QueryRow("SELECT "+syncOutboxSelectCols+" FROM sync_outbox WHERE event_id = ?", in.EventID))
	if err != nil {
		return nil, false, err
	}
	return e, rows > 0, nil
}

func (db *DB) ClaimPendingSyncOutboxEvents(opt store.ClaimOutboxOptions) ([]store.SyncOutboxEvent, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 100
	}
	now := opt.Now.Unix()
	if now <= 0 {
		now = db.now()
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(fmt.Sprintf(`SELECT id, event_id FROM sync_outbox
		WHERE status = 'pending' AND next_retry_at <= ?
		ORDER BY id ASC LIMIT %d`, limit), now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		var eventID string
		if err := rows.Scan(&id, &eventID); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	for _, id := range ids {
		if _, err := tx.Exec("UPDATE sync_outbox SET status = 'processing', updated_at = ? WHERE id = ?", db.now(), id); err != nil {
			return nil, err
		}
	}

	q := "SELECT " + syncOutboxSelectCols + " FROM sync_outbox WHERE id IN ("
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			q += ","
		}
		q += "?"
		args = append(args, id)
	}
	q += ") ORDER BY id ASC"

	outRows, err := tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer outRows.Close()

	var out []store.SyncOutboxEvent
	for outRows.Next() {
		e, err := scanSyncOutbox(outRows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	if err := outRows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (db *DB) MarkSyncOutboxEventSent(eventID string, sentAt time.Time) error {
	sent := sentAt.Unix()
	if sent <= 0 {
		sent = db.now()
	}
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = 'sent', sent_at = ?, updated_at = ?, last_error = ''
		WHERE event_id = ?`, sent, db.now(), eventID)
	return err
}

func (db *DB) MarkSyncOutboxEventRetry(in store.RetryOutboxInput) error {
	status := in.Status
	if status == "" {
		status = store.OutboxStatusPending
	}
	next := in.NextRetryAt.Unix()
	if next <= 0 {
		next = db.now()
	}
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = ?, retry_count = ?, next_retry_at = ?, last_error = ?, updated_at = ?
		WHERE event_id = ?`,
		status, in.RetryCount, next, in.LastError, db.now(), in.EventID,
	)
	return err
}

func (db *DB) MarkSyncOutboxEventDead(eventID, lastError string) error {
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = 'dead', last_error = ?, updated_at = ?
		WHERE event_id = ?`, lastError, db.now(), eventID)
	return err
}

func (db *DB) CreateAdminSyncInboxEvent(eventID string) (bool, error) {
	res, err := db.Exec("INSERT INTO admin_sync_inbox (event_id, created_at) VALUES (?, ?) ON CONFLICT(event_id) DO NOTHING", eventID, db.now())
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (db *DB) HasAdminSyncInboxEvent(eventID string) (bool, error) {
	var existing string
	err := db.QueryRow("SELECT event_id FROM admin_sync_inbox WHERE event_id = ?", eventID).Scan(&existing)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
