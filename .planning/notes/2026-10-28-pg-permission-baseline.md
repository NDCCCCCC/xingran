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
