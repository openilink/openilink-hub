package supamemory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL        string
	ServiceRoleKey string
	Schema         string

	MemoryEnabled  bool
	MemoryTopK     int
	MemoryTable    string
	MemoryMatchRPC string

	BindingsTable           string
	RoutesTable             string
	BotsTable               string
	ProfilesTable           string
	AuditLogsTable          string
	SubscriptionsTable      string
	PlanLimitsTable         string
	UsageCountersTable      string
	UsageEventsTable        string
	DictItemsTable          string
	PlatformMessagesTable   string
	ConversationStatesTable string
	DialogueEventsTable     string

	EmbeddingModel string
}

type Client struct {
	baseURL string
	apiKey  string
	schema  string
	http    *http.Client

	memoryEnabled bool
	memoryTopK    int

	memoryTable             string
	memoryMatchRPC          string
	bindingsTable           string
	routesTable             string
	botsTable               string
	profilesTable           string
	auditLogsTable          string
	subscriptionsTable      string
	planLimitsTable         string
	usageCountersTable      string
	usageEventsTable        string
	dictItemsTable          string
	platformMessagesTable   string
	conversationStatesTable string
	dialogueEventsTable     string
	embeddingModel          string
}

type BindingContext struct {
	BindingID string
	UserID    string
	RoleID    string
	Prompt    *PromptSnapshot
}

type PromptSnapshot struct {
	BindingID       string
	UserID          string
	RoleID          string
	SystemPrompt    string
	UserPrompt      string
	FullPrompt      string
	PromptVersion   int64
	SourceUpdatedAt int64
}

type MemoryRow struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	RoleID     flexID  `json:"bot_id"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	CreatedAt  string  `json:"created_at"`
	Similarity float64 `json:"similarity,omitempty"`
}

type SearchOptions struct {
	TopK           int
	EmbeddingBase  string
	EmbeddingKey   string
	EmbeddingModel string
	CustomHeaders  map[string]string
}

type RecordInput struct {
	UserID  string
	RoleID  string
	Content string
	Source  string
}

type PlatformMessageInput struct {
	UserID            string
	RoleID            string
	ConversationID    string
	Platform          string
	Direction         string
	Role              string
	Content           string
	ItemList          json.RawMessage
	ExternalEventID   string
	ProviderMessageID string
	ExternalChatID    string
	ExternalUserID    string
	ContextToken      string
	Raw               map[string]any
	Meta              map[string]any
	MessageAt         time.Time
}

type AuditLogInput struct {
	EventType string
	SessionID string
	TraceID   string
	Detail    map[string]any
}

type UsageEventInput struct {
	UserID      string
	PeriodMonth string
	Delta       int
	Source      string
	SessionID   string
	TraceID     string
	Detail      map[string]any
}

type UsageLedgerInput struct {
	UserID     string
	Delta      int
	Source     string
	SessionID  string
	TraceID    string
	Detail     map[string]any
	WriteEvent bool
}

type UsageBillingConfig struct {
	Enabled          bool
	TextCharsPerUnit int
}

type DialogueRuntimeFlags struct {
	PlannerOnly      bool
	GuardSoftMode    bool
	FallbackFastPath bool
}

type ConversationStateInput struct {
	ConversationID string
	Stage          string
	ActiveFlow     string
	StatePayload   map[string]any
	Version        int64
}

type ConversationStateRow struct {
	ConversationID string         `json:"conversation_id"`
	Stage          string         `json:"stage"`
	ActiveFlow     string         `json:"active_flow"`
	StatePayload   map[string]any `json:"state_payload"`
	Version        int64          `json:"version"`
}

type DialogueEventInput struct {
	ConversationID string
	TurnID         string
	EventID        string
	EventType      string
	IdempotencyKey string
	EventPayload   map[string]any
}

type EmojiAsset struct {
	URL  string
	Desc string
	Lang string
}

type QuotaStatus struct {
	Allowed      bool
	PlanCode     string
	PeriodMonth  string
	MonthlyLimit int
	Used         int
}

func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	key := strings.TrimSpace(cfg.ServiceRoleKey)
	if base == "" || key == "" {
		return nil, fmt.Errorf("supabase url/key required")
	}
	schema := strings.TrimSpace(cfg.Schema)
	if schema == "" {
		schema = "public"
	}
	topK := cfg.MemoryTopK
	if topK <= 0 {
		topK = 5
	}
	memoryTable := strings.TrimSpace(cfg.MemoryTable)
	if memoryTable == "" {
		memoryTable = "bl_memories"
	}
	matchRPC := strings.TrimSpace(cfg.MemoryMatchRPC)
	if matchRPC == "" {
		matchRPC = "match_memories"
	}
	bindings := strings.TrimSpace(cfg.BindingsTable)
	if bindings == "" {
		bindings = "bl_tool_bindings"
	}
	routes := strings.TrimSpace(cfg.RoutesTable)
	if routes == "" {
		routes = "bl_role_tool_routes"
	}
	bots := strings.TrimSpace(cfg.BotsTable)
	if bots == "" {
		bots = "bl_bots"
	}
	profiles := strings.TrimSpace(cfg.ProfilesTable)
	if profiles == "" {
		profiles = "bl_user_role_profiles"
	}
	auditLogs := strings.TrimSpace(cfg.AuditLogsTable)
	if auditLogs == "" {
		auditLogs = "bl_platform_audit_logs"
	}
	subscriptions := strings.TrimSpace(cfg.SubscriptionsTable)
	if subscriptions == "" {
		subscriptions = "bl_subscriptions"
	}
	planLimits := strings.TrimSpace(cfg.PlanLimitsTable)
	if planLimits == "" {
		planLimits = "bl_plan_limits"
	}
	usageCounters := strings.TrimSpace(cfg.UsageCountersTable)
	if usageCounters == "" {
		usageCounters = "bl_usage_counters"
	}
	usageEvents := strings.TrimSpace(cfg.UsageEventsTable)
	if usageEvents == "" {
		usageEvents = "bl_usage_events"
	}
	dictItems := strings.TrimSpace(cfg.DictItemsTable)
	if dictItems == "" {
		dictItems = "bl_dict_items"
	}
	platformMessages := strings.TrimSpace(cfg.PlatformMessagesTable)
	if platformMessages == "" {
		platformMessages = "bl_platform_messages"
	}
	conversationStates := strings.TrimSpace(cfg.ConversationStatesTable)
	if conversationStates == "" {
		conversationStates = "bl_conversation_states"
	}
	dialogueEvents := strings.TrimSpace(cfg.DialogueEventsTable)
	if dialogueEvents == "" {
		dialogueEvents = "bl_dialogue_events"
	}
	embeddingModel := strings.TrimSpace(cfg.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	return &Client{
		baseURL:                 base,
		apiKey:                  key,
		schema:                  schema,
		http:                    &http.Client{Timeout: 10 * time.Second},
		memoryEnabled:           cfg.MemoryEnabled,
		memoryTopK:              topK,
		memoryTable:             memoryTable,
		memoryMatchRPC:          matchRPC,
		bindingsTable:           bindings,
		routesTable:             routes,
		botsTable:               bots,
		profilesTable:           profiles,
		auditLogsTable:          auditLogs,
		subscriptionsTable:      subscriptions,
		planLimitsTable:         planLimits,
		usageCountersTable:      usageCounters,
		usageEventsTable:        usageEvents,
		dictItemsTable:          dictItems,
		platformMessagesTable:   platformMessages,
		conversationStatesTable: conversationStates,
		dialogueEventsTable:     dialogueEvents,
		embeddingModel:          embeddingModel,
	}, nil
}

func (c *Client) Enabled() bool {
	return c != nil
}

func (c *Client) ResolveBindingContext(ctx context.Context, botProviderID, senderUserID string) (*BindingContext, error) {
	if c == nil {
		return nil, nil
	}
	botProviderID = strings.TrimSpace(botProviderID)
	senderUserID = strings.TrimSpace(senderUserID)
	if senderUserID == "" {
		return nil, nil
	}

	binding, err := c.findBinding(ctx, botProviderID, senderUserID)
	if err != nil || binding == nil {
		return nil, err
	}
	route, err := c.findRoute(ctx, binding.UserID, binding.ID)
	if err != nil || route == nil {
		return &BindingContext{BindingID: binding.ID, UserID: binding.UserID}, err
	}

	return &BindingContext{
		BindingID: binding.ID,
		UserID:    binding.UserID,
		RoleID:    string(route.RoleID),
	}, nil
}

type effectivePromptRow struct {
	FullPrompt    string `json:"full_prompt"`
	SystemPrompt  string `json:"system_prompt"`
	UserPrompt    string `json:"user_prompt"`
	PromptVersion int64  `json:"prompt_version"`
}

func (c *Client) GetEffectiveFullPrompt(ctx context.Context, userID, roleID string) (*PromptSnapshot, error) {
	if c == nil {
		return nil, nil
	}
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)
	if userID == "" || roleID == "" {
		return nil, nil
	}
	body, _ := json.Marshal(map[string]any{
		"p_user_id": userID,
		"p_role_id": anyID(roleID),
	})
	respBody, err := c.do(ctx, http.MethodPost, "/rest/v1/rpc/get_effective_full_prompt", body, nil)
	if err != nil {
		return nil, err
	}

	var rows []effectivePromptRow
	if err := json.Unmarshal(respBody, &rows); err != nil {
		var single effectivePromptRow
		if errSingle := json.Unmarshal(respBody, &single); errSingle != nil {
			return nil, err
		}
		rows = []effectivePromptRow{single}
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &PromptSnapshot{
		UserID:        userID,
		RoleID:        roleID,
		SystemPrompt:  rows[0].SystemPrompt,
		UserPrompt:    rows[0].UserPrompt,
		FullPrompt:    rows[0].FullPrompt,
		PromptVersion: rows[0].PromptVersion,
	}, nil
}

func (c *Client) SearchMemories(ctx context.Context, userID, roleID, query string, opt SearchOptions) ([]MemoryRow, error) {
	if c == nil || !c.memoryEnabled {
		return nil, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	topK := opt.TopK
	if topK <= 0 {
		topK = c.memoryTopK
	}
	if topK <= 0 {
		topK = 5
	}
	retentionDays := c.memoryRetentionDaysForUser(ctx, userID)

	embeddingBase := strings.TrimSpace(opt.EmbeddingBase)
	embeddingKey := strings.TrimSpace(opt.EmbeddingKey)
	if embeddingBase != "" && embeddingKey != "" {
		if rows, err := c.vectorSearch(ctx, userID, roleID, query, topK, embeddingBase, embeddingKey, opt.EmbeddingModel, opt.CustomHeaders); err == nil && len(rows) > 0 {
			retained := filterMemoryRowsByRetention(rows, retentionDays, time.Now().UTC())
			if len(retained) > 0 {
				return retained, nil
			}
		}
	}
	return c.fallbackSearch(ctx, userID, roleID, topK, retentionDays)
}

func (c *Client) RecordMemory(ctx context.Context, in RecordInput) error {
	if c == nil || !c.memoryEnabled {
		return nil
	}
	content := strings.TrimSpace(in.Content)
	if content == "" || strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.RoleID) == "" {
		return nil
	}
	payload := map[string]any{
		"user_id": anyID(in.UserID),
		"bot_id":  anyID(in.RoleID),
		"content": content,
		"source":  fallbackStr(strings.TrimSpace(in.Source), "openilink_chat"),
	}
	body, _ := json.Marshal(payload)
	path := "/rest/v1/" + url.PathEscape(c.memoryTable) + "?select=id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "return=minimal",
	})
	return err
}

func (c *Client) WriteAuditLog(ctx context.Context, in AuditLogInput) error {
	if c == nil {
		return nil
	}
	eventType := strings.TrimSpace(in.EventType)
	if eventType == "" {
		return nil
	}
	payload := map[string]any{
		"event_type":  eventType,
		"session_id":  strings.TrimSpace(in.SessionID),
		"trace_id":    strings.TrimSpace(in.TraceID),
		"detail_json": in.Detail,
	}
	body, _ := json.Marshal(payload)
	path := "/rest/v1/" + url.PathEscape(c.auditLogsTable) + "?select=id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "return=minimal",
	})
	return err
}

func (c *Client) WritePlatformMessage(ctx context.Context, in PlatformMessageInput) error {
	if c == nil {
		return nil
	}
	userID := strings.TrimSpace(in.UserID)
	roleID := strings.TrimSpace(in.RoleID)
	direction := strings.TrimSpace(in.Direction)
	role := strings.TrimSpace(in.Role)
	if userID == "" || roleID == "" || direction == "" || role == "" {
		return nil
	}

	platform := strings.TrimSpace(in.Platform)
	if platform == "" {
		platform = "openilink"
	}

	itemList := in.ItemList
	if len(itemList) == 0 {
		itemList = json.RawMessage("[]")
	}

	payload := map[string]any{
		"user_id":             anyID(userID),
		"role_id":             anyID(roleID),
		"platform":            platform,
		"direction":           direction,
		"role":                role,
		"content":             strings.TrimSpace(in.Content),
		"item_list":           json.RawMessage(itemList),
		"external_event_id":   strings.TrimSpace(in.ExternalEventID),
		"provider_message_id": strings.TrimSpace(in.ProviderMessageID),
		"external_chat_id":    strings.TrimSpace(in.ExternalChatID),
		"external_user_id":    strings.TrimSpace(in.ExternalUserID),
		"context_token":       strings.TrimSpace(in.ContextToken),
		"meta":                in.Meta,
		"raw":                 in.Raw,
	}
	if payload["meta"] == nil {
		payload["meta"] = map[string]any{}
	}
	if payload["raw"] == nil {
		payload["raw"] = map[string]any{}
	}

	if conversationID := strings.TrimSpace(in.ConversationID); isUUID(conversationID) {
		payload["conversation_id"] = conversationID
	}
	if !in.MessageAt.IsZero() {
		payload["message_at"] = in.MessageAt.UTC().Format(time.RFC3339Nano)
	}

	body, _ := json.Marshal(payload)
	path := "/rest/v1/" + url.PathEscape(c.platformMessagesTable) + "?select=id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "return=minimal",
	})
	return err
}

func (c *Client) ResolveConversationID(ctx context.Context, userID, roleID, contextToken, sender string) (string, error) {
	if c == nil {
		return "", nil
	}
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)
	if userID == "" || roleID == "" {
		return "", nil
	}
	paths := make([]string, 0, 2)
	if token := strings.TrimSpace(contextToken); token != "" {
		q := url.Values{}
		q.Set("user_id", "eq."+userID)
		q.Set("role_id", "eq."+roleID)
		q.Set("context_token", "eq."+token)
		q.Set("conversation_id", "not.is.null")
		q.Set("select", "conversation_id")
		q.Set("order", "created_at.desc")
		q.Set("limit", "1")
		paths = append(paths, "/rest/v1/"+url.PathEscape(c.platformMessagesTable)+"?"+q.Encode())
	}
	if extUser := strings.TrimSpace(sender); extUser != "" {
		q := url.Values{}
		q.Set("user_id", "eq."+userID)
		q.Set("role_id", "eq."+roleID)
		q.Set("external_user_id", "eq."+extUser)
		q.Set("conversation_id", "not.is.null")
		q.Set("select", "conversation_id")
		q.Set("order", "created_at.desc")
		q.Set("limit", "1")
		paths = append(paths, "/rest/v1/"+url.PathEscape(c.platformMessagesTable)+"?"+q.Encode())
	}
	for _, path := range paths {
		body, err := c.do(ctx, http.MethodGet, path, nil, nil)
		if err != nil {
			return "", err
		}
		var rows []struct {
			ConversationID string `json:"conversation_id"`
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			return "", err
		}
		if len(rows) == 0 {
			continue
		}
		if id := strings.TrimSpace(rows[0].ConversationID); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func (c *Client) UpsertConversationState(ctx context.Context, in ConversationStateInput) error {
	if c == nil {
		return nil
	}
	conversationID := strings.TrimSpace(in.ConversationID)
	if conversationID == "" {
		return nil
	}
	stage := strings.TrimSpace(in.Stage)
	if stage == "" {
		stage = "idle"
	}
	activeFlow := strings.TrimSpace(in.ActiveFlow)
	if activeFlow == "" {
		activeFlow = "free_chat"
	}
	payload := in.StatePayload
	if payload == nil {
		payload = map[string]any{}
	}
	row := map[string]any{
		"conversation_id": conversationID,
		"stage":           stage,
		"active_flow":     activeFlow,
		"state_payload":   payload,
	}
	if in.Version >= 0 {
		row["version"] = in.Version
	}
	body, _ := json.Marshal(row)
	path := "/rest/v1/" + url.PathEscape(c.conversationStatesTable) + "?on_conflict=conversation_id&select=conversation_id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "resolution=merge-duplicates,return=minimal",
	})
	return err
}

func (c *Client) GetConversationState(ctx context.Context, conversationID string) (*ConversationStateRow, error) {
	if c == nil {
		return nil, nil
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("conversation_id", "eq."+conversationID)
	q.Set("select", "conversation_id,stage,active_flow,state_payload,version")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.conversationStatesTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ConversationID string         `json:"conversation_id"`
		Stage          string         `json:"stage"`
		ActiveFlow     string         `json:"active_flow"`
		StatePayload   map[string]any `json:"state_payload"`
		Version        any            `json:"version"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	version := int64(0)
	switch v := rows[0].Version.(type) {
	case float64:
		version = int64(v)
	case int64:
		version = v
	case int:
		version = int64(v)
	case string:
		n, parseErr := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if parseErr == nil {
			version = n
		}
	}
	return &ConversationStateRow{
		ConversationID: strings.TrimSpace(rows[0].ConversationID),
		Stage:          strings.TrimSpace(rows[0].Stage),
		ActiveFlow:     strings.TrimSpace(rows[0].ActiveFlow),
		StatePayload:   rows[0].StatePayload,
		Version:        version,
	}, nil
}

func (c *Client) AppendDialogueEvent(ctx context.Context, in DialogueEventInput) error {
	if c == nil {
		return nil
	}
	conversationID := strings.TrimSpace(in.ConversationID)
	eventType := strings.TrimSpace(in.EventType)
	idempotencyKey := strings.TrimSpace(in.IdempotencyKey)
	if conversationID == "" || eventType == "" || idempotencyKey == "" {
		return nil
	}
	turnID := strings.TrimSpace(in.TurnID)
	if turnID == "" {
		turnID = "turn_unknown"
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = idempotencyKey
	}
	payload := in.EventPayload
	if payload == nil {
		payload = map[string]any{}
	}
	row := map[string]any{
		"conversation_id": conversationID,
		"turn_id":         turnID,
		"event_id":        eventID,
		"event_type":      eventType,
		"idempotency_key": idempotencyKey,
		"event_payload":   payload,
	}
	body, _ := json.Marshal(row)
	path := "/rest/v1/" + url.PathEscape(c.dialogueEventsTable) + "?select=id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "return=minimal",
	})
	return err
}

type subscriptionRow struct {
	PlanCode string `json:"plan_code"`
}

type planLimitRow struct {
	MonthlyMessageLimit int            `json:"monthly_message_limit"`
	Features            map[string]any `json:"features"`
}

type usageCounterRow struct {
	MessageUsed int `json:"message_used"`
}

func (c *Client) CheckMonthlyQuota(ctx context.Context, userID string) (*QuotaStatus, error) {
	if c == nil {
		return nil, nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	planCode := "free"
	subscription, err := c.getSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if subscription != nil {
		if normalized := normalizePlanCode(subscription.PlanCode); normalized != "" {
			planCode = normalized
		}
	}

	limitRow, err := c.getPlanLimit(ctx, planCode)
	if err != nil {
		return nil, err
	}
	periodMonth := time.Now().UTC().Format("2006-01")
	usageRow, err := c.getUsageByMonth(ctx, userID, periodMonth)
	if err != nil {
		return nil, err
	}
	used := 0
	if usageRow != nil && usageRow.MessageUsed > 0 {
		used = usageRow.MessageUsed
	}

	limit := 0
	if limitRow != nil && limitRow.MonthlyMessageLimit > 0 {
		limit = limitRow.MonthlyMessageLimit
	}
	allowed := true
	if limit > 0 && used >= limit {
		allowed = false
	}
	return &QuotaStatus{
		Allowed:      allowed,
		PlanCode:     planCode,
		PeriodMonth:  periodMonth,
		MonthlyLimit: limit,
		Used:         used,
	}, nil
}

func (c *Client) BumpMonthlyUsage(ctx context.Context, userID string, delta int) error {
	if c == nil {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	if delta <= 0 {
		delta = 1
	}
	body, _ := json.Marshal(map[string]any{
		"p_user_id": anyID(userID),
		"p_delta":   delta,
	})
	_, err := c.do(ctx, http.MethodPost, "/rest/v1/rpc/bump_usage_counter", body, nil)
	return err
}

func (c *Client) BumpUsageLedger(ctx context.Context, in UsageLedgerInput) error {
	if c == nil {
		return nil
	}
	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return nil
	}
	delta := in.Delta
	if delta <= 0 {
		delta = 1
	}
	body, _ := json.Marshal(map[string]any{
		"p_user_id":     anyID(userID),
		"p_delta":       delta,
		"p_source":      strings.TrimSpace(in.Source),
		"p_session_id":  strings.TrimSpace(in.SessionID),
		"p_trace_id":    strings.TrimSpace(in.TraceID),
		"p_detail":      in.Detail,
		"p_write_event": in.WriteEvent,
	})
	_, err := c.do(ctx, http.MethodPost, "/rest/v1/rpc/bump_usage_ledger", body, nil)
	return err
}

func (c *Client) WriteUsageEvent(ctx context.Context, in UsageEventInput) error {
	if c == nil {
		return nil
	}
	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return nil
	}
	periodMonth := strings.TrimSpace(in.PeriodMonth)
	if periodMonth == "" {
		periodMonth = time.Now().UTC().Format("2006-01")
	}
	delta := in.Delta
	if delta <= 0 {
		delta = 1
	}
	payload := map[string]any{
		"user_id":      anyID(userID),
		"period_month": periodMonth,
		"delta":        delta,
		"source":       fallbackStr(strings.TrimSpace(in.Source), "openilink_hub_ai"),
		"session_id":   strings.TrimSpace(in.SessionID),
		"trace_id":     strings.TrimSpace(in.TraceID),
		"detail_json":  in.Detail,
	}
	body, _ := json.Marshal(payload)
	path := "/rest/v1/" + url.PathEscape(c.usageEventsTable) + "?select=id"
	_, err := c.do(ctx, http.MethodPost, path, body, map[string]string{
		"Prefer": "return=minimal",
	})
	return err
}

type dictItemRow struct {
	ItemValue string         `json:"item_value"`
	Ext       map[string]any `json:"ext"`
}

func (c *Client) GetUsageBillingConfig(ctx context.Context) (*UsageBillingConfig, error) {
	if c == nil {
		return nil, nil
	}
	q := url.Values{}
	q.Set("dict_code", "eq.system_runtime_flags")
	q.Set("item_code", "eq.usage_billing_v2")
	q.Set("is_active", "eq.true")
	q.Set("select", "item_value,ext")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.dictItemsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []dictItemRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	enabledRaw := strings.ToLower(strings.TrimSpace(rows[0].ItemValue))
	enabled := enabledRaw == "true" || enabledRaw == "1" || enabledRaw == "yes" || enabledRaw == "on"
	charsPerUnit := 180
	if rows[0].Ext != nil {
		if v, ok := rows[0].Ext["text_chars_per_unit"]; ok {
			if n := parseAnyPositiveInt(v); n > 0 {
				charsPerUnit = n
			}
		} else if v, ok := rows[0].Ext["chars_per_unit"]; ok {
			if n := parseAnyPositiveInt(v); n > 0 {
				charsPerUnit = n
			}
		}
	}
	return &UsageBillingConfig{
		Enabled:          enabled,
		TextCharsPerUnit: charsPerUnit,
	}, nil
}

func (c *Client) GetDialogueRuntimeFlags(ctx context.Context) (*DialogueRuntimeFlags, error) {
	if c == nil {
		return nil, nil
	}
	q := url.Values{}
	q.Set("dict_code", "eq.system_runtime_flags")
	q.Set("item_code", "in.(planner_only,guard_soft_mode,fallback_fast_path)")
	q.Set("is_active", "eq.true")
	q.Set("select", "item_code,item_value")
	path := "/rest/v1/" + url.PathEscape(c.dictItemsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ItemCode  string `json:"item_code"`
		ItemValue string `json:"item_value"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	flags := &DialogueRuntimeFlags{
		PlannerOnly:      false,
		GuardSoftMode:    true,
		FallbackFastPath: false,
	}
	for _, row := range rows {
		v := strings.ToLower(strings.TrimSpace(row.ItemValue))
		enabled := v == "true" || v == "1" || v == "yes" || v == "on"
		switch strings.TrimSpace(row.ItemCode) {
		case "planner_only":
			flags.PlannerOnly = enabled
		case "guard_soft_mode":
			flags.GuardSoftMode = enabled
		case "fallback_fast_path":
			flags.FallbackFastPath = enabled
		}
	}
	return flags, nil
}

func (c *Client) IsEmojiReplyEnabled(ctx context.Context, userID, roleID string) (bool, error) {
	if c == nil {
		return false, nil
	}
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)
	if userID == "" || roleID == "" {
		return false, nil
	}
	q := url.Values{}
	q.Set("id", "eq."+roleID)
	q.Set("user_id", "eq."+userID)
	q.Set("select", "emoji_reply_enabled")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.botsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return false, err
	}
	var rows []struct {
		EmojiReplyEnabled bool `json:"emoji_reply_enabled"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return rows[0].EmojiReplyEnabled, nil
}

func (c *Client) ListEmojiAssets(ctx context.Context) ([]EmojiAsset, error) {
	if c == nil {
		return nil, nil
	}
	q := url.Values{}
	q.Set("dict_code", "eq.emoji_assets")
	q.Set("is_active", "eq.true")
	q.Set("select", "item_name,item_value,ext")
	q.Set("order", "sort_order.asc,created_at.asc")
	path := "/rest/v1/" + url.PathEscape(c.dictItemsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ItemName  string         `json:"item_name"`
		ItemValue string         `json:"item_value"`
		Ext       map[string]any `json:"ext"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	out := make([]EmojiAsset, 0, len(rows))
	for _, row := range rows {
		urlValue := strings.TrimSpace(row.ItemValue)
		if urlValue == "" {
			continue
		}
		enabled := true
		if row.Ext != nil {
			if v, ok := row.Ext["enabled"]; ok {
				if b, okBool := v.(bool); okBool {
					enabled = b
				}
			}
		}
		if !enabled {
			continue
		}
		lang := "default"
		desc := strings.TrimSpace(row.ItemName)
		if row.Ext != nil {
			if v, ok := row.Ext["lang"].(string); ok && strings.TrimSpace(v) != "" {
				lang = strings.TrimSpace(v)
			}
			if v, ok := row.Ext["desc"].(string); ok && strings.TrimSpace(v) != "" {
				desc = strings.TrimSpace(v)
			}
		}
		if desc == "" {
			desc = "表情包"
		}
		out = append(out, EmojiAsset{
			URL:  urlValue,
			Desc: desc,
			Lang: lang,
		})
	}
	return out, nil
}

type bindingRow struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	ExternalAccount string `json:"external_account_id"`
	ExternalChatID  string `json:"external_chat_id"`
	BindingStatus   string `json:"binding_status"`
}

func (c *Client) findBinding(ctx context.Context, botProviderID, senderUserID string) (*bindingRow, error) {
	selects := "id,user_id,external_account_id,external_chat_id,binding_status,updated_at"
	buildPath := func(filters map[string]string) string {
		q := url.Values{}
		q.Set("select", selects)
		q.Set("order", "updated_at.desc")
		q.Set("limit", "1")
		q.Set("binding_status", "eq.active")
		for k, v := range filters {
			q.Set(k, v)
		}
		return "/rest/v1/" + url.PathEscape(c.bindingsTable) + "?" + q.Encode()
	}

	paths := []string{
		buildPath(map[string]string{
			"external_chat_id": "eq." + senderUserID,
		}),
	}
	if botProviderID != "" {
		paths = append([]string{
			buildPath(map[string]string{
				"external_account_id": "eq." + botProviderID,
				"external_chat_id":    "eq." + senderUserID,
			}),
			// Fallback for environments where bindings were written with sender as external_account_id.
			buildPath(map[string]string{
				"external_account_id": "eq." + senderUserID,
			}),
			buildPath(map[string]string{
				"external_account_id": "eq." + botProviderID,
			}),
		}, paths...)
	} else {
		// Same fallback even when provider bot id is unavailable.
		paths = append([]string{
			buildPath(map[string]string{
				"external_account_id": "eq." + senderUserID,
			}),
		}, paths...)
	}
	for _, p := range paths {
		body, err := c.do(ctx, http.MethodGet, p, nil, nil)
		if err != nil {
			return nil, err
		}
		var rows []bindingRow
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		if strings.TrimSpace(rows[0].ID) == "" || strings.TrimSpace(rows[0].UserID) == "" {
			continue
		}
		return &rows[0], nil
	}
	return nil, nil
}

type routeRow struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	RoleID     flexID `json:"role_id"`
	ToolBindID string `json:"tool_binding_id"`
}

func (c *Client) findRoute(ctx context.Context, userID, bindingID string) (*routeRow, error) {
	if userID == "" || bindingID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("select", "id,user_id,role_id,tool_binding_id")
	q.Set("user_id", "eq."+userID)
	q.Set("tool_binding_id", "eq."+bindingID)
	q.Set("route_status", "eq.active")
	q.Set("order", "priority.asc")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.routesTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []routeRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(string(rows[0].RoleID)) == "" {
		return nil, nil
	}
	return &rows[0], nil
}

// flexID accepts either JSON string IDs ("2002") or numeric IDs (2002).
type flexID string

func (id *flexID) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*id = ""
		return nil
	}
	if strings.HasPrefix(raw, "\"") {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*id = flexID(strings.TrimSpace(text))
		return nil
	}
	var num json.Number
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&num); err != nil {
		return err
	}
	*id = flexID(num.String())
	return nil
}

func (c *Client) getSubscription(ctx context.Context, userID string) (*subscriptionRow, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("select", "plan_code")
	q.Set("order", "updated_at.desc")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.subscriptionsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []subscriptionRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (c *Client) getPlanLimit(ctx context.Context, planCode string) (*planLimitRow, error) {
	planCode = normalizePlanCode(planCode)
	if planCode == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("plan_code", "eq."+planCode)
	q.Set("is_active", "eq.true")
	q.Set("select", "monthly_message_limit,features")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.planLimitsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []planLimitRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (c *Client) memoryRetentionDaysForUser(ctx context.Context, userID string) int {
	planCode := "free"
	subscription, err := c.getSubscription(ctx, strings.TrimSpace(userID))
	if err == nil && subscription != nil {
		if normalized := normalizePlanCode(subscription.PlanCode); normalized != "" {
			planCode = normalized
		}
	}
	limitRow, err := c.getPlanLimit(ctx, planCode)
	if err == nil && limitRow != nil {
		if days := memoryRetentionDaysFromFeatures(limitRow.Features); days >= 0 {
			return days
		}
	}
	return defaultMemoryRetentionDays(planCode)
}

func memoryRetentionDaysFromFeatures(features map[string]any) int {
	if features == nil {
		return -1
	}
	for _, key := range []string{"intelligence", "ai", "smartness"} {
		raw, ok := features[key]
		if !ok {
			continue
		}
		nested, ok := raw.(map[string]any)
		if !ok || nested == nil {
			continue
		}
		for _, field := range []string{"memory_retention_days", "memoryRetentionDays"} {
			if v, ok := nested[field]; ok {
				if days, okDays := parseAnyNonNegativeInt(v); okDays {
					return days
				}
			}
		}
	}
	return -1
}

func defaultMemoryRetentionDays(planCode string) int {
	switch normalizePlanCode(planCode) {
	case "m1", "y1", "pro":
		return 180
	case "m2", "y2":
		return 365
	case "y3", "ultra":
		return 0
	default:
		return 30
	}
}

func (c *Client) getUsageByMonth(ctx context.Context, userID, periodMonth string) (*usageCounterRow, error) {
	if userID == "" || periodMonth == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("period_month", "eq."+periodMonth)
	q.Set("select", "message_used")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.usageCountersTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []usageCounterRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (c *Client) vectorSearch(ctx context.Context, userID, roleID, query string, topK int, embeddingBase, embeddingKey, embeddingModel string, customHeaders map[string]string) ([]MemoryRow, error) {
	uid, okUID := parsePositiveInt(userID)
	rid, okRID := parsePositiveInt(roleID)
	if !okUID || !okRID {
		return nil, fmt.Errorf("non-numeric id for vector search")
	}
	vector, err := c.buildEmbedding(ctx, query, embeddingBase, embeddingKey, fallbackStr(strings.TrimSpace(embeddingModel), c.embeddingModel), customHeaders)
	if err != nil || len(vector) == 0 {
		return nil, err
	}
	payload := map[string]any{
		"p_user_id":         uid,
		"p_bot_id":          rid,
		"p_query_embedding": toPgVectorLiteral(vector),
		"p_match_count":     topK,
	}
	body, _ := json.Marshal(payload)
	path := "/rest/v1/rpc/" + url.PathEscape(c.memoryMatchRPC)
	respBody, err := c.do(ctx, http.MethodPost, path, body, nil)
	if err != nil {
		return nil, err
	}

	rows, err := decodeMemoryRows(respBody)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (c *Client) fallbackSearch(ctx context.Context, userID, roleID string, topK int, retentionDays int) ([]MemoryRow, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("bot_id", "eq."+roleID)
	if retentionDays > 0 {
		q.Set("created_at", "gte."+time.Now().UTC().Add(-time.Duration(retentionDays)*24*time.Hour).Format(time.RFC3339Nano))
	}
	q.Set("select", "id,user_id,bot_id,content,source,created_at")
	q.Set("order", "created_at.desc")
	q.Set("limit", strconv.Itoa(topK))
	path := "/rest/v1/" + url.PathEscape(c.memoryTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeMemoryRows(body)
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (c *Client) buildEmbedding(ctx context.Context, text, baseURL, apiKey, model string, customHeaders map[string]string) ([]float64, error) {
	body, _ := json.Marshal(map[string]any{
		"model": model,
		"input": strings.TrimSpace(text),
	})
	endpoint := strings.TrimRight(baseURL, "/") + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range customHeaders {
		kl := strings.ToLower(strings.TrimSpace(k))
		if kl == "authorization" || kl == "content-type" || kl == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding request failed: %d", resp.StatusCode)
	}
	var parsed embeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	return parsed.Data[0].Embedding, nil
}

func decodeMemoryRows(body []byte) ([]MemoryRow, error) {
	var arr []MemoryRow
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}
	var single MemoryRow
	if err := json.Unmarshal(body, &single); err == nil && (single.ID != "" || single.Content != "") {
		return []MemoryRow{single}, nil
	}
	return nil, fmt.Errorf("decode memories failed")
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept-Profile", c.schema)
	req.Header.Set("Content-Profile", c.schema)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("supabase request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func parsePositiveInt(input string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(input), 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func isUUID(v string) bool {
	s := strings.TrimSpace(v)
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			ch := s[i]
			isDigit := ch >= '0' && ch <= '9'
			isLowerHex := ch >= 'a' && ch <= 'f'
			isUpperHex := ch >= 'A' && ch <= 'F'
			if !isDigit && !isLowerHex && !isUpperHex {
				return false
			}
		}
	}
	return true
}

func parseAnyPositiveInt(v any) int {
	switch val := v.(type) {
	case float64:
		if val > 0 {
			return int(val)
		}
	case int:
		if val > 0 {
			return val
		}
	case int64:
		if val > 0 {
			return int(val)
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func parseAnyNonNegativeInt(v any) (int, bool) {
	switch val := v.(type) {
	case float64:
		if val >= 0 {
			return int(val), true
		}
	case int:
		if val >= 0 {
			return val, true
		}
	case int64:
		if val >= 0 {
			return int(val), true
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil && n >= 0 {
			return n, true
		}
	}
	return 0, false
}

func filterMemoryRowsByRetention(rows []MemoryRow, retentionDays int, now time.Time) []MemoryRow {
	if retentionDays <= 0 || len(rows) == 0 {
		return rows
	}
	cutoff := now.UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	out := make([]MemoryRow, 0, len(rows))
	for _, row := range rows {
		createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(row.CreatedAt))
		if err != nil {
			createdAt, err = time.Parse(time.RFC3339, strings.TrimSpace(row.CreatedAt))
		}
		if err != nil {
			continue
		}
		if !createdAt.Before(cutoff) {
			out = append(out, row)
		}
	}
	return out
}

func anyID(input string) any {
	if n, ok := parsePositiveInt(input); ok {
		return n
	}
	return input
}

func toPgVectorLiteral(v []float64) string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(v))
	for _, x := range v {
		parts = append(parts, strconv.FormatFloat(x, 'f', 8, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func fallbackStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func normalizePlanCode(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if code == "" {
		return ""
	}
	switch code {
	case "free", "m1", "m2", "y1", "y2", "y3", "pro", "ultra":
		return code
	default:
		return "free"
	}
}
