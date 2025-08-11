package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// IPRangeInfo IP段信息结构
type IPRangeInfo struct {
	IPRange         string `json:"ip_range"`
	Country         string `json:"country"`
	Region          string `json:"region"`
	City            string `json:"city"`
	ISP             string `json:"isp"`
	DetectionMethod string `json:"detection_method"`
	DetectedAt      string `json:"detected_at"`
}

// IPRangeDetector IP段检测器
type IPRangeDetector struct {
	publicIP    string
	lastInfo    *IPRangeInfo
	httpClient  *http.Client
}

// NewIPRangeDetector 创建IP段检测器
func NewIPRangeDetector() *IPRangeDetector {
	return &IPRangeDetector{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DetectIPRange 检测当前节点的IP段信息
func (d *IPRangeDetector) DetectIPRange() (*IPRangeInfo, error) {
	// 1. 获取公网IP
	publicIP, err := d.getPublicIP()
	if err != nil {
		return nil, fmt.Errorf("获取公网IP失败: %v", err)
	}

	// 如果IP没有变化且已有缓存信息，直接返回
	if d.publicIP == publicIP && d.lastInfo != nil {
		return d.lastInfo, nil
	}

	d.publicIP = publicIP

	// 2. 获取IP地理位置信息
	info, err := d.getIPGeolocation(publicIP)
	if err != nil {
		return nil, fmt.Errorf("获取IP地理位置失败: %v", err)
	}

	// 3. 计算IP段
	ipRange, err := d.calculateIPRange(publicIP)
	if err != nil {
		return nil, fmt.Errorf("计算IP段失败: %v", err)
	}

	info.IPRange = ipRange
	info.DetectionMethod = "auto"
	info.DetectedAt = time.Now().Format("2006-01-02 15:04:05")

	d.lastInfo = info
	return info, nil
}

// getPublicIP 获取公网IP地址
func (d *IPRangeDetector) getPublicIP() (string, error) {
	// 尝试多个IP检测服务
	services := []string{
		"https://ipv4.icanhazip.com",
		"https://api.ipify.org",
		"https://ipinfo.io/ip",
		"https://checkip.amazonaws.com",
	}

	for _, service := range services {
		if ip, err := d.fetchIPFromService(service); err == nil && ip != "" {
			return strings.TrimSpace(ip), nil
		}
	}

	return "", fmt.Errorf("无法从任何服务获取公网IP")
}

// fetchIPFromService 从指定服务获取IP
func (d *IPRangeDetector) fetchIPFromService(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Xbox-Agent/1.0")
	
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// getIPGeolocation 获取IP地理位置信息
func (d *IPRangeDetector) getIPGeolocation(ip string) (*IPRangeInfo, error) {
	// 使用免费的IP地理位置API
	apis := []func(string) (*IPRangeInfo, error){
		d.getInfoFromIPAPI,
		d.getInfoFromIPInfo,
		d.getInfoFromIPStack,
	}

	for _, apiFunc := range apis {
		if info, err := apiFunc(ip); err == nil {
			return info, nil
		}
	}

	// 如果所有API都失败，返回基础信息
	return &IPRangeInfo{
		Country:         "Unknown",
		Region:          "Unknown",
		City:            "Unknown",
		ISP:             "Unknown",
		DetectionMethod: "auto",
		DetectedAt:      time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// getInfoFromIPAPI 从ip-api.com获取信息
func (d *IPRangeDetector) getInfoFromIPAPI(ip string) (*IPRangeInfo, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,regionName,city,isp", ip)
	
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("API返回错误状态")
	}

	return &IPRangeInfo{
		Country: result.Country,
		Region:  result.RegionName,
		City:    result.City,
		ISP:     result.ISP,
	}, nil
}

// getInfoFromIPInfo 从ipinfo.io获取信息
func (d *IPRangeDetector) getInfoFromIPInfo(ip string) (*IPRangeInfo, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Country string `json:"country"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Org     string `json:"org"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &IPRangeInfo{
		Country: result.Country,
		Region:  result.Region,
		City:    result.City,
		ISP:     result.Org,
	}, nil
}

// getInfoFromIPStack 从ipstack.com获取信息 (需要API key，这里简化处理)
func (d *IPRangeDetector) getInfoFromIPStack(ip string) (*IPRangeInfo, error) {
	// 这里可以添加ipstack API调用，需要注册获取API key
	// 暂时返回错误让其他API处理
	return nil, fmt.Errorf("IPStack API未配置")
}

// calculateIPRange 计算IP段
func (d *IPRangeDetector) calculateIPRange(ip string) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("无效的IP地址: %s", ip)
	}

	// 判断是IPv4还是IPv6
	if parsedIP.To4() != nil {
		return d.calculateIPv4Range(ip)
	} else {
		return d.calculateIPv6Range(ip)
	}
}

// calculateIPv4Range 计算IPv4段
func (d *IPRangeDetector) calculateIPv4Range(ip string) (string, error) {
	parsedIP := net.ParseIP(ip).To4()
	if parsedIP == nil {
		return "", fmt.Errorf("无效的IPv4地址: %s", ip)
	}

	// 根据IP地址类型确定子网掩码
	firstOctet := int(parsedIP[0])

	var cidr string
	switch {
	case firstOctet >= 1 && firstOctet <= 126: // A类地址
		cidr = fmt.Sprintf("%d.0.0.0/8", firstOctet)
	case firstOctet >= 128 && firstOctet <= 191: // B类地址
		cidr = fmt.Sprintf("%d.%d.0.0/16", parsedIP[0], parsedIP[1])
	case firstOctet >= 192 && firstOctet <= 223: // C类地址
		cidr = fmt.Sprintf("%d.%d.%d.0/24", parsedIP[0], parsedIP[1], parsedIP[2])
	case firstOctet >= 224 && firstOctet <= 239: // D类地址(组播)
		cidr = fmt.Sprintf("%s/32", ip) // 单个地址
	default:
		cidr = fmt.Sprintf("%s/32", ip) // 默认单个地址
	}

	return cidr, nil
}

// calculateIPv6Range 计算IPv6段
func (d *IPRangeDetector) calculateIPv6Range(ip string) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("无效的IPv6地址: %s", ip)
	}

	// 对于IPv6，通常使用/64子网
	// 简化处理：取前64位
	ipv6 := parsedIP.To16()
	if ipv6 == nil {
		return "", fmt.Errorf("无法转换为IPv6: %s", ip)
	}

	// 将后64位置零
	for i := 8; i < 16; i++ {
		ipv6[i] = 0
	}

	networkIP := net.IP(ipv6)
	return fmt.Sprintf("%s/64", networkIP.String()), nil
}

// GetCachedInfo 获取缓存的IP段信息
func (d *IPRangeDetector) GetCachedInfo() *IPRangeInfo {
	return d.lastInfo
}

// SetManualInfo 手动设置IP段信息
func (d *IPRangeDetector) SetManualInfo(info *IPRangeInfo) {
	info.DetectionMethod = "manual"
	info.DetectedAt = time.Now().Format("2006-01-02 15:04:05")
	d.lastInfo = info
}