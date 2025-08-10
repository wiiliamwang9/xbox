# Xbox过滤器功能使用指南

## 功能概述

Xbox系统现已支持协议级别的黑名单/白名单过滤功能，允许管理员通过Controller API远程管理Agent节点的访问控制规则。

## 已实现功能

### ✅ 核心功能
- **协议黑名单管理** - 支持按协议(http、https、socks5等)设置域名、IP、端口黑名单
- **协议白名单管理** - 支持按协议设置域名、IP、端口白名单  
- **配置查询** - 获取Agent的过滤器配置
- **状态监控** - 查看Agent过滤器状态和统计信息
- **配置回滚** - 支持回滚到历史版本
- **操作类型** - 支持add、remove、replace、clear等操作

### ✅ 技术实现
- **gRPC协议扩展** - 新增黑名单/白名单管理接口
- **Agent端过滤器管理器** - 实现配置持久化和版本管理
- **sing-box配置集成** - 自动生成路由规则并热重载
- **HTTP API接口** - RESTful API供外部调用
- **配置备份机制** - 自动备份和回滚支持

## API接口文档

### 基础信息
- **Controller API地址**: `http://localhost:9000/api/v1`
- **Content-Type**: `application/json`

### 1. 更新黑名单
```bash
POST /api/v1/filter/blacklist
```

**请求示例**:
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "debian-1753875293",
    "protocol": "http",
    "domains": ["facebook.com", "twitter.com"],
    "ips": ["1.2.3.4", "5.6.7.8"],
    "ports": ["8080", "3128"],
    "operation": "add"
  }' \
  http://localhost:9000/api/v1/filter/blacklist
```

**响应示例**:
```json
{
  "success": true,
  "message": "成功更新http协议的黑名单",
  "config_version": "v1754836204",
  "data": {
    "agent_id": "debian-1753875293",
    "protocol": "http",
    "operation": "add",
    "affected_items": {
      "domains": 2,
      "ips": 2,
      "ports": 2
    }
  }
}
```

### 2. 更新白名单
```bash
POST /api/v1/filter/whitelist
```

**请求示例**:
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "debian-1753875293",
    "protocol": "https",
    "domains": ["google.com", "github.com"],
    "ips": ["8.8.8.8", "1.1.1.1"],
    "ports": ["443"],
    "operation": "add"
  }' \
  http://localhost:9000/api/v1/filter/whitelist
```

### 3. 获取过滤器配置
```bash
GET /api/v1/filter/config/{agent_id}
```

**请求示例**:
```bash
# 获取所有协议配置
curl -X GET http://localhost:9000/api/v1/filter/config/debian-1753875293

# 获取特定协议配置
curl -X GET "http://localhost:9000/api/v1/filter/config/debian-1753875293?protocol=http"
```

**响应示例**:
```json
{
  "success": true,
  "message": "配置查询成功",
  "data": {
    "agent_id": "debian-1753875293",
    "filters": [
      {
        "protocol": "http",
        "blacklist_domains": ["example.com", "blocked.site"],
        "blacklist_ips": ["192.168.1.100"],
        "blacklist_ports": ["8080"],
        "whitelist_domains": ["google.com", "github.com"],
        "whitelist_ips": ["8.8.8.8", "1.1.1.1"],
        "whitelist_ports": ["443", "80"],
        "enabled": true,
        "last_updated": "2025-08-10T10:30:06-04:00"
      }
    ],
    "total": 1
  }
}
```

### 4. 获取Agent过滤器状态
```bash
GET /api/v1/filter/status/{agent_id}
```

**请求示例**:
```bash
curl -X GET http://localhost:9000/api/v1/filter/status/debian-1753875293
```

**响应示例**:
```json
{
  "success": true,
  "message": "状态查询成功",
  "data": {
    "agent_id": "debian-1753875293",
    "status": "online",
    "filter_version": "v1754836208",
    "protocols": ["http", "https", "socks5", "shadowsocks", "vmess", "trojan", "vless"],
    "statistics": {
      "total_rules": 15,
      "blacklist_rules": 8,
      "whitelist_rules": 7,
      "enabled_filters": 7,
      "disabled_filters": 0
    },
    "last_updated": "2025-08-10T10:30:08-04:00",
    "singbox_status": "running"
  }
}
```

### 5. 配置回滚
```bash
POST /api/v1/filter/rollback
```

**请求示例**:
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "debian-1753875293",
    "target_version": "",
    "reason": "回滚到上一个版本"
  }' \
  http://localhost:9000/api/v1/filter/rollback
```

## 操作类型说明

### 支持的操作类型
- **add** - 添加规则到现有列表
- **remove** - 从列表中移除指定规则
- **replace** - 替换整个列表
- **clear** - 清空列表

### 支持的协议类型
- **http** - HTTP代理协议
- **https** - HTTPS代理协议
- **socks5** - SOCKS5代理协议
- **shadowsocks** - Shadowsocks协议
- **vmess** - VMess协议
- **trojan** - Trojan协议
- **vless** - VLESS协议

## 工作原理

### 1. 配置流程
1. **API调用** - 通过Controller HTTP API发送配置请求
2. **gRPC转发** - Controller将请求转发给目标Agent
3. **配置更新** - Agent更新本地过滤器配置
4. **生成规则** - Agent生成sing-box路由规则
5. **热重载** - Agent重启sing-box应用新配置

### 2. 规则优先级
1. **黑名单规则** - 优先级最高，匹配时阻断连接
2. **白名单规则** - 其次，匹配时允许连接
3. **默认规则** - 最后，使用系统默认路由

### 3. 配置持久化
- **配置文件** - 存储在Agent的`./configs/filter.json`
- **版本管理** - 每次更新生成新版本号
- **自动备份** - 保留最近10个版本的备份
- **回滚支持** - 可快速回滚到任意历史版本

## 测试和验证

### 自动化测试
```bash
# 运行完整功能测试
./test_filter_functionality.sh

# 指定Agent ID测试
./test_filter_functionality.sh --agent-id your-agent-id

# 指定Controller地址测试
./test_filter_functionality.sh --url http://192.168.1.100:9000/api/v1
```

### 手动测试步骤

1. **添加黑名单**:
```bash
curl -X POST -H "Content-Type: application/json" -d '{
  "agent_id": "your-agent-id",
  "protocol": "http",
  "domains": ["facebook.com"],
  "operation": "add"
}' http://localhost:9000/api/v1/filter/blacklist
```

2. **验证配置**:
```bash
curl -X GET http://localhost:9000/api/v1/filter/config/your-agent-id
```

3. **测试代理连接**:
```bash
# 通过Agent的代理访问被封站点，应该被阻断
curl --proxy socks5://165.254.16.244:1080 http://facebook.com
```

4. **回滚配置**:
```bash
curl -X POST -H "Content-Type: application/json" -d '{
  "agent_id": "your-agent-id",
  "reason": "测试回滚"
}' http://localhost:9000/api/v1/filter/rollback
```

## 故障排除

### 常见问题

1. **API调用失败**
   - 检查Controller是否运行: `curl http://localhost:9000/health`
   - 确认Agent ID正确: 查看Controller日志中的心跳信息

2. **配置不生效**
   - 检查Agent日志: `journalctl -u xbox-agent -f`
   - 验证sing-box状态: Agent会自动重启sing-box

3. **代理连接异常**
   - 确认sing-box进程运行: `ps aux | grep sing-box`
   - 检查代理端口: `netstat -tlnp | grep :1080`

### 日志查看
```bash
# Controller日志
./bin/controller -config configs/config.yaml

# Agent日志 (远程)
sshpass -p 'Asd2025#' ssh -p 22 root@165.254.16.244 'journalctl -u xbox-agent -f'

# sing-box日志 (如果单独运行)
sing-box run -c /path/to/config.json
```

## 安全注意事项

1. **权限控制** - 仅授权管理员访问Controller API
2. **配置验证** - 系统会自动验证配置语法
3. **回滚机制** - 配置错误时可快速回滚
4. **日志审计** - 所有操作都有详细日志记录

## 扩展和定制

### 添加新协议
1. 在`internal/agent/filter/manager.go`中添加协议定义
2. 更新`initDefaultFilters()`函数
3. 重新编译并部署

### 自定义规则生成
1. 修改`GenerateRouteRules()`方法
2. 调整sing-box配置模板
3. 测试规则生效性

---

**版本**: v1.0.0  
**最后更新**: 2025-08-10  
**测试状态**: ✅ 全部通过 (9/9)