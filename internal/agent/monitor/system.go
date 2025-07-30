package monitor

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemMonitor 系统监控器
type SystemMonitor struct {
}

// NewSystemMonitor 创建系统监控器
func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{}
}

// GetLocalIP 获取本地IP地址
func (m *SystemMonitor) GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}

// GetCPUUsage 获取CPU使用率（简化实现）
func (m *SystemMonitor) GetCPUUsage() string {
	// 简化的CPU使用率获取，实际项目中应使用更精确的方法
	return "0.0"
}

// GetMemoryUsage 获取内存使用率
func (m *SystemMonitor) GetMemoryUsage() string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	
	// 转换为MB
	usedMB := mem.Alloc / 1024 / 1024
	return fmt.Sprintf("%.2f", float64(usedMB))
}

// GetDiskUsage 获取磁盘使用率（简化实现）
func (m *SystemMonitor) GetDiskUsage() string {
	// 简化的磁盘使用率获取
	return "0.0"
}

// GetSystemInfo 获取系统信息
func (m *SystemMonitor) GetSystemInfo() map[string]string {
	hostname, _ := os.Hostname()
	
	return map[string]string{
		"hostname":    hostname,
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"go_version":  runtime.Version(),
		"cpu_cores":   strconv.Itoa(runtime.NumCPU()),
		"goroutines":  strconv.Itoa(runtime.NumGoroutine()),
	}
}

// CollectMetrics 收集所有监控指标
func (m *SystemMonitor) CollectMetrics() map[string]string {
	metrics := map[string]string{
		"cpu_usage":    m.GetCPUUsage(),
		"memory_usage": m.GetMemoryUsage(),
		"disk_usage":   m.GetDiskUsage(),
		"timestamp":    time.Now().Format(time.RFC3339),
		"uptime":       m.getUptime(),
	}
	
	// 合并系统信息
	for k, v := range m.GetSystemInfo() {
		metrics[k] = v
	}
	
	return metrics
}

// getUptime 获取系统运行时间（简化实现）
func (m *SystemMonitor) getUptime() string {
	// 读取 /proc/uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			if uptime, err := strconv.ParseFloat(parts[0], 64); err == nil {
				return fmt.Sprintf("%.0f", uptime)
			}
		}
	}
	return "unknown"
}