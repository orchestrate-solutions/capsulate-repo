package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/your-org/capsulate-repo/pkg/metrics"
)

// ContainerStats represents statistics for a single container
type ContainerStats struct {
	ContainerID   string    `json:"container_id"`
	AgentID       string    `json:"agent_id"`
	CPUUsage      float64   `json:"cpu_usage_percent"`
	MemoryUsage   int64     `json:"memory_usage_bytes"`
	MemoryLimit   int64     `json:"memory_limit_bytes"`
	MemoryPercent float64   `json:"memory_usage_percent"`
	DiskRead      int64     `json:"disk_read_bytes"`
	DiskWrite     int64     `json:"disk_write_bytes"`
	NetRx         int64     `json:"network_rx_bytes"`
	NetTx         int64     `json:"network_tx_bytes"`
	Timestamp     time.Time `json:"timestamp"`
}

// Monitor monitors resource usage of Docker containers
type Monitor struct {
	dockerClient   *client.Client
	containerStats map[string]*ContainerStats
	mutex          sync.RWMutex
	interval       time.Duration
	stopChan       chan struct{}
	running        bool
}

// NewMonitor creates a new container monitor
func NewMonitor(interval time.Duration) (*Monitor, error) {
	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}

	monitor := &Monitor{
		dockerClient:   dockerClient,
		containerStats: make(map[string]*ContainerStats),
		interval:       interval,
		stopChan:       make(chan struct{}),
		running:        false,
	}

	return monitor, nil
}

// Start starts the monitoring process
func (m *Monitor) Start() {
	m.mutex.Lock()
	if m.running {
		m.mutex.Unlock()
		return
	}
	m.running = true
	m.mutex.Unlock()

	go m.monitorLoop()
}

// Stop stops the monitoring process
func (m *Monitor) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return
	}

	m.stopChan <- struct{}{}
	m.running = false
}

// GetContainerStats returns statistics for a specific container
func (m *Monitor) GetContainerStats(containerID string) (*ContainerStats, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats, exists := m.containerStats[containerID]
	return stats, exists
}

// GetAllContainerStats returns statistics for all monitored containers
func (m *Monitor) GetAllContainerStats() map[string]*ContainerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to prevent race conditions
	statsCopy := make(map[string]*ContainerStats, len(m.containerStats))
	for id, stats := range m.containerStats {
		// Make a copy of stats
		statsCopy[id] = &ContainerStats{
			ContainerID:   stats.ContainerID,
			AgentID:       stats.AgentID,
			CPUUsage:      stats.CPUUsage,
			MemoryUsage:   stats.MemoryUsage,
			MemoryLimit:   stats.MemoryLimit,
			MemoryPercent: stats.MemoryPercent,
			DiskRead:      stats.DiskRead,
			DiskWrite:     stats.DiskWrite,
			NetRx:         stats.NetRx,
			NetTx:         stats.NetTx,
			Timestamp:     stats.Timestamp,
		}
	}
	return statsCopy
}

// GetContainerStatsByAgentID returns statistics for containers belonging to a specific agent
func (m *Monitor) GetContainerStatsByAgentID(agentID string) []*ContainerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var containerStats []*ContainerStats
	for _, stats := range m.containerStats {
		if stats.AgentID == agentID {
			// Make a copy of stats
			statsCopy := &ContainerStats{
				ContainerID:   stats.ContainerID,
				AgentID:       stats.AgentID,
				CPUUsage:      stats.CPUUsage,
				MemoryUsage:   stats.MemoryUsage,
				MemoryLimit:   stats.MemoryLimit,
				MemoryPercent: stats.MemoryPercent,
				DiskRead:      stats.DiskRead,
				DiskWrite:     stats.DiskWrite,
				NetRx:         stats.NetRx,
				NetTx:         stats.NetTx,
				Timestamp:     stats.Timestamp,
			}
			containerStats = append(containerStats, statsCopy)
		}
	}
	return containerStats
}

// monitorLoop periodically collects statistics for all containers
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectStats()
		case <-m.stopChan:
			return
		}
	}
}

// collectStats collects statistics for all containers
func (m *Monitor) collectStats() {
	ctx := context.Background()

	// Get list of running containers
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		fmt.Printf("Failed to list containers: %v\n", err)
		return
	}

	// Collect stats for each container
	for _, container := range containers {
		// Only monitor git-capsulate containers
		if !isCapsulateContainer(container.Names) {
			continue
		}

		// Extract the agent ID from the container name
		agentID := extractAgentID(container.Names)

		// Get container stats
		stats, err := m.dockerClient.ContainerStats(ctx, container.ID, false)
		if err != nil {
			fmt.Printf("Failed to get stats for container %s: %v\n", container.ID, err)
			continue
		}

		// Parse container stats
		var statsJSON types.StatsJSON
		if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
			fmt.Printf("Failed to decode stats for container %s: %v\n", container.ID, err)
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		// Calculate CPU usage percentage
		cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
		cpuPercent := 0.0
		if systemDelta > 0.0 && cpuDelta > 0.0 {
			cpuPercent = (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0
		}

		// Calculate memory usage percentage
		memoryPercent := 0.0
		if statsJSON.MemoryStats.Limit > 0 {
			memoryPercent = float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100.0
		}

		// Store container stats
		containerStats := &ContainerStats{
			ContainerID:   container.ID,
			AgentID:       agentID,
			CPUUsage:      cpuPercent,
			MemoryUsage:   int64(statsJSON.MemoryStats.Usage),
			MemoryLimit:   int64(statsJSON.MemoryStats.Limit),
			MemoryPercent: memoryPercent,
			DiskRead:      int64(statsJSON.BlkioStats.IoServiceBytesRecursive[0].Value),
			DiskWrite:     int64(statsJSON.BlkioStats.IoServiceBytesRecursive[1].Value),
			NetRx:         int64(statsJSON.Networks["eth0"].RxBytes),
			NetTx:         int64(statsJSON.Networks["eth0"].TxBytes),
			Timestamp:     time.Now(),
		}

		m.mutex.Lock()
		m.containerStats[container.ID] = containerStats
		m.mutex.Unlock()

		// Record metrics
		metrics.RecordGauge("cpu_usage", metrics.ResourceUsage, cpuPercent, "percent", agentID)
		metrics.RecordGauge("memory_usage", metrics.ResourceUsage, float64(statsJSON.MemoryStats.Usage), "bytes", agentID)
		metrics.RecordGauge("memory_percent", metrics.ResourceUsage, memoryPercent, "percent", agentID)
		metrics.RecordGauge("disk_read", metrics.ResourceUsage, float64(containerStats.DiskRead), "bytes", agentID)
		metrics.RecordGauge("disk_write", metrics.ResourceUsage, float64(containerStats.DiskWrite), "bytes", agentID)
		metrics.RecordGauge("net_rx", metrics.ResourceUsage, float64(containerStats.NetRx), "bytes", agentID)
		metrics.RecordGauge("net_tx", metrics.ResourceUsage, float64(containerStats.NetTx), "bytes", agentID)
	}
}

// isCapsulateContainer checks if a container is a git-capsulate container
func isCapsulateContainer(names []string) bool {
	for _, name := range names {
		if len(name) > 0 && len(name) > 11 && name[:11] == "/capsulate-" {
			return true
		}
	}
	return false
}

// extractAgentID extracts the agent ID from container names
func extractAgentID(names []string) string {
	for _, name := range names {
		if len(name) > 0 && len(name) > 11 && name[:11] == "/capsulate-" {
			return name[11:] // Extract the part after "/capsulate-"
		}
	}
	return ""
}

// GlobalMonitor is the default global container monitor
var GlobalMonitor *Monitor

// Initialize the global monitor
func init() {
	// Get monitoring interval from environment variable or use default
	intervalStr := os.Getenv("CAPSULATE_MONITOR_INTERVAL")
	interval := 5 * time.Second // Default interval: 5 seconds
	if intervalStr != "" {
		if parsedInterval, err := time.ParseDuration(intervalStr); err == nil {
			interval = parsedInterval
		}
	}

	// Create global monitor
	monitor, err := NewMonitor(interval)
	if err != nil {
		fmt.Printf("Failed to create container monitor: %v\n", err)
		return
	}

	GlobalMonitor = monitor

	// Start monitoring if enabled
	if os.Getenv("CAPSULATE_MONITOR_DISABLED") != "true" {
		GlobalMonitor.Start()
	}
}

// Start starts the global monitor
func Start() {
	if GlobalMonitor != nil {
		GlobalMonitor.Start()
	}
}

// Stop stops the global monitor
func Stop() {
	if GlobalMonitor != nil {
		GlobalMonitor.Stop()
	}
}

// GetContainerStats returns statistics for a specific container using the global monitor
func GetContainerStats(containerID string) (*ContainerStats, bool) {
	if GlobalMonitor != nil {
		return GlobalMonitor.GetContainerStats(containerID)
	}
	return nil, false
}

// GetAllContainerStats returns statistics for all monitored containers using the global monitor
func GetAllContainerStats() map[string]*ContainerStats {
	if GlobalMonitor != nil {
		return GlobalMonitor.GetAllContainerStats()
	}
	return nil
}

// GetContainerStatsByAgentID returns statistics for containers belonging to a specific agent using the global monitor
func GetContainerStatsByAgentID(agentID string) []*ContainerStats {
	if GlobalMonitor != nil {
		return GlobalMonitor.GetContainerStatsByAgentID(agentID)
	}
	return nil
} 