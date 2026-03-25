package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) CreateWebhookLog(log *store.WebhookLog) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO webhook_logs (bot_id, channel_id, message_id, plugin_id, plugin_version, status)
		VALUES ($1, $2, $3, $4, $5, 'pending') RETURNING id`,
		log.BotID, log.ChannelID, log.MessageID, log.PluginID, log.PluginVersion,
	).Scan(&id)
	return id, err
}

func (db *DB) UpdateWebhookLogRequest(id int64, status, url, method, body string) error {
	_, err := db.Exec(`UPDATE webhook_logs SET status=$1, request_url=$2, request_method=$3, request_body=$4, updated_at=NOW() WHERE id=$5`,
		status, url, method, body, id)
	return err
}

func (db *DB) UpdateWebhookLogResponse(id int64, status string, respStatus int, respBody string, durationMs int) error {
	_, err := db.Exec(`UPDATE webhook_logs SET status=$1, response_status=$2, response_body=$3, duration_ms=$4, updated_at=NOW() WHERE id=$5`,
		status, respStatus, respBody, durationMs, id)
	return err
}

func (db *DB) UpdateWebhookLogResult(id int64, status, scriptError string, replies []string) error {
	repliesJSON, _ := json.Marshal(replies)
	_, err := db.Exec(`UPDATE webhook_logs SET status=$1, script_error=$2, replies=$3, updated_at=NOW() WHERE id=$4`,
		status, scriptError, repliesJSON, id)
	return err
}

func (db *DB) UpdateWebhookLogPluginVersion(id int64, version string) error {
	_, err := db.Exec(`UPDATE webhook_logs SET plugin_version = $1 WHERE id = $2`, version, id)
	return err
}

func (db *DB) ListWebhookLogs(botID, channelID string, limit int) ([]store.WebhookLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, bot_id, channel_id, message_id, plugin_id, plugin_version, status,
		request_url, request_method, request_body,
		response_status, response_body,
		script_error, replies, duration_ms,
		EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT
		FROM webhook_logs WHERE bot_id = $1`
	args := []any{botID}
	if channelID != "" {
		query += " AND channel_id = $2"
		args = append(args, channelID)
	}
	query += " ORDER BY id DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []store.WebhookLog
	for rows.Next() {
		var l store.WebhookLog
		if err := rows.Scan(&l.ID, &l.BotID, &l.ChannelID, &l.MessageID, &l.PluginID, &l.PluginVersion, &l.Status,
			&l.RequestURL, &l.RequestMethod, &l.RequestBody,
			&l.ResponseStatus, &l.ResponseBody,
			&l.ScriptError, &l.Replies, &l.DurationMs,
			&l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) CleanOldWebhookLogs(days int) error {
	_, err := db.Exec("DELETE FROM webhook_logs WHERE created_at < NOW() - INTERVAL '1 day' * $1", days)
	return err
}
