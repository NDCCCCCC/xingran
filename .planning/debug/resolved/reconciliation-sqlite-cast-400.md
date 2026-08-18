---
slug: reconciliation-sqlite-cast-400
status: resolved
trigger: 对账模块两接口 400：/api/v1/asset/reconciliation/exception/list 与 /api/v1/asset/reconciliation/baseline/compare（Bad Request）。GORM 错误 "SQL logic error: unrecognized token: \":\""，SQL 含 PG 专用 cast（a.machine_ip::text、a.user_id::uuid）。另有 7 处 antd 废弃属性警告（用户确认一并修复）。
created: "2026-08-18T01:15:00Z"
updated: "2026-08-18T01:45:00Z"
---

# Debug Session: reconciliation-sqlite-cast-400

## Symptoms

1. **Expected behavior**: SQLite 模式（`database.type: "sqlite"`，延续前序会话环境）下，`/api/v1/asset/reconciliation/exception/list` 与 `/api/v1/asset/reconciliation/baseline/compare` 正常返回 200；前端对账相关页面（Dashboard、例外列表、例外规则）正常渲染，控制台无 antd 弃用警告。
2. **Actual behavior**:
   - `/api/v1/asset/reconciliation/exception/list` → 400，GORM 报 `SQL logic error: unrecognized token: ":"`，SQL 为 4 表 join（sys_data_reconciliation + ops_asset a + reconciliation_normalized rn + sys_user ru），select 列含 `COALESCE(a.machine_ip::text, '') AS asset_ip`、join 条件含 `ru.id = a.user_id::uuid` —— 均为 PG 专用 cast 语法，SQLite 不识别 `::`。
   - `/api/v1/asset/reconciliation/baseline/compare` → 400（后端日志该请求只有 `WARN Client error ... status_code=400`，未见对应 GORM 错误详情；时间戳 01:06:19 早于 exception/list 的 GORM 报错 01:06:21，存在日志乱序可能，待确认是否同因）。
   - 同模块 `/api/v1/asset/reconciliation/exception-rule/list` 请求解密成功、正常返回 —— 问题局限于 exception list 与 baseline compare 的 SQL，非整个模块/中间件故障。
   - 次要症状（用户确认纳入范围）：7 处 antd 弃用属性警告：
     a. `ADUserPage index.tsx:527` — Select `onSearch` 无 `showSearch`
     b. `ADOUPage index.tsx:462` — Card `bodyStyle` → `styles.body`
     c. `ADOUPage index.tsx:505` — Space `direction` → `orientation`
     d. `ADConfigPage index.tsx:431` — Drawer `width` → `size`
     e. `Dashboard index.tsx:349` — Alert `message` → `title`
     f. `ExceptionRulesPage index.tsx:350` — Statistic `valueStyle` → `styles.content`
     g. `ExceptionRuleForm.tsx:129` — Modal `destroyOnClose` → `destroyOnHidden`
3. **Error messages**（逐字摘录）:
   - 后端: `ERRO[2026-08-18 01:06:21] [GORM错误] SELECT sys_data_reconciliation.*, a.devicesn AS asset_code, COALESCE(a.machine_ip::text, '') AS asset_ip, rn.physical_username, ru.username AS responsible_username, rn.ad_username FROM \`sys_data_reconciliation\` LEFT JOIN ops_asset a ON a.id = sys_data_reconciliation.asset_id AND a.deleted_at IS NULL LEFT JOIN reconciliation_normalized rn ON rn.asset_id = a.id LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL WHERE sys_data_reconciliation.deleted_at IS NULL AND (',' || TRIM(COALESCE(sys_data_reconciliation.applied_actions, ''), '{}') || ',') NOT LIKE '%,silence,%' AND \`sys_data_reconciliation\`.\`deleted_at\` IS NULL ORDER BY sys_data_reconciliation.detected_at DESC,sys_data_reconciliation.detected_at DESC LIMIT 20 | 耗时: 0s | 错误: SQL logic error: unrecognized token: ":" (1)`（01:06:21 与 01:06:22 两次）
   - 后端: `WARN[...] Client error ... method=POST path=/api/v1/asset/reconciliation/exception/list ... status_code=400`；`WARN[...] Client error ... method=POST path=/api/v1/asset/reconciliation/baseline/compare ... status_code=400 latency=11`
   - 前端: `POST http://127.0.0.1:4000/api/v1/asset/reconciliation/baseline/compare 400 (Bad Request)`（assetApi.ts:380 → index.tsx:85 queryFn）；`POST .../exception/list 400`（assetApi.ts:232 → useExceptionList.ts:47，React Query 自动重试后仍 400）
4. **Timeline**: 2026-08-18 01:06:19-23 持续发生（每次页面加载必现）。关键背景链：① resolved 会话 `ops-sqlite-tables-uuid-cast`（commit ec4ca04 "fix(ops): sqlite mode missing tables and pg-only cast syntax"）修了 **operations 6 个 service** 的同类 PG cast / sqlite 缺表问题，但修复范围未覆盖 reconciliation/asset 模块 → 本次是其同类遗漏；② 上一会话 `kb-tag-table-stats-400`（investigating，01:06 前后）处理 knowledge_tag 缺表 + building statistics 400，与本症无关但共享 sqlite 环境。
5. **Reproduction**: sqlite 模式后端 + 前端 dev server（127.0.0.1:4000 代理），打开对账 Dashboard / 例外列表页即触发两接口 400；antd 警告在 AD 域三页与对账三页渲染时于控制台输出。
6. **范围确认（用户已选）**: 主攻两接口 400，root cause 确认后顺手修复 7 处 antd 弃用属性。goal = find_and_fix。

## Evidence

- timestamp: 2026-08-18T01:15:00Z — orchestrator 预查（症状分析）: 报错 SQL 的两个 PG 专有片段定位明确 —— select 列 `COALESCE(a.machine_ip::text, '')`（ops_asset.machine_ip 可能是 PG inet/自定义类型列，PG 下需 ::text 归一化）与 join 条件 `ru.id = a.user_id::uuid`（ops_asset.user_id 可能是 text 列存 UUID，PG 下需 cast 才能 join uuid 主键）。SQLite 均不支持 `::` cast 语法（SQLite 用 CAST(x AS type) 且无 inet/uuid 类型）。推测正确修法与 ec4pa04 同款：按 `db.Dialector.Name()` 分支或用 `CAST(... AS TEXT)` 等双方言安全写法。
- timestamp: 2026-08-18T01:15:00Z — 出错 SQL 的生成代码位置待定位：涉及表 sys_data_reconciliation / reconciliation_normalized（对账专有表），推测在 internal/services/ 下 reconciliation 或 asset 相关 service 的 ListExceptions 实现（前端调用栈 assetApi.ts:232/380）。
- timestamp: 2026-08-18T01:15:00Z — baseline/compare 400 无 GORM 错误日志：需 debugger 确认（a）日志乱序其实有 GORM 错误（b）该 handler 在执行 SQL 前失败（参数绑定/校验，如缺少 baseline_id 之类必填参数，或前置查询同样炸但被吞）（c）加密响应中间件差异。注意 exception-rule/list 正常说明加密链路本身无恙。
- timestamp: 2026-08-18T01:15:00Z — 参考先例: commit ec4ca04（fix(ops): sqlite mode missing tables and pg-only cast syntax）是本症同类修复的模板；上一会话还在 internal/core/db/database_test.go 加过"清单测试"守护 sqlite 注册表。
- timestamp: 2026-08-18T01:15:00Z — 环境限制（沿用前序会话记录）: 专用 Grep/Glob 工具可能报 "Executable not found"（claude.exe 路径失效），需用 bash grep 替代；gsd-sdk 不可用（exit 127，本次已复验）。
- timestamp: 2026-08-18T01:15:00Z — 工作区状态: `internal/core/db/tmp_probe_cascade_test.go` 为上一会话残留探针（其记录称"用后即删"未删，本次收尾时处理）；`.planning/STATE.md` 有未提交修改；前序会话的修复已随 ec4ca04 提交。本会话 fix 不得回滚 ec4ca04 内容。
- timestamp: 2026-08-18T01:15:00Z — antd 警告修复映射（antd 6.1）: Card bodyStyle→styles.body；Space direction→orientation；Drawer width→size（注意 size 只接受 large/default，自定义数值宽度需保留 width 的话警告可能无法完全消除——需按 antd 6 实际 API 处理）；Alert message→title；Statistic valueStyle→styles.content；Modal destroyOnClose→destroyOnHidden；Select onSearch→加 showSearch。涉及文件：AD 域 pages/ad-domain/{users,ou,config}/index.tsx + 对账 Dashboard/ExceptionRules/ExceptionRuleForm（前端 src/pages 下，具体路径待 glob）。

## Eliminated

- hypothesis: baseline/compare 400 与 exception/list 同因（PG cast SQL 在 sqlite 炸）
  evidence: reconciliation_baseline.go 全文无任何 :: cast；Compare 只做 sys_config Pluck + 3 个纯 GORM COUNT。直接查 data/xingran.db：sys_config 无 asset.reconciliation.baseline 行（baseline rows: NONE）。handler CompareBaseline 设计即"任何 service error → 400"（前端依赖 400 显示引导 Alert，见 dashboard/index.tsx:86 retry:false + isError 分支渲染"请先到例外规则管理页记录基线"）。后端日志无 GORM 错误与此吻合（查询成功、0 行、业务 error）。既有测试 TestCompareNoBaselineReturnsError 锁定该行为。
  timestamp: 2026-08-18T01:40:00Z

## Evidence

- timestamp: 2026-08-18T01:15:00Z — orchestrator 预查（症状分析）: 报错 SQL 的两个 PG 专有片段定位明确 —— select 列 `COALESCE(a.machine_ip::text, '')`（ops_asset.machine_ip 可能是 PG inet/自定义类型列，PG 下需 ::text 归一化）与 join 条件 `ru.id = a.user_id::uuid`（ops_asset.user_id 可能是 text 列存 UUID，PG 下需 cast 才能 join uuid 主键）。SQLite 均不支持 `::` cast 语法（SQLite 用 CAST(x AS type) 且无 inet/uuid 类型）。推测正确修法与 ec4ca04 同款：按 `db.Dialector.Name()` 分支或用 `CAST(... AS TEXT)` 等双方言安全写法。
- timestamp: 2026-08-18T01:15:00Z — 出错 SQL 的生成代码位置待定位：涉及表 sys_data_reconciliation / reconciliation_normalized（对账专有表），推测在 internal/services/ 下 reconciliation 或 asset 相关 service 的 ListExceptions 实现（前端调用栈 assetApi.ts:232/380）。
- timestamp: 2026-08-18T01:15:00Z — baseline/compare 400 无 GORM 错误日志：需 debugger 确认（a）日志乱序其实有 GORM 错误（b）该 handler 在执行 SQL 前失败（参数绑定/校验，如缺少 baseline_id 之类必填参数，或前置查询同样炸但被吞）（c）加密响应中间件差异。注意 exception-rule/list 正常说明加密链路本身无恙。
- timestamp: 2026-08-18T01:15:00Z — 参考先例: commit ec4ca04（fix(ops): sqlite mode missing tables and pg-only cast syntax）是本症同类修复的模板；上一会话还在 internal/core/db/database_test.go 加过"清单测试"守护 sqlite 注册表。
- timestamp: 2026-08-18T01:15:00Z — 环境限制（沿用前序会话记录）: 专用 Grep/Glob 工具可能报 "Executable not found"（claude.exe 路径失效），需用 bash grep 替代；gsd-sdk 不可用（exit 127，本次已复验）。（本次更新：专用 Grep 工具在本会话可用。）
- timestamp: 2026-08-18T01:15:00Z — 工作区状态: `internal/core/db/tmp_probe_cascade_test.go` 为上一会话残留探针（其记录称"用后即删"未删，本次收尾时处理）；`.planning/STATE.md` 有未提交修改；前序会话的修复已随 ec4ca04 提交。本会话 fix 不得回滚 ec4ca04 内容。
- timestamp: 2026-08-18T01:15:00Z — antd 警告修复映射（antd 6.1）: Card bodyStyle→styles.body；Space direction→orientation；Drawer width→size（注意 size 只接受 large/default，自定义数值宽度需保留 width 的话警告可能无法完全消除——需按 antd 6 实际 API 处理）；Alert message→title；Statistic valueStyle→styles.content；Modal destroyOnClose→destroyOnHidden；Select onSearch→加 showSearch。涉及文件：AD 域 pages/ad-domain/{users,ou,config}/index.tsx + 对账 Dashboard/ExceptionRules/ExceptionRuleForm（前端 src/pages 下，具体路径待 glob）。
- timestamp: 2026-08-18T01:40:00Z — SQL 生成代码定位: internal/services/asset/reconciliation_service.go。PG cast 共 6 处（行号基于当前文件）: ①exceptionListJoinSelect const L368 `COALESCE(a.machine_ip::text,'')` ②exceptionListJoinClause const L386 `ru.id = a.user_id::uuid` ③exceptionListJoinSelectFallback const L395+L396+L398（machine_ip::text + ''::text ×2） ④exceptionListJoinClauseFallback const L407 `a.user_id::uuid` ⑤ListExceptions MV 路径内联 Joins L499 `a.user_id::uuid` ⑥同文件同类: computeByWorkstation L793 `machine_ip::text` + L794 `a.id::text`、fetchWorkstationDeviceIPs L1025 `sys_network_device.id::text`（by-workstation 端点，前端 useWorkstationHealth 调用，sqlite 下同样会炸 — 同文件同类遗漏）。
- timestamp: 2026-08-18T01:40:00Z — 修复模式双先例: (a) ec4ca04 用双方言中立 CAST(x AS TEXT)；(b) 本模块 fix_suggestion_service.go userJoinOn()（2026-08-17 修，同类 sqlite 炸点）用方言分支方法 — PG 保留原 `::text`（零改动），sqlite 走纯等值。同文件 silenceExcludeFilter() 亦是方言分支方法。选 (b)：PG 侧字面量零改动（不破坏 sys_user.id 索引使用），与本模块最新先例一致。
- timestamp: 2026-08-18T01:40:00Z — 第二层根因（关键）: 直接查 data/xingran.db sqlite_master —— reconciliation_normalized 不存在（其余 7 表 ops_asset/sys_user/sys_data_reconciliation/ops_workstation_device/ops_info_points/sys_network_device/sys_config 均在）。probeMaterializedView() 对非 postgres 硬编码 return true（注释假设"SQLite 测试用 view 模拟,setupTestDB 建 view"——单测成立、运行期文件库不成立，config.yaml 注释亦载明 sqlite 下 MV/视图迁移不执行）。=> 只修 cast 会把错误从 "unrecognized token ':'" 变成 "no such table: reconciliation_normalized"。正确修法: sqlite 分支诚实探测 sqlite_master，缺失走 ListExceptions 已有的设计内 fallback 路径（Warnf + 降级 SELECT 无 rn.* 列）。
- timestamp: 2026-08-18T01:40:00Z — Count 阶段在 sqlite 已通过（后端日志只见 Find SQL 报错，Count 未报错）→ silenceExcludeFilter 方言分支已生效，佐证 Join(ops_asset) 无 cast 部分无恙。

## Current Focus

reasoning_checkpoint:
  hypothesis: "exception/list 400 双层根因: (1) ListExceptions 的 SELECT/JOIN 片段硬编码 PG ::cast（sqlite 词法错误 unrecognized token ':'）；(2) probeMaterializedView 对 sqlite 硬编码 return true，但运行期 sqlite 文件库无 reconciliation_normalized（MV 由 PG-only 迁移 168/176 创建，sqlite 分支不执行）→ 即使修了 cast，MV 路径 JOIN 仍报 no such table。baseline/compare 400 是设计内行为（sys_config 无 baseline 行 → service error → handler 400 → 前端引导 Alert），非 bug。"
  confirming_evidence:
    - "后端 GORM 错误日志逐字含 COALESCE(a.machine_ip::text,'') 与 ru.id = a.user_id::uuid，错误 'SQL logic error: unrecognized token: \":\"'（sqlite 词法层不识别 ::）"
    - "python 直查 data/xingran.db: sqlite_master 无 reconciliation_normalized（7 张依赖表齐全）"
    - "probeMaterializedView() L305-308: 非 postgres 无条件 return true"
    - "reconciliation_baseline.go 无任何 cast；直查 sys_config 无 baseline 行；handler 注释+前端 retry:false/isError Alert 证明 400 是契约"
    - "本模块先例 fix_suggestion_service.go userJoinOn() 正是同症状（su.id::text sqlite 炸）的方言分支修法"
  falsification_test: "修复后用运行期同构 sqlite 库（有表、无 view）直跑 ListExceptions 应成功返回（若仍失败则根因判断有误）；有 view 的库（模拟单测/未来补建）也应成功；baseline/compare 在写入 baseline 行后应 200"
  fix_rationale: "方言分支方法（PG 串字面零改动 / sqlite 串纯等值+CAST 中立写法）消除词法错误；probe sqlite 分支改查 sqlite_master 使缺 view 时走设计内 fallback 路径（该路径本就为 MV 缺失场景设计，含降级 SELECT 与 Warnf 提示）。不动 database.go、不建 sqlite 视图（config 已文档化该 dev 限制）、不动 PG 行为"
  blind_spots: "未实际起后端+登录态打真 HTTP 请求（需 auth+加密链路，代价高）——用服务层 sqlite 直跑测试近似验证；by-workstation 的 3 处 cast 修复属同文件同类顺手修复（超两端点字面范围但同模块同类，与用户『ec4ca04 同类遗漏』框架一致）；antd Drawer width→size 的实际 6.x API 需查 node_modules 确认"

hypothesis: (已确认，见 Resolution)
test: 先写红测（运行期同构 sqlite 库跑 ListExceptions 期望无错）→ 修 reconciliation_service.go（方言分支 6 处 + probe sqlite 诚实探测）→ 绿测 + 全包回归 + go build
expecting: sqlite 无 view 库走 fallback 返回 200 数据；有 view 库走 MV 路径正常；PG 路径 SQL 字面零改动；baseline/compare 无需修（设计内）
next_action: 已完成 — 全部验证通过，会话 resolved

## Resolution

root_cause: |
  exception/list 400 为双层根因：
  (1) reconciliation_service.go 的 ListExceptions 查询（MV 路径 + fallback 路径）SELECT/JOIN 片段硬编码 PG 专用 cast —— `a.machine_ip::text`、`ru.id = a.user_id::uuid`、`''::text`。SQLite 词法层不识别 `::`，报 "SQL logic error: unrecognized token: ':'" → handler 400。同文件 computeByWorkstation / fetchWorkstationDeviceIPs 另有 3 处同类 cast（by-workstation 端点潜在炸点）。
  (2) probeMaterializedView() 对非 postgres 硬编码 return true（基于"单测 setupTestDB 建 view"假设），但运行期 sqlite 文件库无 reconciliation_normalized（MV 由 PG-only 归档迁移 168/176 创建，sqlite 分支不执行）。只修 cast 会把错误变成 "no such table: reconciliation_normalized"。
  baseline/compare 400 非 bug：sys_config 无 baseline 行 → service 返回业务 error → handler 按契约返回 400 → 前端依赖该 400 渲染"请先记录基线"引导 Alert（retry:false + isError 分支；既有测试 TestCompareNoBaselineReturnsError 锁定）。
fix: |
  后端 reconciliation_service.go：
  - 4 个方言感知方法（exceptionListJoinSelectDialect / exceptionListResponsibleUserJoinDialect / exceptionListJoinSelectFallbackDialect / exceptionListJoinClauseFallbackDialect）：postgres 返回原 const（字面零改动，不影响 sys_user.id 索引使用与 PG 语义）；sqlite 返回 CAST(x AS TEXT) 中立写法 + 纯等值 join。模式与同模块 fix_suggestion_service.go userJoinOn() 先例一致。
  - probeMaterializedView() sqlite 分支改为诚实探测 sqlite_master（type IN table/view AND name='reconciliation_normalized'），缺失 → false，走 ListExceptions 设计内 fallback 降级路径（Warnf 提示执行 migration_168）。
  - computeByWorkstation / fetchWorkstationDeviceIPs 的 3 处 cast 同样方言分支化（PG 字面零改动）。
  前端 7 处 antd 弃用属性：
  - ExceptionRuleForm.tsx: destroyOnClose→destroyOnHidden
  - ad-domain/configs: Drawer width={1100}→size={1100}（antd 6 size 接受 number，已核 d.ts）+ destroyOnClose→destroyOnHidden
  - ad-domain/ous: Card bodyStyle→styles.body ×2、Space direction→orientation ×3、删除 Select 空 onSearch
  - ad-domain/users: 删除两处 Select 空 onSearch（noop，无需 showSearch）
  - reconciliation/dashboard: Alert message→title ×2、Statistic valueStyle→styles.content ×3
  - reconciliation/exception-rules: Statistic valueStyle→styles.content ×2
verification: |
  - 新增回归测试 reconciliation_sqlite_runtime_test.go：TestListExceptionsSQLiteRuntimeNoView（无视图运行期同构库 → fallback 成功返回 + Warnf）、TestProbeMaterializedViewSQLiteHonest、TestListExceptionsSQLiteWithView（有视图 → MV 路径）全 PASS
  - go test ./internal/services/asset/ 全包 PASS（含既有 TestListExceptionsSilenceFilter 等）
  - go test ./internal/api/v1/ PASS
  - go build ./... 通过
  - 前端 npm run type-check (tsc --noEmit) 通过
  - 残留检查：grep "::text|::uuid" 仅剩 PG-only probe（to_regclass 仅 postgres 路径执行）与 PG 规范 const（刻意保留）
files_changed: |
  internal/services/asset/reconciliation_service.go（方言分支 + probe 修复）
  internal/services/asset/reconciliation_sqlite_runtime_test.go（新增，3 个回归测试）
  xingran-react-frontend/src/components/asset/reconciliation/ExceptionRuleForm.tsx
  xingran-react-frontend/src/pages/ad-domain/configs/index.tsx
  xingran-react-frontend/src/pages/ad-domain/ous/index.tsx
  xingran-react-frontend/src/pages/ad-domain/users/index.tsx
  xingran-react-frontend/src/pages/asset/reconciliation/dashboard/index.tsx
  xingran-react-frontend/src/pages/asset/reconciliation/exception-rules/index.tsx
notes: |
  - 未改动：internal/core/db/database.go / database_test.go / src/lib/api.ts 属上一会话 kb-tag-table-stats-400 的未提交修复（knowledge_tag 注册 + SM2 解密重放），与本会话无关，保持原样。
  - 残留探针 internal/core/db/tmp_probe_cascade_test.go 已删除（上一会话"用后即删"遗留）。
  - 中断恢复说明：session manager 因子代理 429 中断，orchestrator 主上下文接手完成验证与归档。
  - 后续建议（可选）：sqlite 运行期如需 rn.* 字段（physical_username/ad_username），执行 migration_168 等价视图创建后重启；生产 PG 不受影响。
