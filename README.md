# Xbox Sing-box管理系统

企业级分布式sing-box节点管理系统，提供统一的节点管理、配置下发、监控告警和自动化运维能力。

## 🌟 核心特性

### 节点管理
- **多节点统一管理**: 支持大规模sing-box节点集群
- **自动注册发现**: Agent自动注册到Controller
- **状态实时监控**: 节点在线状态、资源使用率监控
- **故障自动恢复**: 智能故障检测和自动恢复机制

### 配置管理
- **远程配置下发**: 统一配置管理和分发
- **配置版本控制**: 支持配置版本管理和回滚
- **热更新**: 无服务中断的配置更新
- **配置验证**: 自动配置语法验证

### 规则管理
- **动态规则管理**: 实时添加、修改、删除路由规则
- **规则优先级**: 支持规则优先级排序
- **批量操作**: 支持规则批量导入导出
- **规则模板**: 预定义规则模板

### 监控告警
- **多维度监控**: 系统、应用、业务三层监控
- **Prometheus集成**: 完整的指标收集和存储
- **Grafana可视化**: 丰富的监控仪表板
- **智能告警**: 基于阈值和趋势的告警机制

### 高可用性
- **容器化部署**: Docker和Docker Compose支持
- **健康检查**: 多层次健康状态检测
- **自动恢复**: 服务异常自动重启和恢复
- **数据备份**: 自动数据备份和恢复

### 🔒 企业级安全
- **TLS + mTLS双向认证**: 所有gRPC通信采用证书双向认证
- **端到端加密**: TLS 1.2/1.3 强加密保护数据传输
- **身份验证**: X.509证书验证防止非法访问
- **安全证书管理**: PKI证书基础设施和自动化工具

## 🏗 系统架构

```
                            ┌─────────────────────────────────────────────────────────┐
                            │                    监控告警层                           │
                            │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
                            │  │ Prometheus  │  │   Grafana   │  │  AlertManager   │ │
                            │  │   (指标)    │  │   (可视化)  │  │     (告警)      │ │
                            │  └─────────────┘  └─────────────┘  └─────────────────┘ │
                            └─────────────────────────────────────────────────────────┘
                                                        │
                            ┌─────────────────────────────────────────────────────────┐
                            │                     应用层                             │
                            │                                                         │
┌─────────────────┐ TLS+mTLS │  ┌──────────────────┐   RESTful API  ┌─────────────────┐ │
│     Agent       │ ◄─────────► │   Controller     │ ◄──────────────│  External       │ │
│   (sing-box)    │  gRPC+证书  │  │   (管理中心)      │                │  Services       │ │
│                 │         │  │                  │                │                 │ │
│ ┌─────────────┐ │         │  │ ┌──────────────┐ │                └─────────────────┘ │
│ │ 监控指标    │ │         │  │ │ gRPC服务     │ │                                      │
│ │ 健康检查    │ │         │  │ │ HTTP API     │ │                                      │
│ │ 自动恢复    │ │         │  │ │ 业务逻辑     │ │                                      │
│ └─────────────┘ │         │  │ └──────────────┘ │                                      │
└─────────────────┘         │  └──────────────────┘                                      │
                            └─────────────────────────────────────────────────────────┘
                                                        │
                            ┌─────────────────────────────────────────────────────────┐
                            │                    数据存储层                           │
                            │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
                            │  │    MySQL    │  │    Redis    │  │   File Storage  │ │
                            │  │  (关系数据)  │  │   (缓存)    │  │    (日志配置)   │ │
                            │  └─────────────┘  └─────────────┘  └─────────────────┘ │
                            └─────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| Docker | 20.10+ | 容器运行时 |
| Docker Compose | 2.0+ | 容器编排 |
| OpenSSL | 1.1.1+ | TLS证书生成 |
| 内存 | 4GB+ | 推荐配置 |
| 磁盘 | 10GB+ | 数据存储空间 |

### 一键部署

```bash
# 1. 下载项目
git clone https://github.com/your-org/xbox.git
cd xbox

# 2. 生成TLS证书（首次部署必需）
chmod +x scripts/generate_tls_certs.sh
./scripts/generate_tls_certs.sh

# 3. 一键安装部署
chmod +x scripts/deploy.sh
./scripts/deploy.sh install
```

### 验证部署

部署完成后，系统将自动启动所有服务：

```bash
# 检查服务状态  
./scripts/deploy.sh status

# 验证服务可用性
curl http://localhost:8080/api/v1/health    # Controller健康检查
curl http://localhost:8081/health           # Agent健康检查

# 验证TLS证书配置
./simple_tls_test.sh                        # TLS配置验证
```

### 访问系统

| 服务 | 地址 | 用途 | 认证 |
|------|------|------|------|
| Controller API | http://localhost:8080 | 管理接口 | - |
| Agent API | http://localhost:8081 | 代理接口 | - |
| Prometheus | http://localhost:9000 | 监控指标 | - |  
| Grafana | http://localhost:3000 | 可视化面板 | admin/xbox123456 |

### 开发环境搭建

如需进行开发，可以按以下步骤搭建开发环境：

```bash
# 1. 安装Go环境
# Go 1.21+

# 2. 安装依赖
go mod tidy

# 3. 生成协议文件
make proto

# 4. 构建应用
make build

# 5. 本地运行
make run-controller  # 启动Controller
make run-agent      # 启动Agent
```

## 📁 项目结构

```
xbox/
├── cmd/                           # 应用入口点
│   ├── agent/main.go             #   Agent启动程序
│   └── controller/main.go        #   Controller启动程序
├── internal/                      # 内部业务逻辑
│   ├── agent/                    #   Agent核心模块
│   │   ├── grpc/client.go        #     gRPC客户端
│   │   ├── monitor/system.go     #     系统监控
│   │   └── singbox/manager.go    #     sing-box管理
│   ├── controller/               #   Controller核心模块
│   │   ├── grpc/                 #     gRPC服务端
│   │   ├── service/              #     业务逻辑层
│   │   └── repository/           #     数据访问层
│   ├── config/config.go          #   配置管理
│   ├── database/database.go      #   数据库连接
│   ├── health/health.go          #   健康检查
│   ├── monitoring/               #   监控模块
│   │   ├── prometheus.go         #     Prometheus指标
│   │   └── metrics.go            #     指标收集
│   ├── recovery/recovery.go      #   故障恢复
│   ├── debug/debug.go           #   调试工具
│   └── models/models.go         #   数据模型
├── pkg/                          # 共享工具包
│   └── logger/logger.go         #   日志工具
├── proto/                        # gRPC协议定义
│   └── agent.proto              #   Agent服务协议
├── api/                          # RESTful API定义
├── configs/                      # 配置文件
│   ├── config.yaml              #   Controller配置
│   ├── agent.yaml               #   Agent配置
│   └── sing-box.json            #   sing-box配置模板
├── certs/                        # TLS证书目录
│   ├── ca/ca-cert.pem           #   CA根证书
│   ├── server/server-cert.pem   #   Controller服务器证书
│   └── client/client-cert.pem   #   Agent客户端证书
├── scripts/                      # 部署和工具脚本
│   ├── deploy.sh                #   一键部署脚本
│   ├── generate_tls_certs.sh    #   TLS证书生成脚本
│   └── init.sql                 #   数据库初始化
├── monitoring/                   # 监控配置
│   └── prometheus.yml           #   Prometheus配置
├── docs/                         # 项目文档
│   ├── development-plan.md      #   开发计划
│   └── api/                     #   API文档
├── Dockerfile.controller         # Controller镜像
├── Dockerfile.agent             # Agent镜像
├── docker-compose.yml           # 容器编排
└── README.md                    # 项目说明
```

## 📖 使用指南

### 服务管理

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

### 监控和调试

```bash
# 查看系统健康状态
curl http://localhost:8080/api/v1/health
curl http://localhost:8081/health

# 查看Prometheus指标
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics

# 查看运行时信息
curl http://localhost:8080/debug/runtime
curl http://localhost:8081/debug/runtime

# 性能分析
curl http://localhost:8080/debug/pprof/
curl http://localhost:8081/debug/pprof/
```

### 数据备份和恢复

```bash
# 创建数据备份
./scripts/deploy.sh backup

# 查看备份文件
ls -la backups/

# 恢复数据（需要先停止服务）
./scripts/deploy.sh stop
# 手动恢复数据库和配置文件
./scripts/deploy.sh start
```

## 🔧 运维管理

### 日志管理

系统采用结构化日志，支持日志轮转和压缩：

- **日志位置**: `./logs/` 目录
- **日志格式**: JSON格式，便于搜索分析
- **轮转策略**: 单文件100MB，保留10个备份
- **保留期限**: 30天自动清理

### 监控指标

系统提供丰富的监控指标：

#### 系统指标
- `xbox_system_cpu_usage_percent` - CPU使用率
- `xbox_system_memory_usage_bytes` - 内存使用量  
- `xbox_system_disk_usage_percent` - 磁盘使用率

#### 应用指标
- `xbox_agent_status` - Agent在线状态
- `xbox_agent_heartbeat_total` - 心跳总数
- `xbox_singbox_status` - sing-box运行状态
- `xbox_singbox_connections_active` - 活跃连接数

#### 业务指标
- `xbox_config_updates_total` - 配置更新次数
- `xbox_rule_updates_total` - 规则更新次数
- `xbox_errors_total` - 错误统计

### 告警配置

系统预配置了以下告警规则：

- **Agent离线告警**: Agent超过1分钟未心跳
- **CPU使用率告警**: CPU使用率超过80%持续5分钟  
- **内存使用率告警**: 内存使用率超过90%持续5分钟
- **sing-box异常告警**: sing-box进程停止运行

## 🔗 API文档

### Controller API

#### 节点管理
- `GET /api/v1/agents` - 获取节点列表
- `GET /api/v1/agents/{id}` - 获取节点详情  
- `PUT /api/v1/agents/{id}` - 更新节点信息
- `DELETE /api/v1/agents/{id}` - 删除节点

#### 配置管理  
- `POST /api/v1/configs` - 创建配置
- `GET /api/v1/configs/{agent_id}` - 获取节点配置
- `PUT /api/v1/configs/{id}` - 更新配置
- `DELETE /api/v1/configs/{id}` - 删除配置

#### 规则管理
- `POST /api/v1/rules` - 创建规则
- `GET /api/v1/rules` - 获取规则列表
- `PUT /api/v1/rules/{id}` - 更新规则  
- `DELETE /api/v1/rules/{id}` - 删除规则

#### 系统接口
- `GET /api/v1/health` - 健康检查
- `GET /api/v1/ready` - 就绪检查
- `GET /metrics` - Prometheus指标
- `GET /debug/runtime` - 运行时信息

### Agent API

- `GET /health` - 健康检查
- `GET /live` - 存活检查
- `GET /metrics` - Prometheus指标
- `GET /debug/runtime` - 运行时信息
- `GET /debug/pprof/` - 性能分析

## ⚠️ 故障排除

### 常见问题

#### 1. 服务启动失败
```bash
# 检查Docker状态
docker --version
docker-compose --version

# 检查端口占用
netstat -tlnp | grep -E ':(8080|8081|9000|3000|3306)'

# 查看错误日志
./scripts/deploy.sh logs
```

#### 2. Agent无法连接Controller
```bash
# 检查网络连通性
docker-compose exec agent ping controller

# 检查gRPC端口
docker-compose exec agent telnet controller 9090

# 检查防火墙设置
systemctl status firewalld
```

#### 3. 数据库连接失败
```bash
# 检查数据库状态
docker-compose exec mysql mysqladmin ping -u root -pxbox123456

# 重启数据库
docker-compose restart mysql

# 查看数据库日志
docker-compose logs mysql
```

#### 4. 监控数据异常
```bash
# 检查Prometheus连接
curl http://localhost:9000/api/v1/targets

# 检查指标端点
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics

# 重启监控服务
docker-compose restart prometheus grafana
```

### 性能优化

#### 系统级优化
- 调整Docker内存限制
- 优化数据库连接池配置
- 启用Redis缓存
- 配置负载均衡

#### 应用级优化
- 调整心跳间隔
- 优化监控数据收集频率
- 配置日志轮转策略
- 启用gRPC连接复用

### 🔒 安全架构

#### TLS + mTLS双向认证
- **端到端加密**: 所有gRPC通信采用TLS 1.2/1.3加密
- **双向身份验证**: Controller和Agent使用X.509证书相互认证
- **证书管理**: PKI基础设施，支持证书生成、验证和轮换
- **防中间人攻击**: 通过证书链验证和主机名检查

#### 证书基础设施
```bash
# 生成证书基础设施
./scripts/generate_tls_certs.sh

# 验证TLS配置
./simple_tls_test.sh

# 测试mTLS认证
./test_mtls_authentication.sh
```

#### 安全特性
- **网络隔离**: Docker内部网络 + TLS加密
- **身份验证**: 基于证书的强身份认证
- **访问控制**: 只允许有效证书的连接
- **数据保护**: 传输中数据全程加密
- **审计能力**: 证书和连接事件日志
- **权限管理**: 最小权限原则

## 🤝 贡献指南

欢迎为项目贡献代码！

### 开发流程
1. Fork项目
2. 创建功能分支
3. 提交代码更改
4. 通过测试
5. 提交Pull Request

### 代码规范
- 遵循Go代码规范
- 添加单元测试
- 更新文档
- 添加变更日志

### 提交规范
```
feat: 添加新功能
fix: 修复bug
docs: 更新文档  
style: 代码格式调整
refactor: 代码重构
test: 添加测试
chore: 构建工具变更
```

## 📞 技术支持

### 获取帮助
- 📖 [项目文档](docs/)
- 🐛 [问题反馈](https://github.com/your-org/xbox/issues)
- 💬 [讨论区](https://github.com/your-org/xbox/discussions)

### 联系方式
- 邮箱: support@your-org.com
- 文档: [完整文档站点](https://docs.your-org.com/xbox)

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源协议。

## 🙏 致谢

感谢以下开源项目：
- [sing-box](https://github.com/SagerNet/sing-box) - 通用代理平台
- [Prometheus](https://prometheus.io/) - 监控告警系统
- [Grafana](https://grafana.com/) - 可视化平台
- [gRPC](https://grpc.io/) - 高性能RPC框架

---

⭐ 如果这个项目对你有帮助，请给我们一个星标！