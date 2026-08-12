# Phase 46 PATTERNS.md — 半自动修复 (R5) 代码模式映射

**Phase:** 46 — 半自动修复（R5, v1.17 资产对账 milestone 收官）
**Generated:** 2026-07-03
**Source:** `.planning/phases/46-r5/46-CONTEXT.md` + `46-RESEARCH.md`

---

## Summary

**File count: 17 total**
- 9 new backend (3 migrations + 4 services + 1 handler + 1 router)
- 5 new frontend (1 page + 1 drawer + 3 utility extensions)
- 3 modified (1 backend handler + 1 router.go + 1 frontend lib)

| # | Layer | File | Role | Closest Analogue |
|---|-------|------|------|------------------|
| 1 | Backend Model | `internal/models/reconciliation_fix_suggestion.go` | GORM model | `internal/models/reconciliation.go:22-60` |
| 2 | Backend Migration | `internal/core/db/migrations/migration_182_create_fix_suggestion_table.go` | Table DDL | `migration_168_reconciliation_tables.go:40-72` |
| 3 | Backend Migration | `internal/core/db/migrations/migration_183_fix_suggestion_unique_index.go` | Partial unique index | `migration_168_reconciliation_tables.go:139-157` |
| 4 | Backend Migration | `internal/core/db/migrations/migration_184_fix_suggestion_config_seeds.go` | INFRA-02 seeds | `migration_169_reconciliation_dicts_configs.go` |
| 5 | Backend Service | `internal/services/asset/fix_suggestion_service.go` | Business logic | `reconciliation_service.go` |
| 6 | Backend Service | `internal/services/asset/fix_suggestion_generator.go` | Cron trigger | `reconciliation_workorder.go` + `reconciliation_detection.go` |
| 7 | Backend Service | `internal/services/asset/fix_suggestion_stats.go` | 7d statistics | `reconciliation_statistics.go` |
| 8 | Backend Handler | `internal/api/v1/asset/fix_suggestion_handler.go` | HTTP handler | `reconciliation_handler.go` |
| 9 | Backend Router | `internal/api/v1/asset/fix_suggestion_router.go` | Route registration | `reconciliation_router.go` |
| 10 | Frontend Page | `src/pages/asset/reconciliation/fix-suggestion/index.tsx` | Main page | `pages/asset/reconciliation/exceptions/index.tsx` |
| 11 | Frontend Drawer | `src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx` | Detail Drawer | `src/components/reconciliation/ReconciliationDrawer.tsx` |
| 12 | Frontend Lib | `src/lib/queryKeys.ts` | Add keys | `queryKeys.ts:44-87` |
| 13 | Frontend Lib | `src/lib/assetApi.ts` | Add fixSuggestionApi | `reconciliationApi` factory |
| 14 | Frontend Lib | `src/lib/opsApi.ts` | Optional Excel export | `excelApi` factory |
| 15 | Modified Backend | `internal/api/v1/asset/reconciliation_handler.go` | Add module const | `reconciliation_handler.go:23-29` |
| 16 | Modified Backend | `internal/api/router.go` | Register new router | `router.go` |
| 17 | Modified Frontend | `src/App.tsx` / `routes` config | Register page route | — |

---

## Backend Patterns

### Pattern 1: GORM model with partial unique index

**Files to create**: `internal/models/reconciliation_fix_suggestion.go`
**Closest analogue**: `internal/models/reconciliation.go:22-60` (SysDataReconciliation)
**Base model**: `internal/models/base.go:11-19` (BaseModel)

**Code excerpt from reconciliation.go:22-60**:
```go
type SysDataReconciliation struct {
    BaseModel

    // 核心标识
    AssetID string `gorm:"type:uuid;not null;column:asset_id;index:idx_recon_asset_id,priority:1" json:"assetId"`
    // ConflictType 取值字典 asset_reconciliation_conflict_type (A/B/C/D/E/F),size:2 与字典值匹配
    ConflictType string `gorm:"size:2;not null;column:conflict_type;index:idx_recon_conflict_type,priority:1" json:"conflictType"`

    // 严重程度
    Severity string `gorm:"size:16;not null;column:severity;index:idx_recon_severity,priority:1" json:"severity"`

    // 三路证据
    PhysicalValue  json.RawMessage `gorm:"type:jsonb;column:physical_value" json:"physicalValue,omitempty"`
    DeclaredValue  json.RawMessage `gorm:"type:jsonb;column:declared_value" json:"declaredValue,omitempty"`
    ConfidenceScore float64        `gorm:"type:decimal(3,2);column:confidence_score" json:"confidenceScore"`

    // 原始快照
    RawSnapshot json.RawMessage `gorm:"type:jsonb;not null;column:raw_snapshot" json:"rawSnapshot"`

    // 资产 IP
    AssetIP *string `gorm:"type:inet;column:asset_ip" json:"assetIp,omitempty"`

    // 例外规则
    ExceptionRuleID *string `gorm:"type:uuid;column:exception_rule_id" json:"exceptionRuleId,omitempty"`

    // 已应用动作
    AppliedActions pq.StringArray `gorm:"type:text[];column:applied_actions" json:"appliedActions,omitempty"`

    // 生命周期
    DetectedAt    time.Time  `gorm:"not null;default:now();column:detected_at;index:idx_recon_detected_at,priority:1" json:"detectedAt"`
    ResolvedAt    *time.Time `gorm:"column:resolved_at" json:"resolvedAt,omitempty"`
    ResolvedBy    *string    `gorm:"column:resolved_by" json:"resolvedBy,omitempty"`

    // 工单
    WorkorderID *string `gorm:"type:uuid;column:workorder_id" json:"workorderId,omitempty"`
}

func (SysDataReconciliation) TableName() string {
    return "sys_data_reconciliation"
}
```

**Key patterns**:
- **Inherit `BaseModel`**: embeds ID/CreatedAt/UpdatedAt/DeletedAt/CreatedBy/UpdatedBy/Version
- **UUID PK**: inherited via BaseModel (gorm:"type:uuid;primary_key")
- **Soft delete**: `BaseModel.DeletedAt gorm.DeletedAt gorm:"index"` (auto from BaseModel)
- **Default UUID**: BaseModel.BeforeCreate auto-generates `uuid.New().String()` (base.go:22-27)
- **GORM tags**:
  - `gorm:"type:uuid;not null;column:asset_id;index:...,priority:1"` for FKs
  - `gorm:"size:N;not null;column:..."` for strings
  - `gorm:"type:decimal(3,2);not null;column:..."` for percentages
  - `gorm:"type:jsonb;column:..."` for JSONB
  - `gorm:"type:text[];column:..."` for PG text arrays
  - `gorm:"type:inet;column:..."` for IP addresses
  - `gorm:"type:cidr;column:..."` for CIDR ranges
- **JSON tags**: camelCase convention `json:"fieldName"`
- **Pointer types for nullable fields**: `*string`, `*time.Time`
- **`TableName()` method**: explicit table name (avoids GORM pluralization)

**R5-specific fields** (24 total, organized by purpose):
```go
// 关联
ExceptionID string `gorm:"type:uuid;not null;column:exception_id;index:idx_fix_suggestion_exception,priority:1" json:"exceptionId"`

// 修复源(D-A1 锁定:仅 user_id)
SuggestedUserID *string `gorm:"size:64;column:suggested_user_id" json:"suggestedUserId,omitempty"`
PreFixUserID    *string `gorm:"size:64;column:pre_fix_user_id" json:"preFixUserId,omitempty"`

// 置信度与原因
ConfidenceScore float64 `gorm:"type:decimal(3,2);not null;column:confidence_score" json:"confidenceScore"`
Reason          string  `gorm:"type:text;not null;column:reason" json:"reason"`

// 状态机(D-B2 6 状态)
FixStatus    string `gorm:"size:16;not null;default:'pending';column:fix_status;index:idx_fix_suggestion_status,priority:1" json:"fixStatus"`
ConflictType string `gorm:"size:2;not null;column:conflict_type" json:"conflictType"`  // 冗余,免 JOIN

// 6 状态时间戳
AcceptedAt   *time.Time `gorm:"column:accepted_at" json:"acceptedAt,omitempty"`
RejectedAt   *time.Time `gorm:"column:rejected_at" json:"rejectedAt,omitempty"`
AppliedAt    *time.Time `gorm:"column:applied_at" json:"appliedAt,omitempty"`
RolledBackAt *time.Time `gorm:"column:rolled_back_at" json:"rolledBackAt,omitempty"`

// 操作人(可空 — 自动生成时为系统)
AcceptedBy   *string `gorm:"size:64;column:accepted_by" json:"acceptedBy,omitempty"`
RejectedBy   *string `gorm:"size:64;column:rejected_by" json:"rejectedBy,omitempty"`
AppliedBy    *string `gorm:"size:64;column:applied_by" json:"appliedBy,omitempty"`
RolledBackBy *string `gorm:"size:64;column:rolled_back_by" json:"rolledBackBy,omitempty"`

// 必填原因(D-C3 审计)
RejectionReason *string `gorm:"type:text;column:rejection_reason" json:"rejectionReason,omitempty"`
RollbackReason  *string `gorm:"type:text;column:rollback_reason" json:"rollbackReason,omitempty"`

// 回滚窗口(D-C2 固定 7d)
RollbackWindowUntil *time.Time `gorm:"column:rollback_window_until" json:"rollbackWindowUntil,omitempty"`

// 多轮版本化(D-B3)
SupersededAt *time.Time `gorm:"column:superseded_at" json:"supersededAt,omitempty"`

// 客户端 IP(审计追溯)
ApplyClientIP    *string `gorm:"size:64;column:apply_client_ip" json:"applyClientIp,omitempty"`
RollbackClientIP *string `gorm:"size:64;column:rollback_client_ip" json:"rollbackClientIp,omitempty"`
```

**Partial unique index** (declared in migration, not in gorm tag — see Pattern 2):
```sql
CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
  ON sys_reconciliation_fix_suggestion (exception_id)
  WHERE fix_status = 'pending' AND superseded_at IS NULL AND deleted_at IS NULL;
```

---

### Pattern 2: Migration with partial unique index (DO $$ block)

**Files to create**: `internal/core/db/migrations/migration_182_create_fix_suggestion_table.go`
`internal/core/db/migrations/migration_183_fix_suggestion_unique_index.go`
**Closest analogue**: `internal/core/db/migrations/migration_168_reconciliation_tables.go:40-157`

**Code excerpt from migration_168:40-72 (AutoMigrate + dialect branch)**:
```go
func Migrate168ReconciliationTables(db *gorm.DB) error {
    log.Println("Running migration 168: Phase 42 R1 reconciliation tables + materialized view + unique index")

    // 1. GORM AutoMigrate 两张主表 + 旁路 sys_reconciliation_exception
    if err := db.AutoMigrate(
        &models.SysDataReconciliation{},
        &models.SysReconciliationException{},
    ); err != nil {
        return fmt.Errorf("AutoMigrate 对账表失败: %w", err)
    }

    // 2-5 仅在 PostgreSQL 执行(SQLite 不支持 INET / CIDR / 物化视图)
    if !isPostgreSQL(db) {
        applogger.Infof("[迁移] reconciliation 物化视图 + 部分唯一索引 跳过(非 PostgreSQL 数据库)")
        return nil
    }
    // ... MV + indexes
}
```

**Code excerpt from migration_168:139-157 (partial unique index pattern)**:
```go
// partial unique index uniq_recon_asset_type_open (D-11 防告警风暴)
// 用 DO $$ 块显式命名,避免 PG 自动命名 `<table>_<col>_key`
// 与 GORM uniqueIndex 期望的 `uni_*` 命名规范冲突
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
    return fmt.Errorf("创建 partial unique index uniq_recon_asset_type_open 失败: %w", err)
}
```

**Key patterns**:
- **AutoMigrate + GORM tag 自动建索引**: Model struct gorm tags auto-create most indexes
- **显式 `CREATE INDEX IF NOT EXISTS`** for additional indexes (status_created, applied_at)
- **`isPostgreSQL(db)` check** before PG-specific SQL (CHECK constraints, partial unique index, MV)
- **DO $$ block + `pg_indexes` check**: idempotent — won't fail on re-run
- **Naming convention**: `uni_*_*` (per `xingran-gorm-sql-constraint-naming-conflict` memory)
- **Partial index WHERE clause**: `WHERE fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL`
- **CHECK constraints** added via `ALTER TABLE ... ADD CONSTRAINT chk_*` (GORM doesn't generate these)
- **CHECK DROP IF EXISTS** before ADD: idempotent re-runs
- **dialect check** in CHECK block: `if !isPostgreSQL(db) { continue }` for SQLite fallback

**R5 migrations to write**:
- `Migrate182CreateFixSuggestionTable`: AutoMigrate + 4 索引 + 2 CHECK 约束
- `Migrate183FixSuggestionUniqueIndex`: 独立 partial unique index 文件（幂等,可在 migration 失败后单独重跑）
- `Migrate184FixSuggestionConfigSeeds`: 4 条 sys_config + 5 条 sys_menu + 1 条 sys_job（INFRA-02）

**Migration registration in database.go** (~line 400+):
```go
// 必须在 AutoMigrate 注册:&models.SysReconciliationFixSuggestion{}
// 必须在 migrations.MigrateNNN(d.DB) 显式调用
// 顺序:在 168/169/170 系列后,如:
if err := migrations.Migrate182CreateFixSuggestionTable(d.DB); err != nil {
    applogger.Errorf("...")
}
if err := migrations.Migrate183FixSuggestionUniqueIndex(d.DB); err != nil {
    applogger.Errorf("...")
}
if err := migrations.Migrate184FixSuggestionConfigSeeds(d.DB); err != nil {
    applogger.Errorf("...")
}
```

---

### Pattern 3: Service interface + private impl + DI constructor (Handler-Service pattern)

**Files to create**: `internal/services/asset/fix_suggestion_service.go`
**Closest analogue**: `internal/services/asset/reconciliation_service.go:131-228` (ReconciliationService interface + impl + constructor)

**Code excerpt from reconciliation_service.go:131-228**:
```go
// ReconciliationService 资产对账异常查询服务接口
type ReconciliationService interface {
    ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error)
    GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error)
    ResolveException(ctx context.Context, id string, userID string, note *string) error
    // ... more
}

// reconciliationServiceImpl ReconciliationService 私有实现
type reconciliationServiceImpl struct {
    db *gorm.DB
    cache cache.Cache
    matcher ReconciliationExceptionService
}

// NewReconciliationService 构造 ReconciliationService 实例
func NewReconciliationService(db *gorm.DB, c cache.Cache, matcher ReconciliationExceptionService) ReconciliationService {
    return &reconciliationServiceImpl{db: db, cache: c, matcher: matcher, mvExists: -1}
}
```

**Key patterns**:
- **Interface in service file** (Go convention: small interface defined in consumer)
- **Private impl** (lowercase first letter)
- **Constructor returns interface type** (not concrete)
- **Dependencies via constructor** (no global state, no setters except for cycle-breaking `SetMatcher`)
- **Optional dependencies** (cache, matcher can be nil → degrades gracefully)
- **GORM DB first** param (followed by cache/services)

**R5 FixSuggestionService interface** (8 methods):
```go
type FixSuggestionService interface {
    // Read endpoints
    ListFixSuggestions(ctx context.Context, params *FixSuggestionListParams) (*base.PageResult, error)
    GetByID(ctx context.Context, id string) (*FixSuggestionDetail, error)
    Stats(ctx context.Context, windowDays int) (*FixSuggestionStatsResponse, error)
    // Write endpoints (D-D3 单条)
    Accept(ctx context.Context, id, userID string) error
    Reject(ctx context.Context, id, userID, reason string) error
    Apply(ctx context.Context, id, userID string) error
    Rollback(ctx context.Context, id, userID, reason string) error
    // Cron trigger (D-A4)
    GenerateFixSuggestions(ctx context.Context) (inserted int, err error)
}

type fixSuggestionServiceImpl struct {
    db            *gorm.DB
    cache         cache.Cache
    configService *system.ConfigService  // D-A3 + D-C5
    noticeHub     *websocket.NoticeHub    // D-C5 告警
}

func NewFixSuggestionService(db *gorm.DB, c cache.Cache, configSvc *system.ConfigService, noticeHub *websocket.NoticeHub) FixSuggestionService {
    return &fixSuggestionServiceImpl{db: db, cache: c, configService: configSvc, noticeHub: noticeHub}
}
```

**R5 ListFixSuggestions pattern** (ListExceptions adaptation):
- Embed `base.BaseListRequest` for current/pageSize/orderByColumn/isAsc
- 5 filter fields: fixStatus, conflictType, responsibleDeptId (JOIN ops_asset), createdFrom/To
- 4-item sort whitelist: `createdAt / confidenceScore / fixStatus / appliedAt`
- JOIN `ops_asset` + `sys_user` for asset_code / asset_user_id
- Use `base.ApplySort` for safe ORDER BY
- `MaxPageSize = 100` to prevent DoS (per `stat-cards-from-list-length-capped-at-100` memory)
- **No `list.length`** for stats — use dedicated COUNT endpoints (D-C5)

---

### Pattern 4: Transaction + conditional UPDATE for concurrency control

**Files to create**: `internal/services/asset/fix_suggestion_service.go` (Apply/Rollback methods)
**Closest analogue**: `internal/services/asset/reconciliation_service.go:587-628` (ResolveException)

**Code excerpt from reconciliation_service.go:587-628 (ResolveException transaction)**:
```go
func (s *reconciliationServiceImpl) ResolveException(ctx context.Context, id string, userID string, note *string) error {
    if id == "" {
        return errors.New("异常ID不能为空")
    }
    if userID == "" {
        return errors.New("当前用户ID不能为空")
    }

    // step 1: 查询异常
    var rec models.SysDataReconciliation
    err := s.db.WithContext(ctx).
        Where("id = ? AND deleted_at IS NULL", id).
        First(&rec).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return errors.New("异常不存在")
        }
        return fmt.Errorf("查询异常失败: %w", err)
    }

    // step 2: 已 resolved 不允许重复 resolve(D-A4-04 锁定)
    if rec.ResolvedAt != nil {
        return errors.New("该异常已标记为已解决")
    }

    // step 3: 构造 updates map(GORM Updates 只 SET 包含字段,避免覆盖其他字段)
    now := time.Now()
    updates := map[string]interface{}{
        "resolved_at": now,
        "resolved_by": userID,
    }
    if note != nil && *note != "" {
        updates["resolution_note"] = *note
    }

    // step 4: UPDATE
    if err := s.db.WithContext(ctx).Model(&rec).Updates(updates).Error; err != nil {
        return fmt.Errorf("更新异常已解决状态失败: %w", err)
    }
    return nil
}
```

**Key patterns**:
- **Validate inputs first** (return error before any DB work)
- **Read-then-check-then-update** pattern (avoid lost update)
- **Defensive re-check** (e.g. `if rec.ResolvedAt != nil` blocks double-resolve)
- **Map-based Updates** (`Updates(map[string]interface{}{...})`) — only SET listed fields
- **GORM ErrRecordNotFound handling** — translate to business error
- **`s.db.WithContext(ctx)`** — propagate context for cancellation/timeout

**R5 Apply transaction pattern** (multi-table, 3-layer defense):
```go
func (s *fixSuggestionServiceImpl) Apply(ctx context.Context, suggestionID, userID string) error {
    if suggestionID == "" || userID == "" {
        return errors.New("参数不能为空")
    }

    tx := s.db.WithContext(ctx).Begin()
    defer tx.Rollback() // 函数退出未 commit 即回滚

    // 1. 读 accepted 建议(必须经 accept 才能 apply)
    var sugg models.SysReconciliationFixSuggestion
    if err := tx.Where("id = ? AND fix_status = ? AND deleted_at IS NULL",
        suggestionID, "accepted").First(&sugg).Error; err != nil {
        return err
    }

    // 2. 反查 exception → asset_id
    var exception models.SysDataReconciliation
    if err := tx.Where("id = ? AND deleted_at IS NULL", sugg.ExceptionID).First(&exception).Error; err != nil {
        return err
    }

    // 3. 读 ops_asset 当前 user_id(用于 pre_fix_user_id)
    var asset models.Asset
    if err := tx.Where("id = ? AND deleted_at IS NULL", exception.AssetID).First(&asset).Error; err != nil {
        return err
    }
    preFixUserID := asset.UserID

    // 4. UPDATE ops_asset.user_id(核心修复写 — D-A1)
    if err := tx.Model(&asset).Update("user_id", sugg.SuggestedUserID).Error; err != nil {
        return err
    }

    // 5. UPDATE 建议状态 + pre_fix_user_id + 7d rollback 窗口
    now := time.Now()
    if err := tx.Model(&sugg).Updates(map[string]interface{}{
        "fix_status":            "applied",
        "applied_at":            now,
        "applied_by":            userID,
        "pre_fix_user_id":       preFixUserID,
        "rollback_window_until": now.Add(7 * 24 * time.Hour),
    }).Error; err != nil {
        return err
    }

    return tx.Commit().Error
}
```

**Layer 1 partial unique index** (D-B4): 阻止 duplicate pending
**Layer 2 conditional UPDATE**: `WHERE fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL`
**Layer 3 RowsAffected check**: `if result.RowsAffected == 0` → return 409 Conflict

---

### Pattern 5: Cache invalidation helper (D-C4 reuse)

**Files to create**: handler-only — no new service file
**Closest analogue**: `internal/services/asset/cache_keys.go:112-133` (InvalidateWorkstationHealth)

**Code excerpt from cache_keys.go:112-133**:
```go
func InvalidateWorkstationHealth(ctx context.Context, c cache.Cache, workstationID string) error {
    if c == nil || workstationID == "" {
        return nil
    }
    return c.Delete(ctx, GetReconciliationHealthByWorkstationKey(workstationID))
}
```

**Key patterns**:
- **Nil-safe** (cache=nil, wsID="" → return nil without error)
- **Wrap cache.Delete** in helper (centralized, testable)
- **Called by handler after service** (strict order: service → invalidate → operlog → response)
- **WSID reverse-lookup pattern** (from reconciliation_handler.go:168-185):
  - Use `gorm.Row().Scan(&wsID)` (returns `sql.ErrNoRows`, not `gorm.ErrRecordNotFound`)
  - Filter `reconciliation_normalized.workstation_id IS NOT NULL`
  - Failure logs warning, doesn't abort (D-A4-04)

**R5 Apply handler cache invalidation** (D-C4 复用):
```go
// 严格顺序: service → invalidate → operlog → response
if err := h.service.Apply(c.Request.Context(), suggestionID, userID); err != nil {
    return
}

// 反查 wsID
var wsID sql.NullString
scanErr := h.core.GetDB().WithContext(c.Request.Context()).
    Table("reconciliation_normalized").
    Select("reconciliation_normalized.workstation_id").
    Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
    Joins("JOIN sys_reconciliation_fix_suggestion ON sys_reconciliation_fix_suggestion.exception_id = sys_data_reconciliation.id").
    Where("sys_reconciliation_fix_suggestion.id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL", suggestionID).
    Limit(1).
    Row().
    Scan(&wsID)
if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
    applogger.Warnf("[fix-suggestion:Apply] R4 query workstation failed: %v", scanErr)
}
if wsID.Valid && wsID.String != "" {
    if invErr := asset.InvalidateWorkstationHealth(c.Request.Context(), h.core.Cache, wsID.String); invErr != nil {
        applogger.Warnf("[fix-suggestion:Apply] invalidate cache failed: %v", invErr)
    }
}
```

**Failure semantics**:
- service error → return (no operlog, no cache invalidate)
- cache delete fail → log warn, **continue** (don't block operlog)
- operlog.Record → must succeed (silent fail is OK in current implementation)

---

### Pattern 6: Cron trigger via sys_job seed

**Files to create**: `internal/services/asset/fix_suggestion_generator.go` (Run method) + 1 row in `migration_184` sys_job seed
**Closest analogue**: `internal/services/asset/reconciliation_workorder.go` (cron-driven) + `migration_169_reconciliation_dicts_configs.go:253-293` (sys_job seed)

**Code excerpt from migration_169:253-293 (sys_job seed pattern)**:
```go
func seedReconciliationSysJobs(db *gorm.DB) error {
    jobs := []struct {
        jobName        string
        invokeTarget   string
        cronExpression string
        remark         string
    }{
        {"对账-物化视图刷新", "reconciliation:refreshView", "@every 5m", "R1: ..."},
        {"对账-Layer3检测", "reconciliation:detectLayer3", "@every 6m", "R1: ..."},
        {"对账-静默期重检测", "reconciliation:detectExpiredSilence", "0 2 * * *", "R2 占位: ..."},
        {"对账-例外规则清理", "reconciliation:cleanupExpiredExceptions", "0 3 * * *", "R3 占位: ..."},
    }

    for _, j := range jobs {
        var existingCount int64
        if err := db.Model(&models.Job{}).Where("job_name = ?", j.jobName).Count(&existingCount).Error; err != nil {
            return err
        }
        if existingCount > 0 {
            continue  // 幂等: 已存在跳过
        }
        remark := j.remark
        job := &models.Job{
            JobName:        j.jobName,
            JobGroup:       "reconciliation",
            InvokeTarget:   j.invokeTarget,
            CronExpression: j.cronExpression,
            MisfirePolicy:  1, // 1 = 立即执行
            Concurrent:     false,
            Status:         models.JobStatusNormal,
            Remark:         &remark,
        }
        if err := db.Create(job).Error; err != nil {
            log.Printf("Migration 169: create sys_job %s failed: %v", j.jobName, err)
            continue
        }
        applogger.Infof("[迁移] sys_job seed: %s (cron=%s, target=%s)", j.jobName, j.cronExpression, j.invokeTarget)
    }
    return nil
}
```

**R5 sys_job seed** (1 row in migration_184):
```go
{"对账-修复建议生成", "reconciliation:generateFixSuggestions", "@every 5m", "R5: 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion (D-A4 触发器)"},
```

**R5 Generator.Run pattern** (cron context, idempotent):
```go
func (s *fixSuggestionServiceImpl) GenerateFixSuggestions(ctx context.Context) (int, error) {
    if s.db == nil {
        return 0, errors.New("db 未初始化")
    }

    // 1. 读取 sys_config: confidence_threshold + enabled
    threshold := s.getConfidenceThreshold(ctx)
    if !s.isFeatureEnabled(ctx) {
        applogger.Infof("[fix-suggestion:Generator] 功能已禁用(enabled=0),跳过本轮")
        return 0, nil
    }

    // 2. 查找候选 Type B 异常(D-A4 触发条件)
    //    confidence >= threshold AND conflict_type='B' AND workorder_id IS NULL
    //    AND deleted_at IS NULL AND resolved_at IS NULL
    //    AND NOT EXISTS (pending suggestion)
    type candidate struct {
        ID               string
        AssetID          string
        ConfidenceScore  float64
    }
    var candidates []candidate
    sql := `
        SELECT r.id, r.asset_id, r.confidence_score
        FROM sys_data_reconciliation r
        WHERE r.conflict_type = 'B'
          AND r.confidence_score >= ?
          AND r.workorder_id IS NULL
          AND r.resolved_at IS NULL
          AND r.deleted_at IS NULL
          AND NOT EXISTS (
            SELECT 1 FROM sys_reconciliation_fix_suggestion s
            WHERE s.exception_id = r.id
              AND s.fix_status = 'pending'
              AND s.superseded_at IS NULL
              AND s.deleted_at IS NULL
          )
    `
    if err := s.db.WithContext(ctx).Raw(sql, threshold).Scan(&candidates).Error; err != nil {
        return 0, fmt.Errorf("查询候选异常失败: %w", err)
    }

    // 3. 逐条 INSERT(物理链路 user_id 通过 reconciliation_normalized JOIN 取)
    inserted := 0
    for _, c := range candidates {
        // 物理链路 user_id 从 MV 取(D-A1 锁定)
        var physUserID *string
        s.db.WithContext(ctx).
            Table("reconciliation_normalized").
            Select("physical_user_id").
            Where("asset_id = ?", c.AssetID).
            Row().Scan(&physUserID)

        if physUserID == nil || *physUserID == "" {
            continue  // D-A1: 仅当 physical_user_id 非空时生成建议
        }

        sugg := &models.SysReconciliationFixSuggestion{
            ExceptionID:     c.ID,
            SuggestedUserID: physUserID,
            ConfidenceScore: c.ConfidenceScore,
            Reason:          fmt.Sprintf("物理链路 user_id=%s, ops_asset 当前无责", *physUserID),
            FixStatus:       "pending",
            ConflictType:    "B",
        }
        if err := s.db.WithContext(ctx).Create(sugg).Error; err != nil {
            applogger.Warnf("[fix-suggestion:Generator] 生成建议失败 exception_id=%s: %v", c.ID, err)
            continue
        }
        inserted++
    }
    applogger.Infof("[fix-suggestion:Generator] 本轮生成 %d 条建议(threshold=%.2f, candidates=%d)", inserted, threshold, len(candidates))
    return inserted, nil
}
```

**Cron registration in scheduler**: `internal/scheduler/reconciliation_fix_suggestion_generator.go` calls `Run(ctx)` on InvokeTarget `reconciliation:generateFixSuggestions`

---

### Pattern 7: Statistics endpoint with COUNT aggregation (no list.length)

**Files to create**: `internal/services/asset/fix_suggestion_stats.go` (Stats method)
**Closest analogue**: `internal/services/asset/reconciliation_statistics.go:172-237` (Summary)

**Code excerpt from reconciliation_statistics.go:172-237 (Summary pattern)**:
```go
type SummaryResult struct {
    TotalAssets      int64  `json:"totalAssets"`
    OpenExceptions   int64  `json:"openExceptions"`
    CriticalOpen     int64  `json:"criticalOpen"`
    Last7dNew        int64  `json:"last7dNew"`
    TopConflictType  string `json:"topConflictType"`
    TopConflictCount int64  `json:"topConflictCount"`
}

func (s *reconciliationStatisticsImpl) Summary(ctx context.Context, filter StatsFilter) (*SummaryResult, error) {
    if s.db == nil {
        return nil, errors.New("db 不能为空")
    }
    days := clampStatsDays(filter.Days)
    sevenDaysAgo := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

    result := &SummaryResult{}

    // 1) TotalAssets
    if err := s.db.WithContext(ctx).
        Model(&models.Asset{}).
        Where("deleted_at IS NULL").
        Count(&result.TotalAssets).Error; err != nil {
        return nil, err
    }
    // ... 4 more COUNT aggregations
    return result, nil
}
```

**R5 FixSuggestionStats pattern** (D-C5):
```go
type FixSuggestionStatsResponse struct {
    WindowDays        int       `json:"windowDays"`
    Pending           int       `json:"pending"`
    Accepted          int       `json:"accepted"`
    Rejected          int       `json:"rejected"`
    Applied           int       `json:"applied"`
    RolledBack        int       `json:"rolledBack"`
    Failed            int       `json:"failed"`
    MisFixRate        float64   `json:"misFixRate"`
    Threshold         float64   `json:"threshold"`
    ThresholdBreached bool      `json:"thresholdBreached"`
    TrendSeries       []TrendPoint `json:"trendSeries"`
}

func (s *fixSuggestionServiceImpl) Stats(ctx context.Context, windowDays int) (*FixSuggestionStatsResponse, error) {
    if windowDays <= 0 {
        windowDays = 7
    }
    if windowDays > MaxPageSize {  // 365
        windowDays = MaxPageSize
    }
    since := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour)

    result := &FixSuggestionStatsResponse{WindowDays: windowDays}

    // 6 COUNT 全部走 Model().Where().Count()(严禁 list.length)
    base := s.db.WithContext(ctx).Model(&models.SysReconciliationFixSuggestion{}).
        Where("deleted_at IS NULL AND created_at >= ?", since)

    // 7d 窗口内 pending/accepted/rejected/failed 计数
    base.Where("fix_status = ?", "pending").Count(&result.Pending)  // 注: 需逐条独立 .Count
    // ... etc

    // Applied/RolledBack 用各自时间戳过滤
    s.db.WithContext(ctx).Model(&models.SysReconciliationFixSuggestion{}).
        Where("deleted_at IS NULL AND fix_status = ? AND applied_at >= ?", "applied", since).
        Count(&result.Applied)
    s.db.WithContext(ctx).Model(&models.SysReconciliationFixSuggestion{}).
        Where("deleted_at IS NULL AND fix_status = ? AND rolled_back_at >= ?", "rolled_back", since).
        Count(&result.RolledBack)

    // MisFixRate = rolledBack / applied (applied=0 → 0,非 NaN/Inf)
    if result.Applied > 0 {
        result.MisFixRate = float64(result.RolledBack) / float64(result.Applied)
    }

    // Threshold + ThresholdBreached
    thresholdStr, _ := s.configService.GetByKey(ctx, "asset.reconciliation.fix.mis_fix_threshold")
    result.Threshold = parseFloat(thresholdStr, 0.01)
    result.ThresholdBreached = result.MisFixRate > result.Threshold

    // TrendSeries: 复用 HealthTrend 形态(date_trunc + FILTER)
    result.TrendSeries, _ = s.statsTrend(ctx, windowDays)

    return result, nil
}
```

**Dialect-aware SQL** (PG `FILTER` vs SQLite `CASE WHEN`):
```go
switch dialect {
case "postgres":
    sql = `SELECT date_trunc('day', created_at) AS date, ... FILTER (WHERE fix_status = 'applied') ...`
default:
    sql = `SELECT strftime('%Y-%m-%d', created_at) AS date, SUM(CASE WHEN fix_status = 'applied' THEN 1 ELSE 0 END) ...`
}
```

---

### Pattern 8: operlog integration (4 OperTypes)

**Files to create**: `internal/api/v1/asset/fix_suggestion_handler.go`
**Closest analogue**: `internal/api/v1/asset/reconciliation_handler.go:114-196` (ResolveException)

**Code excerpt from reconciliation_handler.go:187-188 (operlog.Record pattern)**:
```go
// operlog 写入(CLAUDE.md 强制约定,状态变更 → OperTypeUpdate)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)
```

**Module constant** (D-C3 锁定 in `reconciliation_handler.go:18-29`):
```go
const ModuleReconciliation = "资产对账"
const ModuleReconciliationExceptionRule = "资产对账-例外规则"

// R5 新增:
const ModuleReconciliationFixSuggestion = "资产对账-修复建议"
```

**OperType selection** (D-C3 锁定):
| Endpoint | OperType | Constant | Value |
|----------|----------|----------|-------|
| Accept | OperTypeUpdate | `operlog.OperTypeUpdate` | 2 |
| Reject | OperTypeReject | `operlog.OperTypeReject` | 23 |
| Apply | OperTypeUpdate | `operlog.OperTypeUpdate` | 2 |
| Rollback | **OperTypeReset** | `operlog.OperTypeReset` | 11 |
| Stats | (read, no operlog) | — | — |
| List/Get | (read, no operlog) | — | — |

**Why Reset for Rollback** (D-C3): "密码/密钥重置" 语义 — 最接近"恢复到原值"

**operlog call sites** (4 total):
```go
// Accept handler
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

// Reject handler
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    ModuleReconciliationFixSuggestion, operlog.OperTypeReject)

// Apply handler
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

// Rollback handler (D-C3 强写)
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    ModuleReconciliationFixSuggestion, operlog.OperTypeReset)
```

**OperParam example for Rollback** (full audit chain):
- Module: 资产对账-修复建议
- OperType: 11 (OperTypeReset)
- Title: "回滚修复建议 #{suggestion_id}: asset {asset_code} user_id {pre} -> {post}"
- OperParam: `{"rollbackReason": "...", "preFixUserId": "...", "suggestedUserId": "..."}`
- Method: POST /asset/reconciliation/fix-suggestion/{id}/rollback

**Note**: `rejectionReason` / `rollbackReason` are NOT in the 11 sensitive keywords (per operlog.go:134-169), so `operlog.Record` is sufficient (no need for `RecordWithBody`).

---

### Pattern 9: Handler struct with Core DI + WithCore pattern

**Files to create**: `internal/api/v1/asset/fix_suggestion_handler.go` + `fix_suggestion_router.go`
**Closest analogue**: `internal/api/v1/asset/reconciliation_handler.go:39-53` (handler) + `reconciliation_router.go:30-44` (router)

**Code excerpt from reconciliation_handler.go:39-53**:
```go
type ReconciliationHandler struct {
    service asset.ReconciliationService
    core    *core.Core
}

func NewReconciliationHandler(svc asset.ReconciliationService) *ReconciliationHandler {
    return &ReconciliationHandler{service: svc}
}

func (h *ReconciliationHandler) WithCore(core *core.Core) *ReconciliationHandler {
    if h != nil {
        h.core = core
    }
    return h
}
```

**Code excerpt from reconciliation_router.go:30-44**:
```go
func SetupReconciliationRouter(r *gin.RouterGroup, core *core.Core) {
    // R4:Cache 注入支持 GetByWorkstation 的 5min TTL 缓存
    exceptionSvc := asset.NewReconciliationExceptionService(core.DB.GetDB())
    svc := asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvc)
    handler := NewReconciliationHandler(svc).WithCore(core)

    r.POST("/exception/list", handler.ListExceptions)
    r.POST("/exception/:id", handler.GetExceptionByID)
    r.POST("/exception/:id/resolve", handler.ResolveException)
    r.POST("/by-workstation", handler.GetByWorkstation)
    r.POST("/refresh", handler.Refresh)
}
```

**R5 fix_suggestion_router.go** (5+ endpoints, all POST per XingRan convention):
```go
func SetupFixSuggestionRouter(r *gin.RouterGroup, core *core.Core) {
    // 依赖注入(D-A3 sys_config + D-C5 noticeHub)
    configSvc := system.NewConfigService(core.DB.GetDB())
    svc := asset.NewFixSuggestionService(core.DB.GetDB(), core.Cache, configSvc, core.NoticeHub)
    handler := NewFixSuggestionHandler(svc).WithCore(core)

    // R5 端点(D-D1/D-D2 锁定 5+1)
    r.POST("/fix-suggestion/list", handler.ListFixSuggestions)         // asset:reconciliation:fix:list
    r.POST("/fix-suggestion/:id", handler.GetByID)                     // asset:reconciliation:fix:list
    r.POST("/fix-suggestion/:id/accept", handler.Accept)                // asset:reconciliation:fix:accept
    r.POST("/fix-suggestion/:id/reject", handler.Reject)                // asset:reconciliation:fix:reject
    r.POST("/fix-suggestion/:id/apply", handler.Apply)                  // asset:reconciliation:fix:accept
    r.POST("/fix-suggestion/:id/rollback", handler.Rollback)            // asset:reconciliation:fix:rollback
    r.POST("/fix-suggestion/stats", handler.Stats)                      // asset:reconciliation:fix:stats
}
```

**Router registration in main router** (`internal/api/router.go`):
```go
// 在 reconciliation_router 之后:
asset.SetupFixSuggestionRouter(apiV1.Group("/asset/reconciliation"), core)
```

**Handler skeleton** (mirror reconciliation_handler.go style):
```go
package asset

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/core"
    "github.com/xingran-next/xingran-go-backend/internal/services/asset"
    "github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

const ModuleReconciliationFixSuggestion = "资产对账-修复建议"

type FixSuggestionHandler struct {
    service asset.FixSuggestionService
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

func (h *FixSuggestionHandler) ListFixSuggestions(c *gin.Context) {
    var params asset.FixSuggestionListParams
    if err := c.ShouldBindJSON(&params); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }
    result, err := h.service.ListFixSuggestions(c.Request.Context(), &params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, result)
}

// Accept, Reject, Apply, Rollback mirror ResolveException pattern
// (validation + service call + cache invalidate + operlog.Record + response.Success)
```

---

## Frontend Patterns

### Pattern 10: queryKeys namespace extension (factory function)

**Files to modify**: `xingran-react-frontend/src/lib/queryKeys.ts`
**Closest analogue**: `src/lib/queryKeys.ts:45-87` (reconciliation namespace)

**Code excerpt from queryKeys.ts:45-87**:
```typescript
reconciliation: {
    all: ["reconciliation"] as const,
    summary: (windowDays: number) => ["reconciliation", "summary", windowDays] as const,
    exceptionList: (params: ExceptionListParams) =>
        ["reconciliation", "exception-list", params] as const,
    exceptionDetail: (id: string) =>
        ["reconciliation", "exception-detail", id] as const,
    // ...
    workstationHealth: (workstationId: string) =>
        ["reconciliation", "workstation-health", workstationId] as const,
},
```

**Key patterns**:
- **Tuple factory functions** with `as const` for narrow literal types
- **Discriminator as function param** (e.g. `id`, `params`, `windowDays`)
- **Namespaced under feature root** (`reconciliation.*`)
- **Parallel to backend endpoints** (1 key = 1 endpoint)

**R5 extension** (add 3 keys to `reconciliation.*`):
```typescript
reconciliation: {
    // ... existing R1-R4 keys
    /** Phase 46 R5 — 修复建议列表(分页 + 筛选) */
    fixSuggestionList: (params: FixSuggestionListParams) =>
        ["reconciliation", "fix-suggestion-list", params] as const,
    /** Phase 46 R5 — 修复建议详情 */
    fixSuggestionDetail: (id: string) =>
        ["reconciliation", "fix-suggestion-detail", id] as const,
    /** Phase 46 R5 — 修复建议统计(7d KPI) */
    fixSuggestionStats: (windowDays: number) =>
        ["reconciliation", "fix-suggestion-stats", windowDays] as const,
},
```

**Note**: The `ExceptionListParams` import in queryKeys.ts:13 must be updated to import `FixSuggestionListParams` from `assetApi.ts` if they share a file.

---

### Pattern 11: API factory function (object literal)

**Files to modify**: `xingran-react-frontend/src/lib/assetApi.ts`
**Closest analogue**: `src/lib/assetApi.ts:166-200` (reconciliationApi object literal)

**Code excerpt from assetApi.ts:166-200 (reconciliationApi factory)**:
```typescript
export const reconciliationApi = {
    summary: async (days: number = 7): Promise<SummaryResult> => {
        const res = await post<SummaryResult>(
            "/asset/reconciliation/statistics/summary",
            { days }
        );
        return res.data as SummaryResult;
    },
    byConflictType: async (days: number = 7): Promise<Record<string, number>> => {
        const res = await post<Record<string, number>>(
            "/asset/reconciliation/statistics/by-conflict-type",
            { days }
        );
        return (res.data ?? {}) as Record<string, number>;
    },
    // ...
};
```

**Key patterns**:
- **Object literal** (not class — simpler, no `this` binding issues)
- **All methods return `post<T>()` result** with `res.data` unwrap
- **`post()` wrapper** from `@/lib/api` (NOT raw axios — see CLAUDE.md)
- **Type-safe** — explicit return type annotations
- **Default param values** inline (e.g. `days: number = 7`)

**R5 fixSuggestionApi factory** (8 methods):
```typescript
export const fixSuggestionApi = {
    list: (params: FixSuggestionListParams) =>
        post<PageResult<FixSuggestionListItem>>("/asset/reconciliation/fix-suggestion/list", params),

    getById: (id: string) =>
        post<FixSuggestionDetail>(`/asset/reconciliation/fix-suggestion/${id}`, {}),

    accept: (id: string) =>
        post(`/asset/reconciliation/fix-suggestion/${id}/accept`, {}),

    reject: (id: string, rejectionReason: string) =>
        post(`/asset/reconciliation/fix-suggestion/${id}/reject`, { rejectionReason }),

    apply: (id: string) =>
        post(`/asset/reconciliation/fix-suggestion/${id}/apply`, {}),

    rollback: (id: string, rollbackReason: string) =>
        post(`/asset/reconciliation/fix-suggestion/${id}/rollback`, { rollbackReason }),

    stats: (windowDays: number = 7) =>
        post<FixSuggestionStatsResponse>("/asset/reconciliation/fix-suggestion/stats", { windowDays }),
};
```

**Type definitions** (must add to assetApi.ts):
```typescript
// ==================== Phase 46 R5 — Types ====================

export interface FixSuggestionListParams {
    fixStatus?: 'pending' | 'accepted' | 'rejected' | 'applied' | 'rolled_back' | 'failed';
    conflictType?: 'A' | 'B' | 'C' | 'D' | 'E' | 'F';
    responsibleDeptId?: string;
    createdFrom?: string;
    createdTo?: string;
    current: number;
    pageSize: number;
    orderByColumn?: string;
    isAsc?: boolean;
}

export interface FixSuggestionListItem {
    id: string;
    exceptionId: string;
    assetId: string;
    assetCode: string;
    conflictType: 'B';
    currentUserId: string | null;
    suggestedUserId: string;
    suggestedUsername: string | null;
    preFixUserId: string | null;
    confidenceScore: number;
    reason: string;
    fixStatus: FixStatus;
    createdAt: string;
    appliedAt: string | null;
    rolledBackAt: string | null;
    rollbackWindowUntil: string | null;
}

export interface FixSuggestionDetail {
    id: string;
    exceptionId: string;
    exception: {
        id: string;
        assetId: string;
        assetCode: string;
        conflictType: 'B';
        severity: string;
        confidenceScore: number;
        rawSnapshot: object;
        detectedAt: string;
    };
    suggestion: { /* ... 18 fields ... */ };
    history: FixSuggestionListItem[];
}

export interface FixSuggestionStatsResponse {
    windowDays: number;
    pending: number;
    accepted: number;
    rejected: number;
    applied: number;
    rolledBack: number;
    failed: number;
    misFixRate: number;
    threshold: number;
    thresholdBreached: boolean;
    trendSeries: TrendPoint[];
}
```

---

### Pattern 12: Page with Table + Drawer + KPI cards

**Files to create**: `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx`
**Closest analogue**: `src/pages/asset/reconciliation/exceptions/index.tsx` (R1 pattern)

**Code excerpt from exceptions/index.tsx:84-200 (page structure)**:
```typescript
const Exceptions = () => {
    const { message } = App.useApp();
    const queryClient = useQueryClient();
    const [form] = Form.useForm();
    const [searchParams, setSearchParams] = useSearchParams();
    const permissions = useMenuStore((s) => s.permissions);
    const canResolve = permissions.includes("asset:reconciliation:resolve");

    const [current, setCurrent] = useState(1);
    const [pageSize, setPageSize] = useState(20);

    // 服务端排序
    const { orderByColumn, isAsc, sortOrder, handleTableChange, resetSort } =
        useServerSort<ExceptionListItem>({
            sorterMetas: [
                createSorterMeta<ExceptionListItem>("detectedAt", "date"),
                createSorterMeta<ExceptionListItem>("conflictType"),
                createSorterMeta<ExceptionListItem>("severity"),
            ],
            defaultSort: { orderByColumn: "detectedAt", isAsc: false },
        });

    // 字典
    const conflictTypeDict = useDict("asset_reconciliation_conflict_type");

    // listParams 用 useMemo 稳定 deps(CLAUDE.md 强制)
    const listParams = useMemo<ExceptionListParams>(() => {
        const p: ExceptionListParams = { current, pageSize };
        if (filterValues.conflictType) p.conflictType = filterValues.conflictType;
        if (orderByColumn) {
            p.orderByColumn = orderByColumn;
            p.isAsc = isAsc;
        }
        return p;
    }, [current, pageSize, filterValues, orderByColumn, isAsc]);

    const { data, isLoading, isError } = useExceptionList(listParams);

    // ...
};
```

**Key patterns**:
- **`useMemo` for listParams** (stable deps, prevent infinite loop — see CLAUDE.md useEffect Dependencies)
- **D-05 URL query sync** via `useSearchParams` (filter persistence + shareable URLs)
- **`useServerSort` hook** with `createSorterMeta` for whitelist columns
- **`useDict` hook** for status/conflict_type labels
- **`useMenuStore.permissions` for perm check** (e.g. `canAccept = permissions.includes("asset:reconciliation:fix:accept")`)
- **`queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all })`** after mutations
- **Form + Modal for action inputs** (reject reason, rollback reason)

**R5 FixSuggestion page structure**:
```typescript
// 5 KPI 卡片(stats 端点,D-D1)
<Row gutter={16}>
    <Col span={4}><Card><Statistic title="待处理" value={stats?.pending} /></Card></Col>
    <Col span={4}><Card><Statistic title="7d 应用" value={stats?.applied} /></Card></Col>
    <Col span={4}><Card><Statistic title="7d 回滚" value={stats?.rolledBack} valueStyle={{color: 'red'}} /></Card></Col>
    <Col span={4}><Card><Statistic title="误修复率" value={stats?.misFixRate} suffix="%" valueStyle={{color: stats?.thresholdBreached ? 'red' : 'green'}} /></Card></Col>
    <Col span={4}><Card><Statistic title="7d 拒绝" value={stats?.rejected} /></Card></Col>
</Row>

// 筛选表单
<Card><Form>...</Form></Card>

// 8 列 Table(D-D2 紧凑行)
<Table
    rowKey="id"
    columns={columns}
    dataSource={data?.list ?? []}
    pagination={{...}}
    onRow={(record) => ({ onClick: () => openDrawer(record) })}
/>

// 详情 Drawer(3 Tab)
<FixSuggestionDetailDrawer
    open={drawerOpen}
    suggestionId={selectedId}
    onClose={() => setDrawerOpen(false)}
/>

// Reject Modal
<Modal>
    <Form><Form.Item name="rejectionReason" rules={[{min: 10, message: '至少10字符'}]}>
        <Input.TextArea />
    </Form.Item></Form>
</Modal>

// Rollback Modal (similar, with 7d countdown hint)
```

**useEffect dependency stability** (CLAUDE.md 强制):
```typescript
// ✅ 正确
const params = useMemo<FixSuggestionListParams>(
    () => ({ current, pageSize, fixStatus, conflictType }),
    [current, pageSize, fixStatus, conflictType]
);
const { data } = useFixSuggestionList(params);

// ❌ 错误(对象重建导致无限循环)
const { data } = useFixSuggestionList({ current, pageSize, fixStatus, conflictType });
```

---

### Pattern 13: Drawer with multi-Tab + raw_snapshot rendering

**Files to create**: `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`
**Closest analogue**: `src/components/reconciliation/ReconciliationDrawer.tsx` (R4 3-Tab pattern)

**Key patterns** (from D-D2):
- **3 Tab** structure: 冲突摘要 / 修复详情 / 历史变更
- **Tab 1 冲突摘要**: raw_snapshot 三路数据(physical / declared / ad) + ConflictSignals 标志位 + conflict_type / severity / confidenceScore
- **Tab 2 修复详情**: 时间轴显示 + 当前 ops_asset.user_id vs 建议 user_id + pre_fix_user_id(applied 后) + rollback_window_until 倒计时
- **Tab 3 历史变更**: 同 exception_id 的所有 fix_suggestion 记录 + 状态徽标

**Drawer 基础结构** (antd):
```typescript
<Drawer
    title="修复建议详情"
    open={open}
    onClose={onClose}
    width={720}
    destroyOnClose
>
    {detail && (
        <Tabs defaultActiveKey="summary">
            <TabPane tab="冲突摘要" key="summary">
                {/* raw_snapshot 三路 + signals */}
            </TabPane>
            <TabPane tab="修复详情" key="detail">
                {/* 时间轴 + user_id 对比 */}
            </TabPane>
            <TabPane tab="历史变更" key="history">
                {/* 同 exception_id 的所有记录 */}
            </TabPane>
        </Tabs>
    )}
</Drawer>
```

**Detail loading pattern**:
```typescript
const { data: detail, isLoading } = useQuery({
    queryKey: queryKeys.reconciliation.fixSuggestionDetail(suggestionId),
    queryFn: () => fixSuggestionApi.getById(suggestionId!).then(r => r.data),
    enabled: !!suggestionId && open,
});
```

---

### Pattern 14: Permission check + UI disable

**Files to create**: `fix-suggestion/index.tsx` (page), `AcceptApplyModal.tsx`, `RejectModal.tsx`, `RollbackModal.tsx`
**Closest analogue**: `exceptions/index.tsx:286-318` (resolve button perms)

**Code excerpt from exceptions/index.tsx:286-318 (perm-gated action column)**:
```typescript
{
    title: "解决",
    key: "resolve_btn",
    width: 110,
    fixed: "right",
    render: (_: unknown, record: ExceptionListItem) => {
        if (!canResolve) return null;  // 权限控制
        const resolved = record.resolvedAt !== null && record.resolvedAt !== undefined && record.resolvedAt !== "";
        return (
            <Button
                type="link"
                size="small"
                icon={<CheckOutlined />}
                disabled={resolved}  // 状态控制
                onClick={...}
            >
                {resolved ? "已解决" : "标记已解决"}
            </Button>
        );
    },
},
```

**R5 action column** (D-D3 单条 + 状态动态):
```typescript
const permissions = useMenuStore((s) => s.permissions);
const canAccept = permissions.includes("asset:reconciliation:fix:accept");
const canReject = permissions.includes("asset:reconciliation:fix:reject");
const canRollback = permissions.includes("asset:reconciliation:fix:rollback");

// 操作列动态
render: (_: unknown, record: FixSuggestionListItem) => {
    const isWithin7d = record.rollbackWindowUntil && 
        new Date(record.rollbackWindowUntil).getTime() > Date.now();
    
    if (record.fixStatus === "pending" && (canAccept || canReject)) {
        return (
            <Space>
                {canAccept && <Button onClick={() => handleAccept(record)}>接受</Button>}
                {canReject && <Button onClick={() => handleReject(record)}>拒绝</Button>}
            </Space>
        );
    }
    if (record.fixStatus === "accepted" && canAccept) {
        return <Button onClick={() => handleApply(record)}>应用</Button>;
    }
    if (record.fixStatus === "applied" && isWithin7d && canRollback) {
        return <Button onClick={() => handleRollback(record)}>回滚</Button>;
    }
    return <Tag>{record.fixStatus}</Tag>;
}
```

**5 permission codes** (D-C5 / 命名空间):
- `asset:reconciliation:fix:list` — list + getById
- `asset:reconciliation:fix:accept` — accept + apply (两步式合在同一 perm)
- `asset:reconciliation:fix:reject` — reject
- `asset:reconciliation:fix:rollback` — rollback
- `asset:reconciliation:fix:stats` — stats

---

### Pattern 15: Mutation with optimistic update + invalidation

**Files to create**: `fix-suggestion/index.tsx` (mutation handlers)
**Closest analogue**: `exceptions/index.tsx:362-384` (handleResolveSubmit)

**Code excerpt from exceptions/index.tsx:362-384 (mutation + invalidate)**:
```typescript
const handleResolveSubmit = useCallback(async () => {
    if (!resolveModal.exceptionId) return;
    try {
        const values = await resolveForm.validateFields();
        setResolveModal((prev) => ({ ...prev, submitting: true }));
        await reconciliationApi.exceptionResolve(resolveModal.exceptionId, {
            resolutionNote: values.resolutionNote?.trim() || undefined,
        });
        message.success("已标记为已解决");
        // 触发 dashboard + 异常列表 query 重新拉取
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });
        setResolveModal({ open: false, exceptionId: null, note: "", submitting: false });
        resolveForm.resetFields();
    } catch (err) {
        const errMsg = (err as Error)?.message ?? "标记失败";
        if (errMsg.includes("已解决")) {
            queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });
        }
        message.error(errMsg);
        setResolveModal((prev) => ({ ...prev, submitting: false }));
    }
}, [resolveModal.exceptionId, resolveForm, message, queryClient]);
```

**R5 handleAccept pattern** (with state machine guard):
```typescript
const handleAccept = useCallback(async (record: FixSuggestionListItem) => {
    try {
        await fixSuggestionApi.accept(record.id);
        message.success("已接受建议");
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
    } catch (err) {
        const errMsg = (err as Error)?.message ?? "接受失败";
        if (errMsg.includes("已被处理") || errMsg.includes("409")) {
            // 状态机 guard:已被其他运维处理
            queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.all });
        }
        message.error(errMsg);
    }
}, [queryClient, listParams]);
```

**R5 handleRollback pattern** (with 7d window check):
```typescript
const handleRollbackSubmit = useCallback(async () => {
    if (!rollbackModal.suggestionId) return;
    try {
        const values = await rollbackForm.validateFields();
        if (values.rollbackReason.trim().length < 10) {
            message.error("回滚原因至少 10 字符");
            return;
        }
        setRollbackModal((prev) => ({ ...prev, submitting: true }));
        await fixSuggestionApi.rollback(rollbackModal.suggestionId, values.rollbackReason.trim());
        message.success("已回滚修复");
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionStats(7) });
        // 关闭 Drawer(如当前在 Drawer 中)
        setRollbackModal({ open: false, suggestionId: null, rollbackReason: "", submitting: false });
        rollbackForm.resetFields();
    } catch (err) {
        const errMsg = (err as Error)?.message ?? "回滚失败";
        if (errMsg.includes("回滚窗口已过")) {
            // 7d 已过,刷新 UI 隐藏回滚按钮
            queryClient.invalidateQueries({ queryKey: queryKeys.reconciliation.fixSuggestionList(listParams) });
        }
        message.error(errMsg);
        setRollbackModal((prev) => ({ ...prev, submitting: false }));
    }
}, [rollbackModal.suggestionId, rollbackForm, message, queryClient, listParams]);
```

---

## Migration File Mapping

| Migration # | File | Purpose | Status |
|-------------|------|---------|--------|
| 168 | `migration_168_reconciliation_tables.go` | sys_data_reconciliation + MV + uniq_recon_asset_type_open | **DONE** (R1) |
| 169 | `migration_169_reconciliation_dicts_configs.go` | dicts + 8 configs + 6 workorder_categories + 4 sys_jobs + 6 menu buttons | **DONE** (R1) |
| 170 | `migration_170_fix_asset_list_menu_path.go` | 资产列表 menu path 修复 | **DONE** (R1.5) |
| 182 | `migration_182_create_fix_suggestion_table.go` | AutoMigrate SysReconciliationFixSuggestion + 4 indexes + 2 CHECK | **TODO** (R5) |
| 183 | `migration_183_fix_suggestion_unique_index.go` | partial unique index uniq_fix_suggestion_pending_per_exception | **TODO** (R5) |
| 184 | `migration_184_fix_suggestion_config_seeds.go` | 4 sys_config + 5 sys_menu (1+5) + 1 sys_job | **TODO** (R5) |

**Migration order** (in database.go):
```go
// 现有
if err := migrations.Migrate168ReconciliationTables(d.DB); err != nil { ... }
if err := migrations.Migrate169ReconciliationDictsConfigs(d.DB); err != nil { ... }
if err := migrations.Migrate170FixAssetListMenuPath(d.DB); err != nil { ... }

// R5 新增(在 170 之后)
if err := migrations.Migrate182CreateFixSuggestionTable(d.DB); err != nil { ... }
if err := migrations.Migrate183FixSuggestionUniqueIndex(d.DB); err != nil { ... }
if err := migrations.Migrate184FixSuggestionConfigSeeds(d.DB); err != nil { ... }
```

**AutoMigrate registration** in `database.go:288-369`:
```go
err := d.DB.Migrator().AutoMigrate(
    // ... existing models (270+ lines)
    &models.SysDataReconciliation{},  // 现有
    &models.SysReconciliationException{},  // 现有
    // === R5 新增 ===
    &models.SysReconciliationFixSuggestion{},  // Phase 46 R5
    // ... rest
)
```

**Note**: Numbering skipped from 170 to 182 (Phase 47 likely inserted migrations 171-181 in between). Per the 47-02-SUMMARY.md commit, 195 was the R5 数据清理 migration. R5 will use 182/183/184 per RESEARCH.md §9.1.

---

## Critical Memory Triggers

These are memory entries that MUST be respected during implementation:

| Memory | What to do |
|--------|-----------|
| `stat-cards-from-list-length-capped-at-100` | All stats endpoint use COUNT, never list.length |
| `xingran-server-side-sort-infra` | Use `base.BaseListRequest` + `base.ApplySort` whitelist |
| `xingran-perm-namespace-split-readonly-page` | Perms `asset:reconciliation:fix:*` for R5 |
| `user-prefers-code-fixes-no-db-triggers` | Go service layer + partial unique index, NO DB TRIGGER |
| `xingran-info-point-port-id-varchar` | `ops_asset.user_id` is varchar size:64, no `?::uuid` cast |
| `ad-update-no-such-object-vs-lockout` | No AD managed_by for fix source, use physical chain only |
| `workstation-ad-device-managedby-vs-description` | Physical chain user_id from workstation chain |
| `xingran-migrations-no-sql-autoloader` | `.go` migrations must be explicitly called in database.go |
| `xingran-gorm-sql-constraint-naming-conflict` | Unique index naming `uni_*_*` |
| `migration-sql-name-must-match-model` | Column names match DB schema, not GORM derivation |
| `GORM AutoMigrate 被 PG 物化视图阻塞` | R5 fix_suggestion table NOT referenced by MV (safe) |
| `pg-any-null-three-valued-logic-trap` | NULL-safe WHERE for nullable array columns if filtered |
| `mac-format-deploy-drift-not-cache-stale` | Verify deployment match before suspecting code bugs |
| `mvExists 三态必须显式初始化为 -1` | If R5 service has MV probe, init to -1 not 0 |
| `reconciliation_normalized 三路 MV 联动` | Refresh cascades: `reconciliation_normalized` → `reconciliation_physical_chain` → `reconciliation_user_lookup` |
| `handler 反查 wsID 用 sql.ErrNoRows` | `gorm.Row().Scan()` returns `sql.ErrNoRows`, not `gorm.ErrRecordNotFound` |
| `operlog 11 关键词脱敏` | `rejectionReason` / `rollbackReason` not in keywords, `Record` sufficient |
| `operlog 25 OperType 常量值不可重排` | Use OperTypeReset=11 for rollback (closest to "restore") |
| `Excel 导入 UpsertKey 列必须配 DBField` | N/A for R5 (no Excel import) |
| `client_secret 等敏感关键词 11 个` | N/A for R5 API endpoints (no body fields match keywords) |

---

## Cross-Reference: File Creation Order (atomic commit batches)

**Batch 1: Schema + DI plumbing** (Plan 46-01 Task 1-2)
- `internal/models/reconciliation_fix_suggestion.go` (new)
- `internal/core/db/migrations/migration_182_create_fix_suggestion_table.go` (new)
- `internal/core/db/migrations/migration_183_fix_suggestion_unique_index.go` (new)
- `internal/core/db/database.go` (modify: add to AutoMigrate + register migrations)

**Batch 2: Service layer** (Plan 46-01 Task 3-5)
- `internal/services/asset/fix_suggestion_service.go` (new: interface + impl + Accept/Reject/Apply/Rollback/GetByID/List)
- `internal/services/asset/fix_suggestion_generator.go` (new: cron GenerateFixSuggestions)
- `internal/services/asset/fix_suggestion_stats.go` (new: Stats + StatsTrend)
- `internal/services/asset/cache_keys.go` (modify: add CacheKeyReconciliationFixSuggestion* if needed)
- `internal/core/db/migrations/migration_184_fix_suggestion_config_seeds.go` (new: 4 sys_config + 1 menu + 5 buttons + 1 sys_job)
- `internal/scheduler/reconciliation_fix_suggestion_generator.go` (new: cron registration)

**Batch 3: Handler + Router** (Plan 46-01 Task 6-7)
- `internal/api/v1/asset/fix_suggestion_handler.go` (new)
- `internal/api/v1/asset/fix_suggestion_router.go` (new)
- `internal/api/v1/asset/reconciliation_handler.go` (modify: add ModuleReconciliationFixSuggestion const)
- `internal/api/router.go` (modify: register new router)

**Batch 4: Frontend** (Plan 46-01 Task 8-9)
- `xingran-react-frontend/src/lib/assetApi.ts` (modify: add fixSuggestionApi + types)
- `xingran-react-frontend/src/lib/queryKeys.ts` (modify: add 3 keys)
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx` (new)
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx` (new)
- `xingran-react-frontend/src/App.tsx` or routes config (modify: register `/asset/reconciliation/fix-suggestion`)

**Batch 5: Tests + verification** (Plan 46-01 Task 10-11)
- `internal/services/asset/fix_suggestion_service_test.go` (new: 5 test classes)
- `go build ./...` (verify no compile errors)
- `go test ./...` (verify existing tests pass)

**Batch 6: Plan 46-02 (rollback + monitoring)**
- `fix_suggestion_stats.go` already done in Batch 2
- Optional: `internal/services/asset/fix_suggestion_monitor.go` (SysNotice on threshold breach)
- UI: 5 KPI cards, RejectModal, RollbackModal, 7d countdown in Drawer
- E2E UAT: list → accept → apply → rollback

---

## Summary of Pattern Reuse

| New File | Reuses Pattern From | Key Adaptation |
|----------|--------------------|----------------|
| `reconciliation_fix_suggestion.go` | `reconciliation.go:22-60` | Add 状态机字段 + 操作人 + rollback_window |
| `migration_182` | `migration_168:40-72` | AutoMigrate + 4 indexes (no MV) |
| `migration_183` | `migration_168:139-157` | partial unique index on (exception_id) WHERE pending |
| `migration_184` | `migration_169` | 4 configs + 1 menu + 5 buttons + 1 job (no dicts) |
| `fix_suggestion_service.go` | `reconciliation_service.go:131-228` | Multi-table tx + 状态机 + 7d window check |
| `fix_suggestion_generator.go` | `reconciliation_workorder.go` | cron @every 5m, 物理链路 JOIN |
| `fix_suggestion_stats.go` | `reconciliation_statistics.go:172-237` | 6 COUNT + 7d window + mis_fix_rate |
| `fix_suggestion_handler.go` | `reconciliation_handler.go:114-196` | 5 endpoints + operlog 4 OperTypes |
| `fix_suggestion_router.go` | `reconciliation_router.go:30-44` | 5 routes + 1 stats route |
| `fix-suggestion/index.tsx` | `exceptions/index.tsx:84-540` | 5 KPI + 8 列 + Drawer + Modals |
| `FixSuggestionDetailDrawer.tsx` | R4 ReconciliationDrawer | 3 Tab (冲突摘要/修复详情/历史变更) |
| `queryKeys.ts` | `queryKeys.ts:45-87` | 3 factory functions |
| `assetApi.ts` | `assetApi.ts:166-200` | 7-method object literal + 4 type definitions |

---

**PATTERN MAPPING COMPLETE**

**Output file**: `D:\CODE\ClaudeCode\xingran-go-backend\.claude\worktrees\phase47-discuss\.planning\phases\46-r5\46-PATTERNS.md`

**Next step**: `/gsd-plan-phase` to generate PLAN.md (Task #4).
