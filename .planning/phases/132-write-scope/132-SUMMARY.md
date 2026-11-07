# Phase 132：四域写端点收口 + backfill 范围 + 前端确认框 + Excel 导入收口 + PG contract

- **状态**：已交付（代码由 Lead 代提交，commit `b02e1bac`）
- **波次**：130 → 131 → 132 严格串行；前置 130（`06e892ee`/`cb04d0f1`/`1fd31fad`）与 131（`e62bc7b6`）均已落地后开工
- **规格依据**（LOCKED，只读消费、不重新裁定）：
  - `.planning/notes/261107-v152-batch-import-decision.md`（单条写 403 / 批量裁剪 / 导入不豁免 / 范围文案 / PG contract 覆盖）
  - `.planning/notes/261107-v152-self-scope-decision.md`（ScopeProvider 边界 / Self / 0→All / fail-closed）
  - `.planning/REQUIREMENTS.md` DS-04 / DS-06 + 验收硬约束
  - `.planning/ROADMAP.md` Phase 132 条目
  - agent-8 独立审查 P0-2 / P0-3 / P1-5 补强

---

## 1. 交付清单（14 文件，commit `b02e1bac`）

| 文件 | 交付 |
|---|---|
| `internal/services/operations/scope_write.go`（新） | 写侧可见集基建：`ErrScopeDenied` / `checkWriteScope` / `checkWriteScopeEffective` / `checkWriteScopeSelfUser` / `filterVisibleIDs` / `filterVisibleIDsByExists` / `visibleIDsForEntityWrite` / `checkResourceWriteScope` / `checkSubmittedWriteScope` / 导入逐行判定器 / backfill 快照 / operlog 计数 helper / 导出常量 |
| `internal/services/operations/scope_write_test.go`（新） | A 部分厚单测：五档 / 0 值 / 越界 / 空 orgID / 批量裁剪边界 / All 短路零查询 / 显式 403 / Self 用户列 / 跨表有效机构 |
| `internal/services/operations/scope_write_403_test.go`（新） | F 部分 403 越权矩阵：SQLite 内存夹具（AutoMigrate 两部门 × 两套实体） |
| `internal/services/operations/building_service.go` | Create/Update/Delete 403 收口、BatchDelete 裁剪 |
| `internal/services/operations/floor_service.go` | Create/Update/Delete 403 收口（可空 OrgID 按楼宇继承）、BatchDelete 裁剪（后续 SQL 全改用裁剪后 ID） |
| `internal/services/operations/workstation_service.go` | Create/Update/Delete 403 收口、BatchDelete / BatchUpdatePositions 裁剪、`BackfillDeptResult` 增范围快照字段、backfill 三层可见集注入 + Samples 非 All 不回显 |
| `internal/services/operations/infopoint_service.go` | Create/Update/Delete 403 收口、BatchDelete 裁剪 |
| `internal/services/operations/excel_service.go` | `ImportData` 逐行可见集校验（Upsert 之前）、`ImportResult.ScopeOutOfScope`、行号标记 `__excel_row__` |
| `internal/api/v1/operations/workstation_handler.go` | BatchOperation / positions / backfill 的 operlog 补范围档位 + 四类计数 |
| `internal/api/v1/operations/excel_handler.go` | importData operlog 补 `scopeTier` / `outOfScope` |
| `internal/services/operations/pg_contract_test.go` | 静态断言守护「导入路径含可见集校验」+ 导入归属链解析的 `TEST_PG_DSN` 门控覆盖 |
| `xingran-react-frontend/src/pages/operations/workstations/hooks/useBackfillDept.tsx` | 「本次范围」四类文案 + FailClosed disabled + 挂进 `modalRef.current.update()` 刷新链路 + 3 项 antd 6 lint 门禁修复 |
| `xingran-react-frontend/src/pages/operations/workstations/hooks/__tests__/useBackfillDept.test.tsx` | vitest 四类文案 + disabled + update() 刷新链路断言（10 用例） |
| `xingran-react-frontend/src/types/operations.ts` | `BackfillDeptResult` 同步范围快照字段 |

---

## 2. 403 越权矩阵覆盖表

`scope_write_403_test.go` 共 **7 组测试函数 / 12 个 t.Run 子用例**，全部语义断言（校验数据实际状态，非仅返回值）：

| 组 | 覆盖 | 断言要点 |
|---|---|---|
| `TestWrite403_Create_FourDomains` | building / floor / workstation / infopoint × 越界提交 | 单条 403（`IsScopeDenied`）+ 四表行数零变更；All 档四域合法 Create 全放行（零回归） |
| `TestWrite403_Update_FourDomains` | 双向校验：改前资源越界 / 提交侧越界 / 搬向越界工位 / FailClosed | 403 + `org_id`/`workstation_id` 实际值零变更 |
| `TestWrite403_Delete_FourDomains` | 四域按资源归属删除 | 越界 403 零软删；本范围放行；FailClosed 全拒 |
| `TestWrite403_BatchDelete_FourDomains` | 四域批量裁剪（每域独立 fixture 防归属链顺序污染） | 越界行零变更、仅本范围行变更；FailClosed 零变更；All 档全量放行 |
| `TestWrite403_BatchUpdatePositions` | positions 批量裁剪 | 本范围工位 `position_x/y` 写入、越界工位零变更 |
| `TestWrite403_Backfill_ScopeLimited` | 三层可见集 + 范围快照 + Samples | 档位/请求行数/可见行数/越界行数；非 All 档 `Samples` 空；只写可见集内 `dept_id`；All 档 Samples 保留；FailClosed 空集 |
| `TestWrite403_ImportRows_OutOfScope` | 导入逐行校验 | 越界行不写入、不阻断整批、含行号不含敏感值、内部标记不泄漏；All 零回归；FailClosed 全越界 |
| `TestWrite403_ErrScopeDenied_HTTP403` | P0-2 回归守护 | 显式 `HTTPStatus=403`、统一文案「无权操作该数据」 |

另有 `scope_write_test.go` 的 **6 组**纯函数/裁剪单测（含 All 短路零查询、FailClosed 空集、Self 用户列、跨表有效机构、O(1) 集合边界）与 `pg_contract_test.go` 的静态断言。

---

## 3. 裁剪 / 导入语义实现

### 单条写（B）
- `checkWriteScope` / `checkWriteScopeEffective` 从 ctx 取 `datascope.ScopeResult`（复用 131 `scopeFromContext`/`scopeIsAll`/`orgScopeClause`/`selfUserClause`/`effectiveOrgExpr`）：All 短路、FailClosed 拒、DeptIDs 命中放行、Self 按用户列分派、空 orgID fail-closed。
- **P0-2**：`ErrScopeDenied = apperrors.NewWithHTTPStatus(CodePermissionDenied, http.StatusForbidden, "无权操作该数据")`。`pkg/errors/codes.go:205` 把 1020–1029 段（含 1025）默认映射 401 且 `codes_80_04_test.go:47` 已锁为预期，故必须显式 403；`response.HandleServiceError` 只对 `IsAppError` 透传其 `HTTPStatus`，裸 `fmt.Errorf` 会退化 500。
- Update 双向：改前资源归属（`checkResourceWriteScope`，按实体归属链 JOIN 解析有效机构）+ 提交侧（`checkSubmittedWriteScope`，空 orgID 按 buildingID/inherit 解析），防搬出/搬入越界。
- 四域 `validateOrg`/`validateOrgOptional` 原 UUID + 存在性校验保留，之后追加可见集校验；Delete 按资源归属校验。

### 批量 / backfill（C）
- `visibleIDsForEntityWrite` 在 **service 层、任何写 SQL 之前**一次性 `WHERE id IN ? AND <有效机构 ∈ 可见集>` 裁剪（禁止「先取全量再内存过滤」）；四域 BatchDelete、BatchUpdatePositions 全部经它裁剪。
- 越界行剔除 + 计数入 operlog，**不因含越界行 403 整批**；`ScopeWriteOperParam` / `ScopeWriteBatchStats` 提供「范围档位 / 请求行数 / 可见集内行数 / 越界行数」，All 档同样记录档位（审计链完整）。
- **P0-3 backfill**：候选查询（JOIN 后按工位有效机构）、COUNT（保持全库口径 = 请求行数）、CASE UPDATE（service 层裁剪 toUpdate + SQL WHERE 追加可见集 EXISTS 双保险）三层注入；`Samples` 在**非 All 档恒为空**（逐条明细不回显，收敛 P1-5 侧信道）；`BackfillDeptResult` 回显范围快照（档位/部门名/总数/FailClosed），预览与执行两次请求各自从同一 `c.Request.Context()` 的 `ScopeResult` 派生，注释明确「禁止确认阶段重新解析档位」防范围漂移。

### Excel 导入（D）
- `filterScopeWriteImportRows` 在 `ImportData` 的 Upsert 之前（prepare records 阶段）逐行校验：复用 `effectiveOrgExpr` 单一真实源（`scopeWriteResolveEffOrg` 一次 IN 查询批量解析四域归属链，双侧 `CAST AS TEXT`），集合 map `O(1)` 判定避免逐行查库（裁定 §6）。
- 越界行：**不写入、不阻断整批**，计入 `ImportResult.Errors`（含原始行号 `Row`、`Field`、`Error`，**不含敏感值 `Value`**）与 `ImportResult.ScopeOutOfScope` 聚合计数。
- `importData` operlog（`OperTypeImport`）补 `scopeTier` + `outOfScope`。
- `pg_contract_test.go`：静态断言（不依赖 DSN，CI 恒跑）守护导入路径含校验钩子；`TestPGContract_ScopeWriteImportResolve` 统一 `TEST_PG_DSN` 门控、skip 可观测，只读 SELECT 覆盖四实体归属链 SQL 的 PG 类型契约。

### 前端（E）
- `useBackfillDept.tsx` 新增「本次范围：…」四类文案：All → `本次范围：全库（共 N 条）`；Custom/Dept/DeptChild → `本次范围：{部门名}及子部门（共 N 条）`（多部门列前 3 + 「等 M 个部门」）；Self → `本次范围：仅本人关联工位（共 N 条）`；FailClosed → `本次范围：无（数据权限异常，请联系管理员）` 且 `okButtonProps.disabled=true`。
- 文案 + `okButtonProps` 挂进 `modalRef.current.update()` 两处刷新链路与初始 `Modal.confirm`，Checkbox rerender 不丢失。
- 附带修复 lead 转来的 3 项 antd 6 lint:ci 门禁 FAIL（`size="medium"` / `Alert title=` / `mask: { closable: false }`，对齐 `confirmDelete.ts` 范式）。

---

## 4. 闸门输出

| 闸门 | 结果 |
|---|---|
| `go build ./...` | **全绿** |
| `go test -count=1 ./internal/services/operations/` | **ok**（含 A/F/D 全部新测试） |
| `go test -count=1 ./internal/api/v1/operations/` | **ok** |
| `go test -count=1 ./...`（全量，不带 coverprofile） | 除 `scripts/e2e/TestE2EAllEndpointsLogged` 外全绿；该用例报 `/bin/bash: ....scriptsoperlog_e2e_verify.sh: No such file or directory`（exit 127），为 Windows/bash 路径拼接的**环境既有问题**，与 Phase 132 改动无关（未触碰 `scripts/e2e/**`）。`tests/e2e`、`tests/integration`、`pkg/*`、`internal/*` 全部 ok |
| `npx prettier --check`（3 个改动前端文件） | **All matched files use Prettier code style!** |
| `npx vitest run useBackfillDept.test.tsx` | **10 passed (10)** |
| `npm run lint:ci` | **结果: ALL PASS / errors 0 / ≤ 0**（含此前 3 项 C14/C14d/C14f FAIL 已全消） |

---

## 5. Commit

- **`b02e1bac`** `feat(v1.52-p132): 四域写端点 403 收口 + 批量/backfill 可见集裁剪 + 导入逐行校验 + 范围文案`（14 文件，+2099 / -28；由 Lead 在关键路径告急时钟代提交，文件与本地验证一致）
- 前置链：130 `06e892ee` / `cb04d0f1` / `1fd31fad` → 131 `e62bc7b6` → 132 `b02e1bac` → 133 `5161b39f`（part1）

---

## 6. 遗留项 / 残余风险

1. **P1-5（审查）已知残余风险、接受**：聚合计数（越界行数 / N / 行号）仍是枚举侧信道，本 phase 只做到「不逐条回显明细」+ 聚合计数入 operlog，**不假装已解决**。若未来要求「越界即告警」，应在 operlog 之上加告警通道，而非改回整批 403（会破坏 backfill 可用性，裁定 §6）。
2. **building / floor / infopoint 三个 handler 的 `BatchOperation` operlog 仍是裸 `OperTypeBatch`（无四类计数）**：service 层裁剪已落地、越界行不写入的安全语义完整；但「越界计数入 operlog」在这三个 handler 未补（`workstation_handler.go` 的 BatchOperation / positions / backfill 与 `excel_handler.go` 的 import 已补）。`scope_write.go` 已导出 `ScopeWriteOperParam` 供一键接入，属**审计链完整性的小遗留**，建议 133 收尾或后续 quick 补 3 行调用（不影响安全语义）。
3. `scripts/e2e/TestE2EAllEndpointsLogged` 全量 `go test ./...` 红灯为环境既有（bash 脚本路径），非本 phase 引入，见 §4。
4. 未触碰禁区：`internal/api/router.go`、`pkg/datascope/**`、`pkg/middleware/**`、`internal/services/system/**`、`vdi/**`、`scope_filter.go`、`floor_cache_impl.go`、`dropdown_helper.go`、`excel_query_builder.go`、`excel_export_config.go`、任何 `.env*`、`.planning/notes/*`、其它 agent 的测试文件；未改读路径函数；未 `git add -A`；未提交 `coverage.out`；工作区仅剩他人未提交的 `.env.production`（未碰）。
