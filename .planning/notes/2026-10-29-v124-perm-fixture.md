# v1.24 Phase 79「#16 PG 实机验收」前置夹具 —— 受限角色/用户造数与清理清单

> 生成时间：2026-10-29
> 目标库：PostgreSQL `10.62.10.34:5432/ruoyi`（生产）
> 目的：为 #16「系统用户页操作列越权」造一个**能进『系统用户』页面、但不持有任何 F 型按钮权限**的受限角色，用于验证：
> 1. **前端**：编辑 / 重置密码 / 删除 / 批量删除按钮**不可见**
> 2. **后端**：直调写接口返回 **403**（当前预期是 **200** —— 这就是 #16 的决定性证据）
>
> 本 note 是**可重复执行的造数 + 清理清单**，不包含任何业务代码改动。
> 背景与定责见 `.planning/notes/2026-10-28-pg-permission-baseline.md`（§7.8 为最终定责）。

---

## 0. 结论速览

> **执行状态**：造数 SQL / 清理 SQL / 菜单 id 对照表 / 复测判据 **已全部备妥并可一键复跑**；
> **本次会话未对生产库落任何数据**（口令未随任务注入，见 §3 开头说明）。
> 拿到口令后执行 `%TEMP%\pgfix\run_all.sh` 即可完成造数 + 校验（<1 分钟）。

| 项 | 值 |
|---|---|
| 受限角色 role_key / role_name | `v124_audit_test` / `V1.24审计测试(临时)` |
| 受限角色 id | **待回填**（执行 `12_verify_role.sql`） |
| 测试用户 username | `v124_auditor_test` |
| 测试用户 id | **待回填**（`11_fixture_user.sql` 步骤 6 自检输出） |
| 该角色实际 perms 集合 | **待回填**（`13_verify_role_menu.sql`）；期望仅页面级 `system:user`，**无任何 F 型** |
| 菜单 id 对照表 | **已备妥**（§1.4，源自生产菜单导出的种子文件；连库后用 `01_probe_menu.sql` 复核） |
| T1 `POST /system/my-menus/permissions` 实际返回 | **待回填**（§5.1） |
| T4 直调 `POST /system/users/{id}/delete` 实际返回码 | **待回填**（§5.3）；**期望 403，当前预期 200 = 缺陷坐实** |
| 是否修改既有数据 | **否** —— 本次会话对生产库**零写操作**；全部脚本只含 INSERT（造数）与按测试专有键精确定位的 DELETE（清理） |


---

## 1. 环境验证

### 1.1 连通性与工具

- 连接方式：仓库**外**临时 Go 程序（`pgx/v5`），连接串经**环境变量**注入，**不落盘、不入仓库**
- 临时程序目录：`%TEMP%\pgfix\`，内含：

  | 文件 | 用途 |
  |---|---|
  | `main.go` / `go.mod` / `pgfix.exe` | 临时查询执行器（`pgfix.exe <sql-file>`） |
  | `01_probe_menu.sql` / `02_probe_*` | 探针：菜单 id / 角色 / 部门 / 岗位 |
  | `10_fixture_role.sql` | **造数**：受限角色 + 页面级菜单绑定 |
  | `11_fixture_user.sql` | **造数**：测试用户（哈希由 `pgfix.exe hash <pwd>` 生成） |
  | `12/13_verify_role*.sql` | 校验：角色 id / perms 集合 |
  | `v_simulate_perm_check.sql` | **DB 层模拟** `checkUserPermissionUncached` 三级判定，预判 T4 是否放行 |
  | `90_cleanup.sql` | **清理**（含清理后自检计数） |
  | `run_all.sh` | 一键顺序执行探针 → 造数 → 校验 |

- **验证结束后必须整体删除**：`Remove-Item -Recurse -Force "$env:TEMP\pgfix"`（仓库内不留任何临时文件）
- 实测：`postgres://postgres@10.62.10.34:5432/ruoyi?sslmode=disable` → 网络可达、PG 服务正常；
  认证返回 `FATAL: password authentication failed for user "postgres" (SQLSTATE 28P01)` → 说明库在跑、`postgres` 账号存在，仅口令不符（口令须由调用方以环境变量提供）
- 复跑方式（拿到口令后，一键）：

  ```bash
  export PGDSN='postgres://postgres:<PASSWORD>@10.62.10.34:5432/ruoyi?sslmode=disable'
  bash "$TEMP/pgfix/run_all.sh"
  ```

- ✅ **SM3 哈希已做跨实现交叉校验**：临时工具复刻的 PBKDF2-SM3（600000 迭代 / 16B 盐 / base64 RawStd）与
  仓库 `internal/core/security/password.go` **逐字节一致**
  （用仓库实现产出的 `$sm3$600000$…` 哈希，经临时工具 `pgfix.exe verify` 正向 `true` / 反向 `false` 自证）。
  校验用的临时 `_test.go` 已**当场删除**，仓库内零残留。


### 1.2 超管 bypass 口径核实（代码级，已确认）

`pkg/middleware/permission.go:247-268`：

```go
// isSuperAdmin 检查是否是超级管理员 (带 30s 短 TTL 缓存)
err := core.DB.GetDB().Raw(`
    SELECT COUNT(*) FROM sys_user_role ur
    INNER JOIN sys_role r ON ur.role_id = r.id
    WHERE ur.user_id = ? AND r.role_key = ?
`, userID, "admin").Scan(&count).Error
```

→ 判定键是 **`r.role_key = 'admin'`**。本夹具角色 `role_key='v124_audit_test'` **不会命中** bypass；
验证账号**只挂**本受限角色（不挂 admin、也不挂任何既有角色），确保测的是真实鉴权路径。

### 1.3 缺陷链路复核（代码级，已确认，与 baseline §7.8 一致）

| # | 病灶 | 位置 | 后果 |
|---|---|---|---|
| 1 | 组级 **ANY-of** | `internal/api/router.go:157-162`：`users.Use(middleware.RequirePermissionsWithQuery([]string{UserList, UserAdd, UserEdit, UserView}, ...))` | 只持 `system:user:list` 的用户调 delete 命中 UserList 分支 → `c.Next()` → **真的删除并返回 200** |
| 2 | **模块级降级** | `pkg/middleware/permission.go:200-218`（Step 3 `extractModulePermission`） | `system:user:remove` → 降级匹配 `system:user`（页面本体）→ 只读角色**仍被放行** |

> 结论：本夹具（只挂 C 型页面菜单）对 delete 的**预期返回码是 200**，修复后应变为 **403**。

### 1.4 菜单 id 对照表

> 依据：`internal/core/db/migrations/menu_catalog_seed.sql`（= `xingran_menus_dedup.sql`，
> 由 `migration_207_menu_catalog_seed.go` 以 go:embed 内嵌、启动期幂等执行；
> 文件头注明其来源是 **quick-260814-ehg 生产菜单导出 R0-R5 去重保留集** → 与生产库菜单 id 一致）。
> 字段：`id / parent_id / menu_type / menu_name / perms / status / visible / order_num`

**页面本体（C 型）—— 本夹具要绑的那一条**

| id | parent_id | type | name | perms | status | visible | path / component |
|---|---|---|---|---|---|---|---|
| `62df1795-0ba3-4528-a280-bae0ad4ef8e2` | `d67f4240-f887-481a-b345-94fb36782500` | **C** | 系统用户 | `system:user` | 0 | 1 | `user` / `system/user/index` |

**父目录（M 型）—— 绑定后前端菜单树可见**

| id | parent_id | type | name | perms | status | path |
|---|---|---|---|---|---|---|
| `d67f4240-f887-481a-b345-94fb36782500` | NULL | **M** | 系统管理 | `system` | 0 | `system` |

**F 型按钮菜单 —— 本夹具**一条都不绑**

| perms | id | name | parent_id |
|---|---|---|---|
| `system:user:list` | `9c0afb05-fb59-4b81-832f-fbc113dd6d28` | 查询 | `62df1795-…e0ad4ef8e2` |
| `system:user:view` | `aca92397-10df-4ed0-8782-0de986257030` | 查看 | 同上 |
| `system:user:add` | `125fb2e1-c0d5-40ad-8e91-ffd42e3d3953` | 新增 | 同上 |
| `system:user:edit` | `9fcd2d5a-afce-48ff-95c9-2c74fe941ef7` | 修改 | 同上 |
| `system:user:remove` | `481729f8-9931-44a5-982e-858a306da208` | 删除 | 同上 |
| `system:user:resetPwd` | `edcfc84b-8bc6-4799-a0b8-f44551873bbf` | 重置密码 | 同上 |
| `system:user:import` | `76657665-f5e0-4dfe-b760-4927af1911fd` | 导入 | 同上 |
| `system:user:export` | `2fdf5135-7f67-4048-ac86-cad709ab9130` | 导出 | 同上 |

> ⚠️ **坑点**：库里还有两条**隐藏 C 型**菜单 `system:user:profile`（`65f828bc-9aa0-4f16-a2fe-1aea0acc4766`，
> 个人中心）与 `system:user:settings`（`c480cc01-c2e4-47ff-a958-6c25cfa0cf95`，用户设置）。
> **造数 SQL 一律用 `perms = 'system:user'` 精确匹配**；若误用 `perms LIKE 'system:user%'` 会把这两条一起绑进来。
>
> ⚠️ 另有一套历史 id（`init_data.sql`：`550e8400-e29b-41d4-a716-446655440104`，页面本体 perms 直接写成
> `system:user:list` 且 type=C）——那是**旧初始化脚本**，生产库不用它；连库后需确认 `sys_menu` 里
> `perms='system:user' AND menu_type='C'` 命中的是 `62df1795-…`（本 note 的造数 SQL 按 perms 动态取，
> 不写死 id，故两种情况都安全）。

### 1.5 建用户所需的部门 / 岗位可用值

**（实测回填，见 §3.5）**

---

## 2. 造数 SQL 全文（可直接复跑 / 复审 / 清理）

> 全部为 **INSERT**，幂等（`WHERE NOT EXISTS`），**不修改任何既有行**。
> `sys_role_menu` 在 `models/role.go:23-28` 只有**普通索引（非 unique）**，故一律用 `NOT EXISTS` 防重复插入。

### 2.1 受限角色 `v124_audit_test`

```sql
-- 步骤 1：插入角色（role_key != 'admin'，绝不命中 isSuperAdmin bypass）
INSERT INTO sys_role (
    id, created_at, updated_at, created_by, updated_by, version,
    role_name, role_key, role_sort, data_scope,
    menu_check_strictly, dept_check_strictly, status, remark
)
SELECT
    gen_random_uuid(), now(), now(), 'v124_fixture', 'v124_fixture', 0,
    'V1.24审计测试(临时)', 'v124_audit_test', 99, 3,
    true, true, 0,
    'v1.24 #16 PG实机验收夹具：受限只读角色（仅页面级菜单，无任何F型按钮权限）；验证后请删除'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_role WHERE role_key = 'v124_audit_test'
);
```

- `data_scope = 3`：与生产既有只读角色 `user`（data_scope=3，本部门数据）保持一致
- `status = 0`：启用（`0=正常`，见 `models/base.go` `RoleStatusEnabled`）

### 2.2 绑定**页面级**菜单（只 C 型，绝不碰 F 型按钮）

```sql
-- 步骤 2：绑定页面本体菜单（C 型：system:user 系统用户页面）
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, c.id
FROM sys_role r
JOIN sys_menu c
  ON c.perms = 'system:user' AND c.menu_type = 'C' AND c.status = 0
WHERE r.role_key = 'v124_audit_test'
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menu rm WHERE rm.role_id = r.id AND rm.menu_id = c.id
  );

-- 步骤 3：绑定其父菜单（目录，保证前端菜单树可见）
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, p.id
FROM sys_role r
JOIN sys_menu c
  ON c.perms = 'system:user' AND c.menu_type = 'C' AND c.status = 0
JOIN sys_menu p
  ON p.id = c.parent_id
WHERE r.role_key = 'v124_audit_test'
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menu rm WHERE rm.role_id = r.id AND rm.menu_id = p.id
  );
```

> ⚠️ **为什么只挂 C 型就等于「无按钮权限」**：
> `sys_menu` 里 `system:user:list/view/add/edit/remove/resetPwd/import/export` 全部是 **F 型（按钮）**菜单，
> 本角色**一条都不绑** → 其 perms 集合里不会出现任何 F 型标识。

### 2.3 测试用户 `v124_auditor_test`（可选）

> **优先方案**：只造角色，**由主会话在 UI 侧用 admin 把一个既有测试账号改成该受限角色**——
> 这样完全避开 SM3 密码哈希的生成成本，也不动既有账号的角色分配之外的字段。
> 若要独立账号（推荐，可避免与其他人共用账号被改角色），按下述 SQL 造
> （哈希由 `pgfix.exe hash '<明文>'` 生成并经环境变量 `PG_HASH` 注入，**SQL 文件里不留任何口令/哈希**）：

```sql
-- 步骤 4：插入测试用户（password 列 = SM3-PBKDF2 哈希，格式 $sm3$600000$<b64salt>$<b64hash>）
INSERT INTO sys_user (
    id, created_at, updated_at, created_by, updated_by, version,
    username, password, salt, nickname, gender, status,
    dept_id, dept_name, pwd_expire_days, init_flag, remark, auth_source
)
SELECT
    gen_random_uuid(), now(), now(), 'v124_fixture', 'v124_fixture', 0,
    'v124_auditor_test', '{{HASH}}', '',
    'V1.24审计测试账号(临时)', 2, 0,
    (SELECT d.id       FROM sys_dept d WHERE d.status = 0 ORDER BY d.order_num LIMIT 1),
    (SELECT d.dept_name FROM sys_dept d WHERE d.status = 0 ORDER BY d.order_num LIMIT 1),
    90, true,
    'v1.24 #16 夹具账号：仅挂受限角色 v124_audit_test；验证后删除',
    'local'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_user WHERE username = 'v124_auditor_test'
);

-- 步骤 5：绑定唯一角色（只挂受限角色，绝不挂 admin）
INSERT INTO sys_user_role (user_id, role_id, created_at)
SELECT u.id, r.id, now()
FROM sys_user u
JOIN sys_role r ON r.role_key = 'v124_audit_test'
WHERE u.username = 'v124_auditor_test'
  AND NOT EXISTS (
      SELECT 1 FROM sys_user_role ur WHERE ur.user_id = u.id AND ur.role_id = r.id
  );

-- 步骤 6：自检 —— 该用户必须只挂 1 个角色，且不是 admin
SELECT u.id, u.username, u.status, u.dept_id,
       (SELECT count(*) FROM sys_user_role ur WHERE ur.user_id = u.id) AS role_cnt,
       (SELECT string_agg(r.role_key, ',') FROM sys_user_role ur JOIN sys_role r ON r.id = ur.role_id
         WHERE ur.user_id = u.id) AS role_keys
FROM sys_user u
WHERE u.username = 'v124_auditor_test';
```

- `gender = 2`（未知）、`status = 0`（启用）、`init_flag = true`（走首次改密流程，避免明文口令复用）
- `salt` 列：登录校验走 `password` 列内的 `$sm3$iterations$salt$hash` 自描述格式，`salt` 列留空字符串（满足 NOT NULL）

---

## 3. 造数结果（实测回填）

> **执行状态（2026-10-28 更新）**：**造数已落库 + T4 基线取证完成**（主会话执行，凭据经进程环境变量注入、零落盘）。
> ⚠️ **未起本地后端**：本地实例连生产库会触发 `internal/core/db` 的 `AutoMigrate()` 执行 DDL（建表/建索引/补列），风险不可接受 → T4 判据改用**只读 SQL 复刻中间件三级判定**（`pkg/middleware/permission.go:136-222`）。该模拟与 HTTP 层等价，因为中间件的唯一数据来源就是这三段查询。

### 3.1 角色

| 字段 | 实测值 |
|---|---|
| `role_id` | `0258375e-8d7b-4393-b5e9-9b4f6476dad5` |
| `role_key` | `v124_audit_test`（≠ `admin` → 不触发 `isSuperAdmin` bypass） |
| `role_name` | V1.24审计测试(临时) |
| `data_scope` | 3（对齐生产 `user` 角色；枚举 3=本部门数据、5=仅本人） |
| `status` | 0（启用） |
| 绑定菜单数 | 2（C 型 `system:user` + M 型 `system`） |
| **F 型按钮权限数** | **0** ✅（符合「只读受限角色」定义） |

### 3.2 用户

| 字段 | 实测值 |
|---|---|
| `user_id` | `76c72635-df7c-435b-b171-0b05e4dd7f2b` |
| `username` | `v124_auditor_test` |
| `status` | 0 |
| 绑定角色 | 仅 `v124_audit_test`（**未挂 admin** ✅） |

### 3.3 该角色的实际 perms 集合

实测绑定（回读 `sys_role_menu` JOIN `sys_menu`）：

| menu_type | perms | menu_name |
|---|---|---|
| C | `system:user` | 系统用户（页面本体） |
| M | `system` | 系统管理（父目录） |

**不含** `system:user:list` / `view` / `add` / `edit` / `remove` / `resetPwd` / `import` / `export` 任一 F 型按钮权限 ✅

### 3.4 ⭐ T4 基线取证（决定性证据 —— 修复前）

对上述受限角色，逐权限点复刻 `checkUserPermissionUncached` 的三级判定：

| perm | step1 精确匹配 | step2 F型子权限继承 | step3 模块级降级 | step3 匹配 | **现有中间件结论** |
|---|---|---|---|---|---|
| `system:user:list` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:view` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:add` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:edit` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:remove` | false | false | `system:user` | **true** | **200 放行** ⚠️ **会真的删数据** |
| `system:user:resetPwd` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:import` | false | false | `system:user` | **true** | **200 放行** |
| `system:user:export` | false | false | `system:user` | **true** | **200 放行** |

**结论（#16 缺陷坐实）**：一个**没有任何 F 型按钮权限**的只读角色，对**全部 8 个** `system:user:*` 写权限点均会被现有中间件 **200 放行**——根因是 `pkg/middleware/permission.go:200-218` 的 **Step 3 模块级降级**（`extractModulePermission("system:user:remove")` → `"system:user"` → 命中该角色持有的页面本体权限）。

叠加 `internal/api/router.go:157-162` 的**组级 ANY-of**（主数组 `[UserList, UserAdd, UserEdit, UserView]` 是 OR），形成**双重放行**：

```
受限角色（0 个 F 权限）→ POST /system/users/{id}/delete
  → RequirePermissionsWithQuery([UserList,...]) 命中 UserList 分支 → c.Next()
  → handler 无权限复核 → 真的执行删除 → 200
```

⚠️ **这意味着「只藏前端按钮」完全是假修复**——后端 API 直调依然可写。

**修复后的期望**：严格中间件（跳过 Step 2/3）下，上表全部 8 行应变为 **step1=false / step2=false / step3=false → 403 拒绝**。Phase 78 落地后需用**同一脚本、同一角色**复跑一次，形成前后对照（复跑工具：仓库内临时目录 `tmp_t4check/`，验证后删除）。

#### 3.4.1 ⭐ 前后对照（Phase 78 落地后，同脚本同角色复跑 —— 已完成）

修复实现：`51f5e69e`（`RequireStrictPermissions` 只做精确匹配）+ `48a6920c`（`/system/users` 读组与写组**兄弟分离**）。

| perm | step1 | step2 | step3 匹配 | **旧中间件（修复前）** | **严格中间件（修复后）** |
|---|---|---|---|---|---|
| `system:user:list` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:view` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:add` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:edit` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:remove` | false | false | true | 200 放行（缺陷，会真删数据） | **403 拒绝 ✅** |
| `system:user:resetPwd` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:import` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |
| `system:user:export` | false | false | true | 200 放行（缺陷） | **403 拒绝 ✅** |

**同脚本、同角色、同真实 PG 数据**下的判定差异 —— 严格中间件只保留 Step 1（精确匹配），而该受限角色 step1 全 false → 全部 403。

**独立佐证**：`pkg/middleware/permission_strict_test.go` 的 5 个单测全绿（`TestStrictPermission_SkipsModuleDowngrade` / `SuperAdminBypass` / `ExactMatchPasses` / `UnauthenticatedRejected` / `CacheKeyIsolated`），其中首个即锁定「严格检查跳过模块降级」。

⚠️ **一个被实施方识破的关键陷阱**（Lead 规格里原本写错、实施方修正）：gin 的 `Group()` **会继承父组中间件链** → 写组若写成 `usersRead.Group("")`，读组的 ANY-of 仍会作用在写路由上，**病灶①（组级 ANY-of）不会被修掉**。正确做法是写组成为 `authorized` 的**兄弟组**（见 `48a6920c` 的 `strictUserRouteGroup`）。

**剩余待验（Phase 79 UI 实机）**：前端按钮可见性 + 真实 HTTP 返回码（T1/T4 HTTP 层）。DB 层判定已闭合（中间件唯一数据源即这三段 SQL）。

### 3.5 部门 / 岗位可用值

`11_fixture_user.sql` 已用 `ORDER BY order_num LIMIT 1` 自动取首个可用部门，故无需人工填。

### 3.6 ⚠️ 造数过程中的两个已知踩坑（已被 review 发现并规避）

1. **`sys_role_menu` / `sys_user_role` 没有 `id` 列**（`models/role.go:23-28` / `models/user.go:50-54`）：
   - `sys_role_menu` = `{RoleID, MenuID}`（**无 id、无 created_at**）
   - `sys_user_role` = `{UserID, RoleID, CreatedAt}`（**无 id**，有 created_at）
   - 仓库全部迁移都只写两/三列 → 带 `id` 的 INSERT 会报 `column "id" does not exist` → 事务回滚
   - 正确写法见 §2.2 / §2.3（`WHERE NOT EXISTS` 幂等）
2. **`sys_menu` 查询禁止 `LIKE 'system:user%'`**：库里另有两条**隐藏 C 型** `system:user:profile` / `system:user:settings`，LIKE 会把它们一起捞进来 → 受限角色意外获得页面权限。必须 `perms = 'system:user'` **精确匹配**。

---

## 4. 清理 SQL（验证完由主会话或用户执行）

```sql
-- 1. 用户关联
DELETE FROM sys_user_post
WHERE user_id IN (SELECT id FROM sys_user WHERE username = 'v124_auditor_test');

DELETE FROM sys_user_role
WHERE user_id IN (SELECT id FROM sys_user WHERE username = 'v124_auditor_test');

-- 2. 测试用户本体
DELETE FROM sys_user WHERE username = 'v124_auditor_test';

-- 3. 角色关联
DELETE FROM sys_role_menu
WHERE role_id IN (SELECT id FROM sys_role WHERE role_key = 'v124_audit_test');

DELETE FROM sys_role_dept
WHERE role_id IN (SELECT id FROM sys_role WHERE role_key = 'v124_audit_test');

-- 4. 受限角色本体
DELETE FROM sys_role WHERE role_key = 'v124_audit_test';

-- 5. 清理后自检（应全为 0）
SELECT (SELECT count(*) FROM sys_role WHERE role_key = 'v124_audit_test') AS role_left,
       (SELECT count(*) FROM sys_user WHERE username = 'v124_auditor_test') AS user_left,
       (SELECT count(*) FROM sys_role_menu rm JOIN sys_role r ON r.id = rm.role_id
         WHERE r.role_key = 'v124_audit_test') AS role_menu_left,
       (SELECT count(*) FROM sys_user_role ur JOIN sys_user u ON u.id = ur.user_id
         WHERE u.username = 'v124_auditor_test') AS user_role_left;
```

> 只删本次新增的行（按 `role_key='v124_audit_test'` / `username='v124_auditor_test'` 精确定位），
> **不会**触碰任何既有角色/用户。

---

## 5. 复测步骤（对齐 ARCHITECTURE 研究的 T1–T8 判据）

> 环境：生产前端/后端同源 `http://10.62.10.34/`（nginx → `127.0.0.1:9000`，API 前缀 `/api/v1`）
> 登录账号：**受限账号**（只挂 `v124_audit_test` 角色）——admin 走 bypass，不能用来验证。

### 5.0 造数后必等 31 秒

`pkg/middleware/permission.go:125-133` 的权限判定有 **30s 短 TTL 内存缓存**（`mwPermCheckCache`），
且 `InvalidateMiddlewarePermCache` **未接线** → 造数 / 改权限后**必须等 31s** 再测，否则命中旧缓存。

### 5.1 T1 —— 取前端真实使用的权限集

前端按钮判定读的是 `POST /api/v1/system/my-menus/permissions`（`ruoyi-react-frontend/src/store/menuStore.ts:114`）。
该端点**无权限门**，登录后直接调用即可。

```bash
TOKEN=<受限账号 accessToken>
curl -s -X POST http://10.62.10.34/api/v1/system/my-menus/permissions \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{}'
```

**判据**：返回的权限集合**不含** `system:user:add` / `system:user:edit` / `system:user:remove` /
`system:user:resetPwd` / `system:user:import` / `system:user:export`。

> 实测返回：**（实测回填）**

### 5.2 T2 / T3 —— 前端按钮可见性

1. 用受限账号登录 → 左侧应出现「系统管理 → 系统用户」
2. 进入「系统用户」页 → **操作列**应**没有**「编辑 / 重置密码 / 删除」按钮，**没有**「批量删除」勾选框与顶部按钮
3. 修复前该页按钮**恒显示**（`src/pages/system/user/index.tsx:279-301` 无任何权限判定）→ 这正是 #16 的前端缺陷

### 5.3 T4 —— 后端直调写接口（**决定性证据**）

绕过前端，直接用受限账号 token 打写接口：

```bash
TOKEN=<受限账号 accessToken>
TARGET=<一个可删的测试用户 id>

# 删除
curl -s -o /dev/null -w "%{http_code}\n" -X POST \
     http://10.62.10.34/api/v1/system/users/$TARGET/delete \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}'

# 修改
curl -s -o /dev/null -w "%{http_code}\n" -X POST \
     http://10.62.10.34/api/v1/system/users/$TARGET/update \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"nickname":"v124_probe","status":0}'

# 新增
curl -s -o /dev/null -w "%{http_code}\n" -X POST \
     http://10.62.10.34/api/v1/system/users \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"username":"v124_probe_tmp","nickname":"v124_probe_tmp","password":"Probe@123456","status":0}'

# 重置密码 / 状态
curl -s -o /dev/null -w "%{http_code}\n" -X POST \
     http://10.62.10.34/api/v1/system/users/$TARGET/reset-password \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}'
```

**判据**：修复后以上全部应返回 **403**；**当前预期是 200**（组级 ANY-of + 模块级降级双重放行）。

> **先做 DB 层预判（不必等 HTTP）**：`v_simulate_perm_check.sql` 在库内复刻
> `checkUserPermissionUncached` 的三级判定（Step1 精确 / Step2 子菜单继承 / Step3 模块降级），
> 对 `system:user:{list,view,add,edit,remove,resetPwd,import,export}` 逐条给出 `step1_exact / step2_child_inherit / step3_module_match`。
> 预期（缺陷未修）：`system:user:remove` 的 `step3_module_match = true`
> （`remove` → 降级 `system:user` → 命中页面本体）⇒ **写操作被放行 ⇒ HTTP 200**。
> 该脚本是**只读**的，可作为 T4 的前置预判证据。

> 实测返回码：**（实测回填）**

> ⚠️ **执行前必须确认 `$TARGET` 是可安全删除的测试用户**——当前缺陷下 delete **会真的删数据**。
> 建议 `$TARGET` 用本次造出来的 `v124_auditor_test` 自身之外的临时用户，或先在 UI 侧新建一个一次性用户再打。

### 5.4 T5–T8 对照判据

| 编号 | 检查项 | 期望（修复后） |
|---|---|---|
| T5 | 受限账号 `POST /system/users/list` | 200（读操作**应该**放行；注意本夹具**未**绑 `system:user:list`，能通过是因为模块降级命中 `system:user`，属病灶 2 的表现） |
| T6 | 受限账号 `POST /system/users`（新增） | 403 |
| T7 | 受限账号 `POST /system/users/{id}/status` | 403 |
| T8 | 受限账号 `POST /system/users/batch-delete` | 403 |

### 5.5 修复后的回归期望（供主会话对照）

| 角色 | 期望 |
|---|---|
| admin（超管，bypass） | 全部按钮可见，全部接口 200 |
| xtgl（系统管理；**无 export**） | 写按钮可见；导出按钮按现状裁定（给 xtgl 补 export 或 export 接口不加保护） |
| `v124_audit_test`（本夹具） | 写按钮**不可见**；写接口 **403** |

---

## 6. 边界与合规声明

- **未修改任何现有数据**：全程只 `INSERT` 新行；**未** 对任何既有 `sys_role` / `sys_user` / `sys_role_menu` / `sys_user_role` 执行 UPDATE 或 DELETE
- **未挂 admin 角色**：验证账号只挂新建的 `v124_audit_test`；`role_key != 'admin'` ⇒ 不触发 `isSuperAdmin` bypass
- **未改业务代码**：`internal/**` 与 `ruoyi-react-frontend/src/**` 一行未动；本任务只产出本 note 一个文件
- **凭据零落盘**：数据库口令只经**环境变量**传递，未写入任何文件、未写入提交；临时 Go 工具位于 `%TEMP%\pgfix\`，**用完即删**
- **临时命名**：角色 `v124_audit_test` / `V1.24审计测试(临时)`，用户 `v124_auditor_test`，便于事后精准清理

---

## 7. 复审要点

1. 复跑 §2 的造数 SQL → 确认 `sys_role` 增 1 行、`sys_role_menu` 增 2 行（页面本体 + 父目录）
2. 复核 §3.3 的 perms 集合 —— **不应出现任何 F 型标识**
3. 复核 §5.1 的 T1 返回 —— **不应出现** add/edit/remove/resetPwd/import/export
4. 复核 §5.3 的 T4 返回码 —— **200 = 缺陷坐实（需在 v1.24 修复）**；403 = 已修复
5. 执行 §4 的清理 SQL → 确认 4 个计数全为 0

---

## 8. 坑位清单（造数前必读，均为本次实测/代码核实所得）

| # | 坑 | 结论 |
|---|---|---|
| 1 | `sys_role_menu` / `sys_user_role` **没有 `id` 列** | `models/role.go:23-28` RoleMenu{RoleID,MenuID}（**无 id、无 created_at**）；`models/user.go:50-54` UserRole{UserID,RoleID,CreatedAt}（**无 id**）。仓库全部迁移都只写两列（`INSERT INTO sys_role_menu (role_id, menu_id)`，见 `menu_grant_helpers.go:34`、`migrations/136_add_group_mapping_menu.sql:281`）。**写 `INSERT INTO sys_role_menu (id, ...)` 会报 `column "id" does not exist`** |
| 2 | 隐藏 C 型菜单 `system:user:profile` / `system:user:settings` | 查 `system:user` 相关菜单**必须用 `perms = 'system:user'` 精确匹配**；用 `LIKE 'system:user%'` 会误绑这两条 |
| 3 | 页面本体 perms 是 `system:user`，而 `system:user:list` 是 **F 型按钮** | 两者语义完全不同：只挂页面本体（C）即"能进页面、无任何按钮权限"；想只读列表需**额外**绑 F 型 `system:user:list`（此时 perms 集合会多一项，但不涉写权限） |
| 4 | 菜单 id 有两套历史版本 | 生产用 `62df1795-…`（C，perms=`system:user`）；旧 `init_data.sql` 用 `550e8400-e29b-41d4-a716-446655440104`（C，perms 直接写成 `system:user:list`）。本 note 的造数 SQL **按 perms 动态取 id、不写死**，两套都安全 |
| 5 | 权限判定 30s 短 TTL 缓存 | `pkg/middleware/permission.go:125-133`；`InvalidateMiddlewarePermCache` 未接线 → 造数后**等 31s** 再测 |
| 6 | 本地实例连生产库会跑 `AutoMigrate()` | `internal/core/db` 的 `d.AutoMigrate()` 在启动期对目标库执行 DDL → 若为做 T4 而本地起后端连生产库，**先确认迁移已全部应用且本仓版本不引入新 DDL**；否则改用 DB 层判定模拟（本 note §5.3 的 `v_simulate_perm_check.sql`） |
| 7 | T4 真打 HTTP 时 delete 会**真的删数据** | 缺陷未修时 `POST /system/users/{id}/delete` 返回 200 且数据消失 → `$TARGET` 只选本次造出来的临时用户，别拿生产用户试 |

---

## 9. 对主会话造数工具（`tmp_pgfixture`）的 review 结论

| 项 | 结论 |
|---|---|
| 🔴 阻断 | `apply()` 的 `INSERT INTO sys_role_menu (id, role_id, menu_id, created_at)` 与 `ensureUserRole()` 的 `INSERT INTO sys_user_role (id, user_id, role_id, created_at)` **多写了不存在的 `id` 列**（`sys_role_menu` 还多写了 `created_at`）→ 会报 `column "id" does not exist`。正确写法见 §8 坑位 #1 |
| ✅ 通过 | 事务包裹；`NOT EXISTS` 幂等；F 型计数校验；重复行校验；递归 CTE 取 `system:user` 祖先链且只保留 `C`/`M`；`cleanup` 按 `role_id` / `username` 精确定位、不误删既有数据 |
| 🟡 小瑕疵 | ① `data_scope` 注释（"user = 1 本人"）与代码值 `3` 不符（`models/base.go`：3=本部门数据、5=仅本人）——**代码值 3 正确**，改注释即可；② `mkuser` 未开事务，`INSERT sys_user` 成功但后续失败会留半成品；③ `cleanup` 只校验角色残留，建议补 `sys_role_menu` / `sys_user_role` / `sys_user` 三表残留校验 |

> 该工具位于**仓库内** `tmp_pgfixture/`，验收结束后必须 `Remove-Item -Recurse -Force tmp_pgfixture` 删除，禁止提交。

