package grpc

import (
	"fmt"
	"log"
	"net"

	"github.com/xbox/sing-box-manager/internal/config"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	pb "github.com/xbox/sing-box-manager/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server gRPC服务器
type Server struct {
	config      *config.Config
	grpcServer  *grpc.Server
	agentService service.AgentService
}

// NewServer 创建gRPC服务器实例
func NewServer(cfg *config.Config, agentService service.AgentService) *Server {
	return &Server{
		config:      cfg,
		agentService: agentService,
	}
}

// Start 启动gRPC服务器
func (s *Server) Start() error {
	// 创建监听器
	listener, err := net.Listen("tcp", s.config.GetGRPCAddr())
	if err != nil {
		return fmt.Errorf("创建gRPC监听器失败: %v", err)
	}

	// 创建gRPC服务器选项
	opts := []grpc.ServerOption{
		// TODO: 添加拦截器
		// grpc.UnaryInterceptor(s.unaryInterceptor),
		// grpc.StreamInterceptor(s.streamInterceptor),
	}

	// 如果启用了TLS
	if s.config.GRPC.TLS.Enabled {
		// TODO: 添加TLS配置
		log.Println("警告: TLS配置已启用但未实现")
	}

	// 创建gRPC服务器
	s.grpcServer = grpc.NewServer(opts...)

	// 注册服务
	agentServiceServer := NewAgentServiceServer(s.agentService)
	pb.RegisterAgentServiceServer(s.grpcServer, agentServiceServer)

	// 启用反射服务(用于调试)
	reflection.Register(s.grpcServer)

	log.Printf("gRPC服务器启动成功，监听地址: %s", s.config.GetGRPCAddr())

	// 启动服务器
	return s.grpcServer.Serve(listener)
}

// Stop 停止gRPC服务器
func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Println("正在停止gRPC服务器...")
		s.grpcServer.GracefulStop()
		log.Println("gRPC服务器已停止")
	}
}

// GetServer 获取gRPC服务器实例(用于测试)
func (s *Server) GetServer() *grpc.Server {
	return s.grpcServer
}