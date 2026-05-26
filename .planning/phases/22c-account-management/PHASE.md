---
phase: 22c-account-management
name: 账号管理与密码
milestone: v1.11
status: planned
created: 2026-05-25
depends_on: [22a-vdi-basic-integration, 22b-vm-agent-service]
split_from: 22-sangfor-vdi-integration
waves: [22-07, 22-08]
estimated_hours: 7-9
---

# Phase 22C: 账号管理与密码

## 目标

实现虚拟机内账号的完整管理功能和自动密码轮换，解决 VM 管理员密码定期修改的安全要求。

## 包含的 Waves

- **Wave 7 (22-07)**: 账号管理后端
- **Wave 8 (22-08)**: 密码轮换系统

## 依赖关系

- **Phase 22A**（VM 数据模型）
- **Phase 22B**（Agent 服务）

## 交付物

### 数据层
- 3 个数据表（vdi_vm_accounts, vdi_agents, vdi_audit_logs）
- 密码历史表（vdi_password_history）

### 后端层
- 账号管理后端 API（7 个端点）
- Agent 通信层（HTTPS + JWT）
- 密码轮换调度任务
- 密码策略配置服务

### 功能特性
- 账号 CRUD 操作
- 密码重置功能
- 自动密码轮换（默认 90 天）
- 密码历史记录（防止重复使用）
- 审计日志记录

## 成功标准

当以下所有条件为 TRUE 时，Phase 22C 视为完成：

1. ✅ 可管理 VM 内的账号（CRUD）
2. ✅ 可重置账号密码
3. ✅ 系统可自动轮换密码（默认 90 天）
4. ✅ 密码历史记录防止重复使用最近 N 个密码
5. ✅ 密码策略可配置（长度、复杂度、轮换间隔）
6. ✅ 所有账号操作记录到审计日志
7. ✅ 后端通过 HTTPS 调用 Agent API

## 执行命令

```bash
# 执行整个 Phase 22C
/gsd-execute-phase 22c-account-management
```

## 预估工作量

7-9 小时（1-1.5 个工作日）

## 注意事项

- 涉及生产环境密码修改，需要特别谨慎测试
- 密码轮换任务需要可靠的调度机制
- 审计日志需要长期存储和查询能力
- 密码策略需要符合企业安全要求
