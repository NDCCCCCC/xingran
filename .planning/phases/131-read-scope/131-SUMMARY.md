# Phase 131 交付总结：四域读端点 + Excel 导出独立收口 + 缓存 scope 维度 + PG contract

- 日期：2026-11-07
- 波次：W2（130 → **131** → 132 → 133 串行），依赖 Phase 130 commit `06e892ee` + `cb04d0f1`
- 提交信息：`feat(v1.52-p131): 四域读端点 + Excel 导出可见集收口 + 缓存 scope 维度 + 403/跨用户厚测试`
- commit hash：见文末「闸门与提交」（提交后回填）

## 1. 交付清单

### A. 可见集过滤基建（新文件）
- `internal/services/operations/scope_filter.go`
  - `scopeFromContext(ctx)` → `datascope.FromContext` 薄封装；`scopeIsAll`（未注入/All 短路）
  - `applyOrgScope(query, ctx, orgExpr)`：All 原样返回（零 DB 查询）/ FailClosed `1=0` / DeptIDs `CAST(<orgExpr> AS TEXT) IN ?`
  - `applyReadScope`：Self 档按实体分派（有用户列 → `user_id = ?`；无用户列 → 空集）
  - `orgScopeClause` / `applyScopeToSub` / `wrapScopeExists`：EXISTS 子查询形态统一注入（工作站/楼层/信息点有效机构跨 `ops_floors`/`ops_buildings`）
  - `applyBuildingScope`：楼宇「自身 org_id ∈ 可见集 OR 其下楼层有效机构 ∈ 可见集」双分支（一栋楼跨机构办公）
  - `applyExcelWorkstationScope`：Excel 两条导出路径共用（但各自独立调用）的可见集注入
  - `scopeRejectsRequestedOrg` / `scopeRejectsRequestedOrgFromContext`：越界 orgId 求交（service 层强制，禁止信任 req.OrgID/params）
  - `scopeCacheKey` / `scopeCacheKeyOf`：All/closed/self+hash/d+hash 四档稳定派生
  - **有效机构表达式全部由调用方传 `effectiveOrgExpr(...)` 结果**（`dept_filter.go` 单一真实源），无内联复制；新 SQL 统一 `CAST(... AS TEXT)` 双侧一致
- `internal/services/operations/scope_filter_test.go`：All 短路 / FailClosed / DeptIDs 命中与不命中 / Self 分派 / 越界 orgId / uuid 双侧 CAST 静态断言 / 导出路径 DryRun 静态断言 / scope key 分档

### B. 四域读端点全量收口（只改读路径函数，写路径零触碰）
| 域 | 收口函数 | 端点覆盖 |
|---|---|---|
| workstation | `Statistics`、`filterScope`(List)、`GetFloorWorkstationsAll`(:813 **真实越权面**)、`GetByID`、`SearchWorkstationOptions`、`CountsByFloor`、`GetWorkstationDeptOptions` | :804/:806/:807/:808/:809/:813/:805 |
| floor | `Statistics`（无参数，从 ctx 注入）、`filterScope`(List)、`SearchFloorOptions`、`GetByID`、`GetTree`（按可见集过滤后再缓存） | :753/:754/:755/:756/:757 |
| building | `Statistics`、`filterScope`(List)、`GetByID`、`SearchBuildingOptions` | :700/:701/:702/:703 |
| infopoint | `Statistics`（无参数，从 ctx 注入）、`filterScope`(List)、`GetByID`、`SearchInfoPointOptions` | :983/:984/:985/:986 |

- `GetByID` 四域按资源归属校验：越界 → `record not found`，保持既有 404/空响应形状
- `floor/Statistics`、`infopoint/Statistics` **未改签名**（handler/前端零改动），改为从 ctx 取 ScopeResult 对聚合查询注入范围
- Self 档：workstation 按 `sys_workstation.user_id`、infopoint 经 `workstation_id` 链到 `w.user_id`；floor/building 无用户列 → 空集（裁定 §2）
- 未挂注入中间件 / All 档：完全短路、零额外 SQL、超管零回归（存量 SQLite 测试 `context.Background()` 全绿证明）

### C. 缓存 key 增 scope 维度（PMODEL-01）
- `dropdown_helper.go`：新增 `BuildScopedDropdownCacheKey(ctx, entity, filters)`（`dropdown:{entity}:s{scope}:{all|h<hash>}`），**旧 `BuildDropdownCacheKey` 签名与逐字行为不变**；`floor_cache_impl.go::SearchFloorOptions` 调用点切 scoped
- `floor_cache_impl.go`：`scopedFloorTreeKey` / `scopedFloorBuildingKey`（`floor:tree:s{scope}` / `floor:building:s{scope}:{id}`）；`InvalidateFloorCache` 改 `floor:tree*` + `floor:building:*` 前缀通配，一次清旧 key 与全部 scope 变体
- 未注入 ctx 保持旧精确 key（存量 `geocoding_photo_floor_test.go` 断言零回归）
- 清点表：[131-CACHE-INVENTORY.md](131-CACHE-INVENTORY.md)（逐 key 现状→目标→停用→失效策略 + 审查 P1-1 登记）
- `scope_cache_key_test.go`：不同 scope 不同 key / All 含 `all` 段 / 相同 scope 稳定 / DeptIDs 顺序无关

### D. Excel 导出独立收口（DS-06：不豁免、不走 List、不依赖 params orgId）
- `excel_query_builder.go::WorkstationQueryBuilder.BuildQuery`：`deleted_at IS NULL` 后注入 `applyExcelWorkstationScope`
- `excel_service.go::queryWorkstationIDsForExport`：直接表查询后注入同源 helper（两条路径各自独立，互不复用）
- 新增条件统一 `CAST(... AS TEXT)`；**未依赖 `params["orgId"]`**（workstation FilterMapping 无 orgId，前端传 orgId 会拼出不存在的列）
- `pg_contract_test.go`：新增 `TestPGContract_ExcelExportScope`（门控 `TEST_PG_DSN`，skip 可观测：`TEST_PG_DSN not set; PG contract test skipped`），覆盖两条导出路径的真实 PG 类型契约
- 静态断言守护「导出路径含可见集条件」：`scope_filter_test.go::TestScopeExport_StaticContract_DryRunSQL`（DryRun 渲染，不执行整条 BuildQuery）

### E. 403 越权矩阵 + 跨用户对照实验
- `scope_read_403_test.go`：SQLite 内存夹具，部门 A/B 各一套楼宇/楼层/工位/信息点 + 六档 profile（deptA/deptB/all/noInject/failClosed/selfA）
- 全部语义断言，覆盖端点 × 五档（越界不可见 / 本范围可见 / All 全可见 / 未注入零回归 / FailClosed 空集 / Self 分档）
- 跨用户对照：同一缓存 key 下用户 A/B 不串数据（`floor:tree` + dropdown 端到端，防 PMODEL-01 重演）

### 审查 P1 清单处理
| 项 | 处理 |
|---|---|
| P1-1 `GET /dept-mapping-template` 全库部门树零过滤 | `excel_handler.go::DownloadDeptMappingTemplate` 最小改动：`datascope.FromContext(ctx)` + `AND CAST(d.id AS TEXT) IN ?`（All 短路不变、Self/FailClosed 空集），响应形状零变更；已在清点表与本文登记 |
| P1-2 `counts-by-floor`/`GetByID`/`dept-options`/floor tree 4 端点 | 全部在 B + 403 矩阵逐点断言 |
| P1-3 三档 DeptIDs 空导致缓存 key 碰撞 | `scopeCacheKeyOf` 四档分派（`all`/`closed`/`self+sha1`/`d+sha1`）已解决，审查确认无需改动 |
| P1-8 `excel_query_builder.go` 既有 `::uuid/::text` PG 专有 JOIN | **如实登记为既有技术债**（`TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 锁定 SQLite 必炸），属 Phase 133 或后续清偿，本 phase 不改；新增可见集条件独立用 `CAST(... AS TEXT)` 双方言，PG contract 测试在有 `TEST_PG_DSN` 时真正跑到新增条件 |

## 2. 403 矩阵覆盖表

| 端点 / 函数 | 本范围可见 | 越界不可见 | All 全可见 | 未注入零回归 | FailClosed 空集 | Self 分档 | 测试 |
|---|---|---|---|---|---|---|---|
| workstation List | ✓ | ✓(越界 orgId) | ✓ | ✓ | ✓ | ✓ | `TestScopeRead_FourDomainList_403Matrix` |
| workstation Statistics | ✓ | — | ✓ | — | ✓ | — | `TestScopeRead_FourDomainStatistics_403Matrix` |
| workstation SearchWorkstationOptions | ✓ | ✓(越界 orgId) | — | — | — | — | `TestScopeRead_DropdownAndTree_403Matrix` |
| workstation GetByID | ✓ | ✓ | ✓ | — | ✓ | — | `TestScopeRead_GetByID_403Matrix` |
| workstation **GetFloorWorkstationsAll(:813)** | ✓ | ✓ | ✓ | — | ✓ | — | `TestScopeRead_WorkstationsAll...` |
| workstation CountsByFloor | ✓ | ✓ | ✓ | — | — | — | 同上 |
| workstation GetWorkstationDeptOptions | DryRun 静态 | ✓(越界/FC 空集) | — | — | ✓ | — | 同上 |
| floor List / Statistics / SearchFloorOptions / GetByID / GetTree | ✓ | ✓ | ✓ | ✓ | ✓ | ✓(空集) | 三个矩阵函数 |
| building List / Statistics / SearchBuildingOptions / GetByID | ✓ | ✓ | ✓ | ✓ | ✓ | ✓(空集) | 三个矩阵函数 |
| infopoint List / Statistics / SearchInfoPointOptions / GetByID | ✓ | ✓ | ✓ | ✓ | ✓ | ✓(user_id) | 三个矩阵函数 |
| Excel `queryWorkstationIDsForExport` | ✓ | ✓ | ✓ | — | ✓ | — | `TestScopeRead_ExcelExport_403Matrix` |
| Excel `WorkstationQueryBuilder.BuildQuery` | DryRun 静态(EXISTS/CAST) | — | ✓(短路) | — | — | — | 同上 + `TestPGContract_ExcelExportScope` |
| 跨用户缓存 `floor:tree` / dropdown | ✓端到端 | ✓不串 | — | — | — | — | `TestScopeRead_CrossUserCacheIsolation` |

> 合计：四域 20 个读函数 + Excel 2 条导出路径 + 1 个补收口端点（dept-mapping-template）+ 2 类缓存 key 跨用户对照，全部有语义断言。

## 3. 闸门输出

| 闸门 | 命令 | 结果 |
|---|---|---|
| 编译 | `go build ./internal/services/operations/ ./internal/api/v1/operations/` | 通过 |
| 定向测试 | `go test -count=1 ./internal/services/operations/... ./internal/api/v1/operations/...` | 全绿 |
| 主闸门 | `go test -count=1 ./internal/services/operations/... ./internal/api/... ./pkg/datascope/...` | 全绿（18 个包 ok） |
| 全量 | `go build ./...` + `go test -count=1 ./internal/... ./pkg/... ./cmd/...` | 见文末回填 |
| PG contract | `go test -run TestPGContract_ExcelExportScope -v` | `--- SKIP: TEST_PG_DSN not set; PG contract test skipped`（门控可观测） |
| 覆盖率 | 不跑 `-coverprofile`（避免与厚测试 agent 的 coverage.out 互相覆盖，按 lead 协调由 `coverage_diffcov.out` 统一出数） | — |

## 4. 与 Phase 130 实际 API 的差异

- **无差异**。Phase 130（`06e892ee` + `cb04d0f1`）实际 API：`ScopeResult{All, DeptIDs, ResourceHook, FailClosed, UserID}`、`FromContext/WithContext/Resolve/AllScope/FailClosedScope`、`HookSelfUser`，与规格预期一致。
- **超出裁定文档的利好**：`ScopeResult` 额外带 `UserID string`（裁定 §2 的伪代码未列），正好支撑 Self 档 workstation/infopoint 按 `user_id` 精确过滤，无需另造 context 通道。
- `DataScopePermission` 第二笔（`cb04d0f1`）已把完整 Resolve 结果（DeptIDs 已展开排序）注入 `c.Request.Context()`，本 phase 直接 `FromContext` 消费，未自己查 `sys_dept`。

## 5. 遗留项（明确移交后续 phase，非本 phase 缺口）

1. **P1-8 / `WorkstationQueryBuilder.BuildQuery` 的 `::uuid/::text` PG 专有 JOIN**：SQLite 必炸（既有测试 `TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 锁定），本 phase 不改；导出可见集条件已用双方言 `CAST`，PG contract 在 `TEST_PG_DSN` 下覆盖。属 Phase 133 或后续清偿。
2. **`dept-options` 的 `Raw("(?) UNION ALL ...")` 在 SQLite 语法不支持**（最小复现证实，属既有生产 SQL 限制）：本 phase 403 测试对「本范围可见」改用 DryRun 静态断言守护可见集 WHERE 注入，真实 UNION 由 PG 路径覆盖；生产行为零改动。
3. asset / serverRoom / dedicatedLine / roomDevice 四实体的完整收口（DS-05）→ **Phase 133**；本 phase 仅保证这些端点不因 130 注入回归（未改动其代码）。
4. 四域写路径（Create/Update/Delete/Batch/Backfill/Import）→ **Phase 132**（严格未触碰，含 `workstation_service.go:296/310/346/451/502-648`、`floor_service.go:110/152/240/508` 等）。
5. 前端 DeptSidebar 联动 → Phase 133；backfill 确认框 → Phase 132；本 phase 前端零改动。
6. `operlog` 导出范围档位：DS-06 允许「可选」，`excel_handler.go` 既有 `OperTypeExport` 日志已记录请求参数摘要，本 phase 未改响应形状、未新增范围档位字段（避免牵动 handler 契约）。

## 6. 闸门与提交（提交后回填）

- commit hash：`__COMMIT_HASH__`
- `go build ./...`：`__BUILD__`
- `go test -count=1 ./internal/... ./pkg/... ./cmd/...`：`__FULL_TEST__`
