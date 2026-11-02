# Architecture Research: v1.24「架构层方案与代码审计」

**Domain:** 前端权限链路审计 + CSS 治理架构 + 路由恢复 + 组件拆分 + lint 债务（无代码产出）
**Researched:** 2026-10-28
**Confidence:** HIGH（全部结论回源到文件:行号；后端 sqlite 500 根因已定位到唯一代码点但**未在 sqlite 实例上实测**（本机无 sqlite 运行实例），PG 复测项以「待实测」标注）

> 边界声明：本文件**只写研究文档，未改动任何源码**。所有「修法」均为建议，未实施。

---

## Executive Summary

v1.24 的 7 个范围项里，**#16 行操作权限是唯一带「定责」要求的**。本次审计的核心结论是：

> **#16 不是单点缺陷，而是「前端按钮级完全无判定」+「后端动作粒度被组级 ANY-of 抹平」两个独立缺陷的叠加。**
> 前端侧：52 个含「操作」列的文件中**仅 7 个**引用了权限，45 个**完全无权限判定**（fail-open，且是「无条件显示」而非「加载中默认显示」）。
> 后端侧：`RequirePermissions` 是 **ANY-of 语义**（`pkg/middleware/permission.go:276-303`），`/system/users` 组的 4 个权限点任一命中即放行**全部子路由**（含 delete）——即「只读用户点删除」后端会返回 200。
> 而 `pkg/permission/config.go:223-289` 那张声明了 `UserRemove`/`RoleRemove`/`MenuRemove` 的细粒度映射表是**死表**：`GetPermissionByPath` 只被测试调用，生产路径从不消费。

其余 6 项的架构定位：

| # | 范围项 | 结论 |
|---|--------|------|
| 1 | 输入框双重外框（P0） | 根因确认：`src/index.css:871-880` 的 `.ant-input { border: … !important }` 无差别命中 antd v6 affix-wrapper 内部的裸 `<input class="ant-input">`（该 input 在 antd v6 默认 `border:none`）。同仓已有两处**局部**补丁（`index.css:2056`、`:2077`）证明该问题被撞见过，但只在 `.dept-tree-search` / `.user-form-input` 作用域生效。 |
| 2 | #16 行操作权限 | 见 §1。**前端 fail-open 静态可判**（不需 PG 即可定责），但「后端是否也 fail-open」必须 PG 实测。 |
| 3 | 3f130 空因诊断前端展示 | 后端端点已交付：`POST /ops/workstation-device/{id}/physical-diagnosis`（`internal/api/v1/operations/workstation_device_handler.go:191-192`）。前端 **0 处调用**（全仓 grep `physical-diagnosis` 命中 0），仍用旧 `/physical`（`src/lib/opsApi.ts:620`）并渲染**硬编码静态文案**（`src/components/operations/WorkstationDeviceTable/index.tsx:604-609`）。 |
| 4 | F5 刷新落 /dashboard | 根因确认：**路由未匹配时的 catch-all fallback 抢先命中**，不是认证初始化落地页。`src/router/DynamicRoutes.tsx:315` 的 `<Route path="*" element={<Navigate to="/dashboard" replace />} />` 在 `routeElements` 首帧为空（`:228-230`）时匹配 `/system/user` 并 `replace` 掉原 URL。二阶损伤：`useTabSync` 会把 `activeTab` 强制改成 dashboard（`src/hooks/useTabSync.ts:86-88`）。 |
| 5 | ColumnConfigModal:136 豁免注释 | 该行是 `Modal.confirm({…})`，因文件位于 `src/components/shared/` → 命中 `eslint.config.js:264-269` 的封装层豁免 override（该块把 `no-restricted-syntax` 重声明为仅含内网 IP selector）→ **合法**。任务是补注释文档化，无需改规则。 |
| 6 | lint 17 errors + exhaustive-deps 12 | 实测基线确认：`51 problems (17 errors, 34 warnings)`；`react-hooks/exhaustive-deps` **12 处**、**`react-refresh/only-export-components` 33 处**（与 v1.24 口径完全对齐）。**新增信号**：`eslint-plugin-react-hooks@^7` 引入了 4 条 React Compiler 系新规则，已产生 3 条新 error。 |
| 7 | react-refresh 33 处 + devices 拆分 | 33 处**不是** barrel `index.ts` 造成的（纯 `.ts` barrel 不触发）；画像见 §5.2，**13/33 集中在 3 个「伪 constants.tsx」文件**。 |

---

## 1. (A) #16 权限链路深度审计

### 1.1 完整判定链路图

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ ① 权限真相源 (PostgreSQL)                                                     │
│   sys_menu(id, menu_name, perms, menu_type M/C/F, status, visible, parent_id)  │
│   sys_role_menu(role_id, menu_id)  ← models/role.go:24-27                     │
│   sys_user_role(user_id, role_id)                                             │
└───────────────────────────┬──────────────────────────────────────────────────┘
                            │ ② 聚合
┌───────────────────────────▼──────────────────────────────────────────────────┐
│ pkg/permission/service.go  GetUserPermissions(db, userID) → []string          │
│   ← pkg/middleware/permission.go:271-274 GetCurrentUserPermissions           │
└───────────────────────────┬──────────────────────────────────────────────────┘
                            │ ③ 暴露
┌───────────────────────────▼──────────────────────────────────────────────────┐
│ POST /system/my-menus/permissions                                             │
│   router: internal/api/v1/system/menu_router.go:67   （组内无 RequirePerm）  │
│   handler: internal/api/v1/system/menu_handler.go:335-348                    │
│   挂载: internal/api/router.go:195  SetupUserMenuRouter(authorized...)       │
└───────────────────────────┬──────────────────────────────────────────────────┘
                            │ ④ 前端 API
┌───────────────────────────▼──────────────────────────────────────────────────┐
│ src/lib/menuApi.ts  getUserMenus() / getAllUserMenus() / getUserPermissions()│
└───────────────────────────┬──────────────────────────────────────────────────┘
                            │ ⑤ Store
┌───────────────────────────▼──────────────────────────────────────────────────┐
│ src/store/menuStore.ts                                                        │
│   permissions: string[]   初始 [] (menuStore.ts:55)                          │
│   fetchAll()  :128-169  → Promise.all(3 个 API) 任一失败 → 整体 reject       │
│   fetchPermissions() :102-126 → catch: set(error) + rethrow → permissions 留 []│
│   clearMenus() :187-205 → 清 permissions + sessionStorage.MENU_CACHE          │
│   TTL: 5 分钟内存缓存 (menuStore.ts:3-9, services/cache/TTLMenuCache.ts)     │
└──────────┬────────────────────────────────────┬──────────────────────────────┘
           │ ⑤b 刷新路径 hydrate                 │ ⑥ 消费者
┌──────────▼──────────────────────┐  ┌──────────▼─────────────────────────────┐
│ DynamicRoutes.tsx:50-63 readMenuCache()│  │ 组件（页面）直接 useMenuStore(s=>s.permissions)│
│  sessionStorage, 30 分钟 TTL (:34) │  │  → 自行判断是否渲染按钮                │
│ DynamicRoutes.tsx:179-186 useEffect │  │                                        │
│  → useMenuStore.setState({...})    │  │                                        │
│  ⚠ 在 effect 内 → 首帧之后才生效    │  │                                        │
└─────────────────────────────────────┘  └──────────┬─────────────────────────────┘
                                                      │
        ┌─────────────────────────────────────────────┼──────────────────────────┐
        │ 消费者 1：路由级（菜单级）                    │ 消费者 2：行操作按钮级      │
        │ DynamicRoutes.tsx:227-240                   │ 页面内 actions 数组        │
        │  routeConfigManager.hasPermission(path,     │ ActionButtons.tsx:32-103   │
        │    permissions)                              │  ← 纯渲染器，零权限逻辑     │
        │  routeConfigManager.ts:90-107               │                           │
        │ RouteGuard.tsx:32-64（静态详情路由）         │                           │
        │ sidebar.tsx:112-117（后端已过滤的菜单）      │                           │
        └─────────────────────────────────────────────┴──────────────────────────┘
                                                      │
        ┌─────────────────────────────────────────────────────────────────────────┐
        │ ⑦ 后端 API 动作级（真正的安全边界）                                     │
        │   internal/api/router.go 各路由组 .Use(middleware.RequirePermissions([…]))│
        │   pkg/middleware/permission.go:277-303  ← ANY-of 语义                   │
        │   pkg/middleware/permission.go:136-245  checkUserPermissionUncached    │
        │   pkg/middleware/permission.go:247-268  isSuperAdmin(role_key='admin')  │
        │   pkg/middleware/permission.go:36       mwPermCacheTTL = 30s            │
        │   pkg/middleware/permission.go:68-75    InvalidateMiddlewarePermCache  │
        │                                ⚠ 注释明写「当前未接线」                 │
        └─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 逐节点 fail-open / fail-closed 判定表

| # | 节点 | 代码位置 | 权限**缺失/加载中/失败**时的行为 | 判定 |
|---|------|----------|-----------------------------------|------|
| 1 | `useMenuStore.permissions` 初值 | `src/store/menuStore.ts:55` | `[]`（空数组） | **fail-closed 的数据基础**（但见 #11） |
| 2 | `fetchPermissions` 失败 | `src/store/menuStore.ts:122-125` | `set({error})` + **rethrow**；`permissions` 保持 `[]` | **fail-closed**（数据层） |
| 3 | `fetchAll` 失败 | `src/store/menuStore.ts:161-168` | 三 API `Promise.all`（`:44-48`）任一失败 → 整体 reject，`permissions` 保持 `[]` | **fail-closed**（数据层） |
| 4 | 路由级过滤 `routeConfigManager.hasPermission` | `src/router/routeConfigManager.ts:90-107` | `meta.permissions` 为空 → `return {hasPermission:true}`（`:94`）**显式 fail-open（向后兼容）**；非空 → `some(includes)` → fail-closed | **混合**：有声明时 fail-closed，无声明时 fail-open |
| 5 | 静态详情路由 `RouteGuard` | `src/router/RouteGuard.tsx:36-38, 41-64` | `permissions.length === 0` → 直接放行；否则 `some(includes)` 失败 → `<Navigate fallback>` 或 403 页 | **fail-closed**（有声明时） |
| 6 | 侧边栏菜单 | `src/components/layout/sidebar.tsx:112-117` | `menus` 来自后端已按角色过滤的结果；空 → 渲染空菜单 | **fail-closed**（后端过滤 + 前端不补） |
| 7 | 登录时权限加载 | `src/pages/login/index.tsx:74` | `await Promise.all([fetchMenus(), fetchPermissions()])` —— 任一 reject → catch → **不 navigate**，停在登录页显示错误 | **fail-closed** |
| 8 | 刷新 hydrate | `src/router/DynamicRoutes.tsx:179-186` | effect 内 `setState`；**首帧 `permissions` 仍为 `[]`** | **fail-closed（首帧）**，但首帧与后续帧不一致 → 触发 #4 路由过滤抖动 |
| 9 | `ActionButtons` 组件本身 | `src/components/shared/ActionButtons.tsx:32-103` | 收到什么 actions 就渲染什么，**零权限逻辑** | **fail-open（透传）** |
| 10 | `createDeleteAction` 工厂 | `src/components/shared/ActionButtons.tsx:114-159` | 同上，**零权限逻辑** | **fail-open（透传）** |
| 11 | **页面级行操作组装** | 52 个含 `title: "操作"` 的文件 | **45 个文件完全不引用 permissions** → 按钮**永远渲染**，与权限状态无关 | **fail-open（无条件）** ← **#16 核心** |
| 12 | `VDIRow`（唯一按权限过滤的行组件） | `src/components/table/VDIRow.tsx:43` | `buttons.filter(btn => permissions.includes(btn.permission))` | **fail-closed**（正确样本） |
| 13 | 后端 `RequirePermissions` | `pkg/middleware/permission.go:277-303` | 组内**任一**权限命中即 `c.Next()`（`:293-298`）；全不命中 → 403 | **fail-closed，但粒度是「组级 ANY-of」** ← **#16 第二病灶** |
| 14 | 后端 `checkUserPermissionUncached` | `pkg/middleware/permission.go:136-245` | 精确匹配（`:145-157`）+ 「按钮类(F)子权限 → 父菜单页面访问权」继承（`:159-195`） | **fail-closed**（继承方向是 F→C，不构成越权） |
| 15 | 后端 `isSuperAdmin` 旁路 | `pkg/middleware/permission.go:247-268` | `role_key = 'admin'` → 直接放行 | **显式 fail-open（超管旁路，设计内）** |
| 16 | 后端权限缓存 | `pkg/middleware/permission.go:36` | TTL 30s；改角色后最长 30s 返回旧结果；失效函数**未接线**（`:68` 注释） | **fail-open（陈旧窗口）** ← 复测陷阱 |
| 17 | 前端 TTL 缓存 | `src/store/menuStore.ts:3-9` | 5 分钟内存缓存；`clearMenus` 在登出/401 时清（`:190-197`） | **fail-open（陈旧窗口）** ← 复测陷阱 |

**判定汇总**：
- **菜单级 / 路由级 / 后端 API 级 = fail-closed**（分级粗细不同，但方向一致）
- **行操作按钮级 = fail-open**，且是**最弱的一种 fail-open**：不是「加载中先显示、加载完再隐藏」，而是**根本不判定**（无论权限加载成功、失败还是从未加载，编辑/删除按钮恒在）
- **后端动作粒度 = 组级 ANY-of**，名义 fail-closed、实质可被只读权限穿透

### 1.3 行操作按钮的 52 文件统计画像

统计口径：`grep -rln 'title: "操作"' src/pages --include=*.tsx` = **52 个文件**。按是否引用 `permissions` 分类：

| 类别 | 文件数 | 代表文件 |
|------|--------|----------|
| **无任何权限引用（fail-open）** | **45** | `src/pages/system/user/index.tsx`、`src/pages/system/role/columns/roleColumns.tsx`、`src/pages/system/dept/columns.tsx`、`src/pages/system/menu/index.tsx`、`src/pages/network/devices/index.tsx`、`src/pages/network/credentials/index.tsx`、`src/pages/operations/workstations/columns.tsx`、`src/pages/operations/server-rooms/index.tsx`、`src/pages/duty/pools/index.tsx`、`src/pages/vdi/VDIServerConfig/index.tsx` 等 |
| 有权限引用 | 7 | `src/pages/network/ports/index.tsx:130-132`（`canWrite`）、`src/pages/vdi/VirtualMachineList/index.tsx:46,109,929-935`、`src/pages/network/mac/history/MACHistoryPage.tsx:117-119`、`src/pages/asset/reconciliation/exceptions/index.tsx:85`、`src/pages/asset/reconciliation/fix-suggestion/index.tsx:64`、`src/pages/ad-domain/configs/ReconciliationTab.tsx:76`、`src/pages/operations/workstations/LocationAliasDrawer.tsx:62-65` |

补充口径：使用 `ActionButtons` 契约组件的页面 **39 个**；使用 `createDeleteAction` 的 **11 个**。**两者都不含权限参数**——`ActionButton` 接口（`src/components/shared/ActionButtons.tsx:15-24`）根本没有 `permission` 字段，**契约层面就无法表达按钮级权限**。

### 1.4 前端 fail-open 的具体证据（静态可判，不需 PG）

**样本 1 — 用户管理页**（`src/pages/system/user/index.tsx:273-302`）：

```tsx
{
  title: "操作", key: "action", width: 200, fixed: "right",
  render: (_, record) => (
    <div className="xr-row-ops">
      <Button type="link" size="small" onClick={() => handleEdit(record)}>编辑</Button>
      <Button type="link" size="small" onClick={() => handleResetPassword(record)}>重置密码</Button>
      <Button type="link" size="small" className="xr-op-danger" onClick={() => { confirmDelete({...}); }}>删除</Button>
    </div>
  ),
}
```
整个文件**无 `permissions` / `useMenuStore` 引用**（该文件的权限判定 grep 命中 0）。→ **编辑 / 重置密码 / 删除 三键对任何能进本页面的用户恒显示**。

**样本 2 — 网络设备页**（`src/pages/network/devices/index.tsx:451-513` `buildRowActions`）：同一列内**不一致**的教科书案例：

| 动作 | 行号 | 权限 gate |
|------|------|-----------|
| 采集端口 | `:453-483` | ❌ 无 |
| 详情 | `:484-489` | ❌ 无 |
| 查看 MAC 历史 | `:490-500` | ✅ `canViewMACHistory` |
| **编辑** | `:501-506` | ❌ **无** |
| **删除** | `:507-509` | ❌ **无** |

`canViewMACHistory` 来自 `src/pages/network/devices/index.tsx:178-179`（`menuPermissions.includes("network:mac:list")`）。→ 同一个下拉菜单里，唯一被 gate 的是那个**跨模块联动入口**，而本模块自己的写操作（编辑/删除）反而没有。**这解释了用户观察到的「行为不一致」体感**。

**结构性成因**：`ActionButton` 契约（`src/components/shared/ActionButtons.tsx:15-24`）没有 `permission` 字段；契约组件 `ActionButtons.tsx:32-103` 与工厂 `createDeleteAction`（`:114-159`）都不接受权限参数。**即便页面想 gate，也没有统一入口**——只能像 devices 页那样在 actions 数组里手工 `...(canX ? [...] : [])`（devices 页 `:491-500` 的写法），45 个文件里只有 1 个这么做。

### 1.5 后端侧病灶：组级 ANY-of 抹平了动作粒度

**证据 A — 路由挂载**（`internal/api/router.go:157-162`）：

```go
users.Use(middleware.RequirePermissionsWithQuery([]string{
    string(permission.UserList),   // system:user:list
    string(permission.UserAdd),    // system:user:add
    string(permission.UserEdit),   // system:user:edit
    string(permission.UserView),   // system:user:view
}, []string{"ops:workstation:list", "ops:building:spaces:list"}, core))
```

**证据 B — ANY-of 语义**（`pkg/middleware/permission.go:276-303`）：

```go
// RequirePermissions 需要多个权限中的任意一个
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
    ...
    for _, permission := range permissions {
        if checkUserPermission(core, userID, permission) { c.Next(); return }   // ← 任一命中即放行
    }
    response.Error(c, response.ErrForbidden, "没有访问权限")
}
```

`RequirePermissionsWithQuery`（`:316-350`）同构：主数组 `permissions` 始终是 OR，`isQueryPath`（`:354-362`）只决定**是否追加** `queryExtraPermissions`，**不改变主数组的 OR 语义**。

> ⚠️ **注释与实现矛盾**：`:305-315` 的 doc 注释写「查询类路径额外纳入可视化读权限；**写操作保持严格权限**」「create/update/delete 等写操作仍只受严格权限保护，避免越权」——**实现并未做到**。写操作与读操作走的是同一个 OR 数组。

**结论**：一个只持有 `system:user:list` 的用户，`POST /system/users/{id}/delete` 会命中 `UserList` 分支 → `c.Next()` → **后端返回 200 并真的删除用户**。这比前端按钮多寡严重得多（是安全缺陷，不是 UX 缺陷）。

**证据 C — 细粒度映射表是死表**：`pkg/permission/config.go:223-289` 的 `GetRoutePermissions()` 声明了完整动作级映射（`/system/users/:id/delete` → `UserRemove`，`:231`；`/system/roles/:id/delete` → `RoleRemove`，`:244`；`/system/menus/:id/delete` → `MenuRemove`，`:255` …），配套的 `GetPermissionByPath`（`:292-302`）与 `matchPath`（`:304-330`）也已实现。**但**：

```
$ grep -rn "GetPermissionByPath" --include=*.go . | grep -v .worktrees
./pkg/permission/config.go:292:  // 定义
./pkg/permission/config.go:293:  func GetPermissionByPath(...)
./pkg/permission/config_v1201_test.go:75-83:  // 仅测试
```
→ **生产路径从不消费**。这说明团队**已经设计好了动作级权限模型，只是没有接线**。

**证据 D — network 域已做正确示范**（`internal/api/v1/network/network_router.go`）：
- 端口写操作单独子组：`SetupPortWriteRouter(ports, core)`（`:233`）→ `internal/api/v1/network/port_write_router.go:51` `RequirePermissions([network:port:write])`
- 高危恢复单独收口：`:183-192`，注释明写「**RequirePermissions 为 ANY-of 语义：主组 4 权限点任一即放行，仅持 list 只读权限的用户曾可触发恢复。restore 必须单独强制专用权限点**」

→ **同一个仓里，network 域知道这个坑并已修，system 域没修。** 这构成 #16 后端定责的**最强参照**：修法有现成范式，不需要新设计。

**已用同一 ANY-of 数组的 system 域 5 个资源组**（`internal/api/router.go`）：
| 组 | 行号 | 权限数组 |
|----|------|----------|
| `/system/users` | `:157-162` | List / Add / Edit / View |
| `/system/roles` | `:183-188` | RoleList / RoleAdd / RoleEdit / RoleView |
| `/system/menus` | `:199-204` | MenuList / MenuAdd / MenuEdit / MenuView |
| `/system/depts` | `:216-221` | DeptList / DeptAdd / DeptEdit / DeptView |
| `/system/posts` | `:231-236` | PostList / PostAdd / PostEdit / PostView |

> 注意：`permission.UserRemove` / `RoleRemove` / `MenuRemove` / `DeptRemove` / `PostRemove` 这 5 个常量**已在 `pkg/permission/config.go:16,26,34,42,48` 定义**，但在 `internal/api/router.go` 的挂载中**一次都没被引用**。

### 1.6 PG 复测定责清单（照此执行即可判定责任方）

> 前置（造数据）：在 **PG** 上建受限角色 —— 勾选「用户管理」**页面菜单**（`menu_type='C'`），**不勾**「新增/编辑/删除」等按钮菜单（`menu_type='F'`）。**必须至少勾 1 个菜单**，否则 `len(menuIds)==0` → 不进入 `ON CONFLICT` 分支 → 反而不会 500（见 §2）。
> 造好后**等 30s**（后端权限缓存 TTL，`pkg/middleware/permission.go:36`）或重启后端再复测（§1.7 陷阱 2）。

| # | 检查点 | 看什么 | 判定结论 |
|---|--------|--------|----------|
| **T1** | `POST /system/my-menus/permissions`（无 body） | 响应 `data` 数组内容 | 若含 `system:user:remove` / `system:user:edit` → **后端/数据问题**：回查角色勾选的菜单。→ **T1b** |
| **T1b** | SQL：`SELECT m.menu_name, m.menu_type, m.perms FROM sys_menu m JOIN sys_role_menu rm ON rm.menu_id=m.id WHERE rm.role_id='<受限角色id>'` | 是否勾到了 `menu_type='F'` 的按钮菜单 | 勾到 → 菜单树父子联动把 F 一起带飞（`role_service.go:108-109` 的 `MenuCheckStrictly/DeptCheckStrictly = true` 与前端 Tree 勾选行为不一致） → **数据/配置问题** |
| **T2** | 浏览器 Network → `/system/my-menus/permissions` 响应 vs React DevTools → `useMenuStore.permissions` | 两者是否一致 | Network 正确但 store 为空 → **前端 store 接线问题**（`fetchPermissions` 未触发 / TTL 缓存污染 / hydrate 竞态）；store 正确 → 进 **T3** |
| **T3** | 静态读 `src/pages/system/user/index.tsx:273-302` | 是否引用 `permissions` | **不引用 → 前端 fail-open 定责成立**（此结论**不需要 PG 即可成立**，PG 只用于证明「该用户确实受限」） |
| **T4** | ⚠️ **用 curl 直接打** `POST /system/users/{被测用户id}/delete`（带受限用户 token，绕过前端按钮） | 403 还是 200 | **403 → 后端动作粒度 OK**，#16 纯前端 UX 缺陷；**200 → 后端组级 ANY-of 缺陷坐实**（安全缺陷，优先级高于前端） |
| **T5** | `POST /system/my-menus`（受限用户） | 返回 menus 是否含未授权菜单 | 含 → 菜单级也 fail-open（问题更大） |
| **T6** | 后端日志 grep `[Permission Check]` | `pkg/middleware/permission.go:144,153,155,191` 四处 `Infof`：`userID` / `permission` / `count` / `childPerms` | 确认中间件实际检查的是哪个 permission、命中几条。注意 `:191` 的「child menu match」是 F→C 继承，属设计内 |
| **T7** | 改角色权限后立即复测 | 是否生效 | **不生效 = 缓存**（后端 30s `permission.go:36`；前端 5min `menuStore.ts:3` + sessionStorage 30min `DynamicRoutes.tsx:34`）→ **无效结论，必须清缓存重测** |
| **T8** | 换用户登录后 `sessionStorage.MENU_CACHE` | 是否残留上一用户数据 | 残留 → `menuStore.clearMenus()`（`:187-205`）未生效 → 前端越权闪现（`menuStore.ts:190-192` 注释描述的 DATA-02 场景） |

**定责判据表**：

| T1 | T2 | T3 | T4 | 结论 |
|----|----|----|----|------|
| 含超范围权限 | — | — | — | **后端/数据问题**：菜单授权越界（或菜单树父子联动带飞），前端无责 |
| 正确 | store 空 | — | — | **前端 store 接线问题**（权限没进 store） |
| 正确 | 正确 | 无判定 | — | **前端 fail-open**（按钮恒显示）—— 主结论 |
| 正确 | 正确 | 有判定但判错 | — | **前端判定逻辑 bug**（检查 includes 写反/权限点写错） |
| 正确 | 正确 | — | **200** | **后端组级 ANY-of 缺陷**（安全缺陷，需优先于前端处理） |
| 正确 | 正确 | — | 403 | 后端 OK，#16 = 纯前端 UX 缺陷 → 走前端方案 B/C |

### 1.7 复测陷阱清单（会导致「结论无效」）

1. **后端权限缓存 30s**：`pkg/middleware/permission.go:36` `mwPermCacheTTL = 30 * time.Second`；`:68-75` `InvalidateMiddlewarePermCache` 注释明写「**当前未接线**」→ 改角色后必须等 30s 或重启。
2. **前端权限缓存 5 分钟**：`src/store/menuStore.ts:3`（TTL 内存缓存）+ `:61-76` / `:103-111` 的缓存短路分支 → 换账号后若 `clearMenus` 未触发，前端仍持旧用户 permissions。
3. **sessionStorage 菜单缓存 30 分钟**：`src/router/DynamicRoutes.tsx:34` `MENU_CACHE_TTL_MS` + `:50-63` 校验 → 复测建议用**无痕窗口**。
4. **超级管理员旁路**：`pkg/middleware/permission.go:247-268` `isSuperAdmin` 按 `role_key='admin'` 直接放行 → **复测账号绝不能挂在 admin 角色上**（哪怕是通过角色继承间接挂上）。
5. **F 权限 → C 权限继承**：`pkg/middleware/permission.go:159-195` → 受限用户若持有任一 F 权限，其**父菜单**的 list/view 也会通过。设计内，但会干扰「我以为没给 list」的判断。

---

## 2. sqlite 环境造不出受限用户的根因

### 2.1 根因定位（单一代码点）

`internal/services/system/role_service.go:415-431`：

```go
// 使用 ON CONFLICT DO NOTHING 批量插入菜单权限（忽略已存在的记录）
if len(menuIds) > 0 {
    roleMenus := make([]models.RoleMenu, len(menuIds))
    for i, menuID := range menuIds {
        roleMenus[i] = models.RoleMenu{RoleID: roleID, MenuID: menuID}
    }
    if err := tx.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "role_id"}, {Name: "menu_id"}},
        DoNothing: true,
    }).Create(&roleMenus).Error; err != nil {
        return fmt.Errorf("批量分配菜单权限失败: %w", err)   // ← 事务回滚 → 500
    }
}
```

GORM 由此生成：`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (...) ON CONFLICT (role_id, menu_id) DO NOTHING`

### 2.2 为什么 sqlite 报错、PG 不报错

| 维度 | PostgreSQL | SQLite |
|------|-----------|--------|
| `ON CONFLICT (col) DO NOTHING` 语法合法性 | **即使不存在对应唯一约束也合法**，只是永不触发冲突 | **严格要求**冲突目标匹配某个 PRIMARY KEY 或 UNIQUE 约束，否则报 `ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`（SQLITE_CONSTRAINT, errcode 2067） |
| `sys_role_menu` 是否有 (role_id, menu_id) 唯一约束 | 有（见下） | **无** |
| 运行时结果 | 正常创建 | **事务回滚 → HTTP 500** |

**约束缺失的两处证据**：

1. 模型层只有**普通索引**，没有唯一约束 —— `internal/models/role.go:24-27`：
   ```go
   type RoleMenu struct {
       RoleID string `gorm:"type:uuid;not null;index:idx_sys_role_menu_role_id_menu_id"`
       MenuID string `gorm:"type:uuid;not null;index:idx_sys_role_menu_role_id_menu_id"`
   }
   ```
   `index:` 而非 `uniqueIndex:`/`primaryKey:` → AutoMigrate **不会**建立唯一约束。

2. 唯一索引的迁移**显式跳过非 PG** —— `internal/core/db/migrations/migration_206_add_user_role_role_menu_indexes.go:29-33`：
   ```go
   if !isPostgreSQL(db) {
       applogger.Infof("[迁移] 206 跳过(非 PostgreSQL)")
       return nil
   }
   ```
   且该迁移建的也只是 `CREATE INDEX IF NOT EXISTS`（**普通索引**，`:58`），**本身也不满足 SQLite 的冲突目标要求**。

→ **SQLite 下 `sys_role_menu` 根本没有 (role_id, menu_id) 唯一约束** → `ON CONFLICT (role_id, menu_id)` 必然报错。

### 2.3 影响面与同源问题

- **影响**：sqlite 环境**无法创建任何带菜单授权的角色** → 无法造受限用户 → **#16 在 sqlite 上根本无法复测**。这正是「PG 环境 UI 复测定责」这条要求的存在理由。
- **同源第二处**：`internal/services/system/role_service.go:443-448` 的 `RoleDept` 插入用同样的 `clause.OnConflict{Columns: [role_id, dept_id]}`，而 `internal/models/role.go:30-33` 的 `RoleDept` **连普通索引都没有** → sqlite 下勾部门同样会 500。
- **同仓成文旁证**（说明这是已知的方言泄漏模式，不是 v1.24 新引入）：
  - `internal/core/db/migrations/menu_grant_helpers.go:24-31`：「**SQLite 路径下 helper isPostgreSQL 守卫 return nil（SQLite 不支持 PG 的 ::uuid cast + ON CONFLICT DO NOTHING）**」
  - `internal/models/ad_push_queue.go:51-56`：「同一串文本被两处强制复用，缺一即 ON CONFLICT 失效 → `ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`」
  - `internal/core/db/ad_push_queue_bootstrap_test.go:37`：「会留下一张没有索引的表 → 运行期 ON CONFLICT 42P10」

### 2.4 修法方向（**超出 v1.24 范围，建议单独归档**）

| 方案 | 做法 | 影响面 |
|------|------|--------|
| **R1（推荐）** | 给 `RoleMenu` / `RoleDept` 加**复合主键**：`RoleID string \`gorm:"primaryKey;type:uuid;not null"\`` + `MenuID string \`gorm:"primaryKey;type:uuid;not null"\`` | 模型层改动 → AutoMigrate 自动为两种方言建出主键；PG 侧行为不变（幂等 INSERT 变真幂等）。**需确认 PG 现有数据无重复行**（否则加主键失败） |
| R2 | service 层按方言分支：`if isPostgreSQL { OnConflict } else { 先查后插 }` | 局部改动，但引入方言分支，代码变丑 |
| R3 | 去掉 `Columns` 只保留 `DoNothing`（生成 `ON CONFLICT DO NOTHING` 无冲突目标） | SQLite 3.24+ 支持无冲突目标的 `DO NOTHING`；PG 同样支持。**但**需确认 sqlite driver 版本 ≥ 3.24（`modernc.org/sqlite v1.40.1`，满足） |

**注意**：本项**不属于 v1.24 的 7 个范围项**。v1.24 只需在 §1.6 的 PG 复测中避开 sqlite 即可；R1/R2/R3 建议作为独立 phase 归档到「后端方言兼容性」主题。

---

## 3. (B) CSS 治理的架构层方案

### 3.1 `src/index.css` 的真实身份

**结论：它既不是纯「全局样式表」，也不是「组件样式表」，而是一份 5590 行的「历史沉积型单体样式文件」，同时承担了四份职责。**

实测规模：`5590` 行 / `1346` 个 `!important` / `219` 行 antd 类选择器覆盖 / `96` 个 `/* ==== */` 分区。

分区结构（按行号）：

| 行段 | 内容 | 性质 |
|------|------|------|
| `1` | `@import "tailwindcss"` | 工具链入口 |
| `3-145` | `:root` CSS 变量（品牌色/中性色/间距/阴影/圆角/过渡/字体/布局） | **设计令牌的 CSS 侧手写镜像** |
| `147-385` | `[data-color-mode="light"]` + 暗色 palette | 同上 |
| `386-2200` | **约 50 个 antd 组件逐个覆盖分区**（Header/Sidebar/Card/Button/Input/Table/Modal/Dropdown/Tag/Tabs/…/DatePicker/Form/Tree/Timeline/…/Select） | **组件样式表**（`!important` 覆盖 antd 类名） |
| `2200-2792` | Tailwind 主题配置 + 全局样式 + Tailwind 组件层 | 工具链配置 |
| `2792-5400` | **约 60 个「XXX 暗色模式」分区** | **暗色模式副本**（与 386-2200 逐组件重复一套） |
| `5400-5560` | `Theme Component Overrides (merged from theme-styles.css)` | 从 design-system 合并进来的产物 |
| `5560-5590` | 「原型 Shell 复刻（v1.23）」 | v1.23 新增 |

**两个「假单例」事实**：
- `--input-bg` 在文件内被**定义三次**：`:184`（light `#ffffff`）、`:254`（dark `#0f2e1b`）、`:4773`（原型块 `#fafaf9`）→ 同一个视觉概念有三个真相源。
- 每个 antd 组件被覆盖**两遍**（浅色一遍 + 暗色一遍），共 219 个选择器 —— 这是「暗色模式靠 `[data-color-mode="dark"]` 前缀重复一份」的直接结果。

### 3.2 `src/design-system/` 的角色与既有 token 桥接

```
src/design-system/
├── tokens/            colors.ts (ruoyiBrand 175 色) / typography.ts / spacing.ts / shadows.ts / echartsTheme.ts
├── components/        AntdThemeBridge.tsx  ← ✅ token 桥接已存在
│                       ThemeProvider.tsx   ← 只管 data-color-mode 属性
│                       LayoutProvider.tsx / ColorSwitcher.tsx / ThemeSwitcher.tsx / DensitySwitcher.tsx …
├── themes/            flat2.0 / glassmorphism / luxury-quiet / minimal / neumorphism（light/dark/index）
│                     + theme-styles.css
├── animations/        transitions.ts / keyframes.ts
└── utils/             color.ts / contrast.ts
```

**`AntdThemeBridge.tsx` 的实际作用（已确证）**：

- `:63-175` `useMemo` 构造 antd `ThemeConfig`：
  - `token`（`:74-92`）：`colorPrimary` / `colorInfo` / `colorLink` / `colorSuccess` / `colorWarning` / `colorError` / `colorTextBase` / `colorBgBase` / `colorBgContainer` / `colorBgElevated` / `colorBgLayout` / `colorBorder` / `colorBorderSecondary` / `borderRadius: 8` / `borderRadiusLG: 12` / `fontFamily`
  - `components`（`:93-168`）：`Button` / `Table` / `Input` / `Select` / `Menu` / `Tabs` / `Tag` / `Card` **八组件**
  - `algorithm`（`:65-68`）：`darkAlgorithm` / `compactAlgorithm` 与明暗/密度正交
- `:178-188`：`<ConfigProvider locale={zhCN} theme={antdThemeConfig}><App>…</App></ConfigProvider>`

**→ 结论：项目「已经有」antd theme token 桥接，ConfigProvider 路线是通的。**

文件头注释 `:22-27` 甚至已明写这条路线的前提：

> 「index.css 仅更新 CSS 变量 (--theme-primary 等)，但 **AntD 组件不消费 CSS 变量**；本组件把 ruoyiBrand.greenPrimary 直接喂给 AntD 的 ThemeConfig.token.colorPrimary」

### 3.3 两条路线为什么在打架（这是 #1 双重外框的架构根因）

`AntdThemeBridge` 的文件头 `:26-27` 声称「v1.22 起 token/components 全量映射到 ruoyiBrand，Button/Table/Input/Select/Menu/Tabs/Tag/Card 等内置组件自动品牌化，**无需逐组件 CSS override**」。

**但 `index.css` 里恰恰有 219 行逐组件 CSS override。** 两条路线并存，且 **CSS 侧的 `!important` 赢**。

**可验证的漂移证据**：

| 项 | ConfigProvider token（TS 真相源） | index.css（CSS 真相源） | 谁赢 | 漂移 |
|----|----------------------------------|------------------------|------|------|
| Input 圆角 | `AntdThemeBridge.tsx:89` `borderRadius: 8` + `:126` `Input.borderRadius: 8` | `index.css:877` `border-radius: var(--theme-radius-md) !important`；`--theme-radius-md: 6px`（`index.css:125`） | **CSS**（`!important`） | ✅ **6px ≠ 8px，实际生效 6px** |
| Input 背景 | `:122` `Input.colorBgContainer = ruoyiBrand.cream.surface` = `#FFFFFF`（`colors.ts:76`） | `index.css:874` `background: var(--input-bg) !important` = `#ffffff`（`:184`） | CSS | 色值一致（但**大小写不同**：`#FFFFFF` vs `#ffffff` → 手工镜像） |
| Input 边框 | `:86` `colorBorder = ruoyiBrand.cream.border` = `#DBD7CE`（`colors.ts:84`） | `index.css:875` `border: 1px solid var(--input-border) !important` = `#dbd7ce`（`:185`） | CSS | 色值一致（大小写不同） |
| 聚焦态 | `:123-125` `activeBorderColor` / `activeShadow` | `index.css:888-901` `:focus` 覆盖 | CSS | 双份定义 |

→ **token 注入形同虚设**：`AntdThemeBridge` 写的 `Input.borderRadius: 8` 被 `index.css:877` 的 `!important` 打成 6px。**这正是「两条路线能否收敛为一条」的答案——不能靠并存，必须选一条。**

### 3.4 双重外框（P0）的完整机理

**antd v6 的 affix-wrapper 结构**（`allowClear` / `prefix` / `suffix` 触发）：

```html
<span class="ant-input-affix-wrapper">      ← 边框由 wrapper 承担
  <span class="ant-input-prefix">…</span>
  <input class="ant-input" />               ← antd v6 默认 border:none（wrapper 内部裸 input）
  <span class="ant-input-suffix">…</span>
</span>
```

**冲突点** —— `src/index.css:871-880`：

```css
.ant-input,
.ant-input-number,
.ant-select-selector {
  background: var(--input-bg) !important;
  border: 1px solid var(--input-border) !important;   /* ← 无差别命中 wrapper 内部裸 input */
  color: var(--input-text) !important;
  border-radius: var(--theme-radius-md) !important;
  …
}
```

- `index.css:908-914` 已经正确地给 `.ant-input-affix-wrapper` 单独设了边框 —— 说明**开发者知道 wrapper 有边框**；
- 但 `:871` 的 `.ant-input` 选择器**同时命中了 wrapper 内部那个裸 `<input class="ant-input">`**，给它也加了 1px 边框 → **wrapper 一层 + 内层 input 一层 = 双重外框**；
- `!important`（`:875`）使 antd v6 自带的 `border:none` 失效。

**同仓已撞见过此问题**（局部补丁，非全局修复）：
- `index.css:2056-2060` `.dept-tree-search .ant-input-affix-wrapper .ant-input { border: none !important; … }`
- `index.css:2077-2081` `.user-form-input.ant-input-affix-wrapper .ant-input { border: none !important; … }`

→ **同一模式只在这两个作用域打了补丁，全局 `.ant-input` 规则仍在** → 其余所有带 affix 的输入框（搜索框、清空按钮框、密码框、OTP、数量框）全部双框。这是「局部补丁 vs 全局治理」的经典反模式。

### 3.5 架构决策建议

> **推荐：方案 C（分层）**，但**执行顺序上先做方案 A 的最小收敛**（把 P0 修掉且不制造新的 token/CSS 分叉）。

| 方案 | 内容 | 优点 | 缺点 | 对 v1.22 冻结契约的影响 |
|------|------|------|------|------------------|
| **A. 保留 CSS 局部覆盖** | 继续在 `index.css` 里 `!important` 覆盖 antd | 改动最小，视觉零风险 | ① token 注入永久失效（`borderRadius: 8` 永远被 6px 覆盖）；② 暗色模式双份维护（×2 成本）；③ 与 hex-exemptions 登记制天然冲突（见下） | 冻结契约「design token 体系」**名存实亡**；hex 登记制**继续失明**（CSS 不在管辖内） |
| **B. 收敛到 ConfigProvider token** | 把 219 个 antd 选择器全部迁到 `theme.components` | 单一真相源，暗色由 `algorithm` 自动派生（**暗色 60 个分区可整体删除**） | 工作量极大（暗色分区 2792-5400 约 2600 行）；antd token 未暴露的属性（如 `ant-menu-dark` 的 34 处覆盖）需改用 `ConfigProvider` 之外的手段；回归面 = 全站视觉 | 契约受益最大，但**必须分批**，且 v1.22 已冻结的 `ruoyiBrand` 色值不能变（会连带 `AntdThemeBridge:73-168` 全量重写） |
| **C. 分层：token 定色 / CSS 只管布局**（**推荐**） | ① 颜色/圆角/字号/边框色 → 走 `AntdThemeBridge` token；② 布局/结构/伪元素/滚动条 → 留 CSS；③ CSS 变量保留但**只作 CSS 侧消费**，不再有独立 hex 手写 | ① 单一真相源；② 保留 `!important` 的正当用途（布局覆盖 antd 内部结构）；③ 可增量迁移 | 需要**分类规则**（哪些属性属 token），否则「迁移」无边界 | **兼容性最好**：不动 `ruoyiBrand` 色值、不动 hex 登记制格式、不动 statusHelpers / datetime 常量 |

#### 3.5.1 方案 C 的具体分层规则

| 属性类 | 归属 | 示例（现状 → 目标） |
|--------|------|-------------------|
| 颜色（前景/背景/边框/状态色） | **TS token** → `ruoyiBrand` → `AntdThemeBridge` | `index.css:871-880` 的 `background/color/border-color` → `theme.token.colorBgContainer / colorText / colorBorder` + `theme.components.Input.*` |
| 圆角 | **TS token** | `index.css:877` `var(--theme-radius-md)` → `theme.token.borderRadius`（消除 6/8px 漂移） |
| 字号 / 字重 / 字体栈 | **TS token** | `index.css:878` `font-size: 14px` → `theme.token.fontSize` |
| 阴影 / 聚焦环 | **TS token** | `index.css:892` `box-shadow: 0 0 0 2px …` → `theme.components.Input.activeShadow`（`AntdThemeBridge:125` 已有） |
| 尺寸（width/height/controlHeight） | **TS token** | `index.css` 各处 `height` → `theme.components.*.controlHeight` |
| 布局（flex/grid/position/overflow） | **CSS** | 保留 |
| 伪元素 / 滚动条 / 选中文本 / focus-visible | **CSS** | 保留（`index.css` 已有 `Focus Visible 可访问性` 分区） |
| 结构性类覆盖（`ant-menu-dark` 34 处等） | **CSS 暂留**，逐个评估 | 远期迁 `ConfigProvider` 的 `Menu` token |

#### 3.5.2 对 v1.22 四项冻结契约的影响评估

| 冻结契约 | 位置 | 方案 C 的影响 | 结论 |
|---------|------|--------------|------|
| **design token 体系** | `src/design-system/tokens/`（`colors.ts` 的 `ruoyiBrand` 等） | ✅ **正向**：把 `index.css:29-145` 的 CSS 变量手写镜像降级为「TS token 的 CSS 投影」，消除 6/8px 圆角漂移与 `#FFFFFF`/`#ffffff` 大小写双份 | **不动契约，增强契约** |
| **hex-exemptions 登记制** | `eslint.config.js:150-153`（selector `Literal[value=/#[0-9a-fA-F]{3,8}\b/]`）+ `:184-231`（P68 豁免块）+ `.planning/phases/68-p2/hex-exemptions.md` | ⚠️ **selector 只匹配 JS/TS 的 Literal 节点，`.css` 文件完全不在管辖内** → `index.css` 里的 100+ 个 hex 处于**制度盲区**。方案 C 迁移会**自动收缩** CSS 侧 hex（好事），但**需新增 CSS 侧 lint 通道**（如 `stylelint` 的 `color-hex-length`/`declaration-property-value-disallowed-list`，或把 hex 提取到 TS token 后由现有 selector 管辖） | **需扩展契约**（新增 CSS 侧登记通道），不破坏现有格式 |
| **statusHelpers** | `src/components/shared/statusHelpers.tsx` | 不受影响（状态色映射是 TS 侧，方案 C 正是要把它归到 token 侧） | **无影响** |
| **datetime 常量** | `src/utils/datetime.ts` | 不受影响 | **无影响** |

#### 3.5.3 v1.24 的最小执行建议（不动大架构）

1. **P0 修复（必做）**：在 `index.css` 的 Input 分区（`:868-880` 之后）补一条**全局** affix-wrapper 内层 input 归零规则，与已有局部补丁（`:2056` / `:2077`）同构但不带作用域前缀：
   ```css
   .ant-input-affix-wrapper > .ant-input,
   .ant-input-affix-wrapper > input.ant-input {
     border: none !important;
     background: transparent !important;
     box-shadow: none !important;
   }
   ```
   → 一次性消除**所有** affix 输入框的双框（不只 dept-tree-search / user-form-input）。
2. **消除 token 漂移**：把 `index.css:877` 的 `border-radius: var(--theme-radius-md) !important` 与 `AntdThemeBridge:89/126` 的 `8` 对齐（**二选一**：要么 CSS 改 8px，要么 token 改 6px —— 建议统一到 token 的 8，因为 `AntdThemeBridge:88` 注释写「控件 8px 一档」）。
3. **登记制扩面**：把 `index.css` 纳入 hex 治理（新增 stylelint 通道或 TS token 化），使「design token 体系」真正成为唯一真相源。
4. **不建议在 v1.24 做**：219 个选择器的整体 token 化、暗色 60 分区的删除 —— 那是独立 phase（可命名为「CSS 治理」），v1.24 只做 P0 + 漂移消除 + 登记制扩面。

---

## 4. (C) F5 刷新落 /dashboard 的架构定位

### 4.1 根因判定：**「路由未匹配时的 fallback」，不是「认证初始化时的默认落地页」**

证据（`src/router/DynamicRoutes.tsx`）：

| 行号 | 代码 | 作用 |
|------|------|------|
| `:227-240` | `routeElements = useMemo(() => { if (allMenus.length === 0) return []; … })` | **首帧 `allMenus` 为空 → 路由表为空** |
| `:179-186` | `useEffect(() => { useMenuStore.setState({menus, allMenus, permissions}) }, [cachedMenuData, isAuthenticated])` | hydrate 发生在 **effect** 里 → **首帧之后**才生效 |
| `:275-277` | `if (allMenus.length === 0 && !cachedMenuData) return <InitializingFallback />` | **有 sessionStorage 缓存时不阻塞** → 首帧直接渲染 `<Routes>` |
| `:315` | `<Route path="*" element={<Navigate to="/dashboard" replace />} />` | **catch-all 抢先命中** → `replace` 掉原 URL |

**时间线**：

```
t0  首帧渲染
    ├─ useMenuStore.allMenus = []（store 初值，menuStore.ts:54）
    ├─ routeElements = []（:228-230）
    └─ 渲染 <Routes> → Layout > Outlet 无匹配子路由
         └─ URL = /system/user 命中 path="*"（:315）
              └─ <Navigate to="/dashboard" replace />  ← ★ URL 被改写，原路径丢失
t1  effects 执行
    ├─ hydrate effect（:179-186）→ setState({allMenus, permissions})
    └─ fetchAll() 后台 revalidate（:209-219）
t2  重新渲染 → routeElements 有内容了
    └─ 但当前 URL 已经是 /dashboard → 停在 dashboard
```

**关键佐证 —— `lastPath` 机制本来是对的**：
- `:283-286` `<Route path="/" element={<Navigate to={lastPath && lastPath !== "/" ? lastPath : "/dashboard"} replace />} />` —— 只有当 URL 恰为 `/` 时才用 `lastPath` 还原；
- `:243-247` `saveLastPath(location.pathname)` 在每次 pathname 变化时保存 → F5 前 `lastPath` **确实**是 `/system/user`；
- 但 F5 后 URL 仍是 `/system/user`（不是 `/`），所以 `path="/"` 这条路由不参与匹配 → `lastPath` 还原机制**根本没机会生效**。

→ **判定：这是「路由未匹配时的 fallback」**。`/dashboard` 不是认证初始化时的默认落地页（认证初始化由 `DynamicRoutes:258-270` 的 `initialized` / `isAuthenticated` 门控负责，走的是 `InitializingFallback` / `/login`）。

### 4.2 二阶损伤：激活的 tab 也被改掉

- `src/store/tabsStore.ts:294-303`：`persist` middleware，`name: "tabs-storage"`，`partialize` 保留 `tabs` / `activeTab` / `history` → **F5 后标签数组与 activeTab 都在 localStorage 里存活**（v1.22 契约的一部分）。
- `src/hooks/useTabSync.ts:86-88`：
  ```ts
  if (state.activeTab !== DASHBOARD_KEY) { state.setActiveTab(DASHBOARD_KEY); }
  ```
  F5 → 落 /dashboard → `useTabSync` Effect 1（`:50-98`）判定 `isDashboardPath(pathname)` → **强制把 activeTab 改成 `/dashboard`**。

→ **F5 的完整损伤 = ① URL 跳 /dashboard（原路径被 replace 丢失） + ② activeTab 被强制改为 dashboard（原激活 tab 丢失）**。用户感知是「一刷新就回到首页，之前开的页签也不见了」。

`layoutStore`（`src/store/layoutStore.ts`）本身是**衍生状态**（`:16-24`「布局配置现在从 SettingsStore 衍生」），**不参与路由落地决策** → 修复 F5 不需要动 layoutStore。`tabsStore` 只需确保 `useTabSync` 不会把 activeTab 抢走（当 pathname 恢复正确时，`useTabSync:90-96` 的 else 分支会 `addTab({key: pathname})` 并把 activeTab 设为该 pathname → **自然恢复**）。

### 4.3 修法对比（影响面差别）

| 方案 | 做法 | 影响面 | 评价 |
|------|------|--------|------|
| **F1（推荐）** | **catch-all 挂载门控**：`routeElements.length === 0` 时**不渲染** `<Route path="*">`（或整体返回 `InitializingFallback`），等路由表就绪后再挂 | `DynamicRoutes.tsx` 单文件，**3-5 行**；不动 store、不动缓存、不动 tab 契约 | ✅ 最小、最对症。语义正确：「路由表未就绪时不该做兜底跳转」 |
| **F2** | 把 sessionStorage hydrate 从 `useEffect` 提到**同步**（模块加载或 store 初始化时读） | 牵涉 `readMenuCache`（`:50-63`）的调用时机、版本校验（`:54` `version !== MENU_CACHE_VERSION`）、登出清理（`menuStore.ts:194`）的时序 | ⚠️ 改动扩散到 store + storage 契约；收益（首帧更快）不足以抵消风险 |
| **F3** | catch-all 改为重定向到 `lastPath` 而非 `/dashboard` | 改 `:315` 一行 | ❌ 治标：`lastPath` 可能是已失权路径（会循环），且 **activeTab 仍被 `useTabSync` 抢走**（`:86-88`），二阶损伤不解决 |
| **F4** | 保留 `replace` 改 `push` | 改 `:315` 一行 | ❌ 用户后退键会回到失权页面；且 activeTab 损伤仍在 |

**建议**：**F1**。附带在 `:315` 加一行注释说明「catch-all 仅在路由表就绪后生效」。

**F1 与 v1.22 tab/布局契约的关系**：修好后 F5 → URL 保持原页 → `useTabSync.ts:90-96` 的 else 分支 `addTab({key: pathname, …})` → `tabsStore.addTab`（`tabsStore.ts:64-70`）发现已存在则 `set({activeTab: tab.key})` → **activeTab 自然恢复到刷新前的 tab**。**tabsStore / layoutStore 均无需改动。**

---

## 5. (D) devices 页拆分与 react-refresh 架构约束

### 5.1 devices 页现状结构（`src/pages/network/devices/index.tsx`，**714 行**）

> 注：v1.24 任务描述写「692 行」，实测当前为 **714 行**（`wc -l`）。差异应为 v1.23 之后的增量（`index.tsx` mtime 10-28 02:46）。

**已完成的拆分**（目录已存在）：

```
src/pages/network/devices/
├── index.tsx                          714 行  ← 仍大
├── constants.ts                        47 行  ← VENDOR/DEVICE_TYPE/STATUS 选项 + 颜色映射
├── utils.tsx                           30 行
├── components/
│   ├── DeviceModals.tsx               404 行  ← 快速创建/编辑/详情 三弹窗（最重的一块）
│   ├── DeviceStatsCards.tsx            71 行
│   └── DeptTreePanel.tsx               24 行
├── hooks/
│   ├── useDeviceData.ts               110 行  ← departments/credentials/statistics
│   ├── useDeviceModals.ts              73 行  ← 弹窗可见性 state 机
│   └── __tests__/useDeviceData.test.tsx
└── __tests__/
```

**index.tsx 内部 5 个可独立块**（按行号）：

| 块 | 行号 | 行数 | 独立性 | 可提取为 |
|----|------|------|--------|----------|
| ① 列定义工厂 | `:67-169` `getDeviceTableColumns(props)` | ~103 | **高**（已是纯函数，只接 props） | `components/DeviceTableColumns.tsx`（或 `columns.tsx`） |
| ② 快速创建 + 探测逻辑 | `:285-341` `handleProbe` / `handleQuickCreate` | ~57 | 中（依赖 `quickCreateForm` / `setProbeResult` / `setProbing` / `loadStatistics`） | `hooks/useDeviceQuickCreate.ts` |
| ③ 采集端口逻辑 | `:391-418` `handleCollectPorts` | ~28 | **高**（仅依赖 `setCollectingDeviceId` / `handleApiError`） | 并入 `hooks/useDeviceActions.ts` |
| ④ 行操作组装 | `:451-513` `buildRowActions` | ~63 | **高**（纯 useCallback，只依赖 `collectingDeviceId` / `canViewMACHistory` / `navigate`） | `hooks/useDeviceRowActions.ts`（**同时是 #16 的修复落点**） |
| ⑤ 搜索字段定义 | `:539-563` `searchFields` | ~25 | **高**（纯数据） | 挪进 `constants.ts` |

**目标**：把 ①②③④⑤ 全部提出后，`index.tsx` 收敛到 **~250 行**（页面装配 + 列表请求 + 状态协调）。

**优先级建议**（结合 #16）：

1. **④ `buildRowActions` 提取 + 加权限 gate**（一次改动同时解决 #16 的前端侧与 devices 拆分）
2. **① `getDeviceTableColumns` 提取**（纯函数，零风险）
3. **③ 采集端口 / ② 快速创建提取**（中等风险，需补 hook 测试）
4. **⑤ 搜索字段**（零风险，可随手做）

**拆分时的连带纪律**：
- `DeviceModals.tsx` 已 404 行 —— 提取 ④ 后不要顺手把 ② 塞进 `DeviceModals.tsx`（会变成 460+ 行的第二个大文件）。新建 `hooks/useDeviceQuickCreate.ts`。
- `constants.ts` 目前只放**静态选项**；`searchFields` 依赖 `deviceTypeOptions`（来自 `useDict`），是**动态**的 → 应放 `hooks/useDeviceSearchFields.ts` 或留在页面内，**不要混进 constants.ts**（否则 constants 就不再是纯静态，破坏其「常量」语义）。

### 5.2 react-refresh 33 处分布画像

**实测（`npx eslint src --format json`）**：`react-refresh/only-export-components` = **33 处 / 17 个文件**。规则配置：`eslint.config.js:59` `["warn", { allowConstantExport: true }]`。

> **首要澄清**：这 33 处**与 barrel `index.ts` 无关**。react-refresh 只在「文件同时导出组件与非组件」时告警；纯 `.ts` barrel（只 re-export 组件）不触发。**全仓 33 处中没有一个是 barrel 文件。**

**两大类画像**：

#### 类 A：「伪 constants 文件」导出函数 —— 13 处 / 3 文件（39%）

| 文件 | 处数 | 行号 |
|------|------|------|
| `src/pages/operations/workstations/constants.tsx` | **7** | 17, 31, 36, 41, 46, 51, 56 |
| `src/pages/system/dept/constants.tsx` | 3 | 13, 19, 24 |
| `src/pages/system/dict/constants.tsx` | 3 | 18, 22, 29 |

**共性**：文件名叫 `constants.tsx`，但导出的是**函数**（9 处）与**复合字面量常量**（4 处）。

> ⚠️ **一个反直觉但已实测确认的规则**：`allowConstantExport: true` **并不豁免所有 const**。实测对照：
> - `workstations/constants.tsx:13` `export const STATUS_OPTIONS = WORKSTATION_STATUS_OPTIONS;`（**标识符引用**）→ **未报**
> - `workstations/constants.tsx:17` `export const TYPE_OPTIONS = [{…},{…}]`（**数组字面量**）→ **报**
> - `dept/constants.tsx:10` `export const STATUS_OPTIONS = NORMAL_STOP_OPTIONS;`（标识符引用）→ **未报**
> - `dept/constants.tsx:13` `export const EXTERNAL_ORG_OPTIONS = [...] as const;`（数组字面量）→ **报**
> - `dict/constants.tsx:14` `export const STATUS_CONFIG = ENABLE_DISABLE_TAG_CONFIG;`（标识符引用）→ **未报**
> - `dict/constants.tsx:18` / `:22` `export const DEFAULT_*_FORM_VALUES = { … }`（**对象字面量**）→ **报**
>
> → 结论：react-refresh 的 const 豁免只覆盖「初始化为**字面量或标识符引用**」的常量；**数组 / 对象字面量常量同样会被报**。这解释了类 A 中 13 处里有 4 处是「看起来完全正常的常量」。

**逐行明细**：

| 文件 | 行号 | 导出形态 |
|------|------|----------|
| `src/pages/operations/workstations/constants.tsx` | `:17` | `export const TYPE_OPTIONS = [...]`（**数组字面量**） |
| 同上 | `:31` `:36` `:41` | `export function getWorkstationTypeText` / `getWorkstationStatusText` / `getWorkstationStatusColor` |
| 同上 | `:46` `:51` | `export function renderWorkstationTypeTag` / `renderWorkstationStatusTag`（**返回 JSX** → 必须 `.tsx`） |
| 同上 | `:56` | `export function toWorkstationNode` |
| `src/pages/system/dept/constants.tsx` | `:13` | `export const EXTERNAL_ORG_OPTIONS = [...] as const`（数组字面量） |
| 同上 | `:19` `:24` | `export function renderStatusTag` / `renderExternalOrgTag`（返回 JSX） |
| `src/pages/system/dict/constants.tsx` | `:18` `:22` | `export const DEFAULT_TYPE_FORM_VALUES = {…}` / `DEFAULT_DATA_FORM_VALUES = {…}`（对象字面量） |
| 同上 | `:29` | `export function renderStatusTag`（返回 JSX） |

**典型解法**：把这 3 个文件里的**返回 JSX 的 renderer 函数**拆到独立 `.tsx`（如 `renderers.tsx`），纯数据函数与复合常量留在 `.ts`。**注意 `workstations/constants.tsx` 的 6 个函数被多页 import，重命名/拆分需同步改 import 路径**。
- ⚠️ 改名会**破坏 v1.22 的 COMP-05 迁移契约**吗？不会 —— COMP-05 契约只涉及 antd 组件封装（`AppInput`/`AppSelect`），不涉及这些 helper 文件路径。但需注意 `eslint.config.js` 的 P65/P66 豁免锚点 `__P65_EXEMPTION_FILES__` / `__P66_EXEMPTION_FILES__` 是**按文件路径**登记的 → **改名会让锚点失配**，必须同步跑 `scripts/gen-p65-exemptions.mjs` 重新生成。

#### 类 B：组件文件额外导出工具函数 / hook —— 20 处 / 14 文件（61%）

| 文件 | 处数 | 导出的非组件（**已逐行核对**） | 建议 |
|------|------|---------------------------|------|
| `src/router/DynamicRoutes.tsx` | 3 | **函数**：`getLastPath`(`:85`) / `saveLastPath`(`:95`) / `clearLastPath`(`:108`) | 抽到 `src/router/lastPath.ts`（**同时是 §4 F1 修复的邻接文件**） |
| `src/components/shared/ActionButtons.tsx` | 2 | **函数**：`createDeleteAction`(`:114`) / `createCustomDeleteAction`(`:167`) | 抽到 `src/components/shared/actionBuilders.ts`（**契约冻结组件本体不动**） |
| `src/components/shared/statusHelpers.tsx` | 2 | **函数**：`statusLabel`(`:63`) / `statusColor`(`:68`) | 抽到 `statusHelpers.utils.ts`；**注意：statusHelpers 是 v1.22 冻结契约**，拆分需在契约文档登记 |
| `src/design-system/components/ColorSwitcher.tsx` | 2 | **复合常量**：`primaryColorPresets`(`:36`) / `sidebarColorPresets`(`:57`)（数组字面量） | 抽到同目录 `colorPresets.ts` |
| `src/pages/ad-domain/configs/ReconciliationTab.tsx` | 2 | **复合常量** `MISMATCH_TYPE_META`(`:42`, 对象字面量) + **函数** `rowKeyOf`(`:66`) | 抽到 `ReconciliationTab.helpers.ts` |
| `src/design-system/components/ThemeProvider.tsx` | 1 | **hook** `useThemeContext`(`:23`) | 抽到 `ThemeContext.ts`（context 单独文件是社区标准做法） |
| `src/design-system/components/LayoutProvider.tsx` | 1 | **hook**(`:20`) | 同上，抽 `LayoutContext.ts` |
| `src/components/shared/FormModal.tsx` | 1 | **hook** `useFormModalForm`(`:61`) | 抽到 `FormModal.hooks.ts` |
| `src/components/shared/ModernTag.tsx` | 1 | **函数** `renderStatusTag`(`:75`) | 抽到 `ModernTag.helpers.ts` |
| `src/components/network/port-write/PortWriteModal.tsx` | 1 | **函数** `showAuditLinkToast`(`:78`) | 抽到同目录 `portWriteUtils.ts` |
| `src/pages/operations/building-spaces-3d/components/BuildingModel3D.tsx` | 1 | **非常量** `BuildingModel3DLazy`(`:316`, `lazy()` 调用结果) | 抽到 `BuildingModel3D.lazy.tsx` |
| `src/pages/operations/building-spaces-3d/components/FloorPlan3D.tsx` | 1 | **非常量** `FloorPlan3DLazy`(`:693`) | 同上 |
| `src/pages/settings/index.tsx` | 1 | **复合常量** `userSettingsCategories`(`:226`, 数组字面量) | 抽到 `settings/constants.ts` |
| `src/pages/system/settings/index.tsx` | 1 | **复合常量** `systemSettingsCategories`(`:30`, 数组字面量) | 抽到 `settings/constants.ts` |

**33 处的形态精确统计**（逐行核对后）：

| 导出形态 | 处数 | 说明 |
|---------|------|------|
| **函数 / hook** | **22** | 真正的 react-refresh 病灶（组件文件里塞工具函数） |
| **复合字面量常量 / 非常量 const** | **11** | 数组/对象字面量、`lazy()` 调用结果 —— `allowConstantExport: true` 不豁免 |
| 合计 | 33 | 类 A 13（函数 9 + 复合常量 4）+ 类 B 20（函数/hook 13 + 复合常量 7） |

**共性（类 B）**：
1. 全部是「**组件 + 工具函数/类型/hook** 混在一个文件」；
2. 其中 4 处是**导出的 hook**（`useThemeContext` / `useLayoutContext`）—— 这是 react-refresh 反复命中的最典型原因（社区标准解法：context 与 provider 分文件）；
3. 拆分难度低（纯搬移，无逻辑变更），但**每个文件都是 import 边界** → 必须同步更新 import 路径，且注意 `eslint.config.js` 的 P65/P66 路径锚点。

**架构约束总结**：
> **react-refresh 33 处清零是一次「文件边界重整」，不是「删代码」。** 风险不在逻辑，在**路径锚点**：
> ① `eslint.config.js:347` 的 `__P66_EXEMPTION_FILES__` 锚点（块起于 `:345`）与 `:407` 的 `__P65_EXEMPTION_FILES__` 锚点（块起于 `:405`）是**按路径登记**的（⚠ 行号已随 eslint.config.js 从 v1.23 时的 544 行缩到 **457 行**而漂移，v1.23 文档里的 `:313-368` / `:373-509` 已失效）；
> ② `.planning/phases/68-p2/hex-exemptions.md` 的逐条理由也是按路径；
> ③ v1.23 已确立「每批迁移后跑 `scripts/gen-p65-exemptions.mjs` 收缩清单」的做法。
> → **清零 33 处的执行纪律：每改一批文件 → 跑 gen 脚本 → 确认 `npm run lint` 的 error 数不增（基线 17）→ 同步更新 hex-exemptions 文档路径。**

**推荐批次顺序**（按「零 import 变更 → 单 import 变更 → 多 import 变更」排）：

> ⚠️ **W1 的扩展名变更纪律**：把 `constants.tsx` 降级为 `constants.ts` 时，必须先全仓 grep 该模块的 import 写法 —— 若任何一处写成 `from ".../constants.tsx"`（带显式扩展名），降级会直接断链。**本仓对扩展名解析敏感**（`src/components/shared/ActionButtons.tsx:12` 已有成文注释：「显式扩展名：Windows 大小写不敏感解析会命中 confirmDelete.ts」），说明历史上确实踩过「`.ts` / `.tsx` 互撞」的坑。**W1 必须先 grep 再动手，且与 §5.2 的 gen 脚本重跑同批进行。**

| 批次 | 内容 | 处数 | import 影响面 |
|------|------|------|-------------|
| W1 | 类 A 的 3 个 `constants.tsx`（拆出 renderer 函数 → 剩余部分降级为 `.ts`） | 13 | 中（多页 import；⚠ 扩展名变更风险见下） |
| W2 | `ThemeProvider` / `LayoutProvider` 的 context 抽离 | 2 | 小 |
| W3 | `DynamicRoutes` 的 lastPath 三函数 | 3 | 小（**与 §4 F1 同文件，建议合并做**） |
| W4 | `design-system/ColorSwitcher` | 2 | 小 |
| W5 | `shared/` 下的 4 个（ActionButtons / statusHelpers / FormModal / ModernTag） | 5 | 中（**statusHelpers 属 v1.22 冻结契约，需契约登记**） |
| W6 | 页面级 5 个（ReconciliationTab / BuildingModel3D / FloorPlan3D / settings×2） | 6 | 中 |

### 5.3 lint 存量精确基线（实测 `npx eslint src`）

**总览：`51 problems (17 errors, 34 warnings)`**

| 规则 | 数量 | 级别 | 备注 |
|------|------|------|------|
| `react-refresh/only-export-components` | **33** | warn | §5.2 |
| `react-hooks/exhaustive-deps` | **12** | error | 与 v1.24 口径一致 |
| `react-hooks/set-state-in-effect` | 1 | error | `src/components/dashboard/DashboardView.tsx:161` |
| `react-hooks/preserve-manual-memoization` | 1 | error | `src/hooks/useRPAProgress.ts:78`（React Compiler 规则） |
| `react-hooks/immutability` | 1 | error | `src/hooks/useRealtimeUpdates.ts:119`（`connect` accessed before declared） |
| `react-hooks/globals` | 1 | error | `src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx:51` |
| `local/no-large-dropdown-list` | 1 | error | `src/components/ui/AppSelect.tsx:16` |
| （无 ruleId） | 1 | — | `src/components/shared/PageTable.tsx:180` — **Unused eslint-disable directive**（可直接删） |

**exhaustive-deps 12 处清单**：

| 文件:行 | 缺失/多余依赖 |
|---------|-------------|
| `src/components/dashboard/DashboardView.tsx:151` | missing `message` |
| `src/design-system/components/ColorSwitcher.tsx:109` | missing `preferences.theme.cust…` |
| `src/pages/ad-domain/configs/ReconciliationTab.tsx:115` | **unnecessary** `queryTick` |
| `src/pages/ad/SyncMonitor/index.tsx:91` | missing `fetchLogs`, `fetchStatus` |
| `src/pages/system/captcha-background/hooks/useCaptchaData.ts:47` | missing `setTotal` |
| `src/pages/system/captcha-background/hooks/useCaptchaModals.ts:143,172,183,193,202` | missing `message` ×5 |
| `src/pages/system/captcha-background/index.tsx:64` | missing `loadBackgrounds`, `loadStatistics`, `paginationProps` |
| `src/pages/system/settings/default-theme.tsx:70` | missing `message` |

**⚠️ 重大信号（架构层隐患 #6）**：`eslint-plugin-react-hooks` 已升到 `^7.0.1`（`package.json:79`），该版本引入 **React Compiler 系规则**（`preserve-manual-memoization` / `immutability` / `globals` / `set-state-in-effect`）。这 4 条规则**不是项目自己写的**，是插件升级带来的**新债务源**。它们指向同一个架构问题：**项目尚未适配 React Compiler 的纯渲染约束**。这需要一次**架构级决策**（是否启用 React Compiler / 是否全面转向「派生状态而非 effect 同步」），而不是逐条 disable。

---

## 6. 其他架构层隐患（本研究发现，v1.24 范围外，建议归档）

| # | 隐患 | 证据 | 严重度 | 建议 |
|---|------|------|--------|------|
| **1** | **`pkg/permission/config.go` 的细粒度动作级映射表是死表** | `GetPermissionByPath`（`:292-302`）仅被 `pkg/permission/config_v1201_test.go:82` 调用；生产路径无引用 | **高** | 让 `RequirePermissions` 支持「按路径查表」模式（`GetPermissionByPath` 已就绪），或明确标注为「仅文档/测试用」并从生产期望中移除 |
| **2** | **system 域 5 个资源组用组级 ANY-of 数组** | `internal/api/router.go:157,183,199,216,231` | **高（安全）** | 照抄 `internal/api/v1/network/network_router.go:183-192` + `port_write_router.go:51` 的「写操作单独子组」范式，给 5 个组的 delete/update 单独收口 |
| **3** | **后端权限缓存 30s 且失效函数未接线** | `pkg/middleware/permission.go:36`（TTL）、`:68-75`（`InvalidateMiddlewarePermCache` 注释「当前未接线」） | **中** | 在角色/用户角色变更成功后调用失效函数，消除 30s 提权/降权窗口 |
| **4** | **前端权限双层缓存（5min TTL + 30min sessionStorage）** | `src/store/menuStore.ts:3,61-76,103-111`；`src/router/DynamicRoutes.tsx:34,50-63` | **中** | 明确「权限变更后何时生效」的对外契约，写进 UI 提示或文档 |
| **5** | **两个 antd `ConfigProvider` 嵌套** | `src/App.tsx:9,48`（`@/components/ConfigProvider`）内嵌 `src/design-system/components/AntdThemeBridge.tsx:178`（antd `ConfigProvider`） | **中** | 确认 `@/components/ConfigProvider` 是否也注入 theme；若两者都设 theme，**后者覆盖前者**（React Context 就近优先）→ 需核对 `src/components/ConfigProvider` 的内容 |
| **6** | **未适配 React Compiler 的纯渲染约束** | `eslint-plugin-react-hooks@^7` 的 4 条新规则已产生 3 条 error（§5.3） | **中** | 架构决策：是否启用 React Compiler；否则把 4 条新规则降为 warn 并登记豁免 |
| **7** | **`index.css` 的 CSS 变量被就地重定义 3 次** | `--input-bg`：`:184`（light）/ `:254`（dark）/ `:4773`（原型块 `#fafaf9`） | **中** | 原型 Shell 复刻区块（v1.23）应使用独立变量名，不覆写全局 `--input-*` |
| **8** | **TS token 与 CSS token 已漂移（圆角 8 vs 6）** | `AntdThemeBridge.tsx:89,126`（8）vs `index.css:125,877`（6） | **中** | §3.5.3 第 2 步 |
| **9** | **hex-exemptions 登记制对 CSS 完全失明** | `eslint.config.js:150` selector 是 `Literal[...]`，只匹配 JS/TS 节点 | **中** | 新增 CSS 侧治理通道（§3.5.2） |
| **10** | **`ActionButton` 契约无 `permission` 字段** | `src/components/shared/ActionButtons.tsx:15-24` | **中** | 契约层解：加 `permission?: string` + 组件内 `useMenuStore` 过滤（一次性解决 45 个文件）。⚠️ 该接口是 v1.22 冻结契约（T1），**扩字段是加法，不破坏兼容** |
| **11** | **登录时 permissions 被重复拉取** | `src/pages/login/index.tsx:74` `Promise.all([fetchMenus(), fetchPermissions()])`，而 `fetchMenus` 内部的 `fetchMenuData()`（`menuStore.ts:44-48,80`）已含 `getUserPermissions()` | **低** | 改为只调 `fetchAll()` |
| **12** | **文档漂移** | `CLAUDE.md` 写 React 19.2 / antd 6.1；实际 `package.json:39` antd `^6.6.4`、`:46` react `^19.3.0` | **低** | 更新 CLAUDE.md 的技术栈版本表 |
| **13** | **`ActionButton` 契约无 loading / aria 位** | `src/pages/network/devices/index.tsx:458-464` 自述「契约 ActionButton 无 loading 位：采集进行中降级为 disabled（记 PR 偏差）」 | **低** | 契约加法：`loading?: boolean` + `aria-busy` 透传 |

---

## 7. Recommended Project Structure（v1.24 增量，其余全复用）

```
xingran-react-frontend/src/
├── index.css                          (MOD — ① 全局 affix-wrapper 内层 input 归零
│                                        ② --theme-radius-md 与 token 对齐
│                                        ③ 删掉 PageTable.tsx:180 的失效 disable)
├── design-system/
│   └── components/
│       ├── AntdThemeBridge.tsx        (不动 — token 桥接已完备，§3.2)
│       └── ThemeProvider.tsx          (MOD — context 抽到 ThemeContext.ts, 消除 react-refresh:23)
├── router/
│   ├── DynamicRoutes.tsx              (MOD — ① catch-all 挂载门控（F1）
│   │                                        ② lastPath 三函数抽到 lastPath.ts
│   │                                        ③ getLastPath/saveLastPath/clearLastPath 移出)
│   └── lastPath.ts                    (NEW — 路径记忆工具)
├── components/shared/
│   ├── ActionButtons.tsx              (MOD — createDeleteAction / createCustomDeleteAction
│   │                                        抽到 actionBuilders.ts；接口加 permission?: string)
│   ├── actionBuilders.ts              (NEW)
│   ├── statusHelpers.tsx              (MOD — :63/:68 抽出；v1.22 冻结契约需登记)
│   ├── statusHelpers.utils.ts         (NEW)
│   └── ColumnConfigModal.tsx          (MOD — 仅加豁免注释，:136 合法原因）
├── components/table/
│   └── VDIRow.tsx                     (不动 — 唯一 fail-closed 的按钮级过滤样板，§1.2 #12)
├── pages/network/devices/
│   ├── index.tsx                      (MOD — 714 → ~250 行；④ buildRowActions 提取 + 权限 gate)
│   ├── components/DeviceTableColumns.tsx  (NEW — ① 提取自 index.tsx:67-169)
│   ├── hooks/useDeviceActions.ts      (NEW — ③ 采集端口, 提取自 :391-418)
│   ├── hooks/useDeviceQuickCreate.ts (NEW — ② 快速创建+探测, 提取自 :285-341)
│   └── hooks/useDeviceRowActions.ts   (NEW — ④ 提取自 :451-513 + #16 权限 gate)
├── pages/operations/workstations/
│   └── constants.tsx                  (MOD — 拆分：6 个函数移出 → 消除 react-refresh 7 处)
├── pages/system/dept/constants.tsx   (MOD — 消除 3 处)
├── pages/system/dict/constants.tsx   (MOD — 消除 3 处)
└── pages/settings/ + pages/system/settings/
                                          (MOD — 各消除 1 处)

internal/services/system/role_service.go        (归档到「后端方言兼容性」主题 —
                                                  §2.4 R1：RoleMenu/RoleDept 加复合主键)
internal/models/role.go                          (同上 — :24-33 加 primaryKey)
internal/api/router.go                           (归档 — §6 隐患 2：system 域 5 组写操作单独收口)
pkg/middleware/permission.go                     (归档 — §6 隐患 3：接线 InvalidateMiddlewarePermCache)
```

---

## 8. 风险优先级与 Scaling

| 序 | 风险 | 触发条件 | 影响 | 缓解 |
|----|------|----------|------|------|
| 1 | **后端组级 ANY-of 造成越权**（§1.5） | 任何只持 list 权限的用户调 delete | **数据丢失（用户/角色/菜单被删）** | PG 复测 T4 必须打；修复照抄 network 域范式 |
| 2 | **F5 修复引入路由循环**（§4.3 F1） | catch-all 门控写错，导致 `/` 永远不落地 | 全站白屏 | F1 改动限于 3-5 行；必须验证「F5 原页」「F5 /」「F5 无权限路径」三态 |
| 3 | **react-refresh 清零引发路径锚点失配**（§5.2） | 改文件名未同步 gen 脚本 | lint error 反弹、豁免审计失效 | 每批次后跑 `scripts/gen-p65-exemptions.mjs` + 核对 `npm run lint` error ≤ 17 |
| 4 | **CSS 迁移引发全局视觉回归**（§3.5） | 一次性迁 219 个选择器 | 全站视觉 | 分层推进；每次只迁一个组件族；暗色模式必须同步验证 |
| 5 | **React Compiler 规则持续增生**（§6 #6） | eslint-plugin-react-hooks 继续升级 | lint error 单调上升 | 一次架构决策（启用/不启用），不要逐条 disable |
| 6 | **中间件 30s 缓存导致复测结论无效**（§1.7） | 改角色后立即复测 | 误判 | 复测 SOP 固化「等 30s / 重启 / 无痕窗口」 |

**Scaling Priorities**：
1. **第一瓶颈：权限模型的粒度**（前端按钮级 + 后端动作级）。当前 45/52 的行操作文件无判定、5 个后端资源组为组级 OR —— 这是**随模块增长线性放大的债**（每加一个模块就要重复一次「要不要加 gate」的决策）。**应先做契约层解**（`ActionButton.permission` + 后端按路径查表），而不是逐页补。
2. **第二瓶颈：CSS 治理**（1346 个 `!important` / 219 个 antd 选择器 / 2 份暗色副本）。每加一个 antd 组件就要在 2 处（浅 + 暗）追加覆盖。**分层（方案 C）能把边际成本从 2 降到 1，最终降到 0**。
3. **第三瓶颈：路由/布局就绪时序**（§4）。随动态菜单数量增长，hydrate 竞态窗口会变长。

---

## 9. Sources（本次研究证据清单）

**前端源码**：
- `xingran-react-frontend/src/components/shared/ActionButtons.tsx`（全文 173 行，含 `createDeleteAction` 工厂）
- `xingran-react-frontend/src/store/menuStore.ts`（全文 235 行）
- `xingran-react-frontend/src/utils/authHelpers.ts`（全文 104 行 —— **确认：无任何 hasPerm/checkPerm 权限工具函数**）
- `xingran-react-frontend/src/router/DynamicRoutes.tsx`（全文 319 行）
- `xingran-react-frontend/src/router/RouteGuard.tsx`（全文 67 行）
- `xingran-react-frontend/src/router/routeConfigManager.ts:90-107`
- `xingran-react-frontend/src/components/layout/sidebar.tsx:89-190`
- `xingran-react-frontend/src/store/tabsStore.ts`（全文 345 行）、`src/store/layoutStore.ts:1-70`
- `xingran-react-frontend/src/hooks/useTabSync.ts`（全文 152 行）
- `xingran-react-frontend/src/design-system/components/AntdThemeBridge.tsx`（全文 192 行）、`ThemeProvider.tsx`（全文 61 行）
- `xingran-react-frontend/src/design-system/tokens/colors.ts:72-89`
- `xingran-react-frontend/src/index.css`（5590 行；重点 `:1-145`、`:147-385`、`:868-930`、`:2048-2092`、`:4773`、`:5400-5590`）
- `xingran-react-frontend/src/pages/system/user/index.tsx:273-302`
- `xingran-react-frontend/src/pages/network/devices/index.tsx`（全文 714 行）
- `xingran-react-frontend/src/components/operations/WorkstationDeviceTable/index.tsx:580-612`
- `xingran-react-frontend/src/lib/opsApi.ts:605-635`
- `xingran-react-frontend/src/components/table/VDIRow.tsx:21-43`
- `xingran-react-frontend/src/components/shared/ColumnConfigModal.tsx:110-169`
- `xingran-react-frontend/eslint.config.js:57-59,112-168,184-238,264-269`（全文 457 行）
- `xingran-react-frontend/package.json:39,46,79-80,90`

**后端源码**：
- `internal/api/v1/system/role_handler.go`（全文 279 行）
- `internal/services/system/role_service.go:80-230,380-515`
- `internal/models/role.go`（全文 48 行）
- `internal/core/db/migrations/migration_206_add_user_role_role_menu_indexes.go`（全文 68 行）
- `internal/core/db/migrations/menu_grant_helpers.go`（全文 50 行）
- `internal/models/ad_push_queue.go:51-56`
- `internal/core/db/ad_push_queue_bootstrap_test.go:8,37,46,105`
- `pkg/middleware/permission.go:1-435`（含 `RequirePermissions:277-303`、`RequirePermissionsWithQuery:316-350`、`isQueryPath:354-362`、`checkUserPermissionUncached:136-245`、`isSuperAdmin:247-268`、`mwPermCacheTTL:36`、`InvalidateMiddlewarePermCache:68-75`）
- `pkg/permission/config.go:1-330`（含 `GetRoutePermissions:223-289`、`GetPermissionByPath:292-302`、`matchPath:304-330`）
- `internal/api/router.go:140-269`（system 域权限挂载）
- `internal/api/v1/network/network_router.go:175-239`、`internal/api/v1/network/port_write_router.go:14-51`
- `internal/api/v1/operations/workstation_device_handler.go:147-222`（3f130 诊断端点）

**实测**：
- `npx eslint src --format json`（2026-10-28）→ 51 problems / 17 errors / 34 warnings；react-refresh 33 处 / 17 文件；exhaustive-deps 12 处
- `grep -rln 'title: "操作"' src/pages --include=*.tsx` → 52 文件（7 有权限引用 / 45 无）
- `grep -rn 'physical-diagnosis' src` → 0 命中（前端未接诊断端点）
- `grep -c '!important' src/index.css` → 1346；`grep -c '^\s*\.ant-' src/index.css` → 219
- `grep -rn "GetPermissionByPath" --include=*.go .` → 仅定义 + 测试
- `grep -rn "useMenuStore|hasPerm|checkPerm" src` → 权限工具函数**仅 5 个页面**使用

---
*Architecture research for: v1.24 架构层方案与代码审计*
*Researched: 2026-10-28*
