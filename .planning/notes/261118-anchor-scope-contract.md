# 机构层级数据权限 —— 开发契约（唯一事实来源）

> 本文件是本次开发的**接口契约**。所有参与开发/测试的成员以此为准，不得自行改变语义。
> 设计背景见 `.planning/notes/261117-org-admin-permission-design.md`（v3 章）。
> 目标场景：三层机构「分公司 → 中支 → 支公司」，要求**一个角色/岗位覆盖某一层的所有机构**自动锚定。

---

## 一、强制约束（违反即为错误）

| # | 约束 | 理由 |
|---|---|---|
| C1 | **`MAX(r.data_scope)` 一行不改**；锚点维度**绝不参与** MAX 排序 | `DataScope` 枚举数值越大越窄，新增维度入 MAX 会把 `All(1)` 用户静默收窄 → 静默降权事故 |
| C2 | 锚点维度与角色维度**取并集（宽者胜）**；`All` 优先 | 纯增量，绝不收窄既有用户 |
| C3 | 锚点解析失败 → **fail-closed（空集）**，**绝不回退到 `dept_id`** | 回退会产生"以为管全中支、实际只看自己部门"的静默错范围 |
| C4 | `pkg/datascope` 的 `ScopeResult` 结构**不变**（5 字段） | 各域 `scope_filter.go` / `vm_data_scope_filter.go` 零改动 |
| C5 | 子树上展开**只复用** `queryDeptChildIDs` 已有实现 | 单一真实源，禁止第二套 ancestors LIKE |
| C6 | 状态语义 `0=正常/启用`，`1=停用`；本功能所有新枚举 **0=不参与/非机构** | 全局约定 |
| C7 | 双方言：所有 uuid/varchar 比较必须 `CAST(... AS TEXT)` | PG/SQLite 一致，仓库既有惯例 |

---

## 二、新增数据模型（迁移 252）

### 2.1 `sys_dept.org_level`（int，default 0）

```go
const (
    OrgLevelNone    = 0 // 非机构（普通部门/科室）—— 默认，存量零回归
    OrgLevelBranch  = 1 // 分公司   （如 湖北分公司）
    OrgLevelSubBang = 2 // 中支     （如 武汉中支）
    OrgLevelOffice  = 3 // 支公司   （如 硚口支公司）
    OrgLevelSite    = 4 // 网点/班组（预留）
)
```

### 2.2 `sys_role` / `sys_post` 各加两列

```go
AnchorMode  int    // 0=不参与（默认）
AnchorParam string // 模式参数，多态
```

`AnchorMode` 枚举（**只实现 4 个值**；编号保留 v3 设计的原始值以便追溯，1/4 已论证弃用，不实现也不接受）：

| 值 | 名称 | `AnchorParam` 含义 | 解析规则 |
|---|---|---|---|
| `0` | `AnchorModeNone` | 忽略 | 不产生任何锚点 |
| `2` | `AnchorModeDept` | `dept_id`（UUID 字符串） | 锚点 = 该部门；展开其子树 |
| `3` | `AnchorModeExternalOrg` | 忽略 | 沿 `ancestors` **由近到远**找第一个 `is_external_org=1` 的节点 |
| `5` | `AnchorModeLevel` | 层级整数（`"1"`~`"4"`） | 沿 `ancestors` **由近到远**找第一个 `org_level == param` 的节点 |

> `1`（相对层深）与 `4`（`sys_manager_org`）已弃用：前者被模式 5 取代，后者依赖未治理的机构表。**保存时若收到 1 或 4 必须拒绝**。

---

## 三、锚点解析（核心算法，`pkg/datascope`）

### 3.1 辅助查询（均带 `CAST(... AS TEXT)`）

- `queryUserDeptFull(ctx, db, userID)` → `(deptID string, ancestors string, ok bool)`
  - 查 `sys_user.dept_id`，再查 `sys_dept.ancestors`（`ancestors` 为根→父序、逗号分隔、**无首尾逗号**）
- `queryDeptLevel(ctx, db, deptID)` → `int`（`sys_dept.org_level`，取不到返回 0）
- `queryDeptIsExternal(ctx, db, deptID)` → `bool`（`is_external_org = 1`）

### 3.2 锚点解析（`resolveAnchor`）

```
resolveAnchor(ctx, db, userID, mode, param) -> (anchorDeptID string, ok bool)
  1) (deptID, ancestors, ok) = queryUserDeptFull(userID);  !ok -> (false)
  2) switch mode:
     case AnchorModeDept:
         anchor = param（必须是非空 UUID）；不存在/已软删 -> fail
         ***注意：显式锚点不要求 anchor 是用户祖先，允许跨机构授权***
     case AnchorModeExternalOrg:
         if isExternal(deptID) -> deptID
         else 反转 ancestors 由近到远找第一个 is_external_org=1 的
     case AnchorModeLevel:
         lv = atoi(param); lv<1 || lv>4 -> fail（配置非法）
         if level(deptID) == lv -> deptID
         else 反转 ancestors 由近到远找第一个 org_level == lv 的
  3) 未命中 -> (false)
```

**关键**：`ancestors` 是根→父序，**必须反转**才能实现"由近到远"。禁止对 `ancestors` 做 LIKE 匹配来找"最近的"——LIKE 无法表达"最近的"。

### 3.3 汇总所有来源（`ResolveAnchorGrants`）

```go
// ResolveAnchorGrants 汇总当前用户所有启用的机构维度授权，返回展开后的部门 ID 集（已去重排序）。
// 无任何来源 / 全部未命中 -> 返回空集（nil），调用方按 fail-closed 处理。
func ResolveAnchorGrants(ctx context.Context, db *gorm.DB, userID string) ([]string, error)
```

来源 SQL（两条，**仅启用角色/启用岗位**）：
- 角色：`sys_user_role ⋈ sys_role` where `r.status=0 AND r.anchor_mode<>0`
- 岗位：`sys_user_post ⋈ sys_post` where `p.status=0 AND p.anchor_mode<>0`

对每个 `(mode, param)` 调 `resolveAnchor`，命中则 `queryDeptChildIDs(anchor)` 展开，全部并入结果集 → `normalizeIDs`。

**性能**：单用户通常 0~3 个来源，`queryDeptChildIDs` 单条 SQL。不做进程内缓存（与中间件 30s 权限缓存叠加会产生陈旧窗口，见 §六）。

### 3.4 合并（`Merge`）

```go
// Merge 把机构维度并入角色维度。纯函数，无 IO。
func Merge(roleScope ScopeResult, anchorDeptIDs []string) ScopeResult
```

规则（**严格按序**）：
1. `roleScope.All` → 原样返回（All 优先，超管零回归）
2. `roleScope.FailClosed` 且无锚点 → 原样返回（异常枚举仍 fail-closed）
3. `eff = normalizeIDs(∪(roleScope.DeptIDs, anchorDeptIDs))`；`len(eff) > 0` →
   `ScopeResult{DeptIDs: eff, UserID: roleScope.UserID}`（`Hook=None`, `FailClosed=false`, `All=false`）
   - ⚠️ 落此分支时，若 `roleScope.Hook == HookSelfUser`，**以机构维度胜出**（显式授予的更大范围覆盖"仅本人"）
4. 其余（`eff` 空 且 `Hook==HookSelfUser`）→ 保留 Self
5. 其余（`eff` 空 且 无 Hook）→ `FailClosedScope`

---

## 四、中间件接线（`pkg/middleware/permission.go`）

`DataScopePermission` 中，在现有 `datascope.Resolve(...)` **之后**追加：

```go
anchorIDs, anchorErr := datascope.ResolveAnchorGrants(c.Request.Context(), core.DB.GetDB(), userID)
if anchorErr != nil {
    // 锚点解析失败按"无锚点"处理，但必须告警；角色维度原样保留（不因新增维度收窄）
    applogger.Errorf("解析机构锚点失败（user_id=%s）: %v", userID, anchorErr)
    anchorIDs = nil
}
scopeResult = datascope.Merge(scopeResult, anchorIDs)
injectScopeRequestContext(c, scopeResult)
```

**`Resolve` 调用与 `getUserMaxDataScope` 一行不改。**

---

## 五、配置期校验（必须实现，防静默）

```go
// ValidateAnchor 校验 anchor_mode/anchor_param 的合法性（保存角色/岗位时调用）。
// 返回 error 时 handler 必须 400 拒绝保存。
func ValidateAnchor(ctx context.Context, db *gorm.DB, mode int, param string) error
```

校验项：
1. `mode` 必须 ∈ {0,2,3,5}；1/4 → `"数据范围锚点模式非法"`
2. `mode=0` → 忽略 param（允许任意）
3. `mode=2` → param 非空且为合法 UUID，且 `sys_dept` 中**存在未软删**该部门
4. `mode=3` → param 必须为空；且 `sys_dept` 中 `is_external_org=1 AND deleted_at IS NULL` 的**计数必须 > 0**（否则拒绝：`"组织架构中未标记任何外部机构，该模式无法解析"`）
5. `mode=5` → param 必须是 `"1"`~`"4"` 整数；且 `sys_dept` 中 `org_level=param AND deleted_at IS NULL` 的计数必须 > 0（否则拒绝：`"组织架构中不存在层级为 N 的机构"`）

---

## 六、推导预览（防静默第 2 道防线，前端依赖）

```
POST /system/posts/:id/anchor-preview     （需 system:post:view）
POST /system/roles/:id/anchor-preview     （需 system:role:view）
```

响应：
```json
{
  "mode": 5, "param": "2",
  "resolvedCount": 9, "unresolvedCount": 3, "totalCount": 12,
  "unresolved": [{ "userId": "...", "username": "zhangsan", "nickname": "张三", "reason": "该用户在部门链中无层级为 2 的机构" }],
  "warnings": ["岗位「X」在目标机构内存在 2 名任职人"]  // 可选
}
```

实现要点：
- 取该角色/岗位的**全部任职人**（`sys_user_role` / `sys_user_post`，不看 status）
- 逐个调 `resolveAnchor` + 记录命中/未命中及原因
- 不展开子树（只看锚点是否命中），避免 N 次子树查询

---

## 七、写侧范围（互斥，禁止越界）

| 成员 | 写范围（workspace-relative） |
|---|---|
| **lead** | `internal/models/base.go`、`internal/models/dept.go`、`internal/models/role.go`、`internal/models/post.go`、`pkg/datascope/**`、`pkg/middleware/permission.go`、`internal/core/db/migrations/migration_252_*.go`、`internal/core/db/database.go`、`.planning/notes/261117-org-admin-permission-design.md` |
| **backend-crud** | `internal/services/system/{department_service,role_service,post_service,user_service}.go`、`internal/models/system/requests/{department_requests,role_requests,post_requests,user_requests}.go`、`internal/api/v1/system/{department,role,post}_handler.go`、`internal/api/v1/system/{department,role,post}_router.go`、`internal/api/router.go`（仅新增预览路由挂载） |
| **frontend** | `xingran-react-frontend/src/pages/system/{dept,role,post,user}/**`、`xingran-react-frontend/src/types/system.ts`、`xingran-react-frontend/src/lib/**`（仅新增 API 函数） |
| **verifier** | 只读 + `**/*_test.go`（不得改实现） |

**任何人不得修改自己范围外的文件**；需要跨界时先告知 lead。

---

## 八、既有约定（务必遵守）

- Handler-Service 模式；Service 返回 `error`，Handler 转 `response.Error(...)`
- 响应统一 `response.Success(c, data)` / `response.Error(c, code, msg)`，禁止 `c.JSON()`
- 写操作 success path 末尾、`response.Success` **之前**调 `operlog.Record(...)`；敏感端点用 `RecordWithBody`
- 分页 `current` 从 1 开始，响应含 `list/total/current/pageSize`
- 前端用 `@/lib/api` 的 `post(...)` 包装，禁止裸 axios；类型放 `src/types/system.ts`
- 迁移：`internal/core/db/migrations/migration_252_*.go` + 在 `database.go` 用 `d.runAndRecord(252, "...", ...)` 注册，**非阻断**（失败只告警不中断启动）
- Go 改完必须 `go build ./...` 通过；前端改完 `npm run type-check` + `npm run lint` 通过
- 构建产物一律 `out/`，禁止落仓库根

---

## 九、验收标准（客观可判）

1. `go build ./...` 通过；`go test ./pkg/datascope/ ./pkg/middleware/ ./internal/services/system/` 全绿
2. `npm run type-check` 与 `npm run lint` 通过（前端）
3. 以下用例必须有测试覆盖且通过：
   - 刘洋（`dept=综合管理部`, `ancestors=…,武汉中支`）持 `anchor_mode=5,param="2"` 的岗位 → 可见集含武汉中支及其所有下级（含硚口支公司）
   - 同上但 `param="3"`（支公司）→ **锚点维度**未命中（空集、不回退 `dept_id`）；**合并结果**以角色维度为准：若角色维度也为空则 `FailClosed` 空集，若角色维度非空则保留角色维度且**不** `FailClosed`（契约 C2 纯增量）
   - `All(1)` 角色用户 + 任一锚点 → 结果仍 `All=true`（C1 回归）
   - `Self(5)` 角色用户 + 有效锚点 → 结果 `Hook=None` 且 DeptIDs 非空（机构维度胜出）
   - 无任何锚点来源 → 合并结果与 `Resolve` 原输出**逐字段相等**（C2 零回归）
   - `anchor_mode=1` 或 `4` → `ValidateAnchor` 返回 error
   - **（安全，v1.54 补丁）** 携带锚点的岗位（`anchor_mode∈{2,3,5}`）在操作者无 `system:post:edit` 时，`CheckPostGrants` 必须 `Allowed=false`；普通岗位不受影响
4. 前端：部门编辑可设 `orgLevel`；角色/岗位表单可设锚点模式与参数；用户表单可分配岗位；岗位/角色详情可见推导预览

---

## 十、修订记录

| 日期 | 变更 | 原因 |
|---|---|---|
| 2026-11-17 | §九 用例 B 措辞精确化（区分「锚点维度空集」与「合并结果是否 FailClosed」） | verifier 发现原措辞会让"按字面写测试"失败；真值表已由 `anchor_test.go` 的 `TestContractCaseB_*` 锁定 |
| 2026-11-17 | 新增 §九 第 7 条安全用例 | verifier 报出阻塞漏洞 P-1（低权限者可自助挂载携带锚点的岗位 → 提权至全省）；已补 `anchor_grant_guard.go` + `anchor_grant_test.go` |
| 2026-11-17 | §五 第 3 条「合法 UUID」裁定为**不强制** | 生产 PG 的 `sys_dept.id` 是原生 uuid 类型，非 UUID 串在 SQL 层即被拒；前端已有 UUID 正则；为 SQLite 夹具加形状校验会误伤全部短 ID 夹具而收益极低。真正的防线是「部门必须存在」 |
