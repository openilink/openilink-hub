package sync

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

type fakeStore struct {
	mu      sync.Mutex
	events  []store.SyncOutboxEvent
	sent    map[string]bool
	dead    map[string]bool
	retries map[string]int
}

func (f *fakeStore) ClaimPendingSyncOutboxEvents(opt store.ClaimOutboxOptions) ([]store.SyncOutboxEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []store.SyncOutboxEvent
	for i := range f.events {
		e := &f.events[i]
		if e.Status == store.OutboxStatusPending {
			e.Status = store.OutboxStatusProcessing
			out = append(out, *e)
		}
	}
	return out, nil
}
func (f *fakeStore) MarkSyncOutboxEventSent(eventID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sent == nil {
		f.sent = map[string]bool{}
	}
	f.sent[eventID] = true
	return nil
}
func (f *fakeStore) MarkSyncOutboxEventRetry(in store.RetryOutboxInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.retries == nil {
		f.retries = map[string]int{}
	}
	f.retries[in.EventID] = in.RetryCount
	for i := range f.events {
		if f.events[i].EventID == in.EventID {
			f.events[i].Status = store.OutboxStatusPending
			f.events[i].RetryCount = in.RetryCount
		}
	}
	return nil
}
func (f *fakeStore) MarkSyncOutboxEventDead(eventID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dead == nil {
		f.dead = map[string]bool{}
	}
	f.dead[eventID] = true
	return nil
}

type fakeClient struct {
	messageErr bool
	calls      map[string]int
}

func (f *fakeClient) UpsertMessageEvent(context.Context, json.RawMessage) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls["message"]++
	if f.messageErr {
		return errors.New("temporary")
	}
	return nil
}
func (f *fakeClient) UpsertPromptProfileEvent(context.Context, json.RawMessage) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls["prompt"]++
	return nil
}
func (f *fakeClient) UpsertBindingInvalidatedEvent(context.Context, json.RawMessage) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls["invalidate"]++
	return nil
}

func TestOutboxWorker_SentAndRetryAndDead(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"id": 1})
	fs := &fakeStore{events: []store.SyncOutboxEvent{{
		EventID: "e1", EventType: store.OutboxEventMessageInbound, Status: store.OutboxStatusPending, Payload: payload,
	}, {
		EventID: "e2", EventType: store.OutboxEventPromptProfileChange, Status: store.OutboxStatusPending, Payload: payload,
	}}}
	client := &fakeClient{}
	w := NewOutboxWorker(fs, client, OutboxWorkerConfig{BatchSize: 10, MaxRetries: 2})

	if err := w.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce: %v", err)
	}
	if !fs.sent["e1"] || !fs.sent["e2"] {
		t.Fatalf("expected sent marks: %#v", fs.sent)
	}

	fs2 := &fakeStore{events: []store.SyncOutboxEvent{{
		EventID: "r1", EventType: store.OutboxEventMessageInbound, Status: store.OutboxStatusPending, Payload: payload,
	}}}
	client2 := &fakeClient{messageErr: true}
	w2 := NewOutboxWorker(fs2, client2, OutboxWorkerConfig{BatchSize: 10, MaxRetries: 2})
	_ = w2.runOnce(context.Background())
	if fs2.retries["r1"] != 1 {
		t.Fatalf("expected retry count 1, got %#v", fs2.retries)
	}
	_ = w2.runOnce(context.Background())
	if !fs2.dead["r1"] {
		t.Fatalf("expected dead letter after max retries, got %#v", fs2.dead)
	}
}
