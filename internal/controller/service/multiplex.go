package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xbox/sing-box-manager/internal/models"
	"gorm.io/gorm"
)

// MultiplexService 多路复用配置服务接口
type MultiplexService interface {
	UpdateMultiplexConfig(agentID, protocol string, enabled bool, maxConnections, minStreams int, padding bool, brutalConfig map[string]interface{}) (string, error)
	GetMultiplexConfig(agentID, protocol string) ([]models.MultiplexConfig, error)
	GetAgentMultiplexStats(agentID string) (interface{}, error)
	GetSystemMultiplexStats() (interface{}, error)
	DeleteMultiplexConfig(agentID, protocol string) error
}

// multiplexService 多路复用配置服务实现
type multiplexService struct {
	db          *gorm.DB
	agentClient AgentClient
}

// NewMultiplexService 创建多路复用配置服务
func NewMultiplexService(db *gorm.DB, agentClient AgentClient) MultiplexService {
	return &multiplexService{
		db:          db,
		agentClient: agentClient,
	}
}

// UpdateMultiplexConfig 更新多路复用配置
func (s *multiplexService) UpdateMultiplexConfig(agentID, protocol string, enabled bool, maxConnections, minStreams int, padding bool, brutalConfig map[string]interface{}) (string, error) {
	// 验证Agent是否存在
	var agent models.Agent
	if err := s.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("Agent %s 不存在", agentID)
		}
		return "", fmt.Errorf("查询Agent失败: %w", err)
	}

	// 验证协议类型
	supportedProtocols := map[string]bool{
		"vmess":       true,
		"vless":       true,
		"trojan":      true,
		"shadowsocks": true,
	}
	if !supportedProtocols[protocol] {
		return "", fmt.Errorf("不支持的协议类型: %s", protocol)
	}

	// 验证参数
	if maxConnections < 1 || maxConnections > 32 {
		return "", fmt.Errorf("max_connections 必须在 1-32 之间")
	}
	if minStreams < 1 || minStreams > 32 {
		return "", fmt.Errorf("min_streams 必须在 1-32 之间")
	}

	// 转换brutal配置为JSON
	var brutalJSON models.JSON
	if brutalConfig != nil && len(brutalConfig) > 0 {
		brutalJSON = models.JSON(brutalConfig)
	}

	// 生成配置版本
	configVersion := fmt.Sprintf("v%d", time.Now().Unix())

	// 更新或创建多路复用配置
	var config models.MultiplexConfig
	err := s.db.Where("agent_id = ? AND protocol = ?", agentID, protocol).First(&config).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.MultiplexConfig{
			AgentID:        agentID,
			Protocol:       protocol,
			Enabled:        enabled,
			MultiplexProto: "smux", // 固定为smux
			MaxConnections: maxConnections,
			MinStreams:     minStreams,
			Padding:        padding,
			BrutalConfig:   brutalJSON,
			Status:         "inactive",
			ConfigVersion:  configVersion,
		}
		if err := s.db.Create(&config).Error; err != nil {
			return "", fmt.Errorf("创建多路复用配置失败: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("查询多路复用配置失败: %w", err)
	} else {
		// 更新现有配置
		updates := map[string]interface{}{
			"enabled":         enabled,
			"max_connections": maxConnections,
			"min_streams":     minStreams,
			"padding":         padding,
			"brutal_config":   brutalJSON,
			"config_version":  configVersion,
			"updated_at":      time.Now(),
		}
		if err := s.db.Model(&config).Updates(updates).Error; err != nil {
			return "", fmt.Errorf("更新多路复用配置失败: %w", err)
		}
		config.Enabled = enabled
		config.MaxConnections = maxConnections
		config.MinStreams = minStreams
		config.Padding = padding
		config.BrutalConfig = brutalJSON
		config.ConfigVersion = configVersion
	}

	// 如果Agent在线，通过gRPC推送配置到Agent
	if agent.Status == "online" {
		go func() {
			if err := s.pushMultiplexConfigToAgent(agentID, protocol, config); err != nil {
				// 更新配置状态为error
				s.db.Model(&config).Updates(map[string]interface{}{
					"status":        "error",
					"error_message": err.Error(),
				})
			} else {
				// 更新配置状态为active
				s.db.Model(&config).Updates(map[string]interface{}{
					"status":        "active",
					"error_message": "",
				})
			}
		}()
	}

	return configVersion, nil
}

// GetMultiplexConfig 获取多路复用配置
func (s *multiplexService) GetMultiplexConfig(agentID, protocol string) ([]models.MultiplexConfig, error) {
	var configs []models.MultiplexConfig
	query := s.db.Where("agent_id = ?", agentID)
	
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	
	if err := query.Preload("Agent").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("查询多路复用配置失败: %w", err)
	}
	
	return configs, nil
}

// GetAgentMultiplexStats 获取Agent多路复用统计
func (s *multiplexService) GetAgentMultiplexStats(agentID string) (interface{}, error) {
	type ProtocolStats struct {
		Protocol       string `json:"protocol"`
		Enabled        bool   `json:"enabled"`
		MaxConnections int    `json:"max_connections"`
		MinStreams     int    `json:"min_streams"`
		Status         string `json:"status"`
		LastUpdated    string `json:"last_updated"`
	}

	type AgentStats struct {
		AgentID        string          `json:"agent_id"`
		Hostname       string          `json:"hostname"`
		AgentStatus    string          `json:"agent_status"`
		TotalProtocols int             `json:"total_protocols"`
		EnabledCount   int             `json:"enabled_count"`
		ActiveCount    int             `json:"active_count"`
		Protocols      []ProtocolStats `json:"protocols"`
	}

	var agent models.Agent
	if err := s.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("Agent不存在: %w", err)
	}

	var configs []models.MultiplexConfig
	if err := s.db.Where("agent_id = ?", agentID).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("查询多路复用配置失败: %w", err)
	}

	stats := AgentStats{
		AgentID:        agent.ID,
		Hostname:       agent.Hostname,
		AgentStatus:    agent.Status,
		TotalProtocols: len(configs),
		EnabledCount:   0,
		ActiveCount:    0,
		Protocols:      make([]ProtocolStats, len(configs)),
	}

	for i, config := range configs {
		if config.Enabled {
			stats.EnabledCount++
		}
		if config.Status == "active" {
			stats.ActiveCount++
		}

		stats.Protocols[i] = ProtocolStats{
			Protocol:       config.Protocol,
			Enabled:        config.Enabled,
			MaxConnections: config.MaxConnections,
			MinStreams:     config.MinStreams,
			Status:         config.Status,
			LastUpdated:    config.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return stats, nil
}

// GetSystemMultiplexStats 获取系统多路复用统计
func (s *multiplexService) GetSystemMultiplexStats() (interface{}, error) {
	type SystemStats struct {
		TotalAgents    int64 `json:"total_agents"`
		OnlineAgents   int64 `json:"online_agents"`
		TotalConfigs   int64 `json:"total_configs"`
		EnabledConfigs int64 `json:"enabled_configs"`
		ActiveConfigs  int64 `json:"active_configs"`
		ProtocolStats  map[string]struct {
			Total   int `json:"total"`
			Enabled int `json:"enabled"`
			Active  int `json:"active"`
		} `json:"protocol_stats"`
	}

	var stats SystemStats
	stats.ProtocolStats = make(map[string]struct {
		Total   int `json:"total"`
		Enabled int `json:"enabled"`
		Active  int `json:"active"`
	})

	// 统计Agent数量
	s.db.Model(&models.Agent{}).Count(&stats.TotalAgents)
	s.db.Model(&models.Agent{}).Where("status = ?", "online").Count(&stats.OnlineAgents)

	// 统计配置数量
	s.db.Model(&models.MultiplexConfig{}).Count(&stats.TotalConfigs)
	s.db.Model(&models.MultiplexConfig{}).Where("enabled = ?", true).Count(&stats.EnabledConfigs)
	s.db.Model(&models.MultiplexConfig{}).Where("status = ?", "active").Count(&stats.ActiveConfigs)

	// 按协议统计
	type ProtocolCount struct {
		Protocol string
		Total    int
		Enabled  int
		Active   int
	}

	var protocolCounts []ProtocolCount
	s.db.Raw(`
		SELECT 
			protocol,
			COUNT(*) as total,
			SUM(CASE WHEN enabled = 1 THEN 1 ELSE 0 END) as enabled,
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active
		FROM multiplex_configs 
		GROUP BY protocol
	`).Scan(&protocolCounts)

	for _, pc := range protocolCounts {
		stats.ProtocolStats[pc.Protocol] = struct {
			Total   int `json:"total"`
			Enabled int `json:"enabled"`
			Active  int `json:"active"`
		}{
			Total:   pc.Total,
			Enabled: pc.Enabled,
			Active:  pc.Active,
		}
	}

	return stats, nil
}

// DeleteMultiplexConfig 删除多路复用配置
func (s *multiplexService) DeleteMultiplexConfig(agentID, protocol string) error {
	result := s.db.Where("agent_id = ? AND protocol = ?", agentID, protocol).Delete(&models.MultiplexConfig{})
	if result.Error != nil {
		return fmt.Errorf("删除多路复用配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到要删除的多路复用配置")
	}
	return nil
}

// pushMultiplexConfigToAgent 推送多路复用配置到Agent
func (s *multiplexService) pushMultiplexConfigToAgent(agentID, protocol string, config models.MultiplexConfig) error {
	// 构建gRPC请求
	multiplexConfig := map[string]interface{}{
		"enabled":         config.Enabled,
		"protocol":        "smux", // 固定为smux
		"max_connections": config.MaxConnections,
		"min_streams":     config.MinStreams,
		"padding":         config.Padding,
	}

	// 添加brutal配置（如果存在）
	if config.BrutalConfig != nil && len(config.BrutalConfig) > 0 {
		multiplexConfig["brutal"] = config.BrutalConfig
	}

	// 转换为JSON字符串
	configJSON, err := json.Marshal(multiplexConfig)
	if err != nil {
		return fmt.Errorf("序列化多路复用配置失败: %w", err)
	}

	// 通过gRPC客户端推送配置
	if err := s.agentClient.UpdateMultiplexConfig(agentID, protocol, string(configJSON)); err != nil {
		return fmt.Errorf("推送多路复用配置到Agent失败: %w", err)
	}

	return nil
}