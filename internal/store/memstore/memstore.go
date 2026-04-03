// Package memstore provides an in-memory implementation of store.Store
// for use in the app mock server. Only methods used by the Bot API and
// event delivery paths have real implementations; the rest panic.
package memstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

var errNotImplemented = errors.New("not implemented in memstore")

// Store is an in-memory implementation of store.Store.
type Store struct {
	mu sync.RWMutex

	bots          map[string]*store.Bot
	apps          map[string]*store.App
	installations map[string]*store.AppInstallation // keyed by ID
	tokenIndex    map[string]string                 // app_token → installation ID
	handleIndex   map[string]string                 // "botID:handle" → installation ID

	messages  []store.Message
	msgSeq    atomic.Int64
	contacts  []store.RecentContact
	eventLogs []store.AppEventLog
	logSeq    atomic.Int64
	apiLogs   []store.AppAPILog
}

// Compile-time check that Store implements store.Store.
var _ store.Store = (*Store)(nil)

// New creates a new empty in-memory store.
func New() *Store {
	return &Store{
		bots:          make(map[string]*store.Bot),
		apps:          make(map[string]*store.App),
		installations: make(map[string]*store.AppInstallation),
		tokenIndex:    make(map[string]string),
		handleIndex:   make(map[string]string),
	}
}

// --- Setup helpers (not part of the interface) ---

// AddBot adds a bot to the store.
func (s *Store) AddBot(b *store.Bot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bots[b.ID] = b
}

// AddApp adds an app to the store.
func (s *Store) AddApp(a *store.App) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apps[a.ID] = a
}

// AddInstallation adds an installation and indexes it by token and handle.
func (s *Store) AddInstallation(inst *store.AppInstallation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.installations[inst.ID] = inst
	if inst.AppToken != "" {
		s.tokenIndex[inst.AppToken] = inst.ID
	}
	if inst.Handle != "" && inst.BotID != "" {
		s.handleIndex[inst.BotID+":"+inst.Handle] = inst.ID
	}
}

// AddContact adds a recent contact for GetContacts responses.
func (s *Store) AddContact(c store.RecentContact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contacts = append(s.contacts, c)
}

// GetSentMessages returns all outbound messages recorded by the store.
func (s *Store) GetSentMessages() []store.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []store.Message
	for _, m := range s.messages {
		if m.Direction == "outbound" {
			out = append(out, m)
		}
	}
	return out
}

// Reset clears all recorded messages and logs.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
	s.eventLogs = nil
	s.apiLogs = nil
}

// --- BotStore (implemented) ---

func (s *Store) GetBot(id string) (*store.Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.bots[id]
	if !ok {
		return nil, fmt.Errorf("bot %s not found", id)
	}
	copy := *b
	return &copy, nil
}

func (s *Store) GetAllBots() ([]store.Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []store.Bot
	for _, b := range s.bots {
		out = append(out, *b)
	}
	return out, nil
}

func (s *Store) UpdateBotStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.bots[id]; ok {
		b.Status = status
	}
	return nil
}

func (s *Store) IncrBotMsgCount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.bots[id]; ok {
		b.MsgCount++
	}
	return nil
}

func (s *Store) ListRecentContacts(botID string, limit int) ([]store.RecentContact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.contacts)
	if n > limit {
		n = limit
	}
	out := make([]store.RecentContact, n)
	copy(out, s.contacts[:n])
	return out, nil
}

func (s *Store) ListBotsByUser(string) ([]store.Bot, error) { return nil, nil }
func (s *Store) CreateBot(string, string, string, string, json.RawMessage) (*store.Bot, error) {
	return nil, errNotImplemented
}
func (s *Store) FindBotByProviderID(string, string) (*store.Bot, error) {
	return nil, errNotImplemented
}
func (s *Store) FindBotByCredential(string, string) (*store.Bot, error) {
	return nil, errNotImplemented
}
func (s *Store) UpdateBotCredentials(string, string, json.RawMessage) error { return nil }
func (s *Store) UpdateBotName(string, string) error                         { return nil }
func (s *Store) UpdateBotDisplayName(string, string) error                  { return nil }
func (s *Store) UpdateBotSyncState(string, json.RawMessage) error           { return nil }
func (s *Store) UpdateBotReminder(string, int) error                        { return nil }
func (s *Store) MarkBotReminded(string) error                               { return nil }
func (s *Store) GetBotsNeedingReminder() ([]store.Bot, error)               { return nil, nil }
func (s *Store) DeleteBot(string) error                                     { return nil }
func (s *Store) CountBotsByUser(string) (int, error)                        { return 0, nil }
func (s *Store) GetAdminStats() (*store.AdminStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := &store.AdminStats{
		TotalBots: len(s.bots),
	}
	for _, b := range s.bots {
		if b.Status == "connected" {
			stats.OnlineBots++
		} else if b.Status == "session_expired" {
			stats.ExpiredBots++
		}
	}
	for _, inst := range s.installations {
		if inst.Enabled {
			stats.TotalInstallations++
		}
	}
	for _, m := range s.messages {
		stats.TotalMessages++
		if m.Direction == "inbound" {
			stats.InboundMessages++
		} else {
			stats.OutboundMessages++
		}
	}
	return stats, nil
}

func (s *Store) GetBotStats(userID string) (*store.BotStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := &store.BotStats{}
	for _, b := range s.bots {
		if b.UserID == userID {
			stats.TotalBots++
			if b.Status == "connected" {
				stats.OnlineBots++
			}
			stats.TotalMessages += b.MsgCount
		}
	}
	for _, inst := range s.installations {
		if inst.Enabled {
			// Check if this installation belongs to the user's bot
			if b, ok := s.bots[inst.BotID]; ok && b.UserID == userID {
				stats.TotalInstallations++
			}
		}
	}
	return stats, nil
}
func (s *Store) UpdateBotAIEnabled(string, bool) error { return nil }
func (s *Store) UpdateBotAIModel(string, string) error { return nil }
func (s *Store) LastActivityAt(string) *time.Time      { return nil }

// --- MessageStore (implemented) ---

func (s *Store) SaveMessage(m *store.Message) (store.SaveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.msgSeq.Add(1)
	m.ID = id
	m.CreatedAt = time.Now().Unix()
	s.messages = append(s.messages, *m)
	return store.SaveResult{ID: id, Inserted: true}, nil
}

func (s *Store) GetLatestContextToken(botID string) string {
	// Always return a valid context token so send never fails due to missing token.
	return "mock-context-token"
}

func (s *Store) HasFreshContextToken(botID string, maxAge time.Duration) bool {
	return true
}

func (s *Store) BatchHasFreshContextToken(botIDs []string, maxAge time.Duration) map[string]bool {
	result := make(map[string]bool, len(botIDs))
	for _, id := range botIDs {
		result[id] = true
	}
	return result
}

func (s *Store) GetMessage(int64) (*store.Message, error)                          { return nil, errNotImplemented }
func (s *Store) ListMessages(string, int, int64) ([]store.Message, error)          { return nil, nil }
func (s *Store) ListMessagesBySender(string, string, int) ([]store.Message, error) { return nil, nil }
func (s *Store) ListChannelMessages(string, string, int) ([]store.Message, error)  { return nil, nil }
func (s *Store) GetMessagesSince(string, int64, int) ([]store.Message, error)      { return nil, nil }
func (s *Store) UpdateMediaStatus(string, string, json.RawMessage) error           { return nil }
func (s *Store) UpdateMediaStatusByID(int64, string, json.RawMessage) error        { return nil }
func (s *Store) UpdateMessagePayload(int64, json.RawMessage) error                 { return nil }
func (s *Store) UpdateMediaPayloads(string, string, json.RawMessage) error         { return nil }
func (s *Store) MarkProcessed(int64) error                                         { return nil }
func (s *Store) GetUnprocessedMessages(string, int) ([]store.Message, error)       { return nil, nil }
func (s *Store) PruneMessages(int) (int64, error)                                  { return 0, nil }

// --- AppStore (implemented) ---

func (s *Store) GetApp(id string) (*store.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.apps[id]
	if !ok {
		return nil, fmt.Errorf("app %s not found", id)
	}
	cp := *a
	return &cp, nil
}

func (s *Store) GetInstallationByToken(token string) (*store.AppInstallation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instID, ok := s.tokenIndex[token]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	inst, ok := s.installations[instID]
	if !ok {
		return nil, fmt.Errorf("installation not found")
	}
	cp := *inst
	return &cp, nil
}

func (s *Store) GetInstallationByHandle(botID, handle string) (*store.AppInstallation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instID, ok := s.handleIndex[botID+":"+handle]
	if !ok {
		return nil, fmt.Errorf("handle not found")
	}
	inst, ok := s.installations[instID]
	if !ok {
		return nil, fmt.Errorf("installation not found")
	}
	cp := *inst
	return &cp, nil
}

func (s *Store) ListInstallationsByBot(botID string) ([]store.AppInstallation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []store.AppInstallation
	for _, inst := range s.installations {
		if inst.BotID == botID {
			out = append(out, *inst)
		}
	}
	return out, nil
}

func (s *Store) UpdateAppTools(id string, tools json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.apps[id]; ok {
		a.Tools = tools
	}
	return nil
}

func (s *Store) UpdateInstallationTools(id string, tools json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inst, ok := s.installations[id]; ok {
		inst.Tools = tools
	}
	return nil
}

func (s *Store) GetInstallation(id string) (*store.AppInstallation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.installations[id]
	if !ok {
		return nil, fmt.Errorf("installation %s not found", id)
	}
	cp := *inst
	return &cp, nil
}

func (s *Store) CreateApp(*store.App) (*store.App, error)        { return nil, errNotImplemented }
func (s *Store) GetAppBySlug(string, string) (*store.App, error) { return nil, errNotImplemented }
func (s *Store) ListAppsByOwner(string) ([]store.App, error)     { return nil, nil }
func (s *Store) ListListedApps() ([]store.App, error)            { return nil, nil }
func (s *Store) ListAllApps() ([]store.App, error)               { return nil, nil }
func (s *Store) ListMarketplaceApps() ([]store.App, error)       { return nil, nil }
func (s *Store) UpdateApp(string, string, string, string, string, string, string, string, string, string, string, string, json.RawMessage, json.RawMessage, json.RawMessage) error {
	return nil
}
func (s *Store) UpdateMarketplaceApp(string, string, string, string, string, string, string, string, string, string, string, json.RawMessage, json.RawMessage, json.RawMessage) error {
	return nil
}
func (s *Store) DeleteApp(string) error { return nil }
func (s *Store) InstallApp(string, string) (*store.AppInstallation, error) {
	return nil, errNotImplemented
}
func (s *Store) InstalledAppIDs(string) (map[string]bool, error)                { return nil, nil }
func (s *Store) ListInstallationsByApp(string) ([]store.AppInstallation, error) { return nil, nil }
func (s *Store) UpdateInstallation(string, string, json.RawMessage, json.RawMessage, bool) error {
	return nil
}
func (s *Store) SetAppWebhookVerified(string, bool) error           { return nil }
func (s *Store) UpdateAppWebhookURL(string, string) error           { return nil }
func (s *Store) RegenerateInstallationToken(string) (string, error) { return "", errNotImplemented }
func (s *Store) DeleteInstallation(string) error                    { return nil }
func (s *Store) DeleteInstallationsByAppID(appID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, inst := range s.installations {
		if inst.AppID != appID {
			continue
		}
		delete(s.installations, id)
		if inst.AppToken != "" {
			delete(s.tokenIndex, inst.AppToken)
		}
		if inst.Handle != "" && inst.BotID != "" {
			delete(s.handleIndex, inst.BotID+":"+inst.Handle)
		}
	}
	return nil
}
func (s *Store) CreateOAuthCode(string, string, string, string, string) error { return nil }
func (s *Store) ExchangeOAuthCode(string) (string, string, string, error) {
	return "", "", "", errNotImplemented
}
func (s *Store) CleanExpiredOAuthCodes()                          {}
func (s *Store) RequestListing(string) error                      { return nil }
func (s *Store) ReviewListing(string, bool, string) error         { return nil }
func (s *Store) WithdrawListing(string) error                     { return nil }
func (s *Store) SetListing(string, string) error                  { return nil }
func (s *Store) CreateAppReview(*store.AppReview) error           { return nil }
func (s *Store) ListAppReviews(string) ([]store.AppReview, error) { return nil, nil }

// --- AppLogStore (implemented) ---

func (s *Store) CreateEventLog(log *store.AppEventLog) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.logSeq.Add(1)
	log.ID = id
	log.CreatedAt = time.Now().Unix()
	s.eventLogs = append(s.eventLogs, *log)
	return id, nil
}

func (s *Store) UpdateEventLogDelivered(int64, int, string, int) error  { return nil }
func (s *Store) UpdateEventLogFailed(int64, string, int, int) error     { return nil }
func (s *Store) ListEventLogs(string, int) ([]store.AppEventLog, error) { return nil, nil }
func (s *Store) CreateAPILog(log *store.AppAPILog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.CreatedAt = time.Now().Unix()
	s.apiLogs = append(s.apiLogs, *log)
	return nil
}
func (s *Store) ListAPILogs(string, int) ([]store.AppAPILog, error) { return nil, nil }
func (s *Store) CleanOldAppLogs(int) error                          { return nil }

// --- TraceStore (no-op) ---

func (s *Store) InsertSpan(string, string, string, string, string, string, string, int64, int64, []byte, []byte, string) error {
	return nil
}
func (s *Store) AppendSpan(string, string, string, string, string, string, map[string]any) error {
	return nil
}
func (s *Store) ListRootSpans(string, int) ([]store.TraceSpan, error) { return nil, nil }
func (s *Store) ListSpansByTrace(string) ([]store.TraceSpan, error)   { return nil, nil }

// --- UserStore (stub) ---

func (s *Store) CreateUser(string, string) (*store.User, error) { return nil, errNotImplemented }
func (s *Store) CreateUserFull(string, string, string, string, string) (*store.User, error) {
	return nil, errNotImplemented
}
func (s *Store) GetUserByID(id string) (*store.User, error) {
	return &store.User{ID: id, Username: "mock", Role: "admin", Status: store.StatusActive}, nil
}
func (s *Store) GetUserByUsername(string) (*store.User, error)  { return nil, errNotImplemented }
func (s *Store) GetUserByEmail(string) (*store.User, error)     { return nil, errNotImplemented }
func (s *Store) ListUsers() ([]store.User, error)               { return nil, nil }
func (s *Store) UserCount() (int, error)                        { return 1, nil }
func (s *Store) UpdateUserProfile(string, string, string) error { return nil }
func (s *Store) UpdateUserPassword(string, string) error        { return nil }
func (s *Store) UpdateUserRole(string, string) error            { return nil }
func (s *Store) UpdateUserStatus(string, string) error          { return nil }
func (s *Store) UpdateUserUsername(string, string) error        { return nil }
func (s *Store) DeleteUser(string) error                        { return nil }

// --- SessionStore (stub) ---

func (s *Store) CreateSession(string, string, time.Time) error { return nil }
func (s *Store) GetSession(string) (string, time.Time, error) {
	return "", time.Time{}, errNotImplemented
}
func (s *Store) DeleteSession(string) error          { return nil }
func (s *Store) DeleteExpiredSessions() error        { return nil }
func (s *Store) DeleteSessionsByUserID(string) error { return nil }

// --- ChannelStore (stub) ---

func (s *Store) CreateChannel(string, string, string, *store.FilterRule, *store.AIConfig) (*store.Channel, error) {
	return nil, errNotImplemented
}
func (s *Store) GetChannel(string) (*store.Channel, error)              { return nil, errNotImplemented }
func (s *Store) GetChannelByAPIKey(string) (*store.Channel, error)      { return nil, errNotImplemented }
func (s *Store) ListChannelsByBot(string) ([]store.Channel, error)      { return nil, nil }
func (s *Store) ListChannelsByBotIDs([]string) ([]store.Channel, error) { return nil, nil }
func (s *Store) UpdateChannel(string, string, string, *store.FilterRule, *store.AIConfig, *store.WebhookConfig, bool) error {
	return nil
}
func (s *Store) DeleteChannel(string) error               { return nil }
func (s *Store) RotateChannelKey(string) (string, error)  { return "", errNotImplemented }
func (s *Store) UpdateChannelLastSeq(string, int64) error { return nil }
func (s *Store) CountChannelsByBot(string) (int, error)   { return 0, nil }

// --- CredentialStore (stub) ---

func (s *Store) SaveCredential(*store.Credential) error                    { return nil }
func (s *Store) GetCredentialsByUserID(string) ([]store.Credential, error) { return nil, nil }
func (s *Store) UpdateCredentialSignCount(string, uint32) error            { return nil }
func (s *Store) UpdateCredentialName(string, string, string) (bool, error) { return false, nil }
func (s *Store) DeleteCredential(string, string) error                     { return nil }

// --- OAuthStore (stub) ---

func (s *Store) GetOAuthAccount(string, string) (*store.OAuthAccount, error) {
	return nil, errNotImplemented
}
func (s *Store) CreateOAuthAccount(*store.OAuthAccount) error                 { return nil }
func (s *Store) DeleteOAuthAccount(string, string) error                      { return nil }
func (s *Store) ListOAuthAccountsByUser(string) ([]store.OAuthAccount, error) { return nil, nil }

// --- ConfigStore (stub) ---

func (s *Store) GetConfig(string) (string, error)                     { return "", errNotImplemented }
func (s *Store) SetConfig(string, string) error                       { return nil }
func (s *Store) DeleteConfig(string) error                            { return nil }
func (s *Store) ListConfigByPrefix(string) (map[string]string, error) { return nil, nil }

// --- RegistryStore (stub) ---

func (s *Store) ListRegistries() ([]store.Registry, error) { return nil, nil }
func (s *Store) CreateRegistry(*store.Registry) error      { return nil }
func (s *Store) UpdateRegistryEnabled(string, bool) error  { return nil }
func (s *Store) DeleteRegistry(string) error               { return nil }

// --- PluginStore (stub) ---

func (s *Store) CreatePlugin(*store.Plugin) (*store.Plugin, error)           { return nil, errNotImplemented }
func (s *Store) GetPlugin(string) (*store.Plugin, error)                     { return nil, errNotImplemented }
func (s *Store) GetPluginByName(string) (*store.Plugin, error)               { return nil, errNotImplemented }
func (s *Store) ListPlugins() ([]store.PluginWithLatest, error)              { return nil, nil }
func (s *Store) ListPluginsByOwner(string) ([]store.PluginWithLatest, error) { return nil, nil }
func (s *Store) UpdatePluginMeta(string, *store.Plugin) error                { return nil }
func (s *Store) DeletePlugin(string) error                                   { return nil }
func (s *Store) CreatePluginVersion(*store.PluginVersion) (*store.PluginVersion, error) {
	return nil, errNotImplemented
}
func (s *Store) GetPluginVersion(string) (*store.PluginVersion, error)    { return nil, errNotImplemented }
func (s *Store) ListPluginVersions(string) ([]store.PluginVersion, error) { return nil, nil }
func (s *Store) ListPendingVersions() ([]store.PluginVersion, error)      { return nil, nil }
func (s *Store) SupersedeNonApprovedVersions(string)                      {}
func (s *Store) FindPendingVersion(string) (*store.PluginVersion, error) {
	return nil, errNotImplemented
}
func (s *Store) UpdatePluginVersion(string, *store.PluginVersion) error   { return nil }
func (s *Store) ReviewPluginVersion(string, string, string, string) error { return nil }
func (s *Store) DeletePluginVersion(string) error                         { return nil }
func (s *Store) RecordPluginInstall(string, string) error                 { return nil }
func (s *Store) CancelPluginVersion(string) error                         { return nil }
func (s *Store) FindPluginOwner(string) (string, error)                   { return "", errNotImplemented }
func (s *Store) ResolvePluginScript(string) (string, string, int, error) {
	return "", "", 0, errNotImplemented
}

// --- WebhookLogStore (stub) ---

func (s *Store) CreateWebhookLog(*store.WebhookLog) (int64, error)                   { return 0, nil }
func (s *Store) UpdateWebhookLogRequest(int64, string, string, string, string) error { return nil }
func (s *Store) UpdateWebhookLogResponse(int64, string, int, string, int) error      { return nil }
func (s *Store) UpdateWebhookLogResult(int64, string, string, []string) error        { return nil }
func (s *Store) UpdateWebhookLogPluginVersion(int64, string) error                   { return nil }
func (s *Store) ListWebhookLogs(string, string, int) ([]store.WebhookLog, error)     { return nil, nil }
func (s *Store) CleanOldWebhookLogs(int) error                                       { return nil }

// --- io.Closer ---

func (s *Store) Close() error { return nil }
