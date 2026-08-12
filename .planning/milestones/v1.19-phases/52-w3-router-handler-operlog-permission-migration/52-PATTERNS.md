# Phase 52: W3 — Router/Handler/Operlog/Permission/Migration - Pattern Map

**Mapped:** 2026-07-07
**Files analyzed:** 9 (7 create + 2 modify)
**Analogs found:** 9 / 9 (8 exact/role-match + 1 composite)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `pkg/permission/config.go` (MODIFY) | config | request-response (RBAC code) | `pkg/permission/config.go:186` (self) | exact (insert one line) |
| `internal/models/port_write_audit.go` (CREATE, Path X → Wave 1) | model | file-I/O (GORM Create) | `internal/models/device_port_status.go` + `internal/models/log.go` | exact (composite: uuid PK + BeforeCreate + JSONB) |
| `internal/services/portcollection/cache_keys.go` (CREATE) | config (constants) | n/a (pure constants) | `internal/services/asset/cache_keys.go` | exact |
| `internal/api/v1/network/port_write_handler.go` (CREATE) | controller | request-response + audit write | `internal/api/v1/asset/fix_suggestion_handler.go` | role-match (operlog pattern) + `reconciliation_handler.go` (Module const) |
| `internal/api/v1/network/port_write_router.go` (CREATE) | route | request-response | `internal/api/v1/network/port_router.go` | role-match (same module; port_router has no group-level permission — borrow from network_router.go) |
| `internal/api/v1/network/network_router.go` (MODIFY) | route | request-response | self (line 213 after `SetupPortRouter`) | exact (one-line insert) |
| `internal/core/db/migrations/menu_grant_helpers.go` (CREATE) | utility (migration helper) | batch (INSERT...SELECT) | `internal/core/db/migrations/migration_144_vdi_granular_permissions.go:62-79` (role-menu grant loop) + `136_add_group_mapping_menu.sql:281-284` (INSERT...SELECT idiom) | role-match (helper wraps this idiom) |
| `internal/core/db/migrations/migration_202_port_write_audit.go` (CREATE) | migration | batch (seed + index) | `internal/core/db/migrations/migration_195_reconciliation_exception_rules_menu.go` (count-then-insert menu seed) + `migration_200_fix_suggestion_config_seeds.go:187-201` (F-type button shape) + `migration_201_phase48_component_columns.go` (CREATE INDEX style) | composite (3 analogs merged) |
| `internal/core/db/database.go` (MODIFY) | config (AutoMigrate list) | batch (startup) | self (line 308-391 model list + line 411-417 explicit migration call) | exact (two insertions in known locations) |

---

## Pattern Assignments

### 1. `pkg/permission/config.go` (config, RBAC constant — MODIFY)

**Analog:** self (line 186, `NetworkPortQuery`)
**Insertion point:** immediately after line 186 (`NetworkPortQuery PermissionCode = "network:port:query"`)

**Delta:** add one constant. Group stays under NetworkPort namespace.

```go
// 端口状态查询权限
NetworkPortQuery PermissionCode = "network:port:query"
// 端口写操作权限（Phase 52: shutdown/undo_shutdown/description/dot1x_enable/dot1x_disable/batch）
NetworkPortWrite PermissionCode = "network:port:write"
```

**Do NOT** add anything to `GetRoutePermissions()` (line 200-264) — that table only covers the system module; network routes authorize via router-level `RequirePermissions` (RESEARCH §3.1, VERIFIED).

---

### 2. `internal/models/port_write_audit.go` (model — CREATE)

**Analogs (composite):**
- `internal/models/device_port_status.go:31-72` — UUID PK + `TableName()` + `BeforeCreate` hook shape
- `internal/models/log.go:6-25` — append-only log row (no UpdatedAt, no DeletedAt)
- `internal/models/base.go:30-42` — `BaseTimeLine.BeforeCreate` (preset ID preserved — relevant to RESEARCH §1.1, but Path C is recommended so we don't depend on it)

**Concrete excerpt — `device_port_status.go:31-72` (the model shape to copy):**

```go
type DevicePortStatus struct {
    ID            string `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID      string `gorm:"type:uuid;not null;uniqueIndex:uniq_device_interface,priority:1" json:"deviceId"`
    InterfaceName string `gorm:"size:100;not null" json:"interfaceName"`
    AdminStatus   string `gorm:"size:20" json:"adminStatus,omitempty"`
    // ...
    CreatedAt time.Time `json:"createdAt"`
}

func (DevicePortStatus) TableName() string {
    return "sys_device_port_status"
}

func (d *DevicePortStatus) BeforeCreate(tx *gorm.DB) error {
    if d.ID == "" {
        d.ID = uuid.New().String()
    }
    return nil
}
```

**Delta vs analog (per RESEARCH §3.6 + D-01):**
- Do **not** embed `BaseTimeLine` (it carries `UpdatedAt`; audit is append-only)
- `BeforeValue` / `AfterValue` → `json.RawMessage` with `gorm:"type:jsonb"` (avoid marshal churn)
- `FailureReason` / `OperLogID` → `*string` (nullable)
- Composite index `(device_id, port_id, created_at)` via GORM composite-index tag with explicit name `idx_port_write_audit_device_port_created` (priority:1/2/3) — controls naming (memory `xingran-gorm-sql-constraint-naming-conflict`)
- Single-column index `(created_at)` via separate tag `idx_port_write_audit_created`
- `BeforeCreate` hook generates UUID (copy from analog)
- **Path A (recommended):** no `gorm:"-:migration"` tag — GORM AutoMigrate builds the table from this model; migration_202 only adds seed/index/helper

**Recommended struct (RESEARCH §3.6):**

```go
type PortWriteAudit struct {
    ID             string          `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID       string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:1" json:"deviceId"`
    PortID         string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:2" json:"portId"`
    Action         string          `gorm:"size:32;not null" json:"action"`
    BeforeValue    json.RawMessage `gorm:"type:jsonb" json:"beforeValue"`
    AfterValue     json.RawMessage `gorm:"type:jsonb" json:"afterValue"`
    CommandSent    string          `gorm:"type:text" json:"commandSent"`
    DeviceResponse string          `gorm:"type:text" json:"deviceResponse"`
    Status         string          `gorm:"size:16;not null" json:"status"`
    FailureReason  *string         `gorm:"type:text" json:"failureReason,omitempty"`
    Operator       string          `gorm:"size:50" json:"operator"`
    OperLogID      *string         `gorm:"type:uuid" json:"operLogId,omitempty"`
    CreatedAt      time.Time       `gorm:"not null;index:idx_port_write_audit_device_port_created,priority:3;index:idx_port_write_audit_created" json:"createdAt"`
}

func (PortWriteAudit) TableName() string { return "sys_port_write_audit" }

func (a *PortWriteAudit) BeforeCreate(tx *gorm.DB) error {
    if a.ID == "" {
        a.ID = uuid.New().String()
    }
    return nil
}
```

---

### 3. `internal/services/portcollection/cache_keys.go` (config, pure constants — CREATE)

**Analog:** `internal/services/asset/cache_keys.go:1-51`

**Concrete excerpt — `asset/cache_keys.go:25-51` (const block + R1/R2 staged adoption comment):**

```go
// Cache key 常量模板 — 资产对账模块
//
// 命名规则: `reconciliation:<sub>:<suffix>`
//
// Redis 前缀处理 (CLAUDE.md Cache Key Prefix Handling):
//   - 实际 Redis key 会自动加 `xingran:` 前缀,本文件定义的 key 是"逻辑键"
//
// R1/R2/R3 演进:
//   - R1: 仅 cache_keys 文件就位,无运行时调用(INFRA-03 占位)
const (
    CacheKeyReconciliationDashboard       = "reconciliation:dashboard:%s"
    CacheKeyReconciliationExceptionList   = "reconciliation:exception:list:%s"
    // ...
)
```

**Delta vs analog:** D-10 locks MVP to constants-only (no callers). Two constants, one optional helper pair. Adopt the analog's "R1 staged" comment style to signal INFRA-03 占位.

```go
package portcollection

// Cache key 常量 — 端口写操作结果/批量任务 (Phase 52 INFRA-03 占位)
//
// 命名规则: `port:write:<sub>:<suffix>`  (%s = port_id / batch_id)
//
// MVP 演进 (D-10 锁定):
//   - Phase 52: 仅 cache_keys 文件就位,无运行时调用(INFRA-03 占位)
//   - Phase 53+: 接入 CacheProvider 后启用
const (
    CacheKeyPortWriteResult = "port:write:result:%s" // %s = port_id
    CacheKeyPortWriteBatch  = "port:write:batch:%s"  // %s = batch_id
)

// GetPortWriteResultKey / GetPortWriteBatchKey 占位 helper(Phase 53+ 接入 CacheProvider 时启用)
// func GetPortWriteResultKey(portID string) string { return fmt.Sprintf(CacheKeyPortWriteResult, portID) }
// func GetPortWriteBatchKey(batchID string) string { return fmt.Sprintf(CacheKeyPortWriteBatch, batchID) }
```

**Note:** planner can uncomment the helpers (analogue `asset/cache_keys.go:56-93` defines Get* helpers eagerly). Recommendation: keep them defined — Phase 53 consumes directly without re-editing.

---

### 4. `internal/api/v1/network/port_write_handler.go` (controller, request-response + audit write — CREATE)

**Primary analog:** `internal/api/v1/asset/fix_suggestion_handler.go` (operlog.Record handler pattern — 4 callsites at lines 132/177/214/276)

**Secondary analog:** `internal/api/v1/asset/reconciliation_handler.go:18-35` (Module constant pattern)

#### 4a. Imports + struct + constructor (copy from `fix_suggestion_handler.go:1-43`)

```go
package asset  // → package network

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/core"
    "github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
    "github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

type FixSuggestionHandler struct {       // → PortWriteHandler
    service asset.FixSuggestionService   // → portwrite.PortWriteService
    core    *core.Core
}

func NewFixSuggestionHandler(svc asset.FixSuggestionService) *FixSuggestionHandler {
    return &FixSuggestionHandler{service: svc}
}

func (h *FixSuggestionHandler) WithCore(core *core.Core) *FixSuggestionHandler {
    if h != nil {
        h.core = core
    }
    return h
}
```

**Delta vs analog:**
- DI service is `portwrite.PortWriteService` (Phase 51 shipped)
- Constructor: `NewPortWriteHandler(svc portwrite.PortWriteService) *PortWriteHandler`
- Add `db *gorm.DB` field? No — use `h.core.GetDB()` for both PortWriteAudit INSERT and the D-02 pre-SELECT (matches analog's `h.core.GetDB()` usage)

#### 4b. Module constant (copy from `reconciliation_handler.go:18-35`)

```go
// ModulePortWrite 端口写操作 operlog 模块名常量 (Phase 52 AUDIT-01 锁定)
//
// 注意(D-07):与父菜单名 "端口状态" 解耦 — module 仅作 sys_oper_log.title 显示串,
// 沿用 ROADMAP 历史用名 "端口管理"。
const ModulePortWrite = "端口管理"
```

#### 4c. Write-endpoint operlog pattern (copy from `fix_suggestion_handler.go:109-139` Accept handler)

**Strict order (D-A4-04 / CLAUDE.md operlog 强制约定):** `service → audit INSERT → operlog.Record → response.Success`

```go
// Accept 接受建议 (analog: lines 109-139)
func (h *FixSuggestionHandler) Accept(c *gin.Context) {
    id := c.Param("id")
    if id == "" {
        response.Error(c, http.StatusBadRequest, "建议ID不能为空")
        return
    }
    userID, ok := h.getUserID(c)
    if !ok {
        return
    }

    if err := h.service.Accept(c.Request.Context(), id, userID); err != nil {
        errMsg := err.Error()
        if errMsg == "该建议已被处理或不存在" {                // sentinel→HTTP translation (analog pattern)
            response.Error(c, http.StatusConflict, errMsg)
            return
        }
        response.Error(c, http.StatusInternalServerError, errMsg)
        return
    }

    // operlog 写入(CLAUDE.md 强制约定,状态变更 → OperTypeUpdate)
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

    response.Success(c, gin.H{ "id": id, "acceptedBy": userID })
}
```

**Delta vs analog (per Phase 52 contract):**
1. **Before service call:** pre-SELECT `DevicePortStatus` snapshot for `before_value` (D-02). Analog has no such step.
2. **After service success, before operlog:** INSERT `PortWriteAudit` row (D-04) — analog skips audit table entirely (audit lives in operlog).
3. **Two failure branches** (RESEARCH §3.3, critical):
   - **Sentinel error** (`ErrBatchTooLarge`/`ErrEmptyBatch`/`ErrMixedDevices`/`ErrPortNotFound`/`ErrDeviceNotFound`) → `response.Error(c, 4xx, msg)` — no audit row
   - **`PortResult.Status == "failed"` + nil sentinel err** (transport_error/device_rejected) → `response.Success(c, result)` + audit row (status=failed) — **NOT response.Error**
4. **OperType mapping** (D-15): shutdown/undo_shutdown/dot1x_enable/dot1x_disable → `OperTypeStatus`(=10); description → `OperTypeUpdate`(=2); batch → `OperTypeBatch`(=16)
5. **operator source:** `utils.GetUsername(c)` (CLAUDE.md惯例; analog uses `h.getUserID(c)` helper which reads `c.Get("user_id")` — for audit.operator column use username per D-04)
6. **`device_response` column fill** (RESEARCH A5, since Phase 51 service doesn't expose raw response):
   - succeeded → `"OK"`
   - failed → `result.Error`
   - skipped → `"无需操作"`
7. **audit↔operlog association = Path C** (RESEARCH §1.1 strongly recommended): handler writes audit rows FIRST (capturing `audit_ids`), then `operlog.Record(..., operlog.WithOperParam(batchSummaryJSON))` where `batchSummaryJSON` contains `{"audit_ids":["..."], ...}`. Do **not** add `WithOperID` to operlog package (would break regression_test.go locks). `audit.oper_log_id` column stays NULL in Phase 52.

#### 4d. Sentinel → HTTP translation table (RESEARCH §3.3, plug directly into handler)

| Sentinel / branch | HTTP | response.Error msg | audit row? |
|---|---|---|---|
| `ErrBatchTooLarge` | 400 | "批量端口数超过上限 50" | no |
| `ErrEmptyBatch` | 400 | "批量端口列表为空" | no |
| `ErrMixedDevices` | 400 | "批量端口必须属于同一设备" | no |
| `ErrPortNotFound` | 404 | "端口不存在" | no |
| `ErrDeviceNotFound` | 404 | "设备不存在" | no |
| `fmt.Errorf("query port/device: %w", err)` | 500 | "查询端口失败"/"查询设备失败" | no |
| `PortResult.Status=="failed"` + nil err | **200** | — (response.Success) | **yes** (status=failed) |
| `PortResult.Status=="succeeded"` | 200 | — | yes (status=succeeded) |
| `PortResult.Status=="skipped"` (NoOp) | 200 | — | yes (status=skipped, command_sent="", device_response="无需操作") |

**`errors.Is` usage** (since Phase 51 uses `var Err... = errors.New(...)`, sentinel pattern):

```go
result, err := h.service.Shutdown(c.Request.Context(), req.PortID, operator)
if err != nil {
    switch {
    case errors.Is(err, portwrite.ErrPortNotFound):
        response.Error(c, http.StatusNotFound, "端口不存在")
    case errors.Is(err, portwrite.ErrDeviceNotFound):
        response.Error(c, http.StatusNotFound, "设备不存在")
    default:
        response.Error(c, http.StatusInternalServerError, err.Error())
    }
    return
}
// result.Status may still be "failed" — that's a 200 with audit row, NOT response.Error
```

#### 4e. PortResult → audit row mapping (RESEARCH §3.4)

```go
// after_value填充策略 (D-03):
//   shutdown → {"admin_status":"down"}
//   undo_shutdown → {"admin_status":"up"}
//   dot1x_enable → {"dot1x_enabled":true}
//   dot1x_disable → {"dot1x_enabled":false}
//   description → {"description":"<desc>"}
//   skipped (NoOp) → after_value = before_value (no change)

func buildAuditRow(pr *portwrite.PortResult, before json.RawMessage, action, operator string) *models.PortWriteAudit {
    after := buildAfterValue(pr)  // per D-03
    if pr.Status == "skipped" {
        after = before
    }
    deviceResponse := "OK"
    var failureReason *string
    switch pr.Status {
    case "failed":
        deviceResponse = pr.Error
        s := pr.Error
        failureReason = &s
    case "skipped":
        deviceResponse = "无需操作"
    }
    return &models.PortWriteAudit{
        DeviceID:       /* from pre-SELECT */,
        PortID:         pr.PortID,
        Action:         string(pr.Action),
        BeforeValue:    before,
        AfterValue:     after,
        CommandSent:    pr.CommandSent,   // NoOp/skipped → "" per D-04
        DeviceResponse: deviceResponse,
        Status:         pr.Status,
        FailureReason:  failureReason,
        Operator:       operator,
        // OperLogID: nil per Path C
    }
}
```

#### 4f. Batch handler skeleton (RESEARCH §3.4)

```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeBatch,
    operlog.WithOperParam(batchSummaryJSON))  // {action, batch_size, succeeded, failed, skipped, device_id, audit_ids:[...]}

for _, pr := range append(append(result.Succeeded, result.Failed...), result.Skipped...) {
    auditRow := buildAuditRowFromPortResult(pr, beforeSnapshot[pr.PortID], action, operator)
    if err := h.core.GetDB().Create(auditRow).Error; err != nil {
        applogger.Warnf("port_write audit insert failed portID=%s: %v", pr.PortID, err)
        // 不阻塞响应,继续写下一条 (D specific)
    }
}
response.Success(c, result)
```

**Note on OperType:** single-port handlers should pass the per-action OperType (D-15). batch always uses `OperTypeBatch`.

---

### 5. `internal/api/v1/network/port_write_router.go` (route — CREATE)

**Analogs (composite):**
- `internal/api/v1/network/port_router.go:8-19` — same-module router, kebab naming
- `internal/api/v1/network/network_router.go:206-214` — `ports := r.Group("/ports")` parent group (where `SetupPortWriteRouter` plugs in)
- `internal/api/v1/network/network_router.go:78-84` — group-level `RequirePermissions` pattern (the 2-arg signature)

#### 5a. Kebab route naming (copy from `port_router.go:8-19`)

```go
func SetupPortRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler) {
    handler := NewPortHandler(core)
    r.POST("/list", handler.List)
    r.POST("/collect", handler.Collect)
    r.POST("/collect-all", handler.CollectAll)
    r.POST("/batch-delete", handler.BatchDelete)
}
```

#### 5b. Group-level RequirePermissions (copy from `network_router.go:78-84`)

```go
credentials := r.Group("/credentials")
credentials.Use(middleware.RequirePermissions([]string{
    "network:credential:list",
    "network:credential:add",
    "network:credential:edit",
    "network:credential:delete",
}, core))
```

**CRITICAL:** signature is **2-arg**: `RequirePermissions(permissions []string, core *core.Core)` (RESEARCH §1.4 VERIFIED `pkg/middleware/permission.go:200`). CONTEXT/CLAUDE.md incorrectly wrote 1-arg. The handler-side permission check `Permission(perm, core)` is also 2-arg (line 40).

#### 5c. Delta — Phase 52 router skeleton

```go
package network

import (
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/core"
    "github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
    "github.com/xingran-next/xingran-go-backend/pkg/middleware"
    "github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupPortWriteRouter 端口写操作路由 (Phase 52 D-09)
//
// 子组 /network/ports/write/* + 组级 RequirePermissions([network:port:write])
// 6 端点 kebab 命名(与 /list /collect /batch-delete 同风格)。
func SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core) {
    svc := portwrite.NewPortWriteService(
        core.GetDB(),
        core.DeviceExecutor,
        core.DeviceInfoCollectionService,
    )
    handler := NewPortWriteHandler(svc).WithCore(core)

    write := r.Group("/write")
    write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core))

    write.POST("/shutdown",        handler.Shutdown)
    write.POST("/undo-shutdown",   handler.UndoShutdown)
    write.POST("/description",     handler.SetDescription)
    write.POST("/dot1x-enable",    handler.EnableDot1x)
    write.POST("/dot1x-disable",   handler.DisableDot1x)
    write.POST("/batch",           handler.BatchWrite)
}
```

**Notes:**
- `write := r.Group("/write")` mounts under the existing `ports` group passed in as `r`
- Single permission code (not multi-perm list like credentials) — D-09 specifies one perm for all 6 endpoints
- DI for Phase 51 service uses `core.DeviceExecutor` + `core.DeviceInfoCollectionService` (RESEARCH §2.9 VERIFIED `core_services.go:17-29` + `core.go:276,284`)

---

### 6. `internal/api/v1/network/network_router.go` (route — MODIFY)

**Analog:** self — insertion point is `internal/api/v1/network/network_router.go:213` (after `SetupPortRouter(ports, core, exportHandler)`)

**Concrete excerpt — `network_router.go:205-214`:**

```go
// ==================== 端口状态管理路由（独立权限） ====================
ports := r.Group("/ports")
ports.Use(middleware.RequirePermissionsWithQuery([]string{
    "network:port:query",
}, middleware.OpsSelectorReadPerms, core))
{
    SetupPortRouter(ports, core, exportHandler)
}
```

**Delta:** insert one call inside the `{}` block, after `SetupPortRouter`:

```go
{
    SetupPortRouter(ports, core, exportHandler)
    SetupPortWriteRouter(ports, core)   // Phase 52: 写操作子组 /network/ports/write/*
}
```

**Why after `SetupPortRouter`:** no route collision (write sub-group is `/write/*`, parent group's routes are top-level). `exportHandler` not needed for write router.

---

### 7. `internal/core/db/migrations/menu_grant_helpers.go` (utility — CREATE)

**Analog (INSERT...SELECT role-menu idiom):** `internal/core/db/migrations/136_add_group_mapping_menu.sql:281-284`

```sql
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id FROM ...
```

**Analog (Go-side role-menu grant loop):** `internal/core/db/migrations/migration_144_vdi_granular_permissions.go:62-79` — uses check-then-insert in a transaction. **The Phase 52 helper improves on this** by doing it in a single `INSERT...SELECT...ON CONFLICT DO NOTHING` (idempotent, no check loop).

**D-08 locked SQL:**

```sql
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT rm.role_id, '<newMenuID>'::uuid
FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '<parentMenuName>'
ON CONFLICT DO NOTHING
```

**Delta — helper Go skeleton:**

```go
package migrations

import (
    "fmt"

    "gorm.io/gorm"
)

// GrantNewMenuToRolesHavingParent 把 newMenuID 精准授权给所有已持有父菜单(parentMenuName)的角色。
//
// 解决 antd 父子联动陷阱(memory migration-grant-new-menu-precision-helper):
// 仅 INSERT sys_menu 不 INSERT sys_role_menu 会被父联动带飞视觉但实际 checkedKeys
// 不含 → 路由不生成 → 链接 fallback dashboard。
//
// 幂等:ON CONFLICT DO NOTHING;只波及父已关联角色,admin 走超管旁路自动可见。
//
// 参数:
//   - db: 已连接的 *gorm.DB
//   - parentMenuName: 父菜单 menu_name(如 "端口状态")
//   - newMenuID: 新菜单 menu_id(UUID 字符串)
func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error {
    sql := fmt.Sprintf(`
        INSERT INTO sys_role_menu (role_id, menu_id)
        SELECT rm.role_id, '%s'::uuid
        FROM sys_role_menu rm
        JOIN sys_menu m ON rm.menu_id = m.id
        WHERE m.menu_name = '%s'
        ON CONFLICT DO NOTHING
    `, newMenuID, parentMenuName)
    return db.Exec(sql).Error
}
```

**Notes:**
- `newMenuID` and `parentMenuName` are validated/controlled (migration-internal seed values), so string interpolation is acceptable here. If planner prefers, use parameterized: `db.Exec("... WHERE m.menu_name = ? ... '%s'::uuid ...", parentMenuName, newMenuID)` — but PG `::uuid` cast with placeholder is awkward; explicit format is the project's migration idiom (see migration_201's `fmt.Sprintf` style).
- ON CONFLICT requires a unique constraint on `(role_id, menu_id)` — confirm `sys_role_menu` PK is composite `(role_id, menu_id)` (it is, per migration_144's reliance on dedup).

---

### 8. `internal/core/db/migrations/migration_202_port_write_audit.go` (migration — CREATE)

**Composite analog:**
- **F-type button menu shape:** `migration_200_fix_suggestion_config_seeds.go:171-207` (lines 187-201 the buttonMenu literal)
- **count-then-insert menu seed:** `migration_195_reconciliation_exception_rules_menu.go:40-92`
- **parent menu lookup by name:** `migration_200:99-106` (looks up "资产管理")
- **CREATE INDEX IF NOT EXISTS:** `migration_201_phase48_component_columns.go:88-106`
- **explicit `isPostgreSQL` branch:** `migration_201:45-51`

#### 8a. Function signature (copy from `migration_201:41`)

```go
func Migrate201Phase48ComponentColumns(db *gorm.DB) error {
    log.Println("Running migration 201: ...")
    if !isPostgreSQL(db) {
        // SQLite fallback (AutoMigrate only)
        return nil
    }
    // PostgreSQL branch
}
```

**Delta:** `Migrate202PortWriteAudit(db *gorm.DB) error` — under Path A (RESEARCH §1.2), GORM AutoMigrate builds `sys_port_write_audit` table from `models.PortWriteAudit`, so migration_202 only does: composite index naming safety + menu seed + helper call.

#### 8b. SQLite vs PostgreSQL split (copy from `migration_201:45-51`)

```go
if !isPostgreSQL(db) {
    // SQLite: GORM AutoMigrate (already registered in database.go) builds the table.
    // No menu seed/index needed in SQLite test path — Phase 52 unit tests assert via Go model.
    log.Println("Migration 202: non-PostgreSQL dialect, skip (table created by AutoMigrate)")
    return nil
}
```

`isPostgreSQL` is defined in `migration_mac_history.go:12-14` — already available in the migrations package.

#### 8c. Menu seed — count-then-insert + F-type button shape (copy from `migration_200:171-207`)

```go
// migration_200:171-207
for _, btn := range buttons {
    var existingCount int64
    if err := db.Model(&models.Menu{}).Where("perms = ?", btn.perms).Count(&existingCount).Error; err != nil {
        return err
    }
    if existingCount > 0 {
        continue
    }

    var btnParent models.Menu
    if err := db.Where("perms = ?", "asset:reconciliation:fix:list").First(&btnParent).Error; err != nil {
        log.Printf("... parent not found, skip: %v", err)
        continue
    }

    emptyPath := ""
    icon := "#"
    perms := btn.perms
    buttonMenu := &models.Menu{
        MenuName: btn.name,
        ParentID: &btnParent.ID,
        Path:     &emptyPath,
        MenuType: models.MenuTypeButton, // 'F'
        Visible:  models.VisibleHidden,  // 0
        Status:   models.MenuStatusNormal,
        Perms:    &perms,
        Icon:     &icon,
        OrderNum: btn.orderNum,
        Remark:   btn.remark,
    }
    if err := db.Create(buttonMenu).Error; err != nil {
        // ...
    }
}
```

**Delta vs analog (per D-06 + D-07 + RESEARCH §2.8):**
- Parent lookup by `menu_name = '端口状态'` (NOT '端口管理' — D-07/RESEARCH §2.8 VERIFIED)
- F-type button literal: `MenuName="端口配置"`, `Path=&emptyPath` (RESEARCH §3.2 recommends empty over D-06's "write" — both safe since F-type skips routeGenerator), `Visible=VisibleHidden(0)`, `Icon="#"`, `Perms="network:port:write"`
- **No `IsFrame`/`IsCache` fields** (RESEARCH §3.2 A7 — Go sys_menu has no such columns; memory `xingran-menu-no-java-fields`)
- **After menu insert: call helper** — `migrations.GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID)` (D-08) — this is the **delta** that analog migration_200 explicitly does NOT do (line 209 "不 INSERT sys_role_menu")

```go
// migration_202 menu seed (Phase 52 D-06 + D-08)
var existingCount int64
if err := db.Model(&models.Menu{}).Where("perms = ?", "network:port:write").Count(&existingCount).Error; err != nil {
    return err
}
if existingCount > 0 {
    log.Println("Migration 202: 端口配置 button menu 已存在,跳过 seed + grant")
    return nil
}

var parentMenu models.Menu
if err := db.Where("menu_name = ?", "端口状态").First(&parentMenu).Error; err != nil {
    log.Printf("Migration 202: 父菜单 '端口状态' 未找到,跳过 seed: %v", err)
    return nil  // or return err per planner preference
}

emptyPath := ""
icon := "#"
perms := "network:port:write"
menu := &models.Menu{
    MenuName: "端口配置",
    ParentID: &parentMenu.ID,
    Path:     &emptyPath,
    MenuType: models.MenuTypeButton,
    Visible:  models.VisibleHidden,
    Status:   models.MenuStatusNormal,
    Perms:    &perms,
    Icon:     &icon,
    OrderNum: 100,
    Remark:   "Phase 52: 端口写操作按钮权限(5 单端口 + 1 batch)",
}
if err := db.Create(menu).Error; err != nil {
    return fmt.Errorf("create 端口配置 menu failed: %w", err)
}

// D-08 精准授权:把新菜单授权给所有已持有 "端口状态" 父菜单的角色
if err := GrantNewMenuToRolesHavingParent(db, "端口状态", menu.ID); err != nil {
    log.Printf("Migration 202: WARNING GrantNewMenuToRolesHavingParent failed: %v", err)
    // 非阻断:菜单已 seed,管理员可手动授权
}
```

#### 8d. Composite index (defensive — copy from `migration_201:80-106`)

Under Path A, GORM AutoMigrate creates indexes from model tags (`idx_port_write_audit_device_port_created` + `idx_port_write_audit_created`). migration_202 may add nothing here if model tags suffice. **Optional defensive `CREATE INDEX IF NOT EXISTS`** matches migration_201's idempotent style — only add if planner wants belt-and-suspenders against GORM tag naming drift.

---

### 9. `internal/core/db/database.go` (config — MODIFY)

**Analog:** self — two insertion points.

#### 9a. AutoMigrate model list (line 308-391) — Path A

Insert `&models.PortWriteAudit{}` alongside other port models. Logical neighbor: `&models.DevicePortStatus{}` (line 328).

```go
&models.DevicePortStatus{},
&models.PortWriteAudit{},  // Phase 52: 端口写审计 append-only 表
```

#### 9b. Explicit migration call in postgres branch (line 411-417)

**Concrete excerpt — `database.go:411-417`:**

```go
if d.Type == "postgres" {
    if err := migrations.Migrate175ReconciliationPhysicalLink(d.DB); err != nil {
        applogger.Errorf("reconciliation 前置视图重建失败 (非阻断,留待下次启动): %v", err)
    }
    if err := migrations.Migrate176ReconciliationPhysicalMV(d.DB); err != nil {
        applogger.Errorf("reconciliation_normalized MV 重建失败 (非阻断,留待下次启动): %v", err)
    }
}
```

**Delta — add explicit Migrate202 call (per RESEARCH §1.2 VERIFIED, MigrateNNN functions are NOT auto-called after 260704-ne5 refactor; only 175/176 are called explicitly):**

```go
if d.Type == "postgres" {
    if err := migrations.Migrate175ReconciliationPhysicalLink(d.DB); err != nil {
        applogger.Errorf("reconciliation 前置视图重建失败 (非阻断,留待下次启动): %v", err)
    }
    if err := migrations.Migrate176ReconciliationPhysicalMV(d.DB); err != nil {
        applogger.Errorf("reconciliation_normalized MV 重建失败 (非阻断,留待下次启动): %v", err)
    }
    // Phase 52: 端口写审计表(由 AutoMigrate 建表) + 菜单 seed + 角色授权
    if err := migrations.Migrate202PortWriteAudit(d.DB); err != nil {
        applogger.Errorf("端口写审计迁移失败 (非阻断,留待下次启动): %v", err)
    }
}
```

**Why non-blocking (matching analog pattern):** migration_175/176 errors are logged but don't abort startup. Follow the same pattern — a failed menu seed should not block app boot.

---

## Shared Patterns

### operlog.Record (handler-side audit)

**Source:** `internal/utils/operlog/operlog.go:215-263` (Record function)
**Apply to:** All 6 handler write methods in `port_write_handler.go`

**Signature (5 fixed + variadic, VERIFIED line 215):**

```go
func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption)
```

**Available RecordOptions (VERIFIED — only 3 exist; `WithJsonResult` does NOT exist despite CLAUDE.md/CONTEXT mention):**
- `WithOperParam(s string)` (line 189) — **use this for audit_ids JSON** (Path C)
- `WithStatus(status int)` (line 196)
- `WithErrorMsg(msg string)` (line 203)

**Standard call shape (non-sensitive endpoint, no RecordWithBody):**

```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeStatus)
```

**Batch call with oper_param (Path C):**

```go
summaryJSON := buildBatchSummaryJSON(action, batchSize, result, auditIDs)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operlog.OperTypeBatch,
    operlog.WithOperParam(summaryJSON))
```

**Endpoint is NON-sensitive** (no password/secret/key/token fields) → use `Record`, NOT `RecordWithBody` (which is for endpoints that need to capture+mask the request body).

### RequirePermissions (group-level RBAC)

**Source:** `pkg/middleware/permission.go:200-226`
**Apply to:** `port_write_router.go` (one group-level mount)

**Signature (2-arg, VERIFIED line 200):**

```go
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc
```

**CRITICAL:** CLAUDE.md and 52-CONTEXT both write `RequirePermissions([]string)` (1-arg) — this is WRONG. Always pass `core` as 2nd arg.

### Migration registration (post-260704-ne5)

**Source:** `internal/core/db/database.go:296-421` (comment at 296 + AutoMigrate at 298 + explicit calls 411-417)
**Apply to:** `database.go` MODIFY task

Two-step:
1. Add `&models.PortWriteAudit{}` to `d.DB.Migrator().AutoMigrate(...)` list (line 308-391) — GORM builds table/columns from model tags
2. Add explicit `migrations.Migrate202PortWriteAudit(d.DB)` call in postgres branch (line 411-417) — because **MigrateNNN functions are NOT auto-called** (RESEARCH §1.2 VERIFIED)

### Menu seed + grant pattern (count-then-insert + helper)

**Source (seed):** `migration_195:40-92` + `migration_200:171-207` (F-type button shape)
**Source (grant):** new `menu_grant_helpers.go` (this phase) wraps `INSERT...SELECT...ON CONFLICT DO NOTHING` idiom from `136_add_group_mapping_menu.sql:281-284`
**Apply to:** `migration_202_port_write_audit.go`

Strict order:
1. Lookup parent by `menu_name` (`'端口状态'` — D-07, NOT `'端口管理'`)
2. count-then-insert the F-type button menu (Visible=0, Path="", Icon="#", Perms="network:port:write")
3. Call `GrantNewMenuToRolesHavingParent(db, "端口状态", newMenu.ID)` — the **delta** vs analog migration_200 which explicitly does NOT grant

### Sentinel error → HTTP translation

**Source:** `internal/api/v1/asset/fix_suggestion_handler.go:121-129` (string-equality error mapping pattern — Phase 52 uses `errors.Is` instead since Phase 51 uses sentinel `var Err...`)
**Apply to:** All 6 handler methods in `port_write_handler.go`

Use `errors.Is(err, portwrite.ErrXxx)` switch (see §4d table). Two distinct failure paths: sentinel (4xx, no audit) vs `PortResult.Status=="failed"` (200, audit row).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/core/db/migrations/menu_grant_helpers.go` | utility (helper) | batch INSERT...SELECT | No existing project helper exactly matches; closest idiom is raw SQL in `136_add_group_mapping_menu.sql:281-284` and the Go loop in `migration_144:62-79`. **The helper is the consolidation of this idiom** (memory `migration-grant-new-menu-precision-helper` motivated extraction). Planner has RESEARCH §1.1 + D-08 SQL locked — straight implementation. |

All other files have direct analogs.

---

## Metadata

**Analog search scope:**
- `pkg/permission/` (config.go)
- `pkg/middleware/` (permission.go)
- `internal/utils/operlog/` (operlog.go)
- `internal/api/v1/asset/` (fix_suggestion_handler.go, reconciliation_handler.go)
- `internal/api/v1/network/` (port_router.go, network_router.go)
- `internal/core/db/` (database.go)
- `internal/core/db/migrations/` (migration_144, migration_195, migration_200, migration_201, migration_mac_history, 136_add_group_mapping_menu.sql)
- `internal/models/` (base.go, log.go, menu.go, device_port_status.go)
- `internal/services/` (oper_log_service.go, asset/cache_keys.go, system/cache_keys.go)
- `internal/services/portwrite/` (port_write_service.go, batch_orchestrator.go)

**Files scanned:** 16 source files + 3 planning docs (CONTEXT, RESEARCH, CLAUDE.md)

**Pattern extraction date:** 2026-07-07

**Key RESEARCH-confirmed corrections embedded above (planner MUST honor):**
1. `RequirePermissions` is **2-arg** (`permissions []string, core *core.Core`) — CONTEXT/CLAUDE.md wrote 1-arg (WRONG)
2. **MigrateNNN functions are NOT auto-called** after 260704-ne5 refactor — `database.go` needs explicit `migrations.Migrate202PortWriteAudit(d.DB)` call in postgres branch
3. **D-13 audit↔operlog = Path C** (audit_ids embedded in operlog oper_param, NOT `WithOperID` in operlog package) — adding WithOperID would break `regression_test.go` locks and requires touching Phase 34 stable surface
4. **Only 3 RecordOptions exist**: `WithOperParam`, `WithStatus`, `WithErrorMsg`. `WithJsonResult` is a CONTEXT/CLAUDE.md hallucination (only the `recordConfig.jsonResult` field exists, no exported setter)
5. **Parent menu name = "端口状态"** (NOT "端口管理" — D-07/RESEARCH §2.8 VERIFIED via archive/053_fix_menu_paths_unified.sql:185)
6. **F-type button Path = ""** (empty, NOT D-06's "write") — RESEARCH §3.2 A6 recommends aligning with migration_200 convention; both safe since F-type skips routeGenerator
7. **Go sys_menu has NO `is_frame`/`is_cache` columns** — they live in `Meta` JSONB (RESEARCH §3.2 A7 / memory `xingran-menu-no-java-fields`)
