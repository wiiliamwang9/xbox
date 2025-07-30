-- Xbox Sing-box管理系统数据库初始化脚本

-- 创建数据库
CREATE DATABASE IF NOT EXISTS xbox_manager DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE xbox_manager;

-- 代理节点表
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(64) PRIMARY KEY COMMENT '代理节点ID',
    hostname VARCHAR(255) NOT NULL COMMENT '主机名',
    ip_address VARCHAR(45) NOT NULL COMMENT 'IP地址',
    version VARCHAR(32) COMMENT '版本号',
    status ENUM('online', 'offline', 'error') DEFAULT 'offline' COMMENT '状态',
    last_heartbeat TIMESTAMP NULL COMMENT '最后心跳时间',
    metadata JSON COMMENT '元数据',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_status (status),
    INDEX idx_last_heartbeat (last_heartbeat),
    INDEX idx_ip (ip_address)
) ENGINE=InnoDB COMMENT='代理节点表';

-- 配置表
CREATE TABLE IF NOT EXISTS configs (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '配置ID',
    agent_id VARCHAR(64) NOT NULL COMMENT '代理节点ID',
    config_content TEXT NOT NULL COMMENT '配置内容',
    config_version VARCHAR(32) NOT NULL COMMENT '配置版本',
    status ENUM('pending', 'applied', 'failed') DEFAULT 'pending' COMMENT '应用状态',
    apply_time TIMESTAMP NULL COMMENT '应用时间',
    error_message TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    INDEX idx_agent_version (agent_id, config_version),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB COMMENT='配置表';

-- 规则表
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(64) PRIMARY KEY COMMENT '规则ID',
    agent_id VARCHAR(64) NOT NULL COMMENT '代理节点ID',
    rule_type VARCHAR(32) NOT NULL COMMENT '规则类型',
    content TEXT NOT NULL COMMENT '规则内容',
    priority INT DEFAULT 0 COMMENT '优先级',
    enabled BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    metadata JSON COMMENT '元数据',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    INDEX idx_agent_type (agent_id, rule_type),
    INDEX idx_priority (priority),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB COMMENT='规则表';

-- 监控数据表
CREATE TABLE IF NOT EXISTS monitoring (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '监控ID',
    agent_id VARCHAR(64) NOT NULL COMMENT '代理节点ID',
    metric_type VARCHAR(32) NOT NULL COMMENT '指标类型',
    metric_value DECIMAL(15,2) COMMENT '指标值',
    metric_unit VARCHAR(16) COMMENT '指标单位',
    metadata JSON COMMENT '元数据',
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '时间戳',
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    INDEX idx_agent_metric (agent_id, metric_type),
    INDEX idx_timestamp (timestamp),
    INDEX idx_metric_type (metric_type)
) ENGINE=InnoDB COMMENT='监控数据表';

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id INT AUTO_INCREMENT PRIMARY KEY COMMENT '配置ID',
    config_key VARCHAR(128) NOT NULL UNIQUE COMMENT '配置键',
    config_value TEXT COMMENT '配置值',
    description VARCHAR(255) COMMENT '配置描述',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_key (config_key)
) ENGINE=InnoDB COMMENT='系统配置表';

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '日志ID',
    agent_id VARCHAR(64) COMMENT '代理节点ID',
    operation_type VARCHAR(32) NOT NULL COMMENT '操作类型',
    operation_content JSON COMMENT '操作内容',
    result ENUM('success', 'failed') COMMENT '操作结果',
    error_message TEXT COMMENT '错误信息',
    operator VARCHAR(64) COMMENT '操作者',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
    INDEX idx_agent_operation (agent_id, operation_type),
    INDEX idx_created_at (created_at),
    INDEX idx_result (result)
) ENGINE=InnoDB COMMENT='操作日志表';

-- 插入默认系统配置
INSERT INTO system_configs (config_key, config_value, description) VALUES
('heartbeat_interval', '30', '心跳间隔时间(秒)'),
('config_timeout', '300', '配置应用超时时间(秒)'),
('max_offline_time', '300', '最大离线时间(秒)'),
('log_retention_days', '30', '日志保留天数'),
('monitoring_retention_days', '7', '监控数据保留天数')
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- 创建用户和权限（可选，根据实际需求调整）
-- CREATE USER IF NOT EXISTS 'xbox_user'@'%' IDENTIFIED BY 'xbox_password';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON xbox_manager.* TO 'xbox_user'@'%';
-- FLUSH PRIVILEGES;