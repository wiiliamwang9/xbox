package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	"github.com/xbox/sing-box-manager/internal/models"
)

// MultiplexHandler 多路复用配置处理器
type MultiplexHandler struct {
	multiplexService service.MultiplexService
}

// NewMultiplexHandler 创建多路复用配置处理器
func NewMultiplexHandler(multiplexService service.MultiplexService) *MultiplexHandler {
	return &MultiplexHandler{
		multiplexService: multiplexService,
	}
}

// MultiplexConfigRequest 多路复用配置请求
type MultiplexConfigRequest struct {
	AgentID        string                 `json:"agent_id" binding:"required"`
	Protocol       string                 `json:"protocol" binding:"required,oneof=vmess vless trojan shadowsocks"`
	Enabled        bool                   `json:"enabled"`
	MaxConnections int                    `json:"max_connections" binding:"min=1,max=32"`
	MinStreams     int                    `json:"min_streams" binding:"min=1,max=32"`
	Padding        bool                   `json:"padding"`
	BrutalConfig   map[string]interface{} `json:"brutal_config,omitempty"`
}

// MultiplexConfigResponse 多路复用配置响应
type MultiplexConfigResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	ConfigVersion string `json:"config_version,omitempty"`
}

// MultiplexStatusResponse 多路复用状态响应
type MultiplexStatusResponse struct {
	Success          bool                    `json:"success"`
	Message          string                  `json:"message"`
	MultiplexConfigs []models.MultiplexConfig `json:"multiplex_configs,omitempty"`
}

// UpdateMultiplexConfig 更新多路复用配置
func (h *MultiplexHandler) UpdateMultiplexConfig(c *gin.Context) {
	var req MultiplexConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效: " + err.Error(),
		})
		return
	}

	// 验证max_connections和max_streams不能同时配置的逻辑
	// 根据用户需求，我们只配置max_connections
	if req.MaxConnections <= 0 {
		req.MaxConnections = 4 // 默认值
	}

	// 调用服务层处理多路复用配置
	configVersion, err := h.multiplexService.UpdateMultiplexConfig(
		req.AgentID,
		req.Protocol,
		req.Enabled,
		req.MaxConnections,
		req.MinStreams,
		req.Padding,
		req.BrutalConfig,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, MultiplexConfigResponse{
			Success: false,
			Message: "更新多路复用配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MultiplexConfigResponse{
		Success:       true,
		Message:       "多路复用配置更新成功",
		ConfigVersion: configVersion,
	})
}

// GetMultiplexConfig 获取多路复用配置
func (h *MultiplexHandler) GetMultiplexConfig(c *gin.Context) {
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "agent_id 参数不能为空",
		})
		return
	}

	protocol := c.Query("protocol") // 可选参数

	configs, err := h.multiplexService.GetMultiplexConfig(agentID, protocol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MultiplexStatusResponse{
			Success: false,
			Message: "获取多路复用配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MultiplexStatusResponse{
		Success:          true,
		Message:          "获取多路复用配置成功",
		MultiplexConfigs: configs,
	})
}

// GetMultiplexStatus 获取多路复用状态统计
func (h *MultiplexHandler) GetMultiplexStatus(c *gin.Context) {
	agentID := c.Query("agent_id") // 可选参数，为空时返回所有Agent的统计

	var stats interface{}
	var err error

	if agentID != "" {
		// 获取特定Agent的多路复用状态
		stats, err = h.multiplexService.GetAgentMultiplexStats(agentID)
	} else {
		// 获取系统多路复用状态统计
		stats, err = h.multiplexService.GetSystemMultiplexStats()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取多路复用状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "获取多路复用状态成功",
		"data":    stats,
	})
}

// BatchUpdateMultiplexConfig 批量更新多路复用配置
func (h *MultiplexHandler) BatchUpdateMultiplexConfig(c *gin.Context) {
	type BatchRequest struct {
		Configs []MultiplexConfigRequest `json:"configs" binding:"required,min=1"`
	}

	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数无效: " + err.Error(),
		})
		return
	}

	results := make([]map[string]interface{}, len(req.Configs))
	successCount := 0

	for i, config := range req.Configs {
		// 验证并设置默认值
		if config.MaxConnections <= 0 {
			config.MaxConnections = 4
		}

		configVersion, err := h.multiplexService.UpdateMultiplexConfig(
			config.AgentID,
			config.Protocol,
			config.Enabled,
			config.MaxConnections,
			config.MinStreams,
			config.Padding,
			config.BrutalConfig,
		)

		results[i] = map[string]interface{}{
			"agent_id": config.AgentID,
			"protocol": config.Protocol,
			"success":  err == nil,
		}

		if err == nil {
			results[i]["message"] = "配置更新成功"
			results[i]["config_version"] = configVersion
			successCount++
		} else {
			results[i]["message"] = "配置更新失败: " + err.Error()
		}
	}

	status := http.StatusOK
	if successCount == 0 {
		status = http.StatusInternalServerError
	} else if successCount < len(req.Configs) {
		status = http.StatusPartialContent
	}

	c.JSON(status, gin.H{
		"success":       successCount > 0,
		"message":       "批量更新完成，成功: " + strconv.Itoa(successCount) + "/" + strconv.Itoa(len(req.Configs)),
		"total_count":   len(req.Configs),
		"success_count": successCount,
		"results":       results,
	})
}