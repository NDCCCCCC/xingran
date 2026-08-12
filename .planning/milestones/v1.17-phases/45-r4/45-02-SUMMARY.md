---
phase: 45-r4
plan: 02
subsystem: asset-reconciliation
tags: [INTEGRATE-01, INTEGRATE-02, INTEGRATE-03, R4-closure, cache-invalidation, operlog, ip-resolution, verification, regression-tests]
dependency_graph:
  requires:
    - "45-r4/45-01 (R4 integration — read path + UI + cross-module injection)"
  provides:
    - "ReconciliationWorkorderService.WorkstationIDForException — 旁路反查方法(不动 CreateWorkorderFromException 签名)"
    - "ReconciliationWorkorderService.InvalidateWorkstationHealth — service-level 缓存失效 helper"
    - "ReconciliationExceptionService.MatchException — per-asset 轻量级命中"
    - "R2 scheduler 缓存主动失效(InvalidateWorkstationHealth in createWorkorderBySeverity)"
    - "ResolveException handler 严格 success path 顺序: service → invalidate → operlog → response"
    - "GetByWorkstation 集成 IP 解析链 (asset → workstation → network_device via port → unknown)"
    - "Drawer 申请例外 URL 携带 assetIp + conflictType + workstationId (SC9)"
    - "9 个回归守护测试 (useReconciliationVisibility 4 + HealthCard 5)"
    - "VERIFICATION.md 10 SCs 全部 PASS 证据"
  affects:
    - "internal/scheduler/reconciliation_tasks.go (R2 cache invalidation)"
    - "internal/services/asset/reconciliation_workorder.go (旁路方法 + cache 字段)"
    - "internal/services/asset/reconciliation_service.go (IP 解析链 inline + matcher 注入)"
    - "internal/services/asset/reconciliation_exception.go (MatchException method)"
    - "internal/api/v1/asset/reconciliation_handler.go (ResolveException success path)"
    - "internal/api/v1/asset/reconciliation_router.go (exceptionSvc 注入)"
    - "internal/api/router.go (跨模块 exceptionSvc 注入)"
    - "internal/core/core.go (RegisterReconciliationTasks 新增 cache 参数)"
    - "xingran-react-frontend/src/components/reconciliation/ExceptionMatchList.tsx (SC9 URL params)"
    - "xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx (内联 handleApplyException)"
    - "xingran-react-frontend/src/pages/operations/{assets,workstations}/index.tsx (移除冗余 onApplyException)"
    - ".planning/notes/260627-cross-module-permission.md (R4 实际接入清单)"
    - ".planning/phases/45-r4/45-VERIFICATION.md (10 SCs verification)"
tech-stack:
  added: []
  patterns:
    - "Bypass method (WorkstationIDForException) — 不修改既有方法签名,旁路反查"
    - "Service-level cache invalidation (InvalidateWorkstationHealth on ReconciliationWorkorderService) — nil-safe 设计"
    - "Public MatchException method — 暴露 package-private matchException 供 service 复用"
    - "Inline IP resolution chain (3-level fall-through) — 不抽新文件(B5 修复)"
    - "Conditional render in drawer with self-handled navigation — 父级 onApplyException 变可选"
    - "useCallback placed before early return — react-hooks/rules-of-hooks invariant"
key-files:
  created:
    - .planning/phases/45-r4/45-VERIFICATION.md
    - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx
    - xingran-react-frontend/src/components/reconciliation/hooks/__tests__/useReconciliationVisibility.test.ts
  modified:
    - internal/scheduler/reconciliation_tasks.go
    - internal/services/asset/reconciliation_workorder.go
    - internal/services/asset/reconciliation_service.go
    - internal/services/asset/reconciliation_exception.go
    - internal/api/v1/asset/reconciliation_handler.go
    - internal/api/v1/asset/reconciliation_router.go
    - internal/api/router.go
    - internal/core/core.go
    - xingran-react-frontend/src/components/reconciliation/ExceptionMatchList.tsx
    - xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx
    - xingran-react-frontend/src/pages/operations/assets/index.tsx
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
    - .planning/notes/260627-cross-module-permission.md
decisions:
  - "B2 锁定严格执行:CreateWorkorderFromException / ResolveException service 签名未变"
  - "WorkstationIDForException 旁路方法:不动 CreateWorkorderFromException,反查 + 缓存失效走旁路"
  - "InvalidateWorkstationHealth 在 3 处调用(handler-level + scheduler-level + service-level)"
  - "ReconciliationWorkorderService 新增 cache 字段 + 新构造器 NewReconciliationWorkorderServiceWithCache,旧构造器保留 nil-safe 兼容壳"
  - "GetByWorkstation 构造函数扩展为 3 参数 (db, cache, matcher);SetMatcher 注入 helper"
  - "MatchException 是公开方法,内部复用 package-private matchException;返回 ExceptionMatch 简化版"
  - "IP 解析链 inline 在 reconciliation_service.go 底部(resolveAssetIPChain + fetchWorkstationDeviceIPs 私有函数)"
  - "Drawer 申请例外按钮自包含内联 handleApplyException;父级 onApplyException 变可选(SC9 锁定行为不变)"
  - "ResolveException 严格顺序: service.ResolveException → InvalidateWorkstationHealth → operlog.Record → response.Success"
  - "useCallback 必须在 early return 之前(react-hooks/rules-of-hooks) — Plan 02 修复"
  - "R2 auto-workorder cron 不强制 operlog(系统自动化行为豁免,comment 标注)"
  - "operlog 覆盖范围:用户主动操作(ResolveException + exception-rule CRUD)全量;cron 自动化流程豁免"
metrics:
  duration: ~25min
  completed_date: 2026-06-29
  tasks_completed: 2
  files_modified_count: 13
  test_files_created: 2
  tests_added: 9
  commits: 2
---

# Phase 45 Plan 02: R4 Closure — Cache Invalidation + VERIFICATION + Regression Guard

## One-liner

R2 createWorkorder 缓存主动失效 + ResolveException 严格 success path (service → invalidate → operlog → response) + IP 解析链 inline (asset → workstation → network_device via port → unknown) + 9 个回归守护测试 + VERIFICATION.md 10 SCs 全部 PASS。

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Backend closure — R2 scheduler cache invalidation + IP resolution chain + operlog + ResolveException 顺序 | `09874600` | 9 (8 mod + 1 new doc section) |
| 2 | Frontend tweaks (申请例外 URL params) + VERIFICATION.md + 回归守护测试 | `7c4eed39` | 7 (4 mod + 2 new test + 1 new doc) |

## Files Modified

**Backend (8 files):**
- `internal/scheduler/reconciliation_tasks.go` — `RegisterReconciliationTasks` 新增 cache 参数;`createWorkorderBySeverity` 在 `CreateWorkorderFromException` 成功后调 `WorkstationIDForException` + `InvalidateWorkstationHealth`
- `internal/services/asset/reconciliation_workorder.go` — 新增 `WorkstationIDForException` 旁路方法 + `InvalidateWorkstationHealth` service-level helper;`cache` 字段 + `NewReconciliationWorkorderServiceWithCache` 构造器;`SetCache` 注入 helper
- `internal/services/asset/reconciliation_service.go` — 构造函数扩展为 `(db, cache, matcher)` 3 参数;`GetByWorkstation` 集成 IP 解析链 inline + per-asset `MatchException` 调用;新增 `fetchWorkstationDeviceIPs` + `resolveAssetIPChain` 私有函数
- `internal/services/asset/reconciliation_exception.go` — 新增 `ExceptionMatch` 类型 + `MatchException` method
- `internal/api/v1/asset/reconciliation_handler.go` — `ResolveException` success path 严格顺序 + `applogger` 缓存失效 warn
- `internal/api/v1/asset/reconciliation_router.go` — 注入 `exceptionSvc` 给 `ReconciliationService`
- `internal/api/router.go` — `WorkstationHandler` 跨模块注入同步传 `exceptionSvcForWs`
- `internal/core/core.go` — `RegisterReconciliationTasks` 调用传 `c.Cache`

**Frontend (4 files):**
- `xingran-react-frontend/src/components/reconciliation/ExceptionMatchList.tsx` — 新增 `assetIp` + `conflictType` props;`handleCreateRule` 内联 navigate 携带 2 query params
- `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx` — `onApplyException` 变可选;内联 `handleApplyException` 携带 4 query params;`useCallback` 移至 early return 前
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 移除冗余 `onApplyException` 回调
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — 移除冗余 `onApplyException` 回调

**Documentation (2 files):**
- `.planning/notes/260627-cross-module-permission.md` — 追加 R4 实际接入清单章节(后端 13 + 前端 11 + operlog 3 + 关键决策 6 + R5 移交 4)
- `.planning/phases/45-r4/45-VERIFICATION.md` (NEW) — 10 SCs 全部 PASS + Requirements 3/3 + Convention 14/14 + Memory 0 violations + R4 boundary 11/11

**Test Files (2 new):**
- `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx` (NEW) — 5 tests (visible / hidden / loading / data / empty / click)
- `xingran-react-frontend/src/components/reconciliation/hooks/__tests__/useReconciliationVisibility.test.ts` (NEW) — 4 tests (has perm / empty perms / undefined / wrong perm)

## Verification Results

**Backend:**
- `go build ./...` — exit 0
- `go test ./internal/services/asset/... -count=1 -timeout=120s` — `ok 1.743s`
- `go test ./internal/api/v1/asset/... -count=1 -timeout=120s` — `ok 0.313s`
- `go test ./internal/scheduler/... -count=1 -timeout=120s` — `ok 1.840s`
- `grep -rn "InvalidateWorkstationHealth" internal/ | grep -v _test.go` — 15 occurrences (≥3 处调用)
- `grep -rn "operlog.Record" internal/api/v1/asset/` — 54 occurrences
- `grep -rn "WorkstationIDForException" internal/` — 7 occurrences
- `grep "func.*CreateWorkorderFromException"` — 签名未变 (returns `*models.WorkOrder`)
- `grep "func.*ResolveException"` — service 签名未变 (4-arg returns error)

**Frontend:**
- `cd xingran-react-frontend && npx vitest run src/components/reconciliation` — 9 tests PASS (2 files, ~745ms test time)
- `cd xingran-react-frontend && npx tsc --noEmit` — exit 0
- `cd xingran-react-frontend && npx eslint src/components/reconciliation` — 0 errors 0 warnings
- `cd xingran-react-frontend && npm run build` — `built in 35.45s`

**VERIFICATION.md Status:** 10/10 SCs PASS
- SC1-SC4: PASS (code) / human_needed (UAT browser)
- SC5: PASS (code) / human_needed (perf measurement)
- SC6: PASS (code) / human_needed (UAT)
- SC7: PASS (N+1 optimization verified)
- SC8: PASS (15 Invalidates across 3 sites)
- SC9: PASS (code) / human_needed (UAT URL format)
- SC10: PASS (build + tests all exit 0)

## operlog Coverage Audit

| 写路径 | Module 常量 | OperType | Plan 02 状态 |
|--------|------------|----------|-------------|
| `ReconciliationHandler.ResolveException` | `ModuleReconciliation` ("资产对账") | `OperTypeUpdate` (2) | ✅ Plan 02 补: success path 顺序 + cache invalidate |
| `R2 createWorkorderCritical/High` | (cron 上下文) | (无) | ⏸️ 豁免(系统自动化行为,workorder 后续 lifecycle handler 接手) |
| `exception-rule CRUD` | `ModuleReconciliationExceptionRule` ("资产对账-例外规则") | Create/Update/Delete | ✅ R3 已有,Plan 02 不变 |

**审计结论:** 用户主动操作 100% operlog 覆盖;cron 自动化流程豁免(internal/scheduler/reconciliation_tasks.go:createWorkorderBySeverity 注释已说明)。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `useCallback` 触犯 rules-of-hooks invariant**
- **Found during:** Task 2 lint
- **Issue:** `ReconciliationDrawer.tsx` 把 `useCallback` 放在 `if (!visible) return null` 之后,React 报 "React Hook 'useCallback' is called conditionally"
- **Fix:** 把 `useCallback` 移到所有 early return 之前(放在 `useState` 之后)
- **Files modified:** `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx`
- **Commit:** 7c4eed39

**2. [Rule 1 - Bug] React Compiler `preserve-manual-memoization` 告警**
- **Found during:** Task 2 lint
- **Issue:** `useCallback` 的 deps 含 `asset?.ip` 和 `asset?.conflictType` 链式访问,React Compiler 推断为更广泛的 `asset` 依赖,不匹配显式声明
- **Fix:** deps 简化为 `asset` 整体(对象引用变化触发重渲染)
- **Files modified:** `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx`
- **Commit:** 7c4eed39

**3. [Rule 2 - Missing] test 文件 eslint `no-unsafe-call/no-unsafe-return` warning**
- **Found during:** Task 2 lint
- **Issue:** mock 函数返回 `any` 类型,触发 ESLint unsafe-call/unsafe-return 告警(11 warnings)
- **Fix:** mock 显式类型标注(`as boolean` / `as ReturnType<typeof vi.fn>`);hook import 的 selector 也加显式类型
- **Files modified:** 2 test files
- **Commit:** 7c4eed39

### Plan-Bounded Decisions (B-locks enforced)

- **B1**: `ResolveException` service 签名未变 — 4-arg (ctx, id, userID, note) returns error
- **B2**: `CreateWorkorderFromException` 签名未变 — returns `*models.WorkOrder`
- **B5**: IP 解析链 inline 在 `reconciliation_service.go` 底部 — 不抽新文件

### Notes (informational)

- `R4 实际接入清单` 章节 append 到 `cross-module-permission.md` (后端 13 + 前端 11 + operlog 3 + 关键决策 6 + R5 移交 4)
- `ReconciliationWorkorderService` 新增 `cache` 字段 + `NewReconciliationWorkorderServiceWithCache` 构造器;旧 `NewReconciliationWorkorderService(db, wsHub, noticeSvc)` 保留为 nil-safe 兼容壳
- `ReconciliationExceptionService.MatchException` 是 per-asset 轻量级命中,区别于 `MatchTest` 全规则列表

## Phase 46 Handoff Notes

R4 闭环完整,Phase 46 (R5 半自动修复) 可直接基于以下成果展开:

1. **服务层基础就位**: `ReconciliationService.GetByWorkstation(wsID, window)` 返回 `ByWorkstationResponse{Workstation, HealthScore, Assets, Visible}`;R5 复用同一端点
2. **operlog 路径清晰**: 半自动修复流程 → 写 `ModuleReconciliation` + `OperTypeUpdate/Approve` 即可
3. **缓存失效闭环**: 修复成功后调 `ReconciliationWorkorderService.InvalidateWorkstationHealth(ctx, wsID)` 即可让用户重看页面立即看到新数据
4. **跨模块权限不变**: R5 新增修复 action 需更新 `.planning/notes/260627-cross-module-permission.md` 权限矩阵
5. **回归测试就位**: 9 个 vitest 测试守护核心 hook + 组件行为;R5 新增组件/hook 应在同目录下补齐测试
6. **VERIFICATION 模板**: `.planning/phases/45-r4/45-VERIFICATION.md` 可作 R5 验证模板

## B2 Invariants Verification

```bash
$ grep "func.*CreateWorkorderFromException" internal/services/asset/reconciliation_workorder.go
func (s *ReconciliationWorkorderService) CreateWorkorderFromException(ctx context.Context, exceptionID string) (*models.WorkOrder, error) {
  # 签名未变 ✓

$ grep "func.*ResolveException" internal/services/asset/reconciliation_service.go
func (s *reconciliationServiceImpl) ResolveException(ctx context.Context, id string, userID string, note *string) error {
  # 签名未变 ✓
```

## Commit Hashes

| Hash | Message |
|------|---------|
| `09874600` | feat(45-02): backend closure - R2 cache invalidation + IP resolution chain + operlog |
| `7c4eed39` | feat(45-02): frontend - apply exception URL params + VERIFICATION + regression tests |

## Self-Check

- [x] All 13 files exist and modified
- [x] Both commits exist in git log
- [x] VERIFICATION.md contains 10 SC entries (grep -c "^### SC" = 10)
- [x] Backend tests PASS (asset + api + scheduler)
- [x] Frontend tests PASS (9/9)
- [x] ESLint clean
- [x] TypeScript clean
- [x] Build clean (both backend and frontend)
- [x] B2 invariants preserved
- [x] operlog coverage 100% for user-initiated writes

---

*Plan 45-02 complete — R4 closure deliverable ships for Phase 46 (R5) handoff.*
