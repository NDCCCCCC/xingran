# Phase 22: 深信服桌面云集成 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-25
**Phase:** 22-sangfor-vdi-integration
**Areas discussed:** VM连接方式选择, Agent功能范围, Agent权限设计, Agent安全机制, Agent部署方式

---

## VM连接方式选择

| Option | Description | Selected |
|--------|-------------|----------|
| 专用服务账号（不定期修改） | 创建专用的VM服务账号，仅用于XingRan-Next连接，该账号密码不定期修改。账号权限限制为仅能管理用户。 | |
| **Agent方案（解决密码依赖）** | **在VM内安装轻量级Agent程序。Agent使用JWT令牌认证，安装后独立工作，不依赖VM管理员密码。** | ✓ |
| SSH密钥认证 + 专用账号 | SSH连接使用公钥认证而非密码。XingRan-Next存储私钥，VM存储公钥。适用于Linux VM。 | |
| 密码同步机制 | 使用SSH/WinRM方案，但建立自动化流程：当VM密码修改时，同步更新XingRan-Next中的密码存储。 | |

**User's choice:** Agent方案（解决密码依赖）

**Notes:** 用户提出了关键的安全需求：VM管理员密码需要定期修改（安全要求），这使得SSH/WinRM方案不可行，因为XingRan-Next中存储的密码会失效。Agent方案通过JWT令牌认证，从根本上解决了这个依赖问题。

---

## Agent功能范围

| Option | Description | Selected |
|--------|-------------|----------|
| **账号CRUD操作** | **创建、删除、启用/禁用本地用户账号。Linux: useradd/userdel/usermod; Windows: New-LocalUser/Remove-LocalUser/Enable-LocalUser/Disable-LocalUser。** | ✓ |
| **密码重置功能** | **重置账号密码。当用户忘记密码或需要定期强制修改时，通过Agent远程重置。** | ✓ |
| **系统状态监控** | **Agent定期上报VM的系统状态（CPU、内存、磁盘使用率、运行进程等）。** | ✓ |
| 远程命令执行 | Agent在VM上执行任意命令或脚本。这是最大权限级别，风险最高。 | |

**User's choice:** 账号CRUD操作 + 密码重置功能 + 系统状态监控

**Notes:** 用户明智地排除了高风险的远程命令执行功能。选择的三个功能构成了最小化但完整的Agent能力集，既能满足核心需求，又保持了良好的安全边界。

---

## Agent权限设计（Windows）

| Option | Description | Selected |
|--------|-------------|----------|
| **受限管理员账号（推荐）** | **为Agent创建专用本地账号（如XingRanAgentUser），仅授予「用户管理」权限。使用Restricted Management模式或PowerShell JEA。** | ✓ |
| LocalSystem（高权限） | Agent以LocalSystem身份运行（Windows服务模式）。拥有最高权限，安全风险最高。 | |
| 普通用户 + JEA提权 | Agent以普通用户身份运行，通过PowerShell Remoting或JEA提权执行特定操作。配置复杂，但安全性最好。 | |
| runas模拟管理员 | Agent作为内置Administrator账号的子进程运行。使用runas命令以管理员身份执行具体操作。 | |

**User's choice:** 受限管理员账号（推荐）

**Notes:** 用户选择了最佳的安全/可用性平衡方案。受限管理员账号提供了完成工作所需的最小权限，同时限制了Agent的攻击面。

---

## Agent权限设计（Linux）

| Option | Description | Selected |
|--------|-------------|----------|
| **专用服务账号 + 受限sudo** | **创建专用服务账号（如xingran-agent），通过sudoers.d配置仅允许特定命令。无法修改系统配置、无法安装软件。** | ✓ |
| root权限运行（不推荐） | Agent以root身份运行（systemd服务）。实现简单，但安全风险最高。 | |
| 普通用户 + Capabilities | Agent以普通用户运行，使用Linux Capabilities机制授予特定能力。配置复杂。 | |
| Polkit授权策略 | 使用Polkit（Policy Kit）定义精细的权限策略。灵活但配置复杂。 | |

**User's choice:** 专用服务账号 + 受限sudo（按照Claude的建议选择）

**Notes:** 用户委托Claude推荐，Claude选择了专用服务账号 + 受限sudo方案。这与Windows受限管理员保持一致的安全原则，通过sudoers.d显式允许特定命令，其他命令全部拒绝。

---

## Agent安全机制

| Option | Description | Selected |
|--------|-------------|----------|
| **JWT令牌认证** | **Agent向XingRan-Next注册时，后端生成唯一JWT令牌。每次API调用必须携带有效令牌。** | ✓ |
| **TLS加密通信** | **Agent与XingRan-Next之间通信使用TLS 1.3加密。证书由XingRan-Next CA签发。** | ✓ |
| **操作审计日志** | **Agent所有操作记录审计日志：操作时间、操作人、操作类型、目标账号、执行结果。** | ✓ |
| **速率限制** | **XingRan-Next对Agent请求进行速率限制：每分钟最多100次操作。** | ✓ |

**User's choice:** JWT令牌认证 + TLS加密通信 + 操作审计日志 + 速率限制

**Notes:** 用户选择了全面的安全防护机制，形成了多层防御：认证（JWT）、加密（TLS）、审计（日志）、流量控制（速率限制）。这展示了纵深防御的安全理念。

---

## Agent部署方式

| Option | Description | Selected |
|--------|-------------|----------|
| **VDI镜像预装（推荐）** | **在VDI镜像模板中预装Agent。所有新创建的VM自动包含Agent，开箱即用。** | ✓ |
| VDI API自动安装 | VM创建完成后，XingRan-Next调用VDI API执行安装脚本。灵活，但依赖VDI API支持。 | |
| 手动安装（备用） | 管理员手动登录VM安装Agent。适合测试环境或特殊情况。 | |
| cloud-init/unattend | 在VDI创建VM时，通过cloud-init或unattend.xml自动执行Agent安装脚本。 | |

**User's choice:** VDI镜像预装（推荐）

**Notes:** 用户选择了最优的部署方式。VDI镜像预装提供了最佳的用户体验 - 所有新VM自动包含Agent，零配置。虽然更新Agent需要重新制作镜像，但这是可接受的权衡。

---

## Claude's Discretion

无 — 用户对所有关键决策都提供了明确的选择，未委托Claude决定。

---

## Deferred Ideas

无 — 讨论保持在Phase 22范围内，无超出范围的提议。
