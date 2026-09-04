---
phase: 91-crud-base-repository-t-p1
verified: 2026-09-04T21:05:00Z
status: gaps_found
score: 8/9 must-haves verified
overrides_applied: 0
overrides: []
gaps:
  - truth: "LOC 净减 ≥800（D-07 混合标准第三成分）"
    status: partial
    reason: "git numstat 独立复核：全口径净减 -8（del 1555 / add 1563），仅生产代码口径 +408（del 1182 / add 774），均未达 ≥800。混合标准另两成分（11/11 服务全部复用 GORMRepository + 每服务 CRUD 模板清零）已实证达成；SUMMARY 如实记录未达标与归因（测试基线计划内新增 +678 行：floor_service_crud_test +288 / base service_test +275 / fpt_crud_test +115；条件逐字平移与行为等价论证注释），无虚报。91-04-PLAN Task 4 验收条款预授权「如实记录未达标差距」为合法完成路径。关键分析：F5 组合 2（building/floor typed 接线，估 ~80-120 行）即使实施也无法闭合数字缺口（差距 808 行）——本 gap 只能经用户决策（override 接受偏差或重校准锚点）解决，不可能靠继续删除解决。"
    artifacts:
      - path: ".planning/workstreams/milestone/phases/91-crud-base-repository-t-p1/91-04-SUMMARY.md"
        issue: "LOC 审计表如实记录全口径 -8 / 生产口径 +408，D-07 ≥800 数字成分未达成"
      - path: "internal/services/operations/（11 个 service + base/service.go + pagination_helper.go）"
        issue: "无问题——定性成分（模板清零 + 全量复用）已达成；缺口纯属量化锚点未中"
    missing:
      - "用户决策：接受 D-07 偏差（按下方 override 建议落 frontmatter），或明确重校准量化锚点（如改为「生产代码净减 ≥400 且模板清零」）"
      - "（可选、非闭合手段）F5 组合 2：building/floor typed request 接线扩展 ~80-120 行——仅作为进一步消除重复的候选项，不作为本 gap 的闭合路径"
deferred: []
human_verification:
  - test: "真实前端+后端联调 smoke：登录 → 工位列表分页翻页 → 楼宇/资产列表翻页"
    expected: "工位列表 pageSize 上限 100（A2 收紧后）、current<1 回退 1；楼宇/资产 Total 与列表行数一致（软删行不再计入，F3 变更后行为）；无白屏/报错"
    why_human: "自动化覆盖为 handler 测试 + service sqlite 测试两层拼合；真实浏览器联调与分页组件交互无法编程验证"
  - test: "检查前端各列表页发送的 status/type 字段值形态（是否存在字符串 \"1\"、浮点 1.5 等）"
    expected: "均为整数或省略——typed bind 后畸形值会整体降级跳过全部过滤（REVIEW IN-03），合法 payload 零差异"
    why_human: "需审计前端实际 payload 形态与后端行为变化的交互面，grep 前端代码无法穷尽运行时形态"
---

# Phase 91: CRUD 复用 base.Repository[T] 验证报告

**Phase Goal:** 让 internal/services/operations/ 下 11 个 CRUD services 复用 base.GORMRepository[T] 泛型抽象（gorm scope 函数式），CRUD 模板清零；混合 LOC 标准（净减 ≥800）；go test ./internal/services/operations/... 0 失败；handler 端到端 smoke 通过；前端 JSON 契约零变更。（ROADMAP 原文按 91-CONTEXT D-04/D-06/D-07 修订）
**Verified:** 2026-09-04T21:05:00Z
**Status:** gaps_found（唯一 gap：D-07 LOC ≥800 数字成分未达，定性成分全达成，详见 override 建议）
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | base.GORMRepository[T] 为 gorm scope 函数式完整抽象：六方法 + SortScope/SortScopeWithTail；无 Repository[T] interface、无 Query/WhereCondition DSL；Statistics/SearchOptions 留 service（D-01/D-02/D-04） | ✓ VERIFIED | service.go:15 Scope alias / :22 PageParams / :70 GetByID(变长 scopes) / :112 List(page, scopes...) / :143 BatchDelete / :159 SortScope / :175 SortScopeWithTail；`type Repository[` 与 `type Query struct\|type WhereCondition struct` 计数均为 0；workstation Statistics(:75)/DeptOptions(:97)/BatchUpdatePositions(:330) 原签名保留；SortScope 内部逐字复用 ApplySort 白名单（不重写） |
| 2 | 11/11 operations services 全部复用 GORMRepository，CRUD 模板清零（D-06/D-07 定性底线） | ✓ VERIFIED | 11 文件逐一 grep：countRecords/fetchRecords/buildListQueryFromRequest **定义与引用均计数 0**；`base.GORMRepository[` 组合 11/11 存在（workstation/building/floor/asset/server_room/infopoint/dedicated_line/room_device/door/wall/floor_plan_text）；s.repo./listRepo(). 委托调用每文件 2-6 处 |
| 3 | LOC 净减 ≥800（D-07 量化锚点） | ✗ FAILED | 独立重算 `git diff --numstat 0ab266e..HEAD -- internal/services/ internal/api/v1/operations/`：全口径 add=1563 del=1555 **净减 -8**；生产代码（排除 _test.go）add=774 del=1182 **净减 +408**；测试代码净 -416。与 91-04-SUMMARY 数字逐项吻合（诚实回退，无虚报）。未达 ≥800 |
| 4 | go test ./internal/services/operations/... 0 失败 | ✓ VERIFIED | 本次实跑 `go test -count=1 ./internal/services/operations/` → ok (2.352s)；68 个 phase 行为锁（TestImp77/TestFloor91/TestFloorPlanText91/TestDoor/TestWall/TestServerRoom/TestDedicatedLine/TestRoomDevice/TestInfoPoint/TestAsset/TestBuilding）单独过滤全 PASS |
| 5 | handler 端到端 smoke 通过（handler 测试口径） | ✓ VERIFIED | `go test -count=1 ./internal/api/v1/operations/` → ok (0.442s)；workstation_handler List/SearchOptions typed bind + 降级路径（handler :95-97/:147-149）在测 |
| 6 | 前端 JSON 契约零变更 | ✓ VERIFIED | base.PageResult JSON tags（list/total/current/pageSize）与迁移前 operations 定义逐字一致；operations 侧 `type PageResult = base.PageResult` alias（计数 1）；WorkstationListRequest 仅加性 +FloorCode/Type(*int)/OrgID 三可选字段，buildingId/code 静默忽略语义保留 |
| 7 | 行为锁保留：BatchDelete 空 ids → nil（P1）、-1 跳过（P6）、floor 签名/装饰器零触碰（P7）、room_device joinScope 先于 filterScope（P8） | ✓ VERIFIED | service.go:143-148 空 ids 返回 nil；workstation GetStatus(-1) 双守卫（:277/:453）；floor_cache_impl.go 自基线 0ab266e `git diff` 为空，floor 经 listRepo()(:82) nil-safe 访问器委托 GetByID(:194)/List(:251)；room_device List :169-173 实参顺序 joinScope() → filterScope(req) → SortScope |
| 8 | typesafe 死代码清理 + pagination_helper 修剪（91-04 must_have） | ✓ VERIFIED | building_service_typesafe.go / building_handler_typesafe.go / building_handler_typesafe_test.go 均 No such file；TestBuildingService_TypeSafe 计数 0；calculateOffset 计数 0；extractPagination/extractSortRequest/extractIntParam/extractStringParam/clampPageSize 全部保留（F1 口径：map 服务 List + SearchXxxOptions + Statistics + location_alias 仍消费） |
| 9 | REQUIREMENTS.md § CRUD-REUSE-01..08 措辞同步（D-04/D-06/D-07）且 checkbox 与实际一致 | ✓ VERIFIED | :59 已改 scope 化口径（「补全缺失方法」计数 0）；:64 含 wall_service.go + floor_plan_text_service.go +「（D-06 扩容后剩 7 个）」；:66 含「净减 ≥800」且「target ~2000」计数 0；8 个 checkbox [x]——其中 CRUD-REUSE-08 为 PARTIAL（见 Requirements Coverage） |

**Score:** 8/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/services/base/service.go` | scope 版六方法 + SortScope/SortScopeWithTail | ✓ VERIFIED | 全部签名就位，DSL/interface 清零，198 行 |
| `internal/services/base/service_test.go` | TestBase91_* 契约锁值 | ✓ VERIFIED | 6 个 TestBase91 全 PASS（值切片/JoinsSelect Count/复合排序/BatchDelete nil/表限定 GetByID/双型排序 SQL 断言） |
| `internal/services/base/base_80_05_test.go` | DSL 用例改写 + BatchDelete 期望反转 | ✓ VERIFIED | 4 个 TestBas8005 全 PASS，无 &Query{/WhereCondition{ 残留 |
| `internal/services/operations/pagination_helper.go` | PageResult alias + calculateOffset 修剪 + 五 helper 保留 | ✓ VERIFIED | alias 计数 1；calculateOffset 0；五 helper 各 ≥1 |
| `internal/services/operations/{11 个 service}.go` | repo 组合 + scope 化 + 模板清零 | ✓ VERIFIED | 见 Truths #2/#7；A 型 door/wall SortScopeWithTail(:95/:88)、infopoint WorkID/PointType 兼容分支(:167-182)、server_room orgId 空命中早退(:153-164)逐字保留 |
| `internal/services/operations/floor_service_crud_test.go` | floor 行为基线 6 锁（Wave 0 缺口补齐） | ✓ VERIFIED | 6 个 TestFloor91 全 PASS |
| `internal/services/operations/floor_plan_text_crud_test.go` | fpt 冒烟（TestFloorPlanText91 前缀） | ✓ VERIFIED | 存在（4489 字节），PASS（含 CreatedAt 错开防平局修复） |
| `internal/api/v1/operations/requests/workstation_requests.go` | WorkstationListRequest +3 字段 | ✓ VERIFIED | FloorCode(:17)/Type *int(:19)/OrgID(:20)，加性 JSON 注释(:5) |
| typesafe 三文件 | 整文件删除 | ✓ VERIFIED | ls 报 No such file（删除确认） |
| `.planning/REQUIREMENTS.md` | CRUD-REUSE 措辞同步 | ✓ VERIFIED | 见 Truth #9 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| base/service.go SortScope/SortScopeWithTail | base/list_request.go ApplySort/ResolveSort | 内部复用白名单 | ✓ WIRED | 两 helper 闭包首行均 `db = ApplySort(db, req, allowed)`，无重写 |
| operations/pagination_helper.go | base/service.go | type alias | ✓ WIRED | `type PageResult = base.PageResult` 恰 1 处 |
| workstation_service.go | base/service.go | s.repo.(六方法) | ✓ WIRED | 6 处 s.repo. 调用；接口 List/SearchOptions 已 typed(:46/:56) |
| workstation_handler.go | requests/workstation_requests.go | ShouldBindJSON typed + 降级 | ✓ WIRED | :95-97/:147-149 两处 bind 失败降级零值请求 |
| building/asset_service.go | base/service.go | s.repo.List（map 签名不变） | ✓ WIRED | building:139 / asset:188；List(map) 签名 interface+impl 各 1（共 2）迁移前后一致 |
| floor_service.go | base/service.go | listRepo() nil-safe 访问器 | ✓ WIRED | :194 GetByID / :251 List；P7 装饰器零触碰（floor_cache_impl.go diff 为空） |
| 7 个 typed service | base/service.go | s.repo.(List/Create/Update/Delete/GetByID/BatchDelete) | ✓ WIRED | door/wall/server_room/dedicated_line/fpt/room_device/infopoint 每文件 5-6 处 |
| 91-04-SUMMARY LOC 审计 | git history | git diff --numstat | ✓ WIRED | 验证器独立重算，全口径/生产/测试三分项数字与 SUMMARY 逐项一致 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| GORMRepository.List | []T 值切片 | sqlite 真库（测试）/ PostgreSQL（生产） | Yes | ✓ FLOWING（TestBase91_List_ValueSlice + 11 服务 sqlite 测试打真库断言 Total/List 回填） |
| PageResult alias | list/total/current/pageSize | repo.List 回填 | Yes | ✓ FLOWING（JSON tags 与 handler 透传路径在 handler 测试覆盖） |

（本 phase 为 service 层重构，无 UI 渲染型 artifact；Level 4 以「DB → repo → service → handler」数据链实证替代，全部经 sqlite 真库测试采样。）

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| 全库编译 | `go build ./...` | exit 0 | ✓ PASS |
| base 契约 + 适配 | `go test -count=1 ./internal/services/base/` | ok 0.463s（6 TestBase91 + 4 TestBas8005 全 PASS） | ✓ PASS |
| operations 全包 | `go test -count=1 ./internal/services/operations/` | ok 2.352s | ✓ PASS |
| handler smoke | `go test -count=1 ./internal/api/v1/operations/` | ok 0.442s | ✓ PASS |
| AST 防线 | `go test ./internal/models/ ./internal/utils/operlog/ ./pkg/constants/` | 全 ok（status 常量锁值 / operlog 回归 / pagination+timeouts 锁值） | ✓ PASS |
| phase 行为锁采样 | `-run "TestImp77\|TestFloor91\|TestFloorPlanText91\|TestDoor\|..."` | 68 个 --- PASS / 0 FAIL | ✓ PASS |
| LOC 审计复核 | `git diff --numstat 0ab266e..HEAD ...` | -8 全口径 / +408 生产口径（与 SUMMARY 一致） | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| （无声明 probe） | `find scripts -path '*/tests/probe-*.sh'` | 无匹配文件 | ? SKIP（PLAN/SUMMARY 未声明 probe；handler 测试套件为 RESEARCH 约定的 e2e smoke 口径，已实跑通过） |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CRUD-REUSE-01 | 91-01 | base/service.go 改造为 scope 函数式 GORMRepository（删 interface/DSL，BatchDelete 语义反转；Statistics/SearchOptions 留 service） | ✓ SATISFIED | Truth #1 |
| CRUD-REUSE-02 | 91-02 | workstation pilot 迁移 | ✓ SATISFIED | Truth #2/#7；D-05 typed 接线（接口+handler+3 测试文件） |
| CRUD-REUSE-03 | 91-03 | building 迁移 | ✓ SATISFIED | repo 组合(:57) + List scope 化(:139)，map 签名不变 |
| CRUD-REUSE-04 | 91-03 | floor 迁移 | ✓ SATISFIED | GetByID/List 管道进 repo（listRepo :194/:251），P7 签名零触碰 |
| CRUD-REUSE-05 | 91-03 | asset 迁移 | ✓ SATISFIED | repo 组合(:48) + List scope 化(:188)，component_type IS NULL 恒定过滤保留 |
| CRUD-REUSE-06 | 91-04 | server_room/infopoint/dedicated_line/room_device/door/wall/floor_plan_text 批量迁移（D-06 剩 7 个） | ✓ SATISFIED | 7 文件全部 repo 化，模板清零（Truth #2） |
| CRUD-REUSE-07 | 91-01/03/04 | 迁移后 operations 测试全过 + base/service_test.go 锁泛型契约 | ✓ SATISFIED | operations 包 ok；6 TestBase91 全绿 |
| CRUD-REUSE-08 | 91-04 | LOC 净减审计（混合标准：全部复用 + 模板清零 + ≥800 行，numstat 入 SUMMARY）+ handler 0 回归 | ⚠️ PARTIAL | 复用/模板清零/numstat 审计入 SUMMARY/handler 0 回归均达成；**≥800 数字未达（-8 全口径 / +408 生产口径）**——即本报告唯一 gap |

**Orphaned requirements check:** REQUIREMENTS.md 将 Phase 91 映射到 CRUD-REUSE-01..08；四个 PLAN 的 requirements 字段并集恰为 01..08（01: 01,07 / 02: 02 / 03: 03,04,05,07 / 04: 06,07,08）。无 orphan。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| （phase 触碰的 15 个核心文件） | — | TBD/FIXME/XXX/HACK/PLACEHOLDER/占位实现 | — | 全部计数 0，无 debt marker |
| floor_service.go:299-310 | GetTree LEFT JOIN 未过滤软删楼层 | ⚠️ Warning（迁移前既有，REVIEW WR-02，非本期回归） | 后续硬化项，Phase 92+ 收敛 |
| 8 个 service Update | created_at/created_by 全量覆盖类缺陷未收敛 | ⚠️ Warning（迁移前既有，REVIEW WR-01；3/11 已受保护） | 后续硬化项，有现成修复范式 |
| requests/building_requests.go、workstation_requests.go | 孤儿请求 struct（IN-01/IN-02） | ℹ️ Info | 零消费者死代码，SUMMARY 已列 future cleanup |

### Human Verification Required

1. **前后端联调 smoke（分页/Total 行为变更面）** — 见 frontmatter `human_verification[0]`。两个 checkpoint 批准的行为变更（A2 workstation 分页收紧、F3 building/asset 软删 Total）已在代码注释（building_service.go:132-135、asset_service.go:183）与 SUMMARY 文档化并有测试守护，但真实浏览器联调只能人工执行。
2. **前端 payload 形态审计（IN-03 健壮性面）** — 见 frontmatter `human_verification[1]`。

### 两个 Checkpoint 行为变更文档化核验（验证焦点 #6）

| 变更 | 批准方式 | 代码证据 | 文档证据 | 状态 |
| ---- | -------- | -------- | -------- | ---- |
| A2: workstation 分页收紧（pageSize 上限 10000→100、current<1 回退 1） | 91-02 Task 3 checkpoint auto-approved | workstation_service.go:314 `req.GetPagination()` | 91-02-SUMMARY checkpoint 节 + List 注释 | ✓ 已文档化 |
| F3/A3: building/asset 软删 Total 收紧 | 91-03 Task 4 checkpoint auto-approved | building_service.go:132-135 / asset_service.go:183 注释 | 91-03-SUMMARY checkpoint 节 | ✓ 已文档化 |

### Gaps Summary

**唯一 gap：D-07 的 LOC ≥800 数字成分未达成（truth #3，FAILED→gaps_found）。** 混合标准的另两成分（11/11 全部复用 + 每服务 CRUD 模板清零）经独立 grep/测试实证达成；SUMMARY 如实记录未达标、归因（计划内测试基线 +678 行 + 条件逐字平移 + 行为等价论证注释）与备选（F5 组合 2），验证器独立重算 numstat 与 SUMMARY 逐字吻合——**诚实回退属实，无虚报**。

**本 gap 的特殊性：它不可通过继续施工闭合。** 91-04-PLAN Task 4 验收条款预授权「如实记录未达标差距」为合法完成路径；F5 组合 2（~80-120 行）即使实施，距 808 行缺口仍相差一个量级——继续删除只会触发 plan 明令禁止的「凑数删除」。因此本 gap 的唯一理性出口是**用户决策**：

1. **接受偏差（推荐路径）**——在 VERIFICATION.md frontmatter 落 override（D-07 定性意图已达成，量化锚点系 CONTEXT 时点估算误差）：
   ```yaml
   overrides:
     - must_have: "LOC 净减 ≥800（D-07 混合标准第三成分）"
       reason: "定性成分（11/11 复用 + 模板清零）达成；生产代码净减 +408 与 RESEARCH 严格范围预估（450-550）吻合；测试基线 +678 为计划内 Wave 0 缺口补齐；F5 组合 2 无法闭合数字缺口，继续删除违反「禁止凑数删除」约束"
       accepted_by: "<用户>"
       accepted_at: "<ISO 时间>"
   ```
2. **或重校准锚点**——修订 REQUIREMENTS CRUD-REUSE-08 措辞（如「生产代码净减 ≥400 且模板清零」），沿本期已建立的 REQUIREMENTS 同步先例（commit 3a2efe5 / 963defe）。

其余全部验证项（9 项 truth 中 8 项、全部 artifact、全部 key link、全部行为锁、AST 防线、两个 checkpoint 文档化、REQUIREMENTS 同步）均 VERIFIED。gap 不构成对 Phase 92（缓存层统一）的工程阻塞，但按验证框架规则（存在 FAILED truth）status 记为 gaps_found，交由 orchestrator/用户裁决。

---

_Verified: 2026-09-04T21:05:00Z_
_Verifier: Claude (gsd-verifier)_
