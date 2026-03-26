// Package storetest provides a shared integration test suite for store.Store
// implementations. Each exported Test* function is self-contained and exercises
// one domain (users, bots, messages, etc.) comprehensively.
package storetest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

// RunAll executes every domain test against the given store.
func RunAll(t *testing.T, s store.Store) {
	t.Run("User", func(t *testing.T) { TestUserCRUD(t, s) })
	t.Run("Bot", func(t *testing.T) { TestBotCRUD(t, s) })
	t.Run("Message", func(t *testing.T) { TestMessageCRUD(t, s) })
	t.Run("Channel", func(t *testing.T) { TestChannelCRUD(t, s) })
	t.Run("App", func(t *testing.T) { TestAppCRUD(t, s) })
	t.Run("Plugin", func(t *testing.T) { TestPluginCRUD(t, s) })
	t.Run("Trace", func(t *testing.T) { TestTraceCRUD(t, s) })
	t.Run("Credential", func(t *testing.T) { TestCredentialCRUD(t, s) })
	t.Run("OAuth", func(t *testing.T) { TestOAuthCRUD(t, s) })
	t.Run("Config", func(t *testing.T) { TestConfigCRUD(t, s) })
	t.Run("WebhookLog", func(t *testing.T) { TestWebhookLogCRUD(t, s) })
	t.Run("AppLog", func(t *testing.T) { TestAppLogCRUD(t, s) })
	t.Run("Session", func(t *testing.T) { TestSessionCRUD(t, s) })
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustCreateUser(t *testing.T, s store.Store, username, displayName string) *store.User {
	t.Helper()
	u, err := s.CreateUser(username, displayName)
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", username, err)
	}
	return u
}

func mustCreateBot(t *testing.T, s store.Store, userID, name string) *store.Bot {
	t.Helper()
	b, err := s.CreateBot(userID, name, "test", "", json.RawMessage(`{"token":"abc"}`))
	if err != nil {
		t.Fatalf("CreateBot(%q): %v", name, err)
	}
	return b
}

func mustCreateChannel(t *testing.T, s store.Store, botID, name, handle string) *store.Channel {
	t.Helper()
	ch, err := s.CreateChannel(botID, name, handle, nil, nil)
	if err != nil {
		t.Fatalf("CreateChannel(%q): %v", name, err)
	}
	return ch
}

func mustCreateApp(t *testing.T, s store.Store, ownerID, name, slug string) *store.App {
	t.Helper()
	app, err := s.CreateApp(&store.App{
		OwnerID:     ownerID,
		Name:        name,
		Slug:        slug,
		Description: "test app",
	})
	if err != nil {
		t.Fatalf("CreateApp(%q): %v", name, err)
	}
	return app
}

func int64Ptr(v int64) *int64 { return &v }

// ---------------------------------------------------------------------------
// TestUserCRUD
// ---------------------------------------------------------------------------

func TestUserCRUD(t *testing.T, s store.Store) {
	t.Run("FirstUserIsSuperAdmin", func(t *testing.T) {
		u, err := s.CreateUser("first_user", "First")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if u.Role != store.RoleSuperAdmin {
			t.Errorf("first user role = %q, want %q", u.Role, store.RoleSuperAdmin)
		}
		// Verify via GetUserByID
		got, err := s.GetUserByID(u.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if got.Role != store.RoleSuperAdmin {
			t.Errorf("first user role (from DB) = %q, want %q", got.Role, store.RoleSuperAdmin)
		}
	})

	t.Run("SecondUserIsMember", func(t *testing.T) {
		u, err := s.CreateUser("second_user", "Second")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if u.Role != store.RoleMember {
			t.Errorf("second user role = %q, want %q", u.Role, store.RoleMember)
		}
	})

	t.Run("CreateUserFull", func(t *testing.T) {
		u, err := s.CreateUserFull("fulluser", "full@test.com", "Full User", "hash123", store.RoleAdmin)
		if err != nil {
			t.Fatalf("CreateUserFull: %v", err)
		}
		if u.Email != "full@test.com" {
			t.Errorf("email = %q, want %q", u.Email, "full@test.com")
		}
		if u.Role != store.RoleAdmin {
			t.Errorf("role = %q, want %q", u.Role, store.RoleAdmin)
		}
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		u, err := s.GetUserByUsername("fulluser")
		if err != nil {
			t.Fatalf("GetUserByUsername: %v", err)
		}
		if u.Username != "fulluser" {
			t.Errorf("username = %q, want %q", u.Username, "fulluser")
		}
	})

	t.Run("GetUserByEmail", func(t *testing.T) {
		u, err := s.GetUserByEmail("full@test.com")
		if err != nil {
			t.Fatalf("GetUserByEmail: %v", err)
		}
		if u.Email != "full@test.com" {
			t.Errorf("email = %q, want %q", u.Email, "full@test.com")
		}
	})

	t.Run("ListUsers", func(t *testing.T) {
		users, err := s.ListUsers()
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if len(users) < 3 {
			t.Fatalf("expected >= 3 users, got %d", len(users))
		}
		// Verify ordering by created_at (ascending)
		for i := 1; i < len(users); i++ {
			if users[i].CreatedAt < users[i-1].CreatedAt {
				t.Error("users not ordered by created_at ascending")
				break
			}
		}
	})

	t.Run("UserCount", func(t *testing.T) {
		count, err := s.UserCount()
		if err != nil {
			t.Fatalf("UserCount: %v", err)
		}
		if count < 3 {
			t.Errorf("count = %d, want >= 3", count)
		}
	})

	t.Run("UpdateUserProfile", func(t *testing.T) {
		u, _ := s.GetUserByUsername("fulluser")
		if err := s.UpdateUserProfile(u.ID, "Updated Name", "new@test.com"); err != nil {
			t.Fatalf("UpdateUserProfile: %v", err)
		}
		got, _ := s.GetUserByID(u.ID)
		if got.DisplayName != "Updated Name" {
			t.Errorf("display_name = %q, want %q", got.DisplayName, "Updated Name")
		}
		if got.Email != "new@test.com" {
			t.Errorf("email = %q, want %q", got.Email, "new@test.com")
		}
	})

	t.Run("UpdateUserPassword", func(t *testing.T) {
		u, _ := s.GetUserByUsername("fulluser")
		if err := s.UpdateUserPassword(u.ID, "newhash"); err != nil {
			t.Fatalf("UpdateUserPassword: %v", err)
		}
		got, _ := s.GetUserByID(u.ID)
		if got.PasswordHash != "newhash" {
			t.Errorf("password_hash = %q, want %q", got.PasswordHash, "newhash")
		}
	})

	t.Run("UpdateUserRole", func(t *testing.T) {
		u, _ := s.GetUserByUsername("second_user")
		if err := s.UpdateUserRole(u.ID, store.RoleAdmin); err != nil {
			t.Fatalf("UpdateUserRole: %v", err)
		}
		got, _ := s.GetUserByID(u.ID)
		if got.Role != store.RoleAdmin {
			t.Errorf("role = %q, want %q", got.Role, store.RoleAdmin)
		}
	})

	t.Run("UpdateUserStatus", func(t *testing.T) {
		u, _ := s.GetUserByUsername("second_user")
		if err := s.UpdateUserStatus(u.ID, store.StatusDisabled); err != nil {
			t.Fatalf("UpdateUserStatus: %v", err)
		}
		got, _ := s.GetUserByID(u.ID)
		if got.Status != store.StatusDisabled {
			t.Errorf("status = %q, want %q", got.Status, store.StatusDisabled)
		}
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		_, err := s.GetUserByID("nonexistent-user-id")
		if err == nil {
			t.Error("expected error for non-existent user ID, got nil")
		}
	})

	t.Run("GetUserByUsername_NotFound", func(t *testing.T) {
		_, err := s.GetUserByUsername("nosuchusername")
		if err == nil {
			t.Error("expected error for non-existent username, got nil")
		}
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		_, err := s.GetUserByEmail("nosuch@email.com")
		if err == nil {
			t.Error("expected error for non-existent email, got nil")
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		u, _ := s.GetUserByUsername("second_user")
		if err := s.DeleteUser(u.ID); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		_, err := s.GetUserByID(u.ID)
		if err == nil {
			t.Error("expected error after deleting user, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// TestBotCRUD
// ---------------------------------------------------------------------------

func TestBotCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "botowner", "Bot Owner")

	t.Run("CreateAndGet", func(t *testing.T) {
		b := mustCreateBot(t, s, u.ID, "MyBot")
		got, err := s.GetBot(b.ID)
		if err != nil {
			t.Fatalf("GetBot: %v", err)
		}
		if got.Name != "MyBot" {
			t.Errorf("name = %q, want %q", got.Name, "MyBot")
		}
		if got.Status != "connected" {
			t.Errorf("status = %q, want %q", got.Status, "connected")
		}
		if got.UserID != u.ID {
			t.Errorf("user_id = %q, want %q", got.UserID, u.ID)
		}
	})

	t.Run("ListBotsByUser", func(t *testing.T) {
		bots, err := s.ListBotsByUser(u.ID)
		if err != nil {
			t.Fatalf("ListBotsByUser: %v", err)
		}
		if len(bots) < 1 {
			t.Fatal("expected at least 1 bot")
		}
	})

	t.Run("GetAllBots", func(t *testing.T) {
		bots, err := s.GetAllBots()
		if err != nil {
			t.Fatalf("GetAllBots: %v", err)
		}
		if len(bots) < 1 {
			t.Fatal("expected at least 1 bot")
		}
	})

	t.Run("FindBotByProviderID", func(t *testing.T) {
		b, err := s.CreateBot(u.ID, "ProvBot", "wechat", "wx_123", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("CreateBot: %v", err)
		}
		got, err := s.FindBotByProviderID("wechat", "wx_123")
		if err != nil {
			t.Fatalf("FindBotByProviderID: %v", err)
		}
		if got.ID != b.ID {
			t.Errorf("id = %q, want %q", got.ID, b.ID)
		}
	})

	t.Run("FindBotByCredential", func(t *testing.T) {
		creds := json.RawMessage(`{"api_key":"secret_123"}`)
		b, err := s.CreateBot(u.ID, "CredBot", "test", "", creds)
		if err != nil {
			t.Fatalf("CreateBot: %v", err)
		}
		got, err := s.FindBotByCredential("api_key", "secret_123")
		if err != nil {
			t.Fatalf("FindBotByCredential: %v", err)
		}
		if got.ID != b.ID {
			t.Errorf("id = %q, want %q", got.ID, b.ID)
		}
	})

	t.Run("UpdateBotCredentials", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		newCreds := json.RawMessage(`{"token":"new_token"}`)
		if err := s.UpdateBotCredentials(b.ID, "new_pid", newCreds); err != nil {
			t.Fatalf("UpdateBotCredentials: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		if got.ProviderID != "new_pid" {
			t.Errorf("provider_id = %q, want %q", got.ProviderID, "new_pid")
		}
	})

	t.Run("UpdateBotName", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		if err := s.UpdateBotName(b.ID, "Renamed"); err != nil {
			t.Fatalf("UpdateBotName: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		if got.Name != "Renamed" {
			t.Errorf("name = %q, want %q", got.Name, "Renamed")
		}
	})

	t.Run("UpdateBotStatus", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		if err := s.UpdateBotStatus(b.ID, "session_expired"); err != nil {
			t.Fatalf("UpdateBotStatus: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		if got.Status != "session_expired" {
			t.Errorf("status = %q, want %q", got.Status, "session_expired")
		}
		// Reset to connected for downstream tests
		s.UpdateBotStatus(b.ID, "connected")
	})

	t.Run("UpdateBotSyncState", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		state := json.RawMessage(`{"cursor":"abc"}`)
		if err := s.UpdateBotSyncState(b.ID, state); err != nil {
			t.Fatalf("UpdateBotSyncState: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		var m map[string]string
		json.Unmarshal(got.SyncState, &m)
		if m["cursor"] != "abc" {
			t.Errorf("sync_state cursor = %q, want %q", m["cursor"], "abc")
		}
	})

	t.Run("IncrBotMsgCount", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		before, _ := s.GetBot(b.ID)
		if err := s.IncrBotMsgCount(b.ID); err != nil {
			t.Fatalf("IncrBotMsgCount: %v", err)
		}
		after, _ := s.GetBot(b.ID)
		if after.MsgCount != before.MsgCount+1 {
			t.Errorf("msg_count = %d, want %d", after.MsgCount, before.MsgCount+1)
		}
	})

	t.Run("UpdateBotReminder", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		if err := s.UpdateBotReminder(b.ID, 24); err != nil {
			t.Fatalf("UpdateBotReminder: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		if got.ReminderHours != 24 {
			t.Errorf("reminder_hours = %d, want 24", got.ReminderHours)
		}
	})

	t.Run("MarkBotReminded", func(t *testing.T) {
		bots, _ := s.ListBotsByUser(u.ID)
		b := bots[0]
		if err := s.MarkBotReminded(b.ID); err != nil {
			t.Fatalf("MarkBotReminded: %v", err)
		}
		got, _ := s.GetBot(b.ID)
		if got.LastRemindedAt == nil {
			t.Error("last_reminded_at should not be nil after MarkBotReminded")
		}
	})

	t.Run("GetBotsNeedingReminder", func(t *testing.T) {
		// Just verify no error; time-based logic is hard to test without clock control
		_, err := s.GetBotsNeedingReminder()
		if err != nil {
			t.Fatalf("GetBotsNeedingReminder: %v", err)
		}
	})

	t.Run("CountBotsByUser", func(t *testing.T) {
		count, err := s.CountBotsByUser(u.ID)
		if err != nil {
			t.Fatalf("CountBotsByUser: %v", err)
		}
		if count < 1 {
			t.Errorf("count = %d, want >= 1", count)
		}
	})

	t.Run("GetBotStats", func(t *testing.T) {
		stats, err := s.GetBotStats(u.ID)
		if err != nil {
			t.Fatalf("GetBotStats: %v", err)
		}
		if stats.TotalBots < 1 {
			t.Errorf("total_bots = %d, want >= 1", stats.TotalBots)
		}
	})

	t.Run("GetAdminStats", func(t *testing.T) {
		stats, err := s.GetAdminStats()
		if err != nil {
			t.Fatalf("GetAdminStats: %v", err)
		}
		if stats.TotalBots < 1 {
			t.Errorf("total_bots = %d, want >= 1", stats.TotalBots)
		}
		if stats.TotalUsers < 1 {
			t.Errorf("total_users = %d, want >= 1", stats.TotalUsers)
		}
	})

	t.Run("ListRecentContacts", func(t *testing.T) {
		// Just verify no error (need messages to have real data)
		bots, _ := s.ListBotsByUser(u.ID)
		_, err := s.ListRecentContacts(bots[0].ID, 10)
		if err != nil {
			t.Fatalf("ListRecentContacts: %v", err)
		}
	})

	t.Run("LastActivityAt", func(t *testing.T) {
		// May be nil if no messages yet
		_ = s.LastActivityAt(u.ID)
	})

	t.Run("GetBot_NotFound", func(t *testing.T) {
		_, err := s.GetBot("nonexistent-bot-id")
		if err == nil {
			t.Error("expected error for non-existent bot, got nil")
		}
	})

	t.Run("FindBotByProviderID_NotFound", func(t *testing.T) {
		_, err := s.FindBotByProviderID("nosuchprovider", "nosuchid")
		if err == nil {
			t.Error("expected error for non-existent provider ID, got nil")
		}
	})

	t.Run("FindBotByCredential_NotFound", func(t *testing.T) {
		_, err := s.FindBotByCredential("nosuchkey", "nosuchvalue")
		if err == nil {
			t.Error("expected error for non-existent credential, got nil")
		}
	})

	t.Run("DeleteBot", func(t *testing.T) {
		b := mustCreateBot(t, s, u.ID, "ToDelete")
		if err := s.DeleteBot(b.ID); err != nil {
			t.Fatalf("DeleteBot: %v", err)
		}
		_, err := s.GetBot(b.ID)
		if err == nil {
			t.Error("expected error after deleting bot, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// TestMessageCRUD
// ---------------------------------------------------------------------------

func TestMessageCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "msgowner", "Msg Owner")
	b := mustCreateBot(t, s, u.ID, "MsgBot")
	ch := mustCreateChannel(t, s, b.ID, "MsgChan", "msgchan")

	var savedID int64

	t.Run("SaveMessage_Insert", func(t *testing.T) {
		msgID := int64(1001)
		m := &store.Message{
			BotID:        b.ID,
			ChannelID:    &ch.ID,
			Direction:    "inbound",
			MessageID:    &msgID,
			FromUserID:   "sender1",
			ToUserID:     "recv1",
			MessageType:  1,
			ContextToken: "ctx_token_1",
			ItemList:     json.RawMessage(`[{"type":"text"}]`),
		}
		res, err := s.SaveMessage(m)
		if err != nil {
			t.Fatalf("SaveMessage (insert): %v", err)
		}
		if !res.Inserted {
			t.Error("expected Inserted=true for new message")
		}
		if res.ID == 0 {
			t.Error("expected non-zero ID")
		}
		savedID = res.ID
	})

	t.Run("SaveMessage_Upsert", func(t *testing.T) {
		msgID := int64(1001)
		m := &store.Message{
			BotID:        b.ID,
			Direction:    "inbound",
			MessageID:    &msgID,
			FromUserID:   "sender1",
			MessageState: 2,
			ContextToken: "ctx_token_1",
			ItemList:     json.RawMessage(`[{"type":"text","content":"updated"}]`),
		}
		res, err := s.SaveMessage(m)
		if err != nil {
			t.Fatalf("SaveMessage (upsert): %v", err)
		}
		if res.Inserted {
			t.Error("expected Inserted=false for upsert")
		}
		if res.ID != savedID {
			t.Errorf("upsert ID = %d, want %d", res.ID, savedID)
		}
	})

	t.Run("GetMessage", func(t *testing.T) {
		got, err := s.GetMessage(savedID)
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		if got.BotID != b.ID {
			t.Errorf("bot_id = %q, want %q", got.BotID, b.ID)
		}
	})

	t.Run("ListMessages", func(t *testing.T) {
		// Insert a few more messages
		for i := 0; i < 3; i++ {
			msgID := int64(2000 + i)
			s.SaveMessage(&store.Message{
				BotID:      b.ID,
				Direction:  "inbound",
				MessageID:  &msgID,
				FromUserID: "sender1",
			})
		}
		msgs, err := s.ListMessages(b.ID, 10, 0)
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		if len(msgs) < 4 {
			t.Errorf("expected >= 4 messages, got %d", len(msgs))
		}
		// Verify DESC order
		for i := 1; i < len(msgs); i++ {
			if msgs[i].ID > msgs[i-1].ID {
				t.Error("messages not in DESC order")
				break
			}
		}
	})

	t.Run("ListMessages_BeforeID", func(t *testing.T) {
		all, _ := s.ListMessages(b.ID, 100, 0)
		if len(all) < 2 {
			t.Skip("not enough messages")
		}
		msgs, err := s.ListMessages(b.ID, 10, all[0].ID)
		if err != nil {
			t.Fatalf("ListMessages(beforeID): %v", err)
		}
		for _, m := range msgs {
			if m.ID >= all[0].ID {
				t.Errorf("message id %d should be < %d", m.ID, all[0].ID)
			}
		}
	})

	t.Run("ListMessagesBySender", func(t *testing.T) {
		msgs, err := s.ListMessagesBySender(b.ID, "sender1", 10)
		if err != nil {
			t.Fatalf("ListMessagesBySender: %v", err)
		}
		if len(msgs) < 1 {
			t.Error("expected at least 1 message from sender1")
		}
	})

	t.Run("ListChannelMessages", func(t *testing.T) {
		msgs, err := s.ListChannelMessages(ch.ID, "sender1", 10)
		if err != nil {
			t.Fatalf("ListChannelMessages: %v", err)
		}
		// sender1 should appear since we saved a message with channel_id set
		if len(msgs) < 1 {
			t.Error("expected at least 1 channel message")
		}
	})

	t.Run("GetMessagesSince", func(t *testing.T) {
		msgs, err := s.GetMessagesSince(b.ID, 0, 100)
		if err != nil {
			t.Fatalf("GetMessagesSince: %v", err)
		}
		if len(msgs) < 1 {
			t.Error("expected at least 1 message since id 0")
		}
		// Verify ASC order
		for i := 1; i < len(msgs); i++ {
			if msgs[i].ID < msgs[i-1].ID {
				t.Error("GetMessagesSince not in ASC order")
				break
			}
		}
	})

	t.Run("GetLatestContextToken", func(t *testing.T) {
		token := s.GetLatestContextToken(b.ID)
		if token != "ctx_token_1" {
			t.Errorf("context_token = %q, want %q", token, "ctx_token_1")
		}
	})

	t.Run("HasFreshContextToken", func(t *testing.T) {
		has := s.HasFreshContextToken(b.ID, 1*time.Hour)
		if !has {
			t.Error("expected HasFreshContextToken to be true for recently saved message")
		}
	})

	t.Run("BatchHasFreshContextToken", func(t *testing.T) {
		result := s.BatchHasFreshContextToken([]string{b.ID, "nonexistent"}, 1*time.Hour)
		if !result[b.ID] {
			t.Errorf("expected bot %q to have fresh context token", b.ID)
		}
		if result["nonexistent"] {
			t.Error("nonexistent bot should not have fresh context token")
		}
	})

	t.Run("UpdateMediaStatus", func(t *testing.T) {
		// Insert a message with media_status='downloading'
		msgID := int64(3001)
		s.SaveMessage(&store.Message{
			BotID:       b.ID,
			Direction:   "inbound",
			MessageID:   &msgID,
			FromUserID:  "sender1",
			MediaStatus: "downloading",
		})
		err := s.UpdateMediaStatus(b.ID, "ready", json.RawMessage(`{"key":"val"}`))
		if err != nil {
			t.Fatalf("UpdateMediaStatus: %v", err)
		}
	})

	t.Run("UpdateMediaStatusByID", func(t *testing.T) {
		msgID := int64(3002)
		res, _ := s.SaveMessage(&store.Message{
			BotID:       b.ID,
			Direction:   "inbound",
			MessageID:   &msgID,
			FromUserID:  "sender1",
			MediaStatus: "downloading",
		})
		err := s.UpdateMediaStatusByID(res.ID, "ready", json.RawMessage(`{"k":"v"}`))
		if err != nil {
			t.Fatalf("UpdateMediaStatusByID: %v", err)
		}
	})

	t.Run("MarkProcessed_GetUnprocessed", func(t *testing.T) {
		msgID := int64(4001)
		res, _ := s.SaveMessage(&store.Message{
			BotID:      b.ID,
			Direction:  "inbound",
			MessageID:  &msgID,
			FromUserID: "sender1",
		})
		unproc, err := s.GetUnprocessedMessages(b.ID, 100)
		if err != nil {
			t.Fatalf("GetUnprocessedMessages: %v", err)
		}
		found := false
		for _, m := range unproc {
			if m.ID == res.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("newly saved inbound message should be unprocessed")
		}

		if err := s.MarkProcessed(res.ID); err != nil {
			t.Fatalf("MarkProcessed: %v", err)
		}
		unproc2, _ := s.GetUnprocessedMessages(b.ID, 100)
		for _, m := range unproc2 {
			if m.ID == res.ID {
				t.Error("message should be processed after MarkProcessed")
			}
		}
	})

	t.Run("UpdateMessagePayload", func(t *testing.T) {
		// Save a message with media_status="downloading"
		msgID := int64(5001)
		res, err := s.SaveMessage(&store.Message{
			BotID:       b.ID,
			Direction:   "inbound",
			MessageID:   &msgID,
			FromUserID:  "sender1",
			MediaStatus: "downloading",
		})
		if err != nil {
			t.Fatalf("SaveMessage: %v", err)
		}
		payload := json.RawMessage(`{"media_status":"ready","media_key":"test.jpg"}`)
		if err := s.UpdateMessagePayload(res.ID, payload); err != nil {
			t.Fatalf("UpdateMessagePayload: %v", err)
		}
		got, err := s.GetMessage(res.ID)
		if err != nil {
			t.Fatalf("GetMessage after UpdateMessagePayload: %v", err)
		}
		if got.MediaStatus != "ready" {
			t.Errorf("media_status = %q, want %q", got.MediaStatus, "ready")
		}
	})

	t.Run("UpdateMediaPayloads", func(t *testing.T) {
		// Save messages with media_status="downloading" for the bot
		for i := 0; i < 2; i++ {
			msgID := int64(6001 + i)
			_, err := s.SaveMessage(&store.Message{
				BotID:       b.ID,
				Direction:   "inbound",
				MessageID:   &msgID,
				FromUserID:  "sender1",
				MediaStatus: "downloading",
			})
			if err != nil {
				t.Fatalf("SaveMessage: %v", err)
			}
		}
		payload := json.RawMessage(`{"media_status":"ready","media_key":"test.jpg"}`)
		if err := s.UpdateMediaPayloads(b.ID, "", payload); err != nil {
			t.Fatalf("UpdateMediaPayloads: %v", err)
		}
		// All downloading messages for this bot should be updated
		msgs, err := s.ListMessages(b.ID, 200, 0)
		if err != nil {
			t.Fatalf("ListMessages: %v", err)
		}
		for _, m := range msgs {
			if m.MediaStatus == "downloading" {
				t.Errorf("message %d still has media_status=downloading after UpdateMediaPayloads", m.ID)
			}
		}
	})

	t.Run("GetMessage_NotFound", func(t *testing.T) {
		_, err := s.GetMessage(999999)
		if err == nil {
			t.Error("expected error for non-existent message, got nil")
		}
	})

	t.Run("BatchHasFreshContextToken_Empty", func(t *testing.T) {
		result := s.BatchHasFreshContextToken([]string{}, 1*time.Hour)
		if result == nil {
			t.Error("expected non-nil empty map, got nil")
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("PruneMessages", func(t *testing.T) {
		// Just verify no error with large maxAge (should delete nothing)
		_, err := s.PruneMessages(9999)
		if err != nil {
			t.Fatalf("PruneMessages: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestChannelCRUD
// ---------------------------------------------------------------------------

func TestChannelCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "chanowner", "Chan Owner")
	b := mustCreateBot(t, s, u.ID, "ChanBot")

	var chID string

	t.Run("CreateAndGet", func(t *testing.T) {
		filter := &store.FilterRule{UserIDs: []string{"u1"}}
		ai := &store.AIConfig{Enabled: true, Model: "gpt-4"}
		ch, err := s.CreateChannel(b.ID, "TestChan", "testchan", filter, ai)
		if err != nil {
			t.Fatalf("CreateChannel: %v", err)
		}
		chID = ch.ID

		got, err := s.GetChannel(ch.ID)
		if err != nil {
			t.Fatalf("GetChannel: %v", err)
		}
		if got.Name != "TestChan" {
			t.Errorf("name = %q, want %q", got.Name, "TestChan")
		}
		if got.Handle != "testchan" {
			t.Errorf("handle = %q, want %q", got.Handle, "testchan")
		}
		if !got.Enabled {
			t.Error("new channel should be enabled")
		}
		if got.APIKey == "" {
			t.Error("api_key should be generated")
		}
		if !got.AIConfig.Enabled {
			t.Error("ai_config.enabled should be true")
		}
		if len(got.FilterRule.UserIDs) != 1 || got.FilterRule.UserIDs[0] != "u1" {
			t.Error("filter_rule.user_ids mismatch")
		}
	})

	t.Run("GetChannelByAPIKey", func(t *testing.T) {
		ch, _ := s.GetChannel(chID)
		got, err := s.GetChannelByAPIKey(ch.APIKey)
		if err != nil {
			t.Fatalf("GetChannelByAPIKey: %v", err)
		}
		if got.ID != chID {
			t.Errorf("id = %q, want %q", got.ID, chID)
		}
	})

	t.Run("ListChannelsByBot", func(t *testing.T) {
		chs, err := s.ListChannelsByBot(b.ID)
		if err != nil {
			t.Fatalf("ListChannelsByBot: %v", err)
		}
		if len(chs) < 1 {
			t.Fatal("expected at least 1 channel")
		}
	})

	t.Run("ListChannelsByBotIDs", func(t *testing.T) {
		chs, err := s.ListChannelsByBotIDs([]string{b.ID})
		if err != nil {
			t.Fatalf("ListChannelsByBotIDs: %v", err)
		}
		if len(chs) < 1 {
			t.Fatal("expected at least 1 channel")
		}
	})

	t.Run("UpdateChannel", func(t *testing.T) {
		err := s.UpdateChannel(chID, "Updated", "updatedhandle",
			&store.FilterRule{Keywords: []string{"hello"}},
			&store.AIConfig{Enabled: false},
			&store.WebhookConfig{URL: "https://example.com/hook"},
			false)
		if err != nil {
			t.Fatalf("UpdateChannel: %v", err)
		}
		got, _ := s.GetChannel(chID)
		if got.Name != "Updated" {
			t.Errorf("name = %q, want %q", got.Name, "Updated")
		}
		if got.Enabled {
			t.Error("expected enabled=false after update")
		}
		if got.WebhookConfig.URL != "https://example.com/hook" {
			t.Errorf("webhook url = %q, want %q", got.WebhookConfig.URL, "https://example.com/hook")
		}
	})

	t.Run("RotateChannelKey", func(t *testing.T) {
		oldCh, _ := s.GetChannel(chID)
		newKey, err := s.RotateChannelKey(chID)
		if err != nil {
			t.Fatalf("RotateChannelKey: %v", err)
		}
		if newKey == oldCh.APIKey {
			t.Error("new key should differ from old key")
		}
		got, _ := s.GetChannelByAPIKey(newKey)
		if got.ID != chID {
			t.Error("new key should resolve to same channel")
		}
	})

	t.Run("UpdateChannelLastSeq", func(t *testing.T) {
		if err := s.UpdateChannelLastSeq(chID, 42); err != nil {
			t.Fatalf("UpdateChannelLastSeq: %v", err)
		}
		got, _ := s.GetChannel(chID)
		if got.LastSeq != 42 {
			t.Errorf("last_seq = %d, want 42", got.LastSeq)
		}
	})

	t.Run("CountChannelsByBot", func(t *testing.T) {
		count, err := s.CountChannelsByBot(b.ID)
		if err != nil {
			t.Fatalf("CountChannelsByBot: %v", err)
		}
		if count < 1 {
			t.Errorf("count = %d, want >= 1", count)
		}
	})

	t.Run("GetChannel_NotFound", func(t *testing.T) {
		_, err := s.GetChannel("nonexistent-channel-id")
		if err == nil {
			t.Error("expected error for non-existent channel, got nil")
		}
	})

	t.Run("ListChannelsByBotIDs_Empty", func(t *testing.T) {
		chs, err := s.ListChannelsByBotIDs([]string{})
		if err != nil {
			t.Fatalf("ListChannelsByBotIDs(empty): %v", err)
		}
		if chs != nil {
			t.Errorf("expected nil for empty botIDs, got %d entries", len(chs))
		}
	})

	t.Run("DeleteChannel", func(t *testing.T) {
		ch := mustCreateChannel(t, s, b.ID, "ToDelete", "todelete")
		if err := s.DeleteChannel(ch.ID); err != nil {
			t.Fatalf("DeleteChannel: %v", err)
		}
		_, err := s.GetChannel(ch.ID)
		if err == nil {
			t.Error("expected error after deleting channel")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAppCRUD
// ---------------------------------------------------------------------------

func TestAppCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "appowner", "App Owner")
	b := mustCreateBot(t, s, u.ID, "AppBot")

	var appID string

	t.Run("CreateAndGet", func(t *testing.T) {
		app := mustCreateApp(t, s, u.ID, "TestApp", "test-app")
		appID = app.ID

		got, err := s.GetApp(app.ID)
		if err != nil {
			t.Fatalf("GetApp: %v", err)
		}
		if got.Name != "TestApp" {
			t.Errorf("name = %q, want %q", got.Name, "TestApp")
		}
		if got.Slug != "test-app" {
			t.Errorf("slug = %q, want %q", got.Slug, "test-app")
		}
		if got.WebhookSecret == "" {
			t.Error("webhook_secret should be auto-generated")
		}
		if got.Kind != "app" {
			t.Errorf("kind = %q, want %q", got.Kind, "app")
		}
		if got.Listing != "unlisted" {
			t.Errorf("listing = %q, want %q", got.Listing, "unlisted")
		}
		if got.Status != "active" {
			t.Errorf("status = %q, want %q", got.Status, "active")
		}
	})

	t.Run("GetAppBySlug", func(t *testing.T) {
		got, err := s.GetAppBySlug("test-app")
		if err != nil {
			t.Fatalf("GetAppBySlug: %v", err)
		}
		if got.ID != appID {
			t.Errorf("id = %q, want %q", got.ID, appID)
		}
	})

	t.Run("ListAppsByOwner", func(t *testing.T) {
		apps, err := s.ListAppsByOwner(u.ID)
		if err != nil {
			t.Fatalf("ListAppsByOwner: %v", err)
		}
		if len(apps) < 1 {
			t.Fatal("expected at least 1 app")
		}
	})

	t.Run("ListAllApps", func(t *testing.T) {
		apps, err := s.ListAllApps()
		if err != nil {
			t.Fatalf("ListAllApps: %v", err)
		}
		if len(apps) < 1 {
			t.Fatal("expected at least 1 app")
		}
	})

	t.Run("UpdateApp", func(t *testing.T) {
		err := s.UpdateApp(appID, "Updated App", "new desc", "icon", "http://icon.png",
			"http://home.com", "http://setup.com", "http://redirect.com",
			json.RawMessage(`[{"name":"tool1"}]`), json.RawMessage(`["msg"]`), json.RawMessage(`["read"]`))
		if err != nil {
			t.Fatalf("UpdateApp: %v", err)
		}
		got, _ := s.GetApp(appID)
		if got.Name != "Updated App" {
			t.Errorf("name = %q, want %q", got.Name, "Updated App")
		}
		if got.Description != "new desc" {
			t.Errorf("description = %q, want %q", got.Description, "new desc")
		}
	})

	t.Run("SetAppWebhookVerified", func(t *testing.T) {
		if err := s.SetAppWebhookVerified(appID, true); err != nil {
			t.Fatalf("SetAppWebhookVerified: %v", err)
		}
		got, _ := s.GetApp(appID)
		if !got.WebhookVerified {
			t.Error("expected webhook_verified=true")
		}
	})

	t.Run("ListListedApps", func(t *testing.T) {
		// App starts as "unlisted"; promote it to "listed" first.
		if err := s.RequestListing(appID); err != nil {
			t.Fatalf("RequestListing: %v", err)
		}
		if err := s.ReviewListing(appID, true, ""); err != nil {
			t.Fatalf("ReviewListing(approve): %v", err)
		}
		apps, err := s.ListListedApps()
		if err != nil {
			t.Fatalf("ListListedApps: %v", err)
		}
		if len(apps) < 1 {
			t.Fatal("expected at least 1 listed app")
		}
		// Reset back to unlisted so subsequent tests start from expected state.
		if err := s.ReviewListing(appID, false, "reset"); err != nil {
			t.Fatalf("ReviewListing(reset): %v", err)
		}
	})

	t.Run("UpdateAppWebhookURL", func(t *testing.T) {
		if err := s.UpdateAppWebhookURL(appID, "https://new.com/hook"); err != nil {
			t.Fatalf("UpdateAppWebhookURL: %v", err)
		}
		got, _ := s.GetApp(appID)
		if got.WebhookURL != "https://new.com/hook" {
			t.Errorf("webhook_url = %q, want %q", got.WebhookURL, "https://new.com/hook")
		}
		if got.WebhookVerified {
			t.Error("webhook_verified should be reset to false after UpdateAppWebhookURL")
		}
	})

	t.Run("RequestListing", func(t *testing.T) {
		if err := s.RequestListing(appID); err != nil {
			t.Fatalf("RequestListing: %v", err)
		}
		got, _ := s.GetApp(appID)
		if got.Listing != "pending" {
			t.Errorf("listing = %q, want %q", got.Listing, "pending")
		}
	})

	t.Run("ReviewListing_Reject", func(t *testing.T) {
		if err := s.ReviewListing(appID, false, "not ready"); err != nil {
			t.Fatalf("ReviewListing(reject): %v", err)
		}
		got, _ := s.GetApp(appID)
		if got.Listing != "rejected" {
			t.Errorf("listing = %q, want %q", got.Listing, "rejected")
		}
		if got.ListingRejectReason != "not ready" {
			t.Errorf("reject_reason = %q, want %q", got.ListingRejectReason, "not ready")
		}
	})

	t.Run("ReviewListing_Approve", func(t *testing.T) {
		s.RequestListing(appID)
		if err := s.ReviewListing(appID, true, ""); err != nil {
			t.Fatalf("ReviewListing(approve): %v", err)
		}
		got, _ := s.GetApp(appID)
		if got.Listing != "listed" {
			t.Errorf("listing = %q, want %q", got.Listing, "listed")
		}
	})

	// --- Installations ---
	var instID string

	t.Run("InstallApp", func(t *testing.T) {
		inst, err := s.InstallApp(appID, b.ID)
		if err != nil {
			t.Fatalf("InstallApp: %v", err)
		}
		instID = inst.ID
		if inst.AppToken == "" {
			t.Error("app_token should be generated")
		}
		if !inst.Enabled {
			t.Error("new installation should be enabled")
		}
	})

	t.Run("GetInstallation", func(t *testing.T) {
		got, err := s.GetInstallation(instID)
		if err != nil {
			t.Fatalf("GetInstallation: %v", err)
		}
		if got.AppID != appID {
			t.Errorf("app_id = %q, want %q", got.AppID, appID)
		}
		if got.BotID != b.ID {
			t.Errorf("bot_id = %q, want %q", got.BotID, b.ID)
		}
	})

	t.Run("GetInstallationByToken", func(t *testing.T) {
		inst, _ := s.GetInstallation(instID)
		got, err := s.GetInstallationByToken(inst.AppToken)
		if err != nil {
			t.Fatalf("GetInstallationByToken: %v", err)
		}
		if got.ID != instID {
			t.Errorf("id = %q, want %q", got.ID, instID)
		}
	})

	t.Run("ListInstallationsByApp", func(t *testing.T) {
		insts, err := s.ListInstallationsByApp(appID)
		if err != nil {
			t.Fatalf("ListInstallationsByApp: %v", err)
		}
		if len(insts) < 1 {
			t.Fatal("expected at least 1 installation")
		}
	})

	t.Run("ListInstallationsByBot", func(t *testing.T) {
		insts, err := s.ListInstallationsByBot(b.ID)
		if err != nil {
			t.Fatalf("ListInstallationsByBot: %v", err)
		}
		if len(insts) < 1 {
			t.Fatal("expected at least 1 installation")
		}
	})

	t.Run("UpdateInstallation", func(t *testing.T) {
		cfg := json.RawMessage(`{"key":"val"}`)
		if err := s.UpdateInstallation(instID, "myhandle", cfg, false); err != nil {
			t.Fatalf("UpdateInstallation: %v", err)
		}
		got, _ := s.GetInstallation(instID)
		if got.Handle != "myhandle" {
			t.Errorf("handle = %q, want %q", got.Handle, "myhandle")
		}
		if got.Enabled {
			t.Error("expected enabled=false after update")
		}
	})

	t.Run("GetInstallationByHandle", func(t *testing.T) {
		got, err := s.GetInstallationByHandle(b.ID, "myhandle")
		if err != nil {
			t.Fatalf("GetInstallationByHandle: %v", err)
		}
		if got.ID != instID {
			t.Errorf("id = %q, want %q", got.ID, instID)
		}
	})

	t.Run("RegenerateInstallationToken", func(t *testing.T) {
		oldInst, _ := s.GetInstallation(instID)
		newToken, err := s.RegenerateInstallationToken(instID)
		if err != nil {
			t.Fatalf("RegenerateInstallationToken: %v", err)
		}
		if newToken == oldInst.AppToken {
			t.Error("new token should differ from old token")
		}
		got, _ := s.GetInstallationByToken(newToken)
		if got.ID != instID {
			t.Error("new token should resolve to same installation")
		}
	})

	t.Run("OAuthCode", func(t *testing.T) {
		if err := s.CreateOAuthCode("code123", appID, b.ID, "state1", "challenge1"); err != nil {
			t.Fatalf("CreateOAuthCode: %v", err)
		}
		gotAppID, gotBotID, gotChallenge, err := s.ExchangeOAuthCode("code123")
		if err != nil {
			t.Fatalf("ExchangeOAuthCode: %v", err)
		}
		if gotAppID != appID {
			t.Errorf("appID = %q, want %q", gotAppID, appID)
		}
		if gotBotID != b.ID {
			t.Errorf("botID = %q, want %q", gotBotID, b.ID)
		}
		if gotChallenge != "challenge1" {
			t.Errorf("codeChallenge = %q, want %q", gotChallenge, "challenge1")
		}
		// Code should be consumed
		_, _, _, err = s.ExchangeOAuthCode("code123")
		if err == nil {
			t.Error("expected error re-using consumed code")
		}
	})

	t.Run("CleanExpiredOAuthCodes", func(t *testing.T) {
		s.CleanExpiredOAuthCodes()
	})

	t.Run("DeleteInstallation", func(t *testing.T) {
		if err := s.DeleteInstallation(instID); err != nil {
			t.Fatalf("DeleteInstallation: %v", err)
		}
		_, err := s.GetInstallation(instID)
		if err == nil {
			t.Error("expected error after deleting installation")
		}
	})

	t.Run("GetApp_NotFound", func(t *testing.T) {
		_, err := s.GetApp("nonexistent-app-id")
		if err == nil {
			t.Error("expected error for non-existent app, got nil")
		}
	})

	t.Run("ExchangeOAuthCode_NotFound", func(t *testing.T) {
		_, _, _, err := s.ExchangeOAuthCode("nonexistent-code")
		if err == nil {
			t.Error("expected error for non-existent OAuth code, got nil")
		}
	})

	t.Run("DeleteApp", func(t *testing.T) {
		app2 := mustCreateApp(t, s, u.ID, "ToDelete", "to-delete")
		if err := s.DeleteApp(app2.ID); err != nil {
			t.Fatalf("DeleteApp: %v", err)
		}
		_, err := s.GetApp(app2.ID)
		if err == nil {
			t.Error("expected error after deleting app")
		}
	})
}

// ---------------------------------------------------------------------------
// TestPluginCRUD
// ---------------------------------------------------------------------------

func TestPluginCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "pluginowner", "Plugin Owner")

	var pluginID string
	var versionID string

	t.Run("CreateAndGet", func(t *testing.T) {
		p, err := s.CreatePlugin(&store.Plugin{
			Name:        "my-plugin",
			Namespace:   "ns",
			Description: "A test plugin",
			Author:      "tester",
			Icon:        "icon.png",
			License:     "MIT",
			Homepage:    "https://example.com",
			OwnerID:     u.ID,
		})
		if err != nil {
			t.Fatalf("CreatePlugin: %v", err)
		}
		pluginID = p.ID

		got, err := s.GetPlugin(p.ID)
		if err != nil {
			t.Fatalf("GetPlugin: %v", err)
		}
		if got.Name != "my-plugin" {
			t.Errorf("name = %q, want %q", got.Name, "my-plugin")
		}
		if got.OwnerID != u.ID {
			t.Errorf("owner_id = %q, want %q", got.OwnerID, u.ID)
		}
	})

	t.Run("GetPluginByName", func(t *testing.T) {
		got, err := s.GetPluginByName("my-plugin")
		if err != nil {
			t.Fatalf("GetPluginByName: %v", err)
		}
		if got.ID != pluginID {
			t.Errorf("id = %q, want %q", got.ID, pluginID)
		}
	})

	t.Run("ListPluginsByOwner", func(t *testing.T) {
		plugins, err := s.ListPluginsByOwner(u.ID)
		if err != nil {
			t.Fatalf("ListPluginsByOwner: %v", err)
		}
		// ListPluginsByOwner returns all regardless of latest_version_id
		// Actually it shows all for owner including those without versions
		_ = plugins
	})

	t.Run("UpdatePluginMeta", func(t *testing.T) {
		if err := s.UpdatePluginMeta(pluginID, &store.Plugin{
			Description: "Updated desc",
			Author:      "new author",
			Icon:        "new.png",
			License:     "Apache-2.0",
			Homepage:    "https://new.com",
			Namespace:   "ns2",
		}); err != nil {
			t.Fatalf("UpdatePluginMeta: %v", err)
		}
		got, _ := s.GetPlugin(pluginID)
		if got.Description != "Updated desc" {
			t.Errorf("description = %q, want %q", got.Description, "Updated desc")
		}
	})

	t.Run("CreatePluginVersion", func(t *testing.T) {
		v, err := s.CreatePluginVersion(&store.PluginVersion{
			PluginID:       pluginID,
			Version:        "1.0.0",
			Changelog:      "Initial release",
			Script:         "console.log('hello')",
			ConfigSchema:   json.RawMessage(`[]`),
			MatchTypes:     "*",
			ConnectDomains: "*",
			GrantPerms:     "",
			TimeoutSec:     10,
		})
		if err != nil {
			t.Fatalf("CreatePluginVersion: %v", err)
		}
		versionID = v.ID
		if v.Status != "pending" {
			t.Errorf("status = %q, want %q", v.Status, "pending")
		}
	})

	t.Run("GetPluginVersion", func(t *testing.T) {
		got, err := s.GetPluginVersion(versionID)
		if err != nil {
			t.Fatalf("GetPluginVersion: %v", err)
		}
		if got.Version != "1.0.0" {
			t.Errorf("version = %q, want %q", got.Version, "1.0.0")
		}
		if got.Script != "console.log('hello')" {
			t.Errorf("script mismatch")
		}
	})

	t.Run("ListPluginVersions", func(t *testing.T) {
		versions, err := s.ListPluginVersions(pluginID)
		if err != nil {
			t.Fatalf("ListPluginVersions: %v", err)
		}
		if len(versions) < 1 {
			t.Fatal("expected at least 1 version")
		}
	})

	t.Run("ListPendingVersions", func(t *testing.T) {
		versions, err := s.ListPendingVersions()
		if err != nil {
			t.Fatalf("ListPendingVersions: %v", err)
		}
		found := false
		for _, v := range versions {
			if v.ID == versionID {
				found = true
				break
			}
		}
		if !found {
			t.Error("pending version not found in ListPendingVersions")
		}
	})

	t.Run("FindPendingVersion", func(t *testing.T) {
		got, err := s.FindPendingVersion(pluginID)
		if err != nil {
			t.Fatalf("FindPendingVersion: %v", err)
		}
		if got.ID != versionID {
			t.Errorf("id = %q, want %q", got.ID, versionID)
		}
	})

	t.Run("UpdatePluginVersion", func(t *testing.T) {
		if err := s.UpdatePluginVersion(versionID, &store.PluginVersion{
			Version:        "1.0.1",
			Changelog:      "Bugfix",
			Script:         "console.log('fixed')",
			ConfigSchema:   json.RawMessage(`[]`),
			MatchTypes:     "*",
			ConnectDomains: "*",
			GrantPerms:     "",
			TimeoutSec:     5,
		}); err != nil {
			t.Fatalf("UpdatePluginVersion: %v", err)
		}
		got, _ := s.GetPluginVersion(versionID)
		if got.Version != "1.0.1" {
			t.Errorf("version = %q, want %q", got.Version, "1.0.1")
		}
	})

	t.Run("ReviewPluginVersion_Approve", func(t *testing.T) {
		if err := s.ReviewPluginVersion(versionID, "approved", u.ID, ""); err != nil {
			t.Fatalf("ReviewPluginVersion: %v", err)
		}
		got, _ := s.GetPluginVersion(versionID)
		if got.Status != "approved" {
			t.Errorf("status = %q, want %q", got.Status, "approved")
		}
		// latest_version_id should be updated on the plugin
		plugin, _ := s.GetPlugin(pluginID)
		if plugin.LatestVersionID != versionID {
			t.Errorf("latest_version_id = %q, want %q", plugin.LatestVersionID, versionID)
		}
	})

	t.Run("ListPlugins", func(t *testing.T) {
		// Now that we have an approved version, plugin should appear
		plugins, err := s.ListPlugins()
		if err != nil {
			t.Fatalf("ListPlugins: %v", err)
		}
		found := false
		for _, p := range plugins {
			if p.ID == pluginID {
				found = true
				if p.Version != "1.0.1" {
					t.Errorf("version = %q, want %q", p.Version, "1.0.1")
				}
				break
			}
		}
		if !found {
			t.Error("plugin not found in ListPlugins")
		}
	})

	t.Run("ResolvePluginScript", func(t *testing.T) {
		script, version, timeout, err := s.ResolvePluginScript(versionID)
		if err != nil {
			t.Fatalf("ResolvePluginScript: %v", err)
		}
		if script != "console.log('fixed')" {
			t.Errorf("script mismatch")
		}
		if version != "1.0.1" {
			t.Errorf("version = %q, want %q", version, "1.0.1")
		}
		if timeout != 5 {
			t.Errorf("timeout = %d, want 5", timeout)
		}
	})

	t.Run("SupersedeNonApprovedVersions", func(t *testing.T) {
		// Create a new pending version, then supersede
		v2, _ := s.CreatePluginVersion(&store.PluginVersion{
			PluginID:     pluginID,
			Version:      "2.0.0",
			Script:       "v2",
			ConfigSchema: json.RawMessage(`[]`),
			MatchTypes:   "*",
		})
		s.SupersedeNonApprovedVersions(pluginID)
		got, _ := s.GetPluginVersion(v2.ID)
		if got.Status != "superseded" {
			t.Errorf("status = %q, want %q", got.Status, "superseded")
		}
	})

	t.Run("CancelPluginVersion", func(t *testing.T) {
		v3, _ := s.CreatePluginVersion(&store.PluginVersion{
			PluginID:     pluginID,
			Version:      "3.0.0",
			Script:       "v3",
			ConfigSchema: json.RawMessage(`[]`),
			MatchTypes:   "*",
		})
		if err := s.CancelPluginVersion(v3.ID); err != nil {
			t.Fatalf("CancelPluginVersion: %v", err)
		}
		got, _ := s.GetPluginVersion(v3.ID)
		if got.Status != "cancelled" {
			t.Errorf("status = %q, want %q", got.Status, "cancelled")
		}
	})

	t.Run("RecordPluginInstall", func(t *testing.T) {
		if err := s.RecordPluginInstall(pluginID, u.ID); err != nil {
			t.Fatalf("RecordPluginInstall: %v", err)
		}
		got, _ := s.GetPlugin(pluginID)
		if got.InstallCount < 1 {
			t.Errorf("install_count = %d, want >= 1", got.InstallCount)
		}
	})

	t.Run("FindPluginOwner", func(t *testing.T) {
		ownerID, err := s.FindPluginOwner("my-plugin")
		if err != nil {
			t.Fatalf("FindPluginOwner: %v", err)
		}
		if ownerID != u.ID {
			t.Errorf("owner_id = %q, want %q", ownerID, u.ID)
		}
	})

	t.Run("DeletePluginVersion", func(t *testing.T) {
		v4, _ := s.CreatePluginVersion(&store.PluginVersion{
			PluginID:     pluginID,
			Version:      "4.0.0",
			Script:       "v4",
			ConfigSchema: json.RawMessage(`[]`),
			MatchTypes:   "*",
		})
		if err := s.DeletePluginVersion(v4.ID); err != nil {
			t.Fatalf("DeletePluginVersion: %v", err)
		}
		_, err := s.GetPluginVersion(v4.ID)
		if err == nil {
			t.Error("expected error after deleting plugin version")
		}
	})

	t.Run("GetPlugin_NotFound", func(t *testing.T) {
		_, err := s.GetPlugin("nonexistent-plugin-id")
		if err == nil {
			t.Error("expected error for non-existent plugin, got nil")
		}
	})

	t.Run("ResolvePluginScript_NotApproved", func(t *testing.T) {
		// Create a new pending version and try to resolve it
		pendingV, err := s.CreatePluginVersion(&store.PluginVersion{
			PluginID:     pluginID,
			Version:      "9.0.0",
			Script:       "pending script",
			ConfigSchema: json.RawMessage(`[]`),
			MatchTypes:   "*",
		})
		if err != nil {
			t.Fatalf("CreatePluginVersion: %v", err)
		}
		_, _, _, err = s.ResolvePluginScript(pendingV.ID)
		if err == nil {
			t.Error("expected error resolving non-approved plugin version, got nil")
		}
		// Clean up
		s.CancelPluginVersion(pendingV.ID)
	})

	t.Run("DeletePlugin", func(t *testing.T) {
		p2, _ := s.CreatePlugin(&store.Plugin{
			Name:    "to-delete-plugin",
			OwnerID: u.ID,
		})
		if err := s.DeletePlugin(p2.ID); err != nil {
			t.Fatalf("DeletePlugin: %v", err)
		}
		_, err := s.GetPlugin(p2.ID)
		if err == nil {
			t.Error("expected error after deleting plugin")
		}
	})
}

// ---------------------------------------------------------------------------
// TestTraceCRUD
// ---------------------------------------------------------------------------

func TestTraceCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "traceowner", "Trace Owner")
	b := mustCreateBot(t, s, u.ID, "TraceBot")

	traceID := fmt.Sprintf("tr_test_%d", time.Now().UnixNano())
	rootSpanID := "root0001"

	t.Run("InsertSpan", func(t *testing.T) {
		attrsJSON, _ := json.Marshal(map[string]any{"key": "val"})
		eventsJSON, _ := json.Marshal([]store.SpanEvent{})
		err := s.InsertSpan(traceID, rootSpanID, "", "root-span", "server", "ok", "",
			1000, 2000, attrsJSON, eventsJSON, b.ID)
		if err != nil {
			t.Fatalf("InsertSpan: %v", err)
		}
	})

	t.Run("ListSpansByTrace", func(t *testing.T) {
		spans, err := s.ListSpansByTrace(traceID)
		if err != nil {
			t.Fatalf("ListSpansByTrace: %v", err)
		}
		if len(spans) < 1 {
			t.Fatal("expected at least 1 span")
		}
		if spans[0].TraceID != traceID {
			t.Errorf("trace_id = %q, want %q", spans[0].TraceID, traceID)
		}
	})

	t.Run("ListRootSpans", func(t *testing.T) {
		spans, err := s.ListRootSpans(b.ID, 10)
		if err != nil {
			t.Fatalf("ListRootSpans: %v", err)
		}
		if len(spans) < 1 {
			t.Fatal("expected at least 1 root span")
		}
		for _, sp := range spans {
			if sp.ParentSpanID != "" {
				t.Errorf("root span has parent_span_id = %q", sp.ParentSpanID)
			}
		}
	})

	t.Run("AppendSpan", func(t *testing.T) {
		err := s.AppendSpan(traceID, b.ID, "child-span", "client", "ok", "",
			map[string]any{"extra": true})
		if err != nil {
			t.Fatalf("AppendSpan: %v", err)
		}
		spans, _ := s.ListSpansByTrace(traceID)
		if len(spans) < 2 {
			t.Fatalf("expected at least 2 spans, got %d", len(spans))
		}
		// Find the child and verify parent
		foundChild := false
		for _, sp := range spans {
			if sp.Name == "child-span" {
				foundChild = true
				if sp.ParentSpanID != rootSpanID {
					t.Errorf("parent_span_id = %q, want %q", sp.ParentSpanID, rootSpanID)
				}
			}
		}
		if !foundChild {
			t.Error("child span not found")
		}
	})
}

// ---------------------------------------------------------------------------
// TestCredentialCRUD
// ---------------------------------------------------------------------------

func TestCredentialCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "credowner", "Cred Owner")

	t.Run("SaveAndGet", func(t *testing.T) {
		c := &store.Credential{
			ID:              "cred_001",
			UserID:          u.ID,
			PublicKey:        []byte("pubkey123"),
			AttestationType: "none",
			Transport:       "usb",
			SignCount:       0,
		}
		if err := s.SaveCredential(c); err != nil {
			t.Fatalf("SaveCredential: %v", err)
		}
		creds, err := s.GetCredentialsByUserID(u.ID)
		if err != nil {
			t.Fatalf("GetCredentialsByUserID: %v", err)
		}
		if len(creds) < 1 {
			t.Fatal("expected at least 1 credential")
		}
		if string(creds[0].PublicKey) != "pubkey123" {
			t.Errorf("public_key = %q, want %q", string(creds[0].PublicKey), "pubkey123")
		}
	})

	t.Run("UpdateCredentialSignCount", func(t *testing.T) {
		if err := s.UpdateCredentialSignCount("cred_001", 5); err != nil {
			t.Fatalf("UpdateCredentialSignCount: %v", err)
		}
		creds, _ := s.GetCredentialsByUserID(u.ID)
		if creds[0].SignCount != 5 {
			t.Errorf("sign_count = %d, want 5", creds[0].SignCount)
		}
	})

	t.Run("DeleteCredential", func(t *testing.T) {
		if err := s.DeleteCredential("cred_001", u.ID); err != nil {
			t.Fatalf("DeleteCredential: %v", err)
		}
		creds, _ := s.GetCredentialsByUserID(u.ID)
		if len(creds) != 0 {
			t.Errorf("expected 0 credentials after delete, got %d", len(creds))
		}
	})
}

// ---------------------------------------------------------------------------
// TestOAuthCRUD
// ---------------------------------------------------------------------------

func TestOAuthCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "oauthowner", "OAuth Owner")

	t.Run("CreateAndGet", func(t *testing.T) {
		a := &store.OAuthAccount{
			Provider:   "github",
			ProviderID: "gh_123",
			UserID:     u.ID,
			Username:   "ghuser",
			AvatarURL:  "https://avatar.com/pic.png",
		}
		if err := s.CreateOAuthAccount(a); err != nil {
			t.Fatalf("CreateOAuthAccount: %v", err)
		}
		got, err := s.GetOAuthAccount("github", "gh_123")
		if err != nil {
			t.Fatalf("GetOAuthAccount: %v", err)
		}
		if got.Username != "ghuser" {
			t.Errorf("username = %q, want %q", got.Username, "ghuser")
		}
		if got.AvatarURL != "https://avatar.com/pic.png" {
			t.Errorf("avatar_url = %q", got.AvatarURL)
		}
	})

	t.Run("ListOAuthAccountsByUser", func(t *testing.T) {
		accts, err := s.ListOAuthAccountsByUser(u.ID)
		if err != nil {
			t.Fatalf("ListOAuthAccountsByUser: %v", err)
		}
		if len(accts) < 1 {
			t.Fatal("expected at least 1 account")
		}
	})

	t.Run("GetOAuthAccount_NotFound", func(t *testing.T) {
		_, err := s.GetOAuthAccount("nosuchprovider", "nosuchid")
		if err == nil {
			t.Error("expected error for non-existent OAuth account, got nil")
		}
	})

	t.Run("DeleteOAuthAccount", func(t *testing.T) {
		if err := s.DeleteOAuthAccount("github", "gh_123"); err != nil {
			t.Fatalf("DeleteOAuthAccount: %v", err)
		}
		_, err := s.GetOAuthAccount("github", "gh_123")
		if err == nil {
			t.Error("expected error after deleting oauth account")
		}
	})
}

// ---------------------------------------------------------------------------
// TestConfigCRUD
// ---------------------------------------------------------------------------

func TestConfigCRUD(t *testing.T, s store.Store) {
	t.Run("SetAndGet", func(t *testing.T) {
		if err := s.SetConfig("test.key1", "value1"); err != nil {
			t.Fatalf("SetConfig: %v", err)
		}
		val, err := s.GetConfig("test.key1")
		if err != nil {
			t.Fatalf("GetConfig: %v", err)
		}
		if val != "value1" {
			t.Errorf("value = %q, want %q", val, "value1")
		}
	})

	t.Run("SetConfig_Overwrite", func(t *testing.T) {
		s.SetConfig("test.key1", "updated")
		val, _ := s.GetConfig("test.key1")
		if val != "updated" {
			t.Errorf("value = %q, want %q", val, "updated")
		}
	})

	t.Run("GetConfig_NonExistent", func(t *testing.T) {
		val, err := s.GetConfig("nonexistent.key")
		if err != nil {
			t.Fatalf("GetConfig (nonexistent): %v", err)
		}
		if val != "" {
			t.Errorf("value = %q, want empty string", val)
		}
	})

	t.Run("ListConfigByPrefix", func(t *testing.T) {
		s.SetConfig("test.key2", "val2")
		s.SetConfig("test.key3", "val3")
		m, err := s.ListConfigByPrefix("test.")
		if err != nil {
			t.Fatalf("ListConfigByPrefix: %v", err)
		}
		if len(m) < 3 {
			t.Errorf("expected >= 3 entries, got %d", len(m))
		}
	})

	t.Run("DeleteConfig", func(t *testing.T) {
		if err := s.DeleteConfig("test.key1"); err != nil {
			t.Fatalf("DeleteConfig: %v", err)
		}
		val, _ := s.GetConfig("test.key1")
		if val != "" {
			t.Errorf("value = %q, want empty after delete", val)
		}
	})
}

// ---------------------------------------------------------------------------
// TestWebhookLogCRUD
// ---------------------------------------------------------------------------

func TestWebhookLogCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "whlogowner", "WH Log Owner")
	b := mustCreateBot(t, s, u.ID, "WHLogBot")
	ch := mustCreateChannel(t, s, b.ID, "WHLogChan", "whlogchan")

	var logID int64

	t.Run("CreateWebhookLog", func(t *testing.T) {
		msgID := int64(999)
		id, err := s.CreateWebhookLog(&store.WebhookLog{
			BotID:     b.ID,
			ChannelID: ch.ID,
			MessageID: &msgID,
		})
		if err != nil {
			t.Fatalf("CreateWebhookLog: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
		logID = id
	})

	t.Run("ListWebhookLogs", func(t *testing.T) {
		logs, err := s.ListWebhookLogs(b.ID, "", 10)
		if err != nil {
			t.Fatalf("ListWebhookLogs: %v", err)
		}
		if len(logs) < 1 {
			t.Fatal("expected at least 1 log")
		}
	})

	t.Run("UpdateWebhookLogRequest", func(t *testing.T) {
		err := s.UpdateWebhookLogRequest(logID, "sending", "https://hook.com", "POST", `{"data":"test"}`)
		if err != nil {
			t.Fatalf("UpdateWebhookLogRequest: %v", err)
		}
	})

	t.Run("UpdateWebhookLogResponse", func(t *testing.T) {
		err := s.UpdateWebhookLogResponse(logID, "delivered", 200, `{"ok":true}`, 150)
		if err != nil {
			t.Fatalf("UpdateWebhookLogResponse: %v", err)
		}
	})

	t.Run("UpdateWebhookLogResult", func(t *testing.T) {
		err := s.UpdateWebhookLogResult(logID, "completed", "", []string{"reply1"})
		if err != nil {
			t.Fatalf("UpdateWebhookLogResult: %v", err)
		}
	})

	t.Run("UpdateWebhookLogPluginVersion", func(t *testing.T) {
		err := s.UpdateWebhookLogPluginVersion(logID, "1.0.0")
		if err != nil {
			t.Fatalf("UpdateWebhookLogPluginVersion: %v", err)
		}
	})

	t.Run("ListWebhookLogs_ByChannel", func(t *testing.T) {
		logs, err := s.ListWebhookLogs(b.ID, ch.ID, 10)
		if err != nil {
			t.Fatalf("ListWebhookLogs(channel): %v", err)
		}
		if len(logs) < 1 {
			t.Fatal("expected at least 1 log for channel")
		}
		// Verify the update fields
		log := logs[0]
		if log.RequestURL != "https://hook.com" {
			t.Errorf("request_url = %q, want %q", log.RequestURL, "https://hook.com")
		}
		if log.ResponseStatus != 200 {
			t.Errorf("response_status = %d, want 200", log.ResponseStatus)
		}
	})

	t.Run("CleanOldWebhookLogs", func(t *testing.T) {
		err := s.CleanOldWebhookLogs(9999)
		if err != nil {
			t.Fatalf("CleanOldWebhookLogs: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestAppLogCRUD
// ---------------------------------------------------------------------------

func TestAppLogCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "applogowner", "App Log Owner")
	b := mustCreateBot(t, s, u.ID, "AppLogBot")
	app := mustCreateApp(t, s, u.ID, "AppLogApp", "applog-app")
	inst, err := s.InstallApp(app.ID, b.ID)
	if err != nil {
		t.Fatalf("InstallApp: %v", err)
	}

	var eventLogID int64

	t.Run("CreateEventLog", func(t *testing.T) {
		id, err := s.CreateEventLog(&store.AppEventLog{
			InstallationID: inst.ID,
			TraceID:        "trace_001",
			EventType:      "message.created",
			EventID:        "evt_001",
			RequestBody:    `{"msg":"hello"}`,
		})
		if err != nil {
			t.Fatalf("CreateEventLog: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
		eventLogID = id
	})

	t.Run("ListEventLogs", func(t *testing.T) {
		logs, err := s.ListEventLogs(inst.ID, 10)
		if err != nil {
			t.Fatalf("ListEventLogs: %v", err)
		}
		if len(logs) < 1 {
			t.Fatal("expected at least 1 event log")
		}
	})

	t.Run("UpdateEventLogDelivered", func(t *testing.T) {
		err := s.UpdateEventLogDelivered(eventLogID, 200, `{"ok":true}`, 100)
		if err != nil {
			t.Fatalf("UpdateEventLogDelivered: %v", err)
		}
		logs, _ := s.ListEventLogs(inst.ID, 10)
		found := false
		for _, l := range logs {
			if l.ID == eventLogID {
				found = true
				if l.Status != "delivered" {
					t.Errorf("status = %q, want %q", l.Status, "delivered")
				}
				if l.ResponseStatus != 200 {
					t.Errorf("response_status = %d, want 200", l.ResponseStatus)
				}
			}
		}
		if !found {
			t.Error("event log not found")
		}
	})

	t.Run("UpdateEventLogFailed", func(t *testing.T) {
		id2, _ := s.CreateEventLog(&store.AppEventLog{
			InstallationID: inst.ID,
			TraceID:        "trace_002",
			EventType:      "message.created",
			EventID:        "evt_002",
		})
		err := s.UpdateEventLogFailed(id2, "timeout", 1, 5000)
		if err != nil {
			t.Fatalf("UpdateEventLogFailed: %v", err)
		}
		logs, _ := s.ListEventLogs(inst.ID, 10)
		for _, l := range logs {
			if l.ID == id2 {
				if l.Status != "retrying" {
					t.Errorf("status = %q, want %q", l.Status, "retrying")
				}
			}
		}
	})

	t.Run("CreateAPILog", func(t *testing.T) {
		err := s.CreateAPILog(&store.AppAPILog{
			InstallationID: inst.ID,
			TraceID:        "trace_003",
			Method:         "POST",
			Path:           "/api/messages",
			RequestBody:    `{"text":"hi"}`,
			StatusCode:     200,
			ResponseBody:   `{"ok":true}`,
			DurationMs:     50,
		})
		if err != nil {
			t.Fatalf("CreateAPILog: %v", err)
		}
	})

	t.Run("ListAPILogs", func(t *testing.T) {
		logs, err := s.ListAPILogs(inst.ID, 10)
		if err != nil {
			t.Fatalf("ListAPILogs: %v", err)
		}
		if len(logs) < 1 {
			t.Fatal("expected at least 1 API log")
		}
		if logs[0].Method != "POST" {
			t.Errorf("method = %q, want %q", logs[0].Method, "POST")
		}
	})

	t.Run("CleanOldAppLogs", func(t *testing.T) {
		err := s.CleanOldAppLogs(9999)
		if err != nil {
			t.Fatalf("CleanOldAppLogs: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestSessionCRUD
// ---------------------------------------------------------------------------

func TestSessionCRUD(t *testing.T, s store.Store) {
	u := mustCreateUser(t, s, "sessowner", "Sess Owner")

	t.Run("CreateAndGet", func(t *testing.T) {
		exp := time.Now().Add(24 * time.Hour)
		if err := s.CreateSession("tok_001", u.ID, exp); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		userID, expiresAt, err := s.GetSession("tok_001")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if userID != u.ID {
			t.Errorf("userID = %q, want %q", userID, u.ID)
		}
		// Compare at second precision (epoch storage)
		if expiresAt.Unix() != exp.Unix() {
			t.Errorf("expiresAt = %v, want %v", expiresAt.Unix(), exp.Unix())
		}
	})

	t.Run("GetSession_NotFound", func(t *testing.T) {
		_, _, err := s.GetSession("nonexistent-token")
		if err == nil {
			t.Error("expected error for non-existent session token, got nil")
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		if err := s.DeleteSession("tok_001"); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		_, _, err := s.GetSession("tok_001")
		if err == nil {
			t.Error("expected error after deleting session")
		}
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("DeleteExpiredSessions", func(t *testing.T) {
		// Create an already-expired session
		past := time.Now().Add(-1 * time.Hour)
		s.CreateSession("tok_expired", u.ID, past)
		if err := s.DeleteExpiredSessions(); err != nil {
			t.Fatalf("DeleteExpiredSessions: %v", err)
		}
		_, _, err := s.GetSession("tok_expired")
		if err == nil {
			t.Error("expired session should be deleted")
		}
	})

	t.Run("DeleteSessionsByUserID", func(t *testing.T) {
		s.CreateSession("tok_a", u.ID, time.Now().Add(time.Hour))
		s.CreateSession("tok_b", u.ID, time.Now().Add(time.Hour))
		if err := s.DeleteSessionsByUserID(u.ID); err != nil {
			t.Fatalf("DeleteSessionsByUserID: %v", err)
		}
		_, _, errA := s.GetSession("tok_a")
		_, _, errB := s.GetSession("tok_b")
		if errA == nil || errB == nil {
			t.Error("sessions should be deleted by user ID")
		}
	})
}
