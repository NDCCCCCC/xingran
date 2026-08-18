---
slug: rpa-tasks-table-missing
status: resolved
trigger: RPA 任务管理页报 "查询失败: SQL logic error: no such table: sys_rpa_tasks"（POST /rpa/tasks/list 400, GORM count 查询失败）；同页签 rpa/workers/worker-1/heartbeat 200 正常。
created: "2026-08-18T01:36:00Z"
updated: "2026-08-18T01:43:00Z"
---

# Debug Session: rpa-tasks-table-missing

## Symptoms

1. **Expected behavior**: SQLite 模式下 RPA 任务管理页正常加载任务列表（`/api/v1/rpa/tasks/list` 200）。
2. **Actual behavior**: `POST /api/v1/rpa/tasks/list` → 400，GORM: `SQL logic error: no such table: sys_rpa_tasks (1)`，SQL 为 `SELECT count(*) FROM sys_rpa_tasks WHERE deleted_at IS NULL AND ...`。前端 useTableManager.ts:246 抛错（调用栈 api.ts:512 → index.tsx:65 → useTableManager.ts:238）。同一时刻 `/rpa/workers/worker-1/heartbeat`（外部 RPA worker 10.62.10.34）200 正常 —— 说明 `sys_rpa_workers` 表存在，仅 `sys_rpa_tasks` 缺。
3. **Error messages**（逐字摘录）:
   - 后端: `ERRO ... [GORM错误] SELECT count(*) FROM `sys_rpa_tasks` WHERE deleted_at IS NULL AND `sys_rpa_tasks`.`deleted_at` IS NULL | 耗时: 544.4µs | 错误: SQL logic error: no such table: sys_rpa_tasks (1)`
   - 后端: `WARN ... Client error ... path=/api/v1/rpa/tasks/list ... status_code=400`（请求体已加密，说明请求解密正常，400 是 service 层 DB 错误被映射为 client error）
   - 前端: `POST http://127.0.0.1:4000/api/v1/rpa/tasks/list 400 (Bad Request)` / `Error: 查询失败: SQL logic error: no such table: sys_rpa_tasks (1)`
4. **Timeline**: 2026-08-18 01:35:32（每次打开 RPA 任务页必现）。背景：刚完成 SM2 密钥持久化到 .env 并重启后端（01:31 启动），重启同时让 kb-tag-table-stats-400 会话的 sys_knowledge_tag 注册生效（已 sqlite3 确认建表）。
5. **Reproduction**: sqlite 模式后端 + 前端，打开 RPA 任务管理页即触发。

## Evidence

- timestamp: 2026-08-18T01:36:00Z — **同族问题第三次**: resolved `ops-sqlite-tables-uuid-cast`(6 表)、`kb-tag-table-stats-400`(sys_knowledge_tag，已建表验证) 之后，`sys_rpa_tasks` 是同一缺漏类——sqlite AutoMigrate 漏注册。项目记忆 xingran-db-migration-and-pk-gotcha 已记录此模式。
- timestamp: 2026-08-18T01:36:00Z — 单一事实源确认: `internal/models/rpa/task.go` `Task.TableName() = "sys_rpa_tasks"`；tasks/list 走 `rpamodels.Task`（task_service.go:23/26/81/138/222 全部 rpamodels.Task）。database.go:652-653 注释明确"单一事实源: rpamodels.Worker/Execution（services/rpa 实际使用；internal/models/rpa.go 的 RPAWorker/RPAExecution 为旧定义）"。
- timestamp: 2026-08-18T01:36:00Z — 注册现状: database.go sqlite 分支 L657-661 仅注册 `&rpamodels.Worker{}` / `&rpamodels.Execution{}` / `&models.MACOUIVendor{}`，**无 rpamodels.Task**。646-649 注释解释 Worker/Execution 由归档 SQL(102_add_rpa_tables.sql) 创建、sqlite 需补注册——但同一归档 SQL 里的 tasks 表漏了。
- timestamp: 2026-08-18T01:36:00Z — 实库实证: `sqlite3 data/xingran.db ".tables"` 仅 `sys_rpa_executions` / `sys_rpa_workers`，无 `sys_rpa_tasks` / `sys_rpa_schedules` / `sys_rpa_variables` / `sys_rpa_templates` —— 与报错完全吻合。
- timestamp: 2026-08-18T01:36:00Z — Task tag 核查（task.go:41-48）: `type:jsonb`(合法 type-name,SysDataReconciliation 先例已验证 sqlite 可接受) / `default:300`/`default:5`/`default:0`(常量默认值合法) / `size`/`type:text` 均合法 —— 无函数式默认值/数组类型等 PG-only DDL 片段，**无需 sanitizeSQLiteModelDefaults**，与已注册 Worker/Execution 同款。
- timestamp: 2026-08-18T01:36:00Z — 其余 3 表范围判定: grep `rpamodels.Schedule/Variable/Template` + `RPASchedule/RPAVariable/RPATemplate` 在 internal/services/ 与 internal/api/ **零命中**（无运行期 service/路由查询）——schedules/variables/templates 不在本症链路，本次不注册（避免无谓漂移）。若未来有页面报错再补。
- timestamp: 2026-08-18T01:36:00Z — 环境限制（沿用前两会话）: Grep/Glob 专用工具失效（claude.exe），用 bash grep；gsd-sdk 不可用(exit 127)。
- timestamp: 2026-08-18T01:36:00Z — 工作区状态: 两个 resolved/awaiting 会话修复未提交（ops-sqlite 8 文件 + kb-tag database.go/api.ts），外加 .env 新增 SM2 密钥行。任何 fix 不得回滚这些改动。operations 8 个 excel_service 失败为既有基线。
- timestamp: 2026-08-18T01:36:00Z — 后端当前以 `go run` 后台任务(b26cwocc3)运行中，修复后需重启生效（AutoMigrate 自动补建 sys_rpa_tasks）。

## Eliminated

（暂无）

## Current Focus

hypothesis: sys_rpa_tasks 缺表 = database.go sqlite 分支漏注册 rpamodels.Task（Worker/Execution 已注册但 Task 漏，归档 SQL 假设仅 PG 成立）。与前两会话同一缺漏类，修复模式已三次验证。
test: 修复后全新 sqlite 库 AutoMigrate 后 HasTable(sys_rpa_tasks)=true；对 data/xingran.db 副本升级路径探针确认自愈；回归 TestNewDatabaseSQLite 加断言。
expecting: 注册后 tasks/list 200。
next_action: 修复已高度明确（sqlite 分支 append 注册 rpamodels.Task + TestNewDatabaseSQLite 断言），沿用 kb-tag 会话的"直接补事实源不依赖 cascade"模式。Tag 已核查无需 sanitize。schedules/variables/templates 不纳入。

## Resolution

root_cause: database.go sqlite 分支注册 `rpamodels.Worker/Execution` 时漏了同一归档 SQL(102_add_rpa_tables.sql) 的 `rpamodels.Task` —— sqlite 全新文件库 AutoMigrate 不建 sys_rpa_tasks,/rpa/tasks/list count 查询报 "no such table"。与 ops-sqlite-tables-uuid-cast / kb-tag-table-stats-400 同一缺漏类(第三次)。
fix: database.go sqlite 分支 append 注册 `&rpamodels.Task{}`(含缺漏原因注释;tag 已核查无需 sanitize);database_test.go TestNewDatabaseSQLite 断言追加 "sys_rpa_tasks"。schedules/variables/templates 不注册(无运行期引用)。
verification:
  - go build ./... PASS
  - go test ./internal/core/db/ PASS(含新断言,全新库 HasTable(sys_rpa_tasks)=true)
  - 升级路径临时探针(实库副本 + AutoMigrate): sys_rpa_tasks 自愈建表,sys_rpa_workers 无损 — PASS,探针已删
cycles: 1 (investigation, orchestrator 预填证据直接确认) + 1 (fix)
tdd: no
specialist: go (无映射 skill,直接修复)
follow_up: 需重启后端(后台任务 b26cwocc3 仍跑旧代码)让 AutoMigrate 在实库补建 sys_rpa_tasks,然后刷新 RPA 任务页确认 /rpa/tasks/list 200。
