# 多路复用配置 API 文档

本文档描述了Xbox Sing-box管理系统中多路复用(Multiplex)配置的API接口。

## 概述

多路复用功能允许在单个连接上复用多个数据流，提高连接效率并减少握手开销。系统支持对不同协议(VMess, VLESS, Trojan, Shadowsocks)的多路复用配置进行动态管理。

### 支持的协议
- `vmess` - VMess协议
- `vless` - VLESS协议  
- `trojan` - Trojan协议
- `shadowsocks` - Shadowsocks协议

### 关键特性
- ✅ 动态配置更新，无需重启Agent
- ✅ 支持单个和批量配置操作
- ✅ 配置版本管理和状态跟踪
- ✅ 数据库持久化存储
- ✅ 实时统计和监控
- ✅ 支持Brutal拥塞控制算法

## API端点

### 基础URL
```
http://localhost:9000/api/v1/multiplex
```

---

## 1. 更新多路复用配置

### 请求
```http
POST /api/v1/multiplex/config
Content-Type: application/json

{
    "agent_id": "agent-001",
    "protocol": "vmess", 
    "enabled": true,
    "max_connections": 8,
    "min_streams": 4,
    "padding": false,
    "brutal_config": {
        "up": "100 Mbps",
        "down": "200 Mbps"
    }
}
```

### 请求参数
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `agent_id` | string | ✅ | Agent唯一标识符 |
| `protocol` | string | ✅ | 协议类型 (vmess/vless/trojan/shadowsocks) |
| `enabled` | boolean | ✅ | 是否启用多路复用 |
| `max_connections` | integer | ✅ | 最大连接数 (1-32) |
| `min_streams` | integer | ✅ | 最小流数量 (1-32) |
| `padding` | boolean | ❌ | 是否启用填充，默认false |
| `brutal_config` | object | ❌ | Brutal拥塞控制配置 |

### 响应
```json
{
    "success": true,
    "message": "多路复用配置更新成功",
    "config_version": "v1638360000"
}
```

### 响应字段
| 字段 | 类型 | 描述 |
|------|------|------|
| `success` | boolean | 操作是否成功 |
| `message` | string | 响应消息 |
| `config_version` | string | 配置版本号 |

---

## 2. 获取多路复用配置

### 请求
```http
GET /api/v1/multiplex/config/{agent_id}?protocol={protocol}
```

### 路径参数
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `agent_id` | string | ✅ | Agent唯一标识符 |

### 查询参数
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `protocol` | string | ❌ | 协议类型，为空时返回所有协议配置 |

### 响应
```json
{
    "success": true,
    "message": "获取多路复用配置成功",
    "multiplex_configs": [
        {
            "id": 1,
            "agent_id": "agent-001",
            "protocol": "vmess",
            "enabled": true,
            "multiplex_protocol": "smux",
            "max_connections": 8,
            "min_streams": 4,
            "padding": false,
            "brutal_config": {
                "up": "100 Mbps",
                "down": "200 Mbps"
            },
            "status": "active",
            "config_version": "v1638360000",
            "created_at": "2023-12-01T10:30:00Z",
            "updated_at": "2023-12-01T15:30:00Z"
        }
    ]
}
```

---

## 3. 获取多路复用状态统计

### 请求
```http
GET /api/v1/multiplex/status?agent_id={agent_id}
```

### 查询参数
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| `agent_id` | string | ❌ | Agent ID，为空时返回系统统计 |

### 系统统计响应
```json
{
    "success": true,
    "message": "获取多路复用状态成功",
    "data": {
        "total_agents": 10,
        "online_agents": 8,
        "total_configs": 25,
        "enabled_configs": 18,
        "active_configs": 15,
        "protocol_stats": {
            "vmess": {
                "total": 8,
                "enabled": 6,
                "active": 5
            },
            "vless": {
                "total": 7,
                "enabled": 5,
                "active": 4
            },
            "trojan": {
                "total": 6,
                "enabled": 4,
                "active": 3
            },
            "shadowsocks": {
                "total": 4,
                "enabled": 3,
                "active": 3
            }
        }
    }
}
```

### Agent统计响应
```json
{
    "success": true,
    "message": "获取多路复用状态成功",
    "data": {
        "agent_id": "agent-001",
        "hostname": "node-01.example.com",
        "agent_status": "online",
        "total_protocols": 3,
        "enabled_count": 2,
        "active_count": 2,
        "protocols": [
            {
                "protocol": "vmess",
                "enabled": true,
                "max_connections": 8,
                "min_streams": 4,
                "status": "active",
                "last_updated": "2023-12-01 15:30:00"
            }
        ]
    }
}
```

---

## 4. 批量更新多路复用配置

### 请求
```http
POST /api/v1/multiplex/batch
Content-Type: application/json

{
    "configs": [
        {
            "agent_id": "agent-001",
            "protocol": "vmess",
            "enabled": true,
            "max_connections": 6,
            "min_streams": 2,
            "padding": true
        },
        {
            "agent_id": "agent-001", 
            "protocol": "vless",
            "enabled": true,
            "max_connections": 4,
            "min_streams": 4,
            "padding": false
        }
    ]
}
```

### 响应
```json
{
    "success": true,
    "message": "批量更新完成，成功: 2/2",
    "total_count": 2,
    "success_count": 2,
    "results": [
        {
            "agent_id": "agent-001",
            "protocol": "vmess",
            "success": true,
            "message": "配置更新成功",
            "config_version": "v1638360100"
        },
        {
            "agent_id": "agent-001",
            "protocol": "vless", 
            "success": true,
            "message": "配置更新成功",
            "config_version": "v1638360101"
        }
    ]
}
```

---

## 错误响应

所有API在发生错误时都会返回以下格式的响应：

```json
{
    "success": false,
    "message": "错误描述信息"
}
```

### 常见错误码
| HTTP状态码 | 错误类型 | 描述 |
|------------|----------|------|
| 400 | Bad Request | 请求参数无效 |
| 404 | Not Found | Agent不存在 |
| 500 | Internal Server Error | 服务器内部错误 |
| 207 | Multi-Status | 批量操作部分成功 |

---

## 配置说明

### 多路复用配置结构
系统使用sing-box标准的多路复用配置格式：

```json
{
    "enabled": true,
    "protocol": "smux",
    "max_connections": 4,
    "min_streams": 4,
    "max_streams": 0,
    "padding": false,
    "brutal": {
        "enabled": true,
        "up": "100 Mbps", 
        "down": "200 Mbps"
    }
}
```

### 重要约束
1. **max_connections 和 max_streams 不能同时配置**：系统只使用 max_connections
2. **前提条件**：必须先启用多路复用 (enabled: true)
3. **连接数限制**：max_connections 范围为 1-32
4. **流数量限制**：min_streams 范围为 1-32

### Brutal拥塞控制
Brutal是一种拥塞控制算法，可以指定上传和下载带宽：
- `up`: 上传带宽 (如: "100 Mbps", "50 Kbps") 
- `down`: 下载带宽 (如: "200 Mbps", "1 Gbps")

---

## 使用示例

### cURL命令示例

#### 1. 启用VMess多路复用
```bash
curl -X POST http://localhost:9000/api/v1/multiplex/config \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-001",
    "protocol": "vmess",
    "enabled": true,
    "max_connections": 8,
    "min_streams": 4,
    "padding": false
  }'
```

#### 2. 查询配置
```bash
curl http://localhost:9000/api/v1/multiplex/config/agent-001?protocol=vmess
```

#### 3. 获取系统统计
```bash
curl http://localhost:9000/api/v1/multiplex/status
```

### JavaScript示例
```javascript
// 更新多路复用配置
const updateConfig = async (agentId, protocol, config) => {
    const response = await fetch('http://localhost:9000/api/v1/multiplex/config', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            agent_id: agentId,
            protocol: protocol,
            enabled: config.enabled,
            max_connections: config.maxConnections,
            min_streams: config.minStreams,
            padding: config.padding,
            brutal_config: config.brutal
        })
    });
    
    const result = await response.json();
    return result;
};

// 使用示例
updateConfig('agent-001', 'vmess', {
    enabled: true,
    maxConnections: 8,
    minStreams: 4,
    padding: false,
    brutal: {
        up: "100 Mbps",
        down: "200 Mbps"
    }
}).then(result => {
    console.log('配置更新结果:', result);
});
```

---

## 测试工具

系统提供了完整的API测试脚本：

```bash
# 运行多路复用API测试
./test_multiplex_api.sh
```

测试脚本会验证所有API端点的功能，包括：
- 配置更新
- 配置查询  
- 状态统计
- 批量操作
- 错误处理

---

## 监控和日志

### 日志输出
系统会在以下位置记录多路复用相关日志：
- Controller日志: `logs/xbox/controller/`
- Agent日志: `logs/xbox/agents/`

### 监控指标
通过Prometheus可以监控以下指标：
- `xbox_multiplex_config_total` - 多路复用配置总数
- `xbox_multiplex_config_enabled` - 已启用的多路复用配置数
- `xbox_multiplex_config_active` - 活跃的多路复用配置数
- `xbox_multiplex_connections_total` - 多路复用连接总数

### 配置状态
多路复用配置有以下状态：
- `inactive` - 未激活（新创建或Agent离线）
- `active` - 已激活（成功推送到Agent）
- `error` - 错误状态（推送失败）