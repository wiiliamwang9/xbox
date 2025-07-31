# Xbox Sing-box管理系统 - 部署操作手册

本手册提供Xbox系统的完整部署指南，包括环境准备、安装部署、配置管理、监控运维等内容。

## 目录

- [环境准备](#环境准备)
- [快速部署](#快速部署)
- [详细部署步骤](#详细部署步骤)
- [配置管理](#配置管理)
- [服务管理](#服务管理)
- [监控运维](#监控运维)
- [备份恢复](#备份恢复)
- [故障排除](#故障排除)
- [升级维护](#升级维护)

## 环境准备

### 系统要求

#### 硬件配置

| 环境类型 | CPU | 内存 | 磁盘 | 网络 |
|---------|-----|------|------|------|
| 开发环境 | 2核+ | 4GB+ | 20GB+ | 1Mbps+ |
| 测试环境 | 4核+ | 8GB+ | 50GB+ | 10Mbps+ |
| 生产环境 | 8核+ | 16GB+ | 100GB+ | 100Mbps+ |

#### 操作系统

支持的操作系统：
- **Ubuntu**: 18.04+ / 20.04+ / 22.04+
- **CentOS**: 7+ / 8+
- **RHEL**: 7+ / 8+
- **Debian**: 9+ / 10+ / 11+

#### 软件依赖

| 软件 | 版本要求 | 安装方式 |
|------|----------|----------|
| Docker | 20.10+ | 官方脚本安装 |
| Docker Compose | 2.0+ | Docker插件安装 |
| Git | 2.0+ | 系统包管理器 |
| curl | 7.0+ | 系统包管理器 |

### 环境初始化

#### 安装Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# CentOS/RHEL
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io
sudo systemctl start docker
sudo systemctl enable docker
```

#### 安装Docker Compose

```bash
# 通过Docker CLI插件安装
sudo apt-get update
sudo apt-get install docker-compose-plugin

# 或使用pip安装
pip3 install docker-compose

# 验证安装
docker compose version
```

#### 系统优化

```bash
# 调整系统参数
echo 'vm.max_map_count=262144' | sudo tee -a /etc/sysctl.conf
echo 'fs.file-max=65536' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# 设置防火墙规则
sudo ufw allow 8080/tcp    # Controller HTTP
sudo ufw allow 9090/tcp    # Controller gRPC
sudo ufw allow 8081/tcp    # Agent HTTP
sudo ufw allow 9091/tcp    # Agent gRPC
sudo ufw allow 3000/tcp    # Grafana
sudo ufw allow 9000/tcp    # Prometheus
```

## 快速部署

### 一键部署脚本

```bash
# 1. 下载项目
git clone https://github.com/your-org/xbox.git
cd xbox

# 2. 执行一键部署
chmod +x scripts/deploy.sh
./scripts/deploy.sh install

# 3. 验证部署
./scripts/deploy.sh status
```

### 验证安装

```bash
# 检查服务状态
curl http://localhost:8080/api/v1/health
curl http://localhost:8081/health

# 检查监控服务
curl http://localhost:9000/-/healthy
curl http://localhost:3000/api/health
```

## 详细部署步骤

### 步骤1: 下载和准备

```bash
# 创建工作目录
sudo mkdir -p /opt/xbox
cd /opt/xbox

# 下载源码
git clone https://github.com/your-org/xbox.git .

# 创建必要目录
mkdir -p {logs,data,backups,configs}
```

### 步骤2: 配置环境变量

```bash
# 创建环境配置文件
cat > .env << EOF
# 数据库配置
MYSQL_ROOT_PASSWORD=xbox123456
MYSQL_DATABASE=xbox_manager
MYSQL_USER=xbox
MYSQL_PASSWORD=xbox123456

# 应用配置
XBOX_VERSION=1.0.0
XBOX_ENVIRONMENT=production

# 监控配置
GRAFANA_ADMIN_PASSWORD=xbox123456

# 网络配置
XBOX_SUBNET=172.20.0.0/16
EOF
```

### 步骤3: 自定义配置

#### Controller配置

```bash
# 复制并编辑Controller配置
cp configs/config.yaml configs/config.local.yaml
vim configs/config.local.yaml
```

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "mysql"
  port: 3306
  username: "xbox"
  password: "xbox123456"
  database: "xbox_manager"

grpc:
  host: "0.0.0.0"
  port: 9090
  tls:
    enabled: false

log:
  level: "info"
  format: "json"
  output: "both"
  file: "logs/controller.log"
```

#### Agent配置

```bash
# 复制并编辑Agent配置
cp configs/agent.yaml configs/agent.local.yaml
vim configs/agent.local.yaml
```

```yaml
agent:
  id: ""  # 自动生成
  controller_addr: "controller:9090"
  heartbeat_interval: 30
  singbox_config: "./configs/sing-box.json"
  singbox_binary: "sing-box"

server:
  host: "0.0.0.0"
  port: 8081

log:
  level: "info"
  format: "json"
  output: "both"
  file: "logs/agent.log"
```

### 步骤4: 构建镜像

```bash
# 构建Controller镜像
docker build -f Dockerfile.controller -t xbox/controller:latest .

# 构建Agent镜像
docker build -f Dockerfile.agent -t xbox/agent:latest .

# 验证镜像
docker images | grep xbox
```

### 步骤5: 启动服务

```bash
# 启动所有服务
docker compose up -d

# 查看启动状态
docker compose ps

# 查看日志
docker compose logs -f
```

### 步骤6: 数据库初始化

```bash
# 等待数据库启动
sleep 30

# 验证数据库连接
docker compose exec mysql mysql -u root -pxbox123456 -e "SHOW DATABASES;"

# 检查表结构
docker compose exec mysql mysql -u root -pxbox123456 xbox_manager -e "SHOW TABLES;"
```

## 配置管理

### 配置文件结构

```
configs/
├── config.yaml           # Controller配置模板
├── config.local.yaml     # Controller本地配置
├── agent.yaml            # Agent配置模板
├── agent.local.yaml      # Agent本地配置
├── sing-box.json         # sing-box配置模板
└── prometheus.yml        # Prometheus配置
```

### 配置热更新

```bash
# 更新Controller配置
vim configs/config.local.yaml
docker compose restart controller

# 更新Agent配置
vim configs/agent.local.yaml
docker compose restart agent

# 更新sing-box配置
vim configs/sing-box.json
# 通过API或Web界面推送配置
```

### 环境变量配置

支持的环境变量：

```bash
# 数据库配置
XBOX_DATABASE_HOST=mysql
XBOX_DATABASE_PORT=3306
XBOX_DATABASE_USERNAME=xbox
XBOX_DATABASE_PASSWORD=xbox123456
XBOX_DATABASE_DATABASE=xbox_manager

# 服务配置
XBOX_SERVER_HOST=0.0.0.0
XBOX_SERVER_PORT=8080
XBOX_GRPC_HOST=0.0.0.0
XBOX_GRPC_PORT=9090

# Agent配置
XBOX_AGENT_CONTROLLER_ADDR=controller:9090
XBOX_AGENT_HEARTBEAT_INTERVAL=30

# 日志配置
XBOX_LOG_LEVEL=info
XBOX_LOG_FORMAT=json
XBOX_LOG_OUTPUT=both
```

## 服务管理

### 基本操作

```bash
# 启动所有服务
./scripts/deploy.sh start

# 停止所有服务
./scripts/deploy.sh stop

# 重启所有服务
./scripts/deploy.sh restart

# 查看服务状态
./scripts/deploy.sh status

# 查看服务日志
./scripts/deploy.sh logs [service_name]
```

### 单独服务管理

```bash
# 重启单个服务
docker compose restart controller
docker compose restart agent
docker compose restart mysql

# 查看特定服务日志
docker compose logs -f controller
docker compose logs -f agent

# 进入容器调试
docker compose exec controller bash
docker compose exec agent bash
```

### 服务健康检查

```bash
# Controller健康检查
curl -f http://localhost:8080/api/v1/health

# Agent健康检查
curl -f http://localhost:8081/health

# 数据库健康检查
docker compose exec mysql mysqladmin ping -u root -pxbox123456

# 监控服务检查
curl -f http://localhost:9000/-/healthy
curl -f http://localhost:3000/api/health
```

## 监控运维

### Prometheus监控

#### 访问监控界面

- **Prometheus**: http://localhost:9000
- **Grafana**: http://localhost:3000 (admin/xbox123456)

#### 关键指标监控

```bash
# 查看所有指标
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics

# 查询特定指标
curl "http://localhost:9000/api/v1/query?query=xbox_agent_status"
curl "http://localhost:9000/api/v1/query?query=xbox_system_cpu_usage_percent"
```

#### 自定义告警规则

创建告警配置文件：

```yaml
# monitoring/alert_rules.yml
groups:
  - name: xbox_alerts
    rules:
      - alert: AgentDown
        expr: xbox_agent_status == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Agent {{ $labels.agent_id }} is down"
          description: "Agent has been down for more than 1 minute"

      - alert: HighCPUUsage
        expr: xbox_system_cpu_usage_percent > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage on {{ $labels.agent_id }}"
          
      - alert: HighMemoryUsage
        expr: xbox_system_memory_usage_bytes > 8GB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.agent_id }}"
```

### 日志管理

#### 日志配置

```yaml
# 日志轮转配置
log:
  level: "info"
  format: "json"
  output: "both"
  file: "logs/app.log"
  max_size: 100    # MB
  max_backups: 10
  max_age: 30      # days
```

#### 日志查看和分析

```bash
# 查看实时日志
docker compose logs -f controller
docker compose logs -f agent

# 查看历史日志
ls -la logs/
tail -f logs/controller.log
tail -f logs/agent.log

# 日志分析（使用jq解析JSON日志）
cat logs/controller.log | jq '.level="error"'
cat logs/agent.log | jq '.component="grpc"'
```

### 性能监控

#### 系统资源监控

```bash
# 容器资源使用情况
docker stats

# 磁盘使用情况
df -h
du -sh logs/ data/ backups/

# 网络连接状态
netstat -tlnp | grep -E ':(8080|8081|9090|3306)'
```

#### 应用性能分析

```bash
# pprof性能分析
go tool pprof http://localhost:8080/debug/pprof/profile
go tool pprof http://localhost:8081/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8081/debug/pprof/heap

# goroutine分析
curl http://localhost:8080/debug/pprof/goroutine?debug=1
curl http://localhost:8081/debug/pprof/goroutine?debug=1
```

## 备份恢复

### 自动备份

```bash
# 创建备份脚本
cat > scripts/backup_cron.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/xbox/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# 备份数据库
docker compose exec -T mysql mysqldump -u root -pxbox123456 xbox_manager > "$BACKUP_DIR/mysql_backup.sql"

# 备份配置文件
cp -r configs "$BACKUP_DIR/"

# 备份数据目录
if [ -d "data" ]; then
    cp -r data "$BACKUP_DIR/"
fi

# 压缩备份
tar -czf "$BACKUP_DIR.tar.gz" -C "$BACKUP_DIR" .
rm -rf "$BACKUP_DIR"

# 清理旧备份（保留7天）
find /opt/xbox/backups -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_DIR.tar.gz"
EOF

chmod +x scripts/backup_cron.sh
```

#### 设置定时备份

```bash
# 添加到crontab
echo "0 2 * * * /opt/xbox/scripts/backup_cron.sh" | crontab -

# 查看crontab
crontab -l
```

### 手动备份

```bash
# 使用部署脚本备份
./scripts/deploy.sh backup

# 手动数据库备份
docker compose exec mysql mysqldump -u root -pxbox123456 xbox_manager > backup_$(date +%Y%m%d).sql

# 备份配置和数据
tar -czf xbox_backup_$(date +%Y%m%d).tar.gz configs/ data/ logs/
```

### 数据恢复

#### 恢复数据库

```bash
# 停止服务
./scripts/deploy.sh stop

# 恢复数据库
docker compose up -d mysql
sleep 10
docker compose exec -T mysql mysql -u root -pxbox123456 xbox_manager < backup.sql

# 启动其他服务
./scripts/deploy.sh start
```

#### 恢复配置和数据

```bash
# 解压备份文件
tar -xzf xbox_backup_20240115.tar.gz

# 恢复配置文件
cp -r configs/* ./configs/

# 恢复数据文件
cp -r data/* ./data/

# 重启服务
./scripts/deploy.sh restart
```

## 故障排除

### 常见问题和解决方案

#### 1. 服务启动失败

**问题**: 容器无法启动或频繁重启

**排查步骤**:
```bash
# 查看容器状态
docker compose ps

# 查看详细错误日志
docker compose logs controller
docker compose logs agent

# 检查端口占用
netstat -tlnp | grep -E ':(8080|8081|9090|3306)'

# 检查磁盘空间
df -h

# 检查内存使用
free -h
```

**常见解决方法**:
```bash
# 释放端口占用
sudo fuser -k 8080/tcp
sudo fuser -k 8081/tcp

# 清理磁盘空间
docker system prune -f
docker volume prune -f

# 重启Docker服务
sudo systemctl restart docker
```

#### 2. 数据库连接失败

**问题**: 应用无法连接到MySQL数据库

**排查步骤**:
```bash
# 检查数据库容器状态
docker compose ps mysql

# 检查数据库连接
docker compose exec mysql mysql -u root -pxbox123456 -e "SELECT 1"

# 查看数据库日志
docker compose logs mysql

# 检查网络连通性
docker compose exec controller ping mysql
```

**解决方法**:
```bash
# 重启数据库
docker compose restart mysql

# 检查配置文件
cat configs/config.local.yaml | grep -A 10 database

# 重新初始化数据库
docker compose down -v
docker compose up -d mysql
# 等待数据库启动完成后启动其他服务
```

#### 3. Agent无法连接Controller

**问题**: Agent无法与Controller建立gRPC连接

**排查步骤**:
```bash
# 检查Controller gRPC端口
curl -v http://localhost:9090

# 检查网络连通性
docker compose exec agent ping controller

# 检查防火墙设置
sudo ufw status

# 查看Agent日志
docker compose logs agent | grep -i grpc
```

**解决方法**:
```bash
# 检查Controller配置
cat configs/config.local.yaml | grep -A 5 grpc

# 检查Agent配置
cat configs/agent.local.yaml | grep controller_addr

# 重启服务
docker compose restart controller agent
```

#### 4. 监控服务异常

**问题**: Prometheus或Grafana无法正常工作

**排查步骤**:
```bash
# 检查监控服务状态
curl http://localhost:9000/-/healthy
curl http://localhost:3000/api/health

# 检查配置文件
cat monitoring/prometheus.yml

# 查看日志
docker compose logs prometheus
docker compose logs grafana
```

**解决方法**:
```bash
# 重启监控服务
docker compose restart prometheus grafana

# 重新加载Prometheus配置
curl -X POST http://localhost:9000/-/reload

# 检查指标端点
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics
```

### 性能问题排查

#### 系统性能问题

```bash
# CPU使用率分析
top -p $(docker inspect --format='{{.State.Pid}}' xbox-controller)
top -p $(docker inspect --format='{{.State.Pid}}' xbox-agent)

# 内存使用分析
docker stats --no-stream

# 磁盘I/O分析
iostat -x 1 5

# 网络分析
netstat -i
iftop -i docker0
```

#### 应用性能分析

```bash
# Go应用性能分析
go tool pprof -http=:6060 http://localhost:8080/debug/pprof/profile
go tool pprof -http=:6061 http://localhost:8081/debug/pprof/profile

# 数据库性能分析
docker compose exec mysql mysql -u root -pxbox123456 -e "SHOW PROCESSLIST;"
docker compose exec mysql mysql -u root -pxbox123456 -e "SHOW STATUS LIKE 'Slow_queries';"
```

## 升级维护

### 版本升级

#### 准备升级

```bash
# 备份当前版本
./scripts/deploy.sh backup

# 拉取最新代码
git fetch origin
git checkout v1.1.0  # 或最新版本标签

# 检查变更日志
cat CHANGELOG.md
```

#### 执行升级

```bash
# 停止服务
./scripts/deploy.sh stop

# 重新构建镜像
docker build -f Dockerfile.controller -t xbox/controller:v1.1.0 .
docker build -f Dockerfile.agent -t xbox/agent:v1.1.0 .

# 更新镜像标签
docker tag xbox/controller:v1.1.0 xbox/controller:latest
docker tag xbox/agent:v1.1.0 xbox/agent:latest

# 启动服务
./scripts/deploy.sh start

# 验证升级
curl http://localhost:8080/api/v1/health
./scripts/deploy.sh status
```

#### 回滚版本

```bash
# 如果升级失败，回滚到备份版本
./scripts/deploy.sh stop

# 恢复备份
# ... 按照数据恢复步骤操作

# 使用旧版本镜像
docker tag xbox/controller:v1.0.0 xbox/controller:latest
docker tag xbox/agent:v1.0.0 xbox/agent:latest

./scripts/deploy.sh start
```

### 定期维护

#### 每日维护

```bash
# 检查服务状态
./scripts/deploy.sh status

# 查看系统资源使用
docker stats --no-stream

# 检查日志错误
grep -i error logs/*.log | tail -20

# 备份数据
./scripts/deploy.sh backup
```

#### 每周维护

```bash
# 清理Docker资源
docker system prune -f

# 清理旧日志
find logs/ -name "*.log*" -mtime +7 -delete

# 检查磁盘空间
df -h
du -sh logs/ data/ backups/

# 更新系统包
sudo apt update && sudo apt upgrade -y
```

#### 每月维护

```bash
# 数据库优化
docker compose exec mysql mysql -u root -pxbox123456 xbox_manager -e "OPTIMIZE TABLE agents, configs, rules;"

# 清理旧备份
find backups/ -name "*.tar.gz" -mtime +30 -delete

# 检查安全更新
sudo apt list --upgradable

# 性能基准测试
# 运行负载测试工具
```

### 扩容方案

#### 垂直扩容（增加资源）

```yaml
# docker-compose.yml
services:
  controller:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
          
  agent:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 2G
```

#### 水平扩容（多实例）

```bash
# 启动多个Agent实例
docker compose up -d --scale agent=3

# 使用负载均衡器
# 配置nginx或haproxy进行负载均衡
```

## 安全最佳实践

### 网络安全

```bash
# 配置防火墙
sudo ufw deny in on docker0
sudo ufw allow out on docker0

# 使用TLS加密
# 在生产环境中启用HTTPS和gRPC TLS

# 网络隔离
# 使用自定义Docker网络
docker network create xbox-network --driver bridge
```

### 访问控制

```bash
# 限制管理端口访问
sudo ufw allow from 10.0.0.0/8 to any port 8080
sudo ufw allow from 10.0.0.0/8 to any port 3000

# 配置反向代理
# 使用nginx或traefik作为反向代理
```

### 数据安全

```bash
# 数据加密
# 配置数据库加密
# 启用磁盘加密

# 密钥管理
# 使用Docker secrets或外部密钥管理系统

# 审计日志
# 启用操作审计日志
```

---

本部署手册涵盖了Xbox系统的完整部署和运维流程。如需更多技术支持，请参考项目文档或联系技术团队。