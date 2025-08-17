package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xbox/sing-box-manager/internal/controller/service"
)

// ReportHandler 报告处理器
type ReportHandler struct {
	reportService *service.NodeReportService
}

// NewReportHandler 创建报告处理器
func NewReportHandler(reportService *service.NodeReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// ManualReport 手动触发节点信息上报
// @Summary 手动触发节点信息上报
// @Description 立即执行一次节点信息上报到后端服务
// @Tags report
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/report/manual [post]
func (h *ReportHandler) ManualReport(c *gin.Context) {
	if h.reportService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "节点上报服务未启用",
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.reportService.ReportOnce(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "手动上报失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "手动上报成功",
		Data:    "节点信息已成功上报到后端服务",
	})
}

// GetReportStats 获取上报统计信息
// @Summary 获取上报统计信息
// @Description 获取当前节点的统计信息
// @Tags report
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=service.ReportStats}
// @Router /api/v1/report/stats [get]
func (h *ReportHandler) GetReportStats(c *gin.Context) {
	if h.reportService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "节点上报服务未启用",
		})
		return
	}

	ctx := c.Request.Context()
	stats, err := h.reportService.GetReportStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    stats,
	})
}

// GetReportStatus 获取上报服务状态
// @Summary 获取上报服务状态
// @Description 获取节点上报服务的运行状态
// @Tags report
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=map[string]interface{}}
// @Router /api/v1/report/status [get]
func (h *ReportHandler) GetReportStatus(c *gin.Context) {
	status := map[string]interface{}{
		"enabled": h.reportService != nil,
		"service": "node-report-service",
	}

	if h.reportService != nil {
		status["status"] = "running"
		status["message"] = "节点上报服务正在运行"
		
		// 获取统计信息
		ctx := context.Background()
		if stats, err := h.reportService.GetReportStats(ctx); err == nil {
			status["stats"] = stats
		}
	} else {
		status["status"] = "disabled"
		status["message"] = "节点上报服务未启用"
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    status,
	})
}