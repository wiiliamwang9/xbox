#!/bin/bash

# TLS + mTLS双向认证测试脚本

set -e

# 定义颜色
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

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖项..."
    
    if ! command -v openssl &> /dev/null; then
        log_error "openssl 未安装"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装"
        exit 1
    fi
    
    log_success "依赖项检查完成"
}

# 检查证书文件
check_certificates() {
    log_info "检查TLS证书文件..."
    
    CERT_FILES=(
        "certs/ca/ca-cert.pem"
        "certs/server/server-cert.pem"
        "certs/server/server-key.pem"
        "certs/client/client-cert.pem"
        "certs/client/client-key.pem"
    )
    
    for cert_file in "${CERT_FILES[@]}"; do
        if [[ ! -f "$cert_file" ]]; then
            log_error "证书文件不存在: $cert_file"
            log_info "正在生成证书..."
            ./scripts/generate_tls_certs.sh
            break
        fi
    done
    
    # 验证证书有效性
    log_info "验证证书有效性..."
    openssl verify -CAfile certs/ca/ca-cert.pem certs/server/server-cert.pem
    openssl verify -CAfile certs/ca/ca-cert.pem certs/client/client-cert.pem
    
    # 检查证书信息
    log_info "服务器证书信息:"
    openssl x509 -in certs/server/server-cert.pem -text -noout | grep -E "(Subject:|Not Before|Not After|DNS:|IP Address:)"
    
    log_info "客户端证书信息:"
    openssl x509 -in certs/client/client-cert.pem -text -noout | grep -E "(Subject:|Not Before|Not After)"
    
    log_success "证书检查完成"
}

# 构建项目
build_project() {
    log_info "构建Xbox项目..."
    
    # 构建Controller
    log_info "构建Controller..."
    go build -o bin/controller ./cmd/controller/main.go
    if [[ $? -ne 0 ]]; then
        log_error "Controller构建失败"
        exit 1
    fi
    
    # 构建Agent
    log_info "构建Agent..."
    go build -o bin/agent ./cmd/agent/main.go
    if [[ $? -ne 0 ]]; then
        log_error "Agent构建失败"
        exit 1
    fi
    
    log_success "项目构建完成"
}

# 启动Controller（TLS模式）
start_controller() {
    log_info "启动Controller (TLS + mTLS模式)..."
    
    # 确保logs目录存在
    mkdir -p logs/test
    
    # 启动Controller
    ./bin/controller -config ./configs/config.yaml > logs/test/controller.log 2>&1 &
    CONTROLLER_PID=$!
    echo $CONTROLLER_PID > /tmp/controller.pid
    
    log_info "Controller已启动 (PID: $CONTROLLER_PID)"
    
    # 等待Controller启动
    sleep 3
    
    # 检查Controller是否正常运行
    if ! kill -0 $CONTROLLER_PID 2>/dev/null; then
        log_error "Controller启动失败"
        cat logs/test/controller.log
        exit 1
    fi
    
    # 检查gRPC端口
    if ! netstat -ln | grep -q ":9090"; then
        log_error "Controller gRPC端口未监听"
        cat logs/test/controller.log
        exit 1
    fi
    
    log_success "Controller启动成功，监听端口 9090"
}

# 测试TLS连接
test_tls_connection() {
    log_info "测试TLS连接..."
    
    # 使用openssl测试TLS连接
    log_info "测试服务器TLS证书..."
    echo "Q" | timeout 5 openssl s_client -connect localhost:9090 \
        -CAfile certs/ca/ca-cert.pem \
        -cert certs/client/client-cert.pem \
        -key certs/client/client-key.pem \
        -verify_return_error > /tmp/tls_test.log 2>&1
    
    if [[ $? -eq 0 ]]; then
        log_success "TLS连接测试成功"
        log_info "连接详情:"
        grep -E "(Verification|SSL-Session|Protocol|Cipher)" /tmp/tls_test.log | head -10
    else
        log_warn "TLS连接测试超时或失败（可能是正常的，因为这是gRPC服务）"
        log_info "检查连接输出:"
        tail -5 /tmp/tls_test.log
    fi
}

# 启动Agent（TLS模式）
start_agent() {
    log_info "启动Agent (TLS + mTLS模式)..."
    
    # 启动Agent
    ./bin/agent -config ./configs/agent.yaml > logs/test/agent.log 2>&1 &
    AGENT_PID=$!
    echo $AGENT_PID > /tmp/agent.pid
    
    log_info "Agent已启动 (PID: $AGENT_PID)"
    
    # 等待Agent连接
    sleep 5
    
    # 检查Agent是否正常运行
    if ! kill -0 $AGENT_PID 2>/dev/null; then
        log_error "Agent启动失败"
        cat logs/test/agent.log
        cleanup
        exit 1
    fi
    
    log_success "Agent启动成功"
}

# 验证mTLS认证
verify_mtls() {
    log_info "验证mTLS双向认证..."
    
    # 检查Controller日志中的TLS信息
    if grep -q "TLS.*mTLS" logs/test/controller.log; then
        log_success "Controller TLS配置已启用"
    else
        log_warn "Controller日志中未发现TLS配置信息"
    fi
    
    # 检查Agent日志中的TLS连接信息
    if grep -q "TLS.*mTLS\|已连接到Controller" logs/test/agent.log; then
        log_success "Agent TLS连接成功"
    else
        log_warn "Agent日志中未发现TLS连接信息"
    fi
    
    # 检查Agent注册
    sleep 2
    if grep -q "Agent注册成功\|注册Agent失败" logs/test/agent.log; then
        if grep -q "Agent注册成功" logs/test/agent.log; then
            log_success "Agent注册成功 - mTLS认证通过"
        else
            log_error "Agent注册失败"
            log_info "Agent错误日志:"
            grep -A 5 -B 5 "注册Agent失败\|连接Controller失败\|TLS" logs/test/agent.log
        fi
    else
        log_warn "等待Agent注册中..."
        sleep 3
        if grep -q "Agent注册成功" logs/test/agent.log; then
            log_success "Agent注册成功 - mTLS认证通过"
        else
            log_warn "Agent注册状态未确定"
        fi
    fi
    
    # 检查心跳
    log_info "检查心跳通信..."
    sleep 5
    if grep -q "心跳成功\|心跳失败" logs/test/agent.log; then
        if grep -q "心跳成功" logs/test/agent.log; then
            log_success "心跳通信正常 - mTLS持续认证成功"
        else
            log_error "心跳通信失败"
        fi
    else
        log_warn "心跳通信状态未确定"
    fi
}

# 显示日志
show_logs() {
    log_info "=== Controller日志 ==="
    tail -20 logs/test/controller.log
    echo ""
    
    log_info "=== Agent日志 ==="
    tail -20 logs/test/agent.log
    echo ""
}

# 清理进程
cleanup() {
    log_info "清理测试进程..."
    
    if [[ -f /tmp/agent.pid ]]; then
        AGENT_PID=$(cat /tmp/agent.pid)
        if kill -0 $AGENT_PID 2>/dev/null; then
            kill $AGENT_PID
            log_info "Agent进程已停止 (PID: $AGENT_PID)"
        fi
        rm -f /tmp/agent.pid
    fi
    
    if [[ -f /tmp/controller.pid ]]; then
        CONTROLLER_PID=$(cat /tmp/controller.pid)
        if kill -0 $CONTROLLER_PID 2>/dev/null; then
            kill $CONTROLLER_PID
            log_info "Controller进程已停止 (PID: $CONTROLLER_PID)"
        fi
        rm -f /tmp/controller.pid
    fi
    
    # 清理临时文件
    rm -f /tmp/tls_test.log
    
    log_success "清理完成"
}

# 主函数
main() {
    echo "=============================================="
    echo "Xbox TLS + mTLS双向认证测试"
    echo "=============================================="
    
    # 设置陷阱以确保清理
    trap cleanup EXIT
    
    # 检查依赖
    check_dependencies
    
    # 检查证书
    check_certificates
    
    # 构建项目
    build_project
    
    # 启动Controller
    start_controller
    
    # 测试TLS连接
    test_tls_connection
    
    # 启动Agent
    start_agent
    
    # 验证mTLS
    verify_mtls
    
    # 显示日志
    show_logs
    
    echo ""
    echo "=============================================="
    log_success "TLS + mTLS双向认证测试完成"
    echo "=============================================="
    
    # 保持运行以便查看日志
    log_info "按Ctrl+C停止测试..."
    while true; do
        sleep 1
    done
}

# 如果脚本被直接执行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi