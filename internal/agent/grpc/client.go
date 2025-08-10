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
	"github.com/xbox/sing-box-manager/internal/agent/singbox"
	"github.com/xbox/sing-box-manager/internal/config"
	pb "github.com/xbox/sing-box-manager/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC客户端
type Client struct {
	config       *config.Config
	conn         *grpc.ClientConn
	client       pb.AgentServiceClient
	agentID      string
	token        string
	registered   bool
	monitor      *monitor.SystemMonitor
	singboxMgr   *singbox.Manager
	filterMgr    *filter.FilterManager
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
	
	return &Client{
		config:     cfg,
		agentID:    agentID,
		monitor:    monitor.NewSystemMonitor(),
		singboxMgr: singboxMgr,
		filterMgr:  filterMgr,
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
	req := &pb.RegisterRequest{
		AgentId:   c.agentID,
		Hostname:  hostname,
		IpAddress: c.monitor.GetLocalIP(),
		Version:   "1.0.0",
		Metadata:  systemInfo,
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
			routeRule.Port = port
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
				Tag:    "socks",
				Type:   "socks",
				Listen: "127.0.0.1",
				Port:   1080,
			},
			{
				Tag:    "http",
				Type:   "http",
				Listen: "127.0.0.1",
				Port:   8888,
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

