package sink

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/base64"
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
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
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
	Store store.Store
}

func (s *Webhook) Name() string { return "webhook" }

// DebugReply represents a reply action in debug output.
type DebugReply struct {
	Type     string `json:"type"`               // "text", "forward", "base64"
	Text     string `json:"text,omitempty"`     // for text replies
	Base64   string `json:"base64,omitempty"`   // for base64 (truncated for display)
	Filename string `json:"filename,omitempty"` // for base64
}

// DebugResult holds the result of a script debug execution.
type DebugResult struct {
	Request  *reqData     `json:"request"`
	Response *resData     `json:"response,omitempty"`
	Replies  []DebugReply `json:"replies"`
	Skipped  bool         `json:"skipped"`
	Error    string       `json:"error,omitempty"`
	Logs     []string     `json:"logs"`
	Perms    DebugPerms   `json:"permissions"`
}

type DebugPerms struct {
	Grants  []string `json:"grants"`
	Match   string   `json:"match"`
	Connect string   `json:"connect"`
}

func toDebugReplies(replies []string) []DebugReply {
	var out []DebugReply
	for _, r := range replies {
		out = append(out, DebugReply{Type: "text", Text: r})
	}
	return out
}

// createDebugVM creates a script VM with the same reply/skip/sandbox setup as runScript.
func createDebugVM(script string, msg webhookPayload, req *reqData) (*goja.Runtime, *[]replyAction, *bool, error) {
	grants, _, _ := parseScriptPerms(script)

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.SetMaxCallStackSize(scriptMaxCallStack)
	for _, name := range []string{"eval", "Function"} {
		vm.GlobalObject().Delete(name)
	}

	ctx := map[string]any{"msg": msg}
	if req != nil {
		ctx["req"] = map[string]any{"url": req.URL, "method": req.Method, "headers": req.Headers, "body": req.Body}
	}
	vm.Set("ctx", ctx)

	var replies []replyAction
	skipped := false

	canReply := grants["reply"] || (len(grants) == 0 && !grants["none"])
	if canReply {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value {
			if len(replies) >= scriptMaxReplies {
				return goja.Undefined()
			}
			arg := call.Argument(0)
			if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
				return goja.Undefined()
			}
			if s, ok := arg.Export().(string); ok {
				if strings.HasPrefix(s, "data:") {
					replies = append(replies, replyAction{Base64: s})
				} else {
					replies = append(replies, replyAction{Text: s})
				}
			} else if obj, ok := arg.Export().(map[string]any); ok {
				if fwd, _ := obj["forward"].(bool); fwd {
					replies = append(replies, replyAction{Forward: true})
				} else if b64, _ := obj["base64"].(string); b64 != "" {
					fn, _ := obj["filename"].(string)
					replies = append(replies, replyAction{Base64: b64, Filename: fn})
				}
			}
			return goja.Undefined()
		})
	} else {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value {
			panic(vm.NewGoError(fmt.Errorf("reply() not granted")))
		})
	}
	if grants["skip"] || (len(grants) == 0 && !grants["none"]) {
		vm.Set("skip", func() { skipped = true })
	} else {
		vm.Set("skip", func() { panic(vm.NewGoError(fmt.Errorf("skip() not granted"))) })
	}
	if grants["none"] {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value { panic(vm.NewGoError(fmt.Errorf("blocked by @grant none"))) })
		vm.Set("skip", func() { panic(vm.NewGoError(fmt.Errorf("blocked by @grant none"))) })
	}

	timer := time.AfterFunc(scriptTimeout, func() { vm.Interrupt("timeout") })
	_, err := vm.RunString(script)
	timer.Stop()
	vm.ClearInterrupt()

	return vm, &replies, &skipped, err
}

func replyActionsToDebug(replies []replyAction) []DebugReply {
	var out []DebugReply
	for _, r := range replies {
		switch {
		case r.Text != "":
			out = append(out, DebugReply{Type: "text", Text: r.Text})
		case r.Forward:
			out = append(out, DebugReply{Type: "forward"})
		case r.Base64 != "":
			preview := r.Base64
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			out = append(out, DebugReply{Type: "base64", Base64: preview, Filename: r.Filename})
		}
	}
	return out
}

// DebugRequest executes only the onRequest phase: parse + @match + onRequest.
func DebugRequest(script string, mockMsg webhookPayload, webhookURL string) *DebugResult {
	result := &DebugResult{Logs: []string{}, Replies: []DebugReply{}}

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

	vm, repliesPtr, skippedPtr, err := createDebugVM(script, mockMsg, req)
	if err != nil {
		result.Error = err.Error()
		result.Logs = append(result.Logs, "✕ 脚本错误: "+err.Error())
		return result
	}

	// Run onRequest
	if fn := vm.Get("onRequest"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				result.Error = err.Error()
				result.Logs = append(result.Logs, "✕ onRequest 错误: "+err.Error())
				return result
			}
		}
	}
	result.Logs = append(result.Logs, "✓ onRequest 执行完成")
	result.Replies = replyActionsToDebug(*repliesPtr)
	skipped := *skippedPtr
	result.Skipped = skipped

	if skipped {
		result.Logs = append(result.Logs, "⚠ skip() 被调用")
		return result
	}

	outReq := extractReqFromCtx(vm.Get("ctx").Export(), req)

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
	result := &DebugResult{Logs: []string{}, Replies: []DebugReply{}}

	vm, repliesPtr, _, err := createDebugVM(script, mockMsg, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	if fn := vm.Get("onResponse"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			ctxObj := vm.Get("ctx").ToObject(vm)
			resObj := map[string]any{
				"status":       response.Status,
				"headers":      response.Headers,
				"content_type": response.ContentType,
				"size":         response.Size,
			}
			if response.RawBody != nil {
				resObj["body"] = nil
			} else {
				resObj["body"] = response.Body
			}
			ctxObj.Set("res", resObj)
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				result.Error = err.Error()
				result.Logs = append(result.Logs, "✕ onResponse 错误: "+err.Error())
				return result
			}
		}
	}

	replies := *repliesPtr
	result.Replies = replyActionsToDebug(replies)
	result.Logs = append(result.Logs, "✓ onResponse 执行完成")
	if len(replies) > 0 {
		result.Logs = append(result.Logs, fmt.Sprintf("✓ reply() 调用 %d 次", len(replies)))
	}
	return result
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
	if cfg.URL == "" && cfg.VersionID == "" && cfg.Script == "" {
		return
	}
	start := time.Now()

	// Step 0: Create log entry
	var msgID *int64
	if d.SeqID > 0 {
		msgID = &d.SeqID
	}
	pluginVersion := ""
	logID, logErr := s.Store.CreateWebhookLog(&store.WebhookLog{
		BotID: d.BotDBID, ChannelID: d.Channel.ID, MessageID: msgID, PluginID: cfg.VersionID, PluginVersion: pluginVersion,
	})
	if logErr != nil {
		slog.Error("create webhook log failed", "channel", d.Channel.ID, "err", logErr)
	}

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
	var replies []replyAction
	skipped := false

	// Resolve script: PluginID is a version ID
	script := cfg.Script
	scriptTimeout := scriptTimeout // default 5s
	if cfg.VersionID != "" {
		resolvedScript, resolvedVersion, timeoutSec, err := s.Store.ResolvePluginScript(cfg.VersionID)
		if err != nil {
			slog.Error("webhook plugin resolve failed", "channel", d.Channel.ID, "version_id", cfg.VersionID, "err", err)
		} else {
			script = resolvedScript
			pluginVersion = resolvedVersion
			if timeoutSec > 0 {
				scriptTimeout = time.Duration(timeoutSec) * time.Second
			}
		}
	}
	if pluginVersion != "" {
		s.Store.UpdateWebhookLogPluginVersion(logID, pluginVersion)
	}

	// Step 1: Run script
	if script != "" {
		var err error
		req, res, replies, skipped, err = s.runScript(script, msg, req, d.Channel.ID, scriptTimeout)
		if err != nil {
			slog.Error("webhook script error", "channel", d.Channel.ID, "err", err)
			s.Store.UpdateWebhookLogResult(logID, "error", err.Error(), nil)
			return
		}
		if skipped || req == nil {
			replyTexts := extractReplyTexts(replies)
			s.Store.UpdateWebhookLogResult(logID, "skipped", "", replyTexts)
			return
		}
	}

	// Step 2: Log request
	s.Store.UpdateWebhookLogRequest(logID, "requesting", req.URL, req.Method, truncate(req.Body, 4096))

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
		s.Store.UpdateWebhookLogResponse(logID, status, res.Status, truncate(res.Body, 4096), duration)
	} else {
		s.Store.UpdateWebhookLogResponse(logID, "failed", 0, "", duration)
	}

	// Auto-reply from response {"reply": "..."}
	if res != nil && len(replies) == 0 {
		var body struct{ Reply string }
		if json.Unmarshal([]byte(res.Body), &body) == nil && body.Reply != "" {
			replies = append(replies, replyAction{Text: body.Reply})
		}
	}

	// Step 5: Log
	replyTexts := extractReplyTexts(replies)
	if len(replyTexts) > 0 {
		s.Store.UpdateWebhookLogResult(logID, "success", "", replyTexts)
	}

	// Step 6: Process all replies
	s.processReplies(d, res, replies)
}

// contentTypeToFileInfo maps Content-Type to (filename, itemType).
func contentTypeToFileInfo(ct string) (fileName, itemType string) {
	// Strip parameters (e.g. "image/png; charset=utf-8" → "image/png")
	if i := strings.Index(ct, ";"); i > 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	// Image types
	imageExts := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif",
		"image/webp": ".webp", "image/svg+xml": ".svg", "image/bmp": ".bmp",
		"image/tiff": ".tiff", "image/x-icon": ".ico",
	}
	if ext, ok := imageExts[ct]; ok {
		return "image" + ext, "image"
	}
	if strings.HasPrefix(ct, "image/") {
		return "image." + strings.TrimPrefix(ct, "image/"), "image"
	}

	// Audio types
	audioExts := map[string]string{
		"audio/mpeg": ".mp3", "audio/mp3": ".mp3", "audio/wav": ".wav",
		"audio/ogg": ".ogg", "audio/aac": ".aac", "audio/flac": ".flac",
		"audio/x-m4a": ".m4a", "audio/mp4": ".m4a",
	}
	if ext, ok := audioExts[ct]; ok {
		return "audio" + ext, "voice"
	}
	if strings.HasPrefix(ct, "audio/") {
		return "audio.mp3", "voice"
	}

	// Video types
	videoExts := map[string]string{
		"video/mp4": ".mp4", "video/webm": ".webm", "video/quicktime": ".mov",
		"video/x-msvideo": ".avi", "video/x-matroska": ".mkv",
	}
	if ext, ok := videoExts[ct]; ok {
		return "video" + ext, "video"
	}
	if strings.HasPrefix(ct, "video/") {
		return "video.mp4", "video"
	}

	// Document types
	docMap := map[string]string{
		"application/pdf":  ".pdf",
		"application/zip":  ".zip",
		"application/gzip": ".gz",
		"application/x-tar": ".tar",
		"application/x-rar-compressed": ".rar",
		"application/x-7z-compressed":  ".7z",
		"application/msword":           ".doc",
		"application/vnd.ms-excel":     ".xls",
		"application/vnd.ms-powerpoint": ".ppt",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   ".docx",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	}
	if ext, ok := docMap[ct]; ok {
		return "document" + ext, "file"
	}

	// Fallback
	return "file.bin", "file"
}

// forwardMedia sends the binary HTTP response as a media message through the bot.
func (s *Webhook) forwardMedia(d Delivery, res *resData) {
	fileName, itemType := contentTypeToFileInfo(res.ContentType)

	_, err := d.Provider.Send(context.Background(), provider.OutboundMessage{
		Recipient: d.Message.Sender,
		Data:      res.RawBody,
		FileName:  fileName,
	})
	if err != nil {
		slog.Error("webhook forward media failed", "channel", d.Channel.ID, "type", itemType, "err", err)
		return
	}

	itemList, _ := json.Marshal([]map[string]any{{"type": itemType, "file_name": fileName}})
	s.Store.SaveMessage(&store.Message{
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

// replyAction represents a single reply from a script.
type replyAction struct {
	Text     string // text reply
	Forward  bool   // forward binary response
	Base64   string // base64-encoded data
	Filename string // optional filename for base64
}

func (s *Webhook) runScript(script string, msg webhookPayload, req *reqData, channelID string, timeout ...time.Duration) (
	outReq *reqData, outRes *resData, replies []replyAction, skipped bool, err error,
) {
	// Resolve timeout
	execTimeout := scriptTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		execTimeout = timeout[0]
	}

	// Parse permissions from script metadata
	grants, matchTypes, connectDomains := parseScriptPerms(script)

	// @match enforcement: skip if message type doesn't match
	if !matchTypes["*"] && !matchTypes[msg.MsgType] {
		return req, nil, nil, false, nil
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

	// reply(arg) — unified reply function
	// - reply("text") → text reply
	// - reply("data:image/png;base64,...") → decode data URI and send as media
	// - reply({forward: true}) → forward binary HTTP response
	// - reply({base64: "...", filename: "img.png"}) → decode and send
	canReply := grants["reply"] || (len(grants) == 0 && !grants["none"])
	if canReply {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value {
			if len(replies) >= scriptMaxReplies {
				return goja.Undefined()
			}
			arg := call.Argument(0)
			if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
				return goja.Undefined()
			}

			// String argument
			if s, ok := arg.Export().(string); ok {
				if strings.HasPrefix(s, "data:") {
					// data URI — extract mime and base64
					replies = append(replies, replyAction{Base64: s})
				} else {
					replies = append(replies, replyAction{Text: s})
				}
				return goja.Undefined()
			}

			// Object argument
			if obj, ok := arg.Export().(map[string]any); ok {
				if fwd, _ := obj["forward"].(bool); fwd {
					replies = append(replies, replyAction{Forward: true})
				} else if b64, _ := obj["base64"].(string); b64 != "" {
					fn, _ := obj["filename"].(string)
					replies = append(replies, replyAction{Base64: b64, Filename: fn})
				}
			}
			return goja.Undefined()
		})
	} else {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value {
			panic(vm.NewGoError(fmt.Errorf("reply() not granted — add @grant reply")))
		})
	}

	if grants["skip"] || (len(grants) == 0 && !grants["none"]) {
		vm.Set("skip", func() { skipped = true })
	} else {
		vm.Set("skip", func() {
			panic(vm.NewGoError(fmt.Errorf("skip() not granted — add @grant skip")))
		})
	}

	if grants["none"] {
		vm.Set("reply", func(call goja.FunctionCall) goja.Value {
			panic(vm.NewGoError(fmt.Errorf("reply() blocked by @grant none")))
		})
		vm.Set("skip", func() {
			panic(vm.NewGoError(fmt.Errorf("skip() blocked by @grant none")))
		})
	}

	_ = connectDomains // used after script execution

	// Define onRequest/onResponse as top-level functions (with timeout)
	timer := time.AfterFunc(execTimeout, func() {
		vm.Interrupt("script parse timeout")
	})
	_, err = vm.RunString(script)
	timer.Stop()
	vm.ClearInterrupt()
	if err != nil {
		return nil, nil, nil, false, err
	}

	// Phase 1: onRequest
	if fn := vm.Get("onRequest"); fn != nil && !goja.IsUndefined(fn) {
		if callable, ok := goja.AssertFunction(fn); ok {
			if _, err := runScriptWithTimeout(vm, callable, vm.Get("ctx")); err != nil {
				return nil, nil, nil, false, err
			}
		}
	}

	if skipped {
		return nil, nil, replies, true, nil
	}

	// Extract modified req from ctx
	outReq = extractReqFromCtx(vm.Get("ctx").Export(), req)

	// @connect enforcement: validate URL domain
	if !connectDomains["*"] && outReq.URL != req.URL {
		if !isDomainAllowed(outReq.URL, connectDomains) {
			return nil, nil, nil, false, fmt.Errorf("URL domain not in @connect whitelist: %s", outReq.URL)
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

	return outReq, outRes, replies, false, nil
}

func extractReplyTexts(replies []replyAction) []string {
	var texts []string
	for _, r := range replies {
		if r.Text != "" {
			texts = append(texts, r.Text)
		} else if r.Forward {
			texts = append(texts, "[forward]")
		} else if r.Base64 != "" {
			texts = append(texts, "[base64]")
		}
	}
	return texts
}

// parseDataURI parses "data:image/png;base64,iVBOR..." into mime and raw bytes.
func parseDataURI(uri string) (mime string, data []byte, err error) {
	if !strings.HasPrefix(uri, "data:") {
		// Plain base64 without data URI prefix
		data, err = base64.StdEncoding.DecodeString(uri)
		return "", data, err
	}
	// data:image/png;base64,iVBOR...
	parts := strings.SplitN(uri[5:], ",", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid data URI")
	}
	meta := parts[0] // "image/png;base64"
	mime = strings.Split(meta, ";")[0]
	data, err = base64.StdEncoding.DecodeString(parts[1])
	return mime, data, err
}

func (s *Webhook) processReplies(d Delivery, res *resData, replies []replyAction) {
	for _, r := range replies {
		switch {
		case r.Text != "":
			// Text reply
			_, err := d.Provider.Send(context.Background(), provider.OutboundMessage{
				Recipient: d.Message.Sender, Text: r.Text,
			})
			if err != nil {
				slog.Error("webhook reply failed", "channel", d.Channel.ID, "err", err)
				continue
			}
			itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": r.Text}})
			s.Store.SaveMessage(&store.Message{
				BotID: d.BotDBID, Direction: "outbound", ToUserID: d.Message.Sender, MessageType: 2, ItemList: itemList,
			})

		case r.Forward:
			// Forward binary response
			if res != nil && res.RawBody != nil {
				s.forwardMedia(d, res)
			}

		case r.Base64 != "":
			// Decode base64/data URI and send as media
			mime, data, err := parseDataURI(r.Base64)
			if err != nil {
				slog.Error("webhook base64 decode failed", "channel", d.Channel.ID, "err", err)
				continue
			}
			fileName := r.Filename
			if fileName == "" {
				if mime != "" {
					fileName, _ = contentTypeToFileInfo(mime)
				} else {
					fileName = "file.bin"
				}
			}
			_, err = d.Provider.Send(context.Background(), provider.OutboundMessage{
				Recipient: d.Message.Sender, Data: data, FileName: fileName,
			})
			if err != nil {
				slog.Error("webhook base64 send failed", "channel", d.Channel.ID, "err", err)
				continue
			}
			itemType := "file"
			if mime != "" {
				_, itemType = contentTypeToFileInfo(mime)
			}
			itemList, _ := json.Marshal([]map[string]any{{"type": itemType, "file_name": fileName}})
			s.Store.SaveMessage(&store.Message{
				BotID: d.BotDBID, Direction: "outbound", ToUserID: d.Message.Sender, MessageType: 2, ItemList: itemList,
			})
		}
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

func applyAuth(req *reqData, auth *store.WebhookAuth, body []byte) {
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
	if strings.Contains(ct, ";") {
		ct = strings.TrimSpace(ct[:strings.Index(ct, ";")])
	}
	switch {
	case strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "video/"):
		return true
	}
	switch ct {
	case "application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",   // .docx
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",         // .xlsx
		"application/vnd.openxmlformats-officedocument.presentationml.presentation", // .pptx
		"application/msword",        // .doc
		"application/vnd.ms-excel",  // .xls
		"application/vnd.ms-powerpoint": // .ppt
		return true
	}
	return false
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
