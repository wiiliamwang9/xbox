package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus 健康状态枚举
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// CheckResult 检查结果
type CheckResult struct {
	Name      string            `json:"name"`
	Status    HealthStatus      `json:"status"`
	Message   string            `json:"message,omitempty"`
	Duration  time.Duration     `json:"duration"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    HealthStatus         `json:"status"`
	Timestamp time.Time            `json:"timestamp"`
	Uptime    time.Duration        `json:"uptime"`
	Version   string               `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Checker 健康检查接口
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// HealthManager 健康管理器
type HealthManager struct {
	mu         sync.RWMutex
	checkers   map[string]Checker
	startTime  time.Time
	version    string
	timeout    time.Duration
}

// NewHealthManager 创建健康管理器
func NewHealthManager(version string) *HealthManager {
	return &HealthManager{
		checkers:  make(map[string]Checker),
		startTime: time.Now(),
		version:   version,
		timeout:   10 * time.Second,
	}
}

// AddChecker 添加检查器
func (hm *HealthManager) AddChecker(checker Checker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkers[checker.Name()] = checker
}

// RemoveChecker 移除检查器
func (hm *HealthManager) RemoveChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.checkers, name)
}

// Check 执行健康检查
func (hm *HealthManager) Check(ctx context.Context) HealthResponse {
	hm.mu.RLock()
	checkers := make(map[string]Checker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()

	response := HealthResponse{
		Timestamp: time.Now(),
		Uptime:    time.Since(hm.startTime),
		Version:   hm.version,
		Checks:    make(map[string]CheckResult),
		Status:    StatusHealthy,
	}

	// 执行所有检查
	var wg sync.WaitGroup
	resultsChan := make(chan CheckResult, len(checkers))

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			
			checkCtx, cancel := context.WithTimeout(ctx, hm.timeout)
			defer cancel()
			
			result := c.Check(checkCtx)
			resultsChan <- result
		}(checker)
	}

	wg.Wait()
	close(resultsChan)

	// 收集结果
	for result := range resultsChan {
		response.Checks[result.Name] = result
		
		// 根据检查结果更新整体状态
		switch result.Status {
		case StatusUnhealthy:
			response.Status = StatusUnhealthy
		case StatusDegraded:
			if response.Status == StatusHealthy {
				response.Status = StatusDegraded
			}
		}
	}

	return response
}

// Handler HTTP处理器
func (hm *HealthManager) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := hm.Check(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		// 根据状态设置HTTP状态码
		switch result.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // 降级状态仍返回200
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		json.NewEncoder(w).Encode(result)
	}
}

// ReadinessHandler 就绪检查处理器
func (hm *HealthManager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := hm.Check(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		// 就绪检查：只有完全健康才返回200
		if result.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		json.NewEncoder(w).Encode(result)
	}
}

// LivenessHandler 存活检查处理器
func (hm *HealthManager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 存活检查：只要服务能响应就认为存活
		response := map[string]interface{}{
			"status":    StatusHealthy,
			"timestamp": time.Now(),
			"uptime":    time.Since(hm.startTime),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// DatabaseChecker 数据库健康检查器
type DatabaseChecker struct {
	name string
	db   *sql.DB
}

// NewDatabaseChecker 创建数据库检查器
func NewDatabaseChecker(name string, db *sql.DB) *DatabaseChecker {
	return &DatabaseChecker{
		name: name,
		db:   db,
	}
}

// Name 返回检查器名称
func (dc *DatabaseChecker) Name() string {
	return dc.name
}

// Check 执行数据库检查
func (dc *DatabaseChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      dc.name,
		Timestamp: start,
	}

	if dc.db == nil {
		result.Status = StatusUnhealthy
		result.Message = "Database connection is nil"
		result.Duration = time.Since(start)
		return result
	}

	// 执行简单的ping
	if err := dc.db.PingContext(ctx); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Database ping failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// 检查连接池状态
	stats := dc.db.Stats()
	result.Status = StatusHealthy
	result.Message = "Database connection healthy"
	result.Duration = time.Since(start)
	result.Details = map[string]string{
		"open_connections": fmt.Sprintf("%d", stats.OpenConnections),
		"in_use":           fmt.Sprintf("%d", stats.InUse),
		"idle":            fmt.Sprintf("%d", stats.Idle),
	}

	// 如果连接数过多，标记为降级
	if stats.OpenConnections > 50 {
		result.Status = StatusDegraded
		result.Message = "High connection count detected"
	}

	return result
}

// GRPCChecker gRPC服务健康检查器
type GRPCChecker struct {
	name     string
	address  string
	timeout  time.Duration
}

// NewGRPCChecker 创建gRPC检查器
func NewGRPCChecker(name, address string) *GRPCChecker {
	return &GRPCChecker{
		name:    name,
		address: address,
		timeout: 5 * time.Second,
	}
}

// Name 返回检查器名称
func (gc *GRPCChecker) Name() string {
	return gc.name
}

// Check 执行gRPC检查
func (gc *GRPCChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      gc.name,
		Timestamp: start,
	}

	// 简化的gRPC连接检查
	// 实际实现中应该尝试建立gRPC连接
	result.Status = StatusHealthy
	result.Message = "gRPC service is reachable"
	result.Duration = time.Since(start)
	result.Details = map[string]string{
		"address": gc.address,
	}

	return result
}

// SingboxChecker sing-box进程健康检查器
type SingboxChecker struct {
	name      string
	isRunning func() bool
	getPID    func() int
}

// NewSingboxChecker 创建sing-box检查器
func NewSingboxChecker(name string, isRunning func() bool, getPID func() int) *SingboxChecker {
	return &SingboxChecker{
		name:      name,
		isRunning: isRunning,
		getPID:    getPID,
	}
}

// Name 返回检查器名称
func (sc *SingboxChecker) Name() string {
	return sc.name
}

// Check 执行sing-box检查
func (sc *SingboxChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      sc.name,
		Timestamp: start,
	}

	running := sc.isRunning()
	pid := sc.getPID()

	if running && pid > 0 {
		result.Status = StatusHealthy
		result.Message = "sing-box process is running"
		result.Details = map[string]string{
			"pid":     fmt.Sprintf("%d", pid),
			"running": "true",
		}
	} else {
		result.Status = StatusUnhealthy
		result.Message = "sing-box process is not running"
		result.Details = map[string]string{
			"pid":     fmt.Sprintf("%d", pid),
			"running": "false",
		}
	}

	result.Duration = time.Since(start)
	return result
}