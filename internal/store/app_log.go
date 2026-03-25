package store

type AppEventLog struct {
	ID             int64  `json:"id"`
	InstallationID string `json:"installation_id"`
	TraceID        string `json:"trace_id"`
	EventType      string `json:"event_type"`
	EventID        string `json:"event_id"`
	RequestBody    string `json:"request_body,omitempty"`
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `json:"response_body,omitempty"`
	Status         string `json:"status"`
	RetryCount     int    `json:"retry_count"`
	Error          string `json:"error,omitempty"`
	DurationMs     int    `json:"duration_ms"`
	CreatedAt      int64  `json:"created_at"`
}

type AppAPILog struct {
	ID             int64  `json:"id"`
	InstallationID string `json:"installation_id"`
	TraceID        string `json:"trace_id"`
	Method         string `json:"method"`
	Path           string `json:"path"`
	RequestBody    string `json:"request_body,omitempty"`
	StatusCode     int    `json:"status_code"`
	ResponseBody   string `json:"response_body,omitempty"`
	DurationMs     int    `json:"duration_ms"`
	CreatedAt      int64  `json:"created_at"`
}

type AppLogStore interface {
	CreateEventLog(log *AppEventLog) (int64, error)
	UpdateEventLogDelivered(id int64, respStatus int, respBody string, durationMs int) error
	UpdateEventLogFailed(id int64, errMsg string, retryCount int, durationMs int) error
	ListEventLogs(installationID string, limit int) ([]AppEventLog, error)
	CreateAPILog(log *AppAPILog) error
	ListAPILogs(installationID string, limit int) ([]AppAPILog, error)
	CleanOldAppLogs(days int) error
}
