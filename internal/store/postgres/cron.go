package postgres

import (
	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) CreateCronJob(job *store.CronJob) (*store.CronJob, error) {
	job.ID = uuid.New().String()
	_, err := db.Exec(`INSERT INTO cron_jobs (id, bot_id, user_id, name, cron_expr, message, recipient, enabled, next_run_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		job.ID, job.BotID, job.UserID, job.Name, job.CronExpr, job.Message, job.Recipient, job.Enabled, job.NextRunAt)
	if err != nil {
		return nil, err
	}
	return db.GetCronJob(job.ID)
}

func (db *DB) GetCronJob(id string) (*store.CronJob, error) {
	j := &store.CronJob{}
	err := db.QueryRow(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE id = $1`, id).
		Scan(&j.ID, &j.BotID, &j.UserID, &j.Name, &j.CronExpr, &j.Message, &j.Recipient, &j.Enabled, &j.LastRunAt, &j.NextRunAt, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (db *DB) ListCronJobsByBot(botID string) ([]store.CronJob, error) {
	rows, err := db.Query(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE bot_id = $1 ORDER BY created_at`, botID)
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
	_, err := db.Exec(`UPDATE cron_jobs SET name = $1, cron_expr = $2, message = $3, recipient = $4, enabled = $5, next_run_at = $6 WHERE id = $7`,
		name, cronExpr, message, recipient, enabled, nextRunAt, id)
	return err
}

func (db *DB) DeleteCronJob(id string) error {
	_, err := db.Exec(`DELETE FROM cron_jobs WHERE id = $1`, id)
	return err
}

func (db *DB) ClaimDueCronJobs(now int64) ([]store.CronJob, error) {
	// Atomically claim and return due jobs using UPDATE ... RETURNING.
	rows, err := db.Query(`UPDATE cron_jobs SET next_run_at = NULL
		WHERE enabled = TRUE AND next_run_at IS NOT NULL AND next_run_at <= $1
		RETURNING id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at`, now)
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

func (db *DB) MarkCronJobRun(id string, lastRunAt int64, nextRunAt *int64) error {
	_, err := db.Exec(`UPDATE cron_jobs SET last_run_at = $1, next_run_at = $2 WHERE id = $3`, lastRunAt, nextRunAt, id)
	return err
}

func (db *DB) SetCronJobNextRun(id string, nextRunAt *int64) error {
	_, err := db.Exec(`UPDATE cron_jobs SET next_run_at = $1 WHERE id = $2`, nextRunAt, id)
	return err
}

func (db *DB) ListStuckCronJobs() ([]store.CronJob, error) {
	rows, err := db.Query(`SELECT id, bot_id, user_id, name, cron_expr, message, recipient, enabled, last_run_at, next_run_at, created_at
		FROM cron_jobs WHERE enabled = TRUE AND next_run_at IS NULL`)
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
