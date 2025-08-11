package service

import (
	"fmt"
	"time"

	"github.com/xbox/sing-box-manager/internal/controller/repository"
	"github.com/xbox/sing-box-manager/internal/models"
	pb "github.com/xbox/sing-box-manager/proto"
)

// AgentService Agent业务逻辑接口
type AgentService interface {
	// 注册Agent
	RegisterAgent(req *pb.RegisterRequest) (*pb.RegisterResponse, error)
	// 处理心跳
	ProcessHeartbeat(req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error)
	// 获取Agent状态
	GetAgentStatus(agentID string) (*pb.StatusResponse, error)
	// 获取Agent列表
	GetAgentList(limit, offset int) ([]*models.Agent, int64, error)
	// 获取在线Agent数量
	GetOnlineAgentCount() (int64, error)
	// 检查离线Agent
	CheckOfflineAgents() error
	// 删除Agent
	DeleteAgent(agentID string) error
	// 更新Agent信息
	UpdateAgent(agent *models.Agent) error
}

// agentService Agent业务逻辑实现
type agentService struct {
	agentRepo         repository.AgentRepository
	heartbeatInterval time.Duration
	maxOfflineTime    time.Duration
}

// NewAgentService 创建Agent业务逻辑实例
func NewAgentService(agentRepo repository.AgentRepository) AgentService {
	return &agentService{
		agentRepo:         agentRepo,
		heartbeatInterval: 30 * time.Second,  // 默认30秒心跳间隔
		maxOfflineTime:    5 * time.Minute,   // 默认5分钟超时
	}
}

// RegisterAgent 注册Agent
func (s *agentService) RegisterAgent(req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.AgentId == "" {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	// 检查Agent是否已存在
	existingAgent, err := s.agentRepo.GetByID(req.AgentId)
	if err == nil {
		// Agent已存在，更新信息
		existingAgent.Hostname = req.Hostname
		existingAgent.IPAddress = req.IpAddress
		existingAgent.Version = req.Version
		existingAgent.Status = "online"
		
		// 更新IP段信息
		if req.IpRangeInfo != nil {
			existingAgent.IPRange = req.IpRangeInfo.IpRange
			existingAgent.Country = req.IpRangeInfo.Country
			existingAgent.Region = req.IpRangeInfo.Region
			existingAgent.City = req.IpRangeInfo.City
			existingAgent.ISP = req.IpRangeInfo.Isp
		}
		
		// 更新元数据
		if req.Metadata != nil {
			metadata := make(models.JSON)
			for k, v := range req.Metadata {
				metadata[k] = v
			}
			existingAgent.Metadata = metadata
		}
		
		now := time.Now()
		existingAgent.LastHeartbeat = &now
		
		if err := s.agentRepo.Update(existingAgent); err != nil {
			return &pb.RegisterResponse{
				Success: false,
				Message: fmt.Sprintf("更新Agent失败: %v", err),
			}, nil
		}
		
		return &pb.RegisterResponse{
			Success: true,
			Message: "Agent重新注册成功",
			Token:   s.generateToken(req.AgentId),
		}, nil
	}

	// 创建新Agent
	agent := &models.Agent{
		ID:        req.AgentId,
		Hostname:  req.Hostname,
		IPAddress: req.IpAddress,
		Version:   req.Version,
		Status:    "online",
	}
	
	// 设置IP段信息
	if req.IpRangeInfo != nil {
		agent.IPRange = req.IpRangeInfo.IpRange
		agent.Country = req.IpRangeInfo.Country
		agent.Region = req.IpRangeInfo.Region
		agent.City = req.IpRangeInfo.City
		agent.ISP = req.IpRangeInfo.Isp
	}
	
	// 设置元数据
	if req.Metadata != nil {
		metadata := make(models.JSON)
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		agent.Metadata = metadata
	}
	
	now := time.Now()
	agent.LastHeartbeat = &now

	if err := s.agentRepo.Create(agent); err != nil {
		return &pb.RegisterResponse{
			Success: false,
			Message: fmt.Sprintf("创建Agent失败: %v", err),
		}, nil
	}

	return &pb.RegisterResponse{
		Success: true,
		Message: "Agent注册成功",
		Token:   s.generateToken(req.AgentId),
	}, nil
}

// ProcessHeartbeat 处理心跳
func (s *agentService) ProcessHeartbeat(req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if req.AgentId == "" {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: "Agent ID不能为空",
		}, nil
	}

	// 检查Agent是否存在
	agent, err := s.agentRepo.GetByID(req.AgentId)
	if err != nil {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: "Agent不存在，请重新注册",
		}, nil
	}

	// 如果心跳包含IP段信息更新，先更新Agent信息
	if req.IpRangeInfo != nil {
		agent.IPRange = req.IpRangeInfo.IpRange
		agent.Country = req.IpRangeInfo.Country
		agent.Region = req.IpRangeInfo.Region
		agent.City = req.IpRangeInfo.City
		agent.ISP = req.IpRangeInfo.Isp
		
		if err := s.agentRepo.Update(agent); err != nil {
			return &pb.HeartbeatResponse{
				Success: false,
				Message: fmt.Sprintf("更新IP段信息失败: %v", err),
			}, nil
		}
	}

	// 更新心跳时间和状态
	if err := s.agentRepo.UpdateHeartbeat(req.AgentId); err != nil {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: fmt.Sprintf("更新心跳失败: %v", err),
		}, nil
	}

	// 如果有指标数据，可以在这里处理
	if req.Metrics != nil {
		// 检查是否为卸载状态上报
		if req.Status == "uninstalling" {
			// 处理卸载状态上报
			log.Printf("收到Agent卸载状态上报: AgentID=%s", req.AgentId)
			for key, value := range req.Metrics {
				log.Printf("  %s: %s", key, value)
			}
			
			// TODO: 这里需要调用UninstallService.ProcessUninstallReport()
			// 暂时先记录日志，完整实现需要在server层注入UninstallService
		}
		
		// TODO: 处理其他监控指标数据
		// s.processMetrics(req.AgentId, req.Metrics)
	}

	return &pb.HeartbeatResponse{
		Success:               true,
		Message:               "心跳处理成功",
		NextHeartbeatInterval: int64(s.heartbeatInterval.Seconds()),
	}, nil
}

// GetAgentStatus 获取Agent状态
func (s *agentService) GetAgentStatus(agentID string) (*pb.StatusResponse, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return &pb.StatusResponse{
			Success: false,
		}, fmt.Errorf("获取Agent失败: %v", err)
	}

	// 构建系统信息
	systemInfo := make(map[string]string)
	systemInfo["hostname"] = agent.Hostname
	systemInfo["ip_address"] = agent.IPAddress
	systemInfo["version"] = agent.Version
	systemInfo["created_at"] = agent.CreatedAt.Format(time.RFC3339)
	systemInfo["updated_at"] = agent.UpdatedAt.Format(time.RFC3339)
	
	if agent.LastHeartbeat != nil {
		systemInfo["last_heartbeat"] = agent.LastHeartbeat.Format(time.RFC3339)
	}

	return &pb.StatusResponse{
		Success:      true,
		AgentId:      agent.ID,
		Status:       agent.Status,
		ConfigVersion: "", // TODO: 从配置表获取
		RulesCount:   0,   // TODO: 从规则表获取
		SystemInfo:   systemInfo,
	}, nil
}

// GetAgentList 获取Agent列表
func (s *agentService) GetAgentList(limit, offset int) ([]*models.Agent, int64, error) {
	return s.agentRepo.GetAll(limit, offset)
}

// GetOnlineAgentCount 获取在线Agent数量
func (s *agentService) GetOnlineAgentCount() (int64, error) {
	return s.agentRepo.GetOnlineCount()
}

// CheckOfflineAgents 检查离线Agent
func (s *agentService) CheckOfflineAgents() error {
	offlineAgents, err := s.agentRepo.GetOfflineAgents(s.maxOfflineTime)
	if err != nil {
		return fmt.Errorf("获取离线Agent失败: %v", err)
	}

	// 更新离线Agent状态
	for _, agent := range offlineAgents {
		if agent.Status != "offline" {
			if err := s.agentRepo.UpdateStatus(agent.ID, "offline"); err != nil {
				// 记录错误但继续处理其他Agent
				fmt.Printf("更新Agent %s 状态失败: %v\n", agent.ID, err)
			}
		}
	}

	return nil
}

// DeleteAgent 删除Agent
func (s *agentService) DeleteAgent(agentID string) error {
	// 检查Agent是否存在
	_, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return fmt.Errorf("Agent不存在: %v", err)
	}

	return s.agentRepo.Delete(agentID)
}

// UpdateAgent 更新Agent信息
func (s *agentService) UpdateAgent(agent *models.Agent) error {
	// 检查Agent是否存在
	_, err := s.agentRepo.GetByID(agent.ID)
	if err != nil {
		return fmt.Errorf("Agent不存在: %v", err)
	}

	return s.agentRepo.Update(agent)
}

// generateToken 生成访问令牌
func (s *agentService) generateToken(agentID string) string {
	// TODO: 实现JWT令牌生成
	// 临时返回简单的token
	return fmt.Sprintf("token_%s_%d", agentID, time.Now().Unix())
}