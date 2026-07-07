package storetest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestSyncOutbox(t *testing.T, s store.Store) {
	mkPayload := func(v any) json.RawMessage {
		b, _ := json.Marshal(v)
		return b
	}

	t.Run("EnqueueAndIdempotent", func(t *testing.T) {
		e, created, err := s.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
			EventID:      "evt-1",
			EventType:    store.OutboxEventMessageInbound,
			PartitionKey: "bot-1",
			Payload:      mkPayload(map[string]any{"message_id": 1}),
		})
		if err != nil {
			t.Fatalf("EnqueueSyncOutboxEvent: %v", err)
		}
		if !created || e == nil {
			t.Fatal("expected created outbox event")
		}

		_, created2, err := s.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
			EventID:      "evt-1",
			EventType:    store.OutboxEventMessageInbound,
			PartitionKey: "bot-1",
			Payload:      mkPayload(map[string]any{"message_id": 1}),
		})
		if err != nil {
			t.Fatalf("EnqueueSyncOutboxEvent dedup: %v", err)
		}
		if created2 {
			t.Fatal("expected idempotent enqueue created=false")
		}
	})

	t.Run("ClaimPending", func(t *testing.T) {
		_, _, _ = s.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
			EventID:      "evt-2",
			EventType:    store.OutboxEventMessageOutbound,
			PartitionKey: "bot-1",
			Payload:      mkPayload(map[string]any{"message_id": 2}),
		})

		claimed, err := s.ClaimPendingSyncOutboxEvents(store.ClaimOutboxOptions{
			Limit: 10,
			Now:   time.Now().Add(1 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ClaimPendingSyncOutboxEvents: %v", err)
		}
		if len(claimed) < 2 {
			t.Fatalf("expected >=2 claimed events, got %d", len(claimed))
		}
		for _, e := range claimed {
			if e.Status != store.OutboxStatusProcessing {
				t.Fatalf("expected status processing, got %q", e.Status)
			}
		}
	})

	t.Run("SentRetryDeadLifecycle", func(t *testing.T) {
		if err := s.MarkSyncOutboxEventSent("evt-1", time.Now()); err != nil {
			t.Fatalf("MarkSyncOutboxEventSent: %v", err)
		}

		if err := s.MarkSyncOutboxEventRetry(store.RetryOutboxInput{
			EventID:     "evt-2",
			RetryCount:  1,
			NextRetryAt: time.Now().Add(2 * time.Minute),
			LastError:   "temporary",
			Status:      store.OutboxStatusPending,
		}); err != nil {
			t.Fatalf("MarkSyncOutboxEventRetry: %v", err)
		}

		if err := s.MarkSyncOutboxEventDead("evt-2", "too many retries"); err != nil {
			t.Fatalf("MarkSyncOutboxEventDead: %v", err)
		}

		claimed, err := s.ClaimPendingSyncOutboxEvents(store.ClaimOutboxOptions{
			Limit: 10,
			Now:   time.Now().Add(10 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ClaimPendingSyncOutboxEvents final: %v", err)
		}
		for _, e := range claimed {
			if e.EventID == "evt-1" || e.EventID == "evt-2" {
				t.Fatalf("sent/dead events should not be claimed again: %s", e.EventID)
			}
		}
	})
}
