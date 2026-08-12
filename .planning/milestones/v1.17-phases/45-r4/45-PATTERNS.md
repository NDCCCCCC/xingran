# Phase 45: 工位详情整合 + 资产详情摘要 (R4) - Pattern Map

**Mapped:** 2026-06-28
**Files analyzed:** 16 new/modified
**Analogs found:** 11 / 16 (5 partial — frontend hooks require new component dir)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/asset/reconciliation_service.go` (modify — add `GetByWorkstation`) | service | CRUD/aggregation | existing `reconciliationServiceImpl.ListExceptions` | exact |
| `internal/services/asset/cache_keys.go` (modify — add `invalidate_workstation_health`) | utility | cache invalidation | existing `GetReconciliationHealthByWorkstationKey` helper | exact |
| `internal/api/v1/asset/reconciliation_router.go` (modify — add `/by-workstation` route) | route | request-response | existing `SetupReconciliationRouter` in same file | exact |
| `internal/api/v1/asset/reconciliation_handler.go` (modify — add `GetByWorkstation` handler) | handler | request-response | existing `ReconciliationHandler.ListExceptions` | exact |
| `internal/api/v1/asset/reconciliation_exception_handler.go` (modify — R2 `ResolveException` calls `invalidate_workstation_health`) | handler | request-response | existing `ResolveException` flow | exact |
| `internal/api/v1/operations/workstation_handler.go` (modify — inject `ReconciliationService`, `hasReconciliationPerm`) | handler | request-response + cross-module | existing `WorkstationHandler.GetByID` + `ADAccountHandler` cross-module injection | role-match |
| `src/components/reconciliation/HealthCard.tsx` (new) | component | transform + render | `useDashboard` aggregation pattern (R1 Dashboard) | role-match |
| `src/components/reconciliation/HealthBadge.tsx` (new) | component | render | existing Tag color from list_class in `exception-rules/index.tsx` | role-match |
| `src/components/reconciliation/ReconciliationDrawer.tsx` (new) | component | transform + render | existing 780-px Drawer in `exception-rules/index.tsx` + `MatchTestPanel` embedded | exact |
| `src/components/reconciliation/ReconciliationTimeline.tsx` (new) | component | render | antd `Timeline` first use in R4 | no-analog (no existing Timeline) |
| `src/components/reconciliation/ExceptionMatchList.tsx` (new) | component | render | existing `MatchTestPanel` rule table + `actionTagColor` helper | exact |
| `src/components/reconciliation/hooks/useReconciliationVisibility.ts` (new) | hook | auth check | `useDict` (existing hook w/ perm gating reference) | role-match |
| `src/components/reconciliation/hooks/useWorkstationHealth.ts` (new) | hook | React Query | `useDashboard` hook (parallel useQuery pattern) | exact |
| `src/components/reconciliation/hooks/useAssetHealth.ts` (new) | hook | selector over cache | `useDashboard` slice selectors | role-match |
| `src/components/reconciliation/hooks/useExceptionMatch.ts` (new) | hook | React Query | existing `queryKeys.reconciliation.matchTest` + `reconciliationApi.exceptionRule.test` | exact |
| `src/lib/queryKeys.ts` (modify — add `workstationHealth` + `assetHealth` factories) | utility | registry | existing `reconciliation.*` factories | exact |
| `src/lib/assetApi.ts` (modify — add `byWorkstation` method) | service wrapper | API | existing `reconciliationApi` factory methods | exact |
| `src/components/operations/WorkstationDeviceTable/index.tsx` (modify — add HealthBadge column to AD/asset sub-tables) | component | render | existing `createColumns(canEdit)` factory | role-match |
| `src/pages/operations/workstations/views/CardView.tsx` (modify — add HealthCard to expand top) | component | render | existing expand pattern in `pages/operations/workstations/index.tsx` | role-match |

---

## Pattern Assignments

### `internal/services/asset/reconciliation_service.go` — add `GetByWorkstation` (service, CRUD + cache)

**Analog:** existing `ReconciliationService` interface at lines 128-153 of the same file.

**Interface extension pattern** (line 128-153):
```go
type ReconciliationService interface {
    ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error)
    GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error)
    ResolveException(ctx context.Context, id string, userID string, note *string) error
    // 🆕 R4:
    GetByWorkstation(ctx context.Context, wsID string, window string) (*ByWorkstationResponse, error)
}
```

**Constructor pattern** (lines 169-171):
```go
func NewReconciliationService(db *gorm.DB) ReconciliationService {
    return &reconciliationServiceImpl{db: db, mvExists: -1}
}
```
**R4 extension**: switch to `NewReconciliationService(db *gorm.DB, cache cache.Cache)` to enable `CacheProvider.GetOrSet(...)` (TTL 5min matches R1 MV refresh per D-A4-03).

**Query implementation pattern** — mirror `reconciliation_statistics.go:Summary()`:
- Use GORM aggregate query `SELECT COUNT(*) FILTER (WHERE ...)` / `GROUP BY conflict_type` (CLAUDE.md + project memory `stat-cards-from-list-length-capped-at-100` hard rule).
- LEFT JOIN `reconciliation_normalized` MV on `workstation_id` for asset enumeration.
- Trend: same `date_trunc("day", detected_at)` shape as `reconciliation_statistics.go:HealthTrend()`.

**Service struct extension (R4)**:
```go
type reconciliationServiceImpl struct {
    db         *gorm.DB
    mvExists   int32
    mvExistsMux sync.Mutex
    cache      cache.Cache // 🆕 for GetOrSet
}
```

---

### `internal/services/asset/cache_keys.go` — add `invalidate_workstation_health` (utility, cache invalidation)

**Analog:** existing helpers in same file (lines 82-90).

**Helper pattern** (lines 82-85):
```go
func GetReconciliationHealthByWorkstationKey(workstationID string) string {
    return fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, workstationID)
}
```

**R4 addition** (append to file):
```go
// invalidate_workstation_health 删除该工位的健康度缓存(D-A4-04)
//
// R2 转单 + R2 resolve 完成后调用,让用户重看页面立即看到变化。
// 调用顺序:invalidate → operlog.Record → response.Success。
//
// 复用现有 cache.Delete + StripCachePrefix 模式(本文件 lines 92-107):
//   - 缓存写入时自动加 `xingran:` 前缀
//   - 调用 cache.Delete(ctx, GetReconciliationHealthByWorkstationKey(wsID)) 即可
func invalidate_workstation_health(ctx context.Context, c cache.Cache, workstationID string) error {
    if c == nil || workstationID == "" {
        return nil
    }
    return c.Delete(ctx, GetReconciliationHealthByWorkstationKey(workstationID))
}
```

**Note**: `cache_keys.go` currently has no `cache.Cache` import. R4 plan must add `import "github.com/xingran-next/xingran-go-backend/pkg/cache"` — minimal surface change.

---

### `internal/api/v1/asset/reconciliation_router.go` — add `/by-workstation` (route, request-response)

**Analog:** existing routes in same file (lines 24-31).

**Route registration pattern**:
```go
func SetupReconciliationRouter(r *gin.RouterGroup, core *core.Core) {
    svc := asset.NewReconciliationService(core.DB.GetDB())
    handler := NewReconciliationHandler(svc).WithCore(core)

    r.POST("/exception/list", handler.ListExceptions)
    r.POST("/exception/:id", handler.GetExceptionByID)
    r.POST("/exception/:id/resolve", handler.ResolveException)
    // 🆕 R4:
    r.POST("/by-workstation", handler.GetByWorkstation)
}
```

**Route positioning rule** (CLAUDE.md "Route Pattern" — `POST /list`, `POST /:id`, `POST /:id/update`):
- `/by-workstation` is a named segment (NOT `:id`) — no conflict with existing `/exception/:id`.
- Place after `/exception/:id/resolve` for consistent reading order.

**Constructor note**: Service signature changes from `NewReconciliationService(db)` → `NewReconciliationService(db, core.Cache)`. Update all 3 call sites in `reconciliation_router.go` + `reconciliation_exception_handler.go` constructor call (router wiring) + any test factory.

---

### `internal/api/v1/asset/reconciliation_handler.go` — add `GetByWorkstation` handler

**Analog:** existing `ListExceptions` (lines 52-65) + `GetExceptionByID` (lines 68-85) + `ResolveException` (lines 110+).

**Standard read handler pattern** (lines 52-65):
```go
func (h *ReconciliationHandler) ListExceptions(c *gin.Context) {
    var params asset.ExceptionListParams
    if err := c.ShouldBindJSON(&params); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }
    result, err := h.service.ListExceptions(c.Request.Context(), &params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

**R4 addition** (D-A4-01/02):
```go
// GetByWorkstation 工位对账健康度聚合(D-A4-01/02)
//
// 读操作 — 不调 operlog.Record(参照 ListExceptions 模式)。
//
// 入参:{"workstationId": "uuid", "window": "7d"}
// 出参:ByWorkstationResponse{Workstation, HealthScore, Assets, Visible}
func (h *ReconciliationHandler) GetByWorkstation(c *gin.Context) {
    var req struct {
        WorkstationID string `json:"workstationId" binding:"required"`
        Window        string `json:"window"` // 默认 "7d"
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }
    if req.Window == "" {
        req.Window = "7d"
    }
    result, err := h.service.GetByWorkstation(c.Request.Context(), req.WorkstationID, req.Window)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}
```

**operlog note**: handler is a read endpoint — NO `operlog.Record` call. Mirror `ListExceptions` exactly.

---

### `internal/api/v1/asset/reconciliation_exception_handler.go` — R2 `ResolveException` adds `invalidate_workstation_health` (handler modify)

**Analog:** existing `ResolveException` flow (reconciliation_handler.go:110+).

**Current `ResolveException` body** (after `service.ResolveException` returns nil):
```go
// operlog 写入(CLAUDE.md 强制约定,标记已解决 → OperTypeUpdate)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)
response.Success(c, gin.H{"id": id})
```

**R4 modification** (D-A4-04, invalidates BOTH `CacheKeyReconciliationHealthByWorkstation` + `CacheKeyReconciliationHealthByAsset`):
```go
if err := h.service.ResolveException(...); err != nil {
    response.Error(c, http.StatusInternalServerError, err.Error())
    return
}

// 🆕 R4 / D-A4-04: 缓存主动失效,先于 operlog 避免用户重看仍命中旧缓存
// 工位 ID 来自 sys_data_reconciliation.asset_id → reconciliation_normalized.workstation_id → resolve handler 需要先 SELECT assetWorkstationID
// 简化:同时失效该资产所属工位的健康度缓存(由 service 层 ResolveException 内部完成 lookup + invalidate)

// operlog 写入(强制约定,resolve → OperTypeUpdate)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)
response.Success(c, gin.H{"id": id})
```

**Critical sequencing** (D-A4-04 spec): invalidate → operlog.Record → response.Success. Resolve the workstationID lookup INSIDE `service.ResolveException` (returns it alongside the resolved record) — keeps handler thin.

**Service signature change**:
```go
// Old:
ResolveException(ctx, id, userID, note) error
// New:
ResolveException(ctx, id, userID, note) (workstationID string, err error)
```

The returned `workstationID` flows to handler for cache invalidation.

---

### `internal/api/v1/operations/workstation_handler.go` — inject `ReconciliationService` + `hasReconciliationPerm` (handler modify, cross-module)

**Analog A:** existing `WorkstationHandler` constructor (lines 15-30) + `ADAccountHandler` cross-module service injection (`internal/api/v1/system/ad_account_handler.go:28-36`).

**Analog B:** `cross-module-permission.md §2.3` explicit pseudocode for `hasReconciliationPerm` + `ReconciliationVisible`.

**Constructor pattern** (`workstation_handler.go:15-30`):
```go
type WorkstationHandler struct {
    service opsServices.WorkstationService
    core    *core.Core
}

func NewWorkstationHandler(service opsServices.WorkstationService) *WorkstationHandler {
    return &WorkstationHandler{service: service}
}

func (h *WorkstationHandler) WithCore(core *core.Core) *WorkstationHandler {
    if h != nil {
        h.core = core
    }
    return h
}
```

**Cross-module extension (R4)** — mirror `ADAccountHandler` (ad_account_handler.go:28-36):
```go
type WorkstationHandler struct {
    service             opsServices.WorkstationService
    reconciliationSvc   asset.ReconciliationService  // 🆕 R4 cross-module
    core                *core.Core
}

// Chainable setter (一致 WithXxx 模式)
func (h *WorkstationHandler) WithReconciliationService(svc asset.ReconciliationService) *WorkstationHandler {
    if h != nil {
        h.reconciliationSvc = svc
    }
    return h
}
```

**Existing `GetByID`** (`workstation_handler.go:131-140`):
```go
func (h *WorkstationHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    workstation, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        response.Error(c, apperrors.WorkstationNotFound())
        return
    }
    response.Success(c, workstation)
}
```

**R4 modification** (D-A1-03 + cross-module-permission.md §2.3):
```go
func (h *WorkstationHandler) GetByID(c *gin.Context) {
    ctx := c.Request.Context()
    id := c.Param("id")
    workstation, err := h.service.GetByID(ctx, id)
    if err != nil {
        response.Error(c, apperrors.WorkstationNotFound())
        return
    }

    // 🆕 R4 跨模块调用 + 权限降级(D-A1-03)
    if h.hasReconciliationPerm(c) {
        if recon, err := h.reconciliationSvc.GetByWorkstation(ctx, id, "7d"); err == nil {
            workstation.Reconciliation = recon
            workstation.ReconciliationVisible = true
        } else {
            workstation.ReconciliationVisible = false
        }
    } else {
        workstation.ReconciliationVisible = false
        workstation.ReconciliationHiddenReason = "无资产对账查看权限"
    }
    response.Success(c, workstation)
}

// hasReconciliationPerm 内部权限检查(per cross-module-permission.md §2.3)
func (h *WorkstationHandler) hasReconciliationPerm(c *gin.Context) bool {
    userID, _ := c.Get("userID")
    return h.core.PermissionService.UserHasPerm(userID, "asset:reconciliation:list")
}
```

**Workstation model extension** (new fields on `models.Workstation`):
```go
type Workstation struct {
    // ... existing fields
    Reconciliation         *asset.ByWorkstationResponse `json:"reconciliation,omitempty"`
    ReconciliationVisible  bool                          `json:"reconciliationVisible"`
    ReconciliationHiddenReason string                     `json:"reconciliationHiddenReason,omitempty"`
}
```

**Router wiring update** (`internal/api/router.go:580-606`):
```go
workstationService := opsServices.NewWorkstationService(core.DB.GetDB())
reconciliationSvc := asset.NewReconciliationService(core.DB.GetDB(), core.Cache)  // 🆕
workstationHandler := operations.NewWorkstationHandler(workstationService).
    WithCore(core).
    WithReconciliationService(reconciliationSvc)  // 🆕
```

---

### `src/components/reconciliation/HealthCard.tsx` (new component, render + transform)

**Analog:** R1 Dashboard KPI grid (`pages/asset/reconciliation/dashboard/index.tsx:34-46, 88-114`) — uses `useDashboard` parallel `useQuery` + `Statistic` antd component.

**Statistic card pattern** (dashboard/index.tsx — `Statistic title=... value=...`):
```tsx
<Card>
  <Statistic title="总规则数" value={stats.total} />
</Card>
```

**Score coloring** (useMemo over thresholds, D-A2-03 + UI-SPEC score band table):
```tsx
const scoreBandColor = useMemo(() => {
  if (score >= 80) return "#22c55e";
  if (score >= 60) return "#f59e0b";
  return "#ef4444";
}, [score]);
```

**Loading/Empty/Error states** (UI-SPEC, mirroring existing `Skeleton active paragraph rows:2` in `MatchTestPanel.tsx:80+`):
```tsx
if (isLoading) return <Skeleton active paragraph={{ rows: 2 }} />;
if (isError) return <Result status="error" title="健康度加载失败" extra={<Button onClick={refetch}>重试</Button>} />;
if (data?.healthScore?.total === 0) return <Empty description="该工位暂无关联资产。" />;
```

**Mini chart** — copy pattern from `pages/asset/reconciliation/dashboard/index.tsx` (uses `@/components/charts/EChartsWrapper`):
```tsx
import ReactECharts from "@/components/charts/EChartsWrapper";

<ReactECharts
  option={sparklineOption}  // single line series, no axes, no legend
  style={{ height: 56, width: "100%" }}
  opts={{ renderer: "svg" }}
/>
```

**Permission gate** (D-A1-03):
```tsx
const visible = useReconciliationVisibility();
if (!visible) return null;
```

**useEffect deps stability** (CLAUDE.md hard rule):
- `useWorkstationHealth(workstationId)` — `workstationId` is the ONLY dep (primitive string).
- No inline-object deps.

**Wrap with `React.memo`** — props `{ workstationId: string; onApplyException: () => void }`.

---

### `src/components/reconciliation/HealthBadge.tsx` (new component, render)

**Analog:** Tag color from `list_class` in `pages/asset/asset/reconciliation/exception-rules/index.tsx:248-258` (similar dotted badge pattern).

**Color mapping from useDict** (D-A2-04 + UI-SPEC conflict type color table):
```tsx
const { data: conflictTypeDict } = useDict("asset_reconciliation_conflict_type");

const listClassToHex: Record<string, string> = {
  success: "#22c55e",
  warning: "#f59e0b",
  error: "#ef4444",
  default: "#d4d4d8",
  processing: "#3b82f6",
};

// 6 type keys (A-F) → list_class resolution:
const colorForConflict = (conflictType: string | null): string => {
  if (conflictType === null) return "#22c55e";  // 健康
  const entry = conflictTypeDict?.find((d) => d.dictValue === conflictType);
  const listClass = entry?.listClass ?? "default";
  return listClassToHex[listClass] ?? "#d4d4d8";
};
```

**Hidden state (D-A1-03)**:
```tsx
const visible = useReconciliationVisibility();
if (!visible) return <>{"-"}</>;  // 单一占位 dash,保持列宽一致
```

**Tooltip wrapper** (antd `Tooltip` w/ `mouseEnterDelay={1}`):
```tsx
<Tooltip title={CONFLICT_TYPE_TOOLTIPS[conflictType]} mouseEnterDelay={1}>
  <span
    role={conflictType ? "button" : "img"}
    tabIndex={conflictType ? 0 : -1}
    aria-label={conflictType ? CONFLICT_TYPE_TOOLTIPS[conflictType] : "健康"}
    onClick={conflictType ? () => onClick(assetId, conflictType) : undefined}
    onKeyDown={conflictType ? (e) => { if (e.key === "Enter") onClick(assetId, conflictType); } : undefined}
    style={{
      display: "inline-block",
      width: 8, height: 8, borderRadius: "50%",
      backgroundColor: colorForConflict(conflictType),
      cursor: conflictType ? "pointer" : "default",
    }}
  />
</Tooltip>
```

**CONFLICT_TYPE_TOOLTIPS** — constant object keyed by `"A"` ~ `"F"` (verbatim from UI-SPEC).

**Wrap with `React.memo`**.

---

### `src/components/reconciliation/ReconciliationDrawer.tsx` (new component, render)

**Analog:** existing 780-px Drawer wrapping `MatchTestPanel` in `pages/asset/reconciliation/exception-rules/index.tsx:465-472`:
```tsx
<Drawer
  title="命中测试"
  open={matchTestOpen}
  onClose={() => setMatchTestOpen(false)}
  width={780}
>
  <MatchTestPanel embedded />
</Drawer>
```

**Header `extra` button pattern** (UI-SPEC "申请例外" in header):
```tsx
<Drawer
  title={`资产对账详情 — ${assetCode}`}
  open={open}
  onClose={onClose}
  width={780}            // 锁定,与 R3 exception-rules Drawer 同宽
  extra={
    <Button type="primary" onClick={onApplyException}>
      申请例外
    </Button>
  }
>
  <Tabs activeKey={activeTab} onChange={setActiveTab} tabBarGutter={24}
    items={[summaryTab, timelineTab, exceptionTab]} />
</Drawer>
```

**Tabs composition** (per UI-SPEC):
- Tab 1 `冲突摘要` — read from `useAssetHealth(selectedAssetId, workstationId)` (NO second API call, single source of truth).
- Tab 2 `历史变更` — render `<ReconciliationTimeline records={...} />`.
- Tab 3 `例外规则` — render `<ExceptionMatchList rules={...} />`.

**Single source of truth** — `useWorkstationHealth(workstationId)` is called ONCE at page level; drawer reads the asset slice via `useAssetHealth` (selector over cached query).

**Hidden state** — when `useReconciliationVisibility() === false`, drawer is never rendered (parent guards).

---

### `src/components/reconciliation/ReconciliationTimeline.tsx` (new component, render)

**No existing Timeline component in codebase** — first use of antd `Timeline`.

**Ant Design Timeline mode left** (UI-SPEC):
```tsx
import { Timeline } from "antd";

<Timeline mode="left">
  {records.map((r) => (
    <Timeline.Item
      key={r.id}
      color={CONFLICT_TYPE_HEX[r.conflictType]}
      dot={<HealthBadge conflictType={r.conflictType} interactive={false} size={12} />}
    >
      <div>{CONFLICT_TYPE_LABEL[r.conflictType]} · {dayjs(r.detectedAt).format("YYYY-MM-DD HH:mm")}</div>
      <div>由 {r.resolvedByUsername} 于 {dayjs(r.resolvedAt).format("YYYY-MM-DD HH:mm")} 解决</div>
      <div style={{ fontSize: 12, color: "#6b7280" }}>说明: {r.resolutionNote || "(无)"}</div>
    </Timeline.Item>
  ))}
</Timeline>
```

**Empty/Loading states** (per UI-SPEC):
```tsx
if (loading) return <Skeleton active paragraph={{ rows: 3 }} />;
if (!records?.length) return <Empty description="该资产暂无已解决的冲突记录。" />;
```

**D-A3-02 lock**: raw_snapshot JSONB NOT rendered (timeline is read-only).

---

### `src/components/reconciliation/ExceptionMatchList.tsx` (new component, render)

**Analog A:** `MatchTestPanel` rule table (`src/components/asset/reconciliation/MatchTestPanel.tsx:80+` — list rendering of matched rules).

**Analog B:** `actionTagColor` helper from `pages/asset/reconciliation/exception-rules/index.tsx:477-493` (REUSE directly, do NOT duplicate):
```tsx
// Already defined in exception-rules/index.tsx:478 — copy verbatim to this file
function actionTagColor(action: string): string {
  switch (action) {
    case "silence": return "red";
    case "no_alert": return "orange";
    case "no_notice": return "gold";
    case "no_workorder": return "purple";
    case "skip_severity": return "blue";
    default: return "default";
  }
}
```

**antd List rendering**:
```tsx
import { List, Tag, Empty, Spin, Button } from "antd";

if (loading) return <Spin />;
if (!rules?.length) {
  return (
    <>
      <Empty description="当前没有例外规则覆盖该资产所在 IP 段。" />
      <Button onClick={onCreateRule}>去创建例外规则</Button>
    </>
  );
}

return (
  <List
    size="small"
    dataSource={rules}
    renderItem={(rule) => (
      <List.Item>
        <div>
          <div><strong>{rule.name}</strong> <Tag color={scopeColor(rule.scopeType)}>{rule.scopeType}</Tag></div>
          <div><code>{rule.ipRange}</code></div>
          <Space size={4} wrap>
            {rule.exceptionActions.map((a) => <Tag key={a} color={actionTagColor(a)}>{a}</Tag>)}
          </Space>
          <div style={{ fontSize: 12, color: "#6b7280" }}>{rule.reason}</div>
          {rule.expiresAt && (
            <div style={{ fontSize: 12, color: "#6b7280" }}>
              有效期至 {dayjs(rule.expiresAt).format("YYYY-MM-DD")}
            </div>
          )}
        </div>
      </List.Item>
    )}
  />
);
```

---

### `src/components/reconciliation/hooks/useReconciliationVisibility.ts` (new hook, auth check)

**Analog:** `useDict` (`src/hooks/useDict.ts`) for `useQuery` shape; no existing per-perm gate hook in codebase.

**Pattern** — read perm list from `authStore.perms` via `useSyncExternalStore`:
```tsx
import { useSyncExternalStore } from "react";
import { useAuthStore } from "@/store/authStore";

const REQUIRED_PERM = "asset:reconciliation:list";

/**
 * 跨模块权限门:D-A1-03 静默降级
 *
 * 后端在 WorkstationHandler.GetByID 内已设置 ReconciliationVisible flag
 * (per cross-module-permission.md §2.3)。前端 hook 同步检查 perm — 当
 * 两者不一致时,前端以**后端 visible 字段为准**(defense in depth)。
 */
export function useReconciliationVisibility(): boolean {
  const perms = useAuthStore((s) => s.perms);
  if (!perms) return false;
  return perms.includes(REQUIRED_PERM);
}
```

**Defense-in-depth check** (UI-SPEC "Cross-Module Permission Degradation"):
```tsx
// In useWorkstationHealth — also gate on backend `visible` field
const backendVisible = data?.visible !== false;
return useReconciliationVisibility() && backendVisible;
```

---

### `src/components/reconciliation/hooks/useWorkstationHealth.ts` (new hook, React Query)

**Analog:** `useDashboard` (`src/hooks/useDashboard.ts:49-113`) — parallel `useQuery` with stable `staleTime` + `gcTime` + `refetchOnWindowFocus: false`.

**Pattern**:
```tsx
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import type { ByWorkstationResponse } from "@/lib/assetApi";  // 🆕 add type
import { queryKeys } from "@/lib/queryKeys";
import { useReconciliationVisibility } from "./useReconciliationVisibility";

const STALE_TIME_MS = 5 * 60 * 1000;  // 5 min, matches R1 MV refresh
const GC_TIME_MS = 10 * 60 * 1000;

export function useWorkstationHealth(
  workstationId: string
): UseQueryResult<ByWorkstationResponse> {
  const visible = useReconciliationVisibility();

  return useQuery({
    queryKey: queryKeys.reconciliation.workstationHealth(workstationId),
    queryFn: () => reconciliationApi.byWorkstation({ workstationId, window: "7d" }),
    enabled: visible && Boolean(workstationId),  // D-A1-03 silent degradation
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });
}
```

**useEffect deps stability (CLAUDE.md hard rule)** — `workstationId` is the ONLY dep; `enabled` flag from parent boolean state. No inline-object deps.

---

### `src/components/reconciliation/hooks/useAssetHealth.ts` (new hook, selector)

**Analog:** dashboard slice selectors over `useDashboard()` result.

**Pattern** — reads `useWorkstationHealth` cached data, NO new API call:
```tsx
import { useWorkstationHealth } from "./useWorkstationHealth";

export interface AssetHealthItem {
  assetId: string;
  assetCode: string;
  conflictType: string;
  severity: string;
  exceptionRuleId?: string | null;
  appliedActions?: string[];
  confidenceScore: number;
}

export function useAssetHealth(
  assetId: string | null,
  workstationId: string
): AssetHealthItem | undefined {
  const { data } = useWorkstationHealth(workstationId);
  if (!data?.assets || !assetId) return undefined;
  return data.assets.find((a) => a.assetId === assetId);
}
```

**Single source of truth**: avoids N+1 — drawer Tab 1 reuses workstation-level cache (UI-SPEC).

---

### `src/components/reconciliation/hooks/useExceptionMatch.ts` (new hook, React Query)

**Analog:** existing `queryKeys.reconciliation.matchTest` factory (`src/lib/queryKeys.ts:76-77`) + `reconciliationApi.exceptionRule.test` (`src/lib/assetApi.ts:290-323`).

**Pattern** — copy from `MatchTestPanel.tsx:71-87`:
```tsx
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

export function useExceptionMatch(params: { ip: string; conflictType?: string }) {
  // queryKey 入参对象 useMemo 稳定(CLAUDE.md useEffect 强约束)
  const stableQueryKey = useMemo(
    () => params,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(params)]
  );

  return useQuery({
    queryKey: queryKeys.reconciliation.matchTest(stableQueryKey),
    queryFn: () => reconciliationApi.exceptionRule.test({ ip: params.ip }),
    enabled: Boolean(params.ip),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}
```

---

### `src/lib/queryKeys.ts` — add `workstationHealth` + `assetHealth` factories (modify, registry)

**Analog:** existing `reconciliation.*` factories (`src/lib/queryKeys.ts:45-81`).

**Pattern** — append under `reconciliation:`:
```ts
reconciliation: {
  // ... existing
  /** 工位对账健康度(Phase 45 R4 / by-workstation 端点) */
  workstationHealth: (workstationId: string) =>
    ["reconciliation", "workstation-health", workstationId] as const,
  /** 资产对账健康度(从 workstationHealth 切片,无独立端点) */
  assetHealth: (assetId: string) =>
    ["reconciliation", "asset-health", assetId] as const,
}
```

**Type signature stability**: `as const` tuples preserved (consistent with all other factories).

---

### `src/lib/assetApi.ts` — add `byWorkstation` method (modify, API wrapper)

**Analog:** existing `reconciliationApi` factory methods (entire file pattern).

**Pattern** — append under `reconciliationApi`:
```ts
// ==================== R4 Phase 45 Types ====================

/**
 * 工位对账健康度聚合响应(POST /asset/reconciliation/by-workstation)
 * D-A4-02 锁定
 */
export interface ByWorkstationResponse {
  workstation: { id: string; name: string; code: string };
  healthScore: HealthScore;
  assets: AssetHealthItem[];
  visible: boolean;
}

export interface HealthScore {
  total: number;
  normal: number;
  drift: number;
  conflict: number;
  noData: number;
  exceptionHit: number;
  score: number;     // 0-100, simple ratio (D-A2-03)
  trend: TrendPoint[];
}

export interface AssetHealthItem {
  assetId: string;
  assetCode: string;
  conflictType: string;       // A-F or "" (健康)
  severity: string;           // low/medium/high/critical or ""
  exceptionRuleId?: string | null;
  appliedActions?: string[];
  confidenceScore: number;
}

// ==================== R4 Method ====================

/**
 * 工位对账健康度聚合(D-A4-01/02)
 *
 * 单次拿完顶部卡片 + 资产子表徽标 + 详情跳转锚点(避免 N+1,与 SC7 一致)。
 * 缓存 TTL 5min(后端 CacheProvider.GetOrSet)。
 */
byWorkstation: async (data: { workstationId: string; window?: string }): Promise<ByWorkstationResponse> => {
  const res = await post<ByWorkstationResponse>(
    "/asset/reconciliation/by-workstation",
    { workstationId: data.workstationId, window: data.window ?? "7d" }
  );
  return res.data as ByWorkstationResponse;
},
```

**Note**: `TrendPoint` is already defined at line 37-46 of `assetApi.ts` — REUSE.

---

### `src/components/operations/WorkstationDeviceTable/index.tsx` — add HealthBadge column to AD/asset sub-tables (modify)

**Analog:** existing `createColumns(canEdit)` factory (`index.tsx:271-390`).

**Modification site**: extend the columns array returned by `createColumns(canEdit)`. Add new column between `"状态"` (line 314-324) and `"主设备"` (line 325-334):
```tsx
{
  title: "对账健康",
  key: "reconciliation",
  width: 96,
  render: (_: any, record: WorkstationDevice) => (
    <HealthBadge assetId={record.assetId} conflictType={...} onClick={openDrawer} />
  ),
},
```

**Conflict type resolution**: for AD/asset sub-tables, `record.assetId` may or may not be set. The `HealthBadge` reads from `useWorkstationHealth(workstationId)` cache; the drawer Tab 1 lookup needs the asset UUID to find the conflict type. If `record.assetId` is empty, render `<HealthBadge assetId={null} ... />` (renders `-` placeholder).

**Where to mount the shared Drawer state**: lift `drawerState` to page level (per UI-SPEC "Both pages — at page level: render `<ReconciliationDrawer>` once").

---

### `src/pages/operations/workstations/views/CardView.tsx` — add HealthCard to expand top (modify)

**Analog:** existing `expandable.expandedRowRender` pattern in `pages/operations/workstations/index.tsx:539-551`.

**Current pattern** (index.tsx:539-551):
```tsx
expandable={{
  expandedRowRender: (record: WorkstationOps) => (
    <WorkstationDeviceTable
      workstationId={record.id}
      onDeviceChange={refreshData}
    />
  ),
  ...
}}
```

**R4 modification** — wrap HealthCard ABOVE WorkstationDeviceTable:
```tsx
expandedRowRender: (record: WorkstationOps) => (
  <div>
    <HealthCard
      workstationId={record.id}
      onApplyException={() => navigate(
        `/asset/reconciliation/exception-rules/new?workstationId=${record.id}`
      )}
    />
    <WorkstationDeviceTable workstationId={record.id} onDeviceChange={refreshData} />
  </div>
),
```

**D-A1-03 silent degradation**: `HealthCard` internally returns `null` when `useReconciliationVisibility() === false`. No conditional needed at the call site.

**CardView/FloorPlanView parity** (UI-SPEC "if expand implemented — same as above"): apply same pattern in CardView.tsx if those views have their own expand.

---

## Shared Patterns

### 跨模块 service 注入模式 (cross-module permission + service injection)

**Source:** `internal/api/v1/system/ad_account_handler.go:28-36` + `cross-module-permission.md §2.3`

**Apply to:** `internal/api/v1/operations/workstation_handler.go`

```go
// Pattern: chainable WithXxx setter (与现有 WithCore 一致)
type WorkstationHandler struct {
    service             opsServices.WorkstationService
    reconciliationSvc   asset.ReconciliationService  // 🆕
    core                *core.Core
}

func (h *WorkstationHandler) WithReconciliationService(svc asset.ReconciliationService) *WorkstationHandler {
    if h != nil {
        h.reconciliationSvc = svc
    }
    return h
}

// 在 router wiring(internal/api/router.go:580-606):
workstationHandler := operations.NewWorkstationHandler(workstationService).
    WithCore(core).
    WithReconciliationService(reconciliationSvc)
```

### operlog.Record 强制约定

**Source:** `internal/api/v1/asset/reconciliation_handler.go:115-145` (CreateRule / UpdateRule / DeleteRule patterns) + CLAUDE.md "操作日志记录约定 — 强制"

**Apply to:** `ResolveException` (R4 invalidate addition) — `operlog.Record` is called AFTER `invalidate_workstation_health` and BEFORE `response.Success`:

```go
// 顺序固定:invalidate → operlog.Record → response.Success
asset.InvalidateWorkstationHealth(ctx, h.core.Cache, workstationID)  // 🆕 R4

operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)
response.Success(c, gin.H{"id": id})
```

**Read endpoints (no operlog)**: `GetByWorkstation` handler is read — NO `operlog.Record` call (mirror `ListExceptions` / `GetByID`).

### Cache Key Helper + Strip Prefix 模式

**Source:** `internal/services/asset/cache_keys.go` (entire file)

**Apply to:** R4 `invalidate_workstation_health` helper — append to same file:
- Use `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, wsID)` (constant template already exists).
- `cache.Delete(ctx, key)` — Redis prefix `xingran:` auto-handled (CLAUDE.md Cache Key Prefix Handling).

### React Query + useDict-driven badge colors

**Source:** `src/hooks/useDict.ts` + `src/lib/queryKeys.ts:16-19` + `src/pages/asset/reconciliation/exception-rules/index.tsx:248-258`

**Apply to:** `HealthBadge.tsx` — resolve 6 type colors (A-F) from `useDict("asset_reconciliation_conflict_type")` `listClass` field. Map `success/warning/error/default/processing` → hex constants from UI-SPEC.

### useEffect Dependencies 稳定性

**Source:** CLAUDE.md "useEffect Dependencies" 强约束 + `src/components/asset/reconciliation/MatchTestPanel.tsx:71-75`

**Apply to:** ALL R4 React hooks + components:

```tsx
// ✅ Correct: stable primitive deps
useEffect(() => {
  refetch();
}, [workstationId]);

// ✅ Correct: useMemo for object deps
const stableQueryKey = useMemo(
  () => ({ ip, userId, deptId }),
  // eslint-disable-next-line react-hooks/exhaustive-deps
  [JSON.stringify({ ip, userId, deptId })]
);

// ❌ Wrong: inline-object deps (causes infinite loops)
useEffect(() => { refetch(); }, [{ ip }]);
```

### 静默权限降级 (D-A1-03)

**Source:** `cross-module-permission.md §2.3` + UI-SPEC "Cross-Module Permission Degradation"

**Apply to:** ALL R4 frontend components + backend `WorkstationHandler.GetByID`:

```tsx
// Frontend — 双重防御(useReconciliationVisibility() + 后端 visible 字段)
const visible = useReconciliationVisibility();
const backendVisible = useWorkstationHealth(wsId).data?.visible !== false;
if (!visible || !backendVisible) return null;
```

```go
// Backend — handler-level silent degradation(无 403,无错误暴露)
if h.hasReconciliationPerm(c) {
    if recon, err := h.reconciliationSvc.GetByWorkstation(ctx, id, "7d"); err == nil {
        workstation.Reconciliation = recon
        workstation.ReconciliationVisible = true
    }
} else {
    workstation.ReconciliationVisible = false
    workstation.ReconciliationHiddenReason = "无资产对账查看权限"
}
```

### Statistics 专用 COUNT 端点(严禁 list.length)

**Source:** `internal/services/asset/reconciliation_statistics.go:102-120` + project memory `stat-cards-from-list-length-capped-at-100`

**Apply to:** R4 `reconciliation_service.go:GetByWorkstation` — HealthScore fields (normal/drift/conflict/nodata/exceptionHit) MUST use `SELECT COUNT(*) FILTER (WHERE ...)` aggregate queries, NEVER `len(assets)`. Mirror `reconciliation_statistics.go:Summary()` shape exactly.

---

## No Analog Found

Files requiring new patterns (planner should use RESEARCH.md + UI-SPEC contracts):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `src/components/reconciliation/ReconciliationTimeline.tsx` | component | render | First use of antd Timeline in project — no existing Timeline component |
| `src/components/reconciliation/hooks/useAssetHealth.ts` | hook | selector | First cross-cache selector pattern (slices one query result for another component) |
| `src/components/reconciliation/hooks/useReconciliationVisibility.ts` | hook | auth check | No existing per-perm gate hook — must check `authStore.perms` via `useSyncExternalStore` |

---

## Metadata

**Analog search scope:**
- Backend: `internal/services/asset/`, `internal/api/v1/asset/`, `internal/api/v1/operations/`, `internal/api/v1/system/`, `internal/api/router.go`, `internal/core/`, `pkg/cache/`
- Frontend: `src/lib/assetApi.ts`, `src/lib/queryKeys.ts`, `src/lib/api.ts`, `src/hooks/useDict.ts`, `src/hooks/useDashboard.ts`, `src/pages/asset/reconciliation/`, `src/components/asset/reconciliation/`, `src/components/operations/WorkstationDeviceTable/`, `src/pages/operations/workstations/`

**Files scanned:** 16 backend Go files + 6 frontend TS/TSX files

**Pattern extraction date:** 2026-06-28

**Key constraint reminders (locked by CONTEXT + project memory):**
- 11 operlog 关键词 + 25 OperType 常量 (CLAUDE.md)
- Redis `xingran:` 前缀由 cache.Set 自动处理 (CLAUDE.md Cache Key Prefix Handling)
- 0=启用/正常 1=停用 (CLAUDE.md Status Convention)
- `list.length` 严禁用于统计 — 用 COUNT 端点 (`stat-cards-from-list-length-capped-at-100`)
- useEffect deps 必须稳定 (CLAUDE.md useEffect Dependencies)
- 菜单 SQL migration 字段名匹配 model (`migration-sql-name-must-match-model`)
- 路由不预注册 `/asset/reconciliation/*` 通用路由 (`xingran-excel-import-route-conflict`)