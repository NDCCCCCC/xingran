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
| 受限角色 id | **`0258375e-8d7b-4393-b5e9-9b4f6476dad5`** ✅ 已落库（§3.1） |
| 测试用户 username | `v124_auditor_test` |
| 测试用户 id | **`76c72635-df7c-435b-b171-0b05e4dd7f2b`** ✅ 已落库（§3.2） |
| 该角色实际 perms 集合 | **`{system, system:user}`** —— 仅页面级 `system:user`(C) + 父目录 `system`(M)，**F 型计数 = 0** ✅（§3.3） |
| 菜单 id 对照表 | **已备妥**（§1.4）—— **已用生产库实测复核，库内实际 id 与本表逐条一致**（§3.7） |
| T1 `POST /system/my-menus/permissions` 实际返回 | **HTTP 200** → `["system","system:user"]`，**不含** add/edit/remove/resetPwd/import/export（也不含 list/view）✅（§5.1） |
| T4 直调 `POST /system/users/{id}/delete` 实际返回码 | **HTTP 403 = 后端已守住**（Phase 78 修复后）；**对照**：同一账号临时追加 `xtgl` 后 → **HTTP 200 且真删除** → 证明 403 来自权限门而非接口故障（§5.3） |
| 是否修改既有数据 | **否（既有业务行零改动）** —— 只**新增**夹具角色/用户/关联行；对照实验曾软删**本次造出的测试账号自身**，已即时恢复（§6） |


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

**实测（本次落库用值）**：夹具账号挂在既有普通用户所在的部门

| 项 | 实测值 |
|---|---|
| `dept_id` | `250a1acd-2ddd-4a3a-943c-be555498665f`（团客中心团客销售一部，`dept_code=C23`，`ancestors=6bdb9ce8-…,c066da1d-…`） |
| `sys_user_post` | **未挂任何岗位**（与生产既有普通用户一致：`post_cnt=0`）——本次验证不依赖岗位 |

（更多可选项见 §3.5；造数 SQL 用 `ORDER BY order_num LIMIT 1` 自动取首个可用部门，无需人工填。）

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

#### 3.4.2 ⭐⭐ 真实 HTTP 层实测（Phase 79 —— 本次会话实测，2026-10-28 10:52–11:05）

> §3.4 / §3.4.1 是 **DB 层模拟**（复刻中间件的三段 SQL）。本节是**真实 HTTP 层**实测：
> 本地起后端（连生产 PG + 生产 Redis）→ 用夹具账号的**真实 JWT** → `curl` 直打 → 记录**真实返回码**。
> **结论：DB 层模拟与 HTTP 实测一致（修复后 403），两者互为佐证。**

**实测环境（关键参数可复跑）**

| 项 | 值 | 说明 |
|---|---|---|
| 后端二进制 | `go build -o $TEMP/ruoyi-backend-v124.exe ./cmd/main.go`（构建于 10:51，**包含** `51f5e69e` + `48a6920c` 两个修复 commit） | |
| DB / Redis | `10.62.10.34:5432/ruoyi` / `10.62.10.34:6379` | 生产库/生产缓存 |
| 启动旁路开关 | `SERVER_MODE=test`、`SKIP_AUTOMIGRATE=true`、`SERVER_SKIP_SETUP=true`、`DB_PASSWORD=…` | ① `SKIP_AUTOMIGRATE=true` 在 `server.mode=release` 下会 **fatal 拒绝启动**（`internal/core/core.go:286-291`），故必须同时降 mode；② `SERVER_SKIP_SETUP=true` 跳过 InitData/默认角色菜单 seed |
| JWT 密钥注入 | `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`（**hex**） | ⚠️ **不是** `config.yaml` 注释里写的 `RUOYI_JWT_SM2_*`——真实绑定见 `internal/config/config.go:364-365` |
| 夹具账号 token | 用**同一对** SM2 密钥本地自签（`crypto.GenerateTokenWithSM2`，issuer=`RuoYi-Next`） | 生产 `sys.account.captchaEnabled=slider` + `sys.auth.default.mode=ad` ⇒ **登录通道不可用**，只能自签 token |
| 请求体加密 | 生产库 `sys.request.encryption.enabled=false` ⇒ **明文 JSON 直通**（`require_encryption=false`） | |

**T1 实测** —— `POST /api/v1/system/my-menus/permissions`（无权限门）

```
HTTP 200
{"code":0,"message":"success","data":["system","system:user"], ...}
```

⇒ 权限集 = `{system, system:user}`，**不含** `add` / `edit` / `remove` / `resetPwd` / `import` / `export`
（连 `list` / `view` 也没有——它们是 F 型按钮菜单，本夹具一条不绑）。
配套 `POST /api/v1/system/my-menus` 返回的菜单树也只有两级：`系统管理(M) → 系统用户(C)`。

**T4 实测** —— 同一 token（只挂 `v124_audit_test`）直打写接口，**只读角色全 403**：

| 端点 | 期望 | **实测** | 响应体 |
|---|---|---|---|
| `POST /system/users`（新增） | 403 | **403** | `{"code":403,"message":"没有访问权限"}` |
| `POST /system/users/{id}/update` | 403 | **403** | 同上 |
| `POST /system/users/{id}/status` | 403 | **403** | 同上 |
| `POST /system/users/{id}/delete` | 403 | **403** | 同上 |
| `POST /system/users/batch-delete` | 403 | **403** | 同上 |
| `POST /system/users/{id}/reset-password` | 403 | **403** | 同上 |
| `POST /system/users/import` | 403 | **403** | 同上 |
| `POST /system/users/sync-managers` | 403 | **403** | 同上 |
| `POST /system/users/list`（读） | 200 | **200** | 放行（读组 ANY-of 命中 `system:user` 页面本体） |
| `POST /system/users/{id}`（读单个） | 200 | **200** | 放行 |
| `POST /system/users/statistics`（读） | 200 | **200** | 放行 |
| `POST /system/roles/list`（跨模块） | 403 | **403** | 权限隔离正常 |

**决定性对照（排除「403 只是接口坏了」）**：给**同一个夹具账号**临时追加既有角色 `xtgl`（持 `system:user:remove` 等写权限），
等权限缓存失效后重打 delete：

```
POST /system/users/{自己的 id}/delete   →   HTTP 200  {"code":0,"message":"success"}
```

⇒ 同一个端点，**有权 → 200 且真的删除**（软删 `deleted_at`），**无权 → 403 且数据不动** ⇒
**403 确系权限门按角色生效，不是接口/路由/参数问题**。对照后已把追加的 `xtgl` 关联行移除、
被软删的夹具账号即时恢复（`deleted_at=NULL` + 重挂 `v124_audit_test`）。

> ⚠️ **删除目标只用本次造出的测试账号自身**——绝不拿生产用户试（缺陷未修时 delete 会真删数据）。

**副作用如实记录**（本地实例连生产库 inevitably 会动的表）：
`sys_oper_log`（操作日志，delete/update 各 1 行）、`ad_push_queue`（delete 触发的 AD disable 入队 1 行）、
`sys_menu` / MAC 物化视图（`REFRESH MATERIALIZED VIEW CONCURRENTLY`，后端启动期定时任务）、
Redis 缓存（预热写缓存，非业务数据）。**未触碰任何既有业务行**。

**复跑命令（本地起后端 + 自签 token + curl）**

```powershell
# ① 生成一对固定 SM2 密钥（私钥 32B hex / 公钥 04||X||Y hex），注入后端与自签工具用同一对
# ② 起后端（旁路开关见上表），前台托管会话运行：
$env:DB_PASSWORD = '<口令>'
$env:SERVER_MODE = "test"            # release 下 SKIP_AUTOMIGRATE=true 会 fatal
$env:SKIP_AUTOMIGRATE = "true"       # 跳过 AutoMigrate，避免对生产库发 DDL
$env:SERVER_SKIP_SETUP = "true"      # 跳过 InitData / 默认角色菜单 seed
$env:JWT_SM2_PRIVATE_KEY = '<私钥hex>'   # ⚠️ 变量名不是 RUOYI_JWT_SM2_*
$env:JWT_SM2_PUBLIC_KEY  = '<公钥hex>'
& "$env:TEMP\ruoyi-backend-v124.exe"

# ③ 自签夹具账号 token（同一对密钥，issuer=RuoYi-Next），或直接用生产账号在 UI 侧登录
# ④ T1 / T4
curl.exe -s -X POST http://localhost:9000/api/v1/system/my-menus/permissions -H "Authorization: Bearer $TOKEN" -d '{}'
curl.exe -s -o NUL -w "%{http_code}" -X POST "http://localhost:9000/api/v1/system/users/$TARGET/delete" -H "Authorization: Bearer $TOKEN" -d '{}'
```

> ⚠️ 若在**生产的常驻后端**（`http://10.62.10.34/api/v1`，nginx → `127.0.0.1:9000`）上复测，
> **无需起本地实例**，但**必须等 31s**（§5.0）以避开 30s 权限缓存；且 `$TARGET` 务必用夹具账号自身或另建的一次性用户。

### 3.5 部门 / 岗位可用值

`11_fixture_user.sql` 已用 `ORDER BY order_num LIMIT 1` 自动取首个可用部门，故无需人工填。

### 3.6 ⚠️ 造数过程中的三个踩坑（第 1 条修正了 Lead 的错误推断）

1. **⚠️ 更正：`sys_role_menu` / `sys_user_role` 的生产库表其实【有】`id` / `created_at` 列** ——
   Lead 曾据 `internal/models/role.go:23-28`（`RoleMenu{RoleID, MenuID}`）与 `internal/models/user.go:50-54`
   （`UserRole{UserID, RoleID, CreatedAt}`）**推断**「表没有 id 列」，并据此判定另一 agent 的
   `INSERT ... (id, ...)` 会报 `column "id" does not exist` → **该推断错误**：
   - 实测：**带 `id`（`gen_random_uuid()`）+ `created_at`（`now()`）的 INSERT 执行成功**
   - 真相是 **Go 模型未映射这些列**，而非**库表缺列**（模型不映射 ≠ 表无列）
   - 正确结论：造数 SQL 两种写法都能成功，但**推荐 `WHERE NOT EXISTS` 幂等**
     （`sys_role_menu` 只有普通索引、非 unique，不加 `NOT EXISTS` 会插出重复行）
   - ⚠️ 该错误推断的连带影响：**sqlite / 按 GORM 模型新建的库**才真会缺列报错 → **PG 能跑 ≠ 新库能跑**，
     写造数脚本不要把两者混同
   - 本条同时修正了 §8 坑位 #1 与 §9 review 结论中的对应表述（原文按「表无 id 列」描述，现按本条为准）
2. **`sys_menu` 查询禁止 `LIKE 'system:user%'`**：库里另有两条**隐藏 C 型** `system:user:profile` / `system:user:settings`，LIKE 会把它们一起捞进来 → 受限角色意外获得页面权限。必须 `perms = 'system:user'` **精确匹配**。
3. **本地起后端连生产库的最小旁路**（规避 `AutoMigrate()` 对生产库执行 DDL）：
   - `SERVER_MODE=test` + `SKIP_AUTOMIGRATE=true`（⚠️ **release 模式下该开关 fatal 拒绝启动**，`internal/core/core.go:286-291`，故必须同时降 mode）+ `SERVER_SKIP_SETUP=true` + `DB_PASSWORD`
   - 请求体加密：生产 `sys.request.encryption.enabled=false` ⇒ **明文 JSON 直通**
   - ⚠️ **JWT 环境变量名与 config 注释不符**：注释写 `RUOYI_JWT_SM2_*`，**真实绑定是
     `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`**（`internal/config/config.go:364-365`）→ 按注释注入**静默失效 → 401**
   - ⚠️ 生产登录通道不可用（`sys.account.captchaEnabled=slider` 滑块 + `sys.auth.default.mode=ad`）⇒ 造的口令登不进去；
     解法是本地自签 token（**issuer 必须为 `RuoYi-Next`**），且需注入与前端一致的 SM2 密钥对

### 3.7 菜单 id 生产库实测复核（§1.4 对照表 vs 库内实际值）

本次用 `information_schema` + 直查 `sys_menu` 复核，**§1.4 的 id 与生产库逐条一致**：

| perms | menu_type | 库内实测 id | 与 §1.4 |
|---|---|---|---|
| `system:user` | C | `62df1795-0ba3-4528-a280-bae0ad4ef8e2` | ✅ 一致 |
| `system`（父目录） | M | `d67f4240-f887-481a-b345-94fb36782500` | ✅ 一致 |
| `system:user:list` | F | `9c0afb05-fb59-4b81-832f-fbc113dd6d28` | ✅ 一致 |
| `system:user:view` | F | `aca92397-10df-4ed0-8782-0de986257030` | ✅ 一致 |
| `system:user:add` | F | `125fb2e1-c0d5-40ad-8e91-ffd42e3d3953` | ✅ 一致 |
| `system:user:edit` | F | `9fcd2d5a-afce-48ff-95c9-2c74fe941ef7` | ✅ 一致 |
| `system:user:remove` | F | `481729f8-9931-44a5-982e-858a306da208` | ✅ 一致 |
| `system:user:resetPwd` | F | `edcfc84b-8bc6-4799-a0b8-f44551873bbf` | ✅ 一致 |
| `system:user:import` | F | `76657665-f5e0-4dfe-b760-4927af1911fd` | ✅ 一致 |
| `system:user:export` | F | `2fdf5135-7f67-4048-ac86-cad709ab9130` | ✅ 一致 |
| `system:user:profile`（隐藏 C，**不绑**） | C | `65f828bc-9aa0-4f16-a2fe-1aea0acc4766` | ✅ 一致（§1.4 坑点提示成立） |
| `system:user:settings`（隐藏 C，**不绑**） | C | `c480cc01-c2e4-47ff-a958-6c25cfa0cf95` | ✅ 一致 |

**库内实际 schema 与 GORM 模型不一致（重要！）**：`information_schema` 实测

- `sys_role_menu` = `id uuid NOT NULL DEFAULT gen_random_uuid()` + `role_id uuid NOT NULL` + `menu_id uuid NOT NULL` + `created_at timestamp`
- `sys_user_role` = `id uuid NOT NULL DEFAULT gen_random_uuid()` + `user_id uuid NOT NULL` + `role_id uuid NOT NULL` + `created_at timestamp`
- `sys_user.id` = `uuid NOT NULL`（**不是** `user_id`）；`sys_user.nickname`（**不是** `nick_name`）；`sys_user.auth_source varchar NOT NULL DEFAULT 'local'`
- `sys_role` = 无 `dept_id` 列；`data_scope` 是 `bigint`（枚举 1=全部 3=本部门 5=仅本人，见 `models/base.go:73-77`）

⇒ §8 坑位 #1 / §9 的「`id` 列不存在」结论**只对 GORM 模型（sqlite/新库）成立**；
**生产库实际有 `id`/`created_at` 列**，本次实测带 `id` 的 INSERT **执行成功**（见 §9 修订）。

### 3.8 ⭐ 只读 SQL 等价判据（**不依赖起后端**，可独立复测）

> 本节由**第三个并行会话**（只读复核会话）实测补入：把 T1 / T4 判据**完全落成只读 SQL**，
> 使「生产后端不可用」或「不允许起本地实例连生产库」时，仍能得到**等价结论**。
> 全部 SQL **只 SELECT**，可用 `SET TRANSACTION READ ONLY` / psycopg2 `readonly=True` 会话强制兜底。

#### 3.8.1 环境事实（决定了为什么必须走只读路径）

| 探针 | 实测结果 | 含义 |
|---|---|---|
| `GET http://10.62.10.34:9000/` | **403** + `Server: MinIO` | ⚠️ **9000 端口是 MinIO，不是后端**（note 原文「nginx → 127.0.0.1:9000」的假设在当前环境不成立） |
| `GET http://10.62.10.34/` | **502 Bad Gateway**（nginx/1.18.0） | **生产后端进程当前未运行** ⇒ 走生产 HTTP 复测 T1/T4 现在**做不到** |
| `information_schema` 查 `%migration%/%version%/%schema%` 表 | 仅 `sys_dashboard_versions`（仪表盘版本，无关） | **生产库没有迁移记录表** ⇒ 无法用「迁移版本 vs 仓库最新版本」核对 DDL 风险（仓库最新迁移 = `migration_224_*`） |

⇒ 结论：**只读 SQL 判据是当前唯一无需起实例即可复测的路径**（本地起实例还额外触发
`cmd/main.go:358 importOUIData`（批量写 OUI 厂商数据）、`initMACHistoryRetentionConfig`（可能 INSERT `sys_config`）
与 core 内的 cron 调度器写库 —— 都会突破「不修改既有数据」边界）。

#### 3.8.2 T1 等价 SQL（逐行复刻 `GetUserPermissions`）

对应实现：`internal/services/system/menu_service.go:509-569` `GetUserPermissions`
+ `appendAncestorMenuIDs`（`menu_service.go:280`）。四步：

1. `sys_user_role WHERE user_id = ?` → 2. `DISTINCT menu_id FROM sys_role_menu WHERE role_id IN (...)`
   → 3. 递归上溯 `parent_id` 补祖先（GORM 软删过滤 `deleted_at IS NULL`）
   → 4. `SELECT perms FROM sys_menu WHERE id IN (...) AND perms IS NOT NULL AND perms <> ''`（**不过滤 status**）

```sql
WITH RECURSIVE
tu AS (SELECT id FROM sys_user WHERE username='v124_auditor_test' AND deleted_at IS NULL),
ur AS (SELECT DISTINCT role_id FROM sys_user_role WHERE user_id IN (SELECT id FROM tu)),
rm AS (SELECT DISTINCT menu_id FROM sys_role_menu WHERE role_id IN (SELECT role_id FROM ur)),
chain AS (
    SELECT m.id, m.parent_id FROM sys_menu m
     WHERE m.id IN (SELECT menu_id FROM rm) AND m.deleted_at IS NULL
    UNION
    SELECT p.id, p.parent_id FROM sys_menu p JOIN chain c ON p.id = c.parent_id
     WHERE p.deleted_at IS NULL
)
SELECT m.perms AS perms, m.menu_type, m.menu_name, m.id::text AS menu_id
FROM sys_menu m
WHERE m.id IN (SELECT id FROM chain) AND m.perms IS NOT NULL AND m.perms <> ''
ORDER BY m.perms;
```

**实测结果（与 §3.4.2 的 HTTP 200 返回逐字一致 ✅）**：

```
{"perms":"system",     "menu_type":"M", "menu_name":"系统管理", "menu_id":"d67f4240-f887-481a-b345-94fb36782500"}
{"perms":"system:user","menu_type":"C", "menu_name":"系统用户", "menu_id":"62df1795-0ba3-4528-a280-bae0ad4ef8e2"}
```

⇒ 权限集 = **`{system, system:user}`**，**F 型计数 = 0** ⇒ **T1 判据满足**
（不含 `add`/`edit`/`remove`/`resetPwd`/`import`/`export`，也不含 `list`/`view`）。

#### 3.8.3 T4 等价 SQL（修复前三级判定 vs Phase 78 严格判定，一次跑出两侧）

```sql
WITH target_user AS (SELECT id FROM sys_user WHERE username='v124_auditor_test' AND deleted_at IS NULL),
target_roles AS (SELECT id FROM sys_role WHERE role_key='v124_audit_test' AND deleted_at IS NULL),
held AS (   -- 用户实际持有的菜单（经 role_menu）
    SELECT DISTINCT m.id, m.perms, m.status, m.menu_type
    FROM sys_user_role ur
    JOIN sys_role_menu rm ON rm.role_id = ur.role_id
    JOIN sys_menu m ON m.id = rm.menu_id
    WHERE ur.user_id IN (SELECT id FROM target_user)
      AND ur.role_id IN (SELECT id FROM target_roles)
),
cand(perm) AS (
    VALUES ('system:user:list'),('system:user:view'),('system:user:add'),
           ('system:user:edit'),('system:user:remove'),('system:user:resetPwd'),
           ('system:user:import'),('system:user:export')
)
SELECT c.perm,
  (SELECT count(*) FROM held h WHERE h.perms = c.perm AND h.status = 0) > 0 AS step1_exact,
  EXISTS (   -- Step2 语义严格按 git show 51f5e69e^:pkg/middleware/permission.go 第 2 步
    SELECT 1 FROM sys_menu p2
    WHERE p2.perms = c.perm AND p2.status = 0
      AND EXISTS (
        SELECT 1 FROM sys_menu child
        JOIN held h ON h.perms = child.perms AND h.status = 0
        WHERE child.parent_id = p2.id AND child.menu_type = 'F' AND child.status = 0
          AND child.perms IS NOT NULL AND child.perms <> ''))
  ) AS step2_child_inherit,
  regexp_replace(c.perm, ':[^:]*$', '') AS step3_module_perm,
  (SELECT count(*) FROM held h WHERE h.status = 0
     AND h.perms = regexp_replace(c.perm, ':[^:]*$', '')) > 0 AS step3_module_match,
  -- 修复前：三级任一命中即放行
  ( (SELECT count(*) FROM held h WHERE h.perms = c.perm AND h.status = 0) > 0
    OR EXISTS (SELECT 1 FROM sys_menu p2 WHERE p2.perms = c.perm AND p2.status = 0
        AND EXISTS (SELECT 1 FROM sys_menu child
            JOIN held h ON h.perms = child.perms AND h.status = 0
            WHERE child.parent_id = p2.id AND child.menu_type='F' AND child.status=0
              AND child.perms IS NOT NULL AND child.perms <> ''))
    OR (SELECT count(*) FROM held h WHERE h.status = 0
          AND h.perms = regexp_replace(c.perm, ':[^:]*$', '')) > 0
  ) AS legacy_any_pass,
  -- Phase78 修复后：RequireStrictPermissions 仅 Step1 精确匹配
  (SELECT count(*) FROM held h WHERE h.perms = c.perm AND h.status = 0) > 0 AS strict_exact_pass
FROM cand c ORDER BY c.perm;
```

**实测结果（8 行完全同构）**：

| perm | step1 | step2 | step3 模块降级 | **修复前（legacy_any_pass）** | **修复后（strict_exact_pass）** |
|---|---|---|---|---|---|
| `system:user:list` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:view` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:add` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:edit` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:remove` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:resetPwd` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:import` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |
| `system:user:export` | false | false | `system:user` → **true** | **true → 200 放行** | **false → 403 拒绝** |

⇒ **与 §3.4（修复前 200 放行）、§3.4.1/§3.4.2（修复后 403）三方完全一致** —— DB 层与 HTTP 层互为佐证。

**组级 ANY-of 模拟**（修复前 `users` 写组 `[UserList, UserAdd, UserEdit, UserView]` 是 OR）：

| 组内 perm | 精确匹配 | 模块降级命中 |
|---|---|---|
| `system:user:list` | false | **true** |
| `system:user:add` | false | **true** |
| `system:user:edit` | false | **true** |
| `system:user:view` | false | **true** |

⇒ 组内**任一**命中即 `c.Next()` ⇒ 修复前 delete 等写端点**必被放行**（与 §3.4 的链路图一致）。

#### 3.8.4 菜单 id / 祖先链 / 唯一性 —— 独立复核实测

| 探针 | 实测 | 结论 |
|---|---|---|
| `perms='system:user' AND menu_type='C' AND status=0` | **1** 条 | 精确匹配**唯一命中** `62df1795-…` ✅ |
| `perms='system:user' AND menu_type='C'`（不限 status） | **1** 条 | 无停用的重复项 |
| `perms LIKE 'system:user%'` | **11** 条 | ⚠️ = 1 页面 + **2 隐藏 C** + **8 F 型** ⇒ 量化了 §8 坑位 #2 的风险面 |
| `perms='system:user:profile'` / `'system:user:settings'` | 各 **1** 条 | 两条隐藏 C 型确认存在 |
| 旧 legacy id `550e8400-e29b-41d4-a716-446655440104` | **0** 条 | §8 坑位 #4 成立：生产库**不用**旧 id ✅ |
| `menu_type='F' AND perms LIKE 'system:user:%'` | **8** 条 | 8 个 F 型按钮齐全，与 §1.4 表一一对应 |
| 祖先链（递归 CTE，只留 C/M） | **2 层**：`62df1795(C) → d67f4240(M) → parent=NULL` | 只绑 C+M 即完整，无更深祖先 |

#### 3.8.5 幂等复跑实测（造数脚本可重复执行）

复跑 `10_fixture_role.sql`（3 条语句）：

```
stmt 1 INSERT INTO sys_role        ... (0 rows) affected=0
stmt 2 INSERT INTO sys_role_menu   ... (0 rows) affected=0
stmt 3 INSERT INTO sys_role_menu   ... (0 rows) affected=0
```

⇒ **零重复行**，`WHERE NOT EXISTS` 幂等生效；复跑后 `sys_role_menu` 仍为 **2** 行。

#### 3.8.6 残留校验实测（补 §9 小瑕疵 ③ 建议的三表校验）

| 表 | 夹具残留行数 | 期望 |
|---|---|---|
| `sys_role`（`role_key='v124_audit_test'`） | 1 | 1（待清理） |
| `sys_role_menu` | 2 | 2（待清理） |
| `sys_role_dept` | **0** | 0 ✅ |
| `sys_user`（`username='v124_auditor_test'`） | 1 | 1（待清理） |
| `sys_user_role` | 1 | 1（待清理） |
| `sys_user_post` | **0** | 0 ✅ |

⇒ **无半成品残留**（无「user 建好但 user_role 丢失」之类的中间态）。
另附全库行数快照（供清理前后对账）：`sys_role`=7、`sys_user`=4232、`sys_user_role`=4242、`sys_role_menu`=353。

### 3.9 ⚠️ 并发干扰实测（**验收必须串行化的实证**）

本次只读复核会话在 11:0x 执行 §3.8.2 的 T1 等价 SQL 时，**第一次读到 35 行权限**：

```
ops:manager:org:delete / ops:manager:org:edit / ops:manager:org:list
ops:manager:rule:delete / … / system:dict:add / system:dict:edit /
system:dict:list ×3 / system:notice:add / system:notice:remove /
system:apikey:list / system:config / system:dept / system:menu / system:post /
system:role / system:user + **全部 8 个 F 型按钮** …
```

⇒ 该快照里夹具账号**持有全部 F 型按钮权限**（含 `system:user:add/edit/remove/resetPwd/import/export/list/view`）
**且横跨 5 个模块**——与「受限角色 0 个 F 权限」的定义**直接冲突**。

**根因**（按时间戳反查，非猜测）：

| 时刻 | 事件 | 证据 |
|---|---|---|
| 10:49:30 | 夹具角色 + 2 条 `sys_role_menu` 落库 | `sys_role_menu.created_at` |
| 10:49:45 | 夹具用户落库 | `sys_user.created_at` |
| **11:03:50** | **`sys_user_role` 被重挂 / `sys_user.updated_at` 同时刷新** | `sys_user_role.created_at=03:03:50.920569+00`、`sys_user.updated_at=03:03:50.912907+00`（UTC，= 本地 11:03:50） |
| 11:04+ | 复跑同一 SQL → **2 行**（`system` + `system:user`）✅ | 本会话二次实测 |

⇒ 该窗口内有**另一个并行会话正在改同一夹具**（临时给账号挂宽权限角色做对照实验），
我的查询**恰好落在窗口内** ⇒ 读到的是**中间态快照**。

**给验收的硬规则**：

1. **T1/T4 取证前后各跑一次「权限集快照」**，两次数值一致才认；不一致 ⇒ 判定期间有并发改动。
2. **同一夹具不要被两个会话同时使用**；对照实验（临时追加 `xtgl` 等）与取证**必须串行**。
3. 任何「权限集与夹具定义不符」的结论，**先怀疑并发**，再怀疑代码/数据。
4. 夹具账号的 `sys_user_role.created_at` 会随「重挂」刷新 —— 可用它判断最近一次角色变更时间。

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

#### 4.1 ✅ 清理执行记录（2026-10-28，用户确认后由主会话执行）

用户选择「立即清理」→ 已按本节 SQL 在**单个事务**内执行（先子后父：user_post → user_role → user → role_menu → role_dept → role）：

| 步骤 | 影响行数 |
|---|---|
| `DELETE sys_user_post` | 0 |
| `DELETE sys_user_role` | **1** |
| `DELETE sys_user` | **1** |
| `DELETE sys_role_menu` | **2** |
| `DELETE sys_role_dept` | 0 |
| `DELETE sys_role` | **1** |

**COMMIT_OK**，清理后自检 6 项**全为 0**：

| 表 | 清理前夹具残留 | 清理后 | 期望 |
|---|---|---|---|
| `sys_role`（role_key） | 1 | **0** | 0 ✅ |
| `sys_user`（username） | 1 | **0** | 0 ✅ |
| `sys_role_menu` | 2 | **0** | 0 ✅ |
| `sys_user_role` | 1 | **0** | 0 ✅ |
| `sys_user_post` | 0 | **0** | 0 ✅ |
| `sys_role_dept` | 0 | **0** | 0 ✅ |

**全库总量对照**：`sys_role` 7 → **6**、`sys_user` 4232 → **4231** —— 差值恰为本次删除的 1 个夹具角色 + 1 个夹具用户，
**其余既有行零改动**（与 §3.8.6 记录的清理前基线 `sys_role=7 / sys_user=4232 / sys_user_role=4242 / sys_role_menu=353` 减去夹具完全吻合）。

⇒ **生产库已恢复原状**。证据（角色 id / perms 集合 / HTTP 返回码 / 前后对照表）全部固化在本 note，
不影响后续回溯；如需重跑夹具，按 §2 的造数 SQL 幂等复跑即可。

---

## 5. 复测步骤（对齐 ARCHITECTURE 研究的 T1–T8 判据）

> 环境：生产前端/后端同源 `http://10.62.10.34/`（nginx → `127.0.0.1:9000`，API 前缀 `/api/v1`）
> 登录账号：**受限账号**（只挂 `v124_audit_test` 角色）——admin 走 bypass，不能用来验证。

### 5.0 造数后必等 31 秒

`pkg/middleware/permission.go:125-133` 的权限判定有 **30s 短 TTL 内存缓存**（`mwPermCheckCache`），
且 `InvalidateMiddlewarePermCache` **未接线** → 造数 / 改权限后**必须等 31s** 再测，否则命中旧缓存。

> 📌 **本次实测的例外与提醒**：实测用的是**新起的本地后端进程** ⇒ 进程内权限缓存是**空的**，
> 故**无需等 31s**（且 T1/T4 的第一次检查在日志里可见 `[Permission Check] … module match PASS`）。
> 但若在**生产的常驻后端**（或任何已运行 ≥30s 的实例）上复测，**必须等 31s** 或重启该实例，
> 否则会读到造数前的陈旧权限判定。

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

> **实测返回（2026-10-28 真实 HTTP，Phase 79）**：**HTTP 200**
> ```json
> {"code":0,"message":"success","data":["system","system:user"],"timestamp":1793156127,"request_id":"1793156127706025600"}
> ```
> ⇒ 权限集 = `{system, system:user}`，**不含** `system:user:add` / `edit` / `remove` / `resetPwd` / `import` / `export`
> （**也不含** `list` / `view` —— 二者是 F 型按钮菜单，本夹具一条不绑）。
> 配套 `POST /system/my-menus` 菜单树 = `系统管理(M) → 系统用户(C)` 两级，**无任何 F 型按钮节点**。
> 完整实测记录见 §3.4.2。

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

> **实测返回码（2026-10-28 真实 HTTP，Phase 79）**：**403 = 后端已守住**（Phase 78 修复后）
>
> | 端点 | 实测返回码 | 说明 |
> |---|---|---|
> | `POST /system/users/{id}/delete` | **403** | `{"code":403,"message":"没有访问权限"}`；**数据未动** |
> | `POST /system/users`（新增） | **403** | 同上 |
> | `POST /system/users/{id}/update` | **403** | 同上 |
> | `POST /system/users/{id}/status` | **403** | 同上 |
> | `POST /system/users/batch-delete` | **403** | 同上 |
> | `POST /system/users/{id}/reset-password` | **403** | 同上 |
> | `POST /system/users/import` | **403** | 同上 |
> | `POST /system/users/sync-managers` | **403** | 同上 |
> | **对照**：同账号临时追加 `xtgl` 后 `POST /system/users/{id}/delete` | **200** | `{"code":0,"message":"success"}` 且**真删除**（软删）⇒ 证明 403 来自权限门而非接口故障 |
>
> ⇒ **#16 后端病灶已闭环**：修复前 8/8 放行（§3.4），修复后 8/8 拒绝（§3.4.2 真实 HTTP + §3.4.1 DB 层模拟）。
> 完整实测记录见 §3.4.2。

> ⚠️ **执行前必须确认 `$TARGET` 是可安全删除的测试用户**——当前缺陷下 delete **会真的删数据**。
> 建议 `$TARGET` 用本次造出来的 `v124_auditor_test` 自身之外的临时用户，或先在 UI 侧新建一个一次性用户再打。

### 5.4 T5–T8 对照判据

| 编号 | 检查项 | 期望（修复后） | **实测（Phase 79 真实 HTTP）** |
|---|---|---|---|
| T5 | 受限账号 `POST /system/users/list` | 200（读操作**应该**放行；注意本夹具**未**绑 `system:user:list`，能通过是因为**读组**仍用 `RequirePermissionsWithQuery`（非严格），模块降级命中 `system:user` 页面本体 —— 这是**刻意保留**的 list→父菜单兼容语义，**不是**病灶 2 在写路径上的残留） | **200** ✅（`data.total=4231`） |
| T6 | 受限账号 `POST /system/users`（新增） | 403 | **403** ✅ |
| T7 | 受限账号 `POST /system/users/{id}/status` | 403 | **403** ✅ |
| T8 | 受限账号 `POST /system/users/batch-delete` | 403 | **403** ✅ |
| T8b（追加） | 受限账号 `POST /system/roles/list`（跨模块隔离） | 403 | **403** ✅ |
| T8c（追加） | 受限账号 `POST /system/users/{id}`（读单个） | 200 | **200** ✅ |

> ⚠️ **data_scope 观察**：`v124_audit_test` 的 `data_scope=3`（本部门），但 `POST /system/users/list` 实测 `total=4231`
> 且返回跨部门用户（admin 等）→ **用户列表的数据范围过滤未按 `data_scope` 生效**。
> 这是**独立于 #16 的另一处发现**（`DataScopePermission` 中间件只把 scope 塞进 context，
> 列表查询侧是否消费待查），建议单独立项，**不在 #16 范围内**。

### 5.5 修复后的回归期望（供主会话对照）

| 角色 | 期望 |
|---|---|
| admin（超管，bypass） | 全部按钮可见，全部接口 200 |
| xtgl（系统管理；**无 export**） | 写按钮可见；导出按钮按现状裁定（给 xtgl 补 export 或 export 接口不加保护） |
| `v124_audit_test`（本夹具） | 写按钮**不可见**；写接口 **403** |

---

## 6. 边界与合规声明

> 本 note 由**两个并行会话**合并而成：① 主会话（脚本备妥 + DB 层模拟取证，§2/§3.4/§3.4.1）；
> ② 本前置准备会话（**真实 HTTP 实测**，§3.4.2/§3.7/§5.1/§5.3/§5.4）。以下声明覆盖两者。

- **未修改任何现有业务数据**：全程只 **INSERT** 新行；**未** 对任何既有 `sys_role` / `sys_user` /
  `sys_role_menu` / `sys_user_role` 执行 UPDATE 或 DELETE。
  - ⚠️ **一处例外（已闭环）**：决定性对照实验（§3.4.2）用**本次造出的测试账号自身**当 delete 目标，
    确实触发了一次软删（`sys_user.deleted_at` 被置位）+ `sys_user_role` 关联行被清空 ——
    **目标 100% 是本次新增的夹具数据，不是任何既有用户**；实验后已**即时恢复**
    （`deleted_at=NULL` + 重新绑定 `v124_audit_test`），夹具当前处于**可用状态**（id 见 §3.1/§3.2）。
- **未挂 admin 角色**：夹具账号只挂新建的 `v124_audit_test`；`role_key != 'admin'` ⇒ 不触发 `isSuperAdmin` bypass。
  对照实验期间临时追加过 `xtgl`（**非 admin**），实验后已移除并复核。
- **未改业务代码**：`internal/**` 与 `ruoyi-react-frontend/src/**` 一行未动；本任务只产出本 note 一个文件。
  （仓库内临时目录 `tmp_pgfixture/` 仅含一次性工具脚本，**用完即删**，见 §9。）
- **凭据零落盘**：数据库口令只经**环境变量**传递，未写入任何仓库文件、未写入提交。
  ⚠️ 收尾时已删除 `%TEMP%\v124-pgpass.txt` / `%TEMP%\v124-pgpass2.txt` 等全部临时凭据文件。
- **临时命名**：角色 `v124_audit_test` / `V1.24审计测试(临时)`，用户 `v124_auditor_test`，便于事后精准清理。
- **本地后端进程已终止**：实测用后端以 `terminal_open` 托管会话运行，验收结束已关闭；端口 9000 已释放。

---

## 7. 复审要点

1. 复跑 §2 的造数 SQL → 确认 `sys_role` 增 1 行、`sys_role_menu` 增 2 行（页面本体 + 父目录）
2. 复核 §3.3 的 perms 集合 —— **不应出现任何 F 型标识**
3. 复核 §5.1 的 T1 返回 —— **不应出现** add/edit/remove/resetPwd/import/export
4. 复核 §5.3 的 T4 返回码 —— **200 = 缺陷坐实（需在 v1.24 修复）**；403 = 已修复
   → **本次实测：403（已修复）** ✅；决定性对照（追加 `xtgl`）= 200 + 真删除 ⇒ 403 是权限门生效
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
| 8 | **JWT SM2 密钥的环境变量名是 `JWT_SM2_*`，不是 `RUOYI_JWT_SM2_*`** | `configs/config.yaml:52` 的注释写的是 `RUOYI_JWT_SM2_PRIVATE_KEY`，但**真实绑定**在 `internal/config/config.go:364-365` = `JWT_SM2_PRIVATE_KEY` / `JWT_SM2_PUBLIC_KEY`。按注释注入会**静默失效**（后端走"动态生成密钥对"分支，公钥与自签 token 不匹配 → 401） |
| 9 | **生产登录通道不可用于夹具验证** | `sys.account.captchaEnabled=slider`（滑动验证码无法脚本识别）+ `sys.auth.default.mode=ad`（默认走 AD bind）⇒ 本地造的口令登不进去。可行路径：启动时注入**固定** SM2 密钥对 + 本地用同一对密钥自签 token（`crypto.GenerateTokenWithSM2`，issuer 必须 `RuoYi-Next`） |
| 10 | **起本地后端连生产库的最小旁路开关** | `SERVER_MODE=test` + `SKIP_AUTOMIGRATE=true`（**release 模式下该开关会 fatal 拒绝启动**，`internal/core/core.go:286-291`）+ `SERVER_SKIP_SETUP=true`（跳过 InitData / 默认角色菜单 seed）+ `DB_PASSWORD`。副作用仍在：scheduler 会跑（AD 同步/工单/对账/MAC 物化视图刷新），会写生产表 |
| 11 | 请求体加密的**真实开关在 DB** | `sys.request.encryption.enabled`（生产实测 `false`）⇒ 明文 JSON 请求直通；`security.request_encryption.require_encryption=false` 亦为兜底。若生产改为 `true`，T4 的 curl 必须改走 SM2+SM4 加密体 |
| 12 | `sys_user` 的列名与直觉不同 | 主键列是 **`id`**（不是 `user_id`）、昵称列是 **`nickname`**（不是 `nick_name`）、另有 `auth_source`（`local`/`ad`，本地认证会拒绝 `ad` 来源账号）——写 SQL 前先 `information_schema` 核列名 |
<!-- V124-PITFALL-BLOCK -->
| 13 | **Step 2「子菜单继承」的语义极易写反** | 真实实现（`git show 51f5e69e^:pkg/middleware/permission.go` 第 2 步，注释「子菜单权限自动包含父菜单权限」）= **用户持有 F 型按钮 → 继承其父页面权限**（查 `perms=?` 的菜单的 **F 型子菜单** 的 perms，看用户是否持有其中之一）。**不是**反向的「持有父页面 → 继承 F 型按钮」。本会话第一版模拟写反了，得出 `step2=true` 的错误结论；改正后 8/8 全 `false`（本夹具一个 F 型都没持有）。§3.8.3 的 SQL 已按正确语义写死 |
| 14 | 关联表有 **UNIQUE 索引**（不止普通索引） | 实测：`sys_role_menu` 有 `sys_role_menu_role_id_menu_id_key (role_id, menu_id)` UNIQUE、`sys_user_role` 有 `sys_user_role_user_id_role_id_key (user_id, role_id)` UNIQUE、`sys_role` 有 `idx_sys_role_role_key(role_key)` 与 `idx_sys_role_role_name(role_name)` UNIQUE ⇒ **重复 INSERT 会报 23505 unique violation**（不只是「多一行」）。`WHERE NOT EXISTS` 幂等仍是必须（否则事务回滚） |
| 15 | `sys_menu` / `sys_role` / `sys_user` **都有 `deleted_at` 软删列** | 查角色/用户/菜单必须带 `deleted_at IS NULL`（GORM 默认软删过滤）。本会话的等价 SQL 全部显式带上，避免读到软删行导致 perms 集合虚高 |
| 16 | T1 端点 `GetUserPermissions` **不过滤 `menu.status`** | `menu_service.go:547-550` 只过滤 `perms IS NOT NULL AND perms != ''`（+ GORM 软删），**不看 status** ⇒ **已停用菜单的 perms 仍会出现在前端权限集**。本夹具 2 条菜单均为 `status=0`，当前无影响；一旦菜单被停用，T1 仍会返回其 perms（前端按钮可能仍显示） |
| 17 | **生产后端当前不可用** | `10.62.10.34:9000` 实测返回 `Server: MinIO`（**是 MinIO 不是后端**）；`http://10.62.10.34/`（nginx/1.18.0）返回 **502 Bad Gateway** ⇒ **生产后端进程未运行**，走生产 HTTP 复测 T1/T4 现在做不到，必须起本地实例或用 §3.8 的只读 SQL |
| 18 | **生产库没有迁移记录表** | `information_schema` 里 `%migration%/%version%/%schema%` 只匹配到 `sys_dashboard_versions`（仪表盘版本，无关）⇒ **无法核对「生产库已应用的迁移版本 vs 仓库最新（`migration_224_*`）」** ⇒ §8 坑位 #6 的前置条件**无法满足** ⇒ 本会话**跳过本地起后端**，改走 §3.8 只读判据 |
| 19 | **并行会话并发改同一夹具 ⇒ 读到中间态快照** | 实测见 §3.9：T1 等价 SQL 一次读到 **35 行**权限（含全部 8 个 F 型 + 横跨 5 个模块），根因是 11:03:50 `sys_user_role` 被另一会话重挂。**取证必须前后各取一次快照**，两次一致才认 |

---

## 9. 对主会话造数工具（`tmp_pgfixture`）的 review 结论

| 项 | 结论 |
|---|---|
| ⚠️ **已更正** | 原判「🔴 阻断」**不成立**：`apply()` 的 `INSERT INTO sys_role_menu (id, role_id, menu_id, created_at)` 与 `ensureUserRole()` 的 `INSERT INTO sys_user_role (id, user_id, role_id, created_at)` 在**生产 PG 下实测成功**——生产库表**实有** `id`/`created_at` 列（Lead 曾据 Go 模型未映射而误推断为「表无该列」，详见 §3.6.1）。仅 **sqlite / 按 GORM 模型新建的库**才会报 `column "id" does not exist`。仍建议用 `WHERE NOT EXISTS` 幂等（`sys_role_menu` 只有普通索引、非 unique） |
| ✅ 通过 | 事务包裹；`NOT EXISTS` 幂等；F 型计数校验；重复行校验；递归 CTE 取 `system:user` 祖先链且只保留 `C`/`M`；`cleanup` 按 `role_id` / `username` 精确定位、不误删既有数据 |
| 🟡 小瑕疵 | ① `data_scope` 注释（"user = 1 本人"）与代码值 `3` 不符（`models/base.go`：3=本部门数据、5=仅本人）——**代码值 3 正确**，改注释即可；② `mkuser` 未开事务，`INSERT sys_user` 成功但后续失败会留半成品；③ `cleanup` 只校验角色残留，建议补 `sys_role_menu` / `sys_user_role` / `sys_user` 三表残留校验 |

#### 9.1 ⚠️ 对上表「🔴 阻断」项的实测修订（生产库口径）

上表「🔴 阻断」结论基于 **GORM 模型**（`models/role.go:23-28` 的 `RoleMenu{RoleID, MenuID}` 无 `id` 列）。
**生产库实际 schema 与模型不一致**——`information_schema` 实测（详见 §3.7）：

- `sys_role_menu` 实有 `id uuid NOT NULL DEFAULT gen_random_uuid()` + `created_at timestamp`
- `sys_user_role` 实有 `id uuid NOT NULL DEFAULT gen_random_uuid()` + `created_at timestamp`

⇒ 工具里 `INSERT INTO sys_role_menu (id, role_id, menu_id, created_at)` 与
`INSERT INTO sys_user_role (id, user_id, role_id, created_at)` **在生产库执行成功**（本次造数零报错）。
结论修正为：

| 场景 | 带 `id` 的 INSERT |
|---|---|
| **生产 PG（本次实测）** | ✅ **成功**（列存在且有默认值） |
| sqlite / 按 GORM 模型新建的库 | ❌ 报错 `column "id" does not exist`（模型无该列） |

> 工具位于**仓库内** `tmp_pgfixture/`（纯一次性脚本，无业务逻辑），**验收结束后必须
> `Remove-Item -Recurse -Force tmp_pgfixture` 删除，禁止提交**。
> ⚠️ 提示：并行会话中途曾删除该目录（清理动作），导致工具被重建过一次——**最终收尾务必确认目录已删**。

