# iLink Mock Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a complete mock iLink server with a unified Engine core, two access modes (Direct provider + HTTP), controllable clock, real AES-128-ECB media crypto, and QR binding state machine. Delete the old `internal/provider/mock/` and migrate all tests.

**Architecture:** `internal/provider/ilink/mockserver/` package. A single `Engine` holds all state (messages, media, sessions, QR). `Provider` wraps Engine as `provider.Provider` for direct use. `HTTPServer` wraps Engine as SDK-compatible HTTP endpoints. `FakeClock` enables deterministic zero-wait testing. The SDK's own `EncryptAESECB`/`DecryptAESECB` functions are reused for media crypto.

**Tech Stack:** Go `net/http/httptest`, `crypto/aes`, `encoding/json`, `github.com/openilink/openilink-sdk-go` (types + crypto functions).

---

## Key Design Decisions

- **One Engine, two modes**: Direct Mode (provider.Provider) and HTTP Mode (httptest.Server) share identical Engine state
- **Controllable clock**: `Clock` interface with `FakeClock.Advance()` — no `time.Sleep` anywhere
- **Real AES-128-ECB**: Reuse `ilink.EncryptAESECB` / `ilink.DecryptAESECB` from SDK
- **SDK-accurate HTTP**: Endpoint paths match `openilink-sdk-go v0.4.2` exactly
- **Delete old mock**: Remove `internal/provider/mock/`, update all 22 type assertions + 22 SimulateInbound + 8 SentMessages calls in `integration_test.go`
- **Provider registration**: Register as `"mock"` in `init()` so existing `provider.Get("mock")` still works

## SDK HTTP Endpoint Reference

| SDK Method | HTTP | Path | Auth |
|-----------|------|------|------|
| `Monitor/GetUpdates` | POST | `ilink/bot/getupdates` | Bearer token |
| `SendMessage` | POST | `ilink/bot/sendmessage` | Bearer token |
| `GetConfig` | POST | `ilink/bot/getconfig` | Bearer token |
| `SendTyping` | POST | `ilink/bot/sendtyping` | Bearer token |
| `GetUploadURL` | POST | `ilink/bot/getuploadurl` | Bearer token |
| `UploadFile` (CDN) | POST | `c2c/upload?encrypted_query_param=...&filekey=...` | None |
| `DownloadFile` (CDN) | GET | `c2c/download?encrypted_query_param=...` | None |
| `FetchQRCode` | GET | `ilink/bot/get_bot_qrcode?bot_type=3` | Optional |
| `PollQRStatus` | GET | `ilink/bot/get_qrcode_status?qrcode=...` | Optional |

Auth: `Authorization: Bearer <token>`, `AuthorizationType: ilink_bot_token`

---

## Task 1: Clock interface and FakeClock

**Files:**
- Create: `internal/provider/ilink/mockserver/clock.go`
- Create: `internal/provider/ilink/mockserver/clock_test.go`

**What to do:**

```go
// clock.go
package mockserver

import (
    "sync"
    "time"
)

type Clock interface {
    Now() time.Time
    After(d time.Duration) <-chan time.Time
    NewTimer(d time.Duration) *ClockTimer
}

type ClockTimer struct {
    C    <-chan time.Time
    stop func() bool
    reset func(time.Duration)
}
func (t *ClockTimer) Stop() bool           { return t.stop() }
func (t *ClockTimer) Reset(d time.Duration) { t.reset(d) }

// realClock delegates to the time package.
type realClock struct{}
func (realClock) Now() time.Time                          { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time   { return time.After(d) }
func (realClock) NewTimer(d time.Duration) *ClockTimer {
    t := time.NewTimer(d)
    return &ClockTimer{C: t.C, stop: t.Stop, reset: t.Reset}
}

// FakeClock is a manually-advanced clock for deterministic testing.
type FakeClock struct {
    mu      sync.Mutex
    now     time.Time
    waiters []*fakeWaiter
}

type fakeWaiter struct {
    deadline time.Time
    ch       chan time.Time
    fired    bool
}

func NewFakeClock(now time.Time) *FakeClock { return &FakeClock{now: now} }

func (c *FakeClock) Now() time.Time {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.now
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
    return c.NewTimer(d).C
}

func (c *FakeClock) NewTimer(d time.Duration) *ClockTimer {
    c.mu.Lock()
    defer c.mu.Unlock()
    w := &fakeWaiter{deadline: c.now.Add(d), ch: make(chan time.Time, 1)}
    c.waiters = append(c.waiters, w)
    ct := &ClockTimer{
        C: w.ch,
        stop: func() bool {
            c.mu.Lock()
            defer c.mu.Unlock()
            if w.fired { return false }
            w.fired = true
            return true
        },
        reset: func(nd time.Duration) {
            c.mu.Lock()
            defer c.mu.Unlock()
            w.deadline = c.now.Add(nd)
            w.fired = false
        },
    }
    return ct
}

// Advance moves the clock forward by d and fires any expired timers.
func (c *FakeClock) Advance(d time.Duration) {
    c.mu.Lock()
    c.now = c.now.Add(d)
    now := c.now
    var remaining []*fakeWaiter
    for _, w := range c.waiters {
        if !w.fired && !w.deadline.After(now) {
            w.fired = true
            select {
            case w.ch <- now:
            default:
            }
        }
        if !w.fired {
            remaining = append(remaining, w)
        }
    }
    c.waiters = remaining
    c.mu.Unlock()
}

func (c *FakeClock) Set(t time.Time) {
    c.mu.Lock()
    d := t.Sub(c.now)
    c.mu.Unlock()
    if d > 0 { c.Advance(d) }
}
```

**Test:** `clock_test.go` — verify `Advance` fires timers, `Stop` prevents firing, `After` returns channel.

**Step 1:** Create `clock.go` and `clock_test.go`.

**Step 2:** Run `go test ./internal/provider/ilink/mockserver/... -run TestClock -v`

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): add Clock interface and FakeClock"
```

---

## Task 2: AES-128-ECB crypto helpers

**Files:**
- Create: `internal/provider/ilink/mockserver/crypto.go`
- Create: `internal/provider/ilink/mockserver/crypto_test.go`

**What to do:**

Wrap `ilink.EncryptAESECB` / `ilink.DecryptAESECB` / `ilink.ParseAESKey` from the SDK. Add helper functions for key generation and EQP generation.

```go
// crypto.go
package mockserver

import (
    "crypto/rand"
    "encoding/hex"

    ilink "github.com/openilink/openilink-sdk-go"
)

func generateAESKey() (raw []byte, hexKey string) {
    raw = make([]byte, 16)
    _, _ = rand.Read(raw)
    return raw, hex.EncodeToString(raw)
}

func generateEQP() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return "mock-eqp-" + hex.EncodeToString(b)
}

func encryptMedia(plaintext, key []byte) ([]byte, error) {
    return ilink.EncryptAESECB(plaintext, key)
}

func decryptMedia(ciphertext, key []byte) ([]byte, error) {
    return ilink.DecryptAESECB(ciphertext, key)
}

func parseAESKey(aesKeyBase64 string) ([]byte, error) {
    return ilink.ParseAESKey(aesKeyBase64)
}
```

**Test:** `crypto_test.go` — roundtrip encrypt/decrypt, key mismatch returns error, various key encodings via `ParseAESKey`.

**Step 1:** Create files.

**Step 2:** Run tests.

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): add AES-128-ECB crypto helpers"
```

---

## Task 3: Engine core — types, options, state

**Files:**
- Create: `internal/provider/ilink/mockserver/types.go`
- Create: `internal/provider/ilink/mockserver/engine.go`

**What to do:**

`types.go` — shared request/response types for both modes:

```go
package mockserver

import "encoding/json"

// InboundRequest is what tests/control endpoints use to inject messages.
type InboundRequest struct {
    Sender       string          `json:"sender"`
    Recipient    string          `json:"recipient,omitempty"`
    Text         string          `json:"text,omitempty"`
    Items        []ItemRequest   `json:"items,omitempty"` // for media messages
    GroupID      string          `json:"group_id,omitempty"`
    ContextToken string          `json:"context_token,omitempty"`
    SessionID    string          `json:"session_id,omitempty"`
    MessageState int             `json:"message_state,omitempty"`
}

type ItemRequest struct {
    Type     string `json:"type"` // "text","image","voice","file","video"
    Text     string `json:"text,omitempty"`
    FileName string `json:"file_name,omitempty"`
    Data     []byte `json:"data,omitempty"` // raw media bytes (will be encrypted)
}

// SentMessage records an outbound message.
type SentMessage struct {
    Recipient    string          `json:"recipient"`
    Text         string          `json:"text,omitempty"`
    ContextToken string          `json:"context_token,omitempty"`
    ClientID     string          `json:"client_id"`
    Items        json.RawMessage `json:"items,omitempty"` // raw item_list from SendMessage
    MediaData    []byte          `json:"-"` // decrypted media bytes if media was sent
    FileName     string          `json:"file_name,omitempty"`
    Timestamp    int64           `json:"timestamp"`
}

// MediaInfo describes an uploaded media file.
type MediaInfo struct {
    EQP       string `json:"eqp"`
    AESKey    string `json:"aes_key"`
    MediaType int    `json:"media_type"`
    FileName  string `json:"file_name,omitempty"`
    Size      int    `json:"size"`
}

// Credentials matches ilink provider credentials format.
type Credentials struct {
    BotID       string `json:"bot_id"`
    BotToken    string `json:"bot_token"`
    BaseURL     string `json:"base_url,omitempty"`
    ILinkUserID string `json:"ilink_user_id,omitempty"`
}
```

`engine.go` — Engine struct and constructor:

```go
package mockserver

import (
    "sync"
    "time"

    ilink "github.com/openilink/openilink-sdk-go"
)

type mediaEntry struct {
    ciphertext []byte
    aesKeyHex  string // hex-encoded 16-byte key
    aesKeyRaw  []byte
    mediaType  int
    fileName   string
    rawSize    int
    uploadedAt time.Time
}

type qrSession struct {
    qrCode string
    state  string // "wait", "scanned", "confirmed", "expired"
    creds  Credentials
    ch     chan struct{} // notify state changes
    timer  *ClockTimer
}

type engineOptions struct {
    clock          Clock
    autoConfirmQR  bool
    latency        time.Duration
    messageLoss    float64
    sessionTTL     time.Duration
}

type Option func(*engineOptions)
func WithClock(c Clock) Option              { return func(o *engineOptions) { o.clock = c } }
func WithAutoConfirmQR() Option             { return func(o *engineOptions) { o.autoConfirmQR = true } }
func WithLatency(d time.Duration) Option    { return func(o *engineOptions) { o.latency = d } }
func WithMessageLoss(rate float64) Option   { return func(o *engineOptions) { o.messageLoss = rate } }
func WithSessionTTL(d time.Duration) Option { return func(o *engineOptions) { o.sessionTTL = d } }

type Engine struct {
    mu     sync.Mutex
    clock  Clock
    opts   engineOptions

    // Session
    token  string
    status string // "connected" / "session_expired" / "disconnected"

    // Messages
    inbound chan *ilink.WeixinMessage
    sent    []SentMessage
    sentCh  chan struct{} // notifies WaitForSent
    syncBuf string
    msgSeq  int64

    // Media
    media map[string]*mediaEntry

    // QR
    qr *qrSession

    // Typing
    typing map[string]bool
}

func NewEngine(opts ...Option) *Engine {
    o := engineOptions{clock: realClock{}}
    for _, opt := range opts { opt(&o) }
    return &Engine{
        clock:   o.clock,
        opts:    o,
        status:  "disconnected",
        inbound: make(chan *ilink.WeixinMessage, 100),
        sentCh:  make(chan struct{}, 1),
        media:   make(map[string]*mediaEntry),
        typing:  make(map[string]bool),
    }
}
```

**Step 1:** Create both files.

**Step 2:** `go build ./internal/provider/ilink/mockserver/...`

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): add Engine core types and constructor"
```

---

## Task 4: Engine SDK operations

**Files:**
- Create: `internal/provider/ilink/mockserver/engine_sdk.go`
- Create: `internal/provider/ilink/mockserver/engine_sdk_test.go`

**What to do:**

Implement all SDK-facing Engine methods. These are what the HTTP handlers and Direct Mode provider call.

```go
// engine_sdk.go
package mockserver
```

Methods to implement:

**GetUpdates(ctx, buf)** — Blocks on `e.inbound` channel. Returns messages + updated syncBuf. Respects `e.opts.latency`. On session_expired, returns error with errcode -14.

**SendText(recipient, text, contextToken)** — Records in `e.sent`, notifies `e.sentCh`, generates client ID `sdk-<ms>-<hex>`.

**Push(recipient, text)** — Same as SendText but without contextToken requirement.

**SendMessage(msg)** — Accepts raw `*ilink.WeixinMessage`, records in `e.sent`. If message contains media items with CDN info, decrypt the media and store the decrypted bytes in `SentMessage.MediaData`.

**SendMediaFile(recipient, contextToken, data, fileName, text)** — Auto-uploads media (encrypt + store in `e.media`), then records as sent message.

**GetUploadURL(req)** — Validates params, generates `upload_param`, returns response.

**UploadFile(uploadParam, filekey, ciphertext)** — Stores ciphertext in `e.media[eqp]`, returns download EQP via `x-encrypted-param` header.

**DownloadFile(eqp, aesKeyBase64)** — Looks up `e.media[eqp]`, parses AES key, decrypts, returns plaintext.

**DownloadVoice(eqp, aesKeyBase64, sampleRate)** — Same as DownloadFile. (SDK handles SILK decode on client side; mock stores whatever was uploaded.)

**SendTyping(recipient, ticket, typing)** — Sets `e.typing[recipient]`.

**GetConfig(recipient, contextToken)** — Returns `{typing_ticket: "mock-ticket-<hex>"}`.

**FetchQRCode()** — Creates `qrSession` with state "wait", starts 30s expiry timer via `e.clock.After`. If `autoConfirmQR`, immediately sets state to "confirmed" with default credentials.

**PollQRStatus(ctx, qrCode)** — Blocks on `qr.ch` or ctx cancel. Returns current state. On "confirmed", returns credentials.

**Tests:** Test GetUpdates blocking + cancel, SendText recording, Upload→Download roundtrip with real AES, QR state machine transitions, QR expiry via FakeClock.

**Step 1:** Create files.

**Step 2:** Run tests.

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): implement Engine SDK operations"
```

---

## Task 5: Engine test control operations

**Files:**
- Create: `internal/provider/ilink/mockserver/engine_control.go`
- Create: `internal/provider/ilink/mockserver/engine_control_test.go`

**What to do:**

```go
// engine_control.go
package mockserver
```

**InjectInbound(req InboundRequest)** — Converts `InboundRequest` to `*ilink.WeixinMessage`:
- Auto-assigns `e.msgSeq++` as MessageID
- Timestamp from `e.clock.Now().UnixMilli()`
- If `req.Items` has media with `Data`, encrypt it and create CDN references
- If `req.Text` is set and no Items, create single text item
- Pushes to `e.inbound` channel
- Respects `e.opts.messageLoss` (random drop)

**SentMessages()** — Returns copy of `e.sent`.

**WaitForSent(t, timeout)** — Blocks on `e.sentCh` until at least one new message or timeout. Returns all sent messages.

**AssertSentCount(t, n)** — `t.Helper()`, fails if `len(e.sent) != n`.

**AssertSentContains(t, substring)** — Checks any `SentMessage.Text` contains substring.

**ExpireSession()** — Sets `e.status = "session_expired"`, unblocks any waiting GetUpdates.

**ScanQR()** — Sets `qr.state = "scanned"`, notifies `qr.ch`.

**ConfirmQR(creds)** — Sets `qr.state = "confirmed"`, stores credentials, notifies `qr.ch`.

**ListMedia()** — Returns `[]MediaInfo` from `e.media`.

**Reset()** — Clears all state, re-creates channels.

**SetToken(token)** — Sets engine token (called by Provider.Start).

**SetStatus(status)** — Sets engine status.

**Tests:** InjectInbound → GetUpdates receives it, ExpireSession unblocks GetUpdates, WaitForSent with timeout, AssertSentCount/Contains.

**Step 1:** Create files.

**Step 2:** Run tests.

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): implement Engine test control operations"
```

---

## Task 6: Direct Mode — Provider implementation

**Files:**
- Create: `internal/provider/ilink/mockserver/provider.go`
- Create: `internal/provider/ilink/mockserver/provider_test.go`

**What to do:**

```go
// provider.go
package mockserver

import (
    "context"
    "encoding/json"

    "github.com/openilink/openilink-hub/internal/provider"
)

func init() {
    provider.Register("mock", func() provider.Provider {
        return NewProvider()
    })
}

type Provider struct {
    engine *Engine
    cancel context.CancelFunc
    opts   provider.StartOptions
}

func NewProvider(opts ...Option) *Provider {
    return &Provider{engine: NewEngine(opts...)}
}

func (p *Provider) Engine() *Engine { return p.engine }

func (p *Provider) Name() string  { return "mock" }
func (p *Provider) Status() string { return p.engine.status }

func (p *Provider) Start(ctx context.Context, opts provider.StartOptions) error {
    p.opts = opts
    var creds Credentials
    if opts.Credentials != nil {
        json.Unmarshal(opts.Credentials, &creds)
    }
    if creds.BotToken == "" {
        creds.BotToken = "mock-token"
    }
    p.engine.SetToken(creds.BotToken)
    p.engine.SetStatus("connected")
    if opts.OnStatus != nil {
        opts.OnStatus("connected")
    }

    ctx, p.cancel = context.WithCancel(ctx)
    go p.pollLoop(ctx)
    return nil
}

func (p *Provider) pollLoop(ctx context.Context) {
    buf := ""
    for {
        result, err := p.engine.GetUpdates(ctx, buf)
        if err != nil {
            // Context cancelled or session expired
            if p.engine.status == "session_expired" {
                if p.opts.OnStatus != nil {
                    p.opts.OnStatus("session_expired")
                }
            }
            return
        }
        if result.SyncBuf != "" {
            buf = result.SyncBuf
            if p.opts.OnSyncUpdate != nil {
                data, _ := json.Marshal(map[string]string{"sync_buf": buf})
                p.opts.OnSyncUpdate(data)
            }
        }
        for _, msg := range result.Messages {
            if p.opts.OnMessage != nil {
                p.opts.OnMessage(convertWeixinToInbound(msg))
            }
        }
    }
}

func (p *Provider) Stop() {
    if p.cancel != nil { p.cancel() }
    p.engine.SetStatus("disconnected")
}

func (p *Provider) Send(ctx context.Context, msg provider.OutboundMessage) (string, error) {
    if len(msg.Data) > 0 && msg.FileName != "" {
        return "", p.engine.SendMediaFile(msg.Recipient, msg.ContextToken, msg.Data, msg.FileName, msg.Text)
    }
    if msg.ContextToken != "" {
        return p.engine.SendText(msg.Recipient, msg.Text, msg.ContextToken)
    }
    return p.engine.Push(msg.Recipient, msg.Text)
}

func (p *Provider) SendTyping(ctx context.Context, recipient, ticket string, typing bool) error {
    return p.engine.SendTyping(recipient, ticket, typing)
}

func (p *Provider) GetConfig(ctx context.Context, recipient, contextToken string) (*provider.BotConfig, error) {
    result, err := p.engine.GetConfig(recipient, contextToken)
    if err != nil { return nil, err }
    return &provider.BotConfig{TypingTicket: result.TypingTicket}, nil
}

func (p *Provider) DownloadMedia(ctx context.Context, eqp, aesKey string) ([]byte, error) {
    return p.engine.DownloadFile(eqp, aesKey)
}

func (p *Provider) DownloadVoice(ctx context.Context, eqp, aesKey string, sampleRate int) ([]byte, error) {
    return p.engine.DownloadVoice(eqp, aesKey, sampleRate)
}

// convertWeixinToInbound converts ilink.WeixinMessage to provider.InboundMessage.
// Reuses the same conversion logic as the real ilink provider.
func convertWeixinToInbound(msg *ilink.WeixinMessage) provider.InboundMessage {
    // ... (same item conversion as ilink/ilink.go convertInbound)
}
```

Also implement `provider.Binder` interface:
```go
func (p *Provider) StartBind(ctx context.Context) (*provider.BindSession, error) {
    qr, err := p.engine.FetchQRCode()
    // ... return BindSession with PollStatus calling engine.PollQRStatus
}
```

**Provide `MockCredentials()` package-level function** matching old `mock.Credentials()` signature.

**Tests:** Start → inject → receive message via OnMessage, Send → check Engine.SentMessages, StartBind → ConfirmQR → get credentials.

**Step 1:** Create files.

**Step 2:** Run tests: `go test ./internal/provider/ilink/mockserver/ -v`

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): implement Direct Mode provider"
```

---

## Task 7: HTTP Mode — SDK endpoint handlers

**Files:**
- Create: `internal/provider/ilink/mockserver/http.go`
- Create: `internal/provider/ilink/mockserver/http_sdk.go`
- Create: `internal/provider/ilink/mockserver/http_control.go`
- Create: `internal/provider/ilink/mockserver/http_test.go`

**What to do:**

`http.go` — Server setup:
```go
type HTTPServer struct {
    engine *Engine
    mux    *http.ServeMux
    srv    *httptest.Server
}

func NewHTTPServer(opts ...Option) *HTTPServer {
    e := NewEngine(opts...)
    s := &HTTPServer{engine: e, mux: http.NewServeMux()}
    // Register SDK endpoints
    s.mux.HandleFunc("POST /ilink/bot/getupdates", s.handleGetUpdates)
    s.mux.HandleFunc("POST /ilink/bot/sendmessage", s.handleSendMessage)
    s.mux.HandleFunc("POST /ilink/bot/getconfig", s.handleGetConfig)
    s.mux.HandleFunc("POST /ilink/bot/sendtyping", s.handleSendTyping)
    s.mux.HandleFunc("POST /ilink/bot/getuploadurl", s.handleGetUploadURL)
    s.mux.HandleFunc("POST /c2c/upload", s.handleCDNUpload)
    s.mux.HandleFunc("GET /c2c/download", s.handleCDNDownload)
    s.mux.HandleFunc("GET /ilink/bot/get_bot_qrcode", s.handleFetchQR)
    s.mux.HandleFunc("GET /ilink/bot/get_qrcode_status", s.handlePollQR)
    // Control endpoints
    s.mux.HandleFunc("POST /mock/inbound", s.handleMockInbound)
    s.mux.HandleFunc("GET /mock/sent", s.handleMockSent)
    s.mux.HandleFunc("POST /mock/qr/scan", s.handleMockScan)
    s.mux.HandleFunc("POST /mock/qr/confirm", s.handleMockConfirm)
    s.mux.HandleFunc("POST /mock/session/expire", s.handleMockExpire)
    s.mux.HandleFunc("GET /mock/media", s.handleMockListMedia)
    s.mux.HandleFunc("POST /mock/reset", s.handleMockReset)
    return s
}
func (s *HTTPServer) Start() string { s.srv = httptest.NewServer(s.mux); return s.srv.URL }
func (s *HTTPServer) Handler() http.Handler { return s.mux }
func (s *HTTPServer) Engine() *Engine { return s.engine }
func (s *HTTPServer) Close() { if s.srv != nil { s.srv.Close() } }
```

`http_sdk.go` — Each handler decodes SDK JSON format, calls engine, encodes SDK JSON response. Must match `openilink-sdk-go` request/response structures exactly:
- `handleGetUpdates`: JSON body `{get_updates_buf, base_info}` → `engine.GetUpdates` → `{ret:0, msgs:[], get_updates_buf, sync_buf}`
- `handleSendMessage`: JSON body `{msg, base_info}` → `engine.SendMessage` → `{ret:0}`
- `handleGetConfig`: JSON body `{ilink_user_id, context_token}` → `{ret:0, typing_ticket}`
- `handleSendTyping`: JSON body `{ilink_user_id, typing_ticket, status}` → `{ret:0}`
- `handleGetUploadURL`: JSON body matches SDK `GetUploadURLReq` → `{ret:0, upload_param}`
- `handleCDNUpload`: Binary body, query params `encrypted_query_param` + `filekey` → stores in engine → response header `x-encrypted-param`
- `handleCDNDownload`: Query param `encrypted_query_param` → reads from engine → binary response
- `handleFetchQR`: Query param `bot_type` → `{qrcode, qrcode_img_content}`
- `handlePollQR`: Query param `qrcode` → long-poll → `{status, bot_token, ilink_bot_id, baseurl}`

`http_control.go` — Thin JSON wrappers around engine control methods.

**Tests:** `http_test.go` — Test the full HTTP roundtrip:
1. Start HTTPServer
2. Create `ilink.NewClient(token, ilink.WithBaseURL(srv.URL))` — the real SDK client
3. Inject message via `/mock/inbound`
4. Call `client.Monitor()` → receive the message
5. Call `client.SendText()` → verify via `/mock/sent`
6. Upload + Download media with real AES encryption
7. QR flow: FetchQRCode → `/mock/qr/scan` → `/mock/qr/confirm` → PollQRStatus gets credentials

**Step 1:** Create all four files.

**Step 2:** Run tests.

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "feat(mockserver): implement HTTP Mode with SDK-compatible endpoints"
```

---

## Task 8: Standalone server command

**Files:**
- Create: `cmd/mockserver/main.go`

**What to do:**

```go
package main

import (
    "flag"
    "fmt"
    "log/slog"
    "net/http"

    "github.com/openilink/openilink-hub/internal/provider/ilink/mockserver"
)

func main() {
    listen := flag.String("listen", ":9900", "listen address")
    flag.Parse()

    srv := mockserver.NewHTTPServer()
    fmt.Printf("iLink Mock Server running on http://localhost%s\n", *listen)
    fmt.Println("  POST /mock/inbound     — inject inbound message")
    fmt.Println("  GET  /mock/sent        — view sent messages")
    fmt.Println("  POST /mock/qr/scan     — simulate QR scan")
    fmt.Println("  POST /mock/qr/confirm  — confirm QR binding")
    fmt.Println("  POST /mock/session/expire — expire session")
    fmt.Println("  POST /mock/reset       — reset all state")
    if err := http.ListenAndServe(*listen, srv.Handler()); err != nil {
        slog.Error("server error", "err", err)
    }
}
```

**Step 1:** Create file.

**Step 2:** `go build ./cmd/mockserver/`

**Step 3:** Commit.
```bash
git add cmd/mockserver/
git commit -m "feat: add standalone mock server command"
```

---

## Task 9: Delete old mock, migrate all tests

**Files:**
- Delete: `internal/provider/mock/mock.go`
- Modify: `integration_test.go`
- Modify: Any other test files referencing `provider/mock`

**What to do:**

Delete `internal/provider/mock/` directory entirely.

In `integration_test.go`, update:

1. Import: `mockProvider "github.com/openilink/openilink-hub/internal/provider/mock"` → `"github.com/openilink/openilink-hub/internal/provider/ilink/mockserver"`

2. All 22 type assertions: `inst.Provider.(*mockProvider.Provider)` → `inst.Provider.(*mockserver.Provider)`

3. All 22 SimulateInbound calls: `mock.SimulateInbound(msg)` → `mock.Engine().InjectInbound(mockserver.InboundRequest{...})`
   - Convert `provider.InboundMessage` to `mockserver.InboundRequest` format
   - `InboundRequest` uses `Sender`, `Text`, `Items`, `ContextToken`, `GroupID`, `SessionID`, `MessageState`

4. All 8 SentMessages calls: `mock.SentMessages()` → `mock.Engine().SentMessages()`
   - Return type changes from `[]provider.OutboundMessage` to `[]mockserver.SentMessage`
   - Adjust assertions: `.Text` field, `.Recipient` field etc.

5. All 2 Credentials calls: `mockProvider.Credentials()` → `mockserver.MockCredentials()`

6. Bot creation: `"mock"` provider name stays the same (mockserver registers as "mock" in init).

**Step 1:** Delete `internal/provider/mock/`.

**Step 2:** Update `integration_test.go`.

**Step 3:** Add `time.Sleep` → channel-based waits where needed (some tests may need `engine.WaitForSent` instead of sleeps).

**Step 4:** `go build ./...` — fix compilation errors.

**Step 5:** `go vet ./...`

**Step 6:** Commit.
```bash
git add -A
git commit -m "refactor: migrate all tests from old mock to mockserver"
```

---

## Task 10: End-to-end integration test with HTTP Mode

**Files:**
- Create: `internal/provider/ilink/mockserver/integration_test.go`

**What to do:**

Write a single comprehensive test that exercises the full path through the real iLink provider connected to the mock HTTP server:

```go
func TestHTTPModeFullFlow(t *testing.T) {
    // 1. Start mock HTTP server
    srv := mockserver.NewHTTPServer()
    url := srv.Start()
    defer srv.Close()

    // 2. Create real iLink provider
    p := &ilinkProvider.Provider{}
    creds, _ := json.Marshal(ilinkProvider.Credentials{
        BotToken: "test-token",
        BaseURL:  url,
    })

    // 3. Start provider
    var received []provider.InboundMessage
    var mu sync.Mutex
    p.Start(context.Background(), provider.StartOptions{
        Credentials: creds,
        OnMessage: func(msg provider.InboundMessage) {
            mu.Lock()
            received = append(received, msg)
            mu.Unlock()
        },
        OnStatus: func(status string) {},
        OnSyncUpdate: func(state json.RawMessage) {},
    })
    defer p.Stop()

    // 4. Inject inbound text message via control API
    srv.Engine().InjectInbound(mockserver.InboundRequest{
        Sender: "user1@wx",
        Text:   "hello bot",
    })

    // 5. Wait for message to arrive via Monitor polling
    time.Sleep(100 * time.Millisecond) // real HTTP needs small wait
    mu.Lock()
    require len(received) == 1
    require received[0].Items[0].Text == "hello bot"
    mu.Unlock()

    // 6. Send reply via real provider
    p.Send(ctx, provider.OutboundMessage{
        Recipient: "user1@wx", Text: "hi!", ContextToken: received[0].ContextToken,
    })

    // 7. Verify via control API
    sent := srv.Engine().SentMessages()
    require len(sent) == 1
    require sent[0].Text == "hi!"

    // 8. Media roundtrip: upload image, inject message referencing it, download
    // ... (exercises real AES encrypt/decrypt through the full pipeline)
}
```

**Step 1:** Create test.

**Step 2:** Run: `go test ./internal/provider/ilink/mockserver/ -run TestHTTPModeFullFlow -v`

**Step 3:** Commit.
```bash
git add internal/provider/ilink/mockserver/
git commit -m "test(mockserver): add end-to-end HTTP mode integration test"
```
