#!/bin/bash
# install-linux.sh - Linux Agent 安装脚本

set -e

# 默认参数
BACKEND_URL="${1:-https://xingran-backend.example.com}"
AGENT_ID="${2:-}"
VM_ID="${3:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing XingRan VDI Agent...${NC}"

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}ERROR: This script must be run as root${NC}"
    exit 1
fi

# 验证参数
if [ -z "$AGENT_ID" ] || [ -z "$VM_ID" ]; then
    echo -e "${RED}ERROR: AGENT_ID and VM_ID are required${NC}"
    echo "Usage: $0 <backend_url> <agent_id> <vm_id>"
    exit 1
fi

# 下载 Agent
AGENT_URL="$BACKEND_URL/api/v1/agent/download/linux"
AGENT_DIR="/opt/xingran-agent"
AGENT_BIN="$AGENT_DIR/agent"

mkdir -p "$AGENT_DIR"

echo "Downloading agent from: $AGENT_URL"
if ! wget -O "$AGENT_BIN" "$AGENT_URL"; then
    echo -e "${RED}ERROR: Failed to download agent${NC}"
    exit 1
fi

chmod +x "$AGENT_BIN"
echo -e "${GREEN}Agent downloaded successfully${NC}"

# 创建配置文件
CONFIG_PATH="$AGENT_DIR/config.yaml"
cat > "$CONFIG_PATH" <<EOF
backend_url: $BACKEND_URL
agent_id: $AGENT_ID
vm_id: $VM_ID
listen_addr: ":8443"
heartbeat_interval: 30s
log_level: info
log_path: /var/log/xingran-agent
platform: linux
EOF

echo "Configuration created at: $CONFIG_PATH"

# 创建日志目录
LOG_DIR="/var/log/xingran-agent"
mkdir -p "$LOG_DIR"
chown -R root:root "$LOG_DIR"

# 创建专用服务账号
if ! id xingran-agent &>/dev/null; then
    useradd -m -r -s /bin/bash xingran-agent
    echo -e "${GREEN}Created user: xingran-agent${NC}"
else
    echo -e "${YELLOW}User xingran-agent already exists${NC}"
fi

# 配置 sudoers（用户管理命令）
SUDOERS_FILE="/etc/sudoers.d/xingran-agent"
cat > "$SUDOERS_FILE" <<EOF
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/useradd
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/userdel
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/usermod
xingran-agent ALL=(root) NOPASSWD: /usr/bin/passwd
EOF

chmod 440 "$SUDOERS_FILE"
echo -e "${GREEN}Sudoers configured${NC}"

# 创建 systemd 服务
SERVICE_FILE="/etc/systemd/system/xingran-vm-agent.service"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=XingRan VM Agent
After=network.target

[Service]
Type=simple
User=xingran-agent
WorkingDirectory=$AGENT_DIR
ExecStart=$AGENT_BIN --config=$CONFIG_PATH
Restart=always
RestartSec=10

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$LOG_DIR

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd
systemctl daemon-reload

# 启用并启动服务
echo "Starting service..."
systemctl enable xingran-vm-agent
systemctl start xingran-vm-agent

# 等待服务启动
sleep 2

# 检查服务状态
if systemctl is-active --quiet xingran-vm-agent; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}XingRan VDI Agent installed successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "Service name: xingran-vm-agent"
    echo "Config file: $CONFIG_PATH"
    echo "Log directory: $LOG_DIR"
    echo ""
    echo "To check service status:"
    echo "  systemctl status xingran-vm-agent"
    echo ""
    echo "To view logs:"
    echo "  journalctl -u xingran-vm-agent -f"
    echo "  tail -f $LOG_DIR/*.log"
    echo ""
else
    echo -e "${RED}ERROR: Failed to start service${NC}"
    echo "Check logs:"
    echo "  journalctl -u xingran-vm-agent -n 50"
    exit 1
fi
