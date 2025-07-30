package monitoring

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics Prometheus指标收集器
type PrometheusMetrics struct {
	registry *prometheus.Registry
	
	// 系统指标
	cpuUsage    prometheus.Gauge
	memoryUsage prometheus.Gauge
	diskUsage   prometheus.Gauge
	
	// Agent指标
	agentStatus     prometheus.Gauge
	heartbeatCount  prometheus.Counter
	configUpdates   prometheus.Counter
	ruleUpdates     prometheus.Counter
	
	// sing-box指标
	singboxStatus   prometheus.Gauge
	singboxRestarts prometheus.Counter
	connections     prometheus.Gauge
	
	// 网络指标
	bytesReceived prometheus.Counter
	bytesSent     prometheus.Counter
	
	// 错误指标
	errors prometheus.CounterVec
}

// NewPrometheusMetrics 创建Prometheus指标收集器
func NewPrometheusMetrics() *PrometheusMetrics {
	registry := prometheus.NewRegistry()
	
	pm := &PrometheusMetrics{
		registry: registry,
		
		// 系统指标
		cpuUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_system_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		}),
		memoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_system_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		}),
		diskUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_system_disk_usage_percent",
			Help: "Current disk usage percentage",
		}),
		
		// Agent指标
		agentStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_agent_status",
			Help: "Agent status (1=online, 0=offline)",
		}),
		heartbeatCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_agent_heartbeat_total",
			Help: "Total number of heartbeats sent",
		}),
		configUpdates: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_agent_config_updates_total",
			Help: "Total number of configuration updates",
		}),
		ruleUpdates: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_agent_rule_updates_total",
			Help: "Total number of rule updates",
		}),
		
		// sing-box指标
		singboxStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_singbox_status",
			Help: "sing-box status (1=running, 0=stopped)",
		}),
		singboxRestarts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_singbox_restarts_total",
			Help: "Total number of sing-box restarts",
		}),
		connections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "xbox_singbox_connections_active",
			Help: "Number of active connections",
		}),
		
		// 网络指标
		bytesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_network_bytes_received_total",
			Help: "Total bytes received",
		}),
		bytesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "xbox_network_bytes_sent_total",
			Help: "Total bytes sent",
		}),
		
		// 错误指标
		errors: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "xbox_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"type", "component"},
		),
	}
	
	// 注册所有指标
	registry.MustRegister(
		pm.cpuUsage,
		pm.memoryUsage,
		pm.diskUsage,
		pm.agentStatus,
		pm.heartbeatCount,
		pm.configUpdates,
		pm.ruleUpdates,
		pm.singboxStatus,
		pm.singboxRestarts,
		pm.connections,
		pm.bytesReceived,
		pm.bytesSent,
		&pm.errors,
	)
	
	return pm
}

// UpdateSystemMetrics 更新系统指标
func (pm *PrometheusMetrics) UpdateSystemMetrics(cpuUsage, memoryUsage, diskUsage float64) {
	pm.cpuUsage.Set(cpuUsage)
	pm.memoryUsage.Set(memoryUsage)
	pm.diskUsage.Set(diskUsage)
}

// SetAgentStatus 设置Agent状态
func (pm *PrometheusMetrics) SetAgentStatus(online bool) {
	if online {
		pm.agentStatus.Set(1)
	} else {
		pm.agentStatus.Set(0)
	}
}

// IncrementHeartbeat 增加心跳计数
func (pm *PrometheusMetrics) IncrementHeartbeat() {
	pm.heartbeatCount.Inc()
}

// IncrementConfigUpdates 增加配置更新计数
func (pm *PrometheusMetrics) IncrementConfigUpdates() {
	pm.configUpdates.Inc()
}

// IncrementRuleUpdates 增加规则更新计数
func (pm *PrometheusMetrics) IncrementRuleUpdates() {
	pm.ruleUpdates.Inc()
}

// SetSingboxStatus 设置sing-box状态
func (pm *PrometheusMetrics) SetSingboxStatus(running bool) {
	if running {
		pm.singboxStatus.Set(1)
	} else {
		pm.singboxStatus.Set(0)
	}
}

// IncrementSingboxRestarts 增加sing-box重启计数
func (pm *PrometheusMetrics) IncrementSingboxRestarts() {
	pm.singboxRestarts.Inc()
}

// SetActiveConnections 设置活跃连接数
func (pm *PrometheusMetrics) SetActiveConnections(count float64) {
	pm.connections.Set(count)
}

// AddNetworkTraffic 添加网络流量
func (pm *PrometheusMetrics) AddNetworkTraffic(received, sent float64) {
	pm.bytesReceived.Add(received)
	pm.bytesSent.Add(sent)
}

// IncrementError 增加错误计数
func (pm *PrometheusMetrics) IncrementError(errorType, component string) {
	pm.errors.WithLabelValues(errorType, component).Inc()
}

// StartMetricsServer 启动Prometheus指标服务器
func (pm *PrometheusMetrics) StartMetricsServer(port int) error {
	handler := promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	return server.ListenAndServe()
}

// GetHandler 获取HTTP处理器
func (pm *PrometheusMetrics) GetHandler() http.Handler {
	return promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{})
}