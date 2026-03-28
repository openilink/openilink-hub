package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
)

// fakeClock is a deterministic clock for testing.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time  { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func openTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func createTestBot(t *testing.T, db *sqlite.DB) *store.Bot {
	t.Helper()
	u, err := db.CreateUser("reminder-user", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := db.CreateBot(u.ID, "test-bot", "mock", "mock-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGetBotsNeedingReminder_NoReminder(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	// Bot has no reminder_hours set — should not appear.
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots, got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_NotYetDue(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	// Set reminder to 23 hours and send a message now.
	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance only 22 hours — not yet due.
	clock.Advance(22 * time.Hour)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (not yet due), got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_Due(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance 23 hours + 1 minute — should be due.
	clock.Advance(23*time.Hour + time.Minute)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 1 {
		t.Fatalf("expected 1 bot due for reminder, got %d", len(bots))
	}
	if bots[0].ID != b.ID {
		t.Errorf("expected bot %s, got %s", b.ID, bots[0].ID)
	}
}

func TestGetBotsNeedingReminder_AlreadyRemindedRecently(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance 23.5 hours — due.
	clock.Advance(23*time.Hour + 30*time.Minute)

	// Mark as reminded.
	if err := db.MarkBotReminded(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance only 30 minutes — should NOT be due again (< 1 hour since last remind).
	clock.Advance(30 * time.Minute)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (reminded < 1h ago), got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_DueAgainAfterCooldown(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	if err := db.UpdateBotReminder(b.ID, 22); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance 22.5 hours — due.
	clock.Advance(22*time.Hour + 30*time.Minute)
	if err := db.MarkBotReminded(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance 1 hour + 1 minute past last reminder — should be due again.
	clock.Advance(1*time.Hour + time.Minute)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 1 {
		t.Errorf("expected 1 bot (cooldown passed), got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_MessageResetsTimer(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance 22 hours — not yet due.
	clock.Advance(22 * time.Hour)

	// New message comes in — resets last_msg_at.
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Advance another 22 hours (44 from original, but only 22 from last msg).
	clock.Advance(22 * time.Hour)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (msg reset timer), got %d", len(bots))
	}

	// Advance 2 more hours (24 from last msg, > 23 threshold) — should be due.
	clock.Advance(2 * time.Hour)
	bots, err = db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 1 {
		t.Errorf("expected 1 bot after timer reset expired, got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_ReminderDisabledAfterSet(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	// Enable reminder, send a message.
	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Disable reminder.
	if err := db.UpdateBotReminder(b.ID, 0); err != nil {
		t.Fatal(err)
	}

	// Advance 24 hours — would be due, but reminder is disabled.
	clock.Advance(24 * time.Hour)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (reminder disabled), got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_NoMessages(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	// Enable reminder but never send a message (last_msg_at IS NULL).
	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}

	clock.Advance(24 * time.Hour)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (no messages), got %d", len(bots))
	}
}

func TestGetBotsNeedingReminder_DisconnectedBotIgnored(t *testing.T) {
	db := openTestDB(t)
	clock := &fakeClock{t: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)}
	db.SetClock(clock)

	b := createTestBot(t, db)

	if err := db.UpdateBotReminder(b.ID, 23); err != nil {
		t.Fatal(err)
	}
	if err := db.IncrBotMsgCount(b.ID); err != nil {
		t.Fatal(err)
	}

	// Disconnect bot.
	if err := db.UpdateBotStatus(b.ID, "disconnected"); err != nil {
		t.Fatal(err)
	}

	// Advance 24 hours — would be due, but bot is disconnected.
	clock.Advance(24 * time.Hour)
	bots, err := db.GetBotsNeedingReminder()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 0 {
		t.Errorf("expected 0 bots (disconnected), got %d", len(bots))
	}
}
