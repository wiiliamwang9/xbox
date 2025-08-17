package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Agent 代理节点模型
type Agent struct {
	ID            string         `gorm:"primaryKey;size:64" json:"id"`
	Hostname      string         `gorm:"not null;size:255" json:"hostname"`
	IPAddress     string         `gorm:"not null;size:45;index" json:"ip_address"`
	IPRange       string         `gorm:"size:45;index" json:"ip_range"`        // IP段，如 192.168.1.0/24
	Country       string         `gorm:"size:64;index" json:"country"`         // 国家
	Region        string         `gorm:"size:128;index" json:"region"`         // 地区/省份
	City          string         `gorm:"size:128" json:"city"`                 // 城市
	ISP           string         `gorm:"size:128;index" json:"isp"`            // 运营商
	Version       string         `gorm:"size:32" json:"version"`
	Status             string         `gorm:"type:enum('online','offline','error');default:'offline';index" json:"status"`
	LastHeartbeat      *time.Time     `gorm:"index" json:"last_heartbeat"`
	CurrentConnections int            `gorm:"default:0" json:"current_connections"`
	CPUUsage           float64        `gorm:"type:decimal(5,2);default:0" json:"cpu_usage"`
	MemoryUsage        float64        `gorm:"type:decimal(5,2);default:0" json:"memory_usage"`
	DiskUsage          float64        `gorm:"type:decimal(5,2);default:0" json:"disk_usage"`
	NetworkLatency     int            `gorm:"default:0" json:"network_latency"`
	Metadata           JSON           `gorm:"type:json" json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Configs          []Config          `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"configs,omitempty"`
	Rules            []Rule            `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"rules,omitempty"`
	Monitoring       []Monitor         `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"monitoring,omitempty"`
	Logs             []OpLog           `gorm:"foreignKey:AgentID;constraint:OnDelete:SET NULL" json:"logs,omitempty"`
	MultiplexConfigs []MultiplexConfig `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"multiplex_configs,omitempty"`
}

// Config 配置模型
type Config struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AgentID       string    `gorm:"not null;size:64;index:idx_agent_version,priority:1" json:"agent_id"`
	ConfigContent string    `gorm:"not null;type:text" json:"config_content"`
	ConfigVersion string    `gorm:"not null;size:32;index:idx_agent_version,priority:2" json:"config_version"`
	Status        string    `gorm:"type:enum('pending','applied','failed');default:'pending';index" json:"status"`
	ApplyTime     *time.Time `json:"apply_time"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`

	// 关联关系
	Agent Agent `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
}

// Rule 规则模型
type Rule struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	AgentID   string    `gorm:"not null;size:64;index:idx_agent_type,priority:1" json:"agent_id"`
	RuleType  string    `gorm:"not null;size:32;index:idx_agent_type,priority:2" json:"rule_type"`
	Content   string    `gorm:"not null;type:text" json:"content"`
	Priority  int       `gorm:"default:0;index" json:"priority"`
	Enabled   bool      `gorm:"default:true;index" json:"enabled"`
	Metadata  JSON      `gorm:"type:json" json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	Agent Agent `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
}

// Monitor 监控数据模型
type Monitor struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AgentID     string    `gorm:"not null;size:64;index:idx_agent_metric,priority:1" json:"agent_id"`
	MetricType  string    `gorm:"not null;size:32;index:idx_agent_metric,priority:2;index:idx_metric_type" json:"metric_type"`
	MetricValue float64   `gorm:"type:decimal(15,2)" json:"metric_value"`
	MetricUnit  string    `gorm:"size:16" json:"metric_unit"`
	Metadata    JSON      `gorm:"type:json" json:"metadata"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`

	// 关联关系
	Agent Agent `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ConfigKey   string    `gorm:"not null;uniqueIndex;size:128" json:"config_key"`
	ConfigValue string    `gorm:"type:text" json:"config_value"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OpLog 操作日志模型
type OpLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	AgentID          *string   `gorm:"size:64;index:idx_agent_operation,priority:1" json:"agent_id"`
	OperationType    string    `gorm:"not null;size:32;index:idx_agent_operation,priority:2" json:"operation_type"`
	OperationContent JSON      `gorm:"type:json" json:"operation_content"`
	Result           string    `gorm:"type:enum('success','failed');index" json:"result"`
	ErrorMessage     string    `gorm:"type:text" json:"error_message"`
	Operator         string    `gorm:"size:64" json:"operator"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`

	// 关联关系
	Agent *Agent `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
}

// JSON 自定义JSON类型
type JSON map[string]interface{}

// Value 实现driver.Valuer接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现sql.Scanner接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	
	return json.Unmarshal(bytes, j)
}

// TableName 返回表名
func (Agent) TableName() string {
	return "agents"
}

func (Config) TableName() string {
	return "configs"
}

func (Rule) TableName() string {
	return "rules"
}

func (Monitor) TableName() string {
	return "monitoring"
}

func (SystemConfig) TableName() string {
	return "system_configs"
}

func (OpLog) TableName() string {
	return "operation_logs"
}

// MultiplexConfig 多路复用配置模型
type MultiplexConfig struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	AgentID        string    `gorm:"not null;size:64;index:idx_agent_protocol,priority:1" json:"agent_id"`
	Protocol       string    `gorm:"not null;size:32;index:idx_agent_protocol,priority:2" json:"protocol"`
	Enabled        bool      `gorm:"default:false;index" json:"enabled"`
	MultiplexProto string    `gorm:"column:multiplex_protocol;size:16;default:'smux'" json:"multiplex_protocol"` // 多路复用协议类型，固定为smux
	MaxConnections int       `gorm:"default:4" json:"max_connections"`                 // 最大连接数
	MinStreams     int       `gorm:"default:4" json:"min_streams"`                     // 最小流数量
	Padding        bool      `gorm:"default:false" json:"padding"`                     // 是否启用填充
	BrutalConfig   JSON      `gorm:"type:json" json:"brutal_config"`                   // brutal配置（可选）
	Status         string    `gorm:"type:enum('active','inactive','error');default:'inactive';index" json:"status"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
	ConfigVersion  string    `gorm:"size:32" json:"config_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联关系
	Agent Agent `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
}

func (MultiplexConfig) TableName() string {
	return "multiplex_configs"
}