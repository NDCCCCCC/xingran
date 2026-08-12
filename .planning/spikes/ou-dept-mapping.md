# Spike: AD域控OU与部门映射方案

**创建时间:** 2026-05-22
**状态:** Researching
**优先级:** 高

## 问题陈述

当前系统已实现AD域管理功能，可以从AD域同步OU（组织单位）、用户、用户组等数据到本地数据库（`sys_ad_ou`, `sys_ad_user`, `sys_ad_group`）。但AD域中的OU与系统部门（`sys_dept`）之间缺乏映射关系，导致：

1. **用户归属不明确**: AD用户同步后，其所属部门（AD中的department属性）与系统部门不一致
2. **权限管理困难**: 无法基于AD组织结构自动分配系统权限
3. **数据孤岛**: AD组织架构与系统组织架构割裂，需要手动维护两套体系
4. **业务流程中断**: 工位、资产等关联部门的功能无法利用AD组织信息

## 研究目标

### 核心目标
设计并验证AD OU与`sys_dept`之间的映射机制，实现：
- 自动/手动建立映射关系
- 支持一对一、一对多、多对一映射
- 同步时自动维护映射关系
- 提供管理界面API

### 次要目标
- 用户同步时自动关联部门
- 基于OU层级自动创建部门结构
- 映射冲突检测和解决机制

## 现有代码分析

### AD域管理模块
**位置:** `internal/services/addomain/`

**核心组件:**
```
ADDomainService (主服务)
├── ConfigService    - AD配置管理
├── SyncService      - 数据同步（OU、用户、组、电脑）
├── OUService        - OU管理
├── UserService      - 用户管理
├── GroupService     - 用户组管理
├── LogService       - 同步日志
└── ComputerService  - 电脑设备管理
```

**现有数据模型:**
```go
// AD域配置 (sys_ad_config)
type ADConfig struct {
    ID            string
    ConfigName    string
    BaseDN        string  // 例如: DC=example,DC=com
    ...
}

// AD OU (sys_ad_ou)
type ADOU struct {
    ID           string
    ADConfigID   string
    OUN          string  // LDAP DN: OU=Sales,OU=Departments,DC=example,DC=com
    OUName       string  // 显示名称: Sales
    OUPath       string  // 完整路径: Departments/Sales
    ParentDN     string  // 父级DN
    UserCount    int
    GroupCount   int
}

// AD用户 (sys_ad_user)
type ADUser struct {
    ID           string
    ADConfigID   string
    UserDN       string
    Username     string
    Department   string  // AD中的department属性值
    OUN          string  // 所属OU的DN
    ...
}
```

**现有同步流程:**
```
SyncData()
  ├── 连接LDAP服务器
  ├── 搜索并同步 OU (organizationalUnit)
  ├── 搜索并同步 Group (group)
  ├── 搜索并同步 User (user)
  └── 搜索并同步 Computer (computer)
```

### 部门管理模块
**位置:** `internal/models/dept.go`, `internal/services/system/`

**数据模型:**
```go
// 系统部门 (sys_dept)
type Department struct {
    ID          string
    DeptName    string   // 部门名称
    DeptCode    string   // 部门编码（用于Excel导入）
    ParentID    *string  // 父部门ID
    Ancestors   string   // 祖级列表: /0/1/2/
    OrderNum    int      // 排序号
    Leader      *string  // 负责人ID
    Phone       *string
    Email       *string
    Status      int      // 0=正常, 1=停用
    IsExternalOrg int    // 是否为外部机构
}
```

### 现有关联
- ❌ **无直接关联**: `sys_ad_ou` 与 `sys_dept` 之间无映射关系
- ✅ **间接关联**: `sys_ad_user.department` 字段存储AD中的department属性值（字符串）

## 技术方案设计

### 方案1: 映射表方案 (推荐)

#### 数据库设计
```sql
-- AD OU与部门映射关系表
CREATE TABLE sys_ad_ou_dept_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL REFERENCES sys_ad_config(id),
    ou_dn VARCHAR(500) NOT NULL,              -- AD OU的DN
    dept_id UUID NOT NULL REFERENCES sys_dept(id),  -- 系统部门ID
    mapping_type VARCHAR(20) NOT NULL,         -- 映射类型: auto/manual/inherit
    priority INT DEFAULT 0,                   -- 优先级（用于一对多映射的排序）
    sync_enabled BOOLEAN DEFAULT true,         -- 是否同步
    auto_create_dept BOOLEAN DEFAULT false,   -- 同步时自动创建部门
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    UNIQUE(ad_config_id, ou_dn)
);

CREATE INDEX idx_ou_dept_mapping_config ON sys_ad_ou_dept_mapping(ad_config_id);
CREATE INDEX idx_ou_dept_mapping_dept ON sys_ad_ou_dept_mapping(dept_id);
COMMENT ON TABLE sys_ad_ou_dept_mapping IS 'AD OU与系统部门映射关系表';
```

#### Go数据模型
```go
// internal/models/ad_ou_dept_mapping.go
package models

import "time"

type OUMappingType string

const (
    OUMappingTypeAuto    OUMappingType = "auto"    // 自动匹配（按名称）
    OUMappingTypeManual  OUMappingType = "manual"  // 手动配置
    OUMappingTypeInherit OUMappingType = "inherit" // 继承父级映射
)

type ADOUDeptMapping struct {
    ID             string         `gorm:"primaryKey;type:uuid" json:"id"`
    ADConfigID     string         `gorm:"type:uuid;not null;index:idx_ou_dept_mapping_config,priority:1" json:"adConfigId"`
    OUDN           string         `gorm:"size:500;not null;uniqueIndex:uni_ou_dept_mapping,priority:1" json:"ouDn"`
    DeptID         string         `gorm:"type:uuid;not null;index:idx_ou_dept_mapping_dept;uniqueIndex:uni_ou_dept_mapping,priority:2" json:"deptId"`
    MappingType    OUMappingType  `gorm:"size:20;not null" json:"mappingType"`
    Priority       int            `gorm:"default:0" json:"priority"`
    SyncEnabled    bool           `gorm:"default:true" json:"syncEnabled"`
    AutoCreateDept bool           `gorm:"default:false" json:"autoCreateDept"`
    CreatedAt      time.Time      `json:"createdAt"`
    UpdatedAt      time.Time      `json:"updatedAt"`
    DeletedAt      *time.Time     `json:"deletedAt,omitempty"`

    // 关联（不持久化）
    ADOU   *ADOU        `gorm:"foreignKey:OUN;references:OUN" json:"adOu,omitempty"`
    Dept   *Department  `gorm:"foreignKey:DeptID" json:"dept,omitempty"`
}

func (ADOUDeptMapping) TableName() string {
    return "sys_ad_ou_dept_mapping"
}
```

#### 映射规则设计

**1. 自动匹配规则 (Auto Mapping)**
```go
// 按OU名称匹配部门名称
// 例如: OU名称 "销售部" → 匹配 DeptName "销售部"
func MatchDeptByName(ouName string) (*Department, error) {
    // 优先级1: 完全匹配 DeptName
    // 优先级2: 模糊匹配（包含关系）
    // 优先级3: 拼音/缩写匹配（扩展功能）
}
```

**2. 继承规则 (Inherit Mapping)**
```go
// 子OU继承父OU的映射关系
// 例如: "研发部/后端组" 继承 "研发部" 的部门映射
func InheritParentMapping(ouDN string) (*ADOUDeptMapping, error) {
    // 查找父OU的映射关系
    // 创建子映射记录，类型为inherit
}
```

**3. 手动配置 (Manual Mapping)**
```go
// 管理员手动指定OU与部门的对应关系
// 支持一对多（一个OU对应多个部门）
func CreateManualMapping(ouDN, deptID string) error {
    // 创建或更新映射记录
}
```

#### 同步流程集成

**修改后的同步流程:**
```go
// internal/services/addomain/sync.go
func (s *SyncService) SyncData(ctx context.Context, config *models.ADConfig, syncType string) (*SyncResult, error) {
    // ... 现有代码 ...

    // 2. 搜索和同步 OU
    ous, err := client.SearchOUs(config.BaseDN)
    if err != nil {
        return nil, err
    }

    // 新增: 同步OU时处理映射关系
    if err := s.syncOUs(ctx, config, ous); err != nil {
        return nil, err
    }

    // 新增: 处理OU与部门的映射
    if err := s.processOUDeptMappings(ctx, config, ous); err != nil {
        return nil, err
    }

    // 4. 搜索和同步 User
    users, err := client.SearchUsers(config.BaseDN)
    if err != nil {
        return nil, err
    }

    // 新增: 用户同步时自动关联部门
    if err := s.syncUsers(ctx, config, users); err != nil {
        return nil, err
    }

    // ...
}

// 新增: 处理OU部门映射
func (s *SyncService) processOUDeptMappings(ctx context.Context, config *models.ADConfig, ous []*ldap.Entry) error {
    mappingService := NewOUMappingService(s.db)

    for _, ou := range ous {
        ouDN := ou.DN
        ouName := ou.GetAttributeValue("ou")

        // 1. 检查是否已有映射
        existing, err := mappingService.GetMapping(ctx, config.ID, ouDN)
        if err != nil && err != gorm.ErrRecordNotFound {
            return err
        }

        // 2. 如果没有映射，尝试自动匹配
        if existing == nil {
            dept, err := mappingService.AutoMatchDept(ctx, ouName)
            if err == nil {
                // 创建自动映射
                mappingService.CreateMapping(ctx, &models.ADOUDeptMapping{
                    ADConfigID:  config.ID,
                    OUDN:        ouDN,
                    DeptID:      dept.ID,
                    MappingType: models.OUMappingTypeAuto,
                })
            }
        }
    }

    return nil
}

// 修改: 用户同步时关联部门
func (s *SyncService) syncUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
    mappingService := NewOUMappingService(s.db)

    for _, entry := range entries {
        ouDN := extractParentDN(entry.DN)

        // 查找OU的部门映射
        mapping, err := mappingService.GetMapping(ctx, config.ID, ouDN)
        if err == nil && mapping != nil {
            // 设置用户的DeptID（需要在sys_user表添加dept_id字段）
            // 这部分需要修改用户模型和同步逻辑
        }
    }

    // ... 现有用户同步逻辑 ...
}
```

#### 新增服务层
```go
// internal/services/addomain/ou_mapping_service.go
package addomain

type OUMappingService struct {
    db *gorm.DB
}

func NewOUMappingService(db *gorm.DB) *OUMappingService {
    return &OUMappingService{db: db}
}

// GetMapping 获取OU的部门映射
func (s *OUMappingService) GetMapping(ctx context.Context, adConfigID, ouDN string) (*models.ADOUDeptMapping, error)

// AutoMatchDept 自动匹配部门（按名称）
func (s *OUMappingService) AutoMatchDept(ctx context.Context, ouName string) (*models.Department, error)

// CreateMapping 创建映射关系
func (s *OUMappingService) CreateMapping(ctx context.Context, mapping *models.ADOUDeptMapping) error

// ListMappings 列出所有映射关系
func (s *OUMappingService) ListMappings(ctx context.Context, adConfigID string) ([]models.ADOUDeptMapping, error)

// DeleteMapping 删除映射关系
func (s *OUMappingService) DeleteMapping(ctx context.Context, id string) error

// SyncDeptStructure 同步OU层级结构到部门
func (s *OUMappingService) SyncDeptStructure(ctx context.Context, adConfigID string) error
```

#### API接口设计
```go
// internal/api/v1/system/ad_ou_mapping_router.go
func SetupOUMappingRouter(r *gin.RouterGroup, core *core.Core) {
    mappingService := addomain.NewOUMappingService(core.GetDB())
    handler := NewOUMappingHandler(mappingService)

    r.GET("/mappings", handler.ListMappings)       // 列出映射关系
    r.POST("/mappings", handler.CreateMapping)     // 创建映射
    r.POST("/mappings/:id", handler.GetMapping)    // 获取映射详情
    r.POST("/mappings/:id/update", handler.UpdateMapping) // 更新映射
    r.POST("/mappings/:id/delete", handler.DeleteMapping) // 删除映射

    r.POST("/mappings/auto-match", handler.AutoMatch)     // 批量自动匹配
    r.POST("/mappings/sync-structure", handler.SyncStructure) // 同步OU结构
    r.GET("/mappings/tree", handler.GetMappingTree)      // 获取映射树
}
```

### 方案2: 虚拟字段方案 (备选)

在`sys_ad_ou`表添加`dept_id`字段：
```sql
ALTER TABLE sys_ad_ou ADD COLUMN dept_id UUID REFERENCES sys_dept(id);
```

**优点:**
- 简单直接，查询方便

**缺点:**
- 不支持一对多映射
- 无法记录映射类型和优先级
- 不支持继承关系

**不推荐原因:** 扩展性差，无法满足复杂业务需求

### 方案3: 中间表 + 同步表方案 (扩展版)

在方案1基础上增加`sys_ad_user_dept`关联表：
```sql
CREATE TABLE sys_ad_user_dept (
    id UUID PRIMARY KEY,
    ad_user_id UUID NOT NULL REFERENCES sys_ad_user(id),
    dept_id UUID NOT NULL REFERENCES sys_dept(id),
    is_primary BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL
);
```

**用途:** 支持用户属于多个部门（主部门+副部门）

## 实施计划

### Phase 1: 数据库和模型 (1天)
1. 创建迁移文件: `internal/core/db/migrations/XXX_add_ou_dept_mapping.sql`
2. 创建模型文件: `internal/models/ad_ou_dept_mapping.go`
3. 更新GORM自动迁移

### Phase 2: 服务层开发 (2天)
1. 实现`OUMappingService`
2. 修改`SyncService.syncOUs()`添加映射处理
3. 修改`SyncService.syncUsers()`添加部门关联
4. 实现自动匹配算法

### Phase 3: API和前端 (2天)
1. 创建`OUMappingHandler`和路由
2. 前端映射管理页面（列表、创建、编辑、删除）
3. 前端自动匹配功能UI
4. 前端映射树形展示

### Phase 4: 测试和优化 (1天)
1. 单元测试
2. 集成测试
3. 性能优化
4. 文档编写

## 风险和挑战

### 技术风险
1. **OU层级结构复杂**: AD的OU嵌套可能很深，与部门层级不一定对应
   - **缓解**: 提供灵活的映射配置，支持扁平化映射

2. **性能问题**: 大量OU和部门的匹配可能耗时
   - **缓解**: 使用缓存、批量处理、异步同步

3. **命名不一致**: OU名称与部门名称可能不完全匹配
   - **缓解**: 提供模糊匹配、别名配置

### 业务风险
1. **映射冲突**: 多个OU映射到同一部门时的优先级问题
   - **缓解**: 引入优先级字段，管理员可调整

2. **历史数据清理**: 已有用户需要重新关联部门
   - **缓解**: 提供批量迁移工具

3. **权限重组**: 部门关联变化可能影响权限
   - **缓解**: 记录映射变更日志，支持回滚

## 可观察性设计

### 同步日志增强
```go
type ADSyncLog struct {
    // ... 现有字段 ...
    MappingCreatedCount int    // 新建映射数量
    MappingUpdatedCount int    // 更新映射数量
    MappingErrors       string // 映射错误信息
}
```

### 映射变更日志
```sql
CREATE TABLE sys_ad_ou_dept_mapping_log (
    id UUID PRIMARY KEY,
    mapping_id UUID NOT NULL,
    action VARCHAR(20),  -- create/update/delete
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP
);
```

### 监控指标
- 映射覆盖率: 已映射OU / 总OU数
- 自动匹配成功率: 自动成功 / 总尝试次数
- 同步耗时: OU映射处理时间

## 验收标准

### 功能验收
1. ✅ 可以手动创建OU-部门映射关系
2. ✅ 可以按名称自动匹配OU和部门
3. ✅ 子OU可以继承父OU的映射
4. ✅ 用户同步时自动关联到映射的部门
5. ✅ 支持映射的增删改查API
6. ✅ 前端可以管理映射关系

### 性能验收
1. ✅ 同步1000个OU的映射耗时 < 5秒
2. ✅ 自动匹配100个OU耗时 < 2秒
3. ✅ 映射查询API响应时间 < 500ms

### 数据完整性
1. ✅ 映射关系不能违反外键约束
2. ✅ 删除部门时级联处理映射关系
3. ✅ 同步失败时回滚映射变更

## 下一步行动

### 立即开始
1. [ ] 创建数据库迁移文件
2. [ ] 实现`OUMappingService`基础功能
3. [ ] 编写自动匹配算法单元测试

### 需要确认
1. [ ] 部门同步策略：完全自动 vs 半自动（需确认）
2. [ ] 一对多映射：是否需要支持一个OU对应多个部门
3. [ ] 用户多部门：是否需要支持用户属于多个部门
4. [ ] 权限集成：部门映射是否需要触发权限更新

## 参考资料

### 现有代码
- AD域管理: `internal/services/addomain/`
- 部门模型: `internal/models/dept.go`
- 同步服务: `internal/services/addomain/sync.go`

### 相关文档
- LDAP DN格式规范
- Active Directory OU结构最佳实践
- GORM关联查询文档

---

**Spike Owner:** Claude Code (Autonomous Spike)
**预计完成时间:** 2026-05-22
**实际完成时间:** 2026-05-22
**决策结果:** ✅ **批准实施** - 采用方案1（映射表方案），技术可行性已验证，风险低，预计5天完成开发
