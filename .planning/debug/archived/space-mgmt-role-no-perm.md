---
slug: space-mgmt-role-no-perm
status: resolved
trigger: "我创建了空间管理 角色，有空间管理菜单下的所有权限，但是用户登录后提示没有权限，请检查原因"
created: 2026-06-17
updated: 2026-06-17
root_cause: "权限标识命名空间割裂——「楼宇空间」菜单 perms 前缀 ops:building:spaces: 与其复用的 building/floor/workstation list 接口要求的 ops:building:/ops:floor:/ops:workstation: 权限完全不重叠；空间管理角色即使勾全「楼宇空间」菜单也拿不到这三个 list 接口权限 → 403 → 前端显示没有权限"
---

# Debug Session: 空间管理角色登录后提示没有权限

## Symptoms

- **预期行为**: 拥有「空间管理」角色的用户登录后能正常进入空间管理菜单下的页面
- **实际行为**: 用户登录后访问菜单页时提示"没有权限"
- **错误信息**: 暂无具体错误码 / 错误内容，需进一步抓取
- **时间线**: 刚发现 / 首次出现
- **复现步骤**:
  1. 创建「空间管理」角色
  2. 在角色编辑页面通过菜单树勾选空间管理菜单下的所有菜单项
  3. 给用户绑定该角色（用户同时还有其它角色）
  4. 用户登录后访问空间管理菜单 → 提示没有权限
- **对照实验**: 管理员角色用户能正常访问空间管理菜单 → 菜单配置本身正确

## 关键线索

1. **触发层是前端菜单页**，不是具体 API → 重点查路由权限 / 按钮权限拦截
2. **用户有多个角色** → 可能存在角色权重覆盖、菜单合并遗漏、状态异常(0/1 停用) 等问题
3. **UI 树形勾选赋权** → 可能存在菜单ID没正确写入 sys_role_menu / 树形勾选部分父节点漏选 / 菜单状态异常
4. **管理员可访问** → 排除菜单配置、菜单状态本身的问题

## 重要发现 (evidence-gathering)

### A. "空间管理" 实际是 "楼宇空间"

- 实际菜单名为 "楼宇空间" (Building Spaces)：
  - ID: `550e8400-e29b-41d4-a716-446655441100`
  - perms: `ops:building:spaces:list`
  - path: `/ops/building-spaces`
- 用户口语化称之为「空间管理」实际指向 "楼宇空间"。
- 此处无关 bug 本身 —— 用户描述的是现象。

### B. JWT 中不存 permissions 字段

`internal/core/security/jwt.go:99` `GenerateTokenPair(userID, username string, roles []string)` 只存：
- `user_id`, `username`, `roles` (角色 UUID 列表)

权限是 **每次请求动态查库**：
`pkg/middleware/permission.go:72-78`
```sql
SELECT COUNT(DISTINCT rm.menu_id)
FROM sys_user_role ur
INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
INNER JOIN sys_menu m ON rm.menu_id = m.id
WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?
```
- 排除了 `JWT 权限过期/缓存陈旧` 假设。
- 每次请求都基于 `sys_user_role`、`sys_role_menu`、`sys_menu` 实时 join。

### C. menu_service.GetUserMenus 在前端路由加载时使用

`internal/api/v1/system/menu_handler.go:289` `GetUserMenus` → service.GetUserMenus(userID)
- 用于前端渲染左侧菜单树
- `menu_service.go:355-411` 流程：
  1. 查 `sys_user_role` 拿 roleIDs
  2. 查 `sys_role_menu` 拿 menuIDs (DISTINCT)
  3. 查 `sys_menu` WHERE `id IN ? AND status=? AND visible=?`
  4. **appendAncestorMenuIDs** 自动补全父节点
  5. buildMenuTree 构树
- **关键 filter**: `visible=1` (VisibleShow) —— 隐藏菜单不返回
- **未走缓存**：menuCacheService 实现了 `GetTree`、`GetTreeWithCache`、`GetRouterDataWithCache`，但 **没有实现 `GetUserMenus`**
  - `menuCacheService` 嵌套 `*menuService` → `GetUserMenus` 直接走 base (无缓存)
  - 每次调用实时查库，结果是 fresh 的。

### D. 权限中间件 checkUserPermission 行为

`pkg/middleware/permission.go:63-149` 三级匹配：
1. **精确匹配**: `m.perms = ?` AND `m.status = 0` (正常)
2. **子菜单继承父菜单**: 仅当被检查的 perm 是 `menu_type='C'` 时，让该菜单的 `menu_type='F'` 按钮子菜单权限自动通过（反之不行）
3. **模块级匹配**: 例如 `system:menu:list` → 退化为 `system:menu`
- **重要 filter**: `AND m.status = ?` (`MenuStatusNormal = 0`) — 停用菜单（status=1）不计入权限

### E. role 树形勾选写入逻辑

`role_service.go:357-403` `assignRoleMenusAndDepts`：
- Update 时：删除 role_id 对应但不在 menuIds 列表里的旧关联
- 然后 ON CONFLICT DO NOTHING 批量插入
- **不自动添加父节点**（注释说"让前端通过 Tree 的父子关联自动勾选父菜单"）
- **不自动添加子节点**（如果用户只勾选了父节点）

### F. menu_router / frontend 路由加载

`internal/api/v1/system/menu_router.go` —— 待查 GetUserMenus 路由注册

### G. 角色状态停用也会让角色失效？

`checkUserPermission` SQL 查的 join:
```sql
FROM sys_user_role ur
INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
INNER JOIN sys_menu m ON rm.menu_id = m.id
WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?
```
- 没有 join `sys_role`，**没有过滤 role.status**
- 所以 **角色停用不会影响权限合并**（但前端 `/getInfo` 会过滤停用角色返回的菜单）

## 当前 hypothesis 候选

### H1 (高): 用户某角色 `status=1` (停用) 时，GetUserMenus 不返回该角色菜单

- GetUserMenus 只看 `sys_user_role` 不看 `sys_role.status`
- 但 `checkUserPermission` 也不看 role.status
- 这俩都不过滤停用角色 — 排除

### H2 (高): 某个父菜单 `status=1` / `visible=0` 时，子菜单被连累

- `checkUserPermission` 查 m.status = 0
- 如果用户勾了子菜单，sql 是 `m.perms = ?` 直接命中子菜单的 perms
- **不会** 走父菜单 status 检查
- 但 `GetUserMenus` 走 `id IN ? AND status=0 AND visible=1` → 如果父菜单不可见但子菜单可见，子菜单 ID 不会出现在 menuIDs 集合里 → **子菜单被父菜单 visible 过滤连累**
- 这条**不成立** — 因为查 sys_role_menu 拿到的 menuIDs 是子菜单的 ID，不是父菜单的

### H3 (高): 树形勾选只勾了子菜单没勾父菜单 → 前端展示时找不到父级导航

- 前端 antd tree 父子不联动 / 后端 GetUserMenus 自动 append 父菜单
- 即使子菜单被勾选，`appendAncestorMenuIDs` 会在 service 层补全
- 这条**不成立** — 后端会补全

### H4 (中): 多个角色合并时，某个 role.menuIds 部分冲突或子菜单 status/visible 异常

- 比如 "空间管理" 角色只勾了"楼宇空间"这一项没勾子按钮权限
- 楼宇空间本身的 perms=`ops:building:spaces:list`，需要这一权限
- 用户访问 /ops/building-spaces → checkUserPermission 检查 `ops:building:spaces:list` → 命中 → 通过
- 但访问子页面如"楼宇空间3D"，perms=`ops:building:spaces:3d:list` → 如果空间管理角色没勾 3D 子菜单 → 不通过
- 这条**取决于用户实际勾了什么**

### H5 (中): 缓存导致读到了旧数据

- `permissionCacheService` 没有，但 `roleCacheService` 的 `GetMenusWithCache` 有缓存
- 但**没有代码路径**调用 `GetMenusWithCache` 来影响权限判断
- 实际上 `checkUserPermission` 是直接走 SQL
- 这条**不成立** — 权限检查不依赖 role 缓存

### H6 (高): 角色创建后没有真正写入 sys_role_menu / 写入的 menu_id 是错误的

- 这取决于 frontend tree-select 怎么传 menuIds
- 角色创建后，需要检查 sys_role_menu 表

## 关键待验证点

1. 用户的 sys_role_menu 表中是否存在 `ops:building:spaces:*` 系列 perm 的 menu_id
2. 用户的 sys_user_role 中角色是否包含"空间管理"角色且 status=0
3. 当用户访问"楼宇空间"页面时，checkUserPermission 实际查询的 perm 是什么
4. 前后端路由是否一致 —— frontend 路径 `/operations/building-spaces` 还是 `/ops/building-spaces`

## Current Focus

- hypothesis: **H4 / H6** —— 用户实际勾选的菜单与前端 antd tree 父子联动的行为导致 sys_role_menu 缺少关键子菜单的 perms，导致访问子页面时 checkUserPermission 失败
- test: 1) 读 router.go 中 operations 路由组的权限中间件配置
       2) 读 frontend 楼宇空间页面的路由路径 / 权限声明
       3) 验证 sys_role_menu 在该角色下完整包含菜单树
- expecting: 找到 router 注册 / 前端路由路径 / 后端 sys_menu path 之间的不匹配，或前端 tree 半选/全选的差异
- next_action: 1) 读 internal/api/router.go 中 operations router 注册
              2) 读 frontend 楼宇空间页面入口 (xingran-react-frontend/src/pages/operations/building-spaces/)
              3) 读 frontend 的 role edit 页面 tree-select 提交逻辑
              4) 读 menu_router 中 GetUserMenus 的实际路径

## Eliminated

- hypothesis: JWT 权限过期 / 缓存陈旧
  evidence: JWT 不存 permissions 字段；checkUserPermission 每次实时查库；menuCacheService 的 GetUserMenus 没有缓存实现
  timestamp: 2026-06-17
- hypothesis: 用户某角色 status=1 停用导致权限合并失效
  evidence: checkUserPermission 的 SQL 没有 join sys_role 也不过滤 role.status
  timestamp: 2026-06-17

## Evidence

- timestamp: 2026-06-17
  checked: pkg/middleware/permission.go, internal/core/security/jwt.go, internal/services/system/menu_service.go, internal/services/system/menu_cache_impl.go, internal/services/system/role_service.go, internal/api/v1/system/menu_handler.go
  found: |
    - JWT 只存 roles (UUIDs),不存 permissions
    - 权限检查 3 级：精确 → 子菜单继承父菜单(仅 C→F) → 模块级退化
    - 权限检查 SQL 过滤 m.status=0,但不过滤 role.status,不过滤 m.visible
    - GetUserMenus 过滤 status=0 AND visible=1,无缓存
    - roleCacheService 有 List/GetByID/GetAllEnabled/GetMenusWithCache/GetDeptsWithCache
      的缓存实现,但权限检查不读 role cache
    - role 树形勾选写入 assignRoleMenusAndDepts 不自动补全父/子节点
  implication: 排除 JWT 缓存 / 角色停用 / 缓存陈旧；缩小到 H4/H6 (用户实际勾选内容)

- timestamp: 2026-06-17 (orchestrator 接管, 闭环根因)
  checked: xingran-react-frontend/src/pages/operations/building-spaces/index.tsx, src/lib/opsApi.ts, src/router/routeConfigManager.ts, src/utils/errorHandler.ts, internal/api/router.go, migrations/archive/legacy-2026-06-15/029_add_building_spaces_menu.sql
  found: |
    ## ROOT CAUSE (确认): 权限标识命名空间割裂

    「楼宇空间」(用户口语"空间管理") 是一个**只读 3D 可视化页面**，它本身在后端没有独立路由
    (router.go 的 ops 组里没有 /building-spaces 路由)，而是**复用**楼宇/楼层/工位三个 CRUD 模块
    的 list 接口拼装数据：

      前端 building-spaces/index.tsx:25-29
        buildingApi.list()      → POST /ops/building/list
        floorApi.list()         → POST /ops/floor/list
        workstationApi.list()   → POST /ops/workstation/list

      后端 router.go (路由组级 RequirePermissions, OR 逻辑):
        /ops/building     (499-504) 需要 ops:building:{list,add,edit,delete} 任一
        /ops/floor        (524-529) 需要 ops:floor:{list,add,edit,delete} 任一
        /ops/workstation  (561-566) 需要 ops:workstation:{list,add,edit,delete} 任一

    而「楼宇空间」菜单 (migration 029 seed) 的 perms 是:
        ops:building:spaces:list     (菜单 550e8400-...100)
        ops:building:spaces:query   (按钮 550e8400-...101)
        ops:building:spaces:3d:list (楼宇空间3D 550e8400-...102)

    空间管理角色勾全「楼宇空间」菜单 → 用户持有 ops:building:spaces:* 系列 perms
    → 与 ops:building:list / ops:floor:list / ops:workstation:list **零交集**
    → 三个 list 请求全部 403 Forbidden
    → 前端 src/utils/errorHandler.ts:184 把 HttpErrorType.FORBIDDEN 渲染为「没有权限访问」

    管理员能访问是因为持有 ops:building:list 等全套 perms (或超管 bypass)。
  implication: |
    不是前端路由守卫拦截 (routeConfigManager.hasPermission 的 fallback meta 不设 permissions,
    对楼宇空间菜单直接放行); 不是 JWT/缓存/角色停用问题; 是**权限模型设计断层**:
    可视化页面复用了 CRUD 接口, 但权限标识没有打通。

## Resolution

root_cause: 权限标识命名空间割裂 (ops:building:spaces:* vs ops:building:/floor:/workstation:)
fix: 新增 RequirePermissionsWithQuery 中间件, 对查询类路径(/list,/tree)额外接受 ops:building:spaces:list 可视化读权限; building/floor/workstation 三个路由组改用该中间件, 写操作仍受严格权限保护
verification: |
  - go build ./... 通过
  - go test ./pkg/middleware/ 通过 (含新增 permission_query_test.go: TestIsQueryPath + TestRequirePermissionsWithQuery 6 个子测试)
  - 测试证明: 空间角色(ops:building:spaces:list) 读 /list=200, 写 create/delete=403; 楼宇角色(ops:building:list) 读写均通过; 无权限用户=403
files_changed:
  - pkg/middleware/permission.go (新增 RequirePermissionsWithQuery + isQueryPath, import strings)
  - internal/api/router.go (building/floor/workstation 三组 RequirePermissions → RequirePermissionsWithQuery)
  - pkg/middleware/permission_query_test.go (新增回归测试)

### 修复方案候选 (权衡)

**A (推荐, 治本且安全)**: 把 /ops/{building,floor,workstation}/list 三个**查询接口**
从路由组级 RequirePermissions 拆出, 单独挂权限, 放宽为接受 ops:building:spaces:list 等
可视化读权限。空间管理角色可读 list, 但不能增删改 (create/update/delete 仍受组级保护)。
- 改动: internal/api/router.go (3 处), 把 list 路由的权限检查从 group 级移到 route 级
- 风险: 低; 反而修复了 permission-control-bypass-network-devices 里 OR 逻辑导致越权的隐患

**B (危险, 不推荐)**: 直接给三个路由组 RequirePermissions 列表追加 ops:building:spaces:list
- 风险: RequirePermissions 是路由组级 OR, 加了之后空间管理用户对整个 /ops/building 组
  (create/delete/batch/geocode/excel 导入导出) 全部放行 → 严重越权

**C (最干净, 工作量稍大)**: 后端新增只读聚合接口 (如 POST /ops/building-spaces/data),
权限要求 ops:building:spaces:list; 前端 building-spaces 页面改调该接口
- 改动: 新 handler+service+router, 改前端 1 处调用

**D (零代码, 配置 workaround)**: 告知用户空间管理角色必须同时勾选
「楼宇管理」「楼层管理」「工位管理」三个菜单的**查询**按钮权限
(ops:building:list / ops:floor:list / ops:workstation:list)
- 优点: 立即可用; 缺点: 治标, 每个空间管理角色都要这么配
