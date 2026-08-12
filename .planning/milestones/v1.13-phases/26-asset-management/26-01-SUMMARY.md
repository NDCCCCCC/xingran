# Plan 26-01: Asset Model and Database Schema - 执行摘要

## 执行时间
2026-06-08

## 目标
创建资产管理的 Asset 模型和数据库迁移脚本，建立包含40个字段的完整数据结构。

## 完成的任务

### Task 1: 创建 Asset 模型
- 文件: `internal/models/asset.go`
- 包含所有40个字段，正确使用 GORM 标签
- 遵循项目规范：
  - UUID 主键 (使用 `gen_random_uuid()`)
  - 软删除 (`deleted_at`)
  - 状态值规范 (0=正常, 1=停用)
  - 可选字段使用指针类型 (`*string`, `*int`, `*time.Time`)
- `DeviceSN` 设置为唯一索引 (用于 Excel 导入时的 upsert 操作)

### Task 2: 创建数据库迁移脚本
- 文件: `internal/core/db/migrations/141_create_ops_asset_table.sql`
- 创建 `ops_asset` 表，包含所有40列
- 添加索引以优化常见查询：
  - `idx_asset_devicesn` - 设备序列号
  - `idx_asset_dept_id` - 部门关联
  - `idx_asset_user_id` - 用户关联
  - `idx_asset_status` - 状态查询
  - `idx_asset_deleted_at` - 软删除
  - `idx_asset_dept_status` - 部门过滤组合索引
- 添加注释以提供文档

## 关键实现细节

1. **字段分类**（40个字段）:
   - 核心标识: 3个 (DeviceSN, SequenceNo, FixAssetNo)
   - 设备信息: 4个 (型号、类型、中类、固定资产标识)
   - 用户关联: 4个 (领取人、责任人、p13字段)
   - 部门关联: 3个 (受益部门、部门编码)
   - 状态标识: 4个 (状态、新设备标识、打印状态、拟报废)
   - 时间字段: 6个 (接收、发放、入库、扫码、更新、上线时间)
   - 网络信息: 4个 (有线MAC、无线MAC、加域IP、加域标识)
   - 合同与属性: 2个 (合同号、设备属性)
   - 位置与归属: 6个 (扫码位置、备注、渠道、用途、机构、存放地址)
   - 机构与标准: 3个 (归属机构、标准名称、异常标识)
   - 外部关联: 4个 (使用人、部门名称、责任人岗位、扫码账号)
   - 系统关联: 3个 (dept_id, user_id, machine_user_id)
   - 状态: 1个 (status)

2. **类型约定**:
   - 可选文本字段: `*string` (允许数据库 NULL)
   - 可选数值字段: `*int`
   - 可选日期字段: `*time.Time`
   - 必填字段: 非指针类型
   - 外键字段: `*string` (UUID 存储，最大64字符)

3. **命名规范**:
   - 数据库列名: `snake_case` (符合 PostgreSQL 惯例)
   - JSON 标签: `camelCase` (前端兼容)
   - 模型字段名: `PascalCase` (Go 惯例)

## 验证结果

- ✅ Asset 模型编译无错误
- ✅ 包含所有40个字段
- ✅ GORM 标签正确
- ✅ DeviceSN 唯一索引设置正确
- ✅ 迁移文件创建成功
- ✅ 所有列名使用 snake_case
- ✅ 所有 JSON 标签使用 camelCase
- ✅ 索引创建语句正确
- ✅ 注释添加完整

## 偏差说明
无偏差。

## 后续步骤
1. 执行计划 26-02: 创建 AssetService 接口和实现
2. 执行计划 26-03: 创建 Asset API 处理器和路由
3. 执行计划 26-04: 配置 Excel 导入/导出
4. 执行计划 26-05: 创建前端资产列表页面
5. 执行计划 26-06: 配置菜单和权限

## 自检结果
**状态**: ✅ PASSED

所有任务已完成，代码编译无错误，文件结构符合项目规范。
