---
phase: 260817-hfl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/config/config.go
  - internal/core/db/database.go
  - internal/core/db/database_test.go
  - configs/config.example.yaml
  - configs/config.yaml
  - .gitignore
autonomous: false
requirements:
  - QUICK-260817-HFL
tags: [sqlite, database, dev-environment, glebarez, modernc]

must_haves:
  truths:
    - "config.yaml 中 database.type=sqlite 时,后端连接本地 SQLite 文件启动,不再访问 Supabase"
    - "database.type 缺省/为空时行为与现在完全一致(PostgreSQL fail-fast 校验保留)"
    - "SQLite 模式下 AutoMigrate 从 model 派生建表,InitData seed 正常执行"
    - "PostgreSQL 支持保留(production 路径零改动语义)"
  artifacts:
    - path: "internal/config/config.go"
      provides: "DatabaseConfig.Type / DatabaseConfig.Path 字段 + database.type 默认值"
      contains: "mapstructure:\"type\""
    - path: "internal/core/db/database.go"
      provides: "NewDatabase 按 cfg.Type 分支 + createSQLiteConnection"
      contains: "createSQLiteConnection"
    - path: "internal/core/db/database_test.go"
      provides: "SQLite 分支单元测试(临时文件建库 + Type 断言)"
      contains: "TestNewDatabaseSQLite"
    - path: "configs/config.yaml"
      provides: "dev 环境切到 sqlite 的活动配置"
      contains: "type: \"sqlite\""
  key_links:
    - from: "internal/core/db/database.go NewDatabase"
      to: "github.com/glebarez/sqlite"
      via: "gorm.Open(sqlite.Open(dsn)) (纯 Go 驱动,已在 go.mod,严禁引入 gorm.io/driver/sqlite=CGO)"
      pattern: "glebarez/sqlite"
    - from: "internal/core/core.go initDBAndData"
      to: "Database.AutoMigrate / InitData"
      via: "现有调用链,SQLite 下 d.Type!=\"postgres\" 自动跳过 cleanup/advisory-lock/175-206 迁移块(已有 guard)"
      pattern: "d\\.Type == \"postgres\""
---

<objective>
恢复后端本地 SQLite 支持(纯 Go 驱动),让 dev 环境通过配置开关从 Supabase 远程 PostgreSQL 切到本地 SQLite 文件,解决 Supabase 网络慢的问题。

Purpose: dev 环境启动与日常调试不再受 Supabase 跨国链路延迟影响;生产仍走 PostgreSQL,配置切换即可,不删除任何 PG 代码路径。
Output: `database.type: sqlite` 配置项 + glebarez/sqlite 连接分支 + 单元测试 + 切换好的 config.yaml。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@CLAUDE.md
@.planning/STATE.md
@internal/core/db/database.go
@internal/config/config.go
@internal/models/base.go

<interfaces>
<!-- 已有事实(本次勘察已确认,executor 直接采用,不要重复全局搜索): -->

go.mod 已有直接依赖(无需 go get):
  github.com/glebarez/sqlite v1.11.0        // 纯 Go GORM 驱动(底层 modernc.org/sqlite v1.40.1)
  严禁使用 gorm.io/driver/sqlite — 它传递依赖 mattn/go-sqlite3(CGO),Windows 无 gcc 时 go run 直接失败
  (这正是 2026-08-15 删除旧 sqlite 回退路径的原因,见 database.go:39-45 注释)

internal/config/config.go:70 DatabaseConfig(当前无 Type/Path 字段):
```go
type DatabaseConfig struct {
    Host         string `mapstructure:"host"`
    Port         int    `mapstructure:"port"`
    User         string `mapstructure:"user"`
    Password     string `mapstructure:"password"`
    DBName       string `mapstructure:"dbname"`
    SSLMode      string `mapstructure:"sslmode"`
    MaxOpenConns int    `mapstructure:"max_open_conns"`
    MaxIdleConns int    `mapstructure:"max_idle_conns"`
    MaxLifetime  int    `mapstructure:"max_lifetime"`
}
```
viper 默认值集中在 config.go ~line 461 数据库默认配置区(v.SetDefault("database.host", ...) 附近),
新默认值 v.SetDefault("database.type", "postgres") 与 v.SetDefault("database.path", "data/xingran.db") 加在那里。

internal/core/db/database.go 现状:
- NewDatabase(cfg) line 46: host==""||port<=0 → fail-fast error;固定 dbType="postgres";启动 pool keepalive
- createPostgresConnection(cfg) line 136:gorm.Open(postgres.Open(cfg.GetDSN()), gormConfig)
- AutoMigrate() line 481:cleanupOldConstraints / dropDependentMaterializedViews / advisory lock /
  migrations.Migrate175/176/202-206 全部已在 `if d.Type == "postgres"` guard 内 — SQLite 分支天然跳过,零改动
- BootstrapMissingTables():CREATE INDEX IF NOT EXISTS 语句 SQLite 兼容,无需改动
- init_data.go 无 PG 特有 SQL(已 grep 确认:gen_random_uuid/ON CONFLICT/RETURNING/ILIKE/pg_catalog 均 0 命中),GORM seed 直接可用
- models.BaseModel.ID 由 BeforeCreate 钩子生成 uuid(非 DB DEFAULT),SQLite 下 `type:uuid` 列映射为 TEXT affinity,正常

glebarez/sqlite DSN 写法(走 modernc `_pragma` 参数):
  data/xingran.db?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)

⚠️ 既有回归守护测试冲突(勘察确认,Task 1 必须同步处理):
  internal/core/db/database_test.go:150 TestNewDatabaseRequiresPostgresConfig 是源码断言测试,
  其 banned 列表当前包含 "func createSQLiteConnection(" 与 "sqlite.Open(" — 本计划恢复的
  glebarez 分支恰好命中这两条,不更新该测试则 Task 1 verify 必失败。
  处理:从 banned 列表移除这两条,保留 "\"gorm.io/driver/sqlite\""(CGO 驱动 import 串)与
  "_ \"modernc.org/sqlite\""(防 blank import)两条禁令;fail-fast 断言(Host/Port 检查片段)保留。

.gitignore 现状:line 62 已有 `*.db`,但无 `data/` 目录条目 — 只追加 `data/`(若已存在则跳过)。

已知 dev-only 限制(需在代码注释与 config 注释中声明,不阻塞):
- reconciliation 物化视图/普通 VIEW(175/176)、菜单 seed(202)、sys_config 连接池 seed(203)等 PG-only 迁移块不执行
- 个别 handler 原生 SQL 若含 PG 语法(::uuid cast / ILIKE / pg_catalog)在 SQLite 下会运行期报错 — 按需后续修
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: DatabaseConfig 增加 type/path 字段 + NewDatabase SQLite 分支(纯 Go 驱动)</name>
  <files>internal/config/config.go, internal/core/db/database.go, internal/core/db/database_test.go, .gitignore</files>
  <behavior>
    - Test 1 (新增, sqlite 分支): TestNewDatabaseSQLite — NewDatabase(&DatabaseConfig{Type:"sqlite", Path: filepath.Join(t.TempDir(),"t.db")}) 返回 err==nil,d.Type=="sqlite",ping 通过、keepalive 未启动(d.keepaliveStop 为 nil,同包测试可直接断言);跑完 d.Close()
    - Test 2 (新增, type 缺省不静默落 sqlite): Type 为空字符串按 postgres 处理(host 缺失时报与现有一致的 fail-fast error)
    - Test 3 (更新既有): TestNewDatabaseRequiresPostgresConfig — banned 列表移除 "func createSQLiteConnection(" 与 "sqlite.Open(",保留 "\"gorm.io/driver/sqlite\"" 与 "_ \"modernc.org/sqlite\"";fail-fast 断言保留;更新 doc 注释说明 2026-08-17 以纯 Go glebarez/sqlite 恢复 sqlite 分支,CGO 驱动禁令仍然有效
    - 既有 internal/config、internal/core/db 其余测试全部保持 PASS 不改动
  </behavior>
  <action>
    1) internal/config/config.go:DatabaseConfig 增加两个字段(Type string `mapstructure:"type"` 注释"postgres|sqlite";Path string `mapstructure:"path"` 注释"sqlite 文件路径,仅 type=sqlite 生效"),defaults 区(~line 461 数据库默认配置块)加 v.SetDefault("database.type", "postgres") 与 v.SetDefault("database.path", "data/xingran.db")。GetDSN 不动(仅 PG 使用)。
    2) internal/core/db/database.go:NewDatabase 改为分支:cfg.Type=="sqlite" → createSQLiteConnection(cfg);其他(含空)→ 现有 PG 路径一字不动(host/port fail-fast 保留在原分支内)。sqlite 分支不启动 pool keepalive(本地文件无握手开销),d.Type="sqlite"。更新 NewDatabase 头注释:说明 2026-08-17 以纯 Go 驱动 glebarez/sqlite 恢复 sqlite 分支(区别于已删除的旧 CGO 路径,CGO 禁令由测试守护)。
    3) 新增 createSQLiteConnection(cfg *config.DatabaseConfig) (*gorm.DB, error):import "github.com/glebarez/sqlite";dsn = cfg.Path(空则用 "data/xingran.db" 兜底)+ "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)";用 os.MkdirAll(filepath.Dir(path), 0755) 确保目录存在(仅当 dir 非 "." 时);gorm.Config 复用 createFilteredLogger() + NowFunc UTC + DisableForeignKeyConstraintWhenMigrating:true + SkipDefaultTransaction:true + PrepareStmt:false(与 PG 对齐);打开后 sqlDB.SetMaxOpenConns(4) 并注释原因(SQLite 单写者,WAL+busy_timeout 下小池即可,避免 25 连接无意义);Ping 验证;applogger.Infof("SQLite连接成功: %s", path)。
    4) database_test.go:按 behavior Test 3 更新 TestNewDatabaseRequiresPostgresConfig;新增 TestNewDatabaseSQLite 与 type 缺省回归测试(同包 db,可访问私有字段)。
    5) .gitignore:仅追加 data/ 目录条目(*.db 已存在于 line 62,勿重复添加)。
    禁止:引入 gorm.io/driver/sqlite;改动 createPostgresConnection / cleanupOldConstraints / AutoMigrate 的 PG 逻辑;把 sqlite 设为默认值;改动 TestNewDatabaseRequiresPostgresConfig 之外的既有测试。
  </action>
  <verify>
    <automated>go test ./internal/core/db/ -x &amp;&amp; go test ./internal/config/ -x &amp;&amp; go build ./...</automated>
  </verify>
  <done>
    internal/core/db 与 internal/config 全部测试 PASS(含更新后的 TestNewDatabaseRequiresPostgresConfig 与新增 TestNewDatabaseSQLite);
    go build ./... exit 0;go.mod 中 glebarez/sqlite 保持 direct require 且无 mattn/go-sqlite3 新增
  </done>
</task>

<task type="auto">
  <name>Task 2: 配置文件切换(dev config.yaml → sqlite,example 文档化双模式)</name>
  <files>configs/config.yaml, configs/config.example.yaml</files>
  <action>
    1) configs/config.yaml database 段顶部加 type: "sqlite" 与 path: "data/xingran.db",原有 host/port/user/password/dbname/sslmode/max_* 键全部保留(加注释"以下 PG 键仅在 type=postgres 时生效;切回 Supabase 只需把 type 改回 postgres")。当前 host 指向 db.bkixsntumwntnwpxavfu.supabase.co、sslmode: "require",均原样保留。
    2) configs/config.example.yaml database 段加 type: "postgres"(example 默认保持 PG)+ path 键与中文注释说明两种模式:sqlite=本地开发提速(纯 Go 驱动,数据文件 data/xingran.db,首次启动自动建库建表+seed);postgres=生产/对齐远端 schema。
    3) 两个文件注释中声明 dev-only 限制:sqlite 模式下 reconciliation MV/视图与 202-206 迁移 seed 不执行,个别含 PG 方言的端点可能报错。
    4) 不改 config.prod.example.yaml(生产继续 PG)。
  </action>
  <verify>
    <automated>grep -n "type: \"sqlite\"" configs/config.yaml &amp;&amp; grep -n "type: \"postgres\"" configs/config.example.yaml &amp;&amp; go build ./...</automated>
  </verify>
  <done>config.yaml 指向 sqlite 且 PG 键完整保留;example 文件两种模式均有说明;构建通过</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: 冒烟验证 — SQLite 本地启动 + 登录</name>
  <what-built>
    后端可通过 database.type=sqlite 使用本地文件库启动:AutoMigrate 自动建 ~80 张表、InitData seed 默认 admin/角色/菜单,不再访问 Supabase。
  </what-built>
  <how-to-verify>
    1. 确认 configs/config.yaml 中 database.type: "sqlite"(Task 2 已改)
    2. 删除旧数据文件(若存在):rm -f data/xingran.db
    3. 启动:go run ./cmd/main.go
    4. 期望日志:出现 "SQLite连接成功: data/xingran.db"、"所有表迁移成功",且无 PG/Supabase 连接日志、无 pool-keepalive 日志
    5. 计时对比:启动到 "所有表迁移成功" 应明显快于 Supabase(原 ~30s+)
    6. 前端 npm run dev(localhost:4000)走一遍登录:admin 登录成功、菜单正常加载(注意:202 迁移的端口写审计等补充菜单在 sqlite 下不 seed,属已知限制)
    7. 验证可逆:把 type 改回 postgres,重启,确认 Supabase 路径仍正常(确认后可改回 sqlite)
  </how-to-verify>
  <resume-signal>Type "approved" 或描述问题(报错请贴完整日志行)</resume-signal>
</task>

</tasks>

<verification>
- go build ./... exit 0(无 CGO 依赖引入:go list -deps ./... | grep mattn/go-sqlite3 应为空)
- go test ./internal/core/db/ ./internal/config/ 全 PASS
- config.yaml type=sqlite 冒烟启动成功(人工门)
- config.yaml type 改回 postgres 后 PG 路径仍可用(可逆性,人工门第 7 步)
</verification>

<success_criteria>
- dev 后端一条配置(type: sqlite)切到本地 SQLite,启动显著提速
- PostgreSQL 支持零回归(默认 type=postgres,fail-fast 校验保留,更新后的回归守护测试通过)
- 依赖面无 CGO 新增(modernc.org/sqlite 经 glebarez 已在 go.mod)
</success_criteria>

<output>
创建 `.planning/quick/260817-hfl-supabase-postgresql-sqlite-go-modernc-or/260817-hfl-01-SUMMARY.md`,
记录:分支设计、已知 dev-only 限制清单、人工冒烟结果、是否发现新的 PG 方言端点报错。
</output>
