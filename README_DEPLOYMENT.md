# Xbox Sing-box管理系统 - 部署指南

本文档介绍如何使用Docker部署Xbox Sing-box管理系统。

## 系统架构

```
┌─────────────────┐    gRPC     ┌──────────────────┐    RESTful API    ┌─────────────────┐
│     Agent       │ ◄─────────► │   Controller     │ ◄──────────────── │  External       │
│   (sing-box)    │             │   (管理中心)      │                   │  Services       │
└─────────────────┘             └──────────────────┘                   └─────────────────┘
                                          │
                                          ▼
                                ┌──────────────────┐
                                │      MySQL       │
                                │   (数据存储)      │
                                └──────────────────┘
```

## 快速开始

### 1. 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少4GB可用内存
- 至少10GB可用磁盘空间

### 2. 克隆项目

```bash
git clone <repository-url>
cd xbox
```

### 3. 一键部署

```bash
# 赋予执行权限
chmod +x scripts/deploy.sh

# 完整安装
./scripts/deploy.sh install
```

### 4. 验证部署

部署完成后，访问以下地址验证服务状态：

- **Controller API**: http://localhost:8080/api/v1/health
- **Agent API**: http://localhost:8081/health
- **Prometheus**: http://localhost:9000
- **Grafana**: http://localhost:3000 (admin/xbox123456)

## 服务管理

### 启动服务

```bash
./scripts/deploy.sh start
```

### 停止服务

```bash
./scripts/deploy.sh stop
```

### 重启服务

```bash
./scripts/deploy.sh restart
```

### 查看状态

```bash
./scripts/deploy.sh status
```

### 查看日志

```bash
# 查看所有服务日志
./scripts/deploy.sh logs

# 查看特定服务日志
./scripts/deploy.sh logs controller
./scripts/deploy.sh logs agent
```

## 监控和运维

### 1. Prometheus指标

系统暴露的主要指标：

#### Controller指标
- `xbox_controller_agents_total` - 注册的Agent总数
- `xbox_controller_grpc_requests_total` - gRPC请求总数
- `xbox_controller_http_requests_total` - HTTP请求总数

#### Agent指标
- `xbox_agent_status` - Agent状态 (1=在线, 0=离线)
- `xbox_agent_heartbeat_total` - 心跳总数
- `xbox_system_cpu_usage_percent` - CPU使用率
- `xbox_system_memory_usage_bytes` - 内存使用量
- `xbox_singbox_status` - sing-box状态
- `xbox_singbox_connections_active` - 活跃连接数

### 2. 健康检查

#### Controller健康检查
```bash
curl http://localhost:8080/api/v1/health
```

#### Agent健康检查
```bash
curl http://localhost:8081/health
```

#### 详细健康检查
```bash
# Controller就绪检查
curl http://localhost:8080/api/v1/ready

# Agent存活检查
curl http://localhost:8081/live
```

### 3. 调试端点

#### 性能分析 (pprof)
```bash
# Controller
curl http://localhost:8080/debug/pprof/

# Agent
curl http://localhost:8081/debug/pprof/
```

#### 运行时信息
```bash
# Controller运行时信息
curl http://localhost:8080/debug/runtime

# Agent运行时信息
curl http://localhost:8081/debug/runtime
```

### 4. 日志管理

日志文件位置：
- Controller日志: `./logs/controller/`
- Agent日志: `./logs/agent/`

日志配置：
- 格式: JSON
- 轮转: 100MB/文件
- 保留: 10个备份文件
- 保留期: 30天

## 配置管理

### 1. 环境变量

主要环境变量配置：

```bash
# 数据库配置
XBOX_DATABASE_HOST=mysql
XBOX_DATABASE_PORT=3306
XBOX_DATABASE_USERNAME=xbox
XBOX_DATABASE_PASSWORD=xbox123456

# 服务配置
XBOX_GRPC_HOST=0.0.0.0
XBOX_GRPC_PORT=9090
XBOX_SERVER_HOST=0.0.0.0
XBOX_SERVER_PORT=8080

# Agent配置
XBOX_AGENT_CONTROLLER_ADDR=controller:9090
XBOX_AGENT_HEARTBEAT_INTERVAL=30
```

### 2. 配置文件

- Controller配置: `configs/config.yaml`
- Agent配置: `configs/agent.yaml`
- sing-box配置: `configs/sing-box.json`

## 数据备份

### 自动备份

```bash
./scripts/deploy.sh backup
```

备份包含：
- MySQL数据库
- 配置文件
- 数据目录

### 恢复数据

```bash
# 停止服务
./scripts/deploy.sh stop

# 恢复数据库
docker-compose exec mysql mysql -u root -pxbox123456 xbox_manager < backup.sql

# 启动服务
./scripts/deploy.sh start
```

## 安全考虑

### 1. 网络安全

- 使用内部Docker网络通信
- 仅暴露必要端口
- 支持TLS加密（可配置）

### 2. 访问控制

- 数据库用户权限限制
- API访问控制（可配置JWT）
- 容器以非root用户运行

### 3. 数据安全

- 数据库密码环境变量配置
- 敏感数据加密存储
- 定期安全更新

## 故障排除

### 1. 常见问题

#### 服务无法启动
```bash
# 检查容器状态
docker-compose ps

# 查看错误日志
docker-compose logs controller
docker-compose logs agent
```

#### 数据库连接失败
```bash
# 检查数据库状态
docker-compose exec mysql mysqladmin ping -u root -pxbox123456

# 重启数据库
docker-compose restart mysql
```

#### Agent无法连接Controller
```bash
# 检查网络连通性
docker-compose exec agent ping controller

# 检查gRPC端口
docker-compose exec agent telnet controller 9090
```

### 2. 性能优化

#### 数据库优化
- 调整MySQL配置参数
- 优化索引和查询
- 定期清理历史数据

#### 应用优化
- 调整心跳间隔
- 配置连接池大小
- 启用缓存机制

### 3. 监控告警

#### Grafana告警配置
1. 访问 http://localhost:3000
2. 导入监控面板
3. 配置告警规则
4. 设置通知渠道

#### 自定义告警
```yaml
# 在prometheus.yml中添加告警规则
- name: xbox_alerts
  rules:
  - alert: AgentDown
    expr: xbox_agent_status == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Agent {{ $labels.agent_id }} is down"
```

## 更新和维护

### 系统更新

```bash
# 更新到最新版本
./scripts/deploy.sh update
```

### 清理资源

```bash
# 清理未使用的资源
./scripts/deploy.sh cleanup
```

### 维护窗口

建议定期执行：
- 数据备份 (每日)
- 日志清理 (每周)
- 系统更新 (每月)
- 安全检查 (每月)

## API文档

### Controller API

详细API文档请参考：`docs/api/controller.md`

主要端点：
- `GET /api/v1/agents` - 获取Agent列表
- `POST /api/v1/configs` - 创建配置
- `PUT /api/v1/rules/{id}` - 更新规则

### Agent API

主要端点：
- `GET /health` - 健康检查
- `GET /metrics` - Prometheus指标
- `GET /debug/runtime` - 运行时信息

## 支持

如需技术支持，请：
1. 查看日志文件
2. 检查健康状态
3. 参考故障排除指南
4. 提交Issue（包含详细错误信息）