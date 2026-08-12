# 工位状态字段深度代码分析报告

> 调研编号：260626-n71
> 调研时间：2026-06-26
> 需求：工位选择所属人员后自动改为占用状态（status=1）

---
## 1. 核心结论

**已有枚举值，无需新增。** WorkstationStatusOccupied = 1（占用）已存在于代码中，前端下拉选项/后端枚举/Excel 配置均已就绪。

**待实现联动逻辑：** 当前所有写入路径（Handler -> Service -> DB）均**未实现** user_id 与 status 的联动自动设置。

---

## 2. 数据模型与字段定义

### 2.1 后端模型

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\internal\models\workstation.go

```go
// 工位状态枚举
type WorkstationStatus int

const (
    WorkstationStatusAvailable WorkstationStatus = 0 // 空闲 - 可分配
    WorkstationStatusOccupied  WorkstationStatus = 1 // 占用 - 已分配给用户  <- 已有！
    WorkstationStatusMaintain  WorkstationStatus = 2 // 维护 - 维修中不可用
)

// 工位模型
type Workstation struct {
    BaseModel
    WorkstationName string            gorm:"size:100;not null" json:"name"
    WorkstationType WorkstationType   gorm:"default:0" json:"type"
    Status          WorkstationStatus gorm:"default:0" json:"status"  // 整型，默认 0
    // ...
    UserID   *string gorm:"size:64" json:"userId,omitempty"    // UUID，指针（可 NULL）
    UserName *string gorm:"size:100" json:"userName,omitempty"
    // ...
}
```

- **Status 类型**：WorkstationStatus（int），默认值 0（空闲）
- **UserID 类型**：*string（指针），可 NULL，代表未分配

### 2.2 前端类型

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\types\operations.ts

```typescript
export type WorkstationOpsStatus = 0 | 1 | 2;
```

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\workstations\constants.tsx

```typescript
export const STATUS_OPTIONS = [
    { label: "空闲", value: 0 },
    { label: "占用", value: 1 },  // <- 前端已有
    { label: "维护", value: 2 },
];
```

---

## 3. 状态字段业务值与字典

### 3.1 三值枚举定义

| 值 | 常量名 | 语义 | 场景 |
|----|--------|------|------|
| 0 | WorkstationStatusAvailable | 空闲 | 可分配，无用户 |
| 1 | WorkstationStatusOccupied | 占用 | 已分配给用户 <- **目标状态** |
| 2 | WorkstationStatusMaintain | 维护 | 维修中，不可分配 |

### 3.2 字典配置

**无 sys_dict 记录。** 工位状态是**代码级枚举**（非数据字典），未在 sys_dict / sys_dict_item 表中注册。

---

## 4. 状态字段修改的所有触发点

### 4.1 Handler 层（入口）

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\internal\api\v1\operations\workstation_handler.go

| 方法 | 行号 | 状态修改逻辑 | 备注 |
|------|------|-------------|------|
| Create | 80-92 | 无，直接透传 | 新建默认 status=0 |
| Update | 155-169 | 无，直接透传 | Save() 全量覆盖 |
| Delete | 171-179 | 无 | 软删除 |
| BatchOperation | 181-233 | 无 | 批量操作 |
| BatchUpdatePositions | 235-272 | 无 | 批量位置/尺寸更新 |
| Statistics | 274-283 | 无，仅查询 | - |

**结论：Handler 层无任何自动逻辑。**

### 4.2 Service 层（业务层）

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\internal\services\operations\workstation_service.go

| 方法 | 行号 | 状态修改逻辑 | 备注 |
|------|------|-------------|------|
| Create | 156-159 | 无，直接 Create | 默认 status=0 |
| Update | 161-178 | 无，Save() 全量覆盖 | 见下方说明 |
| Delete | 180-182 | 无 | - |
| GetByID | 184-196 | 无，仅查询 | - |
| List | 198-268 | 无，仅查询 | - |
| BatchDelete | 270-275 | 无 | - |
| BatchUpdatePositions | 277-375 | 无 | 批量位置更新 |

**Update 方法实现（161-178 行）：**

```go
func (s *workstationService) Update(ctx context.Context, workstation *models.Workstation) error {
    var existing models.Workstation
    if err := s.db.WithContext(ctx).Where("id = ?", workstation.ID).First(&existing).Error; err != nil {
        return err
    }
    // 保留创建时间和创建人
    workstation.CreatedAt = existing.CreatedAt
    workstation.CreatedBy = existing.CreatedBy
    return s.db.WithContext(ctx).Save(workstation).Error
}
```

**关键：Save() 是 GORM 全量覆盖操作。** 入参中 status 字段值直接覆盖数据库，无论是否传值。

**无 AssignUser() / ReleaseUser() 等独立方法。**

### 4.3 Excel 导入

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\internal\services\operations\excel_service.go

关键函数：prepareRecordsForUpsert（796-857 行）

```go
// user_id 直接透传，不自动设 status=1
preparedRecord["user_id"] = record["user_id"]
// ... 其他字段直接透传
// 最后调用 standardUpsert (PartialUpdate: false)
```

- PartialUpdate: false（excel_config.go:137 未配置），使用 standardUpsert
- standardUpsert 中 columns = append(columns, "status") 直接写入 Excel 中的 status 值
- **若 Excel 中 user_id 有值但 status 漏填或填 0，数据不一致**

### 4.4 数据库层

无触发器、无约束、无 DEFAULT 表达式基于 user_id 的自动推导。

---

## 5. UserID 变更时的当前行为

### 5.1 修改场景

| 场景 | 操作 | 当前行为 | 问题 |
|------|------|----------|------|
| 分配用户 | 编辑工位，选择用户 | user_id 更新，status 不变（保持原值） | **不一致** |
| 取消用户 | 编辑工位，清空用户 | user_id 置 NULL，status 不变 | **不一致** |
| 手动改状态 | 编辑工位，改状态 | status 独立修改 | 允许手动干预 |
| Excel 导入 | 填 userName | user_id 解析填充，status 不变 | **不一致** |

### 5.2 前端 EditModal 现状

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\workstations\modals\EditModal.tsx

- userId 选择器（206-214 行）：独立 Select，无 onChange 联动 status
- status 选择器（241-249 行）：独立 Select，无 onChange 联动 userId
- **两者完全独立，无联动逻辑**

---

## 6. 前端关联逻辑

### 6.1 编辑模态框

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\workstations\modals\EditModal.tsx

- userId 与 status 为两个独立 Select 组件
- 无 onChange 联动
- 组织/部门级联变化时清空 userId（154-161 行），但不清空 status

### 6.2 新建默认值

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\workstations\index.tsx

```typescript
// 462 行：新建工位默认
{ status: 0, type: 0 }
```

### 6.3 状态标签渲染

**文件：** D:\CODE\ClaudeCode\xingran-go-backend\xingran-react-frontend\src\pages\operations\workstations\constants.tsx

```typescript
export const renderWorkstationStatusTag = (status: number) => {
    // 0 -> 绿色"空闲"
    // 1 -> 蓝色"占用"
    // 2 -> 橙色"维护"
};
```

---

## 7. 相关测试

### 7.1 后端测试

**文件搜索：** **/workstation*_test.go

未找到 workstation 专用测试文件。

### 7.2 前端测试

jest / RTL 测试：未覆盖 workstation 状态联动逻辑

### 7.3 集成测试

未覆盖 user_id -> status 联动场景

---

## 8. 近期相关任务历史

### 8.1 已完成相关任务

| 任务 | 内容 | 路径 |
|------|------|------|
| 工位设备关联 | ops_workstation_device 关联表 + 前端展开行 | ops_workstation_device |
| 主设备序列号 | 工位列表加 primary_device_serial 子查询列 | 进行中 |

### 8.2 系统级约束

- **CLAUDE.md 状态规范**：仅覆盖 2 值枚举（normal/disabled）；工位 3 值枚举**超出**该规范
- **无缓存**：sys_workstation 无 Redis 缓存，无缓存失效逻辑
- **Operlog 约定**：OperTypeUpdate 用于 status 变更（通过 Update 接口）

---

## 9. 实现方案建议

### 方案 A（推荐）：Service 层统一拦截

**修改文件数：2**

#### 改动 1：workstation_service.go 的 Update 方法

在 Save() 前，根据 user_id 是否变化自动设置 status：

```go
// 自动联动：user_id 有值 -> status=1（占用）；user_id 为空 -> status=0（空闲）
if workstation.UserID != nil && *workstation.UserID != "" {
    workstation.Status = models.WorkstationStatusOccupied  // 1
} else {
    workstation.Status = models.WorkstationStatusAvailable  // 0
}
```

#### 改动 2：excel_service.go 的 prepareRecordsForUpsert 或 BatchUpsert

在 standardUpsert 写入 DB 前，根据 user_id 是否存在自动设置 status=1。

#### 改动 3：数据迁移（历史数据修复）

```sql
-- 修复历史数据：user_id 有值但 status!=1 的工位
UPDATE sys_workstation
SET status = 1
WHERE user_id IS NOT NULL AND user_id != '' AND status != 1;
```

### 方案 B：Handler 层 + Excel 层双端拦截

在 Handler 的 Update 和 Excel 的 BatchUpsert 分别做联动判断。
**缺点**：分散在多处，容易遗漏。

### 方案 C：数据库触发器

在 DB 层用 CREATE TRIGGER 自动同步。
**缺点**：违反项目架构（业务逻辑不上 DB），且调试困难。

---

## 10. 实现约束总结

| 约束项 | 状态 | 说明 |
|--------|------|------|
| 枚举值 WorkstationStatusOccupied=1 | YES 已有 | 无需新增 |
| 前端 status 下拉选项 | YES 已有 | 无需修改 |
| 后端 Service Update 联动 | TODO 待实现 | 需在 Save() 前拦截 |
| Excel 导入联动 | TODO 待实现 | 需在 upsert 前拦截 |
| 历史数据迁移 | TODO 待实施 | 需 SQL 脚本 |
| sys_dict 字典注册 | 无需 | 代码级枚举 |
| Redis 缓存失效 | 无需 | 无缓存 |
| 新 OperType | 无需 | OperTypeUpdate 已覆盖 |
| 前端 userId onChange | TODO 待实现 | 可选：前端同步联动 |

---

## 11. 关键代码路径汇总

| 层级 | 文件 | 行号 | 关键操作 |
|------|------|------|----------|
| 模型 | internal/models/workstation.go | 4-10 | 枚举定义 |
| Handler | internal/api/v1/operations/workstation_handler.go | 155-169 | Update 入口 |
| Service | internal/services/operations/workstation_service.go | 161-178 | Update 实现（需修改） |
| Excel | internal/services/operations/excel_service.go | 796-857 | prepareRecordsForUpsert（需修改） |
| Excel 配置 | internal/services/operations/excel_config.go | 134-152 | workstation 配置 |
| 前端类型 | xingran-react-frontend/src/types/operations.ts | 52 | WorkstationOpsStatus |
| 前端常量 | xingran-react-frontend/src/pages/operations/workstations/constants.tsx | - | STATUS_OPTIONS |
| 前端 EditModal | xingran-react-frontend/src/pages/operations/workstations/modals/EditModal.tsx | 206-249 | userId/status Select |

---

## 12. 下一步行动

1. **确认方案**：推荐方案 A（Service 层拦截）
2. **实施改动**：修改 workstation_service.go 的 Update 方法
3. **Excel 改动**：修改 excel_service.go 的 upsert 逻辑
4. **历史数据**：执行数据迁移 SQL
5. **测试验证**：补充 workstation 状态联动测试用例

---

*报告生成时间：2026-06-26*
*调研人：Claude Code*
