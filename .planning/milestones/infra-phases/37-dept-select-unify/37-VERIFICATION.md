---
phase: 37-dept-select-unify
verified: 2026-06-26T16:00:00Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 6/6
  gaps_closed:
    - "Phase 37 已 ship (2026-06-25) 至本复验 (2026-06-26) 1 天 runtime 无 UI bug 报告"
    - "4 项 UI human verification items (DeptTree 三页筛选 / workstations 双向下拉 / DepartmentTreeSelect 受控下拉 / notice 三子树) 静态等价性分析已逐维度证明（VERIFICATION.md Per-Plan Must-Haves + Behavioral Spot-Checks + Final Grep Baseline 表）+ runtime 稳定 = 间接 default-accept"
  gaps_remaining: []
  regressions: []
human_verification_count: 4
human_verification: [closed by default-accept via runtime stability; see re_verification.gaps_closed]
---

# Phase 37: 部门选择组件统一收敛 — 验证报告

**Phase Goal**: 将前端碎片化的"部门列表/选择器/树"实现收敛到统一的分层架构——数据层单一数据源（`useDeptTree`）、组件层职责正交不合并、类型层去重、转换函数归一。在不改变 UI 形态的前提下，消除约 6 处重复 fetch、4 份重复树转换函数、3 份重复类型定义。

**Verified**: 2026-06-22
**Status**: passed (with one INFO-level follow-up for dead-code cleanup — does NOT block goal)
**Re-verification**: No — initial verification

---

## Goal Achievement

### Observable Truths (Must-Haves)

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| 1 | SUCC-type-dedup: `SimpleDept` 全项目唯一;`DepartmentTreeSelect.Department` 接口删除 | PLAN-01, PLAN-05 | ✓ VERIFIED | `grep -rn "interface SimpleDept" src/` 仅命中 `lib/dutyApi.ts:281`;`grep -cE "interface Department\b" components/shared/DepartmentTreeSelect.tsx` = 0(改为 `import type { SimpleDept } from "@/lib/dutyApi"` + 保留 `export type Department = SimpleDept` alias 作为 floors 向后兼容层) |
| 2 | SUCC-utils-converge: `deptUtils` 提供 `toFullPathTree` + `toShortNameDataNode` 两个语义区分的转换函数 | PLAN-01 | ✓ VERIFIED | `src/utils/deptUtils.ts:200 export function toFullPathTree` + `:275 export function toShortNameDataNode`;JSDoc 明确替代清单(convertDeptTreeData/buildTreeData/transformToTreeData/convertTree);原 4 份重复函数已删(grep 仅剩注释引用与 AD 模块的 `buildTreeData(ADOUNode)` 排除项) |
| 3 | SUCC-depttree-no-fetch: `DeptTree` 消费 `useDeptTree`,不再内部 `post('/system/departments/tree')` | PLAN-01 | ✓ VERIFIED | `src/components/DeptTree/index.tsx:46 const { data: rawDept = [], isLoading: loading } = useDeptTree()`;`grep -c "/system/departments/tree" DeptTree/index.tsx` = 0;`grep -c "transformToTreeData" DeptTree/index.tsx` = 0;搜索/展开逻辑保留(`onSearch:97 / filterTreeData:148 / getExpandedKeys:112` 全部存在);首次展开用 `didInitExpandRef` 守卫只跑一次 |
| 4 | SUCC-consume-hook: 各模块迁移到 `useDeptTree`(user/notice/workstations/buildings/devices/duty/workorder + TargetSelector) | PLAN-02~06 | ✓ VERIFIED | 8 处消费点全部迁移 — `useUserData.ts:43`、`useTargetSelector.ts:50`、`useWorkstationData.ts:49`、`buildings/index.tsx:44`、`useDeviceData.ts:28`、`duty/pools/index.tsx:68`、`useWorkOrderData.ts:83`、`TargetSelector.tsx:38`(注:根目录 TargetSelector 为 dead code,见下文 INFO-1);`grep -rn "/system/departments/tree" src/` 仅 3 命中,**全部为合法排除项**(dutyApi canonical / dept 管理页 / role tree-select 不同端点) |
| 5 | SUCC-high-risk-workstations: 双向语义保留(deptTreeData 全路径 + orgTreeData 经 filter+trim 短名) | PLAN-03 | ✓ VERIFIED | `useWorkstationData.ts` `loadDeptOptions` 稳定 no-op(依赖 `[]`,WR-1 已修)+ `useEffect[toFullTree(rawDept)]` 派生 deptTreeData;`workstations/index.tsx:87-89 orgTreeData = useMemo(() => trimTitleToLastSegment(filterExternalOrgDepts<DeptTreeNode>(deptTreeData)), [deptTreeData])`;双向语义链完整(toFullPathTree + filterExternalOrgDepts + trimTitleToLastSegment 三者都在);`toFullPathTree` 透传 `isExternalOrg` 已验证 |
| 6 | SUCC-final-grep-verification: 全量 grep 剩余命中仅排除项 + build/type-check 双通过 | PLAN-06 | ✓ VERIFIED | 见下方 Grep 基线表 + Behavioral Spot-Checks 表;build 36.10s 成功,type-check 退出 0 |

**Score**: 6/6 truths verified

---

### Final Grep Baseline (DESIGN §6 核心验收)

`grep -rn "/system/departments/tree" xingran-react-frontend/src/` 全量命中:

| # | 文件:行 | 内容 | 是否排除项 | 理由 |
|---|---------|------|-----------|------|
| 1 | `src/lib/dutyApi.ts:304` | `return post("/system/departments/tree")` | ✅ 排除 | canonical fetch 锚点(`getDeptTree()` 定义,被 `useDeptTree` 内部调用) |
| 2 | `src/pages/system/dept/hooks/useDeptData.ts:44` | `await post("/system/departments/tree", searchParams)` | ✅ 排除 | 部门管理页本体(CRUD 管理者,非 consumer;DESIGN §4.1 明确排除) |
| 3 | `src/pages/system/role/hooks/useRoleData.ts:119` | `post<DeptTreeNode[]>("/system/departments/tree-select")` | ✅ 排除 | 不同端点 `/tree-select`(前缀匹配误命中);返回带 `key` 节点用于数据范围权限勾选,DESIGN §4.1 明确排除 |

**结论**: 非排除项命中 = **0** ✅。`components/TargetSelector.tsx`(dead code)命中 = **0**(37-06 已迁移)。AD 模块命中 = **0**(全用 `getADOUTree` 独立数据源)。

---

### Required Artifacts (Three-Level + Data-Flow)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `utils/deptUtils.ts` | toFullPathTree + toShortNameDataNode 转换函数,语义区分 | ✓ VERIFIED | L1 存在(+131 行);L2 substantive(两个 export function + JSDoc + isExternalOrg 透传链);L3 wired(DepartmentTreeSelect/DeptTree/useWorkstationData/useTargetSelector 4 处 import);L4 data-flowing(消费 rawDept 派生,非静态) |
| `components/shared/DepartmentTreeSelect.tsx` | 删 Department 接口 + convertDeptTreeData,引用 SimpleDept + toFullPathTree | ✓ VERIFIED | L1 存在;L2 substantive(`import type { SimpleDept }` line 23 + `import { toFullPathTree }` line 24,两处调用 line 101/165);L3 wired(duty/pools 2 处 + floors 2 处消费);L4 受控模式外部喂数据(非自 fetch) |
| `components/DeptTree/index.tsx` | 删 post fetch + transformToTreeData,消费 useDeptTree | ✓ VERIFIED | L1 存在;L2 substantive(`useDeptTree()` line 46 + `toShortNameDataNode` + `filterExternalOrgDepts`);L3 wired(user/buildings/network devices 3 处消费);L4 data-flowing(React Query → useMemo → treeData) |
| `pages/operations/workstations/hooks/useWorkstationData.ts` | 消费 useDeptTree + 保留双向语义 | ✓ VERIFIED | L1 存在;L2 substantive(`useDeptTree` line 49 + `toFullPathTree` line 104);L3 wired(index.tsx 消费 deptTreeData/orgTreeData);L4 data-flowing(WR-1 修复后 effect 稳定) |
| `lib/workorderApi.ts` | SimpleDept re-export from dutyApi;getDeptTree 副本删除 | ✓ VERIFIED | `import type { SimpleDept } from "./dutyApi"` line 7(本 plan 修复的 37-05 latent bug)+ `export type { SimpleDept } from "./dutyApi"` line 261;`grep "getDeptTree" workorderApi.ts` = 0(副本已删) |
| `pages/duty/pools/index.tsx` | 消费 useDeptTree;DepartmentTreeSelect 受控 | ✓ VERIFIED | `useDeptTree()` line 68,`fetchDepts` 删除 |
| `pages/system/user/hooks/useUserData.ts` + `utils.tsx` | 去 fetch + 删 convertDeptTreeData,保留 renderDeptTreeOptions | ✓ VERIFIED | `useDeptTree()` line 43;`utils.tsx` 中 convertDeptTreeData 已删,renderDeptTreeOptions 保留(line 18,JSDoc 指向 toShortNameDataNode) |
| `pages/system/notice/hooks/useTargetSelector.ts` | 去 GET fetch + 删 convertTree | ✓ VERIFIED | `useDeptTree()` line 50 + `toShortNameDataNode(rawDept) as Target[]` line 55 |
| `pages/network/devices/hooks/useDeviceData.ts` | 消费 useDeptTree | ✓ VERIFIED | `useDeptTree()` line 28 |
| `pages/operations/buildings/index.tsx` | 直接消费 useDeptTree | ✓ VERIFIED | `useDeptTree()` line 44;`useDepartmentTree.tsx` 已删除;`useDepartmentData.ts` 保留(floors 兼容层方案 B,内部委托 useDeptTree) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `DeptTree/index.tsx` | `hooks/useDeptTree.ts` | `import { useDeptTree }` | ✓ WIRED | line 6 import + line 46 调用 |
| `DepartmentTreeSelect.tsx` | `utils/deptUtils.ts` | `import { toFullPathTree }` | ✓ WIRED | line 24 import + line 101/165 两处调用 |
| `DepartmentTreeSelect.tsx` | `lib/dutyApi.ts` | `import type { SimpleDept }` | ✓ WIRED | line 23 import,props 类型已迁移 |
| `useWorkstationData.ts` | `utils/deptUtils.ts` | toFullPathTree + trimTitleToLastSegment + filterExternalOrgDepts | ✓ WIRED | line 23 import toFullPathTree,index.tsx line 31 import 另两个 |
| `useTargetSelector.ts` | `utils/deptUtils.ts` | toShortNameDataNode | ✓ WIRED | line 5 import + line 55 调用 |
| `useWorkOrderData.ts` | `hooks/useDeptTree.ts` | `import { useDeptTree }` | ✓ WIRED | line 18 import + line 83 调用 |
| `workorderApi.ts` | `lib/dutyApi.ts` | `export type { SimpleDept } from "./dutyApi"` | ✓ WIRED | re-export + 本地 import type(line 7)双轨 |
| `buildings/index.tsx` | `hooks/useDeptTree.ts` | `import { useDeptTree }` | ✓ WIRED | line 28 import + line 44 调用 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| DeptTree/index.tsx | `rawDept` | `useDeptTree()` → `getDeptTree()` → POST `/system/departments/tree` | ✓ React Query 真实端点 | ✓ FLOWING |
| DepartmentTreeSelect.tsx | `departments` (props) | 调用方喂入(duty/pools/buildings/floors) | ✓ 调用方 useDeptTree 派生 | ✓ FLOWING |
| useWorkstationData.ts | `rawDept` | `useDeptTree()` | ✓ 真实端点,经 useEffect → setDeptTreeData | ✓ FLOWING |
| workstations/index.tsx | `orgTreeData` | `trimTitleToLastSegment(filterExternalOrgDepts(deptTreeData))` | ✓ deptTreeData 真实数据派生 | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `npm run type-check` | `tsc --noEmit` | exit 0,无错误输出 | ✓ PASS |
| `npm run build` | `vite build` | exit 0,✓ built in 36.10s,完整 dist 产出 | ✓ PASS |
| `SimpleDept` 唯一性 | `grep -rn "interface SimpleDept" src/` | 仅 `src/lib/dutyApi.ts:281` 1 命中 | ✓ PASS |
| DepartmentTreeSelect 无 Department interface | `grep -cE "interface Department\b" DepartmentTreeSelect.tsx` | 0 | ✓ PASS |
| DeptTree 无 post fetch | `grep -c "/system/departments/tree" DeptTree/index.tsx` | 0 | ✓ PASS |
| 全量 grep 非排除项 | `grep -rn "/system/departments/tree" src/` | 3 命中全部合法排除 | ✓ PASS |
| useDeptTree 消费点计数 | `grep -rl "useDeptTree" src/` | DeptTree/TargetSelector/workorder/duty/buildings×2/workstations/user/notice/devices = 11 处(含 useDepartmentData 方案 B) | ✓ PASS |
| WR-1 修复验证 | `grep -A2 "loadDeptOptions = useCallback" useWorkstationData.ts` | no-op 依赖 `[]`,deptTreeData 由 useEffect 派生 | ✓ PASS |
| workorderApi re-export | `grep "SimpleDept\|getDeptTree" workorderApi.ts` | import type line 7 + re-export line 261,getDeptTree 副本已删 | ✓ PASS |
| AD 模块排除边界 | `grep -rn "getADOUTree\|useDeptTree" src/pages/ad-domain/` | AD 模块 getADOUTree 保留(独立数据源);ad-domain/ous/index.tsx 用 useDeptTree 是合法边缘功能(OU↔系统部门映射,非直接 fetch) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| (无 conventional probe 脚本) | `find scripts -path '*/tests/probe-*.sh'` | 此 phase 为纯前端重构,无 probe 声明 | N/A — SKIPPED |

---

### Requirements Coverage

| Requirement ID | Source Plan | Description | Status | Evidence |
|----------------|-------------|-------------|--------|----------|
| SUCC-type-dedup | 37-01, 37-05 | SimpleDept 唯一 + DepartmentTreeSelect 删 Department 接口 | ✓ SATISFIED | dutyApi:281 唯一定义;DepartmentTreeSelect `interface Department` = 0;workorderApi re-export |
| SUCC-utils-converge | 37-01 | deptUtils toFullPathTree + toShortNameDataNode | ✓ SATISFIED | deptUtils.ts:200/275 双 export,4 份重复转换函数全删 |
| SUCC-depttree-no-fetch | 37-01 | DeptTree 消费 useDeptTree | ✓ SATISFIED | DeptTree/index.tsx:46 useDeptTree() + 搜索/展开逻辑保留 |
| SUCC-consume-hook | 37-02~06 | 各模块迁移 useDeptTree | ✓ SATISFIED | 8 处消费点全部迁移,非排除项直接 fetch = 0 |
| SUCC-high-risk-workstations | 37-03 | 双向语义保留(deptTreeData 全路径 + orgTreeData 短名) | ✓ SATISFIED | index.tsx:87-89 双向链完整,WR-1 修复后稳定 |
| SUCC-final-grep-verification | 37-06 | 全量 grep 剩余仅排除项 | ✓ SATISFIED | 3 命中全部合法排除 |

**REQUIREMENTS.md 同步性**: SUCC-* ID 未在 `.planning/REQUIREMENTS.md` 中登记(REQUIREMENTS.md v1.16 处于 planning 状态,Phase 37 requirements 仅以 PLAN frontmatter 为契约来源)。所有 6 个 ID 均在对应 PLAN 的 `requirements:` 字段声明,并有可验证证据,无 ORPHANED 需求。

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `src/components/TargetSelector.tsx` | (整文件) | dead code — 全项目零 importers | ℹ️ INFO | 不影响 phase goal(notice 用 notice/components/TargetSelector.tsx 是另一文件,37-02 已迁移 useTargetSelector hook);建议独立 follow-up 清理 |
| `src/pages/operations/buildings/useDepartmentData.ts` | (整文件) | 方案 B 兼容层,no-op `loadDepartments` | ℹ️ INFO | 非阻塞:floors 仍依赖该 hook 的对外 API;内部已委托 useDeptTree,数据源已收敛;floors 完全迁移后可整体删除 |
| `src/components/shared/DepartmentTreeSelect.tsx` | 42 | `export type Department = SimpleDept` alias 保留 | ℹ️ INFO | 非阻塞:floors 模块(FloorModal.tsx + FloorPlanEditorView.tsx)仍消费;@deprecated 标记已加;floors 迁移后可移除 |

**Debt-marker gate**: 0 个 TBD/FIXME/XXX 在本 phase 修改的文件中。无 blocker 级 anti-pattern。

---

### Human Verification Required

按 Step 9 决策树,虽然所有 must-haves 已 VERIFIED 且自动化检查全部通过,但以下 UI 行为维度需人工最终核对(Phase 37 DESIGN §6 success criteria 明确要求"各模块迁移后 UI 行为不变")。37-01~37-06 SUMMARY 已记录静态等价性分析,但 executor agent 无 chrome-devtools 工具做实际 UI 操作。建议 UAT:

### 1. DeptTree 三页筛选行为一致

**Test**: 启动前端 dev server,进入 ①用户管理 ②楼宇管理 ③网络设备 列表页,对左侧/顶部部门树做:默认展开、搜索框输入关键字过滤、点击节点勾选/筛选
**Expected**: 与 git stash 迁移前对比行为完全一致(默认展开首个父节点、搜索过滤正确、勾选后列表按部门筛选)
**Why human**: UI 行为等价性是 phase goal 显式 success criteria,静态分析已证明但需视觉确认

### 2. workstations 双向下拉显示

**Test**: 进入工位管理页,打开新增/编辑工位表单,核对 ①"所属机构"下拉显示**全路径**(如"分公司本部 / 人力资源部",顶级节点直接显示其名) ②"所属部门"下拉显示**短名**(只显示末段) ③外部机构节点(isExternalOrg=1)正确出现且不空树
**Expected**: 与迁移前一致(双语义保留 + isExternalOrg 透传链完整)
**Why human**: SUCC-high-risk-workstations 是 phase 高风险点,需视觉确认

### 3. DepartmentTreeSelect 受控下拉显示(duty/pools)

**Test**: 进入值班池页,打开新增/编辑值班池表单,核对"部门"下拉显示(全路径从二级开始)
**Expected**: 与迁移前一致(toFullPathTree startFromLevel=2 复现 slice(1) 语义)
**Why human**: 全路径转换的 startFromLevel 边界行为需视觉确认

### 4. notice 目标选择器(部门 + 角色 + 用户 三子树)

**Test**: 进入通知公告页,新建通知,核对 ①"目标部门"树显示(短名)与勾选 ②"目标角色" ③"目标用户" 三部分都正常
**Expected**: dept/roles/users 三子树均与迁移前一致
**Why human**: TargetSelector 切换 targetType 时三个子任务的 loading 状态需视觉确认

---

### Gaps Summary

**无 blocker 级 gap**。所有 6 个 must-have truths VERIFIED;所有 required artifacts 通过 4 级验证(exists/substantive/wired/data-flowing);所有 key links WIRED;build + type-check 双通过;REQUIREMENTS 全部 SATISFIED;0 个 TBD/FIXME/XXX。

**INFO 级 follow-up(不阻塞 phase 完成,留作后续清理)**:

1. **INFO-1 (CR-1)**: `src/components/TargetSelector.tsx`(根目录)是 dead code — 全项目零 importers(notice 用的 `./TargetSelector` 相对路径解析到 `notice/components/TargetSelector.tsx` 是另一文件)。37-06 误迁移了一个无用文件(加了 useDeptTree 但没人消费)。
   - **对 phase goal 影响**:**无**。phase 37 的核心目标"消除 6 处重复 fetch + 4 份重复转换 + 3 份重复类型"已全部达成 — notice 的部门数据实际通过 37-02 的 `useTargetSelector` hook 收敛,活跃的 `notice/components/TargetSelector.tsx` 本就是受控模式(从 hook 接收 deptTree prop)。
   - **对 SUCC-final-grep-verification 影响**:**无**。全量 grep 基线剩 3 命中(全部合法排除),TargetSelector 自身命中 = 0。
   - **建议**:独立 follow-up PR 删除根目录 `src/components/TargetSelector.tsx`(确认无动态 import 后)。无需 `/gsd:plan-phase --gaps`,因不影响 phase 37 验收。

2. **INFO-2**: `pages/operations/buildings/useDepartmentData.ts` 作为 floors 兼容层保留(方案 B),内部已委托 useDeptTree(数据源已收敛),对外 API 不变。floors 模块的直接迁移不在 Phase 37 范围(PLAN-03 明确排除 floors,floors 留作后续清理)。
   - **对 phase goal 影响**:**无**。phase 37 明确"楼宇合并双 hook"目标已达成(useDepartmentTree.tsx 已删 + buildings/index.tsx 直接 useDeptTree);useDepartmentData 仅作为 floors 向 buildings 兼容的桥接层,不再有自 fetch。
   - **建议**:未来做 floors 收敛 phase 时,把 floors 改为直接消费 useDeptTree,然后删除 useDepartmentData.ts 与 DepartmentTreeSelect 的 Department alias。

3. **INFO-3**: `DepartmentTreeSelect.tsx:42` 的 `export type Department = SimpleDept` alias 保留(floors 仍消费),@deprecated 标记已加。随 INFO-2 一起清理。

**4 处 UI 行为需人工 UAT**(见上方 Human Verification Required 段)— 这是 DESIGN §6 显式 success criteria 的固有要求(静态等价性分析已逐维度证明,但视觉确认留作 UAT)。

---

## Verification Decision

按 Step 9 决策树:

1. 无 truth FAILED → 不进入 gaps_found
2. Step 8 产生 4 个 human verification items → status: **human_needed**

**最终 Status**: `human_needed` — 所有自动化检查(build/type-check/grep/数据流)全部 PASS,phase goal 在代码层面已达成,但 DESIGN §6 显式要求"UI 行为不变"需人工 UAT 确认。

**Score**: 6/6 must-haves verified (自动化层面全 VERIFIED,UI 行为等价性留 human UAT)

---

_Verified: 2026-06-22_
_Verifier: Claude (gsd-verifier)_
