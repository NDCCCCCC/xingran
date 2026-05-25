# Phase 22: 深信服桌面云集成 - Context

**Gathered:** 2026-05-25
**Status:** Ready for planning

## Phase Boundary

实现深信服VDI（虚拟桌面基础设施）集成，包括VM账号管理功能。**核心挑战：**VM管理员密码需要定期修改（安全要求），这使得SSH/WinRM直连方案不可行（存储的密码会失效）。解决方案是使用VM Agent架构，Agent通过JWT令牌认证，独立工作，不依赖VM管理员密码。

## Implementation Decisions

### D-01: VM连接方式架构决策
**选择：VM Agent架构（而非SSH/WinRM直连）**

**理由：**
- SSH/WinRM方案需要在RuoYi-Next中存储VM管理员密码（用于连接）
- 安全要求定期修改VM管理员密码 → 存储的密码失效 → 无法连接VM
- Agent方案使用JWT令牌认证，安装后独立工作，完全解耦对VM管理员密码的依赖
- Agent通过HTTPS/gRPC主动连接RuoYi-Next，不依赖VDI网络配置

### D-02: Agent功能范围
**选择：账号CRUD操作 + 密码重置功能 + 系统状态监控**

**功能详情：**
- **账号CRUD：** 创建、删除、启用/禁用本地用户账号（Linux: useradd/userdel/usermod; Windows: New-LocalUser/Remove-LocalUser/Enable-LocalUser/Disable-LocalUser）
- **密码重置：** 重置用户密码，记录操作到审计日志
- **系统状态监控：** 定期上报VM系统状态（CPU、内存、磁盘使用率、运行进程等）

**排除的功能：**
- ❌ 远程命令执行 - 风险过高，不在本期范围

### D-03: Agent权限设计（Windows）
**选择：受限管理员账号（Restricted Management）**

**实现方式：**
- 创建专用本地账号（如`RuoyiAgentUser`）
- 仅授予「用户管理」权限
- 使用PowerShell Just Enough Administration (JEA) 或 Restricted Management模式
- 该账号可以：创建/删除用户、重置密码
- 该账号不能：修改注册表、安装软件、访问管理员共享文件夹

### D-04: Agent权限设计（Linux）
**选择：专用服务账号 + 受限sudo**

**实现方式：**
```bash
# 创建专用服务账号
useradd -m ruoyi-agent

# 配置sudoers.d（仅允许用户管理命令）
# /etc/sudoers.d/ruoyi-agent
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/useradd
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/userdel
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/usermod
ruoyi-agent ALL=(root) NOPASSWD: /usr/bin/passwd
```

**权限边界：**
- ✅ 可以执行：用户管理命令（useradd/userdel/passwd）
- ❌ 不能执行：系统配置修改、软件安装、访问其他用户文件

### D-05: Agent安全机制
**选择：JWT令牌认证 + TLS加密通信 + 操作审计日志 + 速率限制**

**安全机制详情：**
1. **JWT令牌认证：**
   - Agent向RuoYi-Next注册时，后端生成唯一JWT令牌
   - 每次API调用必须携带有效令牌
   - 令牌包含：Agent ID、VM ID、权限范围
   - 过期时间：24小时，自动刷新机制

2. **TLS加密通信：**
   - Agent与RuoYi-Next之间使用TLS 1.3加密
   - 证书由RuoYi-Next CA签发，每个Agent有唯一证书
   - 防止中间人攻击和流量嗅探

3. **操作审计日志：**
   - 所有操作记录：时间、操作人（RuoYi用户）、操作类型、目标账号、执行结果
   - 日志同时发送到RuoYi-Next和本地`/var/log/ruoyi-agent.log`
   - 支持合规审计和故障排查

4. **速率限制：**
   - RuoYi-Next对Agent请求进行速率限制：每分钟最多100次操作
   - 防止Agent被恶意利用进行DDoS攻击或暴力破解

### D-06: Agent部署方式
**选择：VDI镜像预装（推荐）**

**实施方式：**
- 在VDI镜像模板中预装Agent
- 所有新创建的VM自动包含Agent，开箱即用
- 自动化程度高，用户体验好

**权衡：**
- ✅ 优点：自动化、零配置、适合大规模部署
- ⚠️ 缺点：更新Agent需要重新制作镜像（可接受的权衡）

**备用方案：**
- 手动安装（仅用于测试环境或特殊情况）

### Claude's Discretion
**Agent通信协议：** HTTP REST vs gRPC
- 建议使用HTTP REST（更简单、易于调试、与现有架构一致）
- 如果需要高性能流式数据传输，可使用gRPC

**Agent心跳间隔：** 建议30秒（可配置）
- 平衡实时性和网络开销

**Agent版本管理：**
- 支持Agent版本查询和远程升级（可选）

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 架构文档
- `docs/项目概述和架构设计.md` — RuoYi-Next整体架构
- `docs/安全和认证设计（国密）.md` — 国密算法使用规范（SM2/SM3/SM4）

### Phase 22研究文档
- `.planning/phases/22-sangfor-vdi-integration/vm_agent_architecture.md` — VM Agent完整架构设计
- `.planning/phases/22-sangfor-vdi-integration/vm_account_security_comparison.md` — 安全方案对比分析
- `.planning/phases/22-sangfor-vdi-integration/22-RESEARCH.md` — VDI集成研究报告
- `.planning/phases/22-sangfor-vdi-integration/22-PATTERNS.md` — 代码模式和架构规范

### 深信服VDI API文档
- `docs/深信服桌面云开放平台（V1.2）.doc` — 官方API文档
- `docs/sangfor_vdi_utf8.txt` — 提取的API文本版本
- `.planning/phases/22-sangfor-vdi-integration/VDI_API_QUICK_REFERENCE.md` — API快速参考

### 已有计划文档（参考，非直接使用）
- `.planning/phases/22-sangfor-vdi-integration/22-01-PLAN.md` — Wave 1: 数据模型与配置基础
- `.planning/phases/22-sangfor-vdi-integration/22-02-PLAN.md` — Wave 2: VDI API客户端与认证
- `.planning/phases/22-sangfor-vdi-integration/22-03-PLAN.md` — Wave 3: VDI服务层实现
- `.planning/phases/22-sangfor-vdi-integration/22-04-PLAN.md` — Wave 4: VDI后端API层
- `.planning/phases/22-sangfor-vdi-integration/22-05-PLAN.md` — Wave 5: VDI前端UI实现
- `.planning/phases/22-sangfor-vdi-integration/22-06-PLAN-SSH-WINRM.md` — SSH/WinRM方案（已废弃，参考Agent方案）

## Existing Code Insights

### Reusable Assets
- **JWT认证系统** (`internal/core/security/jwt.go`) — 可复用用于Agent令牌生成和验证
- **密码加密** (`internal/operations/vdi_password.go`) — AES-128-GCM加密，可用于VDI服务器密码存储
- **Handler-Service模式** (`internal/api/v1/`, `internal/services/`) — 标准架构模式，VDI模块应遵循

### Established Patterns
- **服务接口设计** — 所有服务定义接口，私有实现，构造函数注入依赖
- **CacheProvider接口** (`internal/services/system/cache_provider.go`) — 可用于VDI数据缓存
- **响应格式** — 统一使用`response.Success()` / `response.Error()`包装

### Integration Points
- **主路由** (`internal/api/router.go`) — VDI路由组应注册在`/api/v1/vdi`下
- **数据库迁移** (`internal/core/db/migrations/`) — 需要创建VDI相关表的迁移脚本
- **配置系统** (`configs/config.yaml`, `internal/config/`) — VDI服务器配置应添加到config.yaml

## Specific Ideas

### Agent认证流程
1. Agent首次启动，向RuoYi-Next注册（提供VM ID、Agent版本、OS类型）
2. RuoYi-Next验证VM属于已配置的VDI服务器
3. 生成JWT令牌（包含Agent ID、VM ID、权限范围、24小时有效期）
4. Agent存储令牌，每次API调用携带在Authorization头
5. 令牌即将过期时（提前1小时），Agent自动刷新

### Agent安装脚本
**Windows (PowerShell):**
```powershell
# 下载Agent
Invoke-WebRequest -Uri "https://ruoyi-backend.example.com/api/v1/agent/download/windows" -OutFile "agent.zip"
Expand-Archive -Path "agent.zip" -DestinationPath "C:\Program Files\RuoyiAgent"

# 创建受限管理员账号
New-LocalUser -Name "RuoyiAgentUser" -Password $password -Description "RuoYi VM Agent"
# 授予用户管理权限（使用JEA或Restricted Management）

# 注册Windows服务
New-Service -Name "RuoyiVMAgent" -BinaryPathName "C:\Program Files\RuoyiAgent\agent.exe" -StartupType Automatic
Start-Service "RuoyiVMAgent"

# Agent自动注册到后端
```

**Linux (Bash):**
```bash
# 下载Agent
wget -O /tmp/agent.tar.gz "https://ruoyi-backend.example.com/api/v1/agent/download/linux"
mkdir -p /opt/ruoyi-agent
tar -xzf /tmp/agent.tar.gz -C /opt/ruoyi-agent

# 创建专用服务账号
useradd -m ruoyi-agent

# 配置sudoers.d
cat > /etc/sudoers.d/ruoyi-agent << EOF
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/useradd
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/userdel
ruoyi-agent ALL=(root) NOPASSWD: /usr/sbin/usermod
ruoyi-agent ALL=(root) NOPASSWD: /usr/bin/passwd
EOF

# 注册systemd服务
cat > /etc/systemd/system/ruoyi-vm-agent.service << EOF
[Unit]
Description=Ruoyi VM Agent
After=network.target

[Service]
Type=simple
User=ruoyi-agent
WorkingDirectory=/opt/ruoyi-agent
ExecStart=/opt/ruoyi-agent/agent --config=/opt/ruoyi-agent/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ruoyi-vm-agent
systemctl start ruoyi-vm-agent
```

### Agent数据模型
**新增表：**
- `sys_vdi_agents` — Agent注册信息（agent_id、vm_id、version、os_type、last_heartbeat、status）
- `sys_vdi_vm_accounts` — VM内部账号（vm_id、account_id、username、account_type、is_admin、is_enabled）
- `sys_vdi_vm_audit_logs` — 账号操作审计日志（vm_id、account_id、operation、operator、status、command_used）

## Deferred Ideas

无 — 讨论保持在Phase 22范围内，无超出范围的提议。

---

*Phase: 22-sangfor-vdi-integration*
*Context gathered: 2026-05-25*
