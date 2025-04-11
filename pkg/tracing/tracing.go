package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SpanContext represents the context of a trace span
type SpanContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

// Span represents a single span in a trace
type Span struct {
	Name       string                 `json:"name"`
	Context    SpanContext            `json:"context"`
	ParentID   string                 `json:"parent_id,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time,omitempty"`
	Duration   int64                  `json:"duration_ms,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Events     []SpanEvent            `json:"events,omitempty"`
	Status     SpanStatus             `json:"status"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    int    `json:"code"` // 0: Unset, 1: Ok, 2: Error
	Message string `json:"message,omitempty"`
}

// Tracer manages trace collection
type Tracer struct {
	spans       map[string]*Span
	activeSpans sync.Map
	mutex       sync.Mutex
	enabled     bool
	tracesPath  string
}

// NewTracer creates a new tracer
func NewTracer(tracesPath string, enabled bool) *Tracer {
	// Create traces directory if it doesn't exist
	if enabled && tracesPath != "" {
		os.MkdirAll(tracesPath, 0755)
	}

	return &Tracer{
		spans:      make(map[string]*Span),
		enabled:    enabled,
		tracesPath: tracesPath,
	}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, attributes map[string]interface{}) (context.Context, string) {
	if !t.enabled {
		return ctx, ""
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Generate span and trace IDs
	spanID := generateID()
	
	// Extract parent span ID from context if it exists
	var traceID, parentID string
	parentSpanID := ctx.Value("span_id")
	if parentSpanID != nil {
		parentID = parentSpanID.(string)
		traceID = ctx.Value("trace_id").(string)
	} else {
		// This is a root span, generate a new trace ID
		traceID = generateID()
	}

	// Create the span
	span := &Span{
		Name: name,
		Context: SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
		ParentID:   parentID,
		StartTime:  time.Now(),
		Attributes: attributes,
		Status: SpanStatus{
			Code: 0, // Unset
		},
		Events: make([]SpanEvent, 0),
	}

	t.spans[spanID] = span
	t.activeSpans.Store(spanID, span)

	// Create a new context with span information
	newCtx := context.WithValue(ctx, "span_id", spanID)
	newCtx = context.WithValue(newCtx, "trace_id", traceID)

	return newCtx, spanID
}

// EndSpan ends a span and computes its duration
func (t *Tracer) EndSpan(spanID string, status SpanStatus) {
	if !t.enabled || spanID == "" {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	// Record end time and duration
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime).Milliseconds()
	span.Status = status

	// Remove from active spans
	t.activeSpans.Delete(spanID)

	// Determine if this is a root span (no parent ID)
	if span.ParentID == "" {
		// This is a root span, write the entire trace to disk
		t.exportTrace(span.Context.TraceID)
	}
}

// AddEvent adds an event to a span
func (t *Tracer) AddEvent(spanID string, name string, attributes map[string]interface{}) {
	if !t.enabled || spanID == "" {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)
}

// AddAttribute adds or updates an attribute on a span
func (t *Tracer) AddAttribute(spanID string, key string, value interface{}) {
	if !t.enabled || spanID == "" {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	if span.Attributes == nil {
		span.Attributes = make(map[string]interface{})
	}

	span.Attributes[key] = value
}

// SetStatus sets the status of a span
func (t *Tracer) SetStatus(spanID string, code int, message string) {
	if !t.enabled || spanID == "" {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	span.Status = SpanStatus{
		Code:    code,
		Message: message,
	}
}

// GetActiveSpans returns all active spans
func (t *Tracer) GetActiveSpans() []*Span {
	if !t.enabled {
		return nil
	}

	var activeSpans []*Span
	t.activeSpans.Range(func(key, value interface{}) bool {
		activeSpans = append(activeSpans, value.(*Span))
		return true
	})

	return activeSpans
}

// exportTrace writes a complete trace to disk
func (t *Tracer) exportTrace(traceID string) {
	if !t.enabled || t.tracesPath == "" {
		return
	}

	// Collect all spans for this trace
	var traceSpans []*Span
	for _, span := range t.spans {
		if span.Context.TraceID == traceID {
			traceSpans = append(traceSpans, span)
		}
	}

	if len(traceSpans) == 0 {
		return
	}

	// Generate a filename based on the trace ID and timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(t.tracesPath, fmt.Sprintf("trace-%s-%s.json", traceID[:8], timestamp))

	// Convert spans to JSON
	data, err := json.MarshalIndent(traceSpans, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal trace: %v\n", err)
		return
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write trace to file: %v\n", err)
		return
	}

	// Clean up spans that were part of this trace
	for _, span := range traceSpans {
		delete(t.spans, span.Context.SpanID)
	}
}

// generateID generates a unique ID for spans and traces
func generateID() string {
	// In a production scenario, you would use a proper algorithm to generate IDs
	// For simplicity, we'll use a timestamp-based approach here
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

// GlobalTracer is the default global tracer
var GlobalTracer *Tracer

// Initialize the global tracer
func init() {
	// Get traces path from environment variable or use default
	tracesPath := os.Getenv("CAPSULATE_TRACES_PATH")
	if tracesPath == "" {
		// Default to .capsulate/traces in current directory
		cwd, err := os.Getwd()
		if err == nil {
			tracesPath = filepath.Join(cwd, ".capsulate", "traces")
		}
	}

	// Check if tracing is disabled
	enabled := true
	if os.Getenv("CAPSULATE_TRACES_DISABLED") == "true" {
		enabled = false
	}

	// Initialize the global tracer
	GlobalTracer = NewTracer(tracesPath, enabled)
}

// StartSpan starts a new span using the global tracer
func StartSpan(ctx context.Context, name string, attributes map[string]interface{}) (context.Context, string) {
	return GlobalTracer.StartSpan(ctx, name, attributes)
}

// EndSpan ends a span using the global tracer
func EndSpan(spanID string, status SpanStatus) {
	GlobalTracer.EndSpan(spanID, status)
}

// EndSpanSuccess ends a span with success status
func EndSpanSuccess(spanID string) {
	EndSpan(spanID, SpanStatus{Code: 1}) // Ok
}

// EndSpanError ends a span with error status
func EndSpanError(spanID string, message string) {
	EndSpan(spanID, SpanStatus{Code: 2, Message: message}) // Error
}

// AddEvent adds an event to a span using the global tracer
func AddEvent(spanID string, name string, attributes map[string]interface{}) {
	GlobalTracer.AddEvent(spanID, name, attributes)
}

// AddAttribute adds or updates an attribute on a span using the global tracer
func AddAttribute(spanID string, key string, value interface{}) {
	GlobalTracer.AddAttribute(spanID, key, value)
}

// SetStatus sets the status of a span using the global tracer
func SetStatus(spanID string, code int, message string) {
	GlobalTracer.SetStatus(spanID, code, message)
}

// GetActiveSpans returns all active spans from the global tracer
func GetActiveSpans() []*Span {
	return GlobalTracer.GetActiveSpans()
}

// WithSpan is a helper function that wraps a function call with a span
func WithSpan(ctx context.Context, name string, fn func(context.Context, string) error) error {
	ctx, spanID := StartSpan(ctx, name, nil)
	err := fn(ctx, spanID)
	if err != nil {
		EndSpanError(spanID, err.Error())
		return err
	}
	EndSpanSuccess(spanID)
	return nil
} 