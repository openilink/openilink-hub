package app

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

// --- Mock DB for matching tests ---

type mockAppStore struct {
	installations []store.AppInstallation
	apps          map[string]*store.App
	listErr       error
	getAppErr     error
}

func (m *mockAppStore) ListInstallationsByBot(_ string) ([]store.AppInstallation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.installations, nil
}

func (m *mockAppStore) GetApp(id string) (*store.App, error) {
	if m.getAppErr != nil {
		return nil, m.getAppErr
	}
	if app, ok := m.apps[id]; ok {
		return app, nil
	}
	return nil, errors.New("not found")
}

func (m *mockAppStore) GetInstallationByHandle(botID, handle string) (*store.AppInstallation, error) {
	for i := range m.installations {
		if m.installations[i].Handle == handle {
			return &m.installations[i], nil
		}
	}
	return nil, errors.New("not found")
}

func newMatchDispatcher(store *mockAppStore) *Dispatcher {
	return &Dispatcher{
		appDB: store,
		dbLog: &mockLogDB{},
	}
}

// --- parseCommand tests ---

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		content string
		cmd     string
		args    string
	}{
		{"simple command", "/github list PRs", "github", "list PRs"},
		{"command no args", "/help", "help", ""},
		{"mention then command", "@github /github list PRs", "github", "list PRs"},
		{"mention then command no args", "@bot /status", "status", ""},
		{"mention only", "@github", "", ""},
		{"empty string", "", "", ""},
		{"no slash prefix", "hello world", "", ""},
		{"slash only", "/", "", ""},
		{"mention with space no command", "@bot hello", "", ""},
		{"leading whitespace", "  /cmd arg", "cmd", "arg"},
		{"uppercase command lowered", "/GitHub PRs", "github", "PRs"},
		{"multiple spaces in args", "/cmd  arg1  arg2", "arg1  arg2", ""},
	}

	// Last test case needs correction -- the code does SplitN with 2 parts on
	// "cmd  arg1  arg2" after stripping "/", so parts[0]="cmd", parts[1]=" arg1  arg2"
	// which gets TrimSpace to "arg1  arg2".
	tests[len(tests)-1].cmd = "cmd"
	tests[len(tests)-1].args = "arg1  arg2"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := parseCommand(tt.content)
			if cmd != tt.cmd {
				t.Errorf("parseCommand(%q) cmd = %q, want %q", tt.content, cmd, tt.cmd)
			}
			if args != tt.args {
				t.Errorf("parseCommand(%q) args = %q, want %q", tt.content, args, tt.args)
			}
		})
	}
}

func TestParseCommandMentionFormats(t *testing.T) {
	// @mention /command args -> command, args
	cmd, args := parseCommand("@mybot /deploy production")
	if cmd != "deploy" {
		t.Errorf("cmd = %q, want %q", cmd, "deploy")
	}
	if args != "production" {
		t.Errorf("args = %q, want %q", args, "production")
	}
}

// --- appHasCommand tests ---

func TestAppHasCommand(t *testing.T) {
	tools, _ := json.Marshal([]store.AppTool{
		{Name: "list_prs", Command: "github"},
		{Name: "run_deploy", Command: "Deploy"},
		{Name: "no_slash_tool", Description: "tool without command trigger"},
	})
	app := &store.App{ID: "a1", Tools: tools}

	if !appHasCommand(app, "github") {
		t.Error("should match 'github'")
	}
	if !appHasCommand(app, "deploy") {
		t.Error("should match 'deploy' (case insensitive)")
	}
	if !appHasCommand(app, "GITHUB") {
		t.Error("should match 'GITHUB' (case insensitive)")
	}
	if appHasCommand(app, "unknown") {
		t.Error("should not match 'unknown'")
	}
	if appHasCommand(app, "no_slash_tool") {
		t.Error("should not match tool name when Command is empty")
	}
}

func TestAppHasCommand_NilApp(t *testing.T) {
	if appHasCommand(nil, "cmd") {
		t.Error("nil app should return false")
	}
}

func TestAppHasCommand_EmptyTools(t *testing.T) {
	app := &store.App{Tools: nil}
	if appHasCommand(app, "cmd") {
		t.Error("nil tools should return false")
	}
}

func TestAppHasCommand_InvalidJSON(t *testing.T) {
	app := &store.App{Tools: json.RawMessage(`invalid`)}
	if appHasCommand(app, "cmd") {
		t.Error("invalid JSON should return false")
	}
}

// --- appSubscribesToEvent tests ---

func TestAppSubscribesToEvent(t *testing.T) {
	events, _ := json.Marshal([]string{"message", "reaction.added"})
	app := &store.App{ID: "a1", Events: events}

	tests := []struct {
		eventType string
		want      bool
	}{
		{"message", true},          // exact match
		{"message.text", true},     // wildcard match
		{"message.image", true},    // wildcard match
		{"reaction.added", true},   // exact match
		{"reaction.removed", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := appSubscribesToEvent(app, tt.eventType)
			if got != tt.want {
				t.Errorf("appSubscribesToEvent(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestAppSubscribesToEvent_NilApp(t *testing.T) {
	if appSubscribesToEvent(nil, "message") {
		t.Error("nil app should return false")
	}
}

func TestAppSubscribesToEvent_EmptyEvents(t *testing.T) {
	app := &store.App{Events: nil}
	if appSubscribesToEvent(app, "message") {
		t.Error("nil events should return false")
	}
}

func TestAppSubscribesToEvent_InvalidJSON(t *testing.T) {
	app := &store.App{Events: json.RawMessage(`bad`)}
	if appSubscribesToEvent(app, "message") {
		t.Error("invalid JSON should return false")
	}
}

// --- MatchCommand tests ---

func TestMatchCommand_Success(t *testing.T) {
	cmds, _ := json.Marshal([]store.AppTool{{Name: "list_prs", Command: "github"}})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Tools: cmds},
		},
	}

	d := newMatchDispatcher(store)
	matched, cmd, args, err := d.MatchCommand("b1", "/github list PRs")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if cmd != "github" {
		t.Errorf("cmd = %q, want %q", cmd, "github")
	}
	if args != "list PRs" {
		t.Errorf("args = %q, want %q", args, "list PRs")
	}
	if len(matched) != 1 {
		t.Errorf("matched %d installations, want 1", len(matched))
	}
}

func TestMatchCommand_NoMatch(t *testing.T) {
	cmds, _ := json.Marshal([]store.AppTool{{Name: "list_prs", Command: "github"}})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Tools: cmds},
		},
	}

	d := newMatchDispatcher(store)
	matched, _, _, err := d.MatchCommand("b1", "/unknown foo")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("matched %d installations, want 0", len(matched))
	}
}

func TestMatchCommand_DisabledInstallation(t *testing.T) {
	cmds, _ := json.Marshal([]store.AppTool{{Name: "run_cmd", Command: "cmd"}})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: false, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Tools: cmds},
		},
	}

	d := newMatchDispatcher(store)
	matched, _, _, _ := d.MatchCommand("b1", "/cmd")
	if len(matched) != 0 {
		t.Error("disabled installation should be excluded")
	}
}

func TestMatchCommand_NoRequestURL(t *testing.T) {
	cmds, _ := json.Marshal([]store.AppTool{{Name: "run_cmd", Command: "cmd"}})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: ""},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Tools: cmds},
		},
	}

	d := newMatchDispatcher(store)
	matched, _, _, _ := d.MatchCommand("b1", "/cmd")
	if len(matched) != 1 {
		t.Errorf("installation without request_url should still be included, got %d", len(matched))
	}
}

func TestMatchCommand_NotACommand(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{},
		apps:          map[string]*store.App{},
	}
	d := newMatchDispatcher(store)
	matched, cmd, _, err := d.MatchCommand("b1", "hello world")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if cmd != "" {
		t.Errorf("cmd = %q, want empty", cmd)
	}
	if matched != nil {
		t.Error("matched should be nil for non-command content")
	}
}

func TestMatchCommand_ListError(t *testing.T) {
	store := &mockAppStore{listErr: errFake}
	d := newMatchDispatcher(store)
	_, _, _, err := d.MatchCommand("b1", "/cmd")
	if err == nil {
		t.Error("expected error from list")
	}
}

// --- MatchEvent tests ---

func TestMatchEvent_Success(t *testing.T) {
	events, _ := json.Marshal([]string{"message"})
	scopes, _ := json.Marshal([]string{"message:read"})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Events: events, Scopes: scopes},
		},
	}

	d := newMatchDispatcher(store)
	matched, err := d.MatchEvent("b1", "message.text")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(matched) != 1 {
		t.Errorf("matched %d installations, want 1", len(matched))
	}
}

func TestMatchEvent_NoSubscription(t *testing.T) {
	events, _ := json.Marshal([]string{"reaction.added"})
	scopes, _ := json.Marshal([]string{"message:read"})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Events: events, Scopes: scopes},
		},
	}

	d := newMatchDispatcher(store)
	matched, err := d.MatchEvent("b1", "message.text")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("matched %d, want 0", len(matched))
	}
}

func TestMatchEvent_DisabledExcluded(t *testing.T) {
	events, _ := json.Marshal([]string{"message"})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: false, AppWebhookURL: "http://example.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Events: events},
		},
	}

	d := newMatchDispatcher(store)
	matched, _ := d.MatchEvent("b1", "message.text")
	if len(matched) != 0 {
		t.Error("disabled installations should be excluded")
	}
}

func TestMatchEvent_ListError(t *testing.T) {
	store := &mockAppStore{listErr: errFake}
	d := newMatchDispatcher(store)
	_, err := d.MatchEvent("b1", "message.text")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMatchEvent_GetAppError(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://example.com"},
		},
		apps:      map[string]*store.App{},
		getAppErr: errFake,
	}

	d := newMatchDispatcher(store)
	matched, err := d.MatchEvent("b1", "message.text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// App lookup fails, so no match.
	if len(matched) != 0 {
		t.Errorf("matched %d, want 0", len(matched))
	}
}

// ==================== ParseMention tests ====================

func TestParseMention(t *testing.T) {
	tests := []struct {
		input   string
		handle  string
		command string
		text    string
	}{
		{"@echo-work hello", "echo-work", "", "hello"},
		{"@echo-work /echo hello world", "echo-work", "/echo", "hello world"},
		{"@echo-work /ECHO", "echo-work", "/echo", ""},
		{"@Echo-Work", "echo-work", "", ""},
		{"@github", "github", "", ""},
		{"@github /prs list", "github", "/prs", "list"},
		{"hello", "", "", ""},
		{"", "", "", ""},
		{"/echo hello", "", "", ""},
		{"@ hello", "", "", ""},
		{"  @echo hello  ", "echo", "", "hello"},
	}

	for _, tt := range tests {
		handle, command, text := ParseMention(tt.input)
		if handle != tt.handle || command != tt.command || text != tt.text {
			t.Errorf("ParseMention(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.input, handle, command, text, tt.handle, tt.command, tt.text)
		}
	}
}

// ==================== MatchHandle tests ====================

func TestMatchHandle_Success(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Handle: "echo-work", Enabled: true, AppWebhookURL: "http://a.com"},
			{ID: "i2", AppID: "a1", BotID: "b1", Handle: "echo-family", Enabled: true, AppWebhookURL: "http://b.com"},
		},
		apps: map[string]*store.App{"a1": {ID: "a1"}},
	}
	d := newMatchDispatcher(store)

	inst, err := d.MatchHandle("b1", "echo-work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst == nil || inst.ID != "i1" {
		t.Errorf("expected i1, got %v", inst)
	}
}

func TestMatchHandle_NotFound(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Handle: "echo-work", Enabled: true, AppWebhookURL: "http://a.com"},
		},
		apps: map[string]*store.App{"a1": {ID: "a1"}},
	}
	d := newMatchDispatcher(store)

	inst, _ := d.MatchHandle("b1", "nonexistent")
	if inst != nil {
		t.Errorf("expected nil, got %v", inst)
	}
}

func TestMatchHandle_Disabled(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Handle: "echo-work", Enabled: false, AppWebhookURL: "http://a.com"},
		},
		apps: map[string]*store.App{"a1": {ID: "a1"}},
	}
	d := newMatchDispatcher(store)

	inst, _ := d.MatchHandle("b1", "echo-work")
	if inst != nil {
		t.Errorf("expected nil for disabled, got %v", inst)
	}
}

func TestMatchHandle_NoRequestURL(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Handle: "echo-work", Enabled: true, AppWebhookURL: ""},
		},
		apps: map[string]*store.App{"a1": {ID: "a1"}},
	}
	d := newMatchDispatcher(store)

	inst, _ := d.MatchHandle("b1", "echo-work")
	if inst == nil {
		t.Error("installation without request_url should still be returned")
	}
}

func TestMatchEvent_MultipleInstallations(t *testing.T) {
	events1, _ := json.Marshal([]string{"message"})
	scopes1, _ := json.Marshal([]string{"message:read"})
	events2, _ := json.Marshal([]string{"reaction"})
	scopes2, _ := json.Marshal([]string{"message:read"})
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{ID: "i1", AppID: "a1", BotID: "b1", Enabled: true, AppWebhookURL: "http://a.com"},
			{ID: "i2", AppID: "a2", BotID: "b1", Enabled: true, AppWebhookURL: "http://b.com"},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1", Events: events1, Scopes: scopes1},
			"a2": {ID: "a2", Events: events2, Scopes: scopes2},
		},
	}

	d := newMatchDispatcher(store)
	matched, _ := d.MatchEvent("b1", "message.text")
	if len(matched) != 1 {
		t.Errorf("matched %d, want 1 (only app with message subscription)", len(matched))
	}
	if matched[0].ID != "i1" {
		t.Errorf("matched[0].ID = %q, want %q", matched[0].ID, "i1")
	}
}
