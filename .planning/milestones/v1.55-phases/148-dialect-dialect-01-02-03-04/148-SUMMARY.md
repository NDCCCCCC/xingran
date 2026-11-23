# Phase 148 SUMMARY：DIALECT 双方言（DIALECT-01/02/03/04）

**执行**：gsd-executor exec-148 · wave 2 · `commit:false` · 2026-11-09
**结论**：`## PHASE 148 COMPLETE` — 4 个 REQ 全部 satisfied，5 条 must_haves 全部通过。
**未 git commit**：本 plan `commit:false`，所有改动留工作区，由 Lead 在 W2 收口统一提交；
**未执行任何** `git add` / `git commit` / `git stash` / `git checkout` / `git reset` / 任何分支写。
并行期间每步保持 `go build ./internal/services/operations/` 可编译（曾出现一次 `sort` import 瞬时中间态，
2 分钟内修复并通报 Lead / exec-150；后续采用「无 import 变更」写法彻底避免中间态）。

---

## 1. 改动清单（按 `files_modified` 逐文件）

| 文件 | 改动 |
|---|---|
| `internal/services/operations/excel_export_config.go` | 删 `JoinConfig.RightCast string` 字段及 3 处 `RightCast:"text"` 配置；新增语义枚举 `type RightFieldKind string` + `RightFieldText` / `RightFieldUUID`；`JoinConfig.RightFieldKind` 表达「右表字段是 UUID 列、需转 TEXT 比较」语义；3 处 infoPoint 关联（workstation / network_device / device_port_status）改用 `RightFieldUUID`；`SkipSoftDelete:true` 两处及注释原文保留（`:351` 软删工位恢复名称、`:359` 硬删表无 `deleted_at` 列）。配置/结构体零 SQL 片段字符串。 |
| `internal/services/operations/excel_export_service.go` | `resolveAssociations` 删方言拼接分支，改由 `RightFieldKind==RightFieldUUID` 统一生成 `rightFieldExpr = "CAST(" + join.RightField + " AS TEXT)"`（**单侧**，`IN ?` 参数是 Go 字符串）；删 `logger.Warnf + continue` 静默降级 → `Rows()` / `Scan` / `rows.Err()` / `rows.Close()` 出错一律 `return nil, fmt.Errorf("关联表 %s ...: %w", ...)`（调用方 `ExportData:67-69` 已 `%w` 包装传播）；补 `rows.Err()` 检查与显式 `Close`。 |
| `internal/services/operations/workstation_device_service.go` | 两处 `db.Raw(rawSQL, ...)` PG 专属 CTE（`GetPhysicalDevices` / `GetPhysicalDevicesByWorkstations`）**整体删除**；新增**唯一一份**共享召回 `recallPhysicalRows(ctx, db, wsIDs, statusNormal)`（多次 GORM 链式查询 + Go 层归一化/分组 Top-1/LEFT JOIN 等价合并/最终去重排序），单工位与批量都调用它、批量按 wsID 分组，口径不分叉；新增 `normalizePhysicalMAC` / `normalizePhysicalIface` / `physicalNormColSQL` / `physicalTimeGreater` / `buildPhysicalRecallRow` / `buildPhysicalDevice` / `physicalStableSortRows` 等私有 helper（全部落在本文件内，未新建跨 phase 文件）；`physicalDevicesFetcher` 注入语义保留、生产恒为 nil；过时注释改「历史 CTE 已移除」；**全文件 8 类方言 token 静态归零**（含注释，见 T2 等价性论证的语义化改写）。 |
| `internal/services/operations/workstation_device_77_01_test.go` | 删 `:9-11`「勿为覆盖率改 SQL」注释；新增 `wsd77SeedPhysicalChain` / `wsd77AssertPhysicalDevice` helper；`TestWSD77_GetPhysicalDevices_FrontSegment`、 `TestWSD77_GetPhysicalDevicesByWorkstations_FrontSegment` 由「越过早退后 PG CTE 报错即预期」改写为「建齐链路表 + `require.NoError` + 1 台设备逐字段真实结果断言 + 单条/批量逐字段一致」。 |
| `internal/services/operations/excel_export_chain_77_02_test.go` | 头注释改双方言说明；`TestExp77_ExportData_WorkstationAppendSheets` 补建链路表，物理链路 sheet 由「physErr 降级 0 行」改「真实 1 数据行 + 逐字段抽查（序列号/型号/MAC/IP/Confidence）」；`TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 更名 `..._FixtureBoundary`，明确「夹具缺 ops_floors/ops_buildings 表」而非方言报错，并断言错误不含 `unrecognized token` / `syntax error`。 |
| `internal/services/operations/workstation_physical_diagnosis_test.go` | `newDiagnosisTestService` 不再注入替代召回（直接返回真实 svc），`sqlitePhysicalRecall` 降级为**独立 oracle**；`TestGetPhysicalDevices_LegacySignatureUnchanged` 由「编译期签名 + 委托关系」扩为「签名 + 参数校验 + **真实结果**（`require.NoError` / 1 台 / MAC 逐字段 / 与独立 oracle 一致）+ 诊断 Total 同源」四重守护。 |
| `internal/services/operations/workstation_device_physical_test.go` | `TestGetPhysicalDevices_UserIDEmpty` 由「只建 sys_workstation 表、缺表/PG 语法报错即证明越过 user_id 早退」改写为「建齐 8 张链路表 + 完整链路 + `require.NoError` + 1 台设备逐字段（MAC / Confidence=1.0 / 资产 devicesn / ResponsibleUserID=nil）」；`normalizeIface` 注释改对齐 Go 侧共享实现。 |
| `internal/services/operations/excel_export_dialect_148_test.go`（新增） | T1/T2/T3/T4 配套回归守护：① 源码静态断言（3 个生产文件零方言 token / 无 `RightCast` / 有 `RightFieldKind` / 有 `CAST(` / 有 `recallPhysicalRows` / 无 `WITH workstation_ports`）；② DryRun 式 SQL 捕获断言（真实 SQLite + GORM Query/Row After 回调，断言 SQL 无 `::`、含 `CAST(... AS TEXT)`、3 处 `IN ?`）；③ SQLite 实跑逐字段断言（3 个关联列全非空且正确，含软删工位恢复名称、硬删表不加 `deleted_at`）；④ 反例断言（关联表不存在 → 必须返回 error，禁止静默降级）；⑤ `SkipSoftDelete:true` 两处语义测试；⑥ 单条/批量逐字段一致（在 77_01 内）。 |
| `.golangci.yml` | 启用 `forbidigo`（`linters.enable` 追加）+ `linters.settings.forbidigo.forbid` 覆盖 **8 类方言 token**（`::text` / `::uuid` / `::\w+` / `DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `ILIKE` / `ON CONFLICT`）；`linters.exclusions.rules` 追加**存量白名单**（目录级 path 正则 + 逐条「存量待收敛」注释）；3 个改造过的文件**不在**白名单；**不写** `\.Raw\(` 规则（不一刀切禁全部 raw SQL）。 |
| `148-SUMMARY.md`（本文件） | 改动 / 4 REQ / 验证 / 方案②与性能 / **T3 测试清单固化** / Known Gaps / 未提交。 |

> **文件面纪律**：未触碰 `excel_service.go`（≠ `excel_export_service.go`，归 150）、`workstation_service.go` / `scope_write.go` / `useBackfillDept.tsx` / `types/operations.ts`（150）、5 个 wall/door/text/photo/alias service + `scope_entities.go` + `router.go` + `useFloorPlanEditor.ts` + `133-UAT.md`（149）、`scope_filter.go` + `base/GORMRepository`（151）、`ci.yml` / `scripts/**` / `tests/e2e/**`（145）、`internal/core/security/**`（146）、`internal/services/asset/**`（147）、前端 `api.ts` barrel；未改 `go.mod` / 未 `go get`（零新依赖）；未动 79 条跟踪表与 LOCKED 裁定原文；未写生产数据、未起长驻进程。

---

## 2. 4 REQ 满足情况

| REQ | 判定 | 证据 |
|---|---|---|
| **DIALECT-01**（③-A） | **satisfied** | `RightCast` 字段与 3 处配置删除（源码静态断言 `NotContains "RightCast"`）；`resolveAssociations` 统一单侧 `CAST(id AS TEXT)`（DryRun SQL 捕获断言 `Contains "CAST("` / `"AS TEXT"` / 3×`IN ?`、`NotContains "::"`）；`logger.Warnf+continue` 删除、关联失败返回 error（`TestDialect148_ResolveAssociations_ErrorPropagates` + 反向变异 2 必红）；3 个关联列 SQLite 实跑全部非空且逐字段正确（`TestDialect148_ResolveAssociations_SQLiteRealResult`）；`SkipSoftDelete:true` 两处语义保留（配置断言 + 实跑软删工位名称恢复 + 硬删表不报 no such column）。对应 ROADMAP `:70` 标准 1。 |
| **DIALECT-02**（③-B） | **satisfied** | 两处 raw CTE 整体删除（源码断言 `NotContains "WITH workstation_ports"`）；`DISTINCT ON` / `REGEXP_REPLACE` / `NULLS LAST` / `::text` 在 `workstation_device_service.go` **全文件静态归零**（`TestDialect148_SourceFilesNoDialectTokens`）；单工位与批量共用**唯一** `recallPhysicalRows`（2 处调用、零第二套逻辑）；等价性论证逐条对照 `sqlite_reconciliation_views.go:43-54` 模板写入代码注释（MAC 归一化逐字符核对 / 接口名最长前缀优先 / Top-1 分组 DESC 空值末位 / HistoryLastSeen / 置信度 / 资产优先 COALESCE / `CAST AS TEXT` 类型对齐）；真实结果断言（`TestWSD77_*FrontSegment` 单条批量逐字段一致 + `TestGetPhysicalDevices_UserIDEmpty`）。对应 ROADMAP `:71` 标准 2。 |
| **DIALECT-03**（③-B 配套） | **satisfied** | 4+ 个「报错即预期」测试全部改写为真实结果断言（详见 §5 T3 清单固化）；`go test -count=1 ./internal/services/operations/` 全绿（`ok ... 7.672s`）；`SkipSoftDelete` 语义测试保留并扩充。对应 ROADMAP `:72` 标准 3。 |
| **DIALECT-04**（防复发） | **satisfied** | ① `.golangci.yml` `forbidigo` 8 类规则 + 存量白名单逐目录注释「待收敛」、3 个改造文件不在白名单、未禁 `.Raw(`（`golangci-lint config verify` 通过；T4 反向探针「新增方言标识符形态 → lint 必红 / 删除 → 绿」实测通过）；② 语义原语以 Go 层 helper 落地（`normalizePhysicalMAC` / `physicalTimeGreater` / `CAST AS TEXT` 表达式），配置里无 SQL 片段原语；③ PG 双跑**只登记前置依赖**、不伪造（见 §6 Known Gaps）。对应 ROADMAP `:73-74` 标准 4/5。 |

**5 条成功标准逐条**：标准 1 ✅ / 标准 2 ✅ / 标准 3 ✅ / 标准 4 ✅（含 forbidigo 能力边界如实登记，见 §4）/ 标准 5 ✅（不建 `services: postgres`、不改 `ci.yml`、不加 `isPostgres` 分支、不下沉视图）。

---

## 3. 验证结果（T5 全部命令）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `go test -count=1 ./internal/services/operations/` | **ok** `7.672s`（`TEST_EXIT=0`）；期间 3 次因 T3 改写中间态局部红，均已修正后全绿 |
| 2 | `go build ./...` | **BUILD_EXIT=0**（`go build` 不落盘二进制，产物不落仓库根） |
| 3 | `golangci-lint config verify` | **VERIFY_EXIT=0**（v2.12.2；`exclude_godoc_examples` 为 v1 字段，实测被 jsonschema 拒绝，已按实测移除并注释） |
| 4 | `golangci-lint run --timeout=5m ./internal/services/operations/...` | **`0 issues` / LINT_EXIT=0**（含新增 forbidigo；确认存量白名单不误报、3 个改造文件不在白名单） |
| 5 | T4 反向门禁自检（临时探针 `REGEXP_REPLACE` 标识符 → lint） | **红**：`use of REGEXP_REPLACE forbidden ... (forbidigo)` / `PROBE_LINT_EXIT=1`；删除探针后 **`0 issues` / `CLEAN_LINT_EXIT=0`**（绿） |
| 6 | 反向变异 1：把 `CAST(id AS TEXT)` 临时改回 `::text` | `TestDialect148_SourceFilesNoDialectTokens` **必红**：`--- FAIL ... should not contain "::text"` / `TEST_MUT1_EXIT=1`；`golangci-lint` forbidigo **不命中**（`0 issues` / `LINT_MUT1_EXIT=0`，见 §4 能力边界）；还原后 **绿**（`TEST_REST1_EXIT=0`）。变异为 Python 精确替换 + `trap ... EXIT` 原子还原，末尾打印 `DONE: file restored` |
| 7 | 反向变异 2：把 `return err` 临时改回 `logger.Warnf+continue` | `TestDialect148_ResolveAssociations_ErrorPropagates` **必红**：`--- FAIL ... An error is expected but got nil` / `TEST_MUT2_EXIT=1`；还原后 **绿**（`TEST_REST2_EXIT=0`） |
| 8 | `go test -count=1 ./internal/services/operations/...`（子包） | 本包无子包（`go list` 结果仅 `./internal/services/operations`），命令 1 已覆盖 |
| 9 | `node scripts/check-route-registry.mjs` | **未跑**：本 phase 未动路由 / `router.go`，按 PLAN T5.5 默认不必跑 |

**结论**：只跑相关包、未跑全仓 `go test ./...`（三 executor 并行同工作区，全仓会读到其它 phase 未完成文件）；
**全仓矩阵由 Lead 集成期统一执行**。`TEST_PG_DSN` 本期 unset，未执行任何真实 PG 双跑。

---

## 4. 方案 ② 与性能评估 + forbidigo 能力边界（如实登记）

### 方案 ②（采用）：GORM 链式 + Go 层合并
- **理由**：从根源消除 raw SQL 与方言分叉；复用 `diagnosePhysicalChain`（GORM chainable + Go 侧集合求交）范式；`physicalDevicesFetcher` 注入语义保留、生产恒 nil；单工位/批量共用**唯一** `recallPhysicalRows`，「不存在第二套 recall 逻辑、口径不分叉」。
- **未采用**：方案 ① 双方言窗口函数（保留单条 SQL 性能，但仍是「双方言 SQL」而非机制化根治）；`isPostgres` 方言分支（补丁，两套口径必分叉）；CTE 下沉双方言视图（CTE 带 `?` 参数，视图无法参数化）——均为 PLAN / ROADMAP 禁区。
- **等价性论证**：逐条对照 `sqlite_reconciliation_views.go:43-54` 模板，写在 `workstation_device_service.go` 共享召回块注释里（MAC 去 `[.:-]` 逐字符核对；接口名最长前缀优先、组内互不包含；Top-1 分组 + DESC 空值末位；`HistoryLastSeen` / 置信度 CASE / 资产优先 COALESCE；`CAST(id AS TEXT)` 类型对齐）。

### 性能评估（批量 `GetPhysicalDevicesByWorkstations`）
- **拉取量**：按 `workstationIDs IN ?` 定向预取活动信息点，再按 `portIDs` / `deviceIDs` / `normMacs` / `deviceSNs` 四个 `IN ?` 集合分别预取端口、网络设备、实测 MAC、历史 MAC、资产、AD 设备；**非全表扫描**，每类表一次查询（共约 8 次 GORM 查询），无 N+1（按集合批量，不按工位逐条查）。
- **内存占用**：中间 map 规模正比于「本批工位涉及的端口 / MAC / 资产 / AD 行数」，非常驻；`physicalStableSortRows` 为稳定插入排序，组内行数通常个位数。
- **N+1 风险与缓解**：已用「批量预取 + 内存分组（`dedupSeen` / `candidates` / `out[wsID]`）」缓解；`IN ?` 参数集按 set 去重 + 稳定排序，可复现。
- **待确认**：生产 MAC / AD / 资产表行数（根因报告 `:194` 未决问题），决定超大批量下的内存与 `IN (?)` 参数上限；见 Known Gaps。

### forbidigo 能力边界（实测，不伪造）
- 实测 `golangci-lint 2.12.2`：`forbidigo` 基于 `go/ast` 匹配**标识符 / 选择器调用名**，**不匹配字符串字面量或注释里的 SQL 文本**（受控实验：字符串 `"SELECT id::text, DISTINCT ON ..."`、注释方言 token、raw 字符串一律 `0 issues`；而本地函数 `REGEXP_REPLACE(...)`、方法 `h.NULLS_LAST()` 均能命中）。
- 因此 PLAN 所述「把 `CAST` 改回 `::` → forbidigo 必红」**在纯 forbidigo 下技术上不成立**（`::text` 含特殊字符，永远不可能构成 Go 标识符）。
- **处置（不伪造、不静默跳过）**：
  1. forbidigo 依 PLAN 启用 8 类规则，作为「准入守卫」——能命中以标识符/函数调用形态引入的方言写法（T4 探针实测必红）；
  2. **真正承担「反向变异必红」的门禁**是 `TestDialect148_SourceFilesNoDialectTokens`（Go 源码静态断言，扫字符串+注释，见 T5 变异 1 红绿）；
  3. SUMMARY 如实登记此边界，供 Lead 决定后续是否引入额外 AST/SQL 字符串扫描器（本期不新增依赖）。

---

## 5. T3 测试清单固化（Lead 补充裁决 / checker 小问题 4）

### 5.1 全量定位结果（执行期 grep + 逐文件核对）

定位命令：`grep -rn "require.Error|assert.Error|勿为覆盖率|unrecognized token|PG.*CTE|报错" internal/services/operations/*_test.go`，
并按「把 PG CTE 报错当预期」语义逐条判定，最终清单如下（**逐文件列名**）：

| # | 文件 | 定位 | 性质 | 处置 |
|---|---|---|---|---|
| 1 | `workstation_device_77_01_test.go` | `:9-11`（「勿为覆盖率改 SQL」注释） | 「报错即预期」注释 | **已改写**：删注释，改双方言共享召回说明 |
| 2 | `workstation_device_77_01_test.go` | `:568-600` `TestWSD77_GetPhysicalDevices_FrontSegment` | `require.Error` 断言「PG CTE 报错属预期」 | **已改写**：`require.NoError` + 1 台设备逐字段真实结果 |
| 3 | `workstation_device_77_01_test.go` | `:603-620` `TestWSD77_GetPhysicalDevicesByWorkstations_FrontSegment` | 同上（批量） | **已改写**：`require.NoError` + 1 台 + 单条/批量逐字段一致 |
| 4 | `excel_export_chain_77_02_test.go` | `:17-19` / `:408-411` 头注释 | 「physErr 降级」说明 | **已改写**：改「补建链路表 + 真实数据行」 |
| 5 | `excel_export_chain_77_02_test.go` | `:458-461` 物理链路 sheet 断言 | `require.Len(physRows,1)` 且注释「必报错 → 降级 0 行」 | **已改写**：`require.Len(...,2)` + 逐字段抽查 |
| 6 | `excel_export_chain_77_02_test.go` | `:466-480` `TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` | 「PGOnlySQL」夹具限制文档化 | **已改写**：更名 `..._FixtureBoundary`，明确「夹具缺表」非方言报错 + 断言无 `unrecognized token` / `syntax error` |
| 7 | `workstation_physical_diagnosis_test.go` | `:33` / `:71-87` / `:85-87` 注释 + `newDiagnosisTestService` 注入 | 「CTE 方言替换」层 | **已改写**：不再注入替代召回，`sqlitePhysicalRecall` 降级独立 oracle |
| 8 | `workstation_physical_diagnosis_test.go` | `:401-403` 注释 + `:404-425` `TestGetPhysicalDevices_LegacySignatureUnchanged` ③ | 「PG CTE 无法执行 → 只做签名/委托守护」 | **已改写**：扩为签名 + 参数校验 + 真实结果 + 诊断 Total 同源四重守护 |
| 9 | `workstation_device_physical_test.go` | `:244-257` 注释 | 「只守护 user_id 早退、完整链路需 PG」 | **已改写**：删「需 PG」结论 |
| 10 | `workstation_device_physical_test.go` | `:258-284` `TestGetPhysicalDevices_UserIDEmpty` | `require.Error`「缺表/PG 语法报错即证明越过早退」 | **已改写**：建齐 8 表 + `require.NoError` + 1 台逐字段 |

> 合计 **10 处 / 4 个文件**，其中 `require.Error`「报错即预期」断言 **4 处**（#2 #3 #10 及 #8 的绕过式断言），
> 满足 PLAN「4+ 个测试」与 REQUIREMENTS `:42` 点名的 3 个文件。

### 5.2 执行期 grep 命中但**判定不属于本 phase**、未改动（逐文件列名）

| 文件 | 命中 | 判定 |
|---|---|---|
| `excel_dept_cte_postgres_test.go` | `:120` `require.Error(t, err, "未 CAST 的递归 CTE 在 PostgreSQL 上应当报类型错误")` | **有效契约反例测试**（PG 集成、`TEST_PG_DSN` 门控、故意构造未 CAST 反例），测的是 dept 递归 CTE，非 workstation 物理链路 CTE；属 PLAN 允许「PG contract 仅 skip 可观测」范畴，**不改** |
| `excel_export_chain_77_02_test.go` | `:227` `require.Error(t, err, "code 默认映射为 code 列,recon 表无该列应报错")` | 列映射不存在报错，与方言无关，**保留** |
| `workstation_device_77_01_test.go` | `:297/:302/:348/:353/:356/:405/:452/:509/:564` 等 `require.Error` | 参数校验 / AD/资产查询链的正常错误断言，非「PG CTE 报错即预期」，**保留** |
| `workstation_physical_diagnosis_test.go` | 诊断 4 个测试 `TestGetPhysicalDevicesWithDiagnosis_*` / `TestClassifyPhysicalEmptyReason_*` | 测空因分类，本就跑真实诊断；现在更进一步走真实共享召回，**保留且增强** |
| `workstation_device_physical_test.go` | `TestGetPhysicalDevices_DeviceIdMatch` / `DeviceIdDriftStillResolves` / `DataGoverned` / `InvalidDeviceId` | 用 `physicalJoin` 独立 oracle 做真实 MAC 命中数断言，本就是真结果，**保留** |
| `excel_dept_parent_path_test.go` / `excel_query_builder_p18_test.go` / `scope_filter_test.go` / `excel_export_devices_test.go` / `excel_import_rest_77_03_test.go` / `scope_read_403_test.go` / `excel_import_export_test.go` / `excel_uniqueness_batch_test.go` / `batch_upserter.go` 等 | 注释里提及 `::uuid` / `DISTINCT ON` / `ON CONFLICT` / `PG-only` | 文档性描述或 P1-8 既有回归守护，非「把 PG CTE 报错当预期」的断言，**不改** |

### 5.3 与 150 的 4 个测试交叉核对（Lead 裁决要求）

150 的 4 个测试：`scope_write_403_test.go` / `pg_contract_test.go` / `v152_diff_coverage_test.go` / `workstation_backfill_dept_test.go`
（实测 4 个文件均在 `internal/services/operations/` 存在）。

- **T3 改写清单（§5.1 的 4 个文件）与上述 4 个文件两两交集 = ∅**，无交叉、未改动任何 150 文件。
- **`pg_contract_test.go`**：含 `::uuid` / `::text` 注释（`:215/:218`）并 `:222` 引用旧测试名
  `TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL`。该文件**归 150，本 phase 未改**。
  ⚠️ **移交 Lead**：我把 `PGOnlySQL` 更名为 `..._FixtureBoundary` 后，`pg_contract_test.go:222` 的注释引用
  变成「悬空测试名」。建议 Lead 在集成期把该注释里的旧名改为新名（或保留为历史说明），
  **148 不碰 150 文件面**。
- **`v152_diff_coverage_test.go`**：`:141/:354` 有 `require.Error`（"sys_dept 缺失时 Count 应报错" /
  "查询报错应上抛"），属 150 的正常错误上抛契约，**归 150、未改**（不计入 148 的 T3 清单）。
- `scope_write_403_test.go` / `workstation_backfill_dept_test.go`：无「报错即预期」命中，未改。

---

## 6. Known Gaps（登记，非阻断）

1. **CI PG 双跑前置依赖**（DECISION-CI-PG / DIALECT-04③）：`ci.yml` 无 `services: postgres`、
   `TEST_PG_DSN` 本期 **unset**，PG contract 仅做「skip 可观测」，**零真实 PG 验证**。
   需**新建 PG service + 迁移引导**后方可把 3 个 contract 测试从 skip 可观测升级为双跑硬门禁。
   本期**不建 `services: postgres`、不改 `ci.yml`（归 145）、不伪造双跑**。
   本 phase 的「PG 版期望值」以 fixture 常量（从原 CTE 语义推导）固化进 SQLite 断言，口径可核对。
2. **`REGEXP_REPLACE '\s+'` 与嵌套 REPLACE 的已接受偏差**：按 `sqlite_reconciliation_views.go:27-28`
   口径登记（接口名实际不含制表符/换行，dev 可接受）。本 phase 的 Go 层 `normalizePhysicalIface`
   用 rune 判定去除**全部空白**（空格/制表/换行等），比 SQL 范本「嵌套 REPLACE 只去空格」更精确，
   故该偏差在本实现中已消除；仅作为历史范本口径差异登记。
3. **`IN (?)` 参数上限**：Go 字符串 ids 批量 `IN ?` 在超大 ids 集下有参数上限 / 分片风险
   （SQLite 默认 999 / PG 65535 量级，依驱动与版本）。本期按 set 去重 + 稳定排序控制规模，
   **不引入分片新机制**，登记为后续优化项；与「生产 MAC/AD/资产表行数待确认」（根因报告 `:194`）一并跟踪。
4. **forbidigo 能力边界**：见 §4。纯 forbidigo 不命中字符串字面量里的方言 token；字符串层门禁由
   `TestDialect148_SourceFilesNoDialectTokens` 承担。后续可评估引入 SQL 字符串 AST 扫描器（本期零新依赖）。
5. **存量方言收敛**：全仓实测 **136 个 Go 文件**命中 8 类方言 token，已按目录列入
   `.golangci.yml` 白名单并逐条注释「存量待收敛」；逐步从白名单移除属后续 phase，不在 148 文件面。

---

## 7. 未提交声明

本 plan `commit:false`，所有改动**留工作区**，由 Lead 在 W2 收口统一提交；
**未执行任何** `git add` / `git commit` / `git stash` / `git checkout` / `git reset` / 分支写操作。
未改 `ci.yml`（归 145）、未改 149/150/151/W1 任何文件、未新增外部依赖、未写生产数据、
未启动长驻进程、未动 79 条人工验证跟踪表与 LOCKED 裁定原文。

---

*Phase 148 · exec-148 · 2026-11-09 · 简体中文*
