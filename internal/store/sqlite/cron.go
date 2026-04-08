package sqlite

import (
	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) CreateCronJob(job *store.CronJob) (*store.CronJob, error) {
	job.ID = uuid.New().String()
	_, err := db.Exec(`INSERT INTO cron_jobs (id, bot_id, user_id, name, cron_expr, message, recipient, enabled, next_run_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.BotID, job.UserID, job.Name, job.CronExpr, job.Message, job.Recipient, job.Enabled, job.NextRunAt)
	if err != nil {
		return nil, err
	}
	return db.GetCronJob(job.ID)
}

func (db *DB) GetCronJob(id string) (*store.CronJob, error) {
	j := &store.CronJob{}
	err := db.QueryRow(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE id = ?`, id).
		Scan(&j.ID, &j.BotID, &j.UserID, &j.Name, &j.CronExpr, &j.Message, &j.Recipient, &j.Enabled, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (db *DB) ListCronJobsByBot(botID string) ([]store.CronJob, error) {
	rows, err := db.Query(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE bot_id = ? ORDER BY created_at`, botID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []store.CronJob
	for rows.Next() {
		var j store.CronJob
		if err := rows.Scan(&j.ID, &j.BotID, &j.UserID, &j.Name, &j.CronExpr, &j.Message, &j.Recipient, &j.Enabled, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (db *DB) UpdateCronJob(id, name, cronExpr, message, recipient string, enabled bool, nextRunAt *int64) error {
	_, err := db.Exec(`UPDATE cron_jobs SET name = ?, cron_expr = ?, message = ?, recipient = ?, enabled = ?, next_run_at = ? WHERE id = ?`,
		name, cronExpr, message, recipient, enabled, nextRunAt, id)
	return err
}

func (db *DB) DeleteCronJob(id string) error {
	_, err := db.Exec(`DELETE FROM cron_jobs WHERE id = ?`, id)
	return err
}

func (db *DB) ClaimDueCronJobs(now int64) ([]store.CronJob, error) {
	// Atomically claim due jobs by setting next_run_at = NULL, then return them.
	// SQLite serializes writes, so this is safe against concurrent callers.
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE enabled = 1 AND next_run_at IS NOT NULL AND next_run_at <= ?`, now)
	if err != nil {
		return nil, err
	}
	var jobs []store.CronJob
	for rows.Next() {
		var j store.CronJob
		if err := rows.Scan(&j.ID, &j.BotID, &j.UserID, &j.Name, &j.CronExpr, &j.Message, &j.Recipient, &j.Enabled, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(jobs) > 0 {
		// Mark claimed by clearing next_run_at
		_, err = tx.Exec(`UPDATE cron_jobs SET next_run_at = NULL WHERE enabled = 1 AND next_run_at IS NOT NULL AND next_run_at <= ?`, now)
		if err != nil {
			return nil, err
		}
	}

	return jobs, tx.Commit()
}

func (db *DB) MarkCronJobRun(id string, lastRunAt int64, nextRunAt *int64) error {
	_, err := db.Exec(`UPDATE cron_jobs SET last_run_at = ?, next_run_at = ? WHERE id = ?`, lastRunAt, nextRunAt, id)
	return err
}

func (db *DB) SetCronJobNextRun(id string, nextRunAt *int64) error {
	_, err := db.Exec(`UPDATE cron_jobs SET next_run_at = ? WHERE id = ?`, nextRunAt, id)
	return err
}

func (db *DB) ListStuckCronJobs() ([]store.CronJob, error) {
	rows, err := db.Query(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE enabled = 1 AND next_run_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []store.CronJob
	for rows.Next() {
		var j store.CronJob
		if err := rows.Scan(&j.ID, &j.BotID, &j.UserID, &j.Name, &j.CronExpr, &j.Message, &j.Recipient, &j.Enabled, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
