---
phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
plan: 04
subsystem: infra
tags: [addomain, account-pool, model-cleanup, migration, startup-check, wave-3]

# Dependency graph
requires:
  - phase: 38-03 (Wave 2 删单管理员使用逻辑 + 前端 admin 字段)
    provides: 单管理员 Go 使用代码 = 0；admin 字段前后端零残留
provides:
  - "ADConfig model AdminUsername/AdminPassword tag 放宽（移除 not null）+ @Deprecated（D-02 保留 DB 列）"
  - "migration_164 显式 DROP NOT NULL（兜底 GORM AutoMigrate 不可靠）+ 幂等补迁账号池（D-03）"
  - "启动空池 WARN 校验 checkEmptyAccountPoolOnStartup（不阻断启动，Pitfall 6）"
  - "Phase 38 全部决策 D-01/D-02/D-03/D-04 落地，phase gate 全量 grep 断言通过"

# Tech tracking
tech-stack:
  added:
    - "internal/core/db/migrations/migration_164_phase38_verify_admin_migrated.go（schema 放宽 + 数据补迁）"
  patterns:
    - "启动校验 void 函数（仅 applogger.Warnf，不返回 error 不阻断）+ raw query 避免新导入"
    - "显式 ALTER DROP NOT NULL 兜底 GORM AutoMigrate 既有列约束放宽不可靠（本项目既有踩坑）"
    - "migration 幂等数据补迁：count >0 则 skip（沿用 migration_162 样板）"

key-files:
  created:
    - internal/core/db/migrations/migration_164_phase38_verify_admin_migrated.go
  modified:
    - internal/models/ad_domain.go (AdminUsername/AdminPassword 移除 not null + 每字段 @Deprecated)
    - internal/core/core.go (initAuthFactory 追加启动校验调用 + checkEmptyAccountPoolOnStartup 方法)
    - internal/core/db/database.go (AutoMigrate 注册 Migrate164)

key-decisions:
  - "显式 ALTER DROP NOT NULL（证据驱动添加）：plan Task 2 原仅含数据补迁，但 Phase 38 后新配置不写 admin 字段，若 DB 列仍 NOT NULL 则 INSERT 失败（关键正确性）。GORM AutoMigrate 对既有列 NOT NULL→nullable 放宽不可靠（本项目 memory「gorm-sql-constraint-naming-conflict」印证），故 migration_164 显式 ALTER 兜底。DROP NOT NULL 幂等（已 nullable 时 no-op）。"
  - "启动校验用 raw query（c.GetDB().Table(sys_ad_config)...）而非 models.ADConfig —— core.go 未导入 models 包，raw query 避免新增导入，且与 migration_162 风格一致"
  - "启动校验返回 void 不返回 error（Pitfall 6 / T-38-13）：空池仅 WARN，避免阻塞新环境首次部署；运行时空池由 ErrAllAccountsUnavailable 引导文本兜底（38-02 已实现）"

requirements-completed: [D-01, D-02, D-03]

# Metrics
duration: 20min
completed: 2026-06-23
---

# Phase 38 Plan 04: Wave 3 收尾（model tag + migration_164 + 启动空池校验）Summary

**完成 D-02/D-03 最后一公里：ADConfig model 单管理员字段 tag 放宽 + @Deprecated（DB 列保留）；migration_164 显式 DROP NOT NULL 兜底 + 幂等补迁账号池；启动空池 WARN 校验不阻断启动。Phase 38 全部决策落地。**

## Performance

- **Duration:** ~20 min（内联执行）
- **Tasks:** 2（Task 1 model+启动校验 + Task 2 migration+注册，各独立 commit）
- **Files:** 3 改 + 1 新建

## Accomplishments

- **model tag 放宽（D-02）**：`AdminUsername`/`AdminPassword` 移除 gorm `not null` 约束（`gorm:"size:255"` / `gorm:"size:500"`），每字段加 `@Deprecated Phase 38` 注释。字段本身保留（D-02 不做 DROP COLUMN，DB 列保留兼容）
- **启动空池 WARN 校验（D-03 + Pitfall 6）**：`core.go::initAuthFactory` 末尾追加 `checkEmptyAccountPoolOnStartup`，查启用+同步开启的 ADConfig，账号池 total=0 或 available=0 时 `applogger.Warnf` 引导到账号池 Tab；**返回 void 不阻断启动**（避免阻塞新环境首次部署）
- **migration_164（D-02 + D-03）**：
  - **Schema 放宽**：显式 `ALTER TABLE sys_ad_config ALTER COLUMN admin_username/admin_password DROP NOT NULL`（幂等）——兜底 GORM AutoMigrate 对既有列约束放宽不可靠，保证 Phase 38 后新配置可空值 INSERT
  - **数据补迁**：对启用+同步开启且仍持单管理员凭据的配置，账号池为空则补迁一条（与 migration_162 同款 INSERT）；先 count >0 则 skip（幂等）
- **database.go 注册**：AutoMigrate 调用链（Migrate163 之后）追加 Migrate164

## Task Commits

1. **Task 1: model tag 放宽 + 启动空池 WARN 校验** — `762dafc` (refactor)
2. **Task 2: migration_164 放宽列约束 + 幂等补迁 + 注册** — `3084a04` (refactor)

## Decisions Made

- **显式 ALTER DROP NOT NULL（plan 偏差，证据驱动）**：plan Task 2 仅规划数据补迁。但 Phase 38 Task 2（38-03）移除了 admin 字段的 request binding 后，**新配置不再写 admin 字段**——若 DB 列仍 `NOT NULL`，INSERT 新配置违反约束失败。这是**关键正确性问题**。GORM AutoMigrate 虽在模型列表含 ADConfig（database.go:281），但对既有列 NOT NULL→nullable 的放宽不可靠（本项目 memory「gorm-sql-constraint-naming-conflict」记录 GORM 约束处理踩坑）。故 migration_164 显式 ALTER 兜底，保证 DB 实际允许空值。`DROP NOT NULL` 幂等（已 nullable 时 no-op），SQLite 等不支持该语法的环境记日志跳过不阻断。
- **启动校验 raw query**：core.go 未导入 `internal/models`，用 `c.GetDB().Table("sys_ad_config").Select(...)` + 本地 struct 规避新导入，与 migration_162 风格一致。
- **启动校验 void 返回**：Pitfall 6 / T-38-13 缓解——空池仅 WARN，不阻断启动；运行时空池错误由 38-02 的 `ErrAllAccountsUnavailable` 引导文本兜底。

## Deviations from Plan

### Auto-fixed / Evidence-driven Additions

**1. [证据驱动] migration_164 增加显式 DROP NOT NULL（plan Task 2 原仅数据补迁）**
- **Found during:** Task 2 实现前分析（Phase 38 后新配置空值 INSERT 与 NOT NULL 约束的冲突）
- **Fix:** migration_164 增加 `ALTER ... DROP NOT NULL` schema 放宽步骤（在数据补迁前）
- **Reason:** 关键正确性——无此则新配置创建失败；GORM AutoMigrate 不可靠需显式兜底
- **Committed in:** 3084a04（Task 2）

**2. [文档] model @Deprecated 拆为每字段一个**
- **Found during:** Task 1 验收（`grep @Deprecated` 初版合并注释致 count=1，需 ≥2）
- **Fix:** AdminUsername / AdminPassword 各加独立 @Deprecated 注释（count=2）
- **Committed in:** 762dafc（Task 1）

**Total deviations:** 2（1 证据驱动 schema 补充 + 1 验收对齐）。无 scope creep——DROP NOT NULL 是 Phase 38 正确性的必要部分。

## Issues Encountered

- None（除 38-03 已记录的预先存在 member_ou_dn 测试 schema 漂移，与本 plan 无关）。

## User Setup Required

- **Manual-Only（I-02，D-03 迁移幂等验证）**：生产部署后启动后端一次，确认：(1) migration_164 自动执行，sys_ad_service_accounts 含补迁账号；(2) admin_username/admin_password 列已 nullable；(3) 二次启动日志出现 "skip" 确认幂等。

## Next Phase Readiness

- ✅ D-02 完成：model tag 放宽 + @Deprecated，DB 列保留（不做 DROP COLUMN）
- ✅ D-03 完成：启动空池 WARN 校验（不阻断）+ migration_164 补迁幂等 + 运行时 ErrAllAccountsUnavailable 引导（38-02）
- ✅ Phase 38 全部决策 D-01/D-02/D-03/D-04 落地
- ✅ go build ./... + addomain 测试全绿
- ⚠️ 预先存在测试失败（38-03 已记录）：`TestIntegration_AuthStrategyFactory_SetUserSyncer` member_ou_dn schema 漂移，非 Phase 38 回归

## Known Stubs

None - 纯 model/migration/startup-check 改造，无 UI 占位。

## Threat Flags

None - 本 plan 无新增端点/认证路径。threat_model 全部缓解：
- T-38-13-Startup（启动阻断）：checkEmptyAccountPoolOnStartup void 返回 + 仅 WARN ✓
- T-38-14-Idempotent（重复插入）：count >0 skip + DROP NOT NULL 幂等 ✓
- T-38-15-Leak（日志泄露）：WARN 仅含 config_name/id/total/available 计数，无密码 ✓
- T-38-16-Constraint（约束命名冲突）：仅 ALTER+INSERT 无 DDL 索引/约束 ✓
- T-38-17-DropColumn（误 DROP COLUMN）：D-02 字段保留，仅放宽 not null ✓

---

*Phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing*
*Plan: 04*
*Completed: 2026-06-23*

## Self-Check: PASSED

- ✅ Commit `762dafc` (Task 1) found in git log
- ✅ Commit `3084a04` (Task 2) found in git log
- ✅ `grep -cE "AdminUsername\s+string\s+...not null" ad_domain.go` = 0
- ✅ `grep -cE "AdminPassword\s+string\s+...not null" ad_domain.go` = 0
- ✅ `grep -c "@Deprecated" ad_domain.go` = 2
- ✅ `func checkEmptyAccountPoolOnStartup` 定义存在 + 调用（refs=3）+ 签名 void（无 error）
- ✅ migration_164 文件存在 + `func Migrate164Phase38VerifyAdminMigrated` = 1
- ✅ migration_164 含 `sys_ad_service_accounts`(3) + `deleted_at IS NULL`(1) + skip 日志(3) + DROP NOT NULL(4)
- ✅ `Migrate164Phase38VerifyAdminMigrated` 注册于 database.go（migrate.go 不存在）
- ✅ `go build ./...` exits 0
- ✅ `go test ./internal/services/addomain/` PASS

## Phase 38 整体 grep 硬指标（全波次汇总）

- `decryptPassword(config.AdminPassword)` 变体生产调用 = 0（38-03）
- `NewLDAPClient(config)` 生产调用 = 0（38-02）
- `adminUsername|adminPassword` in adDomainApi.ts/configs/index.tsx/service.go = 0（38-03）
- ADConfig AdminUsername/AdminPassword `not null` = 0（38-04）
