package grpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	backendpb "github.com/xbox/sing-box-manager/proto/backend"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	"github.com/xbox/sing-box-manager/internal/models"
)

// BackendServiceServer 实现 BackendService gRPC 服务
type BackendServiceServer struct {
	backendpb.UnimplementedBackendServiceServer
	agentService      service.AgentService
	multiplexService  service.MultiplexService
	reportService     *service.NodeReportService
}

// NewBackendServiceServer 创建新的 BackendService 服务器
func NewBackendServiceServer(
	agentService service.AgentService,
	multiplexService service.MultiplexService,
	reportService *service.NodeReportService,
) *BackendServiceServer {
	return &BackendServiceServer{
		agentService:     agentService,
		multiplexService: multiplexService,
		reportService:    reportService,
	}
}

// HealthCheck 健康检查
func (s *BackendServiceServer) HealthCheck(ctx context.Context, req *backendpb.HealthCheckRequest) (*backendpb.HealthCheckResponse, error) {
	return &backendpb.HealthCheckResponse{
		Success:   true,
		Status:    "ok",
		Service:   "xbox-controller",
		Timestamp: timestamppb.Now(),
	}, nil
}

// GetAgents 获取Agent列表
func (s *BackendServiceServer) GetAgents(ctx context.Context, req *backendpb.GetAgentsRequest) (*backendpb.GetAgentsResponse, error) {
	// 获取所有agents
	agents, err := s.agentService.GetAllAgents()
	if err != nil {
		return &backendpb.GetAgentsResponse{
			Success: false,
			Message: fmt.Sprintf("获取Agent列表失败: %v", err),
		}, nil
	}

	// 应用过滤器
	filteredAgents := s.filterAgents(agents, req.StatusFilter, req.CountryFilter, req.RegionFilter)

	// 分页处理
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	totalCount := int32(len(filteredAgents))

	if start >= int32(len(filteredAgents)) {
		filteredAgents = []*models.Agent{}
	} else if end > int32(len(filteredAgents)) {
		filteredAgents = filteredAgents[start:]
	} else {
		filteredAgents = filteredAgents[start:end]
	}

	// 转换为protobuf格式
	agentInfos := make([]*backendpb.AgentInfo, len(filteredAgents))
	for i, agent := range filteredAgents {
		agentInfos[i] = s.convertAgentToProto(agent)
	}

	return &backendpb.GetAgentsResponse{
		Success:     true,
		Message:     "获取Agent列表成功",
		Agents:      agentInfos,
		TotalCount:  totalCount,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// GetAgent 获取单个Agent
func (s *BackendServiceServer) GetAgent(ctx context.Context, req *backendpb.GetAgentRequest) (*backendpb.GetAgentResponse, error) {
	if req.AgentId == "" {
		return &backendpb.GetAgentResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	agent, err := s.agentService.GetAgent(req.AgentId)
	if err != nil {
		return &backendpb.GetAgentResponse{
			Success: false,
			Message: fmt.Sprintf("获取Agent失败: %v", err),
		}, nil
	}

	if agent == nil {
		return &backendpb.GetAgentResponse{
			Success: false,
			Message: "Agent不存在",
		}, nil
	}

	return &backendpb.GetAgentResponse{
		Success: true,
		Message: "获取Agent成功",
		Agent:   s.convertAgentToProto(agent),
	}, nil
}

// GetAgentStats 获取Agent统计信息
func (s *BackendServiceServer) GetAgentStats(ctx context.Context, req *backendpb.GetAgentStatsRequest) (*backendpb.GetAgentStatsResponse, error) {
	agents, err := s.agentService.GetAllAgents()
	if err != nil {
		return &backendpb.GetAgentStatsResponse{
			Success: false,
			Message: fmt.Sprintf("获取Agent统计失败: %v", err),
		}, nil
	}

	stats := s.calculateAgentStats(agents)

	return &backendpb.GetAgentStatsResponse{
		Success: true,
		Message: "获取Agent统计成功",
		Stats:   stats,
	}, nil
}

// DeployAgent 部署Agent
func (s *BackendServiceServer) DeployAgent(ctx context.Context, req *backendpb.DeployAgentRequest) (*backendpb.DeployAgentResponse, error) {
	if req.NodeIp == "" || req.SshUser == "" || req.SshPassword == "" {
		return &backendpb.DeployAgentResponse{
			Success: false,
			Message: "部署参数不完整",
		}, nil
	}

	startTime := time.Now()

	// 调用agent服务的部署方法
	agentID, deployLog, err := s.agentService.DeployAgentToNode(
		req.NodeIp,
		int(req.SshPort),
		req.SshUser,
		req.SshPassword,
		req.ControllerUrl,
		req.DeploymentOptions,
	)

	deployTime := int32(time.Since(startTime).Seconds())

	if err != nil {
		return &backendpb.DeployAgentResponse{
			Success:                false,
			Message:                fmt.Sprintf("部署Agent失败: %v", err),
			DeploymentLog:          deployLog,
			DeploymentTimeSeconds:  deployTime,
		}, nil
	}

	return &backendpb.DeployAgentResponse{
		Success:                true,
		Message:                "Agent部署成功",
		AgentId:                agentID,
		DeploymentLog:          deployLog,
		DeploymentTimeSeconds:  deployTime,
	}, nil
}

// UninstallAgent 卸载Agent
func (s *BackendServiceServer) UninstallAgent(ctx context.Context, req *backendpb.UninstallAgentRequest) (*backendpb.UninstallAgentResponse, error) {
	if req.AgentId == "" {
		return &backendpb.UninstallAgentResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	// 调用agent服务的卸载方法
	result, err := s.agentService.UninstallAgent(
		req.AgentId,
		req.ForceUninstall,
		req.Reason,
		int(req.TimeoutSeconds),
	)

	if err != nil {
		return &backendpb.UninstallAgentResponse{
			Success: false,
			Message: fmt.Sprintf("卸载Agent失败: %v", err),
		}, nil
	}

	// Type assertion for the result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return &backendpb.UninstallAgentResponse{
			Success: false,
			Message: "卸载结果格式错误",
		}, nil
	}

	success, _ := resultMap["Success"].(bool)
	message, _ := resultMap["Message"].(string)
	status, _ := resultMap["Status"].(string)
	cleanedFiles, _ := resultMap["CleanedFiles"].([]string)
	cleanupTimeMs, _ := resultMap["CleanupTimeMs"].(int)

	return &backendpb.UninstallAgentResponse{
		Success:         success,
		Message:         message,
		UninstallStatus: status,
		CleanedFiles:    cleanedFiles,
		CleanupTimeMs:   int64(cleanupTimeMs),
	}, nil
}

// UpdateAgentConfig 更新Agent配置
func (s *BackendServiceServer) UpdateAgentConfig(ctx context.Context, req *backendpb.UpdateAgentConfigRequest) (*backendpb.UpdateAgentConfigResponse, error) {
	if req.AgentId == "" || req.ConfigType == "" {
		return &backendpb.UpdateAgentConfigResponse{
			Success: false,
			Message: "Agent ID和配置类型不能为空",
		}, nil
	}

	// 调用agent服务的配置更新方法
	version, err := s.agentService.UpdateAgentConfig(
		req.AgentId,
		req.ConfigType,
		req.ConfigContent,
		req.ForceUpdate,
		req.ConfigVersion,
	)

	if err != nil {
		return &backendpb.UpdateAgentConfigResponse{
			Success: false,
			Message: fmt.Sprintf("更新Agent配置失败: %v", err),
		}, nil
	}

	return &backendpb.UpdateAgentConfigResponse{
		Success:        true,
		Message:        "配置更新成功",
		AppliedVersion: version,
		UpdatedAt:      timestamppb.Now(),
	}, nil
}

// GetAgentMonitoring 获取Agent监控数据
func (s *BackendServiceServer) GetAgentMonitoring(ctx context.Context, req *backendpb.GetAgentMonitoringRequest) (*backendpb.GetAgentMonitoringResponse, error) {
	if req.AgentId == "" {
		return &backendpb.GetAgentMonitoringResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	// 获取监控数据
	monitoringData, err := s.agentService.GetAgentMonitoring(req.AgentId, int(req.DurationMinutes))
	if err != nil {
		return &backendpb.GetAgentMonitoringResponse{
			Success: false,
			Message: fmt.Sprintf("获取监控数据失败: %v", err),
		}, nil
	}

	return &backendpb.GetAgentMonitoringResponse{
		Success:        true,
		Message:        "获取监控数据成功",
		MonitoringData: s.convertMonitoringDataToProto(monitoringData),
	}, nil
}

// GetIPRanges 获取IP段信息
func (s *BackendServiceServer) GetIPRanges(ctx context.Context, req *backendpb.GetIPRangesRequest) (*backendpb.GetIPRangesResponse, error) {
	// 获取IP段信息
	ipRanges, err := s.agentService.GetIPRanges(
		req.CountryFilter,
		req.RegionFilter,
		req.IspFilter,
		req.IncludeAgents,
	)

	if err != nil {
		return &backendpb.GetIPRangesResponse{
			Success: false,
			Message: fmt.Sprintf("获取IP段信息失败: %v", err),
		}, nil
	}

	// 转换为protobuf格式
	protoRanges := make([]*backendpb.IPRangeInfo, len(ipRanges))
	for i, ipRange := range ipRanges {
		protoRanges[i] = s.convertIPRangeToProto(ipRange)
	}

	return &backendpb.GetIPRangesResponse{
		Success:     true,
		Message:     "获取IP段信息成功",
		IpRanges:    protoRanges,
		TotalRanges: int32(len(protoRanges)),
	}, nil
}

// TestAgent 测试Agent
func (s *BackendServiceServer) TestAgent(ctx context.Context, req *backendpb.TestAgentRequest) (*backendpb.TestAgentResponse, error) {
	if req.AgentId == "" {
		return &backendpb.TestAgentResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	// 执行测试
	_, overallStatus, err := s.agentService.TestAgent(
		req.AgentId,
		req.TestType,
		int(req.TimeoutSeconds),
	)

	if err != nil {
		return &backendpb.TestAgentResponse{
			Success: false,
			Message: fmt.Sprintf("测试Agent失败: %v", err),
		}, nil
	}

	// 转换测试结果 - 当前返回简单格式
	protoResults := []*backendpb.TestResult{}
	// TODO: 实现完整的测试结果转换逻辑

	return &backendpb.TestAgentResponse{
		Success:       true,
		Message:       "测试完成",
		TestResults:   protoResults,
		OverallStatus: overallStatus,
	}, nil
}

// GetProtocolInfo 获取协议支持信息
func (s *BackendServiceServer) GetProtocolInfo(ctx context.Context, req *backendpb.GetProtocolInfoRequest) (*backendpb.GetProtocolInfoResponse, error) {
	// 获取协议支持信息
	protocols, systemVersion, err := s.agentService.GetProtocolInfo(req.AgentId)
	if err != nil {
		return &backendpb.GetProtocolInfoResponse{
			Success: false,
			Message: fmt.Sprintf("获取协议支持信息失败: %v", err),
		}, nil
	}

	// 转换为protobuf格式 - 当前协议信息返回简单格式
	protoProtocols := make([]*backendpb.ProtocolSupportInfo, len(protocols))
	for i, protocol := range protocols {
		protoProtocols[i] = &backendpb.ProtocolSupportInfo{
			ProtocolName: protocol,
			Supported:    true,
			Version:      "1.0",
			Capabilities: make(map[string]string),
			Enabled:      true,
		}
	}

	return &backendpb.GetProtocolInfoResponse{
		Success:       true,
		Message:       "获取协议支持信息成功",
		Protocols:     protoProtocols,
		SystemVersion: systemVersion,
	}, nil
}

// 辅助方法

// filterAgents 过滤Agent列表
func (s *BackendServiceServer) filterAgents(agents []*models.Agent, statusFilter, countryFilter, regionFilter string) []*models.Agent {
	var filtered []*models.Agent

	for _, agent := range agents {
		// 状态过滤
		if statusFilter != "" && !strings.EqualFold(agent.Status, statusFilter) {
			continue
		}

		// 国家过滤
		if countryFilter != "" && !strings.Contains(strings.ToLower(agent.Country), strings.ToLower(countryFilter)) {
			continue
		}

		// 地区过滤
		if regionFilter != "" && !strings.Contains(strings.ToLower(agent.Region), strings.ToLower(regionFilter)) {
			continue
		}

		filtered = append(filtered, agent)
	}

	return filtered
}

// convertAgentToProto 转换Agent为protobuf格式
func (s *BackendServiceServer) convertAgentToProto(agent *models.Agent) *backendpb.AgentInfo {
	agentInfo := &backendpb.AgentInfo{
		AgentId:            agent.ID,
		Hostname:           agent.Hostname,
		IpAddress:          agent.IPAddress,
		Status:             agent.Status,
		Country:            agent.Country,
		Region:             agent.Region,
		City:               agent.City,
		CurrentConnections: int32(agent.CurrentConnections),
		CpuUsage:           agent.CPUUsage,
		MemoryUsage:        agent.MemoryUsage,
		DiskUsage:          agent.DiskUsage,
		NetworkLatency:     int32(agent.NetworkLatency),
		Version:            agent.Version,
		Metadata:           make(map[string]string), // TODO: 实现元数据转换
	}

	// 设置时间戳
	if agent.LastHeartbeat != nil && !agent.LastHeartbeat.IsZero() {
		agentInfo.LastHeartbeat = timestamppb.New(*agent.LastHeartbeat)
	}
	if !agent.CreatedAt.IsZero() {
		agentInfo.RegisteredAt = timestamppb.New(agent.CreatedAt)
	}

	// 设置IP段信息
	if agent.IPRange != "" || agent.Country != "" {
		agentInfo.IpRangeInfo = &backendpb.IPRangeInfo{
			IpRange:         agent.IPRange,
			Country:         agent.Country,
			Region:          agent.Region,
			City:            agent.City,
			Isp:             agent.ISP,
			DetectionMethod: "auto",
		}
		// 使用创建时间作为检测时间
		agentInfo.IpRangeInfo.DetectedAt = timestamppb.New(agent.CreatedAt)
	}

	return agentInfo
}

// calculateAgentStats 计算Agent统计信息
func (s *BackendServiceServer) calculateAgentStats(agents []*models.Agent) *backendpb.AgentStats {
	stats := &backendpb.AgentStats{
		AgentsByCountry: make(map[string]int32),
		AgentsByRegion:  make(map[string]int32),
	}

	var totalCPU, totalMemory float64
	var totalConnections int32

	for _, agent := range agents {
		stats.TotalAgents++

		// 状态统计
		switch strings.ToLower(agent.Status) {
		case "online":
			stats.OnlineAgents++
		case "offline":
			stats.OfflineAgents++
		default:
			stats.UnknownAgents++
		}

		// 性能统计
		totalCPU += agent.CPUUsage
		totalMemory += agent.MemoryUsage
		totalConnections += int32(agent.CurrentConnections)

		// 地理位置统计
		if agent.Country != "" {
			stats.AgentsByCountry[agent.Country]++
		}
		if agent.Region != "" {
			stats.AgentsByRegion[agent.Region]++
		}
	}

	// 计算平均值
	if stats.TotalAgents > 0 {
		stats.AvgCpuUsage = totalCPU / float64(stats.TotalAgents)
		stats.AvgMemoryUsage = totalMemory / float64(stats.TotalAgents)
	}
	stats.TotalConnections = totalConnections

	return stats
}

// convertMonitoringDataToProto 转换监控数据为protobuf格式
func (s *BackendServiceServer) convertMonitoringDataToProto(data interface{}) *backendpb.AgentMonitoringData {
	// 这里需要根据实际的监控数据结构进行转换
	// 假设data是一个包含监控信息的结构体
	return &backendpb.AgentMonitoringData{
		AgentId:             "placeholder",
		SingboxStatus:       "running",
		LastUpdate:          timestamppb.Now(),
		ActiveConnections:   0,
		NetworkLatencyMs:    0.0,
	}
}

// convertIPRangeToProto 转换IP段信息为protobuf格式
func (s *BackendServiceServer) convertIPRangeToProto(ipRange interface{}) *backendpb.IPRangeInfo {
	// 这里需要根据实际的IP段数据结构进行转换
	return &backendpb.IPRangeInfo{
		IpRange:         "192.168.1.0/24",
		Country:         "Unknown",
		Region:          "Unknown",
		City:            "Unknown",
		Isp:             "Unknown",
		AgentCount:      0,
		DetectionMethod: "auto",
		DetectedAt:      timestamppb.Now(),
	}
}