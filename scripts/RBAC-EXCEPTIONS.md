# RBAC strict 口径例外登记（RBAC-EXCEPTIONS）

> 适用范围：#87 全域治理口径「写端点逐动作严格权限（`RequireStrictPermissions`
> 单一权限点）」下，仍保留在 ANY-of 读组的**写端点**例外。
>
> 治理依据：`scripts/EXEMPTIONS-POLICY.md`（三要素：reason / 清偿窗口 /
> 到期复核；无理由保留一律禁止）。本文件是该策略在 RBAC 口径上的实例台账。
>
> 本文件只做登记与说明，**不改任何门禁脚本**，也不改 `pkg/permission/config.go`、
> 菜单播种迁移、前端门控（均不在本任务文件所有权内）。

---

## 一、例外条目

### RBAC-EXC-01 · 周期工单模板 4 条 86 新端点 ✅ 已清偿（2026-11-13）

| 字段 | 内容 |
| --- | --- |
| **id** | `RBAC-EXC-01` |
| **端点** | `POST /workorder/periodic/templates/:id/enable`<br>`POST /workorder/periodic/templates/:id/disable`<br>`POST /workorder/periodic/templates/:id/generate`<br>`POST /workorder/periodic/templates/:id/logs` |
| **登记位置** | `internal/api/v1/workorder/workorder_router.go` `SetupPeriodicWorkOrderActionRouter`（4 条挂逐动作 strict 组）；`internal/api/router.go` 的 `periodicActions` 组 |
| **当前挂载** | ✅ 已迁 `RequireStrictPermissions` 严格写组（逐动作单一权限点） |
| **对照** | 同模块模板主体 CRUD（`Create/Update/DeletePeriodic`）在 `SetupPeriodicWorkOrderWriteRouter` 走 strict 写组（逐动作 `ops:workorder:periodic:add/edit/delete`） |
| **清偿结果** | 4 条端点按 #87 逐动作口径全部迁 strict 写组，权限码拆分见下表；播种迁移 `migration_250_workorder_periodic_action_perm_seed.go` 补齐 F 型授权载体 |
| **原 reason** | 见第二节「理由」（历史留档） |
| **原 window（清偿窗口）** | `截止 2027-03-31`（**已于 2026-11-13 提前清偿**） |
| **review（到期复核）** | 见第四节（已执行并通过） |

**清偿后逐动作权限码：**

| 端点 | 权限码 | 语义 |
| --- | --- | --- |
| `POST /workorder/periodic/templates/:id/enable` | `ops:workorder:periodic:enable` | 写（状态变更） |
| `POST /workorder/periodic/templates/:id/disable` | `ops:workorder:periodic:disable` | 写（状态变更） |
| `POST /workorder/periodic/templates/:id/generate` | `ops:workorder:periodic:generate` | 写（生成工单） |
| `POST /workorder/periodic/templates/:id/logs` | `ops:workorder:periodic:logs` | 只读（执行记录，独立码不与写动作混用） |

**清偿落地清单（全链路缺一即回退）：**

1. `pkg/permission/config.go`：新增 4 个权限码常量 + `GetRoutePermissions()` 路由表登记；
2. `internal/core/db/migrations/migration_250_workorder_periodic_action_perm_seed.go`：F 型授权载体播种（父 C 菜单 `ops:workorder:periodic:list`）+ 精准授权给持父菜单角色；
3. `internal/core/db/database.go`：注册 `runAndRecord(250, ...)`；
4. 前端 `src/constants/permissionCodes.ts`：`workorderPeriodic` 域常量；
5. 前端 `src/pages/workorder/periodic/templates/index.tsx`：5 个动作按钮接 `permission` 三态门控；
6. `internal/api/v1/workorder/workorder_router.go`：新增 `SetupPeriodicWorkOrderActionRouter`；
7. `internal/api/router.go`：构建 `periodicActions` 逐动作 strict 组；
8. `internal/api/v1/workorder/rbac_exception_test.go`：断言反转为「已迁 strict 组 / 不在读组」。

---

## 二、理由（为何按任务书豁免、不当次迁 strict）

1. **任务书明文豁免**：#86 恢复的 4 条端点按任务书口径「保持继承组口径」，
   与 #87「写端点逐动作严格权限」全域治理冲突时，本轮按「二选一」选择
   ② 保留例外并书面登记，避免静默保留。任务书未授权改动权限码播种与
   前端门控的所有权文件。
2. **迁移需跨出本任务文件所有权**：方案 ①（迁 strict 写组）必须同时补
   - `pkg/permission/config.go` 新增 `ops:workorder:periodic:enable/disable/generate/logs` 权限码常量；
   - 菜单/权限播种迁移（`internal/core/db/migrations/`）写入新权限点；
   - 前端 `xingran-react-frontend/src/` 对应按钮门控（无新码会全部隐藏）。
   上述文件均在「禁止修改」清单内。按 `EXEMPTIONS-POLICY.md` 第二节第 3 点，
   确需改他人所有权文件才能清偿的，不越界修改，改登记台账 + 明确窗口。
3. **历史事故保护**：这 4 条端点正是 PR #85 误判「废弃」删掉、1.5h 后
   #86 恢复的端点（见 `scripts/FIX-PROTOCOL.md`）。本轮若为了「严格口径」
   再次改动其挂载面而未配齐权限播种/前端门控，会重演「用户可见功能消失」。
   保持 ANY-of 继承组是任务书指定的低风险态。

---

## 三、风险（必须知晓，非「无风险」）

| 风险 | 说明 | 缓解 |
| --- | --- | --- |
| **权限放大** | 仅持 `ops:workorder:periodic:list`（只读）的角色，也能调用 `enable/disable/generate`（状态变更 / 生成工单写操作），违反 #87 最小权限口径 | 例外已登记 + 代码注释引用本文档 + 清偿窗口到期强制复核 |
| **口径不一致** | 同模块模板主体 CRUD 走 strict、4 条动作走 ANY-of，审计 / 路由扫描器会出现两类 permKind | `internal/api/router.go` 与 `workorder_router.go` 注释均标注例外 id，避免被当成「漏挂」顺手改回或误判 |
| **清偿拖延** | 窗口到期若无人推进，例外变成永久技术债 | 第四节复核流程 + 单测 `TestPeriodicActionRoutes_RBACExceptionDeclared` 锁定「文档存在 + 注释引用」，删除文档即测试失败 |

---

## 四、清偿路径与到期复核

**清偿目标**：`RBAC-EXC-01` → 0（4 条端点全部迁 strict 写组，或任务书
明确撤销豁免）。✅ **已于 2026-11-13 完成**，落地清单见第一节表格。原始清偿动作为：

1. `pkg/permission/config.go` 增 4 个动作权限码常量（或复用 `ops:workorder:periodic:edit`
   但需逐动作独立码以符合 #87「逐动作」口径）；
2. 菜单播种迁移写新权限点 + 角色授权数据；
3. 前端按钮门控改为按新码判断；
4. `internal/api/v1/workorder/workorder_router.go` 把 4 条路由从
   `SetupPeriodicWorkOrderRouter` 移出，接入 `SetupPeriodicWorkOrderWriteRouter`
   的 strict 兄弟组（`strictLiteralRouteGroup`）；
5. 删除本条登记，并在「变更记录」留痕。

**到期复核（window `2027-03-31`）**：

- 窗口到期后 **5 个工作日内**，由权限治理专项 owner 复跑：
  - `go test ./internal/api/...`（含 `TestPeriodicActionRoutes_RBACExceptionDeclared`）；
  - `node scripts/scan-routes.cjs` 核对 4 条路由 permKind 是否仍为 `RequirePermissions`；
- 已清偿 → 删除本条登记；未清偿 → 收窄窗口、在变更记录写明阻塞原因并升级为阻塞项；
- 复核结果写回本文件「变更记录」。

---

## 五、代码锚点（注释引用本文档的位置）

- `internal/api/v1/workorder/workorder_router.go` `SetupPeriodicWorkOrderActionRouter`
  函数注释：标注 `RBAC-EXC-01` 已清偿与逐动作权限码清单。
- `internal/api/router.go` `periodicActions` 组注释：标注例外 id 与文档路径。
- `internal/api/v1/workorder/rbac_exception_test.go`：断言文档记录清偿结果、
  4 条端点在 strict 动作组内、不在 ANY-of 读组、且源码注释引用文档路径
  （防「静默回退到读组」）。

---

## 六、变更记录

| 日期 | 变更 | 结果 |
| --- | --- | --- |
| 2026-11-12 | 登记 `RBAC-EXC-01`：#86 恢复的 4 条周期工单动作端点按任务书豁免、保留 ANY-of 读组；补理由 / 风险 / 清偿窗口；代码注释引用本文档 | 例外不再静默保留，可审计、可清偿 |
| 2026-11-13 | **清偿** `RBAC-EXC-01`：4 条动作端点按 #87 逐动作口径迁 `RequireStrictPermissions` 严格写组（enable/disable/generate/logs 各独立码）；同步落地权限码常量、migration_250 F 型播种、前端门控常量与按钮门控；守卫测试断言反转为「已迁 strict 组 / 不在读组」 | 例外归零，权限放大风险消除；非超管角色经 migration_250 授权后功能不消失 |
