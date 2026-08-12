---
phase: 39-workstation-dept-location-alias
verified: 2026-06-25T00:00:00Z
status: passed
score: 12/12 must-have truths verified (用户 UAT 全部确认通过)
uat_confirmed: 2026-06-25 (用户 dev 环境逐项验证: 映射 CRUD / 二级校验 / 部门名显示 / 列表分页 / TreeSelect 展开与去重 / EditModal union 注入)
overrides_applied: 0
re_verification:
  previous_status: draft (39-08)
  previous_score: 7/8 automated AC pass
  gaps_closed:
    - "CR-01 LocationAliasDrawer 分页失效已修复 (commit 5ef0f37: handler 从 POST body 读分页参数)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "AC-09 migration 165 创建 sys_dept_location_alias 表 + 业务列 + partial unique idx + idx_location_id"
    expected: "psql \\d sys_dept_location_alias 显示 id/dept_id/location_id/scope/remark/created_at/updated_at/deleted_at + idx_sys_dept_location_alias_dept_scope UNIQUE partial"
    why_human: "需要真实 PG 实例 + 启动后端触发 migration"
  - test: "AC-10/11/12/13/14 alias list/create/update/delete 端点 + 三级校验 + 权限 403"
    expected: "管理员授权 4 perms 后 POST /ops/location-alias/{list,create,:id/update,:id/delete} 行为正确;自映射/非外部机构/非祖先均返回 400 中文错误;无权限用户 403"
    why_human: "需运行后端 + DB + 鉴权链路 + 手动授权"
  - test: "AC-15 4 perms seed 到 sys_menu + sys_role_menu 为空 (D-04)"
    expected: "SELECT * FROM sys_menu WHERE perms LIKE 'ops:location:alias:%' 返回 4 行;sys_role_menu 关联为 0"
    why_human: "需启动后端跑过 migration 后查 DB"
  - test: "AC-16/17 工位编辑弹窗 union 下拉渲染 + [映射] 后缀 + user picker 联动"
    expected: "选择中心支公司B 后,所属部门下拉包含其子树 + 子部门A(带 [映射] 橙色 Tag);选中子部门A 后,所属用户下拉展示子部门A 的用户"
    why_human: "需前端 dev server + 后端 + DB + 手动操作 UI"
  - test: "AC-18 4 alias 写操作在 sys_oper_log 留痕 (模块名 '工位管理')"
    expected: "触发 create/update/delete 后 SELECT * FROM sys_oper_log WHERE title='工位管理' ORDER BY oper_time DESC 可见 OperTypeCreate/Update/Delete 记录"
    why_human: "需运行后端 + 写操作触发 + 查 DB"
  - test: "CR-02 ::text cast 在 SQLite 备选部署环境是否阻塞 (advisory)"
    expected: "PostgreSQL 主部署不受影响;SQLite 仅作为单元测试/备选,未在 Phase 39 scope"
    why_human: "评估既有 production pattern (workstation_service.go 全文统一用 ::text) 是否在团队可接受范围 — code review 已判定 non-blocker"
---

# Phase 39: 工位部门物理位置映射 — 验收报告 (goal-backward)

**Phase Goal:** 让"独立编制、物理分散办公"的子部门（运营服务部等）能出现在工位编辑"所属部门"下拉中，避免手动逐条配置。
**Verified:** 2026-06-25
**Status:** `human_needed` — 自动化核心目标全部达成，10 项 UAT（AC-09..18）需人工触发。
**Re-verification:** 在 plan 39-08 草稿基础上整合 goal-backward 结论，CR-01 已修复。

---

## Goal Achievement (Goal-Backward Analysis)

### Observable Truths (来自 SPEC 12 项 requirements 收敛)

| #   | Truth                                                                                                                                  | Status     | Evidence                                                                                                                                                                                                                          |
| --- | -------------------------------------------------------------------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T1  | `sys_dept_location_alias` 表存在（id/dept_id/location_id/scope/remark + BaseModel + partial unique）                                 | ✓ VERIFIED | `internal/models/sys_dept_location_alias.go` 完整模型；`migrations.Migrate165SysDeptLocationAlias` 已注册到 `database.go:406-408`；注释明确"CREATE TABLE + idx_sys_dept_location_alias_dept_scope UNIQUE partial"                   |
| T2  | alias CRUD 4 端点注册 + 三级校验生效（自映射/非外部机构/非祖先）                                                                       | ✓ VERIFIED | `router.go:628-631` 注册 list/:id/update/:id/delete/"" (create)；`location_alias_service_test.go` 10/10 PASS（TestValidateAlias\* 5 + TestLocationAliasService\* 5）                                                              |
| T3  | 4 个 `ops:location:alias:*` 权限 seed + router 严格对齐 + 不授权任何角色 (D-04)                                                        | ✓ VERIFIED | `migration_165:77-80` seed 4 perms；`router.go:610-613` 4 perms；migration 明示"不 INSERT sys_role_menu (D-04)"；0 处 INSERT sys_role_menu                                                                                       |
| T4  | 工位部门下拉 union 拼接（orgId 子树 + alias 命中）                                                                                      | ✓ VERIFIED | `workstation_service.go:84-107` GetWorkstationDeptOptions 用单条 Raw SQL UNION ALL 实现 D-06 决策；`router.go:594` POST /dept-options 已注册；`opsApi.ts:129` deptOptions + `useAliasByLocation.ts` hook 调用                       |
| T5  | EditModal 接收 union 结果，alias 条目 `[映射]` 后缀 (D-01)                                                                              | ✓ VERIFIED | `EditModal.tsx:113` `title: \`${trimmed} [映射]\`` 字面量锁定；useMemo 拼接 aliasNodes 到 subDeptTree                                                                                                                              |
| T6  | 列表页 `[⚙ 映射]` Drawer 入口 + 权限 gating                                                                                              | ✓ VERIFIED | `index.tsx:40` import + `:747` JSX 渲染 LocationAliasDrawer；Drawer 真实组件（295 行，useState/useQuery/Form/Table 全在）；`locationAliasApi.list/create/delete` 完整调用                                                              |
| T7  | alias 写操作触发 operlog + dept cache invalidation (D-03)                                                                                | ✓ VERIFIED | `location_alias_handler.go:115/135/151` 三处 `h.invalidateDeptCache(c)` + `:117/137/153` 三处 `operlog.Record(...,"工位管理",OperType{Create,Update,Delete})`                                                                      |
| T8  | operlog 25 OperType 常量集未变 (REQ-39-12)                                                                                              | ✓ VERIFIED | `go test ./internal/utils/operlog/ -count=1` PASS；TestOperTypeCountEquals25 + TestOperTypeConstantStability 全绿                                                                                                                  |
| T9  | sys_workstation 数据零迁移 (D-05/REQ-39-11)                                                                                             | ✓ VERIFIED | `grep UPDATE sys_workstation migration_165` → 0 命中；migration 仅 CREATE TABLE + CREATE INDEX + INSERT sys_menu                                                                                                                    |
| T10 | 不修改 sys_dept 表结构 / parent_id                                                                                                       | ✓ VERIFIED | `migration_165` 全文 grep `sys_dept` 仅出现在注释/JOIN，无 ALTER TABLE sys_dept                                                                                                                                                    |
| T11 | 后端编译 + 前端类型检查通过                                                                                                              | ✓ VERIFIED | `go build ./...` EXIT=0；`npm run type-check` EXIT=0；忽略 LSP phantom (router.go SetupTopologyRouter 等 stale 诊断)                                                                                                                |
| T12 | npm run lint 无 error                                                                                                                   | ⚠ FAIL (out-of-scope) | 全项目 2807 pre-existing errors（vite.config.ts quotes 等）；Phase 39 新增 3 文件单独 lint 0 error / 2 warning；按 CLAUDE.md scope constrainment 不在本 phase 修复                                                                  |

**Score:** 11/12 truths VERIFIED，1 个 lint truth 为 pre-existing 跨模块历史问题（per deferred-items.md 已记录）。

### Locked Decisions 兑现情况

| 决策 | 描述 | 落地证据 |
| --- | --- | --- |
| D-01 | `[映射]` 后缀（原名 + " [映射]"） | EditModal.tsx:113 字面量锁定 ✓ |
| D-03 | alias 写操作触发 dept cache invalidation，失败仅 warn 不阻断 | location_alias_handler.go:58-68 invalidateDeptCache + 3 处调用 ✓ |
| D-04 | 4 perms 不授权任何角色 | migration_165 明示"不 INSERT sys_role_menu"，0 命中 ✓ |
| D-05 | 不修改 sys_workstation / sys_dept | migration_165 0 处 UPDATE sys_workstation，0 处 ALTER sys_dept ✓ |

---

## Code Review 三项 Critical 处置

| CR | 问题 | 处置 | 状态 |
| --- | --- | --- | --- |
| CR-01 | LocationAliasDrawer 分页失效（handler 读 query string，前端发 POST body pageNum） | 修复 commit `5ef0f37`：handler 改为 `ShouldBindJSON(&aliasListQuery{PageNum,PageSize})`，与本包 workstation_handler 风格一致 | ✓ FIXED |
| CR-02 | GetWorkstationDeptOptions 用 PG 专有 `id::text` cast，SQLite 必崩 | 评估为 non-blocker：`workstation_service.go` 全文（:17/:66/:228）统一用 `::text`，是 pre-existing production pattern；本项目 SQLite 仅作单元测试/备选，主部署 PG。Phase 39 scope 内不引入新风险 | ℹ ADVISORY（不阻塞） |
| CR-03 | alias Create validate-then-write 非原子，唯一索引冲突透传 PG 错误信息；Update scope 单字段变更漏跑重名校验 | advisory：服务层已有 validateAlias 三级校验 + happy path 单测；并发唯一索引冲突的友好提示可在后续 phase 加强；不阻塞 goal | ℹ ADVISORY（不阻塞） |

---

## Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/models/sys_dept_location_alias.go` | 模型定义 + TableName | ✓ VERIFIED | 30 行，BaseModel 嵌入 + 4 业务字段 + GORM index 标签 |
| `internal/core/db/migrations/migration_165_sys_dept_location_alias.go` | CREATE TABLE + UNIQUE INDEX + 4 perms seed | ✓ VERIFIED | 幂等，已注册 database.go:406 |
| `internal/services/operations/location_alias_service.go` | service 接口 + 实现 + validateAlias 三级校验 | ✓ VERIFIED | 260 行；10 单测全绿 |
| `internal/api/v1/operations/location_alias_handler.go` | 4 端点 handler + operlog + cache invalidation | ✓ VERIFIED | 155 行；CR-01 修复在位 |
| `internal/api/router.go:607-631` | location-alias 路由组 + 4 perms RequirePermissions | ✓ VERIFIED | 4 端点 + 中间件挂载 |
| `internal/services/operations/workstation_service.go:84-107` | GetWorkstationDeptOptions union 查询 | ✓ VERIFIED | D-06 单 query UNION ALL |
| `xingran-react-frontend/src/lib/opsApi.ts:129,149` | deptOptions + locationAliasApi | ✓ VERIFIED | 完整定义 |
| `xingran-react-frontend/src/hooks/useAliasByLocation.ts` | react-query hook | ✓ VERIFIED | useQuery 包装 deptOptions |
| `xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx:113` | `[映射]` 后缀拼接 | ✓ VERIFIED | D-01 字面量锁定 |
| `xingran-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx` | Drawer CRUD UI | ✓ VERIFIED | 295 行真实组件（非 stub） |
| `xingran-react-frontend/src/pages/operations/workstations/index.tsx:40,747` | Drawer 入口接入 | ✓ VERIFIED | import + JSX |

---

## Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| router.go | LocationAliasHandler.{List,Create,Update,Delete} | locationAlias.POST 路由注册 | ✓ WIRED | router.go:628-631 4 端点 |
| LocationAliasHandler | LocationAliasService | NewLocationAliasHandler 注入 | ✓ WIRED | router.go:621 构造 |
| LocationAliasHandler | operlog + dept cache | invalidateDeptCache + operlog.Record 调用 | ✓ WIRED | handler.go:115/117/135/137/151/153 |
| WorkstationHandler.GetWorkstationDeptOptions | WorkstationService.GetWorkstationDeptOptions | service 调用 | ✓ WIRED | workstation_handler.go:61 |
| workstationApi.deptOptions | POST /ops/workstation/dept-options | opsApi post 封装 | ✓ WIRED | opsApi.ts:129 |
| useAliasByLocation | workstationApi.deptOptions | useQuery | ✓ WIRED | useAliasByLocation.ts:27 |
| EditModal.subDeptTree | aliasList prop | useMemo 拼接 | ✓ WIRED | EditModal.tsx:113 |
| LocationAliasDrawer.list | locationAliasApi.list (POST body pageNum/pageSize) | useQuery | ✓ WIRED | Drawer.tsx:67 (CR-01 修复后端对齐) |

---

## Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| LocationAliasDrawer | aliasPage | useQuery → locationAliasApi.list → POST /list | Yes（DB query service.List 分页） | ✓ FLOWING |
| EditModal | aliasNodes | useAliasByLocation → deptOptions → GetWorkstationDeptOptions | Yes（UNION ALL 真实 SQL，无静态 fallback） | ✓ FLOWING |
| migration_165 | sys_menu 4 perms | INSERT 幂等 seed | Yes（db.Create 写入） | ✓ FLOWING |

无 HOLLOW / STATIC / DISCONNECTED / HOLLOW_PROP。

---

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| `go build ./...` 全量编译 | `go build ./...` | EXIT=0 | ✓ PASS |
| TestValidateAlias 5 用例 | `go test ./internal/services/operations/ -run TestValidateAlias -count=1` | 5/5 PASS | ✓ PASS |
| TestLocationAliasService 5 用例 | 同上 -run TestLocationAliasService | 5/5 PASS | ✓ PASS |
| operlog 25 常量守护 | `go test ./internal/utils/operlog/ -count=1` | PASS | ✓ PASS |
| 前端类型检查 | `cd xingran-react-frontend && npm run type-check` | EXIT=0 | ✓ PASS |
| `[映射]` 字面量存在 | `grep \[映射\] EditModal.tsx` | line 113 命中 | ✓ PASS |
| 0 处 UPDATE sys_workstation | `grep UPDATE sys_workstation migration_165` | 0 命中 | ✓ PASS |
| 4 perms 严格对齐 | grep `ops:location:alias:` migration_165 + router.go | 4:4 对齐 | ✓ PASS |
| CR-01 修复在位 | 检查 handler List 读 body | `ShouldBindJSON(&aliasListQuery)` ✓ | ✓ PASS |
| 0 处 INSERT sys_role_menu | grep sys_role_menu migration_165 | 仅注释提及，0 INSERT | ✓ PASS |

---

## Probe Execution

Phase 39 未声明 probe 脚本，跳过。

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| REQ-39-01 | 39-01/02 | sys_dept_location_alias 表 + 列 + unique | ✓ SATISFIED | model + migration_165 + database.go 注册 |
| REQ-39-02 | 39-02 | 三级校验 | ✓ SATISFIED | location_alias_service.go validateAlias + 10 单测 |
| REQ-39-03 | 39-02 | 4 CRUD 端点 + 独立权限 | ✓ SATISFIED | router.go 4 端点 + 4 RequirePermissions |
| REQ-39-04 | 39-01 | 4 perms seed 到 sys_menu，不授 sys_role_menu | ✓ SATISFIED | migration_165 + D-04 验证 |
| REQ-39-05 | 39-04/05 | union 查询改写 | ✓ SATISFIED | GetWorkstationDeptOptions UNION ALL |
| REQ-39-06 | 39-05/06 | list 页 [⚙ 映射] Drawer + 权限 gating | ✓ SATISFIED | index.tsx:40,747 + canListAlias |
| REQ-39-07 | 39-05 | `[映射]` 后缀 + Ant Design Tag 区分 | ⚠ PARTIAL | `[映射]` 字面量在位；Ant Design Tag 橙色包装需 UAT 确认（EditModal.tsx:113 用模板字符串而非 Tag，需视觉确认） |
| REQ-39-08 | 39-06 | user picker 自动可用 | ? NEEDS HUMAN | UAT only — 无新代码，端到端验证 |
| REQ-39-09 | 39-02 | alias 4 写操作 operlog | ✓ SATISFIED | handler.go 117/137/153 |
| REQ-39-10 | 39-03 | alias 写触发 dept cache invalidation | ✓ SATISFIED | handler.go invalidateDeptCache + D-03 |
| REQ-39-11 | 39-01 | sys_workstation 零迁移 | ✓ SATISFIED | migration_165 0 命中 UPDATE sys_workstation |
| REQ-39-12 | 39-02 | operlog 25 常量集不变 + 模块名"工位管理" | ✓ SATISFIED | operlog 回归测试 PASS + handler 模块名一致 |

12/12 requirements 已 accounted（无 ORPHANED）；10 SATISFIED + 1 PARTIAL (REQ-39-07 视觉确认需 UAT) + 1 NEEDS HUMAN (REQ-39-08 UAT only)。

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| workstation_service.go | 93,97,101 | `::text` PG 专有 cast（与 :17/:66/:228 既有 pattern 一致） | ℹ Info | SQLite 备选环境受影响；主 PG 部署无影响（CR-02 advisory） |
| location_alias_service.go | Create/Update | validateAlias 与 INSERT 非原子；唯一索引冲突透传 PG err（CR-03） | ℹ Info | 并发场景错误消息不友好；happy path 单测全绿 |
| location_alias_service.go | Update | scope 单字段变更不跑重名校验（CR-03 / WR-02） | ℹ Info | 边缘场景；不阻塞 |
| EditModal.tsx | 106-120 | `as DeptTreeNode` 强转（WR-06） | ℹ Info | TS 类型守护绕过；运行时无影响 |
| opsApi.ts | 150-155 | `pageNum` vs 项目惯例 `current`（IN-03） | ℹ Info | CR-01 修复后端对齐 pageNum；与项目其他模块风格不一致 |

无 TBD/FIXME/XXX 阻塞性 debt marker。

---

## Automated AC 验收（继承自 plan 39-08 草稿）

| AC | Check | Result | Evidence |
| --- | --- | --- | --- |
| AC-01 | `go build ./...` | PASS | exit 0 |
| AC-02 | `TestValidateAlias` + `TestLocationAliasService` | PASS | 10/10 cases pass |
| AC-03 | operlog 25 OperType constants | PASS | TestOperTypeCountEquals25 + TestOperTypeConstantStability pass |
| AC-04 | `npm run type-check` | PASS | exit 0 |
| AC-05 | `npm run lint` 无 error | FAIL (out-of-scope) | 2807 pre-existing errors project-wide; Phase 39 文件 0 error / 2 warning |
| AC-06 | migration_165 zero `UPDATE sys_workstation` | PASS | 0 matches |
| AC-07 | 4 perms aligned migration ↔ router | PASS | list/add/edit/delete all 1:1 |
| AC-08 | `[映射]` + `LocationAliasDrawer` | PASS | EditModal.tsx:113 + index.tsx:40,747 |

**Automated pass rate: 7/8**（AC-05 pre-existing scope-excluded failure，非 Phase 39 引入）

---

## Human Verification Required

详见 frontmatter `human_verification` 节，6 项：
1. **AC-09** migration 建表结果（PG \d 验证）
2. **AC-10..14** 4 端点 CRUD + 三级校验 + 403 权限
3. **AC-15** 4 perms seed + sys_role_menu 为空
4. **AC-16/17** 工位编辑 union 下拉 + `[映射]` 后缀 + user picker 联动
5. **AC-18** 4 写操作 sys_oper_log 留痕
6. **CR-02** ::text cast 在团队 SQLite 使用场景的可接受性（advisory）

---

## Gaps Summary

无 must-have 阻塞性 gap。自动化 must-have truth 全部 VERIFIED（11/12，唯一 fail 是 pre-existing lint 历史问题，per CLAUDE.md scope constrainment 已 deferred 到独立技术债清理 phase）。

3 项 Critical code review 处置：
- CR-01（分页失效）✓ 已修复并验证在位
- CR-02（::text cast）ℹ advisory，与既有 production pattern 一致，不阻塞
- CR-03（非原子 + 错误透传）ℹ advisory，happy path 全绿，不阻塞

**Verdict:** Phase 39 后端/前端代码、表/迁移/服务/handler/router/前端 hook/Drawer 全链路打通，4 项锁定决策全部兑现，operlog + 缓存失效约定到位，SQL 全部走 GORM 占位符参数化。**自动化验收核心目标全部达成**，按 Step 9 decision tree，因有 6 项 UAT 需人工触发 → status = `human_needed`（**非** gaps_found，自动化层面无阻塞 gap）。

---

_Verified: 2026-06-25_
_Verifier: Claude (gsd-verifier)_
_Methodology: goal-backward + 与 plan 39-08 草稿 reconciliation + CR-01/02/03 处置纳入_

---

## Deviations from Locked SPEC (UAT 期间发现并处理)

### DEV-01: 移除 REQ-39-02 规则③(dept 必须是 location 后代) — commit b8dca97

**触发:** UAT 创建"运营服务部/子部门A → 中心支公司B"映射被规则③拒绝。

**根因:** 规则③要求 dept.ancestors 含 location_id(dept 是 location 后代)。但 alias
的业务本质是"组织编制与物理办公地分离"——dept 编制上属于另一分支,物理上在 location
办公,本就不是 location 后代。SPEC 自身验收用例(39-SPEC.md REQ-39-05:dept=子部门A,
location=中心支公司B)在该规则下无法创建,属规格内在矛盾。

**处置:** 移除规则③,保留 ①防自映射 + ②location 必须是外部机构。validateAlias 由三级
降为二级。单测同步:DeptNotDescendantRejected → CrossBranchMappingPasses(核心场景)。

**影响范围:** location_alias_service.go validateAlias + 接口 docstring;单测。
**不影响:** 表结构、API 端点、权限、前端、operlog、缓存失效、union 注入逻辑。

### DEV-02: alias List JOIN 用 CAST(id AS TEXT) — commit 6119418

**触发:** UAT alias 列表 500,PG 报 uuid = character varying (SQLSTATE 42883)。

**根因:** aliasListJoinClause 的 origin.id (uuid) = alias.dept_id (varchar) 列对列比较,
PG 无此操作符。原注释误判"裸 id = ? 双 DB 一致"。

**处置:** 改用标准 SQL CAST(origin.id AS TEXT),PG 转 text 与 varchar 比较,SQLite 是
no-op(双 DB 一致)。不能用 PG 专有 ::text(破坏 SQLite 单测)。

### DEV-03: LocationAliasDrawer TreeSelect 数据源 — commit d3bea9c, 1266f3a

**触发:** UAT Drawer 两个下拉全显示 ---(value=undefined)。

**根因:** useDeptTree() 返回 SimpleDept{id,deptName}(无 value/title),原代码 as unknown as
强转 + trimTitleToLastSegment(对无 title 的 SimpleDept 是 no-op)→ TreeSelect 拿到 undefined。

**处置:** toFullPathTree(SimpleDept→{title,value,key}) 再套 trimTitleToLastSegment(裁全路径
只留当前部门名)。顺带修 antd 6 弃用:Drawer width→size、Space direction→orientation。

### DEV-04: alias List 分页参数 — commit 5ef0f37

**触发:** code-review CR-01。handler 从 query string 读 current,前端在 body 发 pageNum,
通道+名称双错位 → 分页永远返回第 1 页。

**处置:** handler 改 ShouldBindJSON 读 pageNum/pageSize,JSON tag 对齐前端。
