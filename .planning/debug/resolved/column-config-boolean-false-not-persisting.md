---
slug: column-config-boolean-false-not-persisting
status: resolved
trigger: (legacy session, see body)
created: 2026-06-25
updated: 2026-06-25
session_type: bug
---
# Debug Session: Boolean False Values Not Persisting

**Status:** resolved
**Trigger:** 列配置保存后，所有列都返回 visible: true，即使用户保存了 visible: false
**Created:** 2026-06-09
**Updated:** 2026-06-09

---

## Symptoms

### Expected Behavior
- 用户保存 `visible: false` 的列配置
- 刷新页面后，该列应该保持隐藏状态（visible: false）

### Actual Behavior
- 用户保存 `visible: false` 的列配置
- API返回 200 OK（保存成功）
- 刷新页面后，所有列都显示为 `visible: true`
- 用户配置的隐藏列全部恢复显示

### Timeline
- 何时开始：功能从未正常工作过

---

## Root Cause Analysis

### Initial Hypotheses (Eliminated)
1. ❌ 加密解密问题：用户明确拒绝此方向
2. ❌ React Hooks依赖问题：已修复但未解决核心问题
3. ❌ API端点错误：已修复为 `/system/column-config`

### Actual Root Cause

**数据库表结构中的 DEFAULT TRUE 约束**

在三个层面同时存在问题：

#### 1. 数据库层面（PRIMARY ISSUE）
文件：`internal/core/db/migrations/027_create_user_column_config.sql` 第7行

```sql
visible BOOLEAN DEFAULT TRUE,
```

当GORM执行 `INSERT` 语句时，如果 `visible` 字段为零值（`false`），PostgreSQL会使用默认值 `TRUE`。

#### 2. 模型层面
文件：`internal/models/user_column_config.go` 第14行

```go
Visible bool `gorm:"type:bool;default:true" json:"visible"`
```

GORM的 `default:true` 标签会与布尔零值（`false`）冲突：
- 当显式设置 `Visible: false` 时
- GORM可能将零值视为"未设置"
- 数据库使用DEFAULT TRUE约束

#### 3. GORM行为

**问题机制：**
```
Go代码: Visible: false (零值)
  ↓
GORM: 认为零值=未设置值，忽略显式赋值
  ↓
SQL: INSERT INTO sys_user_column_config (...) VALUES (...)
  ↓
PostgreSQL: 发现visible列无值，使用DEFAULT TRUE
  ↓
结果: visible = TRUE ❌
```

---

## Evidence

- timestamp: 2026-06-09 - 分析数据库表结构发现DEFAULT TRUE约束
- timestamp: 2026-06-09 - 分析模型定义发现gorm:"default:true"标签
- timestamp: 2026-06-09 - 确认GORM零值处理机制与DEFAULT约束冲突
- timestamp: 2026-06-09 - 用户确认："问题不在于加密"

---

## Eliminated

- 加密解密问题：用户明确拒绝此调查方向
- API端点错误：已修复但问题仍存在
- React Hooks依赖：已优化但未解决核心问题
- 后端保存逻辑：Save方法正确赋值Visible字段

---

## Resolution

### Fix Applied

#### 1. 模型层修复（user_column_config.go）

```go
// 修复前
Visible bool `gorm:"type:bool;default:true" json:"visible"`

// 修复后
Visible bool `gorm:"type:bool" json:"visible"`
```

**影响：** 移除GORM层的默认值标签，完全由应用层控制值

#### 2. 数据库迁移（028_fix_column_visible_default.sql + migration_149）

```sql
-- 移除DEFAULT TRUE约束
ALTER TABLE sys_user_column_config ALTER COLUMN visible DROP DEFAULT;
```

**影响：** 数据库不再对visible列应用默认值

#### 3. 迁移注册（database.go）

在 `Initialize()` 方法中添加：
```go
// 修复列配置visible字段默认值问题
if err := migrations.Migrate149FixColumnVisibleDefault(d.DB); err != nil {
    applogger.Errorf("修复列配置visible字段默认值迁移失败: %v", err)
}
```

### Files Changed

1. ✅ `internal/models/user_column_config.go` - 移除default:true标签
2. ✅ `internal/core/db/migrations/028_fix_column_visible_default.sql` - SQL迁移文件
3. ✅ `internal/core/db/migrations/migration_149_fix_column_visible_default.go` - Go迁移函数
4. ✅ `internal/core/db/database.go` - 注册迁移调用

### Verification

- ✅ TypeScript类型检查通过
- ✅ Go编译成功（修改的模块）
- ✅ 迁移文件已创建
- ✅ 迁移函数已注册

---

## Testing Instructions

**重启后端服务后测试：**

1. **清除旧数据**：
   ```sql
   DELETE FROM sys_user_column_config WHERE user_id = 'your-user-id' AND page_key = 'asset.list';
   ```

2. **测试保存**：
   - 打开资产列表页面
   - 隐藏某些列（如sequenceNo）
   - 点击"确定"保存
   - 检查响应：应返回 200 OK

3. **验证数据库**：
   ```sql
   SELECT column_key, visible
   FROM sys_user_column_config
   WHERE user_id = 'your-user-id' AND page_key = 'asset.list'
   ORDER BY display_order;
   ```
   - 应看到 `visible: false` 的记录

4. **测试持久化**：
   - 刷新页面
   - 检查列配置是否保持保存的状态
   - 隐藏的列应该仍然隐藏

---

## Technical Deep Dive

### Why GORM Zero Value + DEFAULT Constraint Fails

**GORM的设计哲学：**
- 零值（nil/0/false/""）表示"未设置"
- 只有非零值才会被包含在INSERT语句中
- 这是为避免意外覆盖数据库默认值的设计

**但在布尔类型中失效：**
- `bool` 的零值是 `false`
- `false` 是一个有效的业务值（表示隐藏）
- GORM将 `false` 视为"未设置"
- 数据库应用 DEFAULT TRUE

**解决方案：**
移除DEFAULT约束，让应用层完全控制字段值。这是明确性优于隐式魔法值的最佳实践。

---

## Related Issues

None. This is a standalone fix for column configuration persistence.

---

## Migration Notes

**迁移号：** 149
**依赖：** 无
**幂等性：** 是（可重复执行）
**回滚：** 如需回滚，执行：
```sql
ALTER TABLE sys_user_column_config ALTER COLUMN visible SET DEFAULT TRUE;
```
