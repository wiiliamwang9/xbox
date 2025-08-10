package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// FilterHandler 过滤器管理处理器
type FilterHandler struct {
	// TODO: 添加Agent客户端管理器，用于调用Agent的gRPC接口
}

// NewFilterHandler 创建过滤器处理器实例
func NewFilterHandler() *FilterHandler {
	return &FilterHandler{}
}

// BlacklistRequest 黑名单请求结构
type BlacklistRequest struct {
	AgentID   string   `json:"agent_id"`
	Protocol  string   `json:"protocol"`
	Domains   []string `json:"domains,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Operation string   `json:"operation"` // add, remove, replace, clear
}

// WhitelistRequest 白名单请求结构
type WhitelistRequest struct {
	AgentID   string   `json:"agent_id"`
	Protocol  string   `json:"protocol"`
	Domains   []string `json:"domains,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Operation string   `json:"operation"` // add, remove, replace, clear
}

// RollbackRequest 回滚请求结构
type RollbackRequest struct {
	AgentID       string `json:"agent_id"`
	TargetVersion string `json:"target_version,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// FilterResponse 通用响应结构
type FilterResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ConfigVersion string `json:"config_version,omitempty"`
	Data          interface{} `json:"data,omitempty"`
}

// UpdateBlacklist 更新黑名单
func (h *FilterHandler) UpdateBlacklist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req BlacklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}
	
	// 验证必需参数
	if req.AgentID == "" {
		h.sendError(w, http.StatusBadRequest, "agent_id不能为空")
		return
	}
	if req.Protocol == "" {
		h.sendError(w, http.StatusBadRequest, "protocol不能为空")
		return
	}
	if req.Operation == "" {
		h.sendError(w, http.StatusBadRequest, "operation不能为空")
		return
	}
	
	// 验证操作类型
	validOps := []string{"add", "remove", "replace", "clear"}
	if !h.isValidOperation(req.Operation, validOps) {
		h.sendError(w, http.StatusBadRequest, "无效的操作类型: "+req.Operation)
		return
	}
	
	log.Printf("黑名单更新请求: AgentID=%s, Protocol=%s, Operation=%s", req.AgentID, req.Protocol, req.Operation)
	
	// TODO: 调用Agent的gRPC接口
	// 这里先返回模拟响应
	response := FilterResponse{
		Success:       true,
		Message:       "黑名单更新成功",
		ConfigVersion: fmt.Sprintf("v%d", time.Now().Unix()),
	}
	
	h.sendResponse(w, http.StatusOK, response)
}

// UpdateWhitelist 更新白名单
func (h *FilterHandler) UpdateWhitelist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req WhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}
	
	// 验证必需参数
	if req.AgentID == "" {
		h.sendError(w, http.StatusBadRequest, "agent_id不能为空")
		return
	}
	if req.Protocol == "" {
		h.sendError(w, http.StatusBadRequest, "protocol不能为空")
		return
	}
	if req.Operation == "" {
		h.sendError(w, http.StatusBadRequest, "operation不能为空")
		return
	}
	
	// 验证操作类型
	validOps := []string{"add", "remove", "replace", "clear"}
	if !h.isValidOperation(req.Operation, validOps) {
		h.sendError(w, http.StatusBadRequest, "无效的操作类型: "+req.Operation)
		return
	}
	
	log.Printf("白名单更新请求: AgentID=%s, Protocol=%s, Operation=%s", req.AgentID, req.Protocol, req.Operation)
	
	// TODO: 调用Agent的gRPC接口
	// 这里先返回模拟响应
	response := FilterResponse{
		Success:       true,
		Message:       "白名单更新成功",
		ConfigVersion: fmt.Sprintf("v%d", time.Now().Unix()),
	}
	
	h.sendResponse(w, http.StatusOK, response)
}

// GetFilterConfig 获取过滤器配置
func (h *FilterHandler) GetFilterConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	protocol := r.URL.Query().Get("protocol")
	
	if agentID == "" {
		h.sendError(w, http.StatusBadRequest, "agent_id不能为空")
		return
	}
	
	log.Printf("过滤器配置查询请求: AgentID=%s, Protocol=%s", agentID, protocol)
	
	// TODO: 从Agent获取实际配置
	// 这里返回模拟数据
	mockData := map[string]interface{}{
		"agent_id": agentID,
		"filters": []map[string]interface{}{
			{
				"protocol":          "http",
				"blacklist_domains": []string{"example.com", "blocked.site"},
				"blacklist_ips":     []string{"192.168.1.100"},
				"blacklist_ports":   []string{"8080"},
				"whitelist_domains": []string{"google.com", "github.com"},
				"whitelist_ips":     []string{"8.8.8.8", "1.1.1.1"},
				"whitelist_ports":   []string{"443", "80"},
				"enabled":           true,
				"last_updated":      "2025-08-10T09:20:00Z",
			},
		},
	}
	
	response := FilterResponse{
		Success: true,
		Message: "配置查询成功",
		Data:    mockData,
	}
	
	h.sendResponse(w, http.StatusOK, response)
}

// RollbackConfig 回滚配置
func (h *FilterHandler) RollbackConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}
	
	// 验证必需参数
	if req.AgentID == "" {
		h.sendError(w, http.StatusBadRequest, "agent_id不能为空")
		return
	}
	
	log.Printf("配置回滚请求: AgentID=%s, TargetVersion=%s, Reason=%s", 
		req.AgentID, req.TargetVersion, req.Reason)
	
	// TODO: 调用Agent的gRPC接口执行回滚
	response := FilterResponse{
		Success:       true,
		Message:       "配置回滚成功",
		ConfigVersion: req.TargetVersion,
		Data: map[string]interface{}{
			"rolled_back_version": req.TargetVersion,
			"current_version":     fmt.Sprintf("v%d", time.Now().Unix()),
		},
	}
	
	h.sendResponse(w, http.StatusOK, response)
}

// GetAgentStatus 获取Agent状态（包含过滤器信息）
func (h *FilterHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	
	if agentID == "" {
		h.sendError(w, http.StatusBadRequest, "agent_id不能为空")
		return
	}
	
	log.Printf("Agent状态查询请求: AgentID=%s", agentID)
	
	// TODO: 从Agent获取实际状态
	mockStatus := map[string]interface{}{
		"agent_id":       agentID,
		"status":         "online",
		"filter_version": fmt.Sprintf("v%d", time.Now().Unix()),
		"protocols": []string{"http", "https", "socks5", "shadowsocks", "vmess", "trojan", "vless"},
		"last_updated":   "2025-08-10T09:20:00Z",
		"singbox_status": "running",
	}
	
	response := FilterResponse{
		Success: true,
		Message: "状态查询成功",
		Data:    mockStatus,
	}
	
	h.sendResponse(w, http.StatusOK, response)
}

// 辅助方法

// isValidOperation 验证操作类型是否有效
func (h *FilterHandler) isValidOperation(operation string, validOps []string) bool {
	for _, op := range validOps {
		if operation == op {
			return true
		}
	}
	return false
}

// sendResponse 发送成功响应
func (h *FilterHandler) sendResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("响应编码失败: %v", err)
	}
}

// sendError 发送错误响应
func (h *FilterHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	response := FilterResponse{
		Success: false,
		Message: message,
	}
	h.sendResponse(w, statusCode, response)
}

// RegisterRoutes 注册路由
func (h *FilterHandler) RegisterRoutes(router *mux.Router) {
	// 过滤器管理路由
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// 黑名单管理
	api.HandleFunc("/agents/blacklist", h.UpdateBlacklist).Methods("POST")
	
	// 白名单管理
	api.HandleFunc("/agents/whitelist", h.UpdateWhitelist).Methods("POST")
	
	// 获取过滤器配置
	api.HandleFunc("/agents/{agent_id}/filter", h.GetFilterConfig).Methods("GET")
	
	// 配置回滚
	api.HandleFunc("/agents/rollback", h.RollbackConfig).Methods("POST")
	
	// 获取Agent状态
	api.HandleFunc("/agents/{agent_id}/status", h.GetAgentStatus).Methods("GET")
	
	log.Println("过滤器管理路由已注册")
}