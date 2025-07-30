#!/bin/bash

# Docker方式安装脚本（推荐用于开发环境）
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker是否安装
check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker未安装，请先安装Docker"
        log_info "Ubuntu/Debian: curl -fsSL https://get.docker.com | sh"
        log_info "macOS: 下载Docker Desktop"
        exit 1
    fi
    
    if ! command -v docker-compose >/dev/null 2>&1; then
        log_error "docker-compose未安装"
        exit 1
    fi
    
    log_info "Docker环境检查通过"
}

# 创建docker-compose.yml
create_docker_compose() {
    log_info "创建docker-compose.yml..."
    
    cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: xbox-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: xbox123456
      MYSQL_DATABASE: xbox_manager
      MYSQL_USER: xbox_user
      MYSQL_PASSWORD: xbox_password
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    command: --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      timeout: 20s
      retries: 10

  redis:
    image: redis:7-alpine
    container_name: xbox-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  # 开发环境Go容器（可选）
  dev:
    image: golang:1.21-alpine
    container_name: xbox-dev
    working_dir: /app
    volumes:
      - .:/app
      - go_modules:/go/pkg/mod
    environment:
      - CGO_ENABLED=1
      - GOPROXY=https://goproxy.cn,direct
    command: tail -f /dev/null
    depends_on:
      mysql:
        condition: service_healthy

volumes:
  mysql_data:
  redis_data:
  go_modules:
EOF

    log_info "docker-compose.yml创建完成"
}

# 启动服务
start_services() {
    log_info "启动Docker服务..."
    
    # 启动MySQL和Redis
    docker-compose up -d mysql redis
    
    # 等待MySQL启动
    log_info "等待MySQL启动..."
    timeout=60
    while ! docker-compose exec mysql mysqladmin ping -h localhost --silent; do
        sleep 2
        timeout=$((timeout - 2))
        if [ $timeout -le 0 ]; then
            log_error "MySQL启动超时"
            exit 1
        fi
    done
    
    log_info "MySQL启动完成"
    
    # 显示连接信息
    log_info "数据库连接信息:"
    log_info "Host: localhost:3306"
    log_info "Database: xbox_manager"
    log_info "Username: root"
    log_info "Password: xbox123456"
    log_info ""
    log_info "或使用普通用户:"
    log_info "Username: xbox_user"
    log_info "Password: xbox_password"
}

# 安装Go工具（在宿主机上）
install_go_tools() {
    log_info "检查Go环境..."
    
    if ! command -v go >/dev/null 2>&1; then
        log_warn "宿主机未安装Go，将使用Docker容器进行开发"
        return 0
    fi
    
    # 安装protoc插件
    log_info "安装protoc插件..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    
    log_info "Go工具安装完成"
}

# 验证环境
verify_environment() {
    log_info "验证Docker环境..."
    
    # 检查MySQL
    if docker-compose exec mysql mysql -u root -pxbox123456 -e "SELECT 1;" >/dev/null 2>&1; then
        log_info "✓ MySQL连接正常"
    else
        log_error "✗ MySQL连接失败"
        return 1
    fi
    
    # 检查Redis
    if docker-compose exec redis redis-cli ping >/dev/null 2>&1; then
        log_info "✓ Redis连接正常"
    else
        log_error "✗ Redis连接失败"
        return 1
    fi
    
    log_info "Docker环境验证完成！"
}

# 主函数
main() {
    log_info "Xbox Sing-box管理系统 Docker安装脚本"
    log_info "======================================="
    
    check_docker
    create_docker_compose
    start_services
    install_go_tools
    verify_environment
    
    log_info "Docker环境安装完成！"
    log_info ""
    log_info "下一步："
    log_info "1. 生成proto代码: make proto"
    log_info "2. 构建项目: make build"
    log_info "3. 停止服务: docker-compose down"
    log_info "4. 重启服务: docker-compose up -d"
    log_info ""
    log_info "使用Docker容器开发:"
    log_info "docker-compose exec dev sh"
}

# 解析命令行参数
case "${1:-}" in
    --stop)
        log_info "停止Docker服务..."
        docker-compose down
        ;;
    --restart)
        log_info "重启Docker服务..."
        docker-compose restart
        ;;
    --clean)
        log_info "清理Docker资源..."
        docker-compose down -v
        docker volume prune -f
        ;;
    --verify)
        verify_environment
        ;;
    *)
        main
        ;;
esac