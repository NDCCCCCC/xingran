---
status: resolved
trigger: |
  SQLite 切换后（quick task 260817-hfl），go run ./cmd/main.go 启动日志出现 4 类 ERRO：
  1) CREATE TABLE ... PARTITION OF sys_device_mac_history → near "PARTITION": syntax error（×3 个月分区）
  2) Phase 42 R1 startup RefreshView failed: 设置 statement_timeout 失败 → near "SET": syntax error
  3) no such table: sys_rpa_executions / sys_rpa_workers（RPA 扩缩容统计查询）
  4) no such table: sys_mac_oui_vendor（OUI 厂商数据导入）
  应用最终启动成功（:9000）并优雅关闭，均为非阻塞错误。
  第 2 轮验证(17:27): admin 登录成功但 recordLoginLog 报 no such table: sys_logininfor
  第 3 轮验证(17:47): 对账 cron monitorFixSuggestionMisFix → fix-suggestion Stats 报
    SQL logic error: near "'1 day'": syntax error（NOW() - 7 * INTERVAL '1 day' PG 方言）
created: 2026-08-17
updated: 2026-08-17
---

# Debug: sqlite-startup-pg-only-errors

## Symptoms

- **Expected**: SQLite 模式下启动日志无 ERRO；PG-only 特性（分区表、statement_timeout、物化视图）按 `database.type` 守卫跳过或降级
- **Actual**: 4 类 ERRO（见 trigger），功能降级但进程存活
- **Reproduction**: `configs/config.yaml` 设 `database.type: sqlite`，删除 `data/xingran.db`，`go run ./cmd/main.go`
- **Timeline**: 始于 quick task 260817-hfl（Supabase PG → 本地 SQLite 切换）之后的首次完整启动

## Context (from quick-260817-hfl)

- 驱动: glebarez/sqlite v1.11.0 (modernc.org/sqlite, 纯 Go)
- PG-only 守卫模式已存在: `if d.Type == "postgres"` blocks in `internal/core/db/database.go`（cleanupOldConstraints / dropDependentMaterializedViews / advisory lock / migrations 175/176/202-206）
- sys_rpa_workers / sys_rpa_executions / sys_mac_oui_vendor 是归档 SQL（不执行）创建的表，已记录在 `.planning/quick/260817-hfl-supabase-postgresql-sqlite-go-modernc-or/deferred-items.md`
- sys_user_preference 同类问题已通过"模型迁入 internal/models + sqlite 分支 AutoMigrate 注册"修复（commit d2a0cdb），可作为修复范式

## Current Focus

hypothesis: CONFIRMED — 第 3 轮新发现为"PG-only SQL 方言在 sqlite 执行"类(非缺表):fix_suggestion_service.go Stats 6 个 Count 用 `NOW() - ? * INTERVAL '1 day'`,同文件另有 3 处 `su.id::text` JOIN(List/GetByID/history)、Apply 的 `NOW() + INTERVAL '7 day'`、Rollback 的 `> NOW()` DB-side 检查;另有 checkPortStatusDrift cron 的 `::text` cast 与 GenerateFixSuggestions 对 PG-only MV reconciliation_normalized 的查询
reasoning_checkpoint:
  hypothesis: "SQLite(glebarez/modernc)不支持 NOW()/INTERVAL/::cast/gen_random_uuid() —— 临时探针实测(已删): ::text→unrecognized token ':'; NOW()→no such function; gen_random_uuid()→no such function; INTERVAL→no such column; CURRENT_TIMESTAMP 双方言可用且 UTC(与 CDX-M-UTC NowFunc 对齐)。对账 monitor cron(monitorFixSuggestionMisFix,每 10 分钟)触发 Stats → 报用户所贴错误;非阻塞(WARN 忽略)"
  confirming_evidence:
    - "用户真实环境日志(2026-08-17 17:47): Stats pending Count 精确报 near \"'1 day'\": syntax error,任务 WARN 忽略后继续完成"
    - "临时探针 TestTmpDialectProbe(全新 sqlite in-memory)实测 6 种语法行为,与用户错误完全一致;验证后已删除"
    - "代码库已有方言分支范式: statsTrend(date_trunc/strftime)、TopUnresolved(EXTRACT/julianday)、computeHealthTrend、probeMaterializedView(非 PG 早退)"
  falsification_test: "修复后全新/已有 sqlite 库触发 Stats(cron 或直调 service):6 Count + trend 全成功零 ERRO;TestFixSuggestionStatsWindow 在 sqlite 上由'容错失败'转为断言成功"
  fix_rationale: "Stats 6 Count 加方言分支(PG 保留 DB-side INTERVAL 原文 → 静态测试 TestFixSuggestionStatsUsesAppliedAtFilter/DBIntervalUsed 不破;sqlite 用应用层 cutoff 参数 —— in-process 部署 app 与 DB 同一时钟,W-3 时钟漂移论据不适用);::text JOIN 抽 userJoinOn() 方言 helper;Apply 抽 rollbackWindowValue() helper(PG 保留 INTERVAL '7 day' 原文,Apply 函数体不出现 .Add(7*24*time.Hour) → 静态测试不破);Rollback DB-side 检查仅 PG 执行(sqlite 单时钟,step 2 Go-side 判定即权威);checkPortStatusDrift 加 sqlite 变体(去 ::text,PG 原文不动);GenerateFixSuggestions 非 PG 早退(物理链路 MV 为 PG-only,sqlite 永无可生成建议,避免每 5min 'no such table' WARN)"
  blind_spots: "operations 模块(workstation/floor/infopoint/server_room/room_device/excel_query_builder)与 reconciliation_service(ListExceptions COALESCE machine_ip::text、a.user_id::uuid JOIN 等)存在大范围 ::cast —— 属同一方言类但超出本次对账/monitor 范围,用户未报,记录为 deferred;rpa worker Register(gen_random_uuid+NOW)、floor 软删恢复(NOW)、AD assignRole(NOW ×2)同理 deferred;login_log_service.go:172 Clean 删 sys_login_log 与模型表 sys_logininfor 不一致(双方言预存 bug,仅记录)"
next_action: 已完成 —— 用户 2026-08-17 真实环境最终验证确认: sys_logininfor 登录日志写入正常、monitorFixSuggestionMisFix cron 不再报 INTERVAL 方言错误、此前全部缺表/方言问题均不再出现: 1) sys_logininfor —— 重启后端后 admin 登录,确认无 "no such table: sys_logininfor" 且有写入行; 2) INTERVAL 方言错误 —— 重启后端后等 monitorFixSuggestionMisFix cron 下次触发(每小时 7/17/27/37/47/57 分),确认无 "near \"'1 day'\": syntax error",或直接打开前端"资产对账-修复建议"页面(触发 List/Stats)验证列表与统计卡片正常

## Resolution

root_cause: |
  4 类启动 ERRO 均为"PG-only 逻辑在 sqlite 分支无守卫/缺表":
  1) partitionServiceImpl.EnsurePartitionsExist/DropExpiredPartitions 无条件执行 PARTITION OF DDL 与 pg_inherits 目录查询(PG 专属)
  2) RefreshView 无条件 SET statement_timeout + REFRESH MATERIALIZED VIEW CONCURRENTLY;MV 由 PG-only migration_176 创建,sqlite 下不存在
  3) sys_rpa_workers/sys_rpa_executions 模型(internal/models/rpa 包,单一事实源)未注册进 sqlite 分支 AutoMigrate(历史上由归档 SQL 建表)
  4) sys_mac_oui_vendor 模型(internal/models.MACOUIVendor)同理未注册
  验证期第 3 轮(17:47)新根因类 —— "PG-only SQL 方言在 sqlite 运行期执行"(临时探针实测 glebarez/modernc:NOW()/INTERVAL/::cast 全部报错):
  5) fix_suggestion_service.go Stats 6 个 Count 用 `NOW() - ? * INTERVAL '1 day'`(monitorFixSuggestionMisFix cron 触发,即用户所报 near "'1 day'" 错误)
  6) 同文件 3 处 `su.id::text` JOIN(List/GetByID/history)、Apply `NOW() + INTERVAL '7 day'`、Rollback `rollback_window_until > NOW()`
  7) fix_suggestion_generator.go 对 PG-only MV reconciliation_normalized 逐候选查询(候选存在时每 5min WARN)
  8) reconciliation_tasks.go checkPortStatusDrift 4 处 `::text` cast(每日 cron)
fix: |
  1) mac_history_partition.go: 新增 isPostgres() 方言守卫(Dialector.Name()=="postgres"),CreateMonthlyPartition/EnsurePartitionsExist/DropExpiredPartitions 在非 PG 下早退 return nil + Debugf 日志 —— 一处守卫同时覆盖启动与每日 cron 路径
  2) reconciliation_snapshot.go RefreshView 入口:非 postgres 方言早退 return nil + logrus Debug —— 覆盖 startup/cron/handler 三路径
  3) database.go AutoMigrate sqlite 分支:按 d2a0cdb(UserPreference)范式追加注册 &rpamodels.Worker{}、&rpamodels.Execution{}、&models.MACOUIVendor{}(三个模型 tag 无 PG-only DDL 片段,无需 sanitize);同步从 deferred-items.md 移除两条已解决项
  4) 用户决策追加:同范式注册 &models.OperLog{}(sys_oper_log 历史由归档 SQL 创建;tag 已核查无 '(' 函数默认/'[' 数组类型,BaseTimeLine 的 type:uuid 为合法 type-name,ID 由 BeforeCreate 钩子填充,无需 sanitize)
  5) 验证期第 2 轮新发现追加:同范式注册 &models.LoginLog{}(sys_logininfor 历史由归档 SQL 创建,dbprovision 已注册但运行期 AutoMigrate 缺漏;tag 已核查无 PG-only 片段,与 OperLog 同一 BaseTimeLine);database_test.go TestNewDatabaseSQLite 断言表清单追加 5 张修复表作为常驻回归守护
  6) 验证期第 3 轮(方言类,2026-08-17 18:xx):
     a) fix_suggestion_service.go Stats 加 Dialector.Name() 分支:PG 保留 6 条 DB-side INTERVAL 查询原文(W-3 时钟漂移论据仍适用于 C/S 部署,静态回归测试断言原文);sqlite 用应用层 cutoff 参数传入(in-process 同一时钟,与 reconciliation_tasks.go now 参数化同范式)
     b) 新增 userJoinOn() 方言 helper(PG 保留 su.id::text 原文/sqlite 直接等值),替换 List/GetByID/history 3 处 Joins
     c) 新增 rollbackWindowValue(now) helper(PG 返回 gorm.Expr("NOW() + INTERVAL '7 day'") 原文/sqlite 返回 now+7d),Apply 改调 helper —— helper 独立于 Apply 函数体,W-3 静态守护(TestFixSuggestionDBIntervalUsed)保持绿色
     d) Rollback step 5 DB-side 窗口检查仅 postgres 执行(sqlite 无 NOW();in-process 单时钟下 step 2 Go-side 判定即权威)
     e) fix_suggestion_generator.go GenerateFixSuggestions 非 PG 早退(Debugf + return 0,nil)—— 建议生成依赖 PG-only 物理链路 MV,sqlite 永无可生成建议
     f) reconciliation_tasks.go checkPortStatusDrift 加方言分支:PG SQL 原文不动,sqlite 变体去除 4 处 ::text
     g) 测试更新:TestFixSuggestionStatsWindow 由"容错失败"强化为 sqlite 成功断言(Applied=10/RolledBack=1/MisFixRate=0.1/ThresholdBreached=true);List/audit 测试过时注释更新 ×3
     h) 非对账族 5 类方言点(rpa Register/floor 恢复/AD assignRole/login_log Clean 错表/operations 大范围 ::cast)记入 deferred-items.md
  PG 生产路径零改动(所有守卫仅在非 postgres 方言触发;sqlite 分支注册不进 PG migrateList;PG 分支 SQL 逐字保留)
verification: |
  - go build ./... 通过
  - go test: internal/services/asset、internal/services/rpa、internal/core/db 全 PASS;internal/services 的 8 个失败用例(TestCollectBoardsInto_*/TestCollectDeviceInfo_*/TestNormalizeMACAddress)经 stash 对照证实为 HEAD 预存在失败,与本次修改无关
  - 全新 sqlite 文件库实际启动验证:PARTITION OF/statement_timeout/no such table(sys_rpa_*/sys_mac_oui_vendor) 4 类 ERRO 零出现;启动日志显示 "MAC历史分区管理服务初始化完成"、"Phase 42 R1 startup RefreshView succeeded"、"Imported 19 OUI vendors";服务正常监听 :9000
  - 取舍记录:MAC 分区在 sqlite 下选择"跳过"而非"退化为普通单表" —— sys_device_mac_history 本身不在 MigrateModelList(PG 分区表由 SQL 迁移创建),sqlite 下该表不存在属 deferred item #4 既定范畴(按需后续修);跳过分区管理是与 PG 差异最小、改动最小的方案
  - sys_oper_log 修复自验(2026-08-17):go build ./... 通过;go test internal/core/db、internal/utils/operlog、internal/models 全 PASS;全新 sqlite 文件库启动,日志零 ERRO/FATA/PARTITION/statement_timeout/no such table;外部 RPA worker 心跳(10.62.10.34,此前触发该错误的路径)status 200;sqlite_master 确认 sys_oper_log 存在(19 列与 model 一致)且已写入 3 行;服务正常监听 :9000
  - sys_logininfor 修复自验(2026-08-17 17:41):go build ./... 通过;TestNewDatabaseSQLite 端到端(全新 sqlite 文件库 AutoMigrate)PASS,HasTable 断言覆盖 sys_rpa_workers/sys_rpa_executions/sys_mac_oui_vendor/sys_oper_log/sys_logininfor 5 张修复表;一次性临时测试复刻 auth.go recordLoginLog 写入路径(Username/IPAddr/Status/LoginTime INSERT)PASS —— ID 由 BeforeCreate 钩子填充、count=1(临时测试验证后已删除);internal/core/db 全量测试 PASS。注:未动用户正在运行的 main.exe(PID 40288)及其 DB 文件,真实环境验证留给用户
  - 用户最终验证(2026-08-17, 确认): 重启后端后真实环境三项全过 —— 1) admin 登录无 "no such table: sys_logininfor" 且登录日志有写入行; 2) monitorFixSuggestionMisFix cron 触发无 "near \"'1 day'\": syntax error"; 3) 此前全部缺表/方言 ERRO 均不再出现。会话闭环,归档 resolved/
  - 方言类修复自验(2026-08-17 18:13):go build ./... 通过;go test internal/services/asset、internal/scheduler、internal/core/db、internal/api/v1/asset 全 PASS(含 W-2/W-3 静态回归守护);临时端到端测试(全新 sqlite 文件库 NewDatabase+AutoMigrate 真实 schema,验证后已删除)×4 全 PASS —— Stats 复刻用户所报 cron 路径(窗口过滤 Pending=1/PendingAll=2 + trend 正常)、Apply/Rollback 写路径(状态机/回滚窗口≈now+7d/资产恢复/resolved_at B-3 全对)、List/GetByID 读路径、checkPortStatusDrift(1 条漂移计数正确+基线落库)
files_changed:
  - internal/services/mac_history_partition.go
  - internal/services/asset/reconciliation_snapshot.go
  - internal/core/db/database.go (含 OperLog、LoginLog 追加注册)
  - internal/core/db/database_test.go (TestNewDatabaseSQLite 断言追加 5 张修复表)
  - internal/services/asset/fix_suggestion_service.go (Stats 方言分支 + userJoinOn/rollbackWindowValue helper + Rollback DB-side 检查 PG 守卫)
  - internal/services/asset/fix_suggestion_generator.go (非 PG 早退)
  - internal/scheduler/reconciliation_tasks.go (checkPortStatusDrift 方言分支)
  - internal/services/asset/fix_suggestion_service_test.go (StatsWindow 强化 + 注释更新)
  - internal/services/asset/fix_suggestion_audit_test.go (注释更新 ×2)
  - .planning/quick/260817-hfl-supabase-postgresql-sqlite-go-modernc-or/deferred-items.md

## 验证期新发现(已修,用户决策 2026-08-17)

- sys_oper_log 缺表:启动后外部 RPA worker(10.62.10.34)心跳触发 operlog 写入报 "no such table: sys_oper_log"。models.OperLog(internal/models/log.go:154)已存在但未注册进 AutoMigrate —— 与本次 3 张表同一根因类(归档 SQL 建表)。用户决策一并修复:database.go sqlite 分支追加注册 &models.OperLog{},tag 核查无 PG-only 片段,自验通过(见 verification)

## 验证期第 2 轮新发现(已修 2026-08-17 17:41)

- sys_logininfor 缺表:用户真实环境验证时 admin 登录成功,但 recordLoginLog(auth.go:574-596,异步 goroutine Create)报 "no such table: sys_logininfor"。models.LoginLog(internal/models/log.go:28-39,TableName()=sys_logininfor)为单一事实源,scripts/dbprovision/main.go:60 已注册,但运行期 database.go sqlite 分支 AutoMigrate 缺漏 —— 同一根因类(归档 SQL 建表)。
- 排查确认无其他登录/审计类别名表:Grep 全库 sys_logininfor 引用仅此一处模型;recordLoginLog 函数体内无其他表写入。
- 修复:database.go sqlite 分支追加注册 &models.LoginLog{};database_test.go TestNewDatabaseSQLite 断言表清单追加 5 张修复表(sys_rpa_workers/sys_rpa_executions/sys_mac_oui_vendor/sys_oper_log/sys_logininfor)作为常驻回归守护。
- 自验:TestNewDatabaseSQLite PASS(5 表全部 HasTable);一次性临时测试复刻 recordLoginLog INSERT 路径 PASS(ID 钩子填充、count=1,验证后删除);go build ./... 与 internal/core/db 全量测试 PASS。

## 验证期第 3 轮新发现(已修 2026-08-17 18:13)

- PG-only SQL 方言在 sqlite 运行期执行:用户真实环境日志(17:47)显示 monitorFixSuggestionMisFix cron → fix-suggestion Stats 报 `near "'1 day'": syntax error`(NOW()/INTERVAL 为 PG 方言)。
- 临时探针实测(glebarez/modernc,验证后删除):`::text` → unrecognized token ":";`NOW()`/`gen_random_uuid()` → no such function;`INTERVAL` → syntax error;`CURRENT_TIMESTAMP` 双方言可用(UTC)。
- 全库排查(Grep internal/ 排除 migrations/archive/test):对账/monitor 族 6 处全修(见 Resolution fix #6);非对账族 5 类(rpa Register、floor 软删恢复、AD assignRole、login_log Clean 错表 sys_login_log、operations 大范围 ::cast)记入 deferred-items.md,避免范围膨胀。
- 修复遵循库内既有方言分支范式(statsTrend date_trunc/strftime、TopUnresolved EXTRACT/julianday、computeHealthTrend);PG 分支 SQL 逐字保留,W-2/W-3 静态回归守护保持绿色。

## Evidence

- timestamp: 2026-08-17 17:41
  checked: internal/models/log.go:28-39 LoginLog 全字段 tag + scripts/dbprovision/main.go:60 + Grep 全库 sys_logininfor/LoginLog 引用
  found: LoginLog 为单一事实源(全库仅 log.go 一处结构体定义,无 sys_login_log 别名表);dbprovision 已注册但运行期 AutoMigrate 缺漏;tag 无 '(' 函数默认值、无 '[' 数组类型,与 OperLog 同一 BaseTimeLine(ID 由 BeforeCreate 钩子填充)
  implication: 与 OperLog 同范式,sqlite 分支直接 append 注册即可,无需 sanitize
- timestamp: 2026-08-17 17:41
  checked: 一次性临时测试 TestTmpLoginLogInsertSQLite(复刻 auth.go recordLoginLog 字段 INSERT,全新 sqlite 库 AutoMigrate 后执行)
  found: PASS —— sys_logininfor 建表成功,INSERT 成功,ID 非空(BeforeCreate 钩子生效),count=1;临时测试验证后已删除
  implication: 假设证伪测试通过,修复闭环;HasTable 断言已并入 TestNewDatabaseSQLite 常驻回归守护
- timestamp: 2026-08-17 18:05
  checked: 临时探针 TestTmpDialectProbe(全新 sqlite in-memory,glebarez/modernc)— 实测 6 种 PG 语法在 sqlite 的行为
  found: `1::text` → unrecognized token ":";`NOW()` → no such function: NOW;`gen_random_uuid()` → no such function;`NOW() - 7 * INTERVAL '1 day'` → syntax error;`CURRENT_TIMESTAMP` 可用(UTC,与 CDX-M-UTC NowFunc 对齐);datetime 字符串与 CURRENT_TIMESTAMP 比较正常。探针验证后已删除
  implication: 用户所报 Stats 错误根因确认;CURRENT_TIMESTAMP 可作为双方言通用写法;全库 NOW()/INTERVAL/:: 语法点需逐一评估
- timestamp: 2026-08-17 18:06
  checked: Grep 全 internal/(排除 migrations/archive/test/textfsm)扫 NOW()/INTERVAL/ILIKE/date_trunc/::cast + 所有 .Raw(/.Exec( 调用点
  found: 对账/monitor 族 sqlite 路径 PG-only 方言 6 处: fix_suggestion_service.go Stats 6 Count(L300-334)、su.id::text JOIN ×3(L174/239/266)、Apply gorm.Expr("NOW() + INTERVAL '7 day'")(L577)、Rollback `rollback_window_until > NOW()`(L668)、fix_suggestion_generator.go 对 PG-only MV reconciliation_normalized 的逐候选查询(L75)、reconciliation_tasks.go checkPortStatusDrift ::text ×4(L371-377)。非对账族另发现 5 类(rpa Register/floor 恢复/AD assignRole/login_log Clean 错表/operations 大范围 ::cast)已记 deferred-items.md
  implication: 对账族 6 处全修(同一根因类,用户必然陆续踩中);非对账族记录 deferred 避免范围膨胀
- timestamp: 2026-08-17 18:13
  checked: 临时端到端测试(全新 sqlite 文件库 NewDatabase + AutoMigrate 真实 schema,验证后已删除): TestTmpFixSuggestionStatsSQLite(复刻 monitorFixSuggestionMisFix cron 路径)、TestTmpFixSuggestionApplyRollbackSQLite(写路径)、TestTmpFixSuggestionListSQLite(读路径)、TestTmpCheckPortStatusDriftSQLite(漂移 cron)
  found: 全 PASS —— Stats 返回 Pending=1/PendingAll=2(窗口过滤语义正确)+ trend series 正常;Apply 成功(rollback_window_until≈now+7d、ops_asset.user_id 更新、resolved_at 落库 B-3);Rollback 成功(user_id 恢复 old-user);List/GetByID 成功(total=1, history=1);checkPortStatusDrift 成功且计数正确(1 条漂移,基线写 sys_config=1)
  implication: 假设证伪测试通过,修复闭环;期间发现 sys_data_reconciliation.raw_snapshot(jsonb→json.RawMessage)对 TEXT 存储值 Scan 失败,系测试 INSERT artifact(GORM Create 走 BLOB 无此问题),非产品 bug
- timestamp: 2026-08-17
  checked: internal/models/log.go:6-25 OperLog 全字段 tag + internal/models/base.go BaseTimeLine
  found: 无 '(' 函数默认值、无 '[' 数组类型；BaseTimeLine 仅 type:uuid;primary_key（SQLite 合法 type-name），ID 由 BeforeCreate 钩子填充 —— 无需纳入 sanitizeSQLiteModelDefaults 净化范围
  implication: sqlite 分支直接 append 注册即可，PG 存量库零漂移

- timestamp: 2026-08-17
  checked: internal/services/mac_history_partition.go + core.go:434 + scheduler/mac_history_tasks.go:150
  found: EnsurePartitionsExist（启动步骤 9.5）与 DropExpiredPartitions（每日 cron）无条件执行 `PARTITION OF` DDL 和 pg_inherits 目录查询，无任何方言守卫；CreateMonthlyPartition 仅被 EnsurePartitionsExist 内部调用
  implication: 在 EnsurePartitionsExist/DropExpiredPartitions 入口加 sqlite 守卫即可同时覆盖启动与 cron 两条路径，PG 路径零改动
- timestamp: 2026-08-17
  checked: internal/services/asset/reconciliation_snapshot.go RefreshView + core.go:238-246
  found: RefreshView 无条件 SET statement_timeout（第 83 行）+ REFRESH MATERIALIZED VIEW CONCURRENTLY；sqlite 下 migration_176 不执行（d.Type=="postgres" 块），MV 不存在；启动 goroutine 无条件调用
  implication: 在 RefreshView 入口加非 postgres 早退（return nil + DEBUG 日志）可同时覆盖 startup/cron/handler 三路径
- timestamp: 2026-08-17
  checked: internal/models/rpa/worker.go、execution.go vs internal/models/rpa.go；services/rpa/* 引用
  found: 单一事实源是 internal/models/rpa 包（rpamodels.Worker/Execution，services/rpa 全部用它）；internal/models/rpa.go 的 RPAWorker/RPAExecution 无业务引用，属遗留重复定义
  implication: 只需把 rpamodels.Worker/Execution 注册进 sqlite 分支 AutoMigrate，不碰遗留定义
- timestamp: 2026-08-17
  checked: internal/models/mac_oui_vendor.go、mac_history_query_service.go、scripts/dbprovision/main.go:79
  found: models.MACOUIVendor 已存在且为单一事实源（dbprovision 已注册），缺的是运行期 AutoMigrate 注册
  implication: 同 UserPreference 范式追加注册即可
- timestamp: 2026-08-17
  checked: database.go sanitizeSQLiteModelDefaults + 三个候选模型 tag
  found: 净化只遍历 MigrateModelList()；rpamodels.Worker/Execution、MACOUIVendor 的 tag 无 '(' 函数默认值、无 '[' 数组类型（jsonb/uuid/CURRENT_TIMESTAMP 在 SQLite DDL 均为合法 type-name/常量默认）
  implication: 追加注册无需扩展净化范围，风险可控
