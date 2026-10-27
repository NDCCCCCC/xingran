---
phase: quick-261015-ops-fk-drift-guard
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/models/workstation.go
  - internal/services/operations/pg_contract_test.go
autonomous: true
requirements:
  - quick-261015-ops-fk-drift-guard
user_setup: []

must_haves:
  truths:
    - "Workstation.BuildingID 的 gorm tag 声明为 uuid,与 DB 列 sys_workstation.building_id(uuid) 一致,AutoMigrate 不再携带 varchar 语义"
    - "TEST_PG_DSN 未设置时 pg_contract_test.go 全部 t.Skip,本地/CI 无 PG 不红"
    - "TEST_PG_DSN 设置且可达时,三个 List + 三个 Statistics 真实打到 PG 并全部 err==nil('sqlite 全绿 + PG 42883' 类回归从此有守护)"
  artifacts:
    - path: "internal/models/workstation.go"
      provides: "BuildingID 字段 type:uuid tag(照抄 OpsFloor.PlanImageID 模式)"
      contains: "type:uuid"
    - path: "internal/services/operations/pg_contract_test.go"
      provides: "PG-gated 类型契约集成测试(包内测试,package operations)"
      min_lines: 60
  key_links:
    - from: "internal/services/operations/pg_contract_test.go"
      to: "NewFloorService/NewWorkstationService/NewInfoPointService"
      via: "gorm.Open(postgres.Open(TEST_PG_DSN)) 构造服务后调用 List/Statistics"
      pattern: "TEST_PG_DSN"
    - from: "internal/models/workstation.go BuildingID"
      to: "DB sys_workstation.building_id (uuid)"
      via: "gorm type:uuid 契约声明"
      pattern: "type:uuid"
---

<objective>
治理 `sys_workstation.building_id` 的 model/DB 类型漂移(model 对齐 DB),并建立 PG-gated 类型契约回归测试,防住"SQLite 单测全绿 + PG 42883 炸"类回归(源自已归档 debug 会话 ops-list-join-uuid-500 的 Follow-up Notes ①③)。

Purpose: DB 列是 uuid 而 model `Workstation.BuildingID` 是 `size:64`(varchar 语义),自 migration 192/193 时代即漂移;任何按 model tag 推断类型的新 SQL(或新库初始化)都会再次产生 text/varchar = uuid 的 42883。SQLite 动态类型测不出这类错误,本次三个 ops 列表 500 正是实例。

Output: ① `internal/models/workstation.go` BuildingID tag 对齐;② `internal/services/operations/pg_contract_test.go` PG-gated 集成测试;2 个原子 commit。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/debug/resolved/ops-list-join-uuid-500.md
@./CLAUDE.md

<interfaces>
<!-- 执行器直接使用以下契约,无需再探查代码库 -->

From internal/models/operations/floor.go (type:uuid 既有范例,Task 1 必须照抄此写法):
```go
PlanImageID  *string     `gorm:"type:uuid" json:"planImageId,omitempty"`
```

From internal/models/workstation.go (Task 1 修改目标,line 48):
```go
BuildingID   *string `gorm:"size:64" json:"buildingId,omitempty"`    // 楼宇ID
```

From internal/services/operations/ (三个服务构造器都只收 *gorm.DB,测试装配极简):
```go
func NewFloorService(db *gorm.DB) FloorService
func NewWorkstationService(db *gorm.DB) WorkstationService
func NewInfoPointService(db *gorm.DB) InfoPointService

// FloorService (floor_service.go:33,37)
List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
Statistics(ctx context.Context) (*FloorStatisticsResult, error)

// WorkstationService (workstation_service.go:52,56)
List(ctx context.Context, req requests.WorkstationListRequest) (*PageResult, error)
Statistics(ctx context.Context, params map[string]interface{}) (*WorkstationStatisticsResult, error)

// InfoPointService (infopoint_service.go:20,23)
List(ctx context.Context, req requests.InfoPointListRequest) (*PageResult, error)
Statistics(ctx context.Context) (*InfoPointStatisticsResult, error)
```

Import 路径(workstation_service.go:8-11 同款):
```go
"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests" // WorkstationListRequest / InfoPointListRequest
"github.com/ruoyi-next/ruoyi-go-backend/internal/models"                      // models.Workstation
```

List 的 JOIN 路径无条件生效,无需特殊参数:
- workstation: `workstationJoinSelect`/`workstationJoinClause` 常量(workstation_service.go:17,21,9c0e3e5c 修复处)
- floor/infopoint: 各自 joinScope(infopoint_service.go:156-159 / floor_service.go List)
- 零值 request 即可(内部分页有默认值);`params map[string]interface{}` 传 `nil` 即可
</interfaces>

## 已锁决策(不得重新决策)

1. **漂移方向 = model 对齐 DB,不动 schema、不动数据。** 禁止 ALTER TABLE、禁止 migration。DB→varchar 方案会废掉 9c0e3e5c 刚修好的 uuid=uuid JOIN。
2. **附带审计:** grep 全仓库 `sys_workstation.building_id` 引用,确认无按 varchar 方向比较的残留。`excel_query_builder.go:234` 的 `::uuid`(uuid 列上 no-op)合法,保留。
3. **② = PG-gated 集成测试**,读 `TEST_PG_DSN`;未设置或连不上 → `t.Skip`。**明确否决 DryRun 方案**(DryRun 不发 SQL 到 PG,验不出类型错误)。
4. 测试文件: `internal/services/operations/pg_contract_test.go`,包内测试。
5. **提交边界极严:** 工作区有大量无关的未提交 refactor 改动。每个 commit 只 `git add <本次任务文件路径逐个列出>`,禁止 `git add -A`/`git add .`/`git add -u`。不提交 `.planning/` 下任何文件(SUMMARY/STATE 由编排者负责)。

## 规划期已完成的全仓库审计结论(任务内复核即可,不需重查全量)

对 `sys_workstation.building_id`(uuid 列)的全部写入/比较路径逐一定性:

| 位置 | 形式 | 定性 |
|------|------|------|
| infopoint_service.go:159 | `LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id` | uuid=uuid,正确(9c0e3e5c 修复),保留 |
| excel_query_builder.go:234 | `ops_buildings.id = sys_workstation.building_id::uuid` | uuid 列上 `::uuid` 是 no-op,PG 合法;SQLite 会语法错但该路径仅 PG 导出使用(已知双方言欠账,锁决策明确保留) |
| floor_service.go:581-583 (syncWorkstationBuildingID) | `SET building_id = ?` + `building_id != ?` | 参数化写入,PG 从上下文推断 `$1` 为 uuid;合法,保留 |
| migration_158 (索引 DDL) / migration_192/193 | DDL/回填 | uuid-aware,保留 |
| workstation_service.go:21,122 / floor_service.go:257,277,412 / server_room_service.go:118,158 / room_device_service.go:128 / infopoint_service.go:200 | `CAST(x.id AS TEXT) = <表>.building_id` | 比较对象是 ops_floors.building_id / ops_server_rooms.building_id,均为 varchar(64),方向正确,保留 |

**结论: 无按 varchar 方向比较 `sys_workstation.building_id` 的残留。** Task 1 中执行器只需复核该表所列 grep 仍与定性一致;若发现新位置,按"uuid 列直接 = / varchar 列 CAST(pk AS TEXT)=fk"判定,发现真实残留则修复并记入 SUMMARY。
</context>

<tasks>

<task type="auto">
  <name>Task 1: Workstation.BuildingID gorm tag 对齐 DB uuid 列(model 对齐,零 schema 变更)</name>
  <files>internal/models/workstation.go</files>
  <action>
修改 `internal/models/workstation.go` 第 48 行 `BuildingID` 字段:将 gorm tag 从 `gorm:"size:64"` 改为 `gorm:"type:uuid"`,照抄 `internal/models/operations/floor.go:38` `OpsFloor.PlanImageID` 的既有写法(锁决策 1)。保留 Go 类型 `*string` 不变,保留 json tag `json:"buildingId,omitempty"` 不变。

在字段行尾(或上方独立注释行,与 floor.go WorkstationStats 的中文注释风格一致)加一条简短注释说明缘由,要点:DB 列 sys_workstation.building_id 实际为 uuid(migration 192/193 时代即漂移),type:uuid 声明对齐 DB 防止新库/AutoMigrate 产生 varchar 语义 → text=uuid 42883;参考 ops-list-join-uuid-500 会话。

不做任何其他事:不写 migration、不改其他字段、不 touch `internal/models/operations.go`(legacy Floor 结构体)或 `internal/models/operations/floor.go`(OpsFloor.BuildingID 是 varchar(64) 列,与其 DB 一致,不要动)。

安全性依据(已验证,写进 commit message 即可):type:uuid 仅影响 DDL 推断;生产 DB 该列已是 uuid → AutoMigrate no-op;SQLite 动态类型接受任意类型名,OpsFloor.PlanImageID 同 tag 的 sqlite 测试早已全绿,先例成立。

复核规划期审计结论(context 中表格):跑 `grep -rn "sys_workstation\.building_id" --include="*.go" internal/` 逐条比对表格定性仍成立。无残留 → 继续;发现表格外且方向错误的真实残留 → 修复(按 uuid 列比较方向)并在 SUMMARY 单列。

提交前门禁: `go build ./...` 通过 + `go test ./internal/services/operations/` 全绿。

**提交(锁决策 5):** 只 `git add internal/models/workstation.go` 逐文件显式添加,commit message: `fix(models): Workstation.BuildingID gorm tag 对齐 DB uuid 列(ops-list-join-uuid-500 follow-up)`。若审计发现残留并有修复,残留文件一并逐个显式 add,并在 message 里列明。
  </action>
  <verify>
    <automated>grep -n "type:uuid" internal/models/workstation.go && go build ./... && go test ./internal/services/operations/ 2>&1 | tail -3</automated>
  </verify>
  <done>
    workstation.go 中 BuildingID tag 为 `gorm:"type:uuid"`(无 size:64 残留),Go 类型与 json tag 未变;go build ./... 零错误;operations 包测试全绿;该文件单独成 commit,未混入任何无关文件;全仓库无 varchar 方向比较 sys_workstation.building_id 的残留(或已修复并列明)。
  </done>
</task>

<task type="auto">
  <name>Task 2: PG-gated 类型契约集成测试(TEST_PG_DSN,三服务 List+Statistics 直打 PG)</name>
  <files>internal/services/operations/pg_contract_test.go</files>
  <action>
新建 `internal/services/operations/pg_contract_test.go`,`package operations`(与同目录 workstation_update_createdat_test.go 一致,包内测试)。依赖 `gorm.io/driver/postgres`(go.mod 已有,勿新增依赖)。参考装配思路:直接 `gorm.Open(postgres.Open(dsn))` 构造三个服务,不需要起完整 core(已归档 cmd/tmp_verify_lists 程序的同款思路)。

结构(单个测试函数 `TestPGContract_OpsListJoins`,内部 t.Run 分三段,或三个独立测试函数,任选其一但保持一个 helper 共用连接):

1. **Gating:** `dsn := os.Getenv("TEST_PG_DSN")`;为空 → `t.Skip("TEST_PG_DSN not set; PG contract test skipped")`。若 DSN 未含 `connect_timeout`,追加 ` connect_timeout=5`(防坏主机挂死;pgx v5 禁写 keepalives 类 DSN 参数,connect_timeout 安全)。`gorm.Open(postgres.Open(dsn), gorm.OpenLazy)` 后取 `sqlDB` 做 `PingContext(5s 超时)`,失败 → `t.Skipf("TEST_PG_DSN unreachable: %v", err)`。测试结束 `defer sqlDB.Close()`。
2. **装配:** `NewFloorService(db)`、`NewWorkstationService(db)`、`NewInfoPointService(db)`(三构造器签名见 interfaces)。
3. **调用与断言(全部只读,禁止任何 Create/Update/Delete/Raw 写 SQL —— DSN 可能指向生产库):**
   - floor: `List(context.Background(), nil)` → err==nil 且结果非 nil;`Statistics(ctx)` → err==nil
   - workstation: `List(ctx, requests.WorkstationListRequest{})` → err==nil 且非 nil;`Statistics(ctx, nil)` → err==nil
   - infopoint: `List(ctx, requests.InfoPointListRequest{})` → err==nil 且非 nil;`Statistics(ctx)` → err==nil
   - 错误消息断言:任一 err 非 nil 时,`t.Fatalf` 输出完整 err(便于肉眼识别 SQLSTATE 42883);额外对 err 字符串包含 "42883" 的情况给出显式失败信息("PG type contract broken — FK comparison direction mismatch")。
   - 零值 request 即可,不要编造过滤数据;List 内部 JOIN(workstationJoinClause/joinScope)无条件生效,这就是被守护的路径。

关键红线(锁决策 3):不使用 DryRun —— SQL 必须真实发到 PG 执行;不用 mock;不 skip-and-fake-pass(Gating 走 t.Skip,CI 无 PG 时自然跳过不计失败)。

在文件头写包级注释:说明本测试存在的根因(SQLite 动态类型测不出 text/varchar = uuid 42883,2026-10-15 ops 三列表 500 即实例)、启用方式(`TEST_PG_DSN=postgres://... go test -run TestPGContract ./internal/services/operations/`)、只读保证。

提交前门禁: `go build ./...` + `go vet ./internal/services/operations/` + `go test ./internal/services/operations/`(本机无 TEST_PG_DSN → 新测试应显示 skip 且包整体全绿)。

**提交(锁决策 5):** 只 `git add internal/services/operations/pg_contract_test.go`,commit message: `test(operations): PG-gated 类型契约集成测试(TEST_PG_DSN)守护 ops List JOIN 的 uuid/varchar 比较方向`。
  </action>
  <verify>
    <automated>go test -v -run TestPGContract ./internal/services/operations/ 2>&1 | grep -E "SKIP|PASS|FAIL" && go build ./...</automated>
  </verify>
  <done>
    本机(无 TEST_PG_DSN)运行 `-run TestPGContract` 输出 SKIP 而非 FAIL,`go test ./internal/services/operations/` 全绿;文件包含全部 6 项调用(3 List + 3 Statistics)、空 DSN/连不通双 skip 分支、只读保证;该文件单独成 commit。
  </done>
</task>

</tasks>

<verification>
- `go build ./...` 零错误
- `go test ./internal/services/operations/` 全绿(新增测试在无 TEST_PG_DSN 环境下 skip)
- `grep -n "type:uuid" internal/models/workstation.go` 命中 BuildingID 行
- 有 PG 时端到端验证(编排者/用户执行): `TEST_PG_DSN="postgres://user:pass@10.62.10.34/ruoyi?sslmode=disable" go test -v -run TestPGContract ./internal/services/operations/` → 全 PASS
- `git log --oneline -2` 恰好两个新 commit,`git show --stat` 各自只含本任务文件
</verification>

<success_criteria>
1. `Workstation.BuildingID` gorm tag 为 `type:uuid`,与 DB 列真实类型一致,无 schema/数据变更,无 migration 文件
2. 全仓库审计复核通过:无按 varchar 方向比较 `sys_workstation.building_id` 的残留 SQL
3. `pg_contract_test.go` 存在:无 TEST_PG_DSN → skip;有 TEST_PG_DSN → 三服务 List+Statistics 真实打 PG 全部 err==nil
4. 恰好 2 个原子 commit,每个 commit 的 `git show --stat` 只含本任务声明文件,`.planning/` 下零文件入库
</success_criteria>

<output>
完成后创建 `.planning/quick/261015-ops-fk-drift-guard/261015-ops-fk-drift-guard-SUMMARY.md`(由编排者负责提交,executor 不 add)。
</output>
