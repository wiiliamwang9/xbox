#!/bin/bash

# 多路复用配置API测试脚本

set -e

CONTROLLER_URL="http://localhost:9000/api/v1"
AGENT_ID="test-agent-001"
PROTOCOL="vmess"

echo "=== 多路复用配置API测试 ==="
echo "Controller URL: $CONTROLLER_URL"
echo "Agent ID: $AGENT_ID"
echo "Protocol: $PROTOCOL"
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

# 检查依赖
check_dependencies() {
    print_info "检查依赖工具..."
    
    if ! command -v curl >/dev/null 2>&1; then
        print_error "curl 未安装"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        print_info "安装 jq..."
        if command -v apt-get >/dev/null 2>&1; then
            sudo apt-get update >/dev/null && sudo apt-get install -y jq >/dev/null
        elif command -v yum >/dev/null 2>&1; then
            sudo yum install -y jq >/dev/null
        else
            print_error "无法自动安装 jq，请手动安装"
            exit 1
        fi
    fi
    
    print_success "依赖检查完成"
}

# 测试Controller健康状态
test_controller_health() {
    print_info "测试Controller健康状态..."
    
    response=$(curl -s "http://localhost:9000/health" || echo "ERROR")
    if [[ "$response" == "ERROR" ]]; then
        print_error "Controller不可访问: $CONTROLLER_URL"
        return 1
    fi
    
    status=$(echo "$response" | jq -r '.status' 2>/dev/null || echo "unknown")
    if [[ "$status" == "ok" ]]; then
        print_success "Controller健康状态正常"
    else
        print_error "Controller健康状态异常: $status"
        return 1
    fi
}

# 测试更新多路复用配置
test_update_multiplex_config() {
    print_info "测试更新多路复用配置..."
    
    # 构建请求数据
    request_data='{
        "agent_id": "'$AGENT_ID'",
        "protocol": "'$PROTOCOL'",
        "enabled": true,
        "max_connections": 8,
        "min_streams": 4,
        "padding": false,
        "brutal_config": {
            "up": "100 Mbps",
            "down": "200 Mbps"
        }
    }'
    
    echo "请求数据:"
    echo "$request_data" | jq .
    echo
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/config" \
        -H "Content-Type: application/json" \
        -d "$request_data" || echo '{"success": false, "message": "请求失败"}')
    
    echo "响应数据:"
    echo "$response" | jq .
    echo
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        config_version=$(echo "$response" | jq -r '.config_version' 2>/dev/null || echo "unknown")
        print_success "多路复用配置更新成功，版本: $config_version"
    else
        message=$(echo "$response" | jq -r '.message' 2>/dev/null || echo "未知错误")
        print_error "多路复用配置更新失败: $message"
        return 1
    fi
}

# 测试获取多路复用配置
test_get_multiplex_config() {
    print_info "测试获取多路复用配置..."
    
    response=$(curl -s "$CONTROLLER_URL/multiplex/config/$AGENT_ID?protocol=$PROTOCOL" || echo '{"success": false}')
    
    echo "响应数据:"
    echo "$response" | jq .
    echo
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        config_count=$(echo "$response" | jq '.multiplex_configs | length' 2>/dev/null || echo "0")
        print_success "获取多路复用配置成功，配置数量: $config_count"
    else
        message=$(echo "$response" | jq -r '.message' 2>/dev/null || echo "未知错误")
        print_error "获取多路复用配置失败: $message"
        return 1
    fi
}

# 测试获取多路复用状态统计
test_get_multiplex_status() {
    print_info "测试获取多路复用状态统计..."
    
    response=$(curl -s "$CONTROLLER_URL/multiplex/status" || echo '{"success": false}')
    
    echo "响应数据:"
    echo "$response" | jq .
    echo
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        print_success "获取多路复用状态统计成功"
    else
        message=$(echo "$response" | jq -r '.message' 2>/dev/null || echo "未知错误")
        print_error "获取多路复用状态统计失败: $message"
        return 1
    fi
}

# 测试批量更新多路复用配置
test_batch_update_multiplex_config() {
    print_info "测试批量更新多路复用配置..."
    
    # 构建批量请求数据
    request_data='{
        "configs": [
            {
                "agent_id": "'$AGENT_ID'",
                "protocol": "vmess",
                "enabled": true,
                "max_connections": 6,
                "min_streams": 2,
                "padding": true
            },
            {
                "agent_id": "'$AGENT_ID'",
                "protocol": "vless",
                "enabled": true,
                "max_connections": 4,
                "min_streams": 4,
                "padding": false
            }
        ]
    }'
    
    echo "请求数据:"
    echo "$request_data" | jq .
    echo
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/batch" \
        -H "Content-Type: application/json" \
        -d "$request_data" || echo '{"success": false, "message": "请求失败"}')
    
    echo "响应数据:"
    echo "$response" | jq .
    echo
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        success_count=$(echo "$response" | jq -r '.success_count' 2>/dev/null || echo "0")
        total_count=$(echo "$response" | jq -r '.total_count' 2>/dev/null || echo "0")
        print_success "批量更新完成，成功: $success_count/$total_count"
    else
        message=$(echo "$response" | jq -r '.message' 2>/dev/null || echo "未知错误")
        print_error "批量更新失败: $message"
        return 1
    fi
}

# 测试禁用多路复用配置
test_disable_multiplex_config() {
    print_info "测试禁用多路复用配置..."
    
    request_data='{
        "agent_id": "'$AGENT_ID'",
        "protocol": "'$PROTOCOL'",
        "enabled": false,
        "max_connections": 4,
        "min_streams": 4,
        "padding": false
    }'
    
    response=$(curl -s -X POST "$CONTROLLER_URL/multiplex/config" \
        -H "Content-Type: application/json" \
        -d "$request_data" || echo '{"success": false, "message": "请求失败"}')
    
    success=$(echo "$response" | jq -r '.success' 2>/dev/null || echo "false")
    if [[ "$success" == "true" ]]; then
        print_success "多路复用配置禁用成功"
    else
        message=$(echo "$response" | jq -r '.message' 2>/dev/null || echo "未知错误")
        print_error "多路复用配置禁用失败: $message"
        return 1
    fi
}

# 主测试流程
main() {
    echo "开始多路复用配置API测试..."
    echo "时间: $(date)"
    echo

    # 检查依赖
    if ! check_dependencies; then
        exit 1
    fi
    echo

    # 测试Controller健康状态
    if ! test_controller_health; then
        print_error "Controller不可用，跳过API测试"
        exit 1
    fi
    echo

    # 执行各项测试
    local test_results=()
    
    echo "=== API功能测试 ==="
    
    # 测试更新多路复用配置
    if test_update_multiplex_config; then
        test_results+=("更新配置: ✓")
    else
        test_results+=("更新配置: ✗")
    fi
    echo

    # 测试获取多路复用配置
    if test_get_multiplex_config; then
        test_results+=("获取配置: ✓")
    else
        test_results+=("获取配置: ✗")
    fi
    echo

    # 测试获取状态统计
    if test_get_multiplex_status; then
        test_results+=("状态统计: ✓")
    else
        test_results+=("状态统计: ✗")
    fi
    echo

    # 测试批量更新
    if test_batch_update_multiplex_config; then
        test_results+=("批量更新: ✓")
    else
        test_results+=("批量更新: ✗")
    fi
    echo

    # 测试禁用配置
    if test_disable_multiplex_config; then
        test_results+=("禁用配置: ✓")
    else
        test_results+=("禁用配置: ✗")
    fi
    echo

    # 测试结果汇总
    echo "=== 测试结果汇总 ==="
    for result in "${test_results[@]}"; do
        echo "$result"
    done
    echo

    # 计算成功率
    success_count=$(printf '%s\n' "${test_results[@]}" | grep -c "✓" || echo "0")
    total_count=${#test_results[@]}
    success_rate=$((success_count * 100 / total_count))

    echo "测试完成: $success_count/$total_count 成功 (成功率: ${success_rate}%)"
    
    if [[ $success_count -eq $total_count ]]; then
        print_success "所有测试通过！"
        exit 0
    else
        print_error "部分测试失败"
        exit 1
    fi
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi