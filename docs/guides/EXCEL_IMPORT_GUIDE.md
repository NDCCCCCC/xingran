# Excel批量导入功能 - 实施完成指南

## 📋 概述

已成功实现楼宇、楼层、工位、信息点管理的Excel批量导入功能，采用**方案B（优雅架构）**，实现了配置驱动的引用解析、批量Upsert和自动缓存清理。

---

## ✅ 已完成的工作

### 1. 核心组件实现

#### 1.1 ReferenceResolver（引用解析器）
**文件**: `internal/services/operations/reference_resolver.go`

**功能**:
- 批量解析引用值（名称/编码 → ID）
- 按引用类型分组查询，优化性能
- 支持配置化的引用关系

**核心方法**:
```go
ResolveBatch(ctx, refs) // 批量解析
ResolveSingle(ctx, ref) // 单个解析
```

#### 1.2 BatchUpsert（批量更新器）
**文件**: `internal/services/operations/batch_upserter.go`

**功能**:
- 批量插入或更新数据
- 支持UpsertKey配置
- 自动处理重复数据

**核心方法**:
```go
Upsert(ctx, records) // 批量Upsert，返回插入/更新数量
```

#### 1.3 CacheInvalidator（缓存清理器）
**文件**: `internal/services/operations/cache_invalidator.go`

**功能**:
- 按模式清理缓存
- 支持配置化的缓存策略

**核心方法**:
```go
InvalidateByEntityType(ctx, entityType, patterns) // 按实体类型清理
InvalidateByPatterns(ctx, patterns, module)      // 按模式列表清理
```

### 2. Excel配置扩展

#### 2.1 配置结构增强
**文件**: `internal/services/operations/excel_config.go`

**新增字段**:
- `ExcelColumn.UpsertKey` - 是否作为Upsert唯一键
- `ExcelConfig.CachePatterns` - 缓存清理模式
- `ExcelConfig.UniqueKeys` - 唯一键组合

#### 2.2 各模块配置更新

**楼宇配置**:
- 使用 `orgName`（机构名称/编码）自动转换为 `orgId`
- 唯一键: `name` + `orgId`

**楼层配置**:
- 使用 `buildingName` 自动转换为 `buildingId`
- 唯一键: `buildingId` + `floorNo`

**工位配置**:
- 使用 `floorName` 自动转换为 `floorId`
- 唯一键: `floorId` + `workstationName`

**信息点配置**:
- 使用 `workstationName` 自动转换为 `workstationId`
- 唯一键: `workstationId` + `name`

### 3. ExcelService重构

**文件**: `internal/services/operations/excel_service.go`

**新增组件**:
```go
type ExcelService struct {
    db                *gorm.DB
    pwdManager        *security.PasswordManager
    referenceResolver ReferenceResolver    // 新增
    cacheInvalidator  *CacheInvalidator    // 新增
    cache             system.CacheProvider  // 新增
}
```

**重构的ImportData方法**:
- 阶段1: 收集数据和引用请求
- 阶段2: 批量解析引用（性能优化）
- 阶段3: 批量Upsert保存
- 阶段4: 清理缓存

### 4. 数据库迁移

**文件**: `internal/core/db/migrations/080_add_dept_code_field.sql`

**变更**:
- 为 `sys_dept` 表添加 `dept_code` 字段
- 添加唯一索引 `idx_sys_dept_code`
- 为现有数据生成默认编码

### 5. 模型更新

**文件**: `internal/models/dept.go`

**变更**:
```go
type Department struct {
    ...
    DeptCode string `gorm:"size:50;uniqueIndex;not null" json:"deptCode"` // 新增
    ...
}
```

### 6. 路由更新

**文件**: `internal/api/v1/operations/excel_handler.go`

**变更**: 所有 `NewExcelService` 调用添加 `core.Cache` 参数

---

## 🚀 部署步骤

### 步骤1: 执行数据库迁移

```bash
# 方式1: 使用psql命令行
psql -U postgres -d xingran_next -f internal/core/db/migrations/080_add_dept_code_field.sql

# 方式2: 使用Go迁移工具（如果配置了）
go run cmd/migrate/main.go up

# 验证迁移结果
psql -U postgres -d xingran_next -c "\d sys_dept"
```

### 步骤2: 重新编译应用

```bash
# Windows
go build -o xingran-backend.exe ./cmd/main.go

# Linux
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o xingran-backend-linux ./cmd/main.go
```

### 步骤3: 重启应用

**注意**: 请提醒你手动关闭旧程序后再启动新程序

```bash
# 停止旧程序（Ctrl+C或kill命令）

# 启动新程序
.\xingran-backend.exe
```

---

## 📝 Excel模板格式

### 楼宇导入模板

| 楼宇名称 | 所属机构名称/编码 | 城市代码 | 城市名称 | 地址 | 层级 | 状态 | 备注 |
|---------|------------------|---------|---------|------|------|------|------|
| 科技楼 | 技术部/DEPT001 | wuhan | 武汉市 | 光谷大道1号 | 具体楼宇 | 正常 | 主楼 |
| 研发楼 | 研发中心/DEPT002 | wuhan | 武汉市 | 光谷大道2号 | 具体楼宇 | 正常 | - |

**字段说明**:
- **所属机构名称/编码**: 可以填写部门名称或部门编码，系统自动查找
- **层级**: 1=城市级汇总，2=具体楼宇
- **状态**: 0=正常，1=停用

### 楼层导入模板

| 楼层名称 | 楼层号 | 所属楼宇名称 | 状态 | 备注 |
|---------|-------|------------|------|------|
| 一楼 | 1F | 科技楼 | 正常 | - |
| 二楼 | 2F | 科技楼 | 正常 | - |
| 地下一层 | B1 | 科技楼 | 正常 | 停车场 |

**注意事项**:
- **所属楼宇名称**: 必须先在楼宇管理中创建对应楼宇
- **楼层号**: 建议使用 1F, 2F, B1 等格式

### 工位导入模板

| 工位名称 | 所属楼层名称 | 所属部门名称 | 所属用户名 | 工位类型 | 状态 | 备注 |
|---------|------------|------------|-----------|---------|------|------|
| A001 | 一楼 | 技术部 | 张三 | 固定工位 | 空闲 | 靠窗 |
| A002 | 一楼 | 行政部 | - | 灵活工位 | 空闲 | - |
| M001 | 二楼 | 运维部 | 李四 | 管理工位 | 空闲 | 靠近会议室 |

> **所属部门 / 所属用户**（可选，2026-07 quick 260713-df0 扩展）：
> - `所属部门名称`：可选，按 `dept_name` → `sys_dept.id` 自动解析；空值表示不关联
> - `所属用户名`：可选，按 `username` → `sys_user.id` 自动解析；空值表示不关联
> - 解析失败时整行报错，不阻断其他行
> - 前端在工位详情页可展示关联部门/用户

**注意事项**:
- **所属楼层名称**: 必须先在楼层管理中创建对应楼层
- **工位类型**: 0=固定工位，1=灵活工位，2=管理工位
- **状态**: 0=空闲，1=占用，2=维护

### 信息点导入模板

| 信息点名称 | 信息点类型 | 关联工位名称 | 状态 | 备注 |
|-----------|-----------|------------|------|------|
| A001-网口1 | 网络信息点 | A001 | 正常 | - |
| A001-电源1 | 电源信息点 | A001 | 正常 | - |

**注意事项**:
- **关联工位名称**: 必须先在工位管理中创建对应工位
- **信息点类型**: network=网络信息点，power=电源信息点，other=其他
- **状态**: 0=正常，1=故障，2=停用

---

## 🔍 功能验证

### 验证1: 检查ReferenceResolver

```bash
# 运行测试脚本（可选）
go run scripts/tests/test_excel_import.go
```

### 验证2: 下载Excel模板

1. 登录系统
2. 进入对应管理页面（楼宇/楼层/工位/信息点）
3. 点击"导入"按钮
4. 下载Excel模板
5. 检查模板字段是否与上述格式一致

### 验证3: 测试导入流程

1. 按照模板格式填写测试数据
2. 上传Excel文件
3. 检查导入结果：
   - 成功数量
   - 失败数量
   - 错误详情（如有）

### 验证4: 验证引用解析

**测试场景**:
- 导入楼层时，填写不存在的楼宇名称 → 应报错
- 导入工位时，填写已存在的楼层名称 → 应成功
- 导入重复数据（相同楼宇名称） → 应更新而非报错

### 验证5: 验证缓存清理

```bash
# 导入前后检查Redis缓存（如果使用）
redis-cli keys "building:*"
redis-cli keys "floor:*"

# 导入后应该看到缓存被清理
```

---

## ⚠️ 重要注意事项

### 1. 导入顺序

由于存在关联关系，建议按以下顺序导入：

```
1. 楼宇（Building）
   ↓
2. 楼层（Floor）- 依赖楼宇
   ↓
3. 工位（Workstation）- 依赖楼层
   ↓
4. 信息点（InfoPoint）- 依赖工位
```

### 2. 部门编码

- 需要先为部门设置 `dept_code` 字段
- 可以通过部门管理界面设置
- 或使用SQL批量生成

### 3. 重复数据处理

- 系统使用 **Upsert** 策略
- 相同唯一键的记录会被**更新**而非报错
- 更新的字段：除ID和创建时间外的所有字段

### 4. 性能建议

- 单次导入建议不超过 1000 行
- 大批量数据建议分多次导入
- 导入期间避免其他操作

### 5. 错误处理

- 引用解析失败会跳过该行
- 详细错误信息会在结果中返回
- 建议先下载模板，查看示例数据

---

## 🐛 故障排查

### 问题1: "部门编码不存在"

**原因**: 填写的机构名称/编码在数据库中找不到

**解决**:
1. 检查部门管理中是否存在该部门
2. 确认部门是否设置了 `dept_code`
3. 尝试使用部门名称而非编码

### 问题2: "引用记录不存在"

**原因**: 关联的父级记录不存在

**解决**:
1. 按照导入顺序先创建父级记录
2. 检查父级记录的名称是否正确
3. 注意区分相似名称（如"一楼"和"1楼"）

### 问题3: 导入成功但数据未显示

**原因**: 缓存未清理或前端未刷新

**解决**:
1. 刷新页面
2. 清除浏览器缓存
3. 检查导入后是否执行了缓存清理

### 问题4: 部分数据导入失败

**原因**: 数据验证失败

**解决**:
1. 查看返回的错误详情
2. 检查必填字段是否为空
3. 验证枚举值是否正确（如状态、类型）

---

## 📊 架构优势总结

### 1. 配置驱动
新增模块只需在 `ExcelConfig` 中添加配置，无需修改代码

### 2. 性能优化
- 批量解析引用：N个引用只需1次查询
- 批量Upsert：减少数据库往返
- 缓存清理：按模式批量清理

### 3. 用户友好
- Excel中填写名称而非UUID
- 自动处理重复数据（更新而非报错）
- 详细的错误提示

### 4. 可维护性
- 清晰的分层架构
- 通用的引用解析器
- 统一的缓存清理机制

### 5. 可扩展性
- 易于添加新模块
- 支持多种引用类型
- 灵活的配置选项

---

## 📁 文件清单

### 新增文件
```
internal/core/db/migrations/080_add_dept_code_field.sql
internal/services/operations/reference_resolver.go
internal/services/operations/batch_upserter.go
internal/services/operations/cache_invalidator.go
scripts/tests/test_excel_import.go
docs/EXCEL_IMPORT_GUIDE.md (本文件)
```

### 修改文件
```
internal/models/dept.go
internal/services/operations/excel_config.go
internal/services/operations/excel_service.go
internal/api/v1/operations/excel_handler.go
```

---

## 🎯 下一步建议

### 短期优化
1. 添加导入进度条（大文件）
2. 支持导出带ID的Excel（用于二次编辑）
3. 添加数据预览功能

### 长期优化
1. 实现异步导入（任务队列）
2. 添加导入历史记录
3. 支持更复杂的引用关系

---

## 👥 技术支持

如有问题，请检查：
1. 应用日志（`logs/` 目录）
2. 数据库日志
3. 前端控制台错误

---

**实施完成日期**: 2025-01-27
**方案**: 方案B（优雅架构）
**状态**: ✅ 已完成
