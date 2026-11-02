# v1.24 规划输入 — 生产 PG 权限基线与 #16 定责

> 实测时间：2026-10-28｜数据源：PostgreSQL `10.62.10.34:5432/ruoyi`（只读查询）
> 用途：v1.24「#16 行操作权限」定责依据 + 受影响角色范围界定

## 1. 生产库角色清单与权限规模

| role_key | role_name | 菜单数 | 写权限数 |
|---|---|---|---|
| admin | 超级管理员 | 246 | 71 |
| kjgl | 空间管理 | 40 | 14 |
| xtgl | 系统管理 | 34 | 14 |
| adyk | AD域控 | 15 | 4 |
| vm_user | 虚拟机一般用户 | 11 | 2 |
| user | 普通用户 | 5 | **0** |

## 2. system:user 权限矩阵（受影响面）

| role_key | system:user（页面本体） | list | view | add | edit | remove | resetPwd | import | export |
|---|---|---|---|---|---|---|---|---|---|
| admin | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| xtgl | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌（无 export） |
| user | ❌ | — | — | — | — | — | — | — | — |
| kjgl / adyk / vm_user | ❌ | — | — | — | — | — | — | — | — |

**关键事实**：生产现有角色中，**能进入「系统用户」页面的只有 admin 与 xtgl，且两者都持有完整写权限**——不存在「能进页面但无写权限」的现成角色。

## 3. #16 定责结论

**产品缺陷（确认存在）**：`xingran-react-frontend/src/pages/system/user/index.tsx:279-301` 的操作列渲染**没有任何权限判定**——「编辑」「重置密码」「删除」三按钮无条件渲染；批量删除入口（:756）同样无判定。该文件全文无 `permissions.includes` / `hasPerm` 引用。

**对照项目内的正确做法**：
- `src/pages/network/ports/index.tsx:132` — `const canWrite = hasPermission("network:port:write")`，操作列按 `canWrite` 条件展开（v1.19 引入）
- `src/pages/asset/reconciliation/exceptions/index.tsx:86` — `permissions.includes("asset:reconciliation:resolve")`
- `src/pages/asset/reconciliation/fix-suggestion/index.tsx:65-67` — `canAccept` / `canReject` / `canRollback`
- `src/pages/ad-domain/configs/ReconciliationTab.tsx:77` — `canView`
- `src/components/table/VDIRow.tsx:43` — `buttons.filter(btn => permissions.includes(btn.permission))`

**为何生产不复现**：admin 与 xtgl 都有写权限；其余 4 个角色根本进不了该页面（菜单本体 `system:user` 不在其 role_menu 内）。
**为何 UAT 复现了**：sqlite 环境手工插入的角色 + 菜单分配不完整（`role_menu` 记录未走 API），叠加「前端根本无判定」→ 任何能进页面的人都看到写按钮。

**风险定性**：真实风险是**未来配置只读/审计类角色时越权显示操作入口**（后端 `RequirePermissions` 会拦实际请求并返回 403，但 UI 层泄露了不应存在的入口，与 §8 #16「无权限用户的删除按钮仍不可见」的契约相悖）。

## 4. 修复方向（供 v1.24 规划采用）

1. 在 `src/pages/system/user/index.tsx` 引入 `useMenuStore((s) => s.permissions)`，按权限标识条件渲染：
   - 新增按钮 → `system:user:add`
   - 编辑 → `system:user:edit`
   - 删除 / 批量删除 → `system:user:remove`
   - 重置密码 → `system:user:resetPwd`
   - 导入 → `system:user:import`；导出 → `system:user:export`
2. 判定方式沿用 `network/ports/index.tsx` 的 `hasPermission(perm)` 局部 helper 模式（不引入新依赖）
3. **fail-closed vs fail-open 的取舍**：权限集合为空/未加载时默认隐藏写按钮（fail-closed）——与「操作入口不该存在」的契约一致；需评估权限加载完成前按钮闪现的可接受性
4. **验收方式**：构造一个「有 `system:user:list` 但无 `system:user:edit`」的角色，在浏览器确认编辑/删除/重置密码按钮消失；补前端单测断言按钮渲染条件

## 5. 权限标识速查（`sys_menu.perms`，供实现时对照）

| perms | 菜单名 | type |
|---|---|---|
| `system:user` | 系统用户 | C（页面本体） |
| `system:user:list` | 查询 | F |
| `system:user:view` | 查看 | F |
| `system:user:add` | 新增 | F |
| `system:user:edit` | 修改 | F |
| `system:user:remove` | 删除 | F |
| `system:user:resetPwd` | 重置密码 | F |
| `system:user:import` | 导入 | F |
| `system:user:export` | 导出 | F |

## 6. 附注

- 本次为**只读探查**，未对生产库做任何写操作（角色/菜单/用户均未改动）
- 数据库凭据未写入任何仓库文件；连接仅用于本次会话的只读查询

---

## 7. 【2026-10-28 修订】前端全站扫描 + 后端路由扫描（定责最终版）

### 7.1 前端：权限判定覆盖面

全量扫描 `xingran-react-frontend/src/pages/**/*.tsx`（排除 `__tests__` / `.test.`），按「是否含写操作特征」与「是否含权限判定特征」双分类：

| 分类 | 数量 | 说明 |
|---|---|---|
| 有写操作（删除/编辑/重置/新增/导入/导出）的页面 | **119** | — |
| 其中**完全无任何权限判定** | **108（90.8%）** | 无 `permissions.includes` / `hasPermission(` / `canWrite` / `canDelete` / `useMenuStore` 任一引用 |
| 其中已有权限判定 | 11（9.2%） | 见下表 |

**已有权限判定的 11 个页面（正确做法样板）**：

| 页面 | 判定写法 | 位置 |
|---|---|---|
| `pages/network/ports/index.tsx` | `hasPermission("network:port:write")` → `canWrite` | :132 |
| `pages/asset/reconciliation/exceptions/index.tsx` | `permissions.includes("asset:reconciliation:resolve")` | :86 |
| `pages/asset/reconciliation/fix-suggestion/index.tsx` | `canAccept` / `canReject` / `canRollback` | :65-67 |
| `pages/ad-domain/configs/ReconciliationTab.tsx` | `canView = permissions.includes(RECONCILIATION_PERM)` | :77 |
| `components/table/VDIRow.tsx` | `buttons.filter(btn => permissions.includes(btn.permission))` | :43 |
| `pages/vdi/VirtualMachineList/index.tsx` | 局部 `hasPermission(permissions, perm)` helper | :46 |
| `pages/operations/workstations/LocationAliasDrawer.tsx` | 局部 `hasPermission(perm)` helper | :63 |

**结论**：#16 **不是孤例**——系统用户页只是「全站 108 个页面缺操作级权限判定」中的一个样本。

### 7.2 后端：权限中间件覆盖面

> ⚠️ **口径修正（2026-10-28，ARCHITECTURE 研究交叉验证后）**：§7.2 的「`user_router.go` 12 路由零中间件」是**局部扫描口径**——权限是在**主路由挂载处**（`internal/api/router.go`）以 `users.Use(...)` 组级施加的，不在 `user_router.go` 文件内部。因此**不能**据此断言「后端完全无保护」。真实结论见 §7.8。

全量扫描 `internal/api/**/*_router.go` 的 367 条路由注册：

| 分类 | 数量 | 占比 |
|---|---|---|
| 路由总数 | 367 | 100% |
| 带 `RequirePermissions` / `RequireRole` | **27** | **7.4%** |
| **未带任何权限中间件** | **340** | **92.6%** |

**带权限中间件的仅 2 个模块**（`internal/api/v1/system/` 下）：

- `ad_domain_router.go`（12 处，v1.20 引入）
- `manager_rule_router.go`（2 处）

**`internal/api/v1/system/user_router.go` 全部 12 条路由零权限中间件**（局部事实；组级保护在 `router.go`）：

```go
r.POST("", handler.Create)                    // ← 无 RequirePermissions
r.POST("/list", handler.List)
r.POST("/statistics", handler.Statistics)
r.POST("/import", handler.ImportUser)
r.GET("/import/template", handler.DownloadImportTemplate)
r.POST("/batch-delete", handler.BatchDelete)
r.POST("/sync-managers", handler.SyncManagers)
r.POST("/:id", handler.GetByID)
r.POST("/:id/update", handler.Update)
r.POST("/:id/delete", handler.Delete)
r.POST("/:id/status", handler.UpdateStatus)
r.POST("/:id/reset-password", handler.ResetPassword)
```

### 7.3 #16 定责最终结论

**根因定性：系统用户模块前后端双缺口（操作级 RBAC 完全缺失）**

| 层 | 现状 | 唯一屏障 |
|---|---|---|
| 前端操作列 | 无条件渲染编辑/重置密码/删除/批量删除 | 无 |
| 后端 12 条路由 | 无 `RequirePermissions` | JWT 认证 + **菜单级权限**（能否进 `/system/user` 页面） |

**真实风险（比 UAT 描述更严重）**：**无法配置「只读用户管理员」**——任何被授予「系统用户」菜单（`system:user`）的角色，会自动获得完整的用户增删改、重置密码、批量删除、导入能力。这不是「按钮误显示」的 UX 问题，而是**权限粒度缺失**。

**为何生产不复现**：生产仅有 admin 与 xtgl 能进该页，且两者都持完整写权限（见 §2 矩阵）。风险在**未来配置只读/审计类角色时爆发**。

**修复方向（前后端闭环，作为全站治理的样板）**：

*前端*（`src/pages/system/user/index.tsx`）：

- 新增 → `system:user:add`；编辑 → `system:user:edit`；删除/批量删除 → `system:user:remove`；重置密码 → `system:user:resetPwd`；导入 → `system:user:import`；导出 → `system:user:export`
- 判定沿用 `network/ports/index.tsx` 的局部 `hasPermission(perm)` helper 模式（零新依赖）

*后端*（`internal/api/v1/system/user_router.go`）：逐路由补 `middleware.RequirePermissions([...], core)`

**⚠️ 后端补权限的兼容风险（必须处理）**：

1. `xtgl`（系统管理）**没有 `system:user:export`**（见 §2 矩阵）→ 若给 export 接口加保护，xtgl 用户点导出将 403。**二选一**：给 xtgl 补 export 权限，或 export 接口暂不加保护
2. `admin`（超级管理员）通常走 bypass 分支 → 需确认 `RequirePermissions` 中间件对超级管理员的处理
3. 任何**历史上有该菜单但无 F 型权限**的自定义角色会在升级后突然 403 → 上线前需全量核对 `sys_role_menu` 中含 `system:user` 但缺对应 F 权限的角色

### 7.4 范围建议（待 Lead/用户裁定）

| 选项 | 内容 | 评估 |
|---|---|---|
| A. 仅前端判定 | 补前端按钮判定，后端不动 | **安全收益≈0**（API 直调仍可写），仅改善 UX；不推荐作为终点 |
| B. 前后端闭环（**推荐**） | 系统用户模块前后端都补权限判定 | 真实闭环，且作为全站治理的**样板**；范围可控（1 页面 + 1 router） |
| C. 全站治理 | 108 页面 + 340 路由全面补齐 | **v1.24 不现实**（回归面巨大、需全量角色数据核对）；建议独立里程碑 |

### 7.5 由此新发现的技术债（建议独立里程碑，v1.25 候选）

**「操作级 RBAC 全站缺失」**——前端 108/119 页面 + 后端 340/367 路由无操作级权限判定，当前安全模型实为「菜单级权限 + 数据范围(data_scope)」。

- 影响面：几乎所有业务域（duty / knowledge / network / operations / asset / monitor / dashboard / ad-domain / system）
- 改造前置：① 全量角色-菜单-权限数据核对（`sys_role_menu` × `sys_menu.perms`）② 逐模块的权限标识设计 ③ 超管 bypass 口径确认 ④ 灰度与回滚方案
- **v1.24 建议只做 B（系统用户模块样板）**，把全站治理明确登记为独立债务，避免在双框 P0 修复的里程碑里夹带高风险大改造

### 7.6 ⚠️ 后端修复的关键阻碍：`RequirePermissions` 的模块级降级会放行只读角色

读 `pkg/middleware/permission.go` 的 `checkUserPermissionUncached`（:136-222），它有**三级判定**：

| 级 | 逻辑 | 位置 |
|---|---|---|
| 1 | 精确匹配 `m.perms = ?` | :145-157 |
| 2 | **F 型子权限 → 父菜单页面权**继承（用户有任一子按钮权限 → 获得父菜单访问权） | :159-198 |
| 3 | **模块级降级**：`extractModulePermission(perm)` 取最后冒号前的部分再匹配 | :200-218 |

**第 3 级对写操作是致命的**。以「只有 `system:user`（页面本体）+ `system:user:list`，但无 `system:user:edit`」的角色为例，检查 `system:user:edit` 时：

```
Step 1  精确匹配 m.perms = 'system:user:edit'          → count=0，失败
Step 2  查 'system:user:edit' 菜单的 F 型子菜单          → 该菜单本身即 F 型，无子 F → childPerms 为空，跳过
Step 3  extractModulePermission('system:user:edit')
        = 'system:user'  → 查 m.perms = 'system:user'   → count=1（能进页面就有）→ 通过 ❌
```

**即：只要能进「系统用户」页面，`RequirePermissions(["system:user:edit"])` 就会放行写操作。** 直接给 `user_router.go` 套现成中间件**达不到修复目的**。

**结论：必须新增一个「不做模块级降级」的严格检查中间件**（如 `RequireStrictPermissions`），写操作（create/update/delete/batch-delete/reset-password/status/import）改用它；读操作沿用现有 `RequirePermissions`（保留 list→父菜单的兼容语义）。

**设计约束（写进执行计划）**：

1. 新中间件必须**跳过 Step 3**（模块降级），但**保留 Step 1**（精确匹配）与**跳过 Step 2**（F 型子权限继承对写操作同样会放行：`system:user:edit` 是 F 型，其「子 F」为空故本例安全，但其他模块需逐一确认）
2. 超管 bypass 沿用 `isSuperAdmin`（`role_key = 'admin'`，:247-268）→ admin 不受影响
3. 30s 短 TTL 缓存沿用现状（已接受的角色变更 30s 陈旧窗口，:30-32 注释）
4. 新中间件需**单测覆盖**：超管放行 / 有精确权限放行 / 只有父菜单权限拒绝 / 只有子权限拒绝 / 无权限拒绝
5. **数据前置核对**：上线前须查 `sys_role_menu` 中所有含 `system:user` 页面权限但缺对应 F 权限的角色（生产现为 admin + xtgl，两者 F 权限齐全，除 xtgl 缺 `system:user:export`）

### 7.7 修复后的验收口径

| 角色配置 | 期望 |
|---|---|
| admin（超管） | 全部按钮可见，全部接口 200 |
| xtgl（有 edit/remove/resetPwd/add/import） | 同上；**导出按钮按现状**（xtgl 无 export 权限 → 需裁定：给 xtgl 补权限 or export 接口不加保护） |
| 构造角色：仅 `system:user` + `system:user:list` | 编辑/重置密码/删除/批量删除按钮**不可见**；直调写接口返回 403 |
| 构造角色：有 `system:user:edit` 无 remove | 编辑可见、删除不可见；delete 接口 403 |

### 7.8 【最终修正】后端真实保护机制与**两个叠加病灶**

（来源：ARCHITECTURE 研究 §1.5，与本 note §7.2 的局部口径交叉验证后修订）

**真实机制**：用户路由在**主路由挂载处**受组级保护——`internal/api/router.go:157-162`：

```go
users.Use(middleware.RequirePermissionsWithQuery([]string{
    string(permission.UserList),   // system:user:list
    string(permission.UserAdd),    // system:user:add
    string(permission.UserEdit),   // system:user:edit
    string(permission.UserView),   // system:user:view
}, []string{"ops:workstation:list", "ops:building:spaces:list"}, core))
```

**病灶 1 —— 组级 ANY-of 抹平了动作粒度（真实安全缺陷）**

`RequirePermissions` 的语义是**任一命中即放行**（`pkg/middleware/permission.go:276-303`）：

```go
for _, permission := range permissions {
    if checkUserPermission(core, userID, permission) { c.Next(); return }   // ← OR
}
```

`RequirePermissionsWithQuery`（:316-350）同构——主数组**始终是 OR**，`isQueryPath` 只决定是否追加 `queryExtraPermissions`，**不改变主数组的 OR 语义**。

> ⚠️ **注释与实现矛盾**：`permission.go:305-315` 的 doc 注释写「写操作仍只受严格权限保护，避免越权」，**实现并未做到**——写操作与读操作走同一个 OR 数组。

**后果**：只持有 `system:user:list` 的用户，`POST /system/users/{id}/delete` 命中 `UserList` 分支 → `c.Next()` → **后端真的执行删除并返回 200**。这比「前端按钮多寡」严重得多。

**病灶 2 —— 检查函数的模块级降级（§7.6）**

即使把写操作拆成 `RequirePermissions([UserRemove])`（单元素 = 精确要求），`checkUserPermissionUncached` 的 **Step 3 模块级降级**仍会把 `system:user:remove` 降级匹配到 `system:user`（页面本体）→ 只读角色仍放行。

**两个病灶叠加，必须同时解决**：

| # | 病灶 | 位置 | 修法 |
|---|---|---|---|
| 1 | 组级 ANY-of | `internal/api/router.go:157-162`（及 roles/menus/depts/posts 4 个同类组，`:183-236`） | **写操作从 ANY-of 组拆出为独立子组**，参照 network 域范式 `internal/api/v1/network/port_write_router.go:51`（写操作单独子组 + 单一权限点） |
| 2 | 模块级降级 | `pkg/middleware/permission.go:200-218` | 新增**不做 Step 3 降级**的严格检查（写操作用），或对写操作权限点做特殊处理 |

**已有资产（可直接复用，不必新设计）**：

- `pkg/permission/config.go:16,26,34,42,48` 已定义 `UserRemove` / `RoleRemove` / `MenuRemove` / `DeptRemove` / `PostRemove` **5 个常量，但在 `router.go` 挂载中一次都没被引用**
- `pkg/permission/config.go:223-289` 的 `GetRoutePermissions()` 已有**完整动作级映射表**（`/system/users/:id/delete` → `UserRemove` 等），配套 `GetPermissionByPath`（:292-302）与 `matchPath`（:304-330）**已实现但生产路径从不消费** → 「团队已设计好动作级权限模型，只是没有接线」
- **network 域已做正确示范**：`port_write_router.go:51` + `:183-192` 注释明写「RequirePermissions 为 ANY-of 语义：主组 4 权限点任一即放行，仅持 list 只读权限的用户曾可触发恢复。restore 必须单独强制专用权限点」→ **同一个仓里 network 域知道这个坑并已修，system 域没修**

**v1.24 修复范围（据此收敛）**：

- **后端**：`internal/api/router.go` 的 `/system/users` 组 → 读操作保留 ANY-of 组，写操作拆独立子组并逐动作绑定 `UserAdd`/`UserEdit`/`UserRemove`/`UserResetPwd`（复用已定义常量）；配「不做模块降级」的严格检查以封住病灶 2
- **前端**：`src/pages/system/user/index.tsx` 按同一套权限点条件渲染
- **其余 4 个同类组（roles/menus/depts/posts）**：v1.24 内**登记为同源债**，是否一并修由路线图阶段按风险定（roles/menus/depts 的写操作同样是 ANY-of → 同样可被只读角色触发）
- **全站 340 条无保护路由**：v1.25 独立里程碑
