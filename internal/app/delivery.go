package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/database"
)

const (
	// deliveryTimeout is the HTTP timeout for initial event delivery.
	deliveryTimeout = 3 * time.Second

	// envelopeVersion is the current event envelope version.
	envelopeVersion = 1

	// maxResponseBody is the maximum response body size to read.
	maxResponseBody = 4096
)

// Event represents an event to deliver to an app installation.
type Event struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data"`
}

// DeliveryResult holds the outcome of an event delivery attempt.
type DeliveryResult struct {
	Reply      string `json:"reply,omitempty"`
	ReplyType  string `json:"reply_type,omitempty"`
	StatusCode int    `json:"status_code"`
}

// eventEnvelope is the JSON structure POSTed to the app's request_url.
type eventEnvelope struct {
	V              int            `json:"v"`
	Type           string         `json:"type"`
	TraceID        string         `json:"trace_id"`
	InstallationID string         `json:"installation_id"`
	Bot            envelopBot     `json:"bot"`
	Event          *Event         `json:"event"`
}

type envelopBot struct {
	ID string `json:"id"`
}

// syncReply is the optional reply parsed from the app's HTTP response.
type syncReply struct {
	Reply     string `json:"reply"`
	ReplyType string `json:"reply_type"`
}

// eventLogger is the interface used for event logging operations.
// This allows tests to mock the database layer.
type eventLogger interface {
	CreateEventLog(log *database.AppEventLog) (int64, error)
	UpdateEventLogDelivered(id int64, respStatus int, respBody string, durationMs int) error
	UpdateEventLogFailed(id int64, errMsg string, retryCount int, durationMs int) error
}

// appStore is the interface used for app lookup operations.
type appStore interface {
	ListInstallationsByBot(botID string) ([]database.AppInstallation, error)
	GetApp(id string) (*database.App, error)
	GetInstallationByHandle(botID, handle string) (*database.AppInstallation, error)
}

// Dispatcher delivers events to app installations.
type Dispatcher struct {
	DB     *database.DB
	Client *http.Client

	// dbLog and appDB are optional interface overrides for testing.
	// When nil, the real DB is used.
	dbLog eventLogger
	appDB appStore
}

// NewDispatcher creates a new Dispatcher with a default HTTP client.
func NewDispatcher(db *database.DB) *Dispatcher {
	return &Dispatcher{
		DB: db,
		Client: &http.Client{
			Timeout: deliveryTimeout,
		},
	}
}

func (d *Dispatcher) logDB() eventLogger {
	if d.dbLog != nil {
		return d.dbLog
	}
	return d.DB
}

func (d *Dispatcher) store() appStore {
	if d.appDB != nil {
		return d.appDB
	}
	return d.DB
}

// DeliverEvent posts a signed event payload to the installation's request_url
// and logs the delivery attempt. Returns the delivery result or an error.
func (d *Dispatcher) DeliverEvent(inst *database.AppInstallation, event *Event) (*DeliveryResult, error) {
	if inst.RequestURL == "" {
		return nil, fmt.Errorf("installation %s has no request_url configured", inst.ID)
	}

	traceID := "tr_" + uuid.New().String()

	// Determine envelope type.
	envelopeType := "event_callback"
	if isCommandEvent(event.Type) {
		envelopeType = "command"
	}

	envelope := eventEnvelope{
		V:              envelopeVersion,
		Type:           envelopeType,
		TraceID:        traceID,
		InstallationID: inst.ID,
		Bot:            envelopBot{ID: inst.BotID},
		Event:          event,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	// Create event log (pending).
	logEntry := &database.AppEventLog{
		InstallationID: inst.ID,
		TraceID:        traceID,
		EventType:      event.Type,
		EventID:        event.ID,
		RequestBody:    string(body),
	}
	logID, err := d.logDB().CreateEventLog(logEntry)
	if err != nil {
		slog.Error("failed to create event log", "installation", inst.ID, "err", err)
		// Continue delivery even if logging fails.
	}

	// Compute signature.
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := computeSignature(inst.SigningSecret, timestamp, body)

	// Build request.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, inst.RequestURL, bytes.NewReader(body))
	if err != nil {
		d.markFailed(logID, err.Error(), 0, 0)
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Id", inst.AppID)
	req.Header.Set("X-Installation-Id", inst.ID)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", "sha256="+signature)
	req.Header.Set("X-Trace-Id", traceID)

	// Execute request.
	start := time.Now()
	resp, err := d.Client.Do(req)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		d.markFailed(logID, err.Error(), 0, durationMs)
		return nil, fmt.Errorf("http request to %s failed: %w", inst.RequestURL, err)
	}
	defer resp.Body.Close()

	// Read response body (capped).
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	respStr := string(respBody)

	result := &DeliveryResult{
		StatusCode: resp.StatusCode,
	}

	// Parse optional sync reply.
	if len(respBody) > 0 {
		var sr syncReply
		if json.Unmarshal(respBody, &sr) == nil && sr.Reply != "" {
			result.Reply = sr.Reply
			result.ReplyType = sr.ReplyType
			if result.ReplyType == "" {
				result.ReplyType = "text"
			}
		}
	}

	// Update event log.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if logID > 0 {
			if err := d.logDB().UpdateEventLogDelivered(logID, resp.StatusCode, respStr, durationMs); err != nil {
				slog.Error("failed to update event log as delivered", "logID", logID, "err", err)
			}
		}
		slog.Info("event delivered",
			"installation", inst.ID, "trace", traceID,
			"status", resp.StatusCode, "duration_ms", durationMs)
	} else {
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, respStr)
		d.markFailed(logID, errMsg, 0, durationMs)
		return result, fmt.Errorf("delivery failed with status %d", resp.StatusCode)
	}

	return result, nil
}

// computeSignature returns the HMAC-SHA256 hex digest of "{timestamp}:{body}"
// using the signing secret as key.
func computeSignature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// markFailed updates the event log entry as failed.
func (d *Dispatcher) markFailed(logID int64, errMsg string, retryCount, durationMs int) {
	if logID <= 0 {
		return
	}
	if err := d.logDB().UpdateEventLogFailed(logID, errMsg, retryCount, durationMs); err != nil {
		slog.Error("failed to update event log as failed", "logID", logID, "err", err)
	}
}

// isCommandEvent returns true if the event type represents a command invocation.
func isCommandEvent(eventType string) bool {
	return eventType == "command"
}

// NewEvent creates an Event with a generated ID and current timestamp.
func NewEvent(eventType string, data any) *Event {
	return &Event{
		Type:      eventType,
		ID:        "evt_" + uuid.New().String(),
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}
