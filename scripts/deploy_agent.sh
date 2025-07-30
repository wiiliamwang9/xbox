#!/bin/bash

# Xbox Agent远程部署脚本
set -e

# 颜色定义
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

# 检查参数
if [ $# -ne 3 ]; then
    echo "用法: $0 <远程IP> <SSH端口> <SSH密码>"
    echo "示例: $0 165.254.16.244 22 'Asd2025#'"
    exit 1
fi

REMOTE_IP=$1
SSH_PORT=$2
SSH_PASSWORD=$3
REMOTE_USER="root"
CONTROLLER_IP="165.254.16.246"  # 当前Controller节点的IP

log_info "开始部署Agent到远程节点: $REMOTE_IP"

# 检查必要的文件
if [ ! -f "bin/agent" ]; then
    log_error "Agent可执行文件不存在，请先运行 make build"
    exit 1
fi

if [ ! -f "configs/agent-config.yaml" ]; then
    log_error "Agent配置文件不存在"
    exit 1
fi

# 安装sshpass（如果未安装）
if ! command -v sshpass &> /dev/null; then
    log_info "安装sshpass..."
    apt update && apt install -y sshpass
fi

# 检查远程连接
log_info "测试SSH连接到 $REMOTE_IP:$SSH_PORT"
if ! sshpass -p "$SSH_PASSWORD" ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -p $SSH_PORT $REMOTE_USER@$REMOTE_IP "echo '连接成功'" > /dev/null 2>&1; then
    log_error "无法连接到远程服务器 $REMOTE_IP:$SSH_PORT"
    exit 1
fi

log_info "SSH连接测试成功"

# 创建临时目录
TEMP_DIR="/tmp/xbox-agent-deploy-$$"
mkdir -p $TEMP_DIR

# 复制文件到临时目录
cp bin/agent $TEMP_DIR/
cp configs/agent-config.yaml $TEMP_DIR/

# 修改配置文件中的Controller地址
sed -i "s/controller_addr: .*/controller_addr: \"$CONTROLLER_IP:9090\"/" $TEMP_DIR/agent-config.yaml

# 创建远程安装脚本
cat > $TEMP_DIR/install_agent.sh << 'EOF'
#!/bin/bash
set -e

echo "开始安装Xbox Agent..."

# 创建目录
mkdir -p /opt/xbox-agent
mkdir -p /opt/xbox-agent/logs
mkdir -p /opt/xbox-agent/configs

# 复制文件
mv agent /opt/xbox-agent/
mv agent-config.yaml /opt/xbox-agent/configs/
chmod +x /opt/xbox-agent/agent

# 创建systemd服务文件
cat > /etc/systemd/system/xbox-agent.service << 'SERVICE_EOF'
[Unit]
Description=Xbox Agent Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xbox-agent
ExecStart=/opt/xbox-agent/agent -config /opt/xbox-agent/configs/agent-config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICE_EOF

# 重载systemd并启用服务
systemctl daemon-reload
systemctl enable xbox-agent

echo "Xbox Agent安装完成"
echo "启动服务: systemctl start xbox-agent"
echo "查看状态: systemctl status xbox-agent"
echo "查看日志: journalctl -u xbox-agent -f"
EOF

chmod +x $TEMP_DIR/install_agent.sh

# 上传文件到远程服务器
log_info "上传文件到远程服务器..."
sshpass -p "$SSH_PASSWORD" scp -o StrictHostKeyChecking=no -P $SSH_PORT \
    $TEMP_DIR/agent \
    $TEMP_DIR/agent-config.yaml \
    $TEMP_DIR/install_agent.sh \
    $REMOTE_USER@$REMOTE_IP:/tmp/

# 在远程服务器上执行安装
log_info "在远程服务器上执行安装..."
sshpass -p "$SSH_PASSWORD" ssh -o StrictHostKeyChecking=no -p $SSH_PORT $REMOTE_USER@$REMOTE_IP << 'REMOTE_EOF'
cd /tmp
chmod +x install_agent.sh
./install_agent.sh
REMOTE_EOF

# 启动Agent服务
log_info "启动Agent服务..."
sshpass -p "$SSH_PASSWORD" ssh -o StrictHostKeyChecking=no -p $SSH_PORT $REMOTE_USER@$REMOTE_IP << 'REMOTE_EOF'
systemctl start xbox-agent
sleep 3
systemctl status xbox-agent --no-pager
REMOTE_EOF

# 清理临时文件
rm -rf $TEMP_DIR

log_info "Agent部署完成！"
log_info "远程节点: $REMOTE_IP"
log_info "服务状态查看: sshpass -p '$SSH_PASSWORD' ssh -p $SSH_PORT $REMOTE_USER@$REMOTE_IP 'systemctl status xbox-agent'"
log_info "日志查看: sshpass -p '$SSH_PASSWORD' ssh -p $SSH_PORT $REMOTE_USER@$REMOTE_IP 'journalctl -u xbox-agent -f'"
EOF