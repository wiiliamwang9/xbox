package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/xbox/sing-box-manager/internal/config"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	pb "github.com/xbox/sing-box-manager/proto/agent"
	backendpb "github.com/xbox/sing-box-manager/proto/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Server gRPC服务器
type Server struct {
	config           *config.Config
	grpcServer       *grpc.Server
	agentService     service.AgentService
	multiplexService service.MultiplexService
	reportService    *service.NodeReportService
}

// NewServer 创建gRPC服务器实例
func NewServer(cfg *config.Config, agentService service.AgentService, multiplexService service.MultiplexService, reportService *service.NodeReportService) *Server {
	return &Server{
		config:           cfg,
		agentService:     agentService,
		multiplexService: multiplexService,
		reportService:    reportService,
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

	// 如果启用了TLS + mTLS
	if s.config.GRPC.TLS.Enabled {
		log.Println("启用TLS + mTLS双向认证...")
		
		// 加载TLS凭据
		creds, err := s.loadTLSCredentials()
		if err != nil {
			return fmt.Errorf("加载TLS凭据失败: %v", err)
		}
		
		opts = append(opts, grpc.Creds(creds))
		log.Printf("TLS + mTLS配置成功，证书文件: %s", s.config.GetTLSCertFile())
	}

	// 创建gRPC服务器
	s.grpcServer = grpc.NewServer(opts...)

	// 注册服务
	agentServiceServer := NewAgentServiceServer(s.agentService)
	pb.RegisterAgentServiceServer(s.grpcServer, agentServiceServer)
	
	// 注册后端服务接口
	backendServiceServer := NewBackendServiceServer(s.agentService, s.multiplexService, s.reportService)
	backendpb.RegisterBackendServiceServer(s.grpcServer, backendServiceServer)

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

// loadTLSCredentials 加载TLS + mTLS凭据
func (s *Server) loadTLSCredentials() (credentials.TransportCredentials, error) {
	// 获取证书文件的完整路径
	certFile := s.config.GetTLSCertFile()
	keyFile := s.config.GetTLSKeyFile()
	caFile := s.config.GetTLSCAFile()
	
	// 读取服务器证书和私钥
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载服务器证书失败: %v", err)
	}

	// 读取CA证书
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("读取CA证书失败: %v", err)
	}

	// 创建CA证书池
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("解析CA证书失败")
	}

	// 配置TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // mTLS: 要求并验证客户端证书
		ClientCAs:    caCertPool,                     // 验证客户端证书的CA
		MinVersion:   tls.VersionTLS12,               // 最低TLS版本
		MaxVersion:   tls.VersionTLS13,               // 最高TLS版本
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
	}

	log.Printf("TLS配置详情:")
	log.Printf("  服务器证书: %s", certFile)
	log.Printf("  服务器私钥: %s", keyFile)
	log.Printf("  CA证书: %s", caFile)
	log.Printf("  客户端认证: 必须验证")
	log.Printf("  TLS版本: 1.2-1.3")

	return credentials.NewTLS(tlsConfig), nil
}