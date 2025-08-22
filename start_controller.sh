#!/bin/bash

# Xbox Controller 启动脚本
# 自动检查并清理端口占用，启动Controller服务并输出日志到文件
#
# 使用方法:
#   ./start_controller.sh           # 启动后立即退出(适合测试)
#   ./start_controller.sh --daemon  # daemon模式，保持运行(适合生产环境)
#   ./start_controller.sh -d        # daemon模式简写

set -e

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/xbox_controller.log"
CONTROLLER_BINARY="${SCRIPT_DIR}/cmd/controller/controller"
CONFIG_FILE="${SCRIPT_DIR}/configs/config.yaml"

# Controller使用的端口 (从config.yaml获取)
HTTP_PORT=9000
GRPC_PORT=9090
ENVOY_HTTP_PORT=8080
PROMETHEUS_PORT=2112

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $1" >> "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARNING] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $1" >> "$LOG_FILE"
}

# 检查端口是否被占用
check_port_usage() {
    local port=$1
    local pid=$(lsof -ti:$port 2>/dev/null || true)
    echo "$pid"
}

# 强制终止占用端口的进程
kill_port_process() {
    local port=$1
    local process_name=$2
    
    log "检查端口 $port 是否被占用..."
    local pids=$(check_port_usage $port)
    
    if [ -n "$pids" ]; then
        log_warning "端口 $port 被以下进程占用:"
        for pid in $pids; do
            local cmd=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
            log_warning "  PID: $pid, Command: $cmd"
        done
        
        log "强制终止占用端口 $port 的进程..."
        for pid in $pids; do
            if kill -9 $pid 2>/dev/null; then
                log_success "已终止进程 PID: $pid"
            else
                log_error "无法终止进程 PID: $pid"
            fi
        done
        
        # 等待进程完全终止
        sleep 2
        
        # 再次检查
        local remaining_pids=$(check_port_usage $port)
        if [ -n "$remaining_pids" ]; then
            log_error "端口 $port 仍被占用，无法启动$process_name"
            return 1
        else
            log_success "端口 $port 已清理完成"
        fi
    else
        log "端口 $port 未被占用"
    fi
    return 0
}

# 检查必要的文件和目录
check_prerequisites() {
    log "检查启动必要条件..."
    
    # 检查配置文件
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "配置文件不存在: $CONFIG_FILE"
        return 1
    fi
    log_success "配置文件存在: $CONFIG_FILE"
    
    # 检查Go环境
    if ! command -v go &> /dev/null; then
        log_error "Go环境未安装"
        return 1
    fi
    log_success "Go环境检查通过: $(go version)"
    
    # 创建日志目录
    mkdir -p "$(dirname "$LOG_FILE")"
    log_success "日志文件: $LOG_FILE"
    
    return 0
}

# 构建Controller二进制文件
build_controller() {
    log "构建Controller应用..."
    
    cd "$SCRIPT_DIR"
    
    # 生成protobuf代码
    if [ -f "Makefile" ]; then
        if make proto 2>&1 | tee -a "$LOG_FILE"; then
            log_success "Protobuf代码生成完成"
        else
            log_error "Protobuf代码生成失败"
            return 1
        fi
    fi
    
    # 构建Controller
    if go build -o "$CONTROLLER_BINARY" ./cmd/controller 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Controller构建完成"
    else
        log_error "Controller构建失败"
        return 1
    fi
    
    return 0
}

# 检查TLS证书
check_tls_certificates() {
    log "检查TLS证书..."
    
    local cert_dir="${SCRIPT_DIR}/certs"
    local required_certs=(
        "ca/ca-cert.pem"
        "server/server-cert.pem"
        "server/server-key.pem"
    )
    
    for cert in "${required_certs[@]}"; do
        local cert_path="${cert_dir}/${cert}"
        if [ ! -f "$cert_path" ]; then
            log_warning "TLS证书缺失: $cert_path"
            log "尝试生成TLS证书..."
            
            if [ -f "${SCRIPT_DIR}/scripts/generate_tls_certs.sh" ]; then
                if bash "${SCRIPT_DIR}/scripts/generate_tls_certs.sh" 2>&1 | tee -a "$LOG_FILE"; then
                    log_success "TLS证书生成完成"
                    break
                else
                    log_error "TLS证书生成失败"
                    return 1
                fi
            else
                log_error "TLS证书生成脚本不存在"
                return 1
            fi
        fi
    done
    
    log_success "TLS证书检查完成"
    return 0
}

# 启动Controller服务
start_controller() {
    log "启动Xbox Controller服务..."
    
    cd "$SCRIPT_DIR"
    
    # 设置环境变量
    export XBOX_CONFIG_FILE="$CONFIG_FILE"
    export XBOX_LOG_LEVEL="info"
    
    # 启动Controller (后台运行，输出到日志文件)
    nohup "$CONTROLLER_BINARY" -config "$CONFIG_FILE" >> "$LOG_FILE" 2>&1 &
    local controller_pid=$!
    
    # 保存PID到文件
    echo $controller_pid > "${SCRIPT_DIR}/controller.pid"
    
    log "Controller已启动，PID: $controller_pid"
    log "日志文件: $LOG_FILE"
    
    # 等待服务启动
    log "等待Controller服务启动..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://localhost:$HTTP_PORT/api/v1/health" > /dev/null 2>&1; then
            log_success "Controller HTTP API服务已就绪 (端口: $HTTP_PORT)"
            break
        fi
        
        if ! kill -0 $controller_pid 2>/dev/null; then
            log_error "Controller进程已退出"
            return 1
        fi
        
        attempt=$((attempt + 1))
        log "等待服务启动... ($attempt/$max_attempts)"
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "Controller服务启动超时"
        return 1
    fi
    
    # 检查gRPC服务
    if command -v grpcurl &> /dev/null; then
        if grpcurl -plaintext localhost:$GRPC_PORT list > /dev/null 2>&1; then
            log_success "Controller gRPC服务已就绪 (端口: $GRPC_PORT)"
        else
            log_warning "gRPC服务检查失败，但HTTP API正常"
        fi
    else
        log_warning "grpcurl未安装，跳过gRPC服务检查"
    fi
    
    log_success "Xbox Controller启动完成！"
    log "服务信息:"
    log "  - HTTP API: http://localhost:$HTTP_PORT"
    log "  - gRPC服务: localhost:$GRPC_PORT"
    log "  - 进程PID: $controller_pid"
    log "  - 日志文件: $LOG_FILE"
    log "  - PID文件: ${SCRIPT_DIR}/controller.pid"
    
    return 0
}

# 清理函数
cleanup() {
    log "执行清理操作..."
    
    # 如果有PID文件，尝试优雅关闭
    if [ -f "${SCRIPT_DIR}/controller.pid" ]; then
        local pid=$(cat "${SCRIPT_DIR}/controller.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log "发送SIGTERM信号到进程 $pid"
            kill -TERM "$pid" 2>/dev/null || true
            sleep 3
            
            if kill -0 "$pid" 2>/dev/null; then
                log "强制终止进程 $pid"
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm -f "${SCRIPT_DIR}/controller.pid"
    fi
}

# 主函数
main() {
    log "========================================="
    log "Xbox Controller 启动脚本"
    log "========================================="
    
    # 检查是否以daemon模式运行
    DAEMON_MODE=false
    if [[ "$1" == "--daemon" || "$1" == "-d" ]]; then
        DAEMON_MODE=true
        log "以daemon模式启动Controller"
    fi
    
    # 检查必要条件
    if ! check_prerequisites; then
        log_error "启动条件检查失败"
        exit 1
    fi
    
    # 清理端口占用
    log "清理端口占用..."
    kill_port_process $HTTP_PORT "Controller HTTP" || exit 1
    kill_port_process $GRPC_PORT "Controller gRPC" || exit 1
    kill_port_process $ENVOY_HTTP_PORT "Envoy HTTP Proxy" || true  # Envoy可选
    
    # 检查TLS证书
    if ! check_tls_certificates; then
        log_error "TLS证书检查失败"
        exit 1
    fi
    
    # 构建应用
    if ! build_controller; then
        log_error "构建失败"
        exit 1
    fi
    
    # 启动Controller
    if ! start_controller; then
        log_error "启动失败"
        exit 1
    fi
    
    log_success "Xbox Controller启动脚本执行完成"
    log "使用 'tail -f $LOG_FILE' 查看实时日志"
    log "使用 'kill \$(cat ${SCRIPT_DIR}/controller.pid)' 停止服务"
    
    # 如果是daemon模式，保持运行并监控
    if [ "$DAEMON_MODE" = true ]; then
        log "Controller运行在daemon模式，脚本将保持运行状态"
        log "按Ctrl+C停止Controller服务"
        
        # 设置信号处理用于优雅关闭
        trap 'log "收到停止信号，正在关闭Controller..."; cleanup; exit 0' INT TERM
        
        # 监控Controller进程状态
        local controller_pid=$(cat "${SCRIPT_DIR}/controller.pid" 2>/dev/null || echo "")
        if [ -n "$controller_pid" ]; then
            # 等待Controller进程结束或接收信号
            while kill -0 "$controller_pid" 2>/dev/null; do
                sleep 5
            done
            log_warning "Controller进程意外退出"
        fi
        
        # 进程意外退出时执行清理
        cleanup
    else
        # 普通模式：启动后直接退出，不做清理
        log "启动完成，Controller在后台运行"
    fi
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi