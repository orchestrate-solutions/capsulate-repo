package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricType represents the type of metric being recorded
type MetricType string

const (
	// DurationType represents a duration metric (e.g., operation time)
	DurationType MetricType = "duration"
	// CountType represents a count metric (e.g., number of operations)
	CountType MetricType = "count"
	// GaugeType represents a gauge metric (e.g., current resource usage)
	GaugeType MetricType = "gauge"
)

// MetricCategory represents the category of the metric
type MetricCategory string

const (
	// ContainerOps represents container-related operations
	ContainerOps MetricCategory = "container"
	// GitOps represents Git-related operations
	GitOps MetricCategory = "git"
	// FSWrites represents file system write operations
	FSWrites MetricCategory = "fs_write"
	// FSReads represents file system read operations
	FSReads MetricCategory = "fs_read"
	// DependencyOps represents dependency-related operations
	DependencyOps MetricCategory = "dependency"
	// SyncOps represents synchronization operations
	SyncOps MetricCategory = "sync"
	// ResourceUsage represents resource usage metrics
	ResourceUsage MetricCategory = "resource"
)

// Metric represents a single metric measurement
type Metric struct {
	Name      string         `json:"name"`
	Type      MetricType     `json:"type"`
	Category  MetricCategory `json:"category"`
	Value     float64        `json:"value"`
	Unit      string         `json:"unit"`
	AgentID   string         `json:"agent_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// Collector manages metric collection and storage
type Collector struct {
	metrics      []Metric
	startTimes   map[string]time.Time
	mutex        sync.Mutex
	enabled      bool
	metricsPath  string
	flushOnWrite bool
}

// NewCollector creates a new metric collector
func NewCollector(metricsPath string, enabled bool, flushOnWrite bool) *Collector {
	// Create metrics directory if it doesn't exist
	if enabled && metricsPath != "" {
		os.MkdirAll(metricsPath, 0755)
	}

	return &Collector{
		metrics:      make([]Metric, 0),
		startTimes:   make(map[string]time.Time),
		enabled:      enabled,
		metricsPath:  metricsPath,
		flushOnWrite: flushOnWrite,
	}
}

// StartTimer starts a timer for the given operation
func (c *Collector) StartTimer(name string, category MetricCategory, agentID string) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Generate a unique key for this timer
	key := fmt.Sprintf("%s:%s:%s", name, category, agentID)
	c.startTimes[key] = time.Now()
}

// StopTimer stops a timer and records the duration
func (c *Collector) StopTimer(name string, category MetricCategory, agentID string) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Generate the unique key
	key := fmt.Sprintf("%s:%s:%s", name, category, agentID)
	startTime, exists := c.startTimes[key]
	if !exists {
		return
	}

	// Calculate duration and record the metric
	duration := time.Since(startTime)
	delete(c.startTimes, key)

	metric := Metric{
		Name:      name,
		Type:      DurationType,
		Category:  category,
		Value:     float64(duration.Milliseconds()),
		Unit:      "ms",
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	c.metrics = append(c.metrics, metric)

	if c.flushOnWrite {
		c.Flush()
	}
}

// RecordCount records a count metric
func (c *Collector) RecordCount(name string, category MetricCategory, value int, agentID string) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	metric := Metric{
		Name:      name,
		Type:      CountType,
		Category:  category,
		Value:     float64(value),
		Unit:      "count",
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	c.metrics = append(c.metrics, metric)

	if c.flushOnWrite {
		c.Flush()
	}
}

// RecordGauge records a gauge metric
func (c *Collector) RecordGauge(name string, category MetricCategory, value float64, unit string, agentID string) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	metric := Metric{
		Name:      name,
		Type:      GaugeType,
		Category:  category,
		Value:     value,
		Unit:      unit,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	c.metrics = append(c.metrics, metric)

	if c.flushOnWrite {
		c.Flush()
	}
}

// GetMetrics returns all recorded metrics
func (c *Collector) GetMetrics() []Metric {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Return a copy to prevent race conditions
	metricsCopy := make([]Metric, len(c.metrics))
	copy(metricsCopy, c.metrics)
	return metricsCopy
}

// Clear clears all recorded metrics
func (c *Collector) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.metrics = make([]Metric, 0)
	c.startTimes = make(map[string]time.Time)
}

// Flush writes all metrics to disk and clears them
func (c *Collector) Flush() error {
	if !c.enabled || c.metricsPath == "" {
		return nil
	}

	c.mutex.Lock()
	metrics := c.metrics
	c.metrics = make([]Metric, 0)
	c.mutex.Unlock()

	if len(metrics) == 0 {
		return nil
	}

	// Generate a filename based on the current time
	timestamp := time.Now().Format("20060102-150405.000")
	filename := filepath.Join(c.metricsPath, fmt.Sprintf("metrics-%s.json", timestamp))

	// Convert metrics to JSON
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics to file: %v", err)
	}

	return nil
}

// Summary represents a summary of metric statistics
type Summary struct {
	Category    MetricCategory     `json:"category"`
	Operations  map[string]OpStats `json:"operations"`
	TotalCount  int                `json:"total_count"`
	MinDuration float64            `json:"min_duration,omitempty"`
	MaxDuration float64            `json:"max_duration,omitempty"`
	AvgDuration float64            `json:"avg_duration,omitempty"`
}

// OpStats represents statistics for a specific operation
type OpStats struct {
	Count       int     `json:"count"`
	MinDuration float64 `json:"min_duration,omitempty"`
	MaxDuration float64 `json:"max_duration,omitempty"`
	AvgDuration float64 `json:"avg_duration,omitempty"`
	LastValue   float64 `json:"last_value,omitempty"`
}

// Summarize generates a summary of the metrics
func (c *Collector) Summarize() map[MetricCategory]Summary {
	c.mutex.Lock()
	metrics := c.metrics
	c.mutex.Unlock()

	summaries := make(map[MetricCategory]Summary)

	// Initialize summaries
	for _, m := range metrics {
		if _, exists := summaries[m.Category]; !exists {
			summaries[m.Category] = Summary{
				Category:   m.Category,
				Operations: make(map[string]OpStats),
				TotalCount: 0,
			}
		}
	}

	// Process metrics
	for _, m := range metrics {
		summary := summaries[m.Category]
		summary.TotalCount++

		stats, exists := summary.Operations[m.Name]
		if !exists {
			stats = OpStats{
				Count:       0,
				MinDuration: -1,
				MaxDuration: 0,
				AvgDuration: 0,
			}
		}

		stats.Count++
		stats.LastValue = m.Value

		if m.Type == DurationType {
			if stats.MinDuration < 0 || m.Value < stats.MinDuration {
				stats.MinDuration = m.Value
			}
			if m.Value > stats.MaxDuration {
				stats.MaxDuration = m.Value
			}
			// Update average
			stats.AvgDuration = ((stats.AvgDuration * float64(stats.Count-1)) + m.Value) / float64(stats.Count)
		}

		summary.Operations[m.Name] = stats
		summaries[m.Category] = summary
	}

	// Calculate overall min/max/avg for each category
	for cat, summary := range summaries {
		var totalDuration float64
		var count int
		minDuration := -1.0
		maxDuration := 0.0

		for _, stats := range summary.Operations {
			if stats.MinDuration >= 0 {
				if minDuration < 0 || stats.MinDuration < minDuration {
					minDuration = stats.MinDuration
				}
				if stats.MaxDuration > maxDuration {
					maxDuration = stats.MaxDuration
				}
				totalDuration += stats.AvgDuration * float64(stats.Count)
				count += stats.Count
			}
		}

		if count > 0 {
			summary.MinDuration = minDuration
			summary.MaxDuration = maxDuration
			summary.AvgDuration = totalDuration / float64(count)
			summaries[cat] = summary
		}
	}

	return summaries
}

// GetSummaryJSON returns a JSON string containing the metric summary
func (c *Collector) GetSummaryJSON() (string, error) {
	summary := c.Summarize()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %v", err)
	}
	return string(data), nil
}

// GlobalCollector is the default global metric collector
var GlobalCollector *Collector

// Initialize the global collector
func init() {
	// Get metrics path from environment variable or use default
	metricsPath := os.Getenv("CAPSULATE_METRICS_PATH")
	if metricsPath == "" {
		// Default to .capsulate/metrics in current directory
		cwd, err := os.Getwd()
		if err == nil {
			metricsPath = filepath.Join(cwd, ".capsulate", "metrics")
		}
	}

	// Check if metrics are disabled
	enabled := true
	if os.Getenv("CAPSULATE_METRICS_DISABLED") == "true" {
		enabled = false
	}

	// Check if metrics should be flushed immediately
	flushOnWrite := false
	if os.Getenv("CAPSULATE_METRICS_FLUSH_ON_WRITE") == "true" {
		flushOnWrite = true
	}

	// Initialize the global collector
	GlobalCollector = NewCollector(metricsPath, enabled, flushOnWrite)
}

// StartTimer starts a timer for the given operation using the global collector
func StartTimer(name string, category MetricCategory, agentID string) {
	GlobalCollector.StartTimer(name, category, agentID)
}

// StopTimer stops a timer and records the duration using the global collector
func StopTimer(name string, category MetricCategory, agentID string) {
	GlobalCollector.StopTimer(name, category, agentID)
}

// RecordCount records a count metric using the global collector
func RecordCount(name string, category MetricCategory, value int, agentID string) {
	GlobalCollector.RecordCount(name, category, value, agentID)
}

// RecordGauge records a gauge metric using the global collector
func RecordGauge(name string, category MetricCategory, value float64, unit string, agentID string) {
	GlobalCollector.RecordGauge(name, category, value, unit, agentID)
}

// GetMetrics returns all recorded metrics from the global collector
func GetMetrics() []Metric {
	return GlobalCollector.GetMetrics()
}

// GetSummary returns a summary of metrics from the global collector
func GetSummary() map[MetricCategory]Summary {
	return GlobalCollector.Summarize()
}

// GetSummaryJSON returns a JSON string containing the metric summary from the global collector
func GetSummaryJSON() (string, error) {
	return GlobalCollector.GetSummaryJSON()
}

// Clear clears all recorded metrics from the global collector
func Clear() {
	GlobalCollector.Clear()
}

// Flush writes all metrics to disk and clears them from the global collector
func Flush() error {
	return GlobalCollector.Flush()
} 