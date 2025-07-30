#!/bin/bash

# Xbox Sing-box管理系统部署脚本
# 使用方法: ./scripts/deploy.sh [action] [options]

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"
ENV_FILE="$PROJECT_ROOT/.env"

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
    log_info "Checking dependencies..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # 检查Docker daemon是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    log_success "All dependencies are satisfied"
}

# 创建必要的目录
create_directories() {
    log_info "Creating necessary directories..."
    
    mkdir -p "$PROJECT_ROOT/logs"
    mkdir -p "$PROJECT_ROOT/data"
    mkdir -p "$PROJECT_ROOT/monitoring/grafana/dashboards"
    mkdir -p "$PROJECT_ROOT/monitoring/grafana/provisioning"
    
    log_success "Directories created"
}

# 生成配置文件
generate_configs() {
    log_info "Generating configuration files..."
    
    # 创建.env文件
    if [ ! -f "$ENV_FILE" ]; then
        cat > "$ENV_FILE" << EOF
# Xbox管理系统环境变量

# 数据库配置
MYSQL_ROOT_PASSWORD=xbox123456
MYSQL_DATABASE=xbox_manager
MYSQL_USER=xbox
MYSQL_PASSWORD=xbox123456

# 应用配置
XBOX_VERSION=1.0.0
XBOX_ENVIRONMENT=production

# 监控配置
GRAFANA_ADMIN_PASSWORD=xbox123456

# 网络配置
XBOX_SUBNET=172.20.0.0/16
EOF
        log_success "Environment file created: $ENV_FILE"
    else
        log_info "Environment file already exists: $ENV_FILE"
    fi
}

# 构建镜像
build_images() {
    log_info "Building Docker images..."
    
    cd "$PROJECT_ROOT"
    
    # 构建Controller镜像
    log_info "Building Controller image..."
    docker build -f Dockerfile.controller -t xbox/controller:latest .
    
    # 构建Agent镜像
    log_info "Building Agent image..."
    docker build -f Dockerfile.agent -t xbox/agent:latest .
    
    log_success "Docker images built successfully"
}

# 启动服务
start_services() {
    log_info "Starting services..."
    
    cd "$PROJECT_ROOT"
    
    # 启动所有服务
    docker-compose up -d
    
    log_success "All services started"
    
    # 等待服务启动
    log_info "Waiting for services to be ready..."
    sleep 30
    
    # 检查服务状态
    check_services
}

# 停止服务
stop_services() {
    log_info "Stopping services..."
    
    cd "$PROJECT_ROOT"
    docker-compose down
    
    log_success "All services stopped"
}

# 重启服务
restart_services() {
    log_info "Restarting services..."
    
    stop_services
    start_services
    
    log_success "All services restarted"
}

# 检查服务状态
check_services() {
    log_info "Checking service status..."
    
    cd "$PROJECT_ROOT"
    
    # 显示容器状态
    docker-compose ps
    
    # 检查健康状态
    log_info "Checking health status..."
    
    # 检查Controller健康状态
    if curl -f http://localhost:8080/api/v1/health &> /dev/null; then
        log_success "Controller is healthy"
    else
        log_warning "Controller health check failed"
    fi
    
    # 检查Agent健康状态
    if curl -f http://localhost:8081/health &> /dev/null; then
        log_success "Agent is healthy"
    else
        log_warning "Agent health check failed"
    fi
    
    # 检查数据库连接
    if docker-compose exec -T mysql mysqladmin ping -h localhost -u root -pxbox123456 &> /dev/null; then
        log_success "MySQL is healthy"
    else
        log_warning "MySQL health check failed"
    fi
}

# 查看日志
show_logs() {
    local service="$1"
    
    cd "$PROJECT_ROOT"
    
    if [ -n "$service" ]; then
        log_info "Showing logs for service: $service"
        docker-compose logs -f "$service"
    else
        log_info "Showing logs for all services"
        docker-compose logs -f
    fi
}

# 备份数据
backup_data() {
    local backup_dir="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
    
    log_info "Creating backup in: $backup_dir"
    mkdir -p "$backup_dir"
    
    cd "$PROJECT_ROOT"
    
    # 备份数据库
    log_info "Backing up MySQL database..."
    docker-compose exec -T mysql mysqldump -u root -pxbox123456 xbox_manager > "$backup_dir/mysql_backup.sql"
    
    # 备份配置文件
    log_info "Backing up configuration files..."
    cp -r configs "$backup_dir/"
    
    # 备份数据目录
    if [ -d "data" ]; then
        log_info "Backing up data directory..."
        cp -r data "$backup_dir/"
    fi
    
    log_success "Backup completed: $backup_dir"
}

# 清理资源
cleanup() {
    log_info "Cleaning up resources..."
    
    cd "$PROJECT_ROOT"
    
    # 停止并删除容器
    docker-compose down -v
    
    # 删除未使用的镜像
    docker image prune -f
    
    # 删除未使用的卷
    docker volume prune -f
    
    log_success "Cleanup completed"
}

# 更新系统
update_system() {
    log_info "Updating system..."
    
    # 拉取最新代码
    if [ -d "$PROJECT_ROOT/.git" ]; then
        log_info "Pulling latest code..."
        cd "$PROJECT_ROOT"
        git pull
    fi
    
    # 重新构建镜像
    build_images
    
    # 重启服务
    restart_services
    
    log_success "System updated successfully"
}

# 显示帮助信息
show_help() {
    echo "Xbox Sing-box管理系统部署脚本"
    echo ""
    echo "使用方法:"
    echo "  $0 [command] [options]"
    echo ""
    echo "命令:"
    echo "  install      - 安装并启动系统"
    echo "  start        - 启动所有服务"
    echo "  stop         - 停止所有服务"
    echo "  restart      - 重启所有服务"
    echo "  status       - 检查服务状态"
    echo "  logs [service] - 查看日志 (可选指定服务名)"
    echo "  backup       - 备份数据"
    echo "  cleanup      - 清理资源"
    echo "  update       - 更新系统"
    echo "  build        - 构建Docker镜像"
    echo "  help         - 显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 install           # 完整安装"
    echo "  $0 logs controller   # 查看Controller日志"
    echo "  $0 backup           # 备份数据"
}

# 完整安装
install_system() {
    log_info "Starting Xbox system installation..."
    
    check_dependencies
    create_directories
    generate_configs
    build_images
    start_services
    
    log_success "Xbox system installation completed!"
    
    echo ""
    echo "访问地址:"
    echo "  Controller API: http://localhost:8080"
    echo "  Agent API: http://localhost:8081"
    echo "  Prometheus: http://localhost:9000"
    echo "  Grafana: http://localhost:3000 (admin/xbox123456)"
    echo ""
}

# 主函数
main() {
    case "${1:-help}" in
        install)
            install_system
            ;;
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        status)
            check_services
            ;;
        logs)
            show_logs "$2"
            ;;
        backup)
            backup_data
            ;;
        cleanup)
            cleanup
            ;;
        update)
            update_system
            ;;
        build)
            build_images
            ;;
        help|*)
            show_help
            ;;
    esac
}

# 执行主函数
main "$@"