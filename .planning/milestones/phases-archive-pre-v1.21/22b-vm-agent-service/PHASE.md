---
phase: 22b-vm-agent-service
name: VM Agent 服务
milestone: v1.11
status: complete
created: 2026-05-25
completed: 2026-06-02
depends_on: [22a-vdi-basic-integration]
split_from: 22-sangfor-vdi-integration
waves: [22-06]
estimated_hours: 6-8
actual_hours: 6-8
---

# Phase 22B: VM Agent 服务

## 目标

实现部署在虚拟机内的 Agent 服务，支持账号管理和终端访问。Agent 使用 JWT 令牌向后端认证，独立于 VM 管理员密码工作。

## 包含的 Waves

- **Wave 6 (22-06)**: VM Agent 服务实现

## 依赖关系

- **Phase 22A**（需要 VM 数据模型）

## 交付物

### Agent 服务
- Agent 服务主程序（main.go）
- JWT 认证和令牌刷新
- 跨平台账号操作（创建、删除、启用、禁用）
- 伪终端管理（PTY）
- 心跳上报机制

### 部署工具
- Windows 安装脚本（PowerShell）
- Linux 安装脚本（Bash）
- 跨平台构建脚本

### 后端集成
- Agent 注册 API
- Agent 状态查询 API
- Agent 通信层（HTTPS + JWT）

## 成功标准

当以下所有条件为 TRUE 时，Phase 22B 视为完成：

1. ✅ Agent 可在 Windows/Linux 上部署
2. ✅ Agent 可向后端注册并获取 JWT 令牌
3. ✅ Agent 可执行账号 CRUD 操作
4. ✅ Agent 定期发送心跳
5. ✅ 后端可调用 Agent API
6. ✅ Agent 使用 TLS 1.3 加密通信
7. ✅ Agent 有完整的错误处理和重连机制

## 执行命令

```bash
# 执行整个 Phase 22B
/gsd-execute-phase 22b-vm-agent-service
```

## 预估工作量

6-8 小时（1 个工作日）

## 注意事项

- 需要测试用虚拟机（Windows + Linux）
- 需要准备交叉编译环境
- Agent 的安全性至关重要（JWT、TLS、权限控制）
- 可以与 Phase 22A 的尾部并行开发
