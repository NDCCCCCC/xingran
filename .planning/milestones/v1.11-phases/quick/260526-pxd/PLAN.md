# 用户管理表格显示完整部门路径

## 问题描述
用户管理表格需要显示完整部门路径（如：分公司本部/科技创新部/基础运维科），而不是只显示最末级部门。

## 技术方案
参考工位服务的 SQL JOIN 模式，同时构建完整部门路径：

1. **后端修改**：
   - 使用 SQL JOIN 获取部门名称和 ancestors
   - 在 List 方法中添加完整路径构建逻辑
   - 新增 `deptFullName` 字段存储完整路径

2. **前端修改**：
   - 表格列显示 `deptFullName`（完整路径）
   - 模态框继续使用 `deptId`（部门选择器）

3. **清理**：
   - 移除之前的 `deptPath` 相关代码
   - 移除 `buildDepartmentPaths` 方法

## 实施步骤
1. 修改 User 模型：添加 `deptFullName` 字段，移除 `deptPath`
2. 修改 List 方法：使用 SQL JOIN + 构建完整路径
3. 修改前端：表格列显示 `deptFullName`
4. 清理无用代码
