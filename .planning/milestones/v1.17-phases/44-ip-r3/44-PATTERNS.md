# Phase 44: 置信度评分 + IP 段例外 (R3) - Pattern Map

**Mapped:** 2026-06-28
**Files analyzed:** 18 (新建 9 + 修改 9)
**Analogs found:** 17 / 18 (1 项 greenfield: GiST inet_ops 索引无项目内先例，但有 PG 官方 + research 代码示例兜底)

> **命名澄清（沿用 44-CONTEXT/RESEARCH）**：Phase 44 标题里的"置信度评分"已在 Phase 42 (RECON-03) 落地。R3 真正交付 **IP 段例外规则引擎 + 告警降噪**。Planner 不要在 R3 重写评分函数。
>
> **拆分提示（与 ROADMAP 一致）**：建议 2 plans——Plan 44-01 落地"规则引擎 + CRUD + admin 页 + 命中测试"（EXCEPTION-01/02/04 + SC 1-4/6/7/10），Plan 44-02 落地"Excel + 过期清理 cron + 降噪基线/对比 + 转单 SQL"（EXCEPTION-03 + SC 5/8/9）。下文每节标注归属。

---

## File Classification

| # | New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|---|
| 1 | `internal/core/db/migrations/migration_174_reconciliation_exception_gist.go` (新建) | migration | batch (DDL) | `internal/core/db/migrations/migration_168_reconciliation_tables.go:139-156` (DO$$ partial unique index 模式) | role-match (项目无 GiST 先例) |
| 2 | `internal/core/db/database.go:483` (修改) | config | batch | 同文件 `:455-483` migration 168-173 链式注册 | exact |
| 3 | `internal/services/asset/reconciliation_exception_matcher.go` (新建) | service (pure fn) | transform (CIDR 匹配 + actions 合并) | `internal/middleware/apikey.go:110-141` (`net.ParseCIDR` + `ipNet.Contains`) | role-match |
| 4 | `internal/services/asset/reconciliation_exception.go` (修改，扩展 CRUD + 校验 + MatchTest) | service | CRUD | `internal/services/asset/reconciliation_exception.go:60-146` (现有 List/GetByID skeleton) | exact |
| 5 | `internal/services/asset/reconciliation_detection.go` (修改，DetectLayer3 插 Layer 3.5) | service | batch (循环 transform) | 同文件 `:207-332` (DetectLayer3 现有循环 + guard 1/2) | exact |
| 6 | `internal/services/asset/reconciliation_baseline.go` (新建) | service | request-response (snapshot/compare) | `internal/services/system/config_service.go:201-211` (GetByKey) + `internal/services/asset/reconciliation_statistics.go` (COUNT 端点) | role-match (组合) |
| 7 | `internal/scheduler/reconciliation_tasks.go` (修改，填 cleanupExpiredExceptions + 转单 SQL 加 no_workorder) | scheduler | batch (cron) | 同文件 `:42-102` (单 taskType 分发) + `:188-194` (createWorkorderBySeverity SQL) | exact |
| 8 | `internal/services/operations/excel_config.go` (修改，加 `reconciliationExceptionRule` 条目) | config | batch (file I/O → upsert) | 同文件 `:50-152` (building/floor/workstation ExcelConfig) | exact |
| 9 | `internal/api/v1/asset/reconciliation_exception_handler.go` (修改，加 Create/Update/Delete/Test/CleanupNow/Baseline/Compare) | controller | request-response | 同文件 `:33-67` (现有 ListRules/GetRuleByID) + `internal/api/v1/system/config_handler.go:101-121,322-352` (operlog CRUD) | exact (合并 2 analog) |
| 10 | `internal/api/v1/asset/reconciliation_exception_router.go` (修改，加 `/exception-rule/{create,update,delete,test}` + `/baseline/*`) | route | request-response | 同文件 `:17-23` (现有路由注册) | exact |
| 11 | `internal/api/v1/asset/reconciliation_handler.go` (修改，加 `ModuleReconciliationExceptionRule` 常量) | controller | config | 同文件 `:14-19` (`ModuleReconciliation` 常量) | exact |
| 12 | `xingran-react-frontend/src/pages/asset/reconciliation/exception-rules/index.tsx` (新建) | frontend page | request-response (CRUD UI) | `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` (同模块列表页) + `xingran-react-frontend/src/pages/ad-domain/accounts/index.tsx` (统计卡片+列表+Modal CRUD) | role-match (组合) |
| 13 | `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` (修改，silence 默认过滤 + "显示已静默"开关) | frontend page | request-response | 同文件 (现有筛选表单) | exact |
| 14 | `xingran-react-frontend/src/components/asset/reconciliation/ExceptionRuleForm.tsx` (新建) | frontend component | form I/O | `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` (Form + Modal 模式) | role-match |
| 15 | `xingran-react-frontend/src/components/asset/reconciliation/MatchTestPanel.tsx` (新建) | frontend component | request-response (测试 UI) | no analog — greenfield (合并卡片 + 命中规则列表形态，参考 hooks/useExceptionList.ts 的 useQuery 模式) | no analog |
| 16 | `xingran-react-frontend/src/lib/assetApi.ts` (修改，加 exceptionRule.{create,update,delete,test} + baseline.{snapshot,compare}) | frontend api client | request-response | 同文件 `reconciliationApi` 现有定义 (research CITED `:70` ExceptionListParams) | exact |
| 17 | `xingran-react-frontend/src/lib/queryKeys.ts` (无改动 — 已注册 ruleList/ruleDetail/matchTest) | — | — | — | INFRA-05 已就位，无需改 |
| 18 | 测试桩 ×8 (Wave 0，见 VALIDATION.md `:70-77`) | test | unit/integration | `internal/services/asset/reconciliation_*_test.go` (现有 Phase 42/43 测试) + `internal/utils/operlog/regression_test.go` (operlog 回归) | exact |

---

## Pattern Assignments

### 1. `migration_174_reconciliation_exception_gist.go` (新建, migration, batch DDL) — **Plan 44-01**

**Analog:** `internal/core/db/migrations/migration_168_reconciliation_tables.go:139-156` (DO$$ + `pg_indexes` IF NOT EXISTS 模式)

**核心模式（DO$$ 幂等 + 显式命名，规避 xingran-gorm-sql-constraint-naming-conflict）：**
```go
// Source: migration_168:139-156 (CITED)
partialUniqueSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_recon_asset_type_open'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_recon_asset_type_open
                 ON sys_data_reconciliation (asset_id, conflict_type)
                 WHERE resolved_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
`
if err := db.Exec(partialUniqueSQL).Error; err != nil {
    return fmt.Errorf("创建 partial unique index 失败: %w", err)
}
```

**R3 套用（3 段：GiST 索引 + 2 CHECK 约束）**：GiST 索引查 `pg_indexes`；CHECK 约束查 `pg_constraint`（非 pg_indexes）。完整 SQL 见 44-RESEARCH.md `:644-725`（已给 verbatim）。**强制约束**：
- 纯 SQL `DO $$ ... EXECUTE 'ALTER TABLE ... ADD CONSTRAINT chk_recon_exc_xxx'`，**禁止** GORM `check:` tag（Pitfall 1）
- `CREATE INDEX IF NOT EXISTS` 幂等 + model 保持 `gorm:"type:cidr;column:ip_range"`，**不加** `gorm:"index"`（Pitfall 6 防 AutoMigrate 误建 btree）
- 文件头加 `if !isPostgreSQL(db) { return nil }` 兜底（SQLite 跳过）
- 命名规范：`idx_recon_exc_active_range` / `chk_recon_exc_actions` / `chk_recon_exc_severity_override`（参照 legacy `chk_building_org_id_is_uuid` 模式，RESEARCH §State of the Art 已锁定）

**CHECK 白名单**：
- `chk_recon_exc_actions`: `exception_actions <@ ARRAY['no_alert','no_notice','no_workorder','skip_severity','silence']`
- `chk_recon_exc_severity_override`: `severity_override IS NULL OR severity_override IN ('low','medium','high')`（**不含 critical**，Pitfall 8）

---

### 2. `database.go` migration 链式注册 — **Plan 44-01**

**Analog:** `internal/core/db/database.go:455-483` (现有 168-173 链式注册)

**核心模式：**
```go
// Source: database.go:455-458 (CITED)
// Phase 42 R1: 资产对账观测底座 — 主表 + 物化视图 + partial unique index
if err := migrations.Migrate168ReconciliationTables(d.DB); err != nil {
    applogger.Errorf("Phase 42 R1 reconciliation tables 迁移失败: %v", err)
}
```

**R3 套用（在 :483 Migrate173 之后追加，参照链式模式）**：
```go
// Phase 44 R3: sys_reconciliation_exception GiST inet_ops 索引 + CHECK 约束
if err := migrations.Migrate174ReconciliationExceptionGist(d.DB); err != nil {
    applogger.Errorf("Phase 44 R3 reconciliation GiST/CHECK 失败: %v", err)
}
```

**关键约束（项目记忆 xingran-migrations-no-sql-autoloader）**：migrations/*.sql 是孤立文件不会被加载，必须用 `migration_NNN_*.go` 函数显式调用并加入此 AutoMigrate 块。

---

### 3. `reconciliation_exception_matcher.go` (新建, service 纯函数, transform) — **Plan 44-01**

**Analog:** `internal/middleware/apikey.go:110-141` (`isIPAllowed` — 项目内唯一现成 CIDR 匹配)

**核心模式（CIDR 解析 + Contains）：**
```go
// Source: apikey.go:110-141 (CITED, D-R3-A1-03 锁定复用)
func isIPAllowed(clientIP string, whitelist []string) bool {
    ip := net.ParseIP(clientIP)
    if ip == nil { return false }
    for _, allowed := range whitelist {
        if strings.Contains(allowed, "/") {
            _, ipNet, err := net.ParseCIDR(allowed)  // ← R3 复用此行
            if err != nil { continue }
            if ipNet.Contains(ip) {                   // ← R3 复用此行
                return true
            }
        }
        // ...
    }
    return false
}
```

**R3 套用要点（D-R3-A1-03 + A2-01/02/03，RESEARCH §Pattern 1/2 已给 verbatim 代码 :259-354）**：
- `compiledRule` struct = `models.SysReconciliationException` + 预编译的 `*net.IPNet`
- `preloadActiveRules(db)` 一次 `SELECT ... WHERE is_active=0 AND deleted_at IS NULL`，循环前调用，**循环内零 DB 查询**（A1-03 性能架构）
- `matchException(rules, assetIP, assetUserID, conflictType)` 返回 `(ruleID, appliedActions pq.StringArray, finalSeverity, isSilence)` — 多规则遍历取并集（A2-01）
- `mergeActions(originalSeverity, matched)` 纯函数：step1 skip_severity 降级链 → step2 severity_override 取最低 → step3 actions UNION → step4 isSilence 判定（A2-02）
- `applySkipSeverity(s)`：`severityOrder = {low:0, medium:1, high:2, critical:3}`，降一级 low 不再降
- 解析失败的规则 `logrus.Warnf` 跳过，不阻塞检测（沿用 apikey.go 的 `continue` 容错）
- `conflict_types` 空数组 → 匹配全部 B-F（A3-02，service 层 enforce，model 无 DEFAULT）
- `dept`/`user` scope 双条件（A3-01）：IP 命中 **AND** assetUserID ∈ scope_id

**导入模式（参照 reconciliation_detection.go:1-17）**：
```go
import (
    "net"
    "github.com/lib/pq"                    // pq.StringArray
    "github.com/sirupsen/logrus"
    "gorm.io/gorm"
    "github.com/xingran-next/xingran-go-backend/internal/models"
)
```

---

### 4. `reconciliation_exception.go` (修改, service, CRUD) — **Plan 44-01**

**Analog:** 同文件 `:60-146` (现有 R1 skeleton List/GetByID)

**核心模式（interface + 私有 impl + 构造函数，CLAUDE.md Handler-Service 强约束）：**
```go
// Source: reconciliation_exception.go:48-68 (CITED)
type ReconciliationExceptionService interface {
    List(ctx context.Context, params *ExceptionRuleListParams) (*base.PageResult, error)
    GetByID(ctx context.Context, id string) (*models.SysReconciliationException, error)
}

type reconciliationExceptionServiceImpl struct {
    db *gorm.DB
}

func NewReconciliationExceptionService(db *gorm.DB) ReconciliationExceptionService {
    return &reconciliationExceptionServiceImpl{db: db}
}
```

**R3 扩展（interface 加 5 方法）**：
- `Create(ctx, *CreateExceptionRuleRequest) (*models.SysReconciliationException, error)`
- `Update(ctx, id string, *UpdateExceptionRuleRequest) error`
- `Delete(ctx, id string) error` — 软删除（`db.Delete` 填 deleted_at，与 D-R3-A4-03 区分：cron 软停用只改 is_active）
- `MatchTest(ctx, ip, userID, deptID string) (*MatchTestResult, error)` — 用 GiST `>>` SQL（A3-03）
- 可选：`CleanupNow(ctx) (int64, error)`（运维手动触发过期清理）

**List 查询模式（参照 :94-119 基础 query + 过滤 + Count + Find）**：
```go
// Source: reconciliation_exception.go:94-119 (CITED)
query := s.db.WithContext(ctx).
    Model(&models.SysReconciliationException{}).
    Where("sys_reconciliation_exception.deleted_at IS NULL")
if params.IsActive != nil {
    query = query.Where("sys_reconciliation_exception.is_active = ?", *params.IsActive)
}
var total int64
if err := query.Count(&total).Error; err != nil { return nil, err }
var list []models.SysReconciliationException
if err := query.Order("...created_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil { ... }
```

**校验函数（service 层 enforce，V5 Input Validation）**：
- `ValidateCIDR(ipRange string) error` → `net.ParseCIDR`，失败拒绝（T-CIDR-Inject 缓解）
- `ValidateActions(actions pq.StringArray) error` → 白名单 `["no_alert","no_notice","no_workorder","skip_severity","silence"]`，与 `chk_recon_exc_actions` CHECK 同步（Pitfall 8）
- `ValidateSeverityOverride(sev *string) error` → 白名单 `["low","medium","high"]`（**不含 critical**，与 CHECK 同步）
- `ValidateReason(reason string) error` → `len >= 10`（告警风暴缓解，RESEARCH §Security Domain）

**MatchTest GiST SQL 模式（D-R3-A3-03，GORM 占位符防 SQL 注入）**：
```go
// Source: 44-RESEARCH.md:741-749 (verbatim)
err := s.db.WithContext(ctx).
    Where(`ip_range >> ?::inet AND is_active = ? AND deleted_at IS NULL AND scope_type = ?`,
        ip, 0, "global").
    Find(&matched).Error
```

---

### 5. `reconciliation_detection.go` (修改, service, batch transform) — **Plan 44-01**

**Analog:** 同文件 `:207-332` (DetectLayer3 现有循环)

**核心插入点（D-R3-A1-02，RESEARCH §Code Examples :607-641 已给 verbatim）**：
- 位置：guard 2 (24h 节流 `:258-261`) 之后、脏数据防御 (`:263-271`) 之前
- 循环前（rows 加载后 `:213` 附近）调 `activeRules := s.preloadActiveRules()` 一次
- 循环内 guard 2 之后插 Layer 3.5：调 `matchException(activeRules, *row.AssetIP, assetUserID, conflictType)`，命中时填 `exceptionRuleID` + `appliedActions` + 用 `matchedSeverity` 替换 `severity`

**现有 INSERT 构造点（R3 填入新字段，:305-316）**：
```go
// Source: reconciliation_detection.go:305-316 (CITED)
rec := &models.SysDataReconciliation{
    AssetID:         row.AssetID,
    ConflictType:    conflictType,
    Severity:        severity,         // ← R3 改为 matchedSeverity
    // ...
    AssetIP:         row.AssetIP,
    DetectedAt:      time.Now(),
    // R3 新增 ↓
    // ExceptionRuleID:  exceptionRuleID,
    // AppliedActions:   appliedActions,
}
```

**返回值扩展提示**：当前签名 `(int, int, int, int, error)` = (inserted, skipped, skippedSilence, skippedThrottle)。R3 命中例外的记录仍计入 `inserted`（D-R3-A1-01 silence 仍写表），**无需新增返回值**，但可在 logrus 加 `exceptionHit` 计数（可选）。

**导入需补**：`"github.com/lib/pq"`（appliedActions 类型）+ `"net"`（CIDR 解析，若 matchException 放本文件则需；放独立 matcher 文件则只 import matcher 包）。

---

### 6. `reconciliation_baseline.go` (新建, service, request-response) — **Plan 44-02**

**Analog（组合）**：
- `internal/services/system/config_service.go:201-211` (`GetByKey` 读写 sys_config)
- `internal/services/asset/reconciliation_statistics.go` (COUNT 端点，**禁止用 list.length**)

**sys_config 读写模式（D-R3-A4-01）**：
```go
// Source: config_service.go:201-211 (CITED)
func (s *configService) GetByKey(ctx context.Context, configKey string) (*models.Config, error) {
    var config models.Config
    err := s.db.WithContext(ctx).Where("config_key = ?", configKey).First(&config).Error
    // ...
}
```

**R3 套用**：
- config_key: `asset.reconciliation.baseline`
- config_value: JSON `{"snapshot_at":"...","total_exceptions":N,"total_workorders":N,"critical_exceptions":N}`
- `Snapshot(ctx, *BaselineSnapshot) error` — 用 `config_service.Create` 或 `Update`（key 存在则 update）
- `Compare(ctx) (*BaselineCompareResult, error)` — 读 baseline + 现场独立 COUNT，算下降 %

**独立 COUNT 模式（Pitfall 5，规避 stat-cards-from-list-length-capped-at-100）**：
```go
// 禁止：list := ListExceptions(pageSize:99999); len(list)  ← 被 MaxPageSize=100 钳制
// 必须：
var totalExceptions int64
s.db.WithContext(ctx).Model(&models.SysDataReconciliation{}).
    Where("deleted_at IS NULL").Count(&totalExceptions)
```

---

### 7. `reconciliation_tasks.go` (修改, scheduler, batch cron) — **Plan 44-02**

**Analog（同文件双 anchor）**：
- `:42-102` 单 taskType `"reconciliation"` + `params["param"]` switch 分发
- `:188-194` `createWorkorderBySeverity` SQL

**(a) cleanupExpiredExceptions case 填实（D-R3-A4-03，软停用不删）**：
```go
// Source: reconciliation_tasks.go:78-79 (现有 placeholder, CITED)
case "cleanupExpiredExceptions":
    applogger.Infof("[reconciliation:cleanupExpiredExceptions] R1 placeholder, R3 真实实现")
// R3 改为（44-RESEARCH.md:793-801 verbatim）：
case "cleanupExpiredExceptions":
    result := db.WithContext(ctx).
        Model(&models.SysReconciliationException{}).
        Where("expires_at IS NOT NULL AND expires_at < NOW() AND is_active = ? AND deleted_at IS NULL", 0).
        Update("is_active", 1)  // Status Convention: 0=启用 → 1=停用
    applogger.Infof("[reconciliation:cleanupExpiredExceptions] 软停用 %d 条过期例外规则", result.RowsAffected)
```
**幂等性**：WHERE 含 `is_active=0`，第二次 cron 调用 rowsAffected=0（Pitfall 4 防外键断链）。

**(b) 转单 SQL 加 no_workorder 过滤（D-R3-A1-02）**：
```go
// Source: reconciliation_tasks.go:190-194 (现有 SQL, CITED)
db.WithContext(ctx).
    Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL", severity).
    Order("detected_at ASC").Limit(limit).Find(&exceptions)
// R3 改为（追加一条件，44-RESEARCH.md:807-817 verbatim）：
db.WithContext(ctx).
    Where("severity = ? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL "+
          "AND 'no_workorder' != ANY(applied_actions)", severity).
    Order("detected_at ASC").Limit(limit).Find(&exceptions)
```

**sys_job 已 seed（无需新建 job 记录）**：`对账-例外规则清理` cron `0 0 3 * * *` 已在 `:127-131` 注册，InvokeTarget `reconciliation:cleanupExpiredExceptions` 已映射 `:232-233`。R3 仅填 case 实现。

---

### 8. `excel_config.go` (修改, config, batch upsert) — **Plan 44-02**

**Analog:** 同文件 `:101-152` (building/floor/workstation ExcelConfig)

**核心模式（Columns 顺序 = Excel 列序，Reference 名称→UUID，UpsertKey 需 DBField）**：
```go
// Source: excel_config.go:109-116 (building, CITED)
Columns: []ExcelColumn{
    {Field: "name", Header: "楼宇名称", Required: true, MaxLength: 100, UpsertKey: true},
    {Field: "address", Header: "地址", MaxLength: 200, DBField: "position_desc"},
    {Field: "orgName", Header: "所属机构名称/编码", Required: true, MaxLength: 100,
     Reference: "sys_dept.dept_code", DBField: "org_id"},  // 名称→UUID 解析
    // ...
}
```

**R3 套用（D-R3-A4-02，44-RESEARCH.md:823-852 已给 verbatim 完整 Columns）**。**3 个项目记忆强约束**：
1. **xingran-excel-import-column-position-matching**：Columns 顺序严格 = 模板列序（`name / ip_range / conflict_types / exception_actions / severity_override / scope_type / scope_name / expires_at / reason`）
2. **xingran-excel-import-upsertkey-needs-dbfield**：`name` 列 `UpsertKey: true` **必须**配 `DBField: "name"`，否则冲突键失效报 500
3. **xingran-excel-import-route-conflict**：不要在 `router.go` 预注册 `/asset/reconciliation/*`，由 `SetupReconciliationExceptionRouter` 自管（已规避）

**TEXT[] + 条件 Reference 两个 OPEN QUESTION（A3/A4，RESEARCH §Open Questions）**：
- `conflict_types` / `exception_actions` 是 TEXT[]，Excel 单元格是逗号分隔字符串 — planner 需 PoC `validateAndParseRow` 是否自动转 `pq.StringArray`，否则方案 B 后处理 UPDATE
- `scope_name` 按 `scope_type` 动态走不同表（dept→sys_dept.dept_name / user→sys_user.username / global→留空）— 现有 Reference 是静态配置**不支持条件分支**，**必须** service 层 post-process（ImportData 返回 AffectedKeys 后逐条 UPDATE scope_id）。Planner 必须在 Plan 44-02 Task 显式实现此步骤。

---

### 9. `reconciliation_exception_handler.go` (修改, controller, request-response) — **Plan 44-01**

**Analog（组合 2 source）**：
- 同文件 `:33-67` (现有 ListRules/GetRuleByID bind + service + response 模式)
- `internal/api/v1/system/config_handler.go:101-121,322-352` (Create/Update/Delete + operlog)

**核心模式（handler success path 末尾 operlog.Record → response.Success，CLAUDE.md 强制约定）**：
```go
// Source: config_handler.go:101-121 (CITED, Create)
func (h *ConfigHandler) Create(c *gin.Context) {
    var req requests.ConfigCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
        return
    }
    // ... 业务校验 ...
    if err := h.service.Create(c.Request.Context(), &req); err != nil {
        response.Error(c, err)
        return
    }
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "参数管理", operlog.OperTypeCreate)
    response.Success(c, gin.H{"message": "创建成功"})
}
// Update → operlog.OperTypeUpdate ; Delete → operlog.OperTypeDelete
```

**现有 handler 骨架（reconciliation_exception_handler.go:17-31, WithCore 模式）**：
```go
// Source: reconciliation_exception_handler.go:17-31 (CITED)
type ReconciliationExceptionHandler struct {
    service asset.ReconciliationExceptionService
    core    *core.Core
}
func NewReconciliationExceptionHandler(svc asset.ReconciliationExceptionService) *ReconciliationExceptionHandler {
    return &ReconciliationExceptionHandler{service: svc}
}
func (h *ReconciliationExceptionHandler) WithCore(core *core.Core) *ReconciliationExceptionHandler {
    if h != nil { h.core = core }
    return h
}
```

**R3 扩展（6 handler，全部 success path 末尾调 operlog）**：
- `CreateRule` → `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeCreate)`
- `UpdateRule` → `OperTypeUpdate`
- `DeleteRule` → `OperTypeDelete`
- `TestRule` (命中测试) → **读操作不调 operlog**（参照现有 ListRules/GetRuleByID 无 operlog）
- `EnableRule`/`DisableRule`（可选）→ `OperTypeEnable`/`OperTypeDisable` 或 `OperTypeStatus`
- `SnapshotBaseline`/`CompareBaseline` → `OperTypeUpdate`（snapshot 改 sys_config）

**ModuleReconciliationExceptionRule 常量（文件 11）**：加在 `reconciliation_handler.go:19` 旁，值 `"资产对账-例外规则"`（Phase 42 D-16 锁定）。**operlog 回归测试** `internal/utils/operlog/regression_test.go` 自动守护 25 OperType + 11 敏感关键词，新增 module 常量不破坏（module 是自由字符串，不在回归锁定范围）。

---

### 10. `reconciliation_exception_router.go` (修改, route, request-response) — **Plan 44-01**

**Analog:** 同文件 `:17-23` (现有路由注册)

**核心模式（构造 service → handler → 注册 POST 路由，CLAUDE.md Route Pattern）**：
```go
// Source: reconciliation_exception_router.go:17-23 (CITED)
func SetupReconciliationExceptionRouter(r *gin.RouterGroup, core *core.Core) {
    svc := asset.NewReconciliationExceptionService(core.DB.GetDB())
    handler := NewReconciliationExceptionHandler(svc).WithCore(core)
    r.POST("/exception-rule/list", handler.ListRules)
    r.POST("/exception-rule/:id", handler.GetRuleByID)
}
```

**R3 扩展（加 6 路由 + 权限中间件，V4 Access Control）**：
```go
r.POST("/exception-rule/create", middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core), handler.CreateRule)
r.POST("/exception-rule/:id/update", middleware.RequirePermissions([]string{"asset:reconciliation:exception:update"}, core), handler.UpdateRule)
r.POST("/exception-rule/:id/delete", middleware.RequirePermissions([]string{"asset:reconciliation:exception:delete"}, core), handler.DeleteRule)
r.POST("/exception-rule/test", middleware.RequirePermissions([]string{"asset:reconciliation:exception:test"}, core), handler.TestRule)
r.POST("/baseline/snapshot", handler.SnapshotBaseline)
r.POST("/baseline/compare", handler.CompareBaseline)
```
**RequirePermissions 签名**：`pkg/middleware/permission.go:200` `RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc`（CITED）。list/:id 路由现有无权限中间件（R1 skeleton），R3 视情况补 `:list`（参照 xingran-perm-namespace-split-readonly-page 教训：读路径可放宽）。

---

### 12. `exception-rules/index.tsx` (新建, frontend page, CRUD UI) — **Plan 44-01**

**Analog（组合）**：
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` (同模块列表页结构：Form 筛选 + Table + 分页 + useDict + useServerSort)
- `xingran-react-frontend/src/pages/ad-domain/accounts/index.tsx` (Phase 36 admin 页：统计卡片 + 列表 + Modal CRUD，CLAUDE.md §AD Service Account Pool 引用)

**现有同模块 imports 模式（exceptions/index.tsx:25-62, CITED）**：
```tsx
import { useMemo, useState, useEffect, useCallback } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Table, Form, Input, Select, DatePicker, Button, Space, Tag, Card, Empty, App, Modal } from "antd";
import { useQueryClient } from "@tanstack/react-query";
import { useExceptionList } from "@/hooks/useExceptionList";   // ← R3 仿写 useExceptionRuleList
import { useDict } from "@/hooks/useDict";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import { reconciliationApi, type ExceptionListParams } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";                    // ← reconciliation.ruleList 已注册
```

**R3 admin 页结构（D-R3-A2-03 + A4-01）**：
1. 顶部统计卡片（复用 Phase 36 accounts 模式）：总规则数 / 启用数 / 命中数（读 ExceptionRuleStats 端点，reconciliation_statistics.go:77 已就位）
2. 筛选表单：name 模糊 + is_active 筛选 + scope_type 筛选
3. Table：name / ip_range / actions(Tag 多色) / severity_override / scope / expires_at / is_active(开关) / 操作(编辑/删除/命中测试)
4. Modal 表单（→ ExceptionRuleForm 组件，文件 14）
5. 命中测试按钮 → 抽屉/Modal（→ MatchTestPanel 组件，文件 15）
6. "记录当前为基线" 按钮（D-R3-A4-01，调 baseline.snapshot）

**useEffect 稳定性（CLAUDE.md Frontend/React Best Practices 强约束）**：参照 exceptions/index.tsx 注释 `:21-23`——params 用 `useMemo + JSON.stringify` 稳定，配 `keepPreviousData` 翻页不闪烁。

---

### 13. `exceptions/index.tsx` (修改, frontend page) — **Plan 44-01**

**Analog:** 同文件 (现有筛选表单)

**R3 改造（D-R3-A1-01，silence 默认过滤）**：
- `ExceptionListParams` 加 `showSilenced?: boolean`（默认 false）
- 筛选表单加 Switch "显示已静默"
- 列表 hook 透传 `showSilenced` 到后端（后端 ListExceptions SQL 加 `WHERE NOT ('silence' = ANY(applied_actions))` 兜底，44-RESEARCH.md:774-787 verbatim）

---

### 15. `MatchTestPanel.tsx` (新建, frontend component) — **Plan 44-01**

**Analog:** no analog — greenfield（合并卡片 + 命中规则列表形态在项目内无先例）

**最近参考**：
- useQuery 模式：`xingran-react-frontend/src/hooks/useExceptionList.ts`（queryKeys.reconciliation.matchTest 已注册）
- 卡片+列表组合：可参考 dashboard 的统计卡片 + 列表组合（`xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx`）

**R3 实现要点（D-R3-A3-03 + A2-03）**：
- 输入区：IP/CIDR Input（必填）+ user Select（可选）+ dept Select（可选）
- 结果区顶部：合并卡片（finalActions Tag union + finalSeverity + isSilence Badge + needsUserDept 徽标）
- 结果区下方：命中规则 Table（name / IP范围 / actions / override / scope / expires / reason）
- 未指定 user/dept 时 dept/user scope 规则显示"需指定"徽标（A3-03）

---

### 16. `assetApi.ts` (修改, frontend api client) — **Plan 44-01/44-02**

**Analog:** 同文件 `reconciliationApi` 现有定义（research CITED `:70` ExceptionListParams）

**R3 扩展（加 exceptionRule + baseline 命名空间，复用 `post` wrapped 函数，CLAUDE.md Frontend API Calling 强约束）**：
```ts
// 仿现有 reconciliationApi.exception.list 模式
exceptionRule: {
    list: (params) => post('/asset/reconciliation/exception-rule/list', params),
    getById: (id) => post(`/asset/reconciliation/exception-rule/${id}`),
    create: (data) => post('/asset/reconciliation/exception-rule/create', data),
    update: (id, data) => post(`/asset/reconciliation/exception-rule/${id}/update`, data),
    delete: (id) => post(`/asset/reconciliation/exception-rule/${id}/delete`),
    test: (data) => post('/asset/reconciliation/exception-rule/test', data),
},
baseline: {
    snapshot: () => post('/asset/reconciliation/baseline/snapshot'),
    compare: () => post('/asset/reconciliation/baseline/compare'),
}
```

---

### 18. 测试桩 ×8 (Wave 0) — **Plan 44-01/44-02**

**Analog（组合）**：
- `internal/services/asset/reconciliation_*_test.go` (现有 Phase 42/43 测试结构)
- `internal/utils/operlog/regression_test.go` (operlog 回归，**已存在**，自动覆盖 R3 新 module 常量不破坏 25 OperType)

**8 个测试桩清单（VALIDATION.md:70-77 verbatim）**：
| 测试文件 | 覆盖 | 测试类型 |
|---|---|---|
| `reconciliation_exception_matcher_test.go` | EXCEPTION-01/02 (matchException + mergeActions + applySkipSeverity 纯函数) | unit |
| `reconciliation_exception_test.go` 扩展 | EXCEPTION-01/02/04 (Create/Update/Delete/MatchTest + ValidateCIDR/Actions) | unit + integration |
| `reconciliation_detection_test.go` 扩展 | SC 10 (Layer 3.5 命中例外仍写表) | integration |
| `reconciliation_service_test.go` 扩展 | SC 7 (silence 过滤 + ShowSilenced) | integration |
| `reconciliation_baseline_test.go` 新建 | SC 8 (降噪对比 + 基线快照 sys_config 读写) | unit |
| `scheduler/reconciliation_tasks_test.go` 新建/扩展 | EXCEPTION-03 (软停用+幂等) + EXCEPTION-02 (转单 SQL no_workorder) | integration |
| `reconciliation_exception_handler_test.go` 扩展 | AUDIT-01 (CRUD operlog 接入) | integration |
| migration_174 集成测试 | GiST 索引 + CHECK 约束存在性 | integration (PG only) |

**纯函数测试模式（matcher_test）**：纯 Go 不需 DB，参考 reconciliation_detection.go 现有 `ClassifyType` 等纯函数的单测模式（若有）。

---

## Shared Patterns

### Authentication / Authorization
**Source:** `pkg/middleware/permission.go:200` (`RequirePermissions`)
**Apply to:** 文件 10 (所有新路由) + 文件 12 (前端 admin 页 RBAC)
```go
// Source: permission.go:200 (CITED)
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc
```
权限命名空间（44-CONTEXT 锁定）：`asset:reconciliation:exception:{list,create,update,delete,test}`。

### Error Handling (Handler-Service 分层)
**Source:** CLAUDE.md "Go Code Patterns / Error Handling" + `reconciliation_exception_handler.go:42-44`
**Apply to:** 文件 9 (所有新 handler)
- Service 层返回 `error`（不返回 HTTP code）
- Handler 层：`c.ShouldBindJSON` 失败 → `response.Error(c, http.StatusBadRequest, ...)`；service 返回 err → `response.Error(c, http.StatusInternalServerError, err.Error())`；success → `response.Success(c, ...)`
- Service err wrapping：`fmt.Errorf("context: %w", err)`

### Response Wrapping
**Source:** `pkg/response`（CLAUDE.md 强制）
**Apply to:** 文件 9 + 10 所有端点
- `response.Success(c, data)` / `response.Error(c, code, msg)` / `response.Page(c, list, total, current, pageSize)`
- **禁止** raw `c.JSON(...)`（CLAUDE.md Common Gotchas）

### operlog.Record 强制约定
**Source:** `internal/api/v1/system/config_handler.go:119,322,350` + CLAUDE.md "操作日志记录约定 — 强制"
**Apply to:** 文件 9 所有写操作 handler（Create/Update/Delete + 可选 Enable/Disable + baseline snapshot）
```go
// success path 末尾，response.Success 之前
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeCreate)
response.Success(c, ...)
```
- module 常量：`ModuleReconciliationExceptionRule = "资产对账-例外规则"`（文件 11，Phase 42 D-16 锁定）
- OperType 用 25 常量集（Create=1 / Update=2 / Delete=3 / Enable=12 / Disable=13 / Status=10）
- 例外规则 CRUD **无敏感字段**（reason 是普通文本），用 `operlog.Record` 非 `RecordWithBody`
- 回归守护：`internal/utils/operlog/regression_test.go` 自动覆盖，新 module 常量不破坏

### Cache Key Helper
**Source:** `internal/services/asset/cache_keys.go:33-37,67-75` (已就位，INFRA-04)
**Apply to:** 文件 4 (CRUD service) — 写操作后失效缓存
```go
// Source: cache_keys.go:33-37 (CITED, R3 启用)
const CacheKeyReconciliationExceptionRuleList = "reconciliation:exceptionRule:list"
const CacheKeyReconciliationExceptionRuleByID  = "reconciliation:exceptionRule:byID:%s"
func GetReconciliationExceptionRuleByIDKey(id string) string { ... }
```
**前缀处理（CLAUDE.md Cache Key Prefix Handling）**：调用 `cache.Set(ctx, key, ...)` 传逻辑键（不带 `xingran:`），UI 输入的 key 需 `StripCachePrefix` 后再操作。

### Status Value Convention
**Source:** CLAUDE.md "Status Value Convention"
**Apply to:** 文件 4 (`is_active` 字段) + 文件 7 (cleanupExpiredExceptions)
- `is_active`: **0=启用, 1=停用**（与全局 0=enabled 一致）
- 过期软停用：`UPDATE is_active = 1`（0→1 即启用→停用）
- 默认值：新建规则 `is_active=0`（启用）

### Database Migration 链式注册
**Source:** `internal/core/db/database.go:455-483` + 项目记忆 `xingran-migrations-no-sql-autoloader`
**Apply to:** 文件 1 + 2
- migrations/*.sql 孤立不自动加载，必须 `migration_NNN_*.go` 显式调用 + 加入 database.go AutoMigrate 块
- 命名 `Migrate174ReconciliationExceptionGist`，在 Migrate173 之后追加
- 失败 `applogger.Errorf` 不 panic（与现有 168-173 一致，避免单迁移失败阻塞启动）

### PG inet_ops GiST 索引（项目首例）
**Source:** 44-RESEARCH.md §Standard Stack（web search 验证 PG 9.4+ 内置，无需 CREATE EXTENSION）
**Apply to:** 文件 1 + 文件 4 MatchTest SQL
- `CREATE INDEX ... USING gist (ip_range inet_ops) WHERE is_active=0 AND deleted_at IS NULL`
- 命中测试查询：`WHERE ip_range >> ?::inet`（`>>` = cidr 包含 inet）
- GORM 占位符 `?::inet` 防 SQL 注入（T-SQLInject）

### Excel Import（3 项目记忆强约束）
**Source:** `excel_config.go:101-152` + 项目记忆 `xingran-excel-import-*`
**Apply to:** 文件 8
- 按列位置匹配（非表头名）：Columns 顺序 = Excel 列序
- UpsertKey 列必须配 DBField
- 路由冲突规避：不在 router.go 预注册，由 Setup*Router 自管

---

## No Analog Found

| File | Role | Data Flow | Reason | Fallback |
|------|------|-----------|--------|----------|
| `reconciliation_exception_matcher.go` (mergeActions/applySkipSeverity 纯函数) | service pure fn | transform | 项目内无"多规则合并取并集 + severity 降级链"先例 | 44-RESEARCH.md §Pattern 2 :299-354 verbatim 代码 |
| `MatchTestPanel.tsx` (命中测试 UI) | frontend component | request-response | 合并卡片 + 命中规则列表形态无先例 | hooks/useExceptionList.ts (useQuery 模式) + dashboard 统计卡片组合 |
| migration_174 GiST inet_ops 索引 | migration | DDL | 项目无 GiST 索引先例（仅 partial unique index 168） | migration_168:139-156 DO$$ 模式 + 44-RESEARCH.md :644-725 verbatim SQL + PG 官方文档 |

---

## Metadata

**Analog search scope:**
- `internal/core/db/migrations/` (migration 模式)
- `internal/core/db/database.go` (注册链)
- `internal/services/asset/` (reconciliation 全套 + statistics + workorder)
- `internal/services/operations/excel_config.go` (Excel 模式)
- `internal/services/system/config_service.go` (sys_config 读写)
- `internal/api/v1/asset/` (reconciliation handler/router)
- `internal/api/v1/system/config_handler.go` + `apikey_handler.go` (operlog CRUD)
- `internal/scheduler/reconciliation_tasks.go` (cron 分发)
- `internal/middleware/apikey.go` (CIDR 匹配) + `permission.go` (RequirePermissions)
- `internal/models/reconciliation.go` (model 字段)
- `xingran-react-frontend/src/pages/asset/reconciliation/` (同模块前端)
- `xingran-react-frontend/src/pages/ad-domain/accounts/` (admin CRUD 页)
- `xingran-react-frontend/src/lib/assetApi.ts` + `queryKeys.ts` (前端 API)
- `internal/utils/operlog/regression_test.go` (operlog 回归)

**Files scanned:** 18 (含 3 个 greenfield/no-analog 项，均已有 research verbatim 代码或最近参考兜底)
**Pattern extraction date:** 2026-06-28
**Files re-read during mapping:** 0（一次性读取 + Grep 定位，无重复范围）
