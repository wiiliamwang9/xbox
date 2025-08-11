# Xbox Sing-box管理系统 - 节点部署指南

本文档介绍如何直接在节点上部署Xbox Sing-box管理系统，无需Docker容器化。

## 系统架构

```
┌─────────────────┐    gRPC     ┌──────────────────┐    RESTful API    ┌─────────────────┐
│     Agent       │ ◄─────────► │   Controller     │ ◄──────────────── │  External       │
│   (sing-box)    │             │   (管理中心)      │                   │  Services       │
└─────────────────┘             └──────────────────┘                   └─────────────────┘
                                          │
                                          ▼
                                ┌──────────────────┐
                                │     SQLite       │
                                │   (本地存储)      │
                                └──────────────────┘
```

## 快速开始

### 1. 环境要求

#### 系统要求
- **操作系统**: Linux (推荐 Ubuntu 20.04+, CentOS 7+)
- **内存**: 最少 2GB RAM
- **磁盘**: 最少 5GB 可用空间
- **网络**: 能够访问互联网（用于下载 sing-box）

#### 软件依赖
- **Go**: 1.21+ （必需）
- **protoc**: Protocol Buffers 编译器（可选，用于开发）
- **curl**: 用于健康检查（通常系统自带）

### 2. 克隆项目

```bash
git clone <repository-url>
cd xbox
```

### 3. 一键部署

```bash
# 赋予执行权限
chmod +x scripts/deploy.sh

# 完整安装（自动检查依赖、构建、配置、启动）
./scripts/deploy.sh install
```

### 4. 验证部署

部署完成后，验证服务状态：

```bash
# 检查服务状态
./scripts/deploy.sh status

# 验证API端点
curl http://localhost:8080/api/v1/health
```

**预期输出**:
- Controller gRPC端口 (9090) 正常监听
- Controller HTTP端口 (8080) 正常监听  
- Agent 正在运行
- Controller健康检查通过

## 详细部署步骤

### 步骤 1: 环境准备

#### 安装Go (如果未安装)
```bash
# Ubuntu/Debian
wget https://golang.org/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# CentOS/RHEL
wget https://golang.org/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bash_profile
source ~/.bash_profile
```

#### 验证Go安装
```bash
go version
# 预期输出: go version go1.21.x linux/amd64
```

### 步骤 2: 构建应用

```bash
# 生成protobuf代码（如果有protoc）
make proto

# 安装Go依赖
go mod tidy

# 构建应用程序
make build

# 或者使用部署脚本
./scripts/deploy.sh build
```

### 步骤 3: 配置管理

系统会自动生成默认配置文件：

#### Controller配置 (`configs/config.yaml`)
```yaml
server:
  grpc:
    bind: "0.0.0.0"
    port: 9090
  http:
    bind: "0.0.0.0" 
    port: 8080

database:
  type: "sqlite"
  dsn: "./data/controller.db"
  auto_migrate: true

log:
  level: "info"
  output: "./logs/controller.log"

agent:
  heartbeat_timeout: 90
  cleanup_interval: 300

monitoring:
  metrics_port: 9001
  health_check_port: 8081
```

#### Agent配置 (`configs/agent.yaml`)  
```yaml
agent:
  id: "agent-hostname-timestamp"
  controller_addr: "localhost:9090"
  heartbeat_interval: 30
  sing_box_binary: "sing-box"
  sing_box_config: "./configs/sing-box.json"

log:
  level: "info"
  output: "./logs/agent.log"
```

### 步骤 4: 启动服务

```bash
# 启动所有服务
./scripts/deploy.sh start

# 或者分别启动
./scripts/deploy.sh start-controller
./scripts/deploy.sh start-agent
```

## 服务管理

### 基本命令

```bash
# 启动所有服务
./scripts/deploy.sh start

# 停止所有服务  
./scripts/deploy.sh stop

# 重启所有服务
./scripts/deploy.sh restart

# 查看服务状态
./scripts/deploy.sh status

# 查看日志
./scripts/deploy.sh logs [controller|agent|all]
```

### 高级管理

```bash
# 只构建应用程序
./scripts/deploy.sh build

# 查看特定服务日志
./scripts/deploy.sh logs controller
./scripts/deploy.sh logs agent

# 实时监控日志
tail -f logs/controller.log
tail -f logs/agent.log
```

## 远程Agent部署

### 单个Agent部署

```bash
# 部署Agent到远程节点
./scripts/deploy_agent.sh <远程IP> <SSH密码> [Controller地址]

# 示例
./scripts/deploy_agent.sh 192.168.1.100 your_password 192.168.1.10:9090
```

### 批量Agent部署

```bash
# 创建节点列表文件
cat > nodes.txt << EOF
192.168.1.101 password1
192.168.1.102 password2  
192.168.1.103 password3
EOF

# 批量部署
while read ip password; do
  ./scripts/deploy_agent.sh $ip $password
done < nodes.txt
```

## 监控和运维

### 1. 健康检查

#### 基本健康检查
```bash
# Controller健康检查
curl http://localhost:8080/api/v1/health

# Controller就绪检查
curl http://localhost:8080/api/v1/ready

# 检查所有Agent
curl http://localhost:8080/api/v1/agents
```

#### 系统状态检查
```bash
# 检查端口监听
netstat -tlnp | grep -E ":(8080|9090) "

# 检查进程状态
ps aux | grep -E "(controller|agent)" | grep -v grep

# 检查日志错误
grep -i error logs/*.log
```

### 2. 性能监控

#### 系统资源监控
```bash
# CPU和内存使用
top -p $(cat logs/controller.pid) -p $(cat logs/agent.pid)

# 磁盘使用
df -h
du -sh logs/ data/

# 网络连接
ss -tlnp | grep -E ":(8080|9090|1080|8888) "
```

#### 应用指标
```bash
# Controller运行时信息
curl http://localhost:8080/debug/runtime

# Agent运行时信息（如果启用）
curl http://localhost:8081/debug/runtime

# 检查数据库大小
ls -lh data/controller.db
```

### 3. 日志管理

#### 日志文件结构
```
logs/
├── controller.log    # Controller主日志
├── agent.log         # Agent主日志
├── controller.pid    # Controller进程ID
└── agent.pid         # Agent进程ID
```

#### 日志查看和分析
```bash
# 查看最新日志
tail -50 logs/controller.log
tail -50 logs/agent.log

# 实时监控日志
tail -f logs/controller.log | grep -E "(ERROR|WARN)"

# 搜索特定错误
grep -n "error" logs/*.log
grep -n "failed" logs/*.log

# 按时间范围查看日志
grep "2024-01-01" logs/controller.log
```

#### 日志轮转配置
```bash
# 创建logrotate配置
sudo tee /etc/logrotate.d/xbox << EOF
/path/to/xbox/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    sharedscripts
    postrotate
        # 重启服务以重新打开日志文件
        /path/to/xbox/scripts/deploy.sh restart
    endscript
}
EOF
```

## 配置管理

### 1. Controller配置调优

#### 数据库配置
```yaml
database:
  type: "sqlite"
  dsn: "./data/controller.db"
  auto_migrate: true
  # 性能调优
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: "1h"
```

#### 网络配置
```yaml
server:
  grpc:
    bind: "0.0.0.0"
    port: 9090
    # gRPC配置
    max_recv_msg_size: "4MB"
    max_send_msg_size: "4MB"
  http:
    bind: "0.0.0.0"
    port: 8080
    # HTTP超时配置  
    read_timeout: "30s"
    write_timeout: "30s"
```

### 2. Agent配置调优

#### 心跳配置
```yaml
agent:
  heartbeat_interval: 30     # 心跳间隔（秒）
  retry_interval: 5          # 重连间隔（秒）
  max_retry_attempts: 3      # 最大重试次数
```

#### sing-box配置
```yaml
agent:
  sing_box_binary: "sing-box"
  sing_box_config: "./configs/sing-box.json"
  auto_restart: true         # 自动重启sing-box
  restart_delay: "10s"       # 重启延迟
```

### 3. sing-box配置

#### 代理配置示例
```json
{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "inbounds": [
    {
      "type": "socks",
      "tag": "socks",
      "listen": "0.0.0.0",
      "listen_port": 1080
    },
    {
      "type": "http", 
      "tag": "http",
      "listen": "0.0.0.0",
      "listen_port": 8888
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

## 故障排除

### 1. 常见问题

#### 服务启动失败
```bash
# 检查二进制文件
ls -la bin/
file bin/controller bin/agent

# 检查配置文件语法
go run -c configs/config.yaml
go run -c configs/agent.yaml

# 检查端口占用
netstat -tlnp | grep -E ":(8080|9090) "
```

#### Agent无法连接Controller
```bash
# 检查网络连通性
ping controller_host
telnet controller_host 9090

# 检查防火墙
sudo iptables -L | grep 9090
sudo ufw status | grep 9090

# 检查Controller状态
curl http://controller_host:8080/api/v1/health
```

#### sing-box相关问题
```bash
# 检查sing-box安装
which sing-box
sing-box version

# 测试sing-box配置
sing-box check -c configs/sing-box.json

# 查看sing-box日志
grep "sing-box" logs/agent.log
```

### 2. 性能问题

#### 高CPU使用率
```bash
# 查看进程CPU使用
top -p $(cat logs/controller.pid)

# 使用pprof分析
go tool pprof http://localhost:8080/debug/pprof/profile
```

#### 内存泄漏
```bash
# 查看内存使用
ps -o pid,ppid,rss,vsz,cmd -p $(cat logs/controller.pid)

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap
```

#### 连接问题
```bash
# 查看连接数
ss -an | grep -E ":(8080|9090) " | wc -l

# 查看连接状态
ss -tuln | grep -E ":(8080|9090) "
```

### 3. 诊断工具

#### 系统诊断脚本
```bash
#!/bin/bash
echo "=== Xbox系统诊断 ==="
echo "时间: $(date)"
echo ""

echo "=== 系统信息 ==="
uname -a
free -h
df -h

echo ""
echo "=== 服务状态 ==="  
./scripts/deploy.sh status

echo ""
echo "=== 端口监听 ==="
netstat -tlnp | grep -E ":(8080|9090|1080|8888) "

echo ""
echo "=== 最近错误 ==="
grep -i error logs/*.log | tail -10
```

## 安全考虑

### 1. 网络安全

#### 防火墙配置
```bash
# Ubuntu/Debian (ufw)
sudo ufw allow 8080/tcp   # Controller HTTP API
sudo ufw allow 9090/tcp   # Controller gRPC
sudo ufw allow 1080/tcp   # SOCKS5代理
sudo ufw allow 8888/tcp   # HTTP代理

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=9090/tcp
sudo firewall-cmd --permanent --add-port=1080/tcp
sudo firewall-cmd --permanent --add-port=8888/tcp
sudo firewall-cmd --reload
```

#### SSL/TLS配置 (可选)
```yaml
server:
  grpc:
    tls:
      enabled: true
      cert_file: "certs/server.crt" 
      key_file: "certs/server.key"
  http:
    tls:
      enabled: true
      cert_file: "certs/server.crt"
      key_file: "certs/server.key"
```

### 2. 访问控制

#### API访问限制
```yaml
server:
  http:
    allowed_hosts:
      - "localhost"
      - "127.0.0.1"  
      - "your-domain.com"
    rate_limit:
      requests_per_minute: 100
```

#### 认证配置 (可选)
```yaml
auth:
  enabled: true
  jwt_secret: "your-secret-key"
  token_expire: "24h"
```

### 3. 数据安全

#### 数据库加密
```yaml
database:
  type: "sqlite"
  dsn: "./data/controller.db"
  encryption_key: "your-encryption-key"
```

#### 敏感信息保护
```bash
# 设置适当的文件权限
chmod 600 configs/*.yaml
chmod 700 data/
chmod 640 logs/*.log
```

## 维护和更新

### 1. 定期维护任务

#### 日志清理
```bash
# 清理30天前的日志
find logs/ -name "*.log" -mtime +30 -delete

# 压缩历史日志
gzip logs/*.log.old
```

#### 数据库维护
```bash
# SQLite数据库优化
sqlite3 data/controller.db "VACUUM;"
sqlite3 data/controller.db "ANALYZE;"

# 检查数据库大小
ls -lh data/controller.db
```

#### 系统资源检查
```bash
# 检查磁盘使用
df -h
du -sh logs/ data/

# 检查内存使用
free -h

# 清理临时文件
rm -rf /tmp/xbox-*
```

### 2. 系统更新

#### 更新应用程序
```bash
# 停止服务
./scripts/deploy.sh stop

# 更新代码
git pull origin main

# 重新构建
./scripts/deploy.sh build

# 启动服务
./scripts/deploy.sh start
```

#### 配置迁移
```bash
# 备份现有配置
cp -r configs/ configs.backup/

# 更新配置（如需要）
# 编辑 configs/*.yaml

# 验证配置
./scripts/deploy.sh build
```

### 3. 备份和恢复

#### 数据备份
```bash
#!/bin/bash
BACKUP_DIR="backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# 备份数据库
cp data/controller.db $BACKUP_DIR/

# 备份配置
cp -r configs/ $BACKUP_DIR/

# 备份日志
cp -r logs/ $BACKUP_DIR/

echo "备份完成: $BACKUP_DIR"
```

#### 数据恢复
```bash
# 停止服务
./scripts/deploy.sh stop

# 恢复数据
BACKUP_DIR="backup/20240101_120000"
cp $BACKUP_DIR/controller.db data/
cp -r $BACKUP_DIR/configs/* configs/

# 启动服务
./scripts/deploy.sh start
```

## API文档

### Controller API端点

#### 健康检查
- `GET /api/v1/health` - 基本健康检查
- `GET /api/v1/ready` - 就绪状态检查

#### Agent管理
- `GET /api/v1/agents` - 获取Agent列表
- `GET /api/v1/agents/{id}` - 获取特定Agent信息
- `DELETE /api/v1/agents/{id}` - 删除Agent

#### 配置管理
- `POST /api/v1/configs` - 创建配置
- `PUT /api/v1/configs/{id}` - 更新配置
- `GET /api/v1/configs/{id}` - 获取配置

#### 过滤器管理
- `POST /api/v1/filter/blacklist` - 更新黑名单规则
- `POST /api/v1/filter/whitelist` - 更新白名单规则
- `GET /api/v1/filter/config/{agent_id}` - 获取Agent过滤器配置

### 使用示例

```bash
# 获取所有Agent
curl http://localhost:8080/api/v1/agents

# 检查系统健康状态
curl http://localhost:8080/api/v1/health

# 获取运行时信息
curl http://localhost:8080/debug/runtime
```

## 支持和帮助

### 获取帮助

```bash
# 查看部署脚本帮助
./scripts/deploy.sh help

# 查看测试脚本
ls -la test_*.sh

# 运行系统测试
./test_agent_startup.sh
```

### 故障报告

提交问题时，请提供以下信息：

1. **系统信息**
   ```bash
   uname -a
   go version
   ./scripts/deploy.sh status
   ```

2. **错误日志**
   ```bash
   ./scripts/deploy.sh logs
   grep -i error logs/*.log | tail -20
   ```

3. **配置文件**
   ```bash
   cat configs/config.yaml
   cat configs/agent.yaml
   ```

4. **网络状态**
   ```bash
   netstat -tlnp | grep -E ":(8080|9090) "
   ```

### 常见问题FAQ

**Q: Controller启动失败，显示端口被占用？**
A: 检查端口占用情况，停止占用进程或修改配置端口。

**Q: Agent无法连接到Controller？**  
A: 检查网络连通性、防火墙设置和Controller地址配置。

**Q: sing-box下载失败？**
A: 检查网络连接，或手动下载sing-box二进制文件到PATH目录。

**Q: 如何查看详细的错误日志？**
A: 使用 `./scripts/deploy.sh logs` 查看日志，或直接查看 `logs/` 目录下的文件。

---

更多技术支持，请参考项目文档或提交Issue。