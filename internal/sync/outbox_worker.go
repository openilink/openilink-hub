package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

type OutboxWorkerConfig struct {
	BatchSize    int
	PollInterval time.Duration
	MaxRetries   int
}

type OutboxEventStore interface {
	ClaimPendingSyncOutboxEvents(opt store.ClaimOutboxOptions) ([]store.SyncOutboxEvent, error)
	MarkSyncOutboxEventSent(eventID string, sentAt time.Time) error
	MarkSyncOutboxEventRetry(in store.RetryOutboxInput) error
	MarkSyncOutboxEventDead(eventID, lastError string) error
}

type OutboxWorker struct {
	store  OutboxEventStore
	client SupabaseClient
	cfg    OutboxWorkerConfig
}

func NewOutboxWorker(s OutboxEventStore, c SupabaseClient, cfg OutboxWorkerConfig) *OutboxWorker {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 10
	}
	return &OutboxWorker{store: s, client: c, cfg: cfg}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.runOnce(ctx); err != nil {
			slog.Warn("outbox worker run failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *OutboxWorker) runOnce(ctx context.Context) error {
	if w.store == nil || w.client == nil {
		return nil
	}
	events, err := w.store.ClaimPendingSyncOutboxEvents(store.ClaimOutboxOptions{Limit: w.cfg.BatchSize, Now: time.Now()})
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := w.dispatch(ctx, e); err != nil {
			retryCount := e.RetryCount + 1
			if retryCount >= w.cfg.MaxRetries {
				_ = w.store.MarkSyncOutboxEventDead(e.EventID, err.Error())
				continue
			}
			next := time.Now().Add(backoffForRetry(retryCount))
			_ = w.store.MarkSyncOutboxEventRetry(store.RetryOutboxInput{
				EventID:     e.EventID,
				RetryCount:  retryCount,
				NextRetryAt: next,
				LastError:   err.Error(),
				Status:      store.OutboxStatusPending,
			})
			continue
		}
		_ = w.store.MarkSyncOutboxEventSent(e.EventID, time.Now())
	}
	return nil
}

func (w *OutboxWorker) dispatch(ctx context.Context, e store.SyncOutboxEvent) error {
	switch e.EventType {
	case store.OutboxEventMessageInbound, store.OutboxEventMessageOutbound:
		return w.client.UpsertMessageEvent(ctx, e.Payload)
	case store.OutboxEventPromptProfileChange:
		return w.client.UpsertPromptProfileEvent(ctx, e.Payload)
	case store.OutboxEventBindingInvalidated:
		return w.client.UpsertBindingInvalidatedEvent(ctx, e.Payload)
	default:
		return errors.New("unsupported outbox event type")
	}
}

func backoffForRetry(retryCount int) time.Duration {
	if retryCount < 1 {
		retryCount = 1
	}
	base := time.Second
	d := base * time.Duration(1<<(retryCount-1))
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

func ParseOutboxConfig(batchSize, pollMs, maxRetries int) OutboxWorkerConfig {
	cfg := OutboxWorkerConfig{BatchSize: batchSize, MaxRetries: maxRetries}
	if pollMs > 0 {
		cfg.PollInterval = time.Duration(pollMs) * time.Millisecond
	}
	return cfg
}

func ValidateWorkerDeps(s OutboxEventStore, c SupabaseClient) error {
	if s == nil {
		return fmt.Errorf("store is nil")
	}
	if c == nil {
		return fmt.Errorf("supabase client is nil")
	}
	return nil
}
