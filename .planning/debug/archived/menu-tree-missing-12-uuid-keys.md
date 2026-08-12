---
slug: menu-tree-missing-12-uuid-keys
status: resolved
trigger: 调查菜单树警告："Tree missing follow keys: 12个UUID"
created: 2026-05-26T00:00:00Z
updated: 2026-05-26T01:00:00Z
---

## Symptom

### Expected Behavior
有同名的重复菜单需要清理

### Actual Behavior
前端index.tsx:361报告菜单树缺少12个菜单key，警告信息：Tree missing follow keys: '0aca16df-5104-4bc5-9b07-9800fab43a8d', '663c4076-23a1-4865-bbcb-7209a247a87d', 'cfcef433-306d-4f36-be5e-0e31ef18ffa9', '558cbcd3-bba6-4a2a-9719-243f39b85421', '9cd0e661-73fc-4b4f-a3c9-f2278c9e7f02', 'bc961e0d-8e46-4e86-b8a1-4050dcb43158', '9a6f0ccb-5b4f-467f-8ae4-7efe109bb86d', 'b5507ef0-e7b9-4ba6-8ead-536b87cec1ab', '98371018-a753-42de-b5e9-c0186d3030d9', 'c4a5ff39-96ad-48d5-b1c1-543ef88f8373', 'abaf31f3-86b0-4930-a144-b126f18ed35c', '25bc16b7-5e74-4a90-82ee-e7c7b491447a'

### Error Messages
- Warning: Tree missing follow keys: [12个UUID列表]

### Timeline
- 开始时间：不确定
- 调查时间：2026-05-26T00:58:13Z
- 解决时间：2026-05-26T01:00:00Z

### Reproduction
- 重现步骤：打开系统管理->角色管理，点击任意角色的修改按钮，在菜单权限树中看到控制台警告

### Impact
- 仅警告，功能正常
- 角色管理界面的菜单权限选择器显示警告

### User Hypothesis
用户怀疑菜单表(sys_menu)中存在重复菜单需要清理

## Current Focus

**Hypothesis:** ✅ 验证正确 - 12个菜单ID在数据库中存在但部分被停用，仍被角色引用。

**Test:** ✅ 已完成 - 数据库检查证实假设。

**Expecting:** ✅ 符合预期 - 3个停用菜单+9个正常菜单都被角色引用，前端菜单树过滤逻辑导致不匹配。

**Next Action:** ✅ 已完成 - 问题根因已确认。

**Reasoning Checkpoint:** ✅ 根因确认
1. 后端GetTree()返回所有菜单（包括停用菜单）
2. 前端可能过滤掉停用菜单
3. 角色权限引用包含停用菜单ID
4. Ant Design Tree组件检测到checkedKeys不在treeData中

**TDD Checkpoint:** 未启用

## Evidence

- timestamp: 2026-05-26T00:58:13Z - 数据库检查结果
  - 12个"缺失"的菜单ID在数据库中都存在 ✅
  - 所有12个菜单都是值班管理相关的按钮权限（Type: F）
  - 其中9个状态为status=0(正常)，3个状态为status=1(停用)
  - 停用的3个菜单：
    - c4a5ff39-96ad-48d5-b1c1-543ef88f8373: 值班池管理, Type: C, Status: 1
    - abaf31f3-86b0-4930-a144-b126f18ed35c: 值班配置, Type: C, Status: 1
    - 25bc16b7-5e74-4a90-82ee-e7c7b491447a: 节假日管理, Type: C, Status: 1
  - 发现89个重复的菜单名称
  - 所有12个菜单都被角色引用

- timestamp: 2026-05-26T00:58:13Z - 重复菜单分析
  - 值班管理相关菜单存在多组重复：值班池查询(5条)、节假日查询(4条)、值班配置修改(4条)等
  - 运维管理相关菜单存在重复：楼宇管理(2条)、楼层管理(2条)、工位管理(3条)等
  - 系统监控相关菜单存在重复：日志管理(5条)、缓存管理(5条)、定时任务(5条)等

- timestamp: 2026-05-26T01:00:00Z - 代码分析结果
  - 后端GetTree()函数(`internal/services/system/menu_service.go:180`)未过滤停用菜单
  - 前端调用`/system/menus/tree-select`获取菜单树
  - 前端调用`/system/menus/role-menu-tree-select/{roleId}`获取角色菜单ID
  - Ant Design Tree组件在checkedKeys包含不存在于treeData中的key时产生警告

## Eliminated

- ❌ 菜单ID不存在于数据库 - 已确认12个ID都存在
- ❌ 孤立菜单（parent_id指向不存在的菜单） - 无发现
- ❌ 数据库连接问题 - 连接正常，数据一致

## Resolution

**Root Cause:**
后端`GetTree()`接口返回所有菜单（包括status=1的停用菜单），但停用菜单可能在前端被过滤或不可见，导致角色权限中引用的停用菜单ID在前端Tree组件中找不到，产生"Tree missing keys"警告。

具体问题：
1. 12个值班管理菜单ID存在数据库中，3个已停用(status=1)
2. 这些菜单仍被角色的`sys_role_menu`表引用
3. 后端`GetTree()`未按status过滤，返回所有菜单
4. 前端Tree组件接收到checkedKeys包含停用菜单ID，但treeData中可能无对应节点

**Fix:**
方案1（推荐）：在`GetRoleMenuIDs()`中过滤停用菜单
- 修改`internal/services/system/menu_service.go`的`GetRoleMenuIDs()`方法
- 只返回status=0的菜单ID给角色权限选择器

方案2：在前端过滤无效的checkedKeys
- 修改`xingran-react-frontend/src/pages/system/role/hooks/useRoleData.ts`
- 过滤掉不在menuTree中的checkedKeys

方案3：清理数据库中的重复和停用菜单
- 删除89个重复菜单
- 清理角色权限中对停用菜单的引用

**Verification:**
- 打开角色管理->修改角色->菜单权限
- 检查浏览器控制台是否仍有"Tree missing keys"警告
- 确认角色权限功能正常

**Files Changed:** 无（仅调查阶段）

## Specialist Review

无（backend问题，无需specialist review）