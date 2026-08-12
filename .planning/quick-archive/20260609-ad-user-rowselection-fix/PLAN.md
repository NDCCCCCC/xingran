# 修复域控用户页面多选框显示问题

## 问题描述

在添加"全部选择"按钮后，当用户取消全选时，Table 的 rowSelection 没有正确恢复，导致多选框消失。

## 根本原因

通过代码分析和 git 历史对比发现：

1. **原始代码**（提交 e179faa）使用 `userDn` 作为 `selectedRowKeys`：
   ```tsx
   rowSelection={{
     selectedRowKeys: selectedUsers.map(u => u.userDn),
     onChange: (selectedKeys, selectedRows) => {
       setSelectedUsers(selectedRows);
     },
   }}
   ```

2. **当前代码**使用 `id` 作为 `selectedRowKeys`：
   ```tsx
   rowSelection={
     selectAllMode
       ? undefined
       : {
           selectedRowKeys: selectedUsers.map(u => u.id),
           onChange: (selectedKeys, selectedRows) => {
             setSelectedUsers(selectedRows);
           },
         }
   }
   ```

3. **问题分析**：
   - `userDn` 是 AD 域中的唯一标识符（Distinguished Name），所有 AD 用户都有
   - `id` 是后端数据库中的字段，可能对未同步到系统的 AD 用户来说不稳定或为空
   - 当使用 `id` 作为 rowKey 和 selectedRowKeys 时，如果 id 值异常会导致多选框行为异常

## 修复方案

恢复使用 `userDn` 作为 `selectedRowKeys`，确保：
1. 所有 AD 用户都有有效的 `userDn` 值
2. 多选框能够正确显示和工作
3. 保留 `selectAllMode` 条件逻辑（全选模式下隐藏多选框）

## 修改文件

- `xingran-react-frontend/src/pages/ad-domain/users/index.tsx`
  - 第 583 行：将 `selectedUsers.map(u => u.id)` 改为 `selectedUsers.map(u => u.userDn)`
  - 确保使用 `userDn` 作为 rowKey 也要一致

## 执行步骤

1. 检查当前 Table 的 rowKey 属性
2. 修改 rowSelection 配置，使用 userDn 代替 id
3. 测试验证多选框功能
4. 提交修复
