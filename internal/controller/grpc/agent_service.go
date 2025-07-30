package grpc

import (
	"context"
	"log"

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