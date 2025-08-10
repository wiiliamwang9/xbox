package filter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// FilterManager 过滤器管理器
type FilterManager struct {
	mu       sync.RWMutex
	filters  map[string]*ProtocolFilter
	configPath string
	versions []string // 配置版本历史
	currentVersion string
}

// ProtocolFilter 协议过滤器
type ProtocolFilter struct {
	Protocol          string    `json:"protocol"`
	BlacklistDomains  []string  `json:"blacklist_domains"`
	BlacklistIPs      []string  `json:"blacklist_ips"`
	BlacklistPorts    []string  `json:"blacklist_ports"`
	WhitelistDomains  []string  `json:"whitelist_domains"`
	WhitelistIPs      []string  `json:"whitelist_ips"`
	WhitelistPorts    []string  `json:"whitelist_ports"`
	Enabled           bool      `json:"enabled"`
	LastUpdated       time.Time `json:"last_updated"`
}

// FilterConfig 过滤器配置文件结构
type FilterConfig struct {
	Version   string                    `json:"version"`
	Timestamp time.Time                 `json:"timestamp"`
	Filters   map[string]*ProtocolFilter `json:"filters"`
}

// NewFilterManager 创建过滤器管理器
func NewFilterManager(configPath string) *FilterManager {
	fm := &FilterManager{
		filters:    make(map[string]*ProtocolFilter),
		configPath: configPath,
		versions:   make([]string, 0),
		currentVersion: fmt.Sprintf("v%d", time.Now().Unix()),
	}
	
	// 加载现有配置
	if err := fm.loadConfig(); err != nil {
		log.Printf("加载过滤器配置失败: %v", err)
		// 初始化默认协议
		fm.initDefaultFilters()
	}
	
	return fm
}

// initDefaultFilters 初始化默认过滤器
func (fm *FilterManager) initDefaultFilters() {
	protocols := []string{"http", "https", "socks5", "shadowsocks", "vmess", "trojan", "vless"}
	
	for _, protocol := range protocols {
		fm.filters[protocol] = &ProtocolFilter{
			Protocol:          protocol,
			BlacklistDomains:  []string{},
			BlacklistIPs:      []string{},
			BlacklistPorts:    []string{},
			WhitelistDomains:  []string{},
			WhitelistIPs:      []string{},
			WhitelistPorts:    []string{},
			Enabled:           true,
			LastUpdated:       time.Now(),
		}
	}
}

// UpdateBlacklist 更新黑名单
func (fm *FilterManager) UpdateBlacklist(protocol string, domains, ips, ports []string, operation string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	filter, exists := fm.filters[protocol]
	if !exists {
		filter = &ProtocolFilter{
			Protocol:          protocol,
			BlacklistDomains:  []string{},
			BlacklistIPs:      []string{},
			BlacklistPorts:    []string{},
			WhitelistDomains:  []string{},
			WhitelistIPs:      []string{},
			WhitelistPorts:    []string{},
			Enabled:           true,
			LastUpdated:       time.Now(),
		}
		fm.filters[protocol] = filter
	}
	
	switch operation {
	case "add":
		filter.BlacklistDomains = fm.mergeUnique(filter.BlacklistDomains, domains)
		filter.BlacklistIPs = fm.mergeUnique(filter.BlacklistIPs, ips)
		filter.BlacklistPorts = fm.mergeUnique(filter.BlacklistPorts, ports)
	case "remove":
		filter.BlacklistDomains = fm.removeItems(filter.BlacklistDomains, domains)
		filter.BlacklistIPs = fm.removeItems(filter.BlacklistIPs, ips)
		filter.BlacklistPorts = fm.removeItems(filter.BlacklistPorts, ports)
	case "replace":
		filter.BlacklistDomains = domains
		filter.BlacklistIPs = ips
		filter.BlacklistPorts = ports
	case "clear":
		filter.BlacklistDomains = []string{}
		filter.BlacklistIPs = []string{}
		filter.BlacklistPorts = []string{}
	default:
		return fmt.Errorf("不支持的操作: %s", operation)
	}
	
	filter.LastUpdated = time.Now()
	
	// 保存配置并更新版本
	return fm.saveConfig()
}

// UpdateWhitelist 更新白名单
func (fm *FilterManager) UpdateWhitelist(protocol string, domains, ips, ports []string, operation string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	filter, exists := fm.filters[protocol]
	if !exists {
		filter = &ProtocolFilter{
			Protocol:          protocol,
			BlacklistDomains:  []string{},
			BlacklistIPs:      []string{},
			BlacklistPorts:    []string{},
			WhitelistDomains:  []string{},
			WhitelistIPs:      []string{},
			WhitelistPorts:    []string{},
			Enabled:           true,
			LastUpdated:       time.Now(),
		}
		fm.filters[protocol] = filter
	}
	
	switch operation {
	case "add":
		filter.WhitelistDomains = fm.mergeUnique(filter.WhitelistDomains, domains)
		filter.WhitelistIPs = fm.mergeUnique(filter.WhitelistIPs, ips)
		filter.WhitelistPorts = fm.mergeUnique(filter.WhitelistPorts, ports)
	case "remove":
		filter.WhitelistDomains = fm.removeItems(filter.WhitelistDomains, domains)
		filter.WhitelistIPs = fm.removeItems(filter.WhitelistIPs, ips)
		filter.WhitelistPorts = fm.removeItems(filter.WhitelistPorts, ports)
	case "replace":
		filter.WhitelistDomains = domains
		filter.WhitelistIPs = ips
		filter.WhitelistPorts = ports
	case "clear":
		filter.WhitelistDomains = []string{}
		filter.WhitelistIPs = []string{}
		filter.WhitelistPorts = []string{}
	default:
		return fmt.Errorf("不支持的操作: %s", operation)
	}
	
	filter.LastUpdated = time.Now()
	
	// 保存配置并更新版本
	return fm.saveConfig()
}

// GetFilter 获取指定协议的过滤器
func (fm *FilterManager) GetFilter(protocol string) (*ProtocolFilter, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	filter, exists := fm.filters[protocol]
	if !exists {
		return nil, false
	}
	
	// 返回副本防止并发修改
	copy := *filter
	return &copy, true
}

// GetAllFilters 获取所有过滤器
func (fm *FilterManager) GetAllFilters() map[string]*ProtocolFilter {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	result := make(map[string]*ProtocolFilter)
	for k, v := range fm.filters {
		copy := *v
		result[k] = &copy
	}
	
	return result
}

// GetCurrentVersion 获取当前配置版本
func (fm *FilterManager) GetCurrentVersion() string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.currentVersion
}

// Rollback 回滚到指定版本
func (fm *FilterManager) Rollback(targetVersion string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	// 如果目标版本为空，回滚到上一个版本
	if targetVersion == "" {
		if len(fm.versions) < 2 {
			return fmt.Errorf("没有可回滚的版本")
		}
		targetVersion = fm.versions[len(fm.versions)-2]
	}
	
	// 加载备份配置
	backupPath := fmt.Sprintf("%s.%s.backup", fm.configPath, targetVersion)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("版本 %s 的备份文件不存在", targetVersion)
	}
	
	// 读取备份配置
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份配置失败: %v", err)
	}
	
	var config FilterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析备份配置失败: %v", err)
	}
	
	// 恢复配置
	fm.filters = config.Filters
	fm.currentVersion = targetVersion
	
	// 保存当前配置
	return fm.saveConfigFile()
}

// GenerateRouteRules 生成sing-box路由规则
func (fm *FilterManager) GenerateRouteRules() []map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	var rules []map[string]interface{}
	
	for protocol, filter := range fm.filters {
		if !filter.Enabled {
			continue
		}
		
		// 黑名单规则（优先级更高）
		if len(filter.BlacklistDomains) > 0 || len(filter.BlacklistIPs) > 0 {
			rule := map[string]interface{}{
				"protocol": protocol,
				"outbound": "block",
			}
			
			if len(filter.BlacklistDomains) > 0 {
				rule["domain"] = filter.BlacklistDomains
			}
			if len(filter.BlacklistIPs) > 0 {
				rule["ip"] = filter.BlacklistIPs
			}
			if len(filter.BlacklistPorts) > 0 {
				rule["port"] = filter.BlacklistPorts
			}
			
			rules = append(rules, rule)
		}
		
		// 白名单规则
		if len(filter.WhitelistDomains) > 0 || len(filter.WhitelistIPs) > 0 {
			rule := map[string]interface{}{
				"protocol": protocol,
				"outbound": "direct",
			}
			
			if len(filter.WhitelistDomains) > 0 {
				rule["domain"] = filter.WhitelistDomains
			}
			if len(filter.WhitelistIPs) > 0 {
				rule["ip"] = filter.WhitelistIPs
			}
			if len(filter.WhitelistPorts) > 0 {
				rule["port"] = filter.WhitelistPorts
			}
			
			rules = append(rules, rule)
		}
	}
	
	return rules
}

// saveConfig 保存配置并创建备份
func (fm *FilterManager) saveConfig() error {
	// 创建新版本
	newVersion := fmt.Sprintf("v%d", time.Now().Unix())
	
	// 备份当前配置
	if err := fm.backupCurrentConfig(); err != nil {
		log.Printf("备份配置失败: %v", err)
	}
	
	// 更新版本信息
	fm.versions = append(fm.versions, fm.currentVersion)
	fm.currentVersion = newVersion
	
	// 只保留最近10个版本
	if len(fm.versions) > 10 {
		fm.versions = fm.versions[len(fm.versions)-10:]
	}
	
	return fm.saveConfigFile()
}

// saveConfigFile 保存配置文件
func (fm *FilterManager) saveConfigFile() error {
	config := FilterConfig{
		Version:   fm.currentVersion,
		Timestamp: time.Now(),
		Filters:   fm.filters,
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	return os.WriteFile(fm.configPath, data, 0644)
}

// loadConfig 加载配置
func (fm *FilterManager) loadConfig() error {
	if _, err := os.Stat(fm.configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在")
	}
	
	data, err := os.ReadFile(fm.configPath)
	if err != nil {
		return err
	}
	
	var config FilterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	fm.filters = config.Filters
	fm.currentVersion = config.Version
	
	return nil
}

// backupCurrentConfig 备份当前配置
func (fm *FilterManager) backupCurrentConfig() error {
	backupPath := fmt.Sprintf("%s.%s.backup", fm.configPath, fm.currentVersion)
	
	data, err := os.ReadFile(fm.configPath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupPath, data, 0644)
}

// mergeUnique 合并数组并去重
func (fm *FilterManager) mergeUnique(existing, new []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	// 添加现有项
	for _, item := range existing {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	// 添加新项
	for _, item := range new {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// removeItems 从数组中移除指定项
func (fm *FilterManager) removeItems(existing, toRemove []string) []string {
	removeSet := make(map[string]bool)
	for _, item := range toRemove {
		removeSet[item] = true
	}
	
	result := make([]string, 0)
	for _, item := range existing {
		if !removeSet[item] {
			result = append(result, item)
		}
	}
	
	return result
}