# v1.52 独立代码审查清单（2026-11-07）

- **审查员**：agent-8（独立只读，魔鬼代言人 / 安全视角）
- **审查对象**：三份 LOCKED 开工输入裁定 + PR #75 已合入分支的前置生产代码
- **处置原则**：P0 必修、P1 本里程碑内修或登记、P2 登记残余风险

## P0（3 项，全部已处置）

| # | 发现 | 证据 | 处置 |
|---|---|---|---|
| P0-1 | 迁移 235（`ops:workstation:backfill` 权限 seed）被嵌套在 234 的**失败分支**内且重复两次；234 成功（正常路径）时 235 一次不跑；SQLite 分支完全无 235；`migration_235` 无 `_test.go` ⇒ 独立权限项在 PG 正常启动/SQLite 全路径均不 seed | `internal/core/db/database.go:1199-1207`、`:1329-1332` | ✅ **已修**（commit `cb04d0f1`）：PG 分支把 235 移出 234 失败分支独立执行、去重；SQLite 分支补 235。`go build ./...` + `go test ./internal/core/db/...` 全绿 |
| P0-2 | `CodePermissionDenied=1025` 的 `DefaultHTTPStatus()` 落在 `1020≤c<1030 → 401`；`HandleServiceError` 对裸 error 走 500 ⇒ Phase 132 写「越界 403」时 `apperrors.New(CodePermissionDenied)`→401、`fmt.Errorf`→500 | `pkg/errors/codes.go:22,203-206`、`pkg/response/handler_helpers.go:28-39`、`pkg/errors/codes_80_04_test.go:47` | ✅ **已纳入 Phase 132 规格**：`apperrors.NewWithHTTPStatus(CodePermissionDenied, http.StatusForbidden, "无权操作该数据")` 构造 `ErrScopeDenied`，`errors.Is` 可判定（agent-7 独立发现并确认） |
| P0-3 | 已合入 backfill 与三项裁定正面冲突：候选查询全库扫描无可见集裁剪；`BackfillDeptResult.Samples` 前 20 条逐条回显（违反批量裁定 §2「不逐条回显」）；前端预览/执行两次独立调用无快照；`migration_235` bootstrap 授权给所有持父菜单角色，与独立授权意图相反 | `workstation_service.go:502-648`、`router.go:680-683/836`、`useBackfillDept.tsx` | ✅ **已纳入 Phase 132**：候选/COUNT/CASE UPDATE 三层注入可见集；非 All 档 `Samples` 置空（只留聚合计数）；operlog 补档位/四类计数；前端「本次范围」文案 + 同快照语义 + `BackfillDeptResult` 增字段。migration 235 授权面属产品决策，登记为残余 |

## P1（11 项，摘录与处置）

| # | 发现 | 处置 |
|---|---|---|
| P1-1 | `GET /dept-mapping-template`（`router.go:818` → `excel_handler.go:227-237`）返回全库部门树（部门名/全链路 parent_path/部门代码）零范围过滤，是**第 4 个未点名越权面**；三份裁定豁免清单未点名它 | ✅ 裁定文档已补登记为「不豁免，Phase 131 收口」；已下发 agent-6 |
| P1-2 | `counts-by-floor`(:808)、`GetByID`(:809)、`dept-options`(:805)、floor `tree`(:755) 三份裁定未逐一点名（REQUIREMENTS.md DS-03 有清单） | ✅ 已要求 131 的 403 矩阵逐点断言 |
| P1-3 | 「缓存 key 必含 scope 维度」不可验证：All/Self/FailClosed 三档 DeptIDs 都空，只按 DeptIDs 造 key 会三档碰撞 | ✅ 131 的 `scopeCacheKeyOf` 已按 `all`/`closed`/`self+sha1(UserID)`/`d+sha1(deptIDs)` 分档，审查确认已解决 |
| P1-4 | vdi 旧链 `if userID != "" && dataScope > 0` 让 `dataScope=-1` 完全跳过过滤 = All 放行全库；self §4 迁移未登记此口 | ✅ **已修**（commit `cb04d0f1`）：`vm_service_impl.go:650-655` 改 `dataScope != 0`，异常枚走 `ApplyVMDataScopeFilter` 的 default `1=0`；`go test ./internal/services/vdi/...` 全绿 |
| P1-5 | 批量裁定 §2「聚合计数不逐条回显」不足防枚举（越界行数 / N / 行号都是侧信道） | ✅ 登记为「已知残余风险、接受」，132 SUMMARY 明确不假装已解决 |
| P1-6 | ds05 断链 fail-closed 会让脏数据全不可见；§残余风险只「建议 UAT 加脏数据抽查」，未进消费清单、未设阻断；dedicatedLine `org_id` 直挂列 schema 未核实 | 登记：133 交付时须核实 schema + UAT 加脏数据抽查项 |
| P1-7 | `building_service.go:294-303,386-393` 裸 `org_id IN ?` vs `floor_service.go:293` 用 `effectiveOrgExpr` 两侧 CAST，PG 42883 风险（推定、未实测） | 登记：131 收口时统一 CAST；PG contract 补 building 覆盖 |
| P1-8 | `excel_query_builder.go:239-253` 的 `::uuid/::text` 在 SQLite 必炸，且 `excel_export_chain_77_02_test.go:465-477` 把「报错」文档化为预期 ⇒ 新增导出过滤逻辑在 SQLite 单测里永远测不到 | ✅ **已清偿（2026-11-07 二次修复）**：7 处改 `CAST(... AS TEXT)` 双侧比较（与生产 `workstationJoinClause` 同款）；空串 `''::uuid` 在 PG 报 22P02 的整表导出失败一并消除；新增 `excel_query_builder_p18_test.go` 两用例守护（DryRun 静态断言 + SQLite 语法可执行）；`TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 因夹具缺 ops_floors/ops_buildings 仍报错、断言未反转，注释已同步。独立债（`RightCast` 动态拼接 / 物理链路 PG CTE）按 scope constrainment 登记不动 |
| P1-9 | 双门控 `TEST_PG_DSN` / `XINGRAN_TEST_PG_DSN`，CI 无注入 ⇒ 默认全 Skip 且不可观测 | ✅ **已清偿（2026-11-07 二次修复）**：`excel_dept_cte_postgres_test.go:63` 与 `workstation_position_pg_test.go:80` 统一到 `TEST_PG_DSN`（保留 `.env`/`DB_*` fallback）；`ci.yml` 新增 `PG contract skip observability` step，`-v -run` 把 skip 原因打进 CI 日志 |
| P1-10 | `pg_contract_test.go` 文件头「禁止任何写 SQL」被 1c 用例自我破例，且无零净变更断言；DSN 指向生产库时有落库风险 | 登记：保持事务回滚纪律 + 补零净变更断言 |
| P1-11 | `workstation_handler.go:418-425` dryRun 不记 operlog、param 缺范围档位/请求行数/可见集内/越界行数 | ✅ 已纳入 Phase 132 |

## 裁定自洽性结论

**无直接互相打脸的硬矛盾**，但审查指出 3 处语义冲突 + 5 处交接缺口，已处置：

- 冲突 A：self §2 把 floor/building 等 Self 档定为空集，而批量 §5 要 133 推广「Dept/Self 批量裁剪」⇒ 空集裁剪 = 批量「成功但 0 行」与单条写「越界 403」同实体两种结果 → 133 交付时按「空集裁剪」表达，SUMMARY 登记。
- 冲突 B：批量 §4 只讲 workstation 导入逐行校验；Self 用户导入 building/floor ⇒ 全批不写入但整批「成功」 → 133 登记口径。
- 缺口 C（最严重）：三份裁定全以 `ScopeResult` 为唯一输入，但**无一份点名「ops 组未挂 DataScopePermission」这一 DS-02 前置事实** ⇒ **已由 Phase 130 第二笔补齐（ops 26 组挂载 + `datascope_mount_test.go` 守卫）**，不再是缺口。
- 缺口 D：批量 §5 写「131 已按 DS-06 落地导出不豁免」，时态误导（当时 131 未开工）→ 本清单记录实际时序，131 交付后成立。
- 缺口 E：self §4 的 vdi 迁移未登记 P1-4 的 `>0` 负值放行口 → **已修 + 已登记**。

## 测试质量结论

- diff coverage 76.63% 最危险 5 处缺测：① database.go 迁移注册（已由 P0-1 修复）② `workstation_handler.go:375-425` operlog success path ③ `floor_service.go:562-585` validateOrgOptional ④ `reference_resolver.go:508-556` queryUserDeptMap 分块/错误分支 ⑤ `building_service.go:294-303,386-393` 仅 SQLite 覆盖。
  → 由 agent-9 的 `v152_diff_coverage_test.go` 补，门禁目标 ≥80%。
- `pg_contract_test.go:183-192` 内联复制生产 SQL 违反单一真实源；`requireNoPGErr` 只识别 42883 → 登记。

## operlog 结论

`BatchBackfillDeptFromUser` 符合「success path 末尾、`response.Success` 之前 `operlog.Record`」；body 无敏感字段，`Record` 正确（无需 `RecordWithBody`）。偏差见 P1-11。

## 审查后补充裁定（Lead，2026-11-07）

**dedicatedLine 归属链 schema 冲突**（由 Phase 133 只读勘察发现，agent-10）：ds05 裁定 1 假定 `ops_dedicated_lines` 有 `building_id` 或 `org_id` 列，实测模型（`internal/models/operations/dedicated_line.go:17-44`）只有 `source_room_id`/`dest_room_id` → `ops_server_rooms`，全仓 migrations 无该表机构列。

补充口径（133 直接消费，不重新裁定）：`source_room_id → ops_server_rooms.building_id → ops_buildings.org_id` **OR** `dest_room_id → 同源链`，**任一端归属链命中可见集即放行**；两端都不命中 → 子查询 0 行 → 空集，仍 fail-closed。写端点同理（Create/Update 提交的 source/dest 至少一端落可见集，否则 403）。不新增迁移、不加列。理由：专线是「两端」实体，只看源端会误拒正常跨机构专线；任一端命中不扩大越权面。登记入 `133-SUMMARY.md` 的「裁定补充登记」章节。

## 一句话总评

三份裁定文字上以 fail-closed 自洽，但集体漏掉「ops 组未注入 data_scope」与「越界 403 被框架退化成 401/500」两个落地前提 —— 二者已分别由 Phase 130 第二笔与 Phase 132 规格补齐；PR#75 的「全库写 + 全库明细回显 + 权限默认广播 + 迁移 235 永不执行」四重叠加，已通过 P0-1/P0-3 与 Phase 132 收口化解。
