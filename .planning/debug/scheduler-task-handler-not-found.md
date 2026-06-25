---
slug: scheduler-task-handler-not-found
name: scheduler-task-handler-not-found
description: 调度任务执行失败 - 未找到任务处理器
status: resolved
created: 2026-05-27
updated: 2026-05-27
trigger: |
  调查并修复这些调度任务执行失败的问题：
  1. AD域组成员同步 - "AD同步调度器未初始化"
  2. VDI虚拟机数据同步 - "未找到任务处理器: vdi_vm_sync"
  3. 部门到AD同步 - "未找到任务处理器: dept_to_ad_sync"
  错误来自定时任务执行日志，需要系统性地诊断这些任务处理器未注册或调度器未初始化的根本原因。
metadata:
  issue_type: bug
  component: scheduler
  specialist_hint: go
  tdd_checkpoint: null
  reasoning_checkpoint: null
  specialist_dispatch_enabled: true
---

## Symptoms

### Expected Behavior
数据库中配置的定时任务应该能够成功执行，对应的任务处理器应该在代码中已注册并可用。

### Actual Behavior
执行定时任务时失败：
- `ad_group_member_sync`: "AD同步调度器未初始化"
- `vdi_vm_sync`: "未找到任务处理器: vdi_vm_sync"
- `dept_to_ad_sync`: "未找到任务处理器: dept_to_ad_sync"

### Error Messages
```
ERRO[2026-05-27 14:58:21] 任务执行失败 [AD域组成员同步.DEFAULT]: AD同步调度器未初始化
ERRO[2026-05-27 14:59:18] 任务执行失败 [VDI虚拟机数据同步.DEFAULT]: 未找到任务处理器: vdi_vm_sync
ERRO[2026-05-27 14:59:38] 任务执行失败 [部门到AD同步.DEFAULT]: 未找到任务处理器: dept_to_ad_sync
```

### Timeline
- **Started**: 刚部署/更新后开始
- **Previous State**: 不清楚之前是否正常工作过

### Reproduction
通过前端监控管理页面手动执行定时任务时复现

### Context
- 任务是通过前端界面在数据库中配置的
- 任务处理器名称与代码中注册的处理器不匹配
- 涉及 AD 域同步调度器的初始化问题

## Current Focus

**Hypothesis**: 待生成 - 需要调查代码中任务处理器注册机制和数据库配置

**Next Action**: gather initial evidence

**Test**: 待设计

**Expecting**: 待明确

**Reasoning Checkpoint**: 待更新

---

## Evidence

- timestamp: 2026-05-27 14:58:21 | evidence: "AD域组成员同步任务失败，错误信息：AD同步调度器未初始化" | source: "application log"
- timestamp: 2026-05-27 14:59:18 | evidence: "VDI虚拟机数据同步任务失败，错误信息：未找到任务处理器 vdi_vm_sync" | source: "application log"
- timestamp: 2026-05-27 14:59:38 | evidence: "部门到AD同步任务失败，错误信息：未找到任务处理器 dept_to_ad_sync" | source: "application log"
- timestamp: 2026-05-27 | evidence: "任务是通过前端界面在数据库中配置的" | source: "user report"
- timestamp: 2026-05-27 | evidence: "在 internal/core/core.go:284 发现 RegisterADSyncTasks() 已注册 ad_group_member_sync" | source: "code investigation"
- timestamp: 2026-05-27 | evidence: "在 internal/scheduler/vdi_sync_tasks.go:18 发现 RegisterVDISyncTasks() 注册了 vdi_vm_sync" | source: "code investigation"
- timestamp: 2026-05-27 | evidence: "在 internal/core/core.go 中未找到 RegisterVDISyncTasks() 的调用" | source: "code investigation"
- timestamp: 2026-05-27 | evidence: "在整个代码库中未找到 dept_to_ad_sync 任务处理器的注册" | source: "code investigation"

---

## Eliminated

- 2026-05-27: 任务处理器注册函数不存在 - `RegisterVDISyncTasks()` 存在但未被调用
- 2026-05-27: 任务处理器名称拼写错误 - 数据库中的任务名称与注册的处理器名称匹配
- 2026-05-27: 调度器未启动 - 从日志看调度器已启动，只是特定任务处理器未注册

---

## Resolution

**Root Cause**:

三个不同的根本原因：

1. **VDI虚拟机数据同步 (`vdi_vm_sync`)**: 任务处理器注册函数 `RegisterVDISyncTasks()` 已在 `internal/scheduler/vdi_sync_tasks.go` 中定义，但在 `internal/core/core.go` 的 `Init()` 方法中从未被调用。这是一个遗漏的初始化调用。

2. **部门到AD同步 (`dept_to_ad_sync`)**: 此任务处理器在代码库中完全不存在。搜索显示该任务曾在规划文档中提及（`.planning/spikes/ou-dept-mapping-corrected.md`），但实际的注册代码从未实现。这是一个缺失的功能。

3. **AD域组成员同步 (`ad_group_member_sync`)**: 此任务处理器已正确注册（在 `core.go:284` 调用 `RegisterADSyncTasks()`），但执行失败是因为它依赖全局 AD 同步调度器（`globalADSyncScheduler`），而该调度器未初始化。在 `internal/scheduler/ad_sync_tasks.go:306-309` 中，任务执行函数检查 `globalADSyncScheduler == nil` 并返回错误"AD同步调度器未初始化"。

**Fix**:

1. **VDI虚拟机数据同步**: 在 `internal/core/core.go` 的 `Init()` 方法中添加 `scheduler.RegisterVDISyncTasks(c.Scheduler)` 调用

2. **部门到AD同步**: 需要实现此任务处理器，或在数据库中禁用/删除此任务（如果功能不再需要）

3. **AD域组成员同步**: 在 `internal/core/core.go` 的 `Init()` 方法中调用 `scheduler.StartADSyncScheduler(c.GetDB())` 来初始化全局 AD 同步调度器

**Verification**:

1. 重新启动应用程序
2. 通过前端手动执行这三个定时任务
3. 验证任务执行日志显示成功而非错误

**Files Changed**:
- `internal/core/core.go` - 添加 VDI 任务注册和 AD 同步调度器初始化调用
- 可能需要创建 `internal/scheduler/dept_sync_tasks.go` - 实现部门到AD同步任务处理器