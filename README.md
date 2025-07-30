# Xbox Sing-box管理系统

分布式sing-box节点管理系统，包含Agent（代理）和Controller（控制器）两个核心组件。

## 功能特性

- **节点管理**: 统一管理多个sing-box节点
- **配置下发**: 远程配置更新和应用
- **规则管理**: 动态添加、删除、修改路由规则
- **实时监控**: 节点状态监控和性能指标收集
- **RESTful API**: 完整的API接口供第三方集成
- **gRPC通信**: 高性能的节点间通信

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

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Protocol Buffers编译器 (protoc)

### 安装依赖

```bash
# 安装Go依赖
make deps

# 生成protobuf代码
make proto
```

### 数据库初始化

```bash
# 创建数据库和表
mysql -u root -p < scripts/init.sql
```

### 配置文件

复制并修改配置文件：

```bash
cp configs/config.yaml configs/config.local.yaml
# 编辑配置文件，设置数据库连接等参数
```

### 构建和运行

```bash
# 构建应用
make build

# 运行Controller
make run-controller

# 运行Agent
make run-agent
```

## 项目结构

```
xbox/
├── cmd/                    # 应用入口
│   ├── agent/             # Agent启动程序
│   └── controller/        # Controller启动程序
├── internal/              # 内部包
│   ├── agent/            # Agent核心逻辑
│   ├── controller/       # Controller核心逻辑
│   ├── config/           # 配置管理
│   ├── database/         # 数据库操作
│   └── models/           # 数据模型
├── proto/                 # gRPC协议定义
├── api/                   # RESTful API
├── pkg/                   # 共享包
├── scripts/               # 部署脚本
├── configs/               # 配置文件
└── docs/                  # 文档
```

## API文档

### 节点管理

- `GET /api/v1/agents` - 获取节点列表
- `GET /api/v1/agents/{id}` - 获取节点详情
- `PUT /api/v1/agents/{id}` - 更新节点信息
- `DELETE /api/v1/agents/{id}` - 删除节点

### 配置管理

- `POST /api/v1/configs` - 创建配置
- `GET /api/v1/configs/{agent_id}` - 获取节点配置
- `PUT /api/v1/configs/{id}` - 更新配置
- `DELETE /api/v1/configs/{id}` - 删除配置

### 规则管理

- `POST /api/v1/rules` - 创建规则
- `GET /api/v1/rules` - 获取规则列表
- `PUT /api/v1/rules/{id}` - 更新规则
- `DELETE /api/v1/rules/{id}` - 删除规则

## 开发指南

### 环境配置

支持通过环境变量覆盖配置：

```bash
export XBOX_DATABASE_HOST=localhost
export XBOX_DATABASE_PORT=3306
export XBOX_DATABASE_USERNAME=root
export XBOX_DATABASE_PASSWORD=password
```

### 构建命令

```bash
make proto          # 生成protobuf代码
make build          # 构建应用
make test           # 运行测试
make clean          # 清理构建文件
```

## 部署

详细部署文档请参考 [部署指南](docs/deployment.md)

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License