---
phase: "87"
slug: "permission-grant"
status: complete
created: "2026-10-30"
requirements: [RBAC-05, RBAC-06]
---

# Phase 87 研究报告：动作级权限授予闭环

> 只读研究产物，供 `gsd-planner` 直接消费。行号为 2026-10-30 工作区快照。
> 溯源标记：`[VERIFIED: 路径:行号]` = 本会话用 Read/grep 实际打开并逐字引用；`[CITED]` = 官方文档；`[ASSUMED]` = 推导未实证。

---

## User Constraints

> 用户已锁定的决策（**不可更改**，研究只回答「怎么落地」，不重新论证）：

1. **F 型动作权限的播种载体 = 写 migration**。新增 `internal/core/db/migrations/` 下的 migration，播种 F 型菜单 + 授权给已持父菜单的角色。
   - 既有范式参照：`migration_202_port_write_audit.go` / `migration_218_ad_reconciliation_permission.go` / `migration_221_manager_sync_menus.go`。
2. **不做 discuss-phase**（无 CONTEXT.md）。本研究必须足以让 planner 直接写出可执行 PLAN。
3. **要求**：动作级 F 型菜单播种 + 三层证据验收（DB 前后对照 / HTTP 实测 / 组件测试）+ 决定性对照实验（临时加权限点 → 403 转 200）。

---

## Project Constraints (from AGENTS.md)

> 来源：`AGENTS.md`（仓库根，135 行）+ `CLAUDE.md`（GSD 工作区指令已注入）。以下为**对本 phase 有约束力**的条目，逐条列出。

| # | AGENTS.md 原文约束 | 对 Phase 87 的含义 |
|---|---|---|
| P1 | `Status Values: 0 = enabled/normal/visible, 1 = disabled/stopped/hidden` + `Exception: Menu visibility uses 1 = visible, 0 = hidden` | 播种的 F 行必须 `visible=0`（隐藏）、`status=0`（正常）。`internal/models/base.go:93-94` 逐字：`VisibleShow VisibleType = 1 // 显示` / `VisibleHidden VisibleType = 0 // 隐藏` |
| P2 | `Response Wrapper: Use response.Success() and response.Error() in handlers` | 角色授权链路已有 `response.Success`，本 phase 不改 handler |
| P3 | `Handler-Service Pattern` / `Context Propagation: always pass c.Request.Context()` | migration 属 DB 层，无 handler 改动面；若加 handler 只走既有 pattern |
| P4 | `Cache Keys: Use helper functions like GetDictDataByTypeKey(), don't hardcode` | 迁移里不得硬编码缓存键（本 phase 不涉及） |
| P5 | `Pagination: current starts from 1, response uses list, total, current, pageSize` | 不涉及 |
| P6 | `Critical Rules / UUID Validation: org_id must be valid UUID format` | 播种的 `sys_menu.id` 必须是 UUID（`gen_random_uuid()` / Go `uuid.New()`） |
| P7 | `No Raw JSON: Always use response wrappers, never c.JSON()` | 不涉及 |
| P8 | `Import Path: Use full module path github.com/ruoyi-next/ruoyi-go-backend/...` | 新 migration 文件的 import 一律全路径 |
| P9 | `Database Naming: Tables: sys_ prefix, lowercase with underscores; Primary Keys: UUID with gen_random_uuid() default` | `sys_menu` / `sys_role_menu` 新行主键为 UUID |
| P10 | `Migrations: auto-run on startup, versioned in internal/core/db/migrations/`（CLAUDE.md §Database Migrations） | 新 migration 必须进 `database.go` 启动调用块才会执行 |
| P11 | `After any Go code changes, run full build check: go build ./...` | 每个 task 后跑 `go build ./...` |
| P12 | `go test ./...` must pass before commit；`Always ask user before committing` | 「不要 git commit —— 提交由 orchestrator 负责」（`.planning/STATE.md:283`） |
| P13 | `Context Propagation` + `Error Handling: Return errors, don't panic` | migration 内 panic = 启动崩溃，必须降级 |
| P14 | CLAUDE.md：`Request Encryption (SM2+SM4)`、`configurable via security.request_encryption.exclude_paths` | HTTP 实测时明文/密文取决于生产 `sys.request.encryption.enabled`（见 §E16） |
| P15 | CLAUDE.md：`AD/LDAP Connections ... ad.tls_skip_verify config key (default true for dev)` | 与本 phase 无关，但影响 §E16 的 AD 登录阻断 |
| P16 | v1.25 STATE 硬约束：`生产数据只读（验收夹具须单事务清理、既有行零改动）` | 播种只允许**新增行**，不得 UPDATE/DELETE 既有行（见 §G23） |
| P17 | v1.25 STATE 硬约束：`回归基线 —— eslint 0 problems / vitest 569 文件 3960 用例全绿 / vendor-react gzip ≤ 776 kB / go build ./... + go test 全绿` | phase 收口必须复核这四项 |
| P18 | v1.25 STATE 硬约束：`零新依赖` | 本 phase 不得引入任何 npm/go 新依赖 |
| P19 | v1.25 STATE 硬约束：`门控取三态而非纯 fail-closed` | 前端组件测试必须保留三态语义（ready 空权限集 ≠ 未就绪） |
| P20 | v1.25 STATE 硬约束：`不破坏 v1.22 冻结契约与 v1.24 已交付修复` | 角色页改动不得破坏 v1.22 契约组件 |
| P21 | AGENTS.md §Architecture Patterns：`Response Wrapper` + `operlog.Record(...)` 约定（CLAUDE.md 操作日志段） | 角色授权是**已有**端点（`POST /system/roles/:id/update`），本 phase 若新增端点才需 operlog |

---

## Standard Stack

| 层 | 技术 | 本 phase 落点 |
|---|---|---|
| 后端 | Go 1.24 + Gin + GORM v1.30.5 | 新 migration `internal/core/db/migrations/migration_226_*.go` |
| DB | PostgreSQL 18（主）/ SQLite（`glebarez/sqlite`，dev+测试） | 新 migration **双方言**可跑（migration_218/221 范式） |
| 权限中间件 | `pkg/middleware/permission.go`（`RequireStrictPermissions` / `checkUserPermissionStrict`） | 只读消费，**不改** |
| 权限常量 | `pkg/permission/config.go`（`PermissionCode`，223 个） | 只读消费，**不改**（85 已补齐） |
| 权限服务 | `pkg/permission/service.go`（`GetUserPermissions`） | 只读消费 |
| 前端真相源 | `xingran-react-frontend/src/constants/permissionCodes.ts`（`PERMISSION_CODES`，34 区块） | 只读消费（86 Lead 裁定 4：86 不改 TS；87 也**不需要**改 TS —— 78 码已在 TS 声明） |
| 前端门控 | `src/components/shared/useActionPermission.ts`（三态 `decide`） | 只读消费 |
| 前端树 | antd 6.1 `Tree`（角色页 `menuIds` 表单项） | 只读消费 |
| 校验脚本 | `xingran-react-frontend/scripts/check-permission-codes.mjs`（468 行） | **本 phase 必须改**：清空白名单 + 加 F 型识别 + 加硬断言 |
| 路由登记 | `scripts/check-route-registry.mjs`（仓库根 `scripts/`，105 行） | 只读消费（已在 `lint:ci` 链尾） |
| 测试 | Go `testing` + `github.com/glebarez/sqlite`（内存库，原生 DDL）｜vitest + @testing-library/react | 新 migration 测试 + 角色页组件测试 |

---

## Architecture Patterns

### 1. 权限判定四段链路（本 phase 要接通的正是第 1 段的「数据来源」）

```
[播种] sys_menu(F 型行)  ──►  sys_role_menu(role_id, menu_id)  ──►  用户权限集  ──►  门控 / 放行
```

| 段 | 实现位置 | 事实 |
|---|---|---|
| ① F 型码进入用户权限集 | `pkg/permission/service.go:302-316` `GetUserPermissions`：`SELECT DISTINCT m.perms ... WHERE ur.user_id = ? AND m.perms IS NOT NULL AND m.perms != '' AND m.status = ?` —— **无 menu_type 过滤** ⇒ F 型码天然进入前端权限集 | `[VERIFIED: pkg/permission/service.go:302-316]` |
| ①' 前端实际读的那一份 | `internal/services/system/menu_service.go:509-571` `menuService.GetUserPermissions`：`Where("id IN ? AND perms IS NOT NULL AND perms != '' AND status = ?", menuIDs, models.MenuStatusNormal)`（+ `appendAncestorMenuIDs` 补祖先）—— 同样**无 menu_type 过滤**，且只过滤 `status`（PMODEL-02） | `[VERIFIED: internal/services/system/menu_service.go:509-571]` |
| ② 后端放行（写路由） | `pkg/middleware/permission.go:444-462` `checkUserPermissionStrict`：`m.perms = ? AND m.status = ?` **精确匹配**，跳过 Step2 继承与 Step3 模块降级；`RequireStrictPermissions`（:411-437）逐动作单一码挂载 | `[VERIFIED: pkg/middleware/permission.go:411-462]` |
| ②' Step 2「F 型 → 父页面」继承 | `pkg/middleware/permission.go:214-256` `hasInheritedButtonPerm`：查 `perms = permission` 的菜单 → 查其 `parent_id = menu.ID AND menu_type = 'F' AND status = normal` 的子按钮 → 用户持有任一子按钮 perms 即放行父页面权限。**方向：持 F 型按钮 → 继承父页面权限** | `[VERIFIED: pkg/middleware/permission.go:198-256]` |
| ③ 前端入口可见性 | `useActionPermission.decide`：`ready` 且含码 → 可用；`ready` 无码 → **不渲染**；`loading/error` → **渲染但 disabled + hint** | `[VERIFIED: src/components/shared/useActionPermission.ts:100-109]` |
| ④ 角色页授权写库 | `src/pages/system/role/hooks/useRoleActions.ts:181-201` 提交 `menuIds` → `POST /system/roles/:id/update` → `internal/services/system/role_service.go:384-452` `assignRoleMenusAndDepts` 写 `sys_role_menu`（`clause.OnConflict{DoNothing:true}`） | `[VERIFIED: 上述两处]` |

> **结论**：链路 ①②③④ **全部已就绪且已通电**。唯一缺的是 **「F 型码在 `sys_menu` 里有一行」** 这一个数据前提。
> 播种一旦落地，「角色页勾选 → 前端可见 + 后端放行」同源于**同一串权限码**，天然满足 RBAC-05「同源」要求。

### 2. Migration 范式（本 phase 必须沿用）

| 要素 | 事实 |
|---|---|
| 编号 | 文件名 `migration_NNN_<slug>.go`，函数 `MigrateNNN<Name>(db *gorm.DB) error`。**下一个可用编号 = 226**（`registry.go` 内最大版本 225；全仓 grep `migration_226` / `Migrate226` / `runAndRecord(226` **零命中**）`[VERIFIED: internal/core/db/migrations/registry.go:20-50]` |
| 注册 | `internal/core/db/migrations/registry.go` 的 `ActiveMigrations` 追加 `{226, "<name>"}` `[VERIFIED: registry.go:20-50]` |
| 执行 | `internal/core/db/database.go` **两处**调用块都要挂：PG 分支（advisory-lock 块内，:1009-1156）与 sqlite 分支（:1157-1245），用 `d.runAndRecord(226, "<name>", migrations.Migrate226Xxx)` `[VERIFIED: database.go:1018-1156 / 1159-1245]` |
| 记录 | `runAndRecord` 语义：`fn` 的错误**原样返回**；记录 upsert 失败只 WARN 不阻断；**不做任何 skip / 版本比较**——迁移每次启动无条件重跑 `[VERIFIED: database.go:1311-1336]` |
| 记录表 | `sys_schema_migrations(version int PK, name text, applied_at)`，由 `migration_225` 建；`ActiveMigrations` 的每个版本在 225 里被 seed `[VERIFIED: migration_225_schema_migrations_table.go:29-81]` |
| 失败处理档位 | migration_202/218/221 都是**非阻断**：内部 `log.Printf` + `return nil`，外层 `applogger.Errorf("... (非阻断,留待下次启动)")` `[VERIFIED: database.go:1029-1031 / 1120-1122 / 1136-1138]` |
| 幂等写法 | `migration_218`：`db.Where("perms = ?", code).First(&existing)` → 命中则**复用既有行 ID**、跳过创建，但**授权照常执行**；`migration_221`：`ensureManagerSyncButton` 同样「按 perms 先查后建」 `[VERIFIED: migration_218:42-48, 68-92；migration_221:167-197]` |
| 双方言 | `migration_218` / `migration_221` 的授权 SQL 用普通 `?` 占位 + `NOT EXISTS`，**无 PG-only 语法** ⇒ sqlite/PG 同跑 `[VERIFIED: migration_218:96-105；migration_221:200-211]` |

### 3. 校验脚本范式（RBAC-06）

`check-permission-codes.mjs` 的四类漂移 `[VERIFIED: check-permission-codes.mjs:12-17]`：

> a. TS 声明了，但后端常量没有 → 前端永远拿不到该权限
> b. 后端常量有，但 TS 没声明 → 前端漏了按钮门控
> c. TS 声明了，但 sys_menu 未声明 → 该权限永远不会被授予（菜单无授权来源）
> d. sys_menu 声明了 F 型按钮，但 TS 没声明 → 前端漏了按钮门控

---

## Don't Hand-Roll

| 别手搓 | 用现成的 | 证据 |
|---|---|---|
| 角色授权 UI | 已有 antd `Tree`（**不是** TreeSelect）已展示 F 型节点，**不需要**改前端授权组件 | `[VERIFIED: src/pages/system/role/index.tsx:433-456]` + `[VERIFIED: internal/services/system/menu_service.go:181-190]`（见 §C11） |
| F 型菜单建模 | `models.Menu{MenuType: models.MenuTypeButton, Visible: models.VisibleHidden, Status: models.MenuStatusNormal, Path: "", Icon: "#"}` | `[VERIFIED: migration_202:79-93；migration_218:70-84；migration_221:178-191]` |
| 精准授权 SQL | `INSERT INTO sys_role_menu (role_id, menu_id) SELECT rm.role_id, ? FROM sys_role_menu rm WHERE rm.menu_id = ? AND NOT EXISTS (...)` | `[VERIFIED: migration_218:96-105；migration_221:200-211]` |
| 权限门控 | `useActionPermission().decide(code)`，禁止各页自造 `hasPermission` | `[VERIFIED: src/components/shared/useActionPermission.ts:66-74]` |
| 权限码常量 | `pkg/permission/config.go` 的 `PermissionCode` 常量 + `src/constants/permissionCodes.ts` 的 `PERMISSION_CODES`，**不新造码** | `[VERIFIED: pkg/permission/config.go:10-342；src/constants/permissionCodes.ts:21-233]` |
| 迁移测试基建 | `migration_218_ad_reconciliation_permission_test.go` 的**原生 DDL + 内存 sqlite**（不用 GORM AutoMigrate，因 `type:uuid` 等 PG-only 片段） | `[VERIFIED: migration_218_ad_reconciliation_permission_test.go:18-70]` |
| 夹具造数/清理 | `.planning/notes/2026-10-29-v124-perm-fixture.md` §2/§4 全套 SQL（含清理后自检） | 见 §E15 |
| 决定性对照 | v1.24 Phase 79 范式：临时追加角色 → 403 转 200 → 移除 | `[VERIFIED: 2026-10-29-v124-perm-fixture.md:392-403, 854]` |
| HTTP 自签 token | `crypto.GenerateTokenWithSM2` + `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`（**不是** `RUOYI_JWT_SM2_*`） | `[VERIFIED: 2026-10-29-v124-perm-fixture.md:359-361, 453-457]` + `[VERIFIED: internal/config/config.go]`（note §3.6 引 `internal/config/config.go:364-365`） |
| 路由登记 | `scripts/scan-routes.cjs` + `scripts/route-registry.json` + `scripts/check-route-registry.mjs`（**仓库根** `scripts/`，非 frontend 内） | `[VERIFIED: scripts/check-route-registry.mjs:24-34；package.json:14]` |
| 三层证据判定器 | `RequireStrictPermissions` + `checkUserPermissionStrict` 已 v1.24 Phase 78 落地，**勿重做** | `[VERIFIED: pkg/middleware/permission.go:398-462]` |

---

## Common Pitfalls

| # | 坑 | 后果 | 规避 |
|---|---|---|---|
| **P-01** | **用 `menu_name` 定位父菜单** | 「楼宇管理」「楼层管理」「工位管理」「信息点管理」「机房管理」「专线管理」「机房设备管理」**各 2 条同名 C 菜单**；`network:command` 有「命令分发」「配置执行」**2 条同 perms 的 C 菜单**。按名字授权会命中**停用的旧菜单**（`status=1`），而旧菜单在 85 前**不能**写（Step 3 降级要求 `m.status = 0`）⇒ **按名字授权 = 越权授予** | 一律按 **perms + menu_type='C' + status=0** 定位；命中多条时取**全部**作为父集合，授权按「持有任一父」 |
| **P-02** | **直接用 `GrantNewMenuToRolesHavingParent`** | 该 helper ① **PG-only**（sqlite 路径 `return nil`，`isPostgreSQL` 守卫）② **按名字匹配**（P-01）③ 其 PG 测试**全部被 `t.Skip`** 跳过 | 改用 **migration_218 / 221 的 `NOT EXISTS` + parent_id 精确授权**（双方言、可测、精确） |
| **P-03** | **以为 F 型按钮在角色页看不到** | 事实相反：`menuService.GetTree` **无 menu_type / visible / status 过滤** ⇒ F 行**会**出现在角色页树里。真正的问题是**这些行根本不存在** | 不改前端授权组件；只播种数据 |
| **P-04** | **以为 `check-permission-codes.mjs` 能验证「F 型已播种」** | 该脚本对 **Go migration 只扫字符串字面量** ⇒ 一行注释里的码字面量就能满足 c 类。Go 侧**分不出 F 型与 C 型** | 脚本新增「Go 侧 F 型声明块」解析（§B9 方案），并把该块当 F 型进入 d 类 |
| **P-05** | **清空白名单 ≠ 强制播种** | 78 项中 **6 项（ops:manager:*）** 已被 migration_221 的 Go 字面量覆盖 ⇒ 清空白名单后它们**照样过**；另外 72 项**必须**在新 migration 里出现字面量，否则 c 类 exit 1 | 逐项落地 72 码；6 码标注「migration_221 已播种」 |
| **P-06** | **权限缓存陈旧** | 后端 `mwPermCacheTTL = 30 * time.Second` 且 `InvalidateMiddlewarePermCache` **未接线** ⇒ 授权后 ≤30s 才生效；前端 menuStore **5min 内存 TTL** + sessionStorage **30min** ⇒ UI 上看不到新权限 | 后端：起**新进程**（缓存空）或等 31s；前端：清 sessionStorage `menu:cache` 或重新登录 |
| **P-07** | **夹具角色撞 `admin`** | `isSuperAdmin` 判 `r.role_key = 'admin'` ⇒ 撞名直接 bypass，403 变 200，假绿 | 夹具 role_key 用 `p87_*` 前缀，**绝不含** `admin` |
| **P-08** | **sqlite 上用角色 API 造夹具** | `role_service.go:425-430` 的 `clause.OnConflict{DoNothing:true}` 在 sqlite（modernc）下方言不兼容 ⇒ sqlite 无法经 API 分配菜单 | sqlite 路径**直接 INSERT** `sys_role_menu` / `sys_user_role` |
| **P-09** | **`sys_role_menu` 无唯一约束** | `models/role.go:24-27` 只有普通索引（`index:idx_sys_role_menu_role_id_menu_id`），**不是 unique** ⇒ 裸 INSERT 会插重复行 | 授权 SQL 一律带 `NOT EXISTS`（migration_218/221 范式） |
| **P-10** | **`sys_menu.perms` 无唯一索引** | 库内已存在同 perms 多行（`system:dict:list` × 2、`ops:building:query` × 2、`network:command` × 2 条 C 菜单） | 播种前按 `perms` 查「已存在则复用 ID」，**不要**无条件 INSERT |
| **P-11** | **给 `ops:asset:import/export` 挂错父** | 「资产列表」C 菜单 **perms 为 NULL**（`migration_147` 建时不写 perms）⇒ 按 perms 查不到 | 该域按 `menu_name='资产列表' AND menu_type='C'` + `component='operations/assets/index'` 定位 |
| **P-12** | **授权后立刻断言 HTTP 200** | 30s 权限缓存（后端）+ 5min/30min 前端缓存 ⇒ 假失败 | 见 P-06；验收脚本必须显式等待或用新进程 |
| **P-13** | **以为 `check-permission-codes.mjs` 已在 CI 门禁里** | `package.json:14` 的 `lint:ci` 链**不含**它：`eslint && assert-lint-budget && check-exemption-reasons && check-gating-coverage && check-hardcoded-colors && css-contract-check && node ../scripts/check-route-registry.mjs` | RBAC-06 必须**显式把它加进 `lint:ci`**；且 `.github/workflows/frontend-build.yml:44-45` 只跑 `npm run lint`（不是 `lint:ci`）⇒ **两处都要改**才叫「接入 CI 门禁」 |
| **P-14** | **把 `system:role:export` 当成有效动作** | 全仓无该动作的后端路由（只有常量 `config.go:27` + TS 声明 + `resource_action_map.go:53`） | 仍然播种（白名单是权威清单），但在验收报告标注「无路由，仅作授权来源占位」 |
| **P-15** | **连生产库起本地后端** | `SKIP_AUTOMIGRATE=true` 在 `server.mode=release` 下 **fatal 拒绝启动**（`internal/core/core.go:286-291`）；且 `cmd/main.go` 的 `importOUIData` / cron 会写库 | 见 §E16 的最小旁路清单 |

---

## Code Examples

### E1 · migration 骨架（双方言、幂等、精确授权）

> 范式：`migration_218`（按 perms 找父 + 复用既有行 + NOT EXISTS 精确授权）＋ `migration_221`（结构化 seed 表 + 双方言）。

```go
package migrations

// actionPermSeed 一条 F 型动作权限种子。
// parentPerms: 父 C 型菜单的 perms（按 perms 定位，不用 menu_name —— 同名菜单存在）
// parentName:  父菜单名（仅用于日志与 perms 为 NULL 的特例定位）
type actionPermSeed struct {
    name       string // sys_menu.menu_name
    perms      string // sys_menu.perms（单一权限码）
    parentPerms string // 父 C 型菜单 perms；为空时走 parentName 特例
    parentName string  // 父 C 型菜单名（perms 为 NULL 的特例，如「资产列表」）
    orderNum   int
}

// Migrate226ActionPermissionFTypeSeed Phase 87（RBAC-05）：播种 72 条动作级 F 型菜单。
//
// 幂等：① 按 perms 查既有行 → 命中则复用其 ID（不重复创建）；未命中则创建。
//      ② 授权用 NOT EXISTS（sys_role_menu 无唯一约束 —— 见 models/role.go:24-27）。
// 降级：父菜单缺失 → log.Printf + 跳过该条（return nil），不阻塞启动。
func Migrate226ActionPermissionFTypeSeed(db *gorm.DB) error {
    log.Println("Running migration 226: action-level F type permission seed")

    for _, seed := range actionPermissionSeeds {
        // 1) 定位父 C 型菜单（perms 精确 + menu_type='C' + status=0）
        parents, err := resolveParentMenus(db, seed)
        if err != nil || len(parents) == 0 {
            log.Printf("Migration 226: WARNING 父菜单未找到, 跳过 %s: %v", seed.perms, err)
            continue
        }

        // 2) 幂等：按 perms 复用既有行，否则创建 F 型行（parent 取第一父）
        menuID, err := ensureFTypeMenu(db, seed, parents[0].ID)
        if err != nil {
            log.Printf("Migration 226: WARNING 创建 %s 失败(非阻断): %v", seed.perms, err)
            continue
        }

        // 3) 精准授权：授予「持有任一父 C 型菜单」的角色（NOT EXISTS 幂等，双方言）
        if err := grantToParentHolders(db, menuID, parents); err != nil {
            log.Printf("Migration 226: WARNING 授权 %s 失败(非阻断): %v", seed.perms, err)
        }
    }
    log.Println("Migration 226 completed: action-level F type permissions seeded")
    return nil
}

// grantToParentHolders 双方言幂等授权（migration_218:96-105 / migration_221:200-211 范式）。
func grantToParentHolders(db *gorm.DB, menuID string, parents []models.Menu) error {
    ids := make([]string, 0, len(parents))
    for _, p := range parents {
        ids = append(ids, p.ID)
    }
    return db.Exec(
        `INSERT INTO sys_role_menu (role_id, menu_id)
         SELECT rm.role_id, ? FROM sys_role_menu rm
         WHERE rm.menu_id IN ?
           AND NOT EXISTS (
               SELECT 1 FROM sys_role_menu x
               WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
        menuID, ids, menuID).Error
}
```

> ⚠️ 上面的 `grantToParentHolders` 是**示意**。实现时请照抄 migration_221 已验证的 `?` 占位写法（`WHERE rm.menu_id = ?`），多父场景改用 `IN ?`（GORM 的 `IN ?` 展开切片）—— **不要**用 `fmt.Sprintf` 拼 SQL（`menu_grant_helpers_test.go:128-131` 已有反向断言禁止 `fmt.Sprintf`）。

### E2 · F 型行构造（字段口径固定）

```go
emptyPath := ""
icon := "#"
perms := seed.perms
button := &models.Menu{
    MenuName: seed.name,
    ParentID: &parentID,
    Path:     &emptyPath,               // F 型 path 不参与 routeGenerator
    MenuType: models.MenuTypeButton,    // 'F'
    Visible:  models.VisibleHidden,     // 0
    Status:   models.MenuStatusNormal,  // 0
    Perms:    &perms,                   // 单码
    Icon:     &icon,
    OrderNum: seed.orderNum,
    Remark:   "Phase 87: 动作级权限按钮",
}
```
> 逐字来源：`[VERIFIED: migration_202:79-93]`（同款字段）、`[VERIFIED: internal/models/base.go:86]`（`MenuTypeButton MenuType = "F"`）、`[VERIFIED: internal/models/base.go:94]`（`VisibleHidden VisibleType = 0`）。

### E3 · 父菜单解析表（**planner 必须原样落进 PLAN**）

> 解析规则：**`perms = <父码> AND menu_type = 'C' AND status = 0`**；命中 0 条 → 降级跳过；命中 1 条 → 唯一父；命中 ≥2 条 → 父集合全部（授权按「持有任一父」，F 行挂第一条）。
> 唯一特例：`ops:asset:*` 的父「资产列表」**perms 为 NULL** → 按 `menu_name='资产列表' AND menu_type='C' AND component='operations/assets/index'`。

| # | 码族（白名单内） | 码数 | 父 C 型菜单定位 | 父集合大小 | 证据 |
|---|---|---|---|---|---|
| 1 | `system:role:{list,view,add,edit,remove,export}` | 6 | `perms='system:role'` | 1 | `menu_catalog_seed.sql` 行 64：`... '角色管理', 'd67f4240-…', 2, 'role', 'system/role/index', 'C', 1, 0, 'system:role', …` |
| 2 | `system:menu:{list,view,add,edit,remove}` | 5 | `perms='system:menu'` | 1 | 同文件 行 99 |
| 3 | `system:dept:{list,view,add,edit,remove,import,export}` | 7 | `perms='system:dept'` | 1 | 同文件 行 135 |
| 4 | `system:post:{list,view,add,edit,remove}` | 5 | `perms='system:post'` | 1 | 同文件 行 172 |
| 5 | `ops:building:{import,export}` | 2 | `perms='ops:building:list'` | 1（status=0 那条） | 同文件行 `e23f29b1-…`（`status=0`）；同名旧行 `bcf533b5-…` 是 `status=1` |
| 6 | `ops:floor:{import,export}` | 2 | `perms='ops:floor:list'` | 1 | `73a68a97-…`（st=0）/ `d0a87d3b-…`（st=1） |
| 7 | `ops:workstation:{import,export}` | 2 | `perms='ops:workstation:list'` | 1 | `39ef63a1-…`（st=0）/ `02f9dff5-…`（st=1） |
| 8 | `ops:asset:{import,export}` | 2 | **特例**：`menu_name='资产列表' AND menu_type='C' AND component='operations/assets/index'` | 1 | 同文件行 `2bff4f6a-45a5-404a-8007-82c1118b10d6` 的 `perms` 为 **NULL**；`migration_147_fix_asset_menu_structure.go:46-58` 建时不写 Perms |
| 9 | `ops:infopoint:{import,export}` | 2 | `perms='ops:infopoint:list'` | 1 | `db7db630-…`（st=0）/ `2102d99d-…`（st=1） |
| 10 | `ops:serverroom:{import,export}` | 2 | `perms='ops:serverroom:list'` | 1 | `6a8575c2-…`（st=0）/ `a8f1a53a-…`（st=1） |
| 11 | `ops:dedicatedline:{import,export}` | 2 | `perms='ops:dedicatedline:list'` | 1 | `736784d2-…`（st=0）/ `7cb218a5-…`（st=1） |
| 12 | `ops:roomdevice:{import,export}` | 2 | `perms='ops:roomdevice:list'` | 1 | `cf527892-…`（st=0）/ `5209f1dd-…`（st=1） |
| 13 | `ops:walls:{add,edit,delete}` + `ops:doors:{add,edit,delete}` + `ops:floor-plan:edit` | 7 | `perms='ops:floor:list'`（**楼层管理**） | 1 | CAD 平面图编辑器在楼层页：`src/pages/operations/floors/index.tsx:57` `const permFloorPlanEdit = permGate.decide(PERMISSION_CODES.opsFloorPlan.edit);`；后端写路由 `internal/api/router.go:1146-1152 / 1178-1184 / 1206-1211` |
| 14 | `network:device:{list,view,add,edit,delete,export}` | 6 | `perms='network:device'` | 1 | `80b06462-…` |
| 15 | `network:credential:{list,view,add,edit,delete,export}` | 6 | `perms='network:credential'` | 1 | `19d779fa-…` |
| 16 | `network:template:{list,view,add,edit,delete,export}` | 6 | `perms='network:template'` | 1 | `dca88b62-…` |
| 17 | `network:command:{view,export}` | 2 | `perms='network:command' AND status=0` | **2**（命令分发 `99f2b245-…` / 配置执行 `c02cf900-…`） | 两页共用同一码：`src/pages/network/command/index.tsx:41,44` 与 `src/pages/network/executions/index.tsx:36,39` |
| 18 | `network:backup:{list,delete,export}` | 3 | `perms='network:backup'` | 1 | `0eb450f4-…` |
| 19 | `network:discovery:{view,delete,export}` | 3 | `perms='network:discovery'` | 1 | `7e7207d8-…` |
| 20 | `ops:manager:rule:{list,edit,delete}` + `ops:manager:org:{list,edit,delete}` | 6 | `perms='ops:manager:rule:list'` / `'ops:manager:org:list'`（`migration_221` 建） | 1 | **migration_221 已播种**（见 §B7），87 **不重复播种** |
| | **合计** | **78** | | | |

### E4 · 决定性对照实验（最小步骤，v1.24 Phase 79 范式）

```
① 夹具角色 role_key = 'p87_probe_test'（绝不撞 'admin'）→ 绑定父 C 型菜单（仅页面级，0 个 F）
② 该角色下用户登录 → POST /system/my-menus/permissions ⇒ 权限集不含目标动作码
③ POST /system/roles（写路由）⇒ HTTP 403，{"code":403,"message":"没有访问权限"}
④ DB 前后对照：SELECT count(*) FROM <目标表> 调用前后一致 ⇒ 无数据变更
⑤ 决定性对照：INSERT INTO sys_role_menu 给该角色补授「目标动作码对应的 F 型 menu_id」
   （或直接在角色页勾选）→ 等 31s（或重启后端）→ 同一请求 ⇒ HTTP 200 且真实生效
⑥ 撤销 ⑤ 的授权 ⇒ 回到 403
```
> ③ 的 403 由 `response.ErrForbidden` 统一返回，`pkg/middleware/permission.go:434` 逐字：`response.Error(c, response.ErrForbidden, "没有访问权限")` `[VERIFIED]`。
> ④⑤ 的完整 SQL 与清理自检见 §E15 引用。

---

## A. F 型菜单播种（Research Questions A1–A5）

### A1 · 现有 migration 的完整范式

| 问 | 答（带证据） |
|---|---|
| 编号规则 | `migration_NNN_<slug>.go` + `MigrateNNN<Name>(db *gorm.DB) error`。**下一个可用编号 = 226**（registry 内最大 225；全仓 `migration_226|Migrate226|runAndRecord(226` grep = 0 命中）`[VERIFIED: registry.go:20-50]` |
| 文件命名 | 同上（`migration_202_port_write_audit.go` / `migration_218_ad_reconciliation_permission.go` / `migration_221_manager_sync_menus.go` / `migration_225_schema_migrations_table.go`）`[VERIFIED: 目录清单]` |
| 注册方式 | `registry.go` 的 `ActiveMigrations` 追加条目；注释明确「维护约定：database.go 调用块每新增/移除一个 MigrateNNN 调用点，本清单必须同步增删，否则 MissingRecords 对账会误报。记录表只读不控制执行」`[VERIFIED: registry.go:10-19]` |
| 自动执行机制 | 启动期 `internal/core/db/database.go` 的 `d.runAndRecord(version, name, fn)`；**每次启动无条件重跑**（无 skip / 版本比较）`[VERIFIED: database.go:1311-1336]`。PG 分支整块包在 `pg_advisory_lock` 里（`d.acquireMigrationAdvisoryLock()`），未获锁实例 WARN 跳过 `[VERIFIED: database.go:1009-1014]` |
| 幂等写法 | ① 建行：**按 perms 先查后建**，命中复用既有 ID（`migration_218:42-48,68-92`；`migration_221:167-197`）；② 授权：**`NOT EXISTS` 兜底**（`migration_218:96-105`；`migration_221:200-211`）；③ 整表 DDL 用 `CREATE ... IF NOT EXISTS` / `ON CONFLICT DO NOTHING`（`migration_225:32-46,63-68`） |
| 回滚 / 失败处理 | **无回滚机制**（迁移是幂等重跑，不是版本迁移）。失败一律**非阻断降级**：内部 `log.Printf` + `return nil`，外层 `applogger.Errorf("… (非阻断,留待下次启动)")` `[VERIFIED: database.go:1029-1031, 1120-1122, 1136-1138]`。**87 必须沿用此档位**（缺 F 行只让普通角色 403 / 按钮不显示，不影响登录主链路） |
| 双方言要求 | `migration_218` / `migration_221` **双方言**（授权 SQL 用普通 `?` 占位）；`migration_202` 是 PG-only（`isPostgreSQL` 守卫直接 return nil）`[VERIFIED: migration_202:34-38；migration_218:96-105；migration_221:200-211]`。**87 的新 migration 必须双方言**（sqlite 是 dev/测试运行时，`database.go:1157-1245` 有 sqlite 分支） |

### A2 · `GrantNewMenuToRolesHavingParent` 的完整语义

`[VERIFIED: internal/core/db/migrations/menu_grant_helpers.go:26-50]`

```go
func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error {
	if !isPostgreSQL(db) {
		return nil
	}
	const sql = `
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT rm.role_id, $1::uuid
FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = $2
ON CONFLICT DO NOTHING
`
	return db.Exec(sql, newMenuID, parentMenuName).Error
}
```

| 维度 | 语义 |
|---|---|
| 完整语义 | 找**名字等于 `parentMenuName`** 的 `sys_menu` 行 → 取持有该行的角色 → 给这些角色插入 `(role_id, new_menu_id)` |
| 参数化绑定 | `$1::uuid`（newMenuID）+ `$2`（parentMenuName），**参数化**；`menu_grant_helpers_test.go:128-131` 有**反向断言**禁止 `fmt.Sprintf` `[VERIFIED]` |
| PG / sqlite 双路径 | **实为 PG-only**：`isPostgreSQL(db)` 守卫在 sqlite 下 `return nil`（**noop**），注释逐字：「SQLite 不支持 PG 的 ::uuid cast + ON CONFLICT DO NOTHING; helper 是 PG 启动期 migration 调用点, SQLite 测试路径不消费」`[VERIFIED: menu_grant_helpers.go:24-31]` |
| 幂等性 | `ON CONFLICT DO NOTHING` 依赖 `sys_role_menu` 的 `(role_id, menu_id)` 复合 PK / 唯一约束 `[VERIFIED: menu_grant_helpers.go:22-24]` —— ⚠️ **但 `models/role.go:24-27` 的 GORM tag 只是普通索引**（`index:idx_sys_role_menu_role_id_menu_id`），生产库是否有该唯一约束**未经本会话实证** ⇒ 该幂等性在 PG 生产库**存疑**（见 §Open Questions OQ-1） |
| 失败是否阻塞 migration | helper 自身 `return db.Exec(...).Error`（返回错误）；**由调用方决定是否阻断**。`migration_202:102-104` 的处理是**非阻断**：`log.Printf("Migration 202: WARNING GrantNewMenuToRolesHavingParent failed (non-blocking): %v", err)` |
| 三个测试的真实行为 | `[VERIFIED: menu_grant_helpers_test.go:35-137]` ① `TestGrantNewMenuToRolesHavingParent_NonExistentParent`（sqlite）：真跑，断言返回 nil 不 panic；② `TestGrantNewMenuToRolesHavingParent_Idempotent`：**`if !postgresAvailable() { t.Skip(...) }`**，且 `openPostgresDB` 内 `t.Skip("PG integration path not configured in this executor")` ⇒ **本地/CI 恒跳过**；③ `TestGrantNewMenuToRolesHavingParent_OnlyAffectsParentRoles`：**同样恒跳过**；④ 实际只有 `TestGrantNewMenuToRolesHavingParent_ParameterizedOrControlled`（源码 grep 断言）真跑 |
| 门禁 | `postgresAvailable()` 逐字：`return os.Getenv("XINGRAN_PG_TEST_DSN") != ""` `[VERIFIED: menu_grant_helpers_test.go:141-145]` |

> **结论**：`GrantNewMenuToRolesHavingParent` 的**三个测试里只有一个真正执行**（源码 grep），另两个 PG 集成测试是 `t.Skip` 壳。**87 不可依赖它**，理由三条：① sqlite noop（sqlite 分支的 migration 会静默跳过授权）② 按 `menu_name` 匹配（见 A3/P-01）③ PG 唯一约束存疑（OQ-1）。
> **替代方案（prescriptive）**：用 **migration_218 / 221 的 `NOT EXISTS` + parent_id 精确授权**（双方言、可用内存 sqlite 完整测试、精确到 id）。

### A3 · 「父菜单名」定位在本 phase 是否够用 —— **不够**

**答：不够。父菜单名不唯一，且同名副本中含 `status=1` 的停用旧菜单。**

实测证据（`internal/core/db/migrations/menu_catalog_seed.sql`，共 239 条 INSERT，本会话解析出 229 条可定位行）：

| 菜单名 | 同名行数 | 其中 `status=0`（生效） | 其中 `status=1`（停用） | perms |
|---|---|---|---|---|
| 楼宇管理 | 2 | `e23f29b1-e4d8-47ba-836d-705c716fa86b`（`path=operations/buildings`, `component=/operations/buildings/index`） | `bcf533b5-1d5d-4fac-a07c-accc70e88f8d`（`path=buildings`, `component=operations/buildings/index`） | 两行都是 `ops:building:list` |
| 楼层管理 | 2 | `73a68a97-8d04-4b61-8d65-e4f41a162513` | `d0a87d3b-ae2c-4507-bf1d-ac1c8e651462` | `ops:floor:list` |
| 工位管理 | 2 | `39ef63a1-cd98-47c9-acf5-55130758a78d` | `02f9dff5-b271-43c8-b9f0-dfcd0bb5fbb9` | `ops:workstation:list` |
| 信息点管理 | 2 | `db7db630-b34b-454b-88d2-a598ec603671` | `2102d99d-f828-4107-a9d0-61336348424f` | `ops:infopoint:list` |
| 机房管理 | 2 | `6a8575c2-6177-40ef-bad6-070301a55400` | `a8f1a53a-3559-4990-904a-d1dd67671c5b` | `ops:serverroom:list` |
| 专线管理 | 2 | `736784d2-db2a-46e6-8fa1-b141a2b60f79` | `7cb218a5-e3e4-49a7-bdb7-52c73f5f4df2` | `ops:dedicatedline:list` |
| 机房设备管理 | 2 | `cf527892-5324-4f04-8874-a26ee460aa19` | `5209f1dd-b410-49b6-a080-bd925b6522ba` | `ops:roomdevice:list` |
| 命令分发 / 配置执行 | 2（**不同名同 perms**） | `99f2b245-2433-42f9-8a45-d56528b26767`（`network/command/index`）/ `c02cf900-44ce-4de3-990a-5571544d6cb6`（`network/executions/index`） | 无 | 两行都是 `network:command` |

`system:role` / `system:menu` / `system:dept` / `system:post` 各自**唯一**（perms 分别为 `system:role` / `system:menu` / `system:dept` / `system:post`，父目录均为 `d67f4240-f887-481a-b345-94fb36782500`「系统管理」）。

**85/86 铺开的 F 型码落在哪些父菜单下** ⇒ 见 §E3 的 20 行父解析表。

#### A3-1 · 补充实测：库内存在「两套平行菜单树」（live / stopped）

对 `menu_catalog_seed.sql` 全量解析后，**每个 ops 域 C 菜单都有两份同名同 perms 的副本，且两份下各挂着一整套同名 F 行**（status=0 的新树 + status=1 的停用旧树）：

| 域 | live 父（status=0） | stopped 父（status=1） | live 父下已存在的 F 码 |
|---|---|---|---|
| `ops:building` | `e23f29b1-…` 楼宇管理 | `bcf533b5-…` 楼宇管理 | `query` `add` `edit` `delete` |
| `ops:floor` | `73a68a97-…` 楼层管理 | `d0a87d3b-…` 楼层管理 | `query` `add` `edit` `delete` |
| `ops:workstation` | `39ef63a1-…` 工位管理 | `02f9dff5-…` 工位管理 | `query` `add` `edit` `delete` |
| `ops:infopoint` | `db7db630-…` 信息点管理 | `2102d99d-…` 信息点管理 | `query` `add` `edit` `delete` |
| `ops:serverroom` | `6a8575c2-…` 机房管理 | `a8f1a53a-…` 机房管理 | `query` `add` `edit` `delete` |
| `ops:dedicatedline` | `736784d2-…` 专线管理 | `7cb218a5-…` 专线管理 | `query` `add` `edit` `delete` |
| `ops:roomdevice` | `cf527892-…` 机房设备管理 | `5209f1dd-…` 机房设备管理 | `query` `add` `edit` `delete` |
| `ops:asset` | `2bff4f6a-…` 资产列表（**perms 为 NULL**） | — | `list` `add` `edit` `delete` |
| `network:command` | `99f2b245-…` 命令分发 | — （另一条 `c02cf900-…`「配置执行」status=0 但**无任何 F 子节点**） | `execute` |
| `network:backup` | `0eb450f4-…` 配置备份 | — | `add` `restore` `diff` |
| `network:discovery` | `7e7207d8-…` 设备发现 | — | `add` |
| `system:role/menu/dept/post` | 各唯一 | — | **0** |
| `ops:walls` / `ops:doors` / `ops:floor-plan` | **无任何菜单行** | — | **0** |
| `network:device` / `credential` / `template` | 各唯一 | — | **0** |

**三条直接推论（prescriptive）**：

1. **live 父 = `status=0` 那条**（停用旧树对权限系统不可见：`hasInheritedButtonPerm` 与 `checkUserPermissionStrict` 都带 `m.status = normal` 条件）⇒ 父解析表用 `status=0` 过滤是**正确且必要**的。
2. **`ops:asset` 的父「资产列表」perms 为 NULL**（`migration_147_fix_asset_menu_structure.go:46-58` 建时不写 Perms）⇒ 必须走 `menu_name` + `component` 特例定位（见 §E3 第 8 行 / P-11）。
3. **`network:command` 的两个 C 菜单 status 都是 0**，且「配置执行」**无 F 子节点**；但 `network/command/index.tsx` 与 `network/executions/index.tsx` **两页共用同一串码**（`networkCommand.execute` / `.export`）⇒ 父集合 = 两条，F 行挂在「命令分发」（与既有 `network:command:execute` 同父，保持一致），**授权按「持有任一父」**。
   - 补充：Step 2 继承时 `hasInheritedButtonPerm` 用 `db.Where("perms = ? AND status = ?", …).First(&menu)`（`pkg/middleware/permission.go:216`）—— GORM `First` 默认按**主键升序**取第一条，`99f2b245…` < `c02cf900…` ⇒ 继承**恒定解析到「命令分发」**，与 F 行挂哪一条无关。

**判定**：
1. **不能**用 `menu_name` 定位父菜单（同名 + 含停用行）。
2. **必须**用 **`perms` + `menu_type='C'` + `status=0`** 定位；命中多条（仅 `network:command`）时取**父集合全部**。
3. **必须**用 **parent_id 精确授权**（`WHERE rm.menu_id = <父 id>`），不得用名字授权。

### A4 · `sys_menu` 表的 F 型行结构

| 字段 | 值 | 证据 |
|---|---|---|
| `menu_type` | `'F'` | `[VERIFIED: internal/models/base.go:86]` `MenuTypeButton MenuType = "F" // 按钮` |
| `perms` | **单一权限码**（`size:100`，可为 NULL） | `[VERIFIED: internal/models/menu.go:71]` `Perms *string \`gorm:"size:100" json:"perms,omitempty"\`` |
| `visible` | `0`（隐藏） | `[VERIFIED: internal/models/base.go:94]` `VisibleHidden VisibleType = 0 // 隐藏`；`[VERIFIED: internal/models/base.go:93]` `VisibleShow VisibleType = 1 // 显示`（**菜单可见性是布尔语义：1=可见，0=隐藏**） |
| `status` | `0`（正常） | `[VERIFIED: internal/models/base.go:101-102]` `MenuStatusNormal MenuStatus = 0 // 正常` / `MenuStatusStop MenuStatus = 1 // 停用` |
| `path` | `""`（空串，F 型不参与 routeGenerator） | `[VERIFIED: migration_202:77-79]` 注释逐字：「Path="" 空串(F 型 path 不参与 routeGenerator,migration_200 惯例)」 |
| `icon` | `"#"` | `[VERIFIED: migration_202:80]` |
| `parent_id` | 指向父 C 型菜单的 uuid | `[VERIFIED: migration_218:76]`、`[VERIFIED: migration_221:185]` |
| `meta` | F 型一般不写 | `[VERIFIED: migration_218:73-84]`（未设 Meta） |
| **禁止项** | **不得**写 Java sys_menu 风格的 `frame` / `cache` 列（Go 用 `Meta` JSONB 表达） | `[VERIFIED: migration_202:26-28]` 逐字：「**绝对禁止** Java sys_menu 风格的 frame / cache 字段(Go sys_menu 用 Meta JSONB 表达)」 |
| `perms` 是否可多码 | **单码**。库里已有同 perms 多行的先例（`system:dict:list` × 2、`ops:building:query` × 2）—— 但那是历史遗留，**新播种应保持单码单行** |

> 注：`menu_catalog_seed.sql` 里的历史 F 行 `visible=1`（如 `system:user:list` 行 visible=1），而 migration_202/218/221 新建的用 `VisibleHidden(0)`。两者都不影响权限判定（判定只看 `perms` 与 `status`，`visible` 只影响导航栏显示），但 **87 新建的统一用 `visible=0`**。

### A5 · 幂等与重跑

| 问 | 答 |
|---|---|
| 同一 migration 重复执行 | **每次启动无条件重跑**（`runAndRecord` 无 skip 语义）`[VERIFIED: database.go:1318-1319]`。因此**幂等是硬要求** |
| `sys_menu` 里已存在同 perms 的 F 型行时如何判定 | **按 `perms` 查 → 命中则复用既有行 ID、跳过创建**（`migration_218:42-48, 68-70`：`menuID := existing.ID; if !hasPermRow { …create… }`）。**不要**用「count>0 就整段 return」（`migration_202:61-64` 那样会**连授权一起跳过**，导致后加入的角色永远拿不到） |
| `sys_role_menu` 重复 | 用 `NOT EXISTS`（`migration_218:96-105`）或 `clause.OnConflict{DoNothing}`（`role_service.go:425-430`，**但 sqlite 方言不兼容**）⇒ migration 内**只用 `NOT EXISTS`** |
| 「已存在同 perms 但不是 F 型」的边缘 | 库内 `system:dict:list` 同时存在于 C 型（`a6b0bd13-…` 字典管理页 perms）和 F 型（`db3b6c4d-…` 字典类型查询）行。**87 的 72 个码当前无任何行**（§B7），但 `ops:manager:rule:list` 已是 C 型 ⇒ 播种时按 perms 查到的既有行**可能不是 F 型** ⇒ 判定应加 `AND menu_type = 'F'`（`migration_218:44` 用的是 `Where("perms = ?")` 无 type 约束，87 应更严） |
| 父菜单缺失 | **log warning + 跳过该条 + `return nil`**（非阻断）`[VERIFIED: migration_218:59-63；migration_202:70-73]` |

---

## B. 权限码三方漂移校验（Research Questions B6–B9）

### B6 · `check-permission-codes.mjs` 的完整判定逻辑

`[VERIFIED: xingran-react-frontend/scripts/check-permission-codes.mjs]`（468 行）

| 要素 | 事实 |
|---|---|
| 三份对照声明 | ① 前端 `src/constants/permissionCodes.ts`（`PERMISSION_CODES`）② 后端 `pkg/permission/config.go`（`PermissionCode = "..."`）③ 菜单种子 `internal/core/db/migrations/**` 的 `sys_menu.perms` 声明（`:7-11`） |
| a 类 | 「TS 声明了，但后端常量没有」⇒「前端永远拿不到该权限」（`:414-419`） |
| b 类 | 「后端常量有，但 TS 没声明」⇒「前端漏了按钮门控」（`:420-425`） |
| c 类 | 「TS 声明了，但 sys_menu 未声明」⇒「该权限永远不会被授予（菜单无授权来源）」（`:426-435`） |
| d 类 | 「sys_menu 声明了 F 型按钮，但 TS 没声明」⇒「前端漏了按钮门控」（`:436-441`） |
| 作用域口径 | 每个 TS 模块共享同一前缀；前缀本身（如 `system:user`）是**页面本体**权限（C 型），**不参与比对**；比对作用域 = `前缀 + ":"`（`:18-20, 396-403`） |
| 前缀一致性附加检查 | 模块内首个码的前缀之外出现异名前缀 → 报错（`:396-401`）；85-03 放宽了「更深嵌套」（`:391-395`） |
| 菜单侧提取口径 | `.sql` 文件：`INSERT INTO sys_menu (...) VALUES (...)` → 按列位置取 `menu_type='F'` 的行进 `fType`；`INSERT ... SELECT` → 取语句内所有字符串字面量进 `untyped`（`:22-27, 302-334`）。`.go` 文件：**取所有匹配权限码字面量 → 全部进 `untyped`**（`:374-381`） |
| `archive/` 目录 | **不扫描**（`:27, 346`） |
| 权限码字面量正则 | `PERM_LITERAL = /^[a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+$/`（`:44`）——至少两段、段内小写字母数字下划线 |
| `PENDING_SEED_WHITELIST` 结构 | `:57-143`，**78 条**，每条三个字段：`{ code: string, reason: string, evidence: string }`；`reason` 统一形如「system 4 组 F 型菜单 0 声明，87 播种后移除」，`evidence` 统一为「85-RESEARCH §B4 / tmp_ftype.json」或「85-RESEARCH §B / tmp_ftype.json」。派生 `PENDING_SEED_SET`（`:145`） |
| 白名单的作用范围 | **仅作用于 c 类**（`:430-433`）；命中时打印 `⏳ … 在 85 新码待播种白名单内（c 类跳过；87 播种后必须清空白名单）` |
| 空名单时的行为 | `if (PENDING_SEED_WHITELIST.length > 0) { console.log("⚠ PENDING_SEED_WHITELIST 非空（…）") }`（`:449-455`）—— **白名单为空 ⇒ 无任何输出、无警告、exit 0**；**白名单非空 ⇒ 仅警告，仍 exit 0**（不阻断） |
| 白名单为空后 c 类如何生效 | c 类改为对每个 TS 码要求 `menuAllScope.has(code)`，其中 `menuAllScope = fType ∪ untyped`（`:408-410`）⇒ **Go migration 里的字面量即算「已声明」** |

> **B6 关键结论（喂给 §B9 的脚本改造）**：
> ① **Go 侧不区分 F 型与 C 型** —— 一个 Go migration 里的任意字符串字面量（含注释）都能满足 c 类。
> ② 因此「清空白名单」**不能**证明 F 型已播种，只能证明「码字面量出现过」。
> ③ `d` 类（菜单 F 型 vs TS）**只对 SQL 源生效**（`fType` 只从 `.sql` 的 `INSERT ... VALUES` 里来）⇒ Go migration 播种的 F 行**永远进不了 d 类**。

### B7 · 白名单 78 项逐项状态（**播种权威清单**）

> 统计口径复算（本会话实跑脚本）：`TOTAL = 78`。

| 域 | 码数 | 码清单 | `sys_menu` 现状 | 87 处置 |
|---|---|---|---|---|
| `system:role` | 6 | `list` `view` `add` `edit` `remove` `export` | **F 型 0 行** | 播种 6 条 F 行，父 = 角色管理 |
| `system:menu` | 5 | `list` `view` `add` `edit` `remove` | **F 型 0 行** | 播种 5 条，父 = 菜单管理 |
| `system:dept` | 7 | `list` `view` `add` `edit` `remove` `import` `export` | **F 型 0 行** | 播种 7 条，父 = 部门管理 |
| `system:post` | 5 | `list` `view` `add` `edit` `remove` | **F 型 0 行** | 播种 5 条，父 = 岗位管理 |
| `ops:building` | 2 | `import` `export` | 已有 F 型 **6 个**（distinct perms：`query` `add` `edit` `delete` `spaces:query` `spaces:3d:query`；原始行 10 条，因两套平行父各挂一套） | 播种 2 条，父 = 楼宇管理（**status=0** 那条 `e23f29b1-…`） |
| `ops:floor` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 楼层管理 |
| `ops:workstation` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 工位管理 |
| `ops:asset` | 2 | `import` `export` | 已有 F 型 4（`list`/`add`/`edit`/`delete`） | 播种 2 条，父 = **资产列表**（perms 为 NULL 的特例） |
| `ops:infopoint` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 信息点管理 |
| `ops:serverroom` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 机房管理 |
| `ops:dedicatedline` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 专线管理 |
| `ops:roomdevice` | 2 | `import` `export` | 已有 F 型 4 | 播种 2 条，父 = 机房设备管理 |
| `ops:walls` | 3 | `add` `edit` `delete` | **F 型 0 行** | 播种 3 条，父 = 楼层管理 |
| `ops:doors` | 3 | `add` `edit` `delete` | **F 型 0 行** | 播种 3 条，父 = 楼层管理 |
| `ops:floor-plan` | 1 | `edit` | **F 型 0 行** | 播种 1 条，父 = 楼层管理 |
| `network:device` | 6 | `list` `view` `add` `edit` `delete` `export` | **F 型 0 行** | 播种 6 条，父 = 设备管理 |
| `network:credential` | 6 | `list` `view` `add` `edit` `delete` `export` | **F 型 0 行** | 播种 6 条，父 = 授权凭证 |
| `network:template` | 6 | `list` `view` `add` `edit` `delete` `export` | **F 型 0 行** | 播种 6 条，父 = 配置模板 |
| `network:command` | 2 | `view` `export` | 已有 F 型 1（`execute`） | 播种 2 条，父集合 = {命令分发, 配置执行} |
| `network:backup` | 3 | `list` `delete` `export` | 已有 F 型 3（`add`/`restore`/`diff`） | 播种 3 条，父 = 配置备份 |
| `network:discovery` | 3 | `view` `delete` `export` | 已有 F 型 1（`add`） | 播种 3 条，父 = 设备发现 |
| `ops:manager:rule` | 3 | `list` `edit` `delete` | `list` = **C 型菜单 perms**；`edit`/`delete` = **migration_221 已建的 F 行** | **87 不播种**；白名单条目移除并注明 |
| `ops:manager:org` | 3 | `list` `edit` `delete` | 同上（`migration_221`） | **87 不播种**；白名单条目移除并注明 |
| **合计** | **78** | | | **72 条需播种 + 6 条已由 migration_221 覆盖** |

> 「已有 F 型 N 个」一栏用的是 **`check-permission-codes.mjs` 的口径 = distinct perms 数**（`menuFType` 是 Set）。**原始行数**在 ops 域是 2× —— 因库内存在「live（status=0）+ stopped（status=1）」两套平行菜单树，每套下各挂一套同名 F 行（见 §A3-1）。**system 4 组 / network device/credential/template / walls / doors / floor-plan 的 F 行数为 0**。

**白名单 78 项在「脚本可见性」上的分类（本会话实跑复算）**：

| 分类 | 条数 | 说明 |
|---|---|---|
| 已在 SQL 的 `fType` 集合 | **0** | 78 项**没有一项**在脚本的「菜单 F 型」集合里 |
| 已在 Go 的 `untyped` 集合 | **6** | 全部是 `ops:manager:*` —— 因为 `migration_221_manager_sync_menus.go:19-24` 的常量字面量被 Go 侧扫描捕获 |
| **两者都不在（真正待播种）** | **72** | 这 72 项**必须**出现在新 migration 的源码字面量里，否则清空白名单后 c 类 **exit 1** |

> ⇒ **这就是「清空白名单」对 87 的强制力**：72 码缺一即红。**但它只验「字面量存在」，不验「行真的是 F 型」** ⇒ §B9 的脚本改造必须补上。

### B8 · 85-03 产出的后端新码清单 vs `pkg/permission/config.go` / `sys_menu`

| 域 | 新增码数（85-03） | `pkg/permission/config.go` | `sys_menu` F 型 |
|---|---|---|---|
| system 4 组（roles/menus/depts/posts） | **23**（6+5+7+5） | **全部已有**（`config.go:22-52`，含 85-02 注释逐字：「85-02（RBAC-04）：部门 Excel 导入导出逐动作拆分新增… 菜单 F 型播种归 Phase 87。」`config.go:42-45`） | **0** |
| ops 域 import/export | **16**（building/floor/workstation/asset/infopoint/serverroom/dedicatedline/roomdevice 各 2） | 已有（`config.go` ops 区块） | **0** |
| ops 域 walls/doors/floor-plan 既有字面量 | **7** | 已有 | **0** |
| network 域 | **26** | 已有（`config.go:163-208`） | 仅 `backup:add/restore/diff`、`command:execute`、`discovery:add` 共 5 条 |
| manager 域 | **6** | 已有（`config.go:336-341`） | `migration_221` 运行期建（edit/delete）+ C 型 perms（list） |
| **合计** | **78** | **78/78 已有常量** | **72/78 无 F 行** |

> **差异点只有一处，且只有一种方向**：Go 常量 78/78 齐、TS 声明 78/78 齐、**`sys_menu` F 行缺 72**。⇒ 87 的播种清单 = 那 72 条。

### B9 · `check-route-registry.mjs` 与 Go 侧 `TestRouteRegistryCoverage` / `TestRouteRegistryFreshness` 的关系

| 事实 | 证据 |
|---|---|
| `scripts/check-route-registry.mjs` 位于**仓库根 `scripts/`**（**不在** `xingran-react-frontend/scripts/`） | `[VERIFIED: scripts/check-route-registry.mjs:24-26]` 逐字：「本脚本位于仓库根 scripts/ 目录（由 frontend lint:ci 以 `node ../scripts/check-route-registry.mjs` 调用），仓库根 = 本脚本目录的上一级。」 |
| 它**已经**在 `lint:ci` 链**尾** | `[VERIFIED: xingran-react-frontend/package.json:14]` `"lint:ci": "eslint . --max-warnings 0 && node scripts/assert-lint-budget.mjs && node scripts/check-exemption-reasons.mjs && node scripts/check-gating-coverage.mjs && node scripts/check-hardcoded-colors.mjs && node scripts/css-contract-check.js && node ../scripts/check-route-registry.mjs"` |
| 它的职责 | ① **freshness**：重跑 `scripts/scan-routes.cjs` 的扫描，与入库的 `scripts/route-registry.json` 逐条比对，diff 非空 → exit 1；② **coverage**：每条 `isWrite=true` 路由若未被 `GetRoutePermissions()` 登记（`registeredDeadTable=true`）且不在 `deferred.items` 冻结清单 → exit 1；已登记却仍挂在 deferred → exit 1（要求收缩）`[VERIFIED: check-route-registry.mjs:5-17, 39-89]` |
| Go 侧孪生守卫 | `pkg/permission/route_registry_test.go:86-125` `TestRouteRegistryCoverage`：读同一份 `scripts/route-registry.json`，逐条写路由断言 `GetPermissionByPath(r.Path, r.Method) != ""` 或在 deferred 清单；`route_registry_test.go:367` `TestRouteRegistryFreshness`（node 不在 PATH 时 `t.Skip("node 不在 PATH，freshness 重扫退化为 CI 侧 check-route-registry.mjs 承担")`） |
| 结论 | **两层守卫互补，不是二选一**：Go 侧 = 单测（`go test ./pkg/permission/`），CI 侧 = `lint:ci` 链尾的 mjs 脚本 |

**⇒ RBAC-06「校验脚本覆盖本轮新铺开面并接入 CI 门禁」的落地清单（prescriptive）**：

| # | 改动 | 文件 | 判据 |
|---|---|---|---|
| **R-1** | **清空 `PENDING_SEED_WHITELIST`（78 → 0）** | `xingran-react-frontend/scripts/check-permission-codes.mjs:57-143` | 删掉整个数组内容（保留 `const PENDING_SEED_WHITELIST = [];` 与 `PENDING_SEED_SET`），并把 `:47-56` 的注释改为「Phase 87 已播种，白名单永久清空；如需临时豁免须同批在 `86-STATE.md` 登记」 |
| **R-2** | **白名单非空 ⇒ 硬失败（防永久化）** | 同上 `:449-455` | 把「警告」改成 `errors.push("PENDING_SEED_WHITELIST 非空 …")` ⇒ 白名单非空即 exit 1（RBAC-06「零漂移」的机器化守卫） |
| **R-3** | **Go 侧 F 型识别（补 d 类盲区）** | 同上 `:374-381` | 新 migration 里放**机器可读块**，脚本解析后把码加入 `fType`（不是 `untyped`）⇒ ① d 类对 72 条生效（TS 未声明即红）② 「菜单 F 型 N 个」计数真实反映 F 型行 |
| **R-4** | **把脚本接进 `lint:ci`** | `xingran-react-frontend/package.json:14` | 在链中插入 `&& node scripts/check-permission-codes.mjs`（建议放在 `check-gating-coverage.mjs` 之后） |
| **R-5** | **把 `lint:ci` 接进真正的 CI** | `.github/workflows/frontend-build.yml:44-45` | 现在只跑 `npm run lint`；**必须**改成 `npm run lint:ci`（或新增一条 `run: npm run lint:ci` 的 step），否则 R-4 只是本地门禁不算「CI 门禁」。本仓 `.github/workflows/` **只有** `frontend-build.yml` 一个 workflow（已 ls 实证） |
| **R-6** | **本地 CI 复刻脚本同步** | `scripts/check-ci-local.sh:117` | `for s in format:check lint type-check test build` —— 若 R-5 把 workflow 改成 `lint:ci`，本地复刻脚本也应加 `lint:ci`，保持「本地绿 = CI 绿」 |

**R-3 的机器可读块格式（建议，planner 直接落进 PLAN）**：

```go
// F_TYPE_SEED_BEGIN —— check-permission-codes.mjs 解析本块，把里面的码当作
// 「已在 sys_menu 声明的 F 型按钮」参与 d 类校验（普通 Go 字面量只参与 c 类存在性）。
var actionPermissionFTypeSeedCodes = []string{
	"system:role:list",
	// … 72 条
}
// F_TYPE_SEED_END
```

> 脚本解析规则：`// F_TYPE_SEED_BEGIN` 与 `// F_TYPE_SEED_END` 之间的所有字符串字面量 → 进 `menuFType`。
> **同时**必须保证这 72 条**也**出现在普通 Go 扫描范围里（同一文件即可）⇒ c 类自动满足。

---

## C. 角色页授权 UI 的真实形态（Research Questions C10–C12）

### C10 · `menuIds` 表单字段

| 问 | 答 |
|---|---|
| 用什么组件 | **antd `Tree`**（不是 `TreeSelect`）—— `[VERIFIED: src/pages/system/role/index.tsx:17]` `Tree,` 来自 `import { … Tree } from "antd";`；`[VERIFIED: index.tsx:433-456]`：`<Form.Item name="menuIds" label="权限菜单" tooltip="选择角色可以访问的菜单（勾选父菜单时自动勾选所有子菜单）"><Tree checkable checkedKeys={checkedMenuKeys} onCheck={…} treeData={menuTree} … />` |
| 是否展示 F 型按钮节点 | **会展示**（见 C11 的后端证据） |
| 勾选后如何提交 | `onCheck` → `setCheckedMenuKeys(keyArray)` + `editForm.setFieldsValue({ menuIds: keyArray })`（`index.tsx:442-446`）→ `handleSave` 组装 `roleData`（含 `menuIds: (values.menuIds as string[]) || []`）→ `POST /system/roles/:id/update`（编辑）或 `POST /system/roles`（新增）`[VERIFIED: src/pages/system/role/hooks/useRoleActions.ts:181-201]` |
| 后端哪个 handler 接收 | `RoleHandler.Update` / `RoleHandler.Create`（`internal/api/v1/system/role_router.go:46-57`：`groups.Edit.POST("/:id/update", handler.Update)`、`groups.Add.POST("", handler.Create)`）→ `RoleService.UpdateRole` / `CreateRole`（`internal/services/system/role_service.go:118` 与 `:168` 调 `assignRoleMenusAndDepts`） |
| 写入哪张表 | **`sys_role_menu`**（`internal/services/system/role_service.go:392-431`：`tx.Table("sys_role_menu").Where("role_id = ? AND menu_id NOT IN ?", …).Delete(...)` + `tx.Clauses(clause.OnConflict{Columns: role_id/menu_id, DoNothing: true}).Create(&roleMenus)`） |
| 树数据来源 | `useRoleData().loadMenuTree()` → `POST /system/menus/tree-select`（`src/pages/system/role/hooks/useRoleData.ts:100-116`） |
| 已勾选的回读 | `POST /system/menus/role-menu-tree-select/{roleId}` → `{checkedKeys: []}`（`index.tsx:137-140`；后端 `internal/api/v1/system/menu_handler.go:361-378` → `GetRoleMenuIDs`） |

### C11 · **F 型按钮能否在角色页被勾选？—— 能。**

**Lead 的假设「treeData 只加载到 C 型菜单 ⇒ F 型按钮无法被授予」不成立。** 证据链：

1. 路由：`internal/api/v1/system/menu_router.go:32` → `r.POST("/tree-select", handler.GetTree)` `[VERIFIED]`
2. Handler：`internal/api/v1/system/menu_handler.go:82-90`
   ```go
   func (h *MenuHandler) GetTree(c *gin.Context) {
       tree, err := h.service.GetTree(c.Request.Context())
       …
       response.Success(c, tree)
   }
   ```
   `[VERIFIED]`
3. Service：`internal/services/system/menu_service.go:181-190`
   ```go
   func (s *menuService) GetTree(ctx context.Context) ([]models.Menu, error) {
       var menus []models.Menu
       if err := s.db.WithContext(ctx).Order("order_num ASC").Find(&menus).Error; err != nil {
           return nil, fmt.Errorf("查询菜单列表失败: %w", err)
       }
       // 构建树形结构
       menuTree := s.buildMenuTree(menus, nil)
       return menuTree, nil
   }
   ```
   **无 `menu_type` 过滤、无 `visible` 过滤、无 `status` 过滤** ⇒ `M`/`C`/`F` **全部**返回 `[VERIFIED]`
4. `buildMenuTree`（`:345-357`）按 `parent_id` 递归挂树 ⇒ F 行作为 C 行的子节点出现 `[VERIFIED]`
5. 前端 `processTreeData`（`useRoleData.ts:103-111`）只映射 `key` / `title` / `value` / `children`，**不按 menuType 过滤** ⇒ F 节点作为叶子节点渲染，`title` = `menu_name` `[VERIFIED]`

> **结论**：**F 型按钮在角色页是可见、可勾选的**。真正阻塞 RBAC-05 的是**数据缺失**（72 个码在 `sys_menu` 里根本没有 F 行）⇒ **角色页无需任何前端改造**。
>
> ⚠️ 但有两个**必须知晓**的交互事实：
> ① **antd `Tree` 默认 `checkStrictly=false`**（父子联动）：勾一个 F 按钮会把父 C 菜单一并勾上（`onCheck` 返回的 keys 含半选父节点）⇒ 提交时 `menuIds` 同时含父菜单 id —— 这是**期望行为**（父页面权限随之获得）。
> ② **「antd 父子联动陷阱」**（`menu_grant_helpers.go:9-12` 注释逐字）：「仅 INSERT sys_menu 不 INSERT sys_role_menu 会被父联动带飞视觉但实际 checkedKeys 不含 → 路由不生成 → 链接 fallback dashboard。」⇒ **播种必须同时写 `sys_role_menu`**，否则新 F 节点在树上「看起来被勾上」但 `checkedKeys` 不含 → 路由不生成。**这正是用户锁定的「播种 + 授权」两件事必须同批做的原因。**

### C12 · 后端菜单树查询是否过滤 `menu_type`？

**不过滤。** 位置与行号：

| 文件 | 行号 | 事实 |
|---|---|---|
| `internal/services/system/menu_service.go` | **181-190** | `GetTree`：`s.db.WithContext(ctx).Order("order_num ASC").Find(&menus)` —— **无 WHERE 条件** ⇒ 无 menu_type / visible / status 过滤 |
| `internal/services/system/menu_service.go` | **345-357** | `buildMenuTree` —— 只按 parent_id 递归，不看 menu_type |
| `internal/api/v1/system/menu_handler.go` | **82-90** | `GetTree` handler 直接透传 service 结果 |
| `internal/api/v1/system/menu_router.go` | **32** | `r.POST("/tree-select", handler.GetTree)` |

> **对照**：真正过滤 `menu_type` 的是**用户菜单**接口，不是角色页树：
> `pkg/permission/service.go:221` 与 `:247` —— `AND m.menu_type IN ('M', 'C')`（用户导航菜单不含 F 型，正确）。
> `pkg/permission/service.go:302-316`（`GetUserPermissions`）与 `internal/services/system/menu_service.go:551` —— **不过滤 menu_type** ⇒ F 型码进入权限集。

---

## D. 后端权限判定链路（Research Questions D13–D14）

### D13 · 一个 F 型 perms 被勾选后，后端在哪一行把它变成可放行的权限集？

| 步 | 位置 | 逐字事实 |
|---|---|---|
| ① 落库 | `internal/services/system/role_service.go:417-431` | `tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "role_id"}, {Name: "menu_id"}}, DoNothing: true}).Create(&roleMenus)` ⇒ `sys_role_menu` 新增 `(role_id, menu_id)` 行 |
| ② 权限集（前端侧） | `internal/services/system/menu_service.go:551` | `Where("id IN ? AND perms IS NOT NULL AND perms != '' AND status = ?", menuIDs, models.MenuStatusNormal)` —— 该 `menuIDs` 来自 `sys_role_menu`（`:527-528`）⇒ **F 型行的 perms 出现在权限集** |
| ③ 权限集（后端 pkg/permission 版） | `pkg/permission/service.go:304-313` | `SELECT DISTINCT m.perms FROM sys_menu m INNER JOIN sys_role_menu rm … WHERE ur.user_id = ? AND m.perms IS NOT NULL AND m.perms != '' AND m.status = ?` |
| ④ **放行的那一行** | **`pkg/middleware/permission.go:451-457`** | ```sql
   SELECT COUNT(DISTINCT rm.menu_id)
   FROM sys_user_role ur
   INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
   INNER JOIN sys_menu m ON rm.menu_id = m.id
   WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?
   ``` —— `checkUserPermissionStrict` 的**唯一**判定点（`m.perms = ?` 精确匹配） |
| ⑤ 中间件挂载点 | `pkg/middleware/permission.go:411-437` `RequireStrictPermissions`（逐动作单一码）；`:311-337` `RequirePermissions`（ANY-of 组级）；`:350-384` `RequirePermissionsWithQuery`（读路径额外并入 `queryExtraPermissions`，**写路径不受影响**——`isQueryPath` 只认末段 `list`/`tree`，`:388-396`） | 85 已把写路由全部换成 Strict（见 `internal/api/router.go:246-253 / 266-273 / 300-311` 的 `strictUserRouteGroup` 兄弟组） |
| ⑥ Step 2 继承（读路径） | `pkg/middleware/permission.go:214-256` `hasInheritedButtonPerm` | 「查 `perms = permission AND status = normal` 的菜单 → 查其 `parent_id = menu.ID AND menu_type='F' AND status = normal` 的子按钮 → 用户持有任一子按钮 perms 即放行父页面权限」。**方向：持 F 型按钮 → 继承父页面权限（反向不成立）** |
| ⑦ admin bypass | `pkg/middleware/permission.go:281-302` `isSuperAdmin` | `WHERE ur.user_id = ? AND r.role_key = ?` 传 `"admin"` ⇒ **夹具 role_key 绝不能是 `admin`** |
| ⑧ 缓存 | `pkg/middleware/permission.go:33` | `const mwPermCacheTTL = 30 * time.Second`；`InvalidateMiddlewarePermCache`（`:67-83`）**未接线**（`:68` 注释逐字：「当前未接线 — 30s 短 TTL 陈旧已被接受」） |

> **一句话回答 D13**：**`pkg/middleware/permission.go:451-457` 的 `m.perms = ?`** —— 只要 `sys_menu` 里有该 perms 的 F 行、且 `sys_role_menu` 里有对应关联行，`RequireStrictPermissions` 立即放行。

### D14 · `pkg/permission/config.go` 的动作映射表与菜单 F 型 perms 的关系

| 问 | 答 |
|---|---|
| 是同一套码还是两套？ | **同一套码**。`RoutePermission.Permission` 的类型就是 `PermissionCode`，取值全是 `config.go:10-342` 的常量（逐字例：`{"/system/roles", "POST", RoleAdd, "新增角色"}`，`config.go:373`；`RoleAdd PermissionCode = "system:role:add"`，`config.go:24`）。同一个字符串既是**路由→权限码映射**的右端，也是 **`sys_menu.perms` 的值**，也是 **`permissionCodes.ts` 的值** |
| 85-03 接线状态（RBAC-03） | **已接线**。`GetRoutePermissions()`（`config.go:353-650`）与 `GetPermissionByPath(path, method)`（`config.go:653-662`）现在**有两个消费点**：① CI 侧 `scripts/check-route-registry.mjs`（freshness + coverage）② Go 侧 `pkg/permission/route_registry_test.go:86-125` `TestRouteRegistryCoverage`（断言 `GetPermissionByPath(r.Path, r.Method) != ""`）+ `:321` `TestStrictWriteGroupsSingleCode`（写组必须单码）。**运行时判定链走的是 `RequireStrictPermissions` 的字面量码，不是查表** —— 表的职责是「防新增写路由漏登记」（85-RESEARCH §A'.1 的「构建期/CI 校验」方案，运行时查表方案已被否决） |
| 映射表与 F 型菜单的关系 | 映射表 = **路由 → 码**；F 型菜单 = **码 → 授权来源**。两者通过「同一个码字符串」耦合。`check-permission-codes.mjs` 的 b 类（Go 常量 ↔ TS）与 c/d 类（TS ↔ 菜单种子）把三者绑在一起 |
| 匹配语义 | `matchPath`（`config.go:665-689`）：**精确串相等 或 段数一致逐段比较（`:id` 占位段跳过比较）**；**不支持前缀/通配**；不区分 trailing slash；方法必须完全相等（表内全为 `POST`） |

---

## E. 三层证据与决定性对照实验（Research Questions E15–E19）

### E15 · DB 前后对照 —— 夹具模板 `.planning/notes/2026-10-29-v124-perm-fixture.md`

> 已完整读（931 行）。**可复用的步骤**（planner 直接落进 PLAN）：

| 步 | 内容 | 出处（行号） |
|---|---|---|
| 0 | **环境事实**：PG `10.62.10.34:5432/ruoyi`（生产）；`.env` 不可读（口令须由环境变量注入）；`postgres://postgres@10.62.10.34:5432/ruoyi?sslmode=disable` 网络可达但口令不符（`FATAL: password authentication failed for user "postgres" (SQLSTATE 28P01)`） | §1.1（:37-61） |
| 1 | **夹具工具**：仓库外临时 Go 程序（`pgx/v5`），目录 `%TEMP%\pgfix\`，含 `main.go` / `10_fixture_role.sql` / `11_fixture_user.sql` / `12/13_verify_role*.sql` / `v_simulate_perm_check.sql` / `90_cleanup.sql` / `run_all.sh`；**验证后必须整体删除**（`Remove-Item -Recurse -Force "$env:TEMP\pgfix"`） | §1.1（:39-53） |
| 2 | **超管 bypass 口径核实**（代码级）：判定键是 `r.role_key = 'admin'`；夹具 role_key 用 `v124_audit_test`（不撞名）；**验证账号只挂本受限角色**（不挂 admin、不挂任何既有角色） | §1.2（:69-83） |
| 3 | **造数（只 INSERT，幂等 `WHERE NOT EXISTS`，不改任何既有行）**：① `sys_role` 插 `role_key='v124_audit_test'`、`data_scope=3`、`status=0`；② `sys_role_menu` 绑**页面级 C 菜单**（`perms='system:user' AND menu_type='C' AND status=0`）+ 其父目录；③（可选）`sys_user` 插测试账号（SM3 哈希 `$sm3$600000$<b64salt>$<b64hash>`，由临时工具 `pgfix.exe hash` 生成并经环境变量注入，**SQL 文件里不留任何口令/哈希**）；④ `sys_user_role` 绑**唯一**角色；⑤ 自检查询（role_cnt=1、role_keys 不含 admin） | §2.1-2.3（:153-251） |
| 4 | **DB 层前后对照（只读 SQL 复刻中间件三级判定）**：`v_simulate_perm_check.sql` 对每个候选 perm 给出 `step1_exact` / `step2_child_inherit` / `step3_module_match` / `legacy_any_pass` / `strict_exact_pass`；**期望：写权限点 step1=false 且 strict_exact_pass=false → 403** | §3.4 / §3.8.3（:295-322, 544-587） |
| 5 | **「无数据变更」判据**：写接口调用前后 `SELECT count(*)` 目标表一致 | 85-RESEARCH §E1 第 3 条 |
| 6 | **清理 SQL（单事务，先子后父）**：`sys_user_post` → `sys_user_role` → `sys_user` → `sys_role_menu` → `sys_role_dept` → `sys_role`；只按 `role_key`/`username` 精确定位 | §4（:691-719） |
| 7 | **清理后自检（必须全 0）**：`SELECT (SELECT count(*) FROM sys_role WHERE role_key=…), (… sys_user …), (… sys_role_menu JOIN sys_role …), (… sys_user_role JOIN sys_user …)` | §4（:712-718）+ §4.1 实测表（:737-746） |
| 8 | **全库总量对照**：清理前 `sys_role=7 / sys_user=4232 / sys_user_role=4242 / sys_role_menu=353`；清理后 `sys_role=6 / sys_user=4231` ⇒ 差值恰为夹具行数 ⇒ **其余既有行零改动** | §3.8.6（:651）+ §4.1（:748-749） |
| 9 | **并发干扰硬规则**：T1/T4 取证**前后各跑一次权限集快照**，两次数值一致才认；同一夹具不得被两个会话同时使用；对照实验与取证**必须串行** | §3.9（:653-686） |
| 10 | **已知坑（必读）**：① `sys_menu` 查 **`perms = 'system:user'` 精确匹配**，**禁用 `LIKE 'system:user%'`**（会捞进 2 条隐藏 C 菜单）；② `sys_role_menu` **只有普通索引非 unique** ⇒ 一律 `NOT EXISTS`；③ **模型未映射 ≠ 库表无列**（生产库 `sys_role_menu` 实有 `id`/`created_at`）；④ `sys_user.id` 是 `uuid`（非 `user_id`）、`nickname`（非 `nick_name`） | §3.6（:436-457）+ §3.7（:477-486） |

### E16 · HTTP 实测（如何在 AD + 滑块验证码阻断下拿到 403/200）

**结论：走「本地起后端 + 自签 JWT」的既有配方**（v1.24 Phase 79 已实测跑通，见 note §3.4.2）。

| 项 | 值 | 出处 |
|---|---|---|
| 后端端口 | **9000**（`configs/config.yaml:5` `port: 9000`；`host: "0.0.0.0"`） | `[VERIFIED]` |
| 前端 dev server 端口 | **4000**（`xingran-react-frontend/vite.config.ts:242` `port: 4000`），proxy → `http://localhost:9000`（`:247`、`:253`） | `[VERIFIED]` |
| DB / Redis | `10.62.10.34:5432` / `10.62.10.34:6379`（`configs/config.yaml:12-13, 27-28`） | `[VERIFIED]` |
| **AD 阻断** | 生产 `sys.auth.default.mode = ad` ⇒ 登录通道不可用 | note §3.4.2（:361） |
| **滑块阻断** | 生产 `sys.account.captchaEnabled=slider` ⇒ 口令登不进去 | note §3.4.2（:361） |
| **是否有无需登录/无需 AD 的验证路径** | **有**：① **自签 token**（`crypto.GenerateTokenWithSM2`，**issuer 必须为 `RuoYi-Next`**）—— 用与前端一致的 SM2 密钥对本地自签；② **只读 SQL 等价判据**（note §3.8.2/§3.8.3，**全部只 SELECT**，可用 `SET TRANSACTION READ ONLY` 兜底）—— **当生产后端不可用或不允许起实例时，这是当前唯一无需起实例即可复测的路径** | note §3.4.2（:356-362）、§3.8（:487-587） |
| **API Key 认证路径** | **不存在**。`pkg/middleware/` 全目录 grep `X-API-Key|api_key|APIKey|apikey` = **0 命中** ⇒ **没有 API-Key 认证中间件**。`internal/api/v1/system/apikey_router.go` 是「API 密钥管理」业务端点，不是认证通道 | `[VERIFIED: pkg/middleware/ 无匹配；apikey_router.go:17-36]` |
| 起后端的最小旁路（env） | `SERVER_MODE=test`（release 下 `SKIP_AUTOMIGRATE=true` 会 fatal）+ `SKIP_AUTOMIGRATE=true` + `SERVER_SKIP_SETUP=true` + `DB_PASSWORD` + `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`（**hex**；⚠️ **不是** `RUOYI_JWT_SM2_*`，真实绑定见 `internal/config/config.go:364-365`） | note §3.4.2（:353-362, 412-427）+ §3.6（:450-457）+ `[VERIFIED: internal/core/core.go:286-291]` |
| 请求体加密 | 生产 `sys.request.encryption.enabled=false` ⇒ **明文 JSON 直通** | note §3.4.2（:362） |
| **「生产后端不可用」的实测事实** | `GET http://10.62.10.34:9000/` → **403** + `Server: MinIO`（9000 端口是 MinIO 不是后端）；`GET http://10.62.10.34/` → **502 Bad Gateway**（nginx/1.18.0）⇒ **生产后端进程当前未运行** ⇒ 走生产 HTTP 复测**做不到** | note §3.8.1（:495-503） |
| **本地起实例的副作用（如实记录）** | `sys_oper_log`（delete/update 各 1 行）、`ad_push_queue`（1 行）、`sys_menu` / MAC 物化视图（`REFRESH MATERIALIZED VIEW CONCURRENTLY`）、Redis 缓存预热；另有 `cmd/main.go:358 importOUIData`（批量写 OUI）与 `initMACHistoryRetentionConfig`（可能 INSERT `sys_config`）⇒ **「不修改既有数据」边界会被突破** | note §3.4.2（:405-408）+ §3.8.1（:501-503） |
| **缓存等待** | 后端 30s TTL ⇒ 生产常驻实例上复测**必须等 31s** 或重启；本地新进程缓存为空 ⇒ **无需等** | note §3.8.5 前的 §5.0（:761-769） |

**⇒ planner 的 HTTP 实测路径（三选一，按优先级）**：

1. **【推荐】只读 SQL 等价判据**（note §3.8.2 / §3.8.3 模板，**零副作用**，可对生产库直接跑，只 SELECT + `SET TRANSACTION READ ONLY` 兜底）⇒ 产出 DB 层前后对照。
2. **本地起后端 + 自签 token + curl**（note §3.4.2 配方）⇒ 产出真实 HTTP 403/200。**注意副作用登记**。
3. **生产常驻后端 + 31s 等待** —— **当前不可行**（生产后端进程未运行，502）。

### E17 · 组件测试（86 线 `permission.gate.test.tsx` 范式 → 角色授权场景应写什么）

**86 线的 8 页抽样范式**（`src/pages/system/role/__tests__/permission.gate.test.tsx` 本身就是其中一页）：
- 三态构造器：`setPermissionState(permissions, { loaded, error })` → 直接 `useMenuStore.setState({ permissions, permissionsLoaded, permissionsError })` —— **禁止**用 `permissions.length === 0` 判未就绪（文件头注释逐字：「⚠️ 禁止用 `permissions.length === 0` 判「未就绪」」）`[VERIFIED: permission.gate.test.tsx:9-11, 43-52]`
- API mock：`mockApiBatch([...])` + `setGenericFallback({ data: {} })`，`vi.mock("@/lib/api", …createApiTestingModule())` `[VERIFIED: :25-28, 69-83]`
- 断言三路径：**无权限 → 按钮不在 DOM**（`queryByRole(...)` 为 null）/ **未就绪 → 在 DOM 且 `toBeDisabled()` 且 `title` 含「权限」** / **有权限 → 存在且 `not.toBeDisabled()`** `[VERIFIED: :85-184]`
- 第四路径：**ready 空权限集** vs **未就绪** 可区分（`A8-1`）`[VERIFIED: :186-205]`
- 渲染：`renderWithProviders(<RoleManagement />, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) })` `[VERIFIED: :54-58]`
- 超时 30s（整页 + antd Table 慢）`[VERIFIED: :206-207]`

**角色授权场景（87）应写的测试（prescriptive）**：

| # | 用例 | 断言 | 归属文件 |
|---|---|---|---|
| T-1 | **授权闭环正向**：mock `/system/menus/tree-select` 返回一棵含 F 型节点的树（`title='新增'`、`key='f-add'`），打开「编辑角色」→ 勾选该 F 节点 → 提交 → 断言 POST `/system/roles/:id/update` 的 body 里 `menuIds` **含该 F 节点 key** | 前端「勾选 → 提交」链路通 | `src/pages/system/role/__tests__/permission.grant.test.tsx`（新建） |
| T-2 | **父联动语义**：勾 F 节点后，断言提交的 `menuIds` **同时含父 C 菜单 id**（antd `checkStrictly=false` 联动） | 锁定 antd 父子联动行为，避免误改 | 同上 |
| T-3 | **F 节点在树中可见**：断言 F 节点的 `title` 出现在 DOM（证明 `GetTree` 不过滤 menu_type） | 锁定 C11 的结论 | 同上 |
| T-4 | **回读一致**：mock `/system/menus/role-menu-tree-select/:id` 返回含 F 型 id 的 `checkedKeys` → 断言该 F 节点呈勾选态 | 锁定回读链路 | 同上 |
| T-5 | **三态不回归**：把 86 的 A5/A6/A7/A8 四路径**原样搬进 87 的新文件**（或确认 `permission.gate.test.tsx` 未被改动即天然回归） | v1.24 三态契约零破坏 | 复用现有文件即可 |
| T-6 | **门控消费闭环**：`useMenuStore.permissions` 含 `system:role:add` 时「新增角色」可见 —— 这是 86 已覆盖的，87 只需回归 | 前后端同源的前端侧 | 现有 `permission.gate.test.tsx` A7-1 |

> **组件测试能证明什么、不能证明什么**（必须写进验收报告）：
> 能证明：**F 型节点在角色页可被勾选 → `menuIds` 携带 → 提交体含 F 型 id**。
> **不能**证明：后端真的放行（那是 HTTP 层）、DB 里真有 F 行（那是 DB 层）。**三层证据不可互相替代。**

### E18 · 决定性对照（admin bypass 与最小实验步骤）

| 问 | 答 |
|---|---|
| admin bypass 位置 | `[VERIFIED: pkg/middleware/permission.go:281-302]` `isSuperAdmin`：`SELECT COUNT(*) FROM sys_user_role ur INNER JOIN sys_role r ON ur.role_id = r.id WHERE ur.user_id = ? AND r.role_key = ?` 传 `"admin"`（`:293`）。**判定键是 `role_key = 'admin'`，不是 role_name** |
| 夹具角色如何避免撞名 | `role_key` 用 **`p87_*` 前缀**（如 `p87_probe_test`），**绝不含 `admin` 子串**；且**夹具用户只挂这一个角色**（不挂 admin、不挂任何既有角色）—— 否则 `sys_user_role` 多角色会让 `isSuperAdmin` 命中。见 note §1.2（:82-83） |
| 最小实验步骤 | 见 §Code Examples E4（6 步）。核心：③ 403 → ⑤ 临时补授 F 型 `menu_id` → 等 31s / 重启 → 同一请求 200 且真实生效 → ⑥ 撤销 → 回 403 |
| 补授的两种方式 | ① **DB 直插** `INSERT INTO sys_role_menu (role_id, menu_id) SELECT … WHERE NOT EXISTS`（最直接，可控可撤）；② **UI 侧在角色页勾选**（更贴近真实闭环，但需要 admin 登录态，而 admin 走 AD + 滑块阻断 ⇒ **实际不可行**）。⇒ **87 用 ①** |
| 排除「接口坏了」的证据形态 | 同一端点、同一 token：**无权 403 且数据不动** / **有权 200 且真生效** ⇒ 403 确系权限门按角色生效，不是接口/路由/参数问题（note §5.3 对照行，:854） |

### E19 · 85-RESEARCH §B4 的判定 SQL（生产 6 角色回归面）

> ⚠️ **位置更正**：Lead 说的「§B4 的判定 SQL」实际在 **§B' 第 4 条**（`85-RESEARCH.md:103-114`）；§B4（:71-72）只给结论（「完全没有 F 型按钮的模块：`system:role`、`system:menu`、`system:dept`、`system:post`…」）与 `tmp_ftype.json` 口径（131 个 F 型码）。以下逐字摘出。

**§B' 第 4 条的 SQL 原文**（`85-RESEARCH.md:105-113`）：

```sql
-- 某域拆分前：哪些角色依赖 ANY-of 放行写操作
SELECT DISTINCT r.role_key FROM sys_user_role ur
JOIN sys_role_menu rm ON ur.role_id=rm.role_id
JOIN sys_menu m ON rm.menu_id=m.id
WHERE m.perms IN (<组级读码列表>)   -- 如 ops:building:list
  AND NOT EXISTS (SELECT 1 FROM sys_role_menu rm2 JOIN sys_menu m2 ON rm2.menu_id=m2.id
                  WHERE rm2.role_id=ur.role_id AND m2.perms = <对应写码>);
```

**生产基线（§B' 第 4 条引用 `.planning/notes/2026-10-28-pg-permission-baseline.md §1`）**：
> 「生产基线（HIGH，.planning/notes/2026-10-28-pg-permission-baseline.md §1）：仅 6 个角色（admin/kjgl/xtgl/adyk/vm_user/user），admin bypass；user 写权限=0。」

**6 角色的实测权限规模**（`2026-10-28-pg-permission-baseline.md:8-15` 逐字）：

| role_key | role_name | 菜单数 | 写权限数 |
|---|---|---|---|
| admin | 超级管理员 | 246 | 71 |
| kjgl | 空间管理 | 40 | 14 |
| xtgl | 系统管理 | 34 | 14 |
| adyk | AD域控 | 15 | 4 |
| vm_user | 虚拟机一般用户 | 11 | 2 |
| user | 普通用户 | 5 | **0** |

**「受影响生产角色清单」的判定 SQL（87 必须跑，逐条列明）**：

```sql
-- 87 播种后：哪些生产角色会因「被授予新 F 型动作码」而新增写能力
-- （= 持有对应父 C 型菜单的角色 = 85 之前靠 Step3 模块降级就能写的角色）
SELECT r.role_key, r.role_name, m.menu_name AS parent_menu, m.perms AS parent_perms, COUNT(*) AS n
FROM sys_role_menu rm
JOIN sys_role r  ON r.id = rm.role_id
JOIN sys_menu m  ON m.id = rm.menu_id
WHERE m.menu_type = 'C' AND m.status = 0
  AND m.perms IN ( 'system:role','system:menu','system:dept','system:post',
                   'ops:building:list','ops:floor:list','ops:workstation:list',
                   'ops:infopoint:list','ops:serverroom:list','ops:dedicatedline:list',
                   'ops:roomdevice:list','network:device','network:credential',
                   'network:template','network:command','network:backup','network:discovery',
                   'ops:manager:rule:list','ops:manager:org:list' )
GROUP BY r.role_key, r.role_name, m.menu_name, m.perms
ORDER BY r.role_key, m.perms;
```
> 判定口径：**该清单必须为空或逐条列明**（ROADMAP Phase 87 SC5 逐字：「生产库**既有行零改动**：验收夹具在单事务内清理，清理后自检残留 = 0；受影响生产角色清单为空或逐条列明。」`ROADMAP.md:98`）。

---

## F. 86 线的遗留移交（Research Questions F20–F22）

### F20 · 51 条豁免如何被 87 消解

| 类别 | 条数 | 页面 | **87 的处置** |
|---|---:|---|---|
| `no-permission-code` | **26** | system/{dict, notice, config, apikeys, captcha-background}、duty/{config, holidays, pools, schedules, management}、workorder/{categories, orders, periodic/templates}、knowledge/articles、operations/rpa/{tasks, executions}、monitor/{cache, job, logs, server}、vdi/VDIServerConfig、network/mac、ad-domain/{ous, computers}、ad/SyncMonitor、asset/reconciliation/exception-rules | **全部保留为永久 N/A 或另立项** —— 理由：① 后端常量缺失的域（duty/workorder/knowledge/vdi/rpa/asset 旧域）—— 85 已明确为范围外域（`85-RESEARCH.md:166`「无权限中间件的模块…属授予闭环前置，超出 RBAC-01 字面」）；② 后端常量已存在但 **TS 未声明**的域（system dict/notice/config/apikey/captchaBackground、monitor 全域、network mac/port）—— 补 TS 声明属于**前端真相源补齐**，不是「F 型播种」，87 的播种清单（白名单 78 项）**不含**这些码 ⇒ **87 不消解它们**，应作为独立后续项登记 |
| `legacy-two-source` | **5** | network/ports（`network:port:write`）、network/mac/history（`network:mac:export`）、vdi/VirtualMachineList（`vdi:vm:*` × 7）、asset/reconciliation/exceptions（`asset:reconciliation:resolve`）、asset/reconciliation/fix-suggestion（`asset:reconciliation:fix:{accept,reject,rollback}`） | **87 不消解** —— 这些码**不在**白名单 78 项内；且其中 4 个域后端**根本没有对应权限码**（见 F21）⇒ 属「后端先出码 → 再补 TS → 再播种 → 最后收口前端判定」的**三段式后续链路**，非 87 范围。**87 只负责登记**（写进 87-STATE 的移交清单） |
| `read-only` | **12** | workorder/statistics、knowledge/view、duty/my-duty、operations/rpa、operations/rpa/workers、vdi/VirtualMachineDetail、ad-domain/logs、asset/reconciliation、asset/reconciliation/dashboard、operations/building-spaces、operations/building-spaces-3d、monitor/dashboard | **永久 N/A**（无写入口）。**87 不动** —— 改动会打破 `check-gating-coverage.mjs` 的 `EXPECTED_EXEMPTION_COUNT = 51` 恒等式（`25 + 51 = 76`，`check-gating-coverage.mjs:61-63, 280-299`） |
| `non-route-page` | **8** | login、profile、settings、system/settings、system/settings-page、my-notices、dashboard-system、knowledge/articles/modals | **永久 N/A**（非路由页）。**87 不动**（同上） |

> **总结论**：**87 消解 0 条豁免**。51 条全部保持原样，`check-gating-coverage.mjs` 的 `EXPECTED_EXEMPTION_COUNT = 51` 与 `EXPECTED_PAGE_COUNT = 76` **不得改动**。
> **87 唯一与之交互的动作**是：在 `87-STATE.md` 里登记「51 条豁免的后续消解路径」，并把 `no-permission-code` 中「后端已有常量、只缺 TS 声明」的 5+28 码列为**下一 phase 的播种候选**。

### F21 · 6 处旧式直判（第二真相源）

| # | 位置 | 用的字面码 | 后端 `pkg/permission/config.go` 有对应常量？ | `sys_menu` 有 F 行？ | 归口 |
|---|---|---|---|---|---|
| 1 | `src/pages/network/ports/index.tsx:130-132` | `network:port:write` | **✅ 有** —— `[VERIFIED: pkg/permission/config.go:217]` `NetworkPortWrite PermissionCode = "network:port:write"` | ✅ 有（migration_202 建的「端口配置」F 行，perms=`network:port:write`） | 后端动作级拆分 phase（拆 `network:port:{list,edit,…}`）+ 补 TS `networkPort` 区块 |
| 2 | `src/pages/network/mac/history/MACHistoryPage.tsx:117-119` | `network:mac:export` | **❌ 无该码常量** —— config.go 只有 `[VERIFIED: config.go:211]` `NetworkMacQuery PermissionCode = "network:mac:query"` 与 `[config.go:227]` `NetworkMac PermissionCode = "network:mac"` | ❌ 无 | **后端先出码**（`network:mac:export`），再补 TS，再播种 |
| 3 | `src/pages/vdi/VirtualMachineList/index.tsx:46,52,109,929-935` | `vdi:vm:add/bind/remove/restart/start/stop/sync`（7 个） | **❌ 无 `vdi:` 前缀常量**（config.go 无 `vdi:` 任何常量） | ⚠️ **部分有**：`menu_catalog_seed.sql:86` 有 `vdi:vm:add`（F 行，id `770e8400-…-004`）；其余 6 个无 | **死判定**（前端判的是永远拿不到的码）⇒ 后端先出码 |
| 4 | `src/pages/asset/reconciliation/exceptions/index.tsx:85-86` | `asset:reconciliation:resolve` | **❌ 无** | ⚠️ 库内有 `asset:reconciliation:markResolved`（F 行，seed 行）—— **码名不同** | 后端先出码 |
| 5 | `src/pages/asset/reconciliation/fix-suggestion/index.tsx:64-67` | `asset:reconciliation:fix:{accept,reject,rollback}` | **❌ 无**（config.go 无 `asset:` 前缀） | ✅ **有 3 条**：`menu_catalog_seed.sql:234` `asset:reconciliation:fix:accept`；`migration_200:164-168` 建 accept/reject/apply/rollback/stats | 后端补常量 + 补 TS `assetReconciliation` 区块 + 播种（apply/stats 亦已在库里） |
| 6 | `src/pages/ad-domain/configs/ReconciliationTab.tsx:42,63-64` | `ops:ad:config:reconciliation:list` | **❌ 无**（config.go 无该码；`ops:ad:config:*` 只有 add/delete/edit/list/sync/test） | ✅ **有** —— `[VERIFIED: migration_218_ad_reconciliation_permission.go:17]` `const adReconciliationListPerms = "ops:ad:config:reconciliation:list"` + migration_218 建 F 行 | 已在 migration_218 播种；后端 `ad_domain_router.go:177` 现挂该只读码 ⇒ **只差后端补常量 + 补 TS** |

> **Lead 问的「后端有 network:port:* 等常量但前端无区块」的准确答案**：**只有 `network:port:write` 一条既有常量又有 F 行**；其余 5 处后端常量要么**完全没有**、要么**码名不同**。⇒ **6 处旧式直判没有一条能被 87 的播种顺手消解**（它们需要的是「后端出码 → 补 TS → 播种」的完整链路，且其中 4 处连第一步都没做）。
> 另：86-STATE.md §16 表（第 2 行）称 `network:mac:export`「后端 ✅ 有 `network:mac:` 前缀」—— **本会话实测：该前缀下只有 `:query` 与 `:mac`，没有 `:export`**。此为 86-STATE 表述不精确之处，87 的移交清单应按实测口径写。

### F22 · `86-STATE.md` 第十节「缺码待回补登记」完整清单

> `[VERIFIED: .planning/phases/86-frontend-gating/86-STATE.md:234-246]`

原文表格逐字转录：

| 域 | TS 清单现状 | 后端 `pkg/permission/config.go` 现状 | 待回补内容 |
|---|---|---|---|
| system 的 dict / notice / config / apikey / captchaBackground | 无对应区块 | **有** `system:dict:*` / `system:notice:*` / `system:config:*` / `system:apikey:*` / `system:captchaBackground:*` | 补 TS 区块（后端已有常量，只缺前端声明） |
| monitor 全域 | 无 monitor 区块 | 有 `monitor:cache:*` / `monitor:job:*` / `monitor:log*` / `monitor:server:*` 等 28 条 | 补 TS 区块 |
| network 的 mac / port | 无 networkMac / networkPort 区块 | 有 `network:mac:*` / `network:port:*` | 补 TS 区块 |
| ad-domain 的 ou / computer | AD 域仅 `opsADConfig` / `opsADGroup` / `opsADUser` | `ops:ad` 下无 ou / computer 操作码 | 需后端先拆出操作码 |
| `ad-domain/configs/ReconciliationTab` 的 `ops:ad:config:reconciliation:list` | 不在 TS 清单 | 后端无该码 | **旧式直判**，随 W4 收口（不豁免） |
| duty / workorder / knowledge / vdi / rpa / asset（旧域） | 无对应区块 | **后端也没有**对应前缀（85 已明确为范围外域） | 需后端拆分 phase 整体拆分 |

> §86-STATE 第十节开头逐字：「86 全程**不改** `src/constants/permissionCodes.ts`（Lead 裁定 4）。下列写域在 TS 清单无对应区块，已按 `no-permission-code` / `legacy-two-source` 登记豁免，**等后端拆分 phase 回补 TS 声明后逐条移除豁免**」
> ⇒ **87 与第十节无交集**：第十节全部是「TS 侧缺声明 / 后端缺码」，与 87 的「播种 72 条 F 行」是**正交**的两件事。**87 不得改 `permissionCodes.ts`**（那会打破 86 的「零改动核验」`git diff --stat -- src/constants/permissionCodes.ts` 必须为空，见 `86-STATE.md:266`）。

---

## G. 生产影响边界（Research Questions G23–G24）

### G23 · 「生产库既有行零改动」与播种如何自洽

| 问 | 答 |
|---|---|
| 播种是新增行吗 | **是**。`sys_menu` 新增 72 行（F 型）+ `sys_role_menu` 新增 N 行（授权）。**不 UPDATE / 不 DELETE 任何既有 `sys_menu` / `sys_role_menu` / `sys_role` / `sys_user` 行** |
| 与「既有行零改动」如何自洽 | SC5 的原文是「验收夹具在单事务内清理，清理后自检残留 = 0；受影响生产角色清单为空或逐条列明」——**「既有行零改动」指既有行内容不被改写**，新增行不违反。但**「受影响生产角色清单」必须逐条列明**（这是 SC5 的后半句，planner 不得漏）⇒ 必须跑 §E19 的判定 SQL 并把结果写进验收报告 |
| 播种会不会**间接**改动既有行 | **会，但可控且必须登记**：① `pkg/permission/service.go:81-126` `assignAllMenusToAdmin` 在 setup 阶段给 admin **增量补全**全部菜单 ⇒ admin 的 `sys_role_menu` 会新增 72 行（admin 走 bypass ⇒ 无实际影响，但**行数会变**，验收报告须记录）；② `sys_menu` 的新增行会进入 `/system/my-menus` 的菜单树（`menu_type IN ('M','C')` 过滤 ⇒ **F 行不进导航树**，无 UI 变化）；③ 后端 30s 权限缓存会自然过期 |
| **不污染生产库的验证路径** | ① **sqlite 内存库 + 原生 DDL**（`migration_218_ad_reconciliation_permission_test.go:23-70` 范式）⇒ **完全本地、零外部依赖、可进 `go test ./internal/core/db/`**；② **只读 SQL 判据**（note §3.8）⇒ 对生产库只 SELECT，不写；③ **本地起后端连生产库** ⇒ **会写**（`sys_oper_log` / `ad_push_queue` / 物化视图刷新 / OUI 导入），**不推荐**；④ **dry-run 可能性**：**本仓迁移机制没有 dry-run**（每次启动无条件重跑，无 skip 语义，`database.go:1318-1319`）⇒ 想要「不落库预览」只能靠 ① 的 sqlite 测试 |
| 生产库策略 | 迁移会在**生产后端下次启动时自动执行**（`database.go` 调用块）。⇒ 87 的 PLAN 必须包含一条**部署顺序要求**：**先合入 migration（数据），后合入前端/校验脚本（消费）**；且因 85 已把写路由切成 Strict，**87 之前所有非 admin 角色的写操作是 403（fail-closed 窗口期）**——85-RESEARCH §B' 第 4 条已明确声明该窗口期（「85 内不补授…窗口期=85 合入到 87 播种之间，需在 phase plan 显式声明」） |
| 夹具清理 | 单事务内清理（先子后父），清理后 6 项自检全 0（note §4 + §4.1） |

### G24 · 本机/测试环境跑 migration 验证的可行路径

| 问 | 答 |
|---|---|
| PG 在哪 | `10.62.10.34:5432`（远程，`configs/config.yaml:12-13`）；Redis 同主机 6379（`:27-28`） |
| `.env` 可读吗 | **不可读**（安全网拦截）⇒ 口令须经**环境变量**注入 |
| `configs/config.yaml` 指向 | 远程 PG（`host: "10.62.10.34"` / `port: 5432`） |
| **本地跑 migration 验证的可行路径** | **【推荐】sqlite 内存库 + 原生 DDL 测试**（`migration_218_ad_reconciliation_permission_test.go` 范式）：`go test ./internal/core/db/migrations/` 即可，**零外部依赖、零生产风险**。⚠️ 注意：**不要**用 GORM `AutoMigrate` 建表（`type:uuid` 等 PG-only 片段会报错 —— 该文件 :18-21 注释逐字：「原生 DDL（不走 GORM AutoMigrate）：sqlite 测试路径不经 sanitizeSQLiteModelDefaults，model 里的 type:uuid 等 PG-only 片段会直接报错（同 migration_217 测试口径）」） |
| 独立 PG 可用吗 | 本机**无本地 PG**（note §3.8.1：生产后端 9000 端口是 MinIO、nginx 502；`.planning/STATE.md:90` 记「本机无 PG/psql，且远程 PG 需密码（安全网拦截）」）⇒ 除非用户提供口令，**验证路径只能是 sqlite + 只读 SQL** |
| migration 的 test 基建模式 | `internal/core/db/migrations/*_test.go`：`migration_202_port_write_audit_test.go`、`migration_207_menu_catalog_seed_test.go`、`migration_208_dict_seed_test.go`、`migration_209_update_settings_menu_component_test.go`、`migration_210_normalize_menu_parent_id_test.go`、`migration_213_recompute_building_total_floors_test.go`、`migration_216_adpush_drain_sysconfig_test.go`、`migration_217_dept_ou_mismatch_view_test.go`、`migration_218_ad_reconciliation_permission_test.go`、`migration_176_reconciliation_physical_mv_test.go` —— **10 个先例** |
| 双方言验证 | `isPostgreSQL(db)` 守卫（`migration_helpers.go:7-9`）在 sqlite 下返回 false ⇒ **若 87 的新 migration 用 `isPostgreSQL` 守卫，sqlite 测试会全部跳过**（= 假绿）。⇒ **87 的新 migration 必须双方言跑**（用 `NOT EXISTS` + `?` 占位，不用 `::uuid` / `ON CONFLICT`），这样 sqlite 测试才真正执行 |
| CI 侧 Go 测试命令 | `scripts/check-ci-local.sh:74-75`：`go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...`（与 CI 的 `ci.yml` 对齐；⚠️ 本仓 `.github/workflows/` **只有** `frontend-build.yml`，无 `ci.yml` —— `check-ci-local.sh` 引用的 `ci.yml`/`.github/scripts/` 在当前工作区**不存在**，已 ls 实证） |

---

## Validation Architecture

> 参照 `.planning/phases/84-pmodel-gaps/84-VALIDATION.md` 的结构（Test Infrastructure / Sampling Rate / Per-Task Verification Map / Wave 0 Requirements / Manual-Only Verifications / Validation Sign-Off）。planner 可据此产出 `87-VALIDATION.md`。

### Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/glebarez/sqlite`（**内存库 + 原生 DDL**，仿 `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go:23-70`）｜前端 vitest + @testing-library/react（仿 `src/pages/system/role/__tests__/permission.gate.test.tsx`）｜Node 脚本 `node scripts/check-permission-codes.mjs` |
| **Config file** | 无新增——沿用 `internal/core/db/migrations/*_test.go` 与 `xingran-react-frontend/src/pages/system/role/__tests__/` 既有基建 |
| **Quick run command** | `go test ./internal/core/db/migrations/ ./pkg/permission/ ./internal/services/system/` ＋ `cd xingran-react-frontend && node scripts/check-permission-codes.mjs && npx vitest run src/pages/system/role/` |
| **Full suite command** | `go build ./... && go test ./...` ＋ `cd xingran-react-frontend && npm run lint:ci && npm run test` |
| **Estimated runtime** | quick ~40s / full ~3min（Go）+ ~5min（前端 569 文件 3960 用例） |

### Sampling Rate

- **After every task commit:** quick run command
- **After every plan wave:** full suite command
- **Before `/gsd:verify-work`:** Full suite must be green（含 `npm run lint:ci` exit 0 —— 87 之后 `check-permission-codes.mjs` 在链内）
- **Max feedback latency:** 60s（quick）/ 240s（full）

### Per-Task Verification Map

> Task ID 在 PLAN.md 定稿后回填；此处预置**需求级**验证锚点。

| Requirement | Secure Behavior | Test Type | Automated Command | Status |
|-------------|-----------------|-----------|--------------------|--------|
| RBAC-05 | 72 条 F 型码在 `sys_menu` 各有且仅有 1 行（`menu_type='F'`、`status=0`、`visible=0`、`perms` 单码） | unit（sqlite 迁移测试） | `go test -run TestMigrate226 ./internal/core/db/migrations/` | ⬜ pending |
| RBAC-05 | 迁移**幂等**：连续执行 2 次 ⇒ `sys_menu` 行数不变、`sys_role_menu` 行数不变 | unit（红绿：回退幂等即失败） | `go test -run "TestMigrate226.*Idempotent" ./internal/core/db/migrations/` | ⬜ pending |
| RBAC-05 | **精准授权**：仅持有父 C 型菜单的角色获得新 F 行；不持有者不获得 | unit | `go test -run "TestMigrate226.*Grant" ./internal/core/db/migrations/` | ⬜ pending |
| RBAC-05 | **父菜单缺失降级**：返回 nil、不 panic、不新增任何行 | unit | `go test -run "TestMigrate226.*MissingParent" ./internal/core/db/migrations/` | ⬜ pending |
| RBAC-05 | **父集合多值**：`network:command` 的 2 个父 C 菜单都能触发授权 | unit | `go test -run "TestMigrate226.*MultiParent" ./internal/core/db/migrations/` | ⬜ pending |
| RBAC-05 | **角色页可勾选 F 型**：F 节点出现在树中 → 勾选 → 提交的 `menuIds` 含该 F 节点 id | component（vitest） | `cd xingran-react-frontend && npx vitest run src/pages/system/role/__tests__/permission.grant.test.tsx` | ⬜ pending |
| RBAC-05 | **同源**：同一权限码同时驱动前端可见性与后端放行 | unit + component | 上面两条用**同一串码**（取自 `PERMISSION_CODES` / `pkg/permission/config.go` 常量） | ⬜ pending |
| RBAC-06 | **三方零漂移**（TS ↔ Go 常量 ↔ `sys_menu` F 型）：脚本 exit 0 | script | `cd xingran-react-frontend && node scripts/check-permission-codes.mjs` | ⬜ pending |
| RBAC-06 | **白名单已清空且被硬断言守护**（非空 ⇒ exit 1） | script（红绿） | 同上；反向：临时加 1 条白名单 → 断言 exit 1 | ⬜ pending |
| RBAC-06 | **Go 侧 F 型识别生效**：d 类对 72 条生效（把某码从 TS 移除 → 断言红） | script（红绿） | 同上 | ⬜ pending |
| RBAC-06 | **接入 CI 门禁**：`lint:ci` 含 `check-permission-codes.mjs` 且整体 exit 0 | script | `cd xingran-react-frontend && npm run lint:ci` | ⬜ pending |
| RBAC-06 | **路由登记不回归** | go test + script | `go test -run "TestRouteRegistry" ./pkg/permission/` ＋ `node scripts/check-route-registry.mjs` | ⬜ pending |
| 回归 | v1.24 三态门控契约零破坏（ready 空集 ≠ 未就绪） | component | `npx vitest run src/pages/system/role/__tests__/permission.gate.test.tsx`（**不得改动该文件**） | ⬜ pending |
| 回归 | v1.25 回归基线（eslint 0 / vitest 全绿 / go build+test 全绿） | 全量 | `npm run lint:ci && npm run test`；`go build ./... && go test ./...` | ⬜ pending |
| 回归 | **生产既有行零改动**（播种只新增行） | 只读 SQL（人工） | 见 Manual-Only 表 | ⬜ pending |

### Wave 0 Requirements

- [ ] `internal/core/db/migrations/migration_226_action_permission_f_type_seed.go` —— 新 migration（双方言、幂等、精准授权、72 条种子）
- [ ] `internal/core/db/migrations/migration_226_action_permission_f_type_seed_test.go` —— sqlite 原生 DDL 测试（幂等 / 精准授权 / 父缺失降级 / 多父集合 / 既有行零改动）
- [ ] `internal/core/db/migrations/registry.go` —— `ActiveMigrations` 追加 `{226, "action_permission_f_type_seed"}`
- [ ] `internal/core/db/database.go` —— PG 分支 + sqlite 分支各加一条 `d.runAndRecord(226, ...)`
- [ ] `xingran-react-frontend/scripts/check-permission-codes.mjs` —— 清空白名单 + 硬断言 + Go 侧 F 型识别
- [ ] `xingran-react-frontend/package.json` —— `lint:ci` 插入 `node scripts/check-permission-codes.mjs`
- [ ] `.github/workflows/frontend-build.yml` —— `npm run lint` → `npm run lint:ci`
- [ ] `scripts/check-ci-local.sh` —— frontend 段加 `lint:ci`（保持「本地绿 = CI 绿」）
- [ ] `xingran-react-frontend/src/pages/system/role/__tests__/permission.grant.test.tsx` —— 角色授权闭环组件测试（新建）

*其余沿用各包既有测试基建；无框架安装需求（零新依赖）。*

### Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| **生产受影响角色清单**（播种后哪些角色新增写能力） | RBAC-05 / SC5 | 需连生产库只读查询；口令不在环境内 | 跑 §E19 的判定 SQL → 结果为空或逐条列明，写进 `87-VERIFICATION.md` |
| **生产库既有行零改动核对** | SC5 | 需生产库行数快照 | 播种前后各跑一次：`SELECT (SELECT count(*) FROM sys_menu), (SELECT count(*) FROM sys_role_menu), (SELECT count(*) FROM sys_role), (SELECT count(*) FROM sys_user)`；差值 = 72 + N + 0 + 0 |
| **DB 层前后对照（生产）** | RBAC-05 | 需生产库写权限 | 用 note §3.8.2 / §3.8.3 的只读 SQL 模板，对夹具角色跑 `step1_exact` / `strict_exact_pass` ⇒ 403 判定（**全部只 SELECT**，用 `SET TRANSACTION READ ONLY` 兜底） |
| **HTTP 实测 403→200** | RBAC-05 | 本机无 PG/口令；生产后端进程未运行（nginx 502）；AD + 滑块阻断登录 | ① 首选 **note §3.8 只读 SQL 判据**（零副作用）；② 若用户提供生产口令 ⇒ 按 note §3.4.2 起本地后端（`SERVER_MODE=test` + `SKIP_AUTOMIGRATE=true` + `SERVER_SKIP_SETUP=true` + `DB_PASSWORD` + `JWT_SM2_*`）+ 自签 token（issuer=`RuoYi-Next`）+ curl；**副作用须如实登记**（`sys_oper_log` / `ad_push_queue` / 物化视图 / OUI 导入） |
| **决定性对照（临时加权限点 → 403 转 200）** | SC3 | 同上 | 夹具 role_key `p87_*`（绝不撞 `admin`）；补授 F 型 `menu_id` → **等 31s**（或重启后端进程）→ 同端点 200 且真生效 → 撤销 → 回 403 |
| **前端 UI 实机**（角色页勾选后按钮出现） | RBAC-05 | 需登录态（AD + 滑块），本机不可用 | 组件测试 T-1~T-4 覆盖；UI 实机留待用户执行（可参照 `90-visual-evidence/inspection-manual-todo-90.json` 的清单范式产出待办清单） |
| **迁移在生产 PG 上的实际执行** | RBAC-05 | 需生产部署 | 部署后查 `SELECT count(*) FROM sys_menu WHERE menu_type='F' AND perms IN (<72 码>)` ⇒ = 72（或 72 + migration_221 的 4 条既有 manager F 行） |
| **`sys_role_menu` 在生产 PG 是否有 `(role_id, menu_id)` 唯一约束** | 幂等前提 | 需 `\d sys_role_menu` 实查（见 OQ-1） | `psql -c "\d sys_role_menu"`；若无唯一约束，确认 `NOT EXISTS` 幂等写法足以防重复 |

### Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s（quick）
- [ ] `nyquist_compliant: true` set in frontmatter（`87-VALIDATION.md`）
- [ ] 回归基线四项复核通过：eslint 0 problems / vitest 569 文件 3960 用例全绿 / vendor-react gzip ≤ 776 kB / `go build ./...` + `go test` 全绿

**Approval:** pending

---

## Open Questions

> 以下是需要 planner / 用户裁定的疑点。**不得在 PLAN 里默默假设**。

| # | 疑点 | 影响 | 建议处置 | 状态 |
|---|---|---|---|---|
| **OQ-1** | `sys_role_menu` 在**生产 PG** 上是否真有 `(role_id, menu_id)` **唯一约束**？`models/role.go:24-27` 的 GORM tag 只是**普通索引**（`index:idx_sys_role_menu_role_id_menu_id`），而 `menu_grant_helpers.go:22-24` 的注释声称「依赖 sys_role_menu 的 (role_id, menu_id) 复合 PK / 唯一约束（migration_144 已依赖此约束做 dedup）」—— **本会话未实证**（无口令连库） | 决定 `ON CONFLICT DO NOTHING` 是否可用 | **87 一律用 `NOT EXISTS` 写法**（不依赖唯一约束）；`ON CONFLICT` 一律不用 | 待用户在 §Manual-Only 表最后一行的 `\d sys_role_menu` 复核 |
| **OQ-2** | **是否要一并播种 6 条 `ops:manager:*`？** migration_221 已在运行期建 `ops:manager:rule:{edit,delete}` / `ops:manager:org:{edit,delete}` 四个 F 行 + 两个 C 型菜单（`perms='ops:manager:rule:list'` / `'ops:manager:org:list'`），但 **脚本的 Go 侧扫描把 `list` 误当「已声明」而 d 类看不见** | 白名单里 6 项的处置口径 | **建议**：87 **不播种**这 6 项（避免与 migration_221 产生同 perms 重复行），**但**把 `list` 两条的语义在 `87-STATE.md` 登记清楚（`list` = C 型页面本体码，不是 F 型按钮码） | 待 Lead 确认 |
| **OQ-3** | **`ops:building:spaces:query` / `ops:building:spaces:3d:query` 的父是「楼宇空间」/「楼宇空间3D」C 菜单**（`550e8400-…-100` / `550e8400-…-102`），而 `ops:building:import/export` 的父是「楼宇管理」。二者不在同一页。**87 是否要把 import/export 也给一份挂在「楼宇空间」下？** | 树展示 / Step 2 继承的父 | **建议**：不重复挂 —— 后端 `POST /ops/building/import` 属楼宇 CRUD 域，父 = 楼宇管理 | 待 Lead 确认（影响很小） |
| **OQ-4** | **`system:role:export` 无后端路由**（只有常量 + TS 声明 + `resource_action_map.go:53`）。播种它会创建一个「无动作对应」的授权来源 | 白名单完整性 | **建议**：仍然播种（白名单是权威清单，且清空白名单是硬要求），在 `87-VERIFICATION.md` 标注「无路由，仅作授权来源占位」 | 待 Lead 确认 |
| **OQ-5** | **前端菜单缓存 5min（内存）+ 30min（sessionStorage）** 与后端 30s 权限缓存叠加，**UI 实机验收必须在授权后清 sessionStorage 或重新登录**。但 `menuStore` 的 `MENU_CACHE_VERSION = 1` 未变 ⇒ 缓存不自动失效 | UI 实机验收的可复现性 | 验收清单必须显式写「清 sessionStorage `menu:cache` 或重新登录后再断言按钮可见性」 | 已定（写入验收清单） |
| **OQ-6** | **`ops:walls:*` / `ops:doors:*` / `ops:floor-plan:edit` 挂「楼层管理」是本研究的推断**（依据：前端在 `src/pages/operations/floors/index.tsx:57` 使用 `opsFloorPlan.edit`；后端写路由在 `router.go:1146-1152 / 1178-1184 / 1206-1211` 挂在 `ops` 组下）。**但 CAD 平面图编辑器的实际使用页还需人工确认**（可能是「楼宇空间」页而非「楼层管理」页） | 父菜单选择 ⇒ 影响 Step 2 继承与树展示 | **建议**：实地确认「墙体/门/平面图文本」的管理入口页；确认前**不改**这三个码的父（避免猜错） | **待用户/Lead 确认**（唯一实质性待确认项） |
| **OQ-7** | **`ops:floor-plan:list` / `ops:floor-plan:query` / `ops:floor-plan:save` / `ops:walls:list` / `ops:doors:list` 五个码不在 TS 声明也不在白名单**，但 `router.go:1128-1137 / 1160-1169 / 1192-1197` 的读组 ANY-of 引用了它们 ⇒ **墙体/门/平面图文本的读路由当前对非 admin 恒 403**（无任何菜单行承载这些码） | 读路径可用性 | 87 **不处理**（不在白名单、不在 TS）；登记为独立后续项 | 待 Lead 确认（建议独立立项） |

---

*Research produced by gsd-phase-researcher, 2026-10-30。全部 in-repo 离散值均已本会话逐字读取并标注 `[VERIFIED: 路径:行号]`；外部文档引用仅 `AGENTS.md` / `CLAUDE.md` / `.planning/` 内的既有 artifact（标 `[CITED]`）。*
