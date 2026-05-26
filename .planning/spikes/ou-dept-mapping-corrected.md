# Spike: AD域控OU与部门映射方案（更正版）

**创建时间:** 2026-05-22
**更新时间:** 2026-05-22（根据业务细节更正）
**状态:** Ready for Implementation

## 🚨 重要更正

### 原理解偏差
**之前理解**: AD域控 → 系统部门（从AD同步组织结构到系统）
**实际需求**: 系统部门 → AD域控（从系统同步组织结构到AD）

### 核心业务逻辑
1. **部门单向同步**: 系统部门表 → AD域控OU（保持层级结构）
2. **用户登录处理**: 首次登录时，根据AD OU设置系统部门
3. **双向用户同步**: 修改用户信息（含部门）时同步到AD域控

---

## 业务场景详解

### 场景1: 部门结构同步（系统 → AD）

**部门表层级:**
```
中国太平洋财产保险股份有限公司湖北分公司 (根节点)
└── 分公司本部
    └── 科技创新部
    └── 财务部
└── 营业部
    └── 销售一部
```

**对应AD OU结构:**
```
OU=湖北分公司,DC=company,DC=com
└── OU=分公司本部,OU=湖北分公司,DC=company,DC=com
    └── OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com
    └── OU=财务部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com
└── OU=营业部,OU=湖北分公司,DC=company,DC=com
    └── OU=销售一部,OU=营业部,OU=湖北分公司,DC=company,DC=com
```

**命名规则映射:**
- 部门名称 → OU名称
- 部门层级 → OU层级

### 场景2: 用户首次登录（AD OU → 系统部门）

**流程:**
```
用户登录AD认证
    ↓
获取用户所在OU (例如: OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com)
    ↓
反向查找对应的系统部门 (科技创新部)
    ↓
设置用户的dept_id
    ↓
同步完成，用户进入系统
```

### 场景3: 用户信息修改（系统 → AD）

**修改用户部门时:**
```
管理员在系统中将用户从"科技创新部"调到"财务部"
    ↓
系统查找"财务部"对应的AD OU
    ↓
调用LDAP修改用户的ou属性 (移动用户到新OU)
    ↓
更新用户的AD属性 (department, title等)
    ↓
同步完成
```

---

## 技术方案设计

### 核心数据结构

#### 1. 部门-OU映射表（新增）
```sql
CREATE TABLE sys_dept_ou_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dept_id UUID NOT NULL REFERENCES sys_dept(id),
    ad_config_id UUID NOT NULL REFERENCES sys_ad_config(id),
    ou_dn VARCHAR(500) NOT NULL,              -- 完整的LDAP DN
    ou_name VARCHAR(255) NOT NULL,            -- OU名称
    parent_ou_dn VARCHAR(500),                -- 父OU的DN
    sync_enabled BOOLEAN DEFAULT true,        -- 是否同步到AD
    sync_status VARCHAR(20) DEFAULT 'pending', -- pending/synced/failed
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE(dept_id, ad_config_id)
);

CREATE INDEX idx_dept_ou_mapping_dn ON sys_dept_ou_mapping(ou_dn);
CREATE INDEX idx_dept_ou_mapping_dept ON sys_dept_ou_mapping(dept_id);
COMMENT ON TABLE sys_dept_ou_mapping IS '系统部门到AD OU的映射关系表';
```

#### 2. 用户表扩展（修改）
```sql
-- 确保sys_user表有以下字段
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_user_dn VARCHAR(500);  -- AD用户DN
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_ou_dn VARCHAR(500);     -- 用户所在OU的DN
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_synced_at TIMESTAMP;    -- 最后AD同步时间
```

### Go数据模型

```go
// internal/models/dept_ou_mapping.go
package models

import "time"

type DeptOUMapping struct {
    ID         string     `gorm:"primaryKey;type:uuid" json:"id"`
    DeptID     string     `gorm:"type:uuid;not null;uniqueIndex:uni_dept_ou_mapping,priority:0" json:"deptId"`
    ADConfigID string     `gorm:"type:uuid;not null;uniqueIndex:uni_dept_ou_mapping,priority:1" json:"adConfigId"`
    OUDN       string     `gorm:"size:500;not null;index:idx_dept_ou_mapping_dn" json:"ouDn"`
    OUName     string     `gorm:"size:255;not null" json:"ouName"`
    ParentOUDN *string    `gorm:"size:500" json:"parentOuDn,omitempty"`
    SyncEnabled bool      `gorm:"default:true" json:"syncEnabled"`
    SyncStatus string     `gorm:"size:20;default:pending" json:"syncStatus"` // pending/synced/failed
    LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
    CreatedAt  time.Time  `json:"createdAt"`
    UpdatedAt  time.Time  `json:"updatedAt"`

    // 关联（不持久化）
    Dept     *Department `gorm:"foreignKey:DeptID" json:"dept,omitempty"`
    ADConfig *ADConfig   `gorm:"foreignKey:ADConfigID" json:"adConfig,omitempty"`
}

func (DeptOUMapping) TableName() string {
    return "sys_dept_ou_mapping"
}
```

---

## 核心服务设计

### 1. 部门到AD同步服务

```go
// internal/services/addomain/dept_sync_service.go
package addomain

type DeptToADSyncService struct {
    db     *gorm.DB
    ldap   *LDAPClient
    mapper *DeptOUmapper
}

// SyncDeptStructureToAD 同步部门结构到AD OU
func (s *DeptToADSyncService) SyncDeptStructureToAD(ctx context.Context, adConfigID string) (*SyncResult, error) {
    // 1. 获取AD配置
    config, err := s.getADConfig(ctx, adConfigID)
    if err != nil {
        return nil, err
    }

    // 2. 连接LDAP
    if err := s.ldap.Connect(); err != nil {
        return nil, err
    }
    defer s.ldap.Close()

    // 3. 获取部门树（从根节点开始）
    rootDepts, err := s.getRootDepartments(ctx)
    if err != nil {
        return nil, err
    }

    result := &SyncResult{}

    // 4. 递归同步部门树到AD OU
    for _, dept := range rootDepts {
        if err := s.syncDeptTree(ctx, config, dept, config.BaseDN); err != nil {
            return nil, err
        }
        result.DeptCount++
    }

    return result, nil
}

// syncDeptTree 递归同步部门树
func (s *DeptToADSyncService) syncDeptTree(ctx context.Context, config *ADConfig, dept *Department, parentOUDN string) error {
    // 1. 构建当前部门的OU DN
    ouDN := fmt.Sprintf("OU=%s,%s", dept.DeptName, parentOUDN)

    // 2. 在AD中创建OU（如果不存在）
    if err := s.ldap.CreateOU(ouDN, dept.DeptName); err != nil {
        return fmt.Errorf("创建OU失败: %w", err)
    }

    // 3. 更新映射关系
    mapping := &models.DeptOUMapping{
        DeptID:     dept.ID,
        ADConfigID: config.ID,
        OUDN:       ouDN,
        OUName:     dept.DeptName,
        ParentOUDN: &parentOUDN,
        SyncStatus: "synced",
    }
    s.mapper.UpsertMapping(ctx, mapping)

    // 4. 递归处理子部门
    for _, child := range dept.Children {
        if err := s.syncDeptTree(ctx, config, child, ouDN); err != nil {
            return err
        }
    }

    return nil
}

// getRootDepartments 获取根部门（parentId为空的部门）
func (s *DeptToADSyncService) getRootDepartments(ctx context.Context) ([]*Department, error) {
    var depts []*Department
    err := s.db.WithContext(ctx).
        Where("parent_id IS NULL OR parent_id = ''").
        Where("status = 0").
        Find(&depts).Error
    return depts, err
}
```

### 2. LDAP OU操作扩展

```go
// internal/services/addomain/ldap_client.go - 新增方法

// CreateOU 在AD中创建OU
func (c *LDAPClient) CreateOU(ouDN, ouName string) error {
    // 检查OU是否已存在
    exists, err := c OUExists(ouDN)
    if err != nil {
        return err
    }
    if exists {
        return nil // 已存在，跳过
    }

    // 创建新OU
    addRequest := ldap.NewAddRequest(ouDN, nil)
    addRequest.Attribute("objectClass", []string{"organizationalUnit"})
    addRequest.Attribute("ou", []string{ouName})
    addRequest.Attribute("description", []string{fmt.Sprintf("同步自系统部门: %s", ouName)})

    return c.conn.Add(addRequest)
}

// OUExists 检查OU是否存在
func (c *LDAPClient) OUExists(ouDN string) (bool, error) {
    searchRequest := ldap.NewSearchRequest(
        ouDN,
        ldap.ScopeBaseObject,
        ldap.NeverDerefAliases,
        0, 0, false,
        "(objectClass=*)",
        []string{"dn"},
        nil,
    )

    _, err := c.conn.Search(searchRequest)
    if err != nil {
        if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

// MoveUser 移动用户到新OU
func (c *LDAPClient) MoveUser(userDN, newOUDN string) error {
    // LDAP modify DN操作可以移动条目
    modifyDNRequest := ldap.NewModifyDNRequest(userDN, extractRDN(userDN), true, newOUDN)
    return c.conn.ModifyDN(modifyDNRequest)
}

// UpdateUserAttributes 更新用户属性
func (c *LDAPClient) UpdateUserAttributes(userDN string, attributes map[string]string) error {
    modifyRequest := ldap.NewModifyRequest(userDN, nil)

    for attr, value := range attributes {
        modifyRequest.Replace(attr, []string{value})
    }

    return c.conn.Modify(modifyRequest)
}
```

### 3. 用户登录时的OU处理服务

```go
// internal/services/addomain/user_ou_service.go
package addomain

type UserOUService struct {
    db     *gorm.DB
    mapper *DeptOUmapper
}

// HandleUserLoginAD 处理用户AD登录后的部门设置
func (s *UserOUService) HandleUserLoginAD(ctx context.Context, username, adOUDN string) error {
    // 1. 通过用户名查找系统用户
    var user models.User
    if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            // 用户不存在，跳过部门设置（由注册流程处理）
            return nil
        }
        return err
    }

    // 2. 通过AD OU DN查找对应的部门
    deptID, err := s.mapper.FindDeptByOUDN(ctx, adOUDN)
    if err != nil {
        // 未找到映射部门，记录日志但不阻断登录
        applogger.Warnf("用户 %s 的AD OU %s 未找到对应部门", username, adOUDN)
        return nil
    }

    // 3. 更新用户的部门
    if err := s.db.WithContext(ctx).
        Model(&user).
        Update("dept_id", deptID).
        Error; err != nil {
        return fmt.Errorf("更新用户部门失败: %w", err)
    }

    // 4. 记录AD信息到用户表
    s.db.WithContext(ctx).
        Model(&user).
        Updates(map[string]interface{}{
            "ad_user_dn": adOUDN, // 这里应该是用户的完整DN，需要从登录时获取
            "ad_ou_dn":    adOUDN,
            "ad_synced_at": time.Now(),
        })

    applogger.Infof("用户 %s 登录时自动设置部门: dept_id=%s", username, deptID)
    return nil
}
```

### 4. 用户信息修改时的AD同步服务

```go
// internal/services/addomain/user_ad_sync_service.go
package addomain

type UserADSyncService struct {
    db   *gorm.DB
    ldap *LDAPClient
    mapper *DeptOUmapper
}

// SyncUserUpdateToAD 同步用户更新到AD域控
func (s *UserADSyncService) SyncUserUpdateToAD(ctx context.Context, userID string, updateReq *UpdateUserRequest) error {
    // 1. 获取用户信息
    var user models.User
    if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
        return err
    }

    // 如果用户没有AD DN，跳过同步
    if user.ADUserDN == nil || *user.ADUserDN == "" {
        return nil
    }

    // 2. 如果部门变更，需要移动用户到新OU
    if updateReq.DeptID != nil && *updateReq.DeptID != user.DeptID {
        // 查找新部门对应的OU DN
        ouDN, err := s.mapper.FindOUDNByDeptID(ctx, *updateReq.DeptID)
        if err != nil {
            return fmt.Errorf("查找部门OU失败: %w", err)
        }

        // 移动用户到新OU
        if err := s.ldap.MoveUser(*user.ADUserDN, ouDN); err != nil {
            return fmt.Errorf("移动用户到新OU失败: %w", err)
        }

        // 更新用户的ad_ou_dn
        user.ADOUDN = &ouDN
    }

    // 3. 更新其他属性到AD
    attributes := make(map[string]string)

    if updateReq.DisplayName != nil {
        attributes["displayName"] = *updateReq.DisplayName
    }
    if updateReq.Email != nil {
        attributes["mail"] = *updateReq.Email
    }
    if updateReq.Phone != nil {
        attributes["telephoneNumber"] = *updateReq.Phone
    }
    if updateReq.Mobile != nil {
        attributes["mobile"] = *updateReq.Mobile
    }
    if updateReq.Title != nil {
        attributes["title"] = *updateReq.Title
    }

    // 同步部门名称到department属性
    if updateReq.DeptID != nil {
        var dept models.Department
        if err := s.db.WithContext(ctx).Where("id = ?", *updateReq.DeptID).First(&dept).Error; err == nil {
            attributes["department"] = dept.DeptName
        }
    }

    if len(attributes) > 0 {
        if err := s.ldap.UpdateUserAttributes(*user.ADUserDN, attributes); err != nil {
            return fmt.Errorf("更新AD用户属性失败: %w", err)
        }
    }

    return nil
}
```

### 5. 部门-OU映射服务

```go
// internal/services/addomain/dept_ou_mapper.go
package addomain

type DeptOUmapper struct {
    db *gorm.DB
}

// FindDeptByOUDN 通过OU DN查找部门ID
func (m *DeptOUmapper) FindDeptByOUDN(ctx context.Context, ouDN string) (string, error) {
    var mapping models.DeptOUMapping
    err := m.db.WithContext(ctx).
        Where("ou_dn = ? AND sync_enabled = ?", ouDN, true).
        First(&mapping).Error
    if err != nil {
        return "", err
    }
    return mapping.DeptID, nil
}

// FindOUDNByDeptID 通过部门ID查找OU DN
func (m *DeptOUmapper) FindOUDNByDeptID(ctx context.Context, deptID string) (string, error) {
    var mapping models.DeptOUMapping
    err := m.db.WithContext(ctx).
        Where("dept_id = ? AND sync_enabled = ?", deptID, true).
        First(&mapping).Error
    if err != nil {
        return "", err
    }
    return mapping.OUDN, nil
}

// UpsertMapping 创建或更新映射关系
func (m *DeptOUmapper) UpsertMapping(ctx context.Context, mapping *models.DeptOUMapping) error {
    return m.db.WithContext(ctx).
        Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "dept_id"}, {Name: "ad_config_id"}},
            DoUpdates: clause.AssignmentColumns([]string{"ou_dn", "ou_name", "parent_ou_dn", "sync_status", "last_sync_at", "updated_at"}),
        }).
        Create(mapping).Error
}
```

---

## API接口设计

### 1. 部门同步API

```go
// internal/api/v1/system/ad_dept_sync_router.go

func SetupADDeptSyncRouter(r *gin.RouterGroup, core *core.Core) {
    syncService := addomain.NewDeptToADSyncService(core.GetDB())
    handler := NewADDeptSyncHandler(syncService)

    // 同步部门结构到AD
    r.POST("/sync/dept-to-ad", handler.SyncDeptToAD)

    // 查看同步状态
    r.GET("/sync/dept-status/:configId", handler.GetDeptSyncStatus)

    // 手动触发同步
    r.POST("/sync/dept-trigger", handler.TriggerDeptSync)
}
```

### 2. 用户AD信息API

```go
// internal/api/v1/system/user_ad_router.go (扩展)

// 在现有用户更新接口中添加AD同步逻辑
func (h *UserHandler) UpdateUser(c *gin.Context) {
    // ... 现有逻辑 ...

    // 新增: 同步到AD域控
    if h.adSyncService != nil {
        if err := h.adSyncService.SyncUserUpdateToAD(ctx, id, req); err != nil {
            applogger.Warnf("同步用户到AD失败: %v", err)
            // 不阻断主流程，只记录错误
        }
    }

    response.Success(c, user)
}
```

---

## 定时任务配置

```yaml
# configs/config.yaml

scheduler:
  jobs:
    - name: "部门结构同步到AD域控"
      cron: "0 0 2 * * *"  # 每天凌晨2点执行
      type: "dept_to_ad_sync"
      enabled: true
      description: "将系统部门结构同步到AD域控OU"
```

```go
// internal/scheduler/dept_sync_job.go

func RegisterDeptSyncJob(scheduler *Scheduler, service *DeptToADSyncService) {
    scheduler.RegisterJob("dept_to_ad_sync", func() error {
        ctx := context.Background()
        _, err := service.SyncDeptStructureToAD(ctx, getDefaultADConfigID())
        return err
    })
}
```

---

## 修改后的数据流

### 部门同步流程（定时任务）
```
定时任务触发 (每天凌晨2点)
    ↓
读取部门树 (sys_dept)
    ↓
连接AD域控 (LDAP)
    ↓
递归创建/更新OU结构
    ↓
更新映射表 (sys_dept_ou_mapping)
    ↓
记录同步日志
```

### 用户登录流程
```
用户输入AD账号密码
    ↓
LDAP认证成功
    ↓
获取用户所在OU DN
    ↓
查找映射表，获取对应部门ID
    ↓
更新sys_user.dept_id
    ↓
记录AD用户DN和OU DN
    ↓
生成JWT Token，返回登录成功
```

### 用户信息修改流程
```
管理员修改用户信息（部门/姓名/电话等）
    ↓
更新sys_user表
    ↓
查找用户新部门对应的OU DN
    ↓
调用LDAP API:
    - 如果部门变更: 移动用户到新OU
    - 更新用户属性: displayName, mail, telephoneNumber, title, department
    ↓
更新ad_synced_at时间戳
```

---

## 实施计划（修正版）

### Phase 1: 数据库和基础服务 (2天)
1. 创建`sys_dept_ou_mapping`表迁移
2. 扩展`sys_user`表字段
3. 实现`DeptOUmapper`基础功能
4. 扩展LDAP客户端（CreateOU, MoveUser等）

### Phase 2: 部门同步功能 (2天)
1. 实现`DeptToADSyncService`
2. 递归同步部门树到AD OU
3. 定时任务注册和配置
4. 同步日志和错误处理

### Phase 3: 用户登录集成 (1天)
1. 实现`UserOUService`
2. 登录流程集成OU处理
3. 用户首次登录部门自动设置

### Phase 4: 用户修改同步 (1天)
1. 实现`UserADSyncService`
2. 用户更新接口集成AD同步
3. 部门变更时OU移动逻辑

### Phase 5: 测试和优化 (1天)
1. 端到端测试
2. 性能优化
3. 错误处理和重试机制

**总计: 7天**

---

## 风险和挑战

### 技术风险

**1. AD OU权限问题**
- 风险: AD管理员可能不允许程序创建/修改OU
- 缓解: 提供详细的AD权限配置文档，使用专用服务账号

**2. OU命名冲突**
- 风险: 部门名称可能与现有OU冲突
- 缓解: 同步前检查OU是否存在，使用唯一ID作为备选方案

**3. 大量用户的OU移动**
- 风险: 批量移动用户可能耗尽AD连接
- 缓解: 分批处理，控制并发数

### 业务风险

**1. 部门结构不一致**
- 风险: 系统部门结构与AD实际OU结构差异大
- 缓解: 提供手动映射覆盖功能，支持部分同步

**2. 用户登录性能**
- 风险: 登录时查询映射可能影响性能
- 缓解: 使用缓存存储映射关系，异步更新

**3. 同步失败回滚**
- 风险: AD同步失败但系统已更新
- 缓解: 采用"先AD后系统"策略，AD失败则回滚系统操作

---

## 验收标准

### 功能验收
1. ✅ 定时任务能正确同步部门树到AD OU
2. ✅ 用户首次登录时自动设置正确的部门
3. ✅ 修改用户部门时能同步移动用户到新OU
4. ✅ 修改用户属性时能同步更新AD属性
5. ✅ 支持手动触发部门同步
6. ✅ 提供同步状态查询接口

### 性能验收
1. ✅ 同步100个部门到AD耗时 < 10秒
2. ✅ 用户登录时部门设置耗时 < 100ms（使用缓存）
3. ✅ 移动100个用户到新OU耗时 < 30秒

### 数据完整性
1. ✅ 映射表数据与实际AD OU结构一致
2. ✅ 用户表中的ad_ou_dn与AD实际位置一致
3. ✅ 同步失败时有详细日志和错误信息

---

## 参考资料

### LDAP操作
- LDAP ModifyDN操作（移动条目）
- LDAP Add操作（创建OU）
- LDAP Modify操作（更新属性）

### 现有代码
- 部门模型: `internal/models/dept.go`
- 用户模型: `internal/models/user.go`
- LDAP客户端: `internal/services/addomain/ldap_client.go`

---

**Spike Owner:** Claude Code (Autonomous Spike)
**更新时间:** 2026-05-22
**状态:** ✅ **更正完成，准备实施**
**决策结果:** 采用修正方案（系统→AD），保持部门层级结构，用户登录时设置部门，修改时同步到AD
