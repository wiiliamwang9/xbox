package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xbox/sing-box-manager/internal/agent/grpc"
	"github.com/xbox/sing-box-manager/internal/config"
)

var (
	configFile   = flag.String("config", "configs/config.yaml", "配置文件路径")
	showVersion  = flag.Bool("version", false, "显示版本号")
	showHelp     = flag.Bool("help", false, "显示帮助信息")
)

const (
	Version = "0.1.0"
	Name    = "Xbox Agent"
)

func main() {
	flag.Parse()
	
	if *showVersion {
		fmt.Printf("%s %s\n", Name, Version)
		return
	}
	
	if *showHelp {
		fmt.Printf("%s - Xbox Sing-box管理系统代理节点\n\n", Name)
		fmt.Println("选项:")
		flag.PrintDefaults()
		return
	}
	
	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	
	log.Printf("Xbox Agent %s 启动中...", Version)
	
	// 创建gRPC客户端
	client := grpc.NewClient(cfg)
	
	// 连接到Controller
	if err := client.Connect(); err != nil {
		log.Fatalf("连接Controller失败: %v", err)
	}
	defer client.Close()
	
	// 注册Agent
	if err := client.Register(); err != nil {
		log.Fatalf("注册Agent失败: %v", err)
	}
	
	log.Printf("Agent服务已启动")
	log.Printf("Agent ID: %s", client.GetAgentID())
	log.Printf("Controller地址: %s", cfg.Agent.ControllerAddr)
	
	// 启动心跳循环
	go client.StartHeartbeat()
	
	// 如果配置中指定了启动sing-box，则自动启动
	if shouldStartSingbox(cfg) {
		log.Println("正在启动sing-box服务...")
		if err := client.StartSingbox(); err != nil {
			log.Printf("启动sing-box失败: %v", err)
		} else {
			log.Println("sing-box服务已启动")
		}
	}
	
	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	log.Println("正在关闭服务...")
	
	// 停止sing-box服务
	if err := client.StopSingbox(); err != nil {
		log.Printf("停止sing-box失败: %v", err)
	}
}

// shouldStartSingbox 检查是否应该启动sing-box
func shouldStartSingbox(cfg *config.Config) bool {
	// 检查配置文件是否存在
	if cfg.Agent.SingBoxConfig == "" || cfg.Agent.SingBoxBinary == "" {
		return false
	}
	
	// 可以添加更多检查逻辑
	return true
}