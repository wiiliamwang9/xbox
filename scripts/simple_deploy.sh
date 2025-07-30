#!/bin/bash

# 简化的Agent部署脚本
set -e

REMOTE_IP=$1
SSH_PASSWORD=$2
CONTROLLER_IP="165.254.16.246"

echo "=== 部署Agent到 $REMOTE_IP ==="

# 检查连接
echo "1. 测试SSH连接..."
if ! sshpass -p "$SSH_PASSWORD" ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@$REMOTE_IP "echo 'SSH OK'" 2>/dev/null; then
    echo "错误: 无法连接到 $REMOTE_IP"
    exit 1
fi
echo "   连接成功"

# 上传Agent二进制文件
echo "2. 上传Agent程序..."
sshpass -p "$SSH_PASSWORD" scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null bin/agent root@$REMOTE_IP:/tmp/

# 创建配置文件
echo "3. 创建配置文件..."
cat > /tmp/agent-config-$REMOTE_IP.yaml << EOF
log:
  level: "info"
  format: "json"
  output: "stdout"

agent:
  id: ""
  controller_addr: "$CONTROLLER_IP:9090"
  heartbeat_interval: 30
  singbox_config: "./sing-box.json"
  singbox_binary: "sing-box"
EOF

# 上传配置文件
sshpass -p "$SSH_PASSWORD" scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null /tmp/agent-config-$REMOTE_IP.yaml root@$REMOTE_IP:/tmp/agent-config.yaml

# 远程安装和启动
echo "4. 远程安装和启动..."
sshpass -p "$SSH_PASSWORD" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -T root@$REMOTE_IP << 'EOF'
# 创建目录
mkdir -p /opt/xbox-agent/configs

# 移动文件
mv /tmp/agent /opt/xbox-agent/
mv /tmp/agent-config.yaml /opt/xbox-agent/configs/
chmod +x /opt/xbox-agent/agent

# 创建systemd服务
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

# 启动服务
systemctl daemon-reload
systemctl enable xbox-agent
systemctl start xbox-agent

# 等待并检查状态
sleep 3
systemctl status xbox-agent --no-pager
EOF

# 清理临时文件
rm -f /tmp/agent-config-$REMOTE_IP.yaml

echo "=== 部署完成! ==="
echo "查看日志: sshpass -p '$SSH_PASSWORD' ssh root@$REMOTE_IP 'journalctl -u xbox-agent -f'"