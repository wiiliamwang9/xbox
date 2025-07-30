package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	
	// TODO: 连接到Controller
	// TODO: 初始化sing-box管理
	
	log.Printf("Agent服务已启动")
	log.Printf("Controller地址: %s", cfg.Agent.ControllerAddr)
	
	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	log.Println("正在关闭服务...")
}