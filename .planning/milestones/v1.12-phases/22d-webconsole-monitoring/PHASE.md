---
phase: 22d-webconsole-monitoring
name: Web Console 与监控
milestone: v1.11
status: planned
created: 2026-05-25
depends_on: [22a-vdi-basic-integration, 22b-vm-agent-service, 22c-account-management]
split_from: 22-sangfor-vdi-integration
waves: [22-09, 22-10]
estimated_hours: 6-8
---

# Phase 22D: Web Console 与监控

## 目标

实现网页终端和操作审计功能，提供友好的用户访问体验和完整的操作追踪能力。

## 包含的 Waves

- **Wave 9 (22-09)**: Web Console
- **Wave 10 (22-10)**: 审计和监控

## 依赖关系

- **Phase 22A**（VM 数据模型）
- **Phase 22B**（Agent 服务 + PTY）
- **Phase 22C**（账号数据）

## 交付物

### 后端层
- WebSocket 服务器（终端连接）
- 终端会话管理
- 审计日志查询 API
- Agent 监控 API

### 前端层
- 终端组件（xterm.js）
- 终端会话管理 UI
- 操作记录页面
- Agent 监控仪表板

### 功能特性
- 浏览器内访问 VM 终端
- 终端会话创建、关闭、监控
- 操作历史查询和导出
- Agent 在线状态监控

## 成功标准

当以下所有条件为 TRUE 时，Phase 22D 视为完成：

1. ✅ 用户可通过浏览器访问 VM 终端
2. ✅ 终端会话可创建、关闭、监控
3. ✅ 可查询指定 VM 的操作历史
4. ✅ 可监控 Agent 在线状态
5. ✅ 可导出审计日志
6. ✅ WebSocket 连接稳定（重连机制）
7. ✅ 终端操作响应及时（< 100ms 延迟）

## 执行命令

```bash
# 执行整个 Phase 22D
/gsd-execute-phase 22d-webconsole-monitoring
```

## 预估工作量

6-8 小时（1 个工作日）

## 注意事项

- WebSocket 连接稳定性是关键
- 需要处理终端会话的并发访问
- 审计日志需要考虑长期存储和查询性能
- 可以与 Phase 22C 的尾部并行开发
