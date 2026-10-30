---
phase: "87"
slug: "permission-grant"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-30"
---

# Phase 87 — Validation Strategy

> 本 phase 的验证契约（执行期反馈采样）。来源：`87-RESEARCH.md` 的 `## Validation Architecture` 段。
> **Task ID 在 PLAN.md 定稿后回填**；当前 Per-Task Verification Map 用**需求级锚点**预置。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/glebarez/sqlite`（**内存库 + 原生 DDL**，仿 `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go:23-70`）｜前端 vitest + @testing-library/react（仿 `src/pages/system/role/__tests__/permission.gate.test.tsx`）｜Node 脚本 `node scripts/check-permission-codes.mjs` |
| **Config file** | 无新增 —— 沿用 `internal/core/db/migrations/*_test.go` 与 `ruoyi-react-frontend/src/pages/system/role/__tests__/` 既有基建 |
| **Quick run command** | `go test ./internal/core/db/migrations/ ./pkg/permission/ ./internal/services/system/` ＋ `cd ruoyi-react-frontend && node scripts/check-permission-codes.mjs && npx vitest run src/pages/system/role/` |
| **Full suite command** | `go build ./... && go test ./...` ＋ `cd ruoyi-react-frontend && npm run lint:ci && npm run test` |
| **Estimated runtime** | quick ~40s / full ~3min（Go）+ ~5min（前端） |

---

## Sampling Rate

- **After every task commit:** quick run command
- **After every plan wave:** full suite command
- **Before `/gsd:verify-work`:** Full suite must be green（含 `npm run lint:ci` exit 0 —— 87 之后 `check-permission-codes.mjs` 在链内）
- **Max feedback latency:** 60s（quick）/ 240s（full）

---

## Per-Task Verification Map

> Task ID 在 PLAN.md 定稿后回填；此处预置**需求级**验证锚点。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {87-NN-01} | 87-02 | 1 | RBAC-05 | — | 72 条 F 型码在 `sys_menu` 各有且仅有 1 行（`menu_type='F'`、`status=0`、`visible=0`、`perms` 单码） | unit（sqlite 迁移测试） | `go test -run TestMigrate226 ./internal/core/db/migrations/` | ⬜ W0 | ⬜ pending |
| {87-NN-02} | 87-02 | 1 | RBAC-05 | — | 迁移**幂等**：连续执行 2 次 ⇒ `sys_menu` 行数不变、`sys_role_menu` 行数不变 | unit（红绿：回退幂等即失败） | `go test -run "TestMigrate226.*Idempotent" ./internal/core/db/migrations/` | ⬜ W0 | ⬜ pending |
| {87-NN-03} | 87-02 | 1 | RBAC-05 | — | **精准授权**：仅持有父 C 菜单的角色获得新 F 行；不持有者不获得（**停用旧菜单不得命中**） | unit | `go test -run "TestMigrate226.*Grant" ./internal/core/db/migrations/` | ⬜ W0 | ⬜ pending |
| {87-NN-04} | 87-02 | 1 | RBAC-05 | — | **父菜单缺失降级**：返回 nil、不 panic、不新增任何行 | unit | `go test -run "TestMigrate226.*MissingParent" ./internal/core/db/migrations/` | ⬜ W0 | ⬜ pending |
| {87-NN-05} | 87-02 | 1 | RBAC-05 | — | **父集合多值**：命中多条同名父时，持有任一父的角色都获得授权 | unit | `go test -run "TestMigrate226.*MultiParent" ./internal/core/db/migrations/` | ⬜ W0 | ⬜ pending |
| {87-NN-06} | 87-03 | 2 | RBAC-05 | — | **角色页可勾选 F 型**：F 节点出现在树中 → 勾选 → 提交的 `menuIds` 含该 F 节点 id | component（vitest） | `cd ruoyi-react-frontend && npx vitest run src/pages/system/role/__tests__/permission.grant.test.tsx` | ⬜ W0 | ⬜ pending |
| {87-NN-07} | 87-03 | 2 | RBAC-05 | — | **同源**：同一权限码同时驱动前端可见性与后端放行 | unit + component | 上面两条用**同一串码**（取自 `PERMISSION_CODES` / `pkg/permission/config.go` 常量） | ⬜ W0 | ⬜ pending |
| {87-NN-08} | 87-04 | 3 | RBAC-06 | — | **三方零漂移**（TS ↔ Go 常量 ↔ `sys_menu` F 型）：脚本 exit 0 | script | `cd ruoyi-react-frontend && node scripts/check-permission-codes.mjs` | ⬜ W0 | ⬜ pending |
| {87-NN-09} | 87-04 | 3 | RBAC-06 | — | **白名单已清空且被硬断言守护**（非空 ⇒ exit 1） | script（红绿） | 同上；反向：临时加 1 条白名单 → 断言 exit 1 | ⬜ W0 | ⬜ pending |
| {87-NN-10} | 87-04 | 3 | RBAC-06 | — | **Go 侧 F 型识别生效**：d 类对 Go 侧 F 行生效（把某码从 TS 移除 → 断言红） | script（红绿） | 同上 | ⬜ W0 | ⬜ pending |
| {87-NN-11} | 87-04 | 3 | RBAC-06 | — | **接入 CI 门禁**：`lint:ci` 含 `check-permission-codes.mjs` 且整体 exit 0；`.github/workflows/frontend-build.yml` 跑 `lint:ci` | script | `cd ruoyi-react-frontend && npm run lint:ci` | ⬜ W0 | ⬜ pending |
| {87-NN-12} | 87-04 | 3 | RBAC-06 | — | **路由登记不回归** | go test + script | `go test -run "TestRouteRegistry" ./pkg/permission/` ＋ `node scripts/check-route-registry.mjs` | ✅ 既有 | ⬜ pending |
| {87-NN-13} | 87-04 | 3 | 回归 | — | v1.24 三态门控契约零破坏（ready 空集 ≠ 未就绪） | component | `npx vitest run src/pages/system/role/__tests__/permission.gate.test.tsx`（**不得改动该文件**） | ✅ 既有 | ⬜ pending |
| {87-NN-14} | 87-04 | 3 | 回归 | — | v1.25 回归基线（eslint 0 / vitest 全绿 / go build+test 全绿） | 全量 | `npm run lint:ci && npm run test`；`go build ./... && go test ./...` | ✅ 既有 | ⬜ pending |
| {87-NN-15} | 87-04 | 3 | 回归 | — | **86 线恒等式零破坏**：`25 + 51 = 76` | script | `cd ruoyi-react-frontend && npm run check:gating` | ✅ 既有 | ⬜ pending |
| {87-NN-16} | 87-04 | 3 | SC3 | — | **生产既有行零改动**（播种只新增行） | 只读 SQL（人工） | 见 Manual-Only 表 | — | ⬜ pending |

---

## Wave 0 Requirements

- [ ] `internal/core/db/migrations/migration_226_action_permission_f_type_seed.go` —— 新 migration（双方言、幂等、精准授权、72 条种子）
- [ ] `internal/core/db/migrations/migration_226_action_permission_f_type_seed_test.go` —— sqlite 原生 DDL 测试（幂等 / 精准授权 / 父缺失降级 / 多父集合 / 停用旧菜单不得命中 / 既有行零改动）
- [ ] `internal/core/db/migrations/registry.go` —— `ActiveMigrations` 追加 `{226, "action_permission_f_type_seed"}`
- [ ] `internal/core/db/database.go` —— PG 分支 + sqlite 分支各加一条 `d.runAndRecord(226, ...)`
- [ ] `ruoyi-react-frontend/scripts/check-permission-codes.mjs` —— 清空白名单 + 加「Go 侧 F 型声明块」解析 + 加硬断言
- [ ] `ruoyi-react-frontend/package.json` —— `lint:ci` 插入 `node scripts/check-permission-codes.mjs`
- [ ] `.github/workflows/frontend-build.yml` —— `npm run lint` → `npm run lint:ci`
- [ ] `scripts/check-ci-local.sh` —— frontend 段加 `lint:ci`（保持「本地绿 = CI 绿」）
- [ ] `ruoyi-react-frontend/src/pages/system/role/__tests__/permission.grant.test.tsx` —— 角色授权闭环组件测试（新建）

*其余沿用各包既有测试基建；无框架安装需求（**零新依赖**）。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| **生产受影响角色清单**（播种后哪些角色新增写能力） | RBAC-05 / SC5 | 需连生产库只读查询；口令不在环境内 | 跑 `87-RESEARCH.md` §E19 的判定 SQL → 结果为空或逐条列明，写进 `87-VERIFICATION.md` |
| **生产库既有行零改动核对** | SC3 | 需生产库行数快照 | 播种前后各跑一次：`SELECT (SELECT count(*) FROM sys_menu), (SELECT count(*) FROM sys_role_menu), (SELECT count(*) FROM sys_role), (SELECT count(*) FROM sys_user)`；差值 = 72 + N + 0 + 0 |
| **DB 层前后对照（生产）** | RBAC-05 | 需生产库写权限 | 用 `87-RESEARCH.md` §E1 note §3.8.2 / §3.8.3 的只读 SQL 模板，对夹具角色跑 `step1_exact` / `strict_exact_pass` ⇒ 403 判定（**全部只 SELECT**，用 `SET TRANSACTION READ ONLY` 兜底） |
| **HTTP 实测 403→200** | RBAC-05 | 本机无 PG/口令；生产后端进程未运行（nginx 502）；AD + 滑块阻断登录 | ① 首选 **note §3.8 只读 SQL 判据**（零副作用）；② 若用户提供生产口令 ⇒ 按 note §3.4.2 起本地后端（`SERVER_MODE=test` + `SKIP_AUTOMIGRATE=true` + `SERVER_SKIP_SETUP=true` + `DB_PASSWORD` + `JWT_SM2_PRIVATE_KEY`/`JWT_SM2_PUBLIC_KEY`（issuer=`RuoYi-Next`，**不是** `RUOYI_JWT_SM2_*`））+ 自签 token + curl；**副作用须如实登记**（`sys_oper_log` / `ad_push_queue` / 物化视图 / OUI 导入） |
| **决定性对照（临时加权限点 → 403 转 200）** | SC3 | 同上 | 夹具 role_key 用 `p87_*` 前缀（绝不撞 `admin` 的 bypass 键 `role_key='admin'`）；补授 F 型 `menu_id` → **等 31s**（权限缓存 30s）或重启后端进程 → 同端点 200 且真生效 → 撤销 → 回 403 |
| **前端 UI 实机**（角色页勾选后按钮出现） | RBAC-05 | 需登录态（AD + 滑块），本机不可用 | 组件测试 T-1~T-6 覆盖；UI 实机留待用户执行。**必须先清 sessionStorage 的 `menu:cache` 或重新登录**（前端 5min 内存 + 30min sessionStorage 缓存，`MENU_CACHE_VERSION` 未变不会自动失效） |
| **迁移在生产 PG 上的实际执行** | RBAC-05 | 需生产部署 | 部署后查 `SELECT count(*) FROM sys_menu WHERE menu_type='F' AND perms IN (<72 码>)` ⇒ = 72（或 72 + migration_221 的既有 manager F 行） |
| **`sys_role_menu` 在生产 PG 是否有 `(role_id, menu_id)` 唯一约束** | 幂等前提 | 需 `\d sys_role_menu` 实查（OQ-1） | `psql -c "\d sys_role_menu"`；**无论结果如何，87 一律用 `NOT EXISTS` 写法，不依赖唯一约束** |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s (quick) / 240s (full)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

## Lead 裁定的 Open Questions（2026-10-30）

| # | 疑点 | 裁定 | 依据 |
|---|---|---|---|
| **OQ-1** | 生产 PG 的 `sys_role_menu` 是否有 `(role_id, menu_id)` 唯一约束 | **一律用 `NOT EXISTS` 写法，禁用 `ON CONFLICT`** | 唯一约束未实证（模型只有普通索引）；`NOT EXISTS` 不依赖约束即幂等，且双方言可测 |
| **OQ-2** | 6 条 `ops:manager:*` 是否一并播种 | **不播种**（`migration_221` 运行期已建 4 个 F 行 + 2 个 C 型菜单），但**白名单这 6 项照常清空** | 避免同 perms 重复行；`migration_221` 的 Go 字面量已让 c 类判定通过 |
| **OQ-3** | `ops:building:import/export` 是否也给一份挂「楼宇空间」 | **不重复挂**（父 = 楼宇管理） | 后端 `POST /ops/building/import` 属楼宇 CRUD 域 |
| **OQ-4** | `system:role:export` 无后端路由，是否播种 | **仍然播种** | 白名单是权威清单，清空是硬要求；`87-VERIFICATION.md` 标注「无路由，仅作授权来源占位」 |
| **OQ-5** | 前端缓存导致授权后 UI 不刷新 | **已定**：验收清单显式写「清 sessionStorage `menu:cache` 或重新登录后再断言」 | `menuStore` 的 `MENU_CACHE_VERSION = 1` 未变 ⇒ 缓存不自动失效 |
| **OQ-7** | `ops:floor-plan:list/query/save` + `ops:walls:list` + `ops:doors:list` 五码无菜单行承载 ⇒ 读路由对非 admin 恒 403 | **本 phase 不处理，独立立项** | 不在白名单、不在 TS；属 85 范围外的后端出码议题 |
| **OQ-6** | `ops:walls:*`(3) / `ops:doors:*`(3) / `ops:floor-plan:edit`(1) 共 **7 条**的父菜单（推断「楼层管理」） | **✅ 已裁定（用户，2026-10-30）：先播其余 65 条，这 7 条挂起待确认** | 这 7 条**在播种清单内**，父错了会导致树展示错位 + Step 2 继承错位；挂起优于猜错 |

> **OQ-6 的影响面与处置（精确）**：白名单 78 条 = 22 组，其中 `ops:walls`(add/edit/delete) + `ops:doors`(add/edit/delete) + `ops:floor-plan`(edit) = **7 条**落在待播种集合内。
> **用户裁定的处置**：PLAN 把这 7 条**单列为一个「挂起待确认」task**（不阻塞其余 65 条播种）；待用户实地确认 CAD 平面图编辑器 / 墙体 / 门的实际管理入口页后，再补一条 migration 补播这 7 条。
> ⇒ **播种拆分口径**：
> - **第一批（65 条）**：system 4 组 23 条（role 6 / menu 5 / dept 7 / post 5）+ ops 10 组 42 条（building / floor / workstation / asset / infopoint / serverroom / dedicatedline / roomdevice 各 2 = 16；network 6 组 26 条：device 6 / credential 6 / template 6 / command 2 / backup 3 / discovery 3）—— **父菜单已确定**
> - **第二批（7 条，挂起）**：`ops:walls` 3 + `ops:doors` 3 + `ops:floor-plan:edit` 1 —— **父菜单待确认**
> - **第三批（6 条，不播）**：`ops:manager:rule:{list,edit,delete}` + `ops:manager:org:{list,edit,delete}` —— `migration_221` 已覆盖（OQ-2）
> - **合计 65 + 7 + 6 = 78** ✅ 与白名单总数对齐
>
> ⇒ **对 RBAC-06 的影响**：白名单**无法一次全清**（挂起的 7 条仍需留在白名单里，直到第二批补播完成）。
> ⇒ **硬断言设计随之调整**：白名单硬断言只能是「**除挂起的 7 条外，其余清空**」，而不是「全部清空」；且挂起集合必须**显式声明在脚本里**（附 OQ-6 引用），否则 c 类会把 7 条判成漂移并 exit 1。
> ⇒ **PLAN 必须包含这一条**，否则 executor 会撞上「白名单清不空 → 脚本 exit 1 → CI 门禁红」。
