package recovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// RecoveryAction 恢复动作类型
type RecoveryAction string

const (
	ActionRestart   RecoveryAction = "restart"
	ActionReconnect RecoveryAction = "reconnect"
	ActionReset     RecoveryAction = "reset"
	ActionAlert     RecoveryAction = "alert"
	ActionIgnore    RecoveryAction = "ignore"
)

// FailureType 故障类型
type FailureType string

const (
	FailureConnection  FailureType = "connection" 
	FailureProcess     FailureType = "process"
	FailureService     FailureType = "service"
	FailureResource    FailureType = "resource"
	FailureTimeout     FailureType = "timeout"
)

// RecoveryRule 恢复规则
type RecoveryRule struct {
	Name           string         `json:"name"`
	FailureType    FailureType    `json:"failure_type"`
	Component      string         `json:"component"`
	MaxRetries     int            `json:"max_retries"`
	RetryInterval  time.Duration  `json:"retry_interval"`
	Action         RecoveryAction `json:"action"`
	Enabled        bool           `json:"enabled"`
	Cooldown       time.Duration  `json:"cooldown"`
}

// RecoveryAttempt 恢复尝试记录
type RecoveryAttempt struct {
	Timestamp   time.Time      `json:"timestamp"`
	Component   string         `json:"component"`
	FailureType FailureType    `json:"failure_type"`
	Action      RecoveryAction `json:"action"`
	Success     bool           `json:"success"`
	Error       error          `json:"error,omitempty"`
	Duration    time.Duration  `json:"duration"`
}

// ComponentHandler 组件处理器接口
type ComponentHandler interface {
	Name() string
	IsHealthy() bool
	Restart() error
	Reset() error
	GetStatus() map[string]string
}

// RecoveryManager 恢复管理器
type RecoveryManager struct {
	mu              sync.RWMutex
	rules           map[string]*RecoveryRule
	handlers        map[string]ComponentHandler
	attempts        []RecoveryAttempt
	lastAttempts    map[string]time.Time
	ctx             context.Context
	cancel          context.CancelFunc
	running         bool
	checkInterval   time.Duration
	maxAttempts     int
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager(checkInterval time.Duration) *RecoveryManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RecoveryManager{
		rules:         make(map[string]*RecoveryRule),
		handlers:      make(map[string]ComponentHandler),
		attempts:      make([]RecoveryAttempt, 0, 1000),
		lastAttempts:  make(map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
		checkInterval: checkInterval,
		maxAttempts:   1000,
	}
}

// AddRule 添加恢复规则
func (rm *RecoveryManager) AddRule(rule *RecoveryRule) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.rules[rule.Name] = rule
}

// RemoveRule 移除恢复规则
func (rm *RecoveryManager) RemoveRule(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rules, name)
}

// AddHandler 添加组件处理器
func (rm *RecoveryManager) AddHandler(handler ComponentHandler) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.handlers[handler.Name()] = handler
}

// RemoveHandler 移除组件处理器
func (rm *RecoveryManager) RemoveHandler(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.handlers, name)
}

// Start 启动恢复管理器
func (rm *RecoveryManager) Start() {
	rm.mu.Lock()
	if rm.running {
		rm.mu.Unlock()
		return
	}
	rm.running = true
	rm.mu.Unlock()
	
	go rm.monitorLoop()
	log.Println("Recovery manager started")
}

// Stop 停止恢复管理器
func (rm *RecoveryManager) Stop() {
	rm.mu.Lock()
	if !rm.running {
		rm.mu.Unlock()
		return
	}
	rm.running = false
	rm.mu.Unlock()
	
	rm.cancel()
	log.Println("Recovery manager stopped")
}

// monitorLoop 监控循环
func (rm *RecoveryManager) monitorLoop() {
	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rm.checkComponents()
		case <-rm.ctx.Done():
			return
		}
	}
}

// checkComponents 检查所有组件
func (rm *RecoveryManager) checkComponents() {
	rm.mu.RLock()
	handlers := make(map[string]ComponentHandler)
	for name, handler := range rm.handlers {
		handlers[name] = handler
	}
	rm.mu.RUnlock()
	
	for name, handler := range handlers {
		if !handler.IsHealthy() {
			log.Printf("Component %s is unhealthy, attempting recovery", name)
			rm.attemptRecovery(name, FailureService)
		}
	}
}

// attemptRecovery 尝试恢复
func (rm *RecoveryManager) attemptRecovery(component string, failureType FailureType) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// 查找匹配的规则
	var applicableRule *RecoveryRule
	for _, rule := range rm.rules {
		if rule.Enabled && (rule.Component == component || rule.Component == "*") && 
		   rule.FailureType == failureType {
			applicableRule = rule
			break
		}
	}
	
	if applicableRule == nil {
		log.Printf("No recovery rule found for component %s, failure type %s", component, failureType)
		return
	}
	
	// 检查冷却期
	ruleKey := fmt.Sprintf("%s-%s-%s", applicableRule.Name, component, failureType)
	if lastAttempt, exists := rm.lastAttempts[ruleKey]; exists {
		if time.Since(lastAttempt) < applicableRule.Cooldown {
			log.Printf("Recovery for %s is in cooldown period", ruleKey)
			return
		}
	}
	
	// 执行恢复动作
	attempt := RecoveryAttempt{
		Timestamp:   time.Now(),
		Component:   component,
		FailureType: failureType,
		Action:      applicableRule.Action,
	}
	
	start := time.Now()
	success, err := rm.executeRecoveryAction(component, applicableRule.Action)
	attempt.Duration = time.Since(start)
	attempt.Success = success
	attempt.Error = err
	
	// 记录尝试
	rm.attempts = append(rm.attempts, attempt)
	if len(rm.attempts) > rm.maxAttempts {
		rm.attempts = rm.attempts[1:]
	}
	
	rm.lastAttempts[ruleKey] = attempt.Timestamp
	
	if success {
		log.Printf("Recovery successful for component %s using action %s", component, applicableRule.Action)
	} else {
		log.Printf("Recovery failed for component %s using action %s: %v", component, applicableRule.Action, err)
	}
}

// executeRecoveryAction 执行恢复动作
func (rm *RecoveryManager) executeRecoveryAction(component string, action RecoveryAction) (bool, error) {
	handler, exists := rm.handlers[component]
	if !exists {
		return false, fmt.Errorf("no handler found for component %s", component)
	}
	
	switch action {
	case ActionRestart:
		err := handler.Restart()
		return err == nil, err
		
	case ActionReset:
		err := handler.Reset()
		return err == nil, err
		
	case ActionReconnect:
		// 对于重连，我们尝试重启（简化处理）
		err := handler.Restart()
		return err == nil, err
		
	case ActionAlert:
		// 发送告警（这里简化为日志）
		log.Printf("ALERT: Component %s requires attention", component)
		return true, nil
		
	case ActionIgnore:
		// 忽略故障
		log.Printf("Ignoring failure for component %s", component)
		return true, nil
		
	default:
		return false, fmt.Errorf("unknown recovery action: %s", action)
	}
}

// TriggerRecovery 手动触发恢复
func (rm *RecoveryManager) TriggerRecovery(component string, failureType FailureType) error {
	rm.attemptRecovery(component, failureType)
	return nil
}

// GetAttempts 获取恢复尝试历史
func (rm *RecoveryManager) GetAttempts() []RecoveryAttempt {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// 返回副本
	attempts := make([]RecoveryAttempt, len(rm.attempts))
	copy(attempts, rm.attempts)
	return attempts
}

// GetStats 获取统计信息
func (rm *RecoveryManager) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_attempts":     len(rm.attempts),
		"active_rules":       len(rm.rules),
		"registered_handlers": len(rm.handlers),
		"running":           rm.running,
	}
	
	// 按组件统计
	componentStats := make(map[string]int)
	successfulAttempts := 0
	
	for _, attempt := range rm.attempts {
		componentStats[attempt.Component]++
		if attempt.Success {
			successfulAttempts++
		}
	}
	
	stats["success_rate"] = float64(successfulAttempts) / float64(len(rm.attempts))
	stats["component_stats"] = componentStats
	
	return stats
}

// GetRules 获取所有规则
func (rm *RecoveryManager) GetRules() map[string]*RecoveryRule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	rules := make(map[string]*RecoveryRule)
	for name, rule := range rm.rules {
		// 创建副本
		ruleCopy := *rule
		rules[name] = &ruleCopy
	}
	return rules
}

// SetDefaultRules 设置默认恢复规则
func (rm *RecoveryManager) SetDefaultRules() {
	// sing-box进程恢复规则
	rm.AddRule(&RecoveryRule{
		Name:          "singbox-process-restart",
		FailureType:   FailureProcess,
		Component:     "singbox",
		MaxRetries:    3,
		RetryInterval: 30 * time.Second,
		Action:        ActionRestart,
		Enabled:       true,
		Cooldown:      2 * time.Minute,
	})
	
	// gRPC连接恢复规则
	rm.AddRule(&RecoveryRule{
		Name:          "grpc-connection-reconnect",
		FailureType:   FailureConnection,
		Component:     "grpc-client",
		MaxRetries:    5,
		RetryInterval: 10 * time.Second,
		Action:        ActionReconnect,
		Enabled:       true,
		Cooldown:      1 * time.Minute,
	})
	
	// 服务超时恢复规则
	rm.AddRule(&RecoveryRule{
		Name:          "service-timeout-restart",
		FailureType:   FailureTimeout,
		Component:     "*",
		MaxRetries:    2,
		RetryInterval: 60 * time.Second,
		Action:        ActionRestart,
		Enabled:       true,
		Cooldown:      5 * time.Minute,
	})
	
	// 资源不足告警规则
	rm.AddRule(&RecoveryRule{
		Name:          "resource-exhaustion-alert",
		FailureType:   FailureResource,
		Component:     "*",
		MaxRetries:    1,
		RetryInterval: 5 * time.Minute,
		Action:        ActionAlert,
		Enabled:       true,
		Cooldown:      10 * time.Minute,
	})
}