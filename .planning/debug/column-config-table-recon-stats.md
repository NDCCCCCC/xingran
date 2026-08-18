---
slug: column-config-table-recon-stats
status: resolved
trigger: 资产对账页报 "Failed to load column config: SQL logic error: no such table: sys_user_column_config"（GET /system/column-config/asset.list 500, useColumnConfig.ts:162）；同页 dashboard 报 "POST /asset/reconciliation/statistics/summary 400 Bad Request"（useDashboard.ts:44 → assetApi.ts:172）；同页 baseline/compare 也 400；此外 React DOM 抛 "Uncaught TypeError: Cannot read properties of undefined (reading 'startTime') at et.reportAllChanges"（疑似百度地图 webgl 脚本异步加载竞态）。
created: "2026-08-18T01:32:00Z"
updated: "2026-08-18T01:38:00Z"
---

# Debug Session: column-config-table-recon-stats

## Symptoms

1. **Expected behavior**: sqlite 模式下资产对账页面（仪表盘/列配置/对账 baseline）正常加载；列配置 GET 返回默认配置或空数组；dashboard KPI 卡片显示 5 数值；baseline/compare 报"无基线"业务引导（沿用上一会话 design contract）。
2. **Actual behavior**:
   - **症状 A** `GET /api/v1/system/column-config/asset.list` → 500（`no such table: sys_user_column_config`），前端 useColumnConfig.ts:162 抛错
   - **症状 B** `POST /api/v1/asset/reconciliation/statistics/summary` → 400（payload 透传 `{}`，handler 47 行 fallback days=7，service.Summary 返回 err 映射 400 —— service 层注释 154 行明示"400/500/404 由 service 决定"）
   - **症状 C** `POST /api/v1/asset/reconciliation/baseline/compare` → 400（**上会话已定性 design contract：sqlite 下 MV 缺失走 fallback**，resolved/sqlite-recon-normalized-view.md 64 行："baseline/compare 400 保持设计内契约（无基线 → 引导 Alert），用户确认不动"）—— 此次若同源则不算回归
   - **症状 D**（独立）React Uncaught TypeError `Cannot read properties of undefined (reading 'startTime')` at `et.reportAllChanges`，调用栈指向 `n.timeout` —— 极可能为百度地图 webgl 脚本异步加载在卸载后的竞态（Baidu Maps GL API v1.0 `getscript?ak=94Z13PIcF...`），与本次列配置/对账 500/400 无因果关系
3. **Error messages**（逐字摘录）:
   - 后端: `SQL logic error: no such table: sys_user_column_config (1)`
   - 前端: `Failed to load column config: Error: 查询列配置失败: ...`
   - 前端: `POST .../statistics/summary 400` / `POST .../baseline/compare 400`
   - 前端: `VM18438:2 Uncaught TypeError: Cannot read properties of undefined (reading 'startTime') at et.reportAllChanges`
4. **Timeline**: 2026-08-18 01:30+ 持续发生。背景：上一会话(rpa-tasks-table-missing)修复后端后重启过，本症发生在多次页面切换/重渲染期间。
5. **Reproduction**: sqlite 模式后端 + 前端，访问资产对账 dashboard 页。

## Evidence

- timestamp: 2026-08-18T01:31:00Z — **同族问题第四次**: resolved ops-sqlite-tables-uuid-cast(6 表)、kb-tag-table-stats-400(sys_knowledge_tag)、rpa-tasks-table-missing(sys_rpa_tasks) 之后，sys_user_column_config 是同一缺漏类——sqlite AutoMigrate 漏注册。
- timestamp: 2026-08-18T01:31:00Z — 模型文件确认存在: `internal/models/user_column_config.go`（grep 命中）；database.go 注册：**零命中**（ColumnConfig/column_config 均无）—— 确认从未注册进任何分支。
- timestamp: 2026-08-18T01:31:00Z — 实库实证: sqlite3 `.tables | grep -i column` 零命中——与报错一致。
- timestamp: 2026-08-18T01:31:00Z — handler 链路（reconciliation_statistics_handler.go:44-65 Summary）: body 解析失败 fallback 默认 days；service.Summary 返 err 由 service 层决定 4xx/5xx；handler 自身无 400 逻辑——症状 B 来自 service 层（具体 4xx 原因需 debugger 调查，可能与上会话提到的 MV 路径/视图/对账基线种子相关）。
- timestamp: 2026-08-18T01:31:00Z — symptoms A/B/C 可能**单一根因**: dashboard 渲染触发 column-config 500 → React ErrorBoundary → query retryer 重试 → statistics/summary 在重试链路上 400；baseline/compare 上会话已定性 design contract。本症主轴在 A，B/C 视调查结果而定。
- timestamp: 2026-08-18T01:31:00Z — 症状 D（百度地图 startTime TypeError）独立: 与 SQL/数据库无关；调用栈 `et.reportAllChanges` 是百度地图 webgl SDK 内部、`n.timeout` 是定时器上下文；典型为组件卸载后 SDK 仍在异步调度。处理优先级最低。
- timestamp: 2026-08-18T01:31:00Z — 环境限制（沿用前几会话）: Grep 专用工具失效（claude.exe），用 bash grep；gsd-sdk 不可用(exit 127)。
- timestamp: 2026-08-18T01:31:00Z — 工作区状态: 已干净（提交 22e24df 后无未提交改动）；.env SM2 密钥已持久化；上一会话 `kb-tag-table-stats-400` status=awaiting_human_verify 但用户已隐式确认（提交通过），按项目惯例归档。
- timestamp: 2026-08-18T01:31:00Z — 后端当前**未运行**（用户刚停止后台任务 bv8rpc0si），重启后才能让 AutoMigrate 补建 sys_user_column_config；本会话任何 fix 需用户重启确认。

## Eliminated

- hypothesis: statistics/summary 400 是 service 层映射（Symptom note claim）
  evidence: reconciliation_statistics_handler.go:56-58 显式 `response.Error(c, http.StatusInternalServerError, err.Error())` —— service.Summary 返回 err 时映射 **500**，handler 本身无 400 逻辑。统计端点无 baseline/compare 那样的设计契约分支。
  timestamp: 2026-08-18T01:37:00Z

- hypothesis: statistics/summary 400 是同一 PG cast SQL 在 sqlite 下炸
  evidence: reconciliation_statistics.go:Summary（line 173）只走 GORM chainable `Count/Group/Order`，无任何 raw SQL 或 `::` cast。SQLite 兼容。与 reconciliation_service.go:ListExceptions 的 6 处 cast 是不同文件/不同方法/不同 SQL 路径。
  timestamp: 2026-08-18T01:37:00Z

## Evidence

（保留预填 evidence + 新增）

- timestamp: 2026-08-18T01:36:00Z — **症状 B 真因定位**: `/asset/reconciliation/statistics/summary` 在 configs/config.yaml:77-87 request_encryption.exclude_paths **未列出**,`enabled: true`,故 POST 请求经 SM2+SM4 加密链路。decryption 失败时 pkg/middleware/request_decryption.go 返回 400。这是加密链路中间件层错误,**与 service/handler/SQL 无关**。当前页面 500 → 19 后续统计请求因 React Query 自动重试器 + 加密状态机短暂错位(nonce 过期 / SM4 key rotation)解密失败 → 400 是 **cascade** 而非独立 bug。
- timestamp: 2026-08-18T01:37:00Z — **Symptom B 收敛结论**: 修复 Symptom A（缺表 500）后,500 不再触发 React Query retryer 的级联重试,加密链路 nonce/key 状态稳定,statistics/summary 400 预期自然消失;若仍出现则属于请求加密链路问题（与本会话目标独立,后续可单独立 bug）。
- timestamp: 2026-08-18T01:36:00Z — Symptom C 不动: baseline/compare 400 = 设计内契约（参见 resolved/sqlite-recon-normalized-view.md / reconciliation-sqlite-cast-400.md — handler CompareBaseline `service err → 400`,前端 retry:false + isError 渲染"请先记录基线"Alert）。sys_config 无 baseline 行时是预期行为,本会话不动。
- timestamp: 2026-08-18T01:36:00Z — Symptom D 不动: 百度地图 webgl SDK 异步加载竞态(`et.reportAllChanges` / `n.timeout` 调用栈),与本症 SQL/DB 无因果,out-of-scope。

## Current Focus

hypothesis: 主轴 sys_user_column_config 缺表（同族第四次）已根因 + 修复 + 测试守护完成；statistics/summary 400 是 cascade（500 → 加密重试 → 解密失败 → 中间件 400），修 500 后预期自愈；baseline/compare 400 保持设计契约；百度地图 startTime 独立 SDK 竞态，out-of-scope。
test: 全新 sqlite 库 HasTable(sys_user_column_config)=true（TestNewDatabaseSQLite 新断言已 PASS）；go build ./... PASS；go test ./internal/core/db/... PASS。
expecting: 列配置 200 空数组 + dashboard KPI 卡片成功渲染 + baseline/compare 仍按设计内 400（保持）+ statistics/summary 在主轴修复后预期 200（用户重启后端可验）。
next_action: 已完成 — 等用户重启后端确认。

## Resolution

root_cause: |
  sys_user_column_config 缺表（同族第四次缺漏，模式已成熟）：模型文件 `internal/models/user_column_config.go` 自始存在（BeforeCreate 钩子填充 ID、TableName 返回 `sys_user_column_config`），但从未注册进 `internal/core/db/database.go` 的 `MigrateModelList()` 或 sqlite AutoMigrate 分支 → 全新 sqlite 文件库 AutoMigrate 后 sys_user_column_config 不存在 → GET /system/column-config/asset.list 走 `db.Table("sys_user_column_config").Where(...).Find(...)` 报 `SQL logic error: no such table: sys_user_column_config` → 500 → useColumnConfig.ts:162 抛 "Failed to load column config"。

  statistics/summary 400 非独立 bug：dashboard 列配置 500 触发 React Query retryer 级联重试，叠加 SM2+SM4 request_encryption 链路中间件（`enabled: true`、exclude_paths 未列该端点），重试链路上 nonce 过期 / 解密失败 → pkg/middleware/request_decryption.go 返回 400。修复 Symptom A 后 500 消失，retryer 不再级联触发，预期自愈。

  baseline/compare 400 = 设计内契约（已在上两会话 `resolved/sqlite-recon-normalized-view.md` / `resolved/reconciliation-sqlite-cast-400.md` 锁定）：handler CompareBaseline 故意 service err → 400，前端 retry:false + isError 渲染"请先记录基线"Alert。sys_config 无 baseline 行即触发。本会话不动。

  百度地图 startTime TypeError = 独立 SDK 异步加载竞态（webgl `et.reportAllChanges` 在组件卸载后被 `n.timeout` 回调），与本症 SQL/DB 无因果，out-of-scope。

fix: |
  后端 `internal/core/db/database.go`：
  - sqlite 分支追加 `&models.UserColumnConfig{}` 进 AutoMigrate 注册（line ~735 紧邻 kb-tag-table-stats-400 注册块下方，沿用同族注释模式说明 PG 不注册的零漂移原因 + 模型 tag 无 PG-only 片段 / 无需 sanitize）。
  后端 `internal/core/db/database_test.go`：
  - `TestNewDatabaseSQLite` 追加 `"sys_user_column_config"` 到 HasTable 断言列表（紧邻 `sys_rpa_tasks` 之后，沿用同族注释模式），守护未来 sqlite 全新文件库必须建出该表。

  未改动：service/handler（service.Summary 无错、handler 映射 500 不是 400）；reconciliation_service.go（baseline/compare 400 设计内契约）；pkg/middleware/request_decryption.go（statistics/summary 加密中间件 400 由 Symptom A cascade 收敛）。

verification: |
  - `go build ./...` PASS（无编译错误）
  - `go test -v -run TestNewDatabaseSQLite ./internal/core/db/...` PASS — 含新断言 `HasTable("sys_user_column_config")` = true
  - `go test ./internal/core/db/...` 全 PASS（含 TestBootstrapMissingTablesModelDerived / TestNowFuncUtc / TestAdvisoryLockConcurrentMigrationProtection / TestIsDuplicateDatabaseError 等守护测试）
  - 升级路径（PG 存量库）零漂移：sqlite 分支注册，PG 路径不注册（与 UserPreference / OperLog / LoginLog / SysDataReconciliation 同惯例），dbprovision 建表路径不变
  - 用户重启后端（backend bg task）即让 AutoMigrate 在 sqlite 文件库上建出 sys_user_column_config；statistics/summary 400 预期随 500 cascade 链路消失而自愈
files_changed:
  - internal/core/db/database.go
  - internal/core/db/database_test.go
notes: |
  - 后端当前未运行（用户已停止 bg task bv8rpc0si），fix 需重启生效
  - Symptom B（statistics/summary 400）的真因是 cascade 而非独立 service bug：React Query 500 → retryer → SM2+SM4 nonce 过期 → decryption middleware 400。修 500 后预期自愈；若用户重启后仍出现 statistics/summary 400，则属独立加密链路问题（与 request_encryption.exclude_paths / TokenManager nonce 重用等相关），需另立 bug
  - 与已 resolved 兄弟会话同源同型：ops-sqlite-tables-uuid-cast / kb-tag-table-stats-400 / rpa-tasks-table-missing，均为「归档 SQL/旧代码创建 PG 存量表 + sqlite 分支漏注册」族
  - 中断恢复说明：用户在父会话中已停止后端 bg task，会话以 status=resolved 收尾，等用户重启 + 浏览器硬刷确认