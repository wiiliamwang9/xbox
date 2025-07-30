package monitoring

import (
	"log"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu              sync.RWMutex
	prometheus      *PrometheusMetrics
	updateInterval  time.Duration
	stopChan        chan struct{}
	
	// 缓存的指标数据
	lastCPUUsage    float64
	lastMemoryUsage float64
	lastDiskUsage   float64
	isOnline        bool
	singboxRunning  bool
	connectionCount int
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(updateInterval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		prometheus:     NewPrometheusMetrics(),
		updateInterval: updateInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start 启动指标收集
func (mc *MetricsCollector) Start() {
	go mc.collectLoop()
}

// Stop 停止指标收集
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// GetPrometheusMetrics 获取Prometheus指标实例
func (mc *MetricsCollector) GetPrometheusMetrics() *PrometheusMetrics {
	return mc.prometheus
}

// collectLoop 指标收集循环
func (mc *MetricsCollector) collectLoop() {
	ticker := time.NewTicker(mc.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.collectMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

// collectMetrics 收集指标
func (mc *MetricsCollector) collectMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// 收集系统指标
	mc.collectSystemMetrics()
	
	// 更新Prometheus指标
	mc.prometheus.UpdateSystemMetrics(
		mc.lastCPUUsage,
		mc.lastMemoryUsage,
		mc.lastDiskUsage,
	)
	
	mc.prometheus.SetAgentStatus(mc.isOnline)
	mc.prometheus.SetSingboxStatus(mc.singboxRunning)
	mc.prometheus.SetActiveConnections(float64(mc.connectionCount))
}

// collectSystemMetrics 收集系统指标
func (mc *MetricsCollector) collectSystemMetrics() {
	// CPU使用率（简化实现）
	mc.lastCPUUsage = mc.getCPUUsage()
	
	// 内存使用率
	mc.lastMemoryUsage = mc.getMemoryUsage()
	
	// 磁盘使用率（简化实现）
	mc.lastDiskUsage = mc.getDiskUsage()
}

// getCPUUsage 获取CPU使用率
func (mc *MetricsCollector) getCPUUsage() float64 {
	// 简化实现，实际应用中需要更精确的CPU使用率计算
	return 15.5
}

// getMemoryUsage 获取内存使用量
func (mc *MetricsCollector) getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc)
}

// getDiskUsage 获取磁盘使用率
func (mc *MetricsCollector) getDiskUsage() float64 {
	// 简化实现，实际应用中需要读取系统磁盘信息
	return 45.2
}

// UpdateAgentStatus 更新Agent状态
func (mc *MetricsCollector) UpdateAgentStatus(online bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.isOnline = online
}

// UpdateSingboxStatus 更新sing-box状态
func (mc *MetricsCollector) UpdateSingboxStatus(running bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.singboxRunning = running
}

// UpdateConnectionCount 更新连接数
func (mc *MetricsCollector) UpdateConnectionCount(count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.connectionCount = count
}

// RecordHeartbeat 记录心跳
func (mc *MetricsCollector) RecordHeartbeat() {
	mc.prometheus.IncrementHeartbeat()
}

// RecordConfigUpdate 记录配置更新
func (mc *MetricsCollector) RecordConfigUpdate() {
	mc.prometheus.IncrementConfigUpdates()
}

// RecordRuleUpdate 记录规则更新
func (mc *MetricsCollector) RecordRuleUpdate() {
	mc.prometheus.IncrementRuleUpdates()
}

// RecordSingboxRestart 记录sing-box重启
func (mc *MetricsCollector) RecordSingboxRestart() {
	mc.prometheus.IncrementSingboxRestarts()
}

// RecordError 记录错误
func (mc *MetricsCollector) RecordError(errorType, component string) {
	mc.prometheus.IncrementError(errorType, component)
	log.Printf("Error recorded: type=%s, component=%s", errorType, component)
}

// RecordNetworkTraffic 记录网络流量
func (mc *MetricsCollector) RecordNetworkTraffic(received, sent int64) {
	mc.prometheus.AddNetworkTraffic(float64(received), float64(sent))
}

// GetMetricsSnapshot 获取指标快照
func (mc *MetricsCollector) GetMetricsSnapshot() map[string]string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	return map[string]string{
		"cpu_usage":        strconv.FormatFloat(mc.lastCPUUsage, 'f', 2, 64),
		"memory_usage":     strconv.FormatFloat(mc.lastMemoryUsage, 'f', 0, 64),
		"disk_usage":       strconv.FormatFloat(mc.lastDiskUsage, 'f', 2, 64),
		"agent_online":     strconv.FormatBool(mc.isOnline),
		"singbox_running":  strconv.FormatBool(mc.singboxRunning),
		"connection_count": strconv.Itoa(mc.connectionCount),
		"timestamp":        time.Now().Format(time.RFC3339),
	}
}