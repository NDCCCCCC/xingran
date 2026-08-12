# VM账号管理 - 安全方案对比

## 方案对比

### 方案A: 最小权限Agent（推荐）✅

**架构**:
```
┌─────────────────┐
│  XingRan-Next     │
│  (管理平台)      │
└────────┬────────┘
         │ HTTP(S)
         ▼
┌─────────────────┐
│  账号管理服务    │ ◄────── 只负责账号管理逻辑
└────────┬────────┘
         │ gRPC/HTTP
         ▼
┌─────────────────┐
│  VM Agent       │
│  (最小权限)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  VM操作系统     │
│  账号数据库      │
```

**Agent权限**:
- ✅ 只能创建/修改/删除**指定范围的账号**
- ✅ 不能修改系统配置
- ✅ 不能访问敏感文件
- ✅ 操作记录审计日志

**Windows实现**:
```powershell
# 使用"受限的管理员"账户运行Agent
# 该账户只有"用户管理"权限，没有其他管理员权限

# 创建专用服务账户
New-LocalUser -Name "XingRanAgentUser" -Password $password
# 仅授予用户管理权限
# 不能修改注册表、不能安装软件、不能访问其他敏感资源
```

**Linux实现**:
```bash
# 创建专用服务账户
useradd -m xingran-agent
# 授予特定sudo权限（只能管理用户）
echo "xingran-agent ALL=(root) NOPASSWD: /usr/sbin/useradd" >> /etc/sudoers.d/xingran-agent
echo "xingran-agent ALL=(root) NOPASSWD: /usr/bin/passwd" >> /etc/sudoers.d/xingran-agent

# Agent以该用户身份运行，不是root
```

**优点**:
- ✅ 降低安全风险
- ✅ Agent被攻破影响有限
- ✅ 符合最小权限原则

**缺点**:
- ⚠️ 配置稍复杂
- ⚠️ 需要为Windows/Linux分别配置

---

### 方案B: SSH/WinRM直连（无需Agent）✅✅

**架构**:
```
┌─────────────────┐
│  XingRan-Next     │
└────────┬────────┘
         │ SSH/WinRM
         │ (远程连接)
         ▼
┌─────────────────┐
│  VM操作系统     │
│  (无Agent)      │
```

**工作原理**:
- XingRan-Next直接通过**SSH连接**Linux VM
- XingRan-Next直接通过**WinRM连接**Windows VM
- 执行远程命令管理账号

**Linux (SSH)**:
```go
// 使用SSH连接到Linux VM
client, _ := ssh.Dial("tcp", "vm-ip:22", &ssh.ClientConfig{
    User: "root",  // 或其他有权限的用户
    Auth: []ssh.AuthMethod{ssh.Password(password)},
})

// 执行命令创建用户
session, _ := client.NewSession()
session.Run("useradd -m testuser")
session.Run("echo 'testuser:password' | chpasswd")
```

**Windows (WinRM)**:
```go
// 使用WinRM连接到Windows VM
endpoint := "http://vm-ip:5985/wsman"
client := winrm.NewClient(&winrm.Client{
    Endpoint: endpoint,
    Username: "Administrator",
    Password: password,
})

// 执行PowerShell命令
client.Run("New-LocalUser -Name testuser -Password password")
```

**优点**:
- ✅ 无需安装Agent
- ✅ VM纯净，无额外程序
- ✅ 利用系统原生协议

**缺点**:
- ⚠️ 需要VM开放SSH(22)或WinRM(5985/5986)端口
- ⚠️ 需要存储VM的管理员密码
- ⚠️ 网络要求更高（需要稳定的SSH连接）

---

### 方案C: 云厂商CLI工具

**适用场景**: 如果VDI底层是云平台（如阿里云、腾讯云）

**原理**:
- 使用云厂商提供的CLI工具或API
- 例如：`aliyun ecs InvokeCommand`

**优点**:
- ✅ 原生支持
- ✅ 无需Agent

**缺点**:
- ❌ 依赖云厂商特定功能
- ❌ 深信服VDI可能不是基于云平台

---

## 推荐方案总结

| 方案 | 安全性 | 复杂度 | 适用场景 | 推荐度 |
|------|--------|--------|----------|--------|
| **最小权限Agent** | ⭐⭐⭐⭐ | ⭐⭐⭐ | 高安全要求生产环境 | ⭐⭐⭐⭐ |
| **SSH/WinRM直连** | ⭐⭐⭐ | ⭐⭐ | 内网环境，VM数量少 | ⭐⭐⭐⭐⭐ |
| **全权Agent** | ⭐ | ⭐⭐ | 测试环境，快速验证 | ⭐⭐ |
| **云厂商CLI** | ⭐⭐⭐⭐⭐ | ⭐ | 云平台VDI | ⭐ |

---

## SSH/WinRM方案详细设计（推荐）

### 为什么推荐这个方案？

1. **无需Agent**: VM不需要安装任何额外程序
2. **更安全**: 每次操作都需要认证，无持久性权限
3. **更简单**: 架构简单，易于维护
4. **标准化**: 使用业界标准协议（SSH、WinRM）

### 数据模型（简化版）

**只需要存储VM的连接信息**:
```go
type VDIVirtualMachine struct {
    // ... 现有字段 ...

    // SSH连接信息（Linux）
    SSHPort     int    `gorm:"type:int;default:22" json:"ssh_port"`
    SSHUsername string `gorm:"type:varchar(100)" json:"ssh_username"`
    SSHPasswordEncrypted string `gorm:"type:varchar(500)" json:"-"` // SM4加密

    // WinRM连接信息（Windows）
    WinRMPort     int    `gorm:"type:int;default:5985" json:"winrm_port"`
    WinRMUsername string `gorm:"type:varchar(100)" json:"winrm_username"`
    WinRMPasswordEncrypted string `gorm:"type:varchar(500)" json:"-"` // SM4加密
}
```

### 账号管理服务实现

**Linux账号管理（通过SSH）**:
```go
type LinuxAccountManager struct{}

func (m *LinuxAccountManager) CreateAccount(vm *VDIVirtualMachine, username, password string) error {
    // 建立SSH连接
    config := &ssh.ClientConfig{
        User: vm.SSHUsername,
        Auth: []ssh.AuthMethod{
            ssh.Password(decryptPassword(vm.SSHPasswordEncrypted)),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证host key
    }

    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", vm.IPAddress, vm.SSHPort), config)
    if err != nil {
        return fmt.Errorf("SSH连接失败: %w", err)
    }
    defer client.Close()

    // 创建用户
    session, _ := client.NewSession()
    defer session.Close()

    // 执行命令
    commands := []string{
        fmt.Sprintf("useradd -m %s", username),
        fmt.Sprintf("echo '%s:%s' | chpasswd", username, password),
    }

    for _, cmd := range commands {
        if err := session.Run(cmd); err != nil {
            return fmt.Errorf("命令执行失败: %s, 错误: %w", cmd, err)
        }
    }

    return nil
}
```

**Windows账号管理（通过WinRM）**:
```go
type WindowsAccountManager struct{}

func (m *WindowsAccountManager) CreateAccount(vm *VDIVirtualMachine, username, password string) error {
    // 建立WinRM连接
    endpoint := fmt.Sprintf("http://%s:%d/wsman", vm.IPAddress, vm.WinRMPort)
    client := winrm.NewClient(&winrm.Client{
        Endpoint: endpoint,
        Username: vm.WinRMUsername,
        Password: decryptPassword(vm.WinRMPasswordEncrypted),
        Transport: winrm.DefaultTransport,
    })

    // 执行PowerShell命令
    script := fmt.Sprintf(`
        $password = ConvertTo-SecureString '%s' -AsPlainText -Force
        New-LocalUser -Name '%s' -Password $password
    `, password, username)

    _, err := client.RunWithString(script, "")
    if err != nil {
        return fmt.Errorf("WinRM命令执行失败: %w", err)
    }

    return nil
}
```

### 前端UI（与Agent方案相同）

UI层面不需要关心后端是通过Agent还是SSH/WinRM实现的，保持一致的用户体验。

---

## 安全增强措施

### 1. 密钥管理

**方案A**: 使用SSH密钥认证（更安全）
```go
// 不使用密码，使用SSH密钥
config := &ssh.ClientConfig{
    User: vm.SSHUsername,
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(signer), // 使用私钥认证
    },
}
```

**方案B**: 密码定期轮换
- 每月自动更换VM管理员密码
- 密码存储在XingRan数据库中（SM4加密）

### 2. 网络隔离

```
┌─────────────────┐
│  XingRan-Next     │
│  (DMZ区)        │
└────────┬────────┘
         │ 防火墙规则
         │ 仅允许 XingRan → VM SSH/WinRM
         ▼
┌─────────────────┐
│  VM网络         │
│  (内网)         │
└─────────────────┘
```

### 3. 审计日志

记录所有账号操作：
```go
type VDIVMAuditLog struct {
    VMID          string
    AccountID     string
    Operation     string  // create/delete/reset_password
    Operator      string  // XingRan用户
    OperatorIP    string  // 操作来源IP
    Command       string  // 实际执行的命令
    Status        string  // success/failed
    ExecutedAt    time.Time
}
```

### 4. 权限控制

只有特定角色的用户才能执行敏感操作：
```go
// 检查用户是否有"VM管理员"权限
if !hasPermission(c, "vm:admin") {
    return errors.New("权限不足")
}
```

---

## 最终推荐

**推荐方案**: **SSH/WinRM直连**

**理由**:
1. ✅ 无需Agent，VM更纯净
2. ✅ 安全性更好（每次操作都认证）
3. ✅ 架构更简单，维护成本低
4. ✅ 使用标准协议，兼容性好

**实施步骤**:
1. Phase 22 Wave 1-5: 完成基础VDI集成
2. Phase 22 Wave 6: 扩展VM模型（添加SSH/WinRM字段）
3. Phase 22 Wave 7: 实现SSH/WinRM账号管理服务
4. Phase 22 Wave 8: 实现后端API
5. Phase 22 Wave 9: 实现前端UI

**总工作量**: 基础VDI（16-21小时）+ SSH/WinRM账号管理（8-12小时）= 24-33小时（3-4个工作日）

---

需要我为您生成SSH/WinRM方案的详细实施计划吗？
