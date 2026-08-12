# Phase 24: UUID类型一致性优化 - Pattern Map

**Mapped:** 2026-05-27
**Files analyzed:** 3
**Analogs found:** 3 / 3

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `docs/开发规范.md` | documentation | static | `docs/开发规范.md` | exact (existing file to update) |
| `internal/models/base_test.go` | test | unit-test | `internal/services/addomain/member_sync_service_test.go` | pattern-match |
| N/A (no code changes) | - | - | - | - |

## Pattern Assignments

### `docs/开发规范.md` (documentation, static)

**Analog:** Self-file (existing documentation to be updated)

**Current documentation structure** (lines 1-417):
```markdown
## 1. 概述
本文档定义了XingRan Modern项目的开发规范...

## 2. 状态值统一规范
### 2.1 通用规则
- **0 = 启用/正常/显示/有效**
- **1 = 禁用/停用/隐藏/无效**

## 3. 数据库设计规范
### 3.1 表设计规范
1. **表命名**：使用 `sys_` 前缀
2. **主键**：统一使用 UUID 类型，默认值为 `gen_random_uuid()`
```

**Pattern for adding new section:**
```markdown
## UUID 类型处理规范

### 生成策略
- 统一使用 Go 侧 BeforeCreate 钩子生成
- 禁止在迁移脚本中使用 DEFAULT gen_random_uuid()

### 字段类型
- UUID 字段使用 string 类型
- GORM 标签：`gorm:"type:uuid"`
- 可选 UUID 使用指针类型：*string

### 外部系统 UUID
- 外部系统 ID 使用 varchar(100)
- 必须添加 comment 注释说明来源
```

---

### `internal/models/base_test.go` (test, unit-test)

**Analog:** `internal/services/addomain/member_sync_service_test.go`

**Test file structure** (lines 1-33):
```go
package addomain

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupMemberSyncTestDB 创建测试数据库
func setupMemberSyncTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    require.NoError(t, err)

    // 迁移所有相关模型
    err = db.AutoMigrate(
        &models.Department{},
        &models.User{},
    )
    require.NoError(t, err)

    return db
}
```

**Test naming pattern** (lines 107-122):
```go
// TestMemberSyncService_SyncDeptMembers_ConfigValidation 测试配置验证
func TestMemberSyncService_SyncDeptMembers_ConfigValidation(t *testing.T) {
    db := setupMemberSyncTestDB(t)
    service := NewMemberSyncService(db)
    ctx := context.Background()

    deptID, _, _ := createMemberSyncTestData(t, db)

    // 测试逻辑
    result, err := service.SyncDeptMembers(ctx, deptID)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "AD配置不完整")
}
```

**UUID validation pattern** (lines 155-161 from example):
```go
// 验证 UUID 格式
_, err = uuid.Parse(model.ID)
assert.NoError(t, err, "ID should be valid UUID")
```

---

## Shared Patterns

### BaseModel BeforeCreate Hook
**Source:** `internal/models/base.go` (lines 21-27)
**Apply to:** All model understanding and documentation

```go
// BeforeCreate GORM钩子 - 创建前
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
    if b.ID == "" {
        b.ID = uuid.New().String()
    }
    return nil
}
```

### Standard UUID Field Pattern
**Source:** `internal/models/user.go` (lines 20-21)
**Apply to:** All UUID field documentation examples

```go
// UUID 字段使用 string 类型
OrgID    *string    `gorm:"type:uuid" json:"orgId,omitempty"`
ParentID *string    `gorm:"type:uuid" json:"parentId,omitempty"`
```

### External System UUID Pattern
**Source:** `internal/models/vdi.go` (lines 15-16)
**Apply to:** Documentation examples for external system IDs

```go
// 外部系统ID字段
VMID        string `gorm:"type:varchar(100);uniqueIndex;not null" json:"vm_id"`
VdiServerID string `gorm:"type:varchar(100);index;not null" json:"vdi_server_id"`
```

### Database Default UUID Pattern (Legacy - NOT for new tables)
**Source:** `internal/models/dept_group_mapping.go` (line 23)
**Apply to:** Documentation as anti-pattern

```go
// ❌ 反模式：已有表使用 DB 默认值（不推荐用于新表）
ID string `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
```

### Migration Script Pattern (Current Practice)
**Source:** `internal/core/db/migrations/016_create_ad_domain_tables.sql` (lines 10-32)
**Apply to:** Documentation as reference (but NOT recommended for new tables)

```sql
-- 当前模式：使用 DEFAULT gen_random_uuid()（已有表保持现状）
CREATE TABLE IF NOT EXISTS sys_ad_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(100) NOT NULL,
    -- ...
);
```

### New Migration Pattern (Recommended)
**Source:** Context decision D-04
**Apply to:** Documentation as recommended pattern

```sql
-- ✅ 推荐模式：新表不使用 DEFAULT，由 Go 生成
CREATE TABLE IF NOT EXISTS sys_new_table (
    id UUID PRIMARY KEY,  -- 无 DEFAULT，由 Go BeforeCreate 生成
    name VARCHAR(100) NOT NULL
);
```

---

## No Analog Found

None. All identified files have appropriate analogs in the codebase.

---

## Metadata

**Analog search scope:**
- `internal/models/` - Model definitions and patterns
- `internal/services/addomain/*_test.go` - Test file structure patterns
- `docs/开发规范.md` - Documentation structure
- `internal/core/db/migrations/*.sql` - Migration script patterns

**Files scanned:** 10+
**Pattern extraction date:** 2026-05-27

---

## Implementation Notes

### Test File Creation
The new `internal/models/base_test.go` should follow this structure:

1. **Package declaration:** `package models_test` or `package models` (depending on convention)
2. **Imports:** 
   - `testing` for test framework
   - `github.com/stretchr/testify/assert` and `require` for assertions
   - `github.com/google/uuid` for UUID validation
   - `gorm.io/gorm` and `gorm.io/driver/sqlite` for test database
3. **Test functions:**
   - `TestBaseModel_BeforeCreate_GeneratesUUID` - Test auto-generation
   - `TestBaseModel_BeforeCreate_PreservesExistingUUID` - Test ID preservation
4. **Setup pattern:** Use in-memory SQLite for fast, isolated tests

### Documentation Update Pattern
The `docs/开发规范.md` update should:

1. **Add new section:** "UUID 类型处理规范" after "状态值统一规范"
2. **Provide examples:** Both correct and incorrect patterns
3. **Include migration guidance:** Clear SQL examples
4. **Reference BaseModel:** Explain BeforeCreate hook behavior
5. **Warn about anti-patterns:** Explicitly mark discouraged practices

### Key Pattern Insights

1. **All existing models inherit UUID generation** from BaseModel.BeforeCreate
2. **Test framework uses testify** for assertions and require for setup
3. **In-memory SQLite** is the standard for unit tests
4. **External system IDs** consistently use `varchar(100)` with descriptive comments
5. **Optional UUID fields** use pointer type `*string` with `omitempty` JSON tag
6. **Migration scripts vary** - some use DB defaults, some don't (historical inconsistency)
