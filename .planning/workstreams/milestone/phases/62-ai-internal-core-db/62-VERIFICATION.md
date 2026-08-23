---
phase: 62-ai-internal-core-db
verified: 2026-08-15T00:00:00Z
status: human_needed
score: 14/14 must-haves verified
overrides_applied: 0
human_verification:
  - test: "在真实 PostgreSQL(含 R1/R2 旧结构 reconciliation_normalized MV)上执行启动迁移,模拟就地升级"
    expected: "Migrate176 检测到缺 R5 标记列,WARN 输出缺失列清单并走 DROP+CREATE 慢路径,升级后 MV 含全部 18 列"
    why_human: "PG 路径测试在无 XINGRAN_PG_TEST_DSN 时 skip,sqlite 方言守卫短路无法覆盖 information_schema 真实探测;需要带旧 MV 的 PG 实例"
  - test: "双实例并发启动(或手动 psql 持有 pg_advisory_lock(hashtext('xingran-migrations')) 后启动后端)"
    expected: "后拿到锁的实例输出 '[advisory-lock] 另一实例正在执行启动迁移' WARN 并跳过迁移块,先持锁实例正常完成迁移,进程退出后锁正确释放"
    why_human: "多进程并发时序行为,grep 与单进程测试无法验证会话级锁的获取/释放/跳过全链路"
  - test: "全新空库首次启动(不设 SYS_ADMIN_BOOTSTRAP_PASSWORD),观察启动日志"
    expected: "日志出现 '[安全告警] 管理员账户已使用出厂默认密码 admin123' 及修改指引 WARN;设置环境变量后启动日志提示密码已从环境变量读取且无默认密码告警"
    why_human: "需要真实 PG + 完整启动流程,日志输出与种子写入行为需人工确认"
---

# Phase 62: internal/core/db 跨AI评审清零 Verification Report

**Phase Goal:** internal/core/db 的跨AI评审(codex + opencode)发现全部清零——共识 C1-C7 修复 + 单方 HIGH + 锁定 MEDIUM 一并落地,所有迁移保持幂等
**Verified:** 2026-08-15T00:00:00Z
**Status:** human_needed(14/14 自动核验通过,3 项 PG 运行时行为需人工 UAT)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

来源:5 个 PLAN frontmatter must_haves(与 ROADMAP success criteria 一致,无缩水);14 个评审项 ID 全部被 plan 认领且无孤儿。

| #  | Truth | Status | Evidence |
| -- | ----- | ------ | -------- |
| 1 | C1: MV 存在但缺 R5 标记列时走 DROP+CREATE 慢路径 | ✓ VERIFIED | migration_176:126-202 — `information_schema.columns` 查询 4 个 R5 标记列(asset_username/physical_user_id/last_resolved_at/mv_refreshed_at),任意缺失 `schemaOK=false` → WARN 缺失列清单 → 回退慢路径 |
| 2 | C1: ops_asset_physical 回填双路径均执行 | ✓ VERIFIED | migration_176:185(快路径 `backfillOpsAssetPhysical(db)`)+ :314(慢路径) |
| 3 | C1: Type E 批量 resolved 带 EXISTS 门控 + 每次 WARN RowsAffected | ✓ VERIFIED | migration_176:362-375(`SELECT EXISTS (SELECT 1 FROM ops_asset_physical LIMIT 1)` 前置门控,无数据跳过)+ :389(`applogger.Warnf("...受影响行数: %d", res.RowsAffected)`) |
| 4 | CDX-M-IDX: idx_sys_user_nickname 部分索引 | ✓ VERIFIED | migration_175:162 `CREATE INDEX IF NOT EXISTS idx_sys_user_nickname` |
| 5 | CDX-M-IDX: idx_recon_resolved_asset_time 部分索引 | ✓ VERIFIED | migration_176:95 `CREATE INDEX IF NOT EXISTS idx_recon_resolved_asset_time`(且注释说明对 sys_data_reconciliation 未建表场景非致命自愈) |
| 6 | 幂等: 新增 DDL 全部 IF NOT EXISTS / CREATE OR REPLACE,sqlite 双调返回 nil | ✓ VERIFIED | 源码全 DDL 幂等;测试 `TestMigrate175_SqliteDoubleInvocation` / `TestMigrate176_SqliteDoubleInvocation` / `TestMigrate176_AllDDLIdempotent` 锁定,go test PASS |
| 7 | C4: SlowThreshold 慢查询 WARN 真实生效 | ✓ VERIFIED | filter_logger.go:136-148 `slowQueryLog`(elapsed >= SlowThreshold 判定)+ :169-173 Trace 成功路径触发 `applogger.Warnf`;测试 TestTrace_SlowQuery/TestTrace_FastQuery |
| 8 | C4: Info/Warn 尊重 MinLevel,默认 Silent 完全静默;LogMode 真实生效 | ✓ VERIFIED | filter_logger.go:77-84 `shouldEmitInfo`/`shouldEmitWarn`(:89/:102 消费)+ :65-69 `LogMode` 返回新 MinLevel 实例;测试 TestInfoWarn_MinLevelSilent/Info + TestLogMode_RealLevelEffect |
| 9 | C6: GrantNewMenuToRolesHavingParent 参数化,无 fmt.Sprintf | ✓ VERIFIED | menu_grant_helpers.go:40-49 — `$1::uuid` / `$2` 占位符 + `db.Exec(sql, newMenuID, parentMenuName)`,文件内无 fmt.Sprintf;既有 4 个 SQL 片段源码断言保留 |
| 10 | C2: SYS_ADMIN_BOOTSTRAP_PASSWORD 覆盖 + 回退 admin123 大声 WARN;Salt 不再是 "default" | ✓ VERIFIED | init_data.go:233-236 env 读取与回退、:251 `Salt: ""`、:268-270 双条 WARN(含修改指引)、:274 env 生效 Infof |
| 11 | C5: 部门种子逐棵子树 check-and-create | ✓ VERIFIED | init_data.go:78-101 `ensureDept`(按 dept_name + parent_id 语义查,存在写回 ID,真实错误上抛)+ createDefaultDept 改用 ensureDept 逐子树调用 |
| 12 | OC-M-MENUSEED: 菜单循环真实 DB 错误返回;按钮父 ID 缺失返回错误 | ✓ VERIFIED | init_data.go:806/:893 `!errors.Is(err, gorm.ErrRecordNotFound)` → return 包装错误;:883-886 `parentMenuID == ""` → return 错误 |
| 13 | CDX-M-USERROLE: createUserRoleRelations 用 db.Create(&models.UserRole{...}) | ✓ VERIFIED | init_data.go:356,无硬编码 `INSERT INTO sys_user_role` 原生 SQL |
| 14 | C3/CDX-H1/CDX-M-UTC/OC-M-SQLITE/C7/CDX-H2(database.go + core.go 六项) | ✓ VERIFIED | C3: database.go:492-497 advisory lock 包裹 175-205 全块(:536-565 专用 sql.Conn + pg_try_advisory_lock(hashtext) + defer 释放,未获锁 WARN 跳过);CDX-H1: :121-123 建库失败上抛 + :705-717 42P04 容忍为 WARN;CDX-M-UTC: :96/:132 两连接路径 `time.Now().UTC()`;OC-M-SQLITE: :42 Host 已设 Port<=0 WARN;C7: :664-675 `Migrator().CreateTable(&models.APIKey{}...)` 派生 + 6 条 CREATE INDEX IF NOT EXISTS 兜底 + 无 `public.` 前缀(全文无 `public.sys_api_keys`);CDX-H2: core.go:280-286 release 模式 SKIP_AUTOMIGRATE=true → return fmt.Errorf 终止启动 |

**Score:** 14/14 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| migration_176_reconciliation_physical_mv.go | schema 校验 + 双路径回填 + 门控 Type E + 支撑索引 | ✓ VERIFIED | 395 行实质实现,含 `information_schema.columns` |
| migration_175_reconciliation_physical_link.go | idx_sys_user_nickname | ✓ VERIFIED | :162 部分索引 |
| migration_176_..._test.go | 源码断言 + sqlite 双调幂等测试 | ✓ VERIFIED | 7 个测试含 TestMigrate176_SqliteDoubleInvocation |
| filter_logger.go | 慢查询 + MinLevel 生效 | ✓ VERIFIED | slowQueryLog/shouldEmitInfo/shouldEmitWarn/LogMode |
| filter_logger_test.go | 行为单元测试 | ✓ VERIFIED | 7 个测试,全 PASS |
| menu_grant_helpers.go | 参数化 SQL | ✓ VERIFIED | `$1::uuid` 占位符 |
| init_data.go | 凭据加固 + 细粒度幂等 + 菜单错误处理 | ✓ VERIFIED | 见 truths 10-13 |
| init_data_test.go | sqlite 种子行为测试 | ✓ VERIFIED | TestCreateDefaultUser 等 |
| database.go | advisory lock + 错误上抛 + UTC + 回退告警 + model 派生 bootstrap | ✓ VERIFIED | 见 truth 14 |
| database_test.go / core_skipautomigrate_test.go | 纯函数与源码断言 | ✓ VERIFIED | 36 个测试函数,全 PASS |
| core.go | SKIP_AUTOMIGRATE 生产 fatal 守卫 | ✓ VERIFIED | :280-286 |

### Key Link Verification

| From | To | Via | Status |
| ---- | -- | --- | ------ |
| migration_176 快路径 | 慢路径 DROP+CREATE | R5 标记列校验失败回退 | ✓ WIRED |
| migration_176 Type E 清理 | ops_asset_physical 门控 | EXISTS 前置条件 | ✓ WIRED |
| filter_logger Trace | applogger.Warnf | slowQueryLog 判定 | ✓ WIRED |
| menu_grant_helpers | db.Exec 参数绑定 | $1::uuid / $2 | ✓ WIRED |
| createDefaultUser | SYS_ADMIN_BOOTSTRAP_PASSWORD | os.Getenv + 回退告警 | ✓ WIRED |
| 菜单种子循环 | ErrRecordNotFound 判定 | errors.Is 区分 | ✓ WIRED |
| AutoMigrate 迁移块 | pg advisory lock | 专用 sql.Conn 会话锁 | ✓ WIRED |
| createPostgresConnection | createDatabaseIfNotExists 错误 | return fmt.Errorf 上抛 | ✓ WIRED |
| BootstrapMissingTables | models.APIKey/APIKeyUsageLog | Migrator().CreateTable | ✓ WIRED |
| core.go initDBAndData | Server.Mode | release 模式返回错误 | ✓ WIRED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Phase 62 包测试全通过 | `go test ./internal/core/...` | internal/core ok 0.315s; internal/core/db ok 3.105s; internal/core/db/migrations ok 0.796s; internal/core/security ok | ✓ PASS |
| 参数化无字符串插值 | grep fmt.Sprintf menu_grant_helpers.go | 0 匹配 | ✓ PASS |
| BootstrapMissingTables 无硬编码表 DDL | grep `CREATE TABLE.*public.sys_api_keys` database.go | 0 匹配;仅 6 条 CREATE INDEX | ✓ PASS |
| 提交存在 | git log internal/core/db internal/core/core.go | 62-01~62-05 各 feat/test 提交均在(94ddc9e..前溯) | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — 本 phase 无 `scripts/*/tests/probe-*.sh` 约定探针;PLAN/SUMMARY 未声明探针,验证载体为 go test(已执行,见上)。

### Requirements Coverage

REQUIREMENTS.md 无 phase 62 映射(phase_req_ids 为 null,评审项即需求集,与 ROADMAP 一致),无孤儿需求。

| Requirement | Source Plan | Status | Evidence |
| ----------- | ----------- | ------ | -------- |
| C1 | 62-01 | ✓ SATISFIED | truth 1-3 |
| CDX-M-IDX | 62-01 | ✓ SATISFIED | truth 4-5 |
| C4 | 62-02 | ✓ SATISFIED | truth 7-8 |
| C6 | 62-02 | ✓ SATISFIED | truth 9 |
| C2 | 62-03 | ✓ SATISFIED | truth 10(首登强制改密为已批准 deferred,见 deferred-items 记录) |
| C5 | 62-03 | ✓ SATISFIED | truth 11(全量事务化为已批准 deferred) |
| OC-M-MENUSEED | 62-03 | ✓ SATISFIED | truth 12 |
| CDX-M-USERROLE | 62-03 | ✓ SATISFIED | truth 13 |
| C3 | 62-04 | ✓ SATISFIED | truth 14 |
| CDX-H1 | 62-04 | ✓ SATISFIED | truth 14 |
| CDX-M-UTC | 62-04 | ✓ SATISFIED | truth 14 |
| OC-M-SQLITE | 62-04 | ✓ SATISFIED | truth 14 |
| C7 | 62-05 | ✓ SATISFIED | truth 14 |
| CDX-H2 | 62-05 | ✓ SATISFIED | truth 14 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| init_data.go | 618 | TODO(未使用的验证码背景图函数) | ℹ️ Info | 预存在(codex LOW / WR-07 死代码范畴),62-REVIEW.md 已记录为 advisory,非目标缺口 |
| 62-REVIEW.md | — | WR-01~07 7 个 Warning(filter_logger 边界、Migrate176 DROP 顺序、adminDSN Sprintf、PG 测试 skip、零断言测试、死代码 helper) | ⚠️ Warning | advisory 性质,已记录在 62-REVIEW.md;不变量核验全部通过,不构成目标缺口 |

无 TBD/FIXME/XXX 债务标记。

### Human Verification Required

### 1. Migrate176 R1/R2→R5 就地升级(真实 PG)

**Test:** 在带旧结构 reconciliation_normalized MV(缺 R5 标记列)的 PostgreSQL 上执行启动迁移
**Expected:** WARN 输出缺失列清单,走 DROP+CREATE 慢路径,升级后 MV 含全部 18 列,对账服务可读新列
**Why human:** PG 路径测试在无 XINGRAN_PG_TEST_DSN 时 skip;sqlite 方言守卫短路,information_schema 真实探测与 MV DDL 无法在单测覆盖

### 2. Advisory lock 多实例并发

**Test:** 双实例并发启动(或 psql 手动持有 `pg_advisory_lock(hashtext('xingran-migrations'))` 后启动后端)
**Expected:** 后到实例输出跳过 WARN,持锁实例正常完成迁移;进程退出后锁释放(下次启动可再获锁)
**Why human:** 多进程并发时序行为,单进程测试无法验证会话级锁获取/释放/跳过全链路

### 3. 全新空库首启 admin 种子告警

**Test:** 空库首次启动,分别在不设 / 设置 SYS_ADMIN_BOOTSTRAP_PASSWORD 两种方式下观察启动日志
**Expected:** 不设时出现默认密码安全告警 WARN + 修改指引;设置时日志确认从环境变量读取且无默认密码告警
**Why human:** 需要真实 PG + 完整启动流程,日志输出与种子写入需人工确认

### Gaps Summary

无目标缺口。14/14 评审项(C1-C7、CDX-H1/H2、CDX-M-UTC/IDX/USERROLE、OC-M-SQLITE/MENUSEED)全部在代码中验证落地,Phase 62 包测试全 PASS,提交记录完整。已批准的 deferred 项(C2 首登强制改密、C5 全量事务化、LOW 项、schema_migrations 版本表)与 62-REVIEW.md 的 7 个 advisory Warning 不计入缺口。状态为 human_needed 仅因 3 项 PG 运行时行为无法在本环境程序化验证。

---

_Verified: 2026-08-15T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
