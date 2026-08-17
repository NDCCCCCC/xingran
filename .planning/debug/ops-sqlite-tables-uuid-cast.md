---
slug: ops-sqlite-tables-uuid-cast
status: resolved
trigger: 前端运维模块页面全部报错，后端 ops 接口批量 500/400，GORM 日志显示两类 SQLite 错误（no such table / unrecognized token ":"）。config.yaml database.type=sqlite。
created: "2026-08-18T00:30:00Z"
updated: "2026-08-18T00:48:00Z"
---

# Debug Session: ops-sqlite-tables-uuid-cast

## Symptoms

1. **Expected behavior**: SQLite 模式（configs/config.yaml `database.type: "sqlite"`）下启动后端后，运维模块页面（楼宇/楼层/工位/机房/机房设备/信息点/位置别名/字典）正常加载列表与统计数据。
2. **Actual behavior**: 所有 ops list 接口 500，workstation/location-alias/dicts 接口 400/500；building/floor/serverRoom/statistics 部分接口 200（未走坏 SQL 的路径正常）。
3. **Error messages**（GORM 逐字摘录，两类）:
   - 类 A `SQL logic error: no such table: sys_workstation (1)` — 影响 `/ops/workstation/list`、`/ops/workstation/statistics`
   - 类 A `SQL logic error: no such table: sys_dept_location_alias (1)` — 影响 `/ops/location-alias/list`
   - 类 A `SQL logic error: no such table: sys_dict_data (1)` — 影响 `/system/dicts/data/list`（含 ops_info_point_type / ops_dedicated_line_type / ops_isp 等字典类型）
   - 类 B `SQL logic error: unrecognized token: ":" (1)` — 影响 `/ops/building/list`、`/ops/floor/list`、`/ops/serverRoom/list`、`/ops/roomDevice/list`、`/ops/infoPoint/list`
   - 类 B 代表 SQL（含 PG 专有 cast）:
     ```sql
     SELECT ops_buildings.*, (SELECT COUNT(*) FROM sys_workstation ws
       JOIN ops_floors f ON f.id = ws.floor_id::uuid
       WHERE f.building_id::uuid = ops_buildings.id AND ...) AS workstation_count
     FROM `ops_buildings` ...
     ```
     floor/list 同样含 `ops_floors.building_id::uuid`、`sys_files.id = ops_floors.plan_image_id::uuid`；serverRoom 含 `b.id::text`、`f.id::text`；roomDevice 含 `room_id::uuid`、`building_id::uuid`；infoPoint 含 `sys_workstation.id::text`、`floor_id::uuid`、`building_id::uuid`。
4. **Timeline**: 2026-08-18 00:18 前后持续发生；SQLite 本地开发模式。
5. **Reproduction**: 启动后端（sqlite 模式）+ 前端 dev server，打开任意运维管理页面即触发；复现率 100%。

## Evidence

- timestamp: 2026-08-18T00:30:00Z — orchestrator 确认 configs/config.yaml line 14 `type: "sqlite"`，后端以 SQLite 运行
- timestamp: 2026-08-18T00:30:00Z — "SQL logic error" 前缀为 SQLite 错误特征；`::uuid`/`::text` 为 PostgreSQL 专有 cast 语法 → services 层 raw SQL 存在方言硬编码
- timestamp: 2026-08-18T00:30:00Z — 项目记忆（memory: xingran-db-migration-and-pk-gotcha）提示既有模式：sqlite 缺表需注册 AutoMigrate + sanitize；PG 专有语法需方言分支
- timestamp: 2026-08-18T00:45:00Z — knowledge-base.md 不存在，但 resolved/ 两会话与本症完全同族: `sqlite-startup-pg-only-errors.md`(缺表注册 AutoMigrate sqlite 分支 + 方言分支范式) 与 `reconciliation-table-missing-sqlite-20260817.md`(归档迁移建表 → sqlite 缺注册)。前者 blind_spots 明确记录 "operations 模块大范围 ::cast 属同一方言类但 deferred" —— 本次用户报的就是该 deferred 项到期
- timestamp: 2026-08-18T00:45:00Z — deferred-items.md (.planning/quick/260817-hfl-.../deferred-items.md) 给出 cast 点全集地图: workstation_service.go:17,69,255,273,436,446 / floor_service.go:170-171,194,229-230,264 / infopoint_service.go:133-135,182-185,212-214 / server_room_service.go:137-138,209 / room_device_service.go:106,125-126 / excel_query_builder.go:222-236 / building_service.go:153 / workstation_device_service.go:402-488,1785-1843;另 reconciliation_service.go 与 mac_history_query_service.go(依赖 PG 分区表,属 deferred #4,不在本次范围)
- timestamp: 2026-08-18T00:45:00Z — 库内已落地双方言写法范式: workstation_service.go:97-100 与 location_alias_service.go:26 用 `CAST(x AS TEXT)`(标准 SQL,PG/SQLite 行为一致)——比 Dialector 分支更优(零分支);::uuid 方向的 cast 需反转成对 uuid 侧 CAST AS TEXT 比较
- timestamp: 2026-08-18T00:45:00Z — resolved/postgres-uuid-text-type-mismatch.md 证明 ::uuid cast 是 PG 上刻意的修复(varchar 列 JOIN uuid 主键需显式 cast)——不能简单删除,否则 PG 回归;方言分支或 CAST 反转必须保持 PG 语义等价
- timestamp: 2026-08-18T00:45:00Z — Grep/Glob 专用工具在本环境报 "Executable not found"(claude.exe 路径失效),改用 bash grep/rfind 替代;无知识库文件,已读 resolved 会话代替

## Eliminated

（暂无）

## Current Focus

hypothesis: CONFIRMED（双根因，证据见 reasoning_checkpoint）
reasoning_checkpoint:
  hypothesis: "SQLite 模式 ops 列表接口批量失败 = 双根因。(A) 6 张表(sys_workstation/sys_dept_location_alias/sys_dict_data/sys_dict_type/ops_workstation_device/sys_files)的模型未注册进运行期 database.go sqlite 分支 AutoMigrate(MigrateModelList L474-573 与 sqlite append L630-683 均无;其中 4 个+SysFile 恰好注册在 scripts/dbprovision:57-68,81 —— PG 新部署专用)→ 全新 sqlite 库永远缺表。(B) operations 5 个 service 列表/统计 SQL 硬编码 PG 专有 ::uuid/::text cast,SQLite 解析期报 unrecognized token ':'"
  confirming_evidence:
    - "database.go MigrateModelList + sqlite append 列表均无 Workstation/SysDeptLocationAlias/DictData/DictType/WorkstationDevice/SysFile;dbprovision 恰有 DictType/DictData/Workstation/SysDeptLocationAlias(+SysFile) —— 与 sys_logininfor/sys_data_reconciliation 同类缺漏(两次 resolved 先例)"
    - "错误出现模式与 SQL 执行顺序吻合: workstation List 先 Count(Model 无 JOIN)→ 先报 no such table: sys_workstation;building/floor Find 带 ::cast → 解析期 unrecognized token ':' 先于表解析出现"
    - "floor List JOIN sys_files(floor_service:171,230)、workstation joinSelect 子查询引用 ops_workstation_device(workstation_service:16) —— 只修 cast 不建这两张表,接口仍 500,故属本症必要闭环项"
    - "deferred-items.md(2026-08-17 方言排查)逐行枚举了本次全部 cast 点位并预告 'sqlite 下打开对应列表页面即报错' —— 本症即该预告命中"
    - "库内已验证双方言等价写法: workstation_service:97-116 与 location_alias_service:23-30 的 CAST(x AS TEXT)(PG: uuid→text 与 varchar 同族可 =;SQLite: TEXT no-op 比较),生产 PG 已在用"
  falsification_test: "全新 sqlite 库跑 AutoMigrate: 若 6 表 HasTable 已为 true 或建表后 5 service List 仍报 unrecognized token/no such table,则假设被证伪"
  fix_rationale: "(A) 镜像 sys_logininfor/SysDataReconciliation 先例,仅 sqlite 分支 append 注册 6 模型(PG 存量库零漂移,PG 新部署仍走 dbprovision);6 模型 tag 逐一核查无函数式默认/数组类型,无需 sanitize(jsonb type-name 非 STRICT 表可接受,先例 SysDataReconciliation 已验证)。(B) 用已落地 CAST(x AS TEXT) 范式改写 5 service 全部 cast 点 —— cast 方向统一反转到 uuid 侧: x.id = y.col::uuid → CAST(x.id AS TEXT) = y.col;x.id::text = y.col → CAST(x.id AS TEXT) = y.col;PG 语义等价(text 同族比较,与生产已用形状一致),SQLite 合法;单串双方言无分支,改动最小。TestNewDatabaseSQLite 追加 6 表 HasTable 常驻回归断言"
  blind_spots: "(1) 本环境无 PG,PG 回归未实测 —— CAST 形状与生产已用一致,风险低但非零;(2) workstation_device_service/excel_query_builder/reconciliation/mac_history 的 cast 属 deferred 既有记录,超出用户所报 9 个列表端点范围,本次不修;(3) sqlite 新库字典数据为空(PG 由归档 SQL seed),修复后 dict data list 返回空列表 200 —— 数据 seeding 属独立事项;(4) floor_service:104 软删恢复 NOW() 为写路径,不在列表端点链路,维持 deferred"
next_action: fix 已应用并自验通过 —— 待用户重启后端(对既有 data/xingran.db 自动补建 6 表)后在前端运维各页面确认,返回 checkpoint 结果

## Resolution

root_cause: |
  双根因，均在修复前用临时测试复现证实:
  (A) 缺表类: 6 张运维列表页依赖的表(sys_workstation / sys_dept_location_alias / sys_dict_type / sys_dict_data / ops_workstation_device / sys_files)的模型未注册进运行期 database.go sqlite 分支 AutoMigrate —— MigrateModelList() 与 sqlite append 列表均无它们(其中 4 个+SysFile 恰好注册在 scripts/dbprovision,但那只服务 PG 新部署;PG 存量库靠归档 SQL/迁移已有表)。全新 sqlite 库永远缺表 → 列表/统计/字典接口 no such table。与 sys_logininfor / sys_data_reconciliation 两次 resolved 先例同一缺漏类。其中 ops_workstation_device(workstation List joinSelect 子查询)与 sys_files(floor List JOIN plan_image_url)虽不在用户日志显式报错中,但修完 cast 后会成为下一层 no such table(SQLite 解析期语法错误先于表解析,故用户日志只见 cast 错误)—— 属本症必要闭环项。
  (B) 方言类: operations 5 个 service(workstation/building/floor/infopoint/server_room/room_device)的列表/统计 SQL 硬编码 PG 专有 ::uuid/::text cast(uuid 主键列与 varchar 外键列比较时强制转换),SQLite 解析期报 "unrecognized token ':'"。该 cast 在 PG 是刻意修复(resolved/postgres-uuid-text-type-mismatch),不可删除,只能双方言等价改写。deferred-items.md(2026-08-17 方言排查)已逐行预告这些点位,本症即该预告命中。
fix: |
  (A) database.go AutoMigrate sqlite 分支追加注册 6 模型(&models.Workstation{} / &models.SysDeptLocationAlias{} / &models.DictType{} / &models.DictData{} / &models.WorkstationDevice{} / &sysmodels.SysFile{}),镜像 sys_logininfor 先例:PG 分支零改动(存量库零漂移,PG 新部署仍走 dbprovision)。6 模型 tag 逐一核查无函数式默认/数组类型,无需 sanitizeSQLiteModelDefaults;WorkstationDevice 的 3 个 GORM 关联指针因 DisableForeignKeyConstraintWhenMigrating=true 不产生 FK/级联 DDL;SysDeptLocationAlias 的 partial unique index(migration_165)sqlite 下不建(同 GiST 取舍先例)。
  (B) 用库内已落地且生产 PG 在用的 CAST(x AS TEXT) 范式(先例: workstation_service GetWorkstationDeptOptions / location_alias_service)改写全部 cast 点,方向统一反转到 uuid 侧: "x.id = y.col::uuid" → "CAST(x.id AS TEXT) = y.col";"x.id::text = y.col" → "CAST(x.id AS TEXT) = y.col"。PG 语义等价(uuid→text 与 varchar 同族比较),SQLite 合法,单串双方言无运行时分支。共 6 文件: workstation_service.go(joinClause 常量×4 cast、orgId EXISTS×2 处、floorCode×2 处)、building_service.go(workstation_count 子查询×2 cast)、floor_service.go(JOIN×2 组、orgId EXISTS×2)、infopoint_service.go(JOIN×2 组、orgId EXISTS×4 cast)、server_room_service.go(JOIN×2、orgId×1)、room_device_service.go(GetByID×1、List×2)。
  (C) database_test.go TestNewDatabaseSQLite HasTable 断言追加 6 表作常驻回归守护。
  范围外(既有 deferred 记录维持): excel_query_builder.go / workstation_device_service.go / reconciliation_service.go / mac_history_query_service.go 的 cast(导出/物理链路路径,不在所报 9 个列表端点链路)、floor_service.go:104 软删恢复 NOW()(写路径)、sqlite 字典数据 seeding(PG 由归档 SQL seed,sqlite 新库 dict 表为空,接口 200 空列表 —— 数据事项非 schema 事项)。
verification: |
  1. 修复前复现(临时测试,已删): TestTmpMissingTablesRepro —— 全新 sqlite 库 startup AutoMigrate 后 6 表 HasTable 全 false;TestTmpOpsListCastRepro —— 建表后 5 service List 全部报 "SQL logic error: unrecognized token ':' (1)"(与用户日志逐字同 SQL 同错误);workstation Statistics 无 orgId 参数时不走 cast 路径而通过 —— 与用户"部分 statistics 200"症状吻合。
  2. 修复后同一测试全 PASS(6 表全 true;7 个 service 调用零错误)。
  3. JOIN 语义数据场景验证(临时测试 TestTmpOpsListDataScenario,已删): 建 building→floor→workstation→device / infopoint / serverRoom→roomDevice 关联数据链,断言 workstation.floor_name/building_name/primary_device_serial、building.workstation_count=1、floor.building_name、infoPoint.workstation_name、serverRoom.building_name+floor_name、roomDevice.room_name 全部经 CAST 比较正确 JOIN 命中(非仅"无报错")。PASS。
  4. 回归: go build ./... 通过;internal/core/db 全量 PASS(含 TestNewDatabaseSQLite 6 表新断言);internal/services/operations 全量失败集与 HEAD stash 对照逐名一致(8 个 excel_service 预存在失败: TestBatchUpsert_Update/Mixed/WithCamelCaseFields、TestClampPageSize、TestReferenceResolver_ResolveBatch、TestValidator_ValidateFloor/Wall/Door —— 与本次改动无关,零新增失败)。
  5. 真实进程冒烟: go build 二进制以 XINGRAN_DATABASE_PATH 指向临时 sqlite 库启动(viper AutomaticEnv 覆盖生效,日志确认 SQLite连接成功: Temp/xsmoke/smoke.db,用户真实 data/xingran.db 未触碰):"所有表迁移成功"、启动全链零 "no such table"/"unrecognized token"/FATA、:9000 监听、/api/v1/system/auth/public-key HTTP 200(期间外部 RPA worker 心跳 10.62.10.34 亦 200);停服后直接探测 smoke.db sqlite_master —— 6 表全部存在。临时产物(二进制/探针/日志/临时测试)已全部清理。
  6. 待用户真实环境验证: 重启后端(对既有 data/xingran.db 自动补建 6 表)后打开运维各页面。注意: 本机无 PG 环境,PG 回归未实测(CAST 形状与生产已用双方言代码完全一致,风险低)。
files_changed:
  - internal/core/db/database.go (sqlite 分支注册 6 模型 + sysmodels import)
  - internal/core/db/database_test.go (TestNewDatabaseSQLite HasTable 断言 +6 表)
  - internal/services/operations/workstation_service.go (joinClause/EXISTS/floorCode cast → CAST AS TEXT)
  - internal/services/operations/building_service.go (workstation_count 子查询 cast)
  - internal/services/operations/floor_service.go (JOIN×2 组 + orgId EXISTS×2)
  - internal/services/operations/infopoint_service.go (JOIN×2 组 + orgId EXISTS)
  - internal/services/operations/server_room_service.go (JOIN×2 + orgId EXISTS)
  - internal/services/operations/room_device_service.go (GetByID JOIN + List JOIN×2)
