---
phase: 99-operations口径统一
verified: 2026-09-07T02:20:00+08:00
status: gaps_found
score: 4/6 must-haves verified
overrides_applied: 0
re_verification: null
gaps:
  - truth: "orgId 筛选统一：「部门+全部子部门」抽共享 helper 统一四条件口径，6+ 处复制粘贴收敛（V130R-08）"
    status: partial
    reason: "缺陷修复部分完成——8 处内联实现已全部改为四条件形式（中段漏匹配 bug 已修），共享 helper 已建并有 6 个回归测试；但收敛到 helper 的调用方数为 0，PLAN 99-03 Step 3 承诺的「替换四服务中的重复实现」未执行，helper 为 ORPHANED 状态（后续逻辑变更仍需改 8 处）"
    artifacts:
      - path: "internal/services/operations/dept_filter.go"
        issue: "BuildDeptRecursiveFilter 全仓零生产调用方（仅 dept_filter_test.go 引用），ORPHANED"
      - path: "internal/services/operations/workstation_service.go"
        issue: "4 处内联 orgId 筛选（:82/:120/:296/:481）未迁移到共享 helper（四条件形式已正确）"
      - path: "internal/services/operations/infopoint_service.go"
        issue: "1 处内联 orgId 筛选（:198）未迁移到共享 helper（四条件形式已正确）"
      - path: "internal/services/operations/building_service.go"
        issue: "2 处内联 orgId 筛选（:193/:275）未迁移到共享 helper（四条件形式已正确）"
      - path: "internal/services/operations/asset_service.go"
        issue: "1 处内联 orgId 筛选（:276）未迁移到共享 helper（四条件形式已正确）"
    missing:
      - "将 workstation/infopoint/building/asset 共 8 处内联「部门+全部子部门」筛选替换为 BuildDeptRecursiveFilter 调用（helper 现有签名 idColumn 参数可直接承载 b.org_id 等带限定名的列）"
  - truth: "internal/constants 与 pkg/constants 双包合并方案经 discuss 敲定落地（V130R-09 子条款）"
    status: failed
    reason: "99-CONTEXT.md 的 D-03 决策清单（D-03-1..9）无任何双包合并决策（CONTEXT 明言「本次讨论仅针对 Plan 99-04 分页口径收敛」）；internal/constants 仍为独立包、25 个生产文件 import；99-04-PLAN:143 明示 internal/constants import「保留不动」——与 ROADMAP「双包合并方案经 discuss 敲定落地」直接矛盾"
    artifacts:
      - path: "internal/constants/pagination.go"
        issue: "MaxListPageSize=100 / MaxOptionsPageSize=10000 仍留在 internal/constants，未并入 pkg/constants"
    missing:
      - "discuss 补充双包合并决策（或将该子条款显式降级/移除），随后落地常量合并"
  - truth: "base/service.go 注释自认刻意保留处一并处置（V130R-09 子条款）"
    status: failed
    reason: "internal/services/base/service.go:19-20 与 :106 注释仍声称「项目现存三种 clamp 语义（10..100 / 10..10000 / 无 clamp）并存」——99-04 已删除 10..10000 口径（clampPageSize/extractPagination），注释已过时且未按 ROADMAP 要求一并处置；base/service.go 不在任何 99-04/99-05 计划的 files_modified 清单中"
    artifacts:
      - path: "internal/services/base/service.go"
        issue: ":19-20 与 :106 的「三种 clamp 并存」注释在收敛完成后已成过时描述，未更新"
    missing:
      - "更新 base/service.go 两处注释以反映收敛后现实（repo 不 clamp 设计不变，service 层口径已统一走 pkg/query）"
deferred:
  - truth: "OVR 台账补记（V130R-06/V130R-07 各要求一条）"
    addressed_in: "Phase 101"
    evidence: "Phase 101 goal 明确含「七 gate 全绿 + v1.30 audit 落盘」；v1.29-REQUIREMENTS.md:157-158 将 V130R-06/07 的台账补记列为 milestone 级 manual-only 项，audit 收口为既定归宿"
human_verification:
  - test: "CAD 平面图/3D 楼层视图真实渲染验证（floors 页平面图编辑器、building-spaces-3d 的 FloorView3D/BuildingView3D、building-spaces WorkstationView）"
    expected: "切换楼层时工位全集经 GET /ops/workstation/:floorId/workstations-all 加载并在画布/3D 场景中完整显示（>200 工位楼层不截断）"
    why_human: "vitest 仅验证数据获取逻辑（mock 层），Three.js 场景渲染与 UI 交互无法程序化断言"
  - test: "真实并发换楼冲突场景：两个管理员同时对同一楼层发起换楼"
    expected: "后到请求收到乐观锁冲突错误（floor was modified by another request）而非静默成功；工位 building_id 最终收敛到胜出请求的目标楼"
    why_human: "sqlite 顺序回归测试只能模拟过期读取，真实数据库并发时序无法程序化复现"
---

# Phase 99: operations 口径统一 验证报告

**Phase Goal:** operations 域四处口径/语义缺陷统一——List Total 软删过滤、换楼同步乱序、orgId 子部门筛选共享 helper、分页 clamp 三口径收敛（V130R-06..09）
**Verified:** 2026-09-07T02:20:00+08:00
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|------|--------|----------|
| 1 | Total 口径收紧：asset/building List Total 不计软删（V130R-06） | ✓ VERIFIED | building_service.go:145 / asset_service.go:197 均走 `s.repo.List` → base/service.go:113 `Model(new(T))` 起链 → :122 Count 含 `deleted_at IS NULL`；两文件 List 路径无 `.Table()` 残留（余下 `.Table("ops_asset")` 均在 validateDeviceSNUnique/GetDeviceTypes 等非 Total 路径）；回归测试 TestBuildingList_TotalExcludesSoftDeleted 断言软删后 Total=2（PASS） |
| 2 | 换楼同步有序：乐观锁 + 过期读取不再静默成功（V130R-07） | ✓ VERIFIED | floor_service.go:22-23 `ErrFloorNotInExpectedBuilding` 哨兵；:169-170 乐观条件 `WHERE id = ? AND building_id = ?`（旧值）；:184-185 RowsAffected==0 → 返回冲突错误；4 个回归测试全 PASS（含 StaleReadReturnsError 锁定「过期 First 读取不再静默吞掉」语义）。⚠ 残留见下方「注意事项」 |
| 3 | orgId 筛选统一：抽共享 helper + 6+ 处复制粘贴收敛 + 中段漏匹配修复（V130R-08） | ✗ FAILED（部分） | 修复半：8 处内联全部为四条件形式（workstation_service.go:82/:296/:481 + :120 wrap 形式、infopoint_service.go:198、building_service.go:193/:275、asset_service.go:276），dept_filter_test.go 6 测试 PASS（含 MiddleOfHierarchy/BrokenThreeCondition）；收敛半：`BuildDeptRecursiveFilter`（dept_filter.go:21）**全仓零生产调用方**，PLAN 99-03 Step 3「替换四服务中的重复实现」未执行 |
| 4 | 分页 clamp 收敛单一权威 + CAD 端点 + 前端迁移（V130R-09） | ✗ FAILED（部分） | 达成半：pkg/query/pagination.go:43-49 `NormalizePagination` 单表达式委托 :51 `NormalizePaginationWithMax`；pagination_helper.go 死函数已删（全仓 grep extractPagination/clampPageSize 仅 3 处测试注释，零代码命中）；workstation List cap=200 无 10000（workstation_service.go:317）；location_alias:100 / building:142 / asset:191 经 `NormalizePaginationWithMax(MaxOptionsPageSize)`；vdi ListServers（vdi_server_service_impl.go:71）经 `MaxListPageSize`；CAD 端点 router.go:638 注册 + opsApi.ts:102 对齐 + 5 个 CAD/3D 消费者全部迁 `getFloorWorkstationsAll`（workstationApi.list pageSize>=200 生产调用归零；BuildingView3D.tsx:94 为 pageSize:1 计数用途，PLAN 排除清单内）。未达成半：双包合并未 discuss 未落地、base/service.go 过时注释未处置（见 gaps） |
| 5 | 回归纪律：每项修复附回归测试 | ✓ VERIFIED | 99-01 building_total_softdelete_test.go（2 测试）；99-02 floor_move_building_test.go（4 测试）；99-03 dept_filter_test.go（6 测试）；99-04 pkg/query/pagination_99_04_test.go（TestNormalizePaginationWithMax + TestNormalizePagination_Delegates PASS）；99-05 workstation_handler_full_test.go（13 处 workstations-all 引用）+ 7 个前端测试文件（8 文件实跑 83 用例全 PASS） |
| 6 | Gate：go build ./... 0 错误 + 相关包 go test 0 失败 | ✓ VERIFIED | `go build ./...` exit 0；`go test ./pkg/query/... ./internal/services/operations/... ./internal/api/v1/operations/... ./internal/services/vdi/...` 5 包全 ok（operations 14.9s 实跑，余 cached） |

**Score:** 4/6 truths verified（SC3/SC4 主缺陷已修但结构性子条款失败）

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | OVR 台账补记（V130R-06/07） | Phase 101 | Phase 101 goal「v1.30 audit 落盘」；v1.29-REQUIREMENTS.md:157-158 列为 milestone 级 manual-only 项 |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/services/operations/dept_filter.go` | 共享 orgId 筛选 helper | ⚠️ ORPHANED | 存在、实质（四条件）、有测试，但零生产调用方 |
| `pkg/query/pagination.go` | NormalizePaginationWithMax 权威 | ✓ VERIFIED | :51 实现 + :43 单行委托 + 表驱动回归测试 |
| `internal/services/operations/pagination_helper.go` | 死函数删除 | ✓ VERIFIED | 仅存 PageResult 别名 + extractSortRequest/Int/StringParam 活函数 |
| `internal/api/router.go` CAD 端点 | workstations-all 注册 | ✓ VERIFIED | router.go:638 → workstationHandler.GetFloorWorkstationsAll |
| `xingran-react-frontend/src/lib/opsApi.ts` | 新端点方法 | ✓ VERIFIED | :102 `getFloorWorkstationsAll` |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| building/asset List | base.GORMRepository.List | s.repo.List | WIRED | building_service.go:145 / asset_service.go:197 → base/service.go:112 |
| workstation List | pkg/query 权威出口 | query.NormalizePagination | WIRED | workstation_service.go:317 |
| router.go | workstationHandler | GET :floorId/workstations-all | WIRED | router.go:638 + handler :170 swagger 注解 + handler 测试 13 处 |
| 5 个 CAD/3D 消费者 | opsApi.getFloorWorkstationsAll | 前端调用 | WIRED | useFloorPlanEditor.ts:60 / FloorView3D.tsx:68 / BuildingView3D.tsx:157 / WorkstationView.tsx:30 / useWorkstationView.ts:46 |
| **8 处内联 orgId 筛选** | **BuildDeptRecursiveFilter** | **应有调用** | **NOT_WIRED** | 全仓 grep 零生产命中——SC3 收敛断链 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Phase 99 后端回归测试 | `go test -v -run "TestBuildingList_TotalExcludesSoftDeleted\|TestFloorMoveBuilding\|TestBuildDeptRecursiveFilter\|TestWorkstationOrgFilter" ./internal/services/operations/` | 11/11 PASS | ✓ PASS |
| 分页权威委托测试 | `go test -v -run TestNormalizePagination ./pkg/query/` | 2/2 PASS | ✓ PASS |
| 99-05 前端测试（8 文件） | `npx vitest run <8 test files>` | 83/83 PASS | ✓ PASS |
| 相关包测试 gate | `go test`（4 组包） | 5 包全 ok | ✓ PASS |

### Probe Execution

SKIPPED —— 本 phase 无声明/惯例 probe 脚本（find scripts 无 probe-*.sh；fix/refactor 类 phase）。

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| V130R-06 | 99-01 | Total 软删口径收紧 + 软删环境分页器验证 + OVR 补记 | ✓ SATISFIED（OVR 补记 defer→101） | repo Model 起链 + TestBuildingList_TotalExcludesSoftDeleted（软删环境下 Total/分页正确） |
| V130R-07 | 99-02 | 换楼乐观锁 + 过期读取报错 + OVR 补记 | ✓ SATISFIED（OVR 补记 defer→101） | 乐观锁 UPDATE + 哨兵错误 + 4 测试 |
| V130R-08 | 99-03 | 抽共享 helper 统一四条件 + 6+ 处收敛 + 中段漏匹配修复 | ◐ PARTIAL | 四条件修复 ✓ 8 处 + 6 测试；helper 收敛 ✗ 0 调用方 |
| V130R-09 | 99-04/99-05 | 分页单一权威收敛 + CAD 端点 + 前端迁移 + 双包合并 + base 注释处置 | ◐ PARTIAL | 权威出口/迁移/端点/消费者 ✓；双包合并 ✗ 未 discuss 未落地；base/service.go 注释 ✗ 未处置 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| internal/services/operations/dept_filter.go | :21 | ORPHANED（导出 helper 零生产调用方） | ⚠️ Warning | SC3 收敛目标未达成；匹配逻辑后续变更仍需同步 8 处 |
| internal/services/base/service.go | :19-20, :106 | 过时注释（声称「三种 clamp 并存」，实际 10..10000 口径已删） | ⚠️ Warning | 误导后续读者，ROADMAP 明列应处置 |
| phase 修改的 8 个生产文件 | — | TODO/FIXME/TBD/XXX/HACK 扫描 | — | 零命中 ✓ |

### 注意事项（不构成 gap 的残留）

1. **floor_service.go:156-159 First 错误仍静默吞掉（字面义）**：`First(&prev)` 的 err 非 nil 时 oldBuildingID 保持空串、走 Save 分支，First 的 DB 错误不向上传播。discuss 敲定的 V130R-07-1 范围仅为乐观锁（CONTEXT.md:42），且「过期读取 → RowsAffected==0 → 报错」已被 TestFloorMoveBuilding_StaleReadReturnsError 锁定，故 SC2 判 VERIFIED；但若要求字面义「First 失败不再静默吞掉」，需补 err 传播（建议随后续 phase 顺手处理）。
2. **99-04-PLAN:50 明示 requests.PaginationParams.GetPagination/GetPaginationWithMax（system 域）按 CONTEXT 决策豁免**——其独立 clamp 保留不算 SC4 缺口（discuss 授权范围内）。
3. **deferred-items.md 已记录 3 个 full-suite 失败**（network/knowledge/workorder，均在 99-04 前已存在、归属 Phase 97/98 在途工作，干净 worktree 复现证据在案）——本次 gate 按 orchestrator 指定的 4 组相关包运行，全部绿灯，不受影响。
4. **99-02 代码借道 9b861c8 提交入库**（提交信息为 docs(99-03) 但 diff 含 floor 乐观锁 + 测试），账实相符、不影响验证结论。

### Human Verification Required

1. **CAD/3D 楼层视图真实渲染**：切换楼层时工位全集经新端点加载并在平面图/Three.js 场景完整显示（>200 工位不截断）——vitest 仅覆盖 mock 层数据逻辑。
2. **真实并发换楼冲突**：双管理员并发换同一楼层，后到请求应收到 ErrFloorNotInExpectedBuilding 映射的错误响应——sqlite 顺序测试无法复现真实并发时序。

### Gaps Summary

Phase 99 的四项**用户可见缺陷修复全部落地且测试锁定**（软删 Total、换楼乱序、ancestors 中段漏匹配、分页 clamp 多口径），gate 全绿。失败的是三项**结构性子条款**：

1. **V130R-08 收敛断链（最主要缺口）**：`BuildDeptRecursiveFilter` 已建成并有完整测试，但 8 处内联实现一处都没迁——PLAN 99-03 Step 3 明确承诺「替换四服务中的重复实现」却未执行，helper 成 ORPHANED。四条件修复让 8 处口径当前语义一致，但「抽共享 helper 统一口径」的防再发散目的落空。
2. **V130R-09 双包合并缺位**：ROADMAP 要求「internal/constants 与 pkg/constants 双包合并方案经 discuss 敲定落地」，实际 99-CONTEXT 的 D-03 决策不含此项、internal/constants 仍有 25 个生产 import 方，99-04-PLAN 反而明示保留 internal/constants import。
3. **base/service.go 过时注释未处置**：ROADMAP 明列「一并处置」，实际注释原样保留且内容已因收敛而过时。

三项缺口互不阻塞、彼此独立，可用一个 gap-closure plan 收口（迁移 8 处调用 + 常量合并决策/落地或显式降级该子条款 + 两处注释更新）。

---

_Verified: 2026-09-07T02:20:00+08:00_
_Verifier: Claude (gsd-verifier)_
