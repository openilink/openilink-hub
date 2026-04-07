package store

// CronJob represents a scheduled task that sends a message via a bot.
type CronJob struct {
	ID        string `json:"id"`
	BotID     string `json:"bot_id"`
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	CronExpr  string `json:"cron_expr"`
	Message   string `json:"message"`
	Recipient string `json:"recipient"`
	Enabled   bool   `json:"enabled"`
	LastRunAt *int64 `json:"last_run_at,omitempty"`
	NextRunAt *int64 `json:"next_run_at,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type CronStore interface {
	CreateCronJob(job *CronJob) (*CronJob, error)
	GetCronJob(id string) (*CronJob, error)
	ListCronJobsByBot(botID string) ([]CronJob, error)
	UpdateCronJob(id, name, cronExpr, message, recipient string, enabled bool, nextRunAt *int64) error
	DeleteCronJob(id string) error
	// GetDueCronJobs returns enabled jobs whose next_run_at <= now.
	GetDueCronJobs(now int64) ([]CronJob, error)
	// MarkCronJobRun updates last_run_at and next_run_at after a job fires.
	MarkCronJobRun(id string, lastRunAt int64, nextRunAt *int64) error
}
