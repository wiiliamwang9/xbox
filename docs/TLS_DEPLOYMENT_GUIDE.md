# Xbox TLS + mTLS 部署指南

## 概述

本指南详细说明如何在Xbox Sing-box管理系统中部署和配置TLS + mTLS双向认证。通过本指南，您将能够建立一个安全的、企业级的分布式代理管理系统。

## 部署架构

### 安全通信模式
```
┌─────────────────────┐     TLS + mTLS      ┌─────────────────────┐
│                     │ ◄─────────────────► │                     │
│    Xbox Agent       │     证书双向认证      │  Xbox Controller    │
│                     │                     │                     │
│ ┌─────────────────┐ │                     │ ┌─────────────────┐ │
│ │  Client Cert    │ │                     │ │  Server Cert    │ │
│ │  Private Key    │ │                     │ │  Private Key    │ │
│ │  CA Cert        │ │                     │ │  CA Cert        │ │
│ └─────────────────┘ │                     │ └─────────────────┘ │
└─────────────────────┘                     └─────────────────────┘
                │                                       │
                └─────────────── CA Trust ──────────────┘
                        ┌─────────────────┐
                        │   CA Root       │
                        │   Certificate   │
                        └─────────────────┘
```

## 前置条件

### 系统要求
- Linux/macOS/Windows 系统
- Docker 20.10+ 和 Docker Compose 2.0+
- OpenSSL 1.1.1+ (证书生成)
- Go 1.23+ (如需从源码构建)

### 网络要求
- Controller: 监听端口 8080 (HTTP API), 9090 (gRPC TLS)
- Agent: 监听端口 8081 (HTTP API)
- 确保网络连通性和防火墙配置

## 步骤一：证书基础设施部署

### 1.1 生成证书
```bash
# 进入项目目录
cd xbox

# 运行证书生成脚本
./scripts/generate_tls_certs.sh
```

### 1.2 验证证书结构
```bash
# 检查生成的证书文件
tree certs/
# 输出应该是:
# certs/
# ├── ca/
# │   ├── ca-cert.pem      # CA根证书
# │   └── ca-key.pem       # CA私钥
# ├── server/
# │   ├── server-cert.pem  # Controller服务器证书
# │   └── server-key.pem   # Controller服务器私钥
# └── client/
#     ├── client-cert.pem  # Agent客户端证书
#     └── client-key.pem   # Agent客户端私钥
```

### 1.3 证书验证
```bash
# 验证证书链完整性
openssl verify -CAfile certs/ca/ca-cert.pem certs/server/server-cert.pem
openssl verify -CAfile certs/ca/ca-cert.pem certs/client/client-cert.pem

# 检查证书详细信息
openssl x509 -in certs/server/server-cert.pem -text -noout | grep -A 5 "Subject Alternative Name"
```

## 步骤二：配置文件设置

### 2.1 Controller配置 (configs/config.yaml)
```yaml
# gRPC服务配置
grpc:
  host: "0.0.0.0"
  port: 9090
  tls:
    enabled: true                              # 启用TLS + mTLS
    cert_file: "./certs/server/server-cert.pem"  # 服务器证书
    key_file: "./certs/server/server-key.pem"    # 服务器私钥
    ca_file: "./certs/ca/ca-cert.pem"           # CA根证书
    server_name: "xbox-controller"              # 服务器名称

# HTTP服务器配置
server:
  host: "0.0.0.0"
  port: 9000

# 数据库配置
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "xbox123456"
  database: "xbox_manager"
```

### 2.2 Agent配置 (configs/agent.yaml)
```yaml
# gRPC配置
grpc:
  host: "0.0.0.0" 
  port: 9091
  tls:
    enabled: true                              # 启用TLS + mTLS
    cert_file: "./certs/client/client-cert.pem"  # 客户端证书
    key_file: "./certs/client/client-key.pem"    # 客户端私钥
    ca_file: "./certs/ca/ca-cert.pem"           # CA根证书
    server_name: "xbox-controller"              # 服务器名称验证

# Agent配置
agent:
  id: ""                           # 自动生成
  controller_addr: "localhost:9090"  # Controller地址
  heartbeat_interval: 30
  singbox_config: "./configs/sing-box.json"
  singbox_binary: "sing-box"
```

## 步骤三：Docker部署

### 3.1 Docker Compose配置
确保docker-compose.yml包含正确的证书挂载:

```yaml
version: '3.8'

services:
  controller:
    build:
      context: .
      dockerfile: Dockerfile.controller
    ports:
      - "8080:8080"   # HTTP API
      - "9090:9090"   # gRPC TLS
    volumes:
      - ./configs/config.yaml:/app/config.yaml
      - ./certs:/app/certs:ro  # 只读挂载证书
    environment:
      - CONFIG_PATH=/app/config.yaml

  agent:
    build:
      context: .
      dockerfile: Dockerfile.agent
    ports:
      - "8081:8081"   # HTTP API
    volumes:
      - ./configs/agent.yaml:/app/agent.yaml
      - ./certs:/app/certs:ro  # 只读挂载证书
    environment:
      - CONFIG_PATH=/app/agent.yaml
    depends_on:
      - controller
```

### 3.2 部署执行
```bash
# 一键部署
./scripts/deploy.sh install

# 检查服务状态
./scripts/deploy.sh status
```

## 步骤四：安全验证

### 4.1 基础TLS测试
```bash
# 运行TLS配置验证
./simple_tls_test.sh

# 预期输出：
# ✓ 所有证书文件存在
# ✓ 证书链验证通过
# ✓ 私钥匹配性验证
# ✓ TLS连接测试成功
```

### 4.2 mTLS认证测试
```bash
# 运行完整的mTLS测试
./test_mtls_authentication.sh

# 该测试将：
# - 启动Controller (TLS模式)
# - 启动Agent (TLS模式)
# - 验证双向认证
# - 检查心跳通信
```

### 4.3 手动连接验证
```bash
# 测试gRPC TLS连接
grpcurl -insecure \
  -cert certs/client/client-cert.pem \
  -key certs/client/client-key.pem \
  -cacert certs/ca/ca-cert.pem \
  localhost:9090 \
  agent.AgentService/Ping
```

## 步骤五：监控和日志

### 5.1 安全事件监控
```bash
# 查看Controller TLS日志
docker-compose logs controller | grep -i tls

# 查看Agent连接日志
docker-compose logs agent | grep -i "connected\|tls"

# 检查证书加载日志
docker-compose logs controller | grep -i certificate
```

### 5.2 性能监控
```bash
# 检查TLS握手性能
curl http://localhost:8080/metrics | grep tls

# 监控连接状态
curl http://localhost:8081/metrics | grep grpc
```

## 步骤六：生产环境优化

### 6.1 证书管理
```bash
# 设置证书文件权限
chmod 600 certs/*/private-key.pem
chmod 644 certs/*/cert.pem

# 创建证书备份
tar -czf certs-backup-$(date +%Y%m%d).tar.gz certs/

# 设置证书过期监控
echo "0 0 * * * $(pwd)/scripts/check_cert_expiry.sh" | crontab -
```

### 6.2 防火墙配置
```bash
# 允许必要端口
sudo ufw allow 8080/tcp  # HTTP API
sudo ufw allow 9090/tcp  # gRPC TLS
sudo ufw allow 8081/tcp  # Agent API

# 阻止非必要端口
sudo ufw deny 3306/tcp   # 数据库 (仅内部访问)
```

### 6.3 高可用配置
```bash
# 多Controller部署时的负载均衡配置
# 在nginx.conf中配置:
upstream xbox_controllers {
    server controller1:9090;
    server controller2:9090;
    server controller3:9090;
}

server {
    listen 9090 ssl;
    ssl_certificate /path/to/load-balancer-cert.pem;
    ssl_certificate_key /path/to/load-balancer-key.pem;
    
    location / {
        grpc_pass grpc://xbox_controllers;
    }
}
```

## 故障排除

### 常见问题及解决方案

#### 1. 证书验证失败
```bash
# 问题：certificate verify failed
# 解决：检查证书链和CA配置
openssl verify -CAfile certs/ca/ca-cert.pem certs/server/server-cert.pem

# 检查时间同步
ntpdate -s time.nist.gov
```

#### 2. TLS握手失败
```bash
# 问题：tls: handshake failure
# 解决：检查证书配置和网络连通性

# 验证证书匹配性
openssl x509 -noout -modulus -in certs/server/server-cert.pem | openssl md5
openssl rsa -noout -modulus -in certs/server/server-key.pem | openssl md5

# 检查端口可达性
telnet localhost 9090
```

#### 3. 客户端认证失败
```bash
# 问题：tls: bad certificate
# 解决：检查客户端证书配置

# 验证客户端证书
openssl x509 -in certs/client/client-cert.pem -text -noout | grep "Key Usage"

# 检查证书用途
openssl x509 -in certs/client/client-cert.pem -noout -purpose
```

#### 4. 主机名验证失败
```bash
# 问题：certificate is not valid for localhost
# 解决：检查SAN配置或更新server_name配置

# 查看证书SAN
openssl x509 -in certs/server/server-cert.pem -text -noout | grep -A 5 "Subject Alternative Name"

# 重新生成支持所需主机名的证书
# 编辑 scripts/generate_tls_certs.sh 中的SAN配置
```

### 调试工具

#### TLS连接调试
```bash
# 使用OpenSSL测试连接
openssl s_client -connect localhost:9090 \
  -cert certs/client/client-cert.pem \
  -key certs/client/client-key.pem \
  -CAfile certs/ca/ca-cert.pem \
  -verify_return_error

# 详细TLS调试
openssl s_client -connect localhost:9090 \
  -cert certs/client/client-cert.pem \
  -key certs/client/client-key.pem \
  -CAfile certs/ca/ca-cert.pem \
  -debug -state -msg
```

#### 证书分析
```bash
# 证书详细信息
openssl x509 -in certs/server/server-cert.pem -text -noout

# 证书有效期检查
openssl x509 -in certs/server/server-cert.pem -noout -dates

# 证书指纹
openssl x509 -in certs/server/server-cert.pem -noout -fingerprint -sha256
```

## 安全最佳实践

### 1. 证书生命周期管理
- **定期轮换**: 证书每年更新一次
- **过期监控**: 设置30天过期告警
- **安全存储**: 私钥权限限制为600
- **备份策略**: 定期备份证书到安全位置

### 2. 网络安全
- **最小权限**: 只开放必要端口
- **网络隔离**: 使用VPC或内部网络
- **访问控制**: 基于IP地址的访问限制
- **监控审计**: 记录所有TLS连接事件

### 3. 运营安全
- **日志监控**: 实时监控TLS错误和异常
- **性能监控**: 监控TLS握手时间和连接数
- **安全扫描**: 定期进行安全漏洞扫描
- **事件响应**: 建立安全事件响应流程

### 4. 合规要求
- **数据保护**: 满足GDPR/CCPA要求
- **行业标准**: 符合SOC2/ISO27001标准
- **审计跟踪**: 完整的操作审计日志
- **文档管理**: 维护安全配置文档

## 总结

通过本指南，您已经成功部署了企业级的TLS + mTLS安全架构。这个配置提供了：

- **强身份认证**: 双向证书验证
- **数据加密**: 端到端TLS加密
- **完整性保护**: 防止数据篡改
- **监控能力**: 全面的安全监控

确保定期维护证书基础设施，监控安全事件，并遵循最佳实践以维护系统安全性。