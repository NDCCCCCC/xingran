# Phase 84 PG 实机人工验证清单（PMODEL-01 / PMODEL-04）

> **状态**：⏳ 待用户执行（本机无 PG 实例、无 `psql` 客户端，无法代跑）
> **产出回填位置**：`.planning/phases/84-pmodel-gaps/84-VERIFICATION.md` 的「遗留（人工验证项，不阻塞 phase 完成）」表 + `status` 由 `human_needed` 改为 `passed`
> **依据**：`.planning/phases/84-pmodel-gaps/84-VALIDATION.md` § Manual-Only Verifications
> **性质**：**生产库只读 + 单事务夹具清理**；既有行零改动；夹具必须在验收后立即删除并复核

---

## 0. 为什么必须人工做（不可代跑的原因）

| 项 | 原因 |
|---|---|
| PMODEL-01 | 需要一个**真实可登录**的 `data_scope=3` 账号触发应用层过滤（`pkg/middleware/permission.go` 从 DB 读角色 `data_scope` → `internal/api/v1/system/user_handler.go` 抽参 → `internal/services/system/user_service.go:581 applyUserDataScope` → `List` 在 **Count 之前** 插入过滤 → total 同步）。纯 SQL 对照能验 SQL，不能验中间件→handler→service→Count 的整链路 |
| PMODEL-04 | 需要 `\d` 实查**生产库实际列集**与 GORM 模型对照（迁移记录表 + `sys_role_menu` / `sys_user_role` 漂移） |

本机实测结论：5432 / 6379 端口无监听、`psql` 不在 PATH、仓库根无 `.env` 可读（安全网拦截）→ **无法在本机完成**。

---

## 1. 关键口径（读代码得出，勿凭直觉）

### 1.1 `data_scope` 枚举（`internal/models/base.go:72-78`）

| 值 | 含义 | 用户列表过滤 SQL（`user_service.go:581`） |
|---|---|---|
| 1 | 全部数据 | 不过滤 |
| 2 | 自定义 | `dept_id IN (SELECT rd.dept_id FROM sys_role_dept rd JOIN sys_user_role ur ON ur.role_id = rd.role_id WHERE ur.user_id = ?)` |
| **3** | **本部门** | `dept_id = (SELECT dept_id FROM sys_user WHERE id = ?)` |
| 4 | 本部门及子部门 | `dept_id IN (SELECT id FROM sys_dept WHERE id = ? OR ancestors = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors LIKE ?)`（5 条件，深树正确） |
| 5 | 仅本人 | `sys_user.id = ?`（按**主键**，非 `dept_id = userID` 的错位形态） |
| 其他 | 未知枚举 | **fail-closed** → `1=0` |

### 1.2 两条易踩的坑

1. **多角色取 MAX**：`pkg/middleware/permission.go:530` 用 `SELECT COALESCE(MAX(r.data_scope), ?)` 取用户所有启用角色中**最大**的 `data_scope` 值。因为 1 最宽、5 最窄，MAX = **取最窄的那个** → 夹具角色若与其他角色并存，会被最窄角色决定。**验证时该用户必须只挂一个角色**，否则结论会被污染。
2. **权限判定有 30s TTL 缓存**且 `InvalidateMiddlewarePermCache` 未接线 → 改完角色后**必须等 ≥30s**再测，否则中间件读的是旧 `data_scope`。

### 1.3 模型侧已知漂移点（PMODEL-04 的靶子）

| 表 | GORM 模型 | 字段 |
|---|---|---|
| `sys_user_role` | `internal/models/user.go:51` `UserRole` | `user_id`(uuid) / `role_id`(uuid) / **`created_at`** |
| `sys_role_menu` | `internal/models/role.go:24` `RoleMenu` | `role_id`(uuid) / `menu_id`(uuid) / **无 `created_at`** |
| `sys_role_dept` | `internal/models/role.go:30` `RoleDept` | `role_id`(uuid) / `dept_id`(uuid) / **无 `created_at`** |

> `sys_role_menu` 无 `created_at` 而 `sys_user_role` 有 —— 若生产库两表列集不一致或与模型不一致，即为 PMODEL-04 要登记的**显式漂移**。

---

## 2. 执行步骤

### 步骤 A — 基线快照（只读，先做）

```sql
-- A1. 总人数（admin 口径，用于对照）
SELECT count(*) FROM sys_user WHERE deleted_at IS NULL;

-- A2. 是否已有 data_scope=3 的角色（有则优先复用，避免建夹具）
SELECT id, role_name, role_key, data_scope, status FROM sys_role
WHERE data_scope = 3 AND deleted_at IS NULL;

-- A3. 按部门人数分布（选一个「人数少且明确」的部门做断言锚点）
SELECT d.id, d.dept_name, count(u.id) AS user_cnt
FROM sys_dept d
LEFT JOIN sys_user u ON u.dept_id = d.id AND u.deleted_at IS NULL
WHERE d.deleted_at IS NULL
GROUP BY d.id, d.dept_name
ORDER BY user_cnt ASC
LIMIT 10;

-- A4. 列集实查（PMODEL-04 核心）
\d sys_role_menu
\d sys_user_role
\d sys_role_dept
\d sys_schema_migrations   -- 若不存在 → 迁移记录表未落库，属 PMODEL-04 缺口
```

**判据**：
- A2 有结果 → **走步骤 B-0（复用路径）**，无需建夹具角色
- A4 三张关联表列集与 §1.3 模型一致 → PMODEL-04 库侧无漂移；不一致 → 登记为**显式漂移**（不自动改库，记入 84-VERIFICATION 遗留表）

### 步骤 B — 夹具准备（二选一）

#### B-0 复用路径（推荐，零写入）

从 A2 挑一个 `data_scope=3` 的角色 → 找它已关联的用户 → **用户用该账号登录** → 直接跳步骤 C。

#### B-1 建夹具路径（需写库，验收后必须清理）

```sql
BEGIN;

-- B1. 建一个 data_scope=3 的角色
INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, del_flag, created_at, updated_at)
VALUES (gen_random_uuid(), 'tmp-ds3-verify', 'tmp_ds3_verify', 99, 3, 0, '0', now(), now());

-- B2. 关联到某个「已知部门」的用户（把 :uid 换成 A3 里那个部门的某个用户 id）
INSERT INTO sys_user_role (user_id, role_id, created_at)
SELECT u.id, r.id, now()
FROM sys_user u, sys_role r
WHERE r.role_key = 'tmp_ds3_verify'
  AND u.dept_id = '<目标部门 dept_id>'::uuid
  AND u.deleted_at IS NULL
  AND u.id NOT IN (SELECT user_id FROM sys_user_role)   -- 关键：该用户只挂这一个角色
LIMIT 1;

-- B3. 确认只有这一个角色生效
SELECT u.user_name, u.dept_id, r.role_key, r.data_scope
FROM sys_user u JOIN sys_user_role ur ON ur.user_id = u.id JOIN sys_role r ON r.id = ur.role_id
WHERE r.role_key = 'tmp_ds3_verify';

COMMIT;
```

> ⚠️ **B1/B2 违反的两个约束必须遵守**：
> ① 目标用户**不得已有其他角色**（否则 MAX 取值被污染，结论失真）
> ② **不得修改任何既有行的密码 / dept_id / status** —— 夹具只加关联，不改既有数据

### 步骤 C — 应用层实测

1. 用步骤 B 的账号**登录**（拿到 `accessToken`）
2. **等待 ≥30 秒**（权限判定 30s TTL）
3. 调用用户列表：

```bash
curl -s -X POST "http://<生产后端>/api/v1/system/users/list" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <accessToken>" \
  -d '{"current":1,"pageSize":10}'
```

4. 记录 `data.total` 与 `data.list` 中的 `deptId` 集合

**判据（PMODEL-01 核心）**：

| 断言 | 期望 |
|---|---|
| `data.total` | **= 该用户本部门的用户数**（A3 中该部门 `user_cnt`）—— 修复前是全库 total（v1.24 实测基线 4231），修复后必须变小 |
| `data.list[*].deptId` | 全部 == 该用户的 `dept_id`，**无一例外** |
| `data.total` | ≠ 全库 total（若相等 → 过滤未生效，即 PMODEL-01 未闭环） |

### 步骤 D — admin 对照（零变化证明）

用 admin 账号执行**同一请求**（同 `current`/`pageSize`）：

| 断言 | 期望 |
|---|---|
| `data.total` | **== A1 的全库 total**（admin / `data_scope=1` 行为零变化） |
| 与步骤 C 的 `total` 对比 | admin 的 total **严格大于**受限账号的 total（若相等 → 受限账号未被真正过滤） |

### 步骤 E — 决定性对照（可选但推荐，仿 v1.24 #79 范式）

把夹具角色的 `data_scope` 由 3 改成 1 → 等 30s → 重跑步骤 C：

- `data.total` 从「本部门人数」**跳到全库 total** → 证明 total 的变化**由 `data_scope` 单一变量驱动**，排除「碰巧部门人数等于全库」等伪通过

### 步骤 F — 清理（不可跳过）

```sql
-- F1. 删夹具关联
DELETE FROM sys_user_role WHERE role_id = (SELECT id FROM sys_role WHERE role_key = 'tmp_ds3_verify');
-- F2. 删夹具角色
DELETE FROM sys_role WHERE role_key = 'tmp_ds3_verify';
-- F3. 复核既有行零改动
SELECT count(*) FROM sys_user WHERE deleted_at IS NULL;   -- 必须 == A1
```

**清理判据**：F3 的 count **必须等于** A1 的基线；夹具角色与关联**必须为 0 行**。

---

## 3. 回填

把实测结果填回 `.planning/phases/84-pmodel-gaps/84-VERIFICATION.md`：

1. 「遗留（人工验证项，不阻塞 phase 完成）」表 —— 两行状态从 `⏳ 待用户执行` 改为 `✅ 已验证` 并附实测数字
2. frontmatter `status: human_needed` → `status: passed`（闭合集合，非法值会让 GSD CLI 硬失败）
3. 若发现列集漂移 → 在同文件新增「显式漂移登记」小节（表名 / 模型期望列 / 实际列 / 差异 / 处置建议）

---

## 4. 风险提示

| 风险 | 缓解 |
|---|---|
| 夹具污染生产库 | 步骤 F 强制清理 + F3 复核；夹具一律带 `tmp_` 前缀便于识别与二次清扫 |
| 多角色污染 MAX 取值 | B1 的 `NOT IN (SELECT user_id FROM sys_user_role)` 已排除；B3 二次确认 |
| 30s TTL 导致假失败 | 步骤 C 明确要求等待 ≥30s；假红时先确认等待时长再判失败 |
| 用户列表接口的写组/读组权限 | 受限账号还需持 `system:user:list` 权限（读组 ANY-of），否则会是 **403 而非 total 变小**；403 = 权限不足，不是过滤失败，需另行补菜单权限 |
