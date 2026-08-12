# Phase 20: AD域控OU与部门映射 - Discussion Results

**讨论时间:** 2026-05-22
**参与者:** 用户 + Claude Code
**状态:** ✅ 已完成

---

## 讨论问题与决策

### 问题1: OU冲突处理策略

**问题:** 当系统部门名称与AD现有OU名称冲突时，除了创建新OU，是否需要提供其他处理选项？

**用户决策:** ✅ **智能合并策略**
- 先检查OU路径是否匹配
- 如果路径一致则复用现有OU
- 否则创建新OU（带后缀避免冲突）

**影响分析:**
- ✅ 避免创建冗余的OU结构
- ✅ 保持AD域的整洁性
- ⚠️ 需要实现路径匹配算法
- ⚠️ 需要OU存在性检查逻辑

**实施调整:**
- 在`DeptToADSyncService.syncDeptTree()`中添加OU存在性检查
- 实现`checkOUPathMatch()`函数验证路径一致性
- 添加冲突解决逻辑：`resolveOUConflict()`

---

### 问题2: 用户登录时OU无映射处理

**问题:** 如果用户的OU DN在映射表中找不到对应部门，应该如何处理？

**用户决策:** ✅ **设置默认部门（如：未分配部门）**
- 确保用户有部门归属
- 便于后续管理和统计
- 不阻断登录流程

**影响分析:**
- ✅ 用户体验好，有明确的部门归属
- ✅ 便于后续手动调整
- ⚠️ 需要创建"未分配部门"或使用现有的根部门
- ⚠️ 可能需要在用户管理页面标识这些用户

**实施调整:**
- 在`UserOUService.HandleUserLoginAD()`中添加默认部门逻辑
- 配置默认部门ID（或查找"未分配"部门）
- 添加用户标记字段标识需要手动调整的用户
- 提供管理界面筛选和处理这些用户

---

### 问题3: AD同步失败处理策略

**问题:** 如果AD域控同步失败（网络问题、权限问题），应该如何处理？

**用户决策:** ✅ **仅记录日志，不阻断系统操作**
- 系统操作优先，用户体验第一
- AD同步异步重试
- 详细日志记录便于排查

**影响分析:**
- ✅ 系统功能不受AD故障影响
- ✅ 用户体验稳定
- ⚠️ 可能出现系统与AD不一致
- ⚠️ 需要重试机制和数据修复工具

**实施调整:**
- 在`UserADSyncService`中实现异步同步模式
- 添加同步状态追踪：`pending`, `synced`, `failed`
- 实现重试队列（定时任务处理失败的同步）
- 创建同步日志表记录每次尝试
- 提供管理员手动重触发的API

---

### 问题4: 映射表缓存策略

**问题:** 为了提升用户登录时的部门查找性能，应该使用什么缓存策略？

**用户决策:** ✅ **Redis缓存（5分钟TTL）**
- 高性能，支持大规模用户
- 缓存失效机制完善
- 适合分布式部署

**影响分析:**
- ✅ 查询性能优异（<1ms）
- ✅ 支持集群部署
- ✅ TTL自动失效，数据新鲜度有保证
- ⚠️ 依赖Redis服务可用性
- ⚠️ 需要缓存预热和失效逻辑

**实施调整:**
- 在`DeptOUmapper`中添加Redis缓存层
- 实现缓存键模式：`dept_ou_mapping:{config_id}:{ou_dn}`
- 添加缓存失效触发：部门同步完成时清除相关缓存
- 实现缓存预热：服务启动时加载热点映射
- 添加降级策略：Redis不可用时直接查询数据库

---

## 技术方案调整

### 新增组件

#### 1. 智能OU冲突解决器
```go
// internal/services/addomain/ou_conflict_resolver.go

type OUConflictResolver struct {
    ldap *LDAPClient
    db   *gorm.DB
}

// ResolveConflict 解决OU冲突
func (r *OUConflictResolver) ResolveConflict(ctx context.Context, deptName, parentOUDN, intendedOUDN string) (string, error) {
    // 1. 检查intendedOUDN是否存在
    exists, err := r.ldap.OUExists(intendedOUDN)
    if err != nil {
        return "", err
    }

    // 2. 如果不存在，直接使用
    if !exists {
        return intendedOUDN, nil
    }

    // 3. 如果存在，检查路径是否一致（智能合并）
    pathMatch, err := r.checkOUPathMatch(ctx, intendedOUDN, deptName)
    if err != nil {
        return "", err
    }

    if pathMatch {
        // 路径匹配，复用现有OU
        return intendedOUDN, nil
    }

    // 4. 路径不匹配，创建带后缀的新OU
    suffix := 2
    newOUDN := intendedOUDN
    for {
        newOUDN = fmt.Sprintf("OU=%s_%d,%s", deptName, suffix, parentOUDN)
        exists, err := r.ldap.OUExists(newOUDN)
        if err != nil {
            return "", err
        }
        if !exists {
            return newOUDN, nil
        }
        suffix++
    }
}
```

#### 2. 默认部门分配器
```go
// internal/services/addomain/default_dept_assigner.go

type DefaultDeptAssigner struct {
    db *gorm.DB
}

// GetDefaultDept 获取默认部门
func (a *DefaultDeptAssigner) GetDefaultDept(ctx context.Context) (*models.Department, error) {
    // 1. 尝试查找"未分配部门"
    var dept models.Department
    err := a.db.WithContext(ctx).
        Where("dept_name = ? AND status = 0", "未分配").
        First(&dept).Error
    
    if err == nil {
        return &dept, nil
    }

    // 2. 如果不存在，使用根部门
    var rootDepts []models.Department
    err = a.db.WithContext(ctx).
        Where("parent_id IS NULL OR parent_id = ''").
        Where("status = 0").
        Find(&rootDepts).Error
    
    if err == nil && len(rootDepts) > 0 {
        return &rootDepts[0], nil
    }

    // 3. 如果都没有，返回nil
    return nil, fmt.Errorf("无可用默认部门")
}

// MarkForManualReview 标记用户需要人工审核
func (a *DefaultDeptAssigner) MarkForManualReview(ctx context.Context, userID string) error {
    // 可以在用户表添加标记字段，或创建待处理任务
    return a.db.WithContext(ctx).
        Model(&models.User{}).
        Where("id = ?", userID).
        Update("dept_review_required", true).Error
}
```

#### 3. 异步同步服务
```go
// internal/services/addomain/async_sync_service.go

type AsyncSyncService struct {
    db     *gorm.DB
    ldap   *LDAPClient
    queue  chan *SyncTask
}

type SyncTask struct {
    UserID      string
    TaskType    string // "move_user" or "update_attributes"
    Data        map[string]interface{}
    MaxRetries  int
    RetryCount  int
    CreatedAt   time.Time
}

// EnqueueTask 将同步任务加入队列
func (s *AsyncSyncService) EnqueueTask(task *SyncTask) {
    s.queue <- task
}

// ProcessQueue 处理同步队列（后台任务）
func (s *AsyncSyncService) ProcessQueue(ctx context.Context) {
    for task := range s.queue {
        // 执行同步
        err := s.executeSync(ctx, task)
        
        // 记录结果
        s.logSyncResult(task, err)
        
        // 如果失败且未达最大重试次数，重新入队
        if err != nil && task.RetryCount < task.MaxRetries {
            task.RetryCount++
            time.Sleep(time.Duration(task.RetryCount) * time.Minute)
            s.queue <- task
        }
    }
}
```

#### 4. Redis缓存映射器
```go
// internal/services/addomain/cached_dept_ou_mapper.go

type CachedDeptOUMapper struct {
    db        *gorm.DB
    cache     *redis.Client
    localTTL  time.Duration // 本地缓存TTL
}

// FindDeptByOUDN 查找部门（带缓存）
func (m *CachedDeptOUMapper) FindDeptByOUDN(ctx context.Context, adConfigID, ouDN string) (string, error) {
    // 1. 尝试从Redis缓存获取
    cacheKey := fmt.Sprintf("dept_ou_mapping:%s:%s", adConfigID, ouDN)
    cached, err := m.cache.Get(ctx, cacheKey).Result()
    if err == redis.Nil {
        // 缓存未命中
    } else if err != nil {
        // Redis错误，降级到数据库查询
    } else {
        // 缓存命中
        return cached, nil
    }

    // 2. 从数据库查询
    var mapping models.DeptOUMapping
    err = m.db.WithContext(ctx).
        Where("ad_config_id = ? AND ou_dn = ? AND sync_enabled = ?", adConfigID, ouDN, true).
        First(&mapping).Error
    
    if err != nil {
        return "", err
    }

    // 3. 写入缓存（5分钟TTL）
    m.cache.Set(ctx, cacheKey, mapping.DeptID, 5*time.Minute)

    return mapping.DeptID, nil
}

// InvalidateCache 失效缓存
func (m *CachedDeptOUMapper) InvalidateCache(ctx context.Context, adConfigID string) error {
    pattern := fmt.Sprintf("dept_ou_mapping:%s:*", adConfigID)
    iter := m.cache.Scan(ctx, pattern, 0).Iterator()
    for iter.Next(ctx) {
        keys := iter.Val()
        m.cache.Del(ctx, keys)
    }
    return iter.Err()
}
```

---

## 实施计划调整

### 调整后的工作量估算

| Wave | 原估算 | 调整后 | 变化原因 |
|------|--------|--------|----------|
| Wave 1 | 1.5天 | 2天 | 增加冲突解决器、缓存层 |
| Wave 2 | 2天 | 2.5天 | 增加智能合并逻辑、缓存失效 |
| Wave 3 | 1.5天 | 1.5天 | 增加默认部门处理 |
| Wave 4 | 1天 | 1.5天 | 增加异步同步、重试队列 |
| Wave 5 | 1天 | 1天 | 无变化 |
| **总计** | **7天** | **8.5天** | **+1.5天** |

### 新增验收标准

基于讨论结果，新增以下验收标准：

**OU冲突处理:**
- ✅ OU路径一致时能正确复用现有OU
- ✅ OU路径不一致时能创建带后缀的新OU
- ✅ 冲突解决过程有详细日志记录

**默认部门处理:**
- ✅ OU无映射时用户被分配到默认部门
- ✅ 默认部门分配有记录可查
- ✅ 管理员能筛选需要手动调整的用户

**异步同步:**
- ✅ AD同步失败不阻断系统操作
- ✅ 同步任务有重试机制（最多3次）
- ✅ 同步状态可查询（pending/synced/failed）

**缓存性能:**
- ✅ 映射查询性能 < 1ms（缓存命中时）
- ✅ 缓存TTL为5分钟
- ✅ Redis不可用时降级到数据库查询

---

## 风险更新

### 新增风险
| 风险 | 级别 | 缓解措施 |
|------|------|----------|
| 智能合并算法复杂度 | Medium | 充分测试，提供手动覆盖选项 |
| 默认部门管理 | Low | 定期审核默认部门用户 |
| 异步同步队列积压 | Medium | 监控队列长度，自动扩容 |
| Redis单点故障 | Low | 使用Redis集群，本地缓存降级 |

---

## 下一步

基于这些讨论结果，现在需要：
1. ✅ 更新CONTEXT.md - 反映讨论决策
2. ✅ 更新RESEARCH.md - 调整技术方案
3. ✅ 重新生成实施计划 - 包含新增组件
4. ✅ 验证调整后的计划

---

**讨论状态:** ✅ **已完成**
**影响:** 工作量从7天增加到8.5天，技术方案更加完善
**准备状态:** 🟢 **可以继续规划阶段**
