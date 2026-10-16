# VM Agent 测试方案

> **文档日期**：2026-08-12（刷新）
> **Agent 版本**：v1.x（VDI Worker 账号管理 Agent，对接后端 `/api/v1/rpa/workers/*` 与 AD 域控）

## 构建产物

| 文件 | 大小 | 平台 | 位置 |
|------|------|------|------|
| `agent-windows-amd64.exe` | 15MB | Windows x64 | `build/agent-windows-amd64.exe` |
| `agent-linux-amd64` | 15MB | Linux x64 | `build/agent-linux-amd64` |

## 快速测试方案

### 1. 本地配置测试

创建测试配置文件 `test-agent-config.yaml`：

```yaml
backend_url: "http://localhost:9000"
agent_id: "test-agent-001"
vm_id: "test-vm-001"
listen_addr: ":8443"
heartbeat_interval: 30s
log_level: "debug"
log_path: "./logs"
platform: "windows"  # 或 "linux"
jwt_secret: "test-secret-key-for-development-only"
```

**运行测试**：
```bash
# Linux
./build/agent-linux-amd64 --config=test-agent-config.yaml

# Windows
.\build\agent-windows-amd64.exe --config=test-agent-config.yaml
```

### 2. 本地功能测试（无需后端）

#### 测试账号管理功能（Linux）

```bash
# 创建测试用户
sudo useradd testuser001
echo "testuser001:TestPass123!" | sudo chpasswd

# 测试 Agent API
curl -X POST http://localhost:8443/api/v1/health
```

#### 测试账号管理功能（Windows PowerShell）

```powershell
# 创建测试用户
New-LocalUser -Name "testuser001" -Password (ConvertTo-SecureString "TestPass123!" -AsPlainText -Force)

# 测试 Agent API
Invoke-WebRequest -Uri "http://localhost:8443/api/v1/health" -Method POST
```

### 3. 集成测试（需要后端服务器）

#### 前置条件
1. 后端服务器运行在 `http://localhost:9000`
2. 后端已配置 VDI 模块
3. 有测试 VM 和 Agent 记录

#### 测试步骤

```bash
# 1. 注册 Agent
curl -X POST http://localhost:8443/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"test-agent-001","vm_id":"test-vm-001"}'

# 2. 健康检查
curl -X POST http://localhost:8443/api/v1/health

# 3. 创建账号（需要 JWT 令牌）
curl -X POST http://localhost:8443/api/v1/accounts \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser002",
    "password": "TestPass123!",
    "is_admin": false,
    "os_type": "linux"
  }'

# 4. 列出账号
curl -X GET http://localhost:8443/api/v1/accounts \
  -H "Authorization: Bearer <JWT_TOKEN>"

# 5. 禁用账号
curl -X POST http://localhost:8443/api/v1/accounts/testuser002/disable \
  -H "Authorization: Bearer <JWT_TOKEN>"

# 6. 启用账号
curl -X POST http://localhost:8443/api/v1/accounts/testuser002/enable \
  -H "Authorization: Bearer <JWT_TOKEN>"

# 7. 重置密码
curl -X POST http://localhost:8443/api/v1/accounts/testuser002/reset \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"new_password": "NewPass456!"}'

# 8. 删除账号
curl -X DELETE http://localhost:8443/api/v1/accounts/testuser002 \
  -H "Authorization: Bearer <JWT_TOKEN>"

# 9. 心跳测试
curl -X POST http://localhost:8443/api/v1/heartbeat \
  -H "Authorization: Bearer <JWT_TOKEN>"
```

## 虚拟机部署测试

### Windows 虚拟机测试

```powershell
# 1. 复制 Agent 到虚拟机
Copy-Item "build\agent-windows-amd64.exe" "\\VM-IP\C$\Program Files\XingRanAgent\"

# 2. 在虚拟机上执行安装
powershell .\scripts\agent\install-windows.ps1 `
  -BackendURL "https://your-backend.com" `
  -AgentID "win-vm-001" `
  -VMID "vm-uuid-001"

# 3. 检查服务状态
Get-Service XingRanVMAgent

# 4. 查看日志
Get-Content "C:\Program Files\XingRanAgent\logs\*.log"

# 5. 测试 API
Invoke-WebRequest -Uri "https://localhost:8443/api/v1/health" -Method POST -UseBasicParsing
```

### Linux 虚拟机测试

```bash
# 1. 复制 Agent 到虚拟机
scp build/agent-linux-amd64 user@vm-ip:/tmp/

# 2. 在虚拟机上执行安装
ssh user@vm-ip
sudo bash install-linux.sh "https://your-backend.com" "linux-vm-001" "vm-uuid-002"

# 3. 检查服务状态
systemctl status xingran-vm-agent

# 4. 查看日志
journalctl -u xingran-vm-agent -f

# 5. 测试 API
curl -k -X POST https://localhost:8443/api/v1/health
```

## 功能测试清单

### 基础功能

| 测试项 | 测试命令 | 预期结果 |
|--------|----------|----------|
| 健康检查 | `POST /api/v1/health` | 返回 `{"status":"healthy"}` |
| Agent 注册 | `POST /api/v1/register` | 注册成功，获得 JWT 令牌 |
| 心跳上报 | `POST /api/v1/heartbeat` | 心跳接收成功 |

### 账号管理功能

| 测试项 | 测试命令 | 预期结果 |
|--------|----------|----------|
| 创建账号 | `POST /api/v1/accounts` | 账号创建成功 |
| 列出账号 | `GET /api/v1/accounts` | 返回账号列表 |
| 禁用账号 | `POST /api/v1/accounts/{user}/disable` | 账号被禁用 |
| 启用账号 | `POST /api/v1/accounts/{user}/enable` | 账号被启用 |
| 重置密码 | `POST /api/v1/accounts/{user}/reset` | 密码重置成功 |
| 删除账号 | `DELETE /api/v1/accounts/{user}` | 账号被删除 |

### 认证功能

| 测试项 | 测试方法 | 预期结果 |
|--------|----------|----------|
| JWT 认证 | 无令牌访问受保护端点 | 返回 401 |
| 有效令牌 | 使用有效 JWT 访问 | 访问成功 |
| 无效令牌 | 使用过期/伪造 JWT | 返回 401 |
| 令牌刷新 | 令牌即将过期时访问 | 自动刷新成功 |

### 跨平台功能

| 平台 | 测试项 | 验证方法 |
|------|--------|----------|
| Windows | PowerShell 账号操作 | 检查 `Get-LocalUser` |
| Windows | 服务安装 | 检查 `Get-Service` |
| Linux | Shell 账号操作 | 检查 `/etc/passwd` |
| Linux | systemd 服务 | 检查 `systemctl status` |
| Linux | sudoers 配置 | 检查 `/etc/sudoers.d/` |

## 性能测试

### 资源占用测试

```bash
# Windows
Get-Process xingran-vm-agent | Select-Object CPU,WorkingSet,Handles

# Linux
ps aux | grep xingran-vm-agent
top -p $(pgrep xingran-vm-agent)
```

### 并发测试

```bash
# 使用 Apache Bench 进行压力测试
ab -n 1000 -c 10 http://localhost:8443/api/v1/health

# 预期结果：
# - 无内存泄漏
# - CPU 占用稳定
# - 响应时间 < 100ms
```

## 安全测试

### TLS 加密测试

```bash
# 测试 HTTPS 连接
curl -k -X POST https://localhost:8443/api/v1/health

# 检查证书（生产环境）
openssl s_client -connect localhost:8443
```

### 权限测试

```bash
# 测试未授权访问
curl -X POST http://localhost:8443/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"username":"hack","password":"pass"}'

# 预期：返回 401
```

### 注入测试

```bash
# 测试命令注入
curl -X POST http://localhost:8443/api/v1/accounts \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"username":"user;rm -rf /","password":"pass"}'

# 预期：参数验证，拒绝恶意输入
```

## 故障排查

### 常见问题

1. **Agent 无法启动**
   ```bash
   # 检查日志
   # Windows: Get-Content "C:\Program Files\XingRanAgent\logs\*.log"
   # Linux: journalctl -u xingran-vm-agent -n 50

   # 常见原因：
   # - 配置文件路径错误
   # - 端口被占用
   # - 权限不足
   ```

2. **注册失败**
   ```bash
   # 检查网络连接
   curl http://backend-url:9000/health

   # 检查 JWT Secret 配置
   ```

3. **账号操作失败**
   ```bash
   # Windows: 检查管理员权限
   # Linux: 检查 sudoers 配置
   cat /etc/sudoers.d/xingran-agent
   ```

## 集成测试流程

### 完整流程测试

```bash
# 1. 启动后端服务器
cd xingran-go-backend
./xingran-backend.exe

# 2. 创建测试 VM（通过 VDI API）
# 3. 部署 Agent
# 4. 验证 Agent 注册成功
# 5. 执行完整账号 CRUD 测试
# 6. 验证心跳正常上报
# 7. 重启 Agent，验证状态恢复
# 8. 卸载 Agent，验证清理干净
```

## 自动化测试脚本

创建 `test-agent.sh`：

```bash
#!/bin/bash

echo "=== VM Agent 测试套件 ==="

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 测试函数
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local expected=$4

    echo -n "Testing: $name ... "
    response=$(curl -s -X $method $url)

    if [[ $response == *"$expected"* ]]; then
        echo -e "${GREEN}PASS${NC}"
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "Expected: $expected"
        echo "Got: $response"
        return 1
    fi
}

# 健康检查
test_endpoint "Health Check" "POST" "http://localhost:8443/api/v1/health" "healthy"

# 注册
test_endpoint "Agent Register" "POST" "http://localhost:8443/api/v1/register" "success"

# 创建账号
test_endpoint "Create Account" "POST" "http://localhost:8443/api/v1/accounts" "created"

echo "=== 测试完成 ==="
```

## 测试报告模板

```
测试日期: YYYY-MM-DD
测试人员: [姓名]
Agent 版本: 1.0.0
后端版本: [版本号]

## 测试环境
- 操作系统: Windows 11 / Ubuntu 22.04
- 后端地址: http://localhost:9000
- Agent ID: test-agent-001

## 测试结果

| 功能 | Windows | Linux | 备注 |
|------|---------|-------|------|
| 健康检查 | ✅ | ✅ | |
| Agent 注册 | ✅ | ✅ | |
| 账号创建 | ✅ | ✅ | |
| 账号删除 | ✅ | ✅ | |
| 密码重置 | ✅ | ✅ | |
| 账号启用/禁用 | ✅ | ✅ | |
| 心跳上报 | ✅ | ✅ | |
| JWT 认证 | ✅ | ✅ | |
| TLS 加密 | ⚠️ | ⚠️ | 自签名证书 |

## 发现的问题
1. [问题描述]
2. [问题描述]

## 建议
1. [改进建议]
2. [改进建议]
```
