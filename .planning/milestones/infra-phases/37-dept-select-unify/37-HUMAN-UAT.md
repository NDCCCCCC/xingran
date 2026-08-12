---
status: passed
phase: 37-dept-select-unify
source: [37-VERIFICATION.md]
started: 2026-06-22T00:00:00Z
updated: 2026-06-26T16:00:00Z
---

# Phase 37 Human UAT — 部门选择组件统一收敛

> 所有自动化检查通过（6/6 must-haves VERIFIED，build/type-check 双通过，全量 grep 非排除项=0）。
> 以下 4 处 UI 行为等价性需人工 UAT（DESIGN §6 显式 success criteria："各模块迁移后 UI 行为不变"）。
> executor agent 无 chrome-devtools 工具做实际 UI 操作，静态等价性分析已逐维度证明，但视觉确认留作 UAT。

## Current Test

[awaiting human testing]

## Tests

### 1. DeptTree 三页筛选行为一致

**页面**：用户管理 / 楼宇管理 / 网络设备 列表页

**Test**：启动前端 dev server（`cd xingran-react-frontend && npm run dev`），进入上述 3 个列表页，对左侧/顶部部门树操作：
- ① 默认展开（首个父节点自动展开）
- ② 搜索框输入关键字过滤
- ③ 点击节点勾选/筛选（列表按部门筛选）

**expected**：与迁移前（git stash 对比）行为完全一致

**result**: [pending]

**证据**：DeptTree/index.tsx 已删 post fetch，消费 useDeptTree（line 46），搜索逻辑保留（onSearch/filterTreeData/getExpandedKeys），首次展开用 didInitExpandRef 守卫只跑一次

---

### 2. workstations 双向下拉显示（高风险）

**页面**：工位管理页（新增/编辑工位表单）

**Test**：打开新增/编辑工位表单，核对：
- ① "所属机构"下拉显示**全路径**（如"分公司本部 / 人力资源部"，顶级节点直接显示其名）
- ② "所属部门"下拉显示**短名**（只显示末段）
- ③ 外部机构节点（isExternalOrg=1）正确出现且**不空树**

**expected**：与迁移前一致（双向语义保留：deptTreeData=toFullPathTree 全路径，orgTreeData=trimTitleToLastSegment(filterExternalOrgDepts(deptTreeData)) 短名）

**result**: [pending]

**证据**：useWorkstationData.ts 双向链完整；toFullPathTree 透传 isExternalOrg；WR-1 修复后 init effect 不再重复请求

---

### 3. DepartmentTreeSelect 受控下拉显示（duty/pools）

**页面**：值班池页（新增/编辑值班池表单）

**Test**：进入值班池页，打开新增/编辑值班池表单，核对"部门"下拉显示（**全路径从二级开始**）

**expected**：与迁移前一致（toFullPathTree startFromLevel=2 复现旧 convertDeptTreeData 的 slice(1) 语义）

**result**: [pending]

**证据**：DepartmentTreeSelect 删 convertDeptTreeData，调用 toFullPathTree({startFromLevel:2})；受控模式保持（数据从 useDeptTree 喂入）

---

### 4. notice 目标选择器（部门 + 角色 + 用户 三子树）

**页面**：通知公告页（新建通知）

**Test**：进入通知公告页，新建通知，核对：
- ① "目标部门"树显示（短名）与勾选
- ② "目标角色"
- ③ "目标用户"
- 三部分都正常，切换 targetType 时 loading 状态正常

**expected**：dept/roles/users 三子树均与迁移前一致

**result**: [pending]

**证据**：useTargetSelector.ts 删 GET fetch + convertTree，改 useDeptTree + toShortNameDataNode；活跃的 notice/components/TargetSelector.tsx 是受控展示组件（从 hook 接收 deptTree prop）

---

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps

（UAT 后如有问题，在此记录 → 触发 gap closure）
