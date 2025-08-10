#!/bin/bash

# Xbox过滤器功能测试脚本
# 用于测试黑名单/白名单功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
CONTROLLER_API="http://localhost:9000/api/v1"
AGENT_ID="debian-1753875293"  # 根据实际部署的Agent ID调整

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 测试函数

# 1. 测试黑名单添加
test_add_blacklist() {
    log_info "测试添加黑名单..."
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "agent_id": "'$AGENT_ID'",
            "protocol": "http",
            "domains": ["facebook.com", "twitter.com"],
            "ips": ["1.2.3.4", "5.6.7.8"],
            "ports": ["8080", "3128"],
            "operation": "add"
        }' \
        $CONTROLLER_API/filter/blacklist)
    
    echo "响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "黑名单添加成功"
    else
        log_error "黑名单添加失败"
        return 1
    fi
}

# 2. 测试白名单添加
test_add_whitelist() {
    log_info "测试添加白名单..."
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "agent_id": "'$AGENT_ID'",
            "protocol": "https",
            "domains": ["google.com", "github.com"],
            "ips": ["8.8.8.8", "1.1.1.1"],
            "ports": ["443"],
            "operation": "add"
        }' \
        $CONTROLLER_API/filter/whitelist)
    
    echo "响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "白名单添加成功"
    else
        log_error "白名单添加失败"
        return 1
    fi
}

# 3. 测试获取过滤器配置
test_get_filter_config() {
    log_info "测试获取过滤器配置..."
    
    local response=$(curl -s -X GET \
        "$CONTROLLER_API/filter/config/$AGENT_ID")
    
    echo "配置响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "配置查询成功"
    else
        log_error "配置查询失败"
        return 1
    fi
}

# 4. 测试获取特定协议配置
test_get_protocol_filter() {
    log_info "测试获取HTTP协议过滤器配置..."
    
    local response=$(curl -s -X GET \
        "$CONTROLLER_API/filter/config/$AGENT_ID?protocol=http")
    
    echo "HTTP协议配置: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "HTTP协议配置查询成功"
    else
        log_error "HTTP协议配置查询失败"
        return 1
    fi
}

# 5. 测试Agent过滤器状态
test_agent_filter_status() {
    log_info "测试获取Agent过滤器状态..."
    
    local response=$(curl -s -X GET \
        "$CONTROLLER_API/filter/status/$AGENT_ID")
    
    echo "Agent状态: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "Agent状态查询成功"
    else
        log_error "Agent状态查询失败"
        return 1
    fi
}

# 6. 测试黑名单替换
test_replace_blacklist() {
    log_info "测试替换黑名单..."
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "agent_id": "'$AGENT_ID'",
            "protocol": "http",
            "domains": ["badsite.com", "malware.net"],
            "ips": ["192.168.1.100"],
            "ports": ["8888"],
            "operation": "replace"
        }' \
        $CONTROLLER_API/filter/blacklist)
    
    echo "替换响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "黑名单替换成功"
    else
        log_error "黑名单替换失败"
        return 1
    fi
}

# 7. 测试配置回滚
test_config_rollback() {
    log_info "测试配置回滚..."
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "agent_id": "'$AGENT_ID'",
            "target_version": "",
            "reason": "测试回滚功能"
        }' \
        $CONTROLLER_API/filter/rollback)
    
    echo "回滚响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "配置回滚成功"
    else
        log_error "配置回滚失败"
        return 1
    fi
}

# 8. 测试清空黑名单
test_clear_blacklist() {
    log_info "测试清空黑名单..."
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "agent_id": "'$AGENT_ID'",
            "protocol": "http",
            "operation": "clear"
        }' \
        $CONTROLLER_API/filter/blacklist)
    
    echo "清空响应: $response"
    
    if echo "$response" | grep -q '"success":true'; then
        log_success "黑名单清空成功"
    else
        log_error "黑名单清空失败"
        return 1
    fi
}

# 9. 测试无效参数处理
test_invalid_parameters() {
    log_info "测试无效参数处理..."
    
    # 测试缺少必需参数
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "protocol": "http",
            "operation": "add"
        }' \
        $CONTROLLER_API/filter/blacklist)
    
    echo "无效参数响应: $response"
    
    if echo "$response" | grep -q '"success":false'; then
        log_success "无效参数正确处理"
    else
        log_error "无效参数处理失败"
        return 1
    fi
}

# 检查Controller是否运行
check_controller() {
    log_info "检查Controller服务状态..."
    
    if curl -s "http://localhost:9000/health" > /dev/null 2>&1; then
        log_success "Controller服务正在运行"
    else
        log_error "Controller服务未运行，请先启动Controller"
        return 1
    fi
}

# 检查API响应格式
check_api_response() {
    log_info "检查API响应格式..."
    
    local response=$(curl -s "http://localhost:9000/health")
    echo "健康检查响应: $response"
    
    if echo "$response" | grep -q '"status"'; then
        log_success "API响应格式正确"
    else
        log_error "API响应格式异常"
        return 1
    fi
}

# 主测试流程
main() {
    log_info "=== Xbox过滤器功能测试开始 ==="
    
    # 前置检查
    check_controller || exit 1
    check_api_response || exit 1
    
    # 等待服务稳定
    sleep 2
    
    # 执行测试
    local test_count=0
    local pass_count=0
    
    tests=(
        "test_add_blacklist"
        "test_add_whitelist"
        "test_get_filter_config"
        "test_get_protocol_filter"
        "test_agent_filter_status"
        "test_replace_blacklist"
        "test_config_rollback"
        "test_clear_blacklist"
        "test_invalid_parameters"
    )
    
    for test in "${tests[@]}"; do
        echo ""
        log_info "执行测试: $test"
        test_count=$((test_count + 1))
        
        if $test; then
            pass_count=$((pass_count + 1))
        else
            log_error "测试失败: $test"
        fi
        
        # 测试间隔
        sleep 1
    done
    
    echo ""
    log_info "=== 测试结果汇总 ==="
    log_info "总测试数: $test_count"
    log_info "通过测试: $pass_count"
    log_info "失败测试: $((test_count - pass_count))"
    
    if [ $pass_count -eq $test_count ]; then
        log_success "所有测试通过！"
        exit 0
    else
        log_error "部分测试失败！"
        exit 1
    fi
}

# 帮助信息
show_help() {
    echo "Xbox过滤器功能测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help      显示此帮助信息"
    echo "  -a, --agent-id  指定Agent ID (默认: $AGENT_ID)"
    echo "  -u, --url       指定Controller API URL (默认: $CONTROLLER_API)"
    echo ""
    echo "示例:"
    echo "  $0"
    echo "  $0 --agent-id debian-123456"
    echo "  $0 --url http://192.168.1.100:8080/api/v1"
}

# 参数解析
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -a|--agent-id)
            AGENT_ID="$2"
            shift 2
            ;;
        -u|--url)
            CONTROLLER_API="$2"
            shift 2
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 运行主程序
main