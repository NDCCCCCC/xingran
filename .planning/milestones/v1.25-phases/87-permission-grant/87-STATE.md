# Phase 87 状态与口径基线：动作级权限授予闭环

> 记录 Phase 87 的设计决策、播种口径、父解析清单、遗留项与生产边界。
> 状态更新日期：2026-10-30

---

## 一、播种口径（78 = 65 + 7 + 6）

根据权限码三方校验脚本 `xingran-react-frontend/scripts/check-permission-codes.mjs` 中的 `PENDING_SEED_WHITELIST`（共 78 项），经 87-RESEARCH 与用户裁定（2026-10-30，OQ-6），拆分为三个批次：

| 批次 | 条数 | 包含码范围 | 处置与落地计划 | 理由与决策依据 |
|---|---:|---|---|---|
| **第一批（立即播种）** | **65** | system 4 组 23 条 + ops 8 组 16 条 + network 6 组 26 条 | **Plan 87-02 播种（Migration 226）** | 父 C 型菜单定位清晰稳定（`perms + menu_type='C' + status=0`），无业务边界歧义。 |
| **第二批（挂起待确认）** | **7** | `ops:walls:{add,edit,delete}` (3)<br>`ops:doors:{add,edit,delete}` (3)<br>`ops:floor-plan:edit` (1) | **Plan 87-05 补播（Migration 227）** | **用户裁定（2026-10-30，OQ-6）**：推断父菜单为「楼层管理」，但 CAD 平面图编辑器的实际入口页未人工确认。为防树展示与 Step 2 继承错位，先挂起待确认，不阻塞第一批。 |
| **第三批（不重复播种）** | **6** | `ops:manager:rule:{list,edit,delete}` (3)<br>`ops:manager:org:{list,edit,delete}` (3) | **不播种，白名单照常清空** | **OQ-2 / OQ-12 裁定**：`migration_221` 运行期已建立 4 个 F 型按钮 + 2 个 C 型菜单；`list` 本身属于 C 型页面本体权限码，不属于 F 型按钮权限。已通过既有迁移满足 c 类要求。 |
| **合计** | **78** | **65 + 7 + 6 = 78** | 与白名单总数严格对齐 ✅ | 确保 RBAC-06 的漂移防线闭环。 |

---

## 二、第一批 65 条的父解析表

父菜单解析规则统一遵循：`perms = <父码> AND menu_type = 'C' AND status = 0`。
排除了 `status=1` 的停用旧菜单副本，杜绝越权授权。

| # | 动作码清单 | 条数 | 父 C 菜单定位规则 | 父集合大小 | 父菜单名称与说明 |
|---|---|---:|---|---:|---|
| 1 | `system:role:{list,view,add,edit,remove,export}` | 6 | `perms='system:role' AND menu_type='C' AND status=0` | 1 | 角色管理 |
| 2 | `system:menu:{list,view,add,edit,remove}` | 5 | `perms='system:menu' AND menu_type='C' AND status=0` | 1 | 菜单管理 |
| 3 | `system:dept:{list,view,add,edit,remove,import,export}` | 7 | `perms='system:dept' AND menu_type='C' AND status=0` | 1 | 部门管理 |
| 4 | `system:post:{list,view,add,edit,remove}` | 5 | `perms='system:post' AND menu_type='C' AND status=0` | 1 | 岗位管理 |
| 5 | `ops:building:{import,export}` | 2 | `perms='ops:building:list' AND menu_type='C' AND status=0` | 1 | 楼宇管理（过滤掉 status=1 副本） |
| 6 | `ops:floor:{import,export}` | 2 | `perms='ops:floor:list' AND menu_type='C' AND status=0` | 1 | 楼层管理（过滤掉 status=1 副本） |
| 7 | `ops:workstation:{import,export}` | 2 | `perms='ops:workstation:list' AND menu_type='C' AND status=0` | 1 | 工位管理（过滤掉 status=1 副本） |
| 8 | `ops:asset:{import,export}` | 2 | **特例**：`menu_name='资产列表' AND component='operations/assets/index' AND menu_type='C' AND status=0` | 1 | 资产列表（父 perms 在建表时为 NULL） |
| 9 | `ops:infopoint:{import,export}` | 2 | `perms='ops:infopoint:list' AND menu_type='C' AND status=0` | 1 | 信息点管理（过滤掉 status=1 副本） |
| 10 | `ops:serverroom:{import,export}` | 2 | `perms='ops:serverroom:list' AND menu_type='C' AND status=0` | 1 | 机房管理（过滤掉 status=1 副本） |
| 11 | `ops:dedicatedline:{import,export}` | 2 | `perms='ops:dedicatedline:list' AND menu_type='C' AND status=0` | 1 | 专线管理（过滤掉 status=1 副本） |
| 12 | `ops:roomdevice:{import,export}` | 2 | `perms='ops:roomdevice:list' AND menu_type='C' AND status=0` | 1 | 机房设备管理（过滤掉 status=1 副本） |
| 13 | `network:device:{list,view,add,edit,delete,export}` | 6 | `perms='network:device' AND menu_type='C' AND status=0` | 1 | 设备管理 |
| 14 | `network:credential:{list,view,add,edit,delete,export}` | 6 | `perms='network:credential' AND menu_type='C' AND status=0` | 1 | 授权凭证 |
| 15 | `network:template:{list,view,add,edit,delete,export}` | 6 | `perms='network:template' AND menu_type='C' AND status=0` | 1 | 配置模板 |
| 16 | `network:command:{view,export}` | 2 | `perms='network:command' AND menu_type='C' AND status=0` | **2** | **多父集合**：命令分发与配置执行共用码，持有任一父均被授权 |
| 17 | `network:backup:{list,delete,export}` | 3 | `perms='network:backup' AND menu_type='C' AND status=0` | 1 | 配置备份 |
| 18 | `network:discovery:{view,delete,export}` | 3 | `perms='network:discovery' AND menu_type='C' AND status=0` | 1 | 设备发现 |
| | **合计** | **65** | | | |

---

## 三、批次二（7 条）确认与补播登记

- **码清单**：
  1. `ops:walls:add`
  2. `ops:walls:edit`
  3. `ops:walls:delete`
  4. `ops:doors:add`
  5. `ops:doors:edit`
  6. `ops:doors:delete`
  7. `ops:floor-plan:edit`
- **父菜单确认结论（OQ-6）**：
  - **用户确认日期**：2026-10-30
  - **用户确认结论**：**全部挂在「楼层管理」**（`perms='ops:floor:list' AND menu_type='C' AND status=0`）。CAD 平面图编辑器属于楼层管理页面的子功能模块，7 个动作码统一挂载在楼层管理菜单之下。
- **落地载体**：由 **Plan 87-05** 的 **`migration_227_action_permission_f_type_seed_hold.go`** 独立执行补播，保证审计与回溯清晰独立。
- **校验脚本与白名单收口**：补播完成后，`check-permission-codes.mjs` 中的 `PENDING_SEED_HOLD_CODES` 与 `PENDING_SEED_WHITELIST` 同时清空归零，激活规则 B（白名单必须归零）严格态。

---

## 四、批次三（6 条）不播登记

- **码清单**：
  1. `ops:manager:rule:list`
  2. `ops:manager:rule:edit`
  3. `ops:manager:rule:delete`
  4. `ops:manager:org:list`
  5. `ops:manager:org:edit`
  6. `ops:manager:org:delete`
- **理由**：
  - `migration_221` 在启动运行期已建立 4 个 F 行（`rule:edit/delete`, `org:edit/delete`）和 2 个 C 菜单（`rule:list`, `org:list`）。
  - `list` 权限码是 **C 型页面本身的权限**，绝不能作为 F 型按钮播种进 `F_TYPE_SEED` 块（否则 d 类判定会误将其视作缺失 TS 声明的按钮）。
- **白名单处置**：Plan 87-03 中将这 6 项从 `PENDING_SEED_WHITELIST` 中直接移除，因为 Go migration 既有字面量已足够满足 c 类存在性要求。

---

## 五、86 线遗留移交登记（87 不处理）

根据 86-STATE 和 87-RESEARCH 分析，下列遗留项属于独立后续事项，Phase 87 **不予处理**：

1. **51 条门控豁免**：
   - Phase 87 **消解 0 条**。
   - `read-only 12` + `non-route-page 8` = 永久 N/A。
   - `no-permission-code 26` + `legacy-two-source 5` 需要后端动作拆分出码 → 前端补 TS 声明 → 数据库播种。
   - **硬断言保护**：`EXPECTED_EXEMPTION_COUNT=51` / `EXPECTED_PAGE_COUNT=76` **不得改动**（恒等式 `25 + 51 = 76` 强校验）。
2. **6 处旧式直判（第二真相源）**：
   - 仅 `network:port:write` 同时具备后端常量和 F 行。
   - 其余 5 处后端常量缺失或码名不一致，87 无法直接消解。
3. **OQ-7 独立立项（读路由恒 403 缺口）**：
   - `ops:floor-plan:list`
   - `ops:floor-plan:query`
   - `ops:floor-plan:save`
   - `ops:walls:list`
   - `ops:doors:list`
   - 上述 5 个码在 TS 未声明，也不在白名单中，但被 `internal/api/router.go` 读路由 ANY-of 引用。因无任何菜单行承载，非 admin 角色访问对应读路由时恒 403。
   - **87 显式登记为独立后续项**，不在本 phase 顺手修改，避免越界。

---

## 六、生产影响边界

1. **零 UPDATE / 零 DELETE**：
   - Migration 226 仅执行 `INSERT INTO sys_menu` 与 `INSERT INTO sys_role_menu ... WHERE NOT EXISTS`。
   - 生产既有数据行零修改、零删除，完全符合 SC3 规范。
2. **admin 角色间接变更登记**：
   - `pkg/permission/service.go:81-126` 的 `assignAllMenusToAdmin` 在服务初始化 setup 时会自动把全量菜单补入超级管理员。
   - admin 的 `sys_role_menu` 关联表会增加 65 行。因 admin 走 `role_key='admin'` 旁路机制，实际鉴权行为不受影响，但数据行数会有预期内的变化。
3. **前端导航无视觉破相**：
   - 播种的 65 条菜单全部为 `menu_type='F'`（按钮级）。
   - 前端左侧导航树根据 `menu_type IN ('M', 'C')` 过滤，F 型菜单绝不进入导航侧边栏，页面 UI 布局零破坏。
4. **部署顺序约束**：
   - 必须**先上线并执行 migration 226**（完成数据补齐），**后上线前端代码与门禁脚本**。
   - 确保 Strict 写路由放行所依赖的数据库权限记录已经先期就绪。
