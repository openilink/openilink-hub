package postgres

import (
	"fmt"

	"github.com/openilink/openilink-hub/internal/store"
)

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func (db *DB) CreateEventLog(log *store.AppEventLog) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO app_event_logs (installation_id, trace_id, event_type, event_id, request_body, status)
		VALUES ($1,$2,$3,$4,$5,'pending') RETURNING id`,
		log.InstallationID, log.TraceID, log.EventType, log.EventID, log.RequestBody,
	).Scan(&id)
	return id, err
}

func (db *DB) UpdateEventLogDelivered(id int64, respStatus int, respBody string, durationMs int) error {
	_, err := db.Exec(`UPDATE app_event_logs SET status='delivered', response_status=$1, response_body=$2, duration_ms=$3 WHERE id=$4`,
		respStatus, truncateStr(respBody, 4096), durationMs, id)
	return err
}

func (db *DB) UpdateEventLogFailed(id int64, errMsg string, retryCount int, durationMs int) error {
	status := "failed"
	if retryCount < 3 {
		status = "retrying"
	}
	_, err := db.Exec(`UPDATE app_event_logs SET status=$1, error=$2, retry_count=$3, duration_ms=$4 WHERE id=$5`,
		status, errMsg, retryCount, durationMs, id)
	return err
}

func (db *DB) ListEventLogs(installationID string, limit int) ([]store.AppEventLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT id, installation_id, trace_id, event_type, event_id,
		request_body, response_status, response_body,
		status, retry_count, error, duration_ms,
		EXTRACT(EPOCH FROM created_at)::BIGINT
		FROM app_event_logs WHERE installation_id = $1
		ORDER BY id DESC LIMIT %d`, limit), installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []store.AppEventLog
	for rows.Next() {
		var l store.AppEventLog
		if err := rows.Scan(&l.ID, &l.InstallationID, &l.TraceID, &l.EventType, &l.EventID,
			&l.RequestBody, &l.ResponseStatus, &l.ResponseBody,
			&l.Status, &l.RetryCount, &l.Error, &l.DurationMs,
			&l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) CreateAPILog(log *store.AppAPILog) error {
	_, err := db.Exec(`INSERT INTO app_api_logs (installation_id, trace_id, method, path, request_body, status_code, response_body, duration_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		log.InstallationID, log.TraceID, log.Method, log.Path,
		truncateStr(log.RequestBody, 4096), log.StatusCode,
		truncateStr(log.ResponseBody, 4096), log.DurationMs)
	return err
}

func (db *DB) ListAPILogs(installationID string, limit int) ([]store.AppAPILog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT id, installation_id, trace_id, method, path,
		request_body, status_code, response_body, duration_ms,
		EXTRACT(EPOCH FROM created_at)::BIGINT
		FROM app_api_logs WHERE installation_id = $1
		ORDER BY id DESC LIMIT %d`, limit), installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []store.AppAPILog
	for rows.Next() {
		var l store.AppAPILog
		if err := rows.Scan(&l.ID, &l.InstallationID, &l.TraceID, &l.Method, &l.Path,
			&l.RequestBody, &l.StatusCode, &l.ResponseBody, &l.DurationMs,
			&l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) CleanOldAppLogs(days int) error {
	now := db.now()
	_, _ = db.Exec("DELETE FROM app_event_logs WHERE created_at < $1::timestamptz - (INTERVAL '1 day' * $2)", now, days)
	_, _ = db.Exec("DELETE FROM app_api_logs WHERE created_at < $1::timestamptz - (INTERVAL '1 day' * $2)", now, days)
	return nil
}
