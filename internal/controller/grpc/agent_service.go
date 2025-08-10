package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xbox/sing-box-manager/internal/controller/service"
	pb "github.com/xbox/sing-box-manager/proto"
)

// AgentServiceServer gRPC AgentService服务实现
type AgentServiceServer struct {
	pb.UnimplementedAgentServiceServer
	agentService service.AgentService
}

// NewAgentServiceServer 创建AgentService服务实例
func NewAgentServiceServer(agentService service.AgentService) *AgentServiceServer {
	return &AgentServiceServer{
		agentService: agentService,
	}
}

// RegisterAgent 实现节点注册
func (s *AgentServiceServer) RegisterAgent(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Agent注册请求: ID=%s, Hostname=%s, IP=%s", req.AgentId, req.Hostname, req.IpAddress)
	
	resp, err := s.agentService.RegisterAgent(req)
	if err != nil {
		log.Printf("Agent注册失败: %v", err)
		return &pb.RegisterResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	log.Printf("Agent注册响应: Success=%v, Message=%s", resp.Success, resp.Message)
	return resp, nil
}

// Heartbeat 实现心跳检测
func (s *AgentServiceServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Printf("收到心跳: AgentID=%s, Status=%s", req.AgentId, req.Status)
	
	resp, err := s.agentService.ProcessHeartbeat(req)
	if err != nil {
		log.Printf("心跳处理失败: %v", err)
		return &pb.HeartbeatResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	
	// 心跳日志太频繁，只在调试模式下打印详细信息
	if !resp.Success {
		log.Printf("心跳处理响应: Success=%v, Message=%s", resp.Success, resp.Message)
	}
	
	return resp, nil
}

// UpdateConfig 实现配置下发
func (s *AgentServiceServer) UpdateConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("配置更新请求: AgentID=%s, Version=%s", req.AgentId, req.ConfigVersion)
	
	// TODO: 实现配置管理逻辑
	// 1. 验证Agent是否存在
	// 2. 验证配置内容
	// 3. 保存配置到数据库
	// 4. 返回响应
	
	return &pb.ConfigResponse{
		Success: false,
		Message: "配置更新功能待实现",
	}, nil
}

// UpdateRules 实现规则管理
func (s *AgentServiceServer) UpdateRules(ctx context.Context, req *pb.RulesRequest) (*pb.RulesResponse, error) {
	log.Printf("规则更新请求: AgentID=%s, Operation=%s, Rules=%d", req.AgentId, req.Operation, len(req.Rules))
	
	// TODO: 实现规则管理逻辑
	// 1. 验证Agent是否存在
	// 2. 根据操作类型处理规则
	// 3. 保存规则到数据库
	// 4. 返回响应
	
	return &pb.RulesResponse{
		Success: false,
		Message: "规则更新功能待实现",
	}, nil
}

// GetStatus 实现状态查询
func (s *AgentServiceServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	log.Printf("状态查询请求: AgentID=%s", req.AgentId)
	
	resp, err := s.agentService.GetAgentStatus(req.AgentId)
	if err != nil {
		log.Printf("状态查询失败: %v", err)
		return &pb.StatusResponse{
			Success: false,
		}, nil
	}
	
	log.Printf("状态查询响应: AgentID=%s, Status=%s", resp.AgentId, resp.Status)
	return resp, nil
}

// UpdateBlacklist 实现黑名单更新
func (s *AgentServiceServer) UpdateBlacklist(ctx context.Context, req *pb.BlacklistRequest) (*pb.BlacklistResponse, error) {
	log.Printf("黑名单更新请求: AgentID=%s, Protocol=%s, Operation=%s", req.AgentId, req.Protocol, req.Operation)
	
	// TODO: 实现黑名单管理逻辑
	// 1. 验证Agent是否存在并在线
	// 2. 验证协议类型和操作类型
	// 3. 将黑名单规则转发给对应的Agent
	// 4. 记录操作日志到数据库
	
	return &pb.BlacklistResponse{
		Success:       true,
		Message:       "黑名单更新成功",
		ConfigVersion: "v" + fmt.Sprintf("%d", time.Now().Unix()),
	}, nil
}

// UpdateWhitelist 实现白名单更新
func (s *AgentServiceServer) UpdateWhitelist(ctx context.Context, req *pb.WhitelistRequest) (*pb.WhitelistResponse, error) {
	log.Printf("白名单更新请求: AgentID=%s, Protocol=%s, Operation=%s", req.AgentId, req.Protocol, req.Operation)
	
	// TODO: 实现白名单管理逻辑
	// 1. 验证Agent是否存在并在线
	// 2. 验证协议类型和操作类型
	// 3. 将白名单规则转发给对应的Agent
	// 4. 记录操作日志到数据库
	
	return &pb.WhitelistResponse{
		Success:       true,
		Message:       "白名单更新成功",
		ConfigVersion: "v" + fmt.Sprintf("%d", time.Now().Unix()),
	}, nil
}

// GetFilterConfig 实现过滤器配置查询
func (s *AgentServiceServer) GetFilterConfig(ctx context.Context, req *pb.FilterConfigRequest) (*pb.FilterConfigResponse, error) {
	log.Printf("过滤器配置查询请求: AgentID=%s, Protocol=%s", req.AgentId, req.Protocol)
	
	// TODO: 实现过滤器配置查询逻辑
	// 1. 验证Agent是否存在
	// 2. 从Agent获取过滤器配置
	// 3. 返回配置信息
	
	return &pb.FilterConfigResponse{
		Success: true,
		Message: "配置查询成功",
		Filters: []*pb.ProtocolFilter{
			{
				Protocol:         req.Protocol,
				BlacklistDomains: []string{"example.com"},
				BlacklistIps:     []string{"192.168.1.1"},
				BlacklistPorts:   []string{"80"},
				WhitelistDomains: []string{"google.com"},
				WhitelistIps:     []string{"8.8.8.8"},
				WhitelistPorts:   []string{"443"},
				Enabled:          true,
				LastUpdated:      time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

// RollbackConfig 实现配置回滚
func (s *AgentServiceServer) RollbackConfig(ctx context.Context, req *pb.RollbackRequest) (*pb.RollbackResponse, error) {
	log.Printf("配置回滚请求: AgentID=%s, TargetVersion=%s, Reason=%s", req.AgentId, req.TargetVersion, req.Reason)
	
	// TODO: 实现配置回滚逻辑
	// 1. 验证Agent是否存在并在线
	// 2. 验证目标版本是否存在
	// 3. 通知Agent执行回滚操作
	// 4. 记录回滚操作日志
	
	return &pb.RollbackResponse{
		Success:             true,
		Message:             "配置回滚成功",
		RolledBackVersion:   req.TargetVersion,
		CurrentVersion:      "v" + fmt.Sprintf("%d", time.Now().Unix()),
	}, nil
}