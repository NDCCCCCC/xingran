---
status: partial
phase: 42-r1
source: [42-VERIFICATION.md, 42-05-PLAN.md Task 3]
started: 2026-06-27
updated: 2026-06-27
---

# Phase 42-r1 — Manual UAT Required

Phase 42-r1 auto-verification PASSED (status: passed). However, Plan 42-05 Task 3 (manual UAT with dev server) was intentionally deferred to orchestrator because the executor agent cannot start the dev server to test browser-rendered UI in a worktree.

## Current Test

[Awaiting human testing in dev environment]

## Tests

### 1. 5 KPI 卡片数据准确性 (D-06 选型)
**expected**: 5 KPI 卡片数字与 DB COUNT 一致
- Card 1 (全量资产数) = `summary.totalAssets` = `SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL`
- Card 2 (未解决异常数) = `summary.openExceptions` = `sys_data_reconciliation WHERE deleted_at IS NULL AND resolved_at IS NULL`
- Card 3 (critical 级未解决数) = `summary.criticalOpen` (red text)
- Card 4 (7d 新增异常数) = `summary.last7dNew`
- Card 5 (Top1 冲突类型 + 计数) = `summary.topConflictType` + `summary.topConflictCount`
- result: [pending]

### 2. 3 ECharts 渲染与点击导航 (D-05 双向打通)
**expected**: 饼图 / 柱状图 / 趋势图正确渲染,点击扇区/柱条跳转异常列表
- 饼图 (ByConflictType) 6 个扇区 A/B/C/D/E/F,点击任一扇区 → `/asset/reconciliation/exceptions?type=X`
- 柱状图 (BySeverity) 4 个柱 low/medium/high/critical,点击任一柱条 → `/asset/reconciliation/exceptions?severity=Y`
- 趋势图 (HealthTrend) 3 条线 openCount / criticalCount / newCount,默认 7d
- result: [pending]

### 3. 异常列表 admin 页 9 列 + URL 同步 (D-18 只读)
**expected**: 9 列渲染 + URL query 预填筛选 + 默认 detected_at DESC 排序 + 分页正常
- 9 列:detected_at / conflict_type / severity / asset_code / asset_ip / physical_username / responsible_username / exception_rule_id / operlog_btn
- 从图表带 `?type=C` 跳过来,filter 框预填 C,查询结果只显示 Type C
- operlog_btn 点击"查看日志" → `/monitor/operlog?bizId={id}` (只读链接)
- **无"标记已解决"按钮** (D-18)
- result: [pending]

### 4. 父路由 302 (D-04)
**expected**: `/asset/reconciliation` 自动跳转 `/asset/reconciliation/dashboard`
- result: [pending]

### 5. 跨模块权限边界 (W-1 后置验证)
**expected**: 无 `asset:reconciliation:list` 权限时正确拦截
- 用仅含 `ops:workstation:list` 权限的账号登录
- 直接访问 `/asset/reconciliation/dashboard` → 路由守卫拦截(403)
- 切换 admin 账号正常
- result: [pending]

## How to Run UAT

1. **启动 dev server**:
   - 后端:`go run ./cmd/main.go`(或 `.\xingran-backend.exe`)
   - 前端:`cd xingran-react-frontend && npm run dev`
   - 确认两端启动无错误日志

2. **Seed dev DB**:
   - 42-01 migration 已自动跑过(应用启动时)
   - 验证 dict/config/workorder/sys_job seed:
     ```sql
     SELECT COUNT(*) FROM sys_dict_type WHERE dict_type LIKE 'asset_reconciliation%';     -- 期望 4
     SELECT COUNT(*) FROM sys_config WHERE config_key LIKE 'asset.reconciliation.%';       -- 期望 8
     SELECT COUNT(*) FROM sys_workorder_category WHERE name LIKE '对账-%';                  -- 期望 6
     SELECT COUNT(*) FROM sys_job WHERE job_group='reconciliation';                         -- 期望 4
     ```
   - 手动调 `POST /system/job/run` 跑 `reconciliation:refreshView` 让 MV 有数据
   - 或应用启动时 core.Init() 已自动跑一次 StartupRefreshView

3. **运行测试 1-5** (按上方 Tests 段)

## Summary

- total: 5
- passed: 0
- issues: 0
- pending: 5
- skipped: 0
- blocked: 0

## Gaps

[None yet — fill in if any test fails]

## Resolution

- Type "approved" if all 5 tests pass
- If any test fails, describe which KPI / chart / column / permission boundary is wrong, and a Phase 42 gap-closure plan will be created