package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xbox/sing-box-manager/internal/agent/filter"
	"github.com/xbox/sing-box-manager/internal/agent/monitor"
	"github.com/xbox/sing-box-manager/internal/agent/network"
	"github.com/xbox/sing-box-manager/internal/agent/singbox"
	"github.com/xbox/sing-box-manager/internal/agent/uninstall"
	"github.com/xbox/sing-box-manager/internal/config"
	pb "github.com/xbox/sing-box-manager/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC客户端
type Client struct {
	config           *config.Config
	conn             *grpc.ClientConn
	client           pb.AgentServiceClient
	agentID          string
	token            string
	registered       bool
	monitor          *monitor.SystemMonitor
	singboxMgr       *singbox.Manager
	filterMgr        *filter.FilterManager
	ipRangeDetector  *network.IPRangeDetector
	uninstallManager *uninstall.UninstallManager
}

// NewClient 创建gRPC客户端实例
func NewClient(cfg *config.Config) *Client {
	// 生成Agent ID（如果配置中未指定）
	agentID := cfg.Agent.ID
	if agentID == "" {
		hostname, _ := os.Hostname()
		agentID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	}

	// 创建sing-box管理器
	singboxMgr := singbox.NewManager(
		cfg.Agent.SingBoxBinary,
		cfg.Agent.SingBoxConfig,
	)
	
	// 创建过滤器管理器
	filterMgr := filter.NewFilterManager("./configs/filter.json")
	
	// 创建IP段检测器
	ipRangeDetector := network.NewIPRangeDetector()
	
	// 创建卸载管理器
	uninstallManager := uninstall.NewUninstallManager(
		cfg.Agent.SingBoxBinary,
		cfg.Agent.SingBoxConfig,
	)
	
	return &Client{
		config:           cfg,
		agentID:          agentID,
		monitor:          monitor.NewSystemMonitor(),
		singboxMgr:       singboxMgr,
		filterMgr:        filterMgr,
		ipRangeDetector:  ipRangeDetector,
		uninstallManager: uninstallManager,
	}
}

// Connect 连接到Controller
func (c *Client) Connect() error {
	var err error
	
	// 创建gRPC连接
	c.conn, err = grpc.Dial(c.config.Agent.ControllerAddr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("连接Controller失败: %v", err)
	}

	c.client = pb.NewAgentServiceClient(c.conn)
	log.Printf("已连接到Controller: %s", c.config.Agent.ControllerAddr)
	
	return nil
}

// Register 注册Agent到Controller
func (c *Client) Register() error {
	if c.client == nil {
		return fmt.Errorf("gRPC客户端未初始化")
	}

	hostname, _ := os.Hostname()
	
	systemInfo := c.monitor.GetSystemInfo()
	
	// 检测IP段信息
	log.Printf("正在检测节点IP段信息...")
	ipRangeInfo, err := c.ipRangeDetector.DetectIPRange()
	if err != nil {
		log.Printf("IP段检测失败: %v", err)
		// 不阻塞注册流程，创建一个空的IP段信息
		ipRangeInfo = &network.IPRangeInfo{
			Country: "Unknown",
			Region:  "Unknown", 
			City:    "Unknown",
			ISP:     "Unknown",
		}
	} else {
		log.Printf("IP段检测完成:")
		log.Printf("  IP段: %s", ipRangeInfo.IPRange)
		log.Printf("  国家: %s", ipRangeInfo.Country)
		log.Printf("  地区: %s", ipRangeInfo.Region)
		log.Printf("  城市: %s", ipRangeInfo.City)
		log.Printf("  运营商: %s", ipRangeInfo.ISP)
		log.Printf("  检测方法: %s", ipRangeInfo.DetectionMethod)
		log.Printf("  检测时间: %s", ipRangeInfo.DetectedAt)
	}
	
	req := &pb.RegisterRequest{
		AgentId:   c.agentID,
		Hostname:  hostname,
		IpAddress: c.monitor.GetLocalIP(),
		Version:   "1.0.0",
		Metadata:  systemInfo,
		IpRangeInfo: &pb.IPRangeInfo{
			IpRange:        ipRangeInfo.IPRange,
			Country:        ipRangeInfo.Country,
			Region:         ipRangeInfo.Region,
			City:           ipRangeInfo.City,
			Isp:            ipRangeInfo.ISP,
			DetectionMethod: ipRangeInfo.DetectionMethod,
			DetectedAt:     ipRangeInfo.DetectedAt,
		},
	}
	req.Metadata["started"] = time.Now().Format(time.RFC3339)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("注册Agent失败: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("注册失败: %s", resp.Message)
	}

	c.token = resp.Token
	c.registered = true
	log.Printf("Agent注册成功: ID=%s, Token=%s", c.agentID, c.token)
	
	return nil
}

// SendHeartbeat 发送心跳
func (c *Client) SendHeartbeat() error {
	if !c.registered {
		return fmt.Errorf("Agent未注册")
	}

	req := &pb.HeartbeatRequest{
		AgentId: c.agentID,
		Status:  "online",
		Metrics: c.monitor.CollectMetrics(),
	}

	// 检查IP段信息是否有变化（可选发送）
	currentIPInfo, err := c.ipRangeDetector.DetectIPRange()
	if err == nil {
		cachedInfo := c.ipRangeDetector.GetCachedInfo()
		// 如果是新信息或者IP段有变化，则发送IP段信息
		if cachedInfo == nil || cachedInfo.IPRange != currentIPInfo.IPRange {
			log.Printf("检测到IP段变化，将在心跳中上报新的IP段信息: %s", currentIPInfo.IPRange)
			req.IpRangeInfo = &pb.IPRangeInfo{
				IpRange:        currentIPInfo.IPRange,
				Country:        currentIPInfo.Country,
				Region:         currentIPInfo.Region,
				City:           currentIPInfo.City,
				Isp:            currentIPInfo.ISP,
				DetectionMethod: currentIPInfo.DetectionMethod,
				DetectedAt:     currentIPInfo.DetectedAt,
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("发送心跳失败: %v", err)
	}

	if !resp.Success {
		log.Printf("心跳失败: %s", resp.Message)
		// 如果心跳失败，可能需要重新注册
		c.registered = false
		return fmt.Errorf("心跳失败: %s", resp.Message)
	}

	log.Printf("心跳成功，下次间隔: %d秒", resp.NextHeartbeatInterval)
	return nil
}

// StartHeartbeat 启动心跳循环
func (c *Client) StartHeartbeat() {
	interval := time.Duration(c.config.Agent.HeartbeatInterval) * time.Second
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.SendHeartbeat(); err != nil {
				log.Printf("心跳错误: %v", err)
				// 尝试重新注册
				if !c.registered {
					if err := c.Register(); err != nil {
						log.Printf("重新注册失败: %v", err)
					}
				}
			}
		}
	}
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetAgentID 获取Agent ID
func (c *Client) GetAgentID() string {
	return c.agentID
}

// IsRegistered 检查是否已注册
func (c *Client) IsRegistered() bool {
	return c.registered
}

// UpdateConfig 处理配置更新
func (c *Client) UpdateConfig(configData string) error {
	var config singbox.Config
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}

	if err := c.singboxMgr.UpdateConfig(&config); err != nil {
		return fmt.Errorf("更新配置失败: %v", err)
	}

	log.Printf("配置更新成功")
	return nil
}

// UpdateRules 处理规则更新
func (c *Client) UpdateRules(rules []*pb.Rule) error {
	log.Printf("收到规则更新，共 %d 条规则", len(rules))
	
	// 处理规则内容，这里简化处理
	// 实际应用中需要根据rule.Type和rule.Content来解析和应用具体规则
	for _, rule := range rules {
		log.Printf("处理规则 ID=%s, Type=%s, Priority=%d, Enabled=%t", 
			rule.Id, rule.Type, rule.Priority, rule.Enabled)
		
		if rule.Enabled {
			// 根据规则类型处理
			switch rule.Type {
			case "route":
				// 处理路由规则
				log.Printf("应用路由规则: %s", rule.Content)
			case "dns":
				// 处理DNS规则
				log.Printf("应用DNS规则: %s", rule.Content)
			case "inbound":
				// 处理入站规则
				log.Printf("应用入站规则: %s", rule.Content)
			case "outbound":
				// 处理出站规则
				log.Printf("应用出站规则: %s", rule.Content)
			default:
				log.Printf("未知规则类型: %s", rule.Type)
			}
		}
	}

	log.Printf("规则更新成功")
	return nil
}

// GetStatus 获取Agent状态
func (c *Client) GetStatus() map[string]string {
	status := c.monitor.CollectMetrics()
	
	// 添加sing-box状态
	singboxStatus := c.singboxMgr.GetStatus()
	for k, v := range singboxStatus {
		status["singbox_"+k] = v
	}
	
	// 添加Agent状态
	status["agent_id"] = c.agentID
	status["registered"] = fmt.Sprintf("%t", c.registered)
	status["controller_addr"] = c.config.Agent.ControllerAddr
	
	return status
}

// StartSingbox 启动sing-box服务
func (c *Client) StartSingbox() error {
	return c.singboxMgr.Start()
}

// StopSingbox 停止sing-box服务
func (c *Client) StopSingbox() error {
	return c.singboxMgr.Stop()
}

// RestartSingbox 重启sing-box服务
func (c *Client) RestartSingbox() error {
	return c.singboxMgr.Restart()
}

// UpdateBlacklist 更新黑名单
func (c *Client) UpdateBlacklist(protocol string, domains, ips, ports []string, operation string) error {
	if err := c.filterMgr.UpdateBlacklist(protocol, domains, ips, ports, operation); err != nil {
		return fmt.Errorf("更新黑名单失败: %v", err)
	}
	
	// 重新生成sing-box配置并重启
	if err := c.regenerateSingboxConfig(); err != nil {
		return fmt.Errorf("重新生成配置失败: %v", err)
	}
	
	log.Printf("黑名单更新成功: protocol=%s, operation=%s", protocol, operation)
	return nil
}

// UpdateWhitelist 更新白名单
func (c *Client) UpdateWhitelist(protocol string, domains, ips, ports []string, operation string) error {
	if err := c.filterMgr.UpdateWhitelist(protocol, domains, ips, ports, operation); err != nil {
		return fmt.Errorf("更新白名单失败: %v", err)
	}
	
	// 重新生成sing-box配置并重启
	if err := c.regenerateSingboxConfig(); err != nil {
		return fmt.Errorf("重新生成配置失败: %v", err)
	}
	
	log.Printf("白名单更新成功: protocol=%s, operation=%s", protocol, operation)
	return nil
}

// GetFilterConfig 获取过滤器配置
func (c *Client) GetFilterConfig(protocol string) map[string]*filter.ProtocolFilter {
	if protocol == "" {
		return c.filterMgr.GetAllFilters()
	}
	
	result := make(map[string]*filter.ProtocolFilter)
	if filter, exists := c.filterMgr.GetFilter(protocol); exists {
		result[protocol] = filter
	}
	
	return result
}

// RollbackConfig 回滚配置
func (c *Client) RollbackConfig(targetVersion, reason string) error {
	log.Printf("开始配置回滚: target_version=%s, reason=%s", targetVersion, reason)
	
	if err := c.filterMgr.Rollback(targetVersion); err != nil {
		return fmt.Errorf("回滚过滤器配置失败: %v", err)
	}
	
	// 重新生成sing-box配置并重启
	if err := c.regenerateSingboxConfig(); err != nil {
		return fmt.Errorf("重新生成配置失败: %v", err)
	}
	
	log.Printf("配置回滚成功: target_version=%s", targetVersion)
	return nil
}

// regenerateSingboxConfig 重新生成sing-box配置
func (c *Client) regenerateSingboxConfig() error {
	// 获取过滤器规则
	filterRules := c.filterMgr.GenerateRouteRules()
	
	// 读取基础配置模板
	baseConfig, err := c.loadBaseSingboxConfig()
	if err != nil {
		return fmt.Errorf("加载基础配置失败: %v", err)
	}
	
	// 合并过滤器规则到路由配置中
	if baseConfig.Route == nil {
		baseConfig.Route = &singbox.RouteConfig{}
	}
	
	// 添加过滤器规则到现有规则前面（优先级更高）
	existingRules := baseConfig.Route.Rules
	newRules := make([]singbox.RouteRule, 0)
	
	// 添加过滤器生成的规则
	for _, rule := range filterRules {
		routeRule := singbox.RouteRule{
			Outbound: rule["outbound"].(string),
		}
		
		if domains, ok := rule["domain"].([]string); ok {
			routeRule.Domain = domains
		}
		if ips, ok := rule["ip"].([]string); ok {
			routeRule.IP = ips
		}
		if port, ok := rule["port"].(string); ok {
			routeRule.Port = []string{port}
		} else if ports, ok := rule["port"].([]string); ok {
			routeRule.Port = ports
		}
		
		newRules = append(newRules, routeRule)
	}
	
	// 添加原有规则
	newRules = append(newRules, existingRules...)
	baseConfig.Route.Rules = newRules
	
	// 更新sing-box配置
	return c.singboxMgr.UpdateConfig(baseConfig)
}

// loadBaseSingboxConfig 加载基础sing-box配置
func (c *Client) loadBaseSingboxConfig() (*singbox.Config, error) {
	// 这里可以从模板文件加载基础配置
	// 或者从当前配置中提取基础部分
	current := c.singboxMgr.GetConfig()
	if current != nil {
		return current, nil
	}
	
	// 返回默认配置
	return &singbox.Config{
		Log: &singbox.LogConfig{
			Level:     "info",
			Timestamp: true,
		},
		DNS: &singbox.DNSConfig{
			Servers: []singbox.DNSServer{
				{Tag: "cloudflare", Address: "1.1.1.1"},
				{Tag: "local", Address: "223.5.5.5"},
			},
		},
		Inbounds: []singbox.Inbound{
			{
				Tag:        "socks",
				Type:       "socks",
				Listen:     "127.0.0.1",
				ListenPort: 1080,
			},
			{
				Tag:        "http",
				Type:       "http",
				Listen:     "127.0.0.1",
				ListenPort: 8888,
			},
		},
		Outbounds: []singbox.Outbound{
			{Tag: "direct", Type: "direct"},
			{Tag: "block", Type: "block"},
		},
		Route: &singbox.RouteConfig{
			Rules: []singbox.RouteRule{},
		},
	}, nil
}

// GetFilterVersion 获取当前过滤器配置版本
func (c *Client) GetFilterVersion() string {
	return c.filterMgr.GetCurrentVersion()
}

// UpdateMultiplexConfig 更新多路复用配置
func (c *Client) UpdateMultiplexConfig(protocol string, config map[string]interface{}) error {
	log.Printf("开始更新多路复用配置: protocol=%s", protocol)
	
	// 验证协议类型
	supportedProtocols := map[string]bool{
		"vmess": true, "vless": true, "trojan": true, "shadowsocks": true,
	}
	if !supportedProtocols[protocol] {
		return fmt.Errorf("不支持的协议类型: %s", protocol)
	}
	
	// 解析多路复用配置
	enabled, _ := config["enabled"].(bool)
	maxConnections := 4 // 默认值
	if mc, ok := config["max_connections"].(float64); ok {
		maxConnections = int(mc)
	} else if mc, ok := config["max_connections"].(int); ok {
		maxConnections = mc
	}
	
	minStreams := 4 // 默认值
	if ms, ok := config["min_streams"].(float64); ok {
		minStreams = int(ms)
	} else if ms, ok := config["min_streams"].(int); ok {
		minStreams = ms
	}
	
	padding, _ := config["padding"].(bool)
	brutalConfig, _ := config["brutal"].(map[string]interface{})
	
	log.Printf("多路复用配置参数: enabled=%t, maxConnections=%d, minStreams=%d, padding=%t", 
		enabled, maxConnections, minStreams, padding)
	
	// 更新sing-box配置中的多路复用设置
	if err := c.updateSingboxMultiplex(protocol, enabled, maxConnections, minStreams, padding, brutalConfig); err != nil {
		return fmt.Errorf("更新sing-box多路复用配置失败: %v", err)
	}
	
	log.Printf("多路复用配置更新成功: protocol=%s", protocol)
	return nil
}

// GetMultiplexConfig 获取多路复用配置
func (c *Client) GetMultiplexConfig(protocol string) (map[string]interface{}, error) {
	config := c.singboxMgr.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("未找到sing-box配置")
	}
	
	result := make(map[string]interface{})
	
	// 如果指定了协议，返回该协议的多路复用配置
	if protocol != "" {
		multiplexConfig := c.extractMultiplexConfigFromOutbound(config, protocol)
		if multiplexConfig != nil {
			result[protocol] = multiplexConfig
		}
		return result, nil
	}
	
	// 如果未指定协议，返回所有协议的多路复用配置
	supportedProtocols := []string{"vmess", "vless", "trojan", "shadowsocks"}
	for _, proto := range supportedProtocols {
		multiplexConfig := c.extractMultiplexConfigFromOutbound(config, proto)
		if multiplexConfig != nil {
			result[proto] = multiplexConfig
		}
	}
	
	return result, nil
}

// updateSingboxMultiplex 更新sing-box配置中的多路复用设置
func (c *Client) updateSingboxMultiplex(protocol string, enabled bool, maxConnections, minStreams int, padding bool, brutalConfig map[string]interface{}) error {
	config := c.singboxMgr.GetConfig()
	if config == nil {
		return fmt.Errorf("未找到当前配置")
	}
	
	log.Printf("正在更新sing-box多路复用配置...")
	log.Printf("  协议: %s", protocol)
	log.Printf("  启用状态: %t", enabled)
	if enabled {
		log.Printf("  最大连接数: %d", maxConnections)
		log.Printf("  最小流数: %d", minStreams)
		log.Printf("  填充: %t", padding)
		if brutalConfig != nil && len(brutalConfig) > 0 {
			log.Printf("  Brutal配置: %+v", brutalConfig)
		}
	}
	
	// 更新所有匹配协议的出站配置
	updated := false
	updatedOutbounds := []string{}
	
	for i := range config.Outbounds {
		outbound := &config.Outbounds[i]
		if outbound.Type == protocol {
			log.Printf("找到匹配的出站配置: Tag=%s, Type=%s", outbound.Tag, outbound.Type)
			
			// 记录原有配置
			var oldConfig string
			if outbound.Multiplex != nil {
				oldConfig = fmt.Sprintf("enabled=%t, max_conn=%d, min_streams=%d", 
					outbound.Multiplex.Enabled, outbound.Multiplex.MaxConnections, outbound.Multiplex.MinStreams)
			} else {
				oldConfig = "未配置"
			}
			log.Printf("  原配置: %s", oldConfig)
			
			// 构建多路复用配置
			if enabled {
				multiplex := &singbox.MultiplexConfig{
					Enabled:        true,
					Protocol:       "smux",
					MaxConnections: maxConnections,
					MinStreams:     minStreams,
					Padding:        padding,
				}
				
				// 添加brutal配置（如果存在）
				if brutalConfig != nil && len(brutalConfig) > 0 {
					brutal := &singbox.Brutal{
						Enabled: true,
					}
					if up, ok := brutalConfig["up"].(string); ok {
						brutal.Up = up
						log.Printf("  设置Brutal上传带宽: %s", up)
					}
					if down, ok := brutalConfig["down"].(string); ok {
						brutal.Down = down
						log.Printf("  设置Brutal下载带宽: %s", down)
					}
					multiplex.Brutal = brutal
				}
				
				// 设置多路复用配置到出站
				outbound.Multiplex = multiplex
				log.Printf("  新配置: enabled=true, max_conn=%d, min_streams=%d, padding=%t", 
					maxConnections, minStreams, padding)
			} else {
				// 禁用多路复用
				outbound.Multiplex = nil
				log.Printf("  新配置: disabled")
			}
			
			updatedOutbounds = append(updatedOutbounds, outbound.Tag)
			updated = true
		}
	}
	
	if !updated {
		log.Printf("未找到协议为 %s 的出站配置", protocol)
		return fmt.Errorf("未找到协议为 %s 的出站配置", protocol)
	}
	
	log.Printf("多路复用配置更新完成，影响的出站: %v", updatedOutbounds)
	
	// 应用更新后的配置
	log.Printf("正在应用多路复用配置到sing-box...")
	if err := c.singboxMgr.UpdateConfig(config); err != nil {
		log.Printf("应用sing-box配置失败: %v", err)
		return fmt.Errorf("应用sing-box配置失败: %v", err)
	}
	
	log.Printf("sing-box多路复用配置应用成功")
	return nil
}

// extractMultiplexConfigFromOutbound 从出站配置中提取多路复用配置
func (c *Client) extractMultiplexConfigFromOutbound(config *singbox.Config, protocol string) map[string]interface{} {
	for _, outbound := range config.Outbounds {
		if outbound.Type == protocol && outbound.Multiplex != nil {
			result := make(map[string]interface{})
			result["enabled"] = outbound.Multiplex.Enabled
			result["protocol"] = outbound.Multiplex.Protocol
			result["max_connections"] = outbound.Multiplex.MaxConnections
			result["min_streams"] = outbound.Multiplex.MinStreams
			result["padding"] = outbound.Multiplex.Padding
			
			if outbound.Multiplex.Brutal != nil {
				brutal := make(map[string]interface{})
				brutal["enabled"] = outbound.Multiplex.Brutal.Enabled
				brutal["up"] = outbound.Multiplex.Brutal.Up
				brutal["down"] = outbound.Multiplex.Brutal.Down
				result["brutal"] = brutal
			}
			
			result["outbound_protocol"] = protocol
			result["last_updated"] = time.Now().Format("2006-01-02 15:04:05")
			return result
		}
	}
	return nil
}

// UninstallAgent 处理Agent卸载请求
func (c *Client) UninstallAgent(agentID string, forceUninstall bool, reason string, timeoutSeconds int32) error {
	log.Printf("收到Agent卸载请求:")
	log.Printf("  Agent ID: %s", agentID)
	log.Printf("  强制卸载: %t", forceUninstall)
	log.Printf("  卸载原因: %s", reason)
	log.Printf("  超时时间: %d秒", timeoutSeconds)
	
	// 验证Agent ID
	if agentID != c.agentID {
		return fmt.Errorf("Agent ID不匹配: 期望 %s, 收到 %s", c.agentID, agentID)
	}
	
	// 执行sing-box卸载
	result := c.uninstallManager.UninstallSingbox(forceUninstall, timeoutSeconds)
	
	if result.Error != nil {
		log.Printf("sing-box卸载失败: %v", result.Error)
		// 即使卸载失败，也要上报给Controller
	} else {
		log.Printf("sing-box卸载成功，耗时: %v", result.CleanupTime)
	}
	
	// 准备卸载响应
	response := &pb.UninstallResponse{
		Success:         result.Success,
		Message:         result.Message,
		UninstallStatus: result.Status,
		CleanedFiles:    result.CleanedFiles,
		CleanupTime:     result.CleanupTime.Nanoseconds() / 1000000, // 转换为毫秒
	}
	
	// 上报卸载结果给Controller
	if err := c.reportUninstallResult(response); err != nil {
		log.Printf("上报卸载结果失败: %v", err)
		// 不返回错误，继续卸载流程
	}
	
	log.Printf("Agent卸载流程完成")
	
	// 延迟退出，确保响应能够发送
	go func() {
		time.Sleep(2 * time.Second)
		log.Printf("Agent即将退出...")
		os.Exit(0)
	}()
	
	return result.Error
}

// reportUninstallResult 上报卸载结果
func (c *Client) reportUninstallResult(result *pb.UninstallResponse) error {
	if c.client == nil {
		return fmt.Errorf("gRPC客户端未初始化")
	}
	
	log.Printf("上报卸载结果到Controller...")
	
	// 创建特殊的心跳请求来上报卸载状态
	req := &pb.HeartbeatRequest{
		AgentId: c.agentID,
		Status:  "uninstalling",
		Metrics: map[string]string{
			"uninstall_status":    result.UninstallStatus,
			"uninstall_success":   fmt.Sprintf("%t", result.Success),
			"uninstall_message":   result.Message,
			"cleanup_time_ms":     fmt.Sprintf("%d", result.CleanupTime),
			"cleaned_files_count": fmt.Sprintf("%d", len(result.CleanedFiles)),
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("发送卸载结果失败: %v", err)
	}
	
	if !resp.Success {
		return fmt.Errorf("Controller拒绝卸载结果: %s", resp.Message)
	}
	
	log.Printf("卸载结果已成功上报到Controller")
	return nil
}

