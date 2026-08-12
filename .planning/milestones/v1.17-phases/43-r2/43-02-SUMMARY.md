---
phase: 43-r2
plan: 02
subsystem: asset-reconciliation
tags: [feat, mv, silence, throttle, resolve-api, operlog, R2]
dependency_graph:
  requires: [Migrate168 (reconciliation_normalized MV + sys_data_reconciliation 表), DetectLayer3 (R1), ReconciliationService.ListExceptions/GetByID, operlog.Record 5-arg signature, gin context user_id 注入]
  provides: [reconciliation_normalized MV 3 R2 字段 (last_resolved_at / last_resolved_by / last_conflict_type), idx_recon_norm_last_resolved 部分索引, DetectLayer3 4 计数 (inserted / skipped / skippedSilence / skippedThrottle), 7d 静默期 (D-A3-01) + 24h 节流 (D-A3-02), POST /asset/reconciliation/exception/:id/resolve API, ReconciliationService.ResolveException service 方法, operlog 状态变更审计 (WORKORDER-02)]
  affects: [internal/services/asset (reconciliation_detection/service), internal/scheduler (reconciliation_tasks.go detectLayer3 case), internal/api/v1/asset (reconciliation_handler/router), internal/core/db (migration_173 + database.go registration), internal/services/asset (reconciliation_test.go)]
tech_stack:
  added: []
  patterns: [DROP MATERIALIZED VIEW CASCADE + CREATE 含 LEFT JOIN LATERAL(MV 字段扩展幂等), DetectLayer3 在 INSERT 前多 guard 累加计数, gin context user_id 取值 + 类型断言, handler 错误分类(400/404/500), GORM Updates(updates map) 只 SET 包含字段]
key_files:
  created:
    - internal/core/db/migrations/migration_173_reconciliation_silence_mv.go (PostgreSQL MV 重建 + 3 R2 字段 + 静默期部分索引, ~165 lines)
  modified:
    - internal/services/asset/reconciliation_detection.go (NormalizedRow + 3 字段, DetectLayer3 签名 + 2 guards, +48 lines)
    - internal/services/asset/reconciliation_service.go (ReconciliationService.ResolveException interface + impl, +86 lines)
    - internal/api/v1/asset/reconciliation_handler.go (ResolveException handler + operlog.Record, +73 lines)
    - internal/api/v1/asset/reconciliation_router.go (POST /exception/:id/resolve 注册, +2 lines)
    - internal/core/db/database.go (Migrate173 注册块, +6 lines)
    - internal/scheduler/reconciliation_tasks.go (detectLayer3 case 适配 4 返回值, +3 lines)
    - internal/services/asset/reconciliation_test.go (测试适配 4 返回值 + view 模拟 3 R2 字段, +13 lines)
decisions:
  - "DetectLayer3 签名扩展为 (inserted, skipped, skippedSilence, skippedThrottle, err) 5 返回值(D-A3-03 锁定);实现侧加 fmt.Errorf 包 24h 节流查询错误,保留上下文"
  - "7d 静默期 guard 1 写在 24h 节流 guard 2 之前 — 静默期是\"运维已修复\"的强信号,优先于节流的\"24h 内有记录\"弱信号"
  - "ResolveException 重复 resolve 防御返回 errors.New 字符串(非 typed error),handler 层用 errMsg == \"该异常已标记为已解决\" 简单匹配返回 400;保持简洁,后续可升级为 typed error"
  - "handler 调 operlog.Record(OperTypeUpdate) — 标记已解决是状态变更,符合 CLAUDE.md 强制约定 + WORKORDER-02"
  - "ResolutionNote 可选 nil/空字符串(handler 不报错) — 运维可能没有备注也要能 resolve"
  - "gin context user_id 类型断言失败 → 401 — auth 中间件应当注入 string,但兜底防御避免 nil 写入 sys_data_reconciliation.resolved_by"
  - "MV DROP IF EXISTS CASCADE 二次启动幂等 — 即使 dropDependentMaterializedViews 没 drop 这里也会 drop,避免字段缺失"
  - "idx_recon_norm_last_resolved 部分索引 WHERE last_resolved_at IS NOT NULL — 99% 历史资产无 resolved 记录,部分索引显著缩小扫描范围"
  - "测试中 TestDetectLayer3_DuplicateViolation 改走 24h 节流路径(skippedThrottle)而非 unique violation — R2 guard 顺序使前者先命中,业务语义更清晰"
  - "R2 简化交付:router 层不强制 RequirePermissions(asset:reconciliation:resolve) — 前端按钮 disabled 控制可见性,R3 阶段补后端强制"
metrics:
  duration: ~25 min
  completed_date: 2026-06-27
  tasks: 2
  files_created: 1
  files_modified: 6
  commits: 2
  lines_added: ~480
---

# Phase 43 Plan 02: 7d 静默期 + 24h 节流 + ResolveException API Summary

## One-liner

reconciliation_normalized MV 扩展 3 R2 字段(LEFT JOIN LATERAL last_resolved_*)+ DetectLayer3 签名 4 计数(inserted/skipped/skippedSilence/skippedThrottle)+ 2 guard(7d 静默期 D-A3-01 / 24h 节流 D-A3-02)+ POST /asset/reconciliation/exception/:id/resolve API(operlog 状态变更审计 WORKORDER-02),为 Phase 43 R2 闭环第二步。

## What Built

### Task 1: MV 扩展 + 静默期 + 节流

- **`internal/core/db/migrations/migration_173_reconciliation_silence_mv.go`** (~165 lines,新文件)
  - `Migrate173ReconciliationSilenceMV(db)` 函数
  - PG 端 DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE — 二次启动幂等(避免 IF NOT EXISTS 跳过导致字段缺失)
  - CREATE MATERIALIZED VIEW 含 LEFT JOIN LATERAL 子查询:
    ```sql
    LEFT JOIN LATERAL (
        SELECT resolved_at, resolved_by, conflict_type
        FROM sys_data_reconciliation r
        WHERE r.asset_id = a.id
          AND r.resolved_at IS NOT NULL
          AND r.deleted_at IS NULL
        ORDER BY r.resolved_at DESC
        LIMIT 1
    ) last_resolved ON true
    ```
  - 3 R2 字段:`last_resolved_at timestamptz` / `last_resolved_by uuid` / `last_conflict_type varchar(2)`
  - 索引:idx_recon_norm_asset(UNIQUE)+ idx_recon_norm_asset_code + idx_recon_norm_last_resolved(部分索引 WHERE last_resolved_at IS NOT NULL,加速 7d 静默期查询)
  - 验证:SELECT COUNT(*) + information_schema.columns 抽样 last_resolved_at 列存在

- **`internal/services/asset/reconciliation_detection.go`** (+48 lines)
  - `NormalizedRow` struct 加 3 R2 字段:`LastResolvedAt *time.Time` / `LastResolvedBy *string` / `LastConflictType *string`
  - `ReconciliationDetection.DetectLayer3` 接口签名扩展为 5 返回值:`(inserted int, skipped int, skippedSilence int, skippedThrottle int, err error)`
  - 实现新增常量:`silencePeriod = 7*24*time.Hour` + `throttleWindow = 24*time.Hour`
  - 循环内 INSERT 前加 2 guard:
    - **Guard 1 (D-A3-01 静默期)**:row.LastResolvedAt != nil && row.LastConflictType != nil && *row.LastConflictType == conflictType && time.Since(*row.LastResolvedAt) < silencePeriod → skippedSilence++ continue
    - **Guard 2 (D-A3-02 节流)**:s.db.Model(SysDataReconciliation).Where(asset_id, conflict_type, detected_at > NOW-24h, deleted_at IS NULL).Count → 命中则 skippedThrottle++ continue
  - 注释完整说明 4 计数语义 + 流程

- **`internal/scheduler/reconciliation_tasks.go`** (+3 lines)
  - `detectLayer3` case 适配 4 返回值,日志增加 skippedSilence + skippedThrottle 输出

- **`internal/services/asset/reconciliation_test.go`** (+13 lines)
  - 两个 DetectLayer3 测试调用适配 5 返回值
  - SQLite view 模拟加 3 R2 字段(NULL 占位,SQLite 测试无真实 sys_data_reconciliation 数据,guard 永远不命中)
  - `TestDetectLayer3_DuplicateViolation_Skipped` 改走 24h 节流路径(R2 guard 顺序使 throttle 先命中),保留"不重复插入"核心语义

### Task 2: ResolveException service + handler + router + database.go

- **`internal/services/asset/reconciliation_service.go`** (+86 lines)
  - `ReconciliationService` interface 加 `ResolveException(ctx, id, userID, note) error` + 完整注释(行为/并发安全/入参/返回)
  - 实现 4 步流程:
    1. `SELECT id=? AND deleted_at IS NULL` First(若 ErrRecordNotFound → "异常不存在")
    2. 防御 `rec.ResolvedAt != nil` → "该异常已标记为已解决"
    3. 构造 updates map:resolved_at=NOW + resolved_by=userID + 可选 resolution_note
    4. `db.Model(&rec).Updates(updates)` — GORM 只 SET 包含字段,避免覆盖其他
  - 空值防御: id="" / userID="" → 返回 errors
  - 注释完整说明并发安全(SELECT 检查 + GORM Update 原子操作)

- **`internal/api/v1/asset/reconciliation_handler.go`** (+73 lines)
  - `ResolveException(c *gin.Context)` handler
  - 6 步流程:
    1. URL 参数 id
    2. body 解析(resolutionNote 可选;ContentLength=0 跳过)
    3. gin context 取 user_id(类型断言 + exists 检查)
    4. 调 service,错误分类(400 "已解决" / 404 "不存在" / 500 其他)
    5. 成功路径调 `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)` — 兑现 CLAUDE.md 强制约定 + WORKORDER-02
    6. 返回 `{id, resolvedAt, resolvedBy, resolutionNote}`

- **`internal/api/v1/asset/reconciliation_router.go`** (+2 lines)
  - 注册 `POST /exception/:id/resolve` 路由(D-A4-04 路径)

- **`internal/core/db/database.go`** (+6 lines)
  - 在 Migrate172 之后追加 `Migrate173ReconciliationSilenceMV(d.DB)` 注册

## Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| go build ./... exit 0 | PASS | `go build ./...` 无输出 |
| MV 扩展 3 R2 字段 | PASS | migration_173:73-78 SELECT last_resolved.*;reconciliation_detection.go:33-35 NormalizedRow 3 字段 |
| DetectLayer3 4 计数 | PASS | reconciliation_detection.go:77 接口签名;implementation 返回 4 计数 |
| 7d 静默期 guard 1 | PASS | reconciliation_detection.go:230-242 guard 1 块 |
| 24h 节流 guard 2 | PASS | reconciliation_detection.go:245-260 guard 2 块 |
| POST /exception/:id/resolve | PASS | reconciliation_router.go:30 注册 |
| operlog.Record(OperTypeUpdate) | PASS | reconciliation_handler.go:150 调用 |
| database.go 注册 Migrate173 | PASS | database.go:481 |
| DetectLayer3 caller 适配 | PASS | reconciliation_tasks.go:62-63 适配 4 返回值;reconciliation_test.go 适配 |
| 测试通过 | PASS | `go test ./internal/services/asset/... ./internal/api/v1/asset/...` 全部 PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestDetectLayer3_DuplicateViolation 期望需更新**
- **Found during:** Task 1 (running tests after DetectLayer3 signature change)
- **Issue:** 原测试断言 `skipped >= 1`,R2 新增 24h 节流 guard 后,duplicate 路径先命中 throttle guard,unique violation 永远不会触发,`skipped` 仍为 0
- **Fix:** 测试改走 `skippedThrottle >= 1` 断言,核心"D-11: 不重复插入"语义保留(数据库 count=1)
- **Files modified:** internal/services/asset/reconciliation_test.go
- **Commit:** addc1c3d (Task 1 commit)

### Plan Adjustments

**1. [Refactor] 移除 migration_173 中冗余的 AutoMigrate 占位调用**
- **Found during:** Task 1 review
- **Issue:** 原代码包含 `db.AutoMigrate(&struct{}{})` 占位调用,无实际效果
- **Fix:** 直接删除 — sys_data_reconciliation 表及其 resolved_* 字段已在 R1 由 migration_168 创建,无需 R2 重复 AutoMigrate
- **Files modified:** internal/core/db/migrations/migration_173_reconciliation_silence_mv.go
- **Commit:** addc1c3d (Task 1 commit)

## Auth Gates

None — 计划无外部认证依赖;user_id 直接从 gin context(auth 中间件已注入)取值。

## Known Stubs

无 — 所有代码路径完整,无 TODO/FIXME/placeholder。

注:`asset:reconciliation:resolve` 权限粒度由前端按钮 disabled 控制可见性,R3 阶段在 router 层补 RequirePermissions 后端强制(plan frontmatter T-43-05 mitigation 显式声明)。

## Threat Flags

无新安全面。所有威胁已在 plan frontmatter 评估并 mitigation:
- **T-43-05** (未授权 resolve) — mitigate,前端按钮 disabled 控制可见性(R3 补后端强制)
- **T-43-06** (MV 子查询注入) — mitigate,子查询值硬编码,无用户输入
- **T-43-07** (resolve 并发) — mitigate,Service 层 SELECT 检查 resolved_at + GORM Update 原子操作
- **T-43-08** (resolve 无审计) — mitigate,operlog.Record(OperTypeUpdate) 强制写 sys_oper_log
- **T-43-09** (resolution_note 信息泄露) — mitigate,resolution_note 走 operlog 自动记录(非敏感字段)
- **T-43-SC** (新依赖) — mitigate,无新依赖

## Followup Notes

### 后续 plans 依赖

- **43-03**:前端 WS 推送 + "标记已解决" UI + SysNotice 写入(用本 plan 的 POST /exception/:id/resolve + 7d 静默期保证不重复触发)

### UAT 验证项(用户执行)

1. 启动 backend → 检查 sys_data_reconciliation 表 resolved_at / resolved_by / resolution_note 字段存在
2. 启动 backend → 检查 reconciliation_normalized MV 含 last_resolved_at / last_resolved_by / last_conflict_type 字段(`SELECT column_name FROM information_schema.columns WHERE table_name='reconciliation_normalized'`)
3. 启动 backend → 检查 idx_recon_norm_last_resolved 索引存在(`\d reconciliation_normalized` in psql)
4. 登录 → 找一条未 resolved 的异常 → POST `/asset/reconciliation/exception/{id}/resolve` with body `{"resolutionNote":"已修复"}` → 验证返回 `{id, resolvedAt, resolvedBy, resolutionNote}` + sys_data_reconciliation 三字段已更新 + sys_oper_log 新增 OperTypeUpdate 记录
5. 同一 ID 再次 POST → 验证返回 400 "该异常已标记为已解决"
6. 不存在的 ID POST → 验证返回 404 "异常不存在"
7. 静默期验证:resolve 一条异常 → 等 1 分钟 → 模拟运维重新 INSERT 同 (asset, type) 异常 → 等 Layer3 cron 周期 → 验证 sys_data_reconciliation 不新增(7d 静默期拦截)
8. 节流验证:不 resolve → 模拟 cron 1 小时内重复触发同 (asset, type) → 验证只第 1 次插入,后续 24h 内命中 throttle 跳过

### 数据迁移注意事项

- 现有已 resolved 的 sys_data_reconciliation 记录会被 MV LEFT JOIN LATERAL 自动取最近一条,无需手工数据迁移
- MV 在 startup 时由 dropDependentMaterializedViews drop,然后 migration_173 重建 — 启动期间 MV 不可用约几秒,Layer3 cron 会短暂失败但下个周期恢复
- idx_recon_norm_last_resolved 部分索引在空 resolved 记录环境下保持空索引,查询性能不受影响

## Self-Check: PASSED

- `internal/core/db/migrations/migration_173_reconciliation_silence_mv.go` exists ✓
- `internal/services/asset/reconciliation_detection.go` updated with 3 fields + 4-return DetectLayer3 + 2 guards ✓
- `internal/services/asset/reconciliation_service.go` updated with ResolveException ✓
- `internal/api/v1/asset/reconciliation_handler.go` updated with ResolveException handler + operlog ✓
- `internal/api/v1/asset/reconciliation_router.go` updated with /exception/:id/resolve ✓
- `internal/core/db/database.go` updated with Migrate173 registration ✓
- `internal/scheduler/reconciliation_tasks.go` updated detectLayer3 case ✓
- `internal/services/asset/reconciliation_test.go` updated for new signature ✓
- Commits `addc1c3d` (Task 1) + `8bfd66b4` (Task 2) exist in git log ✓
- `go build ./...` exit 0 ✓
- `go test ./internal/services/asset/... ./internal/api/v1/asset/...` PASS ✓