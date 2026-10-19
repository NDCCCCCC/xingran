---
phase: 59-ad-login-guard-reconciliation
verified: 2026-10-19T16:35:48Z
status: human_needed
score: 12/14 must-haves verified
covered_files:
  - ".planning/phases/59-ad-login-guard-reconciliation/59-01-PLAN.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-01-SUMMARY.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-02-PLAN.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-02-SUMMARY.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-03-PLAN.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-03-SUMMARY.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-04-PLAN.md"
  - ".planning/phases/59-ad-login-guard-reconciliation/59-04-SUMMARY.md"
  - "internal/api/v1/system/ad_domain_router.go"
  - "internal/api/v1/system/ad_reconciliation_handler.go"
  - "internal/api/v1/system/ad_reconciliation_handler_test.go"
  - "internal/core/db/database.go"
  - "internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go"
  - "internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go"
  - "internal/core/db/migrations/migration_218_ad_reconciliation_permission.go"
  - "internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go"
  - "internal/core/db/sqlite_reconciliation_views.go"
  - "internal/services/addomain/dept_ou_mismatch_service.go"
  - "internal/services/addomain/dept_ou_mismatch_service_test.go"
  - "internal/services/system/user_sync_service.go"
  - "internal/services/system/user_sync_service_phase59_test.go"
  - "ruoyi-react-frontend/src/lib/adDomainApi.test.ts"
  - "ruoyi-react-frontend/src/lib/adDomainApi.ts"
  - "ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css"
  - "ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx"
  - "ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx"
  - "ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/index.render.test.tsx"
  - "ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx"
covered_digest: "v3:sha256:789b088e75063f07e76dcd0476e12c2fa44b48c00a03abc0438fd84205a0cdc8"
behavior_unverified: 2
overrides_applied: 0
behavior_unverified_items:
  - truth: "空态标题 16px/600 由 ReconciliationTab.css 的 !important 规则在真实浏览器层叠中生效"
    test: "浏览器打开 AD 配置页 → 对账 Tab → 筛调成 0 结果 → DevTools 检查空态标题 computed style"
    expected: "computed font-weight: 600、font-size: 16px"
    why_human: "规则本体（.reconciliation-tab-empty .ant-empty-description > div > div:first-child + font-weight:600 !important）已在源码核实存在且被组件 import，但 jsdom 不计算层叠样式，inline style 的 !important 覆盖无法在 vitest 断言（59-VALIDATION Manual-Only 登记）"
  - truth: "当前仅 2 个 Tab，Tab 栏不换行/不溢出（PLAN 04 verification: backstop）"
    test: "窄屏与宽屏浏览器打开 AD 配置页，观察 Tab 栏形态"
    expected: "『配置列表』『对账』两个标签不换行、不截断"
    why_human: "纯视觉回归断言 jsdom 不可得；backstop 级真话无自动化证据即弃权（insufficient_spec），归 59-UAT 视觉复核"
human_verification:
  - test: "真实 AD 登录守卫拦截（GUARD-01 端到端）"
    expected: "管理员编辑用户 dept → 等队列入队（pending/failed）→ 该用户 AD 登录 → 日志出现『部门回写被活跃队列守卫拦截』WARN（含 userID/username/本地 dept UUID/AD 解析 dept UUID/队列行 ID/action_type/attempts），且 sys_user.dept_id 未被 AD 旧值覆盖"
    why_human: "单元测试已证明 SyncADUser 组件级行为（6/6 PASS），但真实 LDAP 登录链路（ad_authenticator → SyncADUser）需真实 AD 环境与 pending 队列行（59-VALIDATION Manual-Only）"
  - test: "真实 PG 启动冒烟：migration_217 视图 + migration_218 权限 seed + 二次重启幂等"
    expected: "目标 PG 环境启动服务：AutoMigrate 前 DROP 视图、Migrate217 无错、SELECT count(*) FROM v_dept_ou_mismatch 可执行；再次重启同库无 SQLSTATE 0A000『cannot alter type』且视图重建成功；启动日志无『对账查询按钮权限 seed 失败』"
    why_human: "本机无 PG 实例；sqlite 测试覆盖不到 uuid↔varchar CAST、boolean NOT、CREATE OR REPLACE VIEW 的真实执行（DDL 文本守卫 TestPhase59_Migration217_PGDialectDDL 已过，但文本断言≠运行期证明）。本仓库有 MV/视图阻塞 AutoMigrate 启动 FATA 前科，PG 启动是全 phase 风险最高且完全未演练的路径"
  - test: "对账 Tab 浏览器权限渲染（授权与未授权两态）"
    expected: "管理员为角色授予 ops:ad:config:reconciliation:list → 该角色登录后对账 Tab 出数据；未授权角色登录后 Tab 内渲染『无对账查看权限』403 且不发对账数据请求；直接 POST /api/v1/ad-domain/reconciliation/mismatch 被后端 RequirePermissions 拒为 403"
    why_human: "RequirePermissions 的真实运行期拒绝（JWT + 角色矩阵）与浏览器渲染均需部署环境（59-03 coverage D5 / 59-VALIDATION Manual-Only）"
  - test: "生产库父菜单『AD域控配置』（menu_type='C'）存在性确认"
    expected: "生产 sys_menu 存在该父菜单 → migration_218 自动建 F 行并精准授权；若缺失 → 管理员需手建按钮权限 ops:ad:config:reconciliation:list 并为相关角色授权（迁移已按 warning + nil 降级，不会启动失败）"
    why_human: "sqlite 测试无法证明生产数据形态（59-01/02/03/04 四个 PLAN 的 FLAGGED ASSUMPTION 同款）；缺失时普通角色对账 Tab 恒 403"
  - test: "GUARD-01/GUARD-02 验收口径人工复核（FLAGGED ASSUMPTION 闭环）"
    expected: "确认 ROADMAP SC1~SC4 原文即权威验收口径（四个 PLAN frontmatter 的 edge probe 均报 unclassified/unresolved，未假设 backstop）"
    why_human: "probe 分类机制自身无法裁决；四个 PLAN 均显式要求 phase verification 时人工复核验收口径来源"
  - test: "空态标题字重 600（computed style）"
    expected: "computed font-weight: 600（见 behavior_unverified_items 第 1 条）"
    why_human: "jsdom 不计算层叠样式"
  - test: "Tab 栏不溢出视觉回归（backstop）"
    expected: "两个 Tab 标签不换行/不截断（见 behavior_unverified_items 第 2 条）"
    why_human: "纯视觉断言，jsdom 不可得"
---

# Phase 59: 登录回写守卫与部门对账视图 Verification Report

**Phase Goal:** AD 登录回写 dept 不再用旧值回滚本地编辑（信号守卫替代被三方审计否决的时间戳仲裁）；dept↔OU 不一致（含映射指向软删部门）可观测。
**Verified:** 2026-10-19T16:35:48Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

两个目标分量均在代码库成立：GUARD-01 信号守卫真实存在于登录回写唯一写入点（`internal/services/system/user_sync_service.go:428-446`，经 `ad_authenticator.go:131 → SyncADUser` 接入真实登录链路），pending/failed 拦截、无行放行、查询失败保守降级均有通过中的行为测试；GUARD-02 可观测链路视图 → 服务 → 端点 → 权限载体 → 前端 Tab 五级全通，四类口径（含 dept_soft_deleted）有受控行为断言。无 FAILED 真话、无 MISSING/STUB 工件、无 NOT_WIRED 关键链路。未能自动证明的部分全部是环境约束类（真实 AD、真实 PG、浏览器视觉），已结构化为人工验证项 —— 因此状态为 human_needed 而非 passed。

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | SC1：存在 pending/failed update_user 队列行的用户经登录同步时 dept_id 不被覆盖且记 WARN | ✓ VERIFIED | `user_sync_service.go:432-439` 三分支 switch；`TestPhase59_SyncADUser_ActivePendingQueueSkipsDept` / `_ActiveFailedQueueSkipsDept` PASS（真实 sqlite DB + 真实 DeptOUmapper + migration_215 真实队列表，断言 WARN 含 AD 解析 dept UUID，证明守卫在 resolveDeptFromOU 之后拦下而非上游空跑） |
| 2 | SC2：无活跃队列行时回写行为与现状一致，守卫不误伤 | ✓ VERIFIED | 同一唯一 `Model(user).Update("dept_id", ...)` 调用（grep 计数 = 1，无平行分支）；`_NoQueueUpdatesDept` / `_TerminalRowsDoNotGuard` / `_DisableActionDoesNotGuard` PASS，dept 被改为映射部门 ID（正向用例锚定 fixture 真实走进守卫分支） |
| 3 | SC3：v_dept_ou_mismatch 普通 SQL VIEW 可查询四类不一致（含映射指向软删部门） | ✓ VERIFIED | `migration_217` 四分支 UNION ALL + 11 列 D-11 契约（源码逐列读核）；sqlite 行为测试 6/6 PASS（幂等建视图、四类各恰一行、②③ user 三列 NULL、归一化等价 OU 不误报、列顺序 PRAGMA 逐项、空 base_dn 不参与路由）。PG 方言运行时未演练 → 人工项 2 |
| 4 | SC4：只读端点返回对账数据且需登录鉴权 | ✓ VERIFIED | 端点挂 `r.Group("/ad-domain")`（`router.go:519-520` JWTAuth）下 `/reconciliation` 子组（`ad_domain_router.go:148-151` RequirePermissions）；handler 测试 7/7 PASS（400/白名单/分页归一/钳 200/错误/形状/空 `[]`）；服务层在真实视图上 7/7 PASS。真实 403 运行期 → 人工项 3 |
| 5 | WARN 含全量诊断字段（userID/username/本地 dept/AD dept；命中另含队列行 id/action_type/attempts） | ✓ VERIFIED | `user_sync_service.go:435-439` 逐字段读核；测试断言 buffer 含全部字段（含 `deptValue` nil → `<nil>` 防指针地址） |
| 6 | 谓词单一事实源直引，未复制/改写/加表名限定 | ✓ VERIFIED | `user_sync_service.go:831` 直接 `.Where(models.ADPushQueueActivePredicate)`，grep 计数 = 1；常量原文 `status IN ('pending', 'failed')`（`ad_push_queue.go:53`）与 migration_215 部分唯一索引同源 |
| 7 | 守卫查询失败保守降级：跳过覆盖 + WARN，错误不上抛不阻断登录 | ✓ VERIFIED | `user_sync_service.go:434-436`；`_QueueQueryErrorSkipsDeptAndWarns`（DROP 表后）PASS，dept 不变且 buffer 含『守卫查询失败』 |
| 8 | AutoMigrate 之前双方言前置 DROP v_dept_ou_mismatch | ✓ VERIFIED | PG：`database.go:478` 位于 `dropDependentMaterializedViews()`，该函数在 `AutoMigrate()`（:793）内 :825 先行调用；sqlite：`sqlite_reconciliation_views.go:128`；且未误加进只重建三视图的 `ensureSQLiteReconciliationViews` |
| 9 | Migrate217 / Migrate218 各三条启动路径注册 | ✓ VERIFIED | `database.go:1080/1130/1329`（217）与 `:1086/1135/1335`（218）—— PG 分支 / sqlite 分支 / BootstrapMissingTables 旁路 |
| 10 | Go MatchADConfigByDN 与 SQL config_unrouted 七组边界结论一致 | ✓ VERIFIED | `TestPhase59_MatchConsistency` PASS（最长后缀父子域/逗号边界 xcorp/无匹配/空 base/停用/软删/等长歧义 `cnt=1` 全部子用例） |
| 11 | migration_218 F 型权限行 + 精准授权 + 父菜单缺失降级 | ✓ VERIFIED | `migration_218` 源码读核（幂等复用既有行、`INSERT...SELECT...WHERE NOT EXISTS` 只授父菜单已关联角色、缺失时 warning+nil）；测试 4/4 PASS（含后接入角色重跑补授权） |
| 12 | 无权限时 Tab 内渲染 403 且不对账数据零请求 | ✓ VERIFIED | `ReconciliationTab.tsx:232-240` 403 Result（文案逐字）+ `loadList` 首行与 effect 双重 `canView` 短路；测试断言 `getDeptOUMismatchList` 调用次数 0。⚠ 偏差：`useDeptTree()`（:80）在权限门前无条件调用，无权限用户仍触发一次部门树请求 —— 见 WR-01（Warning，开放 disposition，非阻塞） |
| 13 | 空态标题 16px/600 经 !important 在真实浏览器生效 | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS 规则本体存在、被组件 import、包裹类 `reconciliation-tab-empty` 就位（源码核实）；jsdom 不算层叠，无计算样式测试 —— 见 behavior_unverified_items / 人工项 6 |
| 14 | Tab 栏不换行/不溢出（backstop） | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | backstop 级真话，无自动化证据即弃权（insufficient_spec）；仅 2 个 Tab 且沿用 antd 默认折叠 —— 见人工项 7 |

**Score:** 12/14 truths verified（2 present, behavior-unverified）

### Deferred Items

不适用 —— 无按 Step 9b 判定为「后续 phase 覆盖」的缺口。注意一个**非缺口的已登记边界**：视图四类口径不含「部门/用户在路由配置下无任何映射」（第五类），这是 PLAN objective 的锁定决策（只做可观测、不扩口径），ROADMAP SC3 未要求它；Phase 60-62 亦不覆盖，属显式 Known Gap 而非失败真话（59-REVIEW IN-02 另指出同族的「映射挂错配置」盲区，同为口径边界）。

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/services/system/user_sync_service.go` | 守卫（查询 + 三分支 + WARN） | ✓ VERIFIED | 847 行；`findActiveUserUpdateQueue` / `deptValue` 实装；唯一 Update 调用保留 |
| `internal/services/system/user_sync_service_phase59_test.go` | 五类守卫行为 + WARN 断言 | ✓ VERIFIED | 262 行；共用 `setupPhase59GuardFixture`（真实 mapper + 非空 ad_username）+ 6 用例 |
| `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go` | 双方言视图 DDL + 四类口径 body | ✓ VERIFIED | 295 行；4 个可单测纯函数；DDL 本体零 ILIKE/REGEXP_REPLACE/`::`/LIKE/MATERIALIZED（grep 命中均在注释文档内） |
| `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go` | sqlite 建视图/幂等/列序/四类 + PG DDL 文本守卫 | ✓ VERIFIED | 440 行；6/6 PASS |
| `internal/core/db/database.go` | 三路径注册 + PG 前置 DROP | ✓ VERIFIED | 217×3 + 218×3 + :478 DROP（AutoMigrate 前） |
| `internal/core/db/sqlite_reconciliation_views.go` | sqlite 前置 DROP 追加 | ✓ VERIFIED | :128 |
| `internal/services/addomain/dept_ou_mismatch_service.go` | 只读分页服务 | ✓ VERIFIED | 164 行；白名单/escapeLike（先转义反斜杠的正确顺序）/Count 后分页/确定性 ORDER BY/空 `[]` |
| `internal/api/v1/system/ad_reconciliation_handler.go` | 只读 handler | ✓ VERIFIED | 71 行；绑定→白名单→NormalizePagination→response.Page；operlog 调用 = 0（grep 实测） |
| `internal/api/v1/system/ad_domain_router.go` | reconciliation 路由组 + 独立鉴权 | ✓ VERIFIED | :148-152；perms 字面恰 1 处；`TestSetupRouters_AllRegisterRoutes` PASS |
| `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go` | 权限载体 + 精准授权 | ✓ VERIFIED | 97 行；测试 4/4 PASS |
| `ruoyi-react-frontend/src/lib/adDomainApi.ts` | getDeptOUMismatchList + 类型 | ✓ VERIFIED | :835-838 POST `/ad-domain/reconciliation/mismatch` + withDefaultPagination |
| `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx` | 对账 Tab | ✓ VERIFIED | 318 行；权限门/筛选显式提交/7 列/rowKey 组合/状态面全实装；0 hex、0 axios |
| `ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx` | 外层 Tabs + activeTab 持久化 | ✓ VERIFIED | key configs/reconciliation（:342/:524）；`= usePersistedStateController` 恰 1 处 |
| `ReconciliationTab.css` | 空态标题字重覆盖（无 hex） | ✓ VERIFIED | 18 行；仅 font-weight: 600 !important，零色值 |
| 三个前端测试文件 | 契约断言 | ✓ VERIFIED | 3 文件 38 用例全 PASS（本验证实跑） |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `user_sync_service.go` | `models/ad_push_queue.go` | `models.ADPushQueueActivePredicate` + 枚举直引 | ✓ WIRED | grep 计数 1；`ADPushEntityUser`/`ADPushActionUpdateUser` 复用 |
| `user_sync_service_phase59_test.go` | `migration_215_create_ad_push_queue.go` | `Migrate215CreateADPushQueue(db)` | ✓ WIRED | 测试建真实队列表与部分唯一索引，无第二份 DDL |
| `user_sync_service.go` | `addomain/dept_ou_mapper.go` | 守卫插在 `resolveDeptFromOU` 成功之后 | ✓ WIRED | :421 → :432；WARN 断言 AD 解析 dept UUID 证明时序 |
| `migration_217` | 四源表 | 四表 JOIN body | ✓ WIRED | sqlite 真实建表 + 查询行为测试 PASS |
| `database.go` | migration_217/218 | 三路径注册 | ✓ WIRED | grep 各 3 |
| `migration_217` | `ad_config_resolver.go` | config_unrouted 复刻 MatchADConfigByDN 语义 | ✓ WIRED | `TestPhase59_MatchConsistency` 七组双向断言 PASS |
| `dept_ou_mismatch_service.go` | `v_dept_ou_mismatch` | `Table("v_dept_ou_mismatch")` + 参数化 Where | ✓ WIRED | 真实视图上 7/7 测试 PASS；DTO 逐列显式 gorm column 绑定 |
| `ad_domain_router.go` | `ad_reconciliation_handler.go` | `r.Group("/reconciliation")` + RequirePermissions + POST /mismatch | ✓ WIRED | 路由注册测试 PASS |
| `migration_218` | `sys_menu` / `sys_role_menu` | F 行 + NOT EXISTS 精准授权 | ✓ WIRED | 4 个 sqlite 行为测试 PASS |
| `ReconciliationTab.tsx` | `adDomainApi.ts` | `getDeptOUMismatchList` | ✓ WIRED | vitest 断言路径与默认分页 PASS |
| `ReconciliationTab.tsx` | `menuStore.ts` | `permissions.includes(RECONCILIATION_PERM)` | ✓ WIRED | `useMenuStore.setState` 驱动两态测试 PASS |
| `configs/index.tsx` | `ReconciliationTab.tsx` | Tabs 第二 item children | ✓ WIRED | 渲染测试断言两 Tab 标签与持久化 PASS |
| 登录链路 | 守卫 | `ad_authenticator.go:131 → SyncADUser` | ✓ WIRED | 守卫在真实登录路径上，非孤儿代码 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| ReconciliationTab 表格 | `list` | `getDeptOUMismatchList` → 后端 service → `Table("v_dept_ou_mismatch")` 视图 | 是（视图实时计算四源表） | ✓ FLOWING |
| 对账端点 | `list`/`total` | `v_dept_ou_mismatch` 查询（Count + Scan） | 是 | ✓ FLOWING |
| 守卫 WARN | `qrow` | `ad_push_queue` 真实查询（Take） | 是（测试日志可见真实行 ID/attempts） | ✓ FLOWING |
| 空态/错误态 | `list.length` / `error` | 同一请求结果集分支 | 是 | ✓ FLOWING |
| 无孤儿静态回退 | — | 全链路无 hardcoded 数据、无 mock 泄入生产路径 | — | ✓（mock 仅存在于测试文件） |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| GUARD-01 六分支守卫行为 | `go test ./internal/services/system/ -run 'TestPhase59_SyncADUser_' -count=1 -v` | 6/6 PASS | ✓ PASS |
| migration_217 视图（建/幂等/四类/列序/归一化/PG DDL 文本） | `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1 -v` | 6/6 PASS | ✓ PASS |
| migration_218 权限载体（幂等/精准授权/降级/补授权） | `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration218' -count=1 -v` | 4/4 PASS | ✓ PASS |
| 对账服务分页/筛选/LIKE 转义/Go-SQL 一致性 | `go test ./internal/services/addomain/ -run 'TestPhase59_DeptOUMismatch\|TestPhase59_MatchConsistency' -count=1 -v` | 7 用例 + 7 一致性子用例全 PASS | ✓ PASS |
| handler 绑定/白名单/分页/形状 + 全量路由注册 | `go test ./internal/api/v1/system/ -run 'TestPhase59_ReconHandler\|TestSetupRouters_AllRegisterRoutes' -count=1 -v` | 7/7 + 1/1 PASS | ✓ PASS |
| 前端 API/组件/页面渲染契约 | `npx vitest run src/lib/adDomainApi.test.ts src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx src/pages/ad-domain/configs/__tests__/index.render.test.tsx` | 3 文件 38/38 PASS | ✓ PASS |

### Probe Execution

SKIPPED — 本 phase 无 `scripts/*/tests/probe-*.sh` 声明或约定路径（PLAN/SUMMARY/VALIDATION 均未声明 probe）；行为证据由上表命名测试运行提供（守卫/迁移/服务/端点/前端五层全跑）。

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| GUARD-01 | 59-01 | AD 登录写 dept 前，pending/failed 队列行 → 跳过覆盖 + WARN（本地编辑不被 AD 旧值回滚） | ✓ SATISFIED | 守卫实装于登录链路唯一回写点 + 6/6 行为测试；REQUIREMENTS.md traceability 标记 Complete（一致） |
| GUARD-02 | 59-02 / 59-03 / 59-04 | v_dept_ou_mismatch 普通 SQL VIEW + 只读端点，展示不一致及「映射指向软删部门」 | ✓ SATISFIED | 视图（sqlite 行为证明）+ JWT/权限端点 + migration_218 载体 + 前端 Tab 全链路；REQUIREMENTS.md traceability 标记 Complete（一致） |

无孤儿需求：REQUIREMENTS.md 将且仅将 GUARD-01/GUARD-02 映射到 Phase 59，两 ID 均被 PLAN frontmatter 声明并被上表覆盖。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `ReconciliationTab.tsx` | :80 | WR-01：`useDeptTree()` 在权限早退前无条件调用，无权限仍发部门树请求，违反「不发起任何请求」契约字面（测试 mock 掉 useDeptTree 形成断言盲区） | ⚠️ Warning | 部门树端点仅需登录态、无敏感数据、有 5min staleTime 共享缓存；不触及 phase 目标。59-REVIEW-DISPOSITION 开放项，非阻塞 |
| `ReconciliationTab.tsx` | :96-119 | WR-02：loadList 无过期响应守卫，Enter/分页连点可旧响应覆盖新响应 | ⚠️ Warning | UX 竞态，非数据正确性或安全；59-REVIEW-DISPOSITION 开放项，非阻塞 |
| `user_sync_service.go` | :647-666 | IN-01：批量同步路径 dept_id 回写无守卫（与登录路径不对称） | ℹ️ Info | 守卫范围 = 登录路径是 PLAN 锁定范围（D-04）；建议补决策注释防误判为 bug |
| `migration_217_...go` | :198-207 | IN-02：分支①跨配置映射盲区（同 Known Gap 族） | ℹ️ Info | 口径边界，非缺陷；建议登记 Known Gap 清单 |
| `database.go` | :496 | IN-04：DROP 视图后 noop 收尾日志与行为矛盾 | ℹ️ Info | 误导性日志，建议改写 |
| `migration_218_...go` | :66 | IN-05：VisibleHidden 零值遇 default tag 落库为 1（migration_163 同款） | ℹ️ Info | F 型按钮可见性不参与渲染判定，无功能影响 |
| 全部 20 个实现文件 | — | TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER 债务标记 | 无 | grep 扫描 0 命中；无未决债务标记（Debt marker gate 未触发） |

**回归说明：** `TestSNMPClient_WaitForReady_Timeout`（internal/device）偶发失败为已知既有 flake —— 该包最后由 Phase 59 之前数周的 acd4ed6f 触碰、与 Phase 59 任何文件零依赖，本次验证触达的四个 Go 包与前端均无回归。不归因 Phase 59。

### Human Verification Required

7 项（见 frontmatter `human_verification`）：
1. 真实 AD 登录守卫拦截（GUARD-01 端到端）
2. 真实 PG 启动冒烟：migration_217 视图 + migration_218 seed + 二次重启幂等（风险最高项 —— 本仓库有视图阻塞 AutoMigrate 启动 FATA 前科，PG 分支运行期完全未演练）
3. 对账 Tab 浏览器权限渲染（授权/未授权 + 后端真实 403）
4. 生产库父菜单『AD域控配置』C 型存在性（缺失则需管理员手建权限载体）
5. GUARD-01/GUARD-02 验收口径人工复核（四个 PLAN 的 FLAGGED ASSUMPTION 闭环）
6. 空态标题字重 600 computed style
7. Tab 栏不溢出视觉回归（backstop）

### Gaps Summary

无 gaps。四条 ROADMAP Success Criteria 在代码库全部成立且有三层以上证据（存在、实质、接线、数据流）与通过中的行为测试背书；全部 20 个工件实装无 stub；13 条关键链路全 WIRED；10 项禁令（谓词不可改写、无平行分支、无时间戳仲裁、禁 MV、方言红线、无 operlog、精准授权、无 hex/axios、显式提交、无 UUID 列）逐条 grep/读核通过；零新依赖（go.mod / package.json 在阶段窗口零变更）；10 个提交哈希全部存在且消息吻合；17 阶段提交 / 30 文件 / +4384-58。

状态为 human_needed 的唯一原因：Step 8 产出 7 项人工验证 —— 其中 2 项是 ⚠️ PRESENT_BEHAVIOR_UNVERIFIED 真话（jsdom/无 PG 实例的物理约束），其余为环境级（真实 AD、真实 PG、浏览器权限、生产数据形态）与 FLAGGED ASSUMPTION 闭环复核。这些是 phase 自身 VALIDATION 策略预先声明的 Manual-Only 边界，不是执行缺陷。

---

_Verified: 2026-10-19T16:35:48Z_
_Verifier: Claude (gsd-verifier)_
