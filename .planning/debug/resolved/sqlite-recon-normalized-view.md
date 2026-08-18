---
slug: sqlite-recon-normalized-view
status: resolved
trigger: 上一轮 reconciliation-sqlite-cast-400 修复后，sqlite 运行期每请求输出 WARN "reconciliation_normalized 物化视图缺失,异常列表查询降级"。用户选择：sqlite 模式下 bootstrap 补建 reconciliation_normalized 等价视图，消除降级 WARN，dev 环境与 PG 功能对齐。baseline/compare 400 保持设计内契约不动。
created: "2026-08-18T01:50:00Z"
updated: "2026-08-18T02:05:00Z"
---

# Debug Session: sqlite-recon-normalized-view

## Symptoms

1. **Expected behavior**: sqlite dev 模式下 `/api/v1/asset/reconciliation/exception/list` 走完整 MV 路径（含 rn.physical_username / rn.ad_username 字段），后端日志无降级 WARN。
2. **Actual behavior**: 上轮修复后接口已 200，但 `probeMaterializedView()` 诚实探测 sqlite_master 发现无视图 → 每次查询走 fallback 降级路径并输出 `WARN [reconciliation] reconciliation_normalized 物化视图缺失,异常列表查询降级...`（2026-08-18 01:36:54 实测）。用户确认该 WARN 噪音要消除。
3. **Timeline**: 紧接 resolved 会话 reconciliation-sqlite-cast-400（同日凌晨）。该会话曾明确记录"不动 database.go、不建 sqlite 视图（config 已文档化该 dev 限制）"——本次用户决策**推翻**该边界，要求在 sqlite bootstrap 补建等价 VIEW。
4. **范围确认（用户已选）**: 仅 sqlite 补建视图；baseline/compare 400 保持设计内契约不改。

## Evidence

- timestamp: 2026-08-18T01:50:00Z — 上轮归档结论: reconciliation_normalized 是 PG MATERIALIZED VIEW（归档迁移 168/176 创建，sqlite 分支不执行）；probeMaterializedView() sqlite 分支现查 sqlite_master（type IN table/view），创建普通 VIEW 即可满足探测
- timestamp: 2026-08-18T01:50:00Z — 现有测试基线: TestListExceptionsSQLiteWithView（有视图走 MV 路径）已 PASS，可复用为补建后的验证；TestBootstrapMissingTablesModelDerived（kb 会话加的清单测试）守护 sqlite 注册表，新增视图需确认不破坏其断言
- timestamp: 2026-08-18T01:50:00Z — 环境注意: 子代理 429 限额未重置（05:43:58），本会话全程主上下文执行；database.go/database_test.go 有 kb 会话未提交改动（KnowledgeCategory/KnowledgeTag 注册），在其基础上追加，不得回滚
- timestamp: 2026-08-18T01:50:00Z — 待查: 迁移 168/176 的 MV SELECT 定义（决定 sqlite VIEW 的等价 SELECT）；PG MV 需 REFRESH 而 sqlite VIEW 实时，语义等价或更优

## Eliminated

（暂无）

## Current Focus

hypothesis: 在 database.go sqlite bootstrap 分支以 CREATE VIEW IF NOT EXISTS 补建 reconciliation_normalized（SELECT 等价于迁移 168/176 的 MV 定义，sqlite 方言化），即可让 probe 命中、exception/list 走完整路径、WARN 消除；PG 路径零改动。
test: 读迁移 168/176 MV 定义 → sqlite bootstrap 建视图 → 测试验证视图存在且 ListExceptions MV 路径成功
expecting: sqlite 库 bootstrap 后 sqlite_master 含 reconciliation_normalized VIEW；exception/list 含 rn.* 字段返回；WARN 消失；PG 零改动
next_action: 已完成 — 全部验证通过，会话 resolved

## Resolution

root_cause: |
  非新 bug —— 是上一轮修复的已知遗留：reconciliation 三视图（normalized MV + physical_chain/user_lookup 前置 VIEW）由 PG-only 迁移 168/173/175/176/182 创建，sqlite 分支不执行归档迁移 → 运行期文件库永远无视图 → probe 诚实探测后每请求走 fallback + WARN。用户决策推翻上一轮"不建 sqlite 视图"的边界，要求 bootstrap 补建。
  实施中发现的二级坑：sqlite 子查询扁平化限制 —— 三层嵌套时外层 WHERE 引用窗口函数结果列会解析失败（"no such column"）；且 fmt.Sprintf 共享占位符在 port_norm CTE（无 m 别名）中残留 m.interface_name 限定符。均通过最小复现二分定位（python sqlite3 逐字执行 DDL）。
fix: |
  新增 internal/core/db/sqlite_reconciliation_views.go：
  - ensureSQLiteReconciliationViews()：仅 sqlite 分支生效，DROP（依赖逆序）+ CREATE（依赖正序）三视图，挂载在 AutoMigrate 的 else（sqlite）块尾部（Migrate207 之后），失败非阻断（fallback 兜底）。
  - 方言翻译：DISTINCT ON→ROW_NUMBER 窗口（两层结构规避扁平化坑）；LATERAL→关联标量子查询×3；REGEXP_REPLACE('[.:\-]')→嵌套 REPLACE；normalize_iface()→32 分支 CASE（组内最长前缀优先，与 PG 顺序执行语义逐组核对等价，vlanif⊃vlan 包含关系已处理）；NOW()→CURRENT_TIMESTAMP；::cast 去除；is_enabled=TRUE→=1；MV→普通 VIEW（实时计算，dev 语义更优，无需 REFRESH cron）。
  - database_test.go：TestNewDatabaseSQLite 扩展三视图存在性断言（sqlite_master type='view' 直查）+ reconciliation_normalized 可查询性冒烟。
  - 同步更新 reconciliation_service.go:306 / reconciliation_sqlite_runtime_test.go:18 两处"未来补建"过时注释。
  PG 路径零改动（新函数 d.Type 守卫 + 挂载点在 sqlite-only else 块）。
verification: |
  - go build ./... 通过
  - TestNewDatabaseSQLite PASS（fresh 临时库全量 AutoMigrate → 三视图建出且可 SELECT）
  - go test ./internal/core/db/、./internal/services/asset/ 全包 PASS
  - 真实 dev 库（data/xingran.db 副本）逐字执行 DDL：三视图 CREATE+QUERY OK；MV 路径完整查询（上一轮方言修复版 SELECT）OK
  - 合成数据语义验证：MAC 归一化撮合（'AA:BB:CC:..'↔'aabbcc..'）、接口折叠撮合（'GigabitEthernet0/1'↔'GE0/1'）、nickname+dept 用户查找、双 AD 记录窗口去重、LATERAL 等价子查询取值 —— 全部正确
  - 注意：当前 dev 库 ops_asset 为 0 行，三视图返回空属正确行为
files_changed: |
  internal/core/db/sqlite_reconciliation_views.go（新增）
  internal/core/db/database.go（else 块挂载，+8 行）
  internal/core/db/database_test.go（视图断言 + 冒烟，+27 行）
  internal/services/asset/reconciliation_service.go（注释更新 1 行）
  internal/services/asset/reconciliation_sqlite_runtime_test.go（注释更新 2 行）
notes: |
  - 生效方式：后端下次启动时 AutoMigrate 自动补建（DROP+CREATE 幂等，视图定义随代码版本刷新）。
  - probeMaterializedView 每请求探测 sqlite_master，视图建出后下一个请求即走完整路径，WARN 消失。
  - baseline/compare 400 保持设计内契约（无基线 → 引导 Alert），用户确认不动。
  - 本会话全程主上下文执行（子代理 429 限额未重置）。
