# VM Agent 构建与测试指南

## 快速开始

### 1. 构建 Agent

```bash
# 构建所有平台版本
bash scripts/agent/build.sh 1.0.0

# 仅构建 Windows 版本
set GOOS=windows && set GOARCH=amd64 && go build -o build/agent-windows-amd64.exe ./cmd/agent/

# 仅构建 Linux 版本
GOOS=linux GOARCH=amd64 go build -o build/agent-linux-amd64 ./cmd/agent/
```

### 2. 本地测试

#### Linux

```bash
# 运行自动化测试
bash scripts/agent/test-agent.sh

# 手动测试
./build/agent-linux-amd64 --config=scripts/agent/test-config.yaml
```

#### Windows

```cmd
# 运行自动化测试
scripts\agent\test-agent.bat

# 手动测试
.\build\agent-windows-amd64.exe --config=scripts\agent\test-config.yaml
```

### 3. 虚拟机部署

#### Linux 虚拟机

```bash
# 1. 复制 Agent 到虚拟机
scp build/agent-linux-amd64 user@vm-ip:/tmp/agent

# 2. 在虚拟机上执行安装
ssh user@vm-ip
sudo bash /tmp/install-linux.sh "https://backend.com" "agent-id" "vm-id"

# 3. 检查服务状态
systemctl status xingran-vm-agent
journalctl -u xingran-vm-agent -f
```

#### Windows 虚拟机

```powershell
# 1. 复制 Agent 到虚拟机
Copy-Item "build\agent-windows-amd64.exe" "\\VM-IP\C$\Program Files\XingRanAgent\"

# 2. 在虚拟机上执行安装
Enter-PSSession -ComputerName VM-IP
powershell .\install-windows.ps1 -BackendURL "https://backend.com" -AgentID "agent-id" -VMID "vm-id"

# 3. 检查服务状态
Get-Service XingRanVMAgent
```

## API 端点

| 端点 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/api/v1/health` | GET | 健康检查 | 否 |
| `/api/v1/register` | POST | Agent 注册 | 否 |
| `/api/v1/accounts` | POST | 创建账号 | 是 |
| `/api/v1/accounts` | GET | 列出账号 | 是 |
| `/api/v1/accounts/:username` | DELETE | 删除账号 | 是 |
| `/api/v1/accounts/:username/reset` | POST | 重置密码 | 是 |
| `/api/v1/accounts/:username/enable` | POST | 启用账号 | 是 |
| `/api/v1/accounts/:username/disable` | POST | 禁用账号 | 是 |
| `/api/v1/heartbeat` | POST | 心跳上报 | 是 |

## 配置说明

### 配置文件结构

```yaml
backend_url: "https://backend.example.com"  # 后端服务器地址
agent_id: "unique-agent-id"                 # Agent 唯一标识
vm_id: "unique-vm-id"                        # 虚拟机唯一标识
listen_addr: ":8443"                         # 监听地址
heartbeat_interval: "30s"                    # 心跳间隔
log_level: "info"                             # 日志级别
log_path: "/var/log/xingran-agent"             # 日志目录
jwt_secret: "your-secret-key"                 # JWT 密钥
```

### 环境变量

```bash
# Agent 配置
export AGENT_CONFIG_PATH="/path/to/config.yaml"
export BACKEND_URL="https://backend.example.com"
export AGENT_ID="agent-001"
export VM_ID="vm-001"
export JWT_SECRET="your-secret"
```

## 故障排查

### Linux

```bash
# 检查服务状态
systemctl status xingran-vm-agent

# 查看日志
journalctl -u xingran-vm-agent -n 100

# 检查端口占用
netstat -tuln | grep 8443

# 检查进程
ps aux | grep xingran-vm-agent
```

### Windows

```powershell
# 检查服务状态
Get-Service XingRanVMAgent

# 查看日志
Get-Content "C:\Program Files\XingRanAgent\logs\*.log"

# 检查端口占用
netstat -an | findstr 8443

# 检查进程
Get-Process xingran-vm-agent
```

### 常见问题

1. **Agent 无法启动**
   - 检查配置文件路径是否正确
   - 检查端口是否被占用
   - 检查日志文件权限

2. **注册失败**
   - 检查后端服务器是否可访问
   - 检查网络连接
   - 检查 JWT Secret 配置

3. **账号操作失败**
   - Linux: 检查 sudoers 配置
   - Windows: 检查管理员权限
   - 检查用户名是否已存在

## 安全注意事项

⚠️ **生产环境部署前必须修改以下配置**:

1. **JWT Secret**: 使用强随机密钥
2. **TLS 证书**: 使用有效的 SSL/TLS 证书
3. **后端 URL**: 使用 HTTPS 地址
4. **权限控制**: 限制 Agent 账号权限

## 文件结构

```
scripts/agent/
├── build.sh              # 构建脚本
├── install-linux.sh       # Linux 安装脚本
├── install-windows.ps1    # Windows 安装脚本
├── test-agent.sh          # Linux 测试脚本
├── test-agent.bat         # Windows 测试脚本
├── test-config.yaml       # 测试配置示例
└── README.md             # 本文档
```

## 构建产物

构建完成后，在 `build/` 目录下生成以下文件：

| 文件 | 大小 | 描述 |
|------|------|------|
| `agent-linux-amd64` | ~15MB | Linux x64 可执行文件 |
| `agent-windows-amd64.exe` | ~15MB | Windows x64 可执行文件 |
| `agent-linux-amd64-1.0.0.tar.gz` | ~8MB | Linux 压缩包 |

## 更多信息

详细测试方案请参考: `docs/agent-test-plan.md`
