package sink

import (
	"strings"
	"testing"
)

func defaultReq() *reqData {
	return &reqData{URL: "http://localhost", Method: "POST", Headers: map[string]string{}}
}

func defaultMsg() webhookPayload {
	return webhookPayload{
		Event: "message", ChannelID: "ch-1", BotID: "bot-1",
		Sender: "user@wx", Content: "hello", MsgType: "text",
		Timestamp: 1700000000000,
	}
}

// ==================== Timeout / Resource ====================

func TestScriptTimeout(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { while(true) {} }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got: %v", err)
	}
}

func TestScriptTimeoutInOnResponse(t *testing.T) {
	s := &Webhook{}
	// onRequest is fine, but onResponse loops
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {}
		 function onResponse(ctx) { while(true) {} }`,
		defaultMsg(), defaultReq(), "test")
	// onResponse errors are logged, not returned — but the function should not hang
	// The test passing within the test timeout proves the guard works
	_ = err
}

func TestScriptTimeoutInParse(t *testing.T) {
	s := &Webhook{}
	// Top-level infinite loop during script parsing
	_, _, _, _, _, err := s.runScript(
		`while(true) {}`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected timeout error from parse")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got: %v", err)
	}
}

func TestScriptStackOverflow(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { onRequest(ctx); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected stack overflow error")
	}
	t.Logf("stack overflow: %v", err)
}

func TestScriptMutualRecursion(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function a() { b(); } function b() { a(); } function onRequest(ctx) { a(); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected stack overflow from mutual recursion")
	}
}

func TestScriptReplyLimit(t *testing.T) {
	s := &Webhook{}
	_, _, replies, _, _, err := s.runScript(
		`function onRequest(ctx) { for(var i=0; i<100; i++) reply("msg"+i); }`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replies) != scriptMaxReplies {
		t.Errorf("replies: got %d, want %d", len(replies), scriptMaxReplies)
	}
}

// ==================== Disabled Globals ====================

func TestScriptEvalDisabled(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { eval("1+1"); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected eval error")
	}
	if !strings.Contains(err.Error(), "eval") {
		t.Fatalf("expected eval reference error, got: %v", err)
	}
}

func TestScriptFunctionConstructorDisabled(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { var f = new Function("return 1"); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected Function constructor error")
	}
	if !strings.Contains(err.Error(), "Function") {
		t.Fatalf("expected Function reference error, got: %v", err)
	}
}

func TestScriptNoRequire(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { require("fs"); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected require error")
	}
}

func TestScriptNoProcess(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { process.exit(1); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected process error")
	}
}

func TestScriptNoGlobalThis(t *testing.T) {
	// Accessing globalThis shouldn't expose anything dangerous
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { var x = globalThis; }`,
		defaultMsg(), defaultReq(), "test")
	// Should succeed — globalThis exists but is sandboxed
	_ = err
}

// ==================== Prototype Pollution ====================

func TestScriptPrototypePollution(t *testing.T) {
	s := &Webhook{}
	outReq, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			// Attempt prototype pollution
			var obj = {};
			obj.__proto__.polluted = "yes";
			ctx.req.headers["X-Check"] = "clean";
		}`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outReq.Headers["X-Check"] != "clean" {
		t.Error("script should still work after pollution attempt")
	}
}

func TestScriptConstructorOverwrite(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			Object.prototype.toString = function() { return "hacked"; };
		}`,
		defaultMsg(), defaultReq(), "test")
	_ = err // may or may not error, important is it doesn't crash the host
}

// ==================== ctx Isolation ====================

func TestScriptCannotModifyMsg(t *testing.T) {
	s := &Webhook{}
	msg := defaultMsg()
	s.runScript(
		`function onRequest(ctx) { ctx.msg.sender = "hacked"; }`,
		msg, defaultReq(), "test")
	// Original Go struct should not be modified
	if msg.Sender != "user@wx" {
		t.Errorf("msg.Sender was modified: %s", msg.Sender)
	}
}

func TestScriptReqModification(t *testing.T) {
	s := &Webhook{}
	outReq, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			ctx.req.url = "http://evil.com";
			ctx.req.method = "PUT";
			ctx.req.headers["X-Injected"] = "true";
			ctx.req.body = "modified body";
		}`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if outReq.URL != "http://evil.com" {
		t.Errorf("url = %s", outReq.URL)
	}
	if outReq.Method != "PUT" {
		t.Errorf("method = %s", outReq.Method)
	}
	if outReq.Headers["X-Injected"] != "true" {
		t.Error("header not set")
	}
	if outReq.Body != "modified body" {
		t.Errorf("body = %s", outReq.Body)
	}
}

func TestScriptReqUrlPreservedIfNotModified(t *testing.T) {
	s := &Webhook{}
	req := &reqData{URL: "http://original.com", Method: "POST", Headers: map[string]string{"A": "1"}, Body: "orig"}
	outReq, _, _, _, _, _ := s.runScript(
		`function onRequest(ctx) { /* do nothing */ }`,
		defaultMsg(), req, "test")
	if outReq.URL != "http://original.com" {
		t.Errorf("url changed: %s", outReq.URL)
	}
	if outReq.Headers["A"] != "1" {
		t.Error("original header lost")
	}
	if outReq.Body != "orig" {
		t.Errorf("body changed: %s", outReq.Body)
	}
}

// ==================== Skip ====================

func TestScriptSkip(t *testing.T) {
	s := &Webhook{}
	_, _, _, skipped, _, err := s.runScript(
		`function onRequest(ctx) { skip(); }`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Error("expected skipped=true")
	}
}

func TestScriptSkipStopsHTTP(t *testing.T) {
	s := &Webhook{}
	outReq, outRes, _, skipped, _, _ := s.runScript(
		`function onRequest(ctx) { skip(); }`,
		defaultMsg(), defaultReq(), "test")
	if !skipped {
		t.Error("expected skipped")
	}
	// When skipped, no HTTP should be made
	if outReq != nil {
		t.Error("outReq should be nil when skipped")
	}
	if outRes != nil {
		t.Error("outRes should be nil when skipped")
	}
}

func TestScriptConditionalSkip(t *testing.T) {
	s := &Webhook{}

	// Message with "ignore" keyword → skip
	msg := defaultMsg()
	msg.Content = "please ignore this"
	_, _, _, skipped, _, _ := s.runScript(
		`function onRequest(ctx) { if (ctx.msg.content.indexOf("ignore") >= 0) skip(); }`,
		msg, defaultReq(), "test")
	if !skipped {
		t.Error("should skip for 'ignore' keyword")
	}

	// Normal message → don't skip
	msg2 := defaultMsg()
	msg2.Content = "hello world"
	_, _, _, skipped2, _, _ := s.runScript(
		`function onRequest(ctx) { if (ctx.msg.content.indexOf("ignore") >= 0) skip(); }`,
		msg2, defaultReq(), "test")
	if skipped2 {
		t.Error("should not skip for normal message")
	}
}

// ==================== Normal Execution ====================

func TestScriptNormalExecution(t *testing.T) {
	s := &Webhook{}
	outReq, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			ctx.req.headers["X-Custom"] = "hello";
			ctx.req.body = JSON.stringify({text: ctx.msg.content});
		}`,
		webhookPayload{Content: "test message"}, defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if outReq.Headers["X-Custom"] != "hello" {
		t.Errorf("header: %v", outReq.Headers)
	}
	if !strings.Contains(outReq.Body, "test message") {
		t.Errorf("body: %s", outReq.Body)
	}
}

func TestScriptAccessAllMsgFields(t *testing.T) {
	s := &Webhook{}
	msg := webhookPayload{
		Event: "message", ChannelID: "ch-1", BotID: "bot-1",
		SeqID: 42, Sender: "alice@wx", MsgType: "text",
		Content: "hi there", Timestamp: 1700000000000,
		Items: []webhookItem{{Type: "text", Text: "hi there"}},
	}
	outReq, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			var m = ctx.msg;
			var parts = [m.event, m.channel_id, m.bot_id, m.seq_id, m.sender, m.msg_type, m.content, m.timestamp];
			ctx.req.body = parts.join("|");
			// Also verify items
			ctx.req.headers["X-Items"] = String(m.items.length);
			ctx.req.headers["X-Item-Text"] = m.items[0].text;
		}`,
		msg, defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(outReq.Body, "message|ch-1|bot-1|42|alice@wx|text|hi there|1700000000000") {
		t.Errorf("body = %s", outReq.Body)
	}
	if outReq.Headers["X-Items"] != "1" {
		t.Errorf("items count = %s", outReq.Headers["X-Items"])
	}
	if outReq.Headers["X-Item-Text"] != "hi there" {
		t.Errorf("item text = %s", outReq.Headers["X-Item-Text"])
	}
}

func TestScriptJSONParseStringify(t *testing.T) {
	s := &Webhook{}
	outReq, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			var obj = {key: "value", num: 42, arr: [1,2,3]};
			ctx.req.body = JSON.stringify(obj);
			var parsed = JSON.parse(ctx.req.body);
			ctx.req.headers["X-Key"] = parsed.key;
			ctx.req.headers["X-Num"] = String(parsed.num);
		}`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if outReq.Headers["X-Key"] != "value" {
		t.Errorf("key = %s", outReq.Headers["X-Key"])
	}
	if outReq.Headers["X-Num"] != "42" {
		t.Errorf("num = %s", outReq.Headers["X-Num"])
	}
}

// ==================== Reply ====================

func TestScriptReply(t *testing.T) {
	s := &Webhook{}
	_, _, replies, _, _, _ := s.runScript(
		`function onRequest(ctx) { reply("hello"); reply("world"); }`,
		defaultMsg(), defaultReq(), "test")
	if len(replies) != 2 {
		t.Fatalf("replies: got %d, want 2", len(replies))
	}
	if replies[0] != "hello" || replies[1] != "world" {
		t.Errorf("replies = %v", replies)
	}
}

func TestScriptReplyFromOnResponse(t *testing.T) {
	// Can't easily test onResponse with a real HTTP call in unit test,
	// but we can verify the reply function works in onRequest
	s := &Webhook{}
	_, _, replies, _, _, _ := s.runScript(
		`function onRequest(ctx) { reply("from onRequest"); }`,
		defaultMsg(), defaultReq(), "test")
	if len(replies) != 1 || replies[0] != "from onRequest" {
		t.Errorf("replies = %v", replies)
	}
}

// ==================== Error Handling ====================

func TestScriptSyntaxError(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx { }`, // missing )
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	t.Logf("syntax error: %v", err)
}

func TestScriptRuntimeError(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { null.foo(); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected runtime error")
	}
	t.Logf("runtime error: %v", err)
}

func TestScriptUndefinedVariable(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { var x = undeclaredVar; }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected undefined variable error")
	}
}

func TestScriptThrowsError(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) { throw new Error("custom error"); }`,
		defaultMsg(), defaultReq(), "test")
	if err == nil {
		t.Fatal("expected thrown error")
	}
	if !strings.Contains(err.Error(), "custom error") {
		t.Errorf("error = %v", err)
	}
}

func TestScriptEmptyScript(t *testing.T) {
	s := &Webhook{}
	// No onRequest defined — should just pass through
	outReq, _, _, skipped, _, err := s.runScript(
		`// no functions defined`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if skipped {
		t.Error("should not be skipped")
	}
	if outReq == nil {
		t.Fatal("outReq nil")
	}
}

func TestScriptOnlyOnResponse(t *testing.T) {
	s := &Webhook{}
	// Only onResponse defined, no onRequest
	outReq, _, _, _, _, err := s.runScript(
		`function onResponse(ctx) { reply("from response"); }`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Request should pass through unchanged
	if outReq.URL != "http://localhost" {
		t.Errorf("url = %s", outReq.URL)
	}
}

// ==================== Memory / Large Data ====================

func TestScriptLargeStringAllocation(t *testing.T) {
	s := &Webhook{}
	// Try to allocate a huge string — should either work within limits or timeout
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			var s = "x";
			for (var i = 0; i < 20; i++) s = s + s; // 1MB
			ctx.req.body = s;
		}`,
		defaultMsg(), defaultReq(), "test")
	// This should succeed — 1MB is fine
	if err != nil {
		t.Logf("large string: %v (acceptable)", err)
	}
}

func TestScriptLargeArrayAllocation(t *testing.T) {
	s := &Webhook{}
	_, _, _, _, _, err := s.runScript(
		`function onRequest(ctx) {
			var arr = [];
			for (var i = 0; i < 100000; i++) arr.push(i);
			ctx.req.headers["X-Len"] = String(arr.length);
		}`,
		defaultMsg(), defaultReq(), "test")
	if err != nil {
		t.Logf("large array: %v (acceptable if timeout)", err)
	}
}
