# VM Agent 架构设计

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        XingRan-Next 后端                           │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐   │
│  │ VM Account    │   │ VM Agent      │   │ Password      │   │
│  │ Service       │───│ Manager       │───│ Policy        │   │
│  └───────────────┘   └───────────────┘   └───────────────┘   │
│         │                     │                               │
└─────────│─────────────────────│───────────────────────────────┘
          │                     │
          │ HTTP/gRPC           │
          │                     │
┌─────────▼─────────────────────▼───────────────────────────────┐
│                     VM Agent (运行在虚拟机内)                    │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐   │
│  │ Account       │   │ System        │   │ Network       │   │
│  │ Manager       │   │ Monitor       │   │ Listener      │   │
│  └───────────────┘   └───────────────┘   └───────────────┘   │
│         │                     │                               │
└─────────│─────────────────────│───────────────────────────────┘
          │                     │
          │ 系统调用            │ 系统监控
          │                     │
┌─────────▼─────────────────────▼───────────────────────────────┐
│                    虚拟机操作系统 (Windows/Linux)               │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐   │
│  │ Local Users   │   │ System Info   │   │ Network       │   │
│  │ Groups        │   │ Processes     │   │ Configuration │   │
│  └───────────────┘   └───────────────┘   └───────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## VM Agent 功能模块

### 1. 账号管理模块 (Account Manager)

**Windows 支持的操作**:
```powershell
# 创建用户
New-LocalUser -Name $username -Password $password -Description $description

# 修改密码
Set-LocalUser -Name $username -Password $password

# 删除用户
Remove-LocalUser -Name $username

# 启用/禁用用户
Enable-LocalUser -Name $username
Disable-LocalUser -Name $username

# 添加到管理员组
Add-LocalGroupMember -Group "Administrators" -Member $username

# 查询用户信息
Get-LocalUser -Name $username
```

**Linux 支持的操作**:
```bash
# 创建用户
useradd -m -p $password $username

# 修改密码
echo "$username:$password" | chpasswd

# 删除用户
userdel -r $username

# 启用/禁用用户
usermod -e $expiration $username  # 禁用
usermod -e "" $username            # 启用

# 添加到sudo组
usermod -aG sudo $username

# 查询用户信息
id $username
grep $username /etc/passwd
```

### 2. 系统监控模块 (System Monitor)

**监控指标**:
- CPU使用率
- 内存使用率
- 磁盘使用率
- 运行进程列表
- 登录用户列表
- 网络连接状态

### 3. 网络监听模块 (Network Listener)

**功能**:
- 监听指定端口（默认：HTTP 8080, gRPC 9090）
- 心跳检测（每30秒向XingRan-Next报告状态）
- 接收并执行账号管理指令
- 返回执行结果

## Agent 与后端通信协议

### RESTful API 接口

**1. Agent 注册**
```http
POST /api/v1/agent/register
Content-Type: application/json

{
  "vm_id": "vm-12345",
  "agent_version": "1.0.0",
  "os_type": "windows",
  "hostname": "win10-pc01",
  "ip_address": "192.168.1.100"
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "agent_id": "agent-67890",
    "auth_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
    "heartbeat_interval": 30
  }
}
```

**2. 心跳检测**
```http
POST /api/v1/agent/heartbeat
Authorization: Bearer {auth_token}
Content-Type: application/json

{
  "vm_id": "vm-12345",
  "timestamp": 1716600000,
  "status": {
    "cpu_usage": 25.5,
    "memory_usage": 60.2,
    "disk_usage": 45.8,
    "running_processes": 120,
    "logged_in_users": ["administrator", "user01"]
  }
}
```

**3. 账号操作**
```http
POST /api/v1/agent/accounts
Authorization: Bearer {auth_token}
Content-Type: application/json

{
  "vm_id": "vm-12345",
  "operation": "create",
  "account": {
    "username": "testuser",
    "password": "SecurePass123!",
    "account_type": "user",
    "is_admin": false,
    "description": "测试用户"
  }
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "account_id": "vm-12345:testuser",
    "operation_id": "op-98765",
    "status": "success",
    "details": "User created successfully"
  }
}
```

### gRPC 接口（高性能场景）

```protobuf
service VMAgentService {
  // 心跳
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // 账号管理
  rpc CreateAccount(CreateAccountRequest) returns (AccountResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (AccountResponse);
  rpc UpdatePassword(UpdatePasswordRequest) returns (AccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);

  // 系统信息
  rpc GetSystemInfo(SystemInfoRequest) returns (SystemInfoResponse);
}
```

## 安全设计

### 1. Agent 认证

**JWT Token 认证**:
```go
type AgentAuth struct {
    AgentID   string
    VMID      string
    SecretKey string // 每个Agent唯一密钥
    IssuedAt  time.Time
    ExpiresAt time.Time
}
```

### 2. 通信加密

- 使用 TLS 1.3 加密通信
- Agent 证书由 XingRan-Next CA 签发
- 证书有效期：1年，自动续期

### 3. 密码加密存储

```go
// SM4加密VM密码
func EncryptVMPassword(password string) (string, error) {
    key := []byte("xingran-vm-pwd-key-16")
    return sm4.EncryptCFB(key, []byte(password))
}

// 验证密码强度
func ValidatePasswordStrength(password string, policy *VMPasswordPolicy) error {
    if len(password) < policy.MinLength {
        return errors.New("password too short")
    }
    // ... 其他验证
}
```

## Agent 部署方式

### 方式1: VDI 镜像预装

**优点**:
- 所有新创建的VM自动包含Agent
- 统一版本管理

**实施步骤**:
1. 创建VM模板
2. 在模板中安装Agent
3. 配置Agent自动启动
4. 将模板制作为VDI镜像

### 方式2: VDI 创建后自动安装

**优点**:
- 不修改VDI镜像
- 可以动态选择是否安装Agent

**实施步骤**:
1. VM创建完成后，通过VDI API执行安装脚本
2. Agent自动注册到XingRan-Next
3. 更新VM的agent_installed状态

### 方式3: 手动安装（不推荐）

适用于特殊场景，如无法自动安装的VM

## Agent 安装脚本

### Windows Agent 安装脚本

```powershell
# install-windows-agent.ps1
$vmId = "vm-12345"
$agentUrl = "https://xingran-backend.example.com/api/v1/agent/download/windows"
$installPath = "C:\Program Files\XingRanAgent"

# 下载Agent
Invoke-WebRequest -Uri $agentUrl -OutFile "agent.zip"
Expand-Archive -Path "agent.zip" -DestinationPath $installPath

# 配置Agent
$config = @{
    vm_id = $vmId
    server_url = "https://xingran-backend.example.com"
    server_port = 443
    heartbeat_interval = 30
}
$config | ConvertTo-Json | Set-Content "$installPath\config.json"

# 注册Windows服务
New-Service -Name "XingRanVMAgent" `
    -BinaryPathName "$installPath\agent.exe" `
    -StartupType Automatic

Start-Service "XingRanVMAgent"

# 自动注册到后端
Invoke-WebRequest -Uri "$serverUrl/api/v1/agent/register" `
    -Method POST `
    -Body ($config | ConvertTo-Json) `
    -ContentType "application/json"
```

### Linux Agent 安装脚本

```bash
#!/bin/bash
# install-linux-agent.sh

VM_ID="vm-12345"
AGENT_URL="https://xingran-backend.example.com/api/v1/agent/download/linux"
INSTALL_PATH="/opt/xingran-agent"

# 下载Agent
wget -O /tmp/agent.tar.gz $AGENT_URL
mkdir -p $INSTALL_PATH
tar -xzf /tmp/agent.tar.gz -C $INSTALL_PATH

# 配置Agent
cat > $INSTALL_PATH/config.json << EOF
{
  "vm_id": "$VM_ID",
  "server_url": "https://xingran-backend.example.com",
  "server_port": 443,
  "heartbeat_interval": 30
}
EOF

# 注册systemd服务
cat > /etc/systemd/system/xingran-vm-agent.service << EOF
[Unit]
Description=XingRan VM Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_PATH
ExecStart=$INSTALL_PATH/agent --config=$INSTALL_PATH/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable xingran-vm-agent
systemctl start xingran-vm-agent

# 自动注册到后端
curl -X POST "$server_url/api/v1/agent/register" \
  -H "Content-Type: application/json" \
  -d @$INSTALL_PATH/config.json
```

## 后端服务设计

### 1. VMAccountService (账号管理服务)

```go
type VMAccountService interface {
    // CRUD操作
    CreateAccount(ctx context.Context, req *CreateAccountRequest) (*VDIVMAccount, error)
    GetAccount(ctx context.Context, accountID string) (*VDIVMAccount, error)
    ListAccounts(ctx context.Context, vmID string) ([]VDIVMAccount, error)
    UpdateAccount(ctx context.Context, accountID string, req *UpdateAccountRequest) error
    DeleteAccount(ctx context.Context, accountID string) error

    // 账号操作
    ResetPassword(ctx context.Context, accountID, newPassword string) error
    EnableAccount(ctx context.Context, accountID string) error
    DisableAccount(ctx context.Context, accountID string) error
    UnlockAccount(ctx context.Context, accountID string) error

    // 批量操作
    BatchCreateAccounts(ctx context.Context, req []CreateAccountRequest) (*BatchResult, error)
    BatchResetPasswords(ctx context.Context, req []ResetPasswordRequest) (*BatchResult, error)
}
```

### 2. VMAgentManager (Agent管理服务)

```go
type VMAgentManager interface {
    // Agent注册
    RegisterAgent(ctx context.Context, req *RegisterAgentRequest) (*VMAgent, error)
    UnregisterAgent(ctx context.Context, agentID string) error

    // 心跳处理
    HandleHeartbeat(ctx context.Context, req *HeartbeatRequest) error

    // Agent状态
    GetAgentStatus(ctx context.Context, vmID string) (*VMAgentStatus, error)
    ListOnlineAgents(ctx context.Context) ([]VMAgent, error)

    // 远程命令执行
    ExecuteCommand(ctx context.Context, vmID string, cmd *AgentCommand) (*CommandResult, error)
}
```

### 3. AccountPolicyService (密码策略服务)

```go
type AccountPolicyService interface {
    CreatePolicy(ctx context.Context, req *CreatePolicyRequest) (*VMPasswordPolicy, error)
    ValidatePassword(ctx context.Context, password string, policyID string) error
    GetPolicyForVM(ctx context.Context, vmID string) (*VMPasswordPolicy, error)
}
```

## 实施阶段建议

### Phase 22-06: VM账号管理数据模型（1-2天）
- 创建数据表
- 编写迁移脚本
- 定义Service接口

### Phase 22-07: VM Agent开发（3-4天）
- 开发Windows Agent
- 开发Linux Agent
- 实现gRPC/REST通信
- 编写安装脚本

### Phase 22-08: 后端账号管理API（2-3天）
- 实现VMAccountService
- 实现VMAgentManager
- 实现AccountPolicyService
- 创建Handler和Router

### Phase 22-09: 前端账号管理UI（2-3天）
- 账号列表页面
- 账号创建/编辑对话框
- 密码策略配置页面
- Agent状态监控页面

### Phase 22-10: 测试与文档（1-2天）
- 单元测试
- 集成测试
- 用户文档
- API文档

**总计**: 约11-14天工作量
