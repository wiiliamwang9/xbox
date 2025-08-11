package singbox

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Manager sing-box进程管理器
type Manager struct {
	mu          sync.RWMutex
	process     *os.Process
	configPath  string
	binaryPath  string
	running     bool
	lastConfig  *Config
}

// Config sing-box配置结构
type Config struct {
	Version     string             `json:"version,omitempty"`
	Log         *LogConfig         `json:"log,omitempty"`
	DNS         *DNSConfig         `json:"dns,omitempty"`
	Inbounds    []Inbound          `json:"inbounds,omitempty"`
	Outbounds   []Outbound         `json:"outbounds,omitempty"`
	Route       *RouteConfig       `json:"route,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
	NTP         *NTPConfig         `json:"ntp,omitempty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Disabled  bool   `json:"disabled,omitempty"`
	Level     string `json:"level,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
}

// DNSConfig DNS配置
type DNSConfig struct {
	Servers        []DNSServer    `json:"servers,omitempty"`
	Rules          []DNSRule      `json:"rules,omitempty"`
	Final          string         `json:"final,omitempty"`
	Strategy       string         `json:"strategy,omitempty"`
	DisableCache   bool           `json:"disable_cache,omitempty"`
	DisableExpire  bool           `json:"disable_expire,omitempty"`
	IndependentCache bool         `json:"independent_cache,omitempty"`
	ReverseMapping bool           `json:"reverse_mapping,omitempty"`
	FakeIP         *FakeIPConfig  `json:"fakeip,omitempty"`
}

// DNSServer DNS服务器配置
type DNSServer struct {
	Tag                string   `json:"tag,omitempty"`
	Address            string   `json:"address,omitempty"`
	AddressResolver    string   `json:"address_resolver,omitempty"`
	AddressStrategy    string   `json:"address_strategy,omitempty"`
	Strategy           string   `json:"strategy,omitempty"`
	Detour             string   `json:"detour,omitempty"`
	ClientSubnet       string   `json:"client_subnet,omitempty"`
}

// DNSRule DNS规则
type DNSRule struct {
	Inbound    []string `json:"inbound,omitempty"`
	IPVersion  int      `json:"ip_version,omitempty"`
	QueryType  []string `json:"query_type,omitempty"`
	Network    []string `json:"network,omitempty"`
	AuthUser   []string `json:"auth_user,omitempty"`
	Protocol   []string `json:"protocol,omitempty"`
	Domain     []string `json:"domain,omitempty"`
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	DomainRegex []string `json:"domain_regex,omitempty"`
	Geosite    []string `json:"geosite,omitempty"`
	SourceGeoIP []string `json:"source_geoip,omitempty"`
	GeoIP      []string `json:"geoip,omitempty"`
	SourceIP   []string `json:"source_ip_cidr,omitempty"`
	IP         []string `json:"ip_cidr,omitempty"`
	SourcePort []string `json:"source_port,omitempty"`
	Port       []string `json:"port,omitempty"`
	ProcessName []string `json:"process_name,omitempty"`
	ProcessPath []string `json:"process_path,omitempty"`
	PackageName []string `json:"package_name,omitempty"`
	User       []string `json:"user,omitempty"`
	UserID     []int    `json:"user_id,omitempty"`
	ClashMode  string   `json:"clash_mode,omitempty"`
	Invert     bool     `json:"invert,omitempty"`
	Server     string   `json:"server,omitempty"`
	DisableCache bool   `json:"disable_cache,omitempty"`
	RewriteTTL *uint32  `json:"rewrite_ttl,omitempty"`
	ClientSubnet string `json:"client_subnet,omitempty"`
}

// FakeIPConfig FakeIP配置
type FakeIPConfig struct {
	Enabled    bool     `json:"enabled,omitempty"`
	Inet4Range string   `json:"inet4_range,omitempty"`
	Inet6Range string   `json:"inet6_range,omitempty"`
}

// Inbound 入站配置
type Inbound struct {
	Type            string         `json:"type,omitempty"`
	Tag             string         `json:"tag,omitempty"`
	Listen          string         `json:"listen,omitempty"`
	ListenPort      uint16         `json:"listen_port,omitempty"`
	TCPFastOpen     bool           `json:"tcp_fast_open,omitempty"`
	TCPMultiPath    bool           `json:"tcp_multi_path,omitempty"`
	UDPFragment     *bool          `json:"udp_fragment,omitempty"`
	UDPTimeout      string         `json:"udp_timeout,omitempty"`
	ProxyProtocol   bool           `json:"proxy_protocol,omitempty"`
	Sniff           bool           `json:"sniff,omitempty"`
	SniffOverride   bool           `json:"sniff_override_destination,omitempty"`
	SniffTimeout    string         `json:"sniff_timeout,omitempty"`
	DomainStrategy  string         `json:"domain_strategy,omitempty"`
	UDPDisableDomainUnmapping bool `json:"udp_disable_domain_unmapping,omitempty"`
	
	// Mixed/HTTP/SOCKS specific
	Users     []InboundUser `json:"users,omitempty"`
	
	// TLS config
	TLS       *InboundTLS   `json:"tls,omitempty"`
	
	// Transport config
	Transport *Transport    `json:"transport,omitempty"`
}

// InboundUser 入站用户配置
type InboundUser struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// InboundTLS 入站TLS配置
type InboundTLS struct {
	Enabled         bool     `json:"enabled,omitempty"`
	ServerName      string   `json:"server_name,omitempty"`
	Insecure        bool     `json:"insecure,omitempty"`
	ALPN            []string `json:"alpn,omitempty"`
	MinVersion      string   `json:"min_version,omitempty"`
	MaxVersion      string   `json:"max_version,omitempty"`
	CipherSuites    []string `json:"cipher_suites,omitempty"`
	Certificate     []string `json:"certificate,omitempty"`
	CertificatePath string   `json:"certificate_path,omitempty"`
	Key             []string `json:"key,omitempty"`
	KeyPath         string   `json:"key_path,omitempty"`
	ACME            *ACME    `json:"acme,omitempty"`
}

// ACME 自动证书配置
type ACME struct {
	Domain                []string `json:"domain,omitempty"`
	DataDirectory         string   `json:"data_directory,omitempty"`
	DefaultServerName     string   `json:"default_server_name,omitempty"`
	Email                 string   `json:"email,omitempty"`
	Provider              string   `json:"provider,omitempty"`
	DisableHTTPChallenge  bool     `json:"disable_http_challenge,omitempty"`
	DisableTLSALPNChallenge bool   `json:"disable_tls_alpn_challenge,omitempty"`
	AlternativeHTTPPort   uint16   `json:"alternative_http_port,omitempty"`
	AlternativeTLSPort    uint16   `json:"alternative_tls_port,omitempty"`
	ExternalAccount       *ExternalAccount `json:"external_account,omitempty"`
}

// ExternalAccount 外部账户配置
type ExternalAccount struct {
	KeyID  string `json:"key_id,omitempty"`
	MACKey string `json:"mac_key,omitempty"`
}

// Transport 传输层配置
type Transport struct {
	Type             string            `json:"type,omitempty"`
	Host             []string          `json:"host,omitempty"`
	Path             string            `json:"path,omitempty"`
	Method           string            `json:"method,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	IdleTimeout      string            `json:"idle_timeout,omitempty"`
	PingTimeout      string            `json:"ping_timeout,omitempty"`
	WriteBufferSize  int               `json:"write_buffer_size,omitempty"`
	MaxEarlyData     uint32            `json:"max_early_data,omitempty"`
	EarlyDataHeaderName string         `json:"early_data_header_name,omitempty"`
	ServiceName      string            `json:"service_name,omitempty"`
}

// Outbound 出站配置
type Outbound struct {
	Type           string         `json:"type,omitempty"`
	Tag            string         `json:"tag,omitempty"`
	Server         string         `json:"server,omitempty"`
	ServerPort     uint16         `json:"server_port,omitempty"`
	Method         string         `json:"method,omitempty"`
	Password       string         `json:"password,omitempty"`
	Plugin         string         `json:"plugin,omitempty"`
	PluginOpts     string         `json:"plugin_opts,omitempty"`
	Network        string         `json:"network,omitempty"`
	UDPOverTCP     bool           `json:"udp_over_tcp,omitempty"`
	
	// Multiplex
	Multiplex      *MultiplexConfig `json:"multiplex,omitempty"`
	
	// TLS config
	TLS            *OutboundTLS     `json:"tls,omitempty"`
	
	// Transport config  
	Transport      *Transport       `json:"transport,omitempty"`
	
	// Shadowsocks specific
	MultiPassword  []string         `json:"multi_password,omitempty"`
	
	// VMess specific
	UUID           string           `json:"uuid,omitempty"`
	Security       string           `json:"security,omitempty"`
	AlterId        int              `json:"alter_id,omitempty"`
	GlobalPadding  bool             `json:"global_padding,omitempty"`
	AuthenticatedLength bool        `json:"authenticated_length,omitempty"`
	
	// Trojan specific
	
	// VLESS specific
	Flow           string           `json:"flow,omitempty"`
	
	// WireGuard specific
	SystemInterface bool            `json:"system_interface,omitempty"`
	InterfaceName   string          `json:"interface_name,omitempty"`
	LocalAddress    []string        `json:"local_address,omitempty"`
	PrivateKey      string          `json:"private_key,omitempty"`
	PeerPublicKey   string          `json:"peer_public_key,omitempty"`
	PreSharedKey    string          `json:"pre_shared_key,omitempty"`
	Reserved        []uint8         `json:"reserved,omitempty"`
	Workers         int             `json:"workers,omitempty"`
	MTU             uint32          `json:"mtu,omitempty"`
	
	// HTTP specific
	Username       string           `json:"username,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	
	// SOCKS specific  
	Version        string           `json:"version,omitempty"`
	
	// Hysteria specific
	Up             string           `json:"up,omitempty"`
	Down           string           `json:"down,omitempty"`
	Auth           string           `json:"auth,omitempty"`
	AuthStr        string           `json:"auth_str,omitempty"`
	Obfs           string           `json:"obfs,omitempty"`
	ReceiveWindowConn uint64        `json:"recv_window_conn,omitempty"`
	ReceiveWindow  uint64           `json:"recv_window,omitempty"`
	DisableMTUDiscovery bool        `json:"disable_mtu_discovery,omitempty"`
	
	// Common outbound fields
	DialerOptions
}

// MultiplexConfig 多路复用配置
type MultiplexConfig struct {
	Enabled        bool     `json:"enabled,omitempty"`
	Protocol       string   `json:"protocol,omitempty"`
	MaxConnections int      `json:"max_connections,omitempty"`
	MinStreams     int      `json:"min_streams,omitempty"`
	MaxStreams     int      `json:"max_streams,omitempty"`
	Padding        bool     `json:"padding,omitempty"`
	Brutal         *Brutal  `json:"brutal,omitempty"`
}

// Brutal Brutal配置
type Brutal struct {
	Enabled bool   `json:"enabled,omitempty"`
	Up      string `json:"up,omitempty"`
	Down    string `json:"down,omitempty"`
}

// OutboundTLS 出站TLS配置
type OutboundTLS struct {
	Enabled               bool     `json:"enabled,omitempty"`
	DisableSNI           bool     `json:"disable_sni,omitempty"`
	ServerName           string   `json:"server_name,omitempty"`
	Insecure             bool     `json:"insecure,omitempty"`
	ALPN                 []string `json:"alpn,omitempty"`
	MinVersion           string   `json:"min_version,omitempty"`
	MaxVersion           string   `json:"max_version,omitempty"`
	CipherSuites         []string `json:"cipher_suites,omitempty"`
	Certificate          []string `json:"certificate,omitempty"`
	CertificatePath      string   `json:"certificate_path,omitempty"`
	ECH                  *ECHConfig `json:"ech,omitempty"`
	UTLS                 *UTLSConfig `json:"utls,omitempty"`
	Reality              *RealityConfig `json:"reality,omitempty"`
}

// ECHConfig ECH配置
type ECHConfig struct {
	Enabled                bool     `json:"enabled,omitempty"`
	PQSignatureSchemesEnabled bool `json:"pq_signature_schemes_enabled,omitempty"`
	DynamicRecordSizingDisabled bool `json:"dynamic_record_sizing_disabled,omitempty"`
	Config                 []string `json:"config,omitempty"`
}

// UTLSConfig uTLS配置
type UTLSConfig struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// RealityConfig Reality配置
type RealityConfig struct {
	Enabled   bool   `json:"enabled,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	ShortID   string `json:"short_id,omitempty"`
}

// DialerOptions 拨号器选项
type DialerOptions struct {
	Detour              string `json:"detour,omitempty"`
	BindInterface       string `json:"bind_interface,omitempty"`
	Inet4BindAddress    string `json:"inet4_bind_address,omitempty"`
	Inet6BindAddress    string `json:"inet6_bind_address,omitempty"`
	ProtectPath         string `json:"protect_path,omitempty"`
	RoutingMark         int    `json:"routing_mark,omitempty"`
	ReuseAddr           bool   `json:"reuse_addr,omitempty"`
	ConnectTimeout      string `json:"connect_timeout,omitempty"`
	TCPFastOpen         bool   `json:"tcp_fast_open,omitempty"`
	TCPMultiPath        bool   `json:"tcp_multi_path,omitempty"`
	UDPFragment         *bool  `json:"udp_fragment,omitempty"`
	UDPTimeout          string `json:"udp_timeout,omitempty"`
	DomainStrategy      string `json:"domain_strategy,omitempty"`
	FallbackDelay       string `json:"fallback_delay,omitempty"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	GeoIP               *GeoIPConfig    `json:"geoip,omitempty"`
	Geosite             *GeositeConfig  `json:"geosite,omitempty"`
	Rules               []RouteRule     `json:"rules,omitempty"`
	Final               string          `json:"final,omitempty"`
	AutoDetectInterface bool            `json:"auto_detect_interface,omitempty"`
	OverrideAndroidVPN  bool            `json:"override_android_vpn,omitempty"`
	DefaultInterface    string          `json:"default_interface,omitempty"`
	DefaultMark         int             `json:"default_mark,omitempty"`
}

// GeoIPConfig GeoIP配置
type GeoIPConfig struct {
	Path           string `json:"path,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	DownloadDetour string `json:"download_detour,omitempty"`
}

// GeositeConfig Geosite配置
type GeositeConfig struct {
	Path           string `json:"path,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	DownloadDetour string `json:"download_detour,omitempty"`
}

// RouteRule 路由规则
type RouteRule struct {
	Inbound           []string `json:"inbound,omitempty"`
	IPVersion         int      `json:"ip_version,omitempty"`
	Network           []string `json:"network,omitempty"`
	AuthUser          []string `json:"auth_user,omitempty"`
	Protocol          []string `json:"protocol,omitempty"`
	Client            []string `json:"client,omitempty"`
	Domain            []string `json:"domain,omitempty"`
	DomainSuffix      []string `json:"domain_suffix,omitempty"`
	DomainKeyword     []string `json:"domain_keyword,omitempty"`
	DomainRegex       []string `json:"domain_regex,omitempty"`
	Geosite           []string `json:"geosite,omitempty"`
	SourceGeoIP       []string `json:"source_geoip,omitempty"`
	GeoIP             []string `json:"geoip,omitempty"`
	SourceIP          []string `json:"source_ip_cidr,omitempty"`
	SourceIPIsPrivate bool     `json:"source_ip_is_private,omitempty"`
	IP                []string `json:"ip_cidr,omitempty"`
	IPIsPrivate       bool     `json:"ip_is_private,omitempty"`
	SourcePort        []string `json:"source_port,omitempty"`
	SourcePortRange   []string `json:"source_port_range,omitempty"`
	Port              []string `json:"port,omitempty"`
	PortRange         []string `json:"port_range,omitempty"`
	ProcessName       []string `json:"process_name,omitempty"`
	ProcessPath       []string `json:"process_path,omitempty"`
	PackageName       []string `json:"package_name,omitempty"`
	User              []string `json:"user,omitempty"`
	UserID            []int    `json:"user_id,omitempty"`
	ClashMode         string   `json:"clash_mode,omitempty"`
	WIFISSID          []string `json:"wifi_ssid,omitempty"`
	WIFIBSSID         []string `json:"wifi_bssid,omitempty"`
	RuleSet           []string `json:"rule_set,omitempty"`
	RuleSetIPCIDRMatchSource bool `json:"rule_set_ip_cidr_match_source,omitempty"`
	Invert            bool     `json:"invert,omitempty"`
	Outbound          string   `json:"outbound,omitempty"`
}

// ExperimentalConfig 实验性配置
type ExperimentalConfig struct {
	CacheFile          *CacheFileConfig          `json:"cache_file,omitempty"`
	ClashAPI           *ClashAPIConfig           `json:"clash_api,omitempty"`
	V2RayAPI           *V2RayAPIConfig           `json:"v2ray_api,omitempty"`
	Debug              *DebugConfig              `json:"debug,omitempty"`
}

// CacheFileConfig 缓存文件配置
type CacheFileConfig struct {
	Enabled   bool   `json:"enabled,omitempty"`
	Path      string `json:"path,omitempty"`
	CacheID   string `json:"cache_id,omitempty"`
	StoreFakeIP bool `json:"store_fakeip,omitempty"`
	StoreRDRC  bool   `json:"store_rdrc,omitempty"`
	RDRCTimeout string `json:"rdrc_timeout,omitempty"`
}

// ClashAPIConfig Clash API配置
type ClashAPIConfig struct {
	ExternalController string             `json:"external_controller,omitempty"`
	ExternalUI         string             `json:"external_ui,omitempty"`
	ExternalUIDownloadURL string          `json:"external_ui_download_url,omitempty"`
	ExternalUIDownloadDetour string       `json:"external_ui_download_detour,omitempty"`
	Secret             string             `json:"secret,omitempty"`
	DefaultMode        string             `json:"default_mode,omitempty"`
	ModeList           []string           `json:"mode_list,omitempty"`
	StoreMode          bool               `json:"store_mode,omitempty"`
	StoreSelected      bool               `json:"store_selected,omitempty"`
	StoreFakeIP        bool               `json:"store_fakeip,omitempty"`
	CacheFile          string             `json:"cache_file,omitempty"`
	CacheID            string             `json:"cache_id,omitempty"`
	AccessControlAllowOrigin []string     `json:"access_control_allow_origin,omitempty"`
	AccessControlAllowPrivateNetwork bool `json:"access_control_allow_private_network,omitempty"`
}

// V2RayAPIConfig V2Ray API配置
type V2RayAPIConfig struct {
	Listen string                `json:"listen,omitempty"`
	Stats  *V2RayAPIStatsConfig  `json:"stats,omitempty"`
}

// V2RayAPIStatsConfig V2Ray API统计配置
type V2RayAPIStatsConfig struct {
	Enabled   bool     `json:"enabled,omitempty"`
	Inbounds  []string `json:"inbounds,omitempty"`
	Outbounds []string `json:"outbounds,omitempty"`
	Users     []string `json:"users,omitempty"`
}

// DebugConfig 调试配置
type DebugConfig struct {
	Listen             string `json:"listen,omitempty"`
	GCPercent          int    `json:"gc_percent,omitempty"`
	MaxStack           int    `json:"max_stack,omitempty"`
	MaxThreads         int    `json:"max_threads,omitempty"`
	PanicOnFault       bool   `json:"panic_on_fault,omitempty"`
	TraceBack          string `json:"trace_back,omitempty"`
	MemoryLimit        string `json:"memory_limit,omitempty"`
	OOMKiller          bool   `json:"oom_killer,omitempty"`
}

// NTPConfig NTP配置
type NTPConfig struct {
	Enabled       bool     `json:"enabled,omitempty"`
	Server        string   `json:"server,omitempty"`
	ServerPort    uint16   `json:"server_port,omitempty"`
	Interval      string   `json:"interval,omitempty"`
	WriteToSystem bool     `json:"write_to_system,omitempty"`
	Detour        string   `json:"detour,omitempty"`
}

// NewManager 创建sing-box管理器
func NewManager(binaryPath, configPath string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		configPath: configPath,
		running:    false,
	}
}

// Start 启动sing-box进程
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("sing-box已在运行")
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", m.configPath)
	}

	// 启动sing-box进程
	cmd := exec.Command(m.binaryPath, "run", "-c", m.configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动sing-box失败: %v", err)
	}

	m.process = cmd.Process
	m.running = true

	log.Printf("sing-box进程已启动, PID: %d", m.process.Pid)

	// 启动进程监控
	go m.monitorProcess(cmd)

	return nil
}

// Stop 停止sing-box进程
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.process == nil {
		return fmt.Errorf("sing-box未运行")
	}

	// 发送SIGTERM信号
	if err := m.process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("停止sing-box失败: %v", err)
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		_, err := m.process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		m.running = false
		m.process = nil
		log.Println("sing-box进程已停止")
		return err
	case <-time.After(10 * time.Second):
		// 强制杀死进程
		if err := m.process.Kill(); err != nil {
			return fmt.Errorf("强制终止sing-box失败: %v", err)
		}
		m.running = false
		m.process = nil
		log.Println("sing-box进程已被强制终止")
		return nil
	}
}

// Restart 重启sing-box进程
func (m *Manager) Restart() error {
	if m.IsRunning() {
		if err := m.Stop(); err != nil {
			return fmt.Errorf("停止进程失败: %v", err)
		}
	}

	time.Sleep(1 * time.Second) // 等待进程完全停止

	return m.Start()
}

// IsRunning 检查进程是否运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetPID 获取进程ID
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.process != nil {
		return m.process.Pid
	}
	return 0
}

// UpdateConfig 更新配置并重启
func (m *Manager) UpdateConfig(config *Config) error {
	// 备份当前配置
	if err := m.backupConfig(); err != nil {
		log.Printf("备份配置失败: %v", err)
	}

	// 写入新配置
	if err := m.writeConfig(config); err != nil {
		return fmt.Errorf("写入配置失败: %v", err)
	}

	// 验证配置
	if err := m.validateConfig(); err != nil {
		// 配置无效，恢复备份
		m.restoreConfig()
		return fmt.Errorf("配置验证失败: %v", err)
	}

	m.lastConfig = config

	// 重启服务应用新配置
	if m.IsRunning() {
		return m.Restart()
	}

	return nil
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastConfig
}

// LoadConfigFromFile 从文件加载配置
func (m *Manager) LoadConfigFromFile() (*Config, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	m.mu.Lock()
	m.lastConfig = &config
	m.mu.Unlock()

	return &config, nil
}

// PrintConfigInfo 打印详细配置信息
func (m *Manager) PrintConfigInfo() error {
	config, err := m.LoadConfigFromFile()
	if err != nil {
		return err
	}

	log.Println("=== sing-box 配置信息 ===")
	log.Printf("配置文件路径: %s", m.configPath)
	log.Printf("二进制文件路径: %s", m.binaryPath)
	
	// 基础信息
	if config.Version != "" {
		log.Printf("配置版本: %s", config.Version)
	}

	// 日志配置
	if config.Log != nil {
		log.Println("\n--- 日志配置 ---")
		log.Printf("  已禁用: %t", config.Log.Disabled)
		log.Printf("  日志级别: %s", config.Log.Level)
		log.Printf("  输出: %s", config.Log.Output)
		log.Printf("  时间戳: %t", config.Log.Timestamp)
	}

	// DNS配置
	if config.DNS != nil {
		log.Println("\n--- DNS配置 ---")
		log.Printf("  DNS策略: %s", config.DNS.Strategy)
		log.Printf("  最终DNS: %s", config.DNS.Final)
		log.Printf("  禁用缓存: %t", config.DNS.DisableCache)
		log.Printf("  禁用过期: %t", config.DNS.DisableExpire)
		log.Printf("  独立缓存: %t", config.DNS.IndependentCache)
		log.Printf("  反向映射: %t", config.DNS.ReverseMapping)
		
		if len(config.DNS.Servers) > 0 {
			log.Printf("  DNS服务器数量: %d", len(config.DNS.Servers))
			for i, server := range config.DNS.Servers {
				log.Printf("    [%d] Tag: %s, 地址: %s", i+1, server.Tag, server.Address)
				if server.AddressResolver != "" {
					log.Printf("        地址解析器: %s", server.AddressResolver)
				}
				if server.Strategy != "" {
					log.Printf("        策略: %s", server.Strategy)
				}
				if server.Detour != "" {
					log.Printf("        绕行: %s", server.Detour)
				}
			}
		}

		if len(config.DNS.Rules) > 0 {
			log.Printf("  DNS规则数量: %d", len(config.DNS.Rules))
		}

		if config.DNS.FakeIP != nil {
			log.Printf("  FakeIP已启用: %t", config.DNS.FakeIP.Enabled)
			if config.DNS.FakeIP.Inet4Range != "" {
				log.Printf("    IPv4范围: %s", config.DNS.FakeIP.Inet4Range)
			}
			if config.DNS.FakeIP.Inet6Range != "" {
				log.Printf("    IPv6范围: %s", config.DNS.FakeIP.Inet6Range)
			}
		}
	}

	// 入站配置
	if len(config.Inbounds) > 0 {
		log.Println("\n--- 入站配置 ---")
		log.Printf("  入站数量: %d", len(config.Inbounds))
		for i, inbound := range config.Inbounds {
			log.Printf("  [%d] %s (%s)", i+1, inbound.Tag, inbound.Type)
			log.Printf("      监听: %s:%d", inbound.Listen, inbound.ListenPort)
			if inbound.Sniff {
				log.Printf("      流量探测: 启用")
				if inbound.SniffOverride {
					log.Printf("      覆盖目标: 启用")
				}
			}
			if inbound.TCPFastOpen {
				log.Printf("      TCP Fast Open: 启用")
			}
			if inbound.TCPMultiPath {
				log.Printf("      TCP Multipath: 启用")
			}
			if len(inbound.Users) > 0 {
				log.Printf("      用户数量: %d", len(inbound.Users))
			}
			if inbound.TLS != nil && inbound.TLS.Enabled {
				log.Printf("      TLS: 启用")
				if inbound.TLS.ServerName != "" {
					log.Printf("        服务器名: %s", inbound.TLS.ServerName)
				}
			}
		}
	}

	// 出站配置
	if len(config.Outbounds) > 0 {
		log.Println("\n--- 出站配置 ---")
		log.Printf("  出站数量: %d", len(config.Outbounds))
		for i, outbound := range config.Outbounds {
			log.Printf("  [%d] %s (%s)", i+1, outbound.Tag, outbound.Type)
			if outbound.Server != "" {
				log.Printf("      服务器: %s:%d", outbound.Server, outbound.ServerPort)
			}
			if outbound.Method != "" {
				log.Printf("      方法: %s", outbound.Method)
			}
			if outbound.UUID != "" {
				log.Printf("      UUID: %s", outbound.UUID)
			}
			if outbound.Security != "" {
				log.Printf("      安全: %s", outbound.Security)
			}
			if outbound.Network != "" {
				log.Printf("      网络: %s", outbound.Network)
			}
			if outbound.TLS != nil && outbound.TLS.Enabled {
				log.Printf("      TLS: 启用")
				if outbound.TLS.ServerName != "" {
					log.Printf("        服务器名: %s", outbound.TLS.ServerName)
				}
			}
			if outbound.Transport != nil {
				log.Printf("      传输: %s", outbound.Transport.Type)
				if outbound.Transport.Path != "" {
					log.Printf("        路径: %s", outbound.Transport.Path)
				}
			}
			if outbound.Multiplex != nil && outbound.Multiplex.Enabled {
				log.Printf("      多路复用: 启用")
				log.Printf("        协议: %s", outbound.Multiplex.Protocol)
				log.Printf("        最大连接: %d", outbound.Multiplex.MaxConnections)
			}
		}
	}

	// 路由配置
	if config.Route != nil {
		log.Println("\n--- 路由配置 ---")
		log.Printf("  最终出站: %s", config.Route.Final)
		log.Printf("  自动检测接口: %t", config.Route.AutoDetectInterface)
		if config.Route.DefaultInterface != "" {
			log.Printf("  默认接口: %s", config.Route.DefaultInterface)
		}
		if config.Route.DefaultMark != 0 {
			log.Printf("  默认标记: %d", config.Route.DefaultMark)
		}

		if len(config.Route.Rules) > 0 {
			log.Printf("  路由规则数量: %d", len(config.Route.Rules))
			for i, rule := range config.Route.Rules {
				log.Printf("    [%d] -> %s", i+1, rule.Outbound)
				if len(rule.Domain) > 0 {
					log.Printf("        域名: %v", rule.Domain)
				}
				if len(rule.DomainSuffix) > 0 {
					log.Printf("        域名后缀: %v", rule.DomainSuffix)
				}
				if len(rule.DomainKeyword) > 0 {
					log.Printf("        域名关键词: %v", rule.DomainKeyword)
				}
				if len(rule.GeoIP) > 0 {
					log.Printf("        GeoIP: %v", rule.GeoIP)
				}
				if len(rule.Geosite) > 0 {
					log.Printf("        Geosite: %v", rule.Geosite)
				}
				if len(rule.IP) > 0 {
					log.Printf("        IP: %v", rule.IP)
				}
				if len(rule.Port) > 0 {
					log.Printf("        端口: %v", rule.Port)
				}
			}
		}

		if config.Route.GeoIP != nil {
			log.Printf("  GeoIP路径: %s", config.Route.GeoIP.Path)
			if config.Route.GeoIP.DownloadURL != "" {
				log.Printf("  GeoIP下载URL: %s", config.Route.GeoIP.DownloadURL)
			}
		}

		if config.Route.Geosite != nil {
			log.Printf("  Geosite路径: %s", config.Route.Geosite.Path)
			if config.Route.Geosite.DownloadURL != "" {
				log.Printf("  Geosite下载URL: %s", config.Route.Geosite.DownloadURL)
			}
		}
	}

	// 实验性功能
	if config.Experimental != nil {
		log.Println("\n--- 实验性功能 ---")
		
		if config.Experimental.CacheFile != nil {
			log.Printf("  缓存文件: 启用(%t)", config.Experimental.CacheFile.Enabled)
			if config.Experimental.CacheFile.Path != "" {
				log.Printf("    路径: %s", config.Experimental.CacheFile.Path)
			}
			log.Printf("    存储FakeIP: %t", config.Experimental.CacheFile.StoreFakeIP)
		}

		if config.Experimental.ClashAPI != nil {
			log.Printf("  Clash API: %s", config.Experimental.ClashAPI.ExternalController)
			if config.Experimental.ClashAPI.ExternalUI != "" {
				log.Printf("    外部UI: %s", config.Experimental.ClashAPI.ExternalUI)
			}
			if config.Experimental.ClashAPI.Secret != "" {
				log.Printf("    密钥: [已配置]")
			}
		}

		if config.Experimental.V2RayAPI != nil {
			log.Printf("  V2Ray API: %s", config.Experimental.V2RayAPI.Listen)
			if config.Experimental.V2RayAPI.Stats != nil {
				log.Printf("    统计: 启用(%t)", config.Experimental.V2RayAPI.Stats.Enabled)
			}
		}
	}

	// NTP配置
	if config.NTP != nil {
		log.Println("\n--- NTP配置 ---")
		log.Printf("  启用: %t", config.NTP.Enabled)
		if config.NTP.Server != "" {
			log.Printf("  服务器: %s:%d", config.NTP.Server, config.NTP.ServerPort)
		}
		log.Printf("  写入系统: %t", config.NTP.WriteToSystem)
		if config.NTP.Interval != "" {
			log.Printf("  间隔: %s", config.NTP.Interval)
		}
	}

	log.Println("=== 配置信息输出完成 ===")
	return nil
}

// monitorProcess 监控进程状态
func (m *Manager) monitorProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	
	m.mu.Lock()
	m.running = false
	m.process = nil
	m.mu.Unlock()

	if err != nil {
		log.Printf("sing-box进程异常退出: %v", err)
	} else {
		log.Println("sing-box进程正常退出")
	}
}

// writeConfig 写入配置文件
func (m *Manager) writeConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// validateConfig 验证配置文件
func (m *Manager) validateConfig() error {
	// 使用sing-box检查配置
	cmd := exec.Command(m.binaryPath, "check", "-c", m.configPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}
	return nil
}

// backupConfig 备份配置文件
func (m *Manager) backupConfig() error {
	backupPath := m.configPath + ".backup"
	
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(backupPath, data, 0644)
}

// restoreConfig 恢复配置文件
func (m *Manager) restoreConfig() error {
	backupPath := m.configPath + ".backup"
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在")
	}
	
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	
	return os.WriteFile(m.configPath, data, 0644)
}

// GetStatus 获取进程状态信息
func (m *Manager) GetStatus() map[string]string {
	status := map[string]string{
		"running":     fmt.Sprintf("%t", m.IsRunning()),
		"pid":         fmt.Sprintf("%d", m.GetPID()),
		"config_path": m.configPath,
		"binary_path": m.binaryPath,
	}

	if m.IsRunning() {
		status["status"] = "running"
	} else {
		status["status"] = "stopped"
	}

	// 获取配置文件修改时间
	if stat, err := os.Stat(m.configPath); err == nil {
		status["config_modified"] = stat.ModTime().Format(time.RFC3339)
	}

	return status
}