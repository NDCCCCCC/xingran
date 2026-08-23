---
phase: 62-ai-internal-core-db
reviewed: 2026-08-14T16:01:59Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/core/core.go
  - internal/core/core_skipautomigrate_test.go
  - internal/core/db/database.go
  - internal/core/db/database_test.go
  - internal/core/db/filter_logger.go
  - internal/core/db/filter_logger_test.go
  - internal/core/db/init_data.go
  - internal/core/db/init_data_test.go
  - internal/core/db/migrations/menu_grant_helpers.go
  - internal/core/db/migrations/menu_grant_helpers_test.go
  - internal/core/db/migrations/migration_175_reconciliation_physical_link.go
  - internal/core/db/migrations/migration_176_reconciliation_physical_mv.go
  - internal/core/db/migrations/migration_176_reconciliation_physical_mv_test.go
findings:
  critical: 0
  warning: 7
  info: 8
  total: 15
status: issues_found
---

# Phase 62: Code Review Report

**Reviewed:** 2026-08-14T16:01:59Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

对 Phase 62（数据库核心安全加固，跨 AI 评审修复 C1-C7 + 关联项）的 13 个文件做了 standard 深度审查，并对关键不变量逐一验证。

**不变量核验结果（全部通过，未被 62-05 回退）：**

| 不变量 | 结论 |
|--------|------|
| 迁移幂等性（IF NOT EXISTS / ON CONFLICT / CREATE OR REPLACE） | 通过 — 175/176 全部 DDL 幂等，`sys_data_reconciliation` 索引失败非阻断自愈 |
| 62-04 advisory lock + UTC NowFunc + sqlite fallback warn + createDatabaseIfNotExists 错误上抛 | 通过 — `acquireMigrationAdvisoryLock`/`releaseMigrationAdvisoryLock`（database.go:536-581）、两处 `time.Now().UTC()`（:96/:132）、sqlite WARN（:41-44）、错误上抛（:121-123）均在 |
| GrantNewMenuToRolesHavingParent 参数化 SQL | 通过 — `$1::uuid` / `$2` 占位符，无 `fmt.Sprintf`（menu_grant_helpers.go:40-49） |
| FilterLogger SlowThreshold/MinLevel/LogMode 生效 | 部分通过 — `slowQueryLog`/`shouldEmitInfo`/`shouldEmitWarn`/`LogMode` 逻辑正确，但存在两处边界缺陷（见 WR-01/WR-02） |
| admin seed：env 覆盖 + WARN 回退 + 无 Salt:"default" | 通过 — init_data.go:233-278 |
| BootstrapMissingTables 用 Migrator().CreateTable | 通过 — database.go:664-675，无 `public.` 硬编码 DDL |
| SKIP_AUTOMIGRATE release 模式 fatal 守卫 | 通过 — core.go:280-292，守卫在 WARN 之前且 return fmt.Errorf |

无 Critical 级问题。发现 7 个 Warning（2 个 FilterLogger 行为缺口、1 个迁移销毁顺序问题、1 个 DSN 拼接脆弱性、3 个测试有效性问题）和 8 个 Info。

## Structural Findings (fallow)

本次评审未提供 `<structural_findings>` 预扫描载荷，无 fallow 结构化发现可引用。

## Narrative Findings (AI reviewer)

### Warnings

#### WR-01: FilterTypes[LogTypeSQL]=false 的"保留 SQL 日志"分支无任何输出路径，配置项失效

**File:** `internal/core/db/filter_logger.go:157-179`
**Issue:** `Trace()` 在成功且非慢查询路径只有 `if l.config.FilterTypes[LogTypeSQL] { return }` 一个判定，判定不成立时函数直接落尾结束 —— **没有任何输出语句**。配置注释声称 "true = 过滤掉，false = 保留"（:30-31），但把 `FilterTypes[LogTypeSQL]` 设为 `false` 后普通 SQL 日志依然被静默吞掉。该配置开关是彻底的 no-op，会误导排查者以为打开它能看到 SQL。
**Fix:** 在末尾补上实际的输出分支（或删掉该配置项并在注释中声明普通 SQL 永不输出）：

```go
	// 普通 SQL 日志:FilterTypes[LogTypeSQL] 静默(默认 true → 静默)。
	if l.config.FilterTypes[LogTypeSQL] {
		return
	}
	// 保留路径:输出普通 SQL(仅在显式关闭过滤时)
	sql, rows := fc()
	applogger.Debugf("[GORM] %s | 行数: %d | 耗时: %v", sql, rows, time.Since(begin))
```

#### WR-02: Info() 经 applogger.Debugf 输出，LogMode(logger.Info) "生效"被应用日志级别二次压制

**File:** `internal/core/db/filter_logger.go:88-97`
**Issue:** C4 修复声明 "LogMode 调高阈值时可显式打开"（:42）且 `shouldEmitInfo` 判定正确，但 `Info()` 的实际输出走 `applogger.Debugf`。若应用日志级别为 INFO/WARN（生产常态），即使 GORM 层 `LogMode(logger.Info)`，Info 消息仍被应用层吞掉 —— MinLevel 对 Info 路径只"半生效"。同文件 `Warn()` 用 `Warnf`、`Error()` 用 `Errorf`，唯独 Info 降级到 Debug，语义不一致。
**Fix:** `Info()` 输出改用 `applogger.Infof`（与 Warn/Error 的级别对齐），或保留 Debugf 但在 `LogFilterConfig.MinLevel` 注释中明确说明 Info 消息受应用日志级别约束。

#### WR-03: migration_176 先 DROP MV 再校验前置视图，依赖缺失时把已有 MV 删掉后报错退出

**File:** `internal/core/db/migrations/migration_176_reconciliation_physical_mv.go:207-234`
**Issue:** 慢路径执行顺序是：① `DROP MATERIALIZED VIEW IF EXISTS reconciliation_normalized CASCADE`（:209-213）→ ② 探测 `reconciliation_user_lookup` / `reconciliation_physical_chain` 前置视图存在性，缺失则 `return error`（:218-234）。而 database.go 中 Migrate175 的失败是**非阻断的**（仅 Errorf，:499-501）。一旦 175 失败（如临时性 DB 错误）且旧 MV 是 R5 前旧 schema，本迁移会先删掉旧 MV 再以"前置视图缺失"退出 —— 数据库从此没有 reconciliation_normalized，cron `对账-物化视图刷新`（REFRESH CONCURRENTLY）持续 FATA，直到下一次成功启动才自愈。销毁性操作应后置于依赖校验。
**Fix:** 把 :215-234 的前置视图存在性检查整体移到 `dropMV`（:209）之前；缺失时直接 `return error`，不动已有 MV。

#### WR-04: adminDSN 用 Sprintf 拼接 keyword/value 格式，密码含空格/引号/= 时解析失败，与主连接 GetDSN 行为不一致

**File:** `internal/core/db/database.go:115-116`
**Issue:** `createPostgresConnection` 的维护库 DSN 是 `fmt.Sprintf("host=%s port=%d user=%s password=%s ...")` 原样拼接。lib/pq 的 keyword/value DSN 中含空格/单引号的值必须单引号转义，未转义会解析错位。而主连接走 `cfg.GetDSN()`（config.go:528 起，URL 格式 + 百分号编码密码）。结果：含特殊字符的密码**主连接成功、createDatabaseIfNotExists 假失败**，启动被错误地 fail-fast，且报错信息（DSN 解析错）与真实根因（密码格式）相距甚远。
**Fix:** 复用 GetDSN 的编码逻辑生成维护库 DSN，例如：

```go
adminDSN := (&config.DatabaseConfig{
	Host: cfg.Host, Port: cfg.Port, User: cfg.User,
	Password: cfg.Password, DBName: "postgres", SSLMode: cfg.SSLMode,
}).GetDSN()
```

（若 GetDSN 不支持 dbname=postgres 语义可加一个带 dbName 参数的变体。）

#### WR-05: PG 集成测试 helper 无条件 t.Skip，两个 PG 行为测试永远不执行，覆盖信号虚假

**File:** `internal/core/db/migrations/menu_grant_helpers_test.go:147-175`
**Issue:** `postgresAvailable()` 声称"仅当 XINGRAN_PG_TEST_DSN 提供时跑 PG 路径"（:141-145），但 `openPostgresDB`（:149）和全部 seed/assert helper（:155-175）**无条件调用 `t.Skip`**。因此即使设置了 DSN，`TestGrantNewMenuToRolesHavingParent_Idempotent` 与 `_OnlyAffectsParentRoles` 也会在 helper 处跳过 —— C6 参数化 SQL 的幂等性与"仅授权父菜单持有角色"两个核心行为在仓库内**没有任何可执行验证**，仅靠源码 grep 断言兜底。skip 文案还把责任推给"Phase 54 UAT"，形成永久性覆盖缺口。
**Fix:** 要么实现 `openPostgresDB`（`gorm.Open(postgres.Open(os.Getenv("XINGRAN_PG_TEST_DSN")))`）及 seed helpers 并在无 DSN 时 skip，要么删除这两个测试与 `postgresAvailable()`，避免虚假的"有 PG 测试"信号。

#### WR-06: TestTrace_ErrorNoRegression 是零断言的空测试，错误路径回归守护实际不存在

**File:** `internal/core/db/filter_logger_test.go:63-76`
**Issue:** 该测试调用 `l.slowQueryLog(begin, fc)` 后把全部返回值丢弃（`_, _, _ =`），`realErr` 与 `context.Background()` 仅以 `_ =` 赋值占位，函数体内**没有任何断言**，除 panic 外永远不会失败。测试名宣称守护"err != nil 时 Trace 仍输出 ERROR"的既有契约，但该契约没有任何机制被验证 —— C4 改动若回归此路径，此测试不会报警。
**Fix:** 为 `Trace()` 的错误路径建立可观测断言，例如把 `Trace` 的错误输出也提取为可测的私有方法（如 `errorLog(begin, fc, err) (string, bool)`），或注入 applogger 的写接口；至少应断言 `slowQueryLog` 在 `err != nil` 场景不被 `Trace` 顶层调用（当前注释已承认这一点，却未转化为断言）。

#### WR-07: sqliteFallbackWarning 纯函数从未被生产代码调用，NewDatabase 内联重复文案，测试守护的是死代码

**File:** `internal/core/db/database.go:41-44` 与 `:73-82`
**Issue:** `NewDatabase` 在 :41-44 **内联**复刻了与 `sqliteFallbackWarning` 完全相同的条件与格式串，而不是调用该函数。62-04-PLAN.md:137 明确要求 "NewDatabase 中 msg != "" 时 applogger.Warnf(msg)"，但落地变成了双份拷贝。`TestSqliteFallbackWarning`（database_test.go:150-203）测的是生产路径**不经过**的函数 —— 两处文案/条件可静默漂移而测试依然全绿，"提取纯函数便于单测"的设计目标落空。
**Fix:** NewDatabase 改为调用 helper：

```go
	if msg := sqliteFallbackWarning(cfg); msg != "" {
		applogger.Warnf("%s", msg)
	}
```

并删除 :41-44 的内联重复分支。

### Info

#### IN-01: migrationAdvisoryLockKey 常量未被引用，SQL 用内联字面量，注释误导

**File:** `internal/core/db/database.go:701-703`
**Issue:** `const migrationAdvisoryLockKey = "xingran-migrations"` 声明后从未使用；`acquireMigrationAdvisoryLock`（:551）与 `releaseMigrationAdvisoryLock`（:576）各自内联 `'xingran-migrations'` 字面量。修改常量不会影响实际锁键，反而制造"锁键有单一事实源"的假象。
**Fix:** 删除该常量，或改为 `fmt.Sprintf("... hashtext('%s') ...", migrationAdvisoryLockKey)` 在两处共用。

#### IN-02: DefaultLogFilterConfig 是导出 var 且 FilterTypes 为共享 map，浅拷贝会污染全局默认

**File:** `internal/core/db/filter_logger.go:43-50`
**Issue:** 测试与调用方惯用 `cfg := DefaultLogFilterConfig`（struct 拷贝但 **map 引用共享**）。任何一方执行 `cfg.FilterTypes[LogTypeSQL] = false` 都会改写全局默认，影响后续所有 `NewFilterLogger(DefaultLogFilterConfig)`。当前代码未触发，但属埋雷。
**Fix:** `NewFilterLogger` 内对 FilterTypes 做防御性拷贝，或将 DefaultLogFilterConfig 改为返回新值的函数 `DefaultLogFilterConfig()`。

#### IN-03: init_data.go 多处 Count 查询错误未检查

**File:** `internal/core/db/init_data.go:224, 309, 349, 447, 522, 601`
**Issue:** `createDefaultUser` / `createDefaultRole` / `createUserRoleRelations` / `createNetworkDeviceSystemParams` / `createNetworkDeviceScheduledJobs` / `createCaptchaBackgroundSystemParams` 中的 `db.Model(...).Count(&count)` 均丢弃 error。DB 故障时 count 保持 0 走 Create 路径，错误虽会在 Create 处暴露，但错误归因被挪后、且"已存在跳过"判定在故障时给出错误结论。
**Fix:** 统一 `if err := ...Count(&count).Error; err != nil { return fmt.Errorf(...) }`（同文件 `createRequestEncryptionToggleConfig` :925-932 已是正确范式）。

#### IN-04: 约 106 行注释掉的 createCaptchaBackgroundMenus 死代码 + 引用不存在函数的过时注释

**File:** `internal/core/db/init_data.go:617-740`
**Issue:** :619-723 整个函数被块注释包裹（含内部引用未定义的 `log` 包用法，取消注释也无法编译）；:733-739 的注释段描述 `createDutyManagementMenus`，该函数在文件中并不存在。两者都会误导后来者。
**Fix:** 删除 :619-723 注释块与 :733-740 过时注释（git 历史可找回）。

#### IN-05: TestInfoWarn_MinLevelInfo 失败消息中的级别数值写反

**File:** `internal/core/db/filter_logger_test.go:124`
**Issue:** 断言失败时输出 "...no; Warn=4, Info=2 — only when MinLevel <= Warn"，而 GORM 实际取值为 Silent=1, Error=2, Warn=3, Info=4（filter_logger.go:73-74 注释自己也写对了）。断言本身正确，但失败文案会误导排障方向。
**Fix:** 文案改为 "Warn=3, Info=4 — MinLevel=Info(4) >= Warn(3) so Warn also emits"。

#### IN-06: 10s pingCtx 同时用于 CREATE DATABASE，慢存储上首次建库可能超时假失败

**File:** `internal/core/db/database.go:741-758`
**Issue:** `createDatabaseIfNotExists` 用同一个 10s context 做 Ping、EXISTS 检查和 `CREATE DATABASE`。首次建库在慢存储/托管 PG（如 Supabase 冷备库）上可能超 10s，被误判为失败并 fail-fast 启动。EXISTS 查询用 10s 合理；建库 DDL 建议独立更长超时（如 60s）。
**Fix:** 为 `db.ExecContext(createCtx, createQuery)` 使用单独的 `context.WithTimeout(context.Background(), 60*time.Second)`。

#### IN-07: advisory lock 未覆盖 cleanupOldConstraints 与 GORM AutoMigrate 本体，多副本并发时 DDL 仍可能互撞

**File:** `internal/core/db/database.go:461-468` 与 `:492-497`
**Issue:** C3 的 advisory lock 只包裹 175/176/202-205 迁移块；`cleanupOldConstraints()` 与 `d.DB.Migrator().AutoMigrate(...)`（:462-468）在锁外执行。HA/滚动重启场景下两个实例仍会并发执行 80+ 条模型 DDL 与约束 DROP，C3 想排除的竞态类问题在这段路径上依然存在。当前部署形态为单机（core.go:884 注释），故降为提示。
**Fix:** 将 advisory lock 获取上移到 cleanupOldConstraints 之前（未获锁实例跳过整个 AutoMigrate 块，仅保留连接初始化），或在注释中明确记录该路径的多副本限制。

#### IN-08: createDefaultUser 存在性检查与 Create 之间存在 TOCTOU 窗口

**File:** `internal/core/db/init_data.go:224-260`
**Issue:** "Count → Create" 非原子。单实例启动下无碍，但 CDX-H1 专门容忍了并发 bootstrap（42P04），说明多实例同时首启是受支持场景 —— 此时两实例可能同时通过 count==0 检查，后者在 `sys_user.username` UNIQUE 约束上失败并使 InitData 报错（仅 WARN 非阻断，影响有限）。
**Fix:** 捕获 Create 的 UNIQUE 冲突并降级为 Info（"已被并发实例创建"），或用 `INSERT ... ON CONFLICT (username) DO NOTHING` 语义。

---

_Reviewed: 2026-08-14T16:01:59Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
