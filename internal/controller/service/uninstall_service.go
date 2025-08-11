package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xbox/sing-box-manager/internal/models"
)

// UninstallTask 卸载任务
type UninstallTask struct {
	AgentID        string    `json:"agent_id"`
	IP             string    `json:"ip"`
	ForceUninstall bool      `json:"force_uninstall"`
	Reason         string    `json:"reason"`
	TimeoutSeconds int32     `json:"timeout_seconds"`
	DeleteFromDB   bool      `json:"delete_from_db"`
	Status         string    `json:"status"` // pending, sent, in_progress, completed, failed, timeout
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CleanedFiles   []string  `json:"cleaned_files"`
	CleanupTime    int64     `json:"cleanup_time"`
	Error          string    `json:"error,omitempty"`
}

// UninstallService Agent卸载服务接口
type UninstallService interface {
	// 发起卸载请求
	InitiateUninstall(ip string, forceUninstall bool, reason string, timeoutSeconds int32, deleteFromDB bool) (*UninstallTask, error)
	// 处理Agent卸载状态上报
	ProcessUninstallReport(agentID string, status string, metrics map[string]string) error
	// 获取卸载任务状态
	GetUninstallTask(agentID string) (*UninstallTask, error)
	// 获取所有卸载任务
	GetUninstallTasks() ([]*UninstallTask, error)
	// 清理过期任务
	CleanupExpiredTasks() error
}

// uninstallService 卸载服务实现
type uninstallService struct {
	agentService AgentService
	tasks        map[string]*UninstallTask // AgentID -> UninstallTask
	tasksMutex   sync.RWMutex
	cleanupTicker *time.Ticker
}

// NewUninstallService 创建卸载服务实例
func NewUninstallService(agentService AgentService) UninstallService {
	service := &uninstallService{
		agentService: agentService,
		tasks:        make(map[string]*UninstallTask),
	}
	
	// 启动清理goroutine
	service.cleanupTicker = time.NewTicker(5 * time.Minute)
	go service.cleanupLoop()
	
	return service
}

// InitiateUninstall 发起卸载请求
func (s *uninstallService) InitiateUninstall(ip string, forceUninstall bool, reason string, timeoutSeconds int32, deleteFromDB bool) (*UninstallTask, error) {
	// 根据IP查找Agent
	agents, _, err := s.agentService.GetAgentList(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("获取Agent列表失败: %v", err)
	}

	var targetAgent *models.Agent
	for _, agent := range agents {
		if agent.IPAddress == ip {
			targetAgent = agent
			break
		}
	}

	if targetAgent == nil {
		return nil, fmt.Errorf("未找到IP地址为 %s 的Agent", ip)
	}

	// 检查Agent状态
	if targetAgent.Status != "online" && !forceUninstall {
		return nil, fmt.Errorf("Agent不在线，当前状态: %s", targetAgent.Status)
	}

	// 设置默认超时时间
	if timeoutSeconds == 0 {
		timeoutSeconds = 120 // 默认2分钟超时
	}

	// 创建卸载任务
	task := &UninstallTask{
		AgentID:        targetAgent.ID,
		IP:             targetAgent.IPAddress,
		ForceUninstall: forceUninstall,
		Reason:         reason,
		TimeoutSeconds: timeoutSeconds,
		DeleteFromDB:   deleteFromDB,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CleanedFiles:   []string{},
		CleanupTime:    0,
	}

	// 保存任务
	s.tasksMutex.Lock()
	s.tasks[targetAgent.ID] = task
	s.tasksMutex.Unlock()

	log.Printf("创建卸载任务: AgentID=%s, IP=%s, 强制卸载=%t, 超时=%d秒", 
		task.AgentID, task.IP, task.ForceUninstall, task.TimeoutSeconds)

	// 更新Agent状态为卸载中
	targetAgent.Status = "uninstalling"
	if err := s.agentService.UpdateAgent(targetAgent); err != nil {
		log.Printf("更新Agent状态失败: %v", err)
	}

	// 启动超时监控
	go s.monitorUninstallTimeout(task)

	return task, nil
}

// ProcessUninstallReport 处理Agent卸载状态上报
func (s *uninstallService) ProcessUninstallReport(agentID string, status string, metrics map[string]string) error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	task, exists := s.tasks[agentID]
	if !exists {
		return fmt.Errorf("未找到Agent %s 的卸载任务", agentID)
	}

	log.Printf("收到Agent卸载状态上报: AgentID=%s, Status=%s", agentID, status)

	// 更新任务状态
	task.Status = "in_progress"
	task.UpdatedAt = time.Now()

	// 解析metrics中的卸载信息
	if uninstallStatus, ok := metrics["uninstall_status"]; ok {
		task.Status = uninstallStatus
	}

	if uninstallSuccess, ok := metrics["uninstall_success"]; ok && uninstallSuccess == "true" {
		task.Status = "completed"
		
		// 解析清理信息
		if cleanupTimeMs, ok := metrics["cleanup_time_ms"]; ok {
			task.CleanupTime = parseInt64(cleanupTimeMs)
		}
		
		if cleanedFilesCount, ok := metrics["cleaned_files_count"]; ok {
			log.Printf("Agent %s 清理了 %s 个文件", agentID, cleanedFilesCount)
		}

		log.Printf("Agent %s 卸载完成，耗时: %d ms", agentID, task.CleanupTime)

		// 如果需要从数据库删除，执行删除操作
		if task.DeleteFromDB {
			if err := s.agentService.DeleteAgent(agentID); err != nil {
				log.Printf("从数据库删除Agent失败: %v", err)
				task.Error = fmt.Sprintf("删除Agent失败: %v", err)
			} else {
				log.Printf("Agent %s 已从数据库删除", agentID)
			}
		} else {
			// 更新Agent状态为离线
			agent := &models.Agent{
				ID:     agentID,
				Status: "offline",
			}
			if err := s.agentService.UpdateAgent(agent); err != nil {
				log.Printf("更新Agent状态为离线失败: %v", err)
			}
		}
	} else if uninstallSuccess, ok := metrics["uninstall_success"]; ok && uninstallSuccess == "false" {
		task.Status = "failed"
		if errorMsg, ok := metrics["uninstall_message"]; ok {
			task.Error = errorMsg
		}
		log.Printf("Agent %s 卸载失败: %s", agentID, task.Error)
	}

	return nil
}

// GetUninstallTask 获取卸载任务状态
func (s *uninstallService) GetUninstallTask(agentID string) (*UninstallTask, error) {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	task, exists := s.tasks[agentID]
	if !exists {
		return nil, fmt.Errorf("未找到Agent %s 的卸载任务", agentID)
	}

	return task, nil
}

// GetUninstallTasks 获取所有卸载任务
func (s *uninstallService) GetUninstallTasks() ([]*UninstallTask, error) {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	tasks := make([]*UninstallTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// monitorUninstallTimeout 监控卸载超时
func (s *uninstallService) monitorUninstallTimeout(task *UninstallTask) {
	timeout := time.Duration(task.TimeoutSeconds) * time.Second
	
	select {
	case <-time.After(timeout):
		s.tasksMutex.Lock()
		defer s.tasksMutex.Unlock()

		// 检查任务是否已完成
		if currentTask, exists := s.tasks[task.AgentID]; exists {
			if currentTask.Status != "completed" && currentTask.Status != "failed" {
				currentTask.Status = "timeout"
				currentTask.Error = fmt.Sprintf("卸载操作超时（%v）", timeout)
				currentTask.UpdatedAt = time.Now()
				
				log.Printf("Agent %s 卸载超时", task.AgentID)
				
				// 如果强制卸载，直接从数据库删除
				if task.ForceUninstall && task.DeleteFromDB {
					if err := s.agentService.DeleteAgent(task.AgentID); err != nil {
						log.Printf("强制删除超时Agent失败: %v", err)
					} else {
						log.Printf("强制删除超时Agent成功: %s", task.AgentID)
					}
				}
			}
		}
	}
}

// cleanupLoop 清理过期任务
func (s *uninstallService) cleanupLoop() {
	for range s.cleanupTicker.C {
		s.CleanupExpiredTasks()
	}
}

// CleanupExpiredTasks 清理过期任务
func (s *uninstallService) CleanupExpiredTasks() error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	now := time.Now()
	expiredThreshold := 24 * time.Hour // 24小时后清理完成的任务

	for agentID, task := range s.tasks {
		// 清理已完成超过24小时的任务
		if (task.Status == "completed" || task.Status == "failed" || task.Status == "timeout") &&
			now.Sub(task.UpdatedAt) > expiredThreshold {
			
			delete(s.tasks, agentID)
			log.Printf("清理过期卸载任务: AgentID=%s, Status=%s", agentID, task.Status)
		}
	}

	return nil
}

// parseInt64 解析int64
func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}