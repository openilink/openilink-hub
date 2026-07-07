package storetest

import (
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestPromptProfileLifecycle(t *testing.T, s store.Store) {
	t.Run("UpsertAndGetActive", func(t *testing.T) {
		p, changed, err := s.UpsertPromptProfile(store.PromptProfileUpsertInput{
			BotID:           "bot-1",
			SenderUserID:    "wx-user-1",
			BindingID:       "binding-1",
			SystemPrompt:    "system",
			UserPrompt:      "user",
			FullPrompt:      "system\\nuser",
			PromptVersion:   1,
			SourceUpdatedAt: 100,
			Status:          store.PromptProfileStatusActive,
		})
		if err != nil {
			t.Fatalf("UpsertPromptProfile: %v", err)
		}
		if !changed {
			t.Fatal("expected changed=true")
		}
		if p == nil || p.FullPromptHash == "" {
			t.Fatal("expected stored profile with full_prompt_hash")
		}

		active, err := s.GetActivePromptProfile("bot-1", "wx-user-1")
		if err != nil {
			t.Fatalf("GetActivePromptProfile: %v", err)
		}
		if active == nil || active.BindingID != "binding-1" {
			t.Fatalf("unexpected active profile: %#v", active)
		}
	})

	t.Run("VersionProtection", func(t *testing.T) {
		_, changed, err := s.UpsertPromptProfile(store.PromptProfileUpsertInput{
			BotID:           "bot-1",
			SenderUserID:    "wx-user-1",
			BindingID:       "binding-1",
			FullPrompt:      "old",
			PromptVersion:   1,
			SourceUpdatedAt: 99,
			Status:          store.PromptProfileStatusActive,
		})
		if err != nil {
			t.Fatalf("upsert old event: %v", err)
		}
		if changed {
			t.Fatal("expected changed=false for stale event")
		}

		p, err := s.GetPromptProfile("bot-1", "wx-user-1", "binding-1")
		if err != nil {
			t.Fatalf("GetPromptProfile: %v", err)
		}
		if p == nil || p.FullPrompt != "system\\nuser" {
			t.Fatalf("stale event should not overwrite: %#v", p)
		}
	})

	t.Run("SameVersionNewerTimestamp", func(t *testing.T) {
		p, changed, err := s.UpsertPromptProfile(store.PromptProfileUpsertInput{
			BotID:           "bot-1",
			SenderUserID:    "wx-user-1",
			BindingID:       "binding-1",
			FullPrompt:      "updated",
			PromptVersion:   1,
			SourceUpdatedAt: 101,
			Status:          store.PromptProfileStatusActive,
		})
		if err != nil {
			t.Fatalf("upsert newer timestamp: %v", err)
		}
		if !changed {
			t.Fatal("expected changed=true")
		}
		if p.FullPrompt != "updated" {
			t.Fatalf("expected updated full prompt, got %q", p.FullPrompt)
		}
	})

	t.Run("RebindSwitchesActive", func(t *testing.T) {
		p, changed, err := s.UpsertPromptProfile(store.PromptProfileUpsertInput{
			BotID:           "bot-1",
			SenderUserID:    "wx-user-1",
			BindingID:       "binding-2",
			FullPrompt:      "new binding",
			PromptVersion:   2,
			SourceUpdatedAt: 200,
			Status:          store.PromptProfileStatusActive,
		})
		if err != nil {
			t.Fatalf("upsert rebind: %v", err)
		}
		if !changed || p == nil {
			t.Fatal("expected changed rebind profile")
		}

		active, err := s.GetActivePromptProfile("bot-1", "wx-user-1")
		if err != nil {
			t.Fatalf("GetActivePromptProfile: %v", err)
		}
		if active == nil || active.BindingID != "binding-2" {
			t.Fatalf("expected binding-2 active, got %#v", active)
		}

		oldProfile, err := s.GetPromptProfile("bot-1", "wx-user-1", "binding-1")
		if err != nil {
			t.Fatalf("GetPromptProfile old: %v", err)
		}
		if oldProfile == nil || oldProfile.Status != store.PromptProfileStatusInactive {
			t.Fatalf("expected old profile inactive, got %#v", oldProfile)
		}
	})

	t.Run("Invalidate", func(t *testing.T) {
		ok, err := s.InvalidatePromptProfile("bot-1", "wx-user-1", "binding-2")
		if err != nil {
			t.Fatalf("InvalidatePromptProfile: %v", err)
		}
		if !ok {
			t.Fatal("expected invalidate true")
		}

		active, err := s.GetActivePromptProfile("bot-1", "wx-user-1")
		if err != nil {
			t.Fatalf("GetActivePromptProfile after invalidate: %v", err)
		}
		if active != nil {
			t.Fatalf("expected no active profile, got %#v", active)
		}
	})
}
