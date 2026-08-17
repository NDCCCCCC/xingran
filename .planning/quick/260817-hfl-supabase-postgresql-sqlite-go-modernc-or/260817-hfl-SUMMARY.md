---
phase: 260817-hfl
plan: 01
subsystem: database
tags: [sqlite, database, dev-environment, glebarez, modernc]
requires: [github.com/glebarez/sqlite v1.11.0]
provides:
  - database.type / database.path 配置开关(postgres|sqlite)
  - NewDatabase sqlite 分支(纯 Go 驱动,无 CGO)
  - sqlite 下 PG-only DDL 片段净化(sanitizeSQLiteModelDefaults)
affects: [internal/core/db, internal/config, internal/models, internal/core, configs]
tech-stack:
  added: [glebarez/sqlite 分支使用]
  patterns: [schema-cache 投影净化(运行期修改 Parse 缓存,不改 model tag,PG 语义零改动)]
key-files:
  created:
    - internal/models/user_preference.go
  modified:
    - internal/config/config.go
    - internal/core/db/database.go
    - internal/core/db/database_test.go
    - internal/core/core.go
    - internal/models/asset.go
    - internal/models/api_key_usage_log.go
    - internal/models/vdi_sync_log.go
    - internal/services/system/settings_service.go
    - configs/config.example.yaml
    - configs/config.yaml (gitignored, 本地)
    - .gitignore
decisions:
  - "sqlite 分支不启动 pool keepalive(本地文件无 TLS/auth 握手开销)"
  - "SKIP_AUTOMIGRATE 旁路收窄为 postgres-only(sqlite 无 pooler 卡死,新文件库必须全量 AutoMigrate)"
  - "PG-only DDL 片段用 schema 缓存投影净化,不改 model tag(避免 PG AutoMigrate DROP DEFAULT 回归)"
  - "被剥离 gen_random_uuid() 的模型补 BeforeCreate uuid 钩子(PG 行为等价,sqlite 必需)"
metrics:
  duration: ~35min
  completed: 2026-08-17
  tasks: 3 (2 auto + 1 human-verify checkpoint pending)
---

# Phase 260817-hfl Plan 01: 恢复后端本地 SQLite 支持(纯 Go 驱动) Summary

**One-liner:** database.type=sqlite 一条配置把 dev 后端从 Supabase 切到本地 SQLite 文件(纯 Go glebarez 驱动),AutoMigrate ~7s 完成全量建表 + seed,PG 生产路径零改动。

## What Was Built

### Task 1: DatabaseConfig type/path + NewDatabase SQLite 分支 (TDD)

- `internal/config/config.go`: `DatabaseConfig` 增加 `Type`(postgres|sqlite,缺省 postgres)与 `Path`(sqlite 文件路径,默认 `data/xingran.db`)字段 + viper 默认值;`GetDSN` 不动(仅 PG)。
- `internal/core/db/database.go`: `NewDatabase` 按 `cfg.Type=="sqlite"` 分支到新增 `createSQLiteConnection`;空 type 保持 PG fail-fast 语义一字不动(host/port 校验保留在原分支)。
- `createSQLiteConnection`: modernc `_pragma` DSN(busy_timeout 10s / WAL / foreign_keys ON)+ `os.MkdirAll` 建目录 + `MaxOpenConns=4`(SQLite 单写者)+ UTC NowFunc/PrepareStmt:false 与 PG 对齐;不启动 pool keepalive。
- `database_test.go`: 新增 `TestNewDatabaseSQLite`(临时文件建库 + Type/keepalive nil/ping/全量 AutoMigrate 断言)、`TestNewDatabaseEmptyTypeFallsBackToPostgres`(type 空 → PG fail-fast);更新 `TestNewDatabaseRequiresPostgresConfig` banned 列表(移除 `func createSQLiteConnection(` / `sqlite.Open(`,保留 CGO 驱动禁令 `"gorm.io/driver/sqlite"` 与 `_ "modernc.org/sqlite"`)。
- `.gitignore`: 追加 `data/`(`*.db` 已存在)。

### Task 2: 配置文件切换

- `configs/config.yaml`(gitignored,本地): `database.type: "sqlite"` + `path`,全部 PG 键原样保留(切回 Supabase 只需改 type)。
- `configs/config.example.yaml`: `type: "postgres"` 默认 + 双模式中文注释 + dev-only 限制声明;`config.prod.example.yaml` 未动。

### Task 3: 冒烟验证 — 自动化部分已完成,人工部分待 checkpoint

**已自动化验证(executor 完成,两轮):**
- 第一轮: `SQLite连接成功: data/xingran.db` ✓;`所有表迁移成功` 距连接 ~7s(对比 Supabase ~30s+)✓;无 `PostgreSQL连接成功` / 无 `pool-keepalive` 日志 ✓;InitData seed 完整(admin 出厂密码告警、默认角色菜单、rate_limit 配置)✓;`/api/v1/system/auth/public-key` 返回 SM2 公钥、`/api/v1/system/auth/login` 正常参数校验 ✓
- 第二轮(删库冷启动,修复 sys_user_preference 后): 84 表创建,`sys_user_preference` 存在 ✓;启动至迁移完成 ~6s;残留缺表仅 sys_rpa_workers / sys_rpa_executions / sys_mac_oui_vendor(非阻塞)✓
- `go build ./...` exit 0;`go list -deps ./... | grep mattn/go-sqlite3` 为空(无 CGO 新增)✓

**待人工验证(checkpoint):**
- 前端 `npm run dev`(localhost:4000)走一遍 admin 登录 + 菜单加载
- 可逆性:`type` 改回 `postgres` 重启确认 Supabase 路径正常

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] PG-only DDL 片段导致 sqlite AutoMigrate/CreateTable 必失败**
- **Found during:** Task 3 自动化冒烟(plan 勘察断言 "BaseModel.ID 由 BeforeCreate 生成,SQLite 下正常" 漏掉了 5 个自带 `default:gen_random_uuid()` tag 的模型与 2 个 `type:text[]` 字段)
- **Issue:** `CREATE TABLE sys_api_key_usage_logs (... DEFAULT gen_random_uuid() ...)` 在 SQLite 报 `near "(": syntax error`;`text[]` 数组类型 SQLite type-name 语法不接受方括号
- **Fix:** `database.go` 新增 `sanitizeSQLiteModelDefaults()` — 经 `gorm.Statement{DB: d.DB}.Parse` 共享 cacheStore 就地净化缓存 schema(剥离函数式 DEFAULT、`text[]`→`text`),AutoMigrate 与 BootstrapMissingTables 的 sqlite 分支前置调用。model tag 不改,PG 语义零改动(避免 GORM 在 PG 上 DROP DEFAULT 回归)
- **Files modified:** internal/core/db/database.go
- **Commit:** abfa111

**2. [Rule 2 - Missing critical] 被剥离 DB 默认值的模型缺应用层 ID 生成**
- **Found during:** 同上分析 — Asset / APIKeyUsageLog / VDISyncLog 无 BeforeCreate 钩子,sqlite 下(默认值被净化后)GORM Create 会写入空 ID
- **Fix:** 三个模型补 `BeforeCreate` uuid 钩子(`github.com/google/uuid`,与 BaseModel 同模式);PG 下行为等价(非空 ID 直接 INSERT,DB 默认值仅对零值/原生 SQL 路径生效)。VDISyncLog 嵌入 BaseModel 但外层 ID 遮蔽 BaseModel.ID,BaseModel 钩子触达不到,故显式钩子
- **Files modified:** internal/models/asset.go, internal/models/api_key_usage_log.go, internal/models/vdi_sync_log.go
- **Commit:** abfa111

**3. [Rule 3 - Blocking] `.env` 残留 SKIP_AUTOMIGRATE=true 会让 sqlite 新库保持空壳**
- **Found during:** Task 3 自动化冒烟 — 首次冒烟仅建 2 张 bootstrap 表且建表失败
- **Issue:** SKIP_AUTOMIGRATE 旁路是为 Supabase pooler 卡死设计的;sqlite 无此问题,且新文件库必须全量 AutoMigrate
- **Fix:** 旁路收窄为 postgres-only:`database.go AutoMigrate` 内部检查与 `core.go initDBAndData` 分支均加 `Type == "postgres"` 条件;`.env` 不动(切回 PG 时旁路自动恢复生效)
- **Files modified:** internal/core/db/database.go, internal/core/core.go
- **Commit:** abfa111

**4. [Rule 1 - Scope note] Task 2 commit 携带了 config.example.yaml 预存在 WIP**
- **Found during:** Task 2 commit — 会话开始时工作树已有该文件未提交改动(max_open_conns 10→25、max_idle_conns 5→10,Phase 62-DBG-01 池饥饿缓解的注释)
- **Fix:** 与当前 config.yaml 实际值一致且为文档性 example 配置,一并提交(语义正确,非本任务引入)
- **Commit:** 0505049

**5. [Rule 3 - Blocking] sys_user_preference 缺表导致登录后首屏 500(人工冒烟发现)**
- **Found during:** Task 3 人工冒烟 — 登录后 `GET /api/v1/system/settings/preferences` 500:`no such table: sys_user_preference`
- **Issue:** 该表由归档 SQL(archive/legacy-2026-06-15/004 建表 + 005 扩展主题/布局列 + 044 自定义颜色列)创建,从未进入 AutoMigrate;模型 `UserPreference` 定义在 `internal/services/system/settings_service.go`(core/db 不可达)。PG 存量库因历史遗留已存在而无感;全新 SQLite 库缺表
- **Fix:** 模型迁移至 `internal/models/user_preference.go`(schema 单一事实源,005/044 增量列已全部覆盖),`services/system` 保留 `type UserPreference = models.UserPreference` alias(全部调用点 + scripts/dbprovision 零改动);AutoMigrate 仅 sqlite 分支注册该模型 — PG 不注册以避免 GORM 对存量表发起漂移 ALTER(DROP NOT NULL / sidebar_width 默认值 240→280 改写等),PG 新部署由 scripts/dbprovision 建表
- **Files modified:** internal/models/user_preference.go(新建), internal/services/system/settings_service.go, internal/core/db/database.go
- **Commit:** 8e09816(RED 断言) + d2a0cdb(修复)
- **验证:** 删库冷启动后 `sys_user_preference` 建表成功(全库 84 表),登录链路其余表(sys_user/sys_menu/sys_role/sys_config)齐全
- **顺带排查结论(同任务要求 2):** `default_theme_service.go:157` 引用复数表名 `sys_user_preferences` — 该表在归档 SQL 与全代码库中均不存在,是 PG 上同样必失败的 pre-existing typo bug;SyncUserThemeToDefault 为动作型端点(非登录首屏链路),记入 deferred-items.md 不扩大范围。运行时缺表全景:仅 sys_rpa_workers / sys_rpa_executions / sys_mac_oui_vendor(非阻塞,已知限制)

## Known Dev-Only Limitations (sqlite 模式)

| 限制 | 影响 | 处置 |
|------|------|------|
| reconciliation MV/视图(175/176)不建 | 对账看板空 | 已知,d.Type guard 设计如此 |
| 202-206 迁移 seed 不执行(端口写审计菜单、连接池 sys_config、dot1x 列、rpa id DEFAULT、关联索引) | 补充菜单缺失等 | 已知,PG 切回即恢复 |
| `sys_rpa_workers` / `sys_rpa_executions` / `sys_mac_oui_vendor` 缺表 | RPA worker 心跳 400、RPA 调度查询报错(日志 ERRO,非启动阻断) | 这些表由归档 SQL 迁移(102 等)创建、不在 MigrateModelList;不补进 AutoMigrate 列表以避免 PG 端 ALTER/DROP DEFAULT 回归风险 |
| ~~`sys_user_preference` 缺表~~ | ~~登录后首屏 preferences 500~~ | **已修复**(偏差 #5,sqlite 分支 AutoMigrate 注册,commit d2a0cdb) |
| 个别 handler 原生 SQL 含 PG 方言(::uuid/ILIKE/pg_catalog) | 运行期报错 | 按需后续修(参考 quick-260814-211 模式) |
| `pq.StringArray`(text[] 净化为 text) | 存 "{a,b}" 字面量文本,Scan 可 roundtrip;跨库 SQL 函数(如 unnest)不可用 | dev-only 可接受 |
| `default:now()` 等被剥离(如 SysDataReconciliation.DetectedAt,不在迁移列表) | 写入零值时间 | 该模型不经 GORM AutoMigrate 建表,仅归档 SQL;sqlite 下无此表 |

## Test Results

- `go test ./internal/core/db/` PASS(含 TestNewDatabaseSQLite 全量 AutoMigrate 端到端)
- `go test ./internal/config/` PASS
- `go test ./internal/core/` PASS(既有 SKIP_AUTOMIGRATE 源码断言 needle 仍命中)
- `go test ./internal/models/...` PASS
- `go build ./...` exit 0;无 mattn/go-sqlite3(CGO)依赖

## TDD Gate Compliance

RED(7c422a1: TestNewDatabase* 编译失败)→ GREEN(7b7bd0d: 实现通过)→
RED(2b4d08d: AutoMigrate 扩展断言暴露 PG-only DDL 失败)→ GREEN(abfa111: 净化 + 钩子修复)。
双 RED/GREEN 门完整。

## Commits

| Commit | Type | Message |
|--------|------|---------|
| 7c422a1 | test | add failing tests for sqlite branch in NewDatabase |
| 7b7bd0d | feat | restore sqlite branch via pure-Go glebarez driver |
| 0505049 | chore | document dual db modes in config example |
| 2b4d08d | test | extend sqlite test with full AutoMigrate guard |
| abfa111 | fix | sanitize PG-only DDL fragments for sqlite AutoMigrate |
| 8e09816 | test | assert sys_user_preference exists after sqlite AutoMigrate |
| d2a0cdb | fix | create sys_user_preference on sqlite via model-derived AutoMigrate |

## Self-Check

- FOUND: internal/config/config.go (Type/Path 字段)
- FOUND: internal/core/db/database.go (createSQLiteConnection / sanitizeSQLiteModelDefaults / sqlite-only UserPreference 注册)
- FOUND: internal/models/user_preference.go (UserPreference 模型单一事实源)
- FOUND: internal/services/system/settings_service.go (type alias 兼容层)
- FOUND: internal/core/db/database_test.go (TestNewDatabaseSQLite 含 sys_user_preference 断言)
- FOUND: configs/config.yaml (type: "sqlite", 本地 gitignored)
- FOUND commits: 7c422a1, 7b7bd0d, 0505049, 2b4d08d, abfa111, 8e09816, d2a0cdb
- 冷启动验证(删库重跑): 84 表创建,sys_user_preference 存在,启动至 "所有表迁移成功" ~6s,残留缺表仅 rpa/oui 三张非阻塞表

## Self-Check: PASSED
