#!/bin/bash

# Xbox Sing-box管理系统部署脚本 - 节点直接部署
# 使用方法: ./scripts/deploy.sh [action] [options]

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONTROLLER_BINARY="$PROJECT_ROOT/bin/controller"
AGENT_BINARY="$PROJECT_ROOT/bin/agent"
CONTROLLER_CONFIG="$PROJECT_ROOT/configs/config.yaml"
AGENT_CONFIG="$PROJECT_ROOT/configs/agent.yaml"
CONTROLLER_PID_FILE="$PROJECT_ROOT/logs/controller.pid"
AGENT_PID_FILE="$PROJECT_ROOT/logs/agent.pid"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go未安装，请先安装Go 1.21+"
        exit 1
    fi
    
    # 检查Go版本
    GO_VERSION=$(go version | cut -d ' ' -f3 | cut -d 'o' -f2)
    if [[ "$GO_VERSION" < "1.21" ]]; then
        log_error "Go版本过低: $GO_VERSION，需要1.21+"
        exit 1
    fi
    
    # 检查protoc
    if ! command -v protoc &> /dev/null; then
        log_warning "protoc未安装，将跳过proto代码生成"
    fi
    
    log_success "依赖检查完成 - Go版本: $GO_VERSION"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要目录..."
    
    mkdir -p "$PROJECT_ROOT/logs"
    mkdir -p "$PROJECT_ROOT/data"
    mkdir -p "$PROJECT_ROOT/configs"
    mkdir -p "$PROJECT_ROOT/bin"
    
    log_success "目录创建完成"
}

# 生成配置文件
generate_configs() {
    log_info "生成配置文件..."
    
    # 创建Controller配置
    if [ ! -f "$CONTROLLER_CONFIG" ]; then
        cat > "$CONTROLLER_CONFIG" << EOF
server:
  grpc:
    bind: "0.0.0.0"
    port: 9090
  http:
    bind: "0.0.0.0" 
    port: 8080

database:
  type: "sqlite"
  dsn: "./data/controller.db"
  auto_migrate: true

log:
  level: "info"
  output: "./logs/controller.log"

# Agent管理配置
agent:
  heartbeat_timeout: 90    # 心跳超时时间（秒）
  cleanup_interval: 300    # 清理间隔（秒）

# 监控配置
monitoring:
  metrics_port: 9001
  health_check_port: 8081
EOF
        log_success "Controller配置文件已创建: $CONTROLLER_CONFIG"
    else
        log_info "Controller配置文件已存在"
    fi
    
    # 创建Agent配置
    if [ ! -f "$AGENT_CONFIG" ]; then
        cat > "$AGENT_CONFIG" << EOF
agent:
  id: "agent-$(hostname)-$(date +%s)"
  controller_addr: "localhost:9090"
  heartbeat_interval: 30
  sing_box_binary: "sing-box"
  sing_box_config: "./configs/sing-box.json"

log:
  level: "info"
  output: "./logs/agent.log"
EOF
        log_success "Agent配置文件已创建: $AGENT_CONFIG"
    else
        log_info "Agent配置文件已存在"
    fi
    
    # 创建sing-box配置
    if [ ! -f "$PROJECT_ROOT/configs/sing-box.json" ]; then
        cat > "$PROJECT_ROOT/configs/sing-box.json" << 'EOF'
{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "dns": {
    "servers": [
      {
        "tag": "default",
        "address": "8.8.8.8",
        "strategy": "prefer_ipv4"
      }
    ]
  },
  "inbounds": [
    {
      "type": "socks",
      "tag": "socks",
      "listen": "0.0.0.0",
      "listen_port": 1080,
      "sniff": true
    },
    {
      "type": "http",
      "tag": "http",
      "listen": "0.0.0.0",
      "listen_port": 8888,
      "sniff": true
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    },
    {
      "type": "block",
      "tag": "block"
    }
  ],
  "route": {
    "auto_detect_interface": true,
    "final": "direct",
    "rules": [
      {
        "geoip": ["private"],
        "outbound": "direct"
      }
    ]
  }
}
EOF
        log_success "sing-box配置文件已创建"
    else
        log_info "sing-box配置文件已存在"
    fi
}

# 构建应用
build_applications() {
    log_info "构建应用程序..."
    
    cd "$PROJECT_ROOT"
    
    # 生成protobuf代码
    if command -v protoc &> /dev/null && [ -f "./scripts/generate_proto.sh" ]; then
        log_info "生成protobuf代码..."
        make proto || log_warning "protobuf代码生成失败，继续构建..."
    fi
    
    # 安装依赖
    log_info "安装Go依赖..."
    go mod tidy
    
    # 构建Controller
    log_info "构建Controller..."
    go build -o bin/controller ./cmd/controller
    
    # 构建Agent
    log_info "构建Agent..."
    go build -o bin/agent ./cmd/agent
    
    log_success "应用构建完成"
}

# 检查进程状态
check_process_status() {
    local name=$1
    local pid_file=$2
    local binary=$3
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null 2>&1; then
            log_success "$name 正在运行 (PID: $pid)"
            return 0
        else
            log_warning "$name PID文件存在但进程不运行，清理PID文件"
            rm -f "$pid_file"
        fi
    fi
    
    # 检查是否有其他进程
    local processes=$(pgrep -f "$binary" || true)
    if [[ -n "$processes" ]]; then
        log_warning "发现其他 $name 进程: $processes"
        return 2
    fi
    
    log_info "$name 未运行"
    return 1
}

# 启动Controller
start_controller() {
    log_info "启动Controller..."
    
    # 检查是否已在运行
    if check_process_status "Controller" "$CONTROLLER_PID_FILE" "$CONTROLLER_BINARY" > /dev/null 2>&1; then
        log_info "Controller已在运行"
        return 0
    fi
    
    # 检查端口占用
    if netstat -tlnp 2>/dev/null | grep -q ":9090 " || ss -tlnp 2>/dev/null | grep -q ":9090 "; then
        log_error "端口9090已被占用"
        netstat -tlnp 2>/dev/null | grep ":9090 " || ss -tlnp 2>/dev/null | grep ":9090 "
        return 1
    fi
    
    # 启动Controller
    cd "$PROJECT_ROOT"
    nohup "$CONTROLLER_BINARY" -config "$CONTROLLER_CONFIG" > logs/controller.log 2>&1 &
    local controller_pid=$!
    echo $controller_pid > "$CONTROLLER_PID_FILE"
    
    # 等待启动
    log_info "等待Controller启动..."
    for i in {1..10}; do
        if ps -p $controller_pid > /dev/null 2>&1; then
            if netstat -tlnp 2>/dev/null | grep -q ":9090.*$controller_pid" || ss -tlnp 2>/dev/null | grep -q ":9090.*pid=$controller_pid"; then
                log_success "Controller启动成功 (PID: $controller_pid)"
                log_info "gRPC服务: 0.0.0.0:9090"
                log_info "HTTP服务: 0.0.0.0:8080"
                return 0
            fi
        fi
        sleep 1
    done
    
    log_error "Controller启动失败"
    if [[ -f "logs/controller.log" ]]; then
        log_info "错误日志:"
        tail -10 logs/controller.log
    fi
    rm -f "$CONTROLLER_PID_FILE"
    return 1
}

# 启动Agent
start_agent() {
    log_info "启动Agent..."
    
    # 检查是否已在运行
    if check_process_status "Agent" "$AGENT_PID_FILE" "$AGENT_BINARY" > /dev/null 2>&1; then
        log_info "Agent已在运行"
        return 0
    fi
    
    # 检查Controller是否运行
    if ! netstat -tlnp 2>/dev/null | grep -q ":9090 " && ! ss -tlnp 2>/dev/null | grep -q ":9090 "; then
        log_error "Controller未运行，请先启动Controller"
        return 1
    fi
    
    # 启动Agent
    cd "$PROJECT_ROOT"
    nohup "$AGENT_BINARY" -config "$AGENT_CONFIG" > logs/agent.log 2>&1 &
    local agent_pid=$!
    echo $agent_pid > "$AGENT_PID_FILE"
    
    # 等待启动
    sleep 3
    if ps -p $agent_pid > /dev/null 2>&1; then
        log_success "Agent启动成功 (PID: $agent_pid)"
        return 0
    else
        log_error "Agent启动失败"
        if [[ -f "logs/agent.log" ]]; then
            log_info "错误日志:"
            tail -10 logs/agent.log
        fi
        rm -f "$AGENT_PID_FILE"
        return 1
    fi
}

# 停止进程
stop_process() {
    local name=$1
    local pid_file=$2
    local binary=$3
    
    log_info "停止 $name..."
    
    local stopped=false
    
    # 从PID文件停止
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null 2>&1; then
            log_info "停止 $name 进程 (PID: $pid)"
            kill $pid 2>/dev/null || true
            
            # 等待进程退出
            for i in {1..10}; do
                if ! ps -p $pid > /dev/null 2>&1; then
                    log_success "$name 已停止"
                    stopped=true
                    break
                fi
                sleep 1
            done
            
            # 强制停止
            if ! $stopped && ps -p $pid > /dev/null 2>&1; then
                log_warning "强制停止 $name"
                kill -9 $pid 2>/dev/null || true
                stopped=true
            fi
        fi
        
        rm -f "$pid_file"
    fi
    
    # 停止所有相关进程
    local pids=$(pgrep -f "$binary" || true)
    if [[ -n "$pids" ]]; then
        log_warning "停止其他 $name 进程: $pids"
        echo $pids | xargs -r kill 2>/dev/null || true
        sleep 2
        
        # 强制停止
        local remaining_pids=$(pgrep -f "$binary" || true)
        if [[ -n "$remaining_pids" ]]; then
            echo $remaining_pids | xargs -r kill -9 2>/dev/null || true
        fi
        stopped=true
    fi
    
    if $stopped; then
        log_success "$name 已停止"
    else
        log_info "$name 未运行"
    fi
}

# 启动服务
start_services() {
    log_info "启动所有服务..."
    
    # 启动Controller
    if ! start_controller; then
        log_error "Controller启动失败"
        return 1
    fi
    
    # 启动Agent
    if ! start_agent; then
        log_error "Agent启动失败"
        return 1
    fi
    
    log_success "所有服务启动完成"
    check_services
}

# 停止服务
stop_services() {
    log_info "停止所有服务..."
    
    stop_process "Agent" "$AGENT_PID_FILE" "$AGENT_BINARY"
    stop_process "Controller" "$CONTROLLER_PID_FILE" "$CONTROLLER_BINARY"
    
    log_success "所有服务已停止"
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    echo "=========================================="
    check_process_status "Controller" "$CONTROLLER_PID_FILE" "$CONTROLLER_BINARY"
    
    # 检查Controller端口
    if netstat -tlnp 2>/dev/null | grep -q ":9090 " || ss -tlnp 2>/dev/null | grep -q ":9090 "; then
        log_success "Controller gRPC端口 (9090) 正常监听"
    else
        log_warning "Controller gRPC端口 (9090) 未监听"
    fi
    
    if netstat -tlnp 2>/dev/null | grep -q ":8080 " || ss -tlnp 2>/dev/null | grep -q ":8080 "; then
        log_success "Controller HTTP端口 (8080) 正常监听"
    else
        log_warning "Controller HTTP端口 (8080) 未监听"
    fi
    
    echo "=========================================="
    check_process_status "Agent" "$AGENT_PID_FILE" "$AGENT_BINARY"
    
    echo "=========================================="
    
    # 健康检查
    if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        log_success "Controller健康检查通过"
    else
        log_warning "Controller健康检查失败"
    fi
}

# 查看日志
show_logs() {
    local service=${1:-"all"}
    
    case $service in
        "controller")
            if [[ -f "logs/controller.log" ]]; then
                log_info "Controller日志 (最后50行):"
                tail -50 logs/controller.log
            else
                log_warning "Controller日志文件不存在"
            fi
            ;;
        "agent")
            if [[ -f "logs/agent.log" ]]; then
                log_info "Agent日志 (最后50行):"
                tail -50 logs/agent.log
            else
                log_warning "Agent日志文件不存在"
            fi
            ;;
        "all"|*)
            show_logs controller
            echo ""
            show_logs agent
            ;;
    esac
}

# 完整安装
install() {
    log_info "开始完整安装..."
    
    check_dependencies
    create_directories
    generate_configs
    build_applications
    start_services
    
    log_success "安装完成!"
    echo ""
    log_info "服务访问地址:"
    log_info "  Controller API: http://localhost:8080/api/v1/health"
    log_info "  Controller gRPC: localhost:9090"
    echo ""
    log_info "常用命令:"
    log_info "  查看状态: ./scripts/deploy.sh status"
    log_info "  查看日志: ./scripts/deploy.sh logs"
    log_info "  重启服务: ./scripts/deploy.sh restart"
    log_info "  停止服务: ./scripts/deploy.sh stop"
}

# 显示帮助
show_help() {
    echo "Xbox Sing-box管理系统部署脚本"
    echo ""
    echo "用法: ./scripts/deploy.sh [命令]"
    echo ""
    echo "命令:"
    echo "  install    完整安装系统"
    echo "  start      启动所有服务"
    echo "  stop       停止所有服务"
    echo "  restart    重启所有服务"
    echo "  status     查看服务状态"
    echo "  logs [服务] 查看日志"
    echo "             服务选项: controller, agent, all (默认)"
    echo "  build      只构建应用程序"
    echo "  help       显示此帮助"
    echo ""
    echo "示例:"
    echo "  ./scripts/deploy.sh install       # 完整安装"
    echo "  ./scripts/deploy.sh logs agent    # 查看Agent日志"
    echo "  ./scripts/deploy.sh restart       # 重启所有服务"
}

# 主函数
main() {
    case "${1:-help}" in
        "install")
            install
            ;;
        "start")
            start_services
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            stop_services
            sleep 2
            start_services
            ;;
        "status")
            check_services
            ;;
        "logs")
            show_logs "$2"
            ;;
        "build")
            check_dependencies
            build_applications
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"