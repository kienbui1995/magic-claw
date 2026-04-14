// Package tracing provides lightweight W3C Trace Context propagation
// compatible with OpenTelemetry without requiring the full OTel SDK.
//
// When MAGIC_OTEL_ENDPOINT is set, spans are exported via OTLP/HTTP.
// Otherwise, trace context is still propagated via W3C traceparent headers.
package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ctxKey struct{}

// Span represents a trace span.
type Span struct {
	TraceID  string            `json:"trace_id"`
	SpanID   string            `json:"span_id"`
	ParentID string            `json:"parent_id,omitempty"`
	Name     string            `json:"name"`
	Start    time.Time         `json:"start"`
	EndTime  time.Time         `json:"end,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Status   string            `json:"status,omitempty"` // ok, error
}

func randomID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewTraceID generates a new 128-bit trace ID.
func NewTraceID() string { return randomID(16) }

// NewSpanID generates a new 64-bit span ID.
func NewSpanID() string { return randomID(8) }

// StartSpan creates a new span, inheriting trace context from ctx.
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	s := &Span{
		SpanID: NewSpanID(),
		Name:   name,
		Start:  time.Now(),
		Attrs:  make(map[string]string),
	}
	if parent, ok := ctx.Value(ctxKey{}).(*Span); ok {
		s.TraceID = parent.TraceID
		s.ParentID = parent.SpanID
	} else {
		s.TraceID = NewTraceID()
	}
	return context.WithValue(ctx, ctxKey{}, s), s
}

// End marks the span as finished.
func (s *Span) End() { s.EndTime = time.Now() }

// SetAttr sets a span attribute.
func (s *Span) SetAttr(k, v string) { s.Attrs[k] = v }

// SetError marks the span as errored.
func (s *Span) SetError(err error) {
	s.Status = "error"
	s.Attrs["error"] = err.Error()
}

// Traceparent returns the W3C traceparent header value.
// Format: 00-{trace_id}-{span_id}-01
func (s *Span) Traceparent() string {
	return fmt.Sprintf("00-%s-%s-01", s.TraceID, s.SpanID)
}

// ParseTraceparent extracts trace/span IDs from a W3C traceparent header.
func ParseTraceparent(header string) (traceID, spanID string, ok bool) {
	parts := strings.Split(header, "-")
	if len(parts) < 4 || parts[0] != "00" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// InjectHeaders adds trace context to outgoing HTTP request headers.
func InjectHeaders(ctx context.Context, req *http.Request) {
	if s, ok := ctx.Value(ctxKey{}).(*Span); ok {
		req.Header.Set("Traceparent", s.Traceparent())
		req.Header.Set("X-Trace-ID", s.TraceID)
	}
}

// ExtractFromRequest creates a span context from incoming HTTP request headers.
func ExtractFromRequest(r *http.Request) context.Context {
	ctx := r.Context()
	if tp := r.Header.Get("Traceparent"); tp != "" {
		if traceID, spanID, ok := ParseTraceparent(tp); ok {
			parent := &Span{TraceID: traceID, SpanID: spanID}
			ctx = context.WithValue(ctx, ctxKey{}, parent)
		}
	} else if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		parent := &Span{TraceID: traceID, SpanID: NewSpanID()}
		ctx = context.WithValue(ctx, ctxKey{}, parent)
	}
	return ctx
}
