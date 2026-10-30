# 修复用户管理页面部门路径显示 - 任务总结

## 状态
complete

## 描述
修复用户管理页面部门列只显示最后一层部门名称的问题，现在显示从二级部门开始的完整路径。

## 实施的修改

### 后端修改
1. **User 模型** (`internal/models/user.go`)
   - 添加 `DeptPath *string` 字段，用于存储完整部门路径（非持久化）

2. **用户服务** (`internal/services/system/user_service.go`)
   - 在 `List` 方法中添加 `buildDepartmentPaths` 调用
   - 实现 `buildDepartmentPaths` 方法：
     - 批量查询用户的部门及其祖先部门
     - 从二级部门开始构建完整路径（跳过顶级部门）
     - 格式：`分公司本部/科技创新部/基础运维科`

### 前端修改
1. **User 类型** (`xingran-react-frontend/src/types/system.ts`)
   - 添加 `deptPath?: string` 字段

2. **用户管理页面** (`xingran-react-frontend/src/pages/system/user/index.tsx`)
   - 修改部门列渲染逻辑，优先显示 `deptPath`，回退到 `deptName`

## 技术要点
- 使用 GORM 的 `gorm:"-"` 标记非持久化字段
- 批量查询优化：一次查询所有祖先部门，避免 N+1 问题
- 向后兼容：当 `deptPath` 为空时，前端回退到 `deptName`
- 路径构建从二级部门开始（ancestors[1:]），符合用户需求

## 测试验证
- 后端编译通过
- 前端类型检查通过
- 向后兼容性保持

## 影响范围
- 用户列表 API 返回新增 `deptPath` 字段
- 用户管理表格部门列显示更完整的信息
