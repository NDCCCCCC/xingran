---
slug: ops-list-join-uuid-500
status: resolved
created: 2026-10-15
updated: 2026-10-15
trigger: "上次修改楼层管理/工位管理/信息点管理等页面(在楼层管理显示工位数量)后,以上页面列表接口全部 500"
---

# Debug Session: ops-list-join-uuid-500

## Symptoms

- **Expected**: 楼层管理 / 工位管理 / 信息点管理 列表页正常加载
- **Actual**: `POST /api/v1/ops/floor/list`、`POST /api/v1/ops/workstation/list`、`POST /api/v1/ops/infoPoint/list` 全部返回 500;building/list、serverRoom/list、statistics、dropdown-options 正常
- **Errors** (后端日志, 2026-10-15 16:51):
  - floor: `SELECT count(*) FROM "ops_floors" LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = ops_floors.building_id ...` → `ERROR: operator does not exist: text = uuid (SQLSTATE 42883)`
  - workstation: `... LEFT JOIN ops_floors ON CAST(ops_floors.id AS TEXT) = sys_workstation.floor_id LEFT JOIN sys_dept ON CAST(sys_dept.id AS TEXT) = sys_workstation.dept_id LEFT JOIN sys_user ON CAST(sys_user.id AS TEXT) = sys_workstation.user_id ...` → `ERROR: operator does not exist: character varying = uuid (SQLSTATE 42883)`
  - infoPoint: `SELECT count(*) FROM "ops_info_points" LEFT JOIN sys_workstation ON CAST(sys_workstation.id AS TEXT) = ops_info_points.workstation_id ...` → `ERROR: operator does not exist: text = uuid (SQLSTATE 42883)`
- **Timeline**: 上次会话实现"楼层管理显示工位数量"之后开始。之前这些页面正常
- **Reproduction**: 打开楼层管理/工位管理/信息点管理页面即触发

## Initial Hypothesis (orchestrator note, verify not assume)

上次功能修改可能改了 Go model 中外键字段的类型或 column tag(如 string→uuid.UUID 或去掉了 size:64 tag),导致 GORM AutoMigrate 在启动时把 DB 列从 varchar ALTER 成 uuid。而手写 JOIN SQL 仍是旧契约 `CAST(PK AS TEXT) = FK列`,FK 列现在已是 uuid 类型 → 42883。注意项目记忆:此前 ops_info_points.port_id/workstation_id/device_id 是 varchar(size:64),若现在报 uuid 说明 schema 被改动,需查明是哪个 migration / AutoMigrate 改的,以及是否还有其他写路径依赖 varchar 契约。

**已证伪**:model 与 DB schema 都没变;是查询 SQL 的比较方向写错了(见 Evidence)。

## Current Focus

hypothesis: (已确认,见 reasoning_checkpoint)三处手写 SQL 把 uuid 类型列当 varchar 比较 → PG 42883
test: (已完成)真实 PG 上逐字复现三条错误 + 验证修复后 SQL 全部通过
expecting: (已达成)修复后 13 项端到端检查全过
next_action: (已完成)用户已重建重启后端并在 UI 确认三个列表页恢复正常 → resolved
reasoning_checkpoint:
  hypothesis: "三个列表接口的原始 SQL 把 uuid 类型 FK 列当作 varchar 比较:l7k commit 0d3e9670 把 floor 的 sys_files JOIN 从 plan_image_id::uuid 改成 CAST(sys_files.id AS TEXT)=plan_image_id(plan_image_id 实为 uuid, model type:uuid);工作区未提交 Phase 91-02 repo 化又给 infopoint 加了 CAST(ops_buildings.id AS TEXT)=sys_workstation.building_id(building_id 实为 uuid),并给 workstation 的 primary_device_serial 子查询丢了 ::text(workstation_id varchar vs id uuid)→ PG 42883"
  confirming_evidence:
    - "直查 DB(10.62.10.34/ruoyi): sys_workstation.building_id=uuid(历史漂移,migration 192/193 注释为证), ops_floors.plan_image_id=uuid(model type:uuid), ops_workstation_device.workstation_id=varchar, 其余 FK 均 varchar(64)"
    - "对生产库直接执行日志中的 SQL:floor NEW 报 text = uuid、workstation joins+select 报 character varying = uuid、infopoint NEW 报 text = uuid —— 三条错误消息与日志逐字一致"
    - "OLD 提交版 SQL(infopoint ::uuid 写法)对同一库全部通过;三个 FIXED 形式(去掉对 uuid 列的 CAST/恢复 CAST(id AS TEXT))全部通过"
  falsification_test: "若 OLD 形式在同一库也报 42883,或 FIXED 形式仍报错,则本假设错。实测均未发生"
  fix_rationale: "比较方向必须匹配真实列类型:uuid PK 对 uuid FK 直接 = (双方言:PG uuid=uuid / SQLite TEXT=TEXT);uuid PK 对 varchar FK 用 CAST(pk AS TEXT)=fk(现状正确,保留)。改的是错误引入的三处 SQL,不动 schema、不动 model、不动正确的 varchar 路径"
  blind_spots: "① 未审计全仓库其他 uuid FK 列(sys_files 关联等)的同类比较——本次只修三个报错列表路径涉及的列(scope constrainment);② sys_workstation.building_id DB=uuid 与 model size:64 的漂移未治理(AutoMigrate 行为未验),属既有独立问题;③ SQLite 路径仅靠类型系统推理(双方言形式),未实际跑 SQLite 验证"
tdd_checkpoint:

## Eliminated

- hypothesis: 上次修改改了 model 外键字段类型/GORM tag,AutoMigrate 把 DB 列从 varchar ALTER 成 uuid
  evidence: ① 三个 model 文件(floor/infopoint/workstation)git diff HEAD 为空,与 HEAD 一致;② information_schema 直查显示绝大多数 FK 列仍是 varchar(64),仅 sys_workstation.building_id(历史漂移,非本次)与 plan_image_id(model 一直 type:uuid)是 uuid;③ Workstation.BuildingID 的 size:64 tag 自创建以来未变
  timestamp: 2026-10-15

## Evidence

- timestamp: 2026-10-15 (session continuation)
  checked: information_schema.columns 直查 10.62.10.34/ruoyi (cmd/tmp_dbcheck)
  found: PK 全部 uuid;varchar(64) FK: ops_floors.building_id, ops_server_rooms.building_id/floor_id, sys_workstation.dept_id/floor_id/user_id, ops_info_points.workstation_id/device_id/port_id, ops_buildings.org_id;**uuid 类型的 FK: sys_workstation.building_id(与 model size:64 不符,历史漂移), ops_floors.plan_image_id(model 明确 type:uuid)**;sys_files.id/sys_dept.id/sys_user.id=uuid, ops_workstation_device.workstation_id=varchar
  implication: FK 列类型不是清一色 varchar——新 JOIN 写法 "CAST(pk AS TEXT) = fk" 对 uuid FK 列必炸;旧写法 "uuid_pk = varchar_fk::uuid" 对 varchar FK 列才成立,反之亦然。两类列需要各自的比较方向
- timestamp: 2026-10-15 (session continuation)
  checked: git diff HEAD -- internal/services/operations/{infopoint,workstation,floor}_service.go
  found: 工作区未提交改动把 infopoint/workstation 的 JOIN 从旧写法(`ops_floors.id = sys_workstation.floor_id::uuid` 等,注释明说"floor_id/building_id是varchar,而floors/buildings的id是UUID")统一改写为 CAST(pk AS TEXT) = fk 写法,并前置到 Count(repo 化 Phase 91-02)。floor_service.go 的 JOIN(含 sys_files/plan_image_id)无工作区改动(HEAD 即是)
  implication: 破坏性 SQL 来自工作区 refactor,不是 AutoMigrate 改 schema;DB 列类型未变
- timestamp: 2026-10-15 (session continuation)
  checked: internal/core/db/migrations/migration_192/193 注释 + git log model 历史
  found: migration 192/193(旧迁移)明确记载 sys_workstation.building_id 是 uuid、ops_floors.building_id 是 varchar(64);Workstation.BuildingID model 自始至终 size:64(仅空白变化);OpsFloor.PlanImageID 自 fa5f2775 起就是 type:uuid
  implication: sys_workstation.building_id=uuid 是长期存在的 schema 事实,refactor 假设"FK 全 varchar"是错的
- timestamp: 2026-10-15 (session continuation)
  checked: git log -S 定位三处破坏 SQL 的引入点;真实 PG 上执行新旧 SQL 对比
  found: ① floor: commit 0d3e9670(261012-l7k,即触发所述"楼层显示工位数量"功能)把 `sys_files.id = ops_floors.plan_image_id::uuid` 改成 `CAST(sys_files.id AS TEXT) = ops_floors.plan_image_id` → 实测该 SQL 报 `text = uuid`,与日志逐字一致;② workstation: 工作区 refactor 把 workstationJoinSelect 子查询 `workstation_id = sys_workstation.id::text` 的 `::text` 静默丢失(注释声称"常量逐字平移"但实际未逐字) → joins-only 的 count 通过、带 select 子查询的报 `character varying = uuid`,与日志一致;③ infopoint: 工作区 refactor 加 `CAST(ops_buildings.id AS TEXT) = sys_workstation.building_id` → `text = uuid`,而 HEAD 旧写法 `ops_buildings.id = sys_workstation.building_id::uuid` 同库通过(count=1341)
  implication: 三个 500 全部复现且与引入点一一对应;floor 由已提交的 l7k 引入,workstation/infopoint 由未提交的 repo 化引入
- timestamp: 2026-10-15 (session continuation)
  checked: 修复后 SQL 形式在真实 PG 执行 + 端到端 GORM 服务调用(cmd/tmp_verify_lists,已删)
  found: 修复形式(floor 直接 uuid=uuid / infopoint building join 直接 uuid=uuid / workstation 子查询 CAST(sys_workstation.id AS TEXT))全部通过;经真实 GORM 连接调用 NewFloorService/NewWorkstationService/NewInfoPointService 的 List(含 orgId/floorCode 过滤变体)、Statistics、SearchWorkstationOptions、GetWorkstationDeptOptions、三处 GetByID 共 13 项检查 ALL PASS,floor total=50 / workstation total=1643 / infoPoint total=1341(与直查 SQL count 一致)
  implication: 修复在真实 PG 上经 GORM 实际代码路径验证有效
- timestamp: 2026-10-15 (human verify checkpoint)
  checked: 用户确认 — backend rebuilt via `go build -o ruoyi-backend.exe ./cmd/main.go` (BUILD_OK), 重启后在 UI 确认
  found: 楼层管理 / 工位管理 / 信息点管理 三个列表页恢复正常,修复生效
  implication: 问题闭环,session resolved

## Resolution

root_cause: ops 表 FK 列在 PG 里是混合类型(多数 varchar(64),但 sys_workstation.building_id 与 ops_floors.plan_image_id 是 uuid)。两类改动把手写 SQL 的比较方向统一按"FK 全是 varchar"重写:① 已提交 commit 0d3e9670(楼层工位统计功能)把 floor 的 sys_files JOIN 改为 CAST(sys_files.id AS TEXT)=ops_floors.plan_image_id;② 未提交的 Phase 91-02 repo 化给 infopoint 加 CAST(ops_buildings.id AS TEXT)=sys_workstation.building_id,并给 workstation 的 primary_device_serial 子查询丢了 ::text。对 uuid 列产生 text=uuid / varchar=uuid → PG SQLSTATE 42883 → 三个列表接口 500
fix: 三处最小修复,按真实列类型选择比较方向(uuid 列直接 =,varchar 列保持 CAST(pk AS TEXT)=fk):① floor_service.go joinScope: `LEFT JOIN sys_files ON sys_files.id = ops_floors.plan_image_id`(去掉错误 CAST);② infopoint_service.go joinScope: `LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id`(去掉错误 CAST,并修正误导注释);③ workstation_service.go workstationJoinSelect: 子查询恢复 `workstation_id = CAST(sys_workstation.id AS TEXT)`。均保持 PG/SQLite 双方言可用。未动 schema/model/正确的 varchar 路径
verification: go build ./... 通过;go test ./internal/services/operations/ 全绿(5.998s);真实 PG 上直执行修复 SQL 全通过;经 GORM 服务层端到端 13 项检查 ALL PASS(三个 List total 与直查一致,含过滤变体/Statistics/下拉/GetByID);用户重建重启后端后在 UI 确认三个页面恢复正常
files_changed:
  - internal/services/operations/floor_service.go
  - internal/services/operations/infopoint_service.go
  - internal/services/operations/workstation_service.go

## Follow-up Notes (非本次 scope,供后续决策)

1. `sys_workstation.building_id` DB=uuid vs model `size:64`(varchar) 是既有漂移(migration 192/193 时代即如此)。任何新 SQL 触碰该列必须按 uuid 处理;治理(对齐 model 或 schema)需单独决策
2. `excel_query_builder.go:234` 仍用 PG 专有 `sys_workstation.building_id::uuid`(PG 正确,SQLite 会语法错)——双方言欠账,非本次故障点,未动
3. SQLite 单测无法守护此类 PG 类型不匹配(SQLite 动态类型,两种写法都过)——本次三个 500 正是"sqlite 测试全绿 + PG 炸"的实例;若要回归守护需 PG 集成测试或 DryRun SQL 断言
