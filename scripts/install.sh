#!/bin/bash

# Xbox Sing-box管理系统环境安装脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
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
check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_warn "检测到root用户，建议使用普通用户运行"
        read -p "是否继续? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# 检测操作系统
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get >/dev/null 2>&1; then
            OS="ubuntu"
            log_info "检测到Ubuntu/Debian系统"
        elif command -v yum >/dev/null 2>&1; then
            OS="centos"
            log_info "检测到CentOS/RHEL系统"
        else
            log_error "不支持的Linux发行版"
            exit 1
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        log_info "检测到macOS系统"
    else
        log_error "不支持的操作系统: $OSTYPE"
        exit 1
    fi
}

# 安装Go 1.21
install_go() {
    local go_version="1.21.5"
    local go_arch
    
    # 检查Go是否已安装
    if command -v go >/dev/null 2>&1; then
        local current_version=$(go version | cut -d ' ' -f3 | cut -d 'o' -f2)
        if [[ "$current_version" == "1.21"* ]] || [[ "$current_version" > "1.21" ]]; then
            log_info "Go已安装，版本: $current_version"
            return 0
        else
            log_warn "Go版本过低: $current_version，需要1.21+"
        fi
    fi
    
    log_info "开始安装Go $go_version..."
    
    # 确定架构
    case $(uname -m) in
        x86_64) go_arch="amd64" ;;
        aarch64|arm64) go_arch="arm64" ;;
        *) log_error "不支持的架构: $(uname -m)"; exit 1 ;;
    esac
    
    # 下载Go
    local go_file="go${go_version}.linux-${go_arch}.tar.gz"
    if [[ "$OS" == "macos" ]]; then
        go_file="go${go_version}.darwin-${go_arch}.tar.gz"
    fi
    
    local go_url="https://golang.org/dl/${go_file}"
    
    cd /tmp
    log_info "下载Go安装包..."
    if ! curl -LO "$go_url"; then
        log_error "下载Go失败"
        exit 1
    fi
    
    # 安装Go
    log_info "安装Go到/usr/local..."
    if [[ $EUID -eq 0 ]]; then
        rm -rf /usr/local/go
        tar -C /usr/local -xzf "$go_file"
    else
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "$go_file"
    fi
    
    # 设置环境变量
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
    fi
    
    # 临时设置PATH
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
    
    log_info "Go安装完成！版本: $(go version)"
    rm -f "$go_file"
}

# 安装MySQL 8.0
install_mysql() {
    log_info "开始安装MySQL 8.0..."
    
    case "$OS" in
        "ubuntu")
            # 更新包列表
            sudo apt-get update
            
            # 安装MySQL
            if ! dpkg -l | grep -q mysql-server; then
                sudo apt-get install -y mysql-server mysql-client
                
                # 启动MySQL服务
                sudo systemctl start mysql
                sudo systemctl enable mysql
                
                log_info "MySQL安装完成"
                log_warn "请手动运行 'sudo mysql_secure_installation' 来配置MySQL安全设置"
            else
                log_info "MySQL已安装"
            fi
            ;;
            
        "centos")
            # 安装MySQL官方仓库
            if ! rpm -qa | grep -q mysql80-community-release; then
                sudo yum install -y https://dev.mysql.com/get/mysql80-community-release-el7-3.noarch.rpm
            fi
            
            # 安装MySQL
            if ! rpm -qa | grep -q mysql-community-server; then
                sudo yum install -y mysql-community-server
                
                # 启动MySQL服务
                sudo systemctl start mysqld
                sudo systemctl enable mysqld
                
                # 获取临时密码
                local temp_password=$(sudo grep 'temporary password' /var/log/mysqld.log | tail -1 | awk '{print $NF}')
                log_info "MySQL安装完成"
                log_warn "临时密码: $temp_password"
                log_warn "请手动运行 'mysql_secure_installation' 来配置MySQL"
            else
                log_info "MySQL已安装"
            fi
            ;;
            
        "macos")
            if command -v brew >/dev/null 2>&1; then
                if ! brew list | grep -q mysql; then
                    brew install mysql
                    brew services start mysql
                    log_info "MySQL安装完成"
                else
                    log_info "MySQL已安装"
                fi
            else
                log_error "请先安装Homebrew: https://brew.sh/"
                exit 1
            fi
            ;;
    esac
}

# 安装Protocol Buffers
install_protoc() {
    local protoc_version="25.1"
    
    # 检查protoc是否已安装
    if command -v protoc >/dev/null 2>&1; then
        log_info "protoc已安装，版本: $(protoc --version)"
        return 0
    fi
    
    log_info "开始安装Protocol Buffers..."
    
    case "$OS" in
        "ubuntu")
            sudo apt-get update
            sudo apt-get install -y protobuf-compiler
            ;;
            
        "centos")
            # CentOS需要手动安装
            local arch
            case $(uname -m) in
                x86_64) arch="x86_64" ;;
                aarch64) arch="aarch_64" ;;
                *) log_error "不支持的架构"; exit 1 ;;
            esac
            
            cd /tmp
            local protoc_file="protoc-${protoc_version}-linux-${arch}.zip"
            local protoc_url="https://github.com/protocolbuffers/protobuf/releases/download/v${protoc_version}/${protoc_file}"
            
            curl -LO "$protoc_url"
            sudo unzip -o "$protoc_file" -d /usr/local
            sudo chmod +x /usr/local/bin/protoc
            rm -f "$protoc_file"
            ;;
            
        "macos")
            if command -v brew >/dev/null 2>&1; then
                brew install protobuf
            else
                log_error "请先安装Homebrew"
                exit 1
            fi
            ;;
    esac
    
    log_info "protoc安装完成，版本: $(protoc --version)"
}

# 安装Go插件
install_go_plugins() {
    log_info "安装Go插件..."
    
    # 确保Go已配置
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go未找到，请先安装Go"
        exit 1
    fi
    
    # 安装protoc插件
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    
    log_info "Go插件安装完成"
}

# 创建数据库
setup_database() {
    log_info "设置数据库..."
    
    read -p "请输入MySQL root密码: " -s mysql_password
    echo
    
    # 测试连接
    if ! mysql -u root -p"$mysql_password" -e "SELECT 1;" >/dev/null 2>&1; then
        log_error "MySQL连接失败，请检查密码"
        return 1
    fi
    
    # 执行初始化脚本
    if [[ -f "scripts/init.sql" ]]; then
        mysql -u root -p"$mysql_password" < scripts/init.sql
        log_info "数据库初始化完成"
    else
        log_warn "未找到数据库初始化脚本 scripts/init.sql"
    fi
}

# 验证安装
verify_installation() {
    log_info "验证安装..."
    
    # 检查Go
    if command -v go >/dev/null 2>&1; then
        log_info "✓ Go: $(go version)"
    else
        log_error "✗ Go未安装"
        return 1
    fi
    
    # 检查MySQL
    if command -v mysql >/dev/null 2>&1; then
        log_info "✓ MySQL: $(mysql --version)"
    else
        log_error "✗ MySQL未安装"
        return 1
    fi
    
    # 检查protoc
    if command -v protoc >/dev/null 2>&1; then
        log_info "✓ protoc: $(protoc --version)"
    else
        log_error "✗ protoc未安装"
        return 1
    fi
    
    # 检查Go插件
    if command -v protoc-gen-go >/dev/null 2>&1; then
        log_info "✓ protoc-gen-go已安装"
    else
        log_error "✗ protoc-gen-go未安装"
        return 1
    fi
    
    if command -v protoc-gen-go-grpc >/dev/null 2>&1; then
        log_info "✓ protoc-gen-go-grpc已安装"
    else
        log_error "✗ protoc-gen-go-grpc未安装"
        return 1
    fi
    
    log_info "所有依赖验证完成！"
}

# 主函数
main() {
    log_info "Xbox Sing-box管理系统环境安装脚本"
    log_info "=================================="
    
    check_root
    detect_os
    
    # 安装组件
    install_go
    install_mysql
    install_protoc
    install_go_plugins
    
    # 验证安装
    verify_installation
    
    log_info "安装完成！"
    log_info "下一步："
    log_info "1. 重新加载shell配置: source ~/.bashrc"
    log_info "2. 设置数据库: ./scripts/install.sh --setup-db"
    log_info "3. 生成proto代码: make proto"
    log_info "4. 构建项目: make build"
}

# 解析命令行参数
case "${1:-}" in
    --setup-db)
        setup_database
        ;;
    --verify)
        verify_installation
        ;;
    *)
        main
        ;;
esac