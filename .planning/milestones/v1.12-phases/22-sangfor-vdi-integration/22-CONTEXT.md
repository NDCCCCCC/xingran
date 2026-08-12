# Phase 22: 深信服桌面云集成 - Context

**Gathered:** 2026-05-25
**Status:** Ready for planning

## Phase Boundary

实现深信服VDI（虚拟桌面基础设施）集成，包括VM账号管理功能。**核心挑战：**VM管理员密码需要定期修改（安全要求），这使得SSH/WinRM直连方案不可行（存储的密码会失效）。解决方案是使用VM Agent架构，Agent通过JWT令牌认证，独立工作，不依赖VM管理员密码。

## Implementation Decisions

### D-01: VM连接方式架构决策
**选择：VM Agent架构（而非SSH/WinRM直连）**

**理由：**
- SSH/WinRM方案需要在XingRan-Next中存储VM管理员密码（用于连接）
- 安全要求定期修改VM管理员密码 → 存储的密码失效 → 无法连接VM
- Agent方案使用JWT令牌认证，安装后独立工作，完全解耦对VM管理员密码的依赖
- Agent通过HTTPS/gRPC主动连接XingRan-Next，不依赖VDI网络配置

### D-02: Agent功能范围
**选择：账号CRUD操作 + 密码重置功能 + 系统状态监控 + Web Console（网页终端）**

**功能详情：**
- **账号CRUD：** 创建、删除、启用/禁用本地用户账号（Linux: useradd/userdel/usermod; Windows: New-LocalUser/Remove-LocalUser/Enable-LocalUser/Disable-LocalUser）
- **密码重置：** 重置用户密码，记录操作到审计日志
- **系统状态监控：** 定期上报VM系统状态（CPU、内存、磁盘使用率、运行进程等）
- **Web Console（网页终端）：** 提供WebSocket终端连接，用户可在浏览器中直接打开VM终端（类似堡垒机功能）

**Web Console实现方式：**
- Agent启动WebSocket服务器，监听终端连接请求
- 使用xterm.js（前端）+ pty（后端伪终端）实现网页终端
- 支持Linux（SSH伪终端）和Windows（PowerShell伪终端）
- Agent通过本地pty/spawn创建终端会话，通过WebSocket转发I/O
- **安全机制：**
  - WebSocket连接需要JWT令牌认证（与Agent其他API相同）
  - 记录所有终端会话到审计日志（连接时间、用户、操作内容哈希）
  - 会话超时自动断开（默认30分钟，可配置）
  - 支持管理员强制断开会话

**排除的功能：**
- ❌ 远程命令执行 - 风险过高，不在本期范围
- ❌ 文件传输 - Web Console仅用于交互式终端，不支持文件上传/下载

### D-03: Agent权限设计（Windows）
**选择：受限管理员账号（Restricted Management）**

**实现方式：**
- 创建专用本地账号（如`XingRanAgentUser`）
- 授予「用户管理」权限
- **Web Console权限：** 允许创建PowerShell会话（不需要管理员权限，普通用户即可创建交互式Shell）
- 使用PowerShell Just Enough Administration (JEA) 或 Restricted Management模式

**该账号可以：**
- ✅ 创建/删除用户、重置密码
- ✅ 创建交互式PowerShell会话（用于Web Console）

**该账号不能：**
- ❌ 修改注册表
- ❌ 安装软件
- ❌ 访问管理员共享文件夹
- ❌ 执行系统级配置

### D-04: Agent权限设计（Linux）
**选择：专用服务账号 + 受限sudo**

**实现方式：**
```bash
# 创建专用服务账号
useradd -m xingran-agent

# 配置sudoers.d（用户管理命令）
# /etc/sudoers.d/xingran-agent
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/useradd
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/userdel
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/usermod
xingran-agent ALL=(root) NOPASSWD: /usr/bin/passwd

# Web Console伪终端权限（重要）
# xingran-agent可以创建伪终端，但不需要sudo
# 这是因为agent以xingran-agent身份运行，可以创建自己的终端会话
# 如果需要以其他用户身份登录（如root），才需要sudo
```

**权限边界：**
- ✅ 可以执行：用户管理命令（useradd/userdel/passwd）
- ✅ 可以创建：伪终端（pty）用于Web Console（普通用户权限即可）
- ✅ 可以运行：交互式Shell会话（bash/zsh）用于Web Console
- ❌ 不能执行：系统配置修改、软件安装、访问其他用户文件

**Web Console权限说明：**
- Agent以`xingran-agent`身份运行，可以创建自己的终端会话（不需要sudo）
- 终端会话受限于`xingran-agent`的权限（类似于SSH登录）
- 如果用户需要root权限，需要在终端内手动使用`sudo`命令
- Agent不自动提权，保持最小权限原则

### D-05: Agent安全机制
**选择：JWT令牌认证 + TLS加密通信 + 操作审计日志 + 速率限制**

**安全机制详情：**
1. **JWT令牌认证：**
   - Agent向XingRan-Next注册时，后端生成唯一JWT令牌
   - 每次API调用必须携带有效令牌
   - 令牌包含：Agent ID、VM ID、权限范围
   - 过期时间：24小时，自动刷新机制

2. **TLS加密通信：**
   - Agent与XingRan-Next之间使用TLS 1.3加密
   - 证书由XingRan-Next CA签发，每个Agent有唯一证书
   - 防止中间人攻击和流量嗅探

3. **操作审计日志：**
   - 所有操作记录：时间、操作人（XingRan用户）、操作类型、目标账号、执行结果
   - 日志同时发送到XingRan-Next和本地`/var/log/xingran-agent.log`
   - 支持合规审计和故障排查

4. **速率限制：**
   - XingRan-Next对Agent请求进行速率限制：每分钟最多100次操作
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

### D-07: 数据库表命名规范
**选择：统一使用 vdi 前缀（而非 sys_vdi）**

**规范：**
- 所有VDI相关表使用 `vdi_` 前缀（如 `vdi_servers`、`vdi_vms`、`vdi_agents`）
- 遵循项目现有的表命名约定，但不使用 `sys_` 前缀
- 与系统核心表（如 `sys_user`、`sys_role`）区分开

**示例表名：**
- `vdi_servers` — VDI服务器配置表
- `vdi_vms` — 虚拟机表
- `vdi_vm_accounts` — VM内部账号表
- `vdi_agents` — Agent注册信息表
- `vdi_audit_logs` — 操作审计日志表

### D-08: 配置管理系统
**选择：所有配置项集成到参数管理页面 + 支持热启动**

**实现方式：**
1. **参数管理页面配置：**
   - VDI服务器配置、Agent配置、策略配置都可通过参数管理页面修改
   - 配置存储在 `sys_config` 表中
   - 前端参数管理页面支持VDI分类的配置项

2. **热启动支持：**
   - 配置修改后立即生效，无需重启服务
   - 后端使用30秒缓存，配置变更后自动刷新
   - Agent配置变更通过WebSocket推送给在线Agent

3. **配置项示例：**
   - VDI服务器连接配置（endpoint、tenant_id、timeout）
   - Agent配置（心跳间隔、速率限制、日志级别）
   - 密码策略配置（最小长度、复杂度要求、有效期）

### D-09: 前端菜单结构
**选择：虚拟机详情页集成账号管理，避免并列菜单混淆**

**设计理由：**
- **数据关系**：账号属于虚拟机（`vdi_vm_accounts.vm_id` 外键），不存在独立的全局账号
- **用户体验**：在虚拟机详情页直接管理账号，避免菜单间的来回跳转
- **业界实践**：VMware vSphere、Azure Portal等成熟产品都在虚拟机详情页管理该虚拟机的用户/权限

**菜单结构：**
```
虚拟机管理 (一级菜单)
├── 服务器配置 (管理功能)
│   ├── VDI服务器列表
│   ├── 服务器连接测试
│   └── Agent状态监控
├── 虚拟机管理 (业务功能 - 账号管理集成在详情页)
│   ├── 虚拟机列表
│   └── 虚拟机详情
│       └── 账号管理标签页
│           ├── VM账号列表
│           ├── 创建账号
│           ├── 密码重置
│           └── 删除账号
└── 系统设置 (管理功能 - 全局功能)
    ├── 操作审计日志 (全局，审计所有VM的账号操作)
    ├── 密码策略配置
    ├── 权限管理
    └── 批量操作配置
```

**UI布局设计：**
```
┌─────────────────────────────────────────────────┐
│ 虚拟机详情: VM-WEB-001                          │
├─────────────────────────────────────────────────┤
│ [概览] [账号管理] [操作记录] [监控]            │
├─────────────────────────────────────────────────┤
│  账号管理标签页内容：                            │
│  ┌─────────────────────────────────────────┐   │
│  │ [+ 创建账号]  [批量导入]               │   │
│  │ ┌──────┬─────────┬────────┬────────┐  │   │
│  │ │用户名│操作系统 │状态    │操作    │  │   │
│  │ ├──────┼─────────┼────────┼────────┤  │   │
│  │ │admin │Windows  │启用    │重置密码│  │   │
│  │ │guest │Windows  │禁用    │编辑    │  │   │
│  │ └──────┴─────────┴────────┴────────┘  │   │
│  └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

**权限设计：**
- 一级菜单"虚拟机管理"使用 `vdi:visit` 权限
- 管理功能子菜单使用 `vdi:admin` 权限
- 业务功能子菜单使用细粒度权限（如 `vdi:vm:view`、`vdi:vm:operate`）
- 账号管理操作使用细粒度权限（`vdi:account:create`、`vdi:account:reset`、`vdi:account:delete`）
- 遵循项目现有的权限管理模式（`pkg/permission/`）

### D-10: 权限管理集成
**选择：使用项目现有权限系统和WebSocket基础设施**

**现有实现复用：**
- ✅ **数据范围权限**：复用 `internal/models/base.go` 的 `DataScope` 类型
  - `DataScopeAll = 1` (全部数据)
  - `DataScopeCustom = 2` (自定义数据)
  - `DataScopeDept = 3` (本部门数据)
  - `DataScopeDeptChild = 4` (本部门及子部门数据)
  - `DataScopeSelf = 5` (仅本人数据)
  - **实现方式**：Service层查询时根据用户的Role.DataScope过滤数据（参考现有系统服务实现）
- ✅ **WebSocket基础设施**：复用 `internal/websocket/notice_hub.go` 的 `NoticeHub` 和gorilla/websocket
  - 为VDI Console创建新的WebSocket Hub（`VDIConsoleHub`）
  - 复用现有的连接管理、消息广播机制

**权限标识符：**
- `vdi:visit` — 访问虚拟机管理模块
- `vdi:admin` — VDI系统管理（服务器配置、策略配置）
- `vdi:vm:list` — 查看虚拟机列表
- `vdi:vm:view` — 查看虚拟机详情
- `vdi:vm:create` — 创建虚拟机
- `vdi:vm:update` — 更新虚拟机
- `vdi:vm:delete` — 删除虚拟机
- `vdi:vm:operate` — 虚拟机操作（开关机、重启等）
- `vdi:vm:sync` — 同步虚拟机状态
- `vdi:vm:console` — 打开虚拟机控制台（网页终端）
- `vdi:account:list` — 查看VM账号列表
- `vdi:account:create` — 创建VM账号
- `vdi:account:reset` — 重置账号密码
- `vdi:account:delete` — 删除VM账号
- `vdi:audit:log` — 查看操作审计日志

**数据范围权限控制（复用现有DataScope）：**
虚拟机查询和操作必须遵循用户角色的DataScope（参考现有系统服务的实现模式）：

| DataScope | 可查看的虚拟机 | 后端实现 |
|-----------|--------------|----------|
| **DataScopeSelf (5)** | 仅 `bound_user_id = 当前用户ID` 的虚拟机 | 查询时添加WHERE条件 |
| **DataScopeDept (3)** | `bound_user_id IN (部门用户ID)` 的虚拟机 | 关联sys_user表，通过user.dept_id过滤 |
| **DataScopeAll (1)** | 所有虚拟机（无过滤条件） | 管理员角色，无WHERE限制 |

**操作权限分级（前端按钮根据权限动态显示）：**
| 角色/权限 | 可执行的操作 |
|----------|------------|
| **一般用户** | 开机、关机、重启、查看控制台 |
| **部门管理员** | 上述 + 部门内VM创建、配置IP、重命名 |
| **系统管理员** | 所有操作（包括删除、同步、全局配置） |

**权限实现方式：**
- 后端使用现有的 `middleware.Permission()` 中间件
- Service层复用现有的数据范围过滤逻辑（参考`system/*_service.go`）
- 前端路由使用 `useAuth` 和 `usePermission` 检查
- 前端按钮根据权限动态显示/隐藏
- 角色管理中自动包含VDI权限选项（复用现有角色管理界面）

### D-11: 密码轮换策略（CRITICAL - Phase 22核心需求）
**选择：自动定期修改VM管理员密码 + 密码历史记录**

**功能背景：**
这是Phase 22的核心需求！Phase Boundary明确指出：**"VM管理员密码需要定期修改（安全要求），使得SSH/WinRM直连方案不可行（存储的密码会失效）"**

**功能详情：**
1. **自动定期轮换：**
   - 系统自动定期修改VM管理员密码（默认每90天，可配置）
   - 使用现有调度器（`internal/scheduler/cron.go`）实现
   - 密码长度：16-32位随机字符（大写+小写+数字+特殊符号）
   - 提前7天通知管理员密码即将过期

2. **密码存储加密：**
   - 新密码使用AES-128-GCM加密存储在`vdi_vm_accounts.password_encrypted`字段
   - 不记录明文密码
   - 仅在需要时通过Agent发送到VM

3. **密码历史记录表：**
   ```sql
   CREATE TABLE vdi_password_history (
     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     account_id UUID NOT NULL,  -- vdi_vm_accounts.id
     old_password_hash VARCHAR(64) NOT NULL,  -- 旧密码哈希（用于防重复）
     new_password_hash VARCHAR(64) NOT NULL,  -- 新密码哈希
     rotated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
     rotated_by VARCHAR(100),  -- 操作人（system或用户名）
     reason VARCHAR(500)
   );
   ```

4. **密码轮换策略配置（sys_config）：**
   ```sql
   INSERT INTO sys_config (config_name, config_key, config_value, config_type, remark) VALUES
   ('VDI密码轮换启用', 'vdi.password.rotation_enabled', 'true', 'Y', '是否启用VM管理员密码自动轮换'),
   ('VDI密码轮换间隔', 'vdi.password.rotation_interval', '90', 'N', '密码轮换间隔（天）'),
   ('VDI密码提前通知', 'vdi.password.rotation_advance', '7', 'N', '密码过期前提前通知天数'),
   ('VDI密码最小长度', 'vdi.password.min_length', '16', 'N', '密码最小长度'),
   ('VDI密码历史数量', 'vdi.password.history_count', '5', 'N', '记录最近N次密码（防止重复）');
   ```

5. **调度任务定义：**
   ```yaml
   job_name: "VDI密码自动轮换"
   cron_expression: "0 0 2 * * *"  # 每天凌晨2点检查
   description: "检查所有VM管理员账号，自动轮换到期的密码"
   ```

6. **Service层实现要点：**
   ```go
   type PasswordRotationService struct {
       db          *gorm.DB
       agentClient AgentClient
       scheduler   *scheduler.CronScheduler
       encryption  *encryption.PasswordEncryptor
       notifier    *notification.NotificationService
   }

   // CheckAndRotate 检查并轮换密码
   func (s *PasswordRotationService) CheckAndRotate(ctx context.Context) error {
       // 1. 查询需要轮换的账号
       accounts := s.getAccountsDueForRotation(ctx)
       
       for _, account := range accounts {
           // 2. 生成安全随机密码
           newPassword := s.generateSecurePassword()
           
           // 3. 通过Agent调用VM本地命令修改密码
           err := s.agentClient.RotatePassword(ctx, account.VMID, account.Username, newPassword)
           if err != nil {
               logger.Error("密码轮换失败", "vm", account.VMID, "error", err)
               continue
           }
           
           // 4. 加密存储新密码
           encrypted := s.encryption.Encrypt(newPassword)
           s.updateAccountPassword(ctx, account.ID, encrypted)
           
           // 5. 记录密码历史
           s.recordPasswordHistory(ctx, account.ID, oldHash, newHash)
           
           // 6. 发送通知
           s.sendPasswordRotatedNotification(ctx, account)
       }
       
       return nil
   }
   ```

**实现Wave：**
- **Wave 3 (Service层)**: `PasswordRotationService` 实现
- **Wave 4 (后端API)**: 密码轮换API端点、调度任务注册到`internal/scheduler/`
- **Wave 5 (前端UI)**: 密码策略配置页面（系统设置标签页）

**安全保证：**
- 密码生成使用`crypto/rand`（密码学安全的随机数生成器）
- 密码永不以明文形式记录在日志或数据库中
- Agent传输密码时使用现有TLS加密（复用Agent通信机制）
- 支持紧急手动轮换（管理员可立即触发密码轮换）
- 防止密码重复（检查历史记录中的密码哈希）

**排除的功能：**
- ❌ 自动分发密码给用户 - 安全风险，用户通过Web Console自行登录

### Claude's Discretion
**Agent通信协议：** HTTP REST vs gRPC
- 建议使用HTTP REST（更简单、易于调试、与现有架构一致）
- 如果需要高性能流式数据传输，可使用gRPC

**Agent心跳间隔：** 建议30秒（可配置）
- 平衡实时性和网络开销
- 可通过参数管理页面配置

**Agent版本管理：**
- 支持Agent版本查询和远程升级（可选）

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 架构文档
- `docs/项目概述和架构设计.md` — XingRan-Next整体架构
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
1. Agent首次启动，向XingRan-Next注册（提供VM ID、Agent版本、OS类型）
2. XingRan-Next验证VM属于已配置的VDI服务器
3. 生成JWT令牌（包含Agent ID、VM ID、权限范围、24小时有效期）
4. Agent存储令牌，每次API调用携带在Authorization头
5. 令牌即将过期时（提前1小时），Agent自动刷新

### Agent安装脚本
**Windows (PowerShell):**
```powershell
# 下载Agent
Invoke-WebRequest -Uri "https://xingran-backend.example.com/api/v1/agent/download/windows" -OutFile "agent.zip"
Expand-Archive -Path "agent.zip" -DestinationPath "C:\Program Files\XingRanAgent"

# 创建受限管理员账号
New-LocalUser -Name "XingRanAgentUser" -Password $password -Description "XingRan VM Agent"
# 授予用户管理权限（使用JEA或Restricted Management）

# 注册Windows服务
New-Service -Name "XingRanVMAgent" -BinaryPathName "C:\Program Files\XingRanAgent\agent.exe" -StartupType Automatic
Start-Service "XingRanVMAgent"

# Agent自动注册到后端
```

**Linux (Bash):**
```bash
# 下载Agent
wget -O /tmp/agent.tar.gz "https://xingran-backend.example.com/api/v1/agent/download/linux"
mkdir -p /opt/xingran-agent
tar -xzf /tmp/agent.tar.gz -C /opt/xingran-agent

# 创建专用服务账号
useradd -m xingran-agent

# 配置sudoers.d
cat > /etc/sudoers.d/xingran-agent << EOF
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/useradd
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/userdel
xingran-agent ALL=(root) NOPASSWD: /usr/sbin/usermod
xingran-agent ALL=(root) NOPASSWD: /usr/bin/passwd
EOF

# 注册systemd服务
cat > /etc/systemd/system/xingran-vm-agent.service << EOF
[Unit]
Description=XingRan VM Agent
After=network.target

[Service]
Type=simple
User=xingran-agent
WorkingDirectory=/opt/xingran-agent
ExecStart=/opt/xingran-agent/agent --config=/opt/xingran-agent/config.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable xingran-vm-agent
systemctl start xingran-vm-agent
```

### Agent数据模型
**新增表（使用 vdi_ 前缀）：**
- `vdi_servers` — VDI服务器配置表（id、name、endpoint、username、password_encrypted、tenant_id、auth_token、token_expiry）
- `vdi_vms` — 虚拟机表（vm_id、name、resource_id、power_state、ip_address、mac_address、os_type、vdi_server_id、bound_user_id）
- `vdi_vm_accounts` — VM内部账号表（id、vm_id、account_id、username、account_type、os_type、is_admin、is_enabled、sync_status）
- `vdi_agents` — Agent注册信息表（id、agent_id、vm_id、version、os_type、hostname、ip_address、auth_token、last_heartbeat、status）
- `vdi_audit_logs` — 操作审计日志表（id、vm_id、account_id、operation、operator、operator_ip、details、status、error_message、command_used、execution_time）

**配置表（sys_config，使用参数管理）：**
- `vdi.agent.heartbeat_interval` — Agent心跳间隔（秒）
- `vdi.agent.rate_limit` — Agent速率限制（每分钟请求数）
- `vdi.agent.log_level` — Agent日志级别
- `vdi.password.min_length` — 密码最小长度
- `vdi.password.require_uppercase` — 是否需要大写字母
- `vdi.password.require_lowercase` — 是否需要小写字母
- `vdi.password.require_number` — 是否需要数字
- `vdi.password.require_special` — 是否需要特殊字符

## Deferred Ideas

无 — 讨论保持在Phase 22范围内，无超出范围的提议。

---

*Phase: 22-sangfor-vdi-integration*
*Context gathered: 2026-05-25*
