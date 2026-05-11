package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

const syncOutboxSelectCols = `id, event_id, event_type, partition_key, payload,
	status, retry_count,
	EXTRACT(EPOCH FROM next_retry_at)::BIGINT,
	last_error,
	EXTRACT(EPOCH FROM created_at)::BIGINT,
	EXTRACT(EPOCH FROM updated_at)::BIGINT,
	COALESCE(EXTRACT(EPOCH FROM sent_at)::BIGINT, 0)`

func scanSyncOutbox(scanner interface{ Scan(...any) error }) (*store.SyncOutboxEvent, error) {
	e := &store.SyncOutboxEvent{}
	if err := scanner.Scan(
		&e.ID,
		&e.EventID,
		&e.EventType,
		&e.PartitionKey,
		&e.Payload,
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
	if len(e.Payload) == 0 {
		e.Payload = json.RawMessage(`{}`)
	}
	return e, nil
}

func (db *DB) EnqueueSyncOutboxEvent(in store.EnqueueOutboxInput) (*store.SyncOutboxEvent, bool, error) {
	if len(in.Payload) == 0 {
		in.Payload = json.RawMessage(`{}`)
	}
	var id int64
	err := db.QueryRow(`INSERT INTO sync_outbox (event_id, event_type, partition_key, payload, status, next_retry_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, 'pending', $5::timestamptz, $5::timestamptz)
		ON CONFLICT(event_id) DO NOTHING
		RETURNING id`,
		in.EventID, in.EventType, in.PartitionKey, in.Payload, db.now(),
	).Scan(&id)
	created := true
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, false, err
		}
		created = false
	}
	e, err := scanSyncOutbox(db.QueryRow("SELECT "+syncOutboxSelectCols+" FROM sync_outbox WHERE event_id = $1", in.EventID))
	if err != nil {
		return nil, false, err
	}
	return e, created, nil
}

func (db *DB) ClaimPendingSyncOutboxEvents(opt store.ClaimOutboxOptions) ([]store.SyncOutboxEvent, error) {
	limit := opt.Limit
	if limit <= 0 {
		limit = 100
	}
	now := opt.Now
	if now.IsZero() {
		now = db.now()
	}

	rows, err := db.Query(`WITH claimed AS (
		SELECT id FROM sync_outbox
		WHERE status = 'pending' AND next_retry_at <= $1::timestamptz
		ORDER BY id
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	), updated AS (
		UPDATE sync_outbox s
		SET status = 'processing', updated_at = NOW()
		FROM claimed
		WHERE s.id = claimed.id
		RETURNING s.id
	)
	SELECT `+syncOutboxSelectCols+` FROM sync_outbox
	WHERE id IN (SELECT id FROM updated)
	ORDER BY id`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []store.SyncOutboxEvent
	for rows.Next() {
		e, err := scanSyncOutbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (db *DB) MarkSyncOutboxEventSent(eventID string, sentAt time.Time) error {
	if sentAt.IsZero() {
		sentAt = db.now()
	}
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = 'sent', sent_at = $1::timestamptz, updated_at = NOW(), last_error = ''
		WHERE event_id = $2`, sentAt, eventID)
	return err
}

func (db *DB) MarkSyncOutboxEventRetry(in store.RetryOutboxInput) error {
	status := in.Status
	if status == "" {
		status = store.OutboxStatusPending
	}
	next := in.NextRetryAt
	if next.IsZero() {
		next = db.now()
	}
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = $1, retry_count = $2, next_retry_at = $3::timestamptz, last_error = $4, updated_at = NOW()
		WHERE event_id = $5`,
		status, in.RetryCount, next, in.LastError, in.EventID,
	)
	return err
}

func (db *DB) MarkSyncOutboxEventDead(eventID, lastError string) error {
	_, err := db.Exec(`UPDATE sync_outbox
		SET status = 'dead', last_error = $1, updated_at = NOW()
		WHERE event_id = $2`, lastError, eventID)
	return err
}

func (db *DB) CreateAdminSyncInboxEvent(eventID string) (bool, error) {
	var created bool
	err := db.QueryRow(`WITH ins AS (
		INSERT INTO admin_sync_inbox (event_id)
		VALUES ($1)
		ON CONFLICT(event_id) DO NOTHING
		RETURNING event_id
	)
	SELECT EXISTS(SELECT 1 FROM ins)`, eventID).Scan(&created)
	return created, err
}

func (db *DB) HasAdminSyncInboxEvent(eventID string) (bool, error) {
	var existing bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM admin_sync_inbox WHERE event_id = $1)", eventID).Scan(&existing)
	return existing, err
}
