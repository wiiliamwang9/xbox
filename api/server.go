package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xbox/sing-box-manager/api/routes"
	"github.com/xbox/sing-box-manager/internal/config"
	"github.com/xbox/sing-box-manager/internal/controller/service"
)

// Server HTTP API服务器
type Server struct {
	config           *config.Config
	httpServer       *http.Server
	agentService     service.AgentService
	multiplexService service.MultiplexService
}

// NewServer 创建HTTP服务器实例
func NewServer(cfg *config.Config, agentService service.AgentService, multiplexService service.MultiplexService) *Server {
	return &Server{
		config:           cfg,
		agentService:     agentService,
		multiplexService: multiplexService,
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	// 设置Gin模式
	gin.SetMode(s.config.Server.Mode)
	
	// 创建Gin引擎
	r := gin.New()
	
	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	
	// 设置路由
	routes.SetupRoutes(r, s.agentService, s.multiplexService)
	
	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:           s.config.GetServerAddr(),
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	log.Printf("HTTP API服务器启动成功，监听地址: %s", s.config.GetServerAddr())
	
	// 启动服务器
	return s.httpServer.ListenAndServe()
}

// Stop 停止HTTP服务器
func (s *Server) Stop() error {
	if s.httpServer != nil {
		log.Println("正在停止HTTP服务器...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP服务器停止失败: %v", err)
		}
		
		log.Println("HTTP服务器已停止")
	}
	return nil
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}