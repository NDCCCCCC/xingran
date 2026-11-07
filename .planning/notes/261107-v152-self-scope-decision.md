# v1.52 / Phase 130 开工输入裁定：Self 档 ops 域语义与 ScopeProvider 边界

- **裁定日期**：2026-11-07
- **裁定人**：Lead（用户 2026-11-07 授权自主裁定，按工程最优 + fail-closed 原则）
- **状态**：LOCKED —— Phase 130 依此定案，后续 phase 只消费不重新裁定
- **依据**：`.planning/REQUIREMENTS.md` DS-01 / DS-07、`STATE.md` v1.52 硬约束 3、`vm_data_scope_filter.go:87-100`（vdi Self 档按 bound_user_id 按人绑定）

## 1. ScopeProvider 的职责边界（锁定）

ScopeProvider **只统一「部门集展开」三档**，不试图统一 Self/All 的资源语义：

| data_scope | 常量值 | ScopeProvider 行为 | 调用方职责 |
|---|---|---|---|
| All / 0 / unset | 1 / 0 / 未注入 | `All=true`，**特判短路**：不查 `sys_dept`、不展开集合，直接放行 | 读端点不加机构过滤；写端点不做归属校验 |
| Custom | 2 | 查 `sys_role_dept ⋈ sys_user_role`（仅启用角色）得部门 ID 集 | 用「有效机构 ∈ 集合」过滤 |
| Dept | 3 | 集合 = `{用户 dept_id}` | 同上 |
| DeptChild | 4 | 集合 = `{用户 dept_id} ∪ 子树`（`ancestors` LIKE / 递归，单一实现） | 同上 |
| Self | 5 | 部门集 = **空集**，并置 `ResourceHook=SelfUser` | 按 §2 的资源级钩子过滤 |
| 异常枚举（非 0 且 ∉ 1..5） | — | **fail-closed 空集**，`All=false`，`ResourceHook=None` | 全部读返回空 / 写 403 |

**关键不变量**：

1. `0 / unset → All` 是**超管零回归**的硬要求（`permission.go:504-507` 超管早退不注入、`user_service.go:583 case DataScopeAll, 0`）。**严禁**把 0 送进 fail-closed 分支。
2. 多角色 `getUserMaxDataScope` 的 `MAX(r.data_scope)` 语义**不变**：MAX = 数值最大 = 最严档（1=All … 5=Self）；无角色 `COALESCE(..., 5)` → Self。边界用例必须覆盖多角色组合与无角色。
3. All 档短路路径**同样要满足缓存 key 含 scope 维度**（PMODEL-01 教训）：短路是「不查部门集」，不是「跳过 scope 段」。
4. 部门子树展开只允许一处实现：复用 `internal/services/operations/dept_filter.go:BuildDeptRecursiveFilter` 的四条件 + 双侧 `CAST AS TEXT`，**禁止**内联复制 `getChildDepts`（`pkg/middleware/permission.go:610-622` 无 status 过滤、vdi 版带 status=0，二者均不作为真源）。

## 2. Self 档在 ops 域的语义（锁定）

ops 域（工位/楼层/楼宇/信息点/资产/机房/专线/机房设备）是**空间资源域**，不存在「本人拥有」的天然语义；既不能像通用版 `ApplyDataScope` 那样把 `deptField = userID`（`permission.go:604` 自认 id 空间错位），也不能降级为 Dept 或回退 All（违反 fail-closed）。

**裁定：Self 档按「资源归属链上的用户列」过滤，无用户列者返回空集。**

| 实体 | Self 档过滤依据 | 落地 phase |
|---|---|---|
| workstation | `sys_workstation.user_id = 登录人 UUID`（quick-261106-wsd 已有列） | 131（读）/ 132（写） |
| infopoint | 经 `workstation_id` 链到 workstation 的 `user_id` | 131 / 132 |
| floor / building / asset / serverRoom / dedicatedLine / roomDevice | **无用户归属列 → 空集（fail-closed）** | 131 / 132 / 133 |
| workstation-device / location-alias | 关联表/字典，跟随 workstation 可见集；本身不单独定义 Self 语义 | 133 |

**理由**：

- 与 vdi Self 档同构（`bound_user_id = 登录人`），不新造「按部门伪装的本人」语义；
- 不引入任何新 DB 字段 / 迁移（v1.52 新迁移从 236 起编，本裁定不消耗编号）；
- 无用户列的实体返回空集而非报错，保证读端点行为可预期、写端点统一 403；
- 与 `DS-01`「Self 按资源语义钩子裁定，不纳入部门集展开」完全一致。

**钩子接口形态（Phase 130 交付）**：

```go
type ScopeResult struct {
    All          bool     // All/0/unset 特判短路
    DeptIDs      []string // Custom/Dept/DeptChild 的可见部门集；Self/异常枚举为空
    ResourceHook ResourceHook // HookNone | HookSelfUser
    FailClosed   bool     // 异常枚举
}
type ResourceHook int
const (
    HookNone ResourceHook = iota
    HookSelfUser
)
func Resolve(ctx context.Context, db *gorm.DB, userID string, scope models.DataScope) (ScopeResult, error)
```

各域通过 `ScopeResult` 自行决定注入哪种过滤，**ScopeProvider 不感知 ops 表名**（保持包级单一真实源的纯净性）。

## 3. 超管 / 0 值 / 异常枚举边界用例（Phase 130 必交）

1. 超管（`isSuperAdmin` 早退未注入）→ `Resolve(scope=0)` = `All=true`，读端点零回归；
2. 未挂注入中间件（`data_scope` 缺失，零值 0）→ 同上 = All；
3. `scope=1` → `All=true` 且**不产生任何 DB 查询**（短路可断言）；
4. `scope=6` / `-1` → `FailClosed=true` + `DeptIDs` 空 + `HookNone`；
5. 多角色 `{1,3,5}` → 取 5（最严）→ Self；
6. 无角色 → COALESCE 5 → Self；
7. 仅停用角色参与时 `getUserMaxDataScope` 不计入 → COALESCE 5 → Self。

## 4. 与既有三份实现的关系

- `pkg/middleware/permission.go:540` 通用版 `ApplyDataScope` 与 `:627` `ApplyDataScopeFromContext` **为死实现**（全仓无生产调用方，仅 `permission_74_08_test.go` 使用）→ **Phase 133 删除实现 + 删除死测试**，不迁移、不复活（Self 错位 + 字段白名单只有 `dept_id/id`，覆盖不了 `org_id`）。
- `internal/services/system/user_service.go:569` `applyUserDataScope`（生产调用方）→ **Phase 130 迁移**到 ScopeProvider，保留其 `sys_user.id = ?` 的 Self 语义与「Count 之前过滤」时序（`:450-451`）。
- `internal/services/vdi/vm_data_scope_filter.go`（生产调用方）→ **Phase 133 迁移**，保留 `bound_user_id` Self 语义与 `ApplyBoundUserFilter`；迁移后既有 fail-closed 测试零回归。

## 5. 残余风险登记

- Self 档下 floor/building 等实体返回空集，若生产存在「仅本人数据」角色且需管理楼层，需产品侧另行确认；本裁定以安全优先，不在代码里开口子。
- All 档短路绕过部门集查询，若调用方误把短路当作「跳过缓存 scope 段」，会复现 PMODEL-01 跨范围泄漏 → Phase 131 缓存面清点表必须逐 key 断言。
