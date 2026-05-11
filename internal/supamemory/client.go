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

const defaultSystemPrompt = "你是一个温柔、真诚、尊重边界的 AI 伴侣，回答自然简洁，并保持积极支持。"

type Config struct {
	BaseURL        string
	ServiceRoleKey string
	Schema         string

	MemoryEnabled  bool
	MemoryTopK     int
	MemoryTable    string
	MemoryMatchRPC string

	BindingsTable string
	RoutesTable   string
	BotsTable     string
	ProfilesTable string

	EmbeddingModel string
}

type Client struct {
	baseURL string
	apiKey  string
	schema  string
	http    *http.Client

	memoryEnabled bool
	memoryTopK    int

	memoryTable    string
	memoryMatchRPC string
	bindingsTable  string
	routesTable    string
	botsTable      string
	profilesTable  string
	embeddingModel string
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
	RoleID     string  `json:"bot_id"`
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
	embeddingModel := strings.TrimSpace(cfg.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	return &Client{
		baseURL:        base,
		apiKey:         key,
		schema:         schema,
		http:           &http.Client{Timeout: 10 * time.Second},
		memoryEnabled:  cfg.MemoryEnabled,
		memoryTopK:     topK,
		memoryTable:    memoryTable,
		memoryMatchRPC: matchRPC,
		bindingsTable:  bindings,
		routesTable:    routes,
		botsTable:      bots,
		profilesTable:  profiles,
		embeddingModel: embeddingModel,
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

	prompt, err := c.buildPromptSnapshot(ctx, binding.ID, binding.UserID, route.RoleID)
	if err != nil {
		return &BindingContext{
			BindingID: binding.ID,
			UserID:    binding.UserID,
			RoleID:    route.RoleID,
		}, err
	}
	return &BindingContext{
		BindingID: binding.ID,
		UserID:    binding.UserID,
		RoleID:    route.RoleID,
		Prompt:    prompt,
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

	embeddingBase := strings.TrimSpace(opt.EmbeddingBase)
	embeddingKey := strings.TrimSpace(opt.EmbeddingKey)
	if embeddingBase != "" && embeddingKey != "" {
		if rows, err := c.vectorSearch(ctx, userID, roleID, query, topK, embeddingBase, embeddingKey, opt.EmbeddingModel, opt.CustomHeaders); err == nil && len(rows) > 0 {
			return rows, nil
		}
	}
	return c.fallbackSearch(ctx, userID, roleID, topK)
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
			buildPath(map[string]string{
				"external_account_id": "eq." + botProviderID,
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
	RoleID     string `json:"role_id"`
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
	return &rows[0], nil
}

type botPromptRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

type profileRow struct {
	Nickname      string `json:"nickname"`
	Persona       string `json:"persona"`
	Background    string `json:"background"`
	SpeakingStyle string `json:"speaking_style"`
	ReplyLanguage string `json:"reply_language"`
	Tone          string `json:"tone"`
	ResponseLen   string `json:"response_length"`
	UpdatedAt     string `json:"updated_at"`
}

func (c *Client) buildPromptSnapshot(ctx context.Context, bindingID, userID, roleID string) (*PromptSnapshot, error) {
	if userID == "" || roleID == "" {
		return nil, nil
	}
	bot, err := c.getRoleBot(ctx, roleID)
	if err != nil {
		return nil, err
	}
	profile, err := c.getRoleProfile(ctx, userID, roleID)
	if err != nil {
		return nil, err
	}

	roleNameRule := buildRoleNameRule(bot.Name)
	baseSystem := ensureSystemPrompt(bot.SystemPrompt)
	systemPrompt := baseSystem
	if roleNameRule != "" {
		systemPrompt = baseSystem + "\n\n" + roleNameRule
	}
	userPrompt := buildUserPrompt(profile)
	fullPrompt := systemPrompt
	if strings.TrimSpace(userPrompt) != "" {
		fullPrompt = systemPrompt + "\n\n" + userPrompt
	}
	sourceUpdatedAt := parseUnixSeconds(profile.UpdatedAt)
	if sourceUpdatedAt <= 0 {
		sourceUpdatedAt = time.Now().Unix()
	}
	return &PromptSnapshot{
		BindingID:       bindingID,
		UserID:          userID,
		RoleID:          roleID,
		SystemPrompt:    systemPrompt,
		UserPrompt:      userPrompt,
		FullPrompt:      fullPrompt,
		PromptVersion:   sourceUpdatedAt,
		SourceUpdatedAt: sourceUpdatedAt,
	}, nil
}

func (c *Client) getRoleBot(ctx context.Context, roleID string) (*botPromptRow, error) {
	q := url.Values{}
	q.Set("id", "eq."+roleID)
	q.Set("select", "id,name,system_prompt")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.botsTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []botPromptRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &botPromptRow{}, nil
	}
	return &rows[0], nil
}

func (c *Client) getRoleProfile(ctx context.Context, userID, roleID string) (*profileRow, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("role_id", "eq."+roleID)
	q.Set("select", "nickname,persona,background,speaking_style,reply_language,tone,response_length,updated_at")
	q.Set("limit", "1")
	path := "/rest/v1/" + url.PathEscape(c.profilesTable) + "?" + q.Encode()
	body, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	var rows []profileRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &profileRow{}, nil
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

func (c *Client) fallbackSearch(ctx context.Context, userID, roleID string, topK int) ([]MemoryRow, error) {
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("bot_id", "eq."+roleID)
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

func asLine(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return label + "：" + value
}

func buildUserPrompt(profile *profileRow) string {
	if profile == nil {
		return ""
	}
	lines := make([]string, 0, 7)
	if v := asLine("角色昵称", profile.Nickname); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("角色人设", profile.Persona); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("角色背景", profile.Background); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("说话风格", profile.SpeakingStyle); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("语气偏好", profile.Tone); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("回复语言", profile.ReplyLanguage); v != "" {
		lines = append(lines, v)
	}
	if v := asLine("回复长度", profile.ResponseLen); v != "" {
		lines = append(lines, v)
	}
	if len(lines) == 0 {
		return ""
	}
	return "用户角色配置（强约束，必须遵守）：\n" + strings.Join(lines, "\n") + "\n如果与普通聊天习惯冲突，以本配置优先。"
}

func ensureSystemPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return defaultSystemPrompt
	}
	return prompt
}

func buildRoleNameRule(roleName string) string {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return ""
	}
	return "角色身份约束：你的名字是「" + roleName + "」。当用户询问你叫什么名字时，直接回答「" + roleName + "」，不要回避。"
}

func parseUnixSeconds(ts string) int64 {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return 0
	}
	if v, err := strconv.ParseInt(ts, 10, 64); err == nil && v > 0 {
		return v
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.Unix()
	}
	return 0
}
