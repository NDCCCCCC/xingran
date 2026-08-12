# Phase 22-06 至 22-10: VM账号管理系统

**目标**: 扩展Phase 22，增加虚拟机内部账号管理功能，通过VM Agent实现完整的账号生命周期管理。

**依赖**: Phase 22 Wave 1-5（基础VDI集成）

**新增Wave数**: 5个（22-06至22-10）

---

## Wave 6: VM账号管理数据模型（22-06）

**目标**: 创建VM账号管理相关的数据表和基础结构

**任务数**: 4 | **文件数**: 6 | **预估时间**: 2-3小时

### Task 6.1: 创建账号管理数据模型

**文件**: `internal/models/vdi_account.go`

**内容**:
```go
package models

import "github.com/xingran-next/xingran-go-backend/internal/models/base"

// VDIVMAccount 虚拟机内部账号
type VDIVMAccount struct {
    base.Base
    VMID          string `gorm:"type:varchar(100);index;not null" json:"vm_id"`
    AccountID     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"account_id"`
    Username      string `gorm:"type:varchar(100);not null" json:"username"`
    PasswordEncrypted string `gorm:"type:varchar(500);not null" json:"-"`
    AccountType   string `gorm:"type:varchar(50);not null" json:"account_type"`
    OSType        string `gorm:"type:varchar(50);not null" json:"os_type"`
    IsAdmin       bool   `gorm:"type:bool;default:false" json:"is_admin"`
    IsEnabled     bool   `gorm:"type:bool;default:true" json:"is_enabled"`
    Description   *string `gorm:"type:varchar(500)" json:"description"`
    SyncStatus    string `gorm:"type:varchar(50);default:'pending'" json:"sync_status"`
}

// VDIVMAuditLog 账号操作审计日志
type VDIVMAuditLog struct {
    base.Base
    VMID          string `gorm:"type:varchar(100);index" json:"vm_id"`
    AccountID     string `gorm:"type:varchar(100);index" json:"account_id"`
    Operation     string `gorm:"type:varchar(50);not null" json:"operation"`
    Operator      string `gorm:"type:varchar(100);not null" json:"operator"`
    Details       string `gorm:"type:text" json:"details"`
    Status        string `gorm:"type:varchar(50);not null" json:"status"`
    ExecutedAt    *time.Time `json:"executed_at"`
}

// VMPasswordPolicy 密码策略
type VMPasswordPolicy struct {
    base.Base
    PolicyName    string `gorm:"type:varchar(200);not null;uniqueIndex" json:"policy_name"`
    MinLength     int `gorm:"type:int;default:8" json:"min_length"`
    RequireUppercase bool `gorm:"type:bool;default:true" json:"require_uppercase"`
    MaxAgeDays    int `gorm:"type:int;default:90" json:"max_age_days"`
    IsEnabled     bool `gorm:"type:bool;default:true" json:"is_enabled"`
}
```

**验证**:
- [ ] 模型编译通过
- [ ] GORM标签正确
- [ ] JSON序列化正常

---

### Task 6.2: 扩展VM表添加账号字段

**文件**: `internal/models/vdi.go`

**修改**: 在VDIVirtualMachine结构体中添加字段

```go
// 在现有VDIVirtualMachine中添加：
// 初始管理员账号信息
InitialAdminUsername  *string `gorm:"type:varchar(100)" json:"initial_admin_username"`
InitialAdminPasswordEncrypted *string `gorm:"column:initial_admin_password;type:varchar(500)" json:"-"`

// Agent信息
AgentInstalled       bool   `gorm:"type:bool;default:false" json:"agent_installed"`
AgentVersion         *string `gorm:"type:varchar(50)" json:"agent_version"`
AgentLastHeartbeat   *time.Time `json:"agent_last_heartbeat"`

// 账号管理策略
PasswordPolicyID     *string `gorm:"type:varchar(100)" json:"password_policy_id"`
```

**验证**:
- [ ] 字段添加成功
- [ ] 数据库迁移成功

---

### Task 6.3: 创建数据库迁移脚本

**文件**: `internal/core/db/migrations/086_create_vdi_account_tables.sql`

**内容**:
```sql
-- VM账号表
CREATE TABLE IF NOT EXISTS sys_vdi_vm_accounts (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    vm_id VARCHAR(100) NOT NULL,
    account_id VARCHAR(100) NOT NULL,
    username VARCHAR(100) NOT NULL,
    password_encrypted VARCHAR(500) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    os_type VARCHAR(50) NOT NULL,
    is_admin BOOLEAN DEFAULT false,
    is_enabled BOOLEAN DEFAULT true,
    description VARCHAR(500),
    sync_status VARCHAR(50) DEFAULT 'pending',
    synced_at TIMESTAMP,
    last_sync_error TEXT
);

CREATE INDEX idx_vm_accounts_vm_id ON sys_vdi_vm_accounts(vm_id);
CREATE UNIQUE INDEX idx_vm_accounts_account_id ON sys_vdi_vm_accounts(account_id);

-- 审计日志表
CREATE TABLE IF NOT EXISTS sys_vdi_vm_audit_logs (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    vm_id VARCHAR(100),
    account_id VARCHAR(100),
    operation VARCHAR(50) NOT NULL,
    operator VARCHAR(100) NOT NULL,
    details TEXT,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    executed_at TIMESTAMP
);

CREATE INDEX idx_vm_audit_logs_vm_id ON sys_vdi_vm_audit_logs(vm_id);
CREATE INDEX idx_vm_audit_logs_account_id ON sys_vdi_vm_audit_logs(account_id);

-- 密码策略表
CREATE TABLE IF NOT EXISTS sys_vdi_password_policies (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    policy_name VARCHAR(200) NOT NULL UNIQUE,
    min_length INT DEFAULT 8,
    max_length INT DEFAULT 32,
    require_uppercase BOOLEAN DEFAULT true,
    require_lowercase BOOLEAN DEFAULT true,
    require_number BOOLEAN DEFAULT true,
    require_special BOOLEAN DEFAULT true,
    max_age_days INT DEFAULT 90,
    is_enabled BOOLEAN DEFAULT true
);
```

**验证**:
- [ ] 迁移脚本执行成功
- [ ] 表结构创建成功
- [ ] 索引创建成功

---

### Task 6.4: 配置VM账号相关配置

**文件**: `internal/config/vdi_config.go`

**修改**: 添加Agent相关配置

```go
type VDIConfig struct {
    // ... 现有配置 ...

    // Agent配置
    Agent AgentConfig `yaml:"agent" validate:"required"`
}

type AgentConfig struct {
    Enabled         bool          `yaml:"enabled"`
    ServerURL       string        `yaml:"server_url"`
    ServerPort      int           `yaml:"server_port"`
    HeartbeatInterval int         `yaml:"heartbeat_interval"`
    CommandTimeout  time.Duration `yaml:"command_timeout"`
    TLSEnabled      bool          `yaml:"tls_enabled"`
}
```

**文件**: `configs/config.yaml`

**内容**:
```yaml
vdi:
  # ... 现有配置 ...

  # VM Agent配置
  agent:
    enabled: true
    server_url: "https://xingran-backend.example.com"
    server_port: 443
    heartbeat_interval: 30  # 秒
    command_timeout: 60s
    tls_enabled: true
```

**验证**:
- [ ] 配置加载成功
- [ ] 配置验证通过

---

## Wave 7: VM Agent开发（22-07）

**目标**: 开发Windows和Linux版本的VM Agent

**任务数**: 4 | **文件数**: 8+ | **预估时间**: 3-4天

### Task 7.1: 设计Agent通信协议

**文件**: `pkg/agent/protocol.proto`（gRPC定义）

**内容**:
```protobuf
syntax = "proto3";

package xingran.agent;

service VMAgentService {
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc CreateAccount(CreateAccountRequest) returns (AccountResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (AccountResponse);
  rpc UpdatePassword(UpdatePasswordRequest) returns (AccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc GetSystemInfo(SystemInfoRequest) returns (SystemInfoResponse);
}

message HeartbeatRequest {
  string vm_id = 1;
  int64 timestamp = 2;
  SystemStatus status = 3;
}

message SystemStatus {
  double cpu_usage = 1;
  double memory_usage = 2;
  double disk_usage = 3;
  int32 running_processes = 4;
  repeated string logged_in_users = 5;
}

message CreateAccountRequest {
  string vm_id = 1;
  string username = 2;
  string password = 3;
  string account_type = 4;
  bool is_admin = 5;
  string description = 6;
}

message AccountResponse {
  int32 code = 1;
  string message = 2;
  Account account = 3;
}

message Account {
  string account_id = 1;
  string username = 2;
  string account_type = 3;
  bool is_admin = 4;
  bool is_enabled = 5;
}
```

**验证**:
- [ ] proto文件编译成功
- [ ] 生成Go代码

---

### Task 7.2: 开发Windows Agent

**文件**:
- `cmd/agent/windows/main.go` - 入口
- `internal/agent/windows/account_manager.go` - 账号管理
- `internal/agent/windows/system_monitor.go` - 系统监控
- `internal/agent/network/listener.go` - 网络监听

**核心功能**:
```go
// Windows账号管理
type WindowsAccountManager struct{}

func (m *WindowsAccountManager) CreateAccount(username, password string, isAdmin bool) error {
    // 使用PowerShell命令创建用户
    cmd := exec.Command("powershell", "-Command",
        fmt.Sprintf("New-LocalUser -Name %s -Password (ConvertTo-SecureString %s -AsPlainText -Force)", username, password))
    if isAdmin {
        // 添加到管理员组
        cmd2 := exec.Command("powershell", "-Command",
            fmt.Sprintf("Add-LocalGroupMember -Group 'Administrators' -Member %s", username))
        cmd2.Run()
    }
    return cmd.Run()
}

func (m *WindowsAccountManager) DeleteAccount(username string) error {
    cmd := exec.Command("powershell", "-Command",
        fmt.Sprintf("Remove-LocalUser -Name %s", username))
    return cmd.Run()
}
```

**验证**:
- [ ] Windows可执行文件编译成功
- [ ] 账号创建功能正常
- [ ] 心跳功能正常

---

### Task 7.3: 开发Linux Agent

**文件**:
- `cmd/agent/linux/main.go` - 入口
- `internal/agent/linux/account_manager.go` - 账号管理
- `internal/agent/linux/system_monitor.go` - 系统监控
- `internal/agent/network/listener.go` - 网络监听

**核心功能**:
```go
// Linux账号管理
type LinuxAccountManager struct{}

func (m *LinuxAccountManager) CreateAccount(username, password string, isAdmin bool) error {
    // 创建用户
    cmd := exec.Command("useradd", "-m", username)
    if err := cmd.Run(); err != nil {
        return err
    }

    // 设置密码
    cmd = exec.Command("sh", "-c",
        fmt.Sprintf("echo '%s:%s' | chpasswd", username, password))
    if err := cmd.Run(); err != nil {
        return err
    }

    // 添加到sudo组（如果是管理员）
    if isAdmin {
        cmd = exec.Command("usermod", "-aG", "sudo", username)
        return cmd.Run()
    }

    return nil
}

func (m *LinuxAccountManager) DeleteAccount(username string) error {
    cmd := exec.Command("userdel", "-r", username)
    return cmd.Run()
}
```

**验证**:
- [ ] Linux可执行文件编译成功
- [ ] 账号创建功能正常
- [ ] 心跳功能正常

---

### Task 7.4: 创建Agent安装脚本

**文件**:
- `scripts/agent/install-windows-agent.ps1`
- `scripts/agent/install-linux-agent.sh`

**Windows安装脚本**:
```powershell
# install-windows-agent.ps1
param(
    [string]$VMID,
    [string]$ServerURL = "https://xingran-backend.example.com"
)

$AGENT_VERSION = "1.0.0"
$INSTALL_PATH = "C:\Program Files\XingRanAgent"

Write-Host "Installing XingRan VM Agent..."

# 创建安装目录
New-Item -ItemType Directory -Path $INSTALL_PATH -Force | Out-Null

# 下载Agent
$AGENT_URL = "$ServerURL/api/v1/agent/download/windows/$AGENT_VERSION"
Invoke-WebRequest -Uri $AGENT_URL -OutFile "$INSTALL_PATH\agent.zip"
Expand-Archive -Path "$INSTALL_PATH\agent.zip" -DestinationPath $INSTALL_PATH -Force

# 配置Agent
$config = @{
    vm_id = $VMID
    server_url = $ServerURL
    server_port = 443
    heartbeat_interval = 30
    tls_enabled = $true
}
$config | ConvertTo-Json | Set-Content "$INSTALL_PATH\config.json"

# 注册Windows服务
$EXE_PATH = "$INSTALL_PATH\agent.exe"
New-Service -Name "XingRanVMAgent" `
    -BinaryPathName $EXE_PATH `
    -DisplayName "XingRan VM Management Agent" `
    -Description "Agent for managing VM accounts and monitoring system status" `
    -StartupType Automatic

# 启动服务
Start-Service "XingRanVMAgent"

# 注册到后端
Write-Host "Registering agent to server..."
$REGISTER_URL = "$ServerURL/api/v1/agent/register"
Invoke-WebRequest -Uri $REGISTER_URL `
    -Method POST `
    -Body ($config | ConvertTo-Json) `
    -ContentType "application/json"

Write-Host "Agent installed successfully!"
```

**Linux安装脚本**:
```bash
#!/bin/bash
# install-linux-agent.sh

VM_ID="${1:-$(hostname)}"
SERVER_URL="${2:-https://xingran-backend.example.com}"
AGENT_VERSION="1.0.0"
INSTALL_PATH="/opt/xingran-agent"

echo "Installing XingRan VM Agent..."

# 创建安装目录
mkdir -p $INSTALL_PATH

# 下载Agent
AGENT_URL="$SERVER_URL/api/v1/agent/download/linux/$AGENT_VERSION"
wget -O /tmp/agent.tar.gz $AGENT_URL
tar -xzf /tmp/agent.tar.gz -C $INSTALL_PATH

# 配置Agent
cat > $INSTALL_PATH/config.json << EOF
{
  "vm_id": "$VM_ID",
  "server_url": "$SERVER_URL",
  "server_port": 443,
  "heartbeat_interval": 30,
  "tls_enabled": true
}
EOF

# 注册systemd服务
cat > /etc/systemd/system/xingran-vm-agent.service << EOF
[Unit]
Description=XingRan VM Management Agent
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

# 启动服务
systemctl daemon-reload
systemctl enable xingran-vm-agent
systemctl start xingran-vm-agent

# 注册到后端
echo "Registering agent to server..."
REGISTER_URL="$SERVER_URL/api/v1/agent/register"
curl -X POST $REGISTER_URL \
  -H "Content-Type: application/json" \
  -d @$INSTALL_PATH/config.json

echo "Agent installed successfully!"
```

**验证**:
- [ ] Windows脚本测试通过
- [ ] Linux脚本测试通过
- [ ] Agent自动注册成功

---

## Wave 8: 后端账号管理API（22-08）

**目标**: 实现VM账号管理的后端服务和API

**任务数**: 5 | **文件数**: 8 | **预估时间**: 2-3天

### Task 8.1: 实现VMAccountService

**文件**: `internal/services/vdi/vm_account_service.go`

**接口定义**:
```go
package vdi

import "context"

type VMAccountService interface {
    // CRUD操作
    CreateAccount(ctx context.Context, req *CreateAccountRequest) (*VDIVMAccount, error)
    GetAccount(ctx context.Context, accountID string) (*VDIVMAccountDTO, error)
    ListAccounts(ctx context.Context, vmID string, req *ListAccountsRequest) (*PageResult, error)
    UpdateAccount(ctx context.Context, accountID string, req *UpdateAccountRequest) error
    DeleteAccount(ctx context.Context, accountID string) error

    // 账号操作
    ResetPassword(ctx context.Context, accountID, newPassword string) error
    EnableAccount(ctx context.Context, accountID string) error
    DisableAccount(ctx context.Context, accountID string) error
    SyncAccountToVM(ctx context.Context, accountID string) error

    // 批量操作
    BatchCreateAccounts(ctx context.Context, vmID string, req []CreateAccountRequest) (*BatchResult, error)
    BatchDeleteAccounts(ctx context.Context, accountIDs []string) (*BatchResult, error)
}

type CreateAccountRequest struct {
    VMID        string `json:"vm_id" validate:"required"`
    Username    string `json:"username" validate:"required"`
    Password    string `json:"password" validate:"required,min=8"`
    AccountType string `json:"account_type" validate:"required,oneof=admin user service"`
    OSType      string `json:"os_type" validate:"required,oneof=windows linux"`
    IsAdmin     bool   `json:"is_admin"`
    Description string `json:"description"`
}
```

**验证**:
- [ ] 接口定义完整
- [ ] 依赖注入正常

---

### Task 8.2: 实现VMAgentManager

**文件**: `internal/services/vdi/vm_agent_manager.go`

**接口定义**:
```go
type VMAgentManager interface {
    // Agent管理
    RegisterAgent(ctx context.Context, req *RegisterAgentRequest) (*VMAgent, error)
    UnregisterAgent(ctx context.Context, agentID string) error

    // 心跳处理
    HandleHeartbeat(ctx context.Context, req *HeartbeatRequest) error

    // Agent状态
    GetAgentStatus(ctx context.Context, vmID string) (*VMAgentStatus, error)
    ListOnlineAgents(ctx context.Context) ([]VMAgent, error)

    // 远程命令执行
    ExecuteAccountCommand(ctx context.Context, vmID string, cmd *AccountCommand) (*CommandResult, error)
}

type AccountCommand struct {
    VMID      string `json:"vm_id"`
    Operation string `json:"operation"` // create/delete/update_password/enable/disable
    Account   *VDIVMAccount `json:"account"`
}

type CommandResult struct {
    Success    bool   `json:"success"`
    Message    string `json:"message"`
    Data       string `json:"data"`
    ExecutedAt time.Time `json:"executed_at"`
}
```

**验证**:
- [ ] Agent注册功能正常
- [ ] 心跳处理正常
- [ ] 命令执行正常

---

### Task 8.3: 创建VM Account Handler

**文件**: `internal/api/v1/vdi/vm_account_handler.go`

**内容**:
```go
package vdi

import (
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

type VMAccountHandler struct {
    accountService VDIVMAccountService
    agentManager   VMAgentManager
}

func NewVMAccountHandler(accountService VDIVMAccountService, agentManager VMAgentManager) *VMAccountHandler {
    return &VMAccountHandler{
        accountService: accountService,
        agentManager:   agentManager,
    }
}

// CreateAccount 创建VM账号
func (h *VMAccountHandler) CreateAccount(c *gin.Context) {
    var req vdi.CreateAccountRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, 400, "请求参数错误")
        return
    }

    account, err := h.accountService.CreateAccount(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }

    response.Success(c, account)
}

// ListAccounts 查询VM账号列表
func (h *VMAccountHandler) ListAccounts(c *gin.Context) {
    vmID := c.Param("vmId")
    var req vdi.ListAccountsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, 400, "请求参数错误")
        return
    }

    req.VMID = vmID
    result, err := h.accountService.ListAccounts(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }

    response.Success(c, result)
}

// ResetPassword 重置账号密码
func (h *VMAccountHandler) ResetPassword(c *gin.Context) {
    accountID := c.Param("accountId")
    var req struct {
        NewPassword string `json:"new_password" validate:"required,min=8"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, 400, "请求参数错误")
        return
    }

    err := h.accountService.ResetPassword(c.Request.Context(), accountID, req.NewPassword)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }

    response.Success(c, gin.H{"message": "密码重置成功"})
}

// SyncAccount 同步账号到VM
func (h *VMAccountHandler) SyncAccount(c *gin.Context) {
    accountID := c.Param("accountId")

    err := h.accountService.SyncAccountToVM(c.Request.Context(), accountID)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }

    response.Success(c, gin.H{"message": "账号同步成功"})
}
```

**验证**:
- [ ] Handler编译通过
- [ ] 参数验证正常
- [ ] 错误处理完整

---

### Task 8.4: 创建VM Account Router

**文件**: `internal/api/v1/vdi/vm_account_router.go`

**内容**:
```go
package vdi

import (
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/core"
    "github.com/xingran-next/xingran-go-backend/pkg/middleware"
)

func SetupVMAccountRouter(r *gin.RouterGroup, core *core.Core) {
    accountService := vdi.NewVMAccountService(core.GetDB(), core.Cache)
    agentManager := vdi.NewVMAgentManager(core.GetDB(), core.Cache)
    accountHandler := vdi.NewVMAccountHandler(accountService, agentManager)

    // 需要认证
    accountGroup := r.Group("")
    accountGroup.Use(middleware.Auth())
    {
        // 账号CRUD
        accountGroup.POST("/list", accountHandler.ListAccounts)
        accountGroup.POST("", accountHandler.CreateAccount)
        accountGroup.POST("/:accountId", accountHandler.GetAccount)
        accountGroup.POST("/:accountId/update", accountHandler.UpdateAccount)
        accountGroup.POST("/:accountId/delete", accountHandler.DeleteAccount)

        // 账号操作
        accountGroup.POST("/:accountId/reset-password", accountHandler.ResetPassword)
        accountGroup.POST("/:accountId/enable", accountHandler.EnableAccount)
        accountGroup.POST("/:accountId/disable", accountHandler.DisableAccount)
        accountGroup.POST("/:accountId/sync", accountHandler.SyncAccount)

        // 批量操作
        accountGroup.POST("/batch/create", accountHandler.BatchCreateAccounts)
        accountGroup.POST("/batch/delete", accountHandler.BatchDeleteAccounts)
    }

    // Agent端点（Agent调用，不使用用户JWT）
    agentGroup := r.Group("/agent")
    {
        agentGroup.POST("/register", accountHandler.RegisterAgent)
        agentGroup.POST("/heartbeat", accountHandler.HandleHeartbeat)
        agentGroup.POST("/command/result", accountHandler.HandleCommandResult)
    }
}
```

**验证**:
- [ ] 路由注册成功
- [ ] 认证中间件正常
- [ ] 权限验证正常

---

### Task 8.5: 注册到主路由

**文件**: `internal/api/router.go`

**修改**: 在VDI路由组中添加账号路由

```go
// 在VDI路由组中添加
vdiGroup := api.Group("/vdi")
{
    // ... 现有路由 ...

    // VM账号管理
    vdi.SetupVMAccountRouter(vdiGroup, core)
}
```

**验证**:
- [ ] 路由正常工作
- [ ] API可访问

---

## Wave 9: 前端账号管理UI（22-09）

**目标**: 创建VM账号管理的前端界面

**任务数**: 5 | **文件数**: 7 | **预估时间**: 2-3天

### Task 9.1: 创建账号管理类型定义

**文件**: `src/types/vdiAccount.ts`

**内容**:
```typescript
// VM账号类型
export interface VDIVMAccount {
  id: string;
  vm_id: string;
  account_id: string;
  username: string;
  account_type: 'admin' | 'user' | 'service';
  os_type: 'windows' | 'linux';
  is_admin: boolean;
  is_enabled: boolean;
  description?: string;
  sync_status: 'pending' | 'synced' | 'failed';
  synced_at?: string;
  created_at: string;
  updated_at: string;
}

// 创建账号请求
export interface CreateAccountRequest {
  vm_id: string;
  username: string;
  password: string;
  account_type: 'admin' | 'user' | 'service';
  os_type: 'windows' | 'linux';
  is_admin: boolean;
  description?: string;
}

// Agent状态
export interface VMAgentStatus {
  vm_id: string;
  agent_installed: boolean;
  agent_version?: string;
  agent_last_heartbeat?: string;
  online: boolean;
  system_status?: {
    cpu_usage: number;
    memory_usage: number;
    disk_usage: number;
  };
}
```

**验证**:
- [ ] 类型定义完整
- [ ] TypeScript编译通过

---

### Task 9.2: 创建账号管理API客户端

**文件**: `src/lib/vdiAccountApi.ts`

**内容**:
```typescript
import { post } from '@/lib/api';
import type { VDIVMAccount, CreateAccountRequest, PageResult } from '@/types/vdiAccount';

export const vmAccountApi = {
  // 账号CRUD
  list: (vmId: string, params: { current: number; pageSize: number }) =>
    post(`/vdi/vm/${vmId}/accounts/list`, params),

  get: (accountId: string) =>
    post(`/vdi/vm/accounts/${accountId}`),

  create: (data: CreateAccountRequest) =>
    post('/vdi/vm/accounts', data),

  update: (accountId: string, data: Partial<VDIVMAccount>) =>
    post(`/vdi/vm/accounts/${accountId}/update`, data),

  delete: (accountId: string) =>
    post(`/vdi/vm/accounts/${accountId}/delete`),

  // 账号操作
  resetPassword: (accountId: string, newPassword: string) =>
    post(`/vdi/vm/accounts/${accountId}/reset-password`, { new_password: newPassword }),

  enable: (accountId: string) =>
    post(`/vdi/vm/accounts/${accountId}/enable`),

  disable: (accountId: string) =>
    post(`/vdi/vm/accounts/${accountId}/disable`),

  sync: (accountId: string) =>
    post(`/vdi/vm/accounts/${accountId}/sync`),

  // 批量操作
  batchCreate: (vmId: string, accounts: CreateAccountRequest[]) =>
    post(`/vdi/vm/${vmId}/accounts/batch/create`, { accounts }),

  batchDelete: (accountIds: string[]) =>
    post('/vdi/vm/accounts/batch/delete', { account_ids: accountIds }),
};
```

**验证**:
- [ ] API封装完整
- [ ] 类型安全

---

### Task 9.3: 创建账号列表页面

**文件**: `src/pages/vdi/VMAccountList/index.tsx`

**内容**:
```typescript
import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, message, Space, Tag } from 'antd';
import { PlusOutlined, SyncOutlined, LockOutlined, UnlockOutlined } from '@ant-design/icons';
import { vmAccountApi } from '@/lib/vdiAccountApi';
import type { VDIVMAccount } from '@/types/vdiAccount';

const VMAccountList: React.FC = () => {
  const [accounts, setAccounts] = useState<VDIVMAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const vmId = 'vm-12345'; // 从路由参数获取

  useEffect(() => {
    fetchAccounts();
  }, [vmId]);

  const fetchAccounts = async () => {
    setLoading(true);
    try {
      const result = await vmAccountApi.list(vmId, {
        current: 1,
        pageSize: 100,
      });
      setAccounts(result.data.list);
    } catch (error) {
      message.error('获取账号列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (values: any) => {
    try {
      await vmAccountApi.create({
        vm_id: vmId,
        ...values,
      });
      message.success('账号创建成功');
      setCreateModalVisible(false);
      fetchAccounts();
    } catch (error) {
      message.error('账号创建失败');
    }
  };

  const handleResetPassword = async (accountId: string) => {
    Modal.confirm({
      title: '重置密码',
      content: (
        <Input.Password
          placeholder="请输入新密码"
          id="reset-password-input"
        />
      ),
      onOk: async () => {
        const newPassword = (document.getElementById('reset-password-input') as HTMLInputElement)?.value;
        if (!newPassword) {
          message.error('请输入新密码');
          return;
        }
        try {
          await vmAccountApi.resetPassword(accountId, newPassword);
          message.success('密码重置成功');
        } catch (error) {
          message.error('密码重置失败');
        }
      },
    });
  };

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username' },
    {
      title: '类型',
      dataIndex: 'account_type',
      key: 'account_type',
      render: (type: string) => {
        const colors = { admin: 'red', user: 'blue', service: 'green' };
        return <Tag color={colors[type as keyof typeof colors]}>{type}</Tag>;
      },
    },
    {
      title: '系统',
      dataIndex: 'os_type',
      key: 'os_type',
      render: (type: string) => type.toUpperCase(),
    },
    {
      title: '管理员',
      dataIndex: 'is_admin',
      key: 'is_admin',
      render: (isAdmin: boolean) => (isAdmin ? '是' : '否'),
    },
    {
      title: '状态',
      dataIndex: 'is_enabled',
      key: 'is_enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '同步状态',
      dataIndex: 'sync_status',
      key: 'sync_status',
      render: (status: string) => {
        const colors = { pending: 'default', synced: 'success', failed: 'error' };
        const labels = { pending: '待同步', synced: '已同步', failed: '同步失败' };
        return <Tag color={colors[status as keyof typeof colors]}>{labels[status as keyof typeof labels]}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: VDIVMAccount) => (
        <Space>
          <Button
            type="link"
            icon={<LockOutlined />}
            onClick={() => handleResetPassword(record.id)}
          >
            重置密码
          </Button>
          <Button
            type="link"
            icon={<SyncOutlined />}
            onClick={() => handleSync(record.id)}
          >
            同步
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateModalVisible(true)}
        >
          创建账号
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={accounts}
        loading={loading}
        rowKey="id"
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
      />

      <Modal
        title="创建账号"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
      >
        <Form onFinish={handleCreate} layout="vertical">
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少8位' },
            ]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>

          <Form.Item
            name="account_type"
            label="账号类型"
            rules={[{ required: true, message: '请选择账号类型' }]}
          >
            <Select placeholder="请选择账号类型">
              <Select.Option value="user">普通用户</Select.Option>
              <Select.Option value="admin">管理员</Select.Option>
              <Select.Option value="service">服务账号</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="os_type"
            label="操作系统"
            rules={[{ required: true, message: '请选择操作系统' }]}
          >
            <Select placeholder="请选择操作系统">
              <Select.Option value="windows">Windows</Select.Option>
              <Select.Option value="linux">Linux</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item name="is_admin" valuePropName="checked">
            <Checkbox>设为管理员</Checkbox>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="请输入描述" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              创建
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default VMAccountList;
```

**验证**:
- [ ] 页面渲染正常
- [ ] 账号列表显示正常
- [ ] 创建账号功能正常

---

### Task 9.4: 创建Agent状态监控页面

**文件**: `src/pages/vdi/VMAgentMonitor/index.tsx`

**功能**:
- 显示所有VM的Agent在线状态
- 显示系统资源使用率
- 显示最近心跳时间
- 支持远程执行命令

**验证**:
- [ ] 监控页面正常
- [ ] 实时状态更新
- [ ] 心跳显示正常

---

### Task 9.5: 集成到虚拟机详情页

**文件**: `src/pages/vdi/VirtualMachineList/index.tsx`（修改）

**修改**: 在虚拟机详情对话框中添加账号管理标签页

**验证**:
- [ ] 标签页正常显示
- [ ] 账号列表嵌入成功
- [ ] 数据联动正常

---

## Wave 10: 测试与文档（22-10）

**目标**: 完成测试和文档编写

**任务数**: 4 | **文件数**: 5+ | **预估时间**: 1-2天

### Task 10.1: 编写单元测试

**文件**:
- `internal/services/vdi/vm_account_service_test.go`
- `internal/services/vdi/vm_agent_manager_test.go`
- `internal/api/v1/vdi/vm_account_handler_test.go`

**覆盖**:
- Service层逻辑测试
- Agent管理测试
- 参数验证测试

**目标**: 覆盖率 > 80%

---

### Task 10.2: 编写集成测试

**文件**:
- `tests/integration/vm_account_test.go`

**测试场景**:
- 创建账号 → 同步到VM → 验证账号存在
- 修改密码 → 验证新密码可以登录
- 删除账号 → 验证账号已删除
- Agent心跳 → 验证状态更新

---

### Task 10.3: 用户文档

**文件**: `docs/VM账号管理使用手册.md`

**内容**:
1. 功能概述
2. Agent安装指南
3. 账号管理操作指南
4. 常见问题排查
5. 安全注意事项

---

### Task 10.4: API文档

**文件**: `docs/VM账号管理API.md`

**内容**:
- API端点列表
- 请求/响应格式
- 错误码说明
- 调用示例

---

## 总结

**Phase 22 扩展后**: 10个Wave，44个任务，65+文件

**总工作量**: 
- Wave 1-5（基础VDI）: 16-21小时
- Wave 6-10（账号管理）: +25-30小时
- **总计**: 41-51小时（5-7个工作日）

**核心成果**:
✅ 完整的VDI平台集成
✅ VM内部账号完全管理
✅ 生产级安全设计
✅ 完善的前后端功能
✅ 详细的测试和文档
