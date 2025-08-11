package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/xbox/sing-box-manager/internal/agent/grpc"
	"github.com/xbox/sing-box-manager/internal/agent/singbox"
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
	
	// 检查和安装sing-box
	if err := checkAndInstallSingbox(cfg); err != nil {
		log.Fatalf("sing-box检查安装失败: %v", err)
	}
	
	// 创建gRPC客户端
	client := grpc.NewClient(cfg)
	
	// 启动gRPC服务器（用于接收Controller的配置推送）
	server := grpc.NewServer(client, "9091")
	if err := server.Start(); err != nil {
		log.Fatalf("启动gRPC服务器失败: %v", err)
	}
	defer server.Stop()
	
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
	
	// 输出sing-box配置信息
	if err := outputSingboxConfig(cfg); err != nil {
		log.Printf("输出sing-box配置信息失败: %v", err)
	}
	
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
	
	// 检查配置文件是否存在
	if _, err := os.Stat(cfg.Agent.SingBoxConfig); os.IsNotExist(err) {
		log.Printf("sing-box配置文件不存在: %s", cfg.Agent.SingBoxConfig)
		return false
	}
	
	return true
}

// checkAndInstallSingbox 检查和安装sing-box
func checkAndInstallSingbox(cfg *config.Config) error {
	log.Println("检查sing-box安装状态...")
	
	// 确定安装目录
	installDir := "./bin"
	if cfg.Agent.SingBoxBinary != "" {
		installDir = filepath.Dir(cfg.Agent.SingBoxBinary)
	}
	
	// 创建安装器
	installer := singbox.NewInstaller(installDir)
	
	// 检查是否已安装
	installed, version, err := installer.Check()
	if err != nil {
		return fmt.Errorf("检查sing-box状态失败: %v", err)
	}
	
	if installed {
		log.Printf("sing-box已安装，版本: %s", version)
		log.Printf("二进制路径: %s", installer.GetBinaryPath())
		
		// 更新配置中的二进制路径
		if cfg.Agent.SingBoxBinary == "" || cfg.Agent.SingBoxBinary != installer.GetBinaryPath() {
			cfg.Agent.SingBoxBinary = installer.GetBinaryPath()
			log.Printf("已更新sing-box二进制路径: %s", cfg.Agent.SingBoxBinary)
		}
		
		return nil
	}
	
	log.Println("sing-box未安装，开始自动安装...")
	
	// 执行安装
	if err := installer.Install(); err != nil {
		return fmt.Errorf("安装sing-box失败: %v", err)
	}
	
	// 重新检查安装结果
	installed, version, err = installer.Check()
	if err != nil {
		return fmt.Errorf("安装后检查失败: %v", err)
	}
	
	if !installed {
		return fmt.Errorf("sing-box安装失败")
	}
	
	log.Printf("sing-box安装成功，版本: %s", version)
	log.Printf("二进制路径: %s", installer.GetBinaryPath())
	
	// 更新配置中的二进制路径
	cfg.Agent.SingBoxBinary = installer.GetBinaryPath()
	
	return nil
}

// outputSingboxConfig 输出sing-box配置信息
func outputSingboxConfig(cfg *config.Config) error {
	if cfg.Agent.SingBoxConfig == "" {
		log.Println("未指定sing-box配置文件路径")
		return nil
	}
	
	log.Println("正在加载sing-box配置信息...")
	
	// 创建管理器
	manager := singbox.NewManager(cfg.Agent.SingBoxBinary, cfg.Agent.SingBoxConfig)
	
	// 输出配置信息
	if err := manager.PrintConfigInfo(); err != nil {
		return fmt.Errorf("输出配置信息失败: %v", err)
	}
	
	return nil
}