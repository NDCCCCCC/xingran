# Phase 87 证据台账与验证规范：动作级权限授予闭环

> 记录 Phase 87（RBAC-05 / RBAC-06）的三层证据矩阵、决定性对照实验手册、只读 SQL 等价判据、生产人工验证清单与 OQ-7 独立后续项登记。
> 更新日期：2026-10-30

---

## 一、三层证据矩阵

在 RuoYi-Next 的权限架构中，动作级权限授予闭环涉及数据库（PostgreSQL/SQLite）、中间件（Go Gin Strict 鉴权）以及前端（React Antd 树组件）。三层证据矩阵如下，**各层证据互不替代**：

| 证据层级 | 能证明的能力 | **不能证明**的能力 | 交付与取证载体 |
|---|---|---|---|
| **数据库层（DB）** | 权限点是否在 `sys_menu` 落库为 F 型行；角色菜单关联表 `sys_role_menu` 是否精准建立；停用副本是否被隔离 | HTTP 端点真实返回的状态码（403 或 200） | `migration_226_action_permission_f_type_seed_test.go`（7 项 Sqlite 单元测试）+ 生产只读 SQL 判据 |
| **HTTP 传输层** | Strict 鉴权路由真实放行或拦截；无权时 403 且数据库零变更；有权时 200 且业务操作真生效 | 前端按钮是否在界面正常渲染或可勾选 | 自签 JWT curl 测试 / 生产只读 SQL 等价模拟 |
| **前端组件层** | F 型按钮节点在角色授权树中正常展示且可勾选；提交请求体 `menuIds` 精准携带 F 节点 ID 与父 C 菜单 ID；回读选中态一致 | 后端中间件是否实际放行；数据库生产表是否已存在对应 F 行 | `permission.grant.test.tsx`（Vitest 自动化组件测试：T-1 ~ T-4） |

---

## 二、决定性对照实验（最小步骤）

为彻底排除「接口原本就无需鉴权」或「权限系统整体失效导致放行」的伪修复，设计 6 步决定性对照实验：

1. **构造测试夹具角色**：创建测试角色，`role_key='p87_probe_test'`（**绝对禁止使用 `admin`**），仅关联其父 C 型页面菜单（0 个 F 型按钮）。
2. **断言初始权限集**：查询该角色绑定用户的有效权限码集合，确认**严格不包含**目标动作码（如 `system:role:add`）。
3. **负向探测（严格 403）**：使用该受限用户身份向写路由端点发送请求（如 `POST /system/roles`），断言响应状态码为 **403 Forbidden**（`{"code":403,"message":"没有访问权限"}`，依据 `pkg/middleware/permission.go:434`）。
4. **数据库零变更核对**：执行 `SELECT count(*) FROM sys_role`，断言前后行数完全一致，证明无权请求未对生产数据产生脏写入。
5. **临时授权与缓存冷却**：向 `sys_role_menu` 插入该动作码对应的 F 型按钮 `menu_id`；**必须等待 31s**（因为后端权限缓存 `mwPermCacheTTL = 30s`，`InvalidateMiddlewarePermCache` 未接线）或直接重启后端。
6. **正向探测（严格 200）与撤销复原**：重发同一 HTTP 请求，断言响应状态码为 **200 OK 且真实新增数据**；随后删除该临时授权行，再次等待 31s 后重试，断言状态码**严密回到 403**。

---

## 三、只读 SQL 等价判据（推荐路径，零副作用）

鉴于开发机环境约束（无远程 PG 写入权限、无 API-Key 旁路、生产后端当前处于安全关闭态），推荐采用**只读 SQL 等价判据**。
该判据完全复刻 `pkg/middleware/permission.go:451-457` 的 `checkUserPermissionStrict` 内部判断逻辑，执行时建议显式声明只读事务以保证零副作用：

```sql
SET TRANSACTION READ ONLY;

-- 判定某用户是否持有指定动作权限码（等价于后端 Strict 拦截逻辑）
SELECT COUNT(DISTINCT rm.menu_id) AS strict_exact_pass
FROM sys_user_role ur
INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
INNER JOIN sys_menu m ON rm.menu_id = m.id
WHERE ur.user_id = :target_user_id
  AND m.perms = :target_permission_code
  AND m.status = 0;
```

- **判据结论**：
  - 若 `strict_exact_pass = 0` ⇒ 后端 Strict 中间件**判定 403 拒绝**。
  - 若 `strict_exact_pass > 0` ⇒ 后端 Strict 中间件**判定 200 放行**。
- **读路径对照判据**（排除系统整体失效）：
  对同一用户查询其父 C 菜单的只读权限码（如 `m.perms = 'system:role'`），断言 `strict_exact_pass > 0`，证明读路由完全正常，仅写操作被精准拦截。

---

## 四、夹具规范与清理

1. **命名规范**：
   - 夹具角色与用户标识一律使用 `p87_` 前缀（如 `role_key='p87_probe_role'`，`username='p87_test_user'`）。
   - **绝对禁止撞名 `admin`**（`role_key='admin'` 会命中超管 bypass 逻辑，使鉴权判定失真）。
   - 验证账号**只挂**该测试角色，避免多角色继承产生权限污染。
2. **先子后父清理顺序**：
   清理临时夹具必须在单一事务中按外键层级由子及父顺序清理：
   ```sql
   DELETE FROM sys_user_post WHERE user_id IN (SELECT id FROM sys_user WHERE username LIKE 'p87_%');
   DELETE FROM sys_user_role WHERE role_id IN (SELECT id FROM sys_role WHERE role_key LIKE 'p87_%');
   DELETE FROM sys_user WHERE username LIKE 'p87_%';
   DELETE FROM sys_role_menu WHERE role_id IN (SELECT id FROM sys_role WHERE role_key LIKE 'p87_%');
   DELETE FROM sys_role_dept WHERE role_id IN (SELECT id FROM sys_role WHERE role_key LIKE 'p87_%');
   DELETE FROM sys_role WHERE role_key LIKE 'p87_%';
   ```
3. **清理边界红线**：
   - ⚠️ **播种的 65 条 F 行不在清理范围内**！它们是 Phase 87 的系统交付物，绝非测试夹具。清理脚本**严禁删除** `sys_menu` 中由 Migration 226 建立的新行。

---

## 五、缓存与 UI 验收规则

在实机验证过程中，由于多级缓存机制叠加，必须严格遵守以下失效规则：

1. **后端缓存**：
   - `pkg/middleware/permission.go:33` 定义了 `mwPermCacheTTL = 30 * time.Second`。
   - `InvalidateMiddlewarePermCache` 函数目前未在角色授权接口处接线。
   - **规则**：修改角色权限后，**必须等待 31s** 或重启后端服务，方可断言后端鉴权行为的变化。
2. **前端多级缓存**：
   - 前端权限存储包含 5 分钟 React Query 内存缓存与 30 分钟 `sessionStorage` 菜单缓存（`menu:cache`）。
   - `MENU_CACHE_VERSION=1` 未发生变更，浏览器不会自动清空本地缓存。
   - **规则**：进行 UI 实机验收前，**必须先清空浏览器 sessionStorage 的 `menu:cache` 项，或者重新退出登录**，然后再刷新页面验证按钮与树节点的展示。

---

## 六、并发干扰硬规则

1. **快照一致性校验**：
   在执行任何取证查询前后，各运行一次目标角色的权限码快照；只有当两次快照数值完全一致时，期间采集的证据方为有效。
2. **独占夹具**：
   同一组 `p87_` 夹具严禁被两个并发进程或会话同时调用。
3. **串行执行**：
   所有对照实验步骤与生产取证必须严格串行执行，禁止并发压测。

---

## 七、生产侧人工验证清单

本清单整理了上线部署时需执行的人工核查项，每项包含五要素：

| # | 验证项 | 为何必须人工 | 执行命令 / 判定 SQL | 期望结果 | 落点 |
|---|---|---|---|---|---|
| 1 | **受影响生产角色清单**（生产受影响角色清单） | 生产真实角色仅存在于远程生产 PG 库中，开发机无直接读权限 | 运行 87-RESEARCH §E19 判定 SQL：<br>`SELECT DISTINCT r.id, r.role_name, r.role_key FROM sys_role r JOIN sys_role_menu rm ON r.id = rm.role_id JOIN sys_menu m ON rm.menu_id = m.id WHERE m.menu_type = 'C' AND m.perms IN ('system:role', 'system:menu', 'system:dept', 'system:post', 'network:device', 'network:credential', 'network:template', 'network:command', 'network:backup', 'network:discovery') OR (m.perms = 'ops:building:list' AND m.status = 0) OR (m.perms = 'ops:floor:list' AND m.status = 0) OR (m.perms = 'ops:workstation:list' AND m.status = 0) OR (m.menu_name = '资产列表' AND m.component = 'operations/assets/index' AND m.status = 0) OR (m.perms = 'ops:infopoint:list' AND m.status = 0) OR (m.perms = 'ops:serverroom:list' AND m.status = 0) OR (m.perms = 'ops:dedicatedline:list' AND m.status = 0) OR (m.perms = 'ops:roomdevice:list' AND m.status = 0);` | **逐条列明受影响角色清单**（若仅 admin 则表明业务角色未配置） | `87-VERIFICATION.md` |
| 2 | **生产库既有行零改动核对** | 校验生产既有数据未受任何 UPDATE 或 DELETE 污染（SC5 规范） | 迁移前后比对：<br>`SELECT (SELECT count(*) FROM sys_menu), (SELECT count(*) FROM sys_role_menu), (SELECT count(*) FROM sys_role), (SELECT count(*) FROM sys_user);` | 差值正好为 **+65 菜单行、+N 授权行（含 admin 补全的 65 行）、0 角色变更、0 用户变更** | `87-VERIFICATION.md` |
| 3 | **迁移在生产 PG 上的实际执行** | 生产库运行 PostgreSQL，开发机测试基于 SQLite，需确认 PG 方言落库成功 | `SELECT count(*) FROM sys_menu WHERE menu_type = 'F' AND perms IN (<第一批 65 码清单>);` | 计数精确为 **65** | 生产部署日志 |
| 4 | **`\d sys_role_menu` 结构复核** | 确认生产 PG 上 `sys_role_menu` 表是否存在唯一键约束（OQ-1） | `\d sys_role_menu;` 或 `SELECT conname FROM pg_constraint WHERE conrelid = 'sys_role_menu'::regclass;` | 记录约束名称或确认无唯一键；Migration 226 已用 `NOT EXISTS` 保证双方言兼容 | 部署记录 |
| 5 | **HTTP 实测 403 → 200** | 生产后端部署运行后，验证真实网络请求的状态码翻转 | 采用夹具账号或只读 SQL 判据；或按照对照实验 6 步使用 curl 发送带 Token 请求 | 补授前 403 拦截，补授等待 31s 后 200 放行 | 测试报告 |
| 6 | **前端 UI 实机交互** | 涉及真实浏览器环境、AD 域控认证及 Ant Design 交互表现 | 浏览器访问 `/system/role`，清空 `sessionStorage`，打开编辑角色弹窗查看菜单树 | 65 个动作级按钮在树中正常展示，勾选保存后刷新回读仍保持勾选 | 验收报告 |
| 7 | **生产部署顺序规范** | 防止前端先上线调用未放行路由导致 403 爆发 | 按照 CI/CD 流水线编排部署先后次序 | **必须先执行后端 Migration 226（补齐数据），后上线前端包与门禁脚本** | 部署规范 |

---

## 八、OQ-7 独立后续项登记（读路由恒 403 缺口）

经 87-RESEARCH 与 87-STATE 审查，以下 5 个权限码存在架构层面的读路由恒 403 缺口：

1. `ops:floor-plan:list`
2. `ops:floor-plan:query`
3. `ops:floor-plan:save`
4. `ops:walls:list`
5. `ops:doors:list`

- **现状分析**：
  上述 5 个权限码在前端 `src/constants/permissionCodes.ts` 中**均未声明**，在权限白名单中也**未包含**。但在后端 `internal/api/router.go` 中，CAD 平面图、墙体和门的读路由采用了 `RequireStrictPermissions` 并以 `ANY-of` 方式引用了它们。
  由于数据库 `sys_menu` 中没有任何菜单行承载这些码，除 `admin`（走 bypass 机制）之外的所有普通用户或受限角色，在访问相关读端点时**必然恒为 403 Forbidden**。
- **与第二批（7 条）的关系**：
  Phase 87 第二批挂起的 7 条码均为**写操作码**（`ops:walls:{add,edit,delete}`, `ops:doors:{add,edit,delete}`, `ops:floor-plan:edit`）。即使第二批补播完成，上述 5 个**读路由码**依然处于未声明未播种状态。
- **决策结论**：
  **Phase 87 不予处理**，将其显式登记为**独立后续项**，留待后续 CAD 模块专项重构时统一补全前端 TS 映射、后端路由拆分与菜单播种，避免跨 phase 擅自扩容。

---

## 九、回归基线数字核对

- **ESLint 门禁**：0 errors, 0 warnings（`npm run lint:ci` 零漂移）。
- **前端单元测试**：Vitest 全部测试通过（包含新增的 `permission.grant.test.tsx` 3 个测试用例）。
- **门控覆盖恒等式**：已门控 25 + 豁免 51 = 76（`npm run check:gating` exit 0）。
- **后端构建与测试**：`go build ./...` 成功，`go test ./...` 全量通过（含 Migration 226 的 7 个测试）。
- **路由登记测试**：`go test -run "TestRouteRegistry" ./pkg/permission/` 与 `node scripts/check-route-registry.mjs` 全绿。
