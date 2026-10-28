---
phase: 84
title: 权限模型三缺口 技术方案研究
date: 2026-10-28
researcher: gsd-phase-researcher
requirements: PMODEL-01, PMODEL-02, PMODEL-03, PMODEL-04
confidence_marks: HIGH=读代码实证(附 文件:行号) / MEDIUM=推导 / LOW=需实机验证
---

# Phase 84 研究报告：权限模型三缺口

> 只读研究产物。所有行号为 2026-10-28 工作区快照，实施前如有重构请重新核对。

## 现状事实

### A. data_scope 消费现状（PMODEL-01）

**A1. DataScope 枚举（HIGH）** — `internal/models/base.go:70-77`：

```go
type DataScope int
DataScopeAll       DataScope = 1 // 全部数据
DataScopeCustom    DataScope = 2 // 自定义数据
DataScopeDept      DataScope = 3 // 本部门数据
DataScopeDeptChild DataScope = 4 // 本部门及子部门数据
DataScopeSelf      DataScope = 5 // 仅本人数据
```

**A2. 中间件语义（HIGH）** — `pkg/middleware/permission.go`：
- `DataScopePermission`（:470-499）：超管 `isSuperAdmin` 直接 `c.Next()` **不写 context**（:480-483）；否则 `getUserMaxDataScope` 取用户全部启用角色（`r.status = RoleStatusEnabled`）的 `MAX(data_scope)`，无角色时 `COALESCE` 回落 `DataScopeSelf(5)`（:502-513）。然后 `c.Set("data_scope", scope)` + `c.Set("user_id", userID)`（:494-495）。**注意：`c.Set` 只进 gin.Context，不进 `c.Request.Context()`** —— service 层拿不到，必须 handler 显式提取传参。
- `ApplyDataScope`（:516-579）：按 scope 往 `*gorm.DB` 追加 Where；字段名经 `isValidDataScopeField` 白名单校验（:518）；`Custom` 查 `sys_role_dept`（:526-539）；`Dept` 查 `sys_user.dept_id` 等值（:540-553）；`DeptChild` 递归 `getChildDepts`（:554-572）；`Self` 写的是 **`deptField = userID`（:575）—— 对 `sys_user.dept_id` 这是语义 bug**（拿 user id 与 dept id 比较），仅当 deptField 恰为「用户维度字段」时才正确（VDI 场景 `bound_user_id` 就是这样用的）。
- `getChildDepts`（:582-593）：逐层 `parent_id = ?` 递归查询，无缓存、无环检测。
- `ApplyDataScopeFromContext`（:598-618）：从 context 读 `user_id`/`data_scope` 后委托 `ApplyDataScope`。

**A3. 零调用口径修正（HIGH，与立项基线有出入但方向一致）**：`ApplyDataScope` / `ApplyDataScopeFromContext` 在全仓确无调用点（grep 仅定义处）。**但 VDI VM 域存在一条独立的、已上线的 data_scope 消费链**：
- `internal/api/v1/vdi/vm_handler.go:87-101`：handler 从 gin context 提取 `user_id`+`data_scope`，作为**函数参数**传给 `vmService.ListVMs`；
- `internal/services/vdi/vm_service_impl.go:589,651-654`：`ListVMs` 内调 `ApplyVMDataScopeFilter` + `ApplyBoundUserFilter`；
- `internal/services/vdi/vm_data_scope_filter.go:17-100`：独立实现全部 5 档过滤（`Self` 正确用 `bound_user_id`；非 All 隐藏未绑定 VM）；
- **现成 sqlite 测试样板**：`internal/services/vdi/vm_service_test.go:498-543`（`TestApplyVMDataScopeFilter`，逐档 count 断言）。

⇒ PMODEL-01 不是「从零发明」，而是**把 VDI 已验证的模式复制到 system 用户列表**，`ApplyDataScope` 通用函数可以继续不用（或仅修 bug 留档）。

**A4. 用户列表查询现状（HIGH）** — `internal/api/v1/system/user_handler.go:90-110` → `internal/services/system/user_service.go:388-477`：
- GORM 链式查询（非 raw）；`sys_user` 部门字段名 = **`dept_id`**（:411 `dept_id = ?`；JOIN `LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id`，:462；`sys_user.dept_id` 与 `sys_dept.id` 均为 PostgreSQL 原生 uuid，:457 注释）。
- `Count` 在 :449（**过滤条件必须加在 Count 之前**，否则 total 仍全量）；分页 `Find` 在 :475。
- 现有 `RecursiveDeptID` 已示范「ancestors 单查询替代递归」的写法（:427-435，`sys_dept.ancestors` 5 条件模式）。

**A5. ⚠️ 列表缓存陷阱（HIGH）** — `internal/services/system/user_cache_impl.go:157-191`：带缓存版 `List` 用 `buildListCacheKey(params)` 做 **10 分钟**缓存键，key **不含用户身份/data_scope**。若 scope 过滤做进 service 而 cache key 不加维度，**A 用户过滤后的结果会被 B 用户命中 → 跨用户数据泄漏**。这是本需求最大的隐性影响面，cache key 必须扩展 scope 维度（或带 scope 的请求绕过缓存）。

**A6. 挂载点盘点（HIGH）** — `DataScopePermission` 共 4 处（STATE 基线写 5 处，实跑 router.go 为 4：:187 usersRead / :212 usersWrite / :272 depts / :988 vms）：
- `usersRead`（router.go:178-195）：ANY-of `system:user:list|view` + queryExtra `ops:workstation:list|ops:building:spaces:list`（工位页面用户选择器复用此列表）。**PMODEL-01 主战场**。
- `usersWrite`（:210-221）：逐动作严格权限子组（#16 定稿）。DataScope 对写操作无消费点（handler 均按 id 操作），接不接无行为差异。
- `depts`（:260-276）：注释已写明决策先例 ——「部门树 GetTree 不应用数据权限，返回组织结构全树（低敏感）」+ 运维选择器共用（`OpsSelectorReadPerms`）。
- `vms`（:988）：VDI 已消费（A3）。

**A7. admin/全量零变化论证（HIGH）**：admin 走 `isSuperAdmin` 早退（A2），context 无 `data_scope` → handler 侧「取不到就不过滤」即零变化；`data_scope=1`（All）在过滤函数里原样返回 db → 零变化。v1.24 #16 验收角色（无 F 权限）受影响的是**行集**不是 403 判定（scope 过滤发生在权限中间件之后的 handler/service 层，不触碰 `RequireStrictPermissions`/`RequirePermissionsWithQuery` 语义），但**该角色（若 data_scope≠1）看到的列表行数会变少** —— 这正是需求要的行为，验收脚本若断言过行数需更新（MEDIUM，需对照 v1.24 #79 验收记录确认有无行数断言）。

### B. menu.status 过滤现状（PMODEL-02）

**B1. 两个同名实现，必须分清（HIGH）**：
- `internal/services/system/menu_service.go:509-569` `GetUserPermissions`：**前端权限集**（`POST /system/my-menus/permissions`，menu_handler.go:335）。:547-550 过滤条件原文 = `id IN ? AND perms IS NOT NULL AND perms != ''`，**无 status 条件** → 病灶。
- `pkg/permission/service.go:302-316` `GetUserPermissions`：**后端 RBAC 判定/apikey 用**（permission.go:283、internal/middleware/apikey.go:227）。raw SQL **已带 `AND m.status = MenuStatusNormal`**（:312）→ 后端判定半边已修，勿动。

**B2. 菜单树已过滤（HIGH）**：`GetUserMenus`（menu_service.go:434-448 两处查询均 `status = MenuStatusNormal AND visible = VisibleShow`）；`GetAllUserMenus`（:497 只过滤 status、含隐藏）。⇒ 确认「菜单树消失但按钮权限还在」的不一致**真实存在**，本 phase 修权限集半边（与菜单树对齐加 `status`，不加 `visible`——与后端判定路径「只查 status 不查 visible」的既定语义一致，permission.go:132-134 / service.go:299-301 注释）。

**B3. 缓存与失效（HIGH）**：前端权限集经 `menuCacheService.GetUserPermissions` 缓存（menu_cache_impl.go:115-120），TTL 复用 `CacheConfigMenuTree` **30 分钟**，键按 userID 隔离。菜单任意写操作（含 UpdateStatus）**已调用 `InvalidateMenuCache`**，失效 pattern 含 `CacheKeyMenuUserPermissions + ":*"`（:126-135）。⇒ 停用菜单后权限集**主动失效**（近乎立即，兜底 30min）；v1.25 Future 的「`InvalidateMiddlewarePermCache` 未接线」是**另一个缓存**（中间件 30s 判定缓存，permission.go:67-75），本 phase **不需要也不应该**顺手接线 —— PMODEL-02 的改动落在已接好失效的缓存域内，天然无新增陈旧面。

### C. Step 2 语义现状（PMODEL-03）

**C1. 实现原文（HIGH）** — `pkg/middleware/permission.go:169-208`（`checkUserPermissionUncached` Step 2）：
1. 输入：待判权限串 `permission`（Step 1 精确匹配失败后）；
2. 查 `sys_menu` 中 `perms = permission AND status = normal` 的那条菜单（:177）；
3. 查它的**子菜单** `parent_id = menu.id AND menu_type = 'F' AND status = normal`（:181-182），收集非空 perms；
4. 查用户角色是否绑定**任一**子按钮的 perms（:193-199），有 → 放行父页面权限。

等价表述：**「用户持有某页面（C 型/M 型）下任一 F 型按钮权限 → 自动获得该页面的访问权限（list/view 等）」**；方向是 F→父，不是「持父页面→获得全部子按钮」。P1 防提权约束（:173-175）：仅 F 型子→父放行，不允许 C 型子→父。Step 3 模块级降级（`system:user:list → system:user`，:210-228）是另一层，勿与 Step 2 混写。

**C2. 现有命名与注释（HIGH）**：Step 2 逻辑**内联在 `checkUserPermissionUncached` 里**，无独立函数名；注释「尝试通过子菜单权限检查父菜单权限 / 勾选子菜单时自动拥有父菜单的页面访问权限」（:169-171）方向正确但不够权威。`RequireStrictPermissions` doc（:374-386）写「精确匹配 → F 型子权限继承（Step 2）→ 模块级降级（Step 3）」——与实现一致，**v1.24 修过之后无矛盾**，只需随改名同步引用。

### D. 迁移机制与 GORM 漂移现状（PMODEL-04）

**D1. 迁移机制（HIGH）**：
- 编号 Go 迁移 `internal/core/db/migrations/migration_NNN_*.go`，**最新到 224**（migration_224_dept_sync_seed_and_ou_group_unique.go）。
- 启动路径：`internal/core/db/database.go` `AutoMigrate()`（:805）→ 先 GORM `Migrator().AutoMigrate(migrateList...)`（:985）→ 然后**硬编码调用块**逐个执行 `migrations.MigrateNNN(d.DB)`（PG 分支 :1016-1140，sqlite 分支 :1145-1214），**每次启动无条件重跑**，各迁移自幂等（IF NOT EXISTS / count-then-insert / WARN 非阻断，见 migration_219/220/223 注释）。
- 历史 033-190 已移入 `migrations/archive/applied/`（README 记载 replay 方式 = 手动调 `MigrateNNN`）。
- **没有任何已应用版本记录**：全仓无 `gorm_migrations`、无 `sys_schema_migrations`（grep 零命中）；「已应用」的唯一凭证是启动日志。`SKIP_AUTOMIGRATE=true` 旁路（core.go:276-296，Supabase pooler 场景）走 `BootstrapMissingTables`（database.go:1345），连调用块都只跑 215-220 子集（:1402-1463）→ 对账缺口更大。

**D2. GORM 漂移实测（HIGH，模型侧；库表侧 LOW 需实机）**：
- `internal/models/role.go:24-27` `RoleMenu`：仅 `RoleID`+`MenuID`，**无 ID 主键字段、无 CreatedAt**（生产库 `sys_role_menu` 若有 `id`/`created_at` 列则模型未映射，靠 DB default —— 库表实际列集需实机 `\d sys_role_menu` 确认）。
- `internal/models/user.go:51-55` `UserRole`：`UserID`+`RoleID`+`CreatedAt time.Time`（**有 CreatedAt 但无 gorm tag**，AutoMigrate 会建列），**无 ID 字段**。
- AutoMigrate 与漂移的关系（HIGH，机制推断）：GORM AutoMigrate 按**模型**补列 —— 模型没声明的列它**不会建也不会删**；若表由模型建出则表里根本没有 `id` 列；若表由 SQL 迁移手建了 `id` 而模型未映射，AutoMigrate 不会动它（无 model→column 对照）。⇒ 漂移是「静默并存」，不会启动报错，但 `SELECT *`→struct 扫描、审计导出、未来加字段都可能踩坑。

## 实现方案建议

### PMODEL-01：用户列表接入 data_scope（推荐落点）

**方案：仿 VDI 模式，handler 提取 → params 携带 → service 过滤（不接 `ApplyDataScopeFromContext`）**

1. `internal/models/system/requests/user_requests.go`：`UserListParams` 增加非请求字段（不落 JSON tag 或 `json:"-"`）：
   ```go
   // DataScopeUser 数据范围过滤（由 handler 从 gin context 注入，非客户端输入）
   DataScopeUserID string        `json:"-"`
   DataScope       models.DataScope `json:"-"` // 0 = 不过滤（未挂 DataScopePermission / 超管）
   ```
2. `user_handler.go List`（:90-110）：在调 service 前仿 `vm_handler.go:87-99` 提取 `c.Get("user_id")` / `c.Get("data_scope")` 填入 params。**超管/中间件未挂时两值均为零值 → service 零变化**。
3. `user_service.go List`：在 **:447（Count 之前）** 插入过滤分支（复用 `RecursiveDeptID` 的 ancestors 单查询风格，替代递归 `getChildDepts`）：
   - `DataScopeAll` → 不加条件；
   - `Custom` → `dept_id IN (SELECT rd.dept_id FROM sys_role_dept rd JOIN sys_user_role ur ON ur.role_id=rd.role_id WHERE ur.user_id=?)`（子查询内联，避免两次往返）；
   - `Dept` → `dept_id = (SELECT dept_id FROM sys_user WHERE id=?)`；
   - `DeptChild` → `dept_id IN (SELECT id FROM sys_dept WHERE id=? OR ancestors=? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors LIKE ?)`（与 user_service.go:427-435 同款 5 条件；**比递归 getChildDepts 少 N 次往返**，见性能节）；
   - `Self` → **`sys_user.id = ?`（= userID）**，不要照抄 `ApplyDataScope:575` 的 `dept_id = userID` bug；
   - default/0 → 不过滤（context 无 scope 的旧路径）+ 对未知枚举值 fail-closed `1=0`。
4. `user_cache_impl.go`：`buildListCacheKey`（:172-191）**必须**追加 `:scope:<n>:uid:<userID 或 all>` 段（scope==1 时 uid 可省）；或更简单——`DataScope≠0 且 ≠1` 时直接绕过缓存走 DB（列表缓存本就是 10min 优化，非 scope 请求命中缓存即可）。二选一，**不作为 = 泄漏**。
5. `getUserMaxDataScope`/`ApplyDataScope`：只修注释 +（可选）给 `ApplyDataScope:575` 的 Self 分支标注「仅适用于用户维度字段」，或按 deptField 语义拆开；**不做大重构**。

**影响面圈定（推荐清单）**：
| 挂载点 | 本 phase 是否接消费 | 理由 |
|---|---|---|
| usersRead `/system/users/*`（list/statistics/get） | **接 list（必须）**；`GetByID` 可选 | 需求验收点；GetByID 按主键泄露单条的风险可登记不扩面 |
| usersRead `/statistics` | **登记不接**（或同批接，改动同点） | 聚合数字与 list 行集不一致会被前端卡片放大；建议同 batch 用同一 query builder 接上，成本≈0 |
| usersWrite | 不接 | handler 均按 id 操作，无列表语义 |
| depts | **不接** | router.go:262-264 已有显式决策注释「组织全树低敏感」+ 运维选择器共用；接了会破坏 DeptTree 全局选择器 |
| vms | 不接（已消费） | VDI 自有实现 |
| 其它列表（roles/posts/…） | **不做**（Phase 85 边界） | 「合理范围内的其它列表」= 零共享架构成本的 users 域即可；扩到全站列表是独立工程 |

**性能（A4 问题）**：
- `getChildDepts` 递归是逐层 N 次查询（每请求），热点风险真实存在但部门树规模小（<几百节点）时量级可接受（MEDIUM）。**推荐直接不用它**：DeptChild 走 ancestors 模式单查询（代码库已有 4 处先例 + user_service.go:427 5 条件修正版），零递归、零缓存需求。
- 若坚持复用 `ApplyDataScope`（不推荐，含 Self bug），再考虑 `core.Cache`（`ruoyi:` 前缀）缓存子树 ID 列表 + 部门写操作失效 —— 本 phase 不必做。
- `getUserMaxDataScope` 每请求一条 SQL，量级与既有 `isSuperAdmin` 查询同级；中间件层已有进程内 30s 缓存先例（permission.go:24-33），如需可仿 `mwPermCacheEntry` 加 scope 缓存，非必须（MEDIUM）。

**测试策略（A5 问题）**：
- 样板：`internal/services/vdi/vm_service_test.go:498-543`（sqlite 内存库 + AutoMigrate 相关 model + 逐档 count 断言）。
- 夹具构造：**不要走 `roleService.CreateRole`**（sqlite 下 `ON CONFLICT` 方言 500 前科，role_service.go:425-430 `isPostgreSQL` 守卫），直接 `db.Create(&models.Role{})` / `db.Create(&models.UserRole{})` / `db.Create(&models.RoleMenu{})` 摆行；部门建 3 层（A→B→C）验证 ancestors 5 条件对深层子树正确。
- 红绿设计：先写 `TestUserServiceList_DataScope`（scope=3 夹具断言只含本部门、`total` 同步变小；scope=4 含子部门；scope=5 仅本人；scope=1 全量；DataScope 零值 = 全量）→ 实现前跑必红（当前无过滤）→ 实现后绿；再写「回退即失败」守卫：把过滤分支注释掉测试必红（验收时 review diff 即可，不需运行时开关）。
- 缓存泄漏用例：同 params、两个不同 scope 用户先后调带缓存版 List，断言第二次结果 ≠ 第一次（若选择「加 cache key 维度」方案）或两次都走 DB（绕过方案）。
- 回归闸门：`go build ./...` + `go test ./...` 全绿；v1.24 #16 相关测试（严格权限语义）零改动。

### PMODEL-02：menu.status 过滤

- 唯一改动点：`internal/services/system/menu_service.go:547-550` 的 Where 追加 `AND status = ?`（`models.MenuStatusNormal`）。一行级 diff。
- **不修** `pkg/permission/service.go`（后端判定已过滤）；**不接** `InvalidateMiddlewarePermCache`（30min 前端缓存已有 `InvalidateMenuCache` 主动失效，见 B3；中间件缓存接线属 v1.25 Future 条目）。
- 测试：现有 `menu_service_test.go:494` `TestMenuService_GetUserPermissions` 旁新增用例——建角色→绑菜单 A（正常）+ 菜单 B（status=1，两者 perms 非空）→ 断言结果含 A 的 perms 不含 B 的；再加红绿（实现前 B 的 perms 必然出现）。
- 连带核对（MEDIUM）：`menu_cache_impl.go:115` 缓存包装无需改（查询在回源闭包内）。

### PMODEL-03：Step 2 语义固化

- **推荐：改名 + 权威注释块，双管齐下**（改动全在 `pkg/middleware/permission.go` 私有面，零外部影响）：
  1. 把 ：169-208 内联块抽成私有函数 `hasInheritedButtonPerm(core, userID, permission) bool`，函数 doc 用 C1 的四步精确描述 + 「方向：持 F 型按钮 → 继承父页面权限；反向（父→子）不成立」加粗；调用点仅 `checkUserPermissionUncached` 一处（私有，无波及）。
  2. `RequireStrictPermissions` doc（:374-386）把「F 型子权限继承（Step 2）」改为引用新函数名，保持单一事实源。
- **不推荐只加注释**：v1.24 曾因语义写反得出错误中间态，函数名是最强的防复刻错读手段；也不推荐把 Step 3 也改名（超范围）。
- 测试：现有严格权限测试已锁定「只做 Step1」；可加一条 `TestStep2_InheritDirection`（持父页面 perms 不自动获得子按钮 perms——锁反向不成立，防未来有人「补全」成双向）。

### PMODEL-04：迁移记录表 + 漂移登记

**机制现状回顾**：迁移是「幂等重跑」不是「应用一次」（D1），所以传统 `schema_migrations` 的 skip 语义不适用。推荐**记录+对账**而非**控制执行**：

- **推荐方案：自建 `sys_schema_migrations` 轻量记录表 + 启动 upsert + 只读对账**
  1. 新 migration（从 **225** 起编）建表 `sys_schema_migrations(version int PK, name text, applied_at timestamptz default now())`，PG/sqlite 双方言（IF NOT EXISTS）。
  2. `database.go` 两个调用块（PG :1016-1140 / sqlite :1145-1214）每个 `MigrateNNN` 成功后 upsert 记录（或包一层 `runAndRecord(version, name, fn)` helper，一次改造 60 余处调用点可用 sed 式机械替换）；**记录失败只 WARN 不阻断启动**（对齐 migration_223「非阻断档」先例）。历史 033-224 版本号在表创建时批量 seed `applied_at = now()`（无法考证真实时间的显式妥协，注释说明）。
  3. 对账 = 小只读端点（如挂在 monitor/cache 监控域）或 Go 脚本：仓库 registry（新建 `migrations/registry.go` 列出全部当前活跃版本号）vs 库内记录，输出 missing/extra。零新依赖。
- **不推荐**：gorm `Migrator()` 版本记录（gorm 本身无内置版本表，造出来还是自建表）；只写脚本不落表（每次对账要连生产库跑脚本，且 SKIP_AUTOMIGRATE 旁路无痕迹）。
- **GORM 漂移登记**（不改行为，只登记）：
  - 登记处：`.planning/notes/` 新建 note（如 `260628-gorm-schema-drift-register.md`）+ STATE.md Todos 一行指引（STATE 是 hot 文件，note 是详情）；不建议塞 STATE 正文（会随归档丢失）。
  - 条目格式建议：`表名 | 模型 (文件:行) | 库表多出的列 | 现状（靠 DB default）| 风险 | 建议动作（补映射 or 显式声明不动）`。首两条：`sys_role_menu`（models/role.go:24-27，缺 id/created_at 映射）/ `sys_user_role`（models/user.go:51-55，缺 id 映射、CreatedAt 无 tag）。
  - 修复选项（**本 phase 不做**，登记即可）：给模型补 `ID string gorm:"type:uuid;default:gen_random_uuid()"` + `CreatedAt`，让 AutoMigrate 收敛列集——但改动会触发既有库 ALTER/回填评估，且 many2many 联表行为需回归，超出「登记」范围。

## Validation Architecture

| # | 验证维度 | 验证方法 | 自动化命令 / 产物 |
|---|---|---|---|
| V1 | data_scope 过滤正确性 | sqlite 5 档夹具逐档断言行集+total（含 3 层部门树、Custom 角色部门映射、Self=仅本人=按 user.id） | `go test -run TestUserServiceList_DataScope ./internal/services/system/` |
| V2 | data_scope 零回归 | scope=1 / 超管 / 未挂中间件（零值）三路径输出与旧实现逐字节一致 | 同上文件内 `TestDataScope_NoChangeForAllAndAdmin` |
| V3 | 列表缓存隔离 | 同 params 异 scope 用户不串缓存（或绕过路径走 DB） | `go test -run TestDataScopeCache ./internal/services/system/` |
| V4 | menu.status 过滤 | 停用菜单 perms 不入权限集、启用后回归、visible=隐藏仍保留 | `go test -run TestGetUserPermissions ./internal/services/system/` |
| V5 | Step 2 语义方向锁定 | 持 F 型子按钮→父页面 PASS；持父页面→子按钮 FAIL（反向不成立）；Step3 模块降级行为不变 | `go test -run TestStep2 ./pkg/middleware/` + 既有严格权限套件 |
| V6 | 迁移记录表 | 启动后 `sys_schema_migrations` 覆盖全部活跃版本；对账工具 missing=0；二次启动幂等（记录数不变） | sqlite 单测 + `go test ./internal/core/db/...`；人工：PG 实机看表 |
| V7 | 漂移登记存在性 | note 文件含两张表条目 + STATE Todos 指引 | 人工 review |
| V8 | 全局回归 | build+test 全绿；#16 严格权限测试零改动 | `go build ./... && go test ./...` |

**人工/实机验证项（LOW→HIGH 闭环）**：
1. PG 生产/实机：`POST /system/users/list` 用 data_scope=3 夹具角色复测 `total` 不再 4231（v1.24 #79 同款对照法：改 scope 前后对比）；同时跑一次 data_scope=1 角色确认行集零变化。
2. 实机 `SELECT column_name FROM information_schema.columns WHERE table_name IN ('sys_role_menu','sys_user_role')` 把漂移登记表从 LOW 升 HIGH。
3. 实机停用一个含 F 子按钮的菜单 → 前端按钮消失（30min 缓存已被 InvalidateMenuCache 主动失效，应近即时）。
4. SKIP_AUTOMIGRATE=true 启动一次，确认记录表 upsert 在旁路路径的行为符合设计（旁路只跑 215-220 子集，记录应只含这些版本）。

## 波次建议

- **Wave 1（并行三线，零文件交叉）**
  - W1a：PMODEL-03 改名+注释（pkg/middleware/permission.go）——纯文档化，最先落最安全；
  - W1b：PMODEL-02 一行过滤 + 测试（internal/services/system/menu_service.go + test）；
  - W1c：PMODEL-04 迁移记录表 migration_225 + registry + upsert helper（internal/core/db/**）+ 漂移登记 note。
  - 闸门：各自包测试绿 + `go build ./...`。
- **Wave 2（串行，依赖 W1b 确立的测试基线与 W1a 的语义锚点）**
  - W2：PMODEL-01 全链（requests → handler → service → cache key）+ 全套 scope 测试。放最后因为它是唯一改**行为**的项，且 test 文件与 W1b 同包可能轻冲突。
  - 闸门：V1-V3 绿；#16 既有测试零改动确认。
- **Wave 3（收口）**：`go build ./... && go test ./...` 全绿 + V8 人工核对 + 实机项登记（不阻塞 phase 关闭，登记待办）。

## 边界与不做什么

- **不扩面到 RBAC-01/03/04（Phase 85）**：不动 `pkg/permission/config.go` 死表、不拆任何 ANY-of 组、不做 roles/menus/depts/posts 严格化；users 域的写组挂载结构（router.go:210-221）本 phase **零改动**。
- 不接 `InvalidateMiddlewarePermCache`（v1.25 Future：权限缓存陈旧，单独条目）。
- 不改 depts 组数据行为（router.go:262 决策先例）；不新增其它模块列表的 scope 消费。
- 不修 `ApplyDataScope:575` Self bug 之外的大重构；不引入 scope 缓存（除非性能实测需要）。
- 不给 `sys_role_menu`/`sys_user_role` 补模型列（只登记；修复需独立评估 ALTER 影响）。
- 不做迁移**执行控制**（记录表不参与 skip 判断，迁移保持幂等重跑）。
- 不动前端（本 phase 纯后端）；前端按钮消失是 B 的自然结果，三态门控铺开在 Phase 86。
- 零新依赖；不 git commit。

## Sources

- `pkg/middleware/permission.go` :17-75（30s 缓存/InvalidateMiddlewarePermCache 未接线）、:109-232（checkUserPermissionUncached Step1/2/3）、:234-254（extractModulePermission）、:256-278（isSuperAdmin）、:374-399（RequireStrictPermissions doc）、:440-467、:469-499（DataScopePermission）、:502-513（getUserMaxDataScope）、:516-579（ApplyDataScope，:575 Self bug）、:582-593（getChildDepts）、:595-618（ApplyDataScopeFromContext）
- `internal/models/base.go:70-77`（DataScope 枚举）
- `internal/api/router.go:175-284,988`（4 处 DataScopePermission 挂载 + 决策注释）
- `internal/api/v1/system/user_handler.go:81-126`；`user_router.go:32-102`
- `internal/services/system/user_service.go:388-477`（List 查询构造）、:427-435（ancestors 5 条件）、:547（Statistics）；`user_cache_impl.go:157-191`（10min 缓存与 cache key）
- `internal/models/system/requests/user_requests.go:12-26`
- `internal/services/system/menu_service.go:434-455`（菜单树过滤）、:497（GetAllUserMenus）、:509-569（GetUserPermissions 病灶 :547-550）；`menu_cache_impl.go:90-136`（30min 缓存 + InvalidateMenuCache 覆盖 UpdateStatus）
- `pkg/permission/service.go:298-316`（后端判定 GetUserPermissions 已过滤 status）
- `internal/api/v1/system/menu_handler.go:335`（/system/my-menus/permissions）
- VDI 参考：`internal/api/v1/vdi/vm_handler.go:87-107`；`internal/services/vdi/vm_data_scope_filter.go:17-100`；`vm_service_impl.go:589,651-654`；`vm_service_test.go:272-290,498-543`
- 迁移：`internal/core/db/database.go:805-985,1016-1214,1345,1402-1463`；`internal/core/db/migrations/`（最新 migration_224；archive/applied/README）；`internal/core/core.go:276-296`（SKIP_AUTOMIGRATE）
- 模型漂移：`internal/models/role.go:24-27`；`internal/models/user.go:51-55`
- 规划输入：`.planning/STATE.md`、`.planning/ROADMAP.md:172-181`、`.planning/REQUIREMENTS.md:52-57`、`.planning/milestones/v1.24-REQUIREMENTS.md:103-110`
