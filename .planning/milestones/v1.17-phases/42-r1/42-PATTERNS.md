# Phase 42: 资产对账观测底座 (R1) - Pattern Map

**Mapped:** 2026-06-27
**Files analyzed:** 21 new + 6 modified
**Analogs found:** 19 / 21 with strong analog matches (90%+)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| **NEW (Backend)** |||||
| `internal/models/reconciliation.go` | model | CRUD | `internal/models/workorder.go` (WorkOrderCategory, WorkOrderTemplate) | role-match |
| `internal/services/asset/cache_keys.go` | utility | CRUD | `internal/services/system/cache_keys.go` | exact (pattern) |
| `internal/services/asset/reconciliation_service.go` | service | CRUD+event | `internal/services/operations/location_alias_service.go` | role-match |
| `internal/services/asset/reconciliation_detection.go` | service | event-driven | `internal/services/operations/batch_upserter.go` | partial (sql pattern) |
| `internal/services/asset/reconciliation_exception.go` | service | CRUD | `internal/services/operations/location_alias_service.go` | role-match |
| `internal/services/asset/reconciliation_statistics.go` | service | aggregate | `internal/services/operations/asset_service.go` (Statistics) + `internal/services/system/config_service.go` (Statistics) | exact (pattern) |
| `internal/services/asset/reconciliation_snapshot.go` | service | ETL | `internal/core/db/migrations/migration_152_mac_matview.go` | partial (SQL pattern only) |
| `internal/api/v1/asset/reconciliation_handler.go` | handler | request-response | `internal/api/v1/operations/asset_handler.go` | exact |
| `internal/api/v1/asset/reconciliation_exception_handler.go` | handler | request-response | `internal/api/v1/system/ad_account_handler.go` | role-match |
| `internal/api/v1/asset/reconciliation_router.go` | route | request-response | `internal/api/v1/scheduler/job_router.go` + `internal/api/v1/system/apikey_router.go` | exact |
| `internal/scheduler/reconciliation_tasks.go` | cron | event-driven | `internal/scheduler/ad_sync_tasks.go` | role-match |
| `internal/core/db/migrations/migration_168_reconciliation_tables.go` | migration | DDL | `internal/core/db/migrations/migration_148_create_ops_asset_table.go` + `migration_162_ad_service_accounts.go` | exact (D-21) |
| `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go` | migration | DDL+seed | `internal/core/db/migrations/migration_163_ad_account_pool_menu.go` + `migration_165_sys_dept_location_alias.go` | exact (P1+P2) |
| **MODIFIED (Backend)** |||||
| `internal/api/router.go` | router | request-response | (current `internal/api/router.go` ops asset registration) | exact (D-21) |
| `internal/core/db/database.go` | config | migration registration | (current `internal/core/db/database.go` line 399-417) | exact |
| **NEW (Frontend)** |||||
| `src/lib/assetApi.ts` (rename/repurpose) | lib/API | CRUD | `src/lib/opsApi.ts` (assetApi) | exact |
| `src/lib/queryKeys.ts` (modify) | lib/keys | query key factory | `src/lib/queryKeys.ts` (current structure) | exact |
| `src/pages/asset/reconciliation/index.tsx` | page | route 302 redirect | `src/pages/operations/workstation/...` (parent route patterns) | partial |
| `src/pages/asset/reconciliation/dashboard/index.tsx` | page | dashboard | `src/pages/operations/assets/index.tsx` (statistics cards pattern) | exact |
| `src/pages/asset/reconciliation/exceptions/index.tsx` | page | admin list | `src/pages/operations/assets/index.tsx` (table + filter pattern) | exact |
| `src/hooks/useDashboard.ts` | hook | data fetch | `src/hooks/useDeptTree.ts` (TanStack Query pattern) | partial |
| `src/hooks/useExceptionList.ts` | hook | data fetch | (TanStack Query patterns in duty module) | partial |

## Pattern Assignments

### `internal/models/reconciliation.go` (model, CRUD)

**Analog:** `internal/models/workorder.go` (WorkOrderCategory / WorkOrderTemplate)

**Imports pattern** (lines 1-9):
```go
package models

import (
    "time"
    "gorm.io/gorm"
)
```

**BaseModel embedding + TableName pattern** (lines 55-69):
```go
type WorkOrderCategory struct {
    BaseModel
    CategoryName string `gorm:"size:100;not null" json:"categoryName"`
    // ...
}

func (WorkOrderCategory) TableName() string {
    return "sys_workorder_category"
}
```

**Key conventions to follow:**
- Embed `BaseModel` for UUID + soft delete + timestamps (auto-created via `internal/models/base.go:11-19`)
- Use `gorm:"column:lowercase_snake"` to lock DB column names (avoid GORM auto-naming like `Nickname` → `nick_name` trap from `migration-sql-name-must-match-model`)
- Use `gorm:"size:N"` for VARCHAR, `gorm:"type:uuid"` for FK refs
- Define `TableName()` method returning `sys_*` prefix (CLAUDE.md: "sys_ prefix for system tables")

**v0.5 field names confirmed by analog (`internal/models/asset.go:54-58, 86-91`):**
- `ops_asset.MAC1` (column `mac1`) / `ops_asset.MAC2` (column `mac2`)
- `ops_asset.MachineIP` (column `machine_ip`)
- `ops_asset.UserID` (column `user_id`)
- `sys_user_ad_attrs` for AD (separate table, NOT `ad.managed_by_dn` direct)

**Status convention:** `0 = enabled/normal, 1 = disabled/stopped` (CLAUDE.md Status Convention). Exception rule uses `is_active BOOLEAN` (true=active, false=disabled).

---

### `internal/services/asset/cache_keys.go` (utility, CRUD)

**Analog:** `internal/services/system/cache_keys.go` (exact pattern match)

**Constants + helpers pattern** (lines 290-296):
```go
// GetDictDataByTypeKey 根据字典类型构建字典数据缓存键
func GetDictDataByTypeKey(dictType string) string {
    return CacheKeyDictData + ":" + dictType
}
```

**Apply this pattern for reconciliation keys:**
```go
package asset

import "fmt"

const (
    CacheKeyReconciliationDashboard           = "reconciliation:dashboard:%s"
    CacheKeyReconciliationExceptionList       = "reconciliation:exception:list:%s"
    CacheKeyReconciliationExceptionByID       = "reconciliation:exception:byID:%s"
    CacheKeyReconciliationExceptionRuleList   = "reconciliation:exceptionRule:list"
    CacheKeyReconciliationExceptionRuleByID   = "reconciliation:exceptionRule:byID:%s"
    CacheKeyReconciliationViewLastRefresh     = "reconciliation:view:lastRefresh"
    CacheKeyReconciliationHealthByWorkstation = "reconciliation:health:workstation:%s"
    CacheKeyReconciliationHealthByAsset       = "reconciliation:health:asset:%s"
)

func GetReconciliationDashboardKey(scope string) string {
    return fmt.Sprintf(CacheKeyReconciliationDashboard, scope)
}
// ... 8 helpers total (one per const)
```

**Key constraints:**
- Cache keys auto-prefixed `xingran:` by `internal/core/core.go:342` — pass keys WITHOUT prefix
- Redis prefix handling: `if strings.HasPrefix(key, "xingran:") { key = key[6:] }` for user input (CLAUDE.md "Cache Prefix Handling")

---

### `internal/services/asset/reconciliation_service.go` (service, CRUD)

**Analog:** `internal/services/operations/location_alias_service.go` (exact role match)

**Interface + private impl + constructor pattern** (lines 42-89):
```go
type LocationAliasService interface {
    List(ctx context.Context, pageNum, pageSize int) (*PageResult, error)
    GetByID(ctx context.Context, id string) (*models.SysDeptLocationAlias, error)
    Create(ctx context.Context, req *LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error)
    Update(ctx context.Context, id string, req *LocationAliasUpdateRequest) error
    Delete(ctx context.Context, id string) error
}

type locationAliasServiceImpl struct {
    db *gorm.DB
}

func NewLocationAliasService(db *gorm.DB) LocationAliasService {
    return &locationAliasServiceImpl{db: db}
}
```

**Sort field whitelist pattern** (lines 53-60 of `internal/services/operations/asset_service.go`):
```go
var assetAllowedSortFields = map[string]string{
    "devicesn": "devicesn",
    "status":   "status",
    // ...
}
```

**ApplySort usage** (`asset_service.go:197`):
```go
sortReq := extractSortRequest(params)
query = base.ApplySort(query, sortReq, assetAllowedSortFields)
```

**Apply this for reconciliation:**
- Interface: `ReconciliationService` with `ListExceptions`, `GetByID`, `Create`, `Update`, `Delete`, `MarkResolved`
- Private impl: `reconciliationServiceImpl struct { db *gorm.DB }`
- Constructor: `NewReconciliationService(db *gorm.DB) ReconciliationService`
- `reconAllowedSortFields` map: `{"detectedAt": "detected_at", "conflictType": "conflict_type", "severity": "severity"}`

**Important from memory `xingran-server-side-sort-infra`:**
- `BaseListRequest` + `ApplySort` whitelist already in place
- Same package service List must use `params.ListParams` embedded struct

---

### `internal/services/asset/reconciliation_detection.go` (service, event-driven)

**Analog:** `internal/services/operations/batch_upserter.go` (SQL upsert pattern) + `internal/services/operations/location_alias_service.go:172` (duplicate key handling)

**Duplicate key error detection pattern** (lines 168-176):
```go
if err := s.db.WithContext(ctx).Create(alias).Error; err != nil {
    if isDuplicateKeyError(err) {
        return nil, fmt.Errorf("该 dept_id + scope 组合已存在,不可重复创建: ...")
    }
    return nil, fmt.Errorf("创建别名映射失败: %w", err)
}
```

**Apply for Layer 3 detection (D-11):**
- Cron-triggered service that scans `reconciliation_normalized` MV → classifies Type A-F → inserts into `sys_data_reconciliation`
- Catch unique violation on `uniq_recon_asset_type_open(asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL`
- Use `isDuplicateKeyError(err)` from `operations` package (or implement local equivalent)
- Type A: skip insert (only statistic, D-09)
- Type B-F: insert via GORM `.Create()` with conflict-skip catch

**Inverse query pattern for confidence scoring** (`workstation_device_service.go` analog):
```go
// INET comparison via COALESCE (D-08 mac1 优先,mac2 备选)
LEFT JOIN sys_port_mac pm ON pm.mac = COALESCE(a.mac1, a.mac2) AND pm.deleted_at IS NULL
```

---

### `internal/services/asset/reconciliation_exception.go` (service, CRUD)

**Analog:** `internal/services/operations/location_alias_service.go` (exact role match — both are CRUD with multi-field validation)

**Two-tier validation pattern** (lines 145-178):
```go
func (s *locationAliasServiceImpl) Create(ctx context.Context, req *LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error) {
    if req == nil {
        return nil, errors.New("请求体不能为空")
    }
    // scope 兜底
    scope := req.Scope
    if scope == "" {
        scope = aliasDefaultScope
    }
    // ... validation
    if err := s.validateAlias(ctx, alias); err != nil {
        return nil, err
    }
    if err := s.db.WithContext(ctx).Create(alias).Error; err != nil {
        if isDuplicateKeyError(err) {
            return nil, fmt.Errorf("...")
        }
        return nil, fmt.Errorf("创建别名映射失败: %w", err)
    }
    return alias, nil
}
```

**Apply for IP CIDR exception rule (v0.3 R3 is OUT of scope, but skeleton needed for R1 to land):**
- Interface: `ReconciliationExceptionService` with `List/Create/Update/Delete/GetByID`
- Private impl: `reconciliationExceptionServiceImpl struct { db *gorm.DB }`
- Constructor: `NewReconciliationExceptionService(db *gorm.DB) ReconciliationExceptionService`
- Validation: `validateException(ctx, rule)` — CIDR format check, reason ≥10 chars, scope_type enum
- Use `isDuplicateKeyError(err)` for unique constraint on `(ip_range, conflict_types)`

---

### `internal/services/asset/reconciliation_statistics.go` (service, aggregate)

**Analog:** `internal/services/operations/asset_service.go:60-75` (Statistics method — exact match for "don't use list.length" pattern from `stat-cards-from-list-length-capped-at-100` memory)

**Statistics pattern with conditional SUM** (lines 60-75):
```go
// Statistics 统计资产(按 status + nbf_status 聚合,排除软删除)。
// 替代前端「total*0.8/0.15/0.05」的伪造比例占位实现。
func (s *assetService) Statistics(ctx context.Context) (*AssetStatisticsResult, error) {
    var result AssetStatisticsResult
    err := s.db.WithContext(ctx).
        Model(&models.Asset{}).
        Select(
            "COUNT(*) AS total",
            "COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS normal",
            "COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS stopped",
            "COALESCE(SUM(CASE WHEN nbf_status = 1 THEN 1 ELSE 0 END), 0) AS nbf",
        ).
        Scan(&result).Error
    if err != nil {
        return nil, err
    }
    return &result, nil
}
```

**Interface pattern** (analog `internal/services/system/config_service.go:23-34`):
```go
type ConfigService interface {
    Create(ctx context.Context, req *requests.ConfigCreateRequest) error
    // ...
    Statistics(ctx context.Context) (*ConfigStatisticsResult, error)
}

type ConfigStatisticsResult struct {
    Total    int64 `json:"total"`
    Active   int64 `json:"active"`
    Inactive int64 `json:"inactive"`
}
```

**Apply for 6 statistics endpoints (D-06 + F2):**
```go
type ReconciliationStatistics interface {
    Summary(ctx context.Context, filters StatsFilter) (*SummaryResult, error)
    ByConflictType(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    BySeverity(ctx context.Context, filters StatsFilter) (map[string]int64, error)
    HealthTrend(ctx context.Context, filters StatsFilter) ([]TrendPoint, error)
    TopUnresolved(ctx context.Context, limit int) ([]ExceptionSummary, error)
    ExceptionRuleStats(ctx context.Context) ([]RuleStats, error)
}

type SummaryResult struct {
    TotalAssets       int64  `json:"totalAssets"`        // SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL
    OpenExceptions    int64  `json:"openExceptions"`     // SELECT COUNT(*) FROM sys_data_reconciliation WHERE resolved_at IS NULL AND deleted_at IS NULL
    CriticalOpen      int64  `json:"criticalOpen"`       // + severity = 'critical'
    Last7dNew         int64  `json:"last7dNew"`          // detected_at >= NOW() - INTERVAL '7 days'
    TopConflictType   string `json:"topConflictType"`    // GROUP BY conflict_type ORDER BY count DESC LIMIT 1
    TopConflictCount  int64  `json:"topConflictCount"`
}
```

**Critical constraint (from `stat-cards-from-list-length-capped-at-100`):**
- MUST use `SELECT COUNT(*)` / `SUM(CASE WHEN ...)` — never `list.length`
- Test in `internal/services/operations/asset_statistics_test.go:17-62` (sqlite in-memory pattern)

---

### `internal/services/asset/reconciliation_snapshot.go` (service, ETL)

**Analog:** `internal/core/db/migrations/migration_152_mac_matview.go` (SQL pattern only — Go orchestration code does not exist yet, this is new territory)

**Materialized view SQL pattern** (from `migration_152_mac_matview.go`):
```go
// CREATE MATERIALIZED VIEW reconciliation_normalized AS
// SELECT ...
// FROM ops_asset a
// LEFT JOIN sys_port_mac pm ON pm.mac = a.mac1 AND pm.deleted_at IS NULL
// LEFT JOIN sys_info_point ip ON ip.port_id = pm.port_id AND ip.deleted_at IS NULL
// LEFT JOIN sys_workstation_info_point wip ON wip.info_point_id = ip.id
// LEFT JOIN sys_workstation w ON w.id = wip.workstation_id AND w.deleted_at IS NULL
// LEFT JOIN sys_user wu ON wu.id = w.user_id
// LEFT JOIN sys_user_ad_attrs ad ON ad.user_id = COALESCE(w.user_id, a.user_id)
// WHERE a.deleted_at IS NULL;
// CREATE UNIQUE INDEX idx_recon_norm_asset ON reconciliation_normalized(asset_id);
```

**Refresh strategy pattern** (D-01, D-02):
```go
func (s *reconciliationSnapshotService) Refresh(ctx context.Context) error {
    // CONCURRENTLY requires UNIQUE INDEX (asset_id already unique)
    return s.db.WithContext(ctx).Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized").Error
}
```

**Apply for snapshot service:**
- Service: `RefreshView(ctx) error` — execute REFRESH CONCURRENTLY
- On startup: `core.Init()` should call `RefreshView` once (avoid 0-5min cold start, D-02)
- On failure: log only via `applogger.Errorf` — no SysNotice, no cache 告警位 (D-02)

**Key constraint from `migration_152_mac_matview.go`:**
- CREATE UNIQUE INDEX required for CONCURRENTLY
- DDL goes in `migration_NNN_reconciliation_tables.go`, not Go code

---

### `internal/api/v1/asset/reconciliation_handler.go` (handler, request-response)

**Analog:** `internal/api/v1/operations/asset_handler.go` (exact match)

**Struct + WithCore pattern** (lines 14-33):
```go
type AssetHandler struct {
    assetService operations.AssetService
    core         *core.Core
}

func NewAssetHandler(assetService operations.AssetService) *AssetHandler {
    return &AssetHandler{
        assetService: assetService,
    }
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *AssetHandler) WithCore(core *core.Core) *AssetHandler {
    if h != nil {
        h.core = core
    }
    return h
}
```

**Standard CRUD handler with operlog** (lines 36-50):
```go
// Create 创建资产
func (h *AssetHandler) Create(c *gin.Context) {
    var asset models.Asset
    if err := c.ShouldBindJSON(&asset); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }

    if err := h.assetService.Create(c.Request.Context(), &asset); err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeCreate)
    response.Success(c, asset)
}
```

**Statistics handler (no operlog)** (lines 198-207):
```go
// Statistics 资产统计(读操作,不记操作日志)
func (h *AssetHandler) Statistics(c *gin.Context) {
    result, err := h.assetService.Statistics(c.Request.Context())
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    response.Success(c, result)
}
```

**Apply for reconciliation_handler.go:**
- Module name: `ModuleReconciliation = "资产对账"` (D-16 — R1 only this 1 module constant)
- Read endpoints (List/Get/Statistics/...) → no operlog.Record
- Mark resolved (R2 only) → operlog.Record with OperTypeUpdate
- R1 has NO write endpoints (D-18) — observation only

**Apply for reconciliation_exception_handler.go:**
- Module name: same `ModuleReconciliation = "资产对账"` (D-16)
- R3 scope, but skeleton is created in R1 per CONTEXT.md Integration Points

---

### `internal/api/v1/asset/reconciliation_router.go` (route, request-response)

**Analog:** `internal/api/v1/scheduler/job_router.go` (exact match)

**Standard router pattern** (lines 9-34):
```go
func SetupJobRouter(r *gin.RouterGroup, core *core.Core) {
    // 创建服务实例
    jobService := schedulerServices.NewJobService(core.DB.GetDB(), core.Scheduler)
    jobLogService := schedulerServices.NewJobLogService(core.DB.GetDB())

    // 创建Handler实例
    handler := NewJobHandler(jobService, jobLogService).WithCore(core)

    // 注册路由
    r.POST("/list", handler.List)
    r.POST("", handler.Create)
    r.POST("/:id", handler.GetByID)
    r.POST("/:id/update", handler.Update)
    r.POST("/:id/delete", handler.Delete)
    r.POST("/:id/status", handler.UpdateStatus)
    r.POST("/:id/execute", handler.Execute)

    // 日志相关
    r.POST("/logs/list", handler.ListLogs)
    r.POST("/logs/statistics", handler.Statistics)
    r.POST("/logs/clean", handler.CleanLogs)
}
```

**Registration in main router** (`internal/api/router.go:664-693` asset registration):
```go
// 资产管理
assets := ops.Group("/asset")
assets.Use(middleware.RequirePermissions([]string{
    "ops:asset:list",
    "ops:asset:add",
    "ops:asset:edit",
    "ops:asset:delete",
}, core))
{
    assetService := opsServices.NewAssetService(core.DB.GetDB())
    assetHandler := operations.NewAssetHandler(assetService).WithCore(core)
    // ...
}
```

**Apply for R1 (F2):**
- R1 needs `/asset/reconciliation/*` routes registered in `internal/api/router.go`
- Per CONTEXT.md D-21, register under `r.Group("/asset")` (NOT inside `/ops`):
  ```go
  // 在 router.go 中添加 (D-21, F2):
  assetReconciliation := r.Group("/asset/reconciliation")
  assetReconciliation.Use(middleware.RequirePermissions([]string{
      "asset:reconciliation:list",
      "asset:reconciliation:dashboard",
      "asset:reconciliation:export",
  }, core))
  {
      // SetupReconciliationRouter (Statistics + Exception List)
      asset.SetupReconciliationRouter(assetReconciliation, core)
      // SetupReconciliationExceptionRouter (CRUD rules — skeleton in R1, full in R3)
      asset.SetupReconciliationExceptionRouter(assetReconciliation, core)
  }
  ```

**Critical from `Excel 导入路由冲突陷阱` memory:**
- router.go must NOT pre-register `/asset/reconciliation/*` generic routes (avoid handler conflicts)
- Each `SetupXxxRouter` self-manages its routes
- `excel_handler.SetupExcelRouter` MUST NOT pre-register `reconciliationException` entityType

---

### `internal/scheduler/reconciliation_tasks.go` (cron, event-driven)

**Analog:** `internal/scheduler/ad_sync_tasks.go` (exact match — but with R1 caveat from D-10: cron lives in sys_job table, NOT in scheduler package)

**Caveat from CONTEXT.md D-10:**
> **D-10:** Cron 走 sys_job 表（`api/v1/scheduler` 现有页面管理）
>   - 在 `sys_job` 表新增 4 个 job records：MV 刷新 / Layer 3 检测 / 静默期到期重检测 / 临时例外清理
>   - 不引入 `internal/scheduler/reconciliation_tasks.go`（保留 R1 无新增 cron 文件）

**HOWEVER — CONTEXT.md says R1 keeps "no new cron file", but the 4 sys_job records need to be seeded in migration_169.**

**Apply via migration seed** (not new Go file):
- migration_169 inserts 4 records into `sys_job` table (InvokeTarget: `reconciliation:refreshView` / `reconciliation:detectLayer3` / etc.)
- Use existing `models.Job` model from `internal/services/scheduler/job_service.go:138-148`
- Operators modify cron via `POST /monitor/jobs/:id/update` UI

**If a Go file IS needed (alternative pattern from `ad_sync_tasks.go:75-80`):**
```go
// RegisterReconciliationTasks 注册资产对账定时任务
func RegisterReconciliationTasks(scheduler *Scheduler) {
    scheduler.RegisterTask("reconciliation_refresh_view", func(ctx context.Context, params map[string]interface{}) error {
        return executeReconciliationRefreshViewTask(ctx, params)
    })
}
```

**R1 decision:** Stick with sys_job approach (D-10) — migration_169 seeds 4 records.

---

### `internal/core/db/migrations/migration_168_reconciliation_tables.go` (migration, DDL)

**Analog:** `internal/core/db/migrations/migration_148_create_ops_asset_table.go` (DDL) + `internal/core/db/migrations/migration_162_ad_service_accounts.go` (table + sequence)

**DDL with explicit unique constraint** (lines 121-130 of migration_148):
```go
// 显式命名 unique constraint (与 GORM uniqueIndex 命名规范一致,避免 AutoMigrate 重建冲突)
// 注意: 用 DO 块保证幂等,避免重复运行报错
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uni_ops_asset_devicesn'
          AND conrelid = 'ops_asset'::regclass
    ) THEN
        ALTER TABLE ops_asset ADD CONSTRAINT uni_ops_asset_devicesn UNIQUE (devicesn);
    END IF;
END$$;
```

**Migration function signature + log pattern** (lines 9-19):
```go
func Migrate148CreateOpsAssetTable(db *gorm.DB) error {
    log.Println("Running migration 148: Create ops_asset table")
    if db.Migrator().HasTable(&OpsAsset{}) {
        log.Println("Table ops_asset already exists, skipping migration 148...")
        return nil
    }
    // ...
}
```

**Apply for migration_168:**
- Create `sys_data_reconciliation` table (8 fields + BaseModel + JSONB raw_snapshot + INET asset_ip + uuid exception_rule_id FK)
- Create `sys_reconciliation_exception` table (CIDR + reason ≥10 chars + soft delete)
- Create `reconciliation_normalized` materialized view (FULL asset → port_mac → info_point → workstation → user_id chain, D-08)
- Unique constraint `uni_recon_asset_type_open` on (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL (D-11)
- Partial index on `reconciliation_normalized` for asset_id (required for CONCURRENTLY, D-01)
- Index on `sys_data_reconciliation.detected_at` (for 7d query in Statistics)

**Critical constraints from project memory:**
1. `xingran-migrations-no-sql-autoloader`: SQL files don't auto-load — must use `.go` migration function
2. `xingran-gorm-sql-constraint-naming-conflict`: SQL inline UNIQUE uses PG auto-name `*_key`, model uniqueIndex expects `uni_*_*` — use DO $$ blocks to explicitly name constraint `uni_recon_asset_type_open`
3. `migration-sql-name-must-match-model`: column names must match DB schema (use `MachineIP` not `IP`, `MAC1` not `MAC`)

---

### `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go` (migration, DDL+seed)

**Analog:** `internal/core/db/migrations/migration_165_sys_dept_location_alias.go` (table + button permissions seed)

**Menu button permission seed pattern** (lines 73-129):
```go
buttons := []struct {
    name  string
    perms string
}{
    {"工位部门映射查询", "ops:location:alias:list"},
    // ...
}

for _, btn := range buttons {
    // 幂等: perms 已存在则跳过
    var existingCount int64
    if err := db.Table("sys_menu").Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
        continue
    }
    if existingCount > 0 {
        continue
    }
    // ...
    buttonMenu := &models.Menu{
        MenuName: btn.name,
        ParentID: &parentMenu.ID,
        MenuType: "F",  // 按钮
        Visible:  0,    // 隐藏
        Status:   0,    // 正常
        Perms:    &btnPerms,
        // ...
    }
    if err := db.Create(buttonMenu).Error; err != nil {
        continue
    }
}
```

**Apply for migration_169:**
- 4 dict_type + dict_data seeds (P1):
  - `asset_reconciliation_conflict_type` (A-F values)
  - `asset_reconciliation_severity` (low/medium/high/critical)
  - `asset_reconciliation_exception_action` (no_alert/no_notice/no_workorder/skip_severity/silence)
  - `asset_reconciliation_status` (open/resolved)
- 8 sys_config seeds (P2):
  - `asset.reconciliation.view.refresh_interval=5m`
  - `asset.reconciliation.score.physical/declared/ad`
  - `asset.reconciliation.exception.default_expiry_days`
  - `asset.reconciliation.alert.critical_threshold`
  - `asset.reconciliation.alert.silence_after_resolved_hours`
  - `asset.reconciliation.health.score_weights`
- 6 workorder category seeds (Type A-F for R2):
  - `category_name`: "对账异常-A类" / "B类" / "C类" / "D类" / "E类" / "F类"
  - `status=0` (enabled)
- 4 sys_job records (D-10):
  - `job_name`: "对账-物化视图刷新" / "对账-Layer3检测" / "对账-静默期重检测" / "对账-例外规则清理"
  - `invoke_target`: `reconciliation:refreshView` / `detectLayer3` / `detectExpiredSilence` / `cleanupExpiredExceptions`
  - `cron_expression`: `@every 5m` / `@every 6m` / `0 2 * * *` / `0 3 * * *`
- 6 sys_menu records (菜单归属资产管理 / 数据质量):
  - "资产对账" (dir, parent="资产管理" if exists)
  - "对账看板" (menu, perms=`asset:reconciliation:dashboard`)
  - "异常列表" (menu, perms=`asset:reconciliation:list`)
  - "例外规则" (menu, perms=`asset:reconciliation:exception:list`) — R3 skeleton
  - Plus 4 button permissions under each menu

**Critical from `ops 菜单 seed perms 与路由命名不一致` memory:**
- Use single-form `asset:reconciliation:exception:list` (NOT plural)
- Route uses `SetupReconciliationExceptionRouter` with `RequirePermissions(asset:reconciliation:exception:list)`
- menu seed perms MUST match router perms exactly (migration_159 already aligned ops, do same for asset)

---

### `internal/api/router.go` (modify)

**Analog:** existing `internal/api/router.go:664-693` asset registration (F2)

**Current asset block** (lines 664-693):
```go
// 资产管理
assets := ops.Group("/asset")
assets.Use(middleware.RequirePermissions([]string{
    "ops:asset:list",
    "ops:asset:add",
    "ops:asset:edit",
    "ops:asset:delete",
}, core))
{
    assetService := opsServices.NewAssetService(core.DB.GetDB())
    assetHandler := operations.NewAssetHandler(assetService).WithCore(core)
    // ...
}
```

**Apply R1 changes:**
- Add a NEW top-level `assetReconciliation` group (NOT inside `ops`):
  ```go
  // R1: 资产对账观测底座 (D-21, F2)
  assetReconciliation := r.Group("/asset/reconciliation")
  assetReconciliation.Use(middleware.JWTAuth(core.JWTManager))
  assetReconciliation.Use(middleware.OperLogMiddleware(core.OperLogService, core))
  assetReconciliation.Use(middleware.RequirePermissions([]string{
      "asset:reconciliation:list",
      "asset:reconciliation:dashboard",
      "asset:reconciliation:export",
  }, core))
  {
      reconciliationService := assetServices.NewReconciliationService(core.DB.GetDB())
      reconciliationStatisticsService := assetServices.NewReconciliationStatistics(core.DB.GetDB())
      reconciliationExceptionService := assetServices.NewReconciliationExceptionService(core.DB.GetDB())
      
      reconciliationHandler := asset.NewReconciliationHandler(reconciliationService, reconciliationStatisticsService).WithCore(core)
      exceptionHandler := asset.NewReconciliationExceptionHandler(reconciliationExceptionService).WithCore(core)
      
      asset.SetupReconciliationRouter(assetReconciliation, core)
      asset.SetupReconciliationExceptionRouter(assetReconciliation, core)
  }
  ```

**Critical constraints:**
- Don't conflict with existing `/ops/asset/*` (keep both, different prefixes)
- Don't pre-register generic `/asset/reconciliation/*` in router.go — only specific handlers (memory `Excel 导入路由冲突陷阱`)
- operlog middleware required for write paths (R2 onwards; R1 read-only)

---

### `internal/core/db/database.go` (modify)

**Analog:** existing registration lines 399-417

**Current pattern** (lines 393-417):
```go
// Phase 36: AD 域控服务账号池 + 自动故障切换
if err := migrations.Migrate162ADServiceAccounts(d.DB); err != nil {
    applogger.Errorf("AD 服务账号池迁移失败: %v", err)
}
// Phase 36: AD 域控服务账号池菜单 + 权限 seed
if err := migrations.Migrate163ADAccountPoolMenu(d.DB); err != nil {
    applogger.Errorf("AD 域控服务账号池菜单迁移失败: %v", err)
}
// ...
```

**Apply R1 additions:**
```go
// Phase 42 R1: 资产对账观测底座 — 主表 + 物化视图
if err := migrations.Migrate168ReconciliationTables(d.DB); err != nil {
    applogger.Errorf("Phase 42 R1 reconciliation tables 迁移失败: %v", err)
}
// Phase 42 R1: 字典 + config + workorder category + sys_job + menu seed
if err := migrations.Migrate169ReconciliationDictsConfigs(d.DB); err != nil {
    applogger.Errorf("Phase 42 R1 reconciliation seed 迁移失败: %v", err)
}
```

**Constraint from `xingran-migrations-no-sql-autoloader`:**
- Must use `.go` migration function (NOT .sql file)
- Must register here in `database.go` (NOT just drop file in `migrations/`)

---

### `src/lib/assetApi.ts` (NEW frontend, CRUD API)

**Analog:** `src/lib/opsApi.ts` (lines 586-651 — assetApi section, exact match)

**Asset API factory pattern** (lines 588-651):
```typescript
const assetCrudApi = createCrudApi<Asset>({ basePath: "/ops/asset" });

export const assetApi = {
  ...assetCrudApi,

  searchBySerial: async (serial: string) => {
    return await get<Asset>(`/ops/asset/search-by-serial/${serial}`, {});
  },

  getDeviceTypes: async () => {
    return await post<{ value: string; count: number }[]>("/ops/asset/device-types", {});
  },
  // ...

  statistics: async () => {
    const res = await post<{ total: number; normal: number; stopped: number; nbf: number }>("/ops/asset/statistics", {});
    return res.data;
  },
  // ...
};
```

**Apply for `src/lib/assetApi.ts` (R1 dashboard + exception):**
- Use `post` from `@/lib/api` (CLAUDE.md "Use wrapped API functions, NOT raw axios")
- Dashboard endpoints:
  - `reconciliationApi.summary()` → `POST /asset/reconciliation/statistics/summary`
  - `reconciliationApi.byConflictType()` → `POST /asset/reconciliation/statistics/by-conflict-type`
  - `reconciliationApi.bySeverity()` → `POST /asset/reconciliation/statistics/by-severity`
  - `reconciliationApi.healthTrend()` → `POST /asset/reconciliation/statistics/health-trend`
  - `reconciliationApi.topUnresolved()` → `POST /asset/reconciliation/statistics/top-unresolved`
- Exception list endpoint:
  - `reconciliationApi.exceptionList(params)` → `POST /asset/reconciliation/exception/list`

---

### `src/lib/queryKeys.ts` (MODIFY frontend)

**Analog:** current `src/lib/queryKeys.ts:13-42` (exact match — add reconciliation section)

**Current structure** (lines 13-42):
```typescript
export const queryKeys = {
  dict: {
    all: ["dict"] as const,
    list: (dictType: string) => ["dict", dictType] as const,
  },
  list: {
    all: (resource: string) => ["list", resource] as const,
    page: (resource: string, params: Record<string, unknown>) =>
      ["list", resource, params] as const,
  },
  dept: {
    all: ["dept"] as const,
    tree: () => ["dept", "tree"] as const,
  },
  duty: {
    all: ["duty"] as const,
    poolMembers: (deptId: string) => ["duty", "pool-members", deptId] as const,
  },
  // ... etc
} as const;
```

**Apply R1 addition (F6):**
```typescript
export const queryKeys = {
  // ... existing keys
  reconciliation: {
    all: ["reconciliation"] as const,
    summary: () => ["reconciliation", "summary"] as const,
    byConflictType: () => ["reconciliation", "by-conflict-type"] as const,
    bySeverity: () => ["reconciliation", "by-severity"] as const,
    healthTrend: (windowDays: number) => ["reconciliation", "health-trend", windowDays] as const,
    topUnresolved: (limit: number) => ["reconciliation", "top-unresolved", limit] as const,
    exceptionList: (params: ExceptionListParams) => ["reconciliation", "exception-list", params] as const,
    exceptionDetail: (id: string) => ["reconciliation", "exception-detail", id] as const,
  },
} as const;
```

**Constraint:** All keys are tuples `as const` for narrow literal types.

---

### `src/pages/asset/reconciliation/dashboard/index.tsx` (page, dashboard)

**Analog:** `src/pages/operations/assets/index.tsx` (lines 446-466 — statistics cards pattern, exact match)

**Statistics cards pattern** (lines 446-466):
```tsx
{/* 统计卡片 */}
<div className="grid grid-cols-4 gap-4 mb-6">
  <Card>
    <div className="text-2xl font-bold">{statistics.total}</div>
    <div className="text-gray-500">总资产数</div>
  </Card>
  <Card>
    <div className="text-2xl font-bold text-green-600">{statistics.normal}</div>
    <div className="text-gray-500">正常状态</div>
  </Card>
  // ... etc
</div>
```

**Apply for R1 dashboard (D-06 — 5 KPI cards):**
1. 全量资产数 (totalAssets)
2. 未解决异常数 (openExceptions)
3. critical 级未解决数 (criticalOpen) — text-red-600
4. 7d 新增异常数 (last7dNew)
5. Top1 冲突类型 + 计数 (topConflictType + topConflictCount)

**Critical constraint from `stat-cards-from-list-length-capped-at-100`:**
- MUST call 6 dedicated COUNT endpoints
- NEVER call `exceptionList.length`
- Each KPI is an independent `useQuery` with `queryKeys.reconciliation.*` key

**3 chart layout (per CONTEXT.md):**
- Pie chart (Type A-F by conflictType) → echarts-for-react
- Bar chart (severity 4-level)
- Line chart (healthTrend 7d/30d toggle, default 7d)
- Don't pursue echarts tree-shaking (memory `echarts6-customchart-tree-shaking-noop`)

---

### `src/pages/asset/reconciliation/exceptions/index.tsx` (page, admin list)

**Analog:** `src/pages/operations/assets/index.tsx` (lines 559-581 — table + pagination pattern, exact match)

**Table pattern** (lines 559-581):
```tsx
<Table
  rowKey="id"
  columns={tableColumns}
  dataSource={assets}
  loading={loading || configLoading}
  virtual
  pagination={{
    current: paginationProps.current,
    pageSize: paginationProps.pageSize,
    total,
    showSizeChanger: true,
    showQuickJumper: true,
    showTotal: (total) => `共 ${total} 条`,
  }}
  onChange={handleTableChange}
  scroll={{ x: 4200, y: 600 }}
  rowSelection={{
    selectedRowKeys,
    onChange: setSelectedRowKeys,
    columnWidth: 50,
  }}
/>
```

**Apply for R1 exception list:**
- Columns (per CONTEXT.md specifics):
  1. detected_at (default DESC)
  2. conflict_type (useDict("asset_reconciliation_conflict_type"))
  3. severity (useDict("asset_reconciliation_severity"))
  4. asset_code (JOIN)
  5. asset_ip (INET)
  6. physical_username (JOIN sys_user)
  7. responsible_username (JOIN sys_user)
  8. exception_rule_id (R3, hidden in R1)
  9. operlog_btn (R1: "查看日志" link only, NOT "标记已解决" — D-18)
- Filter form (D-05: read URL query string for bidirectional jump):
  - `type=A-F` from pie click → URL `?type=A`
  - `severity=critical` from bar click → URL `?severity=critical`
  - `from/to` date range from trend click → URL `?from=2026-06-20&to=2026-06-27`
- Use `useTableManager` hook (line 165-194 of analog)
- Use `useServerSort` hook (memory `xingran-server-side-sort-infra`)

---

## Shared Patterns

### operlog 强制约定 (CLAUDE.md 强约束)

**Source:** `internal/utils/operlog/operlog.go:214-262` + `internal/utils/operlog/regression_test.go`

**Apply to:** All reconciliation handler write paths (R2 onwards; R1 has NONE per D-18)

```go
// 标准模式 (R1 R2 R3 R4 通用)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产对账", operlog.OperTypeUpdate)
response.Success(c, result)
```

**R1 write operations per CONTEXT.md D-17:**
- ① Layer 3 检测 cron 触发 → `OperTypeSync`
- ② `sys_data_reconciliation` 写入/更新 → `OperTypeCreate` / `OperTypeUpdate`
- ③ 静默期到期重检测 → `OperTypeSync` (R2)
- ④ sys_job 表新增 4 个 cron 记录 → `OperTypeCreate`
- NO `RecordWithBody` needed (R1 writes have no sensitive fields)

**Module constants (D-16, R1 only):**
```go
const ModuleReconciliation = "资产对账"
// R2 adds:
// const ModuleReconciliationExceptionRule = "资产对账-例外规则"
// const ModuleReconciliationAutoWorkorder = "资产对账-自动转工单"
// const ModuleReconciliationReportExport  = "资产对账-报告导出"
```

---

### Status Convention (CLAUDE.md 强约束)

**Source:** CLAUDE.md "Status Value Convention"

**Apply to:**
- `sys_reconciliation_exception.is_active` → `INT` (0=active, 1=disabled) — Note: R3 scope
- `sys_data_reconciliation` uses `resolved_at IS NULL` (no status field) — open=resolved_at IS NULL, resolved=NOT NULL

---

### Cache Prefix Handling (CLAUDE.md 强约束)

**Source:** `internal/core/core.go:342` (Redis prefix `xingran:`)

**Apply to:**
- Reconciliation cache keys (D-24) — call `cache.Set(ctx, "reconciliation:dashboard:summary", value)` (NO `xingran:` prefix)
- Frontend monitor cache UI — strip `xingran:` prefix before operations:
  ```go
  if strings.HasPrefix(key, "xingran:") {
      key = key[6:]
  }
  ```

---

### Server-Side Sort (Phase A 基建)

**Source:** `internal/services/base/apply_sort.go` + memory `xingran-server-side-sort-infra`

**Apply to:**
- Exception list uses `BaseListRequest` + `ApplySort` whitelist
- Same-package `params.ListParams` embedded struct pattern

```go
type ExceptionListParams struct {
    ConflictType string `json:"conflictType"`
    Severity     string `json:"severity"`
    common.ListParams  // embeds BaseListRequest
}
```

---

### Cache Key Helper Functions

**Source:** `internal/services/system/cache_keys.go:294-296` (`GetDictDataByTypeKey`)

**Apply to:** `internal/services/asset/cache_keys.go` — define 8 constants + 8 helper functions (F1/D-24)

---

### Statistics 专用 COUNT 端点 (memory 强约束)

**Source:** `internal/services/operations/asset_service.go:60-75` + memory `stat-cards-from-list-length-capped-at-100`

**Apply to:** ALL 6 dashboard endpoints in `reconciliation_statistics.go` — MUST use `SELECT COUNT(*)` / `SUM(CASE WHEN ...)`, NEVER `list.length`

**Test pattern:** `internal/services/operations/asset_statistics_test.go:17-62` — sqlite in-memory + table create + insert + assert counts

---

### Cross-Module Permission Boundary (memory 强约束)

**Source:** `.planning/notes/260627-cross-module-permission.md` + memory `xingran-perm-namespace-split-readonly-page`

**Apply to:** R4 workstation detail page integration (OUT of R1 scope, but R1 dashboard must use `RequirePermissions` correctly)

**R1 permissions (per CONTEXT.md D-04 + cross-module audit):**
- `asset:reconciliation:list` (异常列表读)
- `asset:reconciliation:dashboard` (看板读)
- `asset:reconciliation:export` (导出读)
- `asset:reconciliation:exception:list` (R3 例外规则列表)
- `asset:reconciliation:exception:create/update/delete` (R3)

**For R1 router.go registration:**
```go
assetReconciliation.Use(middleware.RequirePermissions([]string{
    "asset:reconciliation:list",
    "asset:reconciliation:dashboard",
    "asset:reconciliation:export",
}, core))
```

---

### Excel 导入路由冲突规避 (memory 强约束)

**Source:** `internal/api/router.go` Excel config pre-registration + memory `Excel 导入路由冲突陷阱`

**Apply to:**
- DO NOT pre-register `/asset/reconciliation/*` generic routes in router.go
- DO NOT pre-register `reconciliationException` entityType in `excel_handler.SetupExcelRouter`
- Each `SetupXxxRouter` self-manages its own routes

---

### Migration Unique Index Naming (memory 强约束)

**Source:** memory `xingran-gorm-sql-constraint-naming-conflict` + migration_148 DO block pattern

**Apply to:**
- `sys_data_reconciliation` UNIQUE INDEX `uni_recon_asset_type_open` on (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL — use DO $$ block to explicitly name

---

### Migration SQL ↔ Model Field Name Match (memory 强约束)

**Source:** memory `migration-sql-name-must-match-model` + `internal/models/asset.go:54-58` actual fields

**Apply to:**
- `sys_data_reconciliation` column names must match GORM model `gorm:"column:..."` tags EXACTLY
- v0.5 confirmed: `ops_asset.MAC1` (column `mac1`), `ops_asset.MAC2` (column `mac2`), `ops_asset.MachineIP` (column `machine_ip`)
- For `sys_user_ad_attrs`: confirm actual column names by reading `internal/models/ad_user_attrs.go` before writing migration

---

### Migration .go vs .sql (memory 强约束)

**Source:** memory `xingran-migrations-no-sql-autoloader`

**Apply to:**
- MUST use `migration_NNN_*.go` function (NOT .sql)
- MUST register in `internal/core/db/database.go`

---

### Frontend useDict Hook (memory 强约束)

**Source:** `xingran-react-frontend/src/hooks/useDict.ts` + memory `duty-module-user-field-gotchas`

**Apply to:**
- Conflict type label: `useDict("asset_reconciliation_conflict_type")`
- Severity: `useDict("asset_reconciliation_severity")`
- Exception action: `useDict("asset_reconciliation_exception_action")` (R3)
- Status: `useDict("asset_reconciliation_status")` (R3)

**Gotcha:** `nickname` field MUST be lowercase in JSON tag (memory `duty-module-user-field-gotchas`)

---

### Frontend api.ts Wrapped Functions (CLAUDE.md 强约束)

**Source:** CLAUDE.md "Use wrapped API functions, NOT raw axios"

**Apply to:** ALL frontend API calls — use `post` from `@/lib/api`, NOT raw `axios.post(...)`

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns + CONTEXT.md specifics instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/asset/reconciliation_detection.go` | service | event-driven | Layer 3 engine (Type A-F classifier) is novel — closest analog is `batch_upserter.go` SQL pattern, but classification logic itself is new |
| `internal/services/asset/reconciliation_snapshot.go` | service | ETL | Materialized view refresh service — closest is `migration_152_mac_matview.go` SQL but no Go orchestrator exists yet |
| `src/pages/asset/reconciliation/index.tsx` | page | 302 redirect | Parent route that redirects to `/dashboard` — no exact analog in existing pages |
| `src/pages/asset/reconciliation/dashboard/index.tsx` | page | dashboard | Dashboard with 3 charts + 5 KPI — closest is `assets/index.tsx` (4 KPI), but 3-chart layout is new |

## Metadata

**Analog search scope:**
- `internal/services/operations/` (asset + location_alias + batch_upserter)
- `internal/services/system/` (config + dict_cache)
- `internal/services/scheduler/` (job_service)
- `internal/services/workorder/` (BaseService — R2)
- `internal/api/v1/operations/` (asset_handler)
- `internal/api/v1/system/` (ad_account_handler + apikey_router)
- `internal/api/v1/scheduler/` (job_handler + job_router)
- `internal/core/db/migrations/` (migration_148/152/162/163/165)
- `internal/scheduler/` (ad_sync_tasks.go)
- `internal/models/` (asset.go + workorder.go + base.go)
- `xingran-react-frontend/src/pages/operations/assets/` (asset list page)
- `xingran-react-frontend/src/lib/` (opsApi.ts + queryKeys.ts)

**Files scanned:** 30 (Go: 22, Frontend: 4, Migrations: 4)

**Pattern extraction date:** 2026-06-27

**Coverage:** 19/21 files with strong analog matches (90%+)