# Phase 130 — ScopeProvider 基建（v1.52 / DS-01 + DS-02）

- **裁定输入**：`.planning/notes/261107-v152-self-scope-decision.md`（LOCKED，本 phase 只消费不重新裁定）
- **需求**：`.planning/REQUIREMENTS.md` DS-01 / DS-02 + 验收硬约束
- **Roadmap**：`.planning/ROADMAP.md` Phase 130 条目
- **执行**：2026-11-07，与 Phase 131/132 并行（本 agent 文件面不与 ops service 重叠）

---

## 1. 交付清单

| # | 文件 | 内容 | 提交 |
|---|------|------|------|
| A | `pkg/datascope/datascope.go` | `ScopeResult{All,DeptIDs,ResourceHook,FailClosed,UserID}`、`ResourceHook`(HookNone/HookSelfUser)、`Resolve`、`WithContext`/`FromContext`（私有 typed key）、`AllScope`/`FailClosedScope`、`normalizeIDs`、Custom/Dept/DeptChild 查询（双侧 `CAST AS TEXT`） | 第一笔 `06e892ee`（Lead 亲自落地） |
| A | `pkg/datascope/datascope_test.go` | 12 个厚单测（7 边界 + Custom/Dept/DeptChild 正反 + All 零查询 + MAX→Self + ctx 往返） | 第一笔 `06e892ee` |
| B | `internal/services/system/user_service.go` | `applyUserDataScope` 委托 `datascope.Resolve`，保留 `sys_user.id=?`(Self)、All/0 不过滤、空 scopeUserID→`1=0`、Count 之前过滤 | 第二笔（本文件） |
| C | `internal/api/router.go` | ops 全域 **13 读组 + 13 写组** 共 26 处挂 `DataScopePermission(core)`（既有 5 处不动，总计 31） | 第二笔 |
| C | `scripts/route-registry.json` | `node scripts/scan-routes.cjs` 重生成（路由集合/permKind 不变，仅 `line` 字段随 router.go 行号下移） | 第二笔 |
| C | `internal/api/datascope_mount_test.go` | 挂载计数守卫：31 处总数、26 组逐一覆盖、读组 `Require* → DataScope` 顺序、禁止挂祖先组/豁免组 | 第二笔 |
| D | `pkg/middleware/permission.go` | `DataScopePermission` 注入 request context（普通用户 `Resolve` 完整集 / 超管 `AllScope` 零查询 / 出错 `FailClosedScope` 不 abort），`c.Set` 原行为不变；`ApplyDataScope`/`ApplyDataScopeFromContext` 加 `@Deprecated` 注释（Phase 133 删） | 第二笔 |
| D | `pkg/middleware/permission_datascope_ctx_test.go` | ctx 注入契约回归：超管注入 All 但不补 c.Set、普通用户 c.Set+ctx 双通道、未认证不注入、`WithContext`/`FromContext` 往返（含 UserID）与缺省 | 第二笔 |
| — | 本 SUMMARY | `.planning/phases/130-scope-provider/130-SUMMARY.md`（**新建目录**，非 v1.51 历史归档 `130-device-evidence/`） | 第二笔 |

**API 最终形态（131/132 消费契约，导出名/字段已定稿）**：

```go
type ResourceHook int // HookNone / HookSelfUser
type ScopeResult struct {
    All          bool
    DeptIDs      []string      // Resolve 已展开并 normalizeIDs（去重+去空+排序）
    ResourceHook ResourceHook
    FailClosed   bool
    UserID       string        // 登录人 UUID，All/0 短路也透传
}
func Resolve(ctx context.Context, db *gorm.DB, userID string, scope models.DataScope) (ScopeResult, error)
func WithContext(ctx context.Context, r ScopeResult) context.Context
func FromContext(ctx context.Context) (ScopeResult, bool)
func AllScope(userID string) ScopeResult
func FailClosedScope(userID string) ScopeResult
```

`DataScopePermission` 把 **完整 Resolve 结果**（DeptIDs 已展开）注入 `c.Request.Context()`；
131/132 只需 `datascope.FromContext(ctx)` 即可用 `DeptIDs` 做 `CAST(有效机构 AS TEXT) IN (?)`，
无需改任何 handler 签名、无需自己再查 `sys_dept`、再动 router.go。

**关键工程决策**：DeptChild 子树展开采用 `BuildDeptRecursiveFilter` 四条件语义的**五条件超集**
（`id` / `ancestors=id` / `ancestors LIKE 'id,%'` 链首 / `'%,id,%'` 居中 / `'%,id'` 链尾），
单条 SQL、非递归。生产 `ancestors` 由 `department_service.buildAncestors` 维护为「无首尾逗号」
格式（根 A → 子 B=`A` → 孙 C=`A,B`），缺链首条件会漏 3 层深树的孙部门；
`user_service` 既有 `user_data_scope_test.go` 的 A+B+C 断言因此保持全绿。
子树展开逻辑在 datascope 包内只有一处（禁止内联复制 `getChildDepts` 递归实现），
Phase 131/132 收敛 operations 侧时以本 helper 为单一真实源（方向 operations→datascope，
不产生 pkg→internal 反向依赖）。

---

## 2. 裁定 §3 的 7 个边界用例结果

| # | 用例 | 守护测试 | 结果 |
|---|------|----------|------|
| 1 | 超管（`isSuperAdmin` 早退未注入）→ `Resolve(scope=0)` = All，读端点零回归 | `TestScopeResolve_AllShortCircuit` + `TestDataScopePermission_RequestContext_Admin` + `TestDataScope_NoChangeForAllAndAdmin` | PASS |
| 2 | 未挂注入中间件（`data_scope` 缺失，零值 0）→ All | `TestScopeResolve_AllShortCircuit`（db 传 nil 断言零查询） | PASS |
| 3 | `scope=1` → All 且**不产生任何 DB 查询**（短路可断言） | `TestScopeResolve_AllShortCircuit`（`Resolve(..., nil, ...)` 不碰 db） | PASS |
| 4 | `scope=6 / -1 / 99` → `FailClosed=true` + 空集 + `HookNone` | `TestScopeResolve_FailClosed` | PASS |
| 5 | 多角色 `{1,3,5}` → MAX=5（最严）→ Self | `TestScopeResolve_MultiRoleMaxSemantics`（Resolve 传入 5 → HookSelfUser）+ middleware 既有 `TestGetUserMaxDataScope` 守护 MAX 取值 | PASS |
| 6 | 无角色 → COALESCE 5 → Self | `TestGetUserMaxDataScope`（既有）+ `TestScopeResolve_MultiRoleMaxSemantics` | PASS |
| 7 | 仅停用角色参与时 `getUserMaxDataScope` 不计入 → COALESCE 5 → Self | middleware 既有测试 + `TestScopeResolve_Custom_DisabledRoleExcluded`（停用角色空集 fail-closed） | PASS |

补充（Lead 约定的 ctx 契约）：`WithContext`/`FromContext` 往返含 `UserID` 断言、缺省返回
`(零值,false)`、nil ctx 不 panic、私有 typed key 不被普通 key 污染 —— 均 PASS。

---

## 3. 挂载组数（DS-02）

| 域 | 组数 | 挂载方式 |
|----|------|----------|
| 既有（system usersRead / usersWrite / deptsRead / deptsWrite、vdi vms） | 5 | 不动 |
| ops 读组（building/floor/workstation/location-alias/workstation-device/asset/infoPoint/serverRoom/dedicatedLine/roomDevice/walls/doors/floor-plan-texts） | 13 | `Require*` 之后 `Use(DataScopePermission)`（权限校验之后、handler 之前） |
| ops 严格写组（同上 13 域） | 13 | 与读组互为**兄弟组**（`ops.Group` 直接子组，不继承读组 ANY-of），组定义后 `Use(DataScopePermission)` |
| 显式豁免 | — | `roomPhotos` 几何/照片端点（REQUIREMENTS 映射总表登记豁免）不挂；`ops` 祖先组不挂 |
| **合计** | **31** | `TestDataScopeMount_OpsCoverage` 断言 |

`RequirePermissionsWithQuery` / `RequireStrictPermissions` 自身逻辑**未改**（DS-02 硬要求）。
写组为兄弟组，遵循 router.go:103-104 / :175-177 的 gin 中间件链约束。

---

## 4. 闸门输出

| 闸门 | 命令 | 结果 |
|------|------|------|
| 编译 | `go build ./...` | PASS |
| 分域测试（步骤 2） | `go test -count=1 ./pkg/datascope/... ./pkg/middleware/... ./internal/services/system/... ./internal/api/... ./pkg/permission/...` | 全部 `ok`（含 pkg/permission 路由双层守卫、internal/api strict 守卫、新增挂载计数 + ctx 注入测试） |
| 全量测试（步骤 3） | `go test -count=1 ./internal/... ./pkg/... ./cmd/...` | **全部 ok，exit code 0**（含 `internal/services/operations` 27.734s、`pkg/datascope`、`pkg/middleware`、`pkg/permission`、`internal/api` 等全包） |
| route-registry | `node scripts/scan-routes.cjs` | `总路由: 702 写路由: 311 挂载点: 208`；与入库产物集合一致（freshness 守卫 PASS） |
| strict 守卫 | `strict_mount_count_test.go` 下限 16/188/164 | PASS（源码引用数与 registry 计数均未跌破） |

---

## 5. Commit

- 第一笔（Lead）：`06e892ee` — `feat(v1.52-p130): ScopeProvider 单一真实源 API（最小可用）`（`pkg/datascope/**` 两个文件）
- 第二笔（本 agent）：`__COMMIT_HASH__` — `feat(v1.52-p130): ScopeProvider 单一真实源 + user_service 迁移 + ops 全域 data_scope 注入`
  - `git add` 仅含本 agent 文件；**未** `git add -A`；未提交 `coverage.out`；
  - 未提交并行 agent 的 `internal/services/operations/**`、`.env.production`、`useBackfillDept.tsx`。

---

## 6. 遗留项 / 后续 phase

1. **Phase 131/132**：四域读写端点用 `FromContext(ctx)` 注入可见集过滤、缓存 key 增 scope 维度段、
   Excel 导出独立收口、PG contract 覆盖（本 phase 只建基建，不动 ops service 函数体）。
2. **Phase 133**：删除 `pkg/middleware` 的 `ApplyDataScope` / `ApplyDataScopeFromContext` 死实现与
   `permission_74_08_test.go` 死测试（本 phase 已加 `@Deprecated` 注释，**按裁定未删**）；
   迁移 `internal/services/vdi/vm_data_scope_filter.go` 生产调用方。
3. **子树单一真实源收敛**：`internal/services/operations/dept_filter.go:BuildDeptRecursiveFilter`
   目前仍是四条件变体；131/132 改 ops 文件时应改为复用 datascope 的五条件 helper
   （本 phase 因文件面边界未动 operations）。
4. **残余风险**（裁定 §5）：Self 档下 floor/building 等无用户列实体返回空集；
   All 档短路调用方不得把「不查部门集」误当作「跳过缓存 scope 段」（PMODEL-01）。
