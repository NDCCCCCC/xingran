---
phase: 59-ad-login-guard-reconciliation
plan: 03
subsystem: backend-api
tags: [ad-sync, reconciliation, gin, rbac, migration, gorm, postgres, sqlite, pagination]

requires:
  - phase: 59-ad-login-guard-reconciliation (59-02)
    provides: 普通 SQL 视图 v_dept_ou_mismatch（D-11 十一列，migration_217 双方言三路径注册）
  - phase: 58-compare-and-move
    provides: MatchADConfigByDN 最长后缀唯一匹配契约（Go/SQL 一致性测试的 Go 侧锚点）
  - phase: 57-ad-outbox-lite
    provides: 迁移三路径注册先例（PG 分支 / sqlite 分支 / SKIP_AUTOMIGRATE 旁路）

provides:
  - 只读分页服务 addomain.DeptOUMismatchService（DTO / 四值白名单 / LIKE 转义 / 确定性排序）
  - 端点 POST /api/v1/ad-domain/reconciliation/mismatch（独立鉴权 ops:ad:config:reconciliation:list）
  - migration_218：menu_type='F' 按钮权限行 + 仅授已关联父菜单角色的精准授权（幂等）
  - Go MatchADConfigByDN 与 SQL config_unrouted 七组边界用例的一致性自动化锁定
  - 前端 permissions 数组可读的新权限点载体（59-04 Tab 的 hasPerm 依据）

affects: [59-04-frontend, 60-soft-delete-recovery]

actuals:
  tokens: 12000
  tasks: 3
  commits: 3
  plan_head_before: 619d7a2f44f5ee5a83cb755d8b801a7e66b20afe
  plan_head_after: a640658d7c3aec50e6edc5dd6dcc5780e83a9447

tech-stack:
  added: []
  patterns:
    - "只读对账端点四件套：DTO 与视图列逐列显式 gorm column 绑定 → handler 白名单 + NormalizePagination → service 参数绑定 + 确定性 ORDER BY → response.Page"
    - "权限字符串双端单点：路由 RequirePermissions 与迁移 seed 用同一字面常量，测试 grep 锁定路由侧恰好 1 处"
    - "精准授权 SQL：INSERT...SELECT...WHERE NOT EXISTS 单语句（PG/sqlite 通用、幂等），只授已关联父菜单的角色"
    - "白名单/枚举单一事实源导出数组 + 包内判定函数：handler 遍历导出数组判定，避免为跨包校验新开导出面"

key-files:
  created:
    - internal/services/addomain/dept_ou_mismatch_service.go
    - internal/services/addomain/dept_ou_mismatch_service_test.go
    - internal/api/v1/system/ad_reconciliation_handler.go
    - internal/api/v1/system/ad_reconciliation_handler_test.go
    - internal/core/db/migrations/migration_218_ad_reconciliation_permission.go
    - internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go
  modified:
    - internal/api/v1/system/ad_domain_router.go
    - internal/core/db/database.go

key-decisions:
  - "D-14 路径按真实挂载对齐为 /api/v1/ad-domain/reconciliation/mismatch：AD 路由组挂在 r.Group(\"/ad-domain\")（router.go:519）、前端 baseURL=/api/v1；刻意不挂 /system/ad-config 第二套别名（防双契约漂移），D-14 语义（同组、独立鉴权）不变"
  - "handler 白名单复用导出的 addomain.MismatchTypes 遍历判定：validMismatchType 保持包内私有（plan artifact 边界），四值单一事实源仍在服务层，不为跨包校验新开导出面"
  - "migration_218 精准授权只授已关联『AD域控配置』父菜单的角色：单条 INSERT...SELECT...WHERE NOT EXISTS，不沿用 migration_163 的『所有 status=0 角色』宽授权面（防 antd Tree 父子联动视觉陷阱）"
  - "父菜单缺失 = log warning + 返回 nil（非阻断降级）：migration_163 同款 FLAGGED ASSUMPTION，部署清单登记『管理员手建按钮权限并授权』操作项"
  - "授权步骤不随 perms 行已存在被跳过：重跑迁移会给后接入父菜单的角色补授权（有行为断言锁定）"
  - "Visible/Status 零值 + 模型 default tag → GORM 省略列落 DB 默认值：migration_163 同款行为，F 型按钮可见性不参与前端渲染判定，迁移注释已写明"
  - "GUARD-02 不标 complete：requirements.ready-ids 返回 blocked（59-04 同样声明该需求），共享-ID 门要求全部声明者完成，由最后一个 plan 回填"

patterns-established:
  - "只读对账端点形状：ShouldBindJSON → 白名单 → NormalizePagination → service → response.Page；只读不写 operlog"
  - "权限载体迁移形状：幂等 perms 行 + 精准角色授权 + 三启动路径注册 + 父菜单缺失降级"

requirements-completed: [GUARD-02]

coverage:
  - id: D1
    description: "服务层从 v_dept_ou_mismatch 分页/筛选查询：四类口径可查、mismatchType/deptId 过滤生效、keyword 大小写不敏感且 %/_ 为字面量、空结果非 nil、确定性排序翻页不重不漏"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/services/addomain/ -run 'TestPhase59_DeptOUMismatch' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Go MatchADConfigByDN 与 SQL config_unrouted 在七组边界输入（最长后缀父子域/逗号边界/无匹配/空 base/停用/软删/等长歧义）结论一致"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/services/addomain/ -run 'TestPhase59_MatchConsistency' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "handler 绑定/校验/分页：非法 JSON 与非法 mismatchType 均 400、缺省分页归一 1/10、pageSize 钳制 200、service 错误非 2xx、成功 data 含 list/total/current/pageSize、空 list 序列化 []"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/api/v1/system/ -run 'TestPhase59_ReconHandler' -count=1"
        status: pass
    human_judgment: false
  - id: D4
    description: "路由注册与鉴权声明：/reconciliation 组挂 RequirePermissions(ops:ad:config:reconciliation:list) 并注册 POST /mismatch，不破坏全量路由注册"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/api/v1/system/ -run 'TestSetupRouters_AllRegisterRoutes' -count=1"
        status: pass
      - kind: other
        ref: "grep -c 'ops:ad:config:reconciliation:list' internal/api/v1/system/ad_domain_router.go == 1"
        status: pass
    human_judgment: false
  - id: D5
    description: "无权限角色访问端点被中间件拒为 403（T-59-05 的真实运行期面）"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "RequirePermissions 的 403 行为是既有中间件（Phase 36 起），本计划只声明新权限点；真实拒绝路径需部署环境带 JWT + 角色矩阵验证，归 59-04/UAT。"
  - id: D6
    description: "migration_218：F 行幂等恰 1 条、仅父菜单已关联角色获授权、父菜单缺失降级、后接入角色重跑补授权；database.go 三路径注册"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration218' -count=1"
        status: pass
      - kind: other
        ref: "grep -c 'Migrate218ADReconciliationPermission' internal/core/db/database.go == 3"
        status: pass
    human_judgment: false
  - id: D7
    description: "生产库存在 menu_name='AD域控配置' 且 menu_type='C' 的父菜单（迁移 seed 前提，FLAGGED ASSUMPTION）"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "sqlite 测试无法证明生产数据形态；迁移已按降级设计（warning + nil），真实库验证归 59-VALIDATION.md 部署冒烟，缺失时须管理员手建按钮权限并授权。"

duration: 15min
completed: 2026-10-19
status: complete
---

# Phase 59 Plan 03: 部门↔OU 对账只读端点 + 权限载体 Summary

`v_dept_ou_mismatch` 视图经 `DeptOUMismatchService` 暴露为 `POST /api/v1/ad-domain/reconciliation/mismatch`（独立鉴权 `ops:ad:config:reconciliation:list`），migration_217 视图 + migration_218 按钮权限落地，Go/SQL 的 `config_unrouted` 口径一致性由七组边界用例锁定。

## Performance

- **Duration:** 15min
- **Started:** 2026-10-19T14:14:23Z
- **Completed:** 2026-10-19T14:29:10Z
- **Tasks:** 3
- **Files modified:** 8 (6 created, 2 modified; +1210 insertions)

## Accomplishments

- 只读分页服务 `addomain.DeptOUMismatchService`：四值白名单、LIKE 元字符转义 + `ESCAPE '\'`、先 Count 后分页、确定性 ORDER BY（翻页不重不漏）、空结果序列化 `[]`。
- 端点 `POST /api/v1/ad-domain/reconciliation/mismatch` 挂在现有 `/ad-domain` 组下，由 `RequirePermissions` 独立鉴权；handler 负责绑定/白名单 400/`NormalizePagination`（默认 1/10、上限 200），只读不写 operlog。
- `migration_218` 落地 `menu_type='F'` 按钮权限行（perms=`ops:ad:config:reconciliation:list`，先例 migration_163），**精准授权**只授已关联『AD域控配置』父菜单的角色，三启动路径注册，全路径幂等。
- `TestPhase59_MatchConsistency` 在七组边界输入上锁定 Go `MatchADConfigByDN` 与 SQL 视图 `config_unrouted` 结论一致（D-12 的漂移风险点）。
- 17 个新自动化用例（服务 8 + handler 7 + 迁移 4 组）全绿；`go build ./...` 与三包全量测试无回归。

## Task Commits

Each task was committed atomically:

1. **Task 1: 端到端：服务层分页查询 v_dept_ou_mismatch + Go/SQL 口径一致性** - `2f1c8190` (feat，前一会话已提交，本会话复验 build + 测试通过)
2. **Task 2: 只读 handler 与 /ad-domain/reconciliation 路由注册** - `0d7b782c` (feat)
3. **Task 3: migration_218 权限载体与精准授权 + 三路径注册** - `a640658d` (feat)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `internal/services/addomain/dept_ou_mismatch_service.go` - 只读分页服务：DTO（D-16/D-11）、四值白名单 `MismatchTypes` + `validMismatchType`、`escapeLike`、参数化视图查询 + 确定性排序。
- `internal/services/addomain/dept_ou_mismatch_service_test.go` - sqlite 内存库 fixture（复用 migration_217 建视图）+ 8 个用例，含 `TestPhase59_MatchConsistency` 七组 Go/SQL 一致性。
- `internal/api/v1/system/ad_reconciliation_handler.go` - `ADReconciliationHandler`：绑定 → 白名单 400 → 分页归一 → `response.Page`；无 operlog 调用。
- `internal/api/v1/system/ad_reconciliation_handler_test.go` - fake service 注入 + `gin.CreateTestContext`，7 个用例（400/归一/钳制/错误/形状/空 `[]`）。
- `internal/api/v1/system/ad_domain_router.go` - `/reconciliation` 组 + `RequirePermissions` + `POST /mismatch`，注释记录 D-14 路径对齐决定。
- `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go` - F 型按钮权限行 + 精准授权（NOT EXISTS 幂等）+ 父菜单缺失降级。
- `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go` - 4 个 sqlite 用例（幂等/精准授权/降级/后接入角色补授权）。
- `internal/core/db/database.go` - Migrate218 在 PG 分支 / sqlite 分支 / `BootstrapMissingTables` 旁路各注册一次（非阻断档，`grep -c` = 3）。

## Decisions Made

- **D-14 路径按真实挂载对齐。** CONTEXT 字面 `/system/ad-config/reconciliation/mismatch`，但仓库把 AD 路由组挂在 `r.Group("/ad-domain")` 且前端 baseURL=`/api/v1`，最终路径定为 `/api/v1/ad-domain/reconciliation/mismatch`；刻意不挂第二套别名——别名会制造双契约与长期漂移（plan objective Decision Reconciliation 第 1 条）。
- **handler 白名单遍历导出的 `MismatchTypes`。** `validMismatchType` 按计划保持包内私有；handler 用 `for range addomainServices.MismatchTypes` 判定，四值单一事实源仍在服务层，不为跨包校验新开导出面。
- **精准授权替代宽授权。** migration_163 给「所有 status=0 角色」授权，本迁移改为单条 `INSERT...SELECT...WHERE NOT EXISTS`（PG/sqlite 通用、幂等），只授已关联父菜单的角色——防 antd Tree 父子联动视觉陷阱（memory: migration-grant-new-menu-precision-helper）。
- **授权与创建解耦。** perms 行已存在时跳过创建但**仍执行授权**，后接入父菜单的角色重跑迁移即可补拿权限（`TestPhase59_Migration218_GrantsNewRolesLater` 锁定该行为）。
- **父菜单缺失非阻断降级。** 查不到『AD域控配置』（C 型）时 log warning 返回 nil——FLAGGED ASSUMPTION 的部署清单操作项（管理员手建按钮权限并授权）不变。
- **GUARD-02 不标 complete。** `requirements.ready-ids` 返回 `{"ready":[],"blocked":["GUARD-02"]}`——59-04 同样声明该需求，共享-ID 门（#2388）要求全部声明者完成，REQUIREMENTS traceability 保持 In Progress，由 59-04 回填（沿用 59-02 分档口径）。

## Deviations from Plan

### Harness note (not a code deviation)

- **Protected-branch guard vs `branching_strategy: none`**：仓库配置为 `branching_strategy: "none"`，orchestrator 以 `<sequential_execution>` 指示在配置分支 `main` 上提交（未改配置、未用 `--no-verify`），与 59-01/59-02 同款处置。
- **Task 1 由前一会话提交（`2f1c8190`）**：两次执行因基础设施错误（API 不可用/503）中断，非工作缺陷；本会话复验 `go build ./...` 与 `go test ./internal/services/addomain/ -run 'TestPhase59_DeptOUMismatch|TestPhase59_MatchConsistency' -count=1` 全绿后未重做。

### Auto-fixed Issues

None - plan executed exactly as written.

**Total deviations:** 0 auto-fixed。**Impact on plan:** 无。

## Issues Encountered

None.（三任务验收命令与源断言全部首轮通过。）

## Known Stubs

无。服务/端点/迁移全部实装，无 TODO/FIXME/占位分支。

**已知边界（非 stub，锁定决策）**：① 视图口径不含「有用户但无任何映射」的 `unrecoverable:no-ou-mapping` 形态（59-02 已记录的 Known Gap，本 phase 只做可观测）；② 父菜单缺失时迁移降级跳过，权限点需管理员手建（部署清单操作项，见 D-07 coverage）。

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new_authenticated_read_endpoint | internal/api/v1/system/ad_reconciliation_handler.go | 新增受 JWT + `RequirePermissions` 保护的只读端点，暴露用户名 / 部门名 / DN（T-59-07 信息披露面）。缓解：独立权限点 + migration_218 可授予载体 + 参数化查询（T-59-06）+ pageSize 钳制 200（T-59-08）；无匿名入口。 |
| threat_flag: permission_seed_writes_sys_menu | internal/core/db/migrations/migration_218_ad_reconciliation_permission.go | 迁移向 sys_menu / sys_role_menu 写入权限行与角色关联；授权面刻意收窄为「已关联父菜单的角色」，无全角色宽授权。 |

## User Setup Required

**部署操作项（人工，非代码）**：目标环境的父菜单须存在（`menu_name='AD域控配置'` 且 `menu_type='C'`）；若缺失，管理员需手建按钮权限 `ops:ad:config:reconciliation:list` 并为相关角色授权，否则前端 Tab（59-04）对非超管角色恒不渲染。超管后端短路可见，普通角色依赖该授权。

## Next Phase Readiness

- 59-04 可直接消费：端点 `POST /ad-domain/reconciliation/mismatch`（前端 baseURL `/api/v1`），请求 `{current,pageSize,mismatchType?,deptId?,keyword?}`，响应 `data.{list,total,current,pageSize}`；`hasPerm('ops:ad:config:reconciliation:list')` 控制 Tab 内容渲染（D-17）。
- 服务层 DTO json tag 已按 D-16 定型（camelCase，`DeptOUMismatchRow` 十一字段全可空除 `mismatchType/mismatchDetail`），前端列定义照抄即可。
- **部署前置（人工）**：真实启动一次验证 PG 分支（migration_217 视图 + migration_218 权限 seed，二者均非阻断注册，启动日志核对无 `对账查询按钮权限 seed 失败`）。
- **未完成项**：GUARD-02 在 REQUIREMENTS.md 仍为 In Progress（共享-ID 门 blocked），由 59-04 完成后回填。

## Self-Check

- FOUND: internal/services/addomain/dept_ou_mismatch_service.go
- FOUND: internal/services/addomain/dept_ou_mismatch_service_test.go
- FOUND: internal/api/v1/system/ad_reconciliation_handler.go
- FOUND: internal/api/v1/system/ad_reconciliation_handler_test.go
- FOUND: internal/core/db/migrations/migration_218_ad_reconciliation_permission.go
- FOUND: internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go
- FOUND: 2f1c8190 (task 1, prior session)
- FOUND: 0d7b782c (task 2)
- FOUND: a640658d (task 3)
- `go build ./...` exit 0
- `go test ./internal/services/addomain/ -run 'TestPhase59_' -count=1` → ok
- `go test ./internal/api/v1/system/ -count=1` → ok（全包无回归）
- `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration218' -count=1` → ok（4 用例全 PASS）
- 源断言：路由 perms 字面 = 1；handler 非 `operlog.Record(` 计数 = 0；database.go `Migrate218ADReconciliationPermission` = 3；迁移内 `"C"` 字面 = 2（≥1）

## Self-Check: PASSED

---
*Phase: 59-ad-login-guard-reconciliation*
*Completed: 2026-10-19*
