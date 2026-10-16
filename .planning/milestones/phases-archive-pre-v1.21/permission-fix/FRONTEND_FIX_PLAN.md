# 前端角色权限显示问题修复

## 问题分析

用户报告前端角色管理页面显示"普通用户"角色没有任何权限，但数据库中该角色有17个权限。

## 网络请求分析

用户提供的网络请求显示：
1. **第一次请求**：返回17个checkedKeys（正确）
2. **第二次请求**：返回空checkedKeys（问题）

## 根本原因

**推测**：可能有重复调用`loadRoleMenus`，其中一次使用了错误的roleId（可能是undefined或空字符串）。

## 可能原因

1. **React严格模式**：在开发环境下，useEffect可能被调用两次
2. **Modal打开时的副作用**：`handleModalOpenChange`可能触发了额外的API调用
3. **Tree组件的onCheck事件**：可能在某些情况下触发了重新加载
4. **内联函数引用变化**：`loadRoleMenus`作为内联函数在每次渲染时都是新的引用，可能导致依赖项重新创建

## 修复方案

### 方案1：使用useCallback包装loadRoleMenus

```typescript
// 在index.tsx中，将内联的loadRoleMenus改为useCallback
const loadRoleMenusCallback = useCallback(async (roleId: string) => {
  const result = await post(`/system/menus/role-menu-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
  return result.data.checkedKeys || [];
}, []); // 空依赖数组，因为post函数是稳定的

// 然后在传递给useRoleActions时使用这个稳定函数
} = useRoleActions({
  loadRoles,
  loadStatistics,
  loadRoleMenus: loadRoleMenusCallback, // 使用useCallback包装的版本
  loadRoleDepts: async (roleId: string) => { ... },
  // ...
});
```

### 方案2：添加防重复调用逻辑

在`loadRoleMenus`中添加缓存或防抖：

```typescript
const loadingMenusRef = useRef(new Set<string>());

const loadRoleMenus = useCallback(async (roleId: string) => {
  if (!roleId) {
    console.warn('[loadRoleMenus] roleId is empty, skipping request');
    return [];
  }
  
  if (loadingMenusRef.current.has(roleId)) {
    console.warn('[loadRoleMenus] Already loading roleId:', roleId);
    return [];
  }
  
  loadingMenusRef.current.add(roleId);
  try {
    const result = await post(`/system/menus/role-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
    return result.data.checkedKeys || [];
  } finally {
    loadingMenusRef.current.delete(roleId);
  }
}, []);
```

### 方案3：检查并修复Modal事件处理

确保`handleModalOpenChange`不会触发额外的API调用，并且只在Modal关闭时（open=false）清理状态。

## 建议的完整修复

结合方案1和方案2，既保证函数引用稳定，又防止重复调用。

## 待办事项

- [ ] 确认第二次请求的URL和参数
- [ ] 实施修复方案（useCallback + 防重复调用）
- [ ] 测试修复效果
- [ ] 验证"普通用户"角色权限正确显示
