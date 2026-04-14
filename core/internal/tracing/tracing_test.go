package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartSpan(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-op")
	if span.TraceID == "" || span.SpanID == "" {
		t.Fatal("span should have trace and span IDs")
	}
	if span.ParentID != "" {
		t.Error("root span should have no parent")
	}

	// Child span inherits trace ID
	_, child := StartSpan(ctx, "child-op")
	if child.TraceID != span.TraceID {
		t.Error("child should inherit trace ID")
	}
	if child.ParentID != span.SpanID {
		t.Error("child parent should be parent span ID")
	}
}

func TestTraceparent(t *testing.T) {
	_, span := StartSpan(context.Background(), "test")
	tp := span.Traceparent()
	traceID, spanID, ok := ParseTraceparent(tp)
	if !ok {
		t.Fatal("should parse traceparent")
	}
	if traceID != span.TraceID || spanID != span.SpanID {
		t.Error("parsed IDs should match")
	}
}

func TestExtractInject(t *testing.T) {
	// Create a span and inject into request
	ctx, span := StartSpan(context.Background(), "origin")
	req := httptest.NewRequest("GET", "/", nil)
	InjectHeaders(ctx, req)

	if req.Header.Get("Traceparent") == "" {
		t.Error("should inject traceparent header")
	}

	// Extract from request
	extracted := ExtractFromRequest(req)
	_, child := StartSpan(extracted, "downstream")
	if child.TraceID != span.TraceID {
		t.Error("extracted context should carry same trace ID")
	}
}

func TestExtractFromXTraceID(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Trace-ID", "abc123")
	ctx := ExtractFromRequest(req)
	_, span := StartSpan(ctx, "test")
	if span.TraceID != "abc123" {
		t.Errorf("should use X-Trace-ID, got %s", span.TraceID)
	}
}
