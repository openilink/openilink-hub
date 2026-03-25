package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// OTel span kinds
const (
	SpanKindInternal = "internal"
	SpanKindClient   = "client"
	SpanKindServer   = "server"
)

// OTel status codes
const (
	StatusUnset = "unset"
	StatusOK    = "ok"
	StatusError = "error"
)

// SpanEvent is a timestamped annotation on a span.
type SpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  int64          `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// TraceSpan is a single OTel-style span.
type TraceSpan struct {
	ID            int64          `json:"id"`
	TraceID       string         `json:"trace_id"`
	SpanID        string         `json:"span_id"`
	ParentSpanID  string         `json:"parent_span_id,omitempty"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	StatusCode    string         `json:"status_code"`
	StatusMessage string         `json:"status_message,omitempty"`
	StartTime     int64          `json:"start_time"`
	EndTime       int64          `json:"end_time"`
	Attributes    map[string]any `json:"attributes,omitempty"`
	Events        []SpanEvent    `json:"events,omitempty"`
	BotID         string         `json:"bot_id,omitempty"`
	CreatedAt     int64          `json:"created_at"`
}

func (s *TraceSpan) DurationMs() int64 {
	if s.EndTime > s.StartTime {
		return s.EndTime - s.StartTime
	}
	return 0
}

func genSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func genTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "tr_" + hex.EncodeToString(b)
}

// TraceStore covers the DB operations for tracing.
type TraceStore interface {
	// InsertSpan inserts a fully-formed span (used by Tracer.Flush).
	InsertSpan(traceID, spanID, parentSpanID, name, kind, statusCode, statusMessage string,
		startTime, endTime int64, attrsJSON, eventsJSON []byte, botID string) error
	// AppendSpan adds a span to an existing trace as a child of the root span.
	AppendSpan(traceID, botID, name, kind, statusCode, statusMessage string, attrs map[string]any) error
	ListRootSpans(botID string, limit int) ([]TraceSpan, error)
	ListSpansByTrace(traceID string) ([]TraceSpan, error)
}

// Tracer creates and manages spans for a single trace.
type Tracer struct {
	mu      sync.Mutex
	ts      TraceStore
	traceID string
	botID   string
	spans   []*SpanBuilder
}

// NewTracer creates a tracer for a new message trace.
func NewTracer(ts TraceStore, botID string) *Tracer {
	return &Tracer{
		ts:      ts,
		traceID: genTraceID(),
		botID:   botID,
	}
}

func (t *Tracer) TraceID() string { return t.traceID }

// Start begins a new span.
func (t *Tracer) Start(name string, kind string, attrs map[string]any) *SpanBuilder {
	sb := &SpanBuilder{
		tracer:     t,
		spanID:     genSpanID(),
		name:       name,
		kind:       kind,
		startTime:  time.Now().UnixMilli(),
		attributes: attrs,
		statusCode: StatusUnset,
	}
	t.mu.Lock()
	t.spans = append(t.spans, sb)
	t.mu.Unlock()
	return sb
}

// StartChild begins a child span under a parent.
func (t *Tracer) StartChild(parent *SpanBuilder, name string, kind string, attrs map[string]any) *SpanBuilder {
	sb := t.Start(name, kind, attrs)
	if parent != nil {
		sb.parentSpanID = parent.spanID
	}
	return sb
}

// Flush writes all spans to the database.
func (t *Tracer) Flush() {
	t.mu.Lock()
	spans := make([]*SpanBuilder, len(t.spans))
	copy(spans, t.spans)
	t.mu.Unlock()

	for _, sb := range spans {
		sb.mu.Lock()
		if sb.endTime == 0 {
			sb.endTime = time.Now().UnixMilli()
		}
		attrsJSON, _ := json.Marshal(sb.attributes)
		eventsJSON, _ := json.Marshal(sb.events)
		_ = t.ts.InsertSpan(t.traceID, sb.spanID, sb.parentSpanID, sb.name, sb.kind,
			sb.statusCode, sb.statusMessage, sb.startTime, sb.endTime,
			attrsJSON, eventsJSON, t.botID)
		sb.mu.Unlock()
	}
}

// SpanBuilder builds a span with fluent API.
type SpanBuilder struct {
	mu            sync.Mutex
	tracer        *Tracer
	spanID        string
	parentSpanID  string
	name          string
	kind          string
	startTime     int64
	endTime       int64
	statusCode    string
	statusMessage string
	attributes    map[string]any
	events        []SpanEvent
}

func (sb *SpanBuilder) SpanID() string { return sb.spanID }

func (sb *SpanBuilder) SetAttr(key string, value any) *SpanBuilder {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.attributes == nil {
		sb.attributes = map[string]any{}
	}
	sb.attributes[key] = value
	return sb
}

func (sb *SpanBuilder) SetStatus(code, message string) *SpanBuilder {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.statusCode = code
	sb.statusMessage = message
	return sb
}

func (sb *SpanBuilder) AddEvent(name string, attrs map[string]any) *SpanBuilder {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.events = append(sb.events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now().UnixMilli(),
		Attributes: attrs,
	})
	return sb
}

func (sb *SpanBuilder) End() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.endTime = time.Now().UnixMilli()
	if sb.statusCode == StatusUnset {
		sb.statusCode = StatusOK
	}
}

func (sb *SpanBuilder) EndWithError(err string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.endTime = time.Now().UnixMilli()
	sb.statusCode = StatusError
	sb.statusMessage = err
}
