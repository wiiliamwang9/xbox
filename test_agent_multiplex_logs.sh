#!/bin/bash

# Agent多路复用配置日志演示脚本

set -e

CONTROLLER_URL="http://localhost:9000/api/v1"
AGENT_ID="test-agent-001"

echo "=== Agent多路复用配置日志演示 ==="
echo "时间: $(date)"
echo

# 颜色输出函数
print_success() {
    echo -e "\033[32m✓ $1\033[0m"
}

print_error() {
    echo -e "\033[31m✗ $1\033[0m"
}

print_info() {
    echo -e "\033[34mℹ $1\033[0m"
}

print_warning() {
    echo -e "\033[33m⚠ $1\033[0m"
}

# 检查Agent是否运行
check_agent_running() {
    print_info "检查Agent运行状态..."
    
    if pgrep -f "agent" >/dev/null 2>&1; then
        print_success "Agent进程正在运行"
    else
        print_warning "Agent进程未运行，请先启动Agent"
        print_info "启动命令: ./bin/agent -config configs/agent.yaml"
        return 1
    fi
}

# 模拟配置更新并观察日志
simulate_config_updates() {
    print_info "开始模拟多路复用配置更新..."
    echo
    
    # 配置1: 启用VMess多路复用
    print_info "=== 测试1: 启用VMess多路复用 ==="
    echo "请求配置:"
    config1='{
        "agent_id": "'$AGENT_ID'",
        "protocol": "vmess",
        "enabled": true,
        "max_connections": 8,
        "min_streams": 4,
        "padding": false,
        "brutal_config": {
            "up": "100 Mbps",
            "down": "200 Mbps"
        }
    }'
    echo "$config1" | jq .
    
    print_info "发送配置更新请求..."
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/config" \
        -H "Content-Type: application/json" \
        -d "$config1" || echo '{"success": false, "message": "请求失败"}')
    
    echo "响应结果:"
    echo "$response" | jq .
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        print_success "VMess多路复用配置更新成功"
    else
        print_error "VMess多路复用配置更新失败"
    fi
    
    print_warning "请查看Agent日志输出以查看详细的配置更新过程"
    echo
    sleep 2
    
    # 配置2: 启用VLESS多路复用
    print_info "=== 测试2: 启用VLESS多路复用 ==="
    echo "请求配置:"
    config2='{
        "agent_id": "'$AGENT_ID'",
        "protocol": "vless",
        "enabled": true,
        "max_connections": 6,
        "min_streams": 2,
        "padding": true
    }'
    echo "$config2" | jq .
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/config" \
        -H "Content-Type: application/json" \
        -d "$config2" || echo '{"success": false}')
    
    echo "响应结果:"
    echo "$response" | jq .
    echo
    sleep 2
    
    # 配置3: 禁用Trojan多路复用
    print_info "=== 测试3: 禁用Trojan多路复用 ==="
    echo "请求配置:"
    config3='{
        "agent_id": "'$AGENT_ID'",
        "protocol": "trojan",
        "enabled": false,
        "max_connections": 4,
        "min_streams": 4,
        "padding": false
    }'
    echo "$config3" | jq .
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/config" \
        -H "Content-Type: application/json" \
        -d "$config3" || echo '{"success": false}')
    
    echo "响应结果:"
    echo "$response" | jq .
    echo
    sleep 2
    
    # 配置4: 批量更新配置
    print_info "=== 测试4: 批量更新多路复用配置 ==="
    echo "请求配置:"
    config4='{
        "configs": [
            {
                "agent_id": "'$AGENT_ID'",
                "protocol": "shadowsocks",
                "enabled": true,
                "max_connections": 4,
                "min_streams": 8,
                "padding": false
            },
            {
                "agent_id": "'$AGENT_ID'",
                "protocol": "vmess",
                "enabled": true,
                "max_connections": 12,
                "min_streams": 6,
                "padding": true,
                "brutal_config": {
                    "up": "200 Mbps",
                    "down": "500 Mbps"
                }
            }
        ]
    }'
    echo "$config4" | jq .
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/batch" \
        -H "Content-Type: application/json" \
        -d "$config4" || echo '{"success": false}')
    
    echo "响应结果:"
    echo "$response" | jq .
    echo
}

# 显示期望的Agent日志输出示例
show_expected_logs() {
    print_info "=== 期望的Agent日志输出示例 ==="
    echo
    print_info "当您发送多路复用配置更新请求时，Agent应该输出类似以下的日志:"
    echo
    cat << 'EOF'
2023/12/01 15:30:00 === 收到多路复用配置更新请求 ===
2023/12/01 15:30:00 Agent ID: test-agent-001
2023/12/01 15:30:00 Protocol: vmess
2023/12/01 15:30:00 配置详情:
2023/12/01 15:30:00   启用状态: true
2023/12/01 15:30:00   协议: smux
2023/12/01 15:30:00   最大连接数: 8
2023/12/01 15:30:00   最小流数: 4
2023/12/01 15:30:00   填充: false
2023/12/01 15:30:00   Brutal配置: map[down:200 Mbps up:100 Mbps]
2023/12/01 15:30:00 开始更新多路复用配置: protocol=vmess
2023/12/01 15:30:00 多路复用配置参数: enabled=true, maxConnections=8, minStreams=4, padding=false
2023/12/01 15:30:00 正在更新sing-box多路复用配置...
2023/12/01 15:30:00   协议: vmess
2023/12/01 15:30:00   启用状态: true
2023/12/01 15:30:00   最大连接数: 8
2023/12/01 15:30:00   最小流数: 4
2023/12/01 15:30:00   填充: false
2023/12/01 15:30:00   Brutal配置: map[down:200 Mbps up:100 Mbps]
2023/12/01 15:30:00 找到匹配的出站配置: Tag=vmess-out, Type=vmess
2023/12/01 15:30:00   原配置: 未配置
2023/12/01 15:30:00   设置Brutal上传带宽: 100 Mbps
2023/12/01 15:30:00   设置Brutal下载带宽: 200 Mbps
2023/12/01 15:30:00   新配置: enabled=true, max_conn=8, min_streams=4, padding=false
2023/12/01 15:30:00 多路复用配置更新完成，影响的出站: [vmess-out]
2023/12/01 15:30:00 正在应用多路复用配置到sing-box...
2023/12/01 15:30:00 sing-box多路复用配置应用成功
2023/12/01 15:30:00 多路复用配置更新成功: protocol=vmess
2023/12/01 15:30:00 多路复用配置更新成功 (耗时: 45.2ms)
2023/12/01 15:30:00 配置版本: v1638360000
2023/12/01 15:30:00 === 多路复用配置更新请求处理完成 ===
EOF
    echo
    print_info "关键日志特点:"
    echo "✓ 详细记录接收到的配置参数"
    echo "✓ 显示原配置和新配置的对比"
    echo "✓ 记录每个出站配置的更新过程"
    echo "✓ 显示Brutal带宽配置的设置"
    echo "✓ 记录配置应用到sing-box的过程"
    echo "✓ 显示操作耗时和最终结果"
    echo
}

# 显示日志查看方法
show_log_viewing_methods() {
    print_info "=== 查看Agent日志的方法 ==="
    echo
    print_info "方法1: 实时查看Agent输出"
    echo "如果Agent在前台运行，直接观察控制台输出"
    echo
    print_info "方法2: 查看日志文件"
    echo "tail -f logs/xbox/agents/agent.log"
    echo
    print_info "方法3: 过滤多路复用相关日志"
    echo "tail -f logs/xbox/agents/agent.log | grep -i multiplex"
    echo
    print_info "方法4: 查看最近的配置更新日志"
    echo "tail -n 100 logs/xbox/agents/agent.log | grep -A 20 -B 5 '收到多路复用配置'"
    echo
}

# 主函数
main() {
    echo "开始Agent多路复用配置日志演示..."
    echo
    
    # 检查Agent状态
    if ! check_agent_running; then
        echo
        print_warning "请先启动Agent，然后重新运行此脚本"
        print_info "Agent启动后，它会："
        echo "1. 启动gRPC服务器监听配置推送"
        echo "2. 连接到Controller并注册"
        echo "3. 开始处理多路复用配置更新请求"
        exit 1
    fi
    echo
    
    # 显示期望日志
    show_expected_logs
    
    # 显示日志查看方法
    show_log_viewing_methods
    
    # 询问是否继续测试
    echo
    read -p "是否要继续发送测试配置更新请求? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "演示结束"
        exit 0
    fi
    echo
    
    # 检查依赖
    if ! command -v jq >/dev/null 2>&1; then
        print_info "安装jq..."
        if command -v apt-get >/dev/null 2>&1; then
            sudo apt-get update && sudo apt-get install -y jq
        elif command -v yum >/dev/null 2>&1; then
            sudo yum install -y jq
        else
            print_error "请手动安装jq"
            exit 1
        fi
    fi
    
    # 执行配置更新测试
    simulate_config_updates
    
    print_success "测试完成！请查看Agent控制台或日志文件以观察详细的配置更新过程"
    print_info "建议同时打开两个终端："
    echo "  终端1: 运行Agent并观察日志输出"
    echo "  终端2: 运行此测试脚本发送配置更新请求"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi