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

**产品缺陷（确认存在）**：`ruoyi-react-frontend/src/pages/system/user/index.tsx:279-301` 的操作列渲染**没有任何权限判定**——「编辑」「重置密码」「删除」三按钮无条件渲染；批量删除入口（:756）同样无判定。该文件全文无 `permissions.includes` / `hasPerm` 引用。

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

全量扫描 `ruoyi-react-frontend/src/pages/**/*.tsx`（排除 `__tests__` / `.test.`），按「是否含写操作特征」与「是否含权限判定特征」双分类：

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

全量扫描 `internal/api/**/*_router.go` 的 367 条路由注册：

| 分类 | 数量 | 占比 |
|---|---|---|
| 路由总数 | 367 | 100% |
| 带 `RequirePermissions` / `RequireRole` | **27** | **7.4%** |
| **未带任何权限中间件** | **340** | **92.6%** |

**带权限中间件的仅 2 个模块**（`internal/api/v1/system/` 下）：

- `ad_domain_router.go`（12 处，v1.20 引入）
- `manager_rule_router.go`（2 处）

**`internal/api/v1/system/user_router.go` 全部 12 条路由零权限中间件**：

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
