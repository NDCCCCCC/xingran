# Phase 20: 配置参数管理方案

**目标:** 确保Phase 20所有功能都可以通过参数管理页面热启停，关闭时完全不影响系统运行

## 配置参数设计

### 核心原则
1. **每个功能都有独立的配置开关**
2. **配置变更实时生效，无需重启服务**
3. **功能关闭时有明确的降级策略**
4. **配置参数集中管理，便于运维**

---

## Phase 20配置参数清单

### 1. 总体功能开关

#### 1.1 部门OU映射功能总开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('AD部门OU映射功能', 'sys.ad.dept_ou_mapping.enabled', 'false', 'Y', 'Y',
 '启用部门与AD OU的双向映射功能，包括定时同步、登录设置、修改同步等所有子功能');
```

**功能影响:**
- `false`: 完全禁用Phase 20所有功能
- `true`: 启用Phase 20，受以下子功能开关控制

**降级策略:**
- 定时同步任务不执行
- 用户登录时不处理OU映射
- 用户修改时不同步到AD

---

### 2. 部门定时同步功能

#### 2.1 部门同步开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('部门到AD定时同步', 'sys.ad.dept_ou_mapping.sync_enabled', 'true', 'Y', 'Y',
 '是否启用定时将系统部门树同步到AD域控OU（每天凌晨2点执行）');
```

**功能影响:**
- `false`: 不执行部门定时同步任务
- `true`: 按计划执行同步

**降级策略:**
- 系统部门正常CRUD操作不受影响
- 不影响其他Phase 20功能（登录、修改同步）

#### 2.2 AD配置ID选择
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('AD域控配置ID', 'sys.ad.dept_ou_mapping.ad_config_id', '', 'Y', 'Y',
 '部门同步使用的AD域控配置ID（为空则使用第一个启用的配置）');
```

**功能影响:**
- 为空: 自动选择第一个启用的AD配置
- 指定ID: 使用指定的AD配置进行同步

**降级策略:**
- 配置无效时记录错误日志，不执行同步

#### 2.3 部门同步时间配置
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('部门同步Cron表达式', 'sys.ad.dept_ou_mapping.sync_cron', '0 0 2 * * *', 'N', 'Y',
 '部门定时同步的Cron表达式，默认每天凌晨2点，格式：分 时 日 月 周');
```

**功能影响:**
- 控制定时任务的执行时间
- 可根据业务需求调整

**降级策略:**
- Cron表达式无效时使用默认值（每天2点）

---

### 3. 用户登录部门设置功能

#### 3.1 登录时部门设置开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('用户登录时设置部门', 'sys.ad.dept_ou_mapping.login_dept_enabled', 'true', 'Y', 'Y',
 '是否在用户AD认证登录时，根据所在OU自动设置系统部门');
```

**功能影响:**
- `false`: 用户AD登录成功后不处理部门设置
- `true`: 根据用户OU DN反向查找部门并设置

**降级策略:**
- 用户正常登录，只是部门为空或保持原部门
- 不影响用户认证和系统访问

#### 3.2 OU无映射处理策略
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('OU无映射处理策略', 'sys.ad.dept_ou_mapping.unmapped_ou_policy', 'default_dept', 'N', 'Y',
 '当用户OU在映射表中找不到对应部门时的处理策略：default_dept=设置默认部门, skip=跳过部门设置');
```

**功能影响:**
- `default_dept`: 设置为默认部门（根部门或"未分配"部门）
- `skip`: 跳过部门设置，记录日志

**降级策略:**
- 策略无效时默认使用`default_dept`

#### 3.3 默认部门ID配置
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('OU无映射默认部门', 'sys.ad.dept_ou_mapping.default_dept_id', '', 'N', 'Y',
 '用户OU无映射时设置的默认部门ID（为空则使用根部门或创建"未分配"部门）');
```

**功能影响:**
- 为空: 使用根部门或创建"未分配"部门
- 指定ID: 使用指定的部门

**降级策略:**
- 部门不存在时记录错误，不阻断登录

---

### 4. 用户修改AD同步功能

#### 4.1 用户修改同步开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('用户修改同步到AD', 'sys.ad.dept_ou_mapping.user_sync_enabled', 'true', 'Y', 'Y',
 '是否在管理员修改用户信息（部门、属性）时同步到AD域控');
```

**功能影响:**
- `false`: 用户修改只更新系统数据库，不同步到AD
- `true`: 用户修改时同步到AD域控

**降级策略:**
- 系统操作正常执行，AD同步失败仅记录日志
- 不影响系统功能的正常使用

#### 4.2 同步模式配置
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('AD同步模式', 'sys.ad.dept_ou_mapping.sync_mode', 'async', 'N', 'Y',
 '用户修改同步到AD的模式：async=异步同步（不阻塞），sync=同步同步（阻塞但可靠）');
```

**功能影响:**
- `async`: 异步同步，系统操作立即返回，后台同步到AD
- `sync`: 同步同步，等待AD操作成功后才返回

**降级策略:**
- `async`模式：AD失败不阻断系统操作
- `sync`模式：AD失败记录日志，不影响已完成的系统操作

#### 4.3 同步失败重试配置
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('同步失败重试次数', 'sys.ad.dept_ou_mapping.retry_max_count', '3', 'N', 'Y',
 'AD同步失败时的最大重试次数（0表示不重试）');
```

**功能影响:**
- 控制异步同步的重试次数
- 重试间隔递增：1分钟、2分钟、4分钟

**降级策略:**
- 达到最大重试次数后标记为失败，记录详细日志

---

### 5. 高级功能开关

#### 5.1 智能OU冲突解决开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('智能OU冲突解决', 'sys.ad.dept_ou_mapping.smart_conflict_resolve', 'true', 'Y', 'Y',
 '启用智能OU冲突解决：检查OU路径是否一致，一致则复用，否则创建带后缀的新OU');
```

**功能影响:**
- `true`: 使用智能冲突解决策略
- `false`: 总是创建新OU（带数字后缀）

**降级策略:**
- 关闭时可能创建更多冗余OU，但不影响功能

#### 5.2 部门同步OU冲突后缀
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('OU冲突后缀格式', 'sys.ad.dept_ou_mapping.conflict_suffix_format', '{name}_{index}', 'N', 'Y',
 'OU冲突时的后缀格式，{name}=部门名，{index}=序号，示例：销售部_2');
```

**功能影响:**
- 控制冲突OU的命名格式

**降级策略:**
- 格式无效时使用默认值

#### 5.3 映射缓存开关
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('OU映射缓存启用', 'sys.ad.dept_ou_mapping.cache_enabled', 'true', 'Y', 'Y',
 '是否启用OU映射关系的Redis缓存（关闭则每次查询数据库）');
```

**功能影响:**
- `true`: 使用Redis缓存，性能更好
- `false`: 每次查询数据库，性能较慢但数据最新

**降级策略:**
- Redis不可用时自动降级到数据库查询

#### 5.4 缓存TTL配置
```sql
INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
('映射缓存过期时间', 'sys.ad.dept_ou_mapping.cache_ttl_seconds', '300', 'N', 'Y',
 'OU映射缓存的过期时间（秒），默认5分钟，最小60秒，最大3600秒');
```

**功能影响:**
- 控制缓存的刷新频率

**降级策略:**
- 超出范围时使用默认值300秒

---

## 功能依赖关系

```
sys.ad.dept_ou_mapping.enabled (总开关)
    ├── sys.ad.dept_ou_mapping.sync_enabled (部门定时同步)
    │   ├── sys.ad.dept_ou_mapping.ad_config_id
    │   ├── sys.ad.dept_ou_mapping.sync_cron
    │   └── sys.ad.dept_ou_mapping.smart_conflict_resolve
    ├── sys.ad.dept_ou_mapping.login_dept_enabled (登录时部门设置)
    │   ├── sys.ad.dept_ou_mapping.unmapped_ou_policy
    │   └── sys.ad.dept_ou_mapping.default_dept_id
    └── sys.ad.dept_ou_mapping.user_sync_enabled (用户修改同步)
        ├── sys.ad.dept_ou_mapping.sync_mode
        ├── sys.ad.dept_ou_mapping.retry_max_count
        └── sys.ad.dept_ou_mapping.cache_enabled
            └── sys.ad.dept_ou_mapping.cache_ttl_seconds
```

---

## 热启停实现模式

### 配置读取服务
```go
// internal/services/addomain/config_service.go

type DeptOUMappingConfig struct {
    // 总开关
    Enabled bool
    
    // 部门同步配置
    SyncEnabled      bool
    ADConfigID       string
    SyncCron          string
    
    // 登录部门设置配置
    LoginDeptEnabled  bool
    UnmappedOUPolicy  string // "default_dept" | "skip"
    DefaultDeptID     string
    
    // 用户修改同步配置
    UserSyncEnabled   bool
    SyncMode          string // "async" | "sync"
    RetryMaxCount     int
    
    // 高级功能配置
    SmartConflictResolve bool
    ConflictSuffixFormat  string
    CacheEnabled         bool
    CacheTTLSeconds      int
}

// GetConfig 从数据库读取配置（带缓存）
func GetConfig(ctx context.Context, db *gorm.DB) (*DeptOUMappingConfig, error) {
    config := &DeptOUMappingConfig{}
    
    // 总开关
    config.Enabled = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.enabled", false)
    if !config.Enabled {
        return config, nil // 总开关关闭，其他配置不读取
    }
    
    // 部门同步配置
    config.SyncEnabled = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.sync_enabled", true)
    config.ADConfigID = getStringConfig(ctx, db, "sys.ad.dept_ou_mapping.ad_config_id", "")
    config.SyncCron = getStringConfig(ctx, db, "sys.ad.dept_ou_mapping.sync_cron", "0 0 2 * * *")
    config.SmartConflictResolve = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.smart_conflict_resolve", true)
    
    // 登录部门设置配置
    config.LoginDeptEnabled = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.login_dept_enabled", true)
    config.UnmappedOUPolicy = getStringConfig(ctx, db, "sys.ad.dept_ou_mapping.unmapped_ou_policy", "default_dept")
    config.DefaultDeptID = getStringConfig(ctx, db, "sys.ad.dept_ou_mapping.default_dept_id", "")
    
    // 用户修改同步配置
    config.UserSyncEnabled = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.user_sync_enabled", true)
    config.SyncMode = getStringConfig(ctx, db, "sys.ad.dept_ou_mapping.sync_mode", "async")
    config.RetryMaxCount = getIntConfig(ctx, db, "sys.ad.dept_ou_mapping.retry_max_count", 3)
    
    // 缓存配置
    config.CacheEnabled = getBoolConfig(ctx, db, "sys.ad.dept_ou_mapping.cache_enabled", true)
    config.CacheTTLSeconds = getIntConfig(ctx, db, "sys.ad.dept_ou_mapping.cache_ttl_seconds", 300)
    
    return config, nil
}
```

---

## 功能降级矩阵

| 功能 | 配置关闭 | 执行失败 | 降级策略 |
|------|---------|---------|----------|
| **部门定时同步** | 不执行任务 | 记录错误，继续下次 | 系统部门CRUD正常 |
| **登录时部门设置** | 跳过部门处理 | 记录警告，用户正常登录 | 用户可以正常使用系统 |
| **用户修改AD同步** | 仅更新系统数据库 | 记录日志，系统操作成功 | 系统数据已更新，AD异步重试 |
| **智能冲突解决** | 直接创建带后缀OU | 使用简单后缀策略 | 可能创建更多OU但不影响功能 |
| **Redis缓存** | 每次查询数据库 | 自动降级到数据库查询 | 性能下降但功能正常 |

---

## 配置变更实时生效机制

### 1. 定时任务动态调整
```go
// 内部定时任务框架支持
func (s *DeptToADSyncService) CheckConfig(ctx context.Context) bool {
    config, _ := addomain.GetConfig(ctx, s.db)
    return config.Enabled && config.SyncEnabled
}

// 定时任务每次执行前检查配置
scheduler.AddJob("dept_to_ad_sync", func() error {
    if !service.CheckConfig(ctx) {
        return nil // 配置关闭，跳过本次执行
    }
    return service.SyncDeptStructureToAD(ctx, configID)
})
```

### 2. 登录流程动态检查
```go
func (h *AuthHandler) Login(c *gin.Context) {
    // ... AD认证成功后 ...
    
    // 动态检查配置
    config, _ := addomain.GetConfig(ctx, h.db)
    if !config.Enabled || !config.LoginDeptEnabled {
        // 功能关闭，不处理部门设置
        return response.Success(c, token)
    }
    
    // 功能开启，执行部门设置
    userOUService.SetUserDepartment(ctx, userDN, username)
}
```

### 3. 用户修改动态同步
```go
func (s *UserADSyncService) SyncUserUpdateToAD(ctx context.Context, userID string, req *UpdateUserRequest) error {
    // 动态检查配置
    config, _ := addomain.GetConfig(ctx, s.db)
    if !config.Enabled || !config.UserSyncEnabled {
        return nil // 功能关闭，不同步到AD
    }
    
    // 根据同步模式执行
    if config.SyncMode == "async" {
        go s.syncAsync(ctx, userID, req) // 异步同步
    } else {
        return s.syncSync(ctx, userID, req) // 同步同步
    }
}
```

---

## 参数管理页面配置

### 配置分组展示
在参数管理页面中，Phase 20的配置应该分组展示：

#### 组1: 功能总开关
- AD部门OU映射功能 (主开关)

#### 组2: 部门同步配置
- 部门到AD定时同步
- AD域控配置ID
- 部门同步Cron表达式
- 智能OU冲突解决

#### 组3: 用户登录配置
- 用户登录时设置部门
- OU无映射处理策略
- OU无映射默认部门

#### 组4: 用户修改同步配置
- 用户修改同步到AD
- AD同步模式
- 同步失败重试次数

#### 组5: 性能优化配置
- OU映射缓存启用
- 映射缓存过期时间

---

## 配置验证和安全

### 配置验证规则
```go
func validateConfig(key, value string) error {
    switch key {
    case "sys.ad.dept_ou_mapping.sync_cron":
        // 验证Cron表达式格式
        if _, err := cron.ParseStandard(value); err != nil {
            return fmt.Errorf("无效的Cron表达式: %w", err)
        }
        
    case "sys.ad.dept_ou_mapping.unmapped_ou_policy":
        if value != "default_dept" && value != "skip" {
            return fmt.Errorf("无效的策略，只支持: default_dept, skip")
        }
        
    case "sys.ad.dept_ou_mapping.sync_mode":
        if value != "async" && value != "sync" {
            return fmt.Errorf("无效的同步模式，只支持: async, sync")
        }
        
    case "sys.ad.dept_ou_mapping.cache_ttl_seconds":
        ttl, _ := strconv.Atoi(value)
        if ttl < 60 || ttl > 3600 {
            return fmt.Errorf("缓存TTL必须在60-3600秒之间")
        }
    }
    return nil
}
```

---

## 迁移脚本

```sql
-- Phase 20配置参数迁移
-- 文件: internal/core/db/migrations/XXX_add_phase20_ad_dept_ou_mapping_config.sql

INSERT INTO sys_config (config_name, config_key, config_value, config_type, is_system, remark) VALUES
-- 总开关
('AD部门OU映射功能', 'sys.ad.dept_ou_mapping.enabled', 'false', 'Y', 'Y',
 '启用部门与AD OU的双向映射功能，包括定时同步、登录设置、修改同步等所有子功能'),

-- 部门同步配置
('部门到AD定时同步', 'sys.ad.dept_ou_mapping.sync_enabled', 'true', 'Y', 'Y',
 '是否启用定时将系统部门树同步到AD域控OU（每天凌晨2点执行）'),
('AD域控配置ID', 'sys.ad.dept_ou_mapping.ad_config_id', '', 'Y', 'Y',
 '部门同步使用的AD域控配置ID（为空则使用第一个启用的配置）'),
('部门同步Cron表达式', 'sys.ad.dept_ou_mapping.sync_cron', '0 0 2 * * *', 'N', 'Y',
 '部门定时同步的Cron表达式，默认每天凌晨2点，格式：分 时 日 月 周'),

-- 登录部门设置配置
('用户登录时设置部门', 'sys.ad.dept_ou_mapping.login_dept_enabled', 'true', 'Y', 'Y',
 '是否在用户AD认证登录时，根据所在OU自动设置系统部门'),
('OU无映射处理策略', 'sys.ad.dept_ou_mapping.unmapped_ou_policy', 'default_dept', 'N', 'Y',
 '当用户OU在映射表中找不到对应部门时的处理策略：default_dept=设置默认部门, skip=跳过部门设置'),
('OU无映射默认部门', 'sys.ad.dept_ou_mapping.default_dept_id', '', 'N', 'Y',
 '用户OU无映射时设置的默认部门ID（为空则使用根部门或创建"未分配"部门）'),

-- 用户修改同步配置
('用户修改同步到AD', 'sys.ad.dept_ou_mapping.user_sync_enabled', 'true', 'Y', 'Y',
 '是否在管理员修改用户信息（部门、属性）时同步到AD域控'),
('AD同步模式', 'sys.ad.dept_ou_mapping.sync_mode', 'async', 'N', 'Y',
 '用户修改同步到AD的模式：async=异步同步（不阻塞），sync=同步同步（阻塞但可靠）'),
('同步失败重试次数', 'sys.ad.dept_ou_mapping.retry_max_count', '3', 'N', 'Y',
 'AD同步失败时的最大重试次数（0表示不重试）'),

-- 高级功能配置
('智能OU冲突解决', 'sys.ad.dept_ou_mapping.smart_conflict_resolve', 'true', 'Y', 'Y',
 '启用智能OU冲突解决：检查OU路径是否一致，一致则复用，否则创建带后缀的新OU'),
('OU冲突后缀格式', 'sys.ad.dept_ou_mapping.conflict_suffix_format', '{name}_{index}', 'N', 'Y',
 'OU冲突时的后缀格式，{name}=部门名，{index}=序号，示例：销售部_2'),
('OU映射缓存启用', 'sys.ad.dept_ou_mapping.cache_enabled', 'true', 'Y', 'Y',
 '是否启用OU映射关系的Redis缓存（关闭则每次查询数据库）'),
('映射缓存过期时间', 'sys.ad.dept_ou_mapping.cache_ttl_seconds', '300', 'N', 'Y',
 'OU映射缓存的过期时间（秒），默认5分钟，最小60秒，最大3600秒')

ON CONFLICT (config_key) DO NOTHING;
```

---

## 运维手册

### 功能启停操作流程

#### 启用所有功能
1. 进入参数管理页面
2. 找到"AD部门OU映射功能"，设置为`true`
3. 保存配置（立即生效）

#### 仅启用部门同步，禁用其他功能
1. "AD部门OU映射功能" = `true`
2. "部门到AD定时同步" = `true`
3. "用户登录时设置部门" = `false`
4. "用户修改同步到AD" = `false`

#### 紧急停用所有功能（不影响系统运行）
1. "AD部门OU映射功能" = `false`
2. 保存配置（立即生效，所有Phase 20功能停止）

#### 性能问题排查
1. 如果Redis缓存有问题，"OU映射缓存启用" = `false`
2. 系统自动降级到数据库查询，功能正常但性能下降

#### 故障隔离
- AD域控故障：关闭"用户修改同步到AD"，系统正常运行
- 同步性能问题：关闭"智能OU冲突解决"，使用简单策略
- 缓存问题：系统自动降级到数据库查询

---

**配置参数总数:** 13个
**功能独立开关数:** 5个（总开关 + 4个子功能开关）
**降级保证:** 100%（所有功能关闭时系统正常运行）

---

**建议执行时间:** 在Phase 20 Wave 1中一起实施配置参数创建
**测试重点:** 验证每个开关的独立性和降级策略的正确性
