package database

import (
	"fmt"
	"time"

	"github.com/xbox/sing-box-manager/internal/config"
	"github.com/xbox/sing-box-manager/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.Config) error {
	var err error
	
	// 配置GORM日志级别
	var logLevel logger.LogLevel
	switch cfg.Log.Level {
	case "debug":
		logLevel = logger.Info
	case "info":
		logLevel = logger.Warn
	case "warn", "error":
		logLevel = logger.Error
	default:
		logLevel = logger.Silent
	}
	
	// 数据库连接配置
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}
	
	// 连接数据库
	db, err = gorm.Open(mysql.Open(cfg.GetDSN()), config)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	
	// 获取底层数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	
	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	
	// 自动迁移数据库表
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	
	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate() error {
	// 迁移所有模型
	err := db.AutoMigrate(
		&models.Agent{},
		&models.Config{},
		&models.Rule{},
		&models.Monitor{},
		&models.SystemConfig{},
		&models.OpLog{},
	)
	
	if err != nil {
		return err
	}
	
	// 初始化系统配置
	if err := initSystemConfigs(); err != nil {
		return err
	}
	
	return nil
}

// initSystemConfigs 初始化系统配置
func initSystemConfigs() error {
	configs := []models.SystemConfig{
		{
			ConfigKey:   "heartbeat_interval",
			ConfigValue: "30",
			Description: "心跳间隔时间(秒)",
		},
		{
			ConfigKey:   "config_timeout",
			ConfigValue: "300",
			Description: "配置应用超时时间(秒)",
		},
		{
			ConfigKey:   "max_offline_time",
			ConfigValue: "300",
			Description: "最大离线时间(秒)",
		},
		{
			ConfigKey:   "log_retention_days",
			ConfigValue: "30",
			Description: "日志保留天数",
		},
		{
			ConfigKey:   "monitoring_retention_days",
			ConfigValue: "7",
			Description: "监控数据保留天数",
		},
	}
	
	for _, cfg := range configs {
		// 使用FirstOrCreate避免重复插入
		var existing models.SystemConfig
		result := db.Where("config_key = ?", cfg.ConfigKey).FirstOrCreate(&existing, cfg)
		if result.Error != nil {
			return result.Error
		}
	}
	
	return nil
}

// Health 检查数据库健康状态
func Health() error {
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Ping()
}

// Transaction 执行事务
func Transaction(fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}

// BeginTx 开始事务
func BeginTx() *gorm.DB {
	return db.Begin()
}

// Paginate 分页查询
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		
		if pageSize <= 0 {
			pageSize = 10
		}
		
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// CleanupOldData 清理过期数据
func CleanupOldData() error {
	// 获取配置
	var logRetention, monitoringRetention int
	
	var logConfig models.SystemConfig
	if err := db.Where("config_key = ?", "log_retention_days").First(&logConfig).Error; err == nil {
		if days, err := time.ParseDuration(logConfig.ConfigValue + "h"); err == nil {
			logRetention = int(days.Hours() / 24)
		}
	}
	if logRetention <= 0 {
		logRetention = 30
	}
	
	var monitoringConfig models.SystemConfig
	if err := db.Where("config_key = ?", "monitoring_retention_days").First(&monitoringConfig).Error; err == nil {
		if days, err := time.ParseDuration(monitoringConfig.ConfigValue + "h"); err == nil {
			monitoringRetention = int(days.Hours() / 24)
		}
	}
	if monitoringRetention <= 0 {
		monitoringRetention = 7
	}
	
	// 清理过期日志
	logCutoff := time.Now().AddDate(0, 0, -logRetention)
	if err := db.Where("created_at < ?", logCutoff).Delete(&models.OpLog{}).Error; err != nil {
		return fmt.Errorf("清理过期日志失败: %w", err)
	}
	
	// 清理过期监控数据
	monitoringCutoff := time.Now().AddDate(0, 0, -monitoringRetention)
	if err := db.Where("timestamp < ?", monitoringCutoff).Delete(&models.Monitor{}).Error; err != nil {
		return fmt.Errorf("清理过期监控数据失败: %w", err)
	}
	
	return nil
}