-- 创建多路复用配置表

USE xbox_manager;

CREATE TABLE IF NOT EXISTS `multiplex_configs` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `agent_id` varchar(64) NOT NULL COMMENT 'Agent ID',
    `protocol` varchar(32) NOT NULL COMMENT '协议类型 (vmess, vless, trojan, shadowsocks)',
    `enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用多路复用',
    `multiplex_protocol` varchar(16) DEFAULT 'smux' COMMENT '多路复用协议类型，固定为smux',
    `max_connections` int(11) DEFAULT 4 COMMENT '最大连接数',
    `min_streams` int(11) DEFAULT 4 COMMENT '最小流数量',
    `padding` tinyint(1) DEFAULT 0 COMMENT '是否启用填充',
    `brutal_config` json DEFAULT NULL COMMENT 'brutal配置（可选）',
    `status` enum('active','inactive','error') DEFAULT 'inactive' COMMENT '配置状态',
    `error_message` text COMMENT '错误信息',
    `config_version` varchar(32) DEFAULT NULL COMMENT '配置版本',
    `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_agent_protocol` (`agent_id`, `protocol`),
    KEY `idx_enabled` (`enabled`),
    KEY `idx_status` (`status`),
    UNIQUE KEY `uk_agent_protocol` (`agent_id`, `protocol`) COMMENT '同一Agent的同一协议只能有一个配置'
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='多路复用配置表';

-- 创建索引
CREATE INDEX IF NOT EXISTS `idx_config_version` ON `multiplex_configs` (`config_version`);
CREATE INDEX IF NOT EXISTS `idx_created_at` ON `multiplex_configs` (`created_at`);
CREATE INDEX IF NOT EXISTS `idx_updated_at` ON `multiplex_configs` (`updated_at`);

-- 显示表结构
DESCRIBE multiplex_configs;