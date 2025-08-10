package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// FilterGinHandler Gin框架兼容的过滤器管理处理器
type FilterGinHandler struct {
	// TODO: 添加Agent客户端管理器，用于调用Agent的gRPC接口
}

// NewFilterGinHandler 创建Gin过滤器处理器实例
func NewFilterGinHandler() *FilterGinHandler {
	return &FilterGinHandler{}
}

// BlacklistGinRequest 黑名单请求结构（Gin版本）
type BlacklistGinRequest struct {
	AgentID   string   `json:"agent_id" binding:"required"`
	Protocol  string   `json:"protocol" binding:"required"`
	Domains   []string `json:"domains,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Operation string   `json:"operation" binding:"required"` // add, remove, replace, clear
}

// WhitelistGinRequest 白名单请求结构（Gin版本）
type WhitelistGinRequest struct {
	AgentID   string   `json:"agent_id" binding:"required"`
	Protocol  string   `json:"protocol" binding:"required"`
	Domains   []string `json:"domains,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Operation string   `json:"operation" binding:"required"` // add, remove, replace, clear
}

// RollbackGinRequest 回滚请求结构（Gin版本）
type RollbackGinRequest struct {
	AgentID       string `json:"agent_id" binding:"required"`
	TargetVersion string `json:"target_version,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// FilterGinResponse Gin通用响应结构
type FilterGinResponse struct {
	Success       bool        `json:"success"`
	Message       string      `json:"message"`
	ConfigVersion string      `json:"config_version,omitempty"`
	Data          interface{} `json:"data,omitempty"`
}

// UpdateBlacklist 更新黑名单（Gin版本）
func (h *FilterGinHandler) UpdateBlacklist(c *gin.Context) {
	var req BlacklistGinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	
	// 验证操作类型
	validOps := []string{"add", "remove", "replace", "clear"}
	if !h.isValidOperation(req.Operation, validOps) {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "无效的操作类型: " + req.Operation,
		})
		return
	}
	
	log.Printf("黑名单更新请求: AgentID=%s, Protocol=%s, Operation=%s, Domains=%v, IPs=%v, Ports=%v", 
		req.AgentID, req.Protocol, req.Operation, req.Domains, req.IPs, req.Ports)
	
	// TODO: 调用Agent的gRPC接口
	// 这里先返回模拟响应
	response := FilterGinResponse{
		Success:       true,
		Message:       fmt.Sprintf("成功更新%s协议的黑名单", req.Protocol),
		ConfigVersion: fmt.Sprintf("v%d", time.Now().Unix()),
		Data: map[string]interface{}{
			"agent_id":  req.AgentID,
			"protocol":  req.Protocol,
			"operation": req.Operation,
			"affected_items": map[string]interface{}{
				"domains": len(req.Domains),
				"ips":     len(req.IPs),
				"ports":   len(req.Ports),
			},
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// UpdateWhitelist 更新白名单（Gin版本）
func (h *FilterGinHandler) UpdateWhitelist(c *gin.Context) {
	var req WhitelistGinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	
	// 验证操作类型
	validOps := []string{"add", "remove", "replace", "clear"}
	if !h.isValidOperation(req.Operation, validOps) {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "无效的操作类型: " + req.Operation,
		})
		return
	}
	
	log.Printf("白名单更新请求: AgentID=%s, Protocol=%s, Operation=%s, Domains=%v, IPs=%v, Ports=%v", 
		req.AgentID, req.Protocol, req.Operation, req.Domains, req.IPs, req.Ports)
	
	// TODO: 调用Agent的gRPC接口
	response := FilterGinResponse{
		Success:       true,
		Message:       fmt.Sprintf("成功更新%s协议的白名单", req.Protocol),
		ConfigVersion: fmt.Sprintf("v%d", time.Now().Unix()),
		Data: map[string]interface{}{
			"agent_id":  req.AgentID,
			"protocol":  req.Protocol,
			"operation": req.Operation,
			"affected_items": map[string]interface{}{
				"domains": len(req.Domains),
				"ips":     len(req.IPs),
				"ports":   len(req.Ports),
			},
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// GetFilterConfig 获取过滤器配置（Gin版本）
func (h *FilterGinHandler) GetFilterConfig(c *gin.Context) {
	agentID := c.Param("agent_id")
	protocol := c.Query("protocol")
	
	if agentID == "" {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "agent_id不能为空",
		})
		return
	}
	
	log.Printf("过滤器配置查询请求: AgentID=%s, Protocol=%s", agentID, protocol)
	
	// TODO: 从Agent获取实际配置
	// 这里返回模拟数据
	mockFilters := []map[string]interface{}{
		{
			"protocol":          "http",
			"blacklist_domains": []string{"example.com", "blocked.site"},
			"blacklist_ips":     []string{"192.168.1.100"},
			"blacklist_ports":   []string{"8080"},
			"whitelist_domains": []string{"google.com", "github.com"},
			"whitelist_ips":     []string{"8.8.8.8", "1.1.1.1"},
			"whitelist_ports":   []string{"443", "80"},
			"enabled":           true,
			"last_updated":      time.Now().Format(time.RFC3339),
		},
		{
			"protocol":          "https",
			"blacklist_domains": []string{"malware.site"},
			"blacklist_ips":     []string{},
			"blacklist_ports":   []string{},
			"whitelist_domains": []string{"*.google.com", "*.github.com"},
			"whitelist_ips":     []string{},
			"whitelist_ports":   []string{"443"},
			"enabled":           true,
			"last_updated":      time.Now().Format(time.RFC3339),
		},
	}
	
	// 如果指定了协议，只返回该协议的配置
	if protocol != "" {
		for _, filter := range mockFilters {
			if filter["protocol"].(string) == protocol {
				mockFilters = []map[string]interface{}{filter}
				break
			}
		}
	}
	
	response := FilterGinResponse{
		Success: true,
		Message: "配置查询成功",
		Data: map[string]interface{}{
			"agent_id": agentID,
			"filters":  mockFilters,
			"total":    len(mockFilters),
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// RollbackConfig 回滚配置（Gin版本）
func (h *FilterGinHandler) RollbackConfig(c *gin.Context) {
	var req RollbackGinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}
	
	log.Printf("配置回滚请求: AgentID=%s, TargetVersion=%s, Reason=%s", 
		req.AgentID, req.TargetVersion, req.Reason)
	
	// TODO: 调用Agent的gRPC接口执行回滚
	targetVersion := req.TargetVersion
	if targetVersion == "" {
		targetVersion = fmt.Sprintf("v%d", time.Now().Unix()-3600) // 假设回滚到1小时前
	}
	
	response := FilterGinResponse{
		Success:       true,
		Message:       "配置回滚成功",
		ConfigVersion: targetVersion,
		Data: map[string]interface{}{
			"agent_id":             req.AgentID,
			"rolled_back_version":  targetVersion,
			"current_version":      fmt.Sprintf("v%d", time.Now().Unix()),
			"rollback_reason":      req.Reason,
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// GetAgentFilterStatus 获取Agent过滤器状态（Gin版本）
func (h *FilterGinHandler) GetAgentFilterStatus(c *gin.Context) {
	agentID := c.Param("agent_id")
	
	if agentID == "" {
		c.JSON(http.StatusBadRequest, FilterGinResponse{
			Success: false,
			Message: "agent_id不能为空",
		})
		return
	}
	
	log.Printf("Agent过滤器状态查询请求: AgentID=%s", agentID)
	
	// TODO: 从Agent获取实际状态
	mockStatus := map[string]interface{}{
		"agent_id":       agentID,
		"status":         "online",
		"filter_version": fmt.Sprintf("v%d", time.Now().Unix()),
		"protocols": []string{"http", "https", "socks5", "shadowsocks", "vmess", "trojan", "vless"},
		"statistics": map[string]interface{}{
			"total_rules":      15,
			"blacklist_rules":  8,
			"whitelist_rules":  7,
			"enabled_filters":  7,
			"disabled_filters": 0,
		},
		"last_updated":   time.Now().Format(time.RFC3339),
		"singbox_status": "running",
	}
	
	response := FilterGinResponse{
		Success: true,
		Message: "状态查询成功",
		Data:    mockStatus,
	}
	
	c.JSON(http.StatusOK, response)
}

// 辅助方法

// isValidOperation 验证操作类型是否有效
func (h *FilterGinHandler) isValidOperation(operation string, validOps []string) bool {
	for _, op := range validOps {
		if operation == op {
			return true
		}
	}
	return false
}

// SetupFilterRoutes 设置过滤器相关路由
func SetupFilterRoutes(r *gin.RouterGroup) {
	filterHandler := NewFilterGinHandler()
	
	// 过滤器管理路由组
	filter := r.Group("/filter")
	{
		// 黑名单管理
		filter.POST("/blacklist", filterHandler.UpdateBlacklist)
		
		// 白名单管理
		filter.POST("/whitelist", filterHandler.UpdateWhitelist)
		
		// 配置回滚
		filter.POST("/rollback", filterHandler.RollbackConfig)
		
		// 获取过滤器配置 (修改路由避免冲突)
		filter.GET("/config/:agent_id", filterHandler.GetFilterConfig)
		
		// 获取过滤器状态
		filter.GET("/status/:agent_id", filterHandler.GetAgentFilterStatus)
	}
	
	log.Println("过滤器管理路由已注册 (Gin版本)")
}