package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/xbox/sing-box-manager/api/handlers"
	"github.com/xbox/sing-box-manager/internal/controller/service"
)

// SetupRoutes 设置API路由
func SetupRoutes(r *gin.Engine, agentService service.AgentService, multiplexService service.MultiplexService, reportService *service.NodeReportService) {
	// 创建处理器
	agentHandler := handlers.NewAgentHandler(agentService, nil)
	multiplexHandler := handlers.NewMultiplexHandler(multiplexService)
	
	var reportHandler *handlers.ReportHandler
	if reportService != nil {
		reportHandler = handlers.NewReportHandler(reportService)
	}
	
	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// Agent管理路由
		agents := v1.Group("/agents")
		{
			agents.GET("", agentHandler.GetAgents)              // 获取Agent列表
			agents.GET("/stats", agentHandler.GetAgentStats)    // 获取Agent统计
			agents.GET("/ip-ranges", agentHandler.GetIPRanges)  // 获取IP段信息
			agents.GET("/:id", agentHandler.GetAgent)           // 获取单个Agent
			agents.PUT("/:id", agentHandler.UpdateAgent)        // 更新Agent
			agents.DELETE("/:id", agentHandler.DeleteAgent)     // 删除Agent
			agents.POST("/deploy", agentHandler.DeployAgent)    // 部署Agent
			agents.POST("/uninstall", agentHandler.UninstallAgent) // 卸载Agent
		}
		
		// 过滤器管理路由（黑名单/白名单）
		handlers.SetupFilterRoutes(v1)
		
		// 多路复用配置路由
		multiplex := v1.Group("/multiplex")
		{
			multiplex.POST("/config", multiplexHandler.UpdateMultiplexConfig)           // 更新多路复用配置
			multiplex.GET("/config/:agent_id", multiplexHandler.GetMultiplexConfig)     // 获取多路复用配置
			multiplex.GET("/status", multiplexHandler.GetMultiplexStatus)               // 获取多路复用状态统计
			multiplex.POST("/batch", multiplexHandler.BatchUpdateMultiplexConfig)       // 批量更新多路复用配置
		}
		
		// 节点上报路由
		if reportHandler != nil {
			report := v1.Group("/report")
			{
				report.POST("/manual", reportHandler.ManualReport)     // 手动触发上报
				report.GET("/stats", reportHandler.GetReportStats)     // 获取上报统计信息
				report.GET("/status", reportHandler.GetReportStatus)   // 获取上报服务状态
			}
		}
		
		// TODO: 配置管理路由
		// configs := v1.Group("/configs")
		// {
		//     configs.POST("", configHandler.CreateConfig)
		//     configs.GET("/:agent_id", configHandler.GetConfigs)
		//     configs.PUT("/:id", configHandler.UpdateConfig)
		//     configs.DELETE("/:id", configHandler.DeleteConfig)
		// }
		
		// TODO: 规则管理路由
		// rules := v1.Group("/rules")
		// {
		//     rules.POST("", ruleHandler.CreateRule)
		//     rules.GET("", ruleHandler.GetRules)
		//     rules.PUT("/:id", ruleHandler.UpdateRule)
		//     rules.DELETE("/:id", ruleHandler.DeleteRule)
		// }
		
		// TODO: 监控数据路由
		// monitoring := v1.Group("/monitoring")
		// {
		//     monitoring.GET("/:agent_id", monitoringHandler.GetMetrics)
		//     monitoring.GET("/metrics", monitoringHandler.GetSystemMetrics)
		// }
	}
	
	// 健康检查路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "xbox-controller",
		})
	})
	
	// 根路径
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Xbox Sing-box管理系统 API",
			"version": "v1.0.0",
		})
	})
}