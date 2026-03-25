package app

import (
	"log/slog"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

// retryDelays defines the delays between delivery attempts.
// Attempt 0: immediate (already done by DeliverEvent).
// Attempt 1: after 10 seconds.
// Attempt 2: after 60 seconds.
var retryDelays = []time.Duration{
	0,
	10 * time.Second,
	60 * time.Second,
}

const maxRetries = 3

// DeliverWithRetry attempts to deliver an event, retrying up to maxRetries times.
// The first attempt is synchronous and returns immediately. If it fails, retries
// are scheduled in a background goroutine so the caller is not blocked.
func (d *Dispatcher) DeliverWithRetry(inst *store.AppInstallation, event *Event) *DeliveryResult {
	// First attempt (synchronous).
	result, err := d.DeliverEvent(inst, event)
	if err == nil {
		return result
	}

	slog.Warn("event delivery failed, scheduling retries",
		"installation", inst.ID, "event", event.ID, "err", err)

	// Remaining retries run in the background.
	go d.retryInBackground(inst, event)

	// Return whatever we got from the first attempt (may be nil on hard error).
	if result == nil {
		result = &DeliveryResult{}
	}
	return result
}

// retryInBackground runs retry attempts 1 and 2 with their respective delays.
func (d *Dispatcher) retryInBackground(inst *store.AppInstallation, event *Event) {
	for attempt := 1; attempt < maxRetries; attempt++ {
		delay := retryDelays[attempt]
		slog.Info("retry scheduled",
			"installation", inst.ID, "event", event.ID,
			"attempt", attempt, "delay", delay)

		time.Sleep(delay)

		result, err := d.deliverRetryAttempt(inst, event, attempt)
		if err == nil {
			slog.Info("retry succeeded",
				"installation", inst.ID, "event", event.ID,
				"attempt", attempt, "status", result.StatusCode)
			return
		}

		slog.Warn("retry attempt failed",
			"installation", inst.ID, "event", event.ID,
			"attempt", attempt, "err", err)
	}

	slog.Error("all retry attempts exhausted",
		"installation", inst.ID, "event", event.ID)
}

// deliverRetryAttempt performs a single retry delivery attempt and logs appropriately.
func (d *Dispatcher) deliverRetryAttempt(inst *store.AppInstallation, event *Event, attempt int) (*DeliveryResult, error) {
	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		// The DeliverEvent call already logged with retryCount=0; update that
		// log entry is not practical since we create a new log per attempt.
		// The retryCount in the latest log reflects the overall attempt.
		return result, err
	}
	return result, nil
}
