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
	agentService     service.AgentService
	uninstallService service.UninstallService
}

// NewAgentHandler 创建Agent处理器实例
func NewAgentHandler(agentService service.AgentService, uninstallService service.UninstallService) *AgentHandler {
	return &AgentHandler{
		agentService:     agentService,
		uninstallService: uninstallService,
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

// UninstallAgentRequest 卸载Agent请求
type UninstallAgentRequest struct {
	IP               string `json:"ip" binding:"required"`              // Agent节点IP地址
	ForceUninstall   bool   `json:"force_uninstall"`                   // 是否强制卸载
	Reason           string `json:"reason"`                            // 卸载原因
	TimeoutSeconds   int32  `json:"timeout_seconds"`                   // 超时时间（秒）
	DeleteFromDB     bool   `json:"delete_from_db"`                    // 是否从数据库删除Agent记录
}

// UninstallAgentResponse 卸载Agent响应
type UninstallAgentResponse struct {
	AgentID         string   `json:"agent_id"`         // Agent ID
	IP              string   `json:"ip"`               // Agent IP
	UninstallStatus string   `json:"uninstall_status"` // 卸载状态
	Success         bool     `json:"success"`          // 是否成功
	Message         string   `json:"message"`          // 响应消息
	CleanedFiles    []string `json:"cleaned_files"`    // 已清理的文件列表
	CleanupTime     int64    `json:"cleanup_time"`     // 清理耗时（毫秒）
}

// UninstallAgent 卸载Agent节点
// @Summary 卸载Agent节点
// @Description 根据IP地址卸载Agent节点，包括清理sing-box服务和配置
// @Tags agents
// @Accept json
// @Produce json
// @Param request body UninstallAgentRequest true "卸载参数"
// @Success 200 {object} Response{data=UninstallAgentResponse}
// @Router /api/v1/agents/uninstall [post]
func (h *AgentHandler) UninstallAgent(c *gin.Context) {
	var req UninstallAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "参数无效",
			Error:   err.Error(),
		})
		return
	}

	log.Printf("收到Agent卸载请求: IP=%s, 强制卸载=%t, 原因=%s", req.IP, req.ForceUninstall, req.Reason)

	// 使用卸载服务发起卸载
	task, err := h.uninstallService.InitiateUninstall(
		req.IP, 
		req.ForceUninstall, 
		req.Reason, 
		req.TimeoutSeconds, 
		req.DeleteFromDB,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "发起卸载失败",
			Error:   err.Error(),
		})
		return
	}

	// 构建响应
	response := UninstallAgentResponse{
		AgentID:         task.AgentID,
		IP:              task.IP,
		UninstallStatus: task.Status,
		Success:         true,
		Message:         "Agent卸载任务已创建，等待Agent执行卸载操作",
		CleanedFiles:    task.CleanedFiles,
		CleanupTime:     task.CleanupTime,
	}
	
	if task.DeleteFromDB {
		response.Message += "，完成后将从数据库删除Agent记录"
	}

	log.Printf("Agent卸载任务已创建: AgentID=%s, TaskID=%s", task.AgentID, task.AgentID)

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Agent卸载请求已发起",
		Data:    response,
	})
}

// GetIPRanges 获取所有节点的IP段信息
// @Summary 获取IP段信息
// @Description 获取当前Controller管理的所有节点的IP段信息，支持按国家/地区筛选
// @Tags agents
// @Accept json
// @Produce json
// @Param country query string false "国家筛选"
// @Param region query string false "地区筛选"
// @Param isp query string false "运营商筛选"
// @Success 200 {object} Response{data=IPRangeListResponse}
// @Router /api/v1/agents/ip-ranges [get]
func (h *AgentHandler) GetIPRanges(c *gin.Context) {
	// 获取筛选参数
	country := c.Query("country")
	region := c.Query("region")
	isp := c.Query("isp")
	
	// 获取所有Agent列表
	agents, _, err := h.agentService.GetAgentList(1000, 0) // 获取大量数据以显示所有节点
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取Agent列表失败",
			Error:   err.Error(),
		})
		return
	}
	
	// 处理和筛选IP段信息
	ipRanges := make([]IPRangeItem, 0)
	ipRangeMap := make(map[string]*IPRangeItem) // 用于去重相同IP段
	
	for _, agent := range agents {
		// 应用筛选条件
		if country != "" && agent.Country != country {
			continue
		}
		if region != "" && agent.Region != region {
			continue
		}
		if isp != "" && agent.ISP != isp {
			continue
		}
		
		// 只处理有IP段信息的Agent
		if agent.IPRange == "" {
			continue
		}
		
		// 去重处理：相同IP段的Agent归为一组
		if existingRange, exists := ipRangeMap[agent.IPRange]; exists {
			// 相同IP段，添加到Agent列表中
			existingRange.Agents = append(existingRange.Agents, IPRangeAgent{
				ID:       agent.ID,
				Hostname: agent.Hostname,
				IP:       agent.IPAddress,
				Status:   agent.Status,
			})
			existingRange.AgentCount++
		} else {
			// 新的IP段
			ipRangeItem := &IPRangeItem{
				IPRange:    agent.IPRange,
				Country:    agent.Country,
				Region:     agent.Region,
				City:       agent.City,
				ISP:        agent.ISP,
				AgentCount: 1,
				Agents: []IPRangeAgent{
					{
						ID:       agent.ID,
						Hostname: agent.Hostname,
						IP:       agent.IPAddress,
						Status:   agent.Status,
					},
				},
			}
			ipRangeMap[agent.IPRange] = ipRangeItem
		}
	}
	
	// 将map转换为切片
	for _, item := range ipRangeMap {
		ipRanges = append(ipRanges, *item)
	}
	
	// 统计信息
	stats := IPRangeStats{
		TotalRanges:    len(ipRanges),
		TotalAgents:    len(agents),
		CountryCount:   make(map[string]int),
		RegionCount:    make(map[string]int),
		ISPCount:       make(map[string]int),
	}
	
	// 统计各维度数据
	for _, item := range ipRanges {
		stats.CountryCount[item.Country]++
		stats.RegionCount[item.Region]++
		stats.ISPCount[item.ISP]++
	}
	
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data: IPRangeListResponse{
			IPRanges: ipRanges,
			Stats:    stats,
		},
	})
}

// IPRangeListResponse IP段列表响应
type IPRangeListResponse struct {
	IPRanges []IPRangeItem `json:"ip_ranges"`
	Stats    IPRangeStats  `json:"stats"`
}

// IPRangeItem IP段信息项
type IPRangeItem struct {
	IPRange    string           `json:"ip_range"`    // IP段，如 192.168.1.0/24
	Country    string           `json:"country"`     // 国家
	Region     string           `json:"region"`      // 地区/省份
	City       string           `json:"city"`        // 城市
	ISP        string           `json:"isp"`         // 运营商
	AgentCount int              `json:"agent_count"` // 该IP段下的Agent数量
	Agents     []IPRangeAgent   `json:"agents"`      // 该IP段下的Agent列表
}

// IPRangeAgent IP段下的Agent信息
type IPRangeAgent struct {
	ID       string `json:"id"`       // Agent ID
	Hostname string `json:"hostname"` // 主机名
	IP       string `json:"ip"`       // IP地址
	Status   string `json:"status"`   // 状态
}

// IPRangeStats IP段统计信息
type IPRangeStats struct {
	TotalRanges  int            `json:"total_ranges"`  // 总IP段数
	TotalAgents  int            `json:"total_agents"`  // 总Agent数
	CountryCount map[string]int `json:"country_count"` // 各国家的IP段数量
	RegionCount  map[string]int `json:"region_count"`  // 各地区的IP段数量
	ISPCount     map[string]int `json:"isp_count"`     // 各运营商的IP段数量
}