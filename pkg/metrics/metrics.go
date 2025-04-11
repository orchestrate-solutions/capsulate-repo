package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricType represents different metric categories
type MetricType string

const (
	// GitOps represents Git operations metrics
	GitOps MetricType = "git_ops"
	// ContainerOps represents container operations metrics
	ContainerOps MetricType = "container_ops"
	// FileOps represents file system operations metrics
	FileOps MetricType = "file_ops"
	// DependencyOps represents dependency management operations metrics
	DependencyOps MetricType = "dependency_ops"
	// ResourceUsage represents resource usage metrics
	ResourceUsage MetricType = "resource_usage"
)

// Internal metrics storage
var (
	timers      = make(map[string]time.Time)
	timersMutex sync.Mutex
	
	counters      = make(map[string]int)
	countersMutex sync.Mutex
	
	gauges      = make(map[string]float64)
	gaugesMutex sync.Mutex
)

// StartTimer starts a timer for the specified operation
func StartTimer(operation string, metricType MetricType, agentID string) {
	key := formatKey(string(metricType), operation, agentID)
	
	timersMutex.Lock()
	defer timersMutex.Unlock()
	
	timers[key] = time.Now()
}

// StopTimer stops a timer and records the duration
func StopTimer(operation string, metricType MetricType, agentID string) time.Duration {
	key := formatKey(string(metricType), operation, agentID)
	
	timersMutex.Lock()
	startTime, exists := timers[key]
	delete(timers, key)
	timersMutex.Unlock()
	
	if !exists {
		return 0
	}
	
	duration := time.Since(startTime)
	return duration
}

// RecordCount increments a counter for the specified operation
func RecordCount(operation string, metricType MetricType, count int, agentID string) {
	key := formatKey(string(metricType), operation, agentID)
	
	countersMutex.Lock()
	defer countersMutex.Unlock()
	
	counters[key] += count
}

// RecordGauge sets a gauge value for the specified operation
func RecordGauge(operation string, metricType MetricType, value float64, unit string, agentID string) {
	key := formatKey(string(metricType), operation, agentID)
	
	gaugesMutex.Lock()
	defer gaugesMutex.Unlock()
	
	gauges[key] = value
}

// GetMetrics returns all collected metrics
func GetMetrics() map[string]interface{} {
	result := make(map[string]interface{})
	
	// Add counters
	countersMutex.Lock()
	countersCopy := make(map[string]int)
	for k, v := range counters {
		countersCopy[k] = v
	}
	countersMutex.Unlock()
	result["counters"] = countersCopy
	
	// Add gauges
	gaugesMutex.Lock()
	gaugesCopy := make(map[string]float64)
	for k, v := range gauges {
		gaugesCopy[k] = v
	}
	gaugesMutex.Unlock()
	result["gauges"] = gaugesCopy
	
	return result
}

// Clear clears all collected metrics
func Clear() {
	timersMutex.Lock()
	timers = make(map[string]time.Time)
	timersMutex.Unlock()
	
	countersMutex.Lock()
	counters = make(map[string]int)
	countersMutex.Unlock()
	
	gaugesMutex.Lock()
	gauges = make(map[string]float64)
	gaugesMutex.Unlock()
}

// formatKey creates a consistent key format for metrics
func formatKey(metricType, operation, agentID string) string {
	if agentID == "" {
		return metricType + "." + operation
	}
	return metricType + "." + operation + "." + agentID
}

// GetSummary returns a summary of metrics by category
func GetSummary() map[string]interface{} {
	metrics := GetMetrics()
	summary := make(map[string]interface{})
	
	// Group by metric type
	for metricsType, metricsData := range metrics {
		switch metricsData.(type) {
		case map[string]int:
			byType := make(map[string]map[string]int)
			for key, val := range metricsData.(map[string]int) {
				// Extract type, operation, and agent from key
				parts := splitKey(key)
				if len(parts) >= 2 {
					metricType := parts[0]
					operation := parts[1]
					
					if _, exists := byType[metricType]; !exists {
						byType[metricType] = make(map[string]int)
					}
					byType[metricType][operation] = val
				}
			}
			summary[metricsType] = byType
			
		case map[string]float64:
			byType := make(map[string]map[string]float64)
			for key, val := range metricsData.(map[string]float64) {
				// Extract type, operation, and agent from key
				parts := splitKey(key)
				if len(parts) >= 2 {
					metricType := parts[0]
					operation := parts[1]
					
					if _, exists := byType[metricType]; !exists {
						byType[metricType] = make(map[string]float64)
					}
					byType[metricType][operation] = val
				}
			}
			summary[metricsType] = byType
		}
	}
	
	return summary
}

// splitKey splits a key by dots
func splitKey(key string) []string {
	var result []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			result = append(result, key[start:i])
			start = i + 1
		}
	}
	if start < len(key) {
		result = append(result, key[start:])
	}
	return result
}

// GetSummaryJSON returns a JSON representation of the metrics summary
func GetSummaryJSON() (string, error) {
	summary := GetSummary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %v", err)
	}
	return string(data), nil
}

// Flush writes metrics to disk and clears them
func Flush() error {
	metricsPath := os.Getenv("GIT_CAPSULATE_METRICS_PATH")
	if metricsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			metricsPath = filepath.Join(homeDir, ".git-capsulate", "metrics")
		} else {
			metricsPath = filepath.Join(os.TempDir(), "git-capsulate", "metrics")
		}
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(metricsPath, 0755); err != nil {
		return fmt.Errorf("failed to create metrics directory: %v", err)
	}
	
	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(metricsPath, fmt.Sprintf("metrics-%s.json", timestamp))
	
	// Get metrics summary
	summary := GetSummary()
	
	// Add timestamp
	summary["timestamp"] = time.Now().Format(time.RFC3339)
	
	// Marshal to JSON
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}
	
	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics to file: %v", err)
	}
	
	// Clear metrics
	Clear()
	
	return nil
} 