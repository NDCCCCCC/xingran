# 机构层级数据权限 —— 交付说明（v1.54）

> 本文件是**面向使用者与后续维护者的交付说明**，不是设计稿（设计见 `261117-org-admin-permission-design.md`，接口契约见 `261118-anchor-scope-contract.md`，独立核验见 `261118-anchor-scope-verification.md`）。

---

## ⚠ 后续补丁登记（2026-11-17，均已在本文相应章节详述）

| 补丁 | 触发 | 章节 |
|---|---|---|
| SKIP_AUTOMIGRATE 旁路补列（迁移 252 旁路执行） | 生产 PG 报 `column org_level does not exist` | §十 |
| 部门树 Scoped 收口（部门管理页/系统用户页传 `scoped:true`） | 机构管理员在部门列表看到全库 1060 部门 | §十一 |
| 用户列表数据面读 ctx 注入（applyUserDataScope） | 部署后锚点预览命中，但用户列表仍只见自己 | §十二 |
| D 类三处收口（通知/值班池/工单派单） | 用户确认 | §十三 |
| 编辑用户「权限预览」（菜单权限 + 数据范围） | 用户要求 | §十四 |
| 权限预览改为菜单树/部门树勾选态（与编辑角色模态框一致） | 用户 2026-11-18 要求 | §十四补充 |

---

## 一、这个功能解决什么

**问题**：系统里"机构管理员"只能用「每个机构建一个角色」的方式实现。因为数据范围的锚点被硬编码成"用户自己的部门"：

- `本部门/本部门及下级` → 锚点 = 用户自己的 `dept_id`
- `自定义数据` → 锚点 = 角色上手工勾死的部门清单

而角色是全局共享的。武汉中支的刘洋属于「综合管理部」（在武汉中支下），要让他当中支管理员，`本部门及下级` 只能给他「综合管理部」（一个几乎没有下级的科室），永远拿不到中支的其他部门。

**解法**：新增一个**与数据范围正交的「机构锚点」维度**。锚点不再绑死用户自己的部门，而是按配置动态解析：

```
用户 dept 的 ancestors 链（由近到远） → 找到第一个匹配的机构节点 → 锚点
锚点展开子树（复用 queryDeptChildIDs） → 可见部门集
与角色维度取并集 → 最终可见集
```

于是**一个「中支管理员」岗位/角色，分给所有中支的人，各自得到各自的中支**。

---

## 二、四种锚点模式

| 模式 | 名称 | 参数 | 解析方式 | 适用场景 |
|---|---|---|---|---|
| `0` | 不参与 | — | 不产生任何范围 | 默认；存量零回归 |
| `2` | 指定部门 | `dept_id` | 固定锚点，展开其子树。**允许跨机构**（不要求是用户祖先） | 单机构管理员；省公司派驻；兼任 |
| `3` | 外部机构 | — | 由近到远找第一个 `is_external_org=1` 的节点 | 组织树已按"有职场"标过机构时最省事 |
| `5` | 指定机构层级 | `"1"`~`"4"` | 由近到远找第一个 `org_level == 参数` 的节点 | **三层机构的自适应方案**（见下） |

**模式 5 是关键**，因为机构层级有三层：

```
湖北分公司(org_level=1) → 武汉中支(org_level=2) → 硚口支公司(org_level=3) → 综合管理部(org_level=0)
```

- 岗位「中支管理员」=`anchor_mode=5, param="2"`：
  - 刘洋（综合管理部，level 0）→ 上溯命中 **武汉中支** → 可见集含武汉中支 + 硚口支公司
  - 老王（硚口支公司，level 3）→ 跳过自己（level 3 ≠ 2）→ 上溯命中 **武汉中支**
  - 小李（长沙中支）→ 命中 **长沙中支**，与武汉互不干扰
- 岗位「支公司管理员」=`param="3"`：刘洋**未命中**（链上无 level 3）→ fail-closed 空集，这是**正确的**（他不在任何支公司下）

> ⚠️ 三个岗位可以同时给同一个人，可见集取**并集**。

---

## 三、两条铁律（维护时不可违反）

### 铁律 1：锚点维度**绝不参与** `MAX(data_scope)`

`pkg/middleware/permission.go` 的 `getUserMaxDataScope` 用 `MAX(r.data_scope)` 取**数值最大（最窄）**的档。`DataScope` 只有 1..5，锚点维度是**独立字段**（`anchor_mode`），独立解析后合并。

**如果给 `DataScope` 加一个 `6`**，一个同时持有 `All(1)` 角色的用户，MAX 会变成 6 → **从全库静默降为某机构子树**。这是静默降权事故，代码里有多处注释警示。

### 铁律 2：合并**只放宽不收紧**，`All` 优先，解析失败 fail-closed 且**不回退 `dept_id`**

```
Merge(roleScope, anchorDeptIDs):
  1. roleScope.All                     → 原样返回（All 优先）
  2. roleScope.FailClosed 且无锚点      → 原样返回（异常枚举仍 fail-closed）
  3. 并集非空                          → DeptIDs=并集、Hook=None（Self 也被覆盖）
  4. 并集空 且 Hook==SelfUser          → 保留 Self
  5. 其余                              → FailClosed（空集）
```

- **绝不回退到用户 `dept_id`**：回退会造成"以为管了全中支、实际只看到自己部门"的静默错范围，比显式空集危险得多。

---

## 四、数据前置条件（上线必须做）

### 4.1 给机构节点标注 `org_level`

`sys_dept.org_level`：`0=非机构（默认）/ 1=分公司 / 2=中支 / 3=支公司 / 4=网点`。

**路径**：部门管理 → 编辑 → 「机构层级」下拉。

**必须手工校正的特例**：代码注释里明确写着"中支含分公司本部"这类情况纯靠祖先深度推导必然判错（`manager_rule_service.go:1573` 原有注释），故 `org_level` **不做任何自动推导**（迁移 252 只建列，不填数据），必须由人工标注。

**迁移 252 之后的巡检 SQL**（建议上线后跑一次）：

```sql
-- ① 有多少机构还没标层级
SELECT COUNT(*) FROM sys_dept WHERE is_external_org = 1 AND org_level = 0 AND deleted_at IS NULL;
-- ② 层级与父子关系是否矛盾（子比父层级"更高"）
SELECT c.dept_name AS child, c.org_level AS child_level, p.dept_name AS parent, p.org_level AS parent_level
  FROM sys_dept c JOIN sys_dept p ON p.id = c.parent_id
 WHERE c.org_level > 0 AND p.org_level > 0 AND c.org_level < p.org_level
   AND c.deleted_at IS NULL;
-- ③ 支公司悬空（level=3 但祖先链上无 level=2）
--    （用 ancestors 逐段比对，或先看 ② 是否为空再人工抽查）
```

### 4.2 配置期校验会自动挡错

保存角色/岗位时 `datascope.ValidateAnchor` 会拒绝：
- 模式 1/4（弃用编号）、未知模式
- 模式 2：部门不存在
- 模式 3：组织树里**一个外部机构都没标**（否则该模式永远解析不到）
- 模式 5：参数不是 `"1"`~`"4"`，或**该层级在组织树里一个节点都没有**

这是防静默第 1 道防线：**把错误挡在保存时，而不是让人在运行期看到空数据却不知道为什么**。

---

## 五、安全边界（重要）

### 5.1 授予"携带锚点的岗位"= 授予数据范围，需要 `system:post:edit`

**漏洞背景**（verifier 2026-11-17 发现，已修复）：`user_service` 原本直接写 `sys_user_post`，对 `postIds` **零校验**。而岗位维度锚点按任职人自己的部门链解析。于是：

```
省管理员建岗位「分公司管理员」(anchor_mode=5, param="1")
  ↓
任一持 system:user:edit 的中支管理员，在自己的用户表单里把自己挂上去
  ↓
下次请求解析到「湖北分公司」→ 展开全子树 → 可见集 = 全省
```

**一次表单提交即可提权到全省**，且 operlog 只记"用户管理-修改"。

**修复**（`pkg/datascope/anchor_grant.go` + `internal/api/v1/system/anchor_grant_guard.go`）：

| 规则 | 行为 |
|---|---|
| 岗位不存在/已软删 | 拒绝（400） |
| 岗位已停用 | 拒绝（停用岗位不产生任何权限，挂上会误导） |
| 携带锚点 + 操作者无 `system:post:edit` | 拒绝（400，中文原因含岗位名与锚点描述） |
| 携带锚点 + 策略 `AllowAnchorOnlyForAdmin` + 非超管 | 拒绝 |
| 通过且携带锚点 | `applogger.Warnf("[机构锚点授权] operator=… target=… posts=…")` 可检索审计 |

**策略开关**（`anchor_grant_guard.go` 的 `grantAnchorPolicy`）：
- `RequirePostEditPerm: true`（**默认**）— 需要 `system:post:edit`
- `AllowAnchorOnlyForAdmin: true` — 最严，仅超管可授予
- 两者都 false — 只保留存在性/停用校验（适合"岗位分配本就是超管职能"的小规模部署）

### 5.2 `Self(5)` 角色用户被挂锚点岗位后会放宽

按铁律 2 的规则 3，Self 用户获得锚点后可见集变成机构范围。这是**设计意图**（显式授予的更大范围胜出），但意味着"误挂一次岗位就能扩权"。5.1 的准入校验正是这道门。

### 5.3 预览口径与生效口径不同

- **预览**（`POST /system/{roles,posts}/:id/anchor-preview`）：看**全部任职人**（含停用角色的），用于配置面体检。
- **生效**（`queryAnchorGrants`）：只看 `status=0`（正常/启用）的角色与岗位。

故预览的"已解析 N 人"可能大于实际生效人数。前端提示文案已注明这一点。

---

## 六、接口清单

### 6.1 配置字段（随现有 CRUD 传递，无新端点）

| 端点 | 新增字段 |
|---|---|
| `POST /system/departments`、`/system/departments/:id/update` | `orgLevel: int`（0~4） |
| `POST /system/roles`、`/system/roles/:id/update` | `anchorMode: int`（0/2/3/5）、`anchorParam: string` |
| `POST /system/posts`、`/system/posts/:id/update` | `anchorMode: int`、`anchorParam: string` |
| `POST /system/users`、`/system/users/:id/update` | `postIds: string[]`（现新增准入校验） |

### 6.2 推导预览（只读，不记 operlog）

```
POST /system/roles/:id/anchor-preview     权限：system:role:view（读组自动施加）
POST /system/posts/:id/anchor-preview     权限：system:post:view
```

响应：

```json
{
  "mode": 5,
  "param": "2",
  "resolvedCount": 9,
  "unresolvedCount": 3,
  "totalCount": 12,
  "items": [
    { "userId": "...", "username": "liuyang", "nickname": "刘洋",
      "resolved": true, "anchorDeptId": "<武汉中支 dept_id>" },
    { "userId": "...", "username": "zhangsan", "nickname": "张三",
      "resolved": false, "reason": "该用户在部门链中没有指定层级的机构" }
  ],
  "unresolved": [ /* 同上，仅 resolved=false 的子集（冗余字段，前端以 items 为准） */ ]
}
```

### 6.3 用户列表/详情的岗位回填

- `user_service.List` 新增 `fillUserPosts`：**一次** `sys_user_post ⋈ sys_post` 批量查询（不是 `Preload` 的 N+1），只查当前页 ID。
- `user_service.GetByID` 同步回填 `Posts`（P-7 修复）。
- 缺表/异常时 `Posts = []`（不是 `null`），`applogger.Warnf` 留痕，列表主流程不失败。

---

## 七、前端改动位置

| 页面 | 改动 |
|---|---|
| 部门管理 | 编辑弹窗新增「机构层级」下拉；列表新增「机构层级」列（Tag 配色） |
| 角色管理 | 「数据权限」卡片**下方**新增独立 `Card「机构授权（可选）」`，含 `AnchorScopeFields` + 预览面板。**数据范围下拉一行未改** |
| 岗位管理 | 表单新增「机构授权」区域 + 预览；列表新增「机构授权」列 |
| 用户管理 | 新增「岗位」多选（`postIds`，数据源 `/system/posts/enabled`）；列表新增「岗位」列。**保留原「职位」`title` 自由文本**（那是 AD 同步的职位，与岗位字典不是同一概念） |
| 共享组件 | `src/components/system/AnchorScopeFields.tsx`（受控，切模式自动清脏参数）、`AnchorPreviewPanel.tsx`（含后端未上线时的降级提示）、`src/constants/anchor.ts`、`src/lib/systemAnchorApi.ts` |

---

## 八、变更文件总览

**新增**
- `pkg/datascope/anchor.go` — 锚点解析/校验/汇总/合并/预览
- `pkg/datascope/anchor_grant.go` — 岗位授予准入策略（安全补丁）
- `internal/api/v1/system/anchor_grant_guard.go` — 准入校验接线
- `internal/core/db/migrations/migration_252_dept_org_level_and_anchor_scope.go` — 建列
- 前端 4 个文件（见 §七）

**修改**
- `internal/models/{base,dept,role,post}.go` — 枚举与字段
- `pkg/middleware/permission.go` — `DataScopePermission` 追加锚点合并（`Resolve`/`getUserMaxDataScope` **一字未改**）
- DTO × 3、service × 4（department/role/post/user）、handler × 3、router × 2
- `internal/api/v1/system/user_handler.go` — Create/Update 前调 `guardPostGrants`
- `internal/core/db/database.go` — 注册迁移 252
- `scripts/route-registry.json` — 路由门禁重生成（新增 2 条 preview 路由）

**测试**
- `pkg/datascope/anchor_test.go`（原 14 + verifier 补 6）
- `pkg/datascope/anchor_grant_test.go`（9 条准入回归）
- `pkg/middleware/anchor_mount_test.go`（verifier 补 4 条**中间件级**端到端 —— 原测试只验证纯函数，无法证明中间件真的接了 Merge）
- `internal/core/db/migrations/migration_252_dept_org_level_and_anchor_scope_test.go`（幂等/缺表/列存在/默认值）
- 11 个既有测试夹具补列（`org_level` / `anchor_mode` / `anchor_param`）+ 3 个 addomain 夹具补列

---

## 九、验证状态

| 项 | 结果 |
|---|---|
| `go build ./...` | ✅ 0 error |
| `go test ./...` | ✅ 0 FAIL 包 |
| `npm run type-check` / `npm run lint` | ✅ 均 exit 0 |
| 契约 §九 六条用例 | ✅ 6/6（含中间件级端到端） |
| 静态审查 9 项（C1~C7 合规） | ✅ 9/9 |
| 核验结论 | `PASS-with-issues` → 两个阻塞项（P-1 权限提升、P-2 静默缺表）**已修复并补测** |

**未覆盖（透明声明）**
- 未做真实 PostgreSQL 集成/E2E（仅 SQLite 夹具 + 代码审查）
- 未做浏览器实机点击（仅 type-check / lint / 代码审查）→ **建议上线前走一次真实页面流程**：给某中支的科室人员挂「中支管理员」岗位，确认他能看到该中支全部部门

---

## 十、上线后补丁：SKIP_AUTOMIGRATE 旁路补列（2026-11-17 实测踩中）

**现象**：生产 PG 上执行巡检 SQL 报 `column "org_level" does not exist`。

**根因**：该部署走 `SKIP_AUTOMIGRATE=true` 旁路（`database.go:817` 直接 `return nil` 跳过整个 `AutoMigrate()`），而迁移 252 的正常注册点 `runMenuConsolidationMigrations()`（`database.go:1492`）在 `AutoMigrate()` **内部** —— 旁路路径根本不会执行到。五列（`sys_dept.org_level`、`sys_role/sys_post` 的 `anchor_mode/anchor_param`）全部没建。

**危害**：不止锚点功能不可用 —— models 已加新字段，GORM 查询自动 `SELECT` 新列，缺列会让**部门/角色/岗位管理的全部查询**报 42703。与 migration 219 注释描述的同款盲区。

**修复**：`BootstrapMissingTables()`（旁路唯一入口）在 220 之后补 `runAndRecord(252, ...)`，**失败阻断**（与 219 同档）。幂等：`HasColumn` 守卫 + `sys_schema_migrations` 版本记录，旁路执行后正常路径再启动不会重复。

**运维动作（二选一）**：
1. **推荐**：用含本补丁的二进制重启后端 —— 启动日志出现 `Migration 252 completed: org_level + anchor scope columns ready` 即补列成功，再跑巡检 SQL 确认。
2. **应急**（无法立即重启时）手工执行 DDL，效果等价：

```sql
ALTER TABLE sys_dept ADD COLUMN IF NOT EXISTS org_level SMALLINT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_sys_dept_org_level ON sys_dept(org_level);
ALTER TABLE sys_role  ADD COLUMN IF NOT EXISTS anchor_mode  SMALLINT DEFAULT 0;
ALTER TABLE sys_role  ADD COLUMN IF NOT EXISTS anchor_param VARCHAR(64) DEFAULT '';
ALTER TABLE sys_post  ADD COLUMN IF NOT EXISTS anchor_mode  SMALLINT DEFAULT 0;
ALTER TABLE sys_post  ADD COLUMN IF NOT EXISTS anchor_param VARCHAR(64) DEFAULT '';
```

> 注意：手工 DDL 后 `sys_schema_migrations` 没有 252 的版本记录，下次正常路径启动时迁移会因 `HasColumn` 守卫而跳过加列、但会补记版本 —— 无副作用。

---

## 十一、部门树 Scoped 收口（2026-11-17 补丁）

**现象**：机构管理员（锚点已配）在「部门管理」「系统用户」左侧看到全库部门树（如 1060 个）。

**根因**：部门树接口此前**从不消费 ScopeResult**——`DataScopePermission` 中间件注入了可见集，但 `department_service` 从不读取。本次改动前部门树就是全库可见，属 system 域既有缺口，机构管理员场景下表现为越权面。

**修复（显式 opt-in，零回归）**：
- `DepartmentListParams` 新增 `Scoped bool json:"scoped"`；
- `filterDeptTreeByScope`（`department_service.go`）：未注入/All → 原样；FailClosed/Self → 空树 fail-closed；DeptIDs → 自身命中保留整棵子树、祖先路径保留（值拷贝，防未来缓存污染）；
- 已收口调用点：**部门管理页**（`useDeptData.ts`）、**系统用户页**左侧 `DeptTree`（新增 `scoped` prop，`useDeptTree({scoped})` 走独立 React Query 缓存键）；
- **保持全量的调用点**（配置/选择器场景，`tree-select` 与不带参数的 `tree`）：角色「自定义数据范围」勾选树、岗位锚点部门选择、ops 页面选择器、值班、通知目标等——管理员配置时必须看到全树；ops 选择器若收口会让运维角色（Self 档）看到空树，破坏 `.planning/debug/ops-selector-dept-perm-403.md` 修复的初衷。

**已知取舍**：树显示范围**等于或略宽于**数据可见集——Custom 档"勾父不勾子"时子部门**节点**可见但**数据**不可见。与前端 Phase 133 `filterTreeByVisibleDeptIds` 同语义（级联勾选场景下两者实际一致）；数据面过滤仍由 Resolve 精确清单决定。

**测试**：`internal/services/system/dept_scope_filter_test.go` 7 条（锚点收口/All 零回归/不传 scoped 全量/Self 空树/FailClosed/Custom 子树语义/祖先节点拷贝）；前端 `useDataHooks.test.tsx` mock 点随 hook 实现更新并新增 scoped 用例。

**全量消费点盘点结论（20 调用点/16 页面）**：

| 分类 | 页面 | 处置 |
|---|---|---|
| A 已收口 | 部门管理页、系统用户页左树、**用户表单部门下拉** | `scoped:true` |
| B 必须全量（配置场景） | 角色数据范围勾选树、角色/岗位锚点选择、经理同步规则、AD 域 OU 映射、AD 对账 | 不动 |
| C 保持全量（选择器，数据面已另行收口） | 网络设备/MAC、楼宇/楼层/工位/信息点/机房/机房设备 | 不动（收口会让 Self 档运维空树） |
| D 已决策收口 | 通知公告目标选择、值班池、工单派单 | 本轮改 `useDeptTree({scoped:true})` |

**已知取舍**：树显示范围**等于或略宽于**数据可见集——Custom 档"勾父不勾子"时子部门**节点**可见但**数据**不可见。与前端 Phase 133 `filterTreeByVisibleDeptIds` 同语义（级联勾选场景下两者实际一致）；数据面过滤仍由 Resolve 精确清单决定。

**测试**：`internal/services/system/dept_scope_filter_test.go` 7 条（锚点收口/All 零回归/不传 scoped 全量/Self 空树/FailClosed/Custom 子树语义/祖先节点拷贝）；前端 `useDataHooks.test.tsx` mock 点随 hook 实现更新并新增 scoped 用例。

---

## 十二、用户列表数据面修复：读 ctx 注入（2026-11-17 生产实测）

**现象**：部署后刘洋「锚点预览」显示已解析（命中武汉中心支公司），但系统用户列表**仍只见自己一人**。

**根因**：`applyUserDataScope`（user_service.go）在 service 层**自行调 `datascope.Resolve(ctx, scopeUserID, dataScope)`**，只按角色档位（Self）解析——中间件 `DataScopePermission` 把「Resolve → ResolveAnchorGrants → Merge」后的**最终结果**注入 ctx，这里却完全不消费。两条通道分歧：预览走直连解析所以命中，用户列表走老通道所以 `sys_user.id = 自己`。

**修复**：`applyUserDataScope` 优先 `datascope.FromContext(ctx)` 应用合并后的结果（All/FailClosed/Self/DeptIDs 逐档对齐）；**未注入时**（内部任务/直接调 service 的单测）才回退老通道。老通道代码一字未删，既有测试零回归。VDI 已确认在 `vm_service_impl.go:657` 正确消费 ctx，无需改。

---

## 十三、D 类三处部门选择器收口（2026-11-17 用户确认）

用户确认「公告目标选择、值班池、工单派单都收口」。三处均把 `useDeptTree()` 改为 `useDeptTree({ scoped: true })`：
- `pages/system/notice/hooks/useTargetSelector.ts:50`
- `pages/duty/pools/index.tsx:79`
- `pages/workorder/orders/hooks/useWorkOrderData.ts:89`

语义：机构管理员发公告/建值班池/派工时，只能选择本机构子树内的部门。scoped 与全量树走 React Query 独立缓存键，互不污染。

---

## 十四、编辑用户「权限预览」（2026-11-17 用户要求）

**需求**：编辑用户时角色字段下方加"权限预览"，除了菜单权限也要展示数据范围。

**实现**：
- 后端新端点 `POST /system/users/permission-preview`（读组，继承 `system:user:list/view` + `DataScopePermission`，只读不记 operlog）
  - 请求体 `{deptId, roleIds, postIds}` = **表单当前值（尚未保存）**
  - 响应 `datascope.UserScopePreview`：`dataScope/dataScopeLabel/all`（档位）、`permissions`（菜单权限码汇总去重排序）、`anchorSources[]`（每个角色/岗位锚点来源的命中明细：模式/参数/锚点部门名/子树部门数/未命中原因）、`visibleDeptIds/visibleDeptNames`（最终可见部门）、`notes`（如 All 档说明）、`roleNames/postNames`
  - 锚点解析复用 `ResolveAnchorByDept → resolveAnchorByChain`（与运行期**同一内核**），保证"预览结果 = 保存后结果"；Dept/DeptChild 档以**表单 deptId**为锚点（未保存值也能预览）
- 新增文件：`pkg/datascope/user_permission_preview.go`（+6 条测试）、`internal/api/v1/system/user_permission_preview_handler.go`、前端 `src/components/system/UserPermissionPreviewPanel.tsx`、`src/lib/systemAnchorApi.ts` 增 `previewUserPermission`、`src/types/system.ts` 增两个类型
- 前端用 `Form.Item shouldUpdate` 取实时表单值，按钮点击后展示：菜单权限 Tag 列表、数据范围档位 Alert、锚点来源明细 Table、可见机构 Alert；接口失败内部降级不崩页面

**门禁同步**：`internal/utils/operlog/handler_coverage_test.go` 把新只读 handler 登记 readonly（PreviewUserPermission）；`src/lib/apiFactory.invariants.test.ts` 把 `systemAnchorApi.ts` 登记 EXPECTED_FILES/warning 档 0；`scripts/route-registry.json` 重生成（729 路由）。

**最终验证**（2026-11-17 18:03）：`go build ./...` 0 error；`go test ./...` 0 FAIL 包；前端 `npm run type-check` / `npm run lint` exit 0；`npx vitest run src/pages/system/user/ src/components/system/ src/lib/` 32 文件/349 用例全绿；路由门禁 OK；operlog 门禁 OK。

### 补充（2026-11-18）：权限预览改为树形勾选态

用户反馈"编辑用户中的权限预览应设计成和编辑角色模态框里一样的菜单树和部门树"。改造：

- **后端** `UserScopePreview` 新增三字段：
  - `menuIds[]` —— 选中角色汇总的菜单树节点 ID 集（查 `sys_menu ⋈ sys_role_menu`，含权限码对应的 menu id）；
  - `deptIds[]` —— 最终可见部门 ID 集（与 `visibleDeptIds` 同源）；
  - `allDepts` —— All 档标记（前端把部门树渲染为"全库可见"提示）。
- **前端** `UserPermissionPreviewPanel` 重写：点预览后 `Promise.all` 并发拉 ① `/system/users/permission-preview`（模拟勾选集+锚点明细）② `/system/menus/tree-select`（菜单树）③ `/system/departments/tree-select`（部门树，配置场景全量——与角色模态框同源）；渲染成两个并排 `Card` + `<Tree checkable disabled checkedKeys={menuIds|deptIds}>`（**与编辑角色模态框同款组件与样式**），树下保留数据范围档位 Alert + 机构锚点命中明细表。树 disabled 表示只读展示（预览不提供勾选交互）。
- **验证**（2026-11-18 09:21）：`go build ./...` 0 error；`go test ./...` 0 FAIL 包；前端 type-check / lint exit 0；`npx vitest run src/components/system/ src/pages/system/user/ src/lib/systemAnchorApi.test.ts` 6 文件/38 用例全绿。

> 过程记录：初次 lint 报 `no-irregular-whitespace`——文案里用了全角空格 `　`（U+3000），已改为普通 `" · "` 分隔。提醒：本仓库 lint 严格禁止全角空格等不可见空白字符。

### 补充（2026-11-18）：部门树切账号缓存污染修复（真因）

用户实测：`liuyang-198` 登录后**用户列表 412 人（武汉中支收口正确）**，但左侧部门树仍显示全量；权限预览（点按钮现算）正确显示"命中武汉中心支公司·子树 117 部门"。超管登录用户数 4230（全库）。

**根因**（不是后端问题）：`useDeptTree` 的 React Query key 是 `['dept','tree','scoped']`，**不含登录人 ID**。用户在同一浏览器先以超管登录、退出、再用 liuyang-198 登录——React Query 天沟缓存（gcTime 30min）里还留着超管时代的 scoped 全量树，新账号直接命中旧缓存，**根本没发新请求**。这解释了三者的分歧：
- 用户列表 412 = 每次请求实时算 → 正确
- 权限预览 = 点按钮服务端现算 → 正确
- 左侧树 = React Query 缓存 → 命中超管旧全量树 → 全量

**修复**：`useDeptTree` 的 queryKey 末尾拼当前登录人 ID（`useAuthStore(s => s.user?.id) ?? "anonymous"`），切账号自动隔离缓存。`useInvalidateDept` 用 `['dept']` 前缀失效仍兼容。

**防回归**：`useDataHooks.test.tsx` 新增"切换登录用户后 scoped 部门树缓存隔离"用例——user-a 先拉全量树、切 user-b 后必须重新拉取收口树，并断言两条缓存并存。mock authStore 的 `currentUserID` 可控切换。type-check、26 条 hook 测试、全量 lint 全绿。

**经验**：数据范围相关的**前端缓存**必须把"登录人"作为缓存键维度；否则多账号共用浏览器时，先登录者的收口结果会污染后登录者。这是比"后端过滤写错"更隐蔽的一类越权面（后端每次都算对，前端缓存给了旧答案）。

### 终极真因（2026-11-18 10:28）：缓存装饰器绕过收口 —— 部门树全量的真正原因

前面所有关于"部门树仍全量"的假设（超管短路、前端缓存、二进制未部署）**全部错误**。生产日志给出铁证：

```
{"msg":"[部门树] GetTree 请求诊断","path":"/api/v1/system/departments/tree","scoped":true,...}
```

**只有 handler 的 scoped:true 日志，没有 `[部门树收口] scoped 请求诊断` 日志**——后者在 `filterDeptTreeByScope` 第一行，只要调到就必然打印。说明过滤器根本没被调用。

**根因**：生产的 router.go:92 用 `NewDepartmentServiceWithCache`（缓存装饰器）而非裸 service。看 `department_cache_impl.go:37` 的 `GetTreeWithFilter`：

```go
// 普通树请求（scoped=true，无 DeptName/Status）走缓存：
cacheKey := BuildDeptCacheKey(CacheKeyDeptTree)   // dept:tree[:all]，不含 scoped/userID
GetOrSetJSON(..., func() { return s.queryTree(...) })  // queryTree 全量查询，从不调 filterDeptTreeByScope
```

- `queryTree` 自己查全量树、不收口 → 无过滤器日志
- 缓存 key 不含 scoped/登录人 → 任一用户先拉的全量树被缓存，所有人（含刘洋）都拿到全量
- 只有带 DeptName/Status 的搜索才 `return s.departmentService.GetTreeWithFilter(...)`（含收口）

**这解释了全部现象**：用户列表 412（更早的 applyUserDataScope 修复已部署且它走裸 service 逻辑）、权限预览正确（点按钮现算）、唯独树全量（走缓存装饰器的全量路径）。

**修复**：`departmentCacheService.GetTreeWithFilter` 中 `params.Scoped==true` 时**绕过缓存**，直接委托 `s.departmentService.GetTreeWithFilter`（内含 filterDeptTreeByScope）。收口结果依赖当前登录人，按人实时变化，本就不该共享缓存；非 scoped 全量树继续走缓存零回归。加缓存层诊断日志。

**红绿验证**（关键）：新增 `dept_cache_scoped_test.go`（2 条：缓存装饰器下 scoped 收口 / 不传 scoped 全量）。临时注释掉绕过缓存分支 → `TestDeptCacheService_ScopedRequest_UsesFilter` **FAIL**（长沙中支 D-cs 出现在树里）；还原 → **PASS**（D-cs 消失）。证明测试精确锁住 bug。全量编译 + system/api/datascope/middleware 四包测试全绿。

**排查方法论教训**：当" handler 日志有、service 层日志没有"时，要立刻怀疑**接口被装饰器/代理实现接管**——生产 DI 注入的可能不是你以为的那个实现。我前面浪费多轮猜超管/缓存/二进制版本，就是没查 router.go 的 DI 分支（`NewDepartmentServiceWithCache` vs `NewDepartmentService`）。**先确认运行时真正注入的实现，再看代码逻辑。**

---

## 十五、提交前独立审计 + 全页面部门树收口（2026-11-18）

三路只读审计（后端安全 / 前端质量 / 规范测试）+ 一轮全量消费点盘点，发现并修复：

### 15.1 审计必修复项（4 项）

| 项 | 级别 | 问题 | 修复 |
|---|---|---|---|
| Dept 档树放大 | **P0 越权** | `filterDeptTreeByScope` 旧逻辑"节点自身命中→整棵子树保留"，而 `DataScopeDept(3)` 的 DeptIDs 只含单节点 → 本部门档用户看到下级部门树 | 改为**精确匹配裁剪**：只保留 DeptIDs 集合内节点 + 祖先路径；子树语义由 Resolve 展开进 DeptIDs 保证（DeptChild/锚点档不受影响）。新增用例 6b 锁定锚点子树仍可见 |
| 前端预览白屏 | P1 | `previewUserPermission` 原样透传 data，后端无角色/岗位时数组字段为 JSON null，面板 `.length` → TypeError 白屏 | `previewUserPermission` 对 9 个数组字段统一归一化 `[]` |
| 锚点降级中断 | P1 | `ResolveAnchorGrants` 单来源失败 `return nil, err` 与注释矛盾，一个脏来源让全部授权失效 | 改为跳过失败来源 + `applogger.Warnf` 继续 |
| 导入绕过嫌疑 | P1 | 疑 `ImportUser` 可写锚点岗位绕过 guard | **核实不成立**：user Excel 模板仅 7 列（无岗位列），导入不写 sys_user_post。收紧 `grantAnchorPolicy` 注释，明确关闭校验的提权风险 |

### 15.2 全页面部门树收口（用户要求"运维页也要收口 + 全面无遗漏"）

全量盘点 20 个消费点，逐一分类处置：

- **DeptSidebar 默认 `scoped=true`**（组件唯一用途是运维页左树筛选）→ 一次覆盖：楼层、信息点、机房设备、机房、工位、MAC（6 页左树）
- **6 处 hook 改 `useDeptTree({scoped:true})`**：楼宇页 index、楼宇/楼层共用 useDepartmentData、工位 useWorkstationData、LocationAliasDrawer、网络设备 useDeviceData、全局 TargetSelector
- **楼宇页左树 `<DeptTree scoped>`**（不走 DeptSidebar 的那一处）
- **`/system/departments/list` 加 Scoped opt-in**：`departmentService.List` + `filterDeptListByScope`（扁平列表精确过滤），网络发现页 `useDiscoveryData` 传 `scoped:true`（该页用 list 端点非 tree）
- **保持全量（配置场景，勿动）**：岗位锚点部门选择、角色自定义数据范围勾选、权限预览树（均 tree-select）、AD OU 映射、经理同步机构配置
- **403 历史修复不受影响**：`RequirePermissionsWithQuery`（鉴权层）与 `scoped`（数据范围层）是同一端点的两个独立维度

### 15.3 验证

- 全量盘点确认"应收口清单"全部覆盖，配置场景零误伤
- 新增测试：`dept_list_scoped_test.go`（3 条，扁平列表 scoped/全量/All+Self）
- `go build ./...` 0 error；单次 `go test ./...` **0 FAIL**（曾因并发跑两个 go test 出现 3 个假 FAIL，单次干净运行确认是测试进程资源冲突，非代码问题）；前端 type-check / lint exit 0、`npx vitest run src/components/operations/ src/pages/operations/ src/pages/network/ src/components/DeptTree/` **123 文件 / 699 用例全绿**
- 路由门禁 / operlog 门禁 PASS

**当前状态**：所有审计问题已关闭，全页面部门树收口完成，可提交。
