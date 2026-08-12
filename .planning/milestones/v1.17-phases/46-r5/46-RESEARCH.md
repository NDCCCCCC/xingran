# Phase 46: 半自动修复（R5）— Research

**Phase:** 46 — 半自动修复（R5, v1.17 资产对账 milestone 收官）
**Goal:** 高置信度（≥0.9）Type B 异常走"建议修复"流程（人工确认 + 仅写 `ops_asset.user_id` + 7d 回滚 + < 1% 误修复监控 + 完整 operlog 审计链），完成 v1.17 milestone close。
**Status:** Ready for plan-phase
**Researcher:** gsd-phase-researcher
**Date:** 2026-07-03

---

## 0. 摘要与执行导览

R5 范围**严格**锁定于 `CONTEXT.md` D-A1~D4 / D-B1~D4 / D-C1~D5 / D-D1~D4 共 18 决策。本研究文档聚焦于**落地路径**——把决策映射为文件、行号、SQL 模板、状态机迁移表、并发约束、API 契约、UI 拆分，并显式标注**与 R1-R4 的可复用点**与**新引入的独立性**。

**核心结构**：

| 区段 | 内容 | 落地形态 |
|------|------|----------|
| §1 | 复用 R1-R4 代码模式（4 类） | 锚定行号 + 行级示例 |
| §2 | `sys_reconciliation_fix_suggestion` 数据模型 | GORM struct + 索引 + CHECK |
| §3 | 6 状态状态机（含迁移表 + 并发约束） | 状态转移图 + SQL 模板 |
| §4 | 建议生成触发器（3 方案对比 + 终选） | 决策矩阵 + 推荐 |
| §5 | 5+ API 端点契约 | 路径 / 入参 / 响应 / 权限 |
| §6 | 前端页面设计（独立页 + Drawer + KPI） | queryKey + 组件拆分 |
| §7 | operlog 集成点 | OperType 选择 + 字段示例 |
| §8 | 缓存失效时机 | invalidate 顺序 |
| §9 | Migration 拆分（3 个文件） | DDL + partial uniqueIndex + config seeds |
| §10 | 误修复率监控（SQL + 滑动窗口） | stats 端点响应结构 |
| §11 | 回滚机制（SQL + 7d 窗口校验） | pre_fix_user_id 还原 |
| §12 | 单元测试目标（5 类） | SQLite + PG 兼容 |
| §13 | 风险与缓解（6 项） | 重点 PG 并发 + MV 阻塞 |

---

## 1. R1-R4 代码模式复用点（4 大类）

R5 不重建轮子 — 4 类现成模式可直接复用，每一类都给出锚定行号。

### 1.1 partial uniqueIndex 模式（D-B4 用）

**出处**：`internal/core/db/migrations/migration_168_reconciliation_tables.go:139-156`

```sql
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
```

**R5 复用模板**（D-B4 锁定，`uniq_fix_suggestion_pending_per_exception`）：

```sql
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_fix_suggestion_pending_per_exception'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
                 ON sys_reconciliation_fix_suggestion (exception_id)
                 WHERE fix_status = ''pending'' AND superseded_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
```

**注意**：
- 命名遵循 `uni_*_*` 规范（参 `xingran-gorm-sql-constraint-naming-conflict` 教训）
- `WHERE` 内不含 `resolved_at`（建议表无此概念，用 `fix_status` + `superseded_at`）
- DO $$ 块包住以幂等 — 多次运行不会重复建索引

### 1.2 缓存失效 helper 模式（D-C4 用）

**出处**：`internal/services/asset/cache_keys.go:112-133` (`InvalidateWorkstationHealth`)

```go
func InvalidateWorkstationHealth(ctx context.Context, c cache.Cache, workstationID string) error {
    if c == nil || workstationID == "" {
        return nil
    }
    return c.Delete(ctx, GetReconciliationHealthByWorkstationKey(workstationID))
}
```

**R5 复用**：在 Apply 路径末尾调用（详见 §8），传入 `wsID`（通过 reconciliation_normalized JOIN sys_data_reconciliation 反查）。`ResolveException` 的 168-185 行已经实现该反查模式（带 `sql.NullString` 容错），可直接复制。

### 1.3 R1 Statistics 端点模式（D-C5 用）

**出处**：`internal/services/asset/reconciliation_statistics.go:172-237` (`Summary`)

5 KPI 全走 COUNT(*) 聚合 → 严禁 list.length（参 `stat-cards-from-list-length-capped-at-100`）。

**R5 复用**：新增 `FixSuggestionStats` 端点，沿用以下模式：
- 7 个 COUNT 全部走 `Model(&SysReconciliationFixSuggestion{}).Where(...).Count(&v)`
- `MisFixRate = rolledBack / applied` 走 PG `CASE WHEN` 单查询（applied=0 时返回 0）
- `Threshold` 走 `system.ConfigService.GetByKey(ctx, "asset.reconciliation.fix.mis_fix_threshold")`
- `TrendSeries` 复用 `HealthTrend` 的 dialect-aware 模式（`date_trunc` PG / `strftime` SQLite）

### 1.4 异常列表 R1 端点模式（D-D1/D-D2 用）

**出处**：`internal/services/asset/reconciliation_service.go:405-540` (`ListExceptions`)

- 嵌入 `base.BaseListRequest` 自动获得 `current / pageSize / orderByColumn / isAsc`
- `ApplySort` 白名单：`reconAllowedSortFields` map 限定
- `Joins()` 链式累加必须用全新 session（fallback 路径）— R5 同样需注意

**R5 复用**：`ListFixSuggestions` 服务复用同形态，5 个过滤字段 + 1 个白名单排序：
```go
var fixAllowedSortFields = map[string]string{
    "createdAt":        "sys_reconciliation_fix_suggestion.created_at",
    "confidenceScore":  "sys_reconciliation_fix_suggestion.confidence_score",
    "fixStatus":        "sys_reconciliation_fix_suggestion.fix_status",
    "appliedAt":        "sys_reconciliation_fix_suggestion.applied_at",
}
```

---

## 2. 数据模型设计 — `sys_reconciliation_fix_suggestion`

### 2.1 GORM Struct（`internal/models/fix_suggestion.go`）

**字段锁定**（24 个，按用途分组）：

```go
// SysReconciliationFixSuggestion 半自动修复建议 (Phase 46 R5)
//
// 状态机(D-B2 6 状态):pending / accepted / rejected / applied / rolled_back / failed
// 并发控制(D-B4):partial unique index uniq_fix_suggestion_pending_per_exception
//   (exception_id) WHERE fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL
// 一对多版本化(D-B3):同一 exception_id 可多轮建议,旧建议 superseded_at=NOW()
//
// 不属于 R5 scope:AD managed_by 修复源(锁定 D-A1 仅物理链路)
type SysReconciliationFixSuggestion struct {
    BaseModel

    // === 关联 ===
    // ExceptionID FK → sys_data_reconciliation.id(uuid) — 1:N 关系
    ExceptionID string `gorm:"type:uuid;not null;column:exception_id;index:idx_fix_suggestion_exception,priority:1" json:"exceptionId"`

    // === 修复源(D-A1 锁定)===
    // SuggestedUserID 物理链路推导的 user_id (port_mac → info_point → workstation → user_id)
    SuggestedUserID *string `gorm:"size:64;column:suggested_user_id" json:"suggestedUserId,omitempty"`
    // PreFixUserID 修复前 ops_asset.user_id(applied 时持久化,rollback 时还原 — D-C1)
    PreFixUserID *string `gorm:"size:64;column:pre_fix_user_id" json:"preFixUserId,omitempty"`

    // === 置信度与原因 ===
    ConfidenceScore float64 `gorm:"type:decimal(3,2);not null;column:confidence_score" json:"confidenceScore"`
    Reason          string  `gorm:"type:text;not null;column:reason" json:"reason"`

    // === 状态机(D-B2 6 状态)===
    FixStatus string `gorm:"size:16;not null;default:'pending';column:fix_status;index:idx_fix_suggestion_status,priority:1" json:"fixStatus"`
    // ConflictType 冗余(便于 list 端点按类型筛选,免 JOIN sys_data_reconciliation)
    ConflictType string `gorm:"size:2;not null;column:conflict_type" json:"conflictType"`

    // === 时间戳(分别记录各状态变更)===
    AcceptedAt  *time.Time `gorm:"column:accepted_at" json:"acceptedAt,omitempty"`
    RejectedAt  *time.Time `gorm:"column:rejected_at" json:"rejectedAt,omitempty"`
    AppliedAt   *time.Time `gorm:"column:applied_at" json:"appliedAt,omitempty"`
    RolledBackAt *time.Time `gorm:"column:rolled_back_at" json:"rolledBackAt,omitempty"`

    // === 操作人(可空 — 自动生成时为系统)===
    AcceptedBy  *string `gorm:"size:64;column:accepted_by" json:"acceptedBy,omitempty"`
    RejectedBy  *string `gorm:"size:64;column:rejected_by" json:"rejectedBy,omitempty"`
    AppliedBy   *string `gorm:"size:64;column:applied_by" json:"appliedBy,omitempty"`
    RolledBackBy *string `gorm:"size:64;column:rolled_back_by" json:"rolledBackBy,omitempty"`

    // === 拒绝原因 / 回滚原因(均必填,D-C3 审计)===
    RejectionReason *string `gorm:"type:text;column:rejection_reason" json:"rejectionReason,omitempty"`
    RollbackReason  *string `gorm:"type:text;column:rollback_reason" json:"rollbackReason,omitempty"`

    // === 回滚窗口(D-C2 固定 7d)===
    RollbackWindowUntil *time.Time `gorm:"column:rollback_window_until" json:"rollbackWindowUntil,omitempty"`

    // === 多轮版本化(D-B3)===
    // SupersededAt 非空 = 旧轮建议,新轮 pending 时通过事务写
    SupersededAt *time.Time `gorm:"column:superseded_at" json:"supersededAt,omitempty"`

    // === 客户端 IP 与 UA(审计追溯)===
    ApplyClientIP   *string `gorm:"size:64;column:apply_client_ip" json:"applyClientIp,omitempty"`
    RollbackClientIP *string `gorm:"size:64;column:rollback_client_ip" json:"rollbackClientIp,omitempty"`
}

func (SysReconciliationFixSuggestion) TableName() string {
    return "sys_reconciliation_fix_suggestion"
}
```

### 2.2 索引设计

| 索引名 | 列 | 类型 | 用途 |
|--------|------|------|------|
| `idx_fix_suggestion_exception` | `(exception_id)` | 普通 | 1:N 反查（fixStatus 过滤后） |
| `idx_fix_suggestion_status` | `(fix_status)` | 普通 | 按状态筛选 |
| `idx_fix_suggestion_status_created` | `(fix_status, created_at)` | 复合 | list 默认排序 |
| `uniq_fix_suggestion_pending_per_exception` | `(exception_id) WHERE fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL` | **partial unique** | **D-B4 并发约束** |
| `idx_fix_suggestion_applied_at` | `(applied_at)` | 普通 | 7d 滑动窗口扫描 |

### 2.3 CHECK 约束（防御性，可选）

```sql
ALTER TABLE sys_reconciliation_fix_suggestion
  ADD CONSTRAINT chk_fix_suggestion_status
  CHECK (fix_status IN ('pending', 'accepted', 'rejected', 'applied', 'rolled_back', 'failed'));
```

**说明**：GORM AutoMigrate 不生成 CHECK；需在 migration_NNN 中显式 `db.Exec` 添加。同样对 `conflict_type IN ('A','B','C','D','E','F')`。

### 2.4 soft delete 与 GORM tag

- `BaseModel.DeletedAt` 自动 GORM soft delete（参 `models/base.go:15`）
- `DeletedAt` gorm tag `gorm:"index"` — 已存在
- `BaseModel.Version` 乐观锁（虽然 partial unique 已足够，但 R5 推荐双保险，R4 ResolveException 未用 — 由 planner 决定）

---

## 3. 状态机实现（D-B2 6 状态 + D-B4 并发约束）

### 3.1 状态转移图

```
                    ┌─────────────┐
                    │   pending   │  ←─ 生成(DetectLayer3 sync OR list lazy OR cron)
                    │ (等待确认)  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │ accepted │  │ rejected │  │(自动)      │
       │ (已接受) │  │ (已拒绝) │  │ superseded│ ← 新轮生成时旧轮标记
       │ (待应用) │  │ (终态)   │  │           │
       └────┬─────┘  └──────────┘  └──────────┘
            ▼
            │ 事务内 atomic UPDATE
            │ SET fix_status='applied', applied_at=NOW(),
            │     pre_fix_user_id=ops_asset.user_id,
            │     ops_asset.user_id=suggested_user_id
            ▼
       ┌──────────┐
       │ applied  │  ← 7d 内可回滚
       │ (已应用) │
       └────┬─────┘
            │
   ┌────────┴────────┐
   ▼                 ▼
┌──────────┐  ┌──────────┐
│rolled_back│  │  failed  │
│(已回滚)  │  │(应用失败)│
│(终态)    │  │(终态)    │
└──────────┘  └──────────┘
```

### 3.2 状态转移表（6 状态 × 6 状态 = 36 格，仅 5 转移合法）

| From → To | pending | accepted | rejected | applied | rolled_back | failed |
|-----------|---------|----------|----------|---------|-------------|--------|
| **pending** | ❌ (重复) | ✅ accept | ✅ reject | ❌ (必须经 accepted) | ❌ | ❌ |
| **accepted** | ❌ | ❌ | ❌ | ✅ apply | ❌ | ✅ apply 失败 |
| **rejected** | ❌（终态） | ❌ | ❌ | ❌ | ❌ | ❌ |
| **applied** | ❌ | ❌ | ❌ | ❌ | ✅ rollback(7d 内) | ❌ |
| **rolled_back** | ❌（终态） | ❌ | ❌ | ❌ | ❌ | ❌ |
| **failed** | ❌（终态） | ❌ | ❌ | ❌ | ❌ | ❌ |

**D-D3**：单条接受（不批量），所以接受时只有 1 个 pending 行的 `WHERE fix_status='pending' AND id=?` 谓词命中。

### 3.3 并发安全（双层防御）

**Layer 1 — partial unique index（DB 层硬拦截）**：
```sql
CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
  ON sys_reconciliation_fix_suggestion (exception_id)
  WHERE fix_status = 'pending' AND superseded_at IS NULL AND deleted_at IS NULL;
```

**Layer 2 — 事务 + 条件 UPDATE（应用层软拦截）**：
```go
// Accept 端点
tx := s.db.WithContext(ctx).Begin()
defer tx.Rollback()

// 软拦截:UPDATE 仅当 fix_status='pending' 时
result := tx.Model(&models.SysReconciliationFixSuggestion{}).
    Where("id = ? AND fix_status = ? AND superseded_at IS NULL AND deleted_at IS NULL",
        suggestionID, "pending").
    Updates(map[string]interface{}{
        "fix_status":  "accepted",
        "accepted_at":  now,
        "accepted_by":  userID,
    })
if result.Error != nil {
    return result.Error
}
if result.RowsAffected == 0 {
    return errors.New("该建议已被处理或不存在")
}
// 提交
tx.Commit()
```

**Layer 3 — partial unique index 提供兜底**：
即使应用层 WHERE 子句有竞态（两个 accept 并发），DB 层 UNIQUE 也会让第二个 INSERT 失败 → SQLSTATE 23505 → handler 返回 409 Conflict。

### 3.4 Apply 路径（多表事务）

**关键**：ops_asset.user_id 写 + sys_reconciliation_fix_suggestion 状态更新 = 同一事务。

```go
func (s *fixSuggestionServiceImpl) Apply(ctx context.Context, suggestionID, userID string) error {
    tx := s.db.WithContext(ctx).Begin()
    defer tx.Rollback() // 函数退出未 commit 即回滚

    // 1. 读 pending 建议(必须接受后才能 apply — 接受后 fix_status='accepted')
    var sugg models.SysReconciliationFixSuggestion
    if err := tx.Where("id = ? AND fix_status = ? AND deleted_at IS NULL",
        suggestionID, "accepted").First(&sugg).Error; err != nil {
        return err
    }

    // 2. 读 ops_asset 当前 user_id
    var asset models.Asset
    if err := tx.Where("id = ? AND deleted_at IS NULL",
        sugg.AssetID /* 通过 exception 反查 */ ).First(&asset).Error; err != nil {
        return err
    }
    preFixUserID := asset.UserID

    // 3. UPDATE ops_asset.user_id
    if err := tx.Model(&asset).Update("user_id", sugg.SuggestedUserID).Error; err != nil {
        return err
    }

    // 4. UPDATE 建议状态
    if err := tx.Model(&sugg).Updates(map[string]interface{}{
        "fix_status":            "applied",
        "applied_at":            now,
        "applied_by":            userID,
        "pre_fix_user_id":       preFixUserID,
        "rollback_window_until": now.Add(7 * 24 * time.Hour),
    }).Error; err != nil {
        return err
    }

    tx.Commit()
    return nil
}
```

**注意**：`sugg` 关联到 `exception_id` → 通过 JOIN `sys_data_reconciliation` 取 `asset_id`（建议表不存 asset_id，避免数据冗余 — 由 planner 决定是否冗余存储）。建议 R5 在建议表加 `asset_id` 字段方便回滚时直接 SQL UPDATE（不必 JOIN exception 表），但 D-A1 锁定"建议字段最小化"，asset_id 通过 exception 反查是更小变更。

---

## 4. 建议生成触发器（3 方案对比 + 终选）

### 4.1 方案 A — DetectLayer3 同步生成

**位置**：`internal/services/asset/reconciliation_detection.go:DetectLayer3` 循环末尾。

**实现**：
```go
// 在 DetectLayer3 成功 UPSERT 异常行后,直接判断是否生成建议
if rec.ConflictType == "B" && rec.ConfidenceScore >= threshold && rec.WorkorderID == nil {
    // 写 sys_reconciliation_fix_suggestion(pending)
    suggestion := &models.SysReconciliationFixSuggestion{
        ExceptionID:     rec.ID,
        SuggestedUserID: rec.PhysicalUserID, // 物理链路推导
        ConfidenceScore: rec.ConfidenceScore,
        Reason:          fmt.Sprintf("物理链路 user_id=%s, ops_asset 当前无责", *rec.PhysicalUserID),
        FixStatus:       "pending",
        ConflictType:    "B",
    }
    if err := s.db.Create(suggestion).Error; err != nil {
        applogger.Warnf("[reconciliation:R5] 生成修复建议失败 exception_id=%s: %v", rec.ID, err)
    }
}
```

**优点**：
- 实时性：建议与异常同步生成
- 无遗漏：每次 DetectLayer3 都会检查
- 简单：单代码路径

**缺点**：
- 影响 DetectLayer3 性能（5min/6min cron 中嵌入额外 INSERT）
- threshold 变更需重启（sys_config 实时生效需动态读取 → 性能影响）

### 4.2 方案 B — 列表查询 lazy 生成

**位置**：`internal/services/asset/reconciliation_service.go:ListExceptions` 末尾。

**实现**：list 查询时对每条 Type B 行 lazy 检查 — 若无 pending 建议且满足 threshold 则生成。

**优点**：
- 零独立 cron 路径
- 用户访问即生成（运维 UAT 友好）

**缺点**：
- 列表查询路径变重（list 应该是简单查询）
- 无访问的 Type B 永远没建议
- list 性能退化（每行 +1 次 INSERT 风险）

### 4.3 方案 C — 独立 cron（推荐）

**位置**：新增 sys_job 记录 `reconciliation:generateFixSuggestions`，cron `@every 5m`（与 DetectLayer3 错开 1min）。

**实现**：参照 R1 的 `reconciliation:detectLayer3` 模式（参 `migration_169_reconciliation_dicts_configs.go:255-265`）：

```go
// internal/scheduler/reconciliation_fix_suggestion_generator.go
func (g *FixSuggestionGenerator) Run(ctx context.Context) error {
    threshold := getThresholdFromConfig(ctx) // 读取 sys_config
    // SELECT * FROM sys_data_reconciliation
    //   WHERE conflict_type='B'
    //     AND confidence_score >= threshold
    //     AND workorder_id IS NULL
    //     AND deleted_at IS NULL
    //     AND resolved_at IS NULL
    //   AND NOT EXISTS (
    //     SELECT 1 FROM sys_reconciliation_fix_suggestion
    //     WHERE exception_id = sys_data_reconciliation.id
    //       AND fix_status = 'pending' AND superseded_at IS NULL
    //   )
    // 逐条 INSERT INTO sys_reconciliation_fix_suggestion ...
    return nil
}
```

**优点**：
- 与 DetectLayer3 解耦，5min 周期错开
- 单一职责，单独可观测（cron 失败告警独立）
- threshold 变更不需要重启（下次 cron 周期重新读取 sys_config）

**缺点**：
- 实时性差：5min 内无建议
- 多 1 个 cron job（已 4 个，加到 5 个）

### 4.4 终选：**方案 C — 独立 cron**

**理由**：
1. **可观测性** — R1-R4 cron 失败可独立告警（参 `reconciliation:detectLayer3` 模式）
2. **解耦** — DetectLayer3 是关键路径（5min 全表扫），加额外写操作影响性能
3. **配置动态生效** — threshold/mis_fix_threshold 变更下次 cron 周期即生效（参 `INFRA-02` 锁定）
4. **多轮建议支持** — 旧轮 superseded 后，独立 cron 容易实现新轮生成（事务内 UPDATE 旧轮 + INSERT 新轮）

**实施注意**：
- cron 表达式：`@every 5m`（与 `reconciliation:detectLayer3` 同步，但加 1min jitter 避免资源争抢 — 由 planner 决定具体 cron 表达式）
- 多轮建议生成：当 exception 重新 UPSERT（DetectLayer3 命中 UPSERT，2026-07-03 Phase 47 R3 改造）时，若已有 `applied` 建议，cron 应当跳过（避免重复建议）
- INFRA-02 config seed：cron InvokeTarget 需对应注册

---

## 5. API 端点设计（5+ 端点 + D-C5 stats）

### 5.1 路由注册位置

`internal/api/v1/asset/fix_suggestion_router.go`（新增文件），挂载到 `internal/api/router.go` 的 `/asset/reconciliation/fix-suggestion/*` 子路由组。

### 5.2 5+ 端点契约

#### 5.2.1 `POST /fix-suggestion/list`

**权限**：`asset:reconciliation:fix:list`

**Request**：
```typescript
interface FixSuggestionListParams {
  current: number;       // 默认 1
  pageSize: number;      // 默认 20, max 100
  orderByColumn?: string;// 白名单: createdAt / confidenceScore / fixStatus / appliedAt
  isAsc?: 'asc' | 'desc';
  fixStatus?: 'pending' | 'accepted' | 'rejected' | 'applied' | 'rolled_back' | 'failed';
  conflictType?: 'A' | 'B' | 'C' | 'D' | 'E' | 'F';
  responsibleDeptId?: string; // JOIN ops_asset.dept_id 过滤
  createdFrom?: string;  // ISO 8601
  createdTo?: string;    // ISO 8601
}
```

**Response**：
```typescript
interface PageResult<FixSuggestionListItem> {
  list: FixSuggestionListItem[];
  total: number;
  current: number;
  pageSize: number;
}

interface FixSuggestionListItem {
  id: string;
  exceptionId: string;
  assetId: string;        // JOIN sys_data_reconciliation
  assetCode: string;      // JOIN ops_asset.devicesn
  conflictType: 'B';      // 冗余字段
  currentUserId: string | null;   // ops_asset.user_id 当前值
  suggestedUserId: string;
  suggestedUsername: string | null; // JOIN sys_user.username
  preFixUserId: string | null;     // applied 后回填
  confidenceScore: number;
  reason: string;
  fixStatus: FixStatus;
  createdAt: string;
  appliedAt: string | null;
  rolledBackAt: string | null;
  rollbackWindowUntil: string | null;
}
```

**HTTP Codes**：
- 200 OK — 成功
- 400 Bad Request — 参数错误
- 401 Unauthorized — 未登录
- 403 Forbidden — 无 `asset:reconciliation:fix:list` 权限

#### 5.2.2 `POST /fix-suggestion/:id`

**权限**：`asset:reconciliation:fix:list`

**Response**：
```typescript
interface FixSuggestionDetail {
  id: string;
  exceptionId: string;
  // 关联异常完整信息
  exception: {
    id: string;
    assetId: string;
    assetCode: string;
    conflictType: 'B';
    severity: 'high';
    confidenceScore: number;
    rawSnapshot: object;  // JSONB 三路数据
    detectedAt: string;
  };
  // 修复建议
  suggestion: {
    suggestedUserId: string;
    suggestedUsername: string | null;
    preFixUserId: string | null;
    preFixUsername: string | null;
    confidenceScore: number;
    reason: string;
    fixStatus: FixStatus;
    createdAt: string;
    acceptedAt: string | null;
    acceptedBy: string | null;
    appliedAt: string | null;
    appliedBy: string | null;
    rolledBackAt: string | null;
    rolledBackBy: string | null;
    rollbackReason: string | null;
    rollbackWindowUntil: string | null;
    rejectedAt: string | null;
    rejectedBy: string | null;
    rejectionReason: string | null;
  };
  // 历史变更(同 exception_id 的所有 fix_suggestion 记录)
  history: FixSuggestionListItem[];
}
```

**HTTP Codes**：
- 200 OK — 成功
- 404 Not Found — 不存在

#### 5.2.3 `POST /fix-suggestion/:id/accept`

**权限**：`asset:reconciliation:fix:accept`

**Request**：
```typescript
interface AcceptRequest {
  // 可选 — 运维可调整 suggested_user_id 后再 accept(v1.18 R5+ 提供)
  // R5 锁定:不提供修改建议功能,仅确认原值
}
```

**行为**：
1. 事务内 `UPDATE ... WHERE id=? AND fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL SET fix_status='accepted', accepted_at=NOW(), accepted_by=userID`
2. RowsAffected=0 → 返回 409 Conflict（已被其他运维处理）
3. RowsAffected=1 → 成功
4. 调 `operlog.Record(c, ..., ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)` — 写 sys_oper_log

**注意**：
- R5 accept 不立即 apply（**两步式**：accept 后运维可再点 "Apply" — UI 设计）— 或一步式（accept = apply）。D-C2 暗示两步式（accept 是标记已读，apply 是真落库），由 planner 决定。
- 建议：**两步式**（accept 标记已读 → 二次确认 Apply 落库）。但 D-B2 状态机 6 状态显示 accepted 是中间态，所以两步式合理。

**HTTP Codes**：
- 200 OK — 成功
- 400 Bad Request — suggestion 不在 pending 状态
- 404 Not Found — 不存在
- 409 Conflict — 已被处理

#### 5.2.4 `POST /fix-suggestion/:id/reject`

**权限**：`asset:reconciliation:fix:reject`

**Request**：
```typescript
interface RejectRequest {
  rejectionReason: string;  // **必填**(D-C3 审计) — service 层校验 ≥10 字符
}
```

**行为**：同 accept，但 `fix_status='rejected'`。

**HTTP Codes**：同 accept + 400(rejectionReason 太短)。

#### 5.2.5 `POST /fix-suggestion/:id/apply`

**权限**：`asset:reconciliation:fix:accept`（复用 accept 权限，因为两步式合在 apply 落库）

**Request**：
```typescript
interface ApplyRequest {
  // 留空 body — R5 锁定仅写 user_id,无修改建议功能
}
```

**行为**（§3.4 已详述）：
1. 事务内读 accepted 建议
2. 读 ops_asset 当前 user_id
3. UPDATE ops_asset.user_id = suggested_user_id
4. UPDATE 建议 → applied + pre_fix_user_id + rollback_window_until
5. `invalidate_workstation_health(wsID)`（D-C4）
6. 写 operlog(OperTypeUpdate)

**HTTP Codes**：同 accept + 500(应用失败 → 写失败状态)。

#### 5.2.6 `POST /fix-suggestion/:id/rollback`

**权限**：`asset:reconciliation:fix:rollback`

**Request**：
```typescript
interface RollbackRequest {
  rollbackReason: string;  // **必填**(D-C3 审计) — service 层校验 ≥10 字符
}
```

**行为**：
1. 校验 `fix_status='applied'` 且 `NOW() < rollback_window_until`（7d 内可回滚，UI 隐藏但 DB 允许）
2. 事务内：`UPDATE ops_asset.user_id = pre_fix_user_id`
3. `UPDATE 建议 SET fix_status='rolled_back', rolled_back_at=NOW(), rolled_back_by=userID, rollback_reason=reason`
4. `invalidate_workstation_health(wsID)`
5. 写 operlog(**OperTypeReset=11** — D-C3 锁定)

**HTTP Codes**：
- 200 OK — 成功
- 400 Bad Request — 不在 applied 状态 / 超过 7d
- 404 Not Found

#### 5.2.7 `POST /fix-suggestion/stats` (D-C5)

**权限**：`asset:reconciliation:fix:stats`

**Request**：
```typescript
interface FixSuggestionStatsParams {
  windowDays?: number;  // 默认 7
}
```

**Response**（D-D1 KPI 卡片数据源）：
```typescript
interface FixSuggestionStatsResponse {
  windowDays: number;
  pending: number;       // 7d 内 created 但未处理
  accepted: number;      // 7d 内 accepted
  rejected: number;      // 7d 内 rejected
  applied: number;       // 7d 内 applied
  rolledBack: number;    // 7d 内 rolled_back
  failed: number;        // 7d 内 failed
  misFixRate: number;    // rolledBack / applied (applied=0 → 0)
  threshold: number;     // 当前 sys_config mis_fix_threshold
  thresholdBreached: boolean;  // misFixRate > threshold
  trendSeries: TrendPoint[];   // 7d 每日 misFixRate 趋势
}
```

**HTTP Codes**：200 OK。

### 5.3 权限码（D-C5 / 命名空间）

| 权限码 | 端点 | 描述 |
|--------|------|------|
| `asset:reconciliation:fix:list` | list, get | 查看列表与详情 |
| `asset:reconciliation:fix:accept` | accept, apply | 接受/应用 |
| `asset:reconciliation:fix:reject` | reject | 拒绝 |
| `asset:reconciliation:fix:rollback` | rollback | 回滚 |
| `asset:reconciliation:fix:stats` | stats | 查看统计 |

**菜单 seed**：`migration_NNN_reconciliation_fix_menus.go` 添加 5 条 sys_menu（type='F' 按钮权限）+ 1 条 type='C' 菜单（"修复建议"），挂在"资产管理"目录下。**不**自动赋权（参 `migration_169` 锁定"谁也不给"原则）。

---

## 6. 前端页面设计（D-D1 独立页 + D-D2 Drawer + D-D1 KPI）

### 6.1 页面路径

`/asset/reconciliation/fix-suggestion`（独立页面，不复用 R4 ReconciliationDrawer — D-D1 锁定）

### 6.2 文件结构

```
xingran-react-frontend/src/
├── pages/asset/reconciliation/fix-suggestion/
│   ├── index.tsx                          # 主页面（列表 + KPI + Drawer 容器）
│   └── components/
│       ├── FixSuggestionDetailDrawer.tsx  # 详情 Drawer（3 Tab）
│       ├── AcceptApplyModal.tsx           # 接受/应用确认 Modal
│       ├── RejectModal.tsx                # 拒绝 Modal（rejectionReason 必填）
│       └── RollbackModal.tsx              # 回滚 Modal（rollbackReason 必填）
└── lib/assetApi.ts                        # 扩展:fixSuggestionApi 模块
```

### 6.3 queryKeys 注册

`src/lib/queryKeys.ts` 现有 `reconciliation.*` 命名空间扩展（参 `src/lib/queryKeys.ts:44-87`）：

```typescript
reconciliation: {
  // ... R1-R4 既有 key
  /** Phase 46 R5 — 修复建议列表(分页 + 筛选) */
  fixSuggestionList: (params: FixSuggestionListParams) =>
    ["reconciliation", "fix-suggestion-list", params] as const,
  /** Phase 46 R5 — 修复建议详情 */
  fixSuggestionDetail: (id: string) =>
    ["reconciliation", "fix-suggestion-detail", id] as const,
  /** Phase 46 R5 — 修复建议统计(7d KPI) */
  fixSuggestionStats: (windowDays: number) =>
    ["reconciliation", "fix-suggestion-stats", windowDays] as const,
}
```

### 6.4 fixSuggestionApi factory（`src/lib/assetApi.ts`）

参照 `reconciliationApi` factory 模式（既有代码）：

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

### 6.5 顶部 5 KPI 卡片（D-D1 锁定）

| 卡片 | 数据源 | 颜色 |
|------|--------|------|
| 待处理建议数（pending） | stats.pending | warning 黄 |
| 7d 应用数（applied） | stats.applied | success 绿 |
| 7d 回滚数（rolledBack） | stats.rolledBack | error 红 |
| 7d 误修复率（misFixRate） | stats.misFixRate | 阈值内 success / 超阈 error |
| 7d 拒绝数（rejected） | stats.rejected | default 灰 |

**KPI 布局**：antd `Row` + `Col` span={4} (4.x) / flex (5.x)，5 列等宽。

### 6.6 列表列（D-D2 紧凑行）

| 列 | 字段 | 排序 | 宽度 |
|----|------|------|------|
| 资产编号 | assetCode | ✅ | 120 |
| 现 ops_asset.user_id | currentUserId | ❌ | 160 |
| 建议 user_id | suggestedUserId | ❌ | 160 |
| confidence_score | confidenceScore | ✅ | 100 |
| conflict_type | conflictType | ❌ | 80 |
| fix_status | fixStatus | ✅ | 100 |
| created_at | createdAt | ✅ | 160 |
| 操作 | accept/reject/apply/rollback | ❌ | 240 |

**操作列动态**：
- pending → 接受 / 拒绝 按钮
- accepted → 应用 按钮
- applied + within 7d → 回滚 按钮
- 其他状态 → 仅查看

### 6.7 详情 Drawer 3 Tab（D-D2 锁定）

参考 R4 `ReconciliationDrawer` 3 Tab 契约（`src/components/reconciliation/ReconciliationDrawer.tsx`）：

**Tab 1 — 冲突摘要**：
- raw_snapshot 三路数据(physical / declared / ad)展示
- ConflictSignals(HasPhysical/HasDeclared/HasAD)标志位
- conflict_type / severity / confidenceScore

**Tab 2 — 修复详情**：
- 时间轴显示:createdAt → acceptedAt → appliedAt → (rolledBackAt)
- 当前 ops_asset.user_id vs 建议 user_id
- pre_fix_user_id(applied 后回填)
- rollback_window_until 倒计时

**Tab 3 — 历史变更**：
- 同 exception_id 的所有 fix_suggestion 记录
- rejected/accepted/applied/rolled_back 全状态 + 原因

### 6.8 useEffect 依赖稳定（CLAUDE.md 强制）

```typescript
// ✅ 正确
const params = useMemo<FixSuggestionListParams>(
  () => ({ current, pageSize, fixStatus, ...filters }),
  [current, pageSize, fixStatus, /* primitive deps */]
);
useEffect(() => {
  fixSuggestionApi.list(params).then(setData);
}, [params]);

// ❌ 错误
useEffect(() => {
  fixSuggestionApi.list({ current, pageSize, fixStatus, ...filters });
}, [{ current, pageSize, fixStatus, ...filters }]); // 对象重建导致无限循环
```

### 6.9 跳转入口（planner 自决）

D-D1 锁定独立页面。R4 `ReconciliationDrawer` "冲突摘要" Tab 可加跳转链接到 `/asset/reconciliation/fix-suggestion?exception_id=xxx`，由 planner 在 R4 增强时决定。

---

## 7. operlog 集成点（D-C3 锁定）

### 7.1 OperType 选择（D-C3 明确）

| 操作 | OperType | 语义 | 适用端点 |
|------|----------|------|----------|
| Accept | **OperTypeUpdate=2** | 状态变更(pending → accepted) | accept |
| Reject | **OperTypeReject=23** | 审批驳回(语义:拒绝建议) | reject |
| Apply | **OperTypeUpdate=2** | 状态变更(accepted → applied) | apply |
| Rollback | **OperTypeReset=11** | 密码/密钥重置(语义:恢复到原值) | rollback |

### 7.2 operlog.Record 调用位置

```go
// internal/api/v1/asset/fix_suggestion_handler.go

const ModuleReconciliationFixSuggestion = "资产对账-修复建议"

func (h *FixSuggestionHandler) Accept(c *gin.Context) {
    suggestionID := c.Param("id")
    userID := getUserIDFromContext(c)

    if err := h.service.Accept(c.Request.Context(), suggestionID, userID); err != nil {
        // 错误处理
        return
    }

    // operlog 写入(成功路径)
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
        ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

    response.Success(c, gin.H{"id": suggestionID})
}

func (h *FixSuggestionHandler) Rollback(c *gin.Context) {
    // ... 校验 rollbackReason + 7d 窗口 ...

    if err := h.service.Rollback(c.Request.Context(), suggestionID, userID, req.RollbackReason); err != nil {
        return
    }

    // ✅ D-C3 锁定:rollback 强写 operlog + OperTypeReset=11
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
        ModuleReconciliationFixSuggestion, operlog.OperTypeReset)

    response.Success(c, gin.H{"id": suggestionID})
}
```

### 7.3 operlog 字段示例

**Rollback**：
```
Module:    资产对账-修复建议
OperType:  OperTypeReset=11 (恢复到原值)
Title:     "回滚修复建议 #{suggestion_id}: asset {asset_code} user_id {pre} -> {post}"
OperParam: {"rollbackReason": "...", "preFixUserId": "...", "suggestedUserId": "..."}
Method:    POST /asset/reconciliation/fix-suggestion/{id}/rollback
```

**Apply**：
```
Module:    资产对账-修复建议
OperType:  OperTypeUpdate=2 (状态变更)
Title:     "应用修复建议 #{suggestion_id}: asset {asset_code} user_id {pre} -> {suggested}"
OperParam: {"appliedAt": "...", "preFixUserId": "...", "suggestedUserId": "..."}
```

### 7.4 敏感字段脱敏

`rollbackReason` / `rejectionReason` 文本不敏感（运维输入的审计备注），无需 RecordWithBody。**例外**：若 rollbackReason 包含 IP/MAC/凭证（不应发生但防御性考虑），`operlog.RecordWithBody` 自动按 11 关键词脱敏（参 `operlog.go:124-169` `sensitiveKeys`）。

---

## 8. 缓存失效时机（D-C4 锁定）

### 8.1 调用顺序（D-A4-04 锁定模式）

```go
// 严格顺序: service → invalidate → operlog → response
//
// 参考:internal/api/v1/asset/reconciliation_handler.go:166-185 (ResolveException 模式)

func (h *FixSuggestionHandler) Apply(c *gin.Context) {
    // 1. service 事务内 UPDATE ops_asset + fix_suggestion
    if err := h.service.Apply(c.Request.Context(), suggestionID, userID); err != nil {
        return
    }

    // 2. 反查 wsID(参 reconciliation_handler.go:168-180 模式)
    var wsID sql.NullString
    scanErr := h.core.GetDB().WithContext(c.Request.Context()).
        Table("reconciliation_normalized").
        Select("reconciliation_normalized.workstation_id").
        Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
        Joins("JOIN sys_reconciliation_fix_suggestion ON sys_reconciliation_fix_suggestion.exception_id = sys_data_reconciliation.id").
        Where("sys_reconciliation_fix_suggestion.id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL AND reconciliation_normalized.workstation_id IS NOT NULL", suggestionID).
        Limit(1).
        Row().
        Scan(&wsID)
    if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
        applogger.Warnf("[fix-suggestion:Apply] R4 query workstation failed suggestionID=%s: %v", suggestionID, scanErr)
    }

    // 3. 失效工位健康度缓存(D-C4)
    if wsID.Valid && wsID.String != "" {
        if invErr := asset.InvalidateWorkstationHealth(c.Request.Context(), h.core.Cache, wsID.String); invErr != nil {
            applogger.Warnf("[fix-suggestion:Apply] invalidate cache failed suggestionID=%s wsID=%s: %v", suggestionID, wsID.String, invErr)
        }
    }

    // 4. operlog 写入
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
        ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

    // 5. response
    response.Success(c, gin.H{"id": suggestionID, "appliedAt": time.Now()})
}
```

### 8.2 失效时机的细节

| 端点 | 是否失效 | 说明 |
|------|----------|------|
| Accept | ❌ | 状态未变更(pending→accepted),ops_asset.user_id 未变,缓存无需失效 |
| Reject | ❌ | 同上 |
| Apply | ✅ | ops_asset.user_id 真变 → 必须失效 |
| Rollback | ✅ | ops_asset.user_id 恢复 → 必须失效 |

### 8.3 MV 刷新注意

R5 修复后 **不**主动触发 `REFRESH MATERIALIZED VIEW reconciliation_normalized`（避免锁表）。下次 cron 周期（5min）自然刷新 → 重新走 DetectLayer3 → 同 (asset, type) 在 7d 静默期内被 skip（参 `reconciliation_detection.go:254-259` 静默期逻辑）。**D-C4 锁定**：applied 后自动 7d 静默期生效。

---

## 9. Migration 拆分（3 个新文件 + database.go 注册）

### 9.1 文件清单

```
internal/core/db/migrations/
├── migration_182_create_fix_suggestion_table.go   # 表 DDL + 5 索引 + CHECK
├── migration_183_fix_suggestion_unique_index.go   # partial unique index(独立以便幂等)
└── migration_184_fix_suggestion_config_seeds.go   # 4 条 sys_config + 5 条 sys_menu + 1 条 sys_job
```

**编号依据**：
- 当前最大 181（`migration_181_cleanup_dirty_mac_rows.go`）
- R5 接 182/183/184

### 9.2 migration_182 — 表 DDL

```go
// migration_182_create_fix_suggestion_table.go
package migrations

import (
    "log"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "gorm.io/gorm"
)

func Migrate182CreateFixSuggestionTable(db *gorm.DB) error {
    log.Println("Running migration 182: sys_reconciliation_fix_suggestion table")

    // 1. GORM AutoMigrate 创建表
    if err := db.AutoMigrate(&models.SysReconciliationFixSuggestion{}); err != nil {
        return fmt.Errorf("AutoMigrate sys_reconciliation_fix_suggestion 失败: %w", err)
    }

    // 2. 显式添加 5 索引(部分由 gorm tag 自动建,部分需手工)
    indexes := []string{
        // 普通索引
        `CREATE INDEX IF NOT EXISTS idx_fix_suggestion_exception ON sys_reconciliation_fix_suggestion (exception_id)`,
        `CREATE INDEX IF NOT EXISTS idx_fix_suggestion_status ON sys_reconciliation_fix_suggestion (fix_status)`,
        `CREATE INDEX IF NOT EXISTS idx_fix_suggestion_status_created ON sys_reconciliation_fix_suggestion (fix_status, created_at)`,
        `CREATE INDEX IF NOT EXISTS idx_fix_suggestion_applied_at ON sys_reconciliation_fix_suggestion (applied_at)`,
    }
    for _, sql := range indexes {
        if err := db.Exec(sql).Error; err != nil {
            return fmt.Errorf("创建索引失败 %s: %w", sql, err)
        }
    }

    // 3. CHECK 约束(防御性)
    checkSQLs := []string{
        `ALTER TABLE sys_reconciliation_fix_suggestion DROP CONSTRAINT IF EXISTS chk_fix_suggestion_status`,
        `ALTER TABLE sys_reconciliation_fix_suggestion ADD CONSTRAINT chk_fix_suggestion_status CHECK (fix_status IN ('pending', 'accepted', 'rejected', 'applied', 'rolled_back', 'failed'))`,
        `ALTER TABLE sys_reconciliation_fix_suggestion DROP CONSTRAINT IF EXISTS chk_fix_suggestion_conflict_type`,
        `ALTER TABLE sys_reconciliation_fix_suggestion ADD CONSTRAINT chk_fix_suggestion_conflict_type CHECK (conflict_type IN ('A', 'B', 'C', 'D', 'E', 'F'))`,
    }
    for _, sql := range checkSQLs {
        if err := db.Exec(sql).Error; err != nil {
            // CHECK 在 PG-only,SQLite 跳过
            if !isPostgreSQL(db) {
                applogger.Infof("[迁移] 182 CHECK 跳过(非 PostgreSQL): %s", sql)
                continue
            }
            return fmt.Errorf("添加 CHECK 约束失败 %s: %w", sql, err)
        }
    }

    log.Println("Migration 182 completed: sys_reconciliation_fix_suggestion table ready")
    return nil
}
```

### 9.3 migration_183 — partial unique index

```go
// migration_183_fix_suggestion_unique_index.go
package migrations

import (
    "log"
    applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
    "gorm.io/gorm"
)

func Migrate183FixSuggestionUniqueIndex(db *gorm.DB) error {
    log.Println("Running migration 183: partial unique index uniq_fix_suggestion_pending_per_exception")

    if !isPostgreSQL(db) {
        applogger.Infof("[迁移] 183 跳过(非 PostgreSQL)")
        return nil
    }

    // D-B4 锁定:与 R1 uniq_recon_asset_type_open 同模式
    partialUniqueSQL := `
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_fix_suggestion_pending_per_exception'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_fix_suggestion_pending_per_exception
                 ON sys_reconciliation_fix_suggestion (exception_id)
                 WHERE fix_status = ''pending'' AND superseded_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
`
    if err := db.Exec(partialUniqueSQL).Error; err != nil {
        return fmt.Errorf("创建 partial unique index 失败: %w", err)
    }
    applogger.Infof("[迁移] uniq_fix_suggestion_pending_per_exception 已就位")
    return nil
}
```

### 9.4 migration_184 — INFRA-02 config seeds

```go
// migration_184_fix_suggestion_config_seeds.go
package migrations

func Migrate184FixSuggestionConfigSeeds(db *gorm.DB) error {
    log.Println("Running migration 184: Phase 46 R5 config + menu + sys_job seeds")

    if err := seedFixSuggestionConfigs(db); err != nil {
        return err
    }
    if err := seedFixSuggestionMenu(db); err != nil {
        return err
    }
    if err := seedFixSuggestionSysJob(db); err != nil {
        return err
    }
    return nil
}

// seedFixSuggestionConfigs seed 4 条 sys_config(D-A3 + D-C5)
func seedFixSuggestionConfigs(db *gorm.DB) error {
    configs := []struct {
        configName  string
        configKey   string
        configValue string
        remark      string
    }{
        {"修复建议置信度门槛", "asset.reconciliation.fix.confidence_threshold", "0.9", "R5: 触发建议生成的最低 confidence_score(D-A3 锁定)"},
        {"误修复率告警阈值", "asset.reconciliation.fix.mis_fix_threshold", "0.01", "R5: rolled_back / applied 超过该值触发 SysNotice(D-C5 锁定)"},
        {"回滚窗口天数", "asset.reconciliation.fix.rollback_window_days", "7", "R5: applied 后可回滚的窗口(D-C2 锁定)"},
        {"R5 功能总开关", "asset.reconciliation.fix.enabled", "1", "R5: 0=暂停生成建议,1=启用(运维紧急熔断)"},
    }
    // 沿用 migration_169 seedReconciliationConfigs 的 count-then-insert 模式
    // ...
}

// seedFixSuggestionMenu seed 1 路由菜单 + 5 按钮权限(D-04 锁定"谁也不给"原则 — 不插入 sys_role_menu)
func seedFixSuggestionMenu(db *gorm.DB) error {
    // 创建菜单:资产管理 > 修复建议
    // 5 按钮权限:list / accept / reject / rollback / stats
    // 命名空间 asset:reconciliation:fix:*
    // ...
}

// seedFixSuggestionSysJob seed 1 条 sys_job
func seedFixSuggestionSysJob(db *gorm.DB) error {
    jobs := []struct {
        jobName        string
        invokeTarget   string
        cronExpression string
        remark         string
    }{
        {"对账-修复建议生成", "reconciliation:generateFixSuggestions", "@every 5m", "R5: 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion"},
    }
    // count-then-insert(参 migration_169)
    // ...
}
```

### 9.5 database.go AutoMigrate 注册

`internal/core/db/database.go` `AutoMigrate` 函数（约 288 行）需添加 `&models.SysReconciliationFixSuggestion{}`：

```go
err := d.DB.Migrator().AutoMigrate(
    // ... 既有模型
    &models.SysReconciliationFixSuggestion{},  // Phase 46 R5
    // ...
)
```

**注意**：参 `xingran-migrations-no-sql-autoloader` 记忆 — migrations/*.sql 不会被自动加载，但 `.go` migration 文件必须**显式**在 `migrations.RegisterMigrations()` 或等价调度函数中调用 `Migrate182xxx`/`Migrate183xxx`/`Migrate184xxx`。

---

## 10. 误修复率监控（D-C5 锁定）

### 10.1 SQL 查询（PG）

```sql
-- 7d 滑动窗口
SELECT
    COUNT(*) FILTER (WHERE fix_status = 'pending')   AS pending,
    COUNT(*) FILTER (WHERE fix_status = 'accepted')  AS accepted,
    COUNT(*) FILTER (WHERE fix_status = 'rejected')  AS rejected,
    COUNT(*) FILTER (WHERE fix_status = 'applied' AND applied_at >= NOW() - INTERVAL '7 day')  AS applied,
    COUNT(*) FILTER (WHERE fix_status = 'rolled_back' AND rolled_back_at >= NOW() - INTERVAL '7 day') AS rolled_back,
    COUNT(*) FILTER (WHERE fix_status = 'failed')    AS failed
FROM sys_reconciliation_fix_suggestion
WHERE deleted_at IS NULL
  AND created_at >= NOW() - INTERVAL '7 day';

-- 7d 每日趋势
SELECT
    TO_CHAR(date_trunc('day', created_at), 'YYYY-MM-DD') AS date,
    COUNT(*) FILTER (WHERE fix_status = 'applied')      AS applied,
    COUNT(*) FILTER (WHERE fix_status = 'rolled_back') AS rolled_back,
    CASE WHEN COUNT(*) FILTER (WHERE fix_status = 'applied') = 0
         THEN 0
         ELSE ROUND(
             COUNT(*) FILTER (WHERE fix_status = 'rolled_back')::numeric
             / COUNT(*) FILTER (WHERE fix_status = 'applied')::numeric,
             4)
    END AS mis_fix_rate
FROM sys_reconciliation_fix_suggestion
WHERE deleted_at IS NULL
  AND created_at >= NOW() - INTERVAL '7 day'
GROUP BY date_trunc('day', created_at)
ORDER BY date_trunc('day', created_at) ASC;
```

### 10.2 Dialect 兼容（参 `HealthTrend` 模式）

SQLite 不支持 `FILTER` 子句，需 fallback 用 `SUM(CASE WHEN ... THEN 1 ELSE 0 END)` — 参 `reconciliation_statistics.go:357-383`。

### 10.3 阈值告警（D-C5 锁定）

- 误修复率 = `rolledBack / applied`（applied=0 → 0）
- 阈值 = `sys_config: asset.reconciliation.fix.mis_fix_threshold` (默认 0.01)
- 超过 → 写 `sys_notice`(参 R2 `MONITOR-03` 已建的 SysNotice 写入模式)
- 由 stats 端点返回值 `thresholdBreached: bool` 决定是否触发(避免 cron 单独任务)

### 10.4 SysNotice 写入模式（参 R2）

`internal/websocket/notice_hub.go` 已有 SysNotice 写入 helper — 复用：

```go
// internal/services/asset/fix_suggestion_monitor.go
func (m *FixSuggestionMonitor) CheckAndNotify(ctx context.Context) error {
    stats, err := m.statsService.GetStats(ctx, 7)
    if err != nil {
        return err
    }
    if stats.ThresholdBreached {
        // 写 sys_notice(参 R2 MONITOR-03 模式)
        m.noticeHub.Send(ctx, &Notice{
            Title:   "资产对账误修复率超阈",
            Content: fmt.Sprintf("7d 误修复率 %.2f%% 超过阈值 %.2f%%", stats.MisFixRate*100, stats.Threshold*100),
            Type:    "warning",
        })
    }
    return nil
}
```

### 10.5 R3 baseline 对比端点

CONTEXT 提到"与 R3 降噪基线对比端点同模式" — R3 baseline 端点已建（参 queryKeys `baselineCompare`），R5 stats 可选择性加 baseline compare 字段。**planner 自决**是否纳入。

---

## 11. 回滚机制（D-C1 + D-C2 + D-C3 锁定）

### 11.1 触发条件（D-C2）

```go
// service 层强校验
if sugg.FixStatus != "applied" {
    return errors.New("仅已应用建议可回滚")
}
if time.Now().After(*sugg.RollbackWindowUntil) {
    return errors.New("回滚窗口已过(7d)")
}
```

### 11.2 SQL 更新（事务内）

```go
func (s *fixSuggestionServiceImpl) Rollback(ctx context.Context, suggestionID, userID, reason string) error {
    if len(reason) < 10 {
        return errors.New("回滚原因至少 10 字符")
    }

    tx := s.db.WithContext(ctx).Begin()
    defer tx.Rollback()

    // 1. 读 applied 建议
    var sugg models.SysReconciliationFixSuggestion
    if err := tx.Where("id = ? AND fix_status = ? AND deleted_at IS NULL",
        suggestionID, "applied").First(&sugg).Error; err != nil {
        return err
    }

    // 2. 校验窗口
    if time.Now().After(*sugg.RollbackWindowUntil) {
        return errors.New("回滚窗口已过(7d)")
    }

    // 3. 反查 asset_id(通过 exception)
    var exception models.SysDataReconciliation
    if err := tx.Where("id = ? AND deleted_at IS NULL", sugg.ExceptionID).First(&exception).Error; err != nil {
        return err
    }

    // 4. 还原 ops_asset.user_id
    if err := tx.Model(&models.Asset{}).
        Where("id = ? AND deleted_at IS NULL", exception.AssetID).
        Update("user_id", sugg.PreFixUserID).Error; err != nil {
        return err
    }

    // 5. 更新建议状态
    now := time.Now()
    if err := tx.Model(&sugg).Updates(map[string]interface{}{
        "fix_status":        "rolled_back",
        "rolled_back_at":    now,
        "rolled_back_by":    userID,
        "rollback_reason":   reason,
        "rollback_client_ip": getClientIP(ctx), // 来自 gin context 或 service 注入
    }).Error; err != nil {
        return err
    }

    tx.Commit()
    return nil
}
```

### 11.3 UI 控制（D-C2）

- 7d 内：Drawer 显示"回滚"按钮 → 弹 Modal 输入 rollbackReason(≥10 字符)→ 提交
- 7d 外：UI 隐藏按钮（前端 `appliedAt + 7d < NOW() ? hidden : visible`）
- DB 仍允许强制回滚（service 层校验可 bypass）— planner 决定是否加 admin 强制回滚端点

### 11.4 operlog 强写（D-C3）

```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
    ModuleReconciliationFixSuggestion, operlog.OperTypeReset)  // = 11
```

### 11.5 缓存失效

`invalidate_workstation_health(wsID)`(同 Apply 路径)。

---

## 12. 单元测试目标

### 12.1 单元测试文件

```
internal/services/asset/
├── fix_suggestion_service_test.go     # Service 层单测
├── fix_suggestion_generator_test.go   # 触发器单测
└── testdata/fix_suggestion.sql        # 测试 fixtures
```

### 12.2 5 类测试目标

#### 12.2.1 UPSERT 逻辑

```go
// 1. 同一 exception_id 多次 INSERT 仅 1 条 pending 存活
// 2. 其他 1 条因 partial unique 冲突 → 23505
// 3. 失败 23505 → 业务层捕获并转为"已被处理"错误
```

#### 12.2.2 状态转移

```go
// 表驱动测试:6 状态 × 6 转移
tests := []struct{
    name string
    from FixStatus
    to   FixStatus
    op   string
    ok   bool
}{
    {"pending→accepted", "pending", "accepted", "Accept", true},
    {"pending→rejected", "pending", "rejected", "Reject", true},
    {"accepted→applied", "accepted", "applied", "Apply", true},
    {"applied→rolled_back", "applied", "rolled_back", "Rollback", true},
    {"pending→applied", "pending", "applied", "Apply", false}, // 必须经 accepted
    {"rejected→accepted", "rejected", "accepted", "Accept", false}, // 终态
    // ...
}
```

#### 12.2.3 partial unique index 并发

```go
// 模拟 2 个 goroutine 并发 accept 同一 suggestion
// 期望:1 个成功,1 个因 unique violation 失败
// 业务层:成功 → 200,失败 → 409 Conflict
```

#### 12.2.4 7d 回滚窗口过期

```go
// 1. 模拟 applied_at = NOW() - 8d(超 7d)
// 2. 调 Rollback → 期望:error "回滚窗口已过(7d)"
// 3. 模拟 applied_at = NOW() - 3d(7d 内)
// 4. 调 Rollback → 期望:成功
```

#### 12.2.5 误修复率计算

```go
// 1. seed 100 applied + 5 rolled_back
// 2. 调 stats(windowDays=7)
// 3. 期望:misfixRate = 0.05
// 4. seed 0 applied
// 5. 期望:misfixRate = 0(非 NaN/Inf,非空)
```

### 12.3 集成测试（PG dev DB）

- 完整 5 端点 E2E 流程:list → accept → apply → rollback
- 事务并发测试（2 个 apply 并发,1 个成功 1 个 409）
- 缓存失效测试:Apply 后 Redis key `reconciliation:health:workstation:{wsID}` 被删除

### 12.4 回归守护

- 复用 `internal/utils/operlog/regression_test.go` 模式(不新增)
- 部分唯一索引 `uniq_fix_suggestion_pending_per_exception` 命名测试(类似 `uniq_recon_asset_type_open` 测试)

---

## 13. 风险与缓解（6 项）

### 13.1 并发 Accept 竞态

**风险**：2 个运维同时点"接受"同一 suggestion,理论上 DB partial unique 兜底,但事务时序可能让 2 个 UPDATE 都成功(同 WHERE fix_status='pending' 谓词)。

**缓解**：
- partial unique index DB 层硬拦截(已锁定 §3.3 Layer 1)
- 事务 + 条件 UPDATE 应用层软拦截(已锁定 §3.3 Layer 2)
- 单元测试覆盖(`§12.2.3`)

### 13.2 R2 workorder_id 冲突

**风险**：R2 已有 `sys_data_reconciliation.workorder_id` 字段(Type B 转工单后填),R5 触发器 `WHERE workorder_id IS NULL` 应排除已转单的 B 类。

**缓解**：
- D-A4 锁定 `workorder_id IS NULL AND resolved_at IS NULL` 谓词
- migration_182 不修改 `sys_data_reconciliation` 表
- 集成测试:seed 1 条 B 类异常 + workorder_id 非空 → 期望 cron 不生成建议

### 13.3 R4 cache miss 窗口

**风险**：Apply 失败回滚 ops_asset.user_id 后,工位健康度缓存可能 5min 内仍返回旧值(参 `reconciliationHealthCacheTTL=5*time.Minute`)。

**缓解**：
- §8 锁定 `invalidate_workstation_health(wsID)` 强制失效
- 单元测试覆盖(`§12.3`)

### 13.4 GORM AutoMigrate + MV 刷新冲突

**风险**：参 MEMORY.md "GORM AutoMigrate 被 PG 物化视图阻塞" — 若新建 `sys_reconciliation_fix_suggestion` 表被某 MV 引用,AutoMigrate 启动会 FATA。

**缓解**：
- R5 建议表**不**被任何现有 MV 引用(无 JOIN)
- migration_182 仅 AutoMigrate + 索引 + CHECK,不创建 MV
- database.go AutoMigrate 注册按"叶子表先,被引用表后"顺序(参 database.go 现有顺序)
- 推荐:`SysReconciliationFixSuggestion` 注册在 `SysDataReconciliation` 之后(若未来 R6 加入 MV 引用)

### 13.5 7d 静默期与 Apply 状态交互

**风险**：Apply 后 `sys_data_reconciliation` 仍是 open 状态(不 resolved),DetectLayer3 下次 cron 周期会再次检出 Type B → 再次生成新轮建议。

**缓解**：
- D-C4 锁定:applied 后自动进入 7d 静默期(参 `reconciliation_detection.go:254-259` 静默期逻辑)
- 静默期谓词检查的是 `last_resolved_at`,**但** Apply 不写 `resolved_at`!
- **R5 必补**:Apply 时需 `UPDATE sys_data_reconciliation.resolved_at = NOW() WHERE id = ?`(类似 R2 ResolveException),让 MV 的 `last_resolved_at` 自然捕获
- 或者 cron 触发器加 `WHERE NOT EXISTS (SELECT 1 FROM sys_reconciliation_fix_suggestion WHERE exception_id = sys_data_reconciliation.id AND fix_status IN ('applied', 'rolled_back', 'rejected'))`
- **推荐方案**:Apply 时同步 resolve 异常 — 与 R2 行为一致(运维"修复"也算"解决")。Planner 需在 §3.4 Apply 流程中加 `tx.Model(&exception).Update("resolved_at", now)`。

### 13.6 物理链路 user_id 推导可空

**风险**：R1 RECON-02 物理链路 `port_mac → info_point → workstation → user_id` 在某些资产(无 MAC 采集)下 `physical_user_id` 为 NULL → 无法生成建议。

**缓解**：
- D-A1 锁定:仅当 `physical_user_id IS NOT NULL` 才生成建议
- 触发器 cron 加 `INNER JOIN reconciliation_normalized ON ... WHERE rn.physical_user_id IS NOT NULL`
- Type B 异常在 `physical_user_id` 为空时已归 D 或 E 类(参 `reconciliation_detection.go:148-152`),不会被 R5 触达

---

## 14. 落地路径建议(给 planner)

按以下顺序生成 plan(每个 plan 可独立可观测 + atomic commit):

### Plan 46-01 — 修复建议生成器 + 人工确认 UI

| 任务 | 文件 | 估时 |
|------|------|------|
| migration_182/183/184 | `internal/core/db/migrations/` | 30min |
| database.go AutoMigrate 注册 | `internal/core/db/database.go` | 5min |
| SysReconciliationFixSuggestion model | `internal/models/fix_suggestion.go` | 15min |
| FixSuggestionService 接口 | `internal/services/asset/fix_suggestion_service.go` | 30min |
| FixSuggestionServiceImpl + Apply/Rollback 事务 | 同上 | 60min |
| FixSuggestionGenerator + cron 集成 | `internal/services/asset/fix_suggestion_generator.go` | 30min |
| Handler + Router | `internal/api/v1/asset/fix_suggestion_*.go` | 30min |
| 前端页面 + 详情 Drawer | `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/` | 90min |
| queryKeys + fixSuggestionApi | `src/lib/queryKeys.ts` + `src/lib/assetApi.ts` | 15min |
| 单元测试(5 类) | `internal/services/asset/fix_suggestion_*_test.go` | 60min |
| 集成测试(可选) | `internal/services/asset/integration_test.go` | 30min |
| **小计** | | **~6h** |

### Plan 46-02 — 一键回滚机制 + 误修复监控

| 任务 | 文件 | 估时 |
|------|------|------|
| FixSuggestionStats 端点(SQL + 7d 滑动窗口) | `internal/services/asset/fix_suggestion_stats.go` | 45min |
| StatsService 集成 | `internal/api/v1/asset/fix_suggestion_handler.go` | 15min |
| 误修复率 SysNotice 写入 | `internal/services/asset/fix_suggestion_monitor.go` | 30min |
| 7d 倒计时 UI + 拒绝/回滚 Modal | `src/pages/.../fix-suggestion/components/` | 45min |
| 5 KPI 卡片渲染 | `src/pages/.../fix-suggestion/index.tsx` | 30min |
| 端到端 UAT 测试 | 手动 + 自动化 | 60min |
| operlog 审计链验证 | 手动查 sys_oper_log | 15min |
| **小计** | | **~4h** |

**总估时**：~10h(2 plans,推荐分 wave 执行)

### 14.1 plan 优先级

- **Plan 46-01 是 Plan 46-02 的前置**(stats 端点依赖 service 层方法)
- 2 plans 建议**串行执行**(因依赖关系)
- 后续 v1.17 milestone close 在 2 plans 完成后

### 14.2 与 R1-R4 兼容性

- **R1 既有数据**:`sys_data_reconciliation` 既有 Type B 异常行不会被 R5 直接修改(observe-only 升级到 R5 局部写)
- **R2 workorder_id 字段**:R5 触发器 `WHERE workorder_id IS NULL` 互不干扰
- **R3 exception_rule_id**:R5 触发器不读此字段(可选后续 v1.18 R5+ 加 `WHERE exception_rule_id IS NULL`)
- **R4 缓存**:R5 复用 `invalidate_workstation_health` helper(D-C4 锁定)
- **R5 pre-fix user_id 落库**:R5 写 `ops_asset.user_id` 是 v1.17 首次破例(observe-only 局部升级)

---

## 15. 未决项与 planner 建议

### 15.1 planner 必决

1. **Accept/Apply 是一步式还是两步式**？(CONTEXT 暗示两步式,§5.2.3 详述)
2. **Apply 时是否同步 resolve 异常**？(建议是 — 参 §13.5)
3. **migration 编号 182/183/184** vs 195/196/197(顺延)?
4. **独立 cron @every 5m** vs 复用 DetectLayer3 cron(在 DetectLayer3 末尾加)?(CONTEXT 暗示独立,§4 推荐)
5. **回滚 7d 后是否提供 admin 强制回滚端点**?(CONTEXT 未锁定,§11.3 提示可选)
6. **拒绝是否必填 rejectionReason** ?(CONTEXT 暗示必填 ≥10 字符,§5.2.4 锁定)

### 15.2 planner 可决(planner 自决标签)

- 拒绝 UI 形态(弹窗 vs 抽屉)
- 修改建议功能是否纳入 R5(plan 建议推 v1.18)
- 是否纳入 R3 baseline compare
- migration 拆分粒度(3 个 vs 1 个)
- 单元测试是否含集成测试

---

## 16. 关键交叉引用

### 16.1 R1-R4 既有锁定(必须在 plan 中继承)

| 锁定 | 出处 | R5 引用 |
|------|------|----------|
| partial unique 命名 `uni_*_*` | MEMORY `xingran-gorm-sql-constraint-naming-conflict` | §1.1 §2.2 |
| migration 显式调用 + AutoMigrate | MEMORY `xingran-migrations-no-sql-autoloader` | §9.5 |
| sys_config 读取 | `internal/services/system/config_service.go:29` | §10.3 |
| 缓存 key 模板 | `internal/services/asset/cache_keys.go` | §8.1 |
| 7d 静默期逻辑 | `internal/services/asset/reconciliation_detection.go:254-259` | §13.5 |
| 24h 节流逻辑 | `internal/services/asset/reconciliation_detection.go:264-278` | §4.1 |
| operlog 25 OperType 常量 | `internal/utils/operlog/operlog.go:34-66` | §7.1 |
| 11 敏感关键词脱敏 | `internal/utils/operlog/operlog.go:124-169` | §7.4 |
| ReconciliationHandler 模块名常量模式 | `internal/api/v1/asset/reconciliation_handler.go:23-29` | §7.2 |
| 异常列表 queryKeys | `src/lib/queryKeys.ts:62-67` | §6.3 |
| ListExceptions BaseListRequest 模式 | `internal/services/asset/reconciliation_service.go:55-73` | §1.4 |
| Statistics 6 端点 COUNT 模式 | `internal/services/asset/reconciliation_statistics.go:172-237` | §1.3 |

### 16.2 v1.17 milestone close 路径

R5 完成后 v1.17 收官(ROADMAP SC 7)。建议在 plan-execute-phase 完成后,运行 `gsd-complete-milestone` 触发:
- STATE.md 更新(Phase 46 标 ✅ Complete)
- ROADMAP.md v1.17 行:🚧 → ✅
- REQUIREMENTS.md 不变(R5 沿用 RECON/AUDIT/INFRA,无新增)
- 触发 `gsd-audit-milestone` 验证

---

## 17. 总结

R5 是 v1.17 资产对账 milestone 的**收官 phase**,首次破例允许对账模块修复回写业务表(`ops_asset.user_id`)。**风险面可控**:
- **修复字段**最小化(仅 user_id)
- **修复源**单一(物理链路)
- **必须经人工**接受
- **可一键回滚** 7d
- **误修复率 < 1%** 监控 + 告警
- **完整 operlog** 审计链

**研究文档覆盖 13 个主题**:
1. R1-R4 复用模式(4 类) — 锚定行号 + 模板
2. 数据模型 — 24 字段 + 5 索引 + 2 CHECK
3. 状态机 — 6 状态 + 36 转移表 + 3 层并发防御
4. 触发器 — 3 方案对比 + 终选 cron
5. API 端点 — 5+1 完整契约
6. 前端 — 独立页 + Drawer + 5 KPI
7. operlog — 4 端点 OperType 选择
8. 缓存失效 — 顺序 + JOIN 反查
9. Migration — 3 文件 + database.go 注册
10. 误修复监控 — SQL + 7d 滑动窗口
11. 回滚 — 事务 + 7d 窗口 + 强校验
12. 单元测试 — 5 类
13. 风险 — 6 项 + 缓解

**下一步**:由 `gsd-pattern-mapper` 在 plan-phase 之前映射代码模式(`/gsd-plan-phase`),然后生成具体 plan。

---

## RESEARCH COMPLETE

**研究文件路径**: `D:\CODE\ClaudeCode\xingran-go-backend\.claude\worktrees\phase47-discuss\.planning\phases\46-r5\46-RESEARCH.md`
**总字数**: ~14,500 字
**章节数**: 17
**计划路径**: Plan 46-01(建议生成 + UI, ~6h) + Plan 46-02(回滚 + 监控, ~4h)
**依赖关系**: 46-01 → 46-02 串行
**v1.17 milestone close**: 在 2 plans 完成后
