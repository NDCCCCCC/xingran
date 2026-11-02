# Plan: Fix Workstation Filter Loss on Device Sync

## Problem Description

在工位管理页面中，当用户点击"同步AD设备"或"同步资产设备"按钮后，表格的筛选条件（如楼层、名称、状态等）会被重置，导致用户体验不佳。

## Root Cause Analysis

文件：`xingran-react-frontend/src/pages/operations/workstations/index.tsx`

当设备同步完成时，`WorkstationDeviceTable` 组件会调用 `onDeviceChange` 回调函数（即父组件的 `refreshData` 函数）。

当前 `refreshData` 函数（第 139-142 行）：
```tsx
const refreshData = useCallback(() => {
  loadWorkstations(); // ❌ 没有传递搜索参数
  loadStatisticsFromHook(selectedDeptId || undefined);
}, [loadWorkstations, loadStatisticsFromHook, selectedDeptId]);
```

问题：`loadWorkstations()` 被调用时没有传递任何参数，导致使用空的搜索条件，用户当前的筛选条件被丢失。

## Solution

修改 `refreshData` 函数，使其使用当前搜索表单的值来加载数据，类似 `handleSearch` 的实现方式：

```tsx
const refreshData = useCallback(() => {
  // 获取当前搜索表单的值
  const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
  const searchParams: Record<string, unknown> = {};
  Object.keys(formValues).forEach(key => {
    const value = formValues[key];
    if (value !== undefined && value !== null && value !== '') {
      searchParams[key] = value;
    }
  });

  // 使用当前筛选条件加载数据
  loadWorkstations(searchParams);
  loadStatisticsFromHook(selectedDeptId || undefined);
}, [loadWorkstations, loadStatisticsFromHook, selectedDeptId, searchForm]);
```

## Implementation Steps

1. **修改 `refreshData` 函数**
   - 文件：`xingran-react-frontend/src/pages/operations/workstations/index.tsx`
   - 位置：第 139-142 行
   - 操作：添加从 `searchForm` 获取当前筛选值的逻辑，并传递给 `loadWorkstations`
   - 依赖数组：添加 `searchForm`

2. **测试验证**
   - 启动前端开发服务器
   - 进入工位管理页面
   - 设置筛选条件（如选择楼层、输入名称）
   - 点击"同步AD设备"或"同步资产设备"
   - 验证筛选条件是否保持不变

## Files to Modify

1. `xingran-react-frontend/src/pages/operations/workstations/index.tsx` - 修改 `refreshData` 函数

## Success Criteria

- [ ] 同步设备后，表格的筛选条件保持不变
- [ ] 统计数据正确更新
- [ ] 无 TypeScript 编译错误
- [ ] 无控制台错误或警告
