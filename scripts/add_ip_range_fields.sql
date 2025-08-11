-- 为agents表添加IP段相关字段

USE xbox_manager;

-- 添加IP段字段
ALTER TABLE `agents` 
ADD COLUMN `ip_range` varchar(45) DEFAULT NULL COMMENT 'IP段，如 192.168.1.0/24' AFTER `ip_address`,
ADD COLUMN `country` varchar(64) DEFAULT NULL COMMENT '国家' AFTER `ip_range`,
ADD COLUMN `region` varchar(128) DEFAULT NULL COMMENT '地区/省份' AFTER `country`,
ADD COLUMN `city` varchar(128) DEFAULT NULL COMMENT '城市' AFTER `region`,
ADD COLUMN `isp` varchar(128) DEFAULT NULL COMMENT '运营商' AFTER `city`;

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS `idx_ip_range` ON `agents` (`ip_range`);
CREATE INDEX IF NOT EXISTS `idx_country` ON `agents` (`country`);
CREATE INDEX IF NOT EXISTS `idx_region` ON `agents` (`region`);  
CREATE INDEX IF NOT EXISTS `idx_isp` ON `agents` (`isp`);

-- 显示更新后的表结构
DESCRIBE agents;