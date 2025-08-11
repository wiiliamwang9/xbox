package uninstall

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// UninstallManager Agent卸载管理器
type UninstallManager struct {
	singboxBinaryPath string
	singboxConfigPath string
	cleanupPaths      []string
}

// UninstallResult 卸载结果
type UninstallResult struct {
	Success       bool          `json:"success"`
	Message       string        `json:"message"`
	Status        string        `json:"status"`
	CleanedFiles  []string      `json:"cleaned_files"`
	CleanupTime   time.Duration `json:"cleanup_time"`
	Error         error         `json:"error,omitempty"`
}

// NewUninstallManager 创建卸载管理器
func NewUninstallManager(singboxBinary, singboxConfig string) *UninstallManager {
	manager := &UninstallManager{
		singboxBinaryPath: singboxBinary,
		singboxConfigPath: singboxConfig,
		cleanupPaths: []string{
			// sing-box相关文件
			singboxConfig,
			"./sing-box.json",
			"./configs/sing-box.json",
			"/etc/sing-box/config.json",
			"/usr/local/etc/sing-box/config.json",
			
			// 过滤器配置文件
			"./configs/filter.json",
			"./filter.json",
			
			// 日志文件
			"./logs/sing-box.log",
			"./logs/agent.log",
			"/var/log/sing-box.log",
			
			// systemd服务文件
			"/etc/systemd/system/sing-box.service",
			"/lib/systemd/system/sing-box.service",
			
			// PID文件
			"./sing-box.pid",
			"/var/run/sing-box.pid",
			"/tmp/sing-box.pid",
			
			// 缓存和临时文件
			"./cache/",
			"/tmp/sing-box/",
			"/var/cache/sing-box/",
			
			// 配置备份文件
			"./configs/sing-box.json.bak",
			"./sing-box.json.bak",
		},
	}
	
	// 添加可能的sing-box二进制路径
	if singboxBinary != "" {
		manager.cleanupPaths = append(manager.cleanupPaths, singboxBinary)
	}
	
	return manager
}

// UninstallSingbox 卸载sing-box及相关配置
func (m *UninstallManager) UninstallSingbox(forceUninstall bool, timeoutSeconds int32) *UninstallResult {
	startTime := time.Now()
	result := &UninstallResult{
		Success:      false,
		Status:       "preparing",
		CleanedFiles: make([]string, 0),
		CleanupTime:  0,
	}
	
	log.Printf("开始卸载sing-box服务...")
	log.Printf("  二进制路径: %s", m.singboxBinaryPath)
	log.Printf("  配置文件路径: %s", m.singboxConfigPath)
	log.Printf("  强制卸载: %t", forceUninstall)
	log.Printf("  超时时间: %d秒", timeoutSeconds)
	
	// 设置超时
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // 默认60秒超时
	}
	
	// 创建带超时的上下文
	done := make(chan bool, 1)
	go func() {
		defer func() {
			done <- true
		}()
		
		// 1. 停止sing-box服务
		result.Status = "stopping_service"
		if err := m.stopSingboxService(forceUninstall); err != nil {
			if !forceUninstall {
				result.Error = fmt.Errorf("停止sing-box服务失败: %v", err)
				result.Message = result.Error.Error()
				return
			}
			log.Printf("警告: 停止sing-box服务失败，但强制模式继续: %v", err)
		}
		log.Printf("sing-box服务已停止")
		
		// 2. 清理配置文件和相关文件
		result.Status = "cleaning_files"
		cleanedFiles := m.cleanupFiles(forceUninstall)
		result.CleanedFiles = cleanedFiles
		log.Printf("已清理 %d 个文件", len(cleanedFiles))
		
		// 3. 移除systemd服务文件
		result.Status = "cleaning_service"
		if err := m.removeSystemdService(); err != nil {
			if !forceUninstall {
				result.Error = fmt.Errorf("移除systemd服务失败: %v", err)
				result.Message = result.Error.Error()
				return
			}
			log.Printf("警告: 移除systemd服务失败，但强制模式继续: %v", err)
		}
		
		// 4. 清理sing-box二进制文件（可选）
		result.Status = "cleaning_binary"
		if err := m.cleanupBinary(forceUninstall); err != nil {
			if !forceUninstall {
				result.Error = fmt.Errorf("清理二进制文件失败: %v", err)
				result.Message = result.Error.Error()
				return
			}
			log.Printf("警告: 清理二进制文件失败，但强制模式继续: %v", err)
		}
		
		// 卸载完成
		result.Status = "completed"
		result.Success = true
		result.Message = "sing-box服务卸载完成"
		log.Printf("sing-box服务卸载完成，清理了 %d 个文件", len(result.CleanedFiles))
	}()
	
	// 等待完成或超时
	select {
	case <-done:
		// 正常完成
	case <-time.After(timeout):
		result.Status = "timeout"
		result.Error = fmt.Errorf("卸载操作超时")
		result.Message = fmt.Sprintf("卸载操作超时（%v）", timeout)
		log.Printf("卸载操作超时: %v", timeout)
	}
	
	result.CleanupTime = time.Since(startTime)
	return result
}

// stopSingboxService 停止sing-box服务
func (m *UninstallManager) stopSingboxService(force bool) error {
	log.Printf("正在停止sing-box服务...")
	
	// 方法1: 尝试通过systemctl停止服务
	if err := m.stopSystemdService(); err == nil {
		log.Printf("通过systemctl成功停止sing-box服务")
		return nil
	} else {
		log.Printf("systemctl停止失败: %v", err)
	}
	
	// 方法2: 尝试通过PID文件停止
	if err := m.stopByPidFile(); err == nil {
		log.Printf("通过PID文件成功停止sing-box服务")
		return nil
	} else {
		log.Printf("PID文件停止失败: %v", err)
	}
	
	// 方法3: 尝试通过进程名强制停止
	if err := m.stopByProcessName(force); err != nil {
		if force {
			log.Printf("警告: 强制停止进程失败: %v", err)
			return nil // 强制模式下忽略错误
		}
		return fmt.Errorf("停止sing-box进程失败: %v", err)
	}
	
	log.Printf("通过进程名成功停止sing-box服务")
	return nil
}

// stopSystemdService 通过systemctl停止服务
func (m *UninstallManager) stopSystemdService() error {
	services := []string{"sing-box", "xbox-singbox", "singbox"}
	
	for _, service := range services {
		cmd := exec.Command("systemctl", "stop", service)
		if err := cmd.Run(); err == nil {
			// 等待服务完全停止
			time.Sleep(2 * time.Second)
			
			// 验证服务是否已停止
			statusCmd := exec.Command("systemctl", "is-active", service)
			output, _ := statusCmd.Output()
			if strings.TrimSpace(string(output)) == "inactive" {
				log.Printf("systemd服务 %s 已停止", service)
				return nil
			}
		}
	}
	
	return fmt.Errorf("未找到活动的systemd服务")
}

// stopByPidFile 通过PID文件停止服务
func (m *UninstallManager) stopByPidFile() error {
	pidFiles := []string{
		"./sing-box.pid",
		"/var/run/sing-box.pid",
		"/tmp/sing-box.pid",
	}
	
	for _, pidFile := range pidFiles {
		if _, err := os.Stat(pidFile); err == nil {
			// 读取PID
			pidBytes, err := os.ReadFile(pidFile)
			if err != nil {
				continue
			}
			
			pid := strings.TrimSpace(string(pidBytes))
			if pid == "" {
				continue
			}
			
			// 发送终止信号
			cmd := exec.Command("kill", "-TERM", pid)
			if err := cmd.Run(); err == nil {
				// 等待进程停止
				time.Sleep(3 * time.Second)
				
				// 验证进程是否已停止
				checkCmd := exec.Command("kill", "-0", pid)
				if checkCmd.Run() != nil {
					log.Printf("通过PID文件成功停止进程: %s", pid)
					// 删除PID文件
					os.Remove(pidFile)
					return nil
				}
				
				// 如果进程仍在运行，发送强制终止信号
				forceCmd := exec.Command("kill", "-KILL", pid)
				forceCmd.Run()
				time.Sleep(1 * time.Second)
				os.Remove(pidFile)
				return nil
			}
		}
	}
	
	return fmt.Errorf("未找到有效的PID文件")
}

// stopByProcessName 通过进程名停止服务
func (m *UninstallManager) stopByProcessName(force bool) error {
	// 查找sing-box进程
	cmd := exec.Command("pgrep", "-f", "sing-box")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("未找到sing-box进程")
	}
	
	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		return fmt.Errorf("未找到sing-box进程")
	}
	
	// 停止所有sing-box进程
	for _, pid := range pids {
		log.Printf("正在停止sing-box进程: %s", pid)
		
		// 先尝试优雅停止
		killCmd := exec.Command("kill", "-TERM", pid)
		killCmd.Run()
	}
	
	// 等待进程停止
	time.Sleep(5 * time.Second)
	
	// 检查是否还有进程运行
	checkCmd := exec.Command("pgrep", "-f", "sing-box")
	if output, err := checkCmd.Output(); err == nil {
		remainingPids := strings.Fields(string(output))
		if len(remainingPids) > 0 && force {
			// 强制停止剩余进程
			for _, pid := range remainingPids {
				log.Printf("强制停止sing-box进程: %s", pid)
				forceCmd := exec.Command("kill", "-KILL", pid)
				forceCmd.Run()
			}
			time.Sleep(2 * time.Second)
		}
	}
	
	return nil
}

// cleanupFiles 清理相关文件
func (m *UninstallManager) cleanupFiles(force bool) []string {
	cleanedFiles := make([]string, 0)
	
	log.Printf("开始清理文件...")
	
	for _, path := range m.cleanupPaths {
		if m.cleanupPath(path, force) {
			cleanedFiles = append(cleanedFiles, path)
			log.Printf("已清理: %s", path)
		}
	}
	
	return cleanedFiles
}

// cleanupPath 清理单个路径
func (m *UninstallManager) cleanupPath(path string, force bool) bool {
	if path == "" {
		return false
	}
	
	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		return false // 文件不存在
	}
	
	if info.IsDir() {
		// 清理目录
		return m.cleanupDirectory(path, force)
	} else {
		// 清理文件
		return m.cleanupFile(path, force)
	}
}

// cleanupFile 清理文件
func (m *UninstallManager) cleanupFile(filePath string, force bool) bool {
	if err := os.Remove(filePath); err != nil {
		if force {
			// 强制模式下尝试更改权限后删除
			os.Chmod(filePath, 0666)
			if err := os.Remove(filePath); err != nil {
				log.Printf("警告: 无法删除文件 %s: %v", filePath, err)
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// cleanupDirectory 清理目录
func (m *UninstallManager) cleanupDirectory(dirPath string, force bool) bool {
	if err := os.RemoveAll(dirPath); err != nil {
		if force {
			// 强制模式下尝试递归更改权限后删除
			filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
				if err == nil {
					os.Chmod(path, 0777)
				}
				return nil
			})
			
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("警告: 无法删除目录 %s: %v", dirPath, err)
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// removeSystemdService 移除systemd服务文件
func (m *UninstallManager) removeSystemdService() error {
	serviceFiles := []string{
		"/etc/systemd/system/sing-box.service",
		"/lib/systemd/system/sing-box.service",
		"/etc/systemd/system/xbox-singbox.service",
		"/lib/systemd/system/xbox-singbox.service",
	}
	
	removed := false
	for _, serviceFile := range serviceFiles {
		if _, err := os.Stat(serviceFile); err == nil {
			log.Printf("移除systemd服务文件: %s", serviceFile)
			
			// 先禁用服务
			exec.Command("systemctl", "disable", filepath.Base(serviceFile)).Run()
			
			// 删除服务文件
			if err := os.Remove(serviceFile); err != nil {
				log.Printf("警告: 无法删除服务文件 %s: %v", serviceFile, err)
			} else {
				removed = true
			}
		}
	}
	
	if removed {
		// 重新加载systemd配置
		exec.Command("systemctl", "daemon-reload").Run()
		log.Printf("已重新加载systemd配置")
	}
	
	return nil
}

// cleanupBinary 清理sing-box二进制文件
func (m *UninstallManager) cleanupBinary(force bool) error {
	if m.singboxBinaryPath == "" {
		return nil
	}
	
	// 检查二进制文件是否存在
	if _, err := os.Stat(m.singboxBinaryPath); err != nil {
		return nil // 文件不存在
	}
	
	log.Printf("清理sing-box二进制文件: %s", m.singboxBinaryPath)
	
	// 只在强制模式下删除二进制文件
	if force {
		if err := os.Remove(m.singboxBinaryPath); err != nil {
			return fmt.Errorf("删除二进制文件失败: %v", err)
		}
		log.Printf("已删除sing-box二进制文件")
	} else {
		log.Printf("非强制模式，保留sing-box二进制文件")
	}
	
	return nil
}

// GetCleanupPaths 获取要清理的路径列表
func (m *UninstallManager) GetCleanupPaths() []string {
	return m.cleanupPaths
}

// AddCleanupPath 添加要清理的路径
func (m *UninstallManager) AddCleanupPath(path string) {
	m.cleanupPaths = append(m.cleanupPaths, path)
}