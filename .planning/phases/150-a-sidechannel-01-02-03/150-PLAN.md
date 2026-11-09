---
phase: 150
plan: 01
wave: 2
depends_on: []
files_modified:
  - internal/services/operations/workstation_service.go
  - internal/services/operations/scope_write.go
  - internal/services/operations/excel_service.go
  - internal/services/operations/workstation_backfill_dept_test.go
  - internal/services/operations/scope_write_403_test.go
  - internal/services/operations/v152_diff_coverage_test.go
  - internal/services/operations/pg_contract_test.go
  - internal/services/operations/sidechannel_diff_attack_test.go
  - internal/api/v1/operations/workstation_handler.go
  - internal/api/v1/operations/excel_handler.go
  - internal/api/v1/operations/workstation_handler_full_test.go
  - xingran-react-frontend/src/pages/operations/workstations/hooks/useBackfillDept.tsx
  - xingran-react-frontend/src/types/operations.ts
  - xingran-react-frontend/src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx
  - .planning/phases/150-a-sidechannel-01-02-03/150-UAT-BACKFILL.md
  - .planning/phases/150-a-sidechannel-01-02-03/150-SUMMARY.md
autonomous: true
commit: false
---

# Phase 150 · ① 侧信道根治 A — 响应体去规模化（PLAN 01）

> 本 plan 为 **可直接执行** 的交付物。3 个 executor（E1/E2/E3）文件面互斥并行，Lead 统一提交；**任何 executor 不得执行 git 写操作**。输出与注释一律简体中文。

---

## 1. 必读顺序（开工前，按序读完再动手）

| # | 文件 | 读什么 | 为什么 |
|---|------|--------|--------|
| 1 | `.planning/STATE.md` | 头部 v1.55 active 段（:20-26）、`## Current Position`（:99-104） | 确认 v1.55 active、Phase 从 145 续编、本 phase 尚未开工 |
| 2 | `.planning/ROADMAP.md` | **Phase 150 详情 :88-98**（尤其 :91 根因定性、5 条成功标准 :94-98）、**并行冲突面/函数级归属表 :113-148**、Constraints :150-161、5 项裁定 :163-171、禁区 :173-181、账务 :183-186、Known Gaps :188-200 | W2 `148 ∥ 149 ∥ 150` 的**唯一并行许可凭证**；成功标准 = 本 plan `must_haves` 的逐条来源 |
| 3 | `.planning/REQUIREMENTS.md` | **SIDECHANNEL-01~03 全文 :68-76**、根治定义 :70、A+B 组合 :71、**Constraints :98-106**、5 项裁定 :7-15、Out of Scope :85-96、Traceability :108-136 | REQ 权威口径；DECISION-①「允许重定语义」是推翻 132 产品语义的授权 |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` | **§① 根因 :32-54**（根因纠正、根治定义、方案对比 A+B/C、推荐 A+B、判定、FUTURE-01）、§4 未决问题 :188-199 | 本 phase 的 research 输入；确认「① 不是 SQL 漏 WHERE，是信息论冲突」 |
| 5 | `.planning/milestones/v1.52-phases/132-write-scope/132-SUMMARY.md` | **:65, :74, :103** | 范围快照四类文案原文、「不假装已解决」裁定原文；本期作废对象（**只读不改**，见裁决 D-5） |
| 6 | `CLAUDE.md` / `AGENTS.md` | `response.Success()/Error()` 包装、operlog 写操作约定（`response.Success` **之前** `Record`）、25 个 OperType 常量、`0=正常 1=停用`、前端 wrapped API、`useEffect` 依赖稳定、构建产物只落 `out/` | 全局硬约束 |

**代码实证必读（本 plan 已核对，executor 复核行号）**：
- `internal/services/operations/workstation_service.go:110-134`（`BackfillDeptResult` 字段）、`:606-682`（`BackfillDeptFromUser` 全库 `countQ` 双口径）、`:714-721`（`Samples` 非 All 恒空）。
- `internal/services/operations/excel_service.go:62-84`（`ImportError`/`ImportResult`）、`:333-388`（校验错误先于 scope 过滤）、`:448-461`（`filterScopeWriteImportRows` 调用 + `result.ScopeOutOfScope = outOfScope`）。
- `internal/services/operations/scope_write.go:412-469`（`scopeWriteWorkstationVisibleSQL` / `scopeWriteBackfillSnapshot`）、`:687-722`（`filterScopeWriteImportRows` 越界分支）、`:730-738`（`ScopeWriteOperParam`）。
- `internal/api/v1/operations/workstation_handler.go:435-452`、`internal/api/v1/operations/excel_handler.go:135-154`（**连锁面**，见已确认事实 F-5）。
- `internal/services/operations/pg_contract_test.go:254-271`（`TestScopeWriteImport_StaticContract` 静态断言）、`:273-301`（`TEST_PG_DSN` 门控）。
- `xingran-react-frontend/src/pages/operations/workstations/hooks/useBackfillDept.tsx:110-139`（`rangeText` / `okButtonProps`）、`src/types/operations.ts:111-129`。

---

## 2. 目标

**信息论目标（REQUIREMENTS.md:70 逐字）**：任何非 All 档用户，通过单请求或多次请求组合，都无法从响应体（含状态码、任何数值 / 序号 / 数组长度）推导出「可见集之外的资源规模」或「越界资源是否存在 / 有多少」。

**本 phase（A）= 响应体去规模化**：把 132 有意设计的「带 scope 的 `candidateQ` + 不带 scope 的全库 `countQ` + 相减回显」双口径，以及 Excel 导入 / 批量写的越界位置与计数，从 **HTTP 响应体** 里彻底移除；聚合计数**保留在服务端 operlog 审计链**（不降级）。

**根治 = 推翻「范围快照回显」产品语义**（用户 2026-11-09 DECISION-① 已裁定允许）。**A(150) + B(Phase 151 统一 ScopedDB) 均需完成**；本 phase **不伪造「统一架构已落地」**，B 未落地前不得宣称 ① 已根治（写进 `150-SUMMARY.md`）。

**3 个 REQ 一一对应**：SIDECHANNEL-01 → T1、SIDECHANNEL-02 → T2、SIDECHANNEL-03 → T3 + T4；T5 自验证、T6 SUMMARY。

---

## 3. 已确认事实（不要重新调研）

- **F-1｜① 不是 SQL 漏 WHERE**。8 个 `/statistics` 已全部在 SQL 层用 `wrapScopeExists` / `apply*Scope` 注入可见集（如 `workstation_service.go:159/169`、`asset_service.go:69/74`），COUNT 与明细同源。残余是**产品有意**的双口径差分：`workstation_service.go:645-647` 的 `candidateQ` 带 `visibleCond`，但 `:666-669` **另跑一条刻意不带 scope 的全库 `countQ`**（注释明写「COUNT 保持全库口径 = 请求行数…若给 COUNT 也挂可见集条件会让越界恒 0，丢失审计语义」），`:676-677` 得 `ScopeTotal` / `ScopeOutOfScope = ScopeTotal - ScopeVisible`，`:681-682` 得全库派生 `SkippedNoUser` / `SkippedBadUserDept` → 响应体 `:127-132` 可被差分出越界规模。`Samples` 已收敛（非 All 档恒空，`:719-721`）但聚合数值仍在。
- **F-2｜Excel 导入显式泄漏面**。`ImportError.Row`（`excel_service.go:62-67`）携带实际行号、`ImportResult.ScopeOutOfScope`（`:81-83`）携带越界行数；`scope_write.go:701-722` 的 `outOfScope++` / `ImportError{Row: rowNum}` / `result.Failed++` / `result.Errors = append(...)`。
- **F-3｜批量写审计链**。`ScopeWriteOperParam`（`scope_write.go:730-738`）的 `requested / visible / outOfScope` **不直接回前端但入 operlog 审计链**（132-SUMMARY.md:64-65,103）；`ScopeWriteBatchStats`（`scope_write.go:476`）经 `visibleIDsForEntityWrite`（`:282-320`，**仅被 149 复用、不在 149 写集合**）算聚合。
- **F-4｜间接泄漏面（本 plan 调研新发现，任务书未逐字点名，必须一并根治）**。
  - `excel_handler.go:148-153` 用 `gin.H` 手工回显 `inserted/updated/failed/errors`。越界行虽不写 `ScopeOutOfScope` 字段，但 `result.Failed++` 与 `result.Errors` 数组长度**随越界行数 N−M 变化** → 显式删字段仍可枚举出 N。**必须让越界行静默跳过：不 `Failed++`、不 append `Errors`**。
  - `workstation_handler.go:444-449` 的 backfill operlog 直接引用 `result.ScopeTotal/ScopeVisible/ScopeOutOfScope`；删字段后**编译失败**，必须同步改（裁决 D-4）。
  - 校验顺序：`excel_service.go:370-375` 的普通校验错误**先于** `:454` 的 scope 过滤，故 `Errors` 里可能已有用户自有文件的校验错误（含 `Row`）。这些是用户自己上传内容，不构成越界规模泄漏；**但差分攻击测试必须断言「固定上传文件、改变 N−M 时 `failed` 与 `errors` 长度不变」**。
  - `pg_contract_test.go:269-270` 有 132 遗留静态断言（`result.ScopeOutOfScope = outOfScope` / `__excel_row__`），`:269` 可保留、`:270` 描述须改写；另需新增「响应体结构不含全库规模字段」静态断言。
  - 前端连锁：`src/types/operations.ts:117-127`、`useBackfillDept.tsx:114-160`、`useBackfillDept.test.tsx:34-45,214-280`；后端测试连锁：`workstation_backfill_dept_test.go:115-116,249`、`scope_write_403_test.go:312-336,353-361`、`v152_diff_coverage_test.go:500-501`、`workstation_handler_full_test.go:651-656`。
- **F-5｜UAT 载体实际位置**。`132` 无独立 UAT 文件（仅 `132-SUMMARY.md`）；`133-UAT.md`（`.planning/milestones/v1.52-phases/133-scope-final/`）是 A/B/C/D/E/F/G 段，**无 backfill 段**，其 `:82-91` D1/D2 段归 **149**。ROADMAP :186 明确「不要动 `.planning/milestones/` 归档件 / 不要创建 `.planning/phases/` 目录」（立项期对 roadmapper 的约束）。本 plan 按用户直接指令落盘于 SDK 已确定的 `.planning/phases/150-a-sidechannel-01-02-03/`，UAT 作废重跑**不改归档件**，改在本 phase 目录新建 `150-UAT-BACKFILL.md`（裁决 D-5）。
- **F-6｜`TEST_PG_DSN` 本期仍为前置依赖**。`pg_contract_test.go:34-74` 的 `openPGForContractTest` 在 DSN 未设置时 `t.Skip`；用户 DECISION-CI-PG 裁定本期**不建 `services: postgres`、不伪造双跑**。T4 用「静态断言恒跑 + SQLite 真跑」，PG 侧保持 skip 可观测。
- **F-7｜All 档零回归硬约束**。admin 的业务数值（`candidates/toFill/toCorrect/skippedAlreadyValid/samples`）与 130 前一致；`132-SUMMARY.md:74` 的「全库（共 N 条）」措辞在 All 档**可保留**（任务书许可，非强制）。

---

## 4. Lead 裁决表（解决任务书内部张力，executor 一律以此为准）

| ID | 张力 / 歧义 | **裁决（权威）** | 依据 |
|----|------------|-----------------|------|
| **D-1** | 「All 档响应形状/数值零回归」与「删 `BackfillDeptResult` 字段」冲突 | **统一全档位删除，禁止 `if isAll { 回显 } else { 不回显 }` 档位分支**（该分支即「方言分支」类补丁，且与 151 响应契约层前置冲突）。「All 档零回归」= **业务数值**（`Candidates/ToFill/ToCorrect/SkippedAlreadyValid/Samples` 内容）与 130 前一致 + All 档前端可保留「全部工位 / 全库」措辞；132 新增的 6 个范围快照/全库派生字段按 DECISION-① **全档位移除**。 | ROADMAP :91/:158、REQUIREMENTS :70/:104、DECISION-①；根治=推翻语义而非按档位打补丁 |
| **D-2** | 删哪些字段？成功标准 3 提到作废 `scopeWriteBackfillSnapshot` | **只删 6 个**：`ScopeTotal` / `ScopeVisible` / `ScopeOutOfScope` / `ScopeDeptTotal` / `SkippedNoUser` / `SkippedBadUserDept`。**保留** `DryRun/FixMismatch/Candidates/ToFill/ToCorrect/SkippedAlreadyValid/Samples`（均为 132 前既有、可见集内）与 `ScopeTier`/`ScopeDeptNames`/`ScopeFailClosed`（档位 / 可见部门名 / fail-closed 布尔，**均不泄漏越界规模**）。`scopeWriteBackfillSnapshot`（`scope_write.go:444`）**保留但精简**：不再输出 `deptTotal` 回显、注释改为「不含全库规模」；「作废」指其驱动的**四类「共 N 条」确认框文案**，非删函数。 | ROADMAP 成功标准 1（:95）逐字列举 6 个；REQUIREMENTS SIDECHANNEL-01（:73）逐字；避免扩大连锁面 |
| **D-3** | 全库 `countQ` 是「删」还是「改为带 `visibleCond`」 | **整段删除** `workstation_service.go:655-682` 的 `stats` / `countQ` / 全部 5 个派生赋值（`ScopeTotal/ScopeOutOfScope/SkippedNoUser/SkippedBadUserDept/ScopeVisible`）。理由：删掉 6 个字段后这段 COUNT **唯一消费者已消失**，保留即残留「不带 scope 的 COUNT」死代码，违背「所有 COUNT(*) 与明细同源」。`result.Candidates = len(candidates)`（`:652`，已带 `visibleCond`）保留。 | 任务 T1「去掉全库 countQ…消除双口径」；REQUIREMENTS :73 |
| **D-4** | 删字段后 `workstation_handler.go:444-449` operlog 编译失败，如何「审计链不降级」 | **backfill handler 的 OperParam 去掉 `requested/visible/outOfScope` 三个全库/越界聚合**（该语义随 DECISION-① 作废），改为记录 `fixMismatch / candidates / toFill / toCorrect / scopeTier`（`scopeTier` 取 `opsServices.ScopeWriteTier(c.Request.Context())` 现成函数，或保留 `result.ScopeTier`）。**`ScopeWriteOperParam`（`scope_write.go:730-738`）的 `requested/visible/outOfScope` 完全保留、一个字不改**（任务书点名「保留在服务端审计链但不进 HTTP 响应」），仅补注释说明「仅供 operlog，禁止写入任何 HTTP 响应」。Excel 导入 operlog（`excel_handler.go:143-147`）的 `outOfScope` 保留 → 越界事实仍在审计链。「审计链不降级」= 写入行为 + 档位 + 可见集内计数 + 越界事实仍在 operlog，**不等于必须把全库 N 记进 operlog**。 | 任务书「operlog 审计链不降级」段逐字只点名 `ScopeWriteOperParam`；ROADMAP :96/:161 |
| **D-5** | 132/133 backfill UAT「作废重跑」改哪个文件 | **不改 `.planning/milestones/` 任何归档件**（ROADMAP :186 硬约束），**不动 `133-UAT.md` 的 D1/D2 段**（归 149）。在本 phase 目录**新建 `150-UAT-BACKFILL.md`**：登记 ① `132-SUMMARY.md:65/:74/:103` 四类「本次范围：…（共 N 条）」文案与「范围快照回显」语义作废；② `133-UAT.md` A6「越界计数入 operlog」改写为「越界计数仅入 operlog、HTTP 零回显」；③ 重跑窗口与验收人**由 QA 裁定、AI 不擅自勾选**；④ 明确 D1/D2 归 149、本期不碰。 | ROADMAP :159/:186/:200；任务书「只改 backfill 段，不动 149 的 D1/D2 段」 |
| **D-6** | 越界行「静默跳过」如何与「审计链保留」共存 | `filterScopeWriteImportRows`（`scope_write.go:687-722`）**签名不变**（保留 `result *ImportResult` 参数以避免跨文件连锁），但越界分支**只 `outOfScope++` + `delete(rec,"__excel_row__") + continue`，删除 `result.Failed++` 与 `result.Errors = append(...)`**。`excel_service.go:460` 的 `result.ScopeOutOfScope = outOfScope` **保留**，但字段 tag 改 `json:"-"`（内部字段，供 `excel_handler.go:147` operlog 读取，永不进 HTTP 响应）。`ImportError.Row` **不删结构体字段**（普通校验错误仍用，用户自有数据），但越界行不再产生 `ImportError`。`excel_handler.go:148-153` 的 `gin.H` 形状不变。 | 任务 T2「越界行静默跳过或仅服务端 operlog」；F-4 间接泄漏面 |
| **D-7** | `pg_contract_test.go` 归谁？148 也动测试 | ROADMAP 函数级归属表 :124 的 148 测试集合是 `workstation_device_77_01_test.go` / `excel_export_chain_77_02_test.go` / `workstation_physical_diagnosis_test.go`，**不含 `pg_contract_test.go`**；:135 把「差分攻击测试」归 150。核对 149/151 写集合亦无此文件 → **`pg_contract_test.go` 归 150 独占**，与 148 交集 = ∅。同理 `workstation_handler.go` / `excel_handler.go` 经核对不在 148/149/151 任一写集合，**纳入 150 独占**（因 D-4/D-6 连锁必须改），并在 `150-SUMMARY.md` 登记。 | ROADMAP :119-137 逐格核对；任务 T4 点名 `pg_contract_test.go` |
| **D-8** | FailClosed 前端 `okButtonProps.disabled` 依赖 `scopeFailClosed` | **保留 `ScopeFailClosed` 字段与 `okButtonProps`**（D-2 已裁定保留），`rangeText()` 的 FailClosed 分支改为「本次范围：无（数据权限异常，请联系管理员）」**且不含任何数值**。FailClosed 时候选查询本身经 `visibleCond` 得空集、`ToFill=0`、执行零写库，安全语义不因删 6 字段而降级。 | 132-SUMMARY.md:74 第 4 类文案；`scope_write.go:412` `scopeWriteWorkstationVisibleSQL` |

---

## 5. 任务（XML）

> **并行分组（文件面互斥，已逐格核对）**：
> - **E1** = T1 + T1 连锁后端测试：`workstation_service.go`、`scope_write.go`、`workstation_handler.go`、`workstation_backfill_dept_test.go`、`scope_write_403_test.go`、`v152_diff_coverage_test.go`、`workstation_handler_full_test.go`。
> - **E2** = T2 + T4：`excel_service.go`、`excel_handler.go`、`pg_contract_test.go`、**新增** `sidechannel_diff_attack_test.go`。
> - **E3** = T3 + T6 + UAT：`useBackfillDept.tsx`、`types/operations.ts`、`useBackfillDept.test.tsx`、**新增** `150-UAT-BACKFILL.md`、`150-SUMMARY.md`。
> - **T5** 由 3 个 executor 各自跑本组验证命令后，由 **Lead** 汇总（不并发跑同一 `go test`）；反向变异由 E2 执行。
> - E1 与 E2 的**唯一跨文件接口契约**：`filterScopeWriteImportRows` 签名 `(ctx, db, entityType, config, records, result) ([]map[string]any, int, error)` **不变**，`outOfScope` 返回值语义不变（仅用于 `result.ScopeOutOfScope` / operlog），E1 负责不再写 `result.Failed`/`result.Errors`，E2 负责把 `ScopeOutOfScope` 改 `json:"-"` 并保留 `:460` 赋值。

<tasks>

  <task id="T1" executor="E1" req="SIDECHANNEL-01">
    <name>BackfillDeptResult 去规模化 + 消除全库 COUNT 双口径</name>
    <files>
      internal/services/operations/workstation_service.go;
      internal/services/operations/scope_write.go;
      internal/api/v1/operations/workstation_handler.go;
      internal/services/operations/workstation_backfill_dept_test.go;
      internal/services/operations/scope_write_403_test.go;
      internal/services/operations/v152_diff_coverage_test.go;
      internal/api/v1/operations/workstation_handler_full_test.go
    </files>
    <read_first>本 plan §3 F-1、§4 D-1/D-2/D-3/D-4/D-8；workstation_service.go:110-134 与 :606-682；scope_write.go:440-469；workstation_handler.go:435-452</read_first>
    <action>
      1. `workstation_service.go:114-134` 的 `BackfillDeptResult`：**删除 6 个字段** `ScopeTotal`(:130) / `ScopeVisible`(:131) / `ScopeOutOfScope`(:132) / `ScopeDeptTotal`(:129) / `SkippedNoUser`(:120) / `SkippedBadUserDept`(:121)，连同其注释。**保留** `DryRun/FixMismatch/Candidates/ToFill/ToCorrect/SkippedAlreadyValid/Samples/ScopeTier/ScopeDeptNames/ScopeFailClosed`（D-2）；把 `:124-126` 的「范围快照」注释改写为「Phase 150：不再回显任何全库规模 / 越界规模；`ScopeTier`/`ScopeDeptNames`/`ScopeFailClosed` 仅为可见集内元数据与 fail-closed 提示」。
      2. `workstation_service.go:655-682`：**整段删除** `var stats []struct{...}`、`countQ := s.db.WithContext(ctx).Table("sys_workstation")...`(:666-669)、`countQ.Scan`、`:673-675` 的 `len(stats)==0` 早返回、以及 `:676-682` 的 `ScopeTotal/ScopeOutOfScope/SkippedNoUser/SkippedBadUserDept` 全部赋值（D-3）。**删除 `:662-665` 那条「COUNT 保持全库口径」注释**（它是被推翻的产品语义原文，不得留作处方）。`:652` 的 `result.Candidates = len(candidates)` 与 `:653` 的 `result.ScopeVisible = len(candidates)` 中，**只保留 `result.Candidates`，删 `result.ScopeVisible`**。
      3. `workstation_service.go:613-621`：`scopeWriteBackfillSnapshot(ctx, s.db)` 仍调用，但**不再写 `result.ScopeDeptTotal`**；`deptTotal` 返回值用 `_` 接收。`ScopeTier/ScopeDeptNames/ScopeFailClosed` 三个赋值保留（D-2/D-8）。
      4. `scope_write.go:440-469`：`scopeWriteBackfillSnapshot` 保留，注释改为「仅生成档位 / 可见部门名 / FailClosed，**不含任何全库或越界规模**」；`deptTotal` 仍可内部计算但**不再用于回显**（可简化为返回 `(tier, deptNames, failClosed)` 三值并同步 `workstation_service.go:617`，二选一，以 `go build` 通过为准）；删除后清理未使用的 import（`datascope` 是否仍被 `:412` 等处使用须复核）。
      5. `scope_write.go:730-738` `ScopeWriteOperParam`：**一个字不改其 JSON 内容**（`requested/visible/outOfScope` 全部保留，D-4），仅把注释补一句「本字符串**仅供 `operlog.WithOperParam` 服务端审计链**，严禁写入任何 HTTP 响应体 / `gin.H` / `response.Success` 的 data」。
      6. `scope_write.go:687-722` `filterScopeWriteImportRows`：**签名不变**（D-6/E1-E2 接口契约）。越界分支 `:707-718` 改为：只 `outOfScope++`、`delete(rec, "__excel_row__")`、`continue`；**删除 `:710` 的 `result.Failed++` 与 `:711-715` 的 `result.Errors = append(result.Errors, ImportError{...})`**。`:682` 与 `:708` 的「含行号，不含敏感值 Value」注释改写为「越界行静默跳过，仅 `outOfScope` 入服务端 operlog，不进 HTTP 响应」。`result *ImportResult` 参数保留但函数体内不再写入（若 Go 编译器报未使用参数不会报错，无需删）。
      7. `internal/api/v1/operations/workstation_handler.go:441-450`：按 D-4 改 OperParam 格式串为 `{"fixMismatch":%t,"candidates":%d,"toFill":%d,"toCorrect":%d,"scopeTier":%q}`，参数用 `req.FixMismatch, result.Candidates, result.ToFill, result.ToCorrect, result.ScopeTier`（**不再引用 `result.ScopeTotal/ScopeVisible/ScopeOutOfScope`**）。`operlog.Record` 位置保持在 `response.Success` **之前**（CLAUDE.md 约定），`operlog.OperTypeBatch` **常量值不重排**。
      8. **连锁测试改写**（全部属 150 独占 `*_test.go`）：
         - `workstation_backfill_dept_test.go:115-116`、`:249`：删 `res.SkippedNoUser` / `res.SkippedBadUserDept` 断言，改为断言 `res.Candidates` / `res.SkippedAlreadyValid` / `res.ToFill` 不变（保持原有数值，验证「业务数值零回归」）。
         - `scope_write_403_test.go:312-316`：`res.ScopeVisible` / `res.ScopeTotal` / `res.ScopeOutOfScope` 三条断言**删除**，改为断言响应体序列化后不含 `scopeTotal`/`scopeVisible`/`scopeOutOfScope`/`scopeDeptTotal`/`skippedNoUser`/`skippedBadUserDept` 键（用 `encoding/json`）；保留 `assert.Equal(t,"Custom/Dept/DeptChild", res.ScopeTier)` 与 `assert.Empty(t, res.Samples)`；`:330` `resAll.ScopeTier`、`:331` `len(resAll.Samples)<=20`、`:335-336` `resClosed.ScopeFailClosed` 保留（D-2/D-8），`resClosed.ScopeVisible` 改为断言 `resClosed.Candidates == 0`。
         - `scope_write_403_test.go:353-361` `TestWrite403_ImportRows_OutOfScope`：按 D-6 改断言 —— `assert.Len(t, kept, 1)` / `assert.Equal(t, 1, outOfScope)` **保留**；`assert.Equal(t, 1, result.Failed)` 改为 `assert.Equal(t, 0, result.Failed)`；`require.Len(t, result.Errors, 1)` 改为 `assert.Empty(t, result.Errors)`；删 `:360` 的 `Row == 3` 断言；`:362-365` 的 `__excel_row__` 不泄漏断言保留。
         - `v152_diff_coverage_test.go:492-501`：删 `res.SkippedNoUser` / `res.SkippedBadUserDept` 断言，改为断言 `res.Candidates` 与提前返回不写库。
         - `workstation_handler_full_test.go:651-656`：mock 返回的 `BackfillDeptResult` 去掉被删字段（当前只填 `ToFill/ToCorrect/Samples`，无需改则跳过）；`:665` `assert.Contains(w.Body.String(), "toFill":3)` 保留；**新增**断言响应体不含上述 6 个键。
      9. **禁止**：不得加 `if isAll` 分支回显（D-1）；不得把 `countQ` 改成「带 visibleCond 但仍回显总数」的中间态；不得改 `Samples` 非 All 恒空逻辑；不得改 25 个 OperType 常量。
    </action>
    <acceptance>
      `BackfillDeptResult` 序列化后**零**全库/越界规模字段；`workstation_service.go` 中**不再存在**任何不带 `visibleCond` 的 `COUNT(*)`；`candidateQ` 仍是唯一 COUNT/明细来源；`ToFill`/`ToCorrect`/`Candidates`/`SkippedAlreadyValid`/`Samples` 保留；`go build ./internal/services/operations/ ./internal/api/v1/operations/` 通过。
    </acceptance>
  </task>

  <task id="T2" executor="E2" req="SIDECHANNEL-02">
    <name>Excel 导入响应去行号 / 去越界计数 + 审计链保留</name>
    <files>
      internal/services/operations/excel_service.go;
      internal/api/v1/operations/excel_handler.go
    </files>
    <read_first>本 plan §3 F-2/F-4、§4 D-6/D-7；excel_service.go:62-84 与 :333-388 与 :448-461；excel_handler.go:135-154；scope_write.go:687-738（E1 同步改，接口契约不变）</read_first>
    <action>
      1. `excel_service.go:81-83`：把 `ScopeOutOfScope int json:"scopeOutOfScope,omitempty"` 改为 **`ScopeOutOfScope int json:"-"`**，注释改写为「Phase 150：**仅服务端内部字段**，供 `excel_handler.go` 的 operlog 审计链读取；`json:"-"` 确保永不进入 HTTP 响应。越界行静默跳过，不在响应体回显行号或数量」。**字段名与类型不变**（`excel_handler.go:147` 仍读它）。
      2. `excel_service.go:448-461`：`:454-455` 的 `filterScopeWriteImportRows(...)` 调用与 `:460` 的 `result.ScopeOutOfScope = outOfScope` **保留不动**（D-6 接口契约 + 静态断言 token 不变）。把 `:448-451` 的注释（「计入 ImportResult 错误明细（含行号、不含敏感值）与聚合越界计数」）改写为「越界行静默跳过：不写入、不阻断整批、不进 `Errors`/`Failed`；仅 `outOfScope` 入服务端 operlog；All 档短路零校验零查询」。
      3. `excel_service.go:62-67` `ImportError`：**结构体不改**（`Row` 仍供普通校验错误使用，属用户自有文件内容，不构成越界规模泄漏）；在 `Row` 字段注释补一句「越界行不再产生 `ImportError`（Phase 150），本字段仅用于用户上传文件自身的校验错误定位」。
      4. `excel_handler.go:139-147`：`operlog.Record` 的 OperParam **保留 `outOfScope`**（`result.ScopeOutOfScope` 仍可读，审计链不降级），`opsServices.ScopeWriteTier(...)` 保留，`OperTypeImport` 常量值不重排；注释 `:141-142` 补「Phase 150：越界聚合**仅在此审计链**，HTTP 响应体零回显」。
      5. `excel_handler.go:148-153` `gin.H`：**形状保持 `inserted/updated/failed/errors` 不变**。因 E1 已让越界行不 `Failed++`、不 append `Errors`，此处 `failed` 与 `errors` 长度**不再随 N−M 变化**（F-4 间接泄漏面根治）。**新增注释**说明「`failed` 仅统计用户上传文件自身的校验失败，不含越界行数；`errors` 不含越界行」。
      6. **不要**改 `excel_export_service.go` / `excel_export_config.go`（归 148），**不要**改 `workstation_device_service.go`（归 148），**不要**改 `.golangci.yml`（归 148）。`excel_service.go` ≠ `excel_export_service.go`，务必逐字区分。
      7. **不要**把 `ImportResult.Errors` 整个删掉或改成粗粒度桶 / 频限 / 告警通道（禁区）。
    </action>
    <acceptance>
      越界行不进 `ImportResult.Errors`、不计入 `ImportResult.Failed`；`ScopeOutOfScope` 带 `json:"-"`，`go test` 静态断言其永不序列化；operlog 仍记录 `scopeTier + outOfScope`；`gin.H` 响应体对「同一份上传文件、不同 N−M」产生**逐字节相同**的 `failed`/`errors` 部分。
    </acceptance>
  </task>

  <task id="T3" executor="E3" req="SIDECHANNEL-03（前端/验收/All 档部分）">
    <name>作废四类「共 N 条」文案 + 确认框改可见集内表述 + UAT 作废重跑登记</name>
    <files>
      xingran-react-frontend/src/pages/operations/workstations/hooks/useBackfillDept.tsx;
      xingran-react-frontend/src/types/operations.ts;
      xingran-react-frontend/src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx;
      .planning/phases/150-a-sidechannel-01-02-03/150-UAT-BACKFILL.md（新增）
    </files>
    <read_first>本 plan §3 F-5/F-7、§4 D-1/D-2/D-5/D-8；132-SUMMARY.md:74（只读）；useBackfillDept.tsx:110-160；types/operations.ts:111-129；useBackfillDept.test.tsx:34-45 与 :204-280；133-UAT.md:79-91（只读，确认 D1/D2 归 149）</read_first>
    <action>
      1. `src/types/operations.ts:111-129` `BackfillDeptResult`：**删除 6 个可选字段** `scopeDeptTotal`(:124) / `scopeTotal`(:125) / `scopeVisible`(:126) / `scopeOutOfScope`(:127) / `skippedNoUser`(:117) / `skippedBadUserDept`(:118)；**保留** `dryRun/fixMismatch/candidates/toFill/toCorrect/skippedAlreadyValid/samples/scopeTier/scopeDeptNames/scopeFailClosed`（与 D-2 后端字段集逐字对齐）。注释改为「Phase 150：不含任何全库规模 / 越界规模回显」。
      2. `useBackfillDept.tsx:110-135` `rangeText()`：**四类「本次范围：…（共 N 条）」文案全部作废**，改为不含全库规模、不含任何数值的**可见集内表述**。建议实现（executor 可微调措辞，但**禁止出现 `共 N 条` / `total` / 任何数字**）：
         - FailClosed → `本次范围：无（数据权限异常，请联系管理员）`（原有措辞保留）；
         - All → `本次范围：全部工位`（任务书许可保留「全库」语义，但去掉「共 N 条」）；
         - Self → `本次范围：仅本人关联工位`；
         - Custom/Dept/DeptChild → `本次范围：当前数据权限可见的部门及子部门`（可用 `scopeDeptNames` 拼部门名，但**不得**拼 `scopeDeptTotal` 的「等 M 个部门」数值）。
         **删除 `:116` 的 `const total = p?.scopeTotal ?? p?.candidates ?? 0` 与 `:129` 的 `deptTotal` 读取**；`:117` 的 `tier`、`:118` 的 `scopeFailClosed` 保留。
      3. `useBackfillDept.tsx:141-163` `renderContent()`：`:147-150` 的「共匹配到 N 个绑定了有效用户的工位」（`candidates`，**可见集内**）**保留**；`:156-162` 的「跳过（无用户 / 用户部门无效 / 已一致）N 条」按 D-2 改为**只用 `skippedAlreadyValid`**，文案改「跳过（已一致） N 条」，**删掉 `skippedNoUser` 与 `skippedBadUserDept` 两个加数**（字段已删）。`:137-139` `okButtonProps` **保留**（D-8）。`:164-170` Alert、`:172-194` Checkbox、`:195-204` Samples Table 逻辑不动（注意 `useEffect`/`useCallback` 依赖稳定，不改 `open` 的 `[loading, onSuccess]`）。
      4. `useBackfillDept.tsx:110-113` 注释改写为「Phase 150：不再消费范围快照规模，确认框文案不含全库规模；`scopeTier`/`scopeDeptNames`/`scopeFailClosed` 仅作可见集内提示与 fail-closed 门控」。
      5. `useBackfillDept.test.tsx`：`preview()` fixture（`:34-45`）删 `skippedNoUser`/`skippedBadUserDept`；`:204-280` 的 6 组 `scopeTier:... scopeTotal:... scopeDeptTotal:...` 用例**全部改写**：把 `scopeWriteRangeText(cfg)` 的断言从「含 `共 N 条`」改为「**不含 `共` / 不含数字 / 以 `本次范围：` 开头**」，并新增断言「确认框文本不含 `scopeTotal`/`scopeDeptTotal` 派生数值」。FailClosed 用例保留 `okButtonProps.disabled === true` 断言。`collectScopeWriteTexts` helper 不动。
      6. **新建 `.planning/phases/150-a-sidechannel-01-02-03/150-UAT-BACKFILL.md`**（D-5），内容必须含：
         - 标题「Phase 150 · 132/133 backfill UAT 作废与重跑登记」；
         - **作废清单**：`132-SUMMARY.md:65`（范围快照回显 / 预览与执行两次 ScopeResult 派生）、`:74`（四类「本次范围：…（共 N 条）」文案）、`:103`（「P1-5 已知残余风险、接受；不假装已解决」裁定）→ 均因 DECISION-① 推翻语义而作废；
         - **改写登记**：`133-UAT.md` A6「越界计数入 operlog、不整批 403」→「越界计数**仅**入 operlog、HTTP 响应体零回显、仍不整批 403」；
         - **重跑窗口**：写明「由 QA 裁定重跑窗口与验收人，AI 不擅自勾选完成，79 条人工验证跟踪表保持 0 勾选」；
         - **边界声明**：`133-UAT.md:79-91` 的 **D1/D2 段归 Phase 149，本期一字不动**；`.planning/milestones/` 归档件本期只读；
         - **All 档零回归复核项**：admin 的 `candidates/toFill/toCorrect/skippedAlreadyValid/samples` 与 130 前一致、前端可见「全部工位」措辞。
      7. **禁止**修改 `.planning/milestones/` 下任何文件；**禁止**碰 `133-UAT.md` 的 D1/D2 段（归 149）；**禁止**碰 `useFloorPlanEditor.ts`（归 149）；**禁止**用 wrapped API 之外的裸 axios。
    </action>
    <acceptance>
      前端源码全文检索 `scopeTotal`/`scopeDeptTotal`/`scopeVisible`/`scopeOutOfScope`/`skippedNoUser`/`skippedBadUserDept` = 0 命中；`rangeText()` 输出不含任何数字；`npm run type-check` 通过；`npx vitest run useBackfillDept.test.tsx` 全绿；`150-UAT-BACKFILL.md` 含作废清单 / 重跑窗口 / D1/D2 边界声明。
    </acceptance>
  </task>

  <task id="T4" executor="E2" req="SIDECHANNEL-03（差分攻击测试部分）">
    <name>新增差分攻击测试 + pg_contract 静态断言改写（静态 + SQLite 真跑，不伪造 PG 双跑）</name>
    <files>
      internal/services/operations/sidechannel_diff_attack_test.go（新增）;
      internal/services/operations/pg_contract_test.go
    </files>
    <read_first>本 plan §3 F-6、§4 D-6/D-7；pg_contract_test.go:1-74 与 :254-301；scope_write_403_test.go:306-337（复用 `write403BuildFixture` / `write403Ctx` / `write403DeptOnly` 等 fixture helper）；workstation_backfill_dept_test.go:setupBackfillDeptDB</read_first>
    <action>
      1. **新增 `internal/services/operations/sidechannel_diff_attack_test.go`**（`package operations`，就近 `*_test.go`，用 `github.com/glebarez/sqlite` + `gorm` + `testify`，沿用既有 fixture helper，**零新依赖**）。至少 4 个测试：
         - **(a) `TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields`**：用 `write403Ctx(write403DeptOnly(...))` 调 `BackfillDeptFromUser(dryRun=true)`，把结果 `json.Marshal` 后断言**不含** `scopeTotal`/`scopeVisible`/`scopeOutOfScope`/`scopeDeptTotal`/`skippedNoUser`/`skippedBadUserDept` 六个键；断言**含** `candidates`/`toFill`/`toCorrect`/`skippedAlreadyValid`/`samples`。
         - **(b) `TestSideChannel_Backfill_DiffAttack_OnlyVariesWithM`**（核心）：构造 **两组 SQLite fixture，可见行数 M 固定、全库 N 不同**（例：N=5/M=2 与 N=9/M=2，越界行 3 vs 7，均带有效/无效用户混合以覆盖 `:644` 候选条件）。对每组用 **Dept 与 DeptChild 两个相邻 scope** 各请求一次（共 4 次请求），把 4 个响应体 `json.Marshal` 后按「档位」分组断言：**同档位下两组 N 不同但 M 相同的响应体逐字节相等**（即所有数值字段仅随 M 变化、不随 N−M 变化）。再断言 `samples` 长度非 All 档恒 0、`candidates <= M`。
         - **(c) `TestSideChannel_Import_DiffAttack_SilentSkip`**：构造同一份上传记录（固定若干普通校验错误），改变越界行数 N−M，调 `filterScopeWriteImportRows` 后断言 `result.Failed` 与 `len(result.Errors)` **两组相等**（不随越界行数变化）、越界行不进 `Errors`、`outOfScope` 返回值仍正确递增（审计链保留）、`kept` 只含可见行、`__excel_row__` 不泄漏进 `kept`。断言 `ImportResult` 的 `json.Marshal` **不含** `scopeOutOfScope` 键（`json:"-"` 生效）。
         - **(d) `TestSideChannel_RepeatedRequests_CannotEnumerateN`**：对同一非 All 档 scope 重复请求 10 次，收集全部响应体的数值字段集合（`Row` / 数组长度 / `total` / `failed` / 状态码），断言其中**不存在任何随 N−M 单调变化的量**；显式断言响应体不含 `"total"` / `"row"`（`row` 仅可能来自用户自有校验错误，本用例构造零校验错误的文件以隔离）。
         - **All 档零回归用例**：admin（All）响应体的 `candidates/toFill/toCorrect/skippedAlreadyValid` 与 130 前既有断言一致、`samples` 长度 ≤20（复用 `scope_write_403_test.go:327-331` 口径）。
      2. `pg_contract_test.go:254-271` `TestScopeWriteImport_StaticContract` 改写：
         - **保留** `assertImportHook("filterScopeWriteImportRows(", ...)` 与 `assertImportHook("result.ScopeOutOfScope = outOfScope", ...)`（D-6 下两 token 仍在，且后者现在是「内部字段供 operlog」）。
         - `:270` 的 `__excel_row__` 断言**描述改写**为「导入行号标记保留供内部定位，但越界行不写入响应体」（token 本身仍在 `excel_service.go:386`）。
         - **新增**读 `workstation_service.go` 与 `excel_service.go` 源码的静态断言：`BackfillDeptResult` 结构体不含 `ScopeTotal`/`ScopeVisible`/`ScopeOutOfScope`/`ScopeDeptTotal`/`SkippedNoUser`/`SkippedBadUserDept` 字段名；`ScopeOutOfScope` 的 tag 为 `json:"-"`；`workstation_service.go` 不含 `COUNT(*) AS total` 全库口径片段（防 D-3 死代码复活）。
         - **新增**读 `scope_write.go` 的静态断言：越界分支不再含 `result.Errors = append`（防 D-6 回退）。
         - 测试头部注释声明「**静态断言不依赖 `TEST_PG_DSN`，CI 恒跑**」。
      3. `pg_contract_test.go:273-301` `TestPGContract_ScopeWriteImportResolve` **一字不改**（保持 `openPGForContractTest` 的 `t.Skip` 可观测；本期不注入 DSN、**不伪造双跑**）。在文件头注释（`:10-16`）补一行「Phase 150：差分攻击测试为静态断言 + SQLite 真跑；PG 双跑仍为 `DIALECT-04③` 前置依赖，本期不建 `services: postgres`」。
      4. **禁止**在测试里连真实 PG、伪造 `TEST_PG_DSN`、或把 `t.Skip` 改成 `t.Fatal`；**禁止**改 148 的 4+ 个测试文件。
    </action>
    <acceptance>
      `go test -count=1 -run 'TestSideChannel|TestScopeWriteImport_StaticContract' ./internal/services/operations/` 全绿；未设置 `TEST_PG_DSN` 时 `TestPGContract_*` 仍 skip 可观测且不算失败；**反向变异**（T5）必红。
    </acceptance>
  </task>

  <task id="T5" executor="E1+E2+E3 → Lead 汇总" req="自验证">
    <name>自验证（只跑相关包，不跑全仓）+ 反向变异红绿验证</name>
    <files>无新增（只执行验证命令，不改代码）</files>
    <read_first>本 plan §6 must_haves；CLAUDE.md「Compilation & Build Verification」；AGENTS.md 构建产物只落 out/</read_first>
    <action>
      各 executor 在本组改动完成后**各自前台跑**下列命令（**不并发跑同一 `go test` 以免相互干扰**；由 Lead 收齐 3 组输出）：
      1. 后端相关包（**按实际包，不跑全仓**）：
         `go test -count=1 ./internal/services/operations/ ./internal/api/v1/operations/`
         另外单独跑 T4 定点：`go test -count=1 -run 'TestSideChannel|TestScopeWriteImport_StaticContract|TestPGContract' -v ./internal/services/operations/`
      2. 全量编译：`go build ./...`（**必须跑**，CLAUDE.md 硬约定；产物不落根目录，本命令不产生二进制）。
      3. 前端（在 `xingran-react-frontend/` 下）：
         `npm run type-check`
         `npx vitest run src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx`
         （任务书原文 `npx vitest run <useBackfillDept 相关>` 即此文件；可补 `npx prettier --check` 于 3 个改动前端文件。）
      4. **反向变异（E2 执行，必做）**：临时把 `ImportResult.ScopeOutOfScope` 的 tag 改回 `json:"scopeOutOfScope,omitempty"`（或临时在 `BackfillDeptResult` 加回 `ScopeOutOfScope int json:"scopeOutOfScope"`），重跑 T4 的 (a)/(c) 与 `TestScopeWriteImport_StaticContract` → **必须红**；随后**立即还原**并重跑 → **必须绿**。把「红 → 还原 → 绿」两段输出原样贴进 `150-SUMMARY.md` 的验证结果段。**严禁用 `git checkout` 还原**（禁 git 写操作），用 `edit` 工具手动改回。
      5. **不跑** `go test ./...` 全仓（任务书「只跑相关包，不跑全仓」；132-SUMMARY.md:87 记载 `scripts/e2e/TestE2EAllEndpointsLogged` 为 Windows/bash 路径拼接的环境既有红灯，与本 phase 无关）；**不启动**任何长驻进程 / dev server；**不写**生产数据。
    </action>
    <acceptance>
      4 组命令输出齐全且全绿（PG contract 的 `t.Skip` 属预期，不算失败）；反向变异「红→还原→绿」证据完整；`150-SUMMARY.md` 记录每条命令的真实输出结论。
    </acceptance>
  </task>

  <task id="T6" executor="E3（Lead 复核）" req="SUMMARY">
    <name>写 150-SUMMARY.md（改动 / 3 REQ / 验证 / A+B 不伪造闭环 / Known Gaps / 未提交）</name>
    <files>.planning/phases/150-a-sidechannel-01-02-03/150-SUMMARY.md（新增）</files>
    <read_first>本 plan 全文；ROADMAP :94-98 成功标准、:188-200 Known Gaps；T5 的四组输出与反向变异证据</read_first>
    <action>
      新建 `150-SUMMARY.md`，必含 6 段：
      1. **改动清单**：按 E1/E2/E3 分组逐文件列出改了什么（含 D-4 的 handler operlog 调整、D-6 的 `json:"-"`、D-7 新纳入 150 独占的 `workstation_handler.go` / `excel_handler.go` / `pg_contract_test.go` 说明）。
      2. **3 REQ 满足情况**：`SIDECHANNEL-01` / `SIDECHANNEL-02` / `SIDECHANNEL-03` 逐条对照 ROADMAP :95-98 的 5 条成功标准，逐条给「满足 + 证据（测试名 / 命令）」。**不得**把 SIDECHANNEL-04 / ARCH-01（归 151）写成本 phase 完成。
      3. **验证结果**：贴 T5 四组命令结论 + 反向变异「红 → 还原 → 绿」两段输出；写明 `TEST_PG_DSN` 未设置、PG contract 为 skip 可观测、**本期零真实 PG 双跑、不伪造**。
      4. **明确声明（加粗）**：「**B（Phase 151 统一 ScopedDB）未落地前不得宣称 ① 已根治**」；「A(150) 只完成响应体去规模化，**不能防回归**；未来新增端点仍可能漏挂，须由 B 的 GORM `Scopes` 唯一查询起链点 + 响应契约层兜住」；「**未伪造『统一架构已落地』**」。
      5. **Known Gaps（照抄并落地 ROADMAP :159/:200 口径）**：
         - 「All 档四类文案作废需产品 / QA 确认」；
         - 「132/133 backfill UAT 重跑窗口由 QA 裁定，见 `150-UAT-BACKFILL.md`，本期不擅自勾选」；
         - 「FUTURE-01 收档清理 4 处引用（`.planning/REQUIREMENTS.md`(v1.54) / `ROADMAP.md:91` / `STATE.md` / `.planning/notes/261108-v153-locked-residuals.md:63`）由 Lead 在 v1.55 收档时执行，本期不动」；
         - 「`TEST_PG_DSN` 双跑仍为前置依赖（DIALECT-04③），本期不建 `services: postgres`」。
      6. **明确「未 git commit」**：写明「**本 phase 未执行 `git add` / `git commit` / `git stash` / `git checkout` 等任何 git 写操作**；由 Lead 统一提交（frontmatter `commit:false`）」。
      7. **禁止**在 SUMMARY 里写「① 已根治 / 统一架构已落地 / PG 双跑通过 / UAT 已重跑完成」等不实闭环。
    </action>
    <acceptance>
      `150-SUMMARY.md` 6 段齐全；含加粗的 B 未落地声明与「未 git commit」声明；无任何伪造闭环措辞。
    </acceptance>
  </task>

</tasks>

---

## 6. 验证标准（must_haves，逐条对应 ROADMAP Phase 150 成功标准 :94-98 + 任务书补充）

| # | must_have | 对应 ROADMAP 成功标准 | 可观察证据 |
|---|-----------|---------------------|-----------|
| **MH-1** | **响应体零越界规模信息**：`BackfillDeptResult` 不再含 `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal/SkippedNoUser/SkippedBadUserDept`；`workstation_service.go:666-669` 全库 `countQ` 删除（**不是改挂 visibleCond 后仍回显**）；静态 + 响应体断言「无全库规模字段」；`ToFill`/`ToCorrect` 保留 | :95（成功标准 1） | T1 acceptance；T4(a) + `pg_contract_test.go` 新增静态断言；`json.Marshal` 断言 |
| **MH-2** | **导入响应去行号 / 去越界计数**：越界行静默跳过（不进 `Errors`、不计入 `Failed`、不回显 `Row`）；`ImportResult.ScopeOutOfScope` 改 `json:"-"` 仅内部供 operlog；`ScopeWriteOperParam` 的 `requested/visible/outOfScope` 一字不改、保留服务端审计链；HTTP 响应零回显 | :96（成功标准 2） | T2 acceptance；T4(c)；`excel_handler.go` operlog 仍含 `outOfScope` |
| **MH-3** | **前端四类文案作废 + All 档零回归**：132-SUMMARY.md:74 四类「本次范围：…（共 N 条）」与 `useBackfillDept.tsx` 确认框改不含全库规模的可见集内表述；admin 业务数值（candidates/toFill/toCorrect/skippedAlreadyValid/samples）与 130 前一致；**无 `if isAll` 档位分支** | :97（成功标准 3） | T3 acceptance；前端全文检索 6 字段 0 命中；`npm run type-check` + vitest 全绿；T4 All 档用例 |
| **MH-4** | **差分攻击测试**：构造「全库 N 行、可见 M 行」，Dept 与 DeptChild 两个相邻 scope 各请求一次，断言响应体所有数值字段**仅随 M 变化、不随 N−M 变化**；重复请求不得通过 `Row` / 数组长度 / `total` / 状态码枚举出 N；`pg_contract_test.go` 静态断言恒跑 + SQLite 真跑，`TEST_PG_DSN` 仍为前置依赖、**不伪造双跑** | :98（成功标准 4） | T4 四个测试 + `TestScopeWriteImport_StaticContract`；反向变异「红→绿」 |
| **MH-5** | **All 档零回归**（硬约束）：All 档响应业务数值与 130 前一致、前端可保留「全库 / 全部工位」措辞；`datascope` 未注入 ctx → All 兜底短路不变 | ROADMAP :158 / REQUIREMENTS :104 / 任务书 | T3 + T4 All 档用例；`scope_write_403_test.go:327-331` 口径保留 |
| **MH-6** | **operlog 审计链不降级**：`ScopeWriteOperParam` 聚合数保留在服务端审计链、不进 HTTP 响应；Excel 导入 operlog 仍记 `scopeTier + outOfScope`；backfill OperParam 仍记档位 + 可见集内写入计数（D-4）；25 个 OperType 常量值**不重排** | ROADMAP :96/:161 / REQUIREMENTS :99 | T1 step 7、T2 step 4；`internal/utils/operlog/regression_test.go` 常量锁不红 |
| **MH-7** | **A+B 不伪造闭环**：SUMMARY 明确「B（Phase 151 统一 ScopedDB）未落地前不得宣称 ① 已根治」「不伪造统一架构已落地」；`150-SUMMARY.md` 不出现 ① 已根治 / PG 双跑通过 / UAT 已重跑完成等不实措辞 | ROADMAP :91/:103/:139 / 任务书完成条件 | T6 第 4 段加粗声明 + Known Gaps +「未 git commit」 |

---

## 7. 禁止清单（违反即 PLANNING BLOCKED / 执行判红）

1. **git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git reset` / 任何写 `.git` 的动作（`commit:false`，Lead 统一提交）。反向变异还原**用 `edit` 工具手动改回**。
2. **改其它 phase 文件面**（W2 并行许可凭证 = ROADMAP :119-137 函数级归属表）：
   - **148**：`excel_export_config.go`、`excel_export_service.go`（**≠** 本 phase 的 `excel_service.go`）、`workstation_device_service.go`、`.golangci.yml`、148 的 4+ 个测试。
   - **149**：5 个 `wall/door/floor_plan_text/room_photo/location_alias` service、`scope_entities.go`、`internal/api/router.go`（rooms/photos :1174-1177）、`useFloorPlanEditor.ts`、**`133-UAT.md` 的 D1/D2 段**。
   - **151**：`scope_filter.go` + `base/GORMRepository`、`pkg/query`。
   - **W1 全部**：145/146/147 文件面。
   - `.planning/milestones/**` 归档件（含 `132-SUMMARY.md`、`133-UAT.md`）**只读**。
3. **改回整批 403**：越界行仍「不写入、不阻断整批」，破坏 v1.52 批量语义（132-SUMMARY.md:103 裁定）。
4. **用粗粒度桶 / 频限 / 告警通道代替根治**（禁区 ROADMAP :173-181）：禁止「0 / <10 / 10-100 / 100+」桶、请求频限、FUTURE-01 告警通道、`isPostgres` 式档位分支、`if isAll` 回显分支。
5. **动 79 条跟踪表与 LOCKED 裁定原文**：`.planning/notes/261108-human-verification-tracker.md` 保持 0 勾选；`.planning/notes/261108-v153-locked-residuals.md` 及 `261108-*` 归档件只读；FUTURE-01 的 4 处引用由 Lead 收档清理、本期不动。
6. **改 25 个 operlog 常量**（值 / 数量 / 顺序均不动）；不删 `operlog.Record`；`Record` 保持在 `response.Success` 之前。
7. **启动长驻进程**（dev server / watcher / DB / 后台 `&`）；**写生产数据**；不跑全仓 `go test ./...`；构建产物不落仓库根目录（只落 `out/`，本 phase 不产二进制）；不新增 `.xlsx` 夹具落盘（用 `t.TempDir()` 或内存字节）。
8. **不伪造 PG 双跑**：不建 `services: postgres`、不注入/伪造 `TEST_PG_DSN`、不把 `t.Skip` 改 `t.Fatal`。
9. **不新增 `data/`**、不在仓库根新增启动/构建脚本；Go 单测就近 `*_test.go`，不建 `test/`（单数）。

---

## 8. 完成条件与交付

- 全部 6 个任务完成、`must_haves` MH-1~MH-7 逐条有证据、T5 四组命令 + 反向变异红绿齐全 → 返回：
  `## PLANNING COMPLETE (Phase 150)` + 任务数（6）/ 验证命令 / 文件面一句话。
- 若出现 **W2 跨 phase 写交集**（例如 `excel_service.go` 被 148 误并、`scope_write.go` 被 149 反向修改、`useBackfillDept.tsx` 与 149 前端面冲突）或上述任一禁止项无法回避 → 返回：
  `## PLANNING BLOCKED` + 具体原因（写交集文件 + 冲突 phase + ROADMAP 归属表行号）。

**本 plan 落盘即视为 Phase 150 planning 产物；执行由 `/gsd-execute-phase 150` 按 E1/E2/E3 三路并行、Lead 汇总提交。**
