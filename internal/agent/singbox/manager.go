package singbox

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Manager sing-box进程管理器
type Manager struct {
	mu          sync.RWMutex
	process     *os.Process
	configPath  string
	binaryPath  string
	running     bool
	lastConfig  *Config
}

// Config sing-box配置结构
type Config struct {
	Version  string      `json:"version,omitempty"`
	Log      *LogConfig  `json:"log,omitempty"`
	DNS      *DNSConfig  `json:"dns,omitempty"`
	Inbounds []Inbound   `json:"inbounds,omitempty"`
	Outbounds []Outbound `json:"outbounds,omitempty"`
	Route    *RouteConfig `json:"route,omitempty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level     string `json:"level,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
}

// DNSConfig DNS配置
type DNSConfig struct {
	Servers []DNSServer `json:"servers,omitempty"`
}

// DNSServer DNS服务器配置
type DNSServer struct {
	Tag     string `json:"tag,omitempty"`
	Address string `json:"address,omitempty"`
}

// Inbound 入站配置
type Inbound struct {
	Tag      string      `json:"tag,omitempty"`
	Type     string      `json:"type,omitempty"`
	Listen   string      `json:"listen,omitempty"`
	Port     int         `json:"listen_port,omitempty"`
	Settings interface{} `json:"settings,omitempty"`
}

// Outbound 出站配置
type Outbound struct {
	Tag      string      `json:"tag,omitempty"`
	Type     string      `json:"type,omitempty"`
	Settings interface{} `json:"settings,omitempty"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Rules []RouteRule `json:"rules,omitempty"`
}

// RouteRule 路由规则
type RouteRule struct {
	Domain   []string `json:"domain,omitempty"`
	IP       []string `json:"ip,omitempty"`
	Port     string   `json:"port,omitempty"`
	Outbound string   `json:"outbound,omitempty"`
}

// NewManager 创建sing-box管理器
func NewManager(binaryPath, configPath string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		configPath: configPath,
		running:    false,
	}
}

// Start 启动sing-box进程
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("sing-box已在运行")
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", m.configPath)
	}

	// 启动sing-box进程
	cmd := exec.Command(m.binaryPath, "run", "-c", m.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动sing-box失败: %v", err)
	}

	m.process = cmd.Process
	m.running = true

	log.Printf("sing-box进程已启动, PID: %d", m.process.Pid)

	// 启动进程监控
	go m.monitorProcess(cmd)

	return nil
}

// Stop 停止sing-box进程
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.process == nil {
		return fmt.Errorf("sing-box未运行")
	}

	// 发送SIGTERM信号
	if err := m.process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("停止sing-box失败: %v", err)
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		_, err := m.process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		m.running = false
		m.process = nil
		log.Println("sing-box进程已停止")
		return err
	case <-time.After(10 * time.Second):
		// 强制杀死进程
		if err := m.process.Kill(); err != nil {
			return fmt.Errorf("强制终止sing-box失败: %v", err)
		}
		m.running = false
		m.process = nil
		log.Println("sing-box进程已被强制终止")
		return nil
	}
}

// Restart 重启sing-box进程
func (m *Manager) Restart() error {
	if m.IsRunning() {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("停止进程失败: %v", err)
		}
	}

	time.Sleep(1 * time.Second) // 等待进程完全停止

	return m.Start()
}

// IsRunning 检查进程是否运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetPID 获取进程ID
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.process != nil {
		return m.process.Pid
	}
	return 0
}

// UpdateConfig 更新配置并重启
func (m *Manager) UpdateConfig(config *Config) error {
	// 备份当前配置
	if err := m.backupConfig(); err != nil {
		log.Printf("备份配置失败: %v", err)
	}

	// 写入新配置
	if err := m.writeConfig(config); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}

	// 验证配置
	if err := m.validateConfig(); err != nil {
		// 配置无效，恢复备份
		m.restoreConfig()
		return fmt.Errorf("配置验证失败: %v", err)
	}

	m.lastConfig = config

	// 重启服务应用新配置
	if m.IsRunning() {
		return m.Restart()
	}

	return nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastConfig
}

// monitorProcess 监控进程状态
func (m *Manager) monitorProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	
	m.mu.Lock()
	m.running = false
	m.process = nil
	m.mu.Unlock()

	if err != nil {
		log.Printf("sing-box进程异常退出: %v", err)
	} else {
		log.Println("sing-box进程正常退出")
	}
}

// writeConfig 写入配置文件
func (m *Manager) writeConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// validateConfig 验证配置文件
func (m *Manager) validateConfig() error {
	// 使用sing-box检查配置
	cmd := exec.Command(m.binaryPath, "check", "-c", m.configPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}
	return nil
}

// backupConfig 备份配置文件
func (m *Manager) backupConfig() error {
	backupPath := m.configPath + ".backup"
	
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupPath, data, 0644)
}

// restoreConfig 恢复配置文件
func (m *Manager) restoreConfig() error {
	backupPath := m.configPath + ".backup"
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在")
	}
	
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(m.configPath, data, 0644)
}

// GetStatus 获取进程状态信息
func (m *Manager) GetStatus() map[string]string {
	status := map[string]string{
		"running":     fmt.Sprintf("%t", m.IsRunning()),
		"pid":         fmt.Sprintf("%d", m.GetPID()),
		"config_path": m.configPath,
		"binary_path": m.binaryPath,
	}

	if m.IsRunning() {
		status["status"] = "running"
	} else {
		status["status"] = "stopped"
	}

	// 获取配置文件修改时间
	if stat, err := os.Stat(m.configPath); err == nil {
		status["config_modified"] = stat.ModTime().Format(time.RFC3339)
	}

	return status
}