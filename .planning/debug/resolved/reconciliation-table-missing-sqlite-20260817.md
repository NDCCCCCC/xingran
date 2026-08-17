---
slug: reconciliation-table-missing-sqlite-20260817
status: resolved
trigger: |
  定时任务「对账-自动转工单critical」在 SQLite 模式下失败：
  ERRO [GORM错误] SELECT * FROM `sys_data_reconciliation` WHERE (severity = "critical" AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))) AND `sys_data_reconciliation`.`deleted_at` IS NULL ORDER BY detected_at ASC LIMIT 50 | 错误: SQL logic error: no such table: sys_data_reconciliation (1)
  ERRO [reconciliation:createWorkorderCritical] 失败: 查询 critical 异常失败: SQL logic error: no such table: sys_data_reconciliation (1)
  ERRO 任务执行失败 [对账-自动转工单critical.reconciliation]
created: 2026-08-17
updated: 2026-08-17
goal: find_and_fix
tdd_mode: false
---

# Debug: sys_data_reconciliation table missing on SQLite

## Symptoms

- **Expected behavior**: 定时任务「对账-自动转工单critical」在 sqlite 模式下正常查询 `sys_data_reconciliation` 表并创建工单
- **Actual behavior**: 任务失败，GORM 报 `SQL logic error: no such table: sys_data_reconciliation (1)`
- **Error messages**:
  ```
  ERRO [GORM错误] SELECT * FROM `sys_data_reconciliation` WHERE (severity = "critical" AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL AND (applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions))) AND `sys_data_reconciliation`.`deleted_at` IS NULL ORDER BY detected_at ASC LIMIT 50 | 错误: SQL logic error: no such table: sys_data_reconciliation (1)
  ERRO [reconciliation:createWorkorderCritical] 失败: 查询 critical 异常失败: SQL logic error: no such table: sys_data_reconciliation (1)
  ERRO 任务执行失败 [对账-自动转工单critical.reconciliation]
  ```
- **Timeline**: 2026-08-17 17:47:42 生产日志；近期 commits 正在修 sqlite AutoMigrate 相关 (260817-hfl-01)
- **Reproduction**: sqlite 模式下启动应用，等待/触发对账-自动转工单critical 定时任务

## Known Context (from orchestrator)

- 项目: XingRan-Next (Go + GORM, 双数据库 postgres/sqlite), cwd D:\code\ClaudeCode\guoguo
- 迁移机制: Go 代码 `MigrateNNN` (`internal/core/db/migrations/`), archive SQL 不自动执行
- 近期相关 commits:
  - d2a0cdb fix(260817-hfl-01): create sys_user_preference on sqlite via model-derived AutoMigrate
  - abfa111 fix(260817-hfl-01): sanitize PG-only DDL fragments for sqlite AutoMigrate
- 提示根因同类: `sys_data_reconciliation` 表可能由 PG-only 迁移创建，sqlite 启动路径漏建
- 相关代码: `internal/services/asset/reconciliation*`, `internal/scheduler/reconciliation_tasks*.go`, 模型在 `internal/models/` (搜 `sys_data_reconciliation` 的 TableName)
- 次要疑点: `'no_workorder' != ANY(applied_actions)` 是 PG 数组语法，sqlite 下即使表存在也会报错 — 需一并评估

## Current Focus

reasoning_checkpoint:
  hypothesis: "sys_data_reconciliation 表仅由已归档的 migration_168(archive/applied,启动期不执行)创建;AutoMigrate 的 MigrateModelList() 未注册 models.SysDataReconciliation,导致 sqlite 全新文件库永远缺表。次要因:scheduler 转单 SQL 与 ListExceptions silence 过滤使用 PG 方言 ANY(text[]),sqlite 下即使表存在也报语法错误"
  confirming_evidence:
    - "database.go MigrateModelList()(L474-573)含 SysReconciliationException 但不含 SysDataReconciliation;sqlite 分支额外 append 列表(L630-668)也没有"
    - "migration_168_reconciliation_tables.go 位于 archive/applied/(归档不自动跑);AutoMigrate 注释明确'启动期不再调用 200+ migration 函数'"
    - "生产日志报错语句与 internal/scheduler/reconciliation_tasks.go:272 的 createWorkorderBySeverity 查询逐字匹配"
    - "sanitizeSQLiteModelDefaults 只遍历 MigrateModelList();SysDataReconciliation tag 含 default:now() 与 type:text[] 等 PG-only 片段,sqlite 注册前必须纳入净化"
  falsification_test: "sqlite 全新库跑 d.AutoMigrate() 后 HasTable(sys_data_reconciliation)==false 即证实缺注册;若已 true 则假设被证伪"
  fix_rationale: "镜像 260817-hfl-01 模式:仅 sqlite 分支注册模型建表(PG 存量表零漂移),并将其纳入 sanitizer 净化 PG-only 片段;ANY() 改为方言条件查询(PG 保持 ANY,sqlite 用逗号包裹 LIKE 等价判定),根治而非绕过"
  blind_spots: "reconciliation_detection.go 的 DetectLayer3 INSERT 路径与 reconciliation_normalized MV 在 sqlite 下的行为未逐一验证(MV 重建已被 d.Type==postgres guard 排除,mvAvailable 探测失败会走 fallback 路径);jsonb/inet 类型名在 sqlite 非 STRICT 表下可接受但未经逐列比对"
next_action: apply fix — database.go 注册+净化, reconciliation_tasks.go 方言条件查询, reconciliation_service.go silence 过滤方言条件, database_test.go HasTable 断言

## Evidence

- timestamp: 2026-08-17
  checked: internal/core/db/database.go MigrateModelList() L474-573 + AutoMigrate sqlite 分支 L630-668
  found: SysReconciliationException 已注册(L571,注释承认"原 migration_174 归档不自动跑,此处补注册"),但 SysDataReconciliation 完全未出现在任何 AutoMigrate 列表
  implication: sqlite 全新文件库无 sys_data_reconciliation 表 — 主根因坐实
- timestamp: 2026-08-17
  checked: internal/core/db/migrations/ 目录结构
  found: migration_168_reconciliation_tables.go(建 sys_data_reconciliation 的迁移)在 archive/applied/;database.go L577-601 注释明确启动期不再调用 MigrateNNN
  implication: PG 存量库靠历史迁移有表;sqlite 新库只能靠 AutoMigrate,缺注册=缺表
- timestamp: 2026-08-17
  checked: internal/models/reconciliation.go SysDataReconciliation tag
  found: 含 PG-only 片段 — default:now()(DetectedAt L59)、type:text[](AppliedActions L56);jsonb/inet 类型名 sqlite 非 STRICT 表可接受。BaseModel 有 BeforeCreate uuid 钩子,ID 无函数默认问题
  implication: sqlite 注册该模型前必须纳入 sanitizeSQLiteModelDefaults 净化范围(当前只遍历 MigrateModelList)
- timestamp: 2026-08-17
  checked: internal/scheduler/reconciliation_tasks.go:268-277 createWorkorderBySeverity
  found: WHERE 含 'no_workorder' != ANY(applied_actions) — PG 数组方言,与生产报错 SQL 逐字匹配;reconciliation_service.go:459/509 另有 2 处 NOT('silence' = ANY(...)),其 L455-457 注释声称"SQLite 下 ANY 退化为子串匹配,实际测试通过"是错误的(sqlite 无 ANY 函数,运行期报 no such function)
  implication: 次要根因坐实 — 3 处 ANY() 在 sqlite 均报语法错误,需方言条件化
- timestamp: 2026-08-17
  checked: 静态守护测试
  found: reconciliation_tasks_test.go:150-158 要求源码含 "applied_actions IS NULL OR 'no_workorder' != ANY(applied_actions)" 且不含裸 "AND 'no_workorder' != ANY(applied_actions))";reconciliation_service_test.go:27-38 要求源码含 "NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))"
  implication: 修复必须保留 PG 分支字符串原样出现在源码中(作为 PG 分支),sqlite 分支另起字符串

## Eliminated

- hypothesis: 表由某个仍在启动期执行的 MigrateNNN 创建、只是 sqlite 分支被 guard 跳过
  evidence: migration_168 位于 archive/applied/(归档不自动跑),AutoMigrate 注释明确"启动期不再调用 200+ migration 函数";PG/sqlite 都不跑该迁移,PG 靠的是存量
  timestamp: 2026-08-17
- hypothesis: reconciliation_service.go L455-457 注释声称"ANY 在 SQLite 退化为子串匹配,实际测试通过"
  evidence: sqlite 无 ANY 函数,`'silence' = ANY(x)` 运行期报 no such function;该注释为错误论断(单测全走 mock/view,未覆盖真实 ANY 调用)
  timestamp: 2026-08-17

## Resolution

- root_cause: (1) sys_data_reconciliation 仅由已归档 migration_168(archive/applied,启动期不执行)创建,MigrateModelList() 与 sqlite 分支 append 列表均未注册 models.SysDataReconciliation → sqlite 全新文件库永远缺表,PG 存量库因历史迁移有表所以不暴露。(2) scheduler createWorkorderBySeverity 与 ListExceptions silence 过滤共 3 处使用 PG 数组方言 ANY(text[]),sqlite 无 ANY 函数,即使表存在也报语法错误。
- fix: (1) database.go AutoMigrate sqlite 分支 append &models.SysDataReconciliation{}(镜像 260817-hfl-01 UserPreference 模式,PG 零漂移),并将其纳入 sanitizeSQLiteModelDefaults 净化列表(其 tag 含 default:now()/type:text[] PG-only 片段)。(2) reconciliation_tasks.go createWorkorderBySeverity 改方言条件 WHERE:PG 保持 ANY(),sqlite 用 (',' || TRIM(applied_actions,'{}') || ',') NOT LIKE '%,no_workorder,%' 等价判定。(3) reconciliation_service.go 新增 silenceExcludeFilter() 方言条件 helper,替换 2 处 silence 过滤调用点,并修正错误的"ANY 在 sqlite 退化"注释。(4) database_test.go TestNewDatabaseSQLite HasTable 列表新增 sys_data_reconciliation 回归断言。未 git commit,留待用户 review。
- verification: go build ./... 通过;go test ./internal/core/db/... ./internal/scheduler/... ./internal/services/asset/... ./internal/models/... 全部 PASS;TestNewDatabaseSQLite 端到端验证 sqlite 全新库 AutoMigrate 后 HasTable(sys_data_reconciliation)=true;临时 sqlite 运行时测试(已删除)验证方言条件 WHERE 语义:NULL 行与无 no_workorder 行命中、含 no_workorder 行排除;静态守护测试(TestCreateWorkorderNoWorkorderFilterStatic / reconciliation_service_test)PASS,PG 分支 SQL 字符串原样保留。
- files_changed:
  - internal/core/db/database.go (sanitizeSQLiteModelDefaults 净化列表 + AutoMigrate sqlite 分支注册)
  - internal/scheduler/reconciliation_tasks.go (createWorkorderBySeverity 方言条件 WHERE)
  - internal/services/asset/reconciliation_service.go (silenceExcludeFilter helper + 2 处调用点 + 注释修正)
  - internal/core/db/database_test.go (HasTable 回归断言)

## Archive Note

- human_verify: confirmed fixed by user (2026-08-17)
- commit: main b793304 (5 files, +111/-13)
- archived: 2026-08-17 → resolved/
