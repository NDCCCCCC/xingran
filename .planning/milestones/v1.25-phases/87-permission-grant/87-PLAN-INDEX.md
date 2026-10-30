---
phase: "87-permission-grant"
slug: "permission-grant"
created: "2026-10-30"
plans: 5
---

# Phase 87 计划索引 —— 动作级权限授予闭环

> 目标（ROADMAP 原文）：**动作级 F 型菜单播种 + 三层证据验收（DB 前后对照 / HTTP 实测 / 组件测试）+ 决定性对照实验**
> Requirements：**RBAC-05**（动作级权限可被真实授予）· **RBAC-06**（三方零漂移 + 脚本接入 CI 门禁）

---

## 一、波次结构

| Plan | 波 | 依赖 | 覆盖 requirement | 一句话内容 | 主要文件 |
|---|---:|---|---|---|---|
| **87-01** | 1 | — | RBAC-05 / RBAC-06 | **W0 底座（tracer）**：migration 226 骨架 + sqlite 测试基建 + 脚本 F 块解析骨架；1 条试点码打通端到端 | `migration_226_*.go` / `migration_226_*_test.go` / `registry.go` / `database.go` / `check-permission-codes.mjs` |
| **87-02** | 2 | 87-01 | RBAC-05 | **播种第一批 65 条** + 精准授权 + 幂等 + 父缺失降级 + 多父集合 + 87-STATE 落盘 | `migration_226_*.go` / `migration_226_*_test.go` / `87-STATE.md` |
| **87-03** | 3 | 87-02 | RBAC-06 | **校验脚本改造 + CI 门禁接入**（挂起集合机制 + 白名单硬断言 + `lint:ci` + workflow + 本地复刻） | `check-permission-codes.mjs` / `package.json` / `frontend-build.yml` / `check-ci-local.sh` / `87-CI-GATE.md` |
| **87-04** | 3 | 87-02 | RBAC-05 / RBAC-06 | **授权闭环组件测试 + 三层证据 + 决定性对照实验**（只读 SQL 口径） | `permission.grant.test.tsx` / `87-EVIDENCE.md` |
| **87-05** | 4 | 87-03, 87-04 | RBAC-05 / RBAC-06 | **第二批 7 条补播 + 白名单归零**（OQ-6 用户确认后；**默认挂起**） | `migration_227_*.go` / `registry.go` / `database.go` / `check-permission-codes.mjs` / `87-STATE.md` |

**并行性**：wave 3 的 87-03 与 87-04 **文件零重叠** ⇒ 可并行。wave 4 的 87-05 因同时改 `check-permission-codes.mjs`（与 87-03 重叠）、`registry.go` / `database.go`（与 87-01 重叠）、`87-STATE.md`（与 87-02 重叠）⇒ 强制串行。

---

## 二、播种口径（78 = 65 + 7 + 6，写死不可改）

| 批次 | 条数 | 内容 | 处置 | 落点 |
|---|---:|---|---|---|
| **第一批** | **65** | system 4 组 23（role 6 / menu 5 / dept 7 / post 5）+ ops 8 组 16（building / floor / workstation / asset / infopoint / serverroom / dedicatedline / roomdevice 各 2）+ network 6 组 26（device 6 / credential 6 / template 6 / command 2 / backup 3 / discovery 3） | **87-02 播**（migration 226） | `87-STATE.md` 父解析表 |
| **第二批** | **7** | `ops:walls:{add,edit,delete}` + `ops:doors:{add,edit,delete}` + `ops:floor-plan:edit` | **OQ-6 待用户确认父菜单** ⇒ 87-05 补播（migration 227） | 87-05 的阻断 checkpoint |
| **第三批** | **6** | `ops:manager:rule:{list,edit,delete}` + `ops:manager:org:{list,edit,delete}` | **不播**（`migration_221` 运行期已建 4 个 F 行 + 2 个 C 型菜单，OQ-2）；白名单**照常清空** | `87-STATE.md` 批次三登记 |
| **合计** | **78** | 与 `PENDING_SEED_WHITELIST` 原条数对齐 ✅ | | |

---

## 三、16 条验证锚点 → plan 覆盖表

| 锚点 | 覆盖 plan | 覆盖 task | 自动化命令（取自 `87-VALIDATION.md`） |
|---|---|---|---|
| 87-NN-01 全量 F 行 | 87-01（tracer 形态）/ **87-02**（全量 65 条形态） | 1.2 / 2.2 | `go test -run TestMigrate226 ./internal/core/db/migrations/` |
| 87-NN-02 幂等 | 87-01 / **87-02** | 1.2 / 2.2 | `go test -run "TestMigrate226.*Idempotent" ./internal/core/db/migrations/` |
| 87-NN-03 精准授权（停用旧菜单不得命中） | 87-01 / **87-02** | 1.2 / 2.2 | `go test -run "TestMigrate226.*Grant" ./internal/core/db/migrations/` |
| 87-NN-04 父缺失降级 | 87-01 / **87-02** | 1.2 / 2.2 | `go test -run "TestMigrate226.*MissingParent" ./internal/core/db/migrations/` |
| 87-NN-05 多父集合 | **87-02** | 2.2 | `go test -run "TestMigrate226.*MultiParent" ./internal/core/db/migrations/` |
| 87-NN-06 角色页可勾选 F 型 | **87-04** | 4.1 | `cd ruoyi-react-frontend && npx vitest run src/pages/system/role/__tests__/permission.grant.test.tsx` |
| 87-NN-07 同源 | **87-04** | 4.1 | 同上（同一串码断言） |
| 87-NN-08 三方零漂移 | **87-03** | 3.1 / 3.3 | `cd ruoyi-react-frontend && node scripts/check-permission-codes.mjs` |
| 87-NN-09 白名单硬断言 | **87-03** | 3.3 | 同上（红绿自检：塞 1 条 ⇒ exit 1） |
| 87-NN-10 Go 侧 F 型识别生效 | **87-03** | 3.3 | 同上（红绿自检：TS 少 1 码 ⇒ 红） |
| 87-NN-11 接入 CI 门禁 | **87-03** | 3.2 | `cd ruoyi-react-frontend && npm run lint:ci` |
| 87-NN-12 路由登记不回归 | **87-03** | 3.3 | `go test -run "TestRouteRegistry" ./pkg/permission/` + `node scripts/check-route-registry.mjs` |
| 87-NN-13 v1.24 三态契约零破坏 | **87-04** | 4.1 | `npx vitest run src/pages/system/role/__tests__/permission.gate.test.tsx`（**不得改动该文件**） |
| 87-NN-14 v1.25 回归基线 | **87-03** / **87-04** | 3.3 / 4.3 | `npm run lint:ci && npm run test`；`go build ./... && go test ./...` |
| 87-NN-15 86 线恒等式零破坏 | **87-03** / **87-04** | 3.3 / 4.3 | `cd ruoyi-react-frontend && npm run check:gating` |
| 87-NN-16 生产既有行零改动 | **87-02**（自动化等价值）/ **87-04**（人工核对清单） | 2.2 / 4.3 | `go test -run "TestMigrate226.*ExistingRowsUntouched" ./internal/core/db/migrations/` + 人工只读 SQL 清单 |

---

## 四、红线（四个文件**零 diff**，87 全程不得触碰）

| 文件 | 理由 |
|---|---|
| `ruoyi-react-frontend/src/constants/permissionCodes.ts` | 87 **零新增权限码**；78 码已全部声明；单一声明真相源 |
| `ruoyi-react-frontend/scripts/gating-exemptions.json` | 51 条豁免恒等式（`25 + 51 = 76`） |
| `ruoyi-react-frontend/scripts/check-gating-coverage.mjs` | `EXPECTED_PAGE_COUNT=76` / `EXPECTED_EXEMPTION_COUNT=51` 不得改动 |
| `ruoyi-react-frontend/src/pages/system/role/__tests__/permission.gate.test.tsx` | v1.24 三态门控契约回归基线（只读） |

---

## 五、待确认 / 开放遗留

| # | 项 | 状态 | 落点 |
|---|---|---|---|
| **OQ-6** | 批次二 7 条的**父菜单**（推断「楼层管理」，未确认） | **阻断 87-05**；87-01..87-04 不受影响 | 87-05 的 checkpoint:human-verify + `87-STATE.md` |
| **OQ-7** | `ops:floor-plan:{list,query,save}` + `ops:walls:list` + `ops:doors:list` 五码不在 TS 也不在白名单，却被读组 ANY-of 引用 ⇒ **读路由对非 admin 恒 403** | **87 不处理**，登记为**独立后续项** | `87-STATE.md` + `87-EVIDENCE.md`（双处登记） |
| **OQ-1** | 生产 PG `sys_role_menu` 是否真有 `(role_id, menu_id)` 唯一约束 | 不影响实现（一律 `NOT EXISTS`）；复核结果待人工落盘 | `87-EVIDENCE.md` 人工清单 |
| **OQ-2 / OQ-3 / OQ-4** | 批次三 6 条不播 / `ops:building:import|export` 不重复挂 / `system:role:export` 仍播种 | **已裁定**（用户 / Lead） | `87-STATE.md` |
| **86 线遗留** | 51 条豁免（87 消解 **0** 条）+ 6 处旧式直判（87 消解不了） | **仍开放** | `87-STATE.md` |
| **白名单归零** | 第二批补播后白名单应归零（脚本进入规则 B 严格态） | 87-03 登记为遗留项，87-05 关闭 | `87-CI-GATE.md` + `87-STATE.md` |

---

## 六、执行期关键纪律（摘自各 plan）

1. **部署顺序**：先合 migration（数据），后合前端 / 校验脚本（消费）。87 播种前所有非 admin 角色的写操作恒 403（fail-closed 窗口期）。
2. **禁止**：`GrantNewMenuToRolesHavingParent`（PG-only + 按名字匹配 + 依赖未实证唯一约束）；`ON CONFLICT`；`fmt.Sprintf` 拼 SQL；`isPostgreSQL` 守卫；Java 风格 `frame` / `cache` 列；按 `menu_name` 单独定位父菜单。
3. **白名单**：不能一次全清 —— 挂起集合（7 条）必须**显式声明**在脚本里，否则 c 类判漂移 ⇒ 脚本 exit 1 ⇒ CI 门禁红。
4. **缓存**：后端权限缓存 30s ⇒ 补授后**等 31s 或重启**；前端 5min 内存 + 30min sessionStorage ⇒ UI 实机**清 `menu:cache` 或重新登录**。
5. **夹具**：`role_key` / `username` 一律 `p87_` 前缀、**不撞 `admin`**、验证账号**只挂**本受限角色；清理先子后父 + 自检全 0；**播种的 F 行不在清理范围内**。
6. **零新依赖**：全 phase 不新增任何 npm / Go 包。
