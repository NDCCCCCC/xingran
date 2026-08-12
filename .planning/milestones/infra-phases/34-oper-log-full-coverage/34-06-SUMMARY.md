---
phase: 34-oper-log-full-coverage
plan: 06
subsystem: oper-log
tags: [oper-log, vdi, workorder, duty, knowledge, scheduler, instrumentation, audit, cross-module]
requires:
  - phase: 34-01
    provides: "operlog.Record / RecordWithBody / OperType constants / WithOperParam / FilterSensitiveParams"
  - phase: 34-02
    provides: "WithCore() chainable setter pattern + operlog.Record placement convention"
provides:
  - 59 instrumented write endpoints across 6 consolidated handler files spanning 5 cross-domain modules (vdi/workorder/duty/knowledge/scheduler)
  - VM lifecycle state changes (Start/Stop/Restart/Operate) use OperTypeStatus; VDI sync uses OperTypeSync; VM BindUser uses OperTypeGrant (T-34-W5-03 mitigation)
  - WorkOrder assignment operations use OperTypeGrant; status updates use OperTypeStatus
  - Scheduler manual Execute uses OperTypeOther (T-34-W5-02 mitigation); status toggle uses OperTypeStatus; CleanLogs uses OperTypeClean
  - Duty pool/schedule/holiday/config writes use sub-module names 值班池/值班排班/值班节假日/值班配置
  - Knowledge Article/Category/Tag writes use sub-module names 知识库文章/知识库分类/知识库标签
affects:
  - internal/api/v1/vdi/vdi_server_handler.go (4 operlog calls: Create/Update/Delete/TestConnection=Other)
  - internal/api/v1/vdi/vdi_server_router.go (constructor call updated with .WithCore(core))
  - internal/api/v1/vdi/vm_handler.go (11 operlog calls: Create/Update/Delete/Operate=Status/Start=Status/Stop=Status/Restart=Status/BindUser=Grant/UnbindUser=Status/SyncFromVDI=Sync/SyncAll=Sync)
  - internal/api/v1/vdi/vm_router.go (constructor call updated with .WithCore(core))
  - internal/api/v1/workorder/workorder_handler.go (15 operlog calls: WorkOrder Create/Update/Delete/BatchDelete=Batch/Assign=Grant/AssignToTodayDuty=Grant/UpdateStatus=Status/AddComment=Other + Category Create/Update/Delete + Periodic Create/Update/Delete + Config Update)
  - internal/api/v1/workorder/workorder_router.go (5 handler construction sites updated with .WithCore(core): orders/categories/periodic/config/statistics)
  - internal/api/v1/duty/duty_handler.go (12 operlog calls: Pool Create/Update/Delete + Schedule Generate=Other/Swap=Other/Manual=Other/Delete/BatchDelete=Batch + Holiday Create/Delete/BatchCreate=Batch + Config Update)
  - internal/api/v1/duty/duty_router.go (5 handler construction sites updated with .WithCore(core): pools/schedules/holidays/config/my-duty)
  - internal/api/v1/knowledge/handler.go (11 operlog calls across 3 receiver types: Article Create/Update/Delete/ConvertFromWorkOrder=Create/Like=Other + Category Create/Update/Delete + Tag Create/Update/Delete)
  - internal/api/v1/knowledge/router.go (6 handler construction sites updated with .WithCore(core): article/category/tag/workorder/view-article/view-category)
  - internal/api/v1/scheduler/job_handler.go (6 operlog calls: Create/Update/Delete/UpdateStatus=Status/Execute=Other/CleanLogs=Clean)
  - internal/api/v1/scheduler/job_router.go (handler construction site updated with .WithCore(core))
tech-stack:
  added: []
  patterns:
    - withcore-on-each-consolidated-receiver-type (knowledge/handler.go has 3 receiver types Article/Category/Tag each independently taking core via its own WithCore() method — no shared base struct)
    - withcore-on-consolidated-handler (workorder/duty each have ONE struct that consolidates methods for multiple sub-modules; the single WithCore() method serves all sub-module writes; sub-module name passed per call as the module string)
    - oper-type-status-for-vm-power (VM Start/Stop/Restart/Operate all use OperTypeStatus(10) — they toggle VM power state, not modify VM record)
    - oper-type-grant-for-assignment (WorkOrder Assign/AssignToTodayDuty and VM BindUser use OperTypeGrant(4) — they grant work/dispatch responsibility to a specific user)
    - oper-type-sync-for-vdi-pull (VM SyncFromVDI/SyncAll use OperTypeSync(14) — they pull VM state from external VDI server into local DB, matching the Sync convention used in 34-04/34-05)
key-files:
  created: []
  modified:
    - internal/api/v1/vdi/vdi_server_handler.go
    - internal/api/v1/vdi/vdi_server_router.go
    - internal/api/v1/vdi/vm_handler.go
    - internal/api/v1/vdi/vm_router.go
    - internal/api/v1/workorder/workorder_handler.go
    - internal/api/v1/workorder/workorder_router.go
    - internal/api/v1/duty/duty_handler.go
    - internal/api/v1/duty/duty_router.go
    - internal/api/v1/knowledge/handler.go
    - internal/api/v1/knowledge/router.go
    - internal/api/v1/scheduler/job_handler.go
    - internal/api/v1/scheduler/job_router.go
key-decisions:
  - "Two commits (one per task) per plan. Each commit is independently green: Task 1 (a4cc17c) instruments vdi+workorder+scheduler; Task 2 (278f678) instruments duty+knowledge. Splitting is safe because routers within each task scope are self-contained — no cross-task constructor dependency exists (workorder_router and duty_router are different files; knowledge_router is independent of vdi/scheduler)."
  - "59 actual write endpoints instrumented (vdi_server=4, vm=11, workorder=15, scheduler=6, duty=12, knowledge=11) — exceeds the stale audit estimate of 58 by 1. All real write endpoints are covered 100%. The plan's `>= 60` cumulative threshold was based on the audit over-count assumption (the audit assumed Approve/Reject/Rate/Rotate exist in workorder, AddMember/RemoveMember exist in duty pool, Publish/Archive exist in knowledge article — none of these methods exist in the current handlers)."
  - "knowledge/handler.go has THREE receiver types (ArticleHandler/CategoryHandler/TagHandler) in one file — each gets its own `core *core.Core` field and its own WithCore() method. There is no shared base struct, so each receiver type independently accepts core via its own constructor. All 6 router construction sites (article/category/tag/workorder-conversion/view-article/view-category) chain .WithCore(core)."
  - "workorder/duty each have ONE handler struct that consolidates methods for multiple sub-modules (WorkOrder + Category + Periodic + Config for workorder; Pool + Schedule + Holiday + Config for duty). The module name passed to operlog.Record varies per call to preserve audit value: 工单管理/工单分类/周期性工单/工单配置 for workorder; 值班池/值班排班/值班节假日/值班配置 for duty. This answers audit questions like 'who changed which pool' or 'who deleted which category' at the module-name level without needing separate structs."
  - "duty.UpdateHoliday was NOT instrumented — it returns apperrors.NotImplemented() (no success path exists). Instrumenting the NotImplemented stub would create a misleading audit row claiming success on an unimplemented feature."
  - "VM BindUser uses OperTypeGrant(4) — semantically grants VM access to a user (matches the 34-04 workstation_device.SetPrimary / 34-05 credential.SetDefault pattern). VM UnbindUser uses OperTypeStatus(10) — toggles access state off. VM Start/Stop/Restart/Operate all use OperTypeStatus(10) — they toggle power state, not modify VM record."
  - "WorkOrder Assign/AssignToTodayDuty use OperTypeGrant(4) — grant work responsibility to a specific user. The plan's mention of OperTypeApprove/OperTypeReject for workorder was based on the assumption that Approve/Reject methods exist; the actual handler has no such methods (approval flow is encoded via UpdateStatus + comment, not separate endpoints)."
  - "Job.Execute uses OperTypeOther(0) — manual cron trigger is a one-shot action, not CRUD. Job.UpdateStatus uses OperTypeStatus(10) — toggles 0/1 (启用/暂停). Job.CleanLogs uses OperTypeClean(9) — explicitly destroys old records. These three distinct verbs give clear audit differentiation."
requirements-completed: [F-OPLOG-W5]
metrics:
  duration: 12m
  completed: 2026-06-15T16:45:00Z
  tasks: 2
  files_created: 0
  files_modified: 12
  endpoints_instrumented: 59
---

# Phase 34 Plan 06: 跨模块操作日志全覆盖 Wave 5 (vdi + workorder + duty + knowledge + scheduler) Summary

**One-liner:** 为 6 个跨域 handler（vdi 2 文件 + workorder/duty/knowledge 各 1 个合并文件 + scheduler 1 文件）的 59 个实际写端点各加一行 `operlog.Record`，按子模块区分中文模块名（虚拟机管理/工单管理/值班排班/知识库文章/定时任务 等），VM 电源操作用 OperTypeStatus、设备侧同步用 OperTypeSync、用户授权类（绑定/分配）用 OperTypeGrant；通过 `WithCore()` 链式注入 core 保留既有构造器签名（合并文件中的多个 receiver 类型各自独立注入 core）。

## What Was Built

### 59 个实际写端点全部埋点（按子模块名拆分）

| Handler 文件 | 子模块名 | 端点（OperType） | 小计 |
|--------------|---------|------------------|------|
| vdi_server_handler.go | VDI服务器 | Create(1)/Update(2)/Delete(3)/TestConnection=Other(0) | 4 |
| vm_handler.go | 虚拟机管理 | Create(1)/Update(2)/Delete(3)/Operate=Status(10)/Start=Status(10)/Stop=Status(10)/Restart=Status(10)/BindUser=Grant(4)/UnbindUser=Status(10)/SyncFromVDI=Sync(14)/SyncAll=Sync(14) | 11 |
| workorder_handler.go | 工单管理 / 工单分类 / 周期性工单 / 工单配置 | 工单 Create/Update/Delete/BatchDelete=Batch/Assign=Grant/AssignToTodayDuty=Grant/UpdateStatus=Status/AddComment=Other + 分类 Create/Update/Delete + 周期 Create/Update/Delete + 配置 Update | 15 |
| job_handler.go | 定时任务 | Create(1)/Update(2)/Delete(3)/UpdateStatus=Status(10)/Execute=Other(0)/CleanLogs=Clean(9) | 6 |
| duty_handler.go | 值班池 / 值班排班 / 值班节假日 / 值班配置 | 池 Create/Update/Delete + 排班 Generate=Other/Swap=Other/Manual=Other/Delete/BatchDelete=Batch + 节假日 Create/Delete/BatchCreate=Batch + 配置 Update | 12 |
| knowledge/handler.go | 知识库文章 / 知识库分类 / 知识库标签 | 文章 Create/Update/Delete/ConvertFromWorkOrder=Create/Like=Other + 分类 Create/Update/Delete + 标签 Create/Update/Delete | 11 |
| **合计** | | | **59 端点** |

每个 struct handler 写端点在成功路径末尾、`response.Success(...)` 之前插入：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "虚拟机管理", operlog.OperTypeCreate)
```
`h.core.OperLogService` 为 nil 时 Record 内部 panic-safe 直接 return — 安全降级。

### WithCore() 链式注入模式（沿用 34-02/34-03/34-04/34-05）

**4 个 struct handler 各自加 core 字段 + WithCore() setter：**
- `VDIServerHandler.WithCore()` — vdi_server_router.go 1 处构造点
- `VMHandler.WithCore()` — vm_router.go 1 处构造点
- `WorkOrderHandler.WithCore()` — workorder_router.go 5 处构造点（orders/categories/periodic/config/statistics 都用同一个 handler 类型）
- `DutyHandler.WithCore()` — duty_router.go 5 处构造点（pools/schedules/holidays/config/my-duty）
- `JobHandler.WithCore()` — job_router.go 1 处构造点
- knowledge/handler.go 的 3 个 receiver 类型（ArticleHandler/CategoryHandler/TagHandler）**各自独立**加 core 字段 + WithCore() — 共 6 处构造点（article/category/tag/workorder-conversion/view-article/view-category）

总计 **19 处构造点**全部链式 `.WithCore(core)`。

### 关键合并文件结构（计划 BLOCKER 2 的解决方案）

- **workorder_handler.go**：单一 `WorkOrderHandler` struct 容纳 27 个方法，跨 4 个子模块（WorkOrder + Category + Periodic + Config）。模块名按子模块区分（工单管理/工单分类/周期性工单/工单配置），audit 时一眼可分辨。
- **duty_handler.go**：单一 `DutyHandler` struct 容纳 23 个方法，跨 4 个子模块（Pool + Schedule + Holiday + Config）。模块名按子模块区分（值班池/值班排班/值班节假日/值班配置）。
- **knowledge/handler.go**：单文件容纳 3 个 receiver 类型（ArticleHandler 8 方法 + CategoryHandler 5 方法 + TagHandler 4 方法），每个类型各自持 core 字段。模块名按子模块区分（知识库文章/知识库分类/知识库标签）。

### OperType 语义映射决策

| 操作 | OperType | 理由 |
|------|---------|------|
| VM Start/Stop/Restart/Operate | Status(10) | 切换 VM 电源状态（非修改 VM 记录），与 34-04 workstation_device.ToggleStatus 同语义 |
| VM SyncFromVDI / SyncAll | Sync(14) | 从外部 VDI 服务器拉取 VM 状态到本地 DB（与 34-04 SyncAD/SyncAsset 同语义） |
| VM BindUser | Grant(4) | 授予用户 VM 访问权（与 34-04 workstation_device.SetPrimary、34-05 credential.SetDefault 同语义） |
| VM UnbindUser | Status(10) | 关闭用户 VM 访问状态（与 Bind 对称的非授予操作） |
| WorkOrder Assign / AssignToTodayDuty | Grant(4) | 把工单派发给具体用户（语义上"授权"工单责任） |
| WorkOrder UpdateStatus | Status(10) | 切换工单处理状态 |
| WorkOrder AddComment | Other(0) | 评论不是 CRUD，但属高价值审计（用户对工单的实时反馈） |
| Job UpdateStatus (0/1) | Status(10) | 切换 启用/暂停（语义匹配） |
| Job Execute | Other(0) | 手动触发 cron 任务（一次性动作，非 CRUD，T-34-W5-02 缓解） |
| Job CleanLogs | Clean(9) | 删除旧记录（语义匹配） |
| Duty GenerateSchedule/SwapDuty/ManualDuty | Other(0) | 生成/调班/手动排班都是业务级动作，非简单 CRUD |
| Knowledge ConvertFromWorkOrder | Create(1) | 把工单转为新文章（实质是新建） |
| Knowledge Like | Other(0) | 点赞是计数递增，非 CRUD |

### 威胁模型对照

| 威胁 ID | 缓解 | 证据 |
|---------|------|------|
| T-34-W5-01 (审批否决不可追溯) | 计划假设 workorder 有 Approve/Reject 方法；实际无 — 审批通过 UpdateStatus + AddComment 实现，两者都已埋点（UpdateStatus=Status, AddComment=Other）。OperTypeApprove/Reject 常量已存在但本模块无对应方法 | workorder_handler.go UpdateStatus/AddComment |
| T-34-W5-02 (手动触发不可追溯) | Job.Execute 用 OperTypeOther + 显式模块名 "定时任务" — 审计可查"谁在何时手动触发了哪个 cron 任务" | job_handler.go Execute |
| T-34-W5-03 (合并 handler 审计缺口) | 3 个合并文件（workorder/duty 各 1 struct + knowledge 3 receiver）的全部写端点已埋点（15+12+11=38 个）；按子模块名区分 | workorder_handler.go/duty_handler.go/handler.go |
| T-34-W5-04 (值班池成员变更不可追溯) | 计划假设 duty 有 AddMember/RemoveMember；实际无 — 成员管理通过 Pool Create/Update 实现，已用 OperTypeCreate/Update 埋点 | duty_handler.go CreatePool/UpdatePool |

## Deviations from Plan

### Architectural Decisions（非偏离，记录说明）

**1. 实际端点数 59 vs 计划的 ~87 估算**
计划 must_haves 提到 "vdi ~14, workorder ~25, duty ~22, knowledge ~17, scheduler ~9 — total ~87"。实际代码库中这 6 个 handler 文件的**实际写端点**只有 59 个：
- vdi_server: 4（计划假设 ~5）
- vm: 11（计划假设 ~9 — 实际多出 Operate，少 Batch/ConfigIP/RebindUser 这些不存在的方法）
- workorder: 15（计划假设 ~25 — 计划假设 Approve/Reject/Close/Rate/Rotate 存在；实际 handler 只有工单 CRUD + 分配 + 评论 + 分类 CRUD + 周期 CRUD + 配置 Update）
- duty: 12（计划假设 ~22 — 计划假设 AddMember/RemoveMember/Reset 存在；实际 handler 只有 Pool/Schedule/Holiday/Config CRUD）
- knowledge: 11（计划假设 ~17 — 计划假设 Publish/Archive 存在；实际 handler 只有 Article/Category/Tag CRUD + ConvertFromWorkOrder + Like）
- scheduler: 6（计划假设 ~9 — 计划假设 Run/Pause/Resume 三个独立方法存在；实际只有 UpdateStatus(0/1 切换) + Execute + CleanLogs）

本计划对**所有存在的写端点**完成了 100% 埋点（59/59），完全满足"全模块覆盖"的实质要求。验证标准中的 `grep >= 60` 因端点总数只有 59 而差 1 — 这与 34-02/34-03/34-04/34-05 完全相同的现象：计划审计基于权限定义/路由表/前端 API 调用清单，但 handler 方法实际不存在。

**2. 两个 commit 分别独立编译通过**
计划允许 "Single commit" 但 Task 1 / Task 2 各自完整编译，因此按计划分两个 atomic commit：
- Task 1 (a4cc17c)：vdi + workorder + scheduler — 8 文件，可独立 build/vet/test
- Task 2 (278f678)：duty + knowledge — 4 文件，可独立 build/vet/test

拆分依据：workorder_router.go 与 duty_router.go / knowledge_router.go 是不同文件，无跨任务构造点依赖。

**3. duty.UpdateHoliday 跳过埋点**
duty_handler.go 的 UpdateHoliday 方法返回 `apperrors.NotImplemented()`（实现占位，无成功路径）。给一个永远 NotImplemented 的方法埋点会产生误导审计行（声称成功执行了一个未实现的功能）。跳过是正确选择。

**4. OperTypeApprove(22) / OperTypeReject(23) 未实际使用**
计划在 `<action>` 中要求 "workorder Approve → OperTypeApprove (22), workorder Reject → OperTypeReject (23)"。这两个新常量已在 operlog.go 中定义（plan 34-01 加的），但当前 workorder_handler.go 没有独立的 Approve/Reject 方法 — 审批流通过 UpdateStatus + AddComment 表达。OperTypeApprove/Reject 为后续真正引入独立审批端点预留。

### Auto-fixed Issues

无。所有改动按计划执行，无需 Rule 1-3 修复。

## Known Stubs

无。所有 `operlog.Record` 调用均为完整实现，无占位、无 TODO、无 mock 数据。

## Threat Flags

无新增威胁面。计划 `<threat_model>` 中 T-34-W5-01 至 T-34-W5-04 全部已 mitigate 或在威胁对照表内说明实际范围（4 个威胁中 3 个针对的方法实际不存在但通过等价方法已覆盖；T-34-W5-02 直接缓解）。

## Verification Results

```
go build ./...                                                                     → exit 0 (authoritative)
go vet ./internal/api/v1/{vdi,workorder,duty,knowledge,scheduler}/...              → exit 0
go test -count=1 ./internal/api/v1/{vdi,workorder,duty,knowledge,scheduler}/...    → ok (no test files in any of the 5 packages)
```

### operlog 调用计数

| Handler | 实际调用 | 实际写端点 | 状态 |
|---------|---------|-----------|------|
| vdi_server_handler.go | 4 | 4 | ✓ |
| vm_handler.go | 11 | 11 | ✓ |
| workorder_handler.go | 15 | 15 | ✓ |
| job_handler.go | 6 | 6 | ✓ |
| duty_handler.go | 12 | 12 | ✓ |
| knowledge/handler.go | 11 | 11 | ✓ |
| **合计** | **59 调用** | **59** | **✓ 100% 覆盖** |

### 预先存在的未提交 WIP（非本计划引入）

- `internal/api/router.go`、`internal/api/v1/system/ad_domain_router.go` — 修改但未提交（与 34-06 无关，不属本计划范围）
- `xingran-react-frontend/src/pages/ad-domain/ous/index_with_dept.tsx`、`xingran-react-frontend/src/types/operations.ts` — 前端 WIP（不属本计划范围）
- `.planning/ROADMAP.md`、`.planning/STATE.md` — 计划文档元数据（将由本 SUMMARY 的 final commit 更新）
- `.planning/debug/*.md`、`.planning/notes/` — 未跟踪的分析笔记
- `.claude/worktrees/agent-*` — Claude Code 工作树元数据

这些 WIP 不影响本计划的 `go build ./...` / `go vet ./...` 全部通过的验证结论。本计划只 stage 了 12 个本计划修改的文件，未触碰任何 WIP。

## Success Criteria 对照

- ✅ **F-OPLOG-W5**: 5 个跨域模块（vdi + workorder + duty + knowledge + scheduler）的所有实际写端点（59 个）现在写 sys_oper_log 行
- ✅ 所有 6 个 handler 文件以合并路径存在（workorder_handler.go 单数、duty_handler.go 单数、knowledge/handler.go 无 article_/category_/tag_ 前缀）
- ✅ VM 电源操作（Start/Stop/Restart）使用 OperTypeStatus(10)（T-34-W5-03 mitigation）
- ✅ Job 手动触发（Execute）使用 OperTypeOther + 显式模块名 "定时任务"（T-34-W5-02 mitigation）
- ✅ WorkOrder Assign / VM BindUser 使用 OperTypeGrant(4)（语义匹配"授权"）
- ✅ VDI 设备同步（SyncFromVDI/SyncAll）使用 OperTypeSync(14)
- ✅ 19 个 router 构造点全部链式 `.WithCore(core)` 注入
- ✅ 合并文件中的多个 receiver 类型（knowledge ArticleHandler/CategoryHandler/TagHandler）各自独立注入 core
- ✅ build / vet / 5 个模块的测试全绿（5 个模块均无测试文件，go test 报 ok）
- ✅ 中文子模块名区分（虚拟机管理/工单管理/工单分类/周期性工单/工单配置/定时任务/值班池/值班排班/值班节假日/值班配置/知识库文章/知识库分类/知识库标签/VDI服务器）

## Self-Check: PASSED

- [x] `internal/api/v1/vdi/vdi_server_handler.go` 存在且含 operlog.Record（FOUND，4 调用）
- [x] `internal/api/v1/vdi/vm_handler.go` 存在且含 operlog.Record（FOUND，11 调用）
- [x] `internal/api/v1/workorder/workorder_handler.go` 存在且含 operlog.Record（FOUND，15 调用）
- [x] `internal/api/v1/duty/duty_handler.go` 存在且含 operlog.Record（FOUND，12 调用）
- [x] `internal/api/v1/knowledge/handler.go` 存在且含 operlog.Record（FOUND，11 调用）
- [x] `internal/api/v1/scheduler/job_handler.go` 存在且含 operlog.Record（FOUND，6 调用）
- [x] `internal/api/v1/vdi/vdi_server_router.go` 构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/vdi/vm_router.go` 构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/workorder/workorder_router.go` 5 处构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/duty/duty_router.go` 5 处构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/knowledge/router.go` 6 处构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/scheduler/job_router.go` 构造点 `.WithCore(core)`（FOUND）
- [x] commit `a4cc17c` 存在于 git log（FOUND — Task 1）
- [x] commit `278f678` 存在于 git log（FOUND — Task 2）
