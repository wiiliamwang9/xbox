#!/bin/bash

# Xbox Agent 启动流程测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[信息]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[成功]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[警告]${NC} $1"
}

print_error() {
    echo -e "${RED}[错误]${NC} $1"
}

print_separator() {
    echo "=========================================="
}

# 环境检查
check_environment() {
    print_info "检查环境依赖..."
    
    # 检查Go环境
    if ! command -v go &> /dev/null; then
        print_error "Go未安装或不在PATH中"
        exit 1
    fi
    print_success "Go环境: $(go version)"
    
    # 检查必要工具
    for tool in curl tar; do
        if ! command -v $tool &> /dev/null; then
            print_error "$tool 未安装"
            exit 1
        fi
    done
    print_success "必要工具检查通过"
}

# 构建项目
build_project() {
    print_info "构建Xbox项目..."
    
    # 生成protobuf代码
    if ! make proto; then
        print_error "生成protobuf代码失败"
        exit 1
    fi
    
    # 构建应用
    if ! make build; then
        print_error "构建应用失败"
        exit 1
    fi
    
    print_success "项目构建完成"
}

# 准备测试环境
prepare_test_environment() {
    print_info "准备测试环境..."
    
    # 创建必要目录
    mkdir -p ./bin
    mkdir -p ./configs
    mkdir -p ./logs
    
    # 创建测试配置文件
    if [[ ! -f "./configs/agent.yaml" ]]; then
        cat > ./configs/agent.yaml << EOF
agent:
  id: "test-agent-$(date +%s)"
  controller_addr: "localhost:9090"
  heartbeat_interval: 30
  sing_box_binary: "./bin/sing-box"
  sing_box_config: "./configs/sing-box.json"

log:
  level: "info"
  output: "./logs/agent.log"
EOF
        print_success "创建Agent配置文件: ./configs/agent.yaml"
    fi
    
    # 创建基础的sing-box配置
    if [[ ! -f "./configs/sing-box.json" ]]; then
        cat > ./configs/sing-box.json << EOF
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
    ],
    "strategy": "prefer_ipv4"
  },
  "inbounds": [
    {
      "type": "socks",
      "tag": "socks",
      "listen": "127.0.0.1",
      "listen_port": 1080,
      "sniff": true,
      "sniff_override_destination": true
    },
    {
      "type": "http",
      "tag": "http",
      "listen": "127.0.0.1",
      "listen_port": 8888,
      "sniff": true,
      "sniff_override_destination": true
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
        print_success "创建sing-box配置文件: ./configs/sing-box.json"
    fi
    
    print_success "测试环境准备完成"
}

# 测试Agent启动流程
test_agent_startup() {
    print_info "测试Agent启动流程..."
    
    # 设置超时
    TIMEOUT=60
    
    # 启动Agent（后台）
    print_info "启动Agent服务..."
    
    # 创建日志文件
    touch ./logs/agent.log
    
    # 启动Agent并输出到日志文件
    ./bin/agent -config ./configs/agent.yaml > ./logs/agent_startup.log 2>&1 &
    AGENT_PID=$!
    
    print_info "Agent PID: $AGENT_PID"
    
    # 等待启动完成
    print_info "等待Agent启动完成..."
    sleep 10
    
    # 检查进程是否还在运行
    if ! kill -0 $AGENT_PID 2>/dev/null; then
        print_error "Agent启动失败"
        print_info "启动日志内容:"
        cat ./logs/agent_startup.log
        exit 1
    fi
    
    print_success "Agent进程正在运行"
    
    # 检查日志内容
    print_info "检查启动日志..."
    
    # 等待日志写入
    sleep 5
    
    # 检查关键日志
    check_log_patterns=(
        "Xbox Agent.*启动中"
        "检查sing-box安装状态"
        "sing-box.*安装"
        "Agent服务已启动"
        "=== sing-box 配置信息 ==="
        "配置信息输出完成"
    )
    
    print_info "检查启动日志关键信息..."
    for pattern in "${check_log_patterns[@]}"; do
        if grep -q "$pattern" ./logs/agent_startup.log; then
            print_success "✓ $pattern"
        else
            print_warning "✗ 未找到: $pattern"
        fi
    done
    
    # 显示配置信息部分
    print_info "显示sing-box配置信息输出..."
    print_separator
    if grep -A 50 "=== sing-box 配置信息 ===" ./logs/agent_startup.log; then
        print_success "配置信息输出正常"
    else
        print_warning "未找到配置信息输出"
    fi
    print_separator
    
    # 停止Agent
    print_info "停止Agent服务..."
    kill $AGENT_PID 2>/dev/null || true
    sleep 3
    
    # 强制停止（如果还在运行）
    if kill -0 $AGENT_PID 2>/dev/null; then
        print_warning "强制停止Agent"
        kill -9 $AGENT_PID 2>/dev/null || true
    fi
    
    print_success "Agent测试完成"
}

# 检查sing-box安装
check_singbox_installation() {
    print_info "检查sing-box安装情况..."
    
    # 检查二进制文件
    if [[ -f "./bin/sing-box" ]]; then
        print_success "sing-box二进制文件已安装: ./bin/sing-box"
        
        # 检查版本
        if ./bin/sing-box version; then
            print_success "sing-box版本信息正常"
        else
            print_warning "无法获取sing-box版本信息"
        fi
    else
        print_warning "未找到sing-box二进制文件"
    fi
    
    # 检查系统PATH中的sing-box
    if command -v sing-box &> /dev/null; then
        print_success "系统PATH中存在sing-box: $(which sing-box)"
        print_info "版本: $(sing-box version 2>&1 | head -n 1)"
    else
        print_info "系统PATH中未找到sing-box"
    fi
}

# 清理测试环境
cleanup() {
    print_info "清理测试环境..."
    
    # 停止可能运行的Agent进程
    pkill -f "./bin/agent" 2>/dev/null || true
    
    # 停止可能运行的sing-box进程
    pkill -f "sing-box" 2>/dev/null || true
    
    print_success "清理完成"
}

# 显示测试结果
show_results() {
    print_separator
    print_info "测试结果总结"
    print_separator
    
    print_info "生成的文件:"
    ls -la ./logs/agent_startup.log 2>/dev/null && print_success "- 启动日志: ./logs/agent_startup.log"
    ls -la ./configs/agent.yaml 2>/dev/null && print_success "- Agent配置: ./configs/agent.yaml"
    ls -la ./configs/sing-box.json 2>/dev/null && print_success "- sing-box配置: ./configs/sing-box.json"
    ls -la ./bin/sing-box 2>/dev/null && print_success "- sing-box二进制: ./bin/sing-box"
    
    print_separator
    print_info "如需查看完整启动日志:"
    print_info "cat ./logs/agent_startup.log"
    print_separator
}

# 主函数
main() {
    print_separator
    print_info "Xbox Agent 启动流程测试"
    print_separator
    
    # 捕获退出信号，确保清理
    trap cleanup EXIT
    
    check_environment
    build_project
    prepare_test_environment
    test_agent_startup
    check_singbox_installation
    show_results
    
    print_success "所有测试完成!"
}

# 执行主函数
main "$@"