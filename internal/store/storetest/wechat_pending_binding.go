package storetest

import (
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestWechatPendingBindingLifecycle(t *testing.T, s store.Store) {
	now := time.Now()
	t.Run("create_and_dedup", func(t *testing.T) {
		row, created, err := s.CreateWechatPendingBinding(store.WechatPendingBindingCreateInput{
			EventID:       "prebind-evt-1",
			ProviderBotID: "provider-bot-1",
			BotID:         "bot-1",
			BindingID:     "binding-1",
			RoleID:        "101",
			SessionID:     "session-1",
			Status:        store.WechatPendingBindingStatusPending,
			ExpiresAt:     now.Add(10 * time.Minute),
		})
		if err != nil {
			t.Fatalf("CreateWechatPendingBinding: %v", err)
		}
		if !created {
			t.Fatalf("expected created=true")
		}
		if row == nil || row.ID == 0 {
			t.Fatalf("expected non-nil row with id")
		}

		row2, created2, err := s.CreateWechatPendingBinding(store.WechatPendingBindingCreateInput{
			EventID:       "prebind-evt-1",
			ProviderBotID: "provider-bot-1",
			BotID:         "bot-1",
			BindingID:     "binding-1",
			RoleID:        "101",
			SessionID:     "session-1",
			Status:        store.WechatPendingBindingStatusPending,
			ExpiresAt:     now.Add(10 * time.Minute),
		})
		if err != nil {
			t.Fatalf("CreateWechatPendingBinding dedup: %v", err)
		}
		if created2 {
			t.Fatalf("expected created=false on duplicate event id")
		}
		if row2 == nil || row2.ID != row.ID {
			t.Fatalf("expected duplicate to return same row")
		}
	})

	t.Run("lookup_and_finalize", func(t *testing.T) {
		found, err := s.GetLatestPendingWechatBinding("bot-1", "provider-bot-1", now)
		if err != nil {
			t.Fatalf("GetLatestPendingWechatBinding: %v", err)
		}
		if found == nil {
			t.Fatalf("expected pending binding")
		}
		ok, err := s.FinalizeWechatPendingBinding(found.ID, "ctx-1", "finalize-evt-1", now)
		if err != nil {
			t.Fatalf("FinalizeWechatPendingBinding: %v", err)
		}
		if !ok {
			t.Fatalf("expected finalize ok=true")
		}

		found2, err := s.GetLatestPendingWechatBinding("bot-1", "provider-bot-1", now)
		if err != nil {
			t.Fatalf("GetLatestPendingWechatBinding after finalize: %v", err)
		}
		if found2 != nil {
			t.Fatalf("expected no pending after finalize")
		}
	})
}

