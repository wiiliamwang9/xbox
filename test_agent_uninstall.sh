#!/bin/bash

# Agent卸载功能测试脚本
# 测试Controller卸载Agent功能，包括sing-box清理和状态上报

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置参数
CONTROLLER_URL="http://localhost:9000/api/v1"
CONTROLLER_LOG_FILE="/root/wl/code/xbox/controller_new.log"
TEST_TARGET_IP="195.78.55.64"  # 使用当前Agent的IP进行测试

# 测试统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((PASSED_TESTS++))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((FAILED_TESTS++))
}

# 测试结果汇总
print_test_summary() {
    echo
    echo "========================================="
    echo "          测试结果汇总"
    echo "========================================="
    echo "总测试数: $TOTAL_TESTS"
    echo "通过测试: $PASSED_TESTS"
    echo "失败测试: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}所有测试通过！${NC}"
        exit 0
    else
        echo -e "${RED}有 $FAILED_TESTS 个测试失败${NC}"
        exit 1
    fi
}

# 检查依赖工具
check_dependencies() {
    log_info "检查依赖工具..."
    
    for cmd in curl jq; do
        if ! command -v $cmd &> /dev/null; then
            log_error "缺少必需工具: $cmd"
            exit 1
        fi
    done
    
    log_success "依赖工具检查完成"
}

# 测试Controller健康状态
test_controller_health() {
    ((TOTAL_TESTS++))
    log_info "测试Controller健康状态..."
    
    response=$(curl -s -w "%{http_code}" "$CONTROLLER_URL/../health")
    http_code=${response: -3}
    
    if [ "$http_code" = "200" ]; then
        log_success "Controller健康检查通过"
        return 0
    else
        log_error "Controller健康检查失败，HTTP状态码: $http_code"
        return 1
    fi
}

# 获取目标Agent信息
get_target_agent() {
    log_info "获取目标Agent信息: IP=$TEST_TARGET_IP"
    
    response=$(curl -s "$CONTROLLER_URL/agents")
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        # 查找匹配IP的Agent
        agent_info=$(echo "$response" | jq -r --arg ip "$TEST_TARGET_IP" '.data.items[] | select(.ip_address == $ip) | {id, hostname, ip_address, status}')
        
        if [ -n "$agent_info" ] && [ "$agent_info" != "null" ]; then
            AGENT_ID=$(echo "$agent_info" | jq -r '.id')
            AGENT_STATUS=$(echo "$agent_info" | jq -r '.status')
            log_success "找到目标Agent: ID=$AGENT_ID, 状态=$AGENT_STATUS"
            return 0
        else
            log_error "未找到IP为 $TEST_TARGET_IP 的Agent"
            return 1
        fi
    else
        log_error "获取Agent列表失败"
        return 1
    fi
}

# 测试卸载API请求
test_uninstall_api() {
    ((TOTAL_TESTS++))
    log_info "测试Agent卸载API..."
    
    # 构建卸载请求
    uninstall_request=$(cat <<EOF
{
    "ip": "$TEST_TARGET_IP",
    "force_uninstall": false,
    "reason": "测试卸载功能",
    "timeout_seconds": 60,
    "delete_from_db": false
}
EOF
)

    log_info "发送卸载请求:"
    echo "$uninstall_request" | jq .
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$uninstall_request" \
        "$CONTROLLER_URL/agents/uninstall")
    
    log_info "卸载API响应:"
    echo "$response" | jq .
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        UNINSTALL_AGENT_ID=$(echo "$response" | jq -r '.data.agent_id')
        UNINSTALL_STATUS=$(echo "$response" | jq -r '.data.uninstall_status')
        
        log_success "卸载请求发送成功"
        log_info "  Agent ID: $UNINSTALL_AGENT_ID"
        log_info "  卸载状态: $UNINSTALL_STATUS"
        
        return 0
    else
        error_msg=$(echo "$response" | jq -r '.message // "未知错误"')
        log_error "卸载请求失败: $error_msg"
        return 1
    fi
}

# 测试强制卸载
test_force_uninstall_api() {
    ((TOTAL_TESTS++))
    log_info "测试强制卸载API..."
    
    # 构建强制卸载请求
    force_uninstall_request=$(cat <<EOF
{
    "ip": "$TEST_TARGET_IP",
    "force_uninstall": true,
    "reason": "强制卸载测试",
    "timeout_seconds": 30,
    "delete_from_db": true
}
EOF
)

    log_info "发送强制卸载请求:"
    echo "$force_uninstall_request" | jq .
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$force_uninstall_request" \
        "$CONTROLLER_URL/agents/uninstall")
    
    log_info "强制卸载API响应:"
    echo "$response" | jq .
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        log_success "强制卸载请求发送成功"
        return 0
    else
        error_msg=$(echo "$response" | jq -r '.message // "未知错误"')
        log_error "强制卸载请求失败: $error_msg"
        return 1
    fi
}

# 监控卸载过程
monitor_uninstall_process() {
    ((TOTAL_TESTS++))
    log_info "监控卸载过程..."
    
    start_time=$(date +%s)
    timeout=90  # 90秒超时
    
    log_info "监控Controller日志中的卸载状态..."
    
    while [ $(($(date +%s) - start_time)) -lt $timeout ]; do
        # 检查Controller日志中的卸载相关信息
        if tail -n 50 "$CONTROLLER_LOG_FILE" | grep -q "收到Agent卸载状态上报"; then
            log_success "检测到Agent卸载状态上报"
            
            # 显示最近的卸载日志
            log_info "最近的卸载日志:"
            tail -n 20 "$CONTROLLER_LOG_FILE" | grep -E "(卸载|uninstall)" | tail -n 5
            return 0
        fi
        
        sleep 5
    done
    
    log_error "未在 $timeout 秒内检测到卸载状态上报"
    return 1
}

# 测试卸载后的系统清理
test_system_cleanup() {
    ((TOTAL_TESTS++))
    log_info "测试系统清理效果..."
    
    # 检查sing-box进程
    if pgrep -f sing-box > /dev/null; then
        log_warning "sing-box进程仍在运行"
    else
        log_success "sing-box进程已停止"
    fi
    
    # 检查配置文件
    config_files=(
        "./sing-box.json"
        "./configs/sing-box.json"
        "./configs/filter.json"
    )
    
    cleaned_count=0
    for config_file in "${config_files[@]}"; do
        if [ ! -f "$config_file" ]; then
            ((cleaned_count++))
            log_info "配置文件已清理: $config_file"
        fi
    done
    
    if [ $cleaned_count -gt 0 ]; then
        log_success "已清理 $cleaned_count 个配置文件"
    else
        log_warning "没有检测到配置文件被清理"
    fi
    
    # 检查PID文件
    pid_files=(
        "./sing-box.pid"
        "/var/run/sing-box.pid"
        "/tmp/sing-box.pid"
    )
    
    for pid_file in "${pid_files[@]}"; do
        if [ ! -f "$pid_file" ]; then
            log_info "PID文件已清理: $pid_file"
        fi
    done
    
    return 0
}

# 测试数据库状态更新
test_database_status() {
    ((TOTAL_TESTS++))
    log_info "测试数据库Agent状态更新..."
    
    # 等待状态更新
    sleep 5
    
    response=$(curl -s "$CONTROLLER_URL/agents")
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        # 查找目标Agent的状态
        agent_status=$(echo "$response" | jq -r --arg ip "$TEST_TARGET_IP" '.data.items[] | select(.ip_address == $ip) | .status')
        
        if [ -n "$agent_status" ] && [ "$agent_status" != "null" ]; then
            log_success "Agent状态已更新为: $agent_status"
            
            if [ "$agent_status" = "uninstalling" ] || [ "$agent_status" = "offline" ]; then
                log_success "Agent状态更新正确"
                return 0
            else
                log_warning "Agent状态为: $agent_status，可能卸载未完成"
                return 1
            fi
        else
            log_info "Agent可能已从数据库删除（强制卸载模式）"
            return 0
        fi
    else
        log_error "获取Agent状态失败"
        return 1
    fi
}

# 测试IP参数验证
test_invalid_ip() {
    ((TOTAL_TESTS++))
    log_info "测试无效IP地址处理..."
    
    invalid_request=$(cat <<EOF
{
    "ip": "192.168.999.999",
    "force_uninstall": false,
    "reason": "测试无效IP",
    "timeout_seconds": 30,
    "delete_from_db": false
}
EOF
)
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$invalid_request" \
        "$CONTROLLER_URL/agents/uninstall")
    
    if echo "$response" | jq -e '.code == 404 or .code == 400' > /dev/null; then
        log_success "正确处理无效IP地址"
        return 0
    else
        log_error "未正确处理无效IP地址"
        return 1
    fi
}

# 主测试流程
main() {
    echo "========================================="
    echo "      Xbox Agent卸载功能测试"
    echo "========================================="
    echo
    
    # 检查依赖
    check_dependencies
    
    echo
    log_info "开始Agent卸载功能测试..."
    echo
    
    # 基础测试
    test_controller_health
    
    # 获取目标Agent
    if get_target_agent; then
        echo
        log_info "开始卸载测试..."
        
        # API测试
        test_uninstall_api
        test_invalid_ip
        
        # 如果找到了Agent，进行完整的卸载测试
        if [ -n "$AGENT_ID" ]; then
            echo
            log_info "开始监控卸载过程..."
            monitor_uninstall_process
            test_system_cleanup
            test_database_status
            
            echo
            log_info "进行强制卸载测试..."
            test_force_uninstall_api
        fi
    else
        log_warning "未找到测试目标Agent，跳过完整卸载测试"
        # 仍然测试API的错误处理
        test_invalid_ip
    fi
    
    echo
    print_test_summary
}

# 清理函数
cleanup() {
    log_info "清理测试环境..."
}

# 信号处理
trap cleanup EXIT

# 执行主函数
main "$@"