package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/xbox/sing-box-manager/api"
	"github.com/xbox/sing-box-manager/internal/config"
	"github.com/xbox/sing-box-manager/internal/controller/grpc"
	"github.com/xbox/sing-box-manager/internal/controller/repository"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	"github.com/xbox/sing-box-manager/internal/database"
	"github.com/xbox/sing-box-manager/pkg/logger"
)

var (
	configFile   = flag.String("config", "configs/config.yaml", "配置文件路径")
	showVersion  = flag.Bool("version", false, "显示版本号")
	showHelp     = flag.Bool("help", false, "显示帮助信息")
)

const (
	Version = "0.1.0"
	Name    = "Xbox Controller"
)

func main() {
	flag.Parse()
	
	if *showVersion {
		fmt.Printf("%s %s\n", Name, Version)
		return
	}
	
	if *showHelp {
		fmt.Printf("%s - Xbox Sing-box管理系统控制器\n\n", Name)
		fmt.Println("选项:")
		flag.PrintDefaults()
		return
	}
	
	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	
	log.Printf("Xbox Controller %s 启动中...", Version)
	
	// 初始化数据库
	if err := database.Init(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()
	
	log.Println("数据库连接成功")
	
	// 初始化日志器
	loggerConfig := &logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		File:       cfg.GetLogFile(),
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   true,
	}
	appLogger, _ := logger.NewLogger(loggerConfig)
	
	// 初始化依赖
	db := database.GetDB()
	agentRepo := repository.NewAgentRepository(db)
	agentService := service.NewAgentService(agentRepo)
	
	// 创建Agent客户端和多路复用服务
	agentClient := service.NewAgentClient()
	multiplexService := service.NewMultiplexService(db, agentClient)
	
	// 创建节点上报服务
	var reportService *service.NodeReportService
	if cfg.Report.Enabled {
		reportService = service.NewNodeReportService(db, agentRepo, cfg.Report.BackendURL, *appLogger)
	}
	
	// 创建服务器
	grpcServer := grpc.NewServer(cfg, agentService, multiplexService, reportService)
	httpServer := api.NewServer(cfg, agentService, multiplexService, reportService)
	
	// 使用WaitGroup等待所有服务启动
	var wg sync.WaitGroup
	
	// 启动gRPC服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcServer.Start(); err != nil {
			log.Fatalf("gRPC服务器启动失败: %v", err)
		}
	}()
	
	// 启动HTTP服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(); err != nil {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()
	
	// 启动节点上报服务
	if reportService != nil {
		ctx, cancelReport := context.WithCancel(context.Background())
		go func() {
			reportInterval := time.Duration(cfg.Report.Interval) * time.Second
			reportService.StartReporting(ctx, reportInterval)
		}()
		
		// 延迟关闭报告服务
		defer func() {
			cancelReport()
			if reportService != nil {
				reportService.StopReporting()
			}
		}()
		
		log.Printf("节点上报服务已启动，间隔: %d秒", cfg.Report.Interval)
	}
	
	log.Printf("Controller服务已启动")
	log.Printf("gRPC地址: %s", cfg.GetGRPCAddr())
	log.Printf("HTTP地址: %s", cfg.GetServerAddr())
	
	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	log.Println("正在关闭服务...")
	
	// 优雅关闭服务器
	grpcServer.Stop()
	if err := httpServer.Stop(); err != nil {
		log.Printf("HTTP服务器关闭失败: %v", err)
	}
}