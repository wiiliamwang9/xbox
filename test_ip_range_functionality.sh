#!/bin/bash

# IP段上报和查询功能测试脚本
# 测试Agent IP段检测、上报和Controller端查询API

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
TEST_DURATION=60  # 测试持续时间（秒）

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

# 等待服务就绪
wait_for_service() {
    local url=$1
    local service_name=$2
    local timeout=30
    
    log_info "等待 $service_name 服务启动..."
    
    for i in $(seq 1 $timeout); do
        if curl -s "$url" > /dev/null 2>&1; then
            log_success "$service_name 服务已就绪"
            return 0
        fi
        sleep 1
    done
    
    log_error "$service_name 服务启动超时"
    exit 1
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

# 测试获取Agent列表
test_get_agents() {
    ((TOTAL_TESTS++))
    log_info "测试获取Agent列表..."
    
    response=$(curl -s "$CONTROLLER_URL/agents")
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        agent_count=$(echo "$response" | jq -r '.data.total // 0')
        log_success "成功获取Agent列表，共有 $agent_count 个节点"
        return 0
    else
        log_error "获取Agent列表失败"
        echo "响应: $response"
        return 1
    fi
}

# 测试获取IP段信息
test_get_ip_ranges() {
    ((TOTAL_TESTS++))
    log_info "测试获取IP段信息..."
    
    response=$(curl -s "$CONTROLLER_URL/agents/ip-ranges")
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        range_count=$(echo "$response" | jq -r '.data.stats.total_ranges // 0')
        agent_count=$(echo "$response" | jq -r '.data.stats.total_agents // 0')
        
        log_success "成功获取IP段信息："
        echo "  - IP段数量: $range_count"
        echo "  - Agent数量: $agent_count"
        
        # 显示详细的IP段信息
        if [ "$range_count" -gt 0 ]; then
            echo "  - IP段详情:"
            echo "$response" | jq -r '.data.ip_ranges[] | "    - \(.ip_range) (\(.country), \(.region), \(.isp)) - \(.agent_count)个节点"'
            
            # 显示统计信息
            echo "  - 国家统计:"
            echo "$response" | jq -r '.data.stats.country_count | to_entries[] | "    - \(.key): \(.value)个IP段"'
            
            echo "  - 运营商统计:"
            echo "$response" | jq -r '.data.stats.isp_count | to_entries[] | "    - \(.key): \(.value)个IP段"'
        fi
        
        return 0
    else
        log_error "获取IP段信息失败"
        echo "响应: $response"
        return 1
    fi
}

# 测试IP段筛选功能
test_ip_range_filtering() {
    ((TOTAL_TESTS++))
    log_info "测试IP段筛选功能..."
    
    # 先获取所有IP段信息以找到可用的筛选条件
    all_ranges=$(curl -s "$CONTROLLER_URL/agents/ip-ranges")
    
    if ! echo "$all_ranges" | jq -e '.code == 200' > /dev/null; then
        log_error "无法获取IP段信息进行筛选测试"
        return 1
    fi
    
    # 获取第一个可用的国家进行筛选测试
    first_country=$(echo "$all_ranges" | jq -r '.data.stats.country_count | keys | first // empty')
    
    if [ -n "$first_country" ] && [ "$first_country" != "null" ]; then
        log_info "测试按国家筛选: $first_country"
        
        filtered_response=$(curl -s "$CONTROLLER_URL/agents/ip-ranges?country=$first_country")
        
        if echo "$filtered_response" | jq -e '.code == 200' > /dev/null; then
            filtered_count=$(echo "$filtered_response" | jq -r '.data.stats.total_ranges // 0')
            log_success "按国家筛选成功，筛选结果: $filtered_count 个IP段"
        else
            log_error "按国家筛选失败"
            return 1
        fi
    else
        log_warning "没有可用的国家信息进行筛选测试"
    fi
    
    return 0
}

# 测试Agent IP段信息更新（通过心跳日志）
test_agent_ip_reporting() {
    ((TOTAL_TESTS++))
    log_info "测试Agent IP段信息上报..."
    
    if [ ! -f "$CONTROLLER_LOG_FILE" ]; then
        log_warning "Controller日志文件不存在: $CONTROLLER_LOG_FILE"
        return 1
    fi
    
    # 检查最近的心跳日志
    recent_heartbeats=$(tail -n 50 "$CONTROLLER_LOG_FILE" | grep "收到心跳" | wc -l)
    
    if [ "$recent_heartbeats" -gt 0 ]; then
        log_success "检测到 $recent_heartbeats 个最近的心跳记录"
        
        # 显示最近几个心跳
        log_info "最近的心跳记录:"
        tail -n 20 "$CONTROLLER_LOG_FILE" | grep "收到心跳" | tail -n 5 | while read -r line; do
            echo "  $line"
        done
        
        return 0
    else
        log_error "未检测到Agent心跳记录"
        return 1
    fi
}

# 测试数据库IP段信息持久化
test_database_persistence() {
    ((TOTAL_TESTS++))
    log_info "测试数据库IP段信息持久化..."
    
    # 检查数据库中的IP段字段
    mysql_cmd="mysql -h localhost -u root -pxbox123456 xbox_manager -e"
    
    # 检查MySQL连接
    if ! $mysql_cmd "SELECT 1;" > /dev/null 2>&1; then
        log_error "无法连接到MySQL数据库"
        return 1
    fi
    
    # 检查agents表结构
    if $mysql_cmd "DESCRIBE agents;" | grep -E "(ip_range|country|region|city|isp)" > /dev/null; then
        log_success "数据库IP段字段已正确创建"
        
        # 检查IP段数据
        ip_range_count=$($mysql_cmd "SELECT COUNT(*) as count FROM agents WHERE ip_range IS NOT NULL AND ip_range != '';" | tail -n 1)
        
        if [ "$ip_range_count" -gt 0 ]; then
            log_success "数据库中存在 $ip_range_count 条IP段记录"
            
            # 显示示例数据
            log_info "示例IP段记录:"
            $mysql_cmd "SELECT id, ip_range, country, region, city, isp FROM agents WHERE ip_range IS NOT NULL AND ip_range != '' LIMIT 3;" | while read -r line; do
                echo "  $line"
            done
            
        else
            log_warning "数据库中暂无IP段记录"
        fi
        
        return 0
    else
        log_error "数据库IP段字段未找到"
        return 1
    fi
}

# 压力测试API响应性能
test_api_performance() {
    ((TOTAL_TESTS++))
    log_info "测试API响应性能..."
    
    start_time=$(date +%s%3N)
    response=$(curl -s "$CONTROLLER_URL/agents/ip-ranges")
    end_time=$(date +%s%3N)
    
    response_time=$((end_time - start_time))
    
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        log_success "API响应成功，响应时间: ${response_time}ms"
        
        if [ "$response_time" -lt 1000 ]; then
            log_success "API响应性能良好"
        elif [ "$response_time" -lt 3000 ]; then
            log_warning "API响应稍慢，但在可接受范围内"
        else
            log_error "API响应过慢，需要优化"
            return 1
        fi
        
        return 0
    else
        log_error "API响应失败"
        return 1
    fi
}

# 主测试流程
main() {
    echo "========================================="
    echo "    Xbox IP段上报和查询功能测试"
    echo "========================================="
    echo
    
    # 检查依赖
    check_dependencies
    
    # 等待Controller服务
    wait_for_service "$CONTROLLER_URL/../health" "Controller"
    
    echo
    log_info "开始IP段功能测试..."
    echo
    
    # 执行测试
    test_controller_health
    test_get_agents
    test_get_ip_ranges
    test_ip_range_filtering  
    test_agent_ip_reporting
    test_database_persistence
    test_api_performance
    
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