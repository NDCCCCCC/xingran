# Phase 150 · ① 侧信道根治 A — 响应体去规模化（SUMMARY）

> 执行：`/gsd-execute-phase 150` 的 150 executor（E1+E2+E3 合并执行，文件面与 148/149/151 互斥）。
> 依据：`150-PLAN.md`（314 行 / 6 Task / MH-1~MH-7 / 裁决 D-1~D-8 / F-4~F-5）、ROADMAP Phase 150 段、REQUIREMENTS SIDECHANNEL-01~03 + DECISION-①、根因报告 §①。
> 输出简体中文；**未执行任何 git 写操作**（见第 6 段）。

---

## 1. 改动清单（按 E1/E2/E3 分组，逐文件）

### E1 — T1：响应体去规模化 + 消除全库 COUNT 双口径（SIDECHANNEL-01）

| 文件 | 改动 |
|------|------|
| `internal/services/operations/workstation_service.go` | `BackfillDeptResult` **删除 6 个字段** `ScopeTotal/ScopeVisible/ScopeOutOfScope/ScopeDeptTotal/SkippedNoUser/SkippedBadUserDept` 及其注释；保留 `DryRun/FixMismatch/Candidates/ToFill/ToCorrect/SkippedAlreadyValid/Samples/ScopeTier/ScopeDeptNames/ScopeFailClosed`；`:124-126` 范围快照注释改写为「Phase 150 不再回显任何全库规模 / 越界规模」（D-1/D-2）。`BackfillDeptFromUser`：`scopeWriteBackfillSnapshot` 的 `deptTotal` 改用 `_` 接收、不再写 `result.ScopeDeptTotal`；删 `result.ScopeVisible = len(candidates)`；**整段删除** `:655-682` 的 `stats`/全库 `countQ`/`countQ.Scan`/`len(stats)==0` 早返回/5 个派生赋值及「COUNT 保持全库口径」旧产品语义注释（D-3）。`result.Candidates = len(candidates)` 保留（唯一带 `visibleCond` 的候选来源）。`Samples` 非 All 恒空逻辑、**无 `if isAll` 档位分支回显**。 |
| `internal/services/operations/scope_write.go` | `scopeWriteBackfillSnapshot` 注释改为「仅生成档位 / 可见部门名 / FailClosed，不含任何全库或越界规模」（函数保留，D-2）。`ScopeWriteOperParam`（`:730-738`）**JSON 内容一字不改**（`requested/visible/outOfScope` 全保留），仅补注释「仅供 `operlog.WithOperParam` 服务端审计链，严禁写入任何 HTTP 响应体 / `gin.H` / `response.Success` 的 data」（D-4）。`filterScopeWriteImportRows`（`:687-722`）**签名不变**（E1-E2 接口契约）：越界分支改为只 `outOfScope++` + `delete(rec,"__excel_row__")` + `continue`，**删除 `result.Failed++` 与 `result.Errors = append(...)`**；顺带删除越界行号专用死函数 `scopeWriteImportRowNum` 与循环内 `rowNum` 局部变量；`scopeWriteImportRowOK` 注释同步改为 D-6 静默跳过语义。`ImportError.Row` 结构体字段不删（用户自有校验错误仍用）。 |
| `internal/api/v1/operations/workstation_handler.go` | **D-4 连锁**（原 `:444-449` 直接引用 `result.ScopeTotal/ScopeVisible/ScopeOutOfScope`，删字段会编译失败）：backfill `operlog.WithOperParam` 格式串改为 `{"fixMismatch":%t,"candidates":%d,"toFill":%d,"toCorrect":%d,"scopeTier":%q}`，参数 `req.FixMismatch/result.Candidates/result.ToFill/result.ToCorrect/result.ScopeTier`；`operlog.Record` 仍在 `response.Success` **之前**、`OperTypeBatch` 常量不重排。 |
| `internal/services/operations/workstation_backfill_dept_test.go` | `:115-116`、`:249` 删 `SkippedNoUser/SkippedBadUserDept` 断言，改为断言 `Candidates/ToFill/SkippedAlreadyValid`（业务数值零回归）。 |
| `internal/services/operations/scope_write_403_test.go` | `:312-316` 删 `ScopeVisible/ScopeTotal/ScopeOutOfScope` 三条断言，改为 `json.Marshal` 响应体不含 6 个键；保留 `ScopeTier` 与 `Samples` 断言。`resClosed.ScopeVisible` 改为 `resClosed.Candidates == 0`（D-8）。`TestWrite403_ImportRows_OutOfScope` 按 D-6 改：`result.Failed==0`、`result.Errors` 空、删 `Row==3` 断言；保留 `kept`/`outOfScope`/`__excel_row__` 不泄漏断言。 |
| `internal/services/operations/v152_diff_coverage_test.go` | `TestDiffCov_Backfill_StatsError` 改写为 `TestDiffCov_Backfill_NoGlobalCountQ`（全库 `countQ` 已删，打断 `gorm:row` 不再影响回填，防 D-3 死代码复活）；`TestDiffCov_Backfill_StatsLenZero` 改写为 `TestDiffCov_Backfill_EmptyTable`，删 `SkippedNoUser/SkippedBadUserDept` 断言、改断言 `Candidates/ToFill/ToCorrect/SkippedAlreadyValid/Samples`。 |
| `internal/api/v1/operations/workstation_handler_full_test.go` | `:651-656` mock 返回体不含被删字段（原本只填 `ToFill/ToCorrect/Samples`）；新增断言响应体不含 6 个键；保留 `"toFill":3` 断言。 |

### E2 — T2 + T4：Excel 导入去行号 / 去越界计数 + 差分攻击测试（SIDECHANNEL-02/03）

| 文件 | 改动 |
|------|------|
| `internal/services/operations/excel_service.go` | `ImportResult.ScopeOutOfScope` tag 改 **`json:"-"`**（字段名/类型不变，`excel_handler.go` 仍读它，供 operlog）；`ImportError.Row` 结构体不改，仅补注释「越界行不再产生 ImportError，本字段仅用于用户上传文件自身的校验错误定位」；`:448-451` 注释改写为「越界行静默跳过：不写入、不阻断整批、不进 `Errors`/`Failed`；仅 `outOfScope` 入服务端 operlog；All 档短路零校验零查询」。`:454-455` `filterScopeWriteImportRows(...)` 调用与 `:460` `result.ScopeOutOfScope = outOfScope` **保留不动**（D-6 接口契约 + 静态断言 token）。 |
| `internal/api/v1/operations/excel_handler.go` | `operlog.Record` 的 OperParam **保留 `outOfScope`**（`result.ScopeOutOfScope` 仍可读，审计链不降级）、`ScopeWriteTier` 保留、`OperTypeImport` 常量不重排；补注释「Phase 150：越界聚合仅在此审计链，HTTP 响应体零回显」。`:148-153` `gin.H` **形状 `inserted/updated/failed/errors` 不变**，补注释说明 `failed` 仅统计用户上传文件自身校验失败、不含越界行数，`errors` 不含越界行。 |
| `internal/services/operations/pg_contract_test.go` | `TestScopeWriteImport_StaticContract` 保留 `filterScopeWriteImportRows(` 与 `result.ScopeOutOfScope = outOfScope` 两 token、`:270` `__excel_row__` 描述改写为「导入行号标记保留供内部定位，但越界行不写入响应体」；**新增静态断言**（CI 恒跑、不依赖 `TEST_PG_DSN`）：`workstation_service.go` 不含 6 个规模字段名、不含 `COUNT(*) AS total` 全库口径（防 D-3 复活）；`excel_service.go` 的 `ScopeOutOfScope` 为 `json:"-"`；`scope_write.go` 不含 `result.Errors = append`（防 D-6 回退）。文件头补「差分攻击测试为静态断言 + SQLite 真跑；PG 双跑仍为 DIALECT-04③ 前置依赖，本期不建 `services: postgres`」。`TestPGContract_ScopeWriteImportResolve` **一字不改**（`t.Skip` 可观测）。 |
| `internal/services/operations/sidechannel_diff_attack_test.go`（**新增**） | T4 五个测试（详见第 3 段验证）：(a) 响应体零 6 键；(b) 差分攻击「N=5/M=2 与 N=9/M=2，Dept 与 DeptChild 两相邻 scope 各请求一次，同档位两组响应体逐字节相等」；(c) 导入差分「同一上传文件、不同越界行数，`failed`/`errors` 长度相等、`outOfScope` 仍递增、`ImportResult` 序列化不含 `scopeOutOfScope`」；(d) 重复 10 次请求不可枚举 N；(e) All 档零回归。静态断言 + SQLite 真跑，零新依赖。 |

> **D-7 登记**：`workstation_handler.go` / `excel_handler.go` / `pg_contract_test.go` 经核对不在 148/149/151 任一写集合，因 D-4/D-6 连锁与 T4 静态断言必须改，**纳入 150 独占**（ROADMAP :119-137 逐格核对，与 148 交集 = ∅）。

### E3 — T3 + T6 + UAT：前端四类文案作废 + 验收登记（SIDECHANNEL-03）

| 文件 | 改动 |
|------|------|
| `xingran-react-frontend/src/types/operations.ts` | `BackfillDeptResult` **删除 6 个字段** `scopeDeptTotal/scopeTotal/scopeVisible/scopeOutOfScope/skippedNoUser/skippedBadUserDept`；保留与后端逐字对齐的 10 个字段；注释改为「Phase 150：不含任何全库规模 / 越界规模回显」。 |
| `xingran-react-frontend/src/pages/operations/workstations/hooks/useBackfillDept.tsx` | `rangeText()` 四类「本次范围：…（共 N 条）」全部作废，改为不含任何数值的可见集内表述：FailClosed `本次范围：无（数据权限异常，请联系管理员）` / All `本次范围：全部工位` / Self `本次范围：仅本人关联工位` / Custom·Dept·DeptChild `本次范围：当前数据权限可见的部门及子部门`；删 `total` 与 `deptTotal` 读取。`renderContent()` 跳过文案删 `skippedNoUser`/`skippedBadUserDept` 两个加数，改「跳过（已一致）」只用 `skippedAlreadyValid`；`candidates` 匹配文案、`okButtonProps`（D-8）、Alert / Checkbox / Samples Table 逻辑不动；`useEffect`/`useCallback` 依赖 `[loading, onSuccess]` 不变。 |
| `xingran-react-frontend/src/lib/opsApi.ts` | **编译连锁**（F-4 未逐字点名，但删接口字段后 `backfillDeptFromUser` 的 fallback 对象字面量含 2 个被删字段会 TS 报错）：fallback 删 `skippedNoUser: 0` / `skippedBadUserDept: 0`，其余形状不变。该文件非 `api.ts` barrel、不与 148/149/151 相交，属最小必改连锁，在此登记。 |
| `xingran-react-frontend/src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx` | `preview()` fixture 删 2 字段；6 组 `scopeTier` 用例如数改写：断言「以 `本次范围：` 开头、不含 `共`/任何数字/规模量词」，新增「确认框文本不含派生规模数值」；FailClosed 保留 `okButtonProps.disabled === true`；update() 刷新链路用例同步。`collectScopeWriteTexts` helper 不动。 |
| `.planning/phases/150-a-sidechannel-01-02-03/150-UAT-BACKFILL.md`（**新增**） | D-5 作废 / 重跑登记件：132-SUMMARY.md:65/:74/:103 作废清单、133-UAT.md A6 改写登记、重跑窗口与验收人由 QA 裁定（不擅自勾选）、D1/D2 归 149 边界声明、All 档零回归复核项。不改 `.planning/milestones/` 归档件、不碰 133-UAT.md D1/D2、不勾选任何完成。 |
| `.planning/phases/150-a-sidechannel-01-02-03/150-SUMMARY.md`（本文件） | 6 段齐全（改动 / 3 REQ / 验证 / A+B 不伪造闭环 / Known Gaps / 未 git commit）。 |

---

## 2. 3 REQ 满足情况（逐条对照 ROADMAP :94-98 的 5 条成功标准）

| REQ | 成功标准 | 结论 | 证据 |
|-----|----------|------|------|
| **SIDECHANNEL-01** | :95 标准 1：`BackfillDeptResult` 不再含 6 个规模字段；`:666-669` 全库 `countQ` 删除；静态 + 响应体断言「无全库规模字段」；`ToFill`/`ToCorrect` 保留 | **满足** | `workstation_service.go` 结构字段与整段 `countQ` 删除；`TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields`（json.Marshal 6 键断言）；`TestScopeWriteImport_StaticContract` 新增源码静态断言；`scope_write_403_test.go` 响应体 6 键断言 |
| **SIDECHANNEL-02** | :96 标准 2：`ImportResult.Errors[].Row` 与 `ScopeOutOfScope` 不回显越界位置/数量；越界行静默跳过或仅入 operlog；`ScopeWriteOperParam` 聚合留审计链、HTTP 零回显（审计链不降级） | **满足** | `filterScopeWriteImportRows` 越界分支不 `Failed++`、不 append `Errors`；`ImportResult.ScopeOutOfScope` 改 `json:"-"`；`excel_handler.go` operlog 仍记 `scopeTier + outOfScope`；`ScopeWriteOperParam` 的 `requested/visible/outOfScope` 一字未改；`TestSideChannel_Import_DiffAttack_SilentSkip`；`TestWrite403_ImportRows_OutOfScope` 改后断言 |
| **SIDECHANNEL-03** | :97 标准 3：132-SUMMARY.md:74 四类文案与确认框作废、改可见集内表述；All 档零回归；**无 `if isAll` 分支** | **满足** | `useBackfillDept.tsx` `rangeText()` 四类固定文案（零数值）；前端源码全文检索 6 字段 **0 命中**；`npm run type-check` 全绿、`useBackfillDept.test.tsx` 10/10 全绿；`TestSideChannel_Backfill_AllTier_ZeroRegression`；`150-UAT-BACKFILL.md` 登记 |
| **SIDECHANNEL-03** | :98 标准 4：差分攻击「全库 N / 可见 M，Dept 与 DeptChild 两相邻 scope 各请求一次，响应体数值仅随 M 变化、不随 N−M 变化；重复请求不得通过 Row/数组长度/total/状态码枚举 N」；静态断言恒跑 + SQLite 真跑，`TEST_PG_DSN` 仍为前置依赖、**不伪造双跑** | **满足** | `TestSideChannel_Backfill_DiffAttack_OnlyVariesWithM`（N=5/M=2 vs N=9/M=2 × Dept/DeptChild，4 次请求按档位分组逐字节相等）；`TestSideChannel_RepeatedRequests_CannotEnumerateN`（重复 10 次 + 多请求组合）；`TestSideChannel_Import_DiffAttack_SilentSkip`；反向变异「红 → 还原 → 绿」；`TestPGContract_*` 未设置 DSN 时 `t.Skip` 可观测 |
| **MH-5 All 档零回归 / MH-6 operlog 不降级 / MH-7 不伪造闭环** | ROADMAP :158/:96/:161/:103/:139 | **满足** | All 档业务数值零回归用例；`ScopeWriteOperParam` 保留 + Excel import operlog 保留 `outOfScope` + backfill OperParam 记档位与可见集内计数；25 个 OperType 常量未动（`internal/utils/operlog/regression_test.go` 未红）；本 SUMMARY 第 4/6 段加粗声明 |

> **不纳入本 phase**：`SIDECHANNEL-04` / `ARCH-01`（归 **Phase 151**，B = 统一 ScopedDB 防回归）明确**未**写成本 phase 完成。

---

## 3. 验证结果（T5 四组命令 + 反向变异红绿，均为真实输出）

> 验证窗口说明：148/149 与本 phase 并行写同一 `internal/services/operations/**`。以下四组命令与反向变异先于 **148/149 收口后的稳定窗口**（14:17 与 14:24 两次复现两包全绿）取得真实输出；其后 148 进入新一轮测试文件编辑出现与本 phase 无关的瞬时波动，**148 于 14:55 宣告 COMPLETE 并收口**（`go test` 两包全绿、`go build ./...` exit 0、lint 0 issues），本 phase 于 **15:01 在 148 收口后的最终磁盘态完整复跑 5 组命令，全部全绿**（见下方 [1']~[5']）。本 phase 一字未改 148/149 文件面。

### [1] `go test -count=1 ./internal/services/operations/ ./internal/api/v1/operations/`

```
ok  	github.com/xingran-next/xingran-go-backend/internal/services/operations	8.816s
ok  	github.com/xingran-next/xingran-go-backend/internal/api/v1/operations	0.496s
```
**全绿**（含本 phase 全部连锁测试与 T4 五个测试）。

### [2] `go test -count=1 -run 'TestSideChannel|TestScopeWriteImport_StaticContract|TestPGContract' -v ./internal/services/operations/`

```
--- SKIP: TestPGContract_OpsListJoins (0.00s)
--- SKIP: TestPGContract_WorkstationOrgAndBackfill (0.00s)
--- SKIP: TestPGContract_ExcelExportScope (0.00s)
--- PASS: TestScopeWriteImport_StaticContract (0.00s)
--- SKIP: TestPGContract_ScopeWriteImportResolve (0.00s)
--- PASS: TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields (0.03s)
--- PASS: TestSideChannel_Backfill_DiffAttack_OnlyVariesWithM (0.03s)
--- PASS: TestSideChannel_Import_DiffAttack_SilentSkip (0.00s)
--- PASS: TestSideChannel_RepeatedRequests_CannotEnumerateN (0.02s)
--- PASS: TestSideChannel_Backfill_AllTier_ZeroRegression (0.01s)
PASS
ok  	github.com/xingran-next/xingran-go-backend/internal/services/operations	0.340s
```
**5 个 `TestSideChannel*` + 静态断言全绿**；4 个 `TestPGContract_*` 为 `TEST_PG_DSN` 未设置时的 `t.Skip` 可观测（预期，不算失败）。**本期零真实 PG 双跑、不伪造**。

### [3] `go build ./...`

```
BUILD_DONE   （exit 0，无任何错误输出）
```
**全量编译通过**（产物不落仓库根目录，本命令不产生二进制）。

### [4] `cd xingran-react-frontend && npm run type-check`

```
> xingran-react-frontend@0.0.0 type-check
> tsc --noEmit -p tsconfig.app.json
（exit 0，无错误输出）
```
**通过**。前端源码全文检索 `scopeTotal/scopeDeptTotal/scopeVisible/scopeOutOfScope/skippedNoUser/skippedBadUserDept` = **0 命中**。

### [5] `npx vitest run src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx`

```
 RUN  v4.1.10 D:/CODE/ClaudeCode/ruoyi-go-backend/xingran-react-frontend
 ✓ src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx (10 tests) 59ms
 Test Files  1 passed (1)
      Tests  10 passed (10)
```
**10/10 全绿**。

### [6] 反向变异「红 → edit 还原 → 绿」（T5 step 4，必做）

变异动作：用 `edit` 把 `excel_service.go` 的 `ScopeOutOfScope int \`json:"-"\`` 临时改回 `` `json:"scopeOutOfScope,omitempty"` ``；重跑 `TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields` / `TestSideChannel_Import_DiffAttack_SilentSkip` / `TestScopeWriteImport_StaticContract`。

**① 红（变异态，exit 1）**：
```
=== RUN   TestScopeWriteImport_StaticContract
    pg_contract_test.go:303: ImportResult.ScopeOutOfScope 必须为 json:"-"（仅供 operlog，永不进 HTTP 响应）
--- FAIL: TestScopeWriteImport_StaticContract (0.00s)
=== RUN   TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields
--- PASS: TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields (0.03s)
=== RUN   TestSideChannel_Import_DiffAttack_SilentSkip
    sidechannel_diff_attack_test.go:267:
        Error: "{...\"failed\":1,\"errors\":[...],\"scopeOutOfScope\":2}" should not contain "scopeOutOfScope"
        Error: "{...\"failed\":1,\"errors\":[...],\"scopeOutOfScope\":6}" should not contain "scopeOutOfScope"
    sidechannel_diff_attack_test.go:278: Not equal:   （不同越界行数下响应体不再逐字节相等 → 差分泄漏复活）
--- FAIL: TestSideChannel_Import_DiffAttack_SilentSkip (0.00s)
FAIL
FAIL	github.com/xingran-next/xingran-go-backend/internal/services/operations	0.259s
```

**② 用 `edit` 还原**（严禁 `git checkout`）：`json:"scopeOutOfScope,omitempty"` → `json:"-"`。

**③ 绿（还原态，exit 0）**：
```
=== RUN   TestScopeWriteImport_StaticContract
--- PASS: TestScopeWriteImport_StaticContract (0.00s)
=== RUN   TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields
--- PASS: TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields (0.03s)
=== RUN   TestSideChannel_Import_DiffAttack_SilentSkip
--- PASS: TestSideChannel_Import_DiffAttack_SilentSkip (0.00s)
PASS
ok  	github.com/xingran-next/xingran-go-backend/internal/services/operations	0.297s
```

结论：响应体去规模化（静态断言 + 行为断言）**双向守护有效**；越界计数复活即被差分攻击测试与静态断言同时拦下。

### [1']~[5'] 148 收口后最终磁盘态完整复跑（15:01，全部全绿）

[148 于 14:55 宣告 COMPLETE：`go test` 两包全绿、`go build ./...` exit 0、lint 0 issues；并移交一处注释连锁 —— 148 把 `TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 更名为 `TestExp77_ExportData_WorkstationMainSheet_FixtureBoundary`，本 phase 已把 `pg_contract_test.go:222` 注释同步改为新名（纯注释，属 150 独占文件面）。]

```
=== [1'] go test -count=1 ./internal/services/operations/ ./internal/api/v1/operations/ ===
ok  	github.com/xingran-next/xingran-go-backend/internal/services/operations	32.307s
ok  	github.com/xingran-next/xingran-go-backend/internal/api/v1/operations	0.500s

=== [2'] go test -count=1 -run 'TestSideChannel|TestScopeWriteImport_StaticContract|TestPGContract' ./internal/services/operations/ ===
ok  	github.com/xingran-next/xingran-go-backend/internal/services/operations	0.471s

=== [3'] go build ./... ===
BUILD_EXIT_OK   （exit 0，无错误输出）

=== [4'] cd xingran-react-frontend && npm run type-check ===
> tsc --noEmit -p tsconfig.app.json   （exit 0，无错误输出）

=== [5'] npx vitest run src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx ===
 ✓ src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx (10 tests) 56ms
 Test Files  1 passed (1)
      Tests  10 passed (10)
```

**最终结论：5 组命令在 148/149 全部收口后的磁盘态全部全绿；本 phase 交付可直接由 Lead 统一集成。**

---

## 4. 明确声明（A+B 不伪造闭环）

- **B（Phase 151 统一 ScopedDB）未落地前不得宣称 ① 已根治**。
- **A(150) 只完成响应体去规模化，不能防回归**：未来新增端点仍可能漏挂可见集，必须由 B 的 GORM `Scopes` 唯一查询起链点 + 响应契约层兜住。
- **未伪造「统一架构已落地」**：本 phase 未触碰 `base/GORMRepository`、`pkg/query`、`scope_filter.go`（归 151），未宣称 ① 已根治、未宣称 PG 双跑通过、未宣称 UAT 已重跑完成。
- **不伪造 PG 双跑**：`TEST_PG_DSN` 未设置，`TestPGContract_*` 为 `t.Skip` 可观测、本期零真实 PG 验证；差分攻击为静态断言 + SQLite 真跑。

---

## 5. Known Gaps（照抄并落地 ROADMAP :159/:200 口径）

1. **「All 档四类文案作废需产品 / QA 确认」**：`rangeText()` 已改为零数值可见集内表述，作废登记见 `150-UAT-BACKFILL.md`，但产品 / QA 的正式确认仍待人工。
2. **「132/133 backfill UAT 重跑窗口由 QA 裁定，见 `150-UAT-BACKFILL.md`，本期不擅自勾选」**：79 条人工验证跟踪表保持 0 勾选、不写生产数据。
3. **「FUTURE-01 收档清理 4 处引用（`.planning/REQUIREMENTS.md`(v1.54) / `ROADMAP.md:91` / `STATE.md` / `.planning/notes/261108-v153-locked-residuals.md:63`）由 Lead 在 v1.55 收档时执行，本期不动」**：本 phase 未触碰这些文档与 LOCKED 裁定原文。
4. **「`TEST_PG_DSN` 双跑仍为前置依赖（DIALECT-04③），本期不建 `services: postgres`」**：见第 4 段声明。
5. 并行执行环境说明（**已全部收口，最终态全绿**）：148 / 149 / 150 同写 `internal/services/operations/**`，验证期间先后观察到 148（`excel_export_*` 的 `RightCast` 中间态、`workstation_device_service.go` 的 `sort` 未使用 import、`workstation_device_77_01_test.go` 2 个 `TestWSD77_*` 断言波动、`workstation_device_physical_test.go` 写到一半的 `expected '}', found 'EOF'`、`TestExp77_ExportData_WorkstationAppendSheets` 波动）与 149（`scope_entities_scope149_test.go` 未使用 import、`TestScope149_*` 8→2 个 FAIL）的瞬时编译 / 断言中间态；均为其它 phase 独占文件面，本 phase **一字未改**。148 于 14:55 宣告 COMPLETE 并收口（两包全绿 / build exit 0 / lint 0 issues），本 phase 于 15:01 在最终磁盘态完整复跑 5 组命令全部全绿（见第 3 节 [1']~[5']）。148 移交的 `pg_contract_test.go:222` 注释测试名更名连锁已同步处理。

---

## 6. 未 git commit

**本 phase 未执行 `git add` / `git commit` / `git stash` / `git checkout` / `git reset` 等任何 git 写操作**；反向变异还原使用 `edit` 工具手动改回（严禁 `git checkout`）。全部改动留在工作区，由 **Lead 统一提交**（frontmatter `commit:false`；3 个 executor 并行写同一工作区）。
