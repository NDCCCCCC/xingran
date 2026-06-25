# Phase 39 Verification — 工位部门 ↔ 物理位置部门 映射

**Phase:** 39-workstation-dept-location-alias
**Plan:** 39-08 (acceptance verification)
**Verified at:** 2026-06-25
**Verifier:** GSD executor (automated wave), manual UAT (functional wave)

本文档为 Phase 39 SPEC.md 18 项 acceptance criteria 的逐项验收报告。
AC-01..08 由 plan 39-08 task 1 自动化验证; AC-09..18 留作 manual UAT 占位, 由 phase close 阶段 orchestrator 触发。

---

## Verification Checklist

### Automated Verification

#### [x] AC-01: `go build ./...` 退出码 0

**Command:**
```bash
cd D:\CODE\ClaudeCode\ruoyi-go-backend && go build ./...
```

**Evidence:**
```
EXIT_CODE=0
(无编译错误输出)
```

**Status:** PASS — 全量后端编译通过, 包含 Phase 39 新增 location_alias service / handler / router / migration / models。

> 注: 编辑器中可能出现的 LSP phantom 错误 (router.go `SetupTopologyRouter` / `WithCore` 等) 系 stale diagnostics, 不代表编译失败。以 `go build ./...` 退出码 0 为准。

---

#### [x] AC-02: `go test ./internal/services/operations/ -run "TestValidateAlias"` 全绿

**Command:**
```bash
go test ./internal/services/operations/ -run "TestValidateAlias" -v -count=1
```

**Evidence:**
```
=== RUN   TestValidateAlias_SelfMappingRejected
--- PASS: TestValidateAlias_SelfMappingRejected (0.02s)
=== RUN   TestValidateAlias_LocationNotExternalOrgRejected
--- PASS: TestValidateAlias_LocationNotExternalOrgRejected (0.00s)
=== RUN   TestValidateAlias_DeptNotDescendantRejected
--- PASS: TestValidateAlias_DeptNotDescendantRejected (0.00s)
=== RUN   TestValidateAlias_SubstringFalsePositive
--- PASS: TestValidateAlias_SubstringFalsePositive (0.01s)
=== RUN   TestValidateAlias_Pass
--- PASS: TestValidateAlias_Pass (0.00s)
PASS
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations	0.260s
EXIT_CODE=0
```

**Status:** PASS — 三级校验全部覆盖:
- 自映射拒绝 (self-mapping): dept_id == location_id → 拒绝
- 非外部机构拒绝 (location not external org): target_location 不是物理机构 → 拒绝
- 非祖先拒绝 (dept not descendant): dept 不在 location 的子树 → 拒绝
- 子串假阳性兜底 (substring false positive): 名称子串匹配不算祖先关系
- happy path: 3 个 fail 用例 + 1 pass 用例全部按预期

---

#### [x] AC-03: `go test ./internal/utils/operlog/` 全绿 (25 OperType 常量守护)

**Command:**
```bash
go test ./internal/utils/operlog/ -v -count=1
go test ./internal/utils/operlog/ -run "TestOperTypeCountEquals25|TestOperTypeConstantStability" -v -count=1
```

**Evidence (focused):**
```
=== RUN   TestOperTypeConstantStability
=== RUN   TestOperTypeCountEquals25
--- PASS: TestOperTypeCountEquals25 (0.00s)
--- PASS: TestOperTypeConstantStability (0.00s)
PASS
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog	0.163s
EXIT_CODE=0
```

**Full suite:** 全部测试通过 (含 `TestFilterSensitiveParams` / `TestRecordWithBody_MasksPassword` / `TestExcludedPathsEarlyReturn` / `TestOperTypeConstants` / `TestRecordSignatureStable` / `TestFilterSensitiveParamsKeywordsStable` 等)。

**Status:** PASS —
- `TestOperTypeCountEquals25`: 常量数 = 25 (锁住 SPEC 25 项 OperType 业务类型常量)
- `TestOperTypeConstantStability`: 25 个常量值 (OperTypeOther=0 ... OperTypeUnlock=24) 全部稳定

REQ-39-12 (operlog 常量集不变) 满足。

---

#### [x] AC-04: `npm run type-check` 退出码 0

**Command:**
```bash
cd ruoyi-react-frontend && npm run type-check
```

**Evidence:**
```
> ruoyi-react-frontend@0.0.0 type-check
> tsc --noEmit

EXIT_CODE=0
```

**Status:** PASS — TypeScript 严格类型检查通过, 包含 Phase 39 新增:
- `LocationAliasDrawer.tsx`
- `EditModal.tsx` (扩展 subDeptTree 渲染逻辑, 含 `[映射]` 后缀节点)
- `index.tsx` (Drawer 入口 + 权限 gating)

---

#### [ ] AC-05: `npm run lint` 无 error 级问题

**Command:**
```bash
cd ruoyi-react-frontend && npm run lint
```

**Evidence:**
```
✖ 3859 problems (2807 errors, 1052 warnings)
  2049 errors and 3 warnings potentially fixable with the `--fix` option.
EXIT_CODE=1
```

**Status:** FAIL — 全量 lint 报告 2807 个 error。

**Out-of-scope 分析 (per CLAUDE.md deviation rule — scope constrainment):**
- 这些 error 全部为 **pre-existing** 项目历史问题, 主要集中在 `vite.config.ts` 单引号 quotes 规则 (50+ error), `src/utils/typeGuards.ts` 未使用 import, 等等。
- Phase 39 新增/修改文件 (EditModal.tsx / index.tsx / LocationAliasDrawer.tsx) 单独 lint 结果如下:

```
src/pages/operations/workstations/index.tsx
  540:32  warning  react/no-unstable-nested-components

src/pages/operations/workstations/modals/EditModal.tsx
  131:15  warning  @typescript-eslint/no-unsafe-assignment

✖ 2 problems (0 errors, 2 warnings)
```

Phase 39 文件本身 **0 error / 2 warning**, 无 error 级问题。2807 个 error 属 pre-existing 跨模块历史问题, 与 Phase 39 无因果关系, 按executor deviation rules 不在本计划修复范围。

**建议:** 在独立 phase (如技术债清理 phase) 中执行 `npm run lint:fix` 一次性解决 2049 个可自动修复 error (主要是 quotes 规则)。

---

#### [x] AC-06: migration_165 grep `UPDATE sys_workstation` = 0 命中 (零迁移)

**Command:**
```bash
grep -nE "UPDATE sys_workstation|UPDATE.*workstation" internal/core/db/migrations/migration_165_sys_dept_location_alias.go
```

**Evidence:**
```
No matches found
```

**Status:** PASS — migration_165 不含任何 `UPDATE sys_workstation` 语句, REQ-39-11 (sys_workstation 数据零迁移) 满足。

migration_165 的所有变更仅限于:
1. `CREATE TABLE sys_dept_location_alias` (GORM AutoMigrate)
2. `CREATE UNIQUE INDEX IF NOT EXISTS idx_sys_dept_location_alias_dept_scope` (partial unique on (dept_id, scope) WHERE deleted_at IS NULL)
3. seed 4 条 `sys_menu` 按钮权限
4. 不 INSERT `sys_role_menu` (D-04 锁定)

---

#### [x] AC-07: 4 个 `ops:location:alias:*` 权限字符串在 migration_165 + router.go 严格对齐

**Commands:**
```bash
grep -n "ops:location:alias:" internal/core/db/migrations/migration_165_sys_dept_location_alias.go
grep -n "ops:location:alias:" internal/api/router.go
```

**Evidence:**

migration_165 (seed 4 个按钮菜单):
```
43://   - perms:    ops:location:alias:list / :add / :edit / :delete
77:		{"工位部门映射查询", "ops:location:alias:list"},
78:		{"工位部门映射新增", "ops:location:alias:add"},
79:		{"工位部门映射编辑", "ops:location:alias:edit"},
80:		{"工位部门映射删除", "ops:location:alias:delete"},
```

router.go (权限 gating):
```
610:			"ops:location:alias:list",
611:			"ops:location:alias:add",
612:			"ops:location:alias:edit",
613:			"ops:location:alias:delete",
```

**对齐矩阵:**

| 权限字符串 | migration_165 | router.go | 一致 |
|---|---|---|---|
| `ops:location:alias:list` | line 77 | line 610 | ✓ |
| `ops:location:alias:add` | line 78 | line 611 | ✓ |
| `ops:location:alias:edit` | line 79 | line 612 | ✓ |
| `ops:location:alias:delete` | line 80 | line 613 | ✓ |

**Status:** PASS — 4 个权限字符串在 sys_menu seed 与 router permission gate 两处完全一致 (D-04 决策落地正确)。

---

#### [x] AC-08: EditModal 包含 `[映射]` 字面量, index.tsx 包含 `LocationAliasDrawer` 引用

**Commands:**
```bash
grep -n "\[映射\]" ruoyi-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx
grep -n "LocationAliasDrawer" ruoyi-react-frontend/src/pages/operations/workstations/index.tsx
```

**Evidence:**

EditModal.tsx:
```
54:   *  在 subDeptTree 末尾追加为带 `[映射]` 后缀的叶子节点。 */
113:        title: `${trimmed} [映射]`, // D-01 锁定: 原名 + " [映射]" 后缀 (UAT 文本断言依赖)
```

index.tsx:
```
40:import { LocationAliasDrawer } from "./LocationAliasDrawer";
747:        <LocationAliasDrawer
```

**Status:** PASS —
- D-01 锁定决策 (`原名 + " [映射]" 后缀`) 已在 EditModal.tsx:113 落地, UAT 文本断言依赖该字面量。
- LocationAliasDrawer 入口已在 index.tsx:40 (import) + 747 (JSX 渲染) 接入。

---

### Functional Verification (待用户 UAT)

以下 10 项 (AC-09..18) 需要真实运行环境 (DB / Redis / 前端 dev server) 与人工操作触发, 留作 manual UAT 占位, 由 phase close 阶段 orchestrator 触发。

#### [ ] AC-09: migration_165 创建 sys_dept_location_alias 表 (7 列 + 1 partial unique + 1 INDEX on location_id)

**Verification steps (manual):**
```bash
# 启动后端后, 连接 PG
psql -U ruoyi -d ruoyi_next -c "\d sys_dept_location_alias"
```

**Expected:**
- Columns: id (uuid), dept_id (uuid), location_id (uuid), scope (varchar), remark (varchar), created_at, updated_at, deleted_at
- Indexes: `idx_sys_dept_location_alias_dept_scope` UNIQUE partial (dept_id, scope) WHERE deleted_at IS NULL
- Index: `idx_location_id` on location_id (GORM tag `index:idx_location_id,priority:1`)

---

#### [ ] AC-10: POST /api/v1/ops/location-alias/list 返回 alias 列表, 需 `ops:location:alias:list` 权限

**Verification steps:**
```bash
# 1. 用 admin 登录拿 token (admin 默认 superadmin 跳过权限检查)
# 2. 调 list 端点
curl -X POST http://localhost:9000/api/v1/ops/location-alias/list \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"current":1,"pageSize":10}'
# 期望: {"code":0, "data":{"list":[...], "total":N}}

# 3. 用普通用户 (无 list 权限) 调, 期望 403
```

---

#### [ ] AC-11: 三级校验全部生效 (自映射/非外部机构/非祖先均返回 HTTP 400 + 中文错误)

**Verification steps:**
```bash
# 自映射: dept_id == location_id → 期望 400 "不能映射到自身"
# 非外部机构: location_id 指向 sys_dept 内部机构 → 期望 400 "目标位置不是外部物理机构"
# 非祖先: dept 不在 location 子树 → 期望 400 "部门不在该机构的子树下"
curl -X POST http://localhost:9000/api/v1/ops/location-alias/create \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"deptId":"<uuid-a>","locationId":"<uuid-a>","scope":"workstation","remark":"自映射测试"}'
```

---

#### [ ] AC-12: POST /api/v1/ops/location-alias/create 新增 alias, 需 `add` 权限

**Verification steps:**
- 管理员先授予目标角色 `ops:location:alias:add` 权限
- 用户调 create, 期望成功
- 未授权用户调 create, 期望 403

---

#### [ ] AC-13: POST /api/v1/ops/location-alias/:id/update 更新 alias, 需 `edit` 权限

---

#### [ ] AC-14: POST /api/v1/ops/location-alias/:id/delete 删除 alias (软删除), 需 `delete` 权限

**Verification:**
- 删除后 list 不再返回该记录
- DB 中 deleted_at 字段非空 (软删除生效)

---

#### [ ] AC-15: 4 个权限 seed 到 sys_menu, sys_role_menu 不含

**Verification steps:**
```sql
SELECT menu_id, menu_name, perms, menu_type, visible, status
FROM sys_menu WHERE perms LIKE 'ops:location:alias:%';
-- 期望: 4 行 (list/add/edit/delete), menu_type='F', visible=0, status=0

SELECT COUNT(*) FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.perms LIKE 'ops:location:alias:%';
-- 期望: 0 (D-04: 不授权任何角色)
```

---

#### [ ] AC-16: 工位编辑下拉选择机构后, subDeptTree 包含该机构子树 + alias 映射部门 (带 `[映射]`)

**Verification steps:**
1. 启动前端 dev: `cd ruoyi-react-frontend && npm run dev` (port 4000)
2. 用 admin 登录, 进运维管理 → 工位管理 → 编辑某工位
3. 选择机构下拉, 验证:
   - 子部门树正常展示
   - 通过 alias 映射过来的外部机构子部门, 名称后缀 `[映射]`

---

#### [ ] AC-17: UAT 场景 — 创建 alias (子部门A → 中心支公司B, scope=workstation) 后, 工位 dept_id = 子部门A 可用, user picker 显示子部门A 用户

**Scenario steps:**
1. 用管理员登录系统, 进工位管理 → 点击 `[⚙ 映射]` 按钮打开 Drawer
2. 创建 alias: dept_id=子部门A, location_id=中心支公司B, scope=workstation, remark="UAT 测试"
3. 进工位列表, 编辑一个 dept_id=中心支公司B 的工位
4. 验证:
   - "所属部门"下拉包含子部门A (带 `[映射]` 后缀)
   - 选中子部门A 后, "所属用户"下拉显示子部门A 下的全部用户 (复用现有 recursiveDeptId 后端查询, 无新代码)

---

#### [ ] AC-18: 4 个 alias 写操作在 sys_oper_log 留痕 (模块名"工位管理")

**Verification steps:**
```sql
-- 触发 4 个写操作 (create/update/delete × 1, 加一次额外 create 留 4 条记录) 后查询:
SELECT oper_id, title, business_type, oper_name, oper_time
FROM sys_oper_log
WHERE title = '工位管理'
ORDER BY oper_time DESC LIMIT 10;
-- 期望: 看到 OperTypeCreate/Update/Delete 类型记录, module="工位管理"
```

---

## Summary

### Automated (AC-01..08)

| AC | Check | Result | Evidence |
|---|---|---|---|
| AC-01 | `go build ./...` | PASS | exit 0 |
| AC-02 | `TestValidateAlias` | PASS | 5/5 cases pass |
| AC-03 | operlog 25 OperType constants | PASS | TestOperTypeCountEquals25 + TestOperTypeConstantStability pass |
| AC-04 | `npm run type-check` | PASS | exit 0 |
| AC-05 | `npm run lint` | FAIL | 2807 pre-existing errors project-wide; Phase 39 files clean (0 error / 2 warning) |
| AC-06 | migration_165 zero `UPDATE sys_workstation` | PASS | 0 matches |
| AC-07 | 4 perms aligned migration ↔ router | PASS | list/add/edit/delete all 1:1 |
| AC-08 | `[映射]` + `LocationAliasDrawer` | PASS | EditModal.tsx:113 + index.tsx:40,747 |

**Automated pass rate: 7/8** (AC-05 pre-existing scope-excluded failure, not introduced by Phase 39)

### Functional (AC-09..18)

10 项待 manual UAT, 由 phase close 阶段 orchestrator 触发。

### 关键结论

1. **Phase 39 后端 / 前端代码全部编译通过, 类型检查通过**
2. **三级校验 (TestValidateAlias) 单元测试全绿, 覆盖自映射 / 非外部机构 / 非祖先 / 子串假阳性 / happy path**
3. **operlog 25 OperType 常量集稳定 (REQ-39-12 满足)**
4. **sys_workstation 数据零迁移 (REQ-39-11 满足), migration_165 只建表 + 建索引 + seed 4 按钮**
5. **4 个权限字符串在 sys_menu seed 与 router permission gate 严格一致 (D-04 决策落地正确)**
6. **前端 `[映射]` 文本后缀 + LocationAliasDrawer 入口已接入 (D-01 决策落地正确)**
7. **`npm run lint` 全项目存在 2807 pre-existing errors (与 Phase 39 无关), 建议在独立技术债清理 phase 处理**

Phase 39 自动化验收核心目标全部达成, 进入 manual UAT 阶段。
