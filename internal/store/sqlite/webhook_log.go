package sqlite

import (
	"encoding/json"
	"fmt"

	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) CreateWebhookLog(log *store.WebhookLog) (int64, error) {
	result, err := db.Exec(`INSERT INTO webhook_logs (bot_id, channel_id, message_id, plugin_id, plugin_version, status)
		VALUES (?, ?, ?, ?, ?, 'pending')`,
		log.BotID, log.ChannelID, log.MessageID, log.PluginID, log.PluginVersion,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateWebhookLogRequest(id int64, status, url, method, body string) error {
	_, err := db.Exec(`UPDATE webhook_logs SET status=?, request_url=?, request_method=?, request_body=?, updated_at=unixepoch() WHERE id=?`,
		status, url, method, body, id)
	return err
}

func (db *DB) UpdateWebhookLogResponse(id int64, status string, respStatus int, respBody string, durationMs int) error {
	_, err := db.Exec(`UPDATE webhook_logs SET status=?, response_status=?, response_body=?, duration_ms=?, updated_at=unixepoch() WHERE id=?`,
		status, respStatus, respBody, durationMs, id)
	return err
}

func (db *DB) UpdateWebhookLogResult(id int64, status, scriptError string, replies []string) error {
	repliesJSON, _ := json.Marshal(replies)
	_, err := db.Exec(`UPDATE webhook_logs SET status=?, script_error=?, replies=?, updated_at=unixepoch() WHERE id=?`,
		status, scriptError, repliesJSON, id)
	return err
}

func (db *DB) UpdateWebhookLogPluginVersion(id int64, version string) error {
	_, err := db.Exec(`UPDATE webhook_logs SET plugin_version = ? WHERE id = ?`, version, id)
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
		created_at, updated_at
		FROM webhook_logs WHERE bot_id = ?`
	args := []any{botID}
	if channelID != "" {
		query += " AND channel_id = ?"
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
		var repliesStr string
		if err := rows.Scan(&l.ID, &l.BotID, &l.ChannelID, &l.MessageID, &l.PluginID, &l.PluginVersion, &l.Status,
			&l.RequestURL, &l.RequestMethod, &l.RequestBody,
			&l.ResponseStatus, &l.ResponseBody,
			&l.ScriptError, &repliesStr, &l.DurationMs,
			&l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.Replies = json.RawMessage(repliesStr)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) CleanOldWebhookLogs(days int) error {
	_, err := db.Exec("DELETE FROM webhook_logs WHERE created_at < unixepoch() - 86400 * ?", days)
	return err
}
