---
phase: 45-r4
plan: 01
subsystem: asset-reconciliation
tags: [INTEGRATE-01, INTEGRATE-02, INTEGRATE-03, R4, health-card, health-badge, drawer, cross-module-permission, n+1-fix]
dependency_graph:
  requires: []
  provides:
    - "POST /asset/reconciliation/by-workstation — 工位对账健康度聚合 API (5 KPI + assets + visible flag)"
    - "pkg/middleware.HasUserPermission — query-style permission check (no Abort)"
    - "Workstation.GetByID 注入 reconciliationVisible + reconciliationHiddenReason"
    - "ReconciliationService.InvalidateWorkstationHealth — 缓存失效 helper (Plan 02 调用)"
    - "HealthCard / HealthBadge / ReconciliationDrawer — R4 UI 整合组件"
    - "useWorkstationHealth / useAssetHealth — N+1 修复 (lift state, 单 query 服务 5 维 + 资产徽标)"
  affects:
    - "internal/services/asset/reconciliation_service.go (constructor: db + cache)"
    - "internal/api/router.go (WorkstationHandler cross-module wiring)"
    - "xingran-react-frontend/src/lib/assetApi.ts (byWorkstation + types)"
    - "xingran-react-frontend/src/lib/queryKeys.ts (workstationHealth + assetHealth)"
    - "xingran-react-frontend/src/pages/operations/workstations/index.tsx (HealthCard + lift state)"
    - "xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx (HealthBadge column)"
    - "xingran-react-frontend/src/pages/operations/assets/index.tsx (HealthBadge column + drawer)"
tech-stack:
  added: []
  patterns:
    - "Cross-module service injection (WorkstationHandler.WithReconciliationService)"
    - "Query-style permission helper (HasUserPermission) for silent degradation"
    - "Weak-typed virtual fields (map[string]interface{}) to break import cycle models→asset"
    - "Lift state for N+1 fix (parent useWorkstationHealth → Map<assetId, conflictType>)"
    - "Defense-in-depth enabled gate (visible + workstationId + data?.visible)"
    - "JSON type conversion at handler boundary (asset.ByWorkstationResponse → map[string]interface{})"
key-files:
  created:
    - pkg/middleware/permission_query_helper.go
    - internal/api/v1/operations/workstation_handler_test.go
    - xingran-react-frontend/src/components/reconciliation/HealthCard.tsx
    - xingran-react-frontend/src/components/reconciliation/HealthBadge.tsx
    - xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx
    - xingran-react-frontend/src/components/reconciliation/ReconciliationTimeline.tsx
    - xingran-react-frontend/src/components/reconciliation/ExceptionMatchList.tsx
    - xingran-react-frontend/src/components/reconciliation/index.ts
    - xingran-react-frontend/src/components/reconciliation/hooks/useReconciliationVisibility.ts
    - xingran-react-frontend/src/components/reconciliation/hooks/useWorkstationHealth.ts
    - xingran-react-frontend/src/components/reconciliation/hooks/useAssetHealth.ts
    - xingran-react-frontend/src/components/reconciliation/hooks/useExceptionMatch.ts
  modified:
    - internal/services/asset/reconciliation_service.go
    - internal/services/asset/cache_keys.go
    - internal/api/v1/asset/reconciliation_router.go
    - internal/api/v1/asset/reconciliation_handler.go
    - internal/api/v1/operations/workstation_handler.go
    - internal/api/router.go
    - internal/models/workstation.go
    - xingran-react-frontend/src/lib/queryKeys.ts
    - xingran-react-frontend/src/lib/assetApi.ts
    - xingran-react-frontend/src/pages/operations/workstations/index.tsx
    - xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx
    - xingran-react-frontend/src/pages/operations/assets/index.tsx
decisions:
  - "Workstation.Reconciliation 使用 map[string]interface{} 弱类型避免 models→asset→models 循环依赖(handler 边界做 json marshal/unmarshal 类型转换)"
  - "ReconciliationService 构造函数改为 (db, cache) 注入支持 5min TTL 缓存"
  - "HasUserPermission(c, core, perm) — 接受 core 显式参数(因 gin context 不持有 core);不调 c.Abort()"
  - "HealthCard 仅在 useReconciliationVisibility()===true 时渲染,reconciliationVisible 由 page 顶层读 menuStore"
  - "Drawer state lift 到 page 顶层(workstations/index.tsx + assets/index.tsx 各一处),WorkstationDeviceTable 接收 conflictTypeMap prop(N+1 修复)"
  - "useWorkstationHealth 不在组件内 enabled gate 引用 backend visible 字段(简化)— 仅前端 visible + workstationId 两段 gate;后端 visible 字段由 drawer/reconciliation 数据流保护"
  - "ReconciliationTimeline/ExceptionMatchList 在 Plan 01 留占位 interface,Plan 02 接入实际查询"
  - "Plan 01 不调用 InvalidateWorkstationHealth — 仅 cache_keys.go 定义,Plan 02 在 ResolveException + R2 scheduler 中调用"
metrics:
  duration: ~50min
  completed_date: 2026-06-29
  tasks_completed: 2
  files_modified_count: 21
---

# Phase 45 Plan 01: R4 Integration — Workstation Health + Asset Drawer Summary

## One-liner

工位 expand 顶部嵌入 HealthCard(5 KPI + trend + score)+ 设备子表/资产列表行内 HealthBadge(8px dot,6 色 A-F)+ 跨模块 HasUserPermission 静默降级 + 后端 by-workstation 聚合 API(COUNT FILTER 严格走 DB,无 list.length)。

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Backend — permission helper + ReconciliationService.GetByWorkstation + 跨模块注入 | `b569a628` | 9 (8 mod + 1 test stub) |
| 2 | Frontend — 4 components + 4 hooks + barrel + 整合点 | `4d420f05` | 12 (8 new + 4 mod) |

## Files Modified

**Backend (9 files):**
- `pkg/middleware/permission_query_helper.go` (NEW) — `HasUserPermission(c, core, perm) bool` — 复用 `getUserIDAsString` + `isSuperAdmin` + `checkUserPermission` 链路,无 `c.Abort()` 静默降级
- `internal/services/asset/reconciliation_service.go` — `NewReconciliationService(db, cache)` 签名变更(1→2 参数);新增 `GetByWorkstation(ctx, wsID, window) (*ByWorkstationResponse, error)`;新增 `WorkstationBrief` / `HealthScore` / `AssetHealthItem` / `ByWorkstationResponse` 类型;`computeByWorkstation` 内联 7 步聚合(资产列表 + 异常分桶 + exceptionHit + trend + score + assets 徽标 + IP 解析链)
- `internal/services/asset/cache_keys.go` — 新增 `InvalidateWorkstationHealth(ctx, c, wsID) error` helper(named export,Plan 02 调)
- `internal/api/v1/asset/reconciliation_router.go` — `POST /by-workstation` 路由 + 注入 `core.Cache`
- `internal/api/v1/asset/reconciliation_handler.go` — `GetByWorkstation` handler + `hasReconciliationPerm` 私有方法 + Visible 字段注入
- `internal/api/v1/operations/workstation_handler.go` — `WithReconciliationService` 链式 setter + `GetByID` 调 `hasReconciliationPerm` 门控(无权限时 `ReconciliationVisible=false` + `ReconciliationHiddenReason="无资产对账查看权限"`)
- `internal/api/router.go` — WorkstationHandler 链式注入 `reconciliationSvc`
- `internal/models/workstation.go` — `Reconciliation`/`ReconciliationVisible`/`ReconciliationHiddenReason` 虚拟字段(`gorm:"->;-:migration"` + `map[string]interface{}` 弱类型破循环)
- `internal/api/v1/operations/workstation_handler_test.go` (NEW) — 3 测试 stub:无 reconciliationSvc / 有 reconciliationSvc(无 user_id) / hasReconciliationPerm 无 user_id

**Frontend (12 files):**
- `src/lib/queryKeys.ts` — `workstationHealth(wsId)` + `assetHealth(assetId)` factory
- `src/lib/assetApi.ts` — `ByWorkstationResponse` / `HealthScore` / `AssetHealthItem` 类型 + `reconciliationApi.byWorkstation({workstationId, window})`
- `src/components/reconciliation/HealthCard.tsx` (NEW) — antd Card 5 KPI grid + 趋势 ECharts(56px sparkline) + score Statistic
- `src/components/reconciliation/HealthBadge.tsx` (NEW) — 8px 圆点 + Tooltip(mouseEnterDelay=1)+ useDict 颜色映射 + role/tabIndex a11y
- `src/components/reconciliation/ReconciliationDrawer.tsx` (NEW) — 780px 宽 + 3 Tabs(冲突摘要/历史变更/例外规则)+ 申请例外 extra 按钮
- `src/components/reconciliation/ReconciliationTimeline.tsx` (NEW) — antd Timeline read-only(Plan 02 接入数据)
- `src/components/reconciliation/ExceptionMatchList.tsx` (NEW) — antd List + actionTagColor(从 R3 复制)
- `src/components/reconciliation/index.ts` (NEW) — barrel 导出
- `src/components/reconciliation/hooks/useReconciliationVisibility.ts` (NEW) — 读 `useMenuStore.permissions`(B4 修复)
- `src/components/reconciliation/hooks/useWorkstationHealth.ts` (NEW) — 5min staleTime + 10min gcTime + enabled gate
- `src/components/reconciliation/hooks/useAssetHealth.ts` (NEW) — 从 `useWorkstationHealth` cache 切片(无 N+1)
- `src/components/reconciliation/hooks/useExceptionMatch.ts` (NEW) — R3 模式复用
- `src/pages/operations/workstations/index.tsx` — Lift `useWorkstationHealth` 到 page + `assetConflictMap` 派生 + HealthCard 嵌入 expand + Drawer at page level
- `src/components/operations/WorkstationDeviceTable/index.tsx` — `conflictTypeMap` + `onBadgeClick` props + "对账健康" 列
- `src/pages/operations/assets/index.tsx` — "对账健康" 列 + Drawer at page level

## Verification Results

**Backend:**
- `go build ./...` — exit 0
- `go test ./internal/services/asset/... -count=1` — PASS
- `go test ./internal/api/v1/asset/... -count=1` — PASS
- `go test ./pkg/middleware/... -count=1` — PASS
- `go test ./internal/api/v1/operations/... -count=1 -run Workstation` — PASS (3 tests, 0.201s)
- `grep "PermissionService.UserHasPerm" internal/ pkg/ | grep -v _test.go` — 0 matches (锁住 plan's MUST-HAVE)
- `grep "by-workstation" internal/api/v1/asset/reconciliation_router.go` — present (line 23 + line 38 route)

**Frontend:**
- `cd xingran-react-frontend && npm run build` — exit 0 (built in 34.98s, no type errors)
- `cd xingran-react-frontend && npm run lint` (reconciliation dir only) — 0 errors / 0 warnings
- `grep "authStore.*perms\|state.perms" xingran-react-frontend/src/components/reconciliation/` — 1 match (comment explaining the B4 fix, not actual usage)
- `grep "useMenuStore.*permissions" xingran-react-frontend/src/components/reconciliation/hooks/useReconciliationVisibility.ts` — 1 match (line 19)
- `grep "<HealthCard" xingran-react-frontend/src/pages/operations/workstations/index.tsx` — present (line 581)
- `grep "<HealthBadge" WorkstationDeviceTable assets` — present (WorkstationDeviceTable line 340, assets line 445)
- `grep "conflictTypeMap" workstations/index.tsx WorkstationDeviceTable/index.tsx` — present (N+1 lift wiring intact)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] ReconciliationService signature change broke import cycle**
- **Found during:** Task 1 backend build
- **Issue:** `models → asset → models` 循环依赖(因为 Workstation model 引用 `asset.ByWorkstationResponse`)
- **Fix:** 改用 `Reconciliation map[string]interface{}` 弱类型(plan §"models/workstation.go" 已列为备选方案);handler 边界用 `json.Marshal/Unmarshal` 做类型转换
- **Files modified:** `internal/models/workstation.go`, `internal/api/v1/operations/workstation_handler.go`
- **Commit:** b569a628

**2. [Rule 1 - Bug] HasUserPermission signature 需要 core 注入**
- **Found during:** Task 1 — 现有 `checkUserPermission(core, userID, perm)` 需要 `*core.Core`,而 gin context 不持有 core
- **Fix:** 签名改为 `HasUserPermission(c *gin.Context, core *core.Core, perm string) bool`,handler 调用时显式传 `h.core`;不调 `c.Abort()` 保持静默降级语义
- **Files modified:** `pkg/middleware/permission_query_helper.go`, `internal/api/v1/asset/reconciliation_handler.go`, `internal/api/v1/operations/workstation_handler.go`
- **Commit:** b569a628

**3. [Rule 2 - Missing] WorkstationService stub test 签名校准**
- **Found during:** Task 1 — `stubWorkstationService` 起初用了不正确的接口签名
- **Fix:** 对齐 `opsServices.PositionUpdateItem` / `[]opsServices.DeptOption` / `*opsServices.PageResult` / `*opsServices.WorkstationStatisticsResult`,stub 通过编译
- **Files modified:** `internal/api/v1/operations/workstation_handler_test.go`
- **Commit:** b569a628

**4. [Rule 1 - Bug] HealthCard useMemo 顺序问题**
- **Found during:** Task 2 lint
- **Issue:** `useMemo` 写在 early return 之后,违反 rules-of-hooks
- **Fix:** 把 `useMemo` 移到所有 early return 之前
- **Files modified:** `xingran-react-frontend/src/components/reconciliation/HealthCard.tsx`
- **Commit:** 4d420f05

**5. [Rule 1 - Bug] ReconciliationDrawer useEffect setState 触发 cascading renders**
- **Found during:** Task 2 lint
- **Issue:** `setTimelineLoading(true)` 同步在 useEffect 内,触发 react-hooks/set-state-in-effect 告警
- **Fix:** Plan 01 不实际触发 fetch(留给 Plan 02 接入),只保留 local state 字段
- **Files modified:** `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx`
- **Commit:** 4d420f05

**6. [Rule 1 - Bug] Workstation 4 步 fit 卡片 not 真实数据**
- **Found during:** Task 2 — `if (false) setTimelineLoading(true)` 死代码触发 no-unused-vars
- **Fix:** 完全删除占位 setState 调用,改用 `useState(false)` 固定值,Plan 02 接入时再启用
- **Files modified:** `xingran-react-frontend/src/components/reconciliation/ReconciliationDrawer.tsx`
- **Commit:** 4d420f05

### Plan-Bounded Decisions (out of scope per must_haves)

- Plan 01 不修改 `internal/api/v1/asset/reconciliation_exception_handler.go`(该文件无 ResolveException,锁定 B1)
- Plan 01 不调用 `InvalidateWorkstationHealth`(锁定 B2;Plan 02 调)
- Plan 01 不实现 IP 解析链的新文件(B5:inline 在 `computeByWorkstation` 内)
- IP CIDR → 例外规则命中(Plan 02 接入实时查询)

## Known Issues for Plan 02

1. **ReconciliationTimeline 实际数据查询**:Plan 01 仅固定组件契约,Plan 02 接入 `sys_data_reconciliation` WHERE `resolved_at IS NOT NULL` ORDER BY `resolved_at DESC` 查询
2. **ExceptionMatchList 实际数据查询**:Plan 01 仅固定组件契约,Plan 02 接入 `sys_reconciliation_exception` 命中 IP CIDR 规则
3. **ResolveException 主动缓存失效**:`InvalidateWorkstationHealth` 在 `reconciliation_handler.go:ResolveException` success path 调用(锁定 D-A4-04 + B1/B2)
4. **R2 scheduler 主动缓存失效**:workorder 创建后调 `InvalidateWorkstationHealth`
5. **`WorkstationService.GetByID` 是否也应缓存 reconciliation 数据**:Plan 01 在 `WorkstationHandler.GetByID` 调 `reconciliationSvc.GetByWorkstation`(已走 5min 缓存);Plan 02 评估是否需要单独失效工位详情缓存
6. **`reconciliationApi.exceptionRule.test` 是否在 ExceptionMatchList 真正使用**:Plan 01 留 `useExceptionMatch` hook 备用,Plan 02 决定是否用它替代预加载规则
7. **健康度得分公式(简单比 vs 加权)**:Plan 01 用 D-A2-03 锁定的简单比;Plan 02 评估是否在 dashboard 改权重公式
8. **Asset type 缺 workstationId 字段**:Plan 01 在 assets page 用 `workstationId: null` 占位,Plan 02 决定是否扩展 Asset type + 后端 list 返回 workstationId
9. **operlog.Record 不在 GetByWorkstation 调用**:Plan 01 锁定 GetByWorkstation 是 read 路径(无 operlog);但 Plan 02 应审计"打开抽屉"是否算用户操作(per Phase 34 全模块覆盖)

## Notes for Plan 02

- T-45-05 / T-45-10 mitigations 仍在 Plan 02 scope,`InvalidateWorkstationHealth` helper 已就位
- `asset.InvalidateWorkstationHealth(ctx, c, wsID)` 已在 cache_keys.go,nil-safe,可直接调
- 后端 `ReconciliationService.ResolveException` 签名可能需要扩展(返回 workstationID 用于 cache 失效)— Plan 02 评估
- Phase 34 operlog 全模块覆盖审计可叠加在 Plan 02 Task 1
