package handlers

import (
	"net/http"
	"strconv"
	"os/exec"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xbox/sing-box-manager/internal/controller/service"
	"github.com/xbox/sing-box-manager/internal/models"
)

// AgentHandler Agent API处理器
type AgentHandler struct {
	agentService service.AgentService
}

// NewAgentHandler 创建Agent处理器实例
func NewAgentHandler(agentService service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// GetAgents 获取Agent列表
// @Summary 获取Agent列表
// @Description 获取所有Agent节点列表，支持分页
// @Tags agents
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页数量，默认10"
// @Param status query string false "状态筛选: online, offline, error"
// @Success 200 {object} Response{data=AgentListResponse}
// @Router /api/v1/agents [get]
func (h *AgentHandler) GetAgents(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	
	offset := (page - 1) * limit
	
	var agents []*models.Agent
	var total int64
	var err error
	
	// 根据状态筛选
	if status != "" {
		agents, total, err = h.agentService.GetAgentList(limit, offset)
		// TODO: 实现按状态筛选
	} else {
		agents, total, err = h.agentService.GetAgentList(limit, offset)
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取Agent列表失败",
			Error:   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: AgentListResponse{
			Items: agents,
			Total: total,
			Page:  page,
			Limit: limit,
		},
	})
}

// GetAgent 获取单个Agent详情
// @Summary 获取Agent详情
// @Description 根据Agent ID获取详细信息
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} Response{data=models.Agent}
// @Router /api/v1/agents/{id} [get]
func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Agent ID不能为空",
		})
		return
	}
	
	statusResp, err := h.agentService.GetAgentStatus(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Agent不存在",
			Error:   err.Error(),
		})
		return
	}
	
	if !statusResp.Success {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Agent不存在",
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    statusResp,
	})
}

// DeleteAgent 删除Agent
// @Summary 删除Agent
// @Description 删除指定的Agent节点
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/{id} [delete]
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Agent ID不能为空",
		})
		return
	}
	
	if err := h.agentService.DeleteAgent(agentID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "删除Agent失败",
			Error:   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Agent删除成功",
	})
}

// UpdateAgent 更新Agent信息
// @Summary 更新Agent信息
// @Description 更新Agent节点信息
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param agent body UpdateAgentRequest true "Agent更新信息"
// @Success 200 {object} Response
// @Router /api/v1/agents/{id} [put]
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Agent ID不能为空",
		})
		return
	}
	
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "请求参数错误",
			Error:   err.Error(),
		})
		return
	}
	
	// TODO: 实现Agent更新逻辑
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Agent更新功能待实现",
	})
}

// GetAgentStats 获取Agent统计信息
// @Summary 获取Agent统计信息
// @Description 获取Agent节点的统计数据
// @Tags agents
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=AgentStatsResponse}
// @Router /api/v1/agents/stats [get]
func (h *AgentHandler) GetAgentStats(c *gin.Context) {
	onlineCount, err := h.agentService.GetOnlineAgentCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取统计信息失败",
			Error:   err.Error(),
		})
		return
	}
	
	// TODO: 获取更多统计信息
	stats := AgentStatsResponse{
		OnlineCount:  int(onlineCount),
		OfflineCount: 0, // TODO: 实现
		TotalCount:   0, // TODO: 实现
		ErrorCount:   0, // TODO: 实现
	}
	
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    stats,
	})
}

// Response API响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AgentListResponse Agent列表响应
type AgentListResponse struct {
	Items []*models.Agent `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// UpdateAgentRequest Agent更新请求
type UpdateAgentRequest struct {
	Hostname string                 `json:"hostname"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AgentStatsResponse Agent统计响应
type AgentStatsResponse struct {
	OnlineCount  int `json:"online_count"`
	OfflineCount int `json:"offline_count"`
	TotalCount   int `json:"total_count"`
	ErrorCount   int `json:"error_count"`
}

// DeployAgentRequest 部署Agent请求
type DeployAgentRequest struct {
	NodeIp      string `json:"node_ip" binding:"required"`
	SshPort     int    `json:"ssh_port" binding:"required"`
	SshUser     string `json:"ssh_user" binding:"required"`
	SshPassword string `json:"ssh_password" binding:"required"`
}

// DeployAgent 部署Agent到远程节点
// @Summary 部署Agent
// @Description 在指定的远程节点上部署Agent
// @Tags agents
// @Accept json
// @Produce json
// @Param request body DeployAgentRequest true "部署参数"
// @Success 200 {object} Response{data=string}
// @Router /api/v1/agents/deploy [post]
func (h *AgentHandler) DeployAgent(c *gin.Context) {
	var req DeployAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数无效",
			Error:   err.Error(),
		})
		return
	}

	// 调用部署脚本
	scriptPath := "/root/wl/code/xbox/scripts/deploy_agent.sh"
	cmd := exec.Command("bash", scriptPath, req.NodeIp, strconv.Itoa(req.SshPort), req.SshPassword)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Agent部署失败",
			Error:   fmt.Sprintf("执行部署脚本失败: %v, 输出: %s", err, output),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Agent部署成功",
		Data:    string(output),
	})
}