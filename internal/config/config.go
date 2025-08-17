package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Log      LogConfig      `mapstructure:"log"`
	Agent    AgentConfig    `mapstructure:"agent"`
	Report   ReportConfig   `mapstructure:"report"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
	Options  string `mapstructure:"options"`
}

// GRPCConfig gRPC服务配置
type GRPCConfig struct {
	Host string    `mapstructure:"host"`
	Port int       `mapstructure:"port"`
	TLS  TLSConfig `mapstructure:"tls"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // 启用TLS
	CertFile   string `mapstructure:"cert_file"`   // 证书文件路径
	KeyFile    string `mapstructure:"key_file"`    // 私钥文件路径
	CAFile     string `mapstructure:"ca_file"`     // CA证书文件路径
	ServerName string `mapstructure:"server_name"` // 服务器名称（客户端用于验证）
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json, text
	Output     string `mapstructure:"output"` // stdout, file
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
}

// AgentConfig Agent配置
type AgentConfig struct {
	ID               string `mapstructure:"id"`
	ControllerAddr   string `mapstructure:"controller_addr"`
	HeartbeatInterval int   `mapstructure:"heartbeat_interval"` // 秒
	SingBoxConfig    string `mapstructure:"singbox_config"`
	SingBoxBinary    string `mapstructure:"singbox_binary"`
}

// ReportConfig 节点上报配置
type ReportConfig struct {
	Enabled      bool   `mapstructure:"enabled"`          // 是否启用定时上报
	BackendURL   string `mapstructure:"backend_url"`      // 后端服务地址
	Interval     int    `mapstructure:"interval"`         // 上报间隔（秒）
	Timeout      int    `mapstructure:"timeout"`          // 请求超时（秒）
	RetryCount   int    `mapstructure:"retry_count"`      // 重试次数
	RetryDelay   int    `mapstructure:"retry_delay"`      // 重试延迟（秒）
}

var globalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	
	// 设置配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认配置文件路径
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}
	
	// 环境变量配置
	v.SetEnvPrefix("XBOX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// 设置默认值
	setDefaults(v)
	
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}
	
	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	
	globalConfig = &config
	return &config, nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return globalConfig
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// Server默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "release")
	
	// Database默认配置
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "xbox_manager")
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.options", "parseTime=True&loc=Local")
	
	// GRPC默认配置
	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 9090)
	v.SetDefault("grpc.tls.enabled", false)
	
	// Log默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.file", "logs/app.log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age", 30)
	
	// Agent默认配置
	v.SetDefault("agent.heartbeat_interval", 30)
	v.SetDefault("agent.controller_addr", "localhost:9090")
	v.SetDefault("agent.singbox_config", "./sing-box.json")
	v.SetDefault("agent.singbox_binary", "sing-box")
	
	// Report默认配置
	v.SetDefault("report.enabled", true)
	v.SetDefault("report.backend_url", "http://localhost:8080")
	v.SetDefault("report.interval", 60)  // 60秒，即每分钟上报一次
	v.SetDefault("report.timeout", 30)
	v.SetDefault("report.retry_count", 3)
	v.SetDefault("report.retry_delay", 5)
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.Charset,
		c.Database.Options,
	)
}

// GetServerAddr 获取HTTP服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetGRPCAddr 获取gRPC服务器地址
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf("%s:%d", c.GRPC.Host, c.GRPC.Port)
}

// GetLogFile 获取日志文件路径
func (c *Config) GetLogFile() string {
	if c.Log.File == "" {
		return ""
	}
	
	if filepath.IsAbs(c.Log.File) {
		return c.Log.File
	}
	
	return filepath.Join(".", c.Log.File)
}