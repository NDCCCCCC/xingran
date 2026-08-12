# Phase 24: UUID类型一致性优化 - Context

**Gathered:** 2026-05-27
**Status:** Ready for planning

## Phase Boundary

统一代码库中的 UUID 类型处理模式，确保新代码遵循一致的规范。本阶段不涉及大规模重构已有表，而是建立清晰的规范指导未来开发，并通过文档和测试确保规范得到遵守。

**核心目标：**
1. 统一 UUID 生成策略为 Go 侧生成
2. 保持 Go 层使用 string 类型（向后兼容）
3. 明确外部系统 UUID 字段处理方式
4. 建立迁移脚本的编写规范
5. 补充单元测试确保规范执行
6. 更新开发规范文档

**不包含：**
- 不修改现有表的 UUID 列定义
- 不进行数据迁移
- 不改变 API 接口（保持 string 类型）

## Implementation Decisions

### D-01: UUID 生成策略
**决策：统一使用 Go 侧生成**

**实施方式：**
- 所有模型使用 `BeforeCreate` 钩子生成 UUID
- UUID 生成使用 `uuid.New().String()`
- 新建表的迁移脚本不使用 `DEFAULT gen_random_uuid()`
- BaseModel 已有 `BeforeCreate` 钩子，所有继承模型自动支持

**理由：**
- 跨数据库兼容（不依赖 PostgreSQL 特定功能）
- 代码控制 UUID 生成逻辑，便于测试和调试
- 与现有大部分模型模式一致

**代码示例：**
```go
// BaseModel 已实现 BeforeCreate
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
    if b.ID == "" {
        b.ID = uuid.New().String()
    }
    return nil
}

// 新模型只需继承 BaseModel
type NewModel struct {
    BaseModel
    Name string `gorm:"size:100"`
}
```

### D-02: Go 层类型定义
**决策：保持使用 string 类型**

**实施方式：**
- UUID 字段类型继续使用 `string`
- GORM 标签使用 `gorm:"type:uuid"`
- JSON 标签使用 `json:"id"` 或 `json:"userId"` 等

**理由：**
- 向后兼容，无需修改 API 和前端代码
- JSON 序列化/反序列化简单
- 避免引入自定义 UUID 类型的复杂性

**代码示例：**
```go
type UserModel struct {
    BaseModel
    // UUID 字段使用 string 类型
    OrgID    string  `gorm:"type:uuid" json:"orgId"`
    ParentID *string `gorm:"type:uuid" json:"parentId,omitempty"`
}
```

### D-03: 外部系统 UUID 字段处理
**决策：外部 UUID 使用 varchar + 注释说明**

**实施方式：**
- 外部系统（如 VDI）返回的 ID 使用 `varchar(100)`
- 添加 GORM comment 注释说明来源
- 添加代码注释说明格式

**理由：**
- 灵活处理各种 ID 格式（UUID、字符串、数字等）
- 避免因格式验证导致的数据同步失败
- 与内部 UUID 字段明确区分

**代码示例：**
```go
// VDIModel VDI 相关表模型
// VMID: 深信服VDI返回的虚拟机ID，格式示例：vm-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
type VDIVM struct {
    BaseModel
    // 外部系统ID字段
    VMID        string `gorm:"type:varchar(100);uniqueIndex;not null;comment:VDI虚拟机ID" json:"vmId"`
    VdiServerID string `gorm:"type:varchar(100);index;not null;comment:VDI服务器ID" json:"vdiServerId"`
    ResourceID  string `gorm:"type:varchar(100);index;comment:VDI资源组ID" json:"resourceId"`

    // 内部关联使用 UUID 类型
    BoundUserID *string `gorm:"type:uuid" json:"boundUserId,omitempty"`
}
```

### D-04: 迁移脚本编写规范
**决策：新表不使用 DB 默认值，统一 Go 生成**

**规范要求：**
- 新建表的主键列不使用 `DEFAULT gen_random_uuid()`
- 所有 UUID 列由 Go 层 BeforeCreate 钩子生成
- 可选外键字段可考虑使用 DB 默认值（特殊场景）

**迁移脚本模板：**
```sql
-- ✅ 正确：新表创建时不使用默认值
CREATE TABLE sys_new_table (
    id UUID PRIMARY KEY,  -- 无 DEFAULT，由 Go 生成
    name VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,  -- 外键也无 DEFAULT
    assignee_id UUID,  -- 可选外键也无 DEFAULT
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ❌ 错误：新表不要使用 DEFAULT gen_random_uuid()
CREATE TABLE sys_new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()  -- 不要这样做
);
```

**已有表保持现状：**
- 不修改现有表的 UUID 列定义
- 不移除现有的 `DEFAULT gen_random_uuid()`
- 不添加数据迁移脚本

### D-05: 单元测试要求
**决策：添加 UUID 生成和验证的单元测试**

**测试覆盖：**
1. BaseModel.BeforeCreate 钩子测试
2. UUID 格式验证测试
3. 可选 UUID 字段（*string）的测试

**测试示例：**
```go
// internal/models/base_test.go
package models_test

func TestBaseModel_BeforeCreate_GeneratesUUID(t *testing.T) {
    model := &BaseModel{}
    err := model.BeforeCreate(nil)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, model.ID)
    
    // 验证 UUID 格式
    _, err = uuid.Parse(model.ID)
    assert.NoError(t, err, "ID should be valid UUID")
}

func TestBaseModel_BeforeCreate_PreservesExistingUUID(t *testing.T) {
    existingID := uuid.New().String()
    model := &BaseModel{ID: existingID}
    
    err := model.BeforeCreate(nil)
    
    assert.NoError(t, err)
    assert.Equal(t, existingID, model.ID, "Should not overwrite existing ID")
}
```

### D-06: 开发规范文档更新
**决策：更新开发规范文档，明确 UUID 处理规范**

**更新内容：**
1. 在 `docs/开发规范.md` 添加 UUID 处理章节
2. 说明 Go 侧生成策略
3. 提供外部系统 UUID 字段处理指南
4. 包含迁移脚本编写规范
5. 添加代码示例和反模式警告

**文档结构：**
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
- 示例：`gorm:"type:varchar(100);comment:VDI虚拟机ID"`

### 迁移脚本规范
[具体规范]

### 反模式警告
- ❌ 不要在 GORM 标签中使用 default:gen_random_uuid()
- ❌ 不要混用多种 UUID 生成方式
```

### Claude's Discretion

以下方面可以由实现者决定：

1. **测试文件组织**：测试放在 `internal/models/` 还是 `internal/models/*/testdata/`
2. **文档格式**：使用 Markdown 还是其他格式
3. **代码注释详细程度**：只要说明外部 UUID 来源即可，详细程度自定
4. **可选外键的默认值处理**：根据具体场景决定是否需要 DB 默认值

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目架构文档
- `docs/项目概述和架构设计.md` — 整体架构
- `docs/开发规范.md` — 需要更新的开发规范
- `docs/API响应规范.md` — API 响应格式

### 核心代码文件
- `internal/models/base.go` — BaseModel 和 BeforeCreate 钩子
- `internal/core/db/migrations/` — 迁移脚本目录
- `github.com/google/uuid` — UUID 生成库文档

### 参考模型
- `internal/models/vdi.go` — 外部系统 UUID 字段示例
- `internal/models/dept_group_mapping.go` — 使用 DB 默认值的示例（不推荐用于新表）
- `internal/models/user.go` — 标准 UUID 字段示例

## Existing Code Insights

### Reusable Assets
- **BaseModel.BeforeCreate** — 已实现 UUID 生成，所有模型自动继承
- **uuid.New().String()** — 项目统一使用的 UUID 生成方法
- **GORM type:uuid** — 统一的数据库列类型

### Established Patterns
- **继承 BaseModel** — 所有模型继承 BaseModel 获得 UUID 生成能力
- **BeforeCreate 钩子** — 在创建前自动生成 UUID
- **可选 UUID 字段** — 使用 `*string` 表示可空的 UUID 字段

### Integration Points
- **迁移脚本** — 新迁移脚本需遵循无默认值规范
- **单元测试** — 添加到 `internal/models/` 测试文件
- **开发文档** — 更新 `docs/开发规范.md`

### Code Statistics
- 总计约 116 个 UUID 字段使用 `gorm:"type:uuid"`
- 约 50 个模型使用 BeforeCreate 生成（目标状态）
- 约 20 个表有 DB 默认值（保持现状）
- 约 10 个表仅依赖 DB 生成（保持现状）

## Specific Ideas

### 迁移脚本检查清单
创建新表时使用以下检查：
```bash
# 1. 检查是否有 DEFAULT gen_random_uuid()
grep "DEFAULT gen_random_uuid()" migrations/xxx_create_table.sql

# 2. 确认没有 DB 默认值
# ✅ 正确: id UUID PRIMARY KEY
# ❌ 错误: id UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

### 代码审查要点
- 新建表的主列没有 `DEFAULT gen_random_uuid()`
- 外部系统 ID 使用 `varchar(100)` + comment
- 新模型继承 BaseModel 而非手动实现 BeforeCreate

### 单元测试命名
- `TestBaseModel_BeforeCreate_GeneratesUUID`
- `TestBaseModel_BeforeCreate_PreservesExistingUUID`
- `Test[ModelName]_UUIDFields`

## Deferred Ideas

以下想法不在本期范围：

- **大规模重构现有表** — 工作量大，收益不明显，现有数据可正常运行
- **引入强类型 UUID** — 需要修改 API 和前端，向后兼容性问题
- **外部 UUID 类型统一** — 依赖外部系统格式，风险高

## Noted for Later

如果未来需要进一步优化，可考虑：
- 自动化检查脚本（CI 中检测不符合规范的迁移脚本）
- golangci-lint 规则（检测 DB 默认值使用）
- 性能对比测试（Go 生成 vs DB 生成）

---

*Phase: 24-uuid*
*Context gathered: 2026-05-27*
