package sink

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
)

// Webhook pushes messages to a configured HTTP endpoint.
//
// Script API (two-phase middleware):
//
//	// ctx.msg  — inbound message (read-only)
//	// ctx.req  — {url, method, headers, body} (modify before send)
//	// ctx.res  — {status, headers, body} (available in onResponse)
//	// ctx.reply(text) — send reply back to user via bot
//	// ctx.skip()      — cancel this webhook delivery
//	//
//	// Export two optional functions:
//	function onRequest(ctx) {
//	    ctx.req.headers["X-Custom"] = "value";
//	    ctx.req.body = JSON.stringify({text: ctx.msg.content});
//	}
//	function onResponse(ctx) {
//	    var data = JSON.parse(ctx.res.body);
//	    if (data.answer) ctx.reply(data.answer);
//	}
type Webhook struct {
	DB *database.DB
}

func (s *Webhook) Name() string { return "webhook" }

// DebugResult holds the result of a script debug execution.
type DebugResult struct {
	Request  *reqData `json:"request"`            // modified request (nil if skipped)
	Response *resData `json:"response,omitempty"` // HTTP response
	Replies  []string `json:"replies"`            // reply() calls
	Skipped  bool     `json:"skipped"`            // skip() was called
	Error    string   `json:"error,omitempty"`    // script error
	Logs     []string `json:"logs"`               // execution trace
	Perms    DebugPerms `json:"permissions"`      // parsed permissions
}

type DebugPerms struct {
	Grants  []string `json:"grants"`
	Match   string   `json:"match"`
	Connect string   `json:"connect"`
}

// DebugRequest executes only the onRequest phase: parse + @match + onRequest.
// Returns the modified request for the frontend to send.
func DebugRequest(script string, mockMsg webhookPayload, webhookURL string) *DebugResult {
	result := &DebugResult{Logs: []string{}, Replies: []string{}}

	grants, matchTypes, connectDomains := parseScriptPerms(script)
	result.Perms = DebugPerms{
		Match:   joinKeys(matchTypes),
		Connect: joinKeys(connectDomains),
	}
	for k := range grants {
		result.Perms.Grants = append(result.Perms.Grants, k)
	}

	if !matchTypes["*"] && !matchTypes[mockMsg.MsgType] {
		result.Skipped = true
		result.Logs = append(result.Logs, fmt.Sprintf("@match 过滤：消息类型 %q 不匹配 %s，跳过", mockMsg.MsgType, joinKeys(matchTypes)))
		return result
	}
	result.Logs = append(result.Logs, "✓ @match 通过")

	body, _ := json.Marshal(mockMsg)
	if webhookURL == "" {
		webhookURL = "https://httpbin.org/post"
	}
	req := &reqData{URL: webhookURL, Method: "POST", Headers: map[string]string{"Content-Type": "application/json"}, Body: string(body)}

	// Run onRequest only (runScript also runs onResponse + HTTP, but we use a trick:
	// we call the script phases manually)
	vm, outReq, replies, skipped, err := runOnRequest(script, mockMsg, req)
	_ = vm // keep for onResponse later
	if err != nil {
		result.Error = err.Error()
		result.Logs = append(result.Logs, "✕ 脚本错误: "+err.Error())
		return result
	}
	result.Logs = append(result.Logs, "✓ onRequest 执行完成")
	result.Replies = replies
	result.Skipped = skipped

	if skipped {
		result.Logs = append(result.Logs, "⚠ skip() 被调用")
		return result
	}

	if !connectDomains["*"] && outReq != nil && outReq.URL != req.URL {
		if !isDomainAllowed(outReq.URL, connectDomains) {
			result.Error = fmt.Sprintf("@connect 拦截: %s 不在白名单", outReq.URL)
			result.Logs = append(result.Logs, "✕ "+result.Error)
			return result
		}
	}

	result.Request = outReq
	if outReq != nil {
		result.Logs = append(result.Logs, fmt.Sprintf("✓ 请求: %s %s (%d 字节)", outReq.Method, outReq.URL, len(outReq.Body)))
	}
	return result
}

// DebugResponse executes only the onResponse phase with the HTTP response from frontend.
func DebugResponse(script string, mockMsg webhookPayload, response *resData) *DebugResult {
	result := &DebugResult{Logs: []string{}, Replies: []string{}}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.SetMaxCallStackSize(scriptMaxCallStack)
	for _, name := range []string{"eval", "Function"} {
		vm.GlobalObject().Delete(name)
	}

	var replies []string
	vm.Set("ctx", map[string]any{"msg": mockMsg})
	vm.Set("reply", func(text string) {
		if len(replies) < scriptMaxReplies {
			replies = append(replies, text)
		}
	})
	vm.Set("skip", func() {})

	timer := time.AfterFunc(scriptTimeout, func() { vm.Interrupt("timeout") })
	_, err := vm.RunString(script)
	timer.Stop()
	vm.ClearInterrupt()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	if fn := vm.Get("onResponse"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			ctxObj := vm.Get("ctx").ToObject(vm)
			ctxObj.Set("res", map[string]any{
				"status":  response.Status,
				"headers": response.Headers,
				"body":    response.Body,
			})
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				result.Error = err.Error()
				result.Logs = append(result.Logs, "✕ onResponse 错误: "+err.Error())
				return result
			}
		}
	}

	result.Replies = replies
	result.Logs = append(result.Logs, "✓ onResponse 执行完成")
	if len(replies) > 0 {
		result.Logs = append(result.Logs, fmt.Sprintf("✓ reply() 调用 %d 次", len(replies)))
	}
	return result
}

// runOnRequest executes only the onRequest phase of a script.
func runOnRequest(script string, msg webhookPayload, req *reqData) (*goja.Runtime, *reqData, []string, bool, error) {
	grants, _, _ := parseScriptPerms(script)

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.SetMaxCallStackSize(scriptMaxCallStack)
	for _, name := range []string{"eval", "Function"} {
		vm.GlobalObject().Delete(name)
	}

	ctx := map[string]any{
		"msg": msg,
		"req": map[string]any{"url": req.URL, "method": req.Method, "headers": req.Headers, "body": req.Body},
	}
	vm.Set("ctx", ctx)

	var replies []string
	skipped := false

	if grants["reply"] || (len(grants) == 0 && !grants["none"]) {
		vm.Set("reply", func(text string) { if len(replies) < scriptMaxReplies { replies = append(replies, text) } })
	} else {
		vm.Set("reply", func(text string) { panic(vm.NewGoError(fmt.Errorf("reply() not granted"))) })
	}
	if grants["skip"] || (len(grants) == 0 && !grants["none"]) {
		vm.Set("skip", func() { skipped = true })
	} else {
		vm.Set("skip", func() { panic(vm.NewGoError(fmt.Errorf("skip() not granted"))) })
	}
	if grants["none"] {
		vm.Set("reply", func(text string) { panic(vm.NewGoError(fmt.Errorf("blocked by @grant none"))) })
		vm.Set("skip", func() { panic(vm.NewGoError(fmt.Errorf("blocked by @grant none"))) })
	}

	timer := time.AfterFunc(scriptTimeout, func() { vm.Interrupt("timeout") })
	_, err := vm.RunString(script)
	timer.Stop()
	vm.ClearInterrupt()
	if err != nil {
		return vm, nil, nil, false, err
	}

	if fn := vm.Get("onRequest"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				return vm, nil, nil, false, err
			}
		}
	}

	if skipped {
		return vm, nil, replies, true, nil
	}

	outReq := extractReqFromCtx(vm.Get("ctx").Export(), req)
	return vm, outReq, replies, false, nil
}

// MockPayload creates a test webhookPayload for debugging.
func MockPayload(sender, content, msgType string) webhookPayload {
	if sender == "" {
		sender = "test_user@debug"
	}
	if content == "" {
		content = "Hello from debug"
	}
	if msgType == "" {
		msgType = "text"
	}
	return webhookPayload{
		Event:     "message",
		ChannelID: "debug-channel",
		BotID:     "debug-bot",
		SeqID:     1,
		Sender:    sender,
		MsgType:   msgType,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		Items: []webhookItem{
			{Type: msgType, Text: content},
		},
	}
}

func joinKeys(m map[string]bool) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

func (s *Webhook) Handle(d Delivery) {
	cfg := d.Channel.WebhookConfig
	if cfg.URL == "" && cfg.PluginID == "" && cfg.Script == "" {
		return
	}
	start := time.Now()

	// Step 0: Create log entry
	var msgID *int64
	if d.SeqID > 0 {
		msgID = &d.SeqID
	}
	pluginVersion := ""
	logID, _ := s.DB.CreateWebhookLog(&database.WebhookLog{
		BotID: d.BotDBID, ChannelID: d.Channel.ID, MessageID: msgID, PluginID: cfg.PluginID, PluginVersion: pluginVersion,
	})

	msg := buildPayload(d)
	body, _ := json.Marshal(msg)
	req := &reqData{
		URL:     cfg.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(body),
	}
	applyAuth(req, cfg.Auth, body)

	var res *resData
	var replies []string
	skipped := false

	// Resolve script
	script := cfg.Script
	if cfg.PluginID != "" {
		plugin, err := s.DB.GetPlugin(cfg.PluginID)
		if err != nil {
			slog.Error("webhook plugin not found", "channel", d.Channel.ID, "plugin", cfg.PluginID, "err", err)
		} else if plugin.Status != "approved" {
			slog.Warn("webhook plugin not approved", "channel", d.Channel.ID, "plugin", cfg.PluginID, "status", plugin.Status)
		} else {
			script = plugin.Script
			pluginVersion = plugin.Version
		}
	}
	// Update log with resolved version
	if pluginVersion != "" {
		s.DB.Exec("UPDATE webhook_logs SET plugin_version = $1 WHERE id = $2", pluginVersion, logID)
	}

	// Step 1: Run script (onRequest + HTTP + onResponse)
	var forwarded bool
	if script != "" {
		var err error
		req, res, replies, skipped, forwarded, err = s.runScript(script, msg, req, d.Channel.ID)
		if err != nil {
			slog.Error("webhook script error", "channel", d.Channel.ID, "err", err)
			s.DB.UpdateWebhookLogResult(logID, "error", err.Error(), nil)
			return
		}
		if skipped || req == nil {
			s.DB.UpdateWebhookLogResult(logID, "skipped", "", replies)
			return
		}
	}

	// Step 2: Log request details
	s.DB.UpdateWebhookLogRequest(logID, "requesting", req.URL, req.Method, truncate(req.Body, 4096))

	// Step 3: Send HTTP
	if res == nil {
		res = doHTTP(req, d.Channel.ID)
	}

	duration := int(time.Since(start).Milliseconds())

	// Step 4: Log response
	if res != nil {
		status := "success"
		if res.Status >= 400 {
			status = "failed"
		}
		s.DB.UpdateWebhookLogResponse(logID, status, res.Status, truncate(res.Body, 4096), duration)
	} else {
		s.DB.UpdateWebhookLogResponse(logID, "failed", 0, "", duration)
	}

	// Auto-reply from response {"reply": "..."}
	if res != nil && len(replies) == 0 {
		var body struct{ Reply string }
		if json.Unmarshal([]byte(res.Body), &body) == nil && body.Reply != "" {
			replies = append(replies, body.Reply)
		}
	}

	// Step 5: Log final result with replies
	if len(replies) > 0 {
		s.DB.UpdateWebhookLogResult(logID, "success", "", replies)
	}

	// Step 6: Forward binary response as media if requested
	if forwarded && res != nil && res.RawBody != nil {
		go s.forwardMedia(d, res)
	}

	s.sendReplies(d, replies)
}

// forwardMedia sends the binary HTTP response as a media message through the bot.
func (s *Webhook) forwardMedia(d Delivery, res *resData) {
	ct := res.ContentType
	fileName := "response"
	switch {
	case strings.HasPrefix(ct, "image/"):
		ext := strings.TrimPrefix(ct, "image/")
		if i := strings.Index(ext, ";"); i > 0 { ext = ext[:i] }
		fileName = "image." + ext
	case strings.HasPrefix(ct, "audio/"):
		fileName = "audio.mp3"
	case strings.HasPrefix(ct, "video/"):
		fileName = "video.mp4"
	case strings.HasPrefix(ct, "application/pdf"):
		fileName = "document.pdf"
	default:
		fileName = "file.bin"
	}

	_, err := d.Provider.Send(context.Background(), provider.OutboundMessage{
		Recipient: d.Message.Sender,
		Data:      res.RawBody,
		FileName:  fileName,
	})
	if err != nil {
		slog.Error("webhook forward media failed", "channel", d.Channel.ID, "err", err)
		return
	}

	itemList, _ := json.Marshal([]map[string]any{{"type": "file", "file_name": fileName}})
	s.DB.SaveMessage(&database.Message{
		BotID:       d.BotDBID,
		Direction:   "outbound",
		ToUserID:    d.Message.Sender,
		MessageType: 2,
		ItemList:    itemList,
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

const (
	scriptTimeout      = 5 * time.Second
	scriptMaxCallStack = 64
	scriptMaxReplies   = 10
)

// runScriptWithTimeout runs a goja function with a timeout guard.
// If the script exceeds the deadline, the VM is interrupted.
func runScriptWithTimeout(vm *goja.Runtime, fn goja.Callable, args ...goja.Value) (goja.Value, error) {
	done := make(chan struct{})
	timer := time.AfterFunc(scriptTimeout, func() {
		vm.Interrupt("script execution timeout")
	})
	defer timer.Stop()

	var result goja.Value
	var err error
	go func() {
		defer close(done)
		result, err = fn(goja.Undefined(), args...)
	}()
	<-done

	// Clear interrupt for potential reuse (won't happen but good hygiene)
	vm.ClearInterrupt()
	return result, err
}

// parseScriptPerms extracts @grant, @match, @connect from the script metadata.
func parseScriptPerms(script string) (grants map[string]bool, matchTypes map[string]bool, connectDomains map[string]bool) {
	grants = map[string]bool{}
	matchTypes = map[string]bool{"*": true} // default: all
	connectDomains = map[string]bool{"*": true}

	metaRe := regexp.MustCompile(`//\s*@(\w+)\s+(.+)`)
	hasGrant, hasMatch, hasConnect := false, false, false

	for _, line := range strings.Split(script, "\n") {
		m := metaRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		switch m[1] {
		case "grant":
			if !hasGrant {
				hasGrant = true
				grants = map[string]bool{}
			}
			for _, g := range strings.Split(strings.TrimSpace(m[2]), ",") {
				grants[strings.TrimSpace(g)] = true
			}
		case "match":
			if !hasMatch {
				hasMatch = true
				matchTypes = map[string]bool{}
			}
			for _, t := range strings.Split(strings.TrimSpace(m[2]), ",") {
				matchTypes[strings.TrimSpace(t)] = true
			}
		case "connect":
			if !hasConnect {
				hasConnect = true
				connectDomains = map[string]bool{}
			}
			for _, d := range strings.Split(strings.TrimSpace(m[2]), ",") {
				connectDomains[strings.TrimSpace(d)] = true
			}
		}
	}
	return
}

func (s *Webhook) runScript(script string, msg webhookPayload, req *reqData, channelID string) (
	outReq *reqData, outRes *resData, replies []string, skipped bool, forwarded bool, err error,
) {
	// Parse permissions from script metadata
	grants, matchTypes, connectDomains := parseScriptPerms(script)

	// @match enforcement: skip if message type doesn't match
	if !matchTypes["*"] && !matchTypes[msg.MsgType] {
		return req, nil, nil, false, false, nil
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.SetMaxCallStackSize(scriptMaxCallStack)

	// Sandbox: remove potentially dangerous globals
	for _, name := range []string{"eval", "Function"} {
		vm.GlobalObject().Delete(name)
	}

	// Build ctx
	ctx := map[string]any{
		"msg": msg,
		"req": map[string]any{
			"url":     req.URL,
			"method":  req.Method,
			"headers": req.Headers,
			"body":    req.Body,
		},
	}
	vm.Set("ctx", ctx)

	// @grant enforcement: only expose granted APIs
	if grants["reply"] || grants["none"] == false && len(grants) == 0 {
		// grant reply or no @grant declared (backward compat: all allowed)
		vm.Set("reply", func(text string) {
			if len(replies) < scriptMaxReplies {
				replies = append(replies, text)
			}
		})
	} else {
		vm.Set("reply", func(text string) {
			panic(vm.NewGoError(fmt.Errorf("reply() not granted — add @grant reply")))
		})
	}

	if grants["skip"] || grants["none"] == false && len(grants) == 0 {
		vm.Set("skip", func() { skipped = true })
	} else {
		vm.Set("skip", func() {
			panic(vm.NewGoError(fmt.Errorf("skip() not granted — add @grant skip")))
		})
	}

	// If @grant none, block both
	if grants["none"] {
		vm.Set("reply", func(text string) {
			panic(vm.NewGoError(fmt.Errorf("reply() blocked by @grant none")))
		})
		vm.Set("skip", func() {
			panic(vm.NewGoError(fmt.Errorf("skip() blocked by @grant none")))
		})
	}

	// forward() — forward binary response as media to user
	if grants["forward"] || (len(grants) == 0 && !grants["none"]) {
		vm.Set("forward", func() { forwarded = true })
	} else {
		vm.Set("forward", func() {
			panic(vm.NewGoError(fmt.Errorf("forward() not granted — add @grant forward")))
		})
	}
	if grants["none"] {
		vm.Set("forward", func() {
			panic(vm.NewGoError(fmt.Errorf("forward() blocked by @grant none")))
		})
	}

	_ = connectDomains // used after script execution

	// Define onRequest/onResponse as top-level functions (with timeout)
	timer := time.AfterFunc(scriptTimeout, func() {
		vm.Interrupt("script parse timeout")
	})
	_, err = vm.RunString(script)
	timer.Stop()
	vm.ClearInterrupt()
	if err != nil {
		return nil, nil, nil, false, false, err
	}

	// Phase 1: onRequest
	if fn := vm.Get("onRequest"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				return nil, nil, nil, false, false, err
			}
		}
	}

	if skipped {
		return nil, nil, replies, true, false, nil
	}

	// Extract modified req from ctx
	outReq = extractReqFromCtx(vm.Get("ctx").Export(), req)

	// @connect enforcement: validate URL domain
	if !connectDomains["*"] && outReq.URL != req.URL {
		if !isDomainAllowed(outReq.URL, connectDomains) {
			return nil, nil, nil, false, false, fmt.Errorf("URL domain not in @connect whitelist: %s", outReq.URL)
		}
	}

	// Execute HTTP
	outRes = doHTTP(outReq, channelID)

	// Phase 2: onResponse
	if outRes != nil {
		if fn := vm.Get("onResponse"); fn != nil && !goja.IsUndefined(fn) {
			if callable, ok := goja.AssertFunction(fn); ok {
				ctxObj := vm.Get("ctx").ToObject(vm)
				resObj := map[string]any{
					"status":       outRes.Status,
					"headers":      outRes.Headers,
					"content_type": outRes.ContentType,
					"size":         outRes.Size,
				}
				if outRes.RawBody != nil {
					resObj["body"] = nil // binary — body not available in JS
				} else {
					resObj["body"] = outRes.Body
				}
				ctxObj.Set("res", resObj)
				if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
					slog.Error("webhook onResponse error", "channel", channelID, "err", err)
				}
			}
		}
	}

	return outReq, outRes, replies, false, forwarded, nil
}

func (s *Webhook) sendReplies(d Delivery, replies []string) {
	for _, text := range replies {
		if text == "" {
			continue
		}
		_, err := d.Provider.Send(context.Background(), provider.OutboundMessage{
			Recipient: d.Message.Sender,
			Text:      text,
		})
		if err != nil {
			slog.Error("webhook reply failed", "channel", d.Channel.ID, "err", err)
			continue
		}
		itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": text}})
		s.DB.SaveMessage(&database.Message{
			BotID:       d.BotDBID,
			Direction:   "outbound",
			ToUserID:    d.Message.Sender,
			MessageType: 2,
			ItemList:    itemList,
		})
	}
}

// --- Types ---

type reqData struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// ResData holds HTTP response data (exported for debug API).
type ResData = resData

type resData struct {
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	ContentType string            `json:"content_type,omitempty"`
	Size        int               `json:"size,omitempty"`
	RawBody     []byte            `json:"-"` // binary content, not serialized
}

type webhookPayload struct {
	Event     string        `json:"event"`
	ChannelID string        `json:"channel_id"`
	BotID     string        `json:"bot_id"`
	SeqID     int64         `json:"seq_id"`
	Sender    string        `json:"sender"`
	MsgType   string        `json:"msg_type"`
	Content   string        `json:"content"`
	Timestamp int64         `json:"timestamp"`
	Items     []webhookItem `json:"items"`
}

type webhookItem struct {
	Type     string `json:"type"`               // "text", "image", "voice", "file", "video"
	Text     string `json:"text,omitempty"`
	FileName string `json:"file_name,omitempty"`
	MediaURL string `json:"media_url,omitempty"` // download URL (MinIO or CDN proxy)
	FileSize int64  `json:"file_size,omitempty"`
	// Voice
	PlayTime int `json:"play_time,omitempty"`
	// Video
	PlayLength  int `json:"play_length,omitempty"`
	ThumbWidth  int `json:"thumb_width,omitempty"`
	ThumbHeight int `json:"thumb_height,omitempty"`
	// Quoted message
	RefTitle string       `json:"ref_title,omitempty"`
	RefItem  *webhookItem `json:"ref_item,omitempty"`
}

func buildPayload(d Delivery) webhookPayload {
	items := make([]webhookItem, len(d.Message.Items))
	for i, item := range d.Message.Items {
		items[i] = convertWebhookItem(item)
	}
	return webhookPayload{
		Event: "message", ChannelID: d.Channel.ID, BotID: d.BotDBID,
		SeqID: d.SeqID, Sender: d.Message.Sender, MsgType: d.MsgType,
		Content: d.Content, Timestamp: d.Message.Timestamp, Items: items,
	}
}

func convertWebhookItem(item provider.MessageItem) webhookItem {
	wi := webhookItem{
		Type:     item.Type,
		Text:     item.Text,
		FileName: item.FileName,
	}
	if item.Media != nil {
		wi.MediaURL = item.Media.URL
		wi.FileSize = item.Media.FileSize
		wi.PlayTime = item.Media.PlayTime
		wi.PlayLength = item.Media.PlayLength
		wi.ThumbWidth = item.Media.ThumbWidth
		wi.ThumbHeight = item.Media.ThumbHeight
	}
	if item.RefMsg != nil {
		wi.RefTitle = item.RefMsg.Title
		ref := convertWebhookItem(item.RefMsg.Item)
		wi.RefItem = &ref
	}
	return wi
}

// --- HTTP helpers ---

func applyAuth(req *reqData, auth *database.WebhookAuth, body []byte) {
	if auth == nil {
		return
	}
	switch auth.Type {
	case "bearer":
		req.Headers["Authorization"] = "Bearer " + auth.Token
	case "header":
		req.Headers[auth.Name] = auth.Value
	case "hmac":
		mac := hmac.New(sha256.New, []byte(auth.Secret))
		mac.Write(body)
		req.Headers["X-Hub-Signature"] = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}
}

func isBinaryContentType(ct string) bool {
	return strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "audio/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.HasPrefix(ct, "application/octet-stream") ||
		strings.HasPrefix(ct, "application/pdf")
}

func doHTTP(req *reqData, channelID string) *resData {
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewReader([]byte(req.Body)))
	if err != nil {
		slog.Error("webhook build failed", "channel", channelID, "err", err)
		return nil
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		slog.Error("webhook delivery failed", "channel", channelID, "err", err)
		return nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	if resp.StatusCode >= 400 {
		slog.Warn("webhook error status", "channel", channelID, "status", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	res := &resData{
		Status: resp.StatusCode, Headers: headers,
		ContentType: ct, Size: len(respBody),
	}
	if isBinaryContentType(ct) {
		res.RawBody = respBody // keep binary, body stays empty
	} else {
		res.Body = string(respBody)
	}
	return res
}

func extractReqFromCtx(obj any, fallback *reqData) *reqData {
	m, ok := obj.(map[string]any)
	if !ok {
		return fallback
	}
	rm, ok := m["req"].(map[string]any)
	if !ok {
		return fallback
	}
	out := &reqData{URL: fallback.URL, Method: fallback.Method, Headers: make(map[string]string), Body: fallback.Body}
	for k, v := range fallback.Headers {
		out.Headers[k] = v
	}
	if u, ok := rm["url"].(string); ok && u != "" {
		out.URL = u
	}
	if m, ok := rm["method"].(string); ok && m != "" {
		out.Method = m
	}
	if b, ok := rm["body"].(string); ok {
		out.Body = b
	}
	if h, ok := rm["headers"].(map[string]any); ok {
		for k, v := range h {
			if vs, ok := v.(string); ok {
				out.Headers[k] = vs
			}
		}
	}
	return out
}

// isDomainAllowed checks if a URL's host is in the allowed domains set.
func isDomainAllowed(rawURL string, allowed map[string]bool) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if allowed[host] {
		return true
	}
	// Check if any allowed domain is a suffix (e.g. ".feishu.cn" matches "open.feishu.cn")
	for domain := range allowed {
		if strings.HasSuffix(host, "."+domain) || host == domain {
			return true
		}
	}
	return false
}
