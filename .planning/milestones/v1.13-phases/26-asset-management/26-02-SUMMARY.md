# Plan 26-02: Asset Service Layer - 执行摘要

## 执行时间
2026-06-08

## 目标
实现 Asset 服务层，包含 CRUD 操作、UUID 验证和引用解析功能。

## 完成的任务

### Task 1: 创建 Asset 服务
- 文件: `internal/services/operations/asset_service.go`
- 实现了 AssetService 接口，包含以下方法：
  - `Create` - 创建资产，验证部门和用户存在性，验证设备序列号唯一性
  - `Update` - 更新资产，包含完整的验证逻辑
  - `Delete` - 删除资产（软删除）
  - `GetByID` - 根据 ID 获取资产
  - `List` - 查询资产列表，支持分页和筛选
  - `BatchDelete` - 批量删除资产
- 验证功能：
  - `validateDept` - 验证部门 UUID 格式和存在性
  - `validateUser` - 验证用户 UUID 格式和存在性
  - `validateDeviceSNUnique` - 验证设备序列号唯一性
- 筛选支持：
  - 设备序列号模糊查询
  - 设备型号模糊查询
  - 部门筛选（包含子部门）
  - 用户筛选
  - 状态筛选
  - 拟报废状态筛选

### Task 2: 扩展 Reference Resolver
- 文件: `internal/services/operations/reference_resolver.go`
- 添加了通用方法：
  - `ResolveDept` - 通过部门名称或编码解析部门 ID
  - `ResolveUser` - 通过用户名或昵称解析用户 ID
- 添加了资产专用方法：
  - `ResolveAssetDept` - 资产导入时解析部门 ID
  - `ResolveAssetUser` - 资产导入时解析用户 ID

## 关键实现细节

1. **UUID 验证模式**:
   - 复用 `building_service.go` 中的 `uuidPattern` 常量
   - 使用正则表达式验证 UUID 格式

2. **部门筛选（包含子部门）**:
   - 使用 `ancestors` 字段匹配子部门
   - 查询条件：`id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?`

3. **错误处理**:
   - 使用 `apperrors.ParamInvalid` 处理验证错误
   - 提供清晰的错误消息

4. **分页和排序**:
   - 复用 `extractPagination` 和 `calculateOffset` 辅助函数
   - 默认排序：`created_at DESC`（最新在前）

## 验证结果

- ✅ AssetService 接口定义完整
- ✅ 所有 CRUD 方法实现
- ✅ dept_id 和 user_id UUID 格式验证
- ✅ 部门和用户存在性验证
- ✅ DeviceSN 唯一性验证（创建和更新）
- ✅ List 方法支持所有必需的筛选条件
- ✅ applyDeptFilter 包含子部门查询
- ✅ ReferenceResolver 添加了 ResolveAssetDept 和 ResolveAssetUser
- ✅ 构建无错误

## 偏差说明
无偏差。

## 后续步骤
1. 执行计划 26-03: 创建 Asset API 处理器和路由
2. 执行计划 26-04: 配置 Excel 导入/导出
3. 执行计划 26-05: 创建前端资产列表页面
4. 执行计划 26-06: 配置菜单和权限

## 自检结果
**状态**: ✅ PASSED

所有任务已完成，代码编译无错误，服务层实现符合项目规范。
