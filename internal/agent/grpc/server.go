package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/xbox/sing-box-manager/proto/agent"
	"google.golang.org/grpc"
)

// Server Agent gRPC服务器
type Server struct {
	pb.UnimplementedAgentServiceServer
	client *Client
	port   string
	server *grpc.Server
}

// NewServer 创建Agent gRPC服务器
func NewServer(client *Client, port string) *Server {
	return &Server{
		client: client,
		port:   port,
	}
}

// Start 启动gRPC服务器
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	s.server = grpc.NewServer()
	pb.RegisterAgentServiceServer(s.server, s)

	log.Printf("Agent gRPC服务器启动在端口: %s", s.port)
	
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Printf("gRPC服务器错误: %v", err)
		}
	}()

	return nil
}

// Stop 停止gRPC服务器
func (s *Server) Stop() {
	if s.server != nil {
		log.Println("正在停止Agent gRPC服务器...")
		s.server.GracefulStop()
	}
}

// UpdateMultiplexConfig 处理多路复用配置更新请求
func (s *Server) UpdateMultiplexConfig(ctx context.Context, req *pb.MultiplexConfigRequest) (*pb.MultiplexConfigResponse, error) {
	log.Printf("=== 收到多路复用配置更新请求 ===")
	log.Printf("Agent ID: %s", req.AgentId)
	log.Printf("Protocol: %s", req.Protocol)
	log.Printf("配置详情:")
	
	if req.MultiplexConfig != nil {
		log.Printf("  启用状态: %t", req.MultiplexConfig.Enabled)
		log.Printf("  协议: %s", req.MultiplexConfig.Protocol)
		log.Printf("  最大连接数: %d", req.MultiplexConfig.MaxConnections)
		log.Printf("  最小流数: %d", req.MultiplexConfig.MinStreams)
		log.Printf("  填充: %t", req.MultiplexConfig.Padding)
		if len(req.MultiplexConfig.Brutal) > 0 {
			log.Printf("  Brutal配置: %+v", req.MultiplexConfig.Brutal)
		}
	}

	// 检查Agent ID是否匹配
	if req.AgentId != s.client.GetAgentID() {
		log.Printf("Agent ID不匹配: 期望=%s, 收到=%s", s.client.GetAgentID(), req.AgentId)
		return &pb.MultiplexConfigResponse{
			Success: false,
			Message: "Agent ID不匹配",
		}, nil
	}

	// 转换protobuf配置为map格式
	config := make(map[string]interface{})
	if req.MultiplexConfig != nil {
		config["enabled"] = req.MultiplexConfig.Enabled
		config["protocol"] = req.MultiplexConfig.Protocol
		config["max_connections"] = int(req.MultiplexConfig.MaxConnections)
		config["min_streams"] = int(req.MultiplexConfig.MinStreams)
		config["padding"] = req.MultiplexConfig.Padding
		
		if len(req.MultiplexConfig.Brutal) > 0 {
			brutal := make(map[string]interface{})
			for k, v := range req.MultiplexConfig.Brutal {
				brutal[k] = v
			}
			config["brutal"] = brutal
		}
	}

	// 调用客户端的多路复用配置更新方法
	startTime := time.Now()
	err := s.client.UpdateMultiplexConfig(req.Protocol, config)
	duration := time.Since(startTime)

	configVersion := fmt.Sprintf("v%d", time.Now().Unix())
	
	if err != nil {
		log.Printf("多路复用配置更新失败: %v (耗时: %v)", err, duration)
		return &pb.MultiplexConfigResponse{
			Success:       false,
			Message:       fmt.Sprintf("配置更新失败: %v", err),
			ConfigVersion: configVersion,
		}, nil
	}

	log.Printf("多路复用配置更新成功 (耗时: %v)", duration)
	log.Printf("配置版本: %s", configVersion)
	log.Printf("=== 多路复用配置更新请求处理完成 ===")

	return &pb.MultiplexConfigResponse{
		Success:       true,
		Message:       "多路复用配置更新成功",
		ConfigVersion: configVersion,
	}, nil
}

// GetMultiplexConfig 处理获取多路复用配置请求
func (s *Server) GetMultiplexConfig(ctx context.Context, req *pb.MultiplexStatusRequest) (*pb.MultiplexStatusResponse, error) {
	log.Printf("=== 收到获取多路复用配置请求 ===")
	log.Printf("Agent ID: %s", req.AgentId)
	log.Printf("Protocol: %s", req.Protocol)

	// 检查Agent ID是否匹配
	if req.AgentId != s.client.GetAgentID() {
		log.Printf("Agent ID不匹配: 期望=%s, 收到=%s", s.client.GetAgentID(), req.AgentId)
		return &pb.MultiplexStatusResponse{
			Success: false,
			Message: "Agent ID不匹配",
		}, nil
	}

	// 获取多路复用配置
	configs, err := s.client.GetMultiplexConfig(req.Protocol)
	if err != nil {
		log.Printf("获取多路复用配置失败: %v", err)
		return &pb.MultiplexStatusResponse{
			Success: false,
			Message: fmt.Sprintf("获取配置失败: %v", err),
		}, nil
	}

	// 转换配置为protobuf格式
	var multiplexConfigs []*pb.ProtocolMultiplex
	for protocol, configInterface := range configs {
		if configInterface == nil {
			continue
		}
		
		// 类型断言为map[string]interface{}
		configMap, ok := configInterface.(map[string]interface{})
		if !ok {
			continue
		}

		protocolMultiplex := &pb.ProtocolMultiplex{
			Protocol: protocol,
			Enabled:  false,
		}

		if enabled, ok := configMap["enabled"].(bool); ok {
			protocolMultiplex.Enabled = enabled
		}

		if enabled, ok := configMap["enabled"].(bool); ok && enabled {
			multiplexConfig := &pb.MultiplexConfig{
				Enabled: true,
			}

			if proto, ok := configMap["protocol"].(string); ok {
				multiplexConfig.Protocol = proto
			}
			if maxConn, ok := configMap["max_connections"].(int); ok {
				multiplexConfig.MaxConnections = int32(maxConn)
			}
			if minStreams, ok := configMap["min_streams"].(int); ok {
				multiplexConfig.MinStreams = int32(minStreams)
			}
			if padding, ok := configMap["padding"].(bool); ok {
				multiplexConfig.Padding = padding
			}
			if brutal, ok := configMap["brutal"].(map[string]interface{}); ok && brutal != nil {
				brutalMap := make(map[string]string)
				for k, v := range brutal {
					if str, ok := v.(string); ok {
						brutalMap[k] = str
					}
				}
				multiplexConfig.Brutal = brutalMap
			}

			protocolMultiplex.MultiplexConfig = multiplexConfig
		}

		if lastUpdated, ok := configMap["last_updated"].(string); ok {
			protocolMultiplex.LastUpdated = lastUpdated
		}

		multiplexConfigs = append(multiplexConfigs, protocolMultiplex)
		
		log.Printf("协议 %s 配置: enabled=%t", protocol, protocolMultiplex.Enabled)
	}

	log.Printf("成功获取 %d 个协议的多路复用配置", len(multiplexConfigs))
	log.Printf("=== 获取多路复用配置请求处理完成 ===")

	return &pb.MultiplexStatusResponse{
		Success:          true,
		Message:          "获取多路复用配置成功",
		MultiplexConfigs: multiplexConfigs,
	}, nil
}

// UpdateConfig 处理配置更新请求（保留原有接口兼容性）
func (s *Server) UpdateConfig(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	log.Printf("收到配置更新请求: Agent=%s, Version=%s", req.AgentId, req.ConfigVersion)
	
	if req.AgentId != s.client.GetAgentID() {
		return &pb.ConfigResponse{
			Success: false,
			Message: "Agent ID不匹配",
		}, nil
	}

	// 调用客户端的配置更新方法
	err := s.client.UpdateConfig(req.ConfigContent)
	if err != nil {
		log.Printf("配置更新失败: %v", err)
		return &pb.ConfigResponse{
			Success:        false,
			Message:        fmt.Sprintf("配置更新失败: %v", err),
			AppliedVersion: req.ConfigVersion,
		}, nil
	}

	log.Printf("配置更新成功: Version=%s", req.ConfigVersion)
	return &pb.ConfigResponse{
		Success:        true,
		Message:        "配置更新成功",
		AppliedVersion: req.ConfigVersion,
	}, nil
}

// UpdateRules 处理规则更新请求
func (s *Server) UpdateRules(ctx context.Context, req *pb.RulesRequest) (*pb.RulesResponse, error) {
	log.Printf("收到规则更新请求: Agent=%s, Operation=%s, Rules=%d", 
		req.AgentId, req.Operation, len(req.Rules))
	
	if req.AgentId != s.client.GetAgentID() {
		return &pb.RulesResponse{
			Success: false,
			Message: "Agent ID不匹配",
		}, nil
	}

	// 调用客户端的规则更新方法
	err := s.client.UpdateRules(req.Rules)
	if err != nil {
		log.Printf("规则更新失败: %v", err)
		return &pb.RulesResponse{
			Success: false,
			Message: fmt.Sprintf("规则更新失败: %v", err),
		}, nil
	}

	log.Printf("规则更新成功")
	return &pb.RulesResponse{
		Success: true,
		Message: "规则更新成功",
	}, nil
}