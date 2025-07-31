# Xbox Sing-box管理系统 - API文档

本文档详细介绍Xbox系统的RESTful API和gRPC接口。

## 目录

- [Controller RESTful API](#controller-restful-api)
- [Agent API](#agent-api)
- [gRPC接口](#grpc接口)
- [监控和调试接口](#监控和调试接口)
- [错误码说明](#错误码说明)

## Controller RESTful API

### 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **认证方式**: 暂无（后续可添加JWT）

### 节点管理

#### 获取节点列表

```http
GET /api/v1/agents
```

**查询参数**:
- `status` (string, optional): 过滤状态 (`online`, `offline`, `error`)
- `page` (int, optional): 页码，默认1
- `limit` (int, optional): 每页数量，默认20

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 5,
    "page": 1,
    "limit": 20,
    "agents": [
      {
        "id": "agent-001",
        "hostname": "node-01",
        "ip_address": "192.168.1.100",
        "version": "1.0.0",
        "status": "online",
        "last_heartbeat": "2024-01-15T10:30:00Z",
        "metadata": {
          "os": "linux",
          "arch": "amd64",
          "cpu_cores": "4",
          "memory_gb": "8"
        },
        "created_at": "2024-01-15T08:00:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

#### 获取节点详情

```http
GET /api/v1/agents/{agent_id}
```

**路径参数**:
- `agent_id` (string): Agent唯一标识

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "agent-001",
    "hostname": "node-01",
    "ip_address": "192.168.1.100",
    "version": "1.0.0",
    "status": "online",
    "last_heartbeat": "2024-01-15T10:30:00Z",
    "system_info": {
      "cpu_usage": 15.5,
      "memory_usage": 2048,
      "disk_usage": 45.2,
      "uptime": 86400
    },
    "singbox_status": {
      "running": true,
      "pid": 12345,
      "connections": 150,
      "config_version": "v1.2.3"
    }
  }
}
```

#### 更新节点信息

```http
PUT /api/v1/agents/{agent_id}
```

**请求体**:
```json
{
  "metadata": {
    "description": "Updated description",
    "tags": ["production", "asia"]
  }
}
```

#### 删除节点

```http
DELETE /api/v1/agents/{agent_id}
```

### 配置管理

#### 创建配置

```http
POST /api/v1/configs
```

**请求体**:
```json
{
  "agent_id": "agent-001",
  "name": "production-config",
  "content": "{\"log\":{\"level\":\"info\"},...}",
  "version": "1.0.1"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "Configuration created successfully",
  "data": {
    "id": 123,
    "agent_id": "agent-001",
    "name": "production-config", 
    "version": "1.0.1",
    "status": "pending",
    "checksum": "sha256:abc123...",
    "created_at": "2024-01-15T10:35:00Z"
  }
}
```

#### 获取节点配置

```http
GET /api/v1/configs/{agent_id}
```

**查询参数**:
- `version` (string, optional): 指定版本
- `active_only` (bool, optional): 仅返回激活配置

#### 更新配置

```http
PUT /api/v1/configs/{config_id}
```

#### 删除配置

```http
DELETE /api/v1/configs/{config_id}
```

### 规则管理

#### 创建规则

```http
POST /api/v1/rules
```

**请求体**:
```json
{
  "rule_id": "rule-001",
  "name": "Block Ads",
  "type": "route",
  "content": "{\"domain\":[\"ads.example.com\"],\"outbound\":\"block\"}",
  "priority": 100,
  "enabled": true,
  "metadata": {
    "category": "security",
    "description": "Block advertising domains"
  }
}
```

#### 获取规则列表

```http
GET /api/v1/rules
```

**查询参数**:
- `type` (string, optional): 规则类型过滤
- `enabled` (bool, optional): 启用状态过滤
- `agent_id` (string, optional): 按Agent过滤

#### 更新规则

```http
PUT /api/v1/rules/{rule_id}
```

#### 删除规则

```http
DELETE /api/v1/rules/{rule_id}
```

#### 批量应用规则

```http
POST /api/v1/rules/batch-apply
```

**请求体**:
```json
{
  "agent_ids": ["agent-001", "agent-002"],
  "rule_ids": ["rule-001", "rule-002"],
  "operation": "apply"
}
```

### 监控数据

#### 获取系统指标

```http
GET /api/v1/metrics/system
```

**查询参数**:
- `agent_id` (string, optional): 指定Agent
- `start_time` (string): 开始时间 (RFC3339)
- `end_time` (string): 结束时间 (RFC3339)
- `metric_name` (string, optional): 指标名称

#### 获取告警信息

```http
GET /api/v1/alerts
```

**查询参数**:
- `status` (string): 告警状态 (`firing`, `resolved`)
- `severity` (string): 严重程度 (`critical`, `warning`, `info`)

## Agent API

### 基础信息

- **Base URL**: `http://localhost:8081`
- **用途**: Agent状态查询和调试

### 健康检查

#### 健康状态检查

```http
GET /health
```

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "24h30m15s",
  "version": "1.0.0",
  "checks": {
    "grpc_connection": {
      "status": "healthy",
      "message": "Connected to controller",
      "duration": "2ms"
    },
    "singbox_process": {
      "status": "healthy", 
      "message": "Process running",
      "duration": "1ms",
      "details": {
        "pid": "12345",
        "running": "true"
      }
    }
  }
}
```

#### 就绪检查

```http
GET /ready
```

#### 存活检查

```http
GET /live
```

### 系统状态

#### 获取Agent状态

```http
GET /status
```

**响应示例**:
```json
{
  "agent_id": "agent-001",
  "status": "online",
  "controller_connected": true,
  "last_heartbeat": "2024-01-15T10:30:00Z",
  "system_metrics": {
    "cpu_usage": 15.5,
    "memory_usage": 2048,
    "disk_usage": 45.2,
    "network_rx": 1024000,
    "network_tx": 512000
  },
  "singbox_status": {
    "running": true,
    "pid": 12345,
    "config_version": "1.0.1",
    "connections": 150,
    "uptime": 86400
  }
}
```

## gRPC接口

### 服务定义

```protobuf
service AgentService {
  rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc UpdateConfig(ConfigRequest) returns (ConfigResponse);
  rpc UpdateRules(RulesRequest) returns (RulesResponse);
  rpc GetStatus(StatusRequest) returns (StatusResponse);
}
```

### 节点注册

**请求**:
```protobuf
message RegisterRequest {
  string agent_id = 1;
  string hostname = 2;
  string ip_address = 3;
  string version = 4;
  map<string, string> metadata = 5;
}
```

**响应**:
```protobuf
message RegisterResponse {
  bool success = 1;
  string message = 2;
  string token = 3;
}
```

### 心跳检测

**请求**:
```protobuf
message HeartbeatRequest {
  string agent_id = 1;
  string status = 2;
  map<string, string> metrics = 3;
}
```

**响应**:
```protobuf
message HeartbeatResponse {
  bool success = 1;
  string message = 2;
  int64 next_heartbeat_interval = 3;
}
```

### 配置下发

**请求**:
```protobuf
message ConfigRequest {
  string agent_id = 1;
  string config_content = 2;
  string config_version = 3;
  bool force_update = 4;
}
```

### 规则管理

**请求**:
```protobuf
message RulesRequest {
  string agent_id = 1;
  repeated Rule rules = 2;
  string operation = 3; // add, delete, update, replace
}

message Rule {
  string id = 1;
  string type = 2;
  string content = 3;
  int32 priority = 4;
  bool enabled = 5;
  map<string, string> metadata = 6;
}
```

## 监控和调试接口

### Prometheus指标

#### Controller指标端点

```http
GET /metrics
```

**主要指标**:
- `xbox_controller_agents_total` - 注册Agent总数
- `xbox_controller_grpc_requests_total` - gRPC请求总数
- `xbox_controller_http_requests_total` - HTTP请求总数
- `xbox_controller_config_updates_total` - 配置更新总数

#### Agent指标端点

```http
GET /metrics
```

**主要指标**:
- `xbox_agent_status` - Agent状态
- `xbox_agent_heartbeat_total` - 心跳总数
- `xbox_system_cpu_usage_percent` - CPU使用率
- `xbox_system_memory_usage_bytes` - 内存使用量
- `xbox_singbox_status` - sing-box状态
- `xbox_singbox_connections_active` - 活跃连接数

### 调试接口

#### pprof性能分析

```http
GET /debug/pprof/
GET /debug/pprof/goroutine
GET /debug/pprof/heap
GET /debug/pprof/profile
GET /debug/pprof/trace
```

#### 运行时信息

```http
GET /debug/runtime
```

**响应示例**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "go_version": "go1.21.0",
  "goos": "linux",
  "goarch": "amd64",
  "num_cpu": 4,
  "num_goroutine": 25,
  "memory": {
    "alloc": 2048000,
    "total_alloc": 5096000,
    "sys": 8192000,
    "heap_objects": 1024
  },
  "gc": {
    "num_gc": 10,
    "pause_total_ns": 1000000,
    "last_gc": "2024-01-15T10:25:00Z"
  }
}
```

## 错误码说明

### HTTP状态码

| 状态码 | 说明 |
|-------|------|
| 200 | 请求成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 422 | 参数验证失败 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

### 业务错误码

| 错误码 | 说明 |
|-------|------|
| 10001 | Agent不存在 |
| 10002 | Agent离线 |
| 10003 | 配置格式错误 |
| 10004 | 规则验证失败 |
| 10005 | 数据库连接失败 |
| 10006 | gRPC连接失败 |
| 10007 | sing-box进程异常 |

### 错误响应格式

```json
{
  "code": 400,
  "error": "INVALID_PARAMETER",
  "message": "Agent ID is required",
  "details": {
    "field": "agent_id",
    "reason": "missing required field"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 使用示例

### 使用curl测试API

```bash
# 获取Agent列表
curl -X GET http://localhost:8080/api/v1/agents

# 创建配置
curl -X POST http://localhost:8080/api/v1/configs \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"agent-001","name":"test-config","content":"{...}","version":"1.0.0"}'

# 获取健康状态
curl -X GET http://localhost:8081/health

# 获取Prometheus指标
curl -X GET http://localhost:8080/metrics
```

### 使用JavaScript SDK

```javascript
// Controller API客户端
class XboxControllerAPI {
  constructor(baseURL = 'http://localhost:8080/api/v1') {
    this.baseURL = baseURL;
  }

  async getAgents(params = {}) {
    const url = new URL(`${this.baseURL}/agents`);
    Object.keys(params).forEach(key => 
      url.searchParams.append(key, params[key])
    );
    
    const response = await fetch(url);
    return response.json();
  }

  async createConfig(config) {
    const response = await fetch(`${this.baseURL}/configs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });
    return response.json();
  }
}

// 使用示例
const api = new XboxControllerAPI();
const agents = await api.getAgents({ status: 'online' });
console.log(agents);
```

## 版本兼容性

- **API版本**: v1
- **向后兼容**: 支持
- **废弃通知**: 提前30天通知
- **版本策略**: 语义化版本控制

---

更多详细信息请参考项目文档或联系技术支持团队。