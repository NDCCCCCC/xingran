---
phase: 148
plan: 01
wave: 2
depends_on: []
files_modified:
  - internal/services/operations/excel_export_config.go
  - internal/services/operations/excel_export_service.go
  - internal/services/operations/workstation_device_service.go
  - internal/services/operations/workstation_device_77_01_test.go
  - internal/services/operations/excel_export_chain_77_02_test.go
  - internal/services/operations/workstation_physical_diagnosis_test.go
  - .golangci.yml
  - .planning/phases/148-dialect-dialect-01-02-03-04/148-SUMMARY.md
autonomous: true
commit: false
---

# Phase 148: DIALECT 双方言 — 可直接执行计划

> **执行约定**：本 plan 在 W2 与 149 / 150 并行、三个 executor 写同一工作区。
> `commit:false` —— **禁止任何 git 写操作**（add / commit / stash / checkout / reset），由 Lead 在本波收口统一提交。
> 所有写操作严格限定在 `files_modified` 的独占文件面内（见「并行冲突面」硬约束）。

---

## 1. 必读顺序（开工前按序读完，均为只读）

| # | 文件 | 读什么 / 为什么 |
|---|------|-----------------|
| 1 | `.planning/STATE.md` | v1.55 active、W2 `148 ∥ 149 ∥ 150` 并行拓扑、79 条跟踪表保持 0 勾选 |
| 2 | `.planning/ROADMAP.md` | **Phase 148 详情 `:64-74`（5 条成功标准）**、并行冲突面/函数级归属表 `:113-148`、Constraints `:150-161`、禁区 `:173-181`、Known Gaps `:188-200` |
| 3 | `.planning/REQUIREMENTS.md` | **DIALECT-01~04 全文 `:38-44`** + 5 项裁定 `:7-17`（尤其 DECISION-CI-PG `:13`）+ Constraints `:98-106` + Traceability `:116-119` |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` | **§③-A `:78-84`**、**§③-B `:88-101`**、共性架构解 `:153-162`、两套改写范本与「不算根治」边界 `:179-184`、未决问题 `:194`、证据索引 `:211` |
| 5 | `CLAUDE.md` / `AGENTS.md` | 项目硬约定：Handler-Service / `response.*()` / SQLite+PG 双方言（`CAST AS TEXT`）/ e2e 归 `tests/e2e`、集成归 `tests/integration`、构建产物只落 `out/`、Excel 模板动态生成不落盘、Go 单测就近 `*_test.go` |
| 6 | 范本 A | `internal/core/db/sqlite_reconciliation_views.go:14-35`（方言翻译规则）、`:43-54`（等价性论证模板） |
| 7 | 范本 B（推荐） | `internal/services/operations/workstation_device_service.go:701-799` `diagnosePhysicalChain`、`:160-169` `physicalDevicesFetcher` 注入注释 |
| 8 | P1-8 既有改法 | `internal/services/operations/excel_query_builder.go:253-261`（双侧 `CAST AS TEXT`，**另一条独立路径，勿与 `resolveAssociations` 混为一谈**） |
| 9 | 代码现状 | `excel_export_config.go:12-20`（`JoinConfig.RightCast`）、`:351/355/359`（3 处配置）、`excel_export_service.go:103-189`（`resolveAssociations`）、`workstation_device_service.go:368-535` 与 `:2116-2234`（两处 CTE） |

> 读完如发现根因报告与代码现状冲突 → 立即停手，返回 `## PLANNING BLOCKED` 并列出具体冲突。
> **本 plan 起草时已逐项核对**：`RightCast` 字段 / 3 处配置 / `::` 拼接 / `logger.Warnf+continue` / `diagnosePhysicalChain` / `physicalDevicesFetcher` / 4 处 `DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `id::text` / 3 个测试文件均存在且与报告一致，**无冲突**。

---

## 2. 目标

把 LOCKED 残余 ③（`RightCast:"text"` ③-A + PG 专属 CTE ③-B）**从根源**修掉，使「SQLite+PG 双方言」由靠约定升级为靠机制：

1. **DIALECT-01**：删除「配置承载 `"::"` 方言片段」的机制，改语义枚举 + service 统一单侧 `CAST(id AS TEXT)`；消除「关联列整列空但导出仍成功」的静默降级。
2. **DIALECT-02**：两处 recall 路径从根源消除 PG 专属 raw CTE，改 GORM 链式查询 + Go 层合并，单条/批量口径不分叉。
3. **DIALECT-03**：改写 4+ 个「报错即预期」测试为真实结果断言，`SkipSoftDelete` 两处语义保留。
4. **DIALECT-04**：`forbidigo` 方言 token 门禁（存量白名单逐步收敛）+ `TEST_PG_DSN` 双跑登记为前置依赖，**不伪造双跑**。

**「根治」判定线**（按用户「不打补丁」标准）：方言 token 归零、静默降级消除、测试真断言、门禁准入；**加 `isPostgres` 方言分支 / CTE 下沉视图 / 一刀切禁 raw SQL / 伪造 PG 双跑 一律不算根治**。

---

## 3. 已确认事实（直接作为执行输入，禁止重新调研）

### ③-A `RightCast:"text"`
- `excel_export_config.go:12-20` 定义 `JoinConfig.RightCast string`；**仅 3 处使用**：`:351`（`sys_workstation` workstationName，带 `SkipSoftDelete:true`）、`:355`（`sys_network_device` deviceName）、`:359`（`sys_device_port_status` portName，带 `SkipSoftDelete:true`）。
- `excel_export_service.go:143-149` 把 `RightCast` 拼成 `rightFieldExpr = "id::text"`；`:153-159` 用于 `Table(join.Table).Select(...).Where(rightFieldExpr+" IN ?", ids)`。
- PG 下 uuid 列与 varchar 左值需类型对齐；SQLite 下 `::` 报 `unrecognized token ":"`，该错误被 `:160-165` 的 `logger.Warnf + continue` **静默吞掉 → 关联列整列空但导出仍成功**。
- `resolveAssociations`（`:103`）返回 `error`，调用方 `:67-69` 已 `fmt.Errorf("解析关联数据失败: %w", err)` 包装传播 → **去掉 `Warnf+continue`、改 `return err` 可直接落地，无需改调用方**。
- P1-8 已把**另一条** `excel_query_builder.go:253-256` 路径改为双侧 `CAST AS TEXT`；`resolveAssociations` 是独立路径。**根治 = 删 `RightCast` 承载方言片段的机制，改语义枚举 + service 统一单侧 `CAST(id AS TEXT)`**（`IN ?` 参数是 Go 字符串，不需要双侧；P1-8 双侧是因 JOIN 两边都是列）。
- `SkipSoftDelete:true` 两处语义必须保留：`:351`「109 个软删工位恢复名称显示」、`:359`「`sys_device_port_status` 是硬删除表、无 `deleted_at` 列」。

### ③-B PG 专属 CTE
- `workstation_device_service.go:368-535`（`GetPhysicalDevices` 单工位）、`:2116-2234`（`GetPhysicalDevicesByWorkstations` 批量）用 `db.Raw(rawSQL, ...)`，含 **4 处 `DISTINCT ON`**（`:453/470/484/497`、批量段对应）、`REGEXP_REPLACE`（接口名 `'\s+'` 与 MAC `'[.:\-]'` 归一化）、`NULLS LAST`（`:465/481/495/533/534`）、6+ 处 `id::text`（`:438/450/521/524/2164/2168` 等）。
- 现有 SQLite 测试**故意把报错当预期**：`workstation_device_77_01_test.go:9-11`（「勿为覆盖率改 SQL」注释）、`:568-620`（4+ 个 `require.Error`）、`excel_export_chain_77_02_test.go:458-480`、`workstation_physical_diagnosis_test.go:401-403`。
- **两套现成改写范本**：
  1. `internal/core/db/sqlite_reconciliation_views.go:14-35`：`DISTINCT ON` → `ROW_NUMBER() OVER(PARTITION BY...)+WHERE rn=1`；`REGEXP_REPLACE` → 嵌套 `REPLACE`；`NULLS LAST` → `(x IS NULL), x`；等价性论证模板 `:43-54`（组内最长前缀优先 / 组间包含关系 / 逐组核对）。**保留单条 SQL 性能的中间项**。
  2. `workstation_device_service.go:701-799` `diagnosePhysicalChain`：**刻意用 GORM chainable + Go 侧集合求交，避免 PG 专属 CTE**；`:160-169` 的 `physicalDevicesFetcher` 注入注释保证「不存在第二套 recall 逻辑、口径不分叉」。**推荐此方案（从根源消除 raw SQL）**。
- **加 `isPostgres` 方言分支是补丁不是根治**（两套口径必分叉，被 `:166-168` 注释反对）；CTE 带 `?` 参数、视图无法参数化 → **下沉双方言视图不适用**。
- CTE 结果口径：MAC 归一化（去 `[.:\-]` + LOWER）、接口名归一化（LOWER + 去 `\s+` + 前缀折叠）、置信度 `confidence DESC`、Top-1 去重、`collected_at/last_seen/updated_at DESC NULLS LAST`、`HistoryLastSeen`。
- **4+ 个「报错即预期」测试必须同步改写为真实结果断言**，否则根治后测试红。

### 防复发 DIALECT-04
- 用户裁定 **CI-PG 本期不建 `services: postgres`**：`.github/workflows/ci.yml:68/77` 明确 `TEST_PG_DSN` unset、PG contract 只做「skip 可观测」，**零真实 PG 验证**。
- 存量规模：406 Go 文件含 `.Raw(`、58 含 `::text/::uuid`、22 含 `DISTINCT ON` → **不得一刀切禁全部 raw SQL**，只能禁方言 token 且**存量先白名单逐步收敛**。
- `.golangci.yml` 当前为 v2 语法（`version:"2"` / `linters.default: none` / `enable: [staticcheck]` / `linters.exclusions.rules`），`forbidigo` 规则须按 v2 `linters.settings.forbidigo` + `linters.enable` + 存量白名单（exclusions）写法落。

### 并行冲突面（硬约束，唯一并行许可凭证 = ROADMAP `:119-137` 函数级归属表）
- W2 `148 ∥ 149 ∥ 150`，三 phase 写集合两两交集 = ∅ 方可并行。
- **148 独占**：`excel_export_config.go`、`excel_export_service.go`、`workstation_device_service.go`、4+ 测试、`.golangci.yml`。
- **148 不得碰**：`excel_service.go`（≠ `excel_export_service.go`，归 150）、`workstation_service.go`（150）、`scope_write.go`（150）、`useBackfillDept.tsx`（150）、`wall/door/floor_plan_text/room_photo/location_alias_service.go` + `scope_entities.go` + `internal/api/router.go` + `useFloorPlanEditor.ts` + `133-UAT.md`（149）、`scope_filter.go` + `base/GORMRepository`（151）、`internal/utils/operlog/**` + `scripts/**` + `tests/e2e/**` + `scripts/README.md` + `.github/workflows/ci.yml`（145）、`internal/core/security/**`（146）、`internal/services/asset/**`（147）。
- **`.golangci.yml` 归 148、`ci.yml` 归 145**：148 **只改 lint 配置、不并发改 `ci.yml`**（ROADMAP `:148` 硬登记；`lint:ci` 由 145 预留步骤名接入，148 只提供规则）。若实测必须改 `ci.yml`，在 145 落定后由 148 单独追加 commit，**绝不并发改同一文件**。

---

## 4. Lead 裁决表（执行期不得翻案）

| 裁决项 | 裁定 | 依据 / 边界 |
|--------|------|-------------|
| ③-B 改写方案 | **推荐方案 ②**：拆多次 GORM 链式查询 + Go 层归一化/去重/排序，复用 `diagnosePhysicalChain` / `physicalDevicesFetcher`，**从根源消除 raw SQL**；方案 ①（窗口函数）仅作保留单条 SQL 性能的中间项 | 根因报告 `:93-95`；`physicalDevicesFetcher` 注入保证单条/批量不分叉 |
| 方言分支 | **禁** `isPostgres` 方言分支 / 两套 SQL | 两套口径必分叉，是补丁不是根治（ROADMAP `:175`；`:166-168` 注释） |
| CTE 下沉 | **禁** 下沉双方言视图 | CTE 带 `?` 参数，视图无法参数化 |
| 双方言对齐 | 禁 `::uuid` / `ILIKE` / 裸 `ON CONFLICT`；uuid/varchar 用 `CAST AS TEXT`；本 phase 导出路径**单侧**足够 | REQUIREMENTS `:100`；根因报告 `:82` |
| PG 双跑 | **禁伪造**：不建 `services: postgres`、不注入 `TEST_PG_DSN`、不写假双跑证据；**只登记前置依赖** | DECISION-CI-PG（REQ `:13`、ROADMAP `:169`、`:198`） |
| `.golangci.yml` 归属 | **归 148**；`ci.yml` 归 145，148 不动 `ci.yml` | ROADMAP `:147-148` 全局唯一归属硬登记 |
| forbidigo 范围 | **只禁 8 类方言 token + 存量白名单逐步收敛**；不得一刀切禁全部 `.Raw(` | 406 文件含 `.Raw(`，REQ `:44`、ROADMAP `:73` |
| 静默降级 | 关联查询失败必须**可见/返回 error**，不得「整列空仍导出成功」 | DIALECT-01（REQ `:40`、ROADMAP `:70`） |
| git | **禁所有 git 写操作**；`commit:false`，Lead 统一提交 | 三 executor 并行同工作区 |
| 兼容 | `SkipSoftDelete` 两处语义保留；零新外部依赖；不动 79 条跟踪表 / LOCKED 裁定原文 | REQ `:104`、ROADMAP `:158` |

---

## 5. 任务

<task id="1">
  <title>T1 · DIALECT-01：删除 RightCast 方言机制 + 消除静默降级</title>
  <files>
    internal/services/operations/excel_export_config.go
    internal/services/operations/excel_export_service.go
    就近新增/改写 internal/services/operations/excel_export_*_test.go（DryRun 静态断言 + SQLite 实跑断言）
  </files>
  <action>
    1. **删机制**：从 `excel_export_config.go:12-20` 的 `JoinConfig` **删除 `RightCast string` 字段及其注释**；删除 `:351 / :355 / :359` 三处 `RightCast: "text"` 配置。
    2. **改语义枚举**：在 `JoinConfig` 上新增语义枚举（例如 `type RightFieldKind string` + 常量 `RightFieldText` / `RightFieldUUID`），表达「右表字段是 UUID 列、需转 TEXT 比较」这一**语义**，而非承载 `"text"` / `"::"` 方言片段。**任何配置/结构体字段都不得出现 SQL 片段字符串**（`::` / `CAST` 字面量）。三处 infoPoint 关联（workstation / network_device / device_port_status）改用该枚举标注。
       - 保留 `SkipSoftDelete bool` 及两处 `true` 语义（`:351` 软删恢复名称、`:359` 硬删表无 `deleted_at`），注释原文不删。
    3. **service 统一单侧 CAST**：`excel_export_service.go:143-149` 删除 `RightCast` 拼接分支，改为由 `resolveAssociations` 统一生成 `rightFieldExpr = "CAST(" + join.RightField + " AS TEXT)"`（**单侧**，`IN ?` 参数是 Go 字符串；SQLite 接受 `CAST(x AS TEXT)`，PG 接受 `CAST(uuid AS TEXT)`）。`Select(join.RightField+", "+join.SelectField)` 保持不变；`SkipSoftDelete` 的 `deleted_at IS NULL` 分支（`:157-159`）保持不变。
    4. **去掉静默降级**：`:160-165` 的 `rows, err := queryBuilder.Rows()` 之后，把 `if err != nil { logger.Warnf(...); continue }` **改为 `return nil, fmt.Errorf("关联表 %s 查询失败: %w", join.Table, err)`**（调用方 `:67-69` 已包装传播，导出不再「整列空仍成功」）。`rows.Scan` 出错（`:169-171`）同理由 `continue` 改为 `return nil, fmt.Errorf(...)`，不得静默吞。`defer rows.Close()` 保持/补齐。
    5. **P1-8 式 DryRun 静态断言**（就近 `*_test.go`）：构造 infoPoint 导出配置，用 GORM `Session{DryRun: true}` 捕获 `resolveAssociations` 实际生成的 SQL，断言：① SQL **不含 `::`**；② 含 `CAST(... AS TEXT)`；③ 三处关联 `IN ?` 子句存在；④ 同步断言全仓 `excel_export_config.go` / `excel_export_service.go` 源码不含 `::` 字面量（防 `RightCast` 机制回流）。
    6. **SQLite 实跑断言**（就近 `*_test.go`，`t.TempDir()` 自建 SQLite、不落盘 `.xlsx`）：insert 3 组关联 fixture（含 1 个软删工位对应 `:351` 语义、1 个硬删表 `sys_device_port_status` 对应 `:359` 语义），执行导出路径，断言 **workstationName / deviceName / portName 三个关联列全部非空且值正确**（逐字段），并断言 `err == nil`（不再有静默吞错误）。
       - 追加**反例断言**：人为让关联查询报错（如临时改表名为不存在），断言 `resolveAssociations` **返回 error** 而非「日志 + 空列」。
  </action>
</task>

<task id="2">
  <title>T2 · DIALECT-02：两处 CTE 改 Go 层合并（方案 ②），方言 token 归零</title>
  <files>
    internal/services/operations/workstation_device_service.go
  </files>
  <action>
    1. **实施范围**：`GetPhysicalDevices`（`:368-535`）与 `GetPhysicalDevicesByWorkstations`（`:2116-2234`）两段 `db.Raw(rawSQL, ...)` CTE **整体删除**，改为 GORM 链式查询 + Go 层归一化/去重/排序/合并。
    2. **复用单一 recall 逻辑**（硬约束）：抽取/复用 `diagnosePhysicalChain`（`:701-799`）的 GORM chainable + Go 侧集合求交范式，把「MAC→port→infoPoint→workstation」反推链做成**一份**共享实现（单工位与批量都调用同一入口，批量按 workstationID 分组后套用同一套归一化/去重/Top-1 逻辑），确保「**不存在第二套 recall 逻辑、口径不分叉**」。`physicalDevicesFetcher`（`:160-169`）注入语义保留，生产路径恒为 nil。
       - 若需新增私有 helper，落在本文件内；**不得新建跨 phase 文件、不得改 `base/GORMRepository`**。
    3. **逐条对齐 PG 版口径**（用 `sqlite_reconciliation_views.go:43-54` 的等价性论证模板，逐条写成注释/测试断言）：
       - **MAC 归一化**：PG `LOWER(REGEXP_REPLACE(COALESCE(mac,''),'[.:\-]','','g'))` → Go `strings.ToLower` + 去除 `.` `:` `-`（等价性：字符集合逐字符核对）。
       - **接口名归一化**：PG `REGEXP_REPLACE(LOWER(iface),'\s+','','g')` + 前缀折叠 → Go 侧等价折叠。**`\s+` 与「嵌套 REPLACE 只去空格」的已接受偏差**按 `sqlite_reconciliation_views.go:27-28` 口径登记（接口名实际不含制表符/换行，dev 可接受），写入 SUMMARY Known Gaps。
       - **Top-1 去重**：PG `DISTINCT ON (mac, device_id, interface_name) ... ORDER BY collected_at DESC NULLS LAST` → Go 按 `(mac, device_id, interface_name)` 分组、以 `collected_at` 降序取首条（NULL 排最后，对齐 `NULLS LAST`），再用 `confidence DESC NULLS LAST` 取最终 Top-1（对齐 `:533-534`）。
       - **HistoryLastSeen**：按 `last_seen DESC NULLS LAST` / `updated_at DESC NULLS LAST` 逐字段取，保证与 PG 版一致。
       - **置信度**：`confidence` 排序方向、NULL 位置与 PG 版 `:533` 一致。
       - **字段合并「资产优先」**：对齐 `:400` 注释的 `COALESCE(a.devicesn, mac/mac_address, hist/mac_address)` 等资产字段优先规则。
    4. **方言 token 归零**：改写后 `:368-535` 与 `:2116-2234` 范围内 `DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `::text` / `::uuid` **全部为 0**（本文件其余注释中如仍提及这些 token，改为「历史 CTE 已移除」的说明，不保留可被执行命中的 SQL 片段）。用静态断言（T5）守护。
    5. **不加** `isPostgres` 分支、**不下沉视图**、**不新增外部依赖**、**不启动长驻进程**。
  </action>
</task>

<task id="3">
  <title>T3 · DIALECT-03：改写 4+ 个「报错即预期」测试为真实结果断言</title>
  <files>
    internal/services/operations/workstation_device_77_01_test.go
    internal/services/operations/excel_export_chain_77_02_test.go
    internal/services/operations/workstation_physical_diagnosis_test.go
    其余执行期 grep 命中的「报错即预期」*_test.go（就近包内）
    （如需）tests/integration/ 下新增 *_test.go
  </files>
  <action>
    1. **全量定位**：先 `grep -rn "require.Error\|assert.Error\|勿为覆盖率\|unrecognized token\|PG.*CTE\|报错" internal/services/operations/*_test.go`，列出全部「把 PG CTE 报错当预期」的断言（REQUIREMENTS/ROADMAP 点名 3 个文件 4+ 处：`workstation_device_77_01_test.go:9-11,568-620`、`excel_export_chain_77_02_test.go:458-480`、`workstation_physical_diagnosis_test.go:401-403`），逐一登记，**不得遗漏**。
    2. **改写为真实结果断言**：删除「预期 `err != nil` / 预期 SQLite 报错」的断言与 `:9-11`「勿为覆盖率改 SQL」类注释，改为对**逐字段真实结果**断言：
       - 单工位 `GetPhysicalDevices` 与批量 `GetPhysicalDevicesByWorkstations` 对同一 fixture 返回结果一致（设备数、Each `Devicesn / MacAddress / InterfaceName / Confidence / HistoryLastSeen` 逐字段）；
       - MAC/接口名归一化结果、置信度 Top-1、NULL 排序与 PG 版口径一致（用 T2 的等价性论证表逐项对照）；
       - 断言 `err == nil`（根治后不应再报错）。
    3. **SQLite 与 PG 逐字段一致性**：SQLite 用内存/temp DB fixture 实跑；PG 侧因 `TEST_PG_DSN` 本期 unset，**只保留「skip 可观测」契约测试原样、不伪造 PG 结果**；把「PG 版期望值」以 fixture 常量（从原 CTE 语义推导）固化进 SQLite 断言，保证口径对齐可核对。
    4. **`SkipSoftDelete:true` 两处语义保留**：新增/保留测试覆盖 `:351` 软删工位名称恢复（软删行仍能被关联到）与 `:359` 硬删表 `sys_device_port_status` 不加 `deleted_at IS NULL`（不报 SQLSTATE 42703）。
    5. **分层**：Go 单测就近放业务包 `internal/services/operations/*_test.go`；跨包/多表装配的集成测试放 `tests/integration/*_test.go`（AGENTS.md 分层：禁止混入 `tests/e2e/`，禁止 `test/` 单数目录，禁止 `test_` 前缀源码）。**不改 `tests/e2e/**`（归 145）**。
    6. 若改写涉及 `excel_export_chain_77_02_test.go` 的导出链断言，与 T1 的 SQLite 实跑断言衔接，避免重复 fixture 冲突（三 executor 同工作区，注意测试函数名不重名）。
  </action>
</task>

<task id="4">
  <title>T4 · DIALECT-04：forbidigo 方言门禁（存量白名单）+ PG 双跑前置登记</title>
  <files>
    .golangci.yml
  </files>
  <action>
    1. **启用 `forbidigo`**（golangci-lint v2 语法，沿用现有 `version:"2"` / `linters.default: none` / `enable` / `exclusions` 结构）：
       - 在 `linters.enable` 追加 `forbidigo`；
       - 在 `linters.settings.forbidigo` 配置 `forbid` 正则，覆盖 **8 类方言 token**：`::text` / `::uuid`（含通配 `::\w+`）、`DISTINCT ON`、`REGEXP_REPLACE`、`NULLS LAST`、`ILIKE`、`ON CONFLICT`（REQ `:44`、ROADMAP `:73` 口径 7~8 类，按枚举全覆），并 `exclude_godoc_examples: false`；
       - **配置里禁止出现可拼装的 SQL 片段原语之外的裸 SQL**，语义原语若抽取（见第 4 点）不得在配置里写 SQL 字面量。
    2. **存量白名单逐步收敛（硬要求）**：
       - 先用 `grep` 枚举实际命中文件（406 文件含 `.Raw(`、58 含 `::text/::uuid`、22 含 `DISTINCT ON`），**只把命中方言 token 的存量文件列入 `linters.exclusions.rules` 的 `path` 白名单**，逐文件写注释说明「存量待收敛」；
       - 本 phase 改造过的 `excel_export_config.go` / `excel_export_service.go` / `workstation_device_service.go` **不得进白名单**（它们应已归零）;
       - **不得一刀切禁全部 raw SQL** —— `forbidigo` 只匹配方言 token，不写 `\.Raw\(` 规则；保留其它 400+ 文件合法 `.Raw(` 使用。
       - 追加一条**反向门禁自检**：新增方言片段到任意非白名单 Go 文件时 `golangci-lint run` 必红（T5 变异验证）。
    3. **`TEST_PG_DSN` 双跑只登记前置依赖、不伪造**：
       - **不改 `.github/workflows/ci.yml`**（归 145，ROADMAP `:148` 硬登记）；不加 `services: postgres`、不注入 DSN、不写「已双跑」证据；
       - 在 **T6 的 SUMMARY** 显式写「需新建 PG service + 迁移引导后方可升级，本期 `TEST_PG_DSN` unset、零真实 PG 验证、PG contract 仅 skip 可观测」。
    4. **视预算抽 `dialect` 语义原语**（可选，非阻断）：如预算允许，在 `internal/services/operations/` 就近抽 `CastToText` / `Top1PerGroup` / `NormalizeMac` 等**语义** helper（返回 `clause.Expr` 或 Go 值，封装 `CAST AS TEXT`），供 T1/T2 复用；**禁止在配置/结构体里出现 SQL 片段字符串**。不抽取不阻断 DIALECT-04 主项（forbidigo 门禁 + 前置登记必做）。
  </action>
</task>

<task id="5">
  <title>T5 · 自验证（只跑相关包，不跑全仓）+ 反向变异红绿</title>
  <files>
    （只读验证，不改文件；变异为临时改动后立即还原）
  </files>
  <action>
    1. **相关包测试**（按实际包，不跑全仓 `go test ./...` —— 三 executor 并行同工作区，全仓会读到其它 phase 未完成文件）：
       - `go test -count=1 ./internal/services/operations/` （主包）
       - `go test -count=1 ./internal/services/operations/...` （若存在子包；按 `go list` 实际结果调整）
       - 若新增 `tests/integration/` 用例：`go test -count=1 ./tests/integration/`
    2. **构建检查**：`go build ./...`（AGENTS.md/CLAUDE.md 硬约定：改 Go 后必跑；产物不落根目录、无需 `-o`，`go build` 不落盘二进制）。若构建因其它 phase 并行中途文件报错，记录并待 Lead 集成期复核，**不擅自改其它 phase 文件**。
    3. **反向变异（红绿验证，必须还原）**：
       - 把 T1 的 `CAST(id AS TEXT)` 临时改回 `"id::text"`，断言 ① DryRun 静态断言（导出路径 SQL 不含 `::`）**必红**；② `golangci-lint run --timeout=5m ./internal/services/operations/...` 的 `forbidigo` **必红**；然后**还原为 `CAST` → 复跑全绿**。
       - 把 `resolveAssociations` 的 `return err` 临时改回 `logger.Warnf + continue`，断言 T1 反例断言（关联查询失败应返回 error）**必红**，还原 → 绿。
    4. **lint 门禁**：`golangci-lint run --timeout=5m ./internal/services/operations/...` 全绿（含新增 forbidigo）；确认存量白名单文件不误报、改造过的 3 个文件不在白名单。
    5. **`node scripts/check-route-registry.mjs` 若相关**：本 phase 未动路由/router.go，默认**不必跑**；仅当执行期意外触及路由登记时再跑并确认零漂移。
    6. 汇总 T1~T5 全部命令与红绿结果，写入 T6 SUMMARY「验证结果」段。
  </action>
</task>

<task id="6">
  <title>T6 · 写 148-SUMMARY.md（改动 / 4 REQ / 验证 / 方案②与性能 / Known Gaps / 未提交）</title>
  <files>
    .planning/phases/148-dialect-dialect-01-02-03-04/148-SUMMARY.md
  </files>
  <action>
    写入简体中文 SUMMARY，必须包含：
    1. **改动清单**：按 `files_modified` 逐文件列改动（`RightCast` 删除 / 语义枚举 / 单侧 CAST / 静默降级消除 / 两处 CTE Go 层合并 / 4+ 测试改写 / forbidigo 规则与白名单）。
    2. **4 REQ 满足情况**：DIALECT-01/02/03/04 逐条对照 REQUIREMENTS `:40-44` 与 ROADMAP `:70-74` 5 条成功标准，标 `satisfied` + 证据（命令输出 / 断言名）。
    3. **验证结果**：T5 全部命令（相关包 `go test -count=1` / `go build ./...` / lint / 反向变异红绿 / check-route-registry 是否跑）与结论；明确「只跑相关包、未跑全仓，全仓矩阵由 Lead 集成期统一执行」。
    4. **方案 ② 与性能评估**：说明采用「GORM 链式 + Go 层合并」而非窗口函数方案 ① / 方言分支的理由；评估批量 `GetPhysicalDevicesByWorkstations` 的拉取量、内存占用、N+1 风险与缓解（批量预取 + 内存分组），登记「生产 MAC/AD/资产表行数待确认」（根因报告未决问题 `:194`）。
    5. **Known Gaps**（明确登记，非阻断）：
       - **CI PG 双跑前置依赖**：`ci.yml` 无 `services: postgres`、`TEST_PG_DSN` unset；需「新建 PG service + 迁移引导」后方可把 3 个 contract 测试从 skip 可观测升级为双跑硬门禁；本期**不伪造双跑**。
       - **`REGEXP_REPLACE \s+` 与嵌套 REPLACE 的已接受偏差**：按 `sqlite_reconciliation_views.go:27-28` 口径登记（接口名实际不含制表符/换行）。
       - **`IN (?)` 参数上限**：Go 字符串 ids 批量 `IN ?` 在超大 ids 集下的参数上限/分片风险，登记为后续优化项（本期不引入新机制）。
    6. **明确「未 git commit」**：写「本 plan `commit:false`，所有改动留工作区，由 Lead 在 W2 收口统一提交；未执行任何 git add / commit / stash / checkout」。
    7. **不写**：不动 79 条人工验证跟踪表、不改 LOCKED 裁定原文、不改 `ci.yml`、不新增外部依赖、不写生产数据。
  </action>
</task>

---

## 6. 验证标准（must_haves，逐条对应 ROADMAP Phase 148 `:70-74` 5 条成功标准）

- [ ] **MH-1（对应标准 1 · DIALECT-01）** `JoinConfig.RightCast` 字段与 3 处 `RightCast:"text"` 配置删除；`resolveAssociations` 统一 `CAST(id AS TEXT)` **单侧**；`logger.Warnf + continue` 静默降级删除、关联查询失败**返回 error**（不得整列空仍导出成功）；3 个关联列 SQLite 实跑**全部非空且逐字段正确**；P1-8 式 DryRun 静态断言「导出路径 SQL 不含 `::`」通过；`SkipSoftDelete:true` 两处语义保留。
- [ ] **MH-2（对应标准 2 · DIALECT-02）** 两处 recall 路径（`GetPhysicalDevices` / `GetPhysicalDevicesByWorkstations`）的 `DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `id::text` **静态归零**；单条/批量复用同一份 recall 逻辑、口径不分叉；对同一 fixture 的 MAC 归一化 / 接口名归一化 / 置信度 Top-1 / `HistoryLastSeen` 逐字段与 PG 版口径对齐（用 `sqlite_reconciliation_views.go:43-54` 等价性论证模板逐条比对照表）。
- [ ] **MH-3（对应标准 3 · DIALECT-03）** 4+ 个「把 PG CTE 报错当预期」测试（含 `workstation_device_77_01_test.go:9-11,568-620`、`excel_export_chain_77_02_test.go:458-480`、`workstation_physical_diagnosis_test.go:401-403` 及 grep 全量命中）**全部改写为真实结果断言**、`SkipSoftDelete` 语义测试保留；`go test -count=1 ./internal/services/operations/` 全绿。
- [ ] **MH-4（对应标准 4 · DIALECT-04①②）** `.golangci.yml` 的 `forbidigo` 禁 `::text|::uuid|DISTINCT ON|REGEXP_REPLACE|NULLS LAST|ILIKE|ON CONFLICT`，**存量白名单逐文件列出实际命中并注释「待收敛」**、改造过的 3 个文件不在白名单、**不得一刀切禁全部 `.Raw(`**；`lint:ci` 接入后新增方言片段（反向变异）必红、还原必绿；视预算抽 `dialect` 语义原语且配置里无 SQL 片段。
- [ ] **MH-5（对应标准 5 · DIALECT-04③）** `TEST_PG_DSN` 双跑在 SUMMARY 显式登记为**前置依赖**（需新建 PG service + 迁移引导后方可升级）；本期**不建 `services: postgres`、不改 `ci.yml`、不伪造双跑**；**不加** `isPostgres` 方言分支、**不把** CTE 下沉双方言视图。

---

## 7. 禁止（执行期红线）

1. **git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git reset` / 任何分支写 —— 全禁（`commit:false`，Lead 统一提交）。
2. **改其它 phase 文件面**：尤其 `excel_service.go`（≠ `excel_export_service.go`）/ `workstation_service.go` / `scope_write.go` / `useBackfillDept.tsx`（归 150）；`wall/door/floor_plan_text/room_photo/location_alias_service.go` + `scope_entities.go` + `internal/api/router.go` + `useFloorPlanEditor.ts` + `133-UAT.md`（归 149）；`scope_filter.go` + `base/GORMRepository`（归 151）；`internal/utils/operlog/**` + `scripts/**` + `tests/e2e/**` + `scripts/README.md` + `.github/workflows/ci.yml`（归 145）；`internal/core/security/**`（146）；`internal/services/asset/**`（147）。**`.golangci.yml` 归 148、`ci.yml` 归 145，148 只改 lint 配置不并发改 `ci.yml`**。
3. **加 `isPostgres` 方言分支 / 两套 SQL**（补丁不是根治）。
4. **CTE 下沉双方言视图**（带 `?` 参数，视图无法参数化）。
5. **改 `ci.yml`**（含加 `services: postgres` / 注入 `TEST_PG_DSN` / 伪造双跑证据）。
6. **新增外部依赖**（零新依赖，复用现成范本）。
7. **启动长驻进程**（dev server / watcher / DB 一律不起；限时任务用受管后台）。
8. **写生产数据**（不连生产库、不写 `sys_*` 生产表）。
9. **动 79 条人工验证跟踪表与 LOCKED 裁定原文**（`.planning/notes/261108-v153-locked-residuals.md` 等归档件只读，保持 0 勾选）。
10. **一刀切禁全部 raw SQL**（406 文件含 `.Raw(`）、**Excel 模板落盘 `.xlsx`**、**构建产物落仓库根**、**新建 `test/` 单数目录**、**把 `*_test.go` 放进 `tests/e2e/` 充当源码**。

---

*Plan 起草：gsd-planner · Phase 148 · wave 2 · depends_on: [] · autonomous: true · commit: false*
*事实基线：STATE.md(v1.55 active) / ROADMAP.md:64-74,113-148 / REQUIREMENTS.md:38-44,98-106 / 根因报告 §③-A/:78-84 §③-B/:88-101 共性架构解/:153-162；代码现状已逐项核对无冲突。*
