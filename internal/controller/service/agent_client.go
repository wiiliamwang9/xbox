package service

import (
	"context"
	"fmt"
	"time"

	pb "github.com/xbox/sing-box-manager/proto/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient Agent gRPC客户端接口
type AgentClient interface {
	UpdateMultiplexConfig(agentID, protocol, configJSON string) error
	GetMultiplexConfig(agentID, protocol string) (string, error)
	UpdateConfig(agentID, configContent, configVersion string) error
	UpdateBlacklist(agentID, protocol string, domains, ips, ports []string, operation string) error
	UpdateWhitelist(agentID, protocol string, domains, ips, ports []string, operation string) error
	RollbackConfig(agentID, targetVersion, reason string) error
}

// agentClient Agent gRPC客户端实现
type agentClient struct {
	connections map[string]*grpc.ClientConn
	timeout     time.Duration
}

// NewAgentClient 创建Agent gRPC客户端
func NewAgentClient() AgentClient {
	return &agentClient{
		connections: make(map[string]*grpc.ClientConn),
		timeout:     30 * time.Second,
	}
}

// getConnection 获取或创建到Agent的gRPC连接
func (c *agentClient) getConnection(agentID string) (*grpc.ClientConn, error) {
	// 检查是否已有连接
	if conn, exists := c.connections[agentID]; exists {
		return conn, nil
	}

	// TODO: 这里需要从数据库或配置中获取Agent的地址
	// 临时使用默认地址，实际应该根据agentID查询Agent信息获取IP和端口
	address := fmt.Sprintf("agent-%s:9090", agentID) // 假设的地址格式

	// 创建新连接
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接到Agent %s 失败: %w", agentID, err)
	}

	c.connections[agentID] = conn
	return conn, nil
}

// UpdateMultiplexConfig 更新Agent的多路复用配置
func (c *agentClient) UpdateMultiplexConfig(agentID, protocol, configJSON string) error {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 解析配置JSON并构建gRPC请求
	// 这里简化处理，实际应该解析JSON并填充MultiplexConfig字段
	req := &pb.MultiplexConfigRequest{
		AgentId:  agentID,
		Protocol: protocol,
		MultiplexConfig: &pb.MultiplexConfig{
			Enabled:        true,     // 从JSON解析
			Protocol:       "smux",   // 固定为smux
			MaxConnections: 4,        // 从JSON解析
			MinStreams:     4,        // 从JSON解析
			Padding:        false,    // 从JSON解析
			Brutal:         make(map[string]string), // 从JSON解析
		},
	}

	resp, err := client.UpdateMultiplexConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("调用Agent UpdateMultiplexConfig失败: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	return nil
}

// GetMultiplexConfig 获取Agent的多路复用配置
func (c *agentClient) GetMultiplexConfig(agentID, protocol string) (string, error) {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return "", err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req := &pb.MultiplexStatusRequest{
		AgentId:  agentID,
		Protocol: protocol,
	}

	resp, err := client.GetMultiplexConfig(ctx, req)
	if err != nil {
		return "", fmt.Errorf("调用Agent GetMultiplexConfig失败: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	// 将响应转换为JSON字符串返回
	// 这里简化处理，实际应该将resp.MultiplexConfigs序列化为JSON
	return fmt.Sprintf(`{"success": true, "configs": %v}`, resp.MultiplexConfigs), nil
}

// UpdateConfig 更新Agent配置
func (c *agentClient) UpdateConfig(agentID, configContent, configVersion string) error {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req := &pb.ConfigRequest{
		AgentId:       agentID,
		ConfigContent: configContent,
		ConfigVersion: configVersion,
		ForceUpdate:   false,
	}

	resp, err := client.UpdateConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("调用Agent UpdateConfig失败: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	return nil
}

// UpdateBlacklist 更新Agent黑名单
func (c *agentClient) UpdateBlacklist(agentID, protocol string, domains, ips, ports []string, operation string) error {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req := &pb.BlacklistRequest{
		AgentId:   agentID,
		Protocol:  protocol,
		Domains:   domains,
		Ips:       ips,
		Ports:     ports,
		Operation: operation,
	}

	resp, err := client.UpdateBlacklist(ctx, req)
	if err != nil {
		return fmt.Errorf("调用Agent UpdateBlacklist失败: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	return nil
}

// UpdateWhitelist 更新Agent白名单
func (c *agentClient) UpdateWhitelist(agentID, protocol string, domains, ips, ports []string, operation string) error {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req := &pb.WhitelistRequest{
		AgentId:   agentID,
		Protocol:  protocol,
		Domains:   domains,
		Ips:       ips,
		Ports:     ports,
		Operation: operation,
	}

	resp, err := client.UpdateWhitelist(ctx, req)
	if err != nil {
		return fmt.Errorf("调用Agent UpdateWhitelist失败: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	return nil
}

// RollbackConfig 回滚Agent配置
func (c *agentClient) RollbackConfig(agentID, targetVersion, reason string) error {
	conn, err := c.getConnection(agentID)
	if err != nil {
		return err
	}

	client := pb.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req := &pb.RollbackRequest{
		AgentId:       agentID,
		TargetVersion: targetVersion,
		Reason:        reason,
	}

	resp, err := client.RollbackConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("调用Agent RollbackConfig失败: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Agent返回错误: %s", resp.Message)
	}

	return nil
}

// Close 关闭所有连接
func (c *agentClient) Close() {
	for agentID, conn := range c.connections {
		if err := conn.Close(); err != nil {
			fmt.Printf("关闭Agent %s 连接失败: %v\n", agentID, err)
		}
	}
	c.connections = make(map[string]*grpc.ClientConn)
}