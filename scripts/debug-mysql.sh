#!/bin/bash

# MySQL连接调试脚本
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

# 检查是否为root用户
is_root() {
    [[ $EUID -eq 0 ]]
}

# 执行命令（自动处理sudo）
run_cmd() {
    if is_root; then
        "$@"
    else
        sudo "$@"
    fi
}

# 检查MySQL服务状态
check_mysql_service() {
    log_info "检查MySQL服务状态..."
    
    if command -v systemctl >/dev/null 2>&1; then
        # 尝试不同的服务名
        local service_names=("mysql" "mysqld" "mariadb")
        local service_found=false
        
        for service in "${service_names[@]}"; do
            if systemctl list-unit-files | grep -q "^${service}.service"; then
                service_found=true
                if systemctl is-active --quiet "$service"; then
                    log_info "✓ MySQL服务($service)正在运行"
                    return 0
                else
                    log_error "✗ MySQL服务($service)未运行"
                    log_info "尝试启动MySQL服务..."
                    if run_cmd systemctl start "$service"; then
                        log_info "✓ MySQL服务启动成功"
                        return 0
                    else
                        log_error "✗ MySQL服务启动失败"
                    fi
                fi
                break
            fi
        done
        
        if [[ "$service_found" == "false" ]]; then
            log_error "✗ 未找到MySQL服务，可能未安装"
            return 1
        fi
    elif command -v service >/dev/null 2>&1; then
        if service mysql status >/dev/null 2>&1; then
            log_info "✓ MySQL服务正在运行"
        else
            log_error "✗ MySQL服务未运行"
            log_info "尝试启动MySQL服务..."
            run_cmd service mysql start
        fi
    else
        log_warn "无法检查MySQL服务状态"
    fi
}

# 检查MySQL端口
check_mysql_port() {
    log_info "检查MySQL端口3306..."
    
    if netstat -tlnp 2>/dev/null | grep -q ":3306 "; then
        log_info "✓ MySQL端口3306已监听"
        netstat -tlnp 2>/dev/null | grep ":3306 "
    else
        log_error "✗ MySQL端口3306未监听"
    fi
}

# 测试root连接（无密码）
test_root_no_password() {
    log_info "测试root用户无密码连接..."
    
    if mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
        log_info "✓ root用户可以无密码连接"
        return 0
    else
        log_warn "✗ root用户无法无密码连接"
        return 1
    fi
}

# 重置root密码
reset_root_password() {
    log_info "重置MySQL root密码..."
    
    # 停止MySQL服务
    log_info "停止MySQL服务..."
    run_cmd systemctl stop mysql 2>/dev/null || run_cmd service mysql stop 2>/dev/null || true
    
    # 创建临时初始化文件
    local init_file="/tmp/mysql-init"
    cat > "$init_file" << 'EOF'
ALTER USER 'root'@'localhost' IDENTIFIED BY 'xbox123456';
FLUSH PRIVILEGES;
EOF
    
    # 以安全模式启动MySQL
    log_info "以安全模式启动MySQL..."
    mysqld_safe --skip-grant-tables --init-file="$init_file" &
    local mysql_pid=$!
    
    # 等待MySQL启动
    sleep 10
    
    # 杀死安全模式进程
    kill $mysql_pid 2>/dev/null || true
    sleep 5
    
    # 正常启动MySQL
    run_cmd systemctl start mysql 2>/dev/null || run_cmd service mysql start 2>/dev/null || true
    
    # 清理临时文件
    rm -f "$init_file"
    
    log_info "root密码已重置为: xbox123456"
}

# 使用socket连接设置密码
setup_password_via_socket() {
    log_info "通过socket连接设置密码..."
    
    # 查找MySQL socket文件
    local socket_file=""
    for path in /var/run/mysqld/mysqld.sock /tmp/mysql.sock /var/lib/mysql/mysql.sock; do
        if [[ -S "$path" ]]; then
            socket_file="$path"
            break
        fi
    done
    
    if [[ -z "$socket_file" ]]; then
        log_error "未找到MySQL socket文件"
        return 1
    fi
    
    log_info "找到socket文件: $socket_file"
    
    # 通过socket连接设置密码
    if mysql -u root --socket="$socket_file" -e "ALTER USER 'root'@'localhost' IDENTIFIED BY 'xbox123456'; FLUSH PRIVILEGES;" 2>/dev/null; then
        log_info "✓ 密码设置成功"
        return 0
    else
        log_error "✗ 密码设置失败"
        return 1
    fi
}

# 创建数据库和用户
create_database() {
    local password="xbox123456"
    
    log_info "创建数据库和用户..."
    
    # 测试连接
    if ! mysql -u root -p"$password" -e "SELECT 1;" >/dev/null 2>&1; then
        log_error "无法连接MySQL，密码可能不正确"
        return 1
    fi
    
    # 创建数据库
    mysql -u root -p"$password" -e "
        CREATE DATABASE IF NOT EXISTS xbox_manager DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
        CREATE USER IF NOT EXISTS 'xbox_user'@'%' IDENTIFIED BY 'xbox_password';
        GRANT ALL PRIVILEGES ON xbox_manager.* TO 'xbox_user'@'%';
        FLUSH PRIVILEGES;
    "
    
    log_info "✓ 数据库和用户创建完成"
    
    # 执行初始化脚本
    if [[ -f "scripts/init.sql" ]]; then
        mysql -u root -p"$password" < scripts/init.sql
        log_info "✓ 数据库初始化完成"
    fi
}

# 显示连接信息
show_connection_info() {
    log_info "MySQL连接信息:"
    echo "Host: localhost"
    echo "Port: 3306"
    echo "Root Password: xbox123456"
    echo "Database: xbox_manager"
    echo "User: xbox_user"
    echo "User Password: xbox_password"
    echo ""
    echo "测试连接命令:"
    echo "mysql -u root -pxbox123456 -e 'SELECT 1;'"
    echo "mysql -u xbox_user -pxbox_password xbox_manager -e 'SHOW TABLES;'"
}

# 主函数
main() {
    log_info "MySQL连接问题调试脚本"
    log_info "========================"
    
    check_mysql_service
    check_mysql_port
    
    # 尝试无密码连接
    if test_root_no_password; then
        # 设置密码
        mysql -u root -e "ALTER USER 'root'@'localhost' IDENTIFIED BY 'xbox123456'; FLUSH PRIVILEGES;"
        log_info "✓ root密码已设置"
    else
        # 尝试用常见密码连接
        local common_passwords=("" "root" "password" "123456" "xbox123456")
        local success=false
        
        for pwd in "${common_passwords[@]}"; do
            if [[ -z "$pwd" ]]; then
                continue
            fi
            
            if mysql -u root -p"$pwd" -e "SELECT 1;" >/dev/null 2>&1; then
                log_info "✓ 找到可用密码: $pwd"
                success=true
                break
            fi
        done
        
        if [[ "$success" == "false" ]]; then
            log_warn "无法使用常见密码连接，尝试重置密码..."
            
            # 尝试socket连接
            if ! setup_password_via_socket; then
                # 完全重置
                reset_root_password
            fi
        fi
    fi
    
    # 创建数据库
    create_database
    
    # 显示连接信息
    show_connection_info
    
    log_info "调试完成！"
}

# 解析命令行参数
case "${1:-}" in
    --reset-password)
        reset_root_password
        ;;
    --test-connection)
        mysql -u root -pxbox123456 -e "SELECT 'Connection successful!' as status;"
        ;;
    --show-info)
        show_connection_info
        ;;
    *)
        main
        ;;
esac