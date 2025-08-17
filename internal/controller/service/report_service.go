package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xbox/sing-box-manager/internal/controller/repository"
	"github.com/xbox/sing-box-manager/internal/models"
	"github.com/xbox/sing-box-manager/pkg/logger"
	"gorm.io/gorm"
)

// NodeReportService 节点上报服务
type NodeReportService struct {
	db           *gorm.DB
	agentRepo    repository.AgentRepository
	httpClient   *http.Client
	backendURL   string
	logger       logger.Logger
	reportTicker *time.Ticker
	stopChan     chan struct{}
}

// ReportRequest 上报请求结构
type ReportRequest struct {
	ReportTime time.Time       `json:"report_time"`
	Controller ControllerInfo  `json:"controller"`
	Nodes      []NodeInfo      `json:"nodes"`
	Stats      ReportStats     `json:"stats"`
}

// ControllerInfo Controller信息
type ControllerInfo struct {
	ControllerID string `json:"controllerId"`
	Version      string `json:"version"`
	Address      string `json:"address"`
	Status       string `json:"status"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	AgentID            string     `json:"agentId"`
	Hostname           string     `json:"hostname"`
	IPAddress          string     `json:"ipAddress"`
	IPRange            string     `json:"ipRange"`
	Country            string     `json:"country"`
	Region             string     `json:"region"`
	City               string     `json:"city"`
	ISP                string     `json:"isp"`
	Status             string     `json:"status"`
	LastHeartbeat      *time.Time `json:"lastHeartbeat"`
	Version            string     `json:"version"`
	PortRange          string     `json:"portRange"`
	BandwidthMbps      int        `json:"bandwidthMbps"`
	IPQuality          string     `json:"ipQuality"`
	Provider           string     `json:"provider"`
	SupportedProtocols string     `json:"supportedProtocols"`
	Metadata           string     `json:"metadata"`
}

// ReportStats 统计信息
type ReportStats struct {
	TotalNodes    int `json:"totalNodes"`
	OnlineNodes   int `json:"onlineNodes"`
	OfflineNodes  int `json:"offlineNodes"`
	ErrorNodes    int `json:"errorNodes"`
	TotalIPRanges int `json:"totalIpRanges"`
}

// NewNodeReportService 创建节点上报服务
func NewNodeReportService(db *gorm.DB, agentRepo repository.AgentRepository, backendURL string, logger logger.Logger) *NodeReportService {
	return &NodeReportService{
		db:         db,
		agentRepo:  agentRepo,
		backendURL: backendURL,
		logger:     logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// StartReporting 开始定时上报
func (s *NodeReportService) StartReporting(ctx context.Context, interval time.Duration) {
	s.logger.Info("开始启动节点信息定时上报服务，上报间隔: %s", interval)
	
	// 立即执行一次上报
	s.reportNodeInfo(ctx)
	
	// 创建定时器
	s.reportTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-s.reportTicker.C:
				s.reportNodeInfo(ctx)
			case <-s.stopChan:
				s.logger.Info("节点信息上报服务已停止")
				return
			case <-ctx.Done():
				s.logger.Info("节点信息上报服务因上下文取消而停止")
				return
			}
		}
	}()
	
	s.logger.Info("节点信息定时上报服务启动成功")
}

// StopReporting 停止定时上报
func (s *NodeReportService) StopReporting() {
	s.logger.Info("正在停止节点信息上报服务")
	
	if s.reportTicker != nil {
		s.reportTicker.Stop()
	}
	
	close(s.stopChan)
	s.logger.Info("节点信息上报服务已停止")
}

// reportNodeInfo 执行节点信息上报
func (s *NodeReportService) reportNodeInfo(ctx context.Context) {
	startTime := time.Now()
	s.logger.Info("开始执行节点信息上报")
	
	// 收集节点信息
	nodes, stats, err := s.collectNodeInfo(ctx)
	if err != nil {
		s.logger.Error("收集节点信息失败: %v", err)
		return
	}
	
	// 构建上报请求
	reportRequest := ReportRequest{
		ReportTime: time.Now(),
		Controller: ControllerInfo{
			ControllerID: "xbox-controller-001",
			Version:      "v1.2.3",
			Address:      "localhost:9000",
			Status:       "running",
		},
		Nodes: nodes,
		Stats: stats,
	}
	
	// 发送上报请求
	if err := s.sendReportRequest(ctx, reportRequest); err != nil {
		s.logger.Error("发送节点信息上报失败: %v", err)
		return
	}
	
	duration := time.Since(startTime)
	s.logger.Info("节点信息上报完成，耗时: %v，上报节点数: %d", duration, len(nodes))
}

// collectNodeInfo 收集节点信息
func (s *NodeReportService) collectNodeInfo(ctx context.Context) ([]NodeInfo, ReportStats, error) {
	s.logger.Debug("开始收集节点信息")
	
	// 获取所有Agent
	agents, err := s.agentRepo.GetAllAgents(ctx)
	if err != nil {
		return nil, ReportStats{}, fmt.Errorf("获取Agent列表失败: %v", err)
	}
	
	nodes := make([]NodeInfo, 0, len(agents))
	stats := ReportStats{
		TotalNodes: len(agents),
	}
	
	ipRangeSet := make(map[string]bool) // 用于统计唯一IP段数量
	
	for _, agent := range agents {
		nodeInfo := s.convertAgentToNodeInfo(agent)
		nodes = append(nodes, nodeInfo)
		
		// 统计各状态节点数量
		switch agent.Status {
		case "online":
			stats.OnlineNodes++
		case "offline":
			stats.OfflineNodes++
		case "error":
			stats.ErrorNodes++
		}
		
		// 统计IP段数量
		if agent.IPRange != "" {
			ipRangeSet[agent.IPRange] = true
		}
	}
	
	stats.TotalIPRanges = len(ipRangeSet)
	
	s.logger.Debug("节点信息收集完成，总节点: %d，在线: %d，离线: %d，错误: %d，IP段: %d",
		stats.TotalNodes, stats.OnlineNodes, stats.OfflineNodes, stats.ErrorNodes, stats.TotalIPRanges)
	
	return nodes, stats, nil
}

// convertAgentToNodeInfo 将Agent转换为NodeInfo
func (s *NodeReportService) convertAgentToNodeInfo(agent *models.Agent) NodeInfo {
	nodeInfo := NodeInfo{
		AgentID:       agent.ID,
		Hostname:      agent.Hostname,
		IPAddress:     agent.IPAddress,
		IPRange:       agent.IPRange,
		Country:       agent.Country,
		Region:        agent.Region,
		City:          agent.City,
		ISP:           agent.ISP,
		Status:        agent.Status,
		LastHeartbeat: agent.LastHeartbeat,
		Version:       agent.Version,
	}
	
	// 设置默认端口范围
	if nodeInfo.PortRange == "" {
		nodeInfo.PortRange = "8000-8999"
	}
	
	// 设置默认带宽
	if nodeInfo.BandwidthMbps == 0 {
		nodeInfo.BandwidthMbps = 1000 // 默认1000Mbps
	}
	
	// 设置IP质量
	if nodeInfo.IPQuality == "" {
		nodeInfo.IPQuality = "标准"
	}
	
	// 设置供应商
	if nodeInfo.Provider == "" {
		if nodeInfo.ISP != "" {
			nodeInfo.Provider = nodeInfo.ISP
		} else {
			nodeInfo.Provider = "Xbox-Provider"
		}
	}
	
	// 设置支持的协议
	nodeInfo.SupportedProtocols = "HTTP,SOCKS5,Shadowsocks,VMess,Trojan,VLESS"
	
	// 设置元数据
	if agent.Metadata != nil {
		if metadataBytes, err := json.Marshal(agent.Metadata); err == nil {
			nodeInfo.Metadata = string(metadataBytes)
		}
	}
	
	return nodeInfo
}

// sendReportRequest 发送上报请求
func (s *NodeReportService) sendReportRequest(ctx context.Context, request ReportRequest) error {
	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}
	
	// 构建HTTP请求
	url := fmt.Sprintf("%s/api/ip-pool/report", s.backendURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Xbox-Controller/1.0")
	
	s.logger.Debug("发送节点信息上报请求到: %s，节点数量: %d", url, len(request.Nodes))
	
	// 发送请求
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态: %d", resp.StatusCode)
	}
	
	// 解析响应
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 检查业务状态码
	if response.Code != 200 {
		return fmt.Errorf("业务处理失败: %s", response.Message)
	}
	
	s.logger.Info("节点信息上报成功: %s", response.Data)
	return nil
}

// ReportOnce 立即执行一次上报
func (s *NodeReportService) ReportOnce(ctx context.Context) error {
	s.logger.Info("执行手动节点信息上报")
	s.reportNodeInfo(ctx)
	return nil
}

// GetReportStats 获取上报统计信息
func (s *NodeReportService) GetReportStats(ctx context.Context) (*ReportStats, error) {
	_, stats, err := s.collectNodeInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}