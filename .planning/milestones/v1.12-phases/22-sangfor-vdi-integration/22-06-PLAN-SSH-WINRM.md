# Phase 22-06: VM账号管理（SSH/WinRM方案）

**目标**: 使用SSH/WinRM协议直接管理虚拟机内部账号，无需Agent

**技术栈**:
- Linux: golang.org/x/crypto/ssh
- Windows: github.com/masterzen/winrm
- 安全: SM4密码加密存储

**工作量**: 3个Wave，12个任务，8-12小时

---

## Wave 6: 数据模型扩展（22-06）

**目标**: 扩展VM表，添加SSH/WinRM连接信息

**任务数**: 3 | **文件数**: 3 | **预估时间**: 1-2小时

### Task 6.1: 扩展VM数据模型

**文件**: `internal/models/vdi.go`

**修改**: 添加SSH/WinRM连接字段

```go
package models

import "time"

// VDIVirtualMachine 虚拟机模型（扩展版）
type VDIVirtualMachine struct {
    Base

    // === 原有字段 ===
    VMID           string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"vm_id"`
    Name           string    `gorm:"type:varchar(200);not null" json:"name"`
    ResourceID     string    `gorm:"type:varchar(100);index" json:"resource_id"`
    Status         int       `gorm:"type:int;default:0" json:"status"`
    PowerState     string    `gorm:"type:varchar(50)" json:"power_state"`
    IPAddress      string    `gorm:"type:varchar(50)" json:"ip_address"`
    MACAddress     string    `gorm:"type:varchar(50)" json:"mac_address"`
    OSType         string    `gorm:"type:varchar(50)" json:"os_type"` // windows/linux
    VdiServerID    string    `gorm:"type:varchar(100);index" json:"vdi_server_id"`

    // === 新增：SSH/WinRM连接信息 ===
    
    // SSH连接配置（Linux）
    SSHEnabled         bool    `gorm:"type:bool;default:false" json:"ssh_enabled"`
    SSHPort            int     `gorm:"type:int;default:22" json:"ssh_port"`
    SSHUsername        string  `gorm:"type:varchar(100)" json:"ssh_username"`
    SSHPasswordEncrypted string `gorm:"column:ssh_password;type:varchar(500)" json:"-"`
    SSHAuthMethod      string  `gorm:"type:varchar(50);default:'password'" json:"ssh_auth_method"` // password/publickey
    SSHPrivateKeyEncrypted *string `gorm:"type:text" json:"-"` // 如果使用公钥认证

    // WinRM连接配置（Windows）
    WinRMEnabled         bool    `gorm:"type:bool;default:false" json:"winrm_enabled"`
    WinRMPort            int     `gorm:"type:int;default:5985" json:"winrm_port"`
    WinRMUseHTTPS        bool    `gorm:"type:bool;default:false" json:"winrm_use_https"`
    WinRMUsername        string  `gorm:"type:varchar(100)" json:"winrm_username"`
    WinRMPasswordEncrypted string `gorm:"column:winrm_password;type:varchar(500)" json:"-"`

    // 连接状态
    LastConnectionTest  *time.Time `json:"last_connection_test"`
    ConnectionTestStatus string    `gorm:"type:varchar(50)" json:"connection_test_status"` // success/failed/unknown
    ConnectionTestError  *string   `gorm:"type:text" json:"connection_test_error"`

    // 账号管理策略
    AllowAccountManagement bool `gorm:"type:bool;default:true" json:"allow_account_management"`
    AccountManagementPolicyID *string `gorm:"type:varchar(100)" json:"account_management_policy_id"`
}
```

**验证**:
- [ ] 模型编译通过
- [ ] GORM标签正确
- [ ] JSON序列化正常（密码字段不返回前端）

---

### Task 6.2: 创建账号管理相关模型

**文件**: `internal/models/vdi_account.go`

**内容**:
```go
package models

import "time"

// VDIVMAccount 虚拟机内部账号
type VDIVMAccount struct {
    Base

    VMID          string `gorm:"type:varchar(100);index;not null" json:"vm_id"`
    AccountID     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"account_id"`
    
    // 账号信息
    Username      string `gorm:"type:varchar(100);not null" json:"username"`
    // 注意：不存储密码，密码只在创建/重置时使用
    AccountType   string `gorm:"type:varchar(50);not null" json:"account_type"` // admin/user/service
    OSType        string `gorm:"type:varchar(50);not null" json:"os_type"`      // windows/linux
    
    // 账号属性
    IsAdmin       bool   `gorm:"type:bool;default:false" json:"is_admin"`
    IsEnabled     bool   `gorm:"type:bool;default:true" json:"is_enabled"`
    Description   *string `gorm:"type:varchar(500)" json:"description"`
    
    // Linux特定属性
    UID           *int    `json:"uid"`
    GID           *int    `json:"gid"`
    HomeDir       *string `gorm:"type:varchar(500)" json:"home_dir"`
    Shell         *string `gorm:"type:varchar(200)" json:"shell"`
    
    // 同步状态
    SyncStatus    string `gorm:"type:varchar(50);default:'pending'" json:"sync_status"` // pending/synced/failed
    SyncedAt      *time.Time `json:"synced_at"`
    LastSyncError *string `gorm:"type:text" json:"last_sync_error"`
    
    // 元数据
    CreatedBy     string `gorm:"type:varchar(100)" json:"created_by"`
    UpdatedBy     string `gorm:"type:varchar(100)" json:"updated_by"`
}

func (VDIVMAccount) TableName() string {
    return "sys_vdi_vm_accounts"
}

// VDIVMAuditLog 账号操作审计日志
type VDIVMAuditLog struct {
    Base

    VMID          string `gorm:"type:varchar(100);index" json:"vm_id"`
    AccountID     string `gorm:"type:varchar(100);index" json:"account_id"`

    // 操作信息
    Operation     string `gorm:"type:varchar(50);not null" json:"operation"` // create/delete/reset_password/enable/disable/sync
    Operator      string `gorm:"type:varchar(100);not null" json:"operator"`
    OperatorIP    string `gorm:"type:varchar(50)" json:"operator_ip"`

    // 操作详情
    Details       string `gorm:"type:text" json:"details"` // JSON格式的详细信息
    
    // 执行结果
    Status        string `gorm:"type:varchar(50);not null" json:"status"` // success/failed
    ErrorMessage  *string `gorm:"type:text" json:"error_message"`
    
    // 执行记录
    CommandUsed   string `gorm:"type:text" json:"command_used"` // 实际执行的SSH/WinRM命令
    ExecutionTime int    `gorm:"type:int" json:"execution_time"` // 毫秒
    ExecutedAt    *time.Time `json:"executed_at"`
}

func (VDIVMAuditLog) TableName() string {
    return "sys_vdi_vm_audit_logs"
}

// VMAccountManagementPolicy 账号管理策略
type VMAccountManagementPolicy struct {
    Base

    PolicyName    string `gorm:"type:varchar(200);not null;uniqueIndex" json:"policy_name"`
    
    // 密码策略
    MinPasswordLength      int     `gorm:"type:int;default:8" json:"min_password_length"`
    MaxPasswordLength      int     `gorm:"type:int;default:32" json:"max_password_length"`
    RequireUppercase       bool    `gorm:"type:bool;default:true" json:"require_uppercase"`
    RequireLowercase       bool    `gorm:"type:bool;default:true" json:"require_lowercase"`
    RequireNumber          bool    `gorm:"type:bool;default:true" json:"require_number"`
    RequireSpecial         bool    `gorm:"type:bool;default:true" json:"require_special"`
    
    // 账号命名规则
    UsernamePattern        *string `gorm:"type:varchar(200)" json:"username_pattern"` // 正则表达式
    UsernamePrefix         *string `gorm:"type:varchar(50)" json:"username_prefix"`
    
    // 默认属性
    DefaultShell           *string `gorm:"type:varchar(200)" json:"default_shell"` // Linux
    DefaultHomeBase        *string `gorm:"type:varchar(500)" json:"default_home_base"` // /home
    
    // 应用范围
    ApplyToVMs             *string `gorm:"type:text" json:"apply_to_vms"` // JSON数组
    ApplyToAllVMs          bool    `gorm:"type:bool;default:false" json:"apply_to_all_vms"`
    
    IsEnabled              bool    `gorm:"type:bool;default:true" json:"is_enabled"`
}

func (VMAccountManagementPolicy) TableName() string {
    return "sys_vdi_account_management_policies"
}
```

**验证**:
- [ ] 所有模型编译通过
- [ ] 表名设置正确
- [ ] 索引配置合理

---

### Task 6.3: 创建数据库迁移脚本

**文件**: `internal/core/db/migrations/086_create_vdi_account_tables.sql`

**内容**:
```sql
-- ========== 扩展VM表，添加SSH/WinRM字段 ==========
-- 这些字段添加到现有的 sys_vdi_vm 表
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_enabled BOOLEAN DEFAULT false;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_port INT DEFAULT 22;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_username VARCHAR(100);
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_password VARCHAR(500);
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_auth_method VARCHAR(50) DEFAULT 'password';
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS ssh_private_key TEXT;

ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS winrm_enabled BOOLEAN DEFAULT false;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS winrm_port INT DEFAULT 5985;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS winrm_use_https BOOLEAN DEFAULT false;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS winrm_username VARCHAR(100);
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS winrm_password VARCHAR(500);

ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS last_connection_test TIMESTAMP;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS connection_test_status VARCHAR(50);
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS connection_test_error TEXT;

ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS allow_account_management BOOLEAN DEFAULT true;
ALTER TABLE sys_vdi_vm ADD COLUMN IF NOT EXISTS account_management_policy_id VARCHAR(100);

-- ========== VM账号表 ==========
CREATE TABLE IF NOT EXISTS sys_vdi_vm_accounts (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    vm_id VARCHAR(100) NOT NULL,
    account_id VARCHAR(100) NOT NULL,
    username VARCHAR(100) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    os_type VARCHAR(50) NOT NULL,
    is_admin BOOLEAN DEFAULT false,
    is_enabled BOOLEAN DEFAULT true,
    description VARCHAR(500),

    -- Linux特定字段
    uid INT,
    gid INT,
    home_dir VARCHAR(500),
    shell VARCHAR(200),

    -- 同步状态
    sync_status VARCHAR(50) DEFAULT 'pending',
    synced_at TIMESTAMP,
    last_sync_error TEXT,

    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

CREATE INDEX idx_vm_accounts_vm_id ON sys_vdi_vm_accounts(vm_id);
CREATE UNIQUE INDEX idx_vm_accounts_account_id ON sys_vdi_vm_accounts(account_id);

-- ========== 审计日志表 ==========
CREATE TABLE IF NOT EXISTS sys_vdi_vm_audit_logs (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    vm_id VARCHAR(100),
    account_id VARCHAR(100),
    operation VARCHAR(50) NOT NULL,
    operator VARCHAR(100) NOT NULL,
    operator_ip VARCHAR(50),
    details TEXT,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    command_used TEXT,
    execution_time INT,
    executed_at TIMESTAMP
);

CREATE INDEX idx_vm_audit_logs_vm_id ON sys_vdi_vm_audit_logs(vm_id);
CREATE INDEX idx_vm_audit_logs_account_id ON sys_vdi_vm_audit_logs(account_id);
CREATE INDEX idx_vm_audit_logs_operation ON sys_vdi_vm_audit_logs(operation);
CREATE INDEX idx_vm_audit_logs_operator ON sys_vdi_vm_audit_logs(operator);

-- ========== 账号管理策略表 ==========
CREATE TABLE IF NOT EXISTS sys_vdi_account_management_policies (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    policy_name VARCHAR(200) NOT NULL UNIQUE,

    -- 密码策略
    min_password_length INT DEFAULT 8,
    max_password_length INT DEFAULT 32,
    require_uppercase BOOLEAN DEFAULT true,
    require_lowercase BOOLEAN DEFAULT true,
    require_number BOOLEAN DEFAULT true,
    require_special BOOLEAN DEFAULT true,

    -- 命名规则
    username_pattern VARCHAR(200),
    username_prefix VARCHAR(50),

    -- 默认属性
    default_shell VARCHAR(200),
    default_home_base VARCHAR(500),

    -- 应用范围
    apply_to_vms TEXT,
    apply_to_all_vms BOOLEAN DEFAULT false,

    is_enabled BOOLEAN DEFAULT true
);
```

**验证**:
- [ ] 迁移脚本执行成功
- [ ] 所有字段添加成功
- [ ] 索引创建正确

---

## Wave 7: SSH/WinRM连接实现（22-07）

**目标**: 实现通过SSH/WinRM连接VM并执行账号管理命令

**任务数**: 4 | **文件数**: 4 | **预估时间**: 3-4小时

### Task 7.1: 实现SSH连接管理器

**文件**: `internal/services/vdi/ssh_manager.go`

**内容**:
```go
package vdi

import (
    "fmt"
    "net"
    "time"

    "golang.org/x/crypto/ssh"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/operations"
)

type SSHManager struct {
    timeout time.Duration
}

func NewSSHManager() *SSHManager {
    return &SSHManager{
        timeout: 30 * time.Second,
    }
}

// TestConnection 测试SSH连接
func (m *SSHManager) TestConnection(vm *models.VDIVirtualMachine) error {
    if !vm.SSHEnabled {
        return fmt.Errorf("SSH未启用")
    }

    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    // 执行简单命令验证连接
    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    output, err := session.CombinedOutput("uname -a")
    if err != nil {
        return fmt.Errorf("命令执行失败: %w", err)
    }

    // 记录连接成功
    now := time.Now()
    vm.LastConnectionTest = &now
    vm.ConnectionTestStatus = "success"
    vm.ConnectionTestError = nil

    return nil
}

// CreateAccount 在Linux VM上创建账号
func (m *SSHManager) CreateAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount, password string) error {
    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    // 构建命令序列
    commands := []string{
        // 创建用户
        fmt.Sprintf("sudo useradd -m %s", account.Username),
        // 设置密码
        fmt.Sprintf("echo '%s:%s' | sudo chpasswd", account.Username, password),
    }

    // 如果是管理员，添加到sudo组
    if account.IsAdmin {
        commands = append(commands, fmt.Sprintf("sudo usermod -aG sudo %s", account.Username))
    }

    // 执行命令
    for _, cmd := range commands {
        session, err := client.NewSession()
        if err != nil {
            return fmt.Errorf("创建session失败: %w", err)
        }

        output, err := session.CombinedOutput(cmd)
        session.Close()

        if err != nil {
            return fmt.Errorf("命令执行失败 [%s]: %s, 错误: %w", cmd, string(output), err)
        }
    }

    // 更新同步状态
    now := time.Now()
    account.SyncStatus = "synced"
    account.SyncedAt = &now
    account.LastSyncError = nil

    return nil
}

// DeleteAccount 在Linux VM上删除账号
func (m *SSHManager) DeleteAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    cmd := fmt.Sprintf("sudo userdel -r %s", account.Username)

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return fmt.Errorf("删除账号失败: %s, 错误: %w", string(output), err)
    }

    return nil
}

// ResetPassword 重置账号密码
func (m *SSHManager) ResetPassword(vm *models.VDIVirtualMachine, account *models.VDIVMAccount, newPassword string) error {
    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    cmd := fmt.Sprintf("echo '%s:%s' | sudo chpasswd", account.Username, newPassword)

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return fmt.Errorf("密码重置失败: %s, 错误: %w", string(output), err)
    }

    return nil
}

// EnableAccount 启用账号
func (m *SSHManager) EnableAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    cmd := fmt.Sprintf("sudo usermod -U %s", account.Username)

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    if err := session.Run(cmd); err != nil {
        return fmt.Errorf("启用账号失败: %w", err)
    }

    account.IsEnabled = true
    return nil
}

// DisableAccount 禁用账号
func (m *SSHManager) DisableAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.connect(vm)
    if err != nil {
        return err
    }
    defer client.Close()

    cmd := fmt.Sprintf("sudo usermod -L %s", account.Username)

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    if err := session.Run(cmd); err != nil {
        return fmt.Errorf("禁用账号失败: %w", err)
    }

    account.IsEnabled = false
    return nil
}

// connect 建立SSH连接
func (m *SSHManager) connect(vm *models.VDIVirtualMachine) (*ssh.Client, error) {
    // 解密密码
    password, err := operations.DecryptVDIPassword(vm.SSHPasswordEncrypted)
    if err != nil {
        return nil, fmt.Errorf("密码解密失败: %w", err)
    }

    // 配置SSH客户端
    config := &ssh.ClientConfig{
        User: vm.SSHUsername,
        Auth: []ssh.AuthMethod{
            ssh.Password(password),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证host key
        Timeout: m.timeout,
    }

    // 建立连接
    address := fmt.Sprintf("%s:%d", vm.IPAddress, vm.SSHPort)
    client, err := ssh.Dial("tcp", address, config)
    if err != nil {
        return nil, fmt.Errorf("SSH连接失败 [%s]: %w", address, err)
    }

    return client, nil
}
```

**验证**:
- [ ] SSH连接建立成功
- [ ] 账号创建命令正确
- [ ] 错误处理完整

---

### Task 7.2: 实现WinRM连接管理器

**文件**: `internal/services/vdi/winrm_manager.go`

**内容**:
```go
package vdi

import (
    "fmt"
    "crypto/tls"
    "net/http"

    "github.com/masterzen/winrm"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/operations"
)

type WinRMManager struct {
    timeout time.Duration
}

func NewWinRMManager() *WinRMManager {
    return &WinRMManager{
        timeout: 30 * time.Second,
    }
}

// TestConnection 测试WinRM连接
func (m *WinRMManager) TestConnection(vm *models.VDIVirtualMachine) error {
    if !vm.WinRMEnabled {
        return fmt.Errorf("WinRM未启用")
    }

    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    // 执行简单命令验证连接
    _, err = client.RunWithString("whoami", "")

    if err != nil {
        return fmt.Errorf("WinRM连接测试失败: %w", err)
    }

    // 记录连接成功
    now := time.Now()
    vm.LastConnectionTest = &now
    vm.ConnectionTestStatus = "success"
    vm.ConnectionTestError = nil

    return nil
}

// CreateAccount 在Windows VM上创建账号
func (m *WinRMManager) CreateAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount, password string) error {
    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    // 构建PowerShell脚本
    script := fmt.Sprintf(`
        $password = ConvertTo-SecureString '%s' -AsPlainText -Force
        New-LocalUser -Name '%s' -Password $password -Description '%s'
    `, password, account.Username, safeString(account.Description))

    // 如果是管理员，添加到管理员组
    if account.IsAdmin {
        script += fmt.Sprintf("\nAdd-LocalGroupMember -Group 'Administrators' -Member '%s'", account.Username)
    }

    // 执行脚本
    _, err = client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("创建账号失败: %w", err)
    }

    // 更新同步状态
    now := time.Now()
    account.SyncStatus = "synced"
    account.SyncedAt = &now
    account.LastSyncError = nil

    return nil
}

// DeleteAccount 在Windows VM上删除账号
func (m *WinRMManager) DeleteAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    script := fmt.Sprintf("Remove-LocalUser -Name '%s' -Force", account.Username)

    _, err = client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("删除账号失败: %w", err)
    }

    return nil
}

// ResetPassword 重置账号密码
func (m *WinRMManager) ResetPassword(vm *models.VDIVirtualMachine, account *models.VDIVMAccount, newPassword string) error {
    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    script := fmt.Sprintf(`
        $password = ConvertTo-SecureString '%s' -AsPlainText -Force
        $user = Get-LocalUser -Name '%s'
        $user | Set-LocalUser -Password $password
    `, newPassword, account.Username)

    _, err = client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("密码重置失败: %w", err)
    }

    return nil
}

// EnableAccount 启用账号
func (m *WinRMManager) EnableAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    script := fmt.Sprintf("Enable-LocalUser -Name '%s'", account.Username)

    _, err = client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("启用账号失败: %w", err)
    }

    account.IsEnabled = true
    return nil
}

// DisableAccount 禁用账号
func (m *WinRMManager) DisableAccount(vm *models.VDIVirtualMachine, account *models.VDIVMAccount) error {
    client, err := m.createClient(vm)
    if err != nil {
        return err
    }

    script := fmt.Sprintf("Disable-LocalUser -Name '%s'", account.Username)

    _, err = client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("禁用账号失败: %w", err)
    }

    account.IsEnabled = false
    return nil
}

// createClient 创建WinRM客户端
func (m *WinRMManager) createClient(vm *models.VDIVirtualMachine) (*winrm.Client, error) {
    // 解密密码
    password, err := operations.DecryptVDIPassword(vm.WinRMPasswordEncrypted)
    if err != nil {
        return nil, fmt.Errorf("密码解密失败: %w", err)
    }

    // 构建端点
    protocol := "http"
    if vm.WinRMUseHTTPS {
        protocol = "https"
    }
    endpoint := fmt.Sprintf("%s://%s:%d/wsman", protocol, vm.IPAddress, vm.WinRMPort)

    // 创建客户端
    params := winrm.DefaultParameters()
    params.TransportDecorator = func() winrm.Transporter {
        return &winrm.ClientAuthRequest{
            Username: vm.WinRMUsername,
            Password: password,
            TLS: &tls.Config{
                InsecureSkipVerify: true, // 生产环境应该验证证书
            },
        }
    }

    client, err := winrm.NewClientWithParameters(endpoint, params)
    if err != nil {
        return nil, fmt.Errorf("创建WinRM客户端失败: %w", err)
    }

    return client, nil
}

func safeString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

**验证**:
- [ ] WinRM连接建立成功
- [ ] PowerShell脚本正确
- [ ] 错误处理完整

---

### Task 7.3: 实现统一的VM账号管理服务

**文件**: `internal/services/vdi/vm_account_service.go`

**内容**:
```go
package vdi

import (
    "context"
    "fmt"

    "github.com/xingran-next/xingran-go-backend/internal/models"
    "gorm.io/gorm"
)

type VMAccountService interface {
    // 连接测试
    TestConnection(ctx context.Context, vmID string) error
    
    // 账号CRUD
    CreateAccount(ctx context.Context, req *CreateAccountRequest) (*models.VDIVMAccount, error)
    GetAccount(ctx context.Context, accountID string) (*models.VDIVMAccount, error)
    ListAccounts(ctx context.Context, vmID string, req *ListAccountsRequest) (*PageResult, error)
    DeleteAccount(ctx context.Context, accountID string) error
    
    // 账号操作
    ResetPassword(ctx context.Context, accountID, newPassword string) error
    EnableAccount(ctx context.Context, accountID string) error
    DisableAccount(ctx context.Context, accountID string) error
    SyncAccount(ctx context.Context, accountID string) error
    
    // 批量操作
    BatchCreateAccounts(ctx context.Context, req []CreateAccountRequest) (*BatchResult, error)
    BatchDeleteAccounts(ctx context.Context, accountIDs []string) (*BatchResult, error)
}

type vmAccountService struct {
    db          *gorm.DB
    sshManager  *SSHManager
    winrmManager *WinRMManager
}

func NewVMAccountService(db *gorm.DB) VMAccountService {
    return &vmAccountService{
        db:          db,
        sshManager:  NewSSHManager(),
        winrmManager: NewWinRMManager(),
    }
}

// CreateAccount 创建账号（自动选择SSH或WinRM）
func (s *vmAccountService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*models.VDIVMAccount, error) {
    // 1. 验证VM存在
    var vm models.VDIVirtualMachine
    if err := s.db.WithContext(ctx).Where("vm_id = ?", req.VMID).First(&vm).Error; err != nil {
        return nil, fmt.Errorf("VM不存在: %w", err)
    }
    
    // 2. 验证密码策略
    if err := s.validatePassword(req.Password, &vm); err != nil {
        return nil, err
    }
    
    // 3. 创建账号记录
    account := &models.VDIVMAccount{
        VMID:        req.VMID,
        AccountID:   fmt.Sprintf("%s:%s", req.VMID, req.Username),
        Username:    req.Username,
        AccountType: req.AccountType,
        OSType:      vm.OSType,
        IsAdmin:     req.IsAdmin,
        IsEnabled:   true,
        Description: &req.Description,
        SyncStatus:  "pending",
        CreatedBy:   req.Operator,
    }
    
    // 4. 根据OS类型执行不同的命令
    var err error
    if vm.OSType == "linux" {
        if !vm.SSHEnabled {
            return nil, fmt.Errorf("SSH未启用，无法管理Linux VM账号")
        }
        err = s.sshManager.CreateAccount(&vm, account, req.Password)
    } else if vm.OSType == "windows" {
        if !vm.WinRMEnabled {
            return nil, fmt.Errorf("WinRM未启用，无法管理Windows VM账号")
        }
        err = s.winrmManager.CreateAccount(&vm, account, req.Password)
    } else {
        return nil, fmt.Errorf("不支持的操作系统类型: %s", vm.OSType)
    }
    
    if err != nil {
        account.SyncStatus = "failed"
        account.LastSyncError = &err.Error()
        s.db.Save(account)
        return nil, fmt.Errorf("创建账号失败: %w", err)
    }
    
    // 5. 保存到数据库
    if err := s.db.WithContext(ctx).Create(account).Error; err != nil {
        return nil, fmt.Errorf("保存账号记录失败: %w", err)
    }
    
    // 6. 记录审计日志
    s.logAudit(ctx, &vm, account, "create", req.Operator, "success", "")
    
    return account, nil
}

// ResetPassword 重置密码
func (s *vmAccountService) ResetPassword(ctx context.Context, accountID, newPassword string, operator string) error {
    // 1. 获取账号
    var account models.VDIVMAccount
    if err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&account).Error; err != nil {
        return fmt.Errorf("账号不存在: %w", err)
    }
    
    // 2. 获取VM
    var vm models.VDIVirtualMachine
    if err := s.db.WithContext(ctx).Where("vm_id = ?", account.VMID).First(&vm).Error; err != nil {
        return fmt.Errorf("VM不存在: %w", err)
    }
    
    // 3. 验证密码策略
    if err := s.validatePassword(newPassword, &vm); err != nil {
        return err
    }
    
    // 4. 执行密码重置
    var err error
    commandUsed := ""
    if vm.OSType == "linux" {
        err = s.sshManager.ResetPassword(&vm, &account, newPassword)
        commandUsed = fmt.Sprintf("echo '%s:%s' | sudo chpasswd", account.Username, "***")
    } else {
        err = s.winrmManager.ResetPassword(&vm, &account, newPassword)
        commandUsed = fmt.Sprintf("Set-LocalUser -Name '%s' -Password ***", account.Username)
    }
    
    if err != nil {
        s.logAudit(ctx, &vm, &account, "reset_password", operator, "failed", err.Error())
        return fmt.Errorf("密码重置失败: %w", err)
    }
    
    // 5. 记录审计日志
    s.logAudit(ctx, &vm, &account, "reset_password", operator, "success", commandUsed)
    
    return nil
}

// 其他方法实现...
```

**验证**:
- [ ] 服务层逻辑正确
- [ ] SSH/WinRM选择逻辑正确
- [ ] 审计日志记录完整

---

### Task 7.4: 密码验证工具

**文件**: `internal/services/vdi/password_validator.go`

**内容**:
```go
package vdi

import (
    "errors"
    "regexp"
    "unicode"
    
    "github.com/xingran-next/xingran-go-backend/internal/models"
)

func (s *vmAccountService) validatePassword(password string, vm *models.VDIVirtualMachine) error {
    // 获取密码策略
    var policy models.VMAccountManagementPolicy
    if vm.AccountManagementPolicyID != nil {
        if err := s.db.Where("id = ?", *vm.AccountManagementPolicyID).First(&policy).Error; err != nil {
            // 如果策略不存在，使用默认策略
            policy = s.getDefaultPolicy()
        }
    } else {
        policy = s.getDefaultPolicy()
    }
    
    // 检查长度
    if len(password) < policy.MinPasswordLength {
        return fmt.Errorf("密码长度不能少于%d位", policy.MinPasswordLength)
    }
    if len(password) > policy.MaxPasswordLength {
        return fmt.Errorf("密码长度不能超过%d位", policy.MaxPasswordLength)
    }
    
    // 检查大写字母
    if policy.RequireUppercase {
        if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
            return errors.New("密码必须包含大写字母")
        }
    }
    
    // 检查小写字母
    if policy.RequireLowercase {
        if !regexp.MustCompile(`[a-z]`).MatchString(password) {
            return errors.New("密码必须包含小写字母")
        }
    }
    
    // 检查数字
    if policy.RequireNumber {
        if !regexp.MustCompile(`[0-9]`).MatchString(password) {
            return errors.New("密码必须包含数字")
        }
    }
    
    // 检查特殊字符
    if policy.RequireSpecial {
        if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
            return errors.New("密码必须包含特殊字符")
        }
    }
    
    return nil
}

func (s *vmAccountService) getDefaultPolicy() models.VMAccountManagementPolicy {
    return models.VMAccountManagementPolicy{
        MinPasswordLength: 8,
        MaxPasswordLength: 32,
        RequireUppercase: true,
        RequireLowercase: true,
        RequireNumber:    true,
        RequireSpecial:   false,
    }
}
```

**验证**:
- [ ] 密码验证规则正确
- [ ] 策略应用正确

---

## Wave 8: 后端API和前端UI（22-08）

**目标**: 实现后端API和前端界面

**任务数**: 5 | **文件数**: 6 | **预估时间**: 4-5小时

### Task 8.1: 创建VMAccountHandler

**文件**: `internal/api/v1/vdi/vm_account_handler.go`

**内容**（参考之前的Agent方案，但连接测试改为SSH/WinRM）:
```go
package vdi

import (
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

type VMAccountHandler struct {
    accountService VMAccountService
}

func NewVMAccountHandler(accountService VMAccountService) *VMAccountHandler {
    return &VMAccountHandler{accountService: accountService}
}

// TestConnection 测试VM连接
func (h *VMAccountHandler) TestConnection(c *gin.Context) {
    vmID := c.Param("vmId")
    
    err := h.accountService.TestConnection(c.Request.Context(), vmID)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }
    
    response.Success(c, gin.H{"message": "连接测试成功"})
}

// CreateAccount 创建VM账号
func (h *VMAccountHandler) CreateAccount(c *gin.Context) {
    var req CreateAccountRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, 400, "请求参数错误")
        return
    }
    
    // 从JWT获取操作人
    req.Operator = c.GetString("username")
    
    account, err := h.accountService.CreateAccount(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, 500, err.Error())
        return
    }
    
    response.Success(c, account)
}

// 其他Handler方法...
```

---

### Task 8.2: 创建VMAccountRouter

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
    accountService := vdi.NewVMAccountService(core.GetDB())
    accountHandler := vdi.NewVMAccountHandler(accountService)
    
    // 需要认证和权限
    accountGroup := r.Group("/accounts")
    accountGroup.Use(middleware.Auth())
    accountGroup.Use(middleware.Permission("vm:admin")) // VM管理权限
    {
        // 连接测试
        accountGroup.POST("/test-connection/:vmId", accountHandler.TestConnection)
        
        // 账号CRUD
        accountGroup.POST("/list", accountHandler.ListAccounts)
        accountGroup.POST("", accountHandler.CreateAccount)
        accountGroup.POST("/:accountId", accountHandler.GetAccount)
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
}
```

---

### Task 8.3: 前端类型定义

**文件**: `src/types/vdiAccount.ts`

**内容**（与之前Agent方案相同）

---

### Task 8.4: 前端API客户端

**文件**: `src/lib/vdiAccountApi.ts`

**内容**（与之前Agent方案相同）

---

### Task 8.5: 虚拟机详情页扩展

**文件**: `src/pages/vdi/VirtualMachineList/index.tsx`（修改）

**修改**: 在VM详情对话框中添加"账号管理"标签页

**新增功能**:
1. 显示VM的SSH/WinRM配置状态
2. 账号列表表格
3. 创建账号按钮
4. 连接测试按钮
5. 批量操作支持

---

## 总结

### SSH/WinRM方案 vs Agent方案对比

| 特性 | SSH/WinRM方案 | Agent方案 |
|------|---------------|----------|
| **需要安装Agent** | ❌ 不需要 | ✅ 需要 |
| **安全性** | ⭐⭐⭐⭐ 更安全（每次认证） | ⭐⭐⭐ Agent需保护 |
| **复杂度** | ⭐⭐ 简单 | ⭐⭐⭐⭐ 较复杂 |
| **维护成本** | ⭐ 低 | ⭐⭐⭐ 高 |
| **网络要求** | 需开放SSH/WinRM端口 | Agent主动出网 |
| **跨平台** | ✅ 标准协议 | ✅ 需分别开发 |

### 最终工作量

**Phase 22 总计**: 8个Wave，31个任务

| Wave | 计划 | 任务 | 文件 | 时间 |
|------|------|------|------|------|
| 1-5 | 基础VDI集成 | 22 | 38 | 16-21h |
| 6 | 数据模型扩展 | 3 | 3 | 1-2h |
| 7 | SSH/WinRM实现 | 4 | 4 | 3-4h |
| 8 | API和前端 | 5 | 6 | 4-5h |
| **总计** | - | **34** | **51** | **24-32h (3-4个工作日)**

---

## 核心优势

✅ **无需Agent**: VM纯净，无额外程序
✅ **更安全**: 每次操作都认证，无持久性权限
✅ **更简单**: 架构清晰，易于维护
✅ **标准化**: 使用业界标准协议
✅ **成本更低**: 开发、测试、维护成本都更低

这个方案比Agent方案更适合您的需求！
