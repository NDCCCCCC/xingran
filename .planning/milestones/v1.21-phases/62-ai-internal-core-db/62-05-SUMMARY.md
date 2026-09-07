---
phase: 62-ai-internal-core-db
plan: 05
subsystem: database
tags: [postgres, gorm, migrator, bootstrap, skip-automigrate, fail-fast, schema-drift]

# Dependency graph
requires:
  - phase: 62-ai-internal-core-db
    plan: 04
    provides: "advisory lock / UTC NowFunc / sqlite 回退告警 / createDatabase 错误上抛(database.go 基线,本 plan 在其上修改)"
provides:
  - "BootstrapMissingTables 经 gorm.Migrator().CreateTable 从 model 派生建表 (C7 修复,APIKey schema 第三份拷贝消除)"
  - "SKIP_AUTOMIGRATE=true 在 server.mode=release 下 initDBAndData 返回错误终止启动 (CDX-H2 修复)"
affects:
  - "Phase 63+ APIKey model 加列时 bootstrap 旁路自动同步(无需手改 DDL)"
  - "生产部署规范:SKIP_AUTOMIGRATE 仅限 dev,pooler 环境生产部署需走 AutoMigrate 或 dbprovision 工具"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "单一事实源建表:gorm.Migrator().HasTable 判定 + CreateTable(&models.X{}) 派生,与 AutoMigrate(MigrateModelList) 同源防漂移"
    - "CreateTable 在 PrepareStmt:false 连接走 simple protocol,规避 AutoMigrate 80+ DDL 批量路径的 pooler 死锁"
    - "环境变量旁路开关的生产模式 fatal 守卫:mode 判定置于分支最前(Warnf 之前),fail-fast 优于静默半初始化"
    - "源码断言测试(os.ReadFile + strings.Contains/Count)锁定重构不变量(禁硬编码 DDL / 必留索引 / 守卫先于 WARN)"

key-files:
  created:
    - internal/core/core_skipautomigrate_test.go (CDX-H2 源码断言回归守护)
  modified:
    - internal/core/db/database.go (BootstrapMissingTables 重写为 model 派生 + 显式索引兜底)
    - internal/core/db/database_test.go (C7 源码断言追加)
    - internal/core/core.go (initDBAndData SKIP_AUTOMIGRATE release fatal 守卫)

key-decisions:
  - "D-62-05-01 (CreateTable vs 硬编码 DDL):CreateTable 与 AutoMigrate 走同一 MigrateModelList 事实源,model 加列即同步;simple protocol 无 pooler 死锁,原硬编码 DDL 的存在理由被等价替代"
  - "D-62-05-02 (六条显式索引保留):model tag 只覆盖 key_prefix 单列索引,usage log 的 api_key_id/created_at/user_id 查询索引与 keys 表 user_id/deleted_at 索引由 CREATE INDEX IF NOT EXISTS 兜底,幂等可重入"
  - "D-62-05-03 (fatal 而非 WARN):codex 建议二选一,选 fatal 守卫而非让 bootstrap 跑全量迁移 —— bootstrap 的存在理由就是'最小旁路',扩成全量等于第二个 AutoMigrate;封死生产误用 + 保留 dev 旁路是语义最诚实的组合"
  - "D-62-05-04 (public. 前缀消除):显式索引去掉 public. 硬编码,跟随 search_path,与 CreateTable 行为一致"

patterns-established:
  - "model 派生 bootstrap:HasTable → CreateTable → 显式索引 IF NOT EXISTS 兜底,三段式幂等补建"
  - "危险环境变量守卫模式:dev 旁路开关必须在生产模式 fatal,守卫代码先于任何旁路日志"

requirements-completed: [C7, CDX-H2]

# Metrics
duration: 28min
completed: 2026-08-14
---

# Phase 62 Plan 05: BootstrapMissingTables model 派生 + SKIP_AUTOMIGRATE 生产 fatal 守卫 Summary

**APIKey schema 第三份拷贝消除(gorm.Migrator().CreateTable 单一事实源派生)+ SKIP_AUTOMIGRATE 生产模式 fail-fast 封死半初始化旁路**

## Performance

- **Duration:** 28 min (含并发 merge 干扰恢复与全量回归)
- **Commits:** 4 (test→feat→test→feat,标准 TDD 双门)

## What Was Built

### Task 1: C7 — BootstrapMissingTables 改用 gorm.Migrator().CreateTable

重构前后对照:

| 维度 | 重构前(硬编码 DDL) | 重构后(model 派生) |
|------|---------------------|---------------------|
| 表结构来源 | 手写第三份拷贝(model tag + MigrateModelList 之外) | `Migrator().CreateTable(&models.APIKey{})` / `(&models.APIKeyUsageLog{})`,与 AutoMigrate 同源 |
| 漂移风险 | model 加列即漂移,旁路建出错误表结构 | model 加列即同步,单一事实源 |
| schema 前缀 | 硬编码 `public.` | 无前缀,跟随 search_path |
| pooler 兼容 | raw SQL simple protocol | CreateTable 在 PrepareStmt:false 上同为 simple protocol,等价 |
| 幂等性 | CREATE TABLE IF NOT EXISTS | HasTable 判定 + CreateTable,等价 |
| 索引 | 6 条 CREATE INDEX IF NOT EXISTS | 原样保留(model tag 未覆盖的索引面兜底) |

- 建表失败错误信息具体化:`创建 sys_api_keys 失败: %w` / `创建 sys_api_key_usage_logs 失败: %w`(替代笼统 `DDL[%d] failed`)
- 函数签名与导出名不变,core.go 调用点零改动(签名兼容)
- docstring 更新:删除"直接用 raw SQL 补建"描述,补充 C7 单一事实源说明与生产 fatal 指引

### Task 2: CDX-H2 — SKIP_AUTOMIGRATE 生产模式 fatal 守卫

- **触发条件:** `os.Getenv("SKIP_AUTOMIGRATE") == "true"` 且 `c.Config.Server.Mode == "release"`(configs/config.prod.yaml 即 release)
- **行为:** `initDBAndData` 返回错误 → 启动终止(fail-fast)
- **错误文案:** `SKIP_AUTOMIGRATE=true 禁止在生产模式(server.mode=release)使用:旁路补建仅覆盖部分表,会产生半初始化系统;请移除该环境变量后重启` —— 指明风险(半初始化)与处置(移除变量)
- **dev 能力保留:** debug/非 release 模式 WARN 旁路 + BootstrapMissingTables 原样保留(Supabase pooler 应急不丢)
- 守卫位置:分支体最前面(Warnf 之前),先 fatal 再旁路
- 注释块同步:补充"生产模式 fatal;仅保证 api key 两表,不跑 175/176/202-205 迁移,属 dev 应急"

## Verification Results

- `go build ./...` 退出码 0
- `go test ./internal/core/db/ -v` 全部 PASS(含 Plan 01-04 既有测试无回归)
- `go test ./internal/core/ -v` 全部 PASS(含新增源码断言 + 既有 CoreSplit 测试)
- `go test ./...` 全量:40 包 ok;唯一 FAIL 为 `internal/api/v1/auth` TestADLoginWithOUProcessing —— **预存在失败**(该目录自初始提交 ea528c6 后未改动,本 plan diff 不触及 AD 登录代码路径),已记录 deferred-items.md
- 验收 grep 全过:
  - `CREATE TABLE IF NOT EXISTS public.sys_api_keys` 计数 = 0
  - `Migrator().CreateTable(&models.APIKey{})` / `(&models.APIKeyUsageLog{})` 存在
  - `CREATE INDEX IF NOT EXISTS idx_api_keys_* / idx_api_key_logs_*` 计数 = 6
  - `Server.Mode == "release"` + `半初始化` + `BootstrapMissingTables()` 均在 core.go
- **62-04 改动完整性确认:** advisory lock(acquireMigrationAdvisoryLock/releaseMigrationAdvisoryLock)、UTC NowFunc(两处)、sqliteFallbackWarning、createDatabaseIfNotExists 错误上抛 + 42P04 容忍 —— 全部在 database.go 中原样保留(本 plan 仅替换 BootstrapMissingTables 函数体与 docstring)

## TDD Gate Compliance

- Task 1: RED fb3cc70(`test(62-05)` 先行,断言失败确认)→ GREEN b774cc3(`feat(62-05)` 实现后 PASS)
- Task 2: RED 42ce9e4(`test(62-05)` 守卫断言失败确认)→ GREEN 94ddc9e(`feat(62-05)` 实现后 PASS)
- 两 task 均为 test commit 先于 feat commit,gate 序列合规;无 REFACTOR 需要

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] commitlint subject-case 拒绝 Task 2 提交信息**
- **Found during:** Task 2 commit
- **Issue:** subject 以大写 `SKIP_AUTOMIGRATE` 开头触发 `subject-case` 规则(commitlint)
- **Fix:** subject 改写为小写开头 `fatal guard for SKIP_AUTOMIGRATE in release mode`(body 内容不变)
- **Files modified:** 无代码文件,仅提交信息

**2. [Rule 3 - Blocking] 主工作树并发 merge 干扰(用户侧 stash + merge attempt)**
- **Found during:** Task 2 commit 阶段
- **Issue:** 另一会话在主工作树执行 `git stash`(stash@{0}: before merge attempt)+ merge 尝试,期间出现 index.lock 竞争与数百个 UU 冲突文件;工作树状态多次振荡
- **Fix:** 等待 index.lock 释放与 merge abort 后验证:三个已落地 commit(fb3cc70/b774cc3/42ce9e4)完整保留于 HEAD,core.go 守卫代码在磁盘上完整,随后以合规提交信息完成 94ddc9e 提交
- **Files modified:** 无额外文件;未触碰用户 stash(按 destructive git 禁令)

无其他偏离 —— 计划主体按原文执行。

## Known Stubs

无。

## Threat Flags

无新增威胁面。威胁登记表处置闭环:
- T-62-12 (mitigate→已落地): release 模式 SKIP_AUTOMIGRATE fail-fast
- T-62-13 (mitigate→已落地): CreateTable model 派生 + 显式索引兜底

## Phase 62 全 phase Deferred 清单汇总(计划 output 要求)

1. **强制首登改密(C2 部分遗留):** 默认 admin/admin123 + 固定 salt "default" 仍在 init_data.go;Plan 02 已修 salt 随机化(若已含),但首登强制改密机制未做 —— 建议后续 phase 在登录响应加 change_required 标志
2. **LOW 项(双评审):** createSQLiteConnection 配置语义(Host 复用为路径)、SQLite PRAGMA(WAL/busy_timeout/foreign_keys)、cleanupOldConstraints DROP INDEX 吞错、auditConstraintNaming 无 LIMIT、seed 幂等粒度文档化、NULL_STRING_PTR 死代码、RuoYi 品牌字符串、Migrate203 seed Remark 措辞、Migrate204 PG 版本可移植性注释
3. **schema_migrations 版本表(codex suggestion #1):** AutoMigrate 短路 + 迁移状态单一事实源 —— 建议作为独立 phase 立项(涉及新表 + AutoMigrate 流程改造,超本 phase 范围)
4. **预存在测试失败(本次发现,见 Deviations/deferred-items.md):** internal/api/v1/auth TestADLoginWithOUProcessing 两个子用例断言 OU 处理触发标志失败,与 Phase 62 无关,待 /gsd-debug 立项

## Self-Check: PASSED

- 文件存在:internal/core/db/database.go / internal/core/db/database_test.go / internal/core/core.go / internal/core/core_skipautomigrate_test.go 全部 FOUND
- Commits 存在:fb3cc70 / b774cc3 / 42ce9e4 / 94ddc9e 全部在 git log FOUND
- 62-04 改动保留:advisory lock / UTC / sqlite warn / createDatabase 上抛 均在 database.go FOUND
