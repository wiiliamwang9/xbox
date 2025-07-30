# Xbox Sing-box管理系统开发计划

## 项目概述

Xbox Sing-box管理系统是一个分布式节点管理系统，包含Agent（代理）和Controller（控制器）两个核心组件，通过gRPC通信实现节点的统一管理和配置下发。

## 开发阶段规划

### 阶段一：基础架构搭建 ✅ **已完成**

#### 1.1 项目初始化
- [x] 初始化Go模块和项目结构
- [x] 创建Makefile和构建脚本
- [x] 设置项目目录结构

#### 1.2 协议定义
- [x] 定义gRPC协议文件(proto/agent.proto)
- [x] 生成gRPC Go代码
- [x] 创建protobuf生成脚本

#### 1.3 数据库设计
- [x] 设计MySQL数据库表结构
- [x] 创建数据库初始化脚本
- [x] 实现GORM数据模型

#### 1.4 配置管理
- [x] 实现Viper配置管理
- [x] 创建YAML配置文件
- [x] 实现数据库连接管理

**验收标准**：
- [x] 项目可以成功编译
- [x] 数据库连接正常
- [x] 配置文件加载正常
- [x] Proto代码生成无错误

---

### 阶段二：Controller开发 🚧 **进行中**

#### 2.1 gRPC服务端实现
- [ ] 实现AgentService gRPC服务
  - [ ] RegisterAgent - 节点注册
  - [ ] Heartbeat - 心跳检测
  - [ ] UpdateConfig - 配置下发
  - [ ] UpdateRules - 规则管理
  - [ ] GetStatus - 状态查询
- [ ] 实现服务端拦截器(认证、日志、监控)
- [ ] 添加TLS支持

**文件位置**：
- `internal/controller/grpc/server.go`
- `internal/controller/grpc/agent_service.go`
- `internal/controller/grpc/interceptors.go`

#### 2.2 RESTful API实现
- [ ] 实现Gin HTTP服务器
- [ ] 节点管理API
  - [ ] `GET /api/v1/agents` - 获取节点列表
  - [ ] `GET /api/v1/agents/{id}` - 获取节点详情
  - [ ] `PUT /api/v1/agents/{id}` - 更新节点信息
  - [ ] `DELETE /api/v1/agents/{id}` - 删除节点
- [ ] 配置管理API
  - [ ] `POST /api/v1/configs` - 创建配置
  - [ ] `GET /api/v1/configs/{agent_id}` - 获取节点配置
  - [ ] `PUT /api/v1/configs/{id}` - 更新配置
  - [ ] `DELETE /api/v1/configs/{id}` - 删除配置
- [ ] 规则管理API
- [ ] 监控数据API

**文件位置**：
- `api/handlers/agent.go`
- `api/handlers/config.go`
- `api/handlers/rule.go`
- `api/middleware/auth.go`
- `api/routes/routes.go`

#### 2.3 业务逻辑层
- [ ] 实现Agent管理服务
- [ ] 实现配置管理服务
- [ ] 实现规则管理服务
- [ ] 实现监控数据收集
- [ ] 实现数据库操作层(Repository模式)

**文件位置**：
- `internal/controller/service/agent_service.go`
- `internal/controller/service/config_service.go`
- `internal/controller/service/rule_service.go`
- `internal/controller/repository/agent_repo.go`

#### 2.4 系统功能
- [ ] 实现节点状态监控
- [ ] 实现配置版本管理
- [ ] 实现操作日志记录
- [ ] 实现数据清理任务

**验收标准**：
- [ ] gRPC服务可以正常接收请求
- [ ] RESTful API功能完整
- [ ] 数据库操作正常
- [ ] 支持多节点管理

---

### 阶段三：Agent开发 📋 **待开始**

#### 3.1 gRPC客户端实现
- [ ] 实现与Controller的gRPC连接
- [ ] 实现节点注册逻辑
- [ ] 实现心跳机制
- [ ] 实现配置接收和应用
- [ ] 实现规则接收和应用
- [ ] 实现状态上报

**文件位置**：
- `internal/agent/grpc/client.go`
- `internal/agent/grpc/connection.go`

#### 3.2 Sing-box管理
- [ ] 实现sing-box进程管理
- [ ] 实现配置文件管理
- [ ] 实现规则动态更新
- [ ] 实现进程监控和重启

**文件位置**：
- `internal/agent/singbox/manager.go`
- `internal/agent/singbox/config.go`
- `internal/agent/singbox/process.go`

#### 3.3 系统监控
- [ ] 实现系统资源监控
- [ ] 实现网络状态监控
- [ ] 实现sing-box性能监控
- [ ] 实现监控数据上报

**文件位置**：
- `internal/agent/monitor/system.go`
- `internal/agent/monitor/network.go`
- `internal/agent/monitor/reporter.go`

#### 3.4 配置和规则管理
- [ ] 实现配置文件验证
- [ ] 实现配置热更新
- [ ] 实现规则优先级管理
- [ ] 实现回滚机制

**验收标准**：
- [ ] Agent可以成功注册到Controller
- [ ] 心跳机制正常工作
- [ ] 配置下发和应用正常
- [ ] sing-box进程管理正常

---

### 阶段四：监控和运维 📋 **待开始**

#### 4.1 监控数据收集
- [ ] 实现Prometheus指标暴露
- [ ] 实现监控数据聚合
- [ ] 实现告警规则
- [ ] 实现监控数据可视化

**文件位置**：
- `internal/monitoring/prometheus.go`
- `internal/monitoring/metrics.go`
- `configs/prometheus.yml`

#### 4.2 日志和调试
- [ ] 实现结构化日志
- [ ] 实现日志轮转
- [ ] 实现调试接口
- [ ] 实现性能分析

**文件位置**：
- `pkg/logger/logger.go`
- `internal/debug/debug.go`

#### 4.3 健康检查
- [ ] 实现服务健康检查
- [ ] 实现依赖项检查
- [ ] 实现自动恢复机制

**文件位置**：
- `internal/health/health.go`
- `api/handlers/health.go`

#### 4.4 部署脚本
- [ ] 创建Docker镜像构建
- [ ] 创建Docker Compose配置
- [ ] 创建Kubernetes部署文件
- [ ] 创建服务启动脚本

**文件位置**：
- `Dockerfile`
- `docker-compose.yml`
- `k8s/deployment.yaml`
- `scripts/deploy.sh`

**验收标准**：
- [ ] 监控指标完整
- [ ] 日志记录完善
- [ ] 部署自动化
- [ ] 运维工具齐全

---

## 当前状态

### ✅ 已完成（阶段一）
- 项目结构搭建
- gRPC协议定义
- 数据库设计
- 基础配置管理
- 编译环境配置

### 🚧 正在进行（阶段二）
- 需要实现Controller的gRPC服务
- 需要实现RESTful API
- 需要实现业务逻辑层

### 📋 待开始
- 阶段三：Agent开发
- 阶段四：监控和运维

## 下一步工作

1. **立即开始**：实现Controller的gRPC服务端
2. **优先级**：AgentService的RegisterAgent和Heartbeat方法
3. **并行开发**：RESTful API的节点管理接口

## 文件结构规划

```
xbox/
├── cmd/                    # 应用入口 ✅
├── internal/              # 内部包
│   ├── controller/        # Controller核心逻辑 🚧
│   │   ├── grpc/         # gRPC服务实现
│   │   ├── service/      # 业务逻辑层
│   │   └── repository/   # 数据访问层
│   ├── agent/            # Agent核心逻辑 📋
│   │   ├── grpc/         # gRPC客户端
│   │   ├── singbox/      # sing-box管理
│   │   └── monitor/      # 监控模块
│   ├── config/           # 配置管理 ✅
│   ├── database/         # 数据库操作 ✅
│   └── models/           # 数据模型 ✅
├── api/                  # RESTful API 📋
├── pkg/                  # 共享包 📋
├── proto/                # gRPC协议 ✅
├── scripts/              # 脚本 ✅
├── configs/              # 配置文件 ✅
└── docs/                 # 文档 🚧
```

**图例**：✅ 已完成 | 🚧 进行中 | 📋 待开始

---

## 技术债务和优化

### 技术债务
- [ ] 添加单元测试
- [ ] 添加集成测试
- [ ] 优化错误处理
- [ ] 添加代码文档

### 性能优化
- [ ] 数据库连接池优化
- [ ] gRPC连接复用
- [ ] 缓存机制实现
- [ ] 并发性能优化

### 安全增强
- [ ] JWT认证实现
- [ ] TLS证书管理
- [ ] API限流实现
- [ ] 输入验证加强