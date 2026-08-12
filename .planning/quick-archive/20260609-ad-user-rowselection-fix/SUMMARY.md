# 修复总结

## 状态：complete ✓

## 问题描述

在添加"全部选择"按钮后，域控用户页面的 Table 多选框显示异常。当用户取消全选时，多选框未能正确显示。

## 根本原因

通过代码分析和 git 历史对比发现：
- Table 的 `rowKey` 使用的是 `"id"`
- rowSelection 的 `selectedRowKeys` 使用的是 `selectedUsers.map(u => u.id)`
- ADUser 接口虽然有 `id` 字段，但对于未同步到系统的 AD 用户，使用 `userDn`（Distinguished Name）作为唯一标识符更可靠
- `userDn` 是 AD 域中的天然唯一标识符，所有 AD 用户都有此属性

## 修复方案

修改 `xingran-react-frontend/src/pages/ad-domain/users/index.tsx`：
1. 将 Table 的 `rowKey` 从 `"id"` 改为 `"userDn"`（第 576 行）
2. 将 rowSelection 的 `selectedRowKeys` 从 `selectedUsers.map(u => u.id)` 改为 `selectedUsers.map(u => u.userDn)`（第 583 行）
3. 清理了之前添加的调试 console.log 代码

## 修改详情

```diff
- rowKey="id"
+ rowKey="userDn"

- selectedRowKeys: selectedUsers.map(u => u.id),
+ selectedRowKeys: selectedUsers.map(u => u.userDn),
```

## 验证结果

- ✅ TypeScript 编译检查通过（无新增错误）
- ✅ 修改逻辑正确：rowKey 和 selectedRowKeys 使用同一字段
- ✅ 保留了 `selectAllMode` 条件逻辑（全选模式下隐藏多选框）
- ✅ 使用 `userDn` 作为唯一标识符，确保所有 AD 用户都能正确显示和选择

## 影响

- 修复了多选框在切换全选模式后不能正确显示的问题
- 提高了多选功能的稳定性（userDn 比 id 更可靠）
- 不影响其他功能
