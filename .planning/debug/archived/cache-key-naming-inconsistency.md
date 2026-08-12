---
slug: cache-key-naming-inconsistency
status: resolved
trigger: "缓存键命名不一致问题调查"
created: "2026-06-04T00:00:00Z"
updated: "2026-06-04T00:00:00Z"
---

# 缓存键命名不一致问题调查

## Symptoms

### Expected Behavior
缓存管理界面应该显示所有系统中的缓存键，且命名方式统一规范。

### Actual Behavior
缓存管理界面显示的缓存键不完整，怀疑有部分缓存没有显示。

### Error Messages
N/A

### Timeline
问题最近被发现，具体时间未知。

### Reproduction
查看缓存管理界面，发现显示的缓存键数量少于预期。

### Additional Context
需要检查：
1. 所有缓存键的命名方式是否统一
2. 缓存管理 API 的键获取逻辑
3. 是否存在命名不规范导致键被遗漏

## Current Focus

### hypothesis
待调查：可能存在多个缓存键命名规范（有/无前缀、大小写不一致等），导致缓存管理 API 无法正确获取所有缓存键。

### next_action
gather initial evidence: 搜索所有缓存键定义位置，分析命名规范一致性。

### test
待定

### expecting
待定

### reasoning_checkpoint
待定

### tdd_checkpoint
待定

## Evidence

- timestamp: 2026-06-04T00:00:00Z
  source: codebase_analysis
  finding: |
    发现缓存键命名存在两套规范：
    1. **新规范** (internal/services/system/cache_keys.go): 
       - 使用 CacheKeyManager 构建键
       - 定义了标准键常量如 CacheKeyUserByID = "user:id"
       - 构建函数如 BuildUserCacheKey() 产生 "cache:user:id" 格式
       
    2. **旧规范** (internal/services/data_cache_service.go):
       - 直接使用简单常量如 CacheKeyUserByID = "user:id"  
       - 构建函数如 GetUserByIDKey() 产生 "user:id:uuid" 格式
       - 没有统一的 "cache:" 前缀

- timestamp: 2026-06-04T00:00:01Z
  source: cache_implementation_analysis
  finding: |
    分析发现实际的缓存键命名存在以下不一致：
    
    **Redis 层面** (pkg/cache/redis.go):
    - 所有键都自动添加 "xingran:" 前缀
    - 存储键: cache.Set(ctx, "user:1", value) → Redis 键 "xingran:user:1"
    
    **服务层面** (两套规范并存):
    - system/cache_keys.go: BuildUserCacheKey() → "cache:user:id:uuid" → "xingran:cache:user:id:uuid"
    - data_cache_service.go: GetUserByIDKey() → "user:id:uuid" → "xingran:user:id:uuid"
    
    **监控层面** (internal/services/monitor/cache_service.go):
    - normalizeCacheKeyForService() 函数会移除 "xingran:" 前缀用于显示
    - 但无法区分带 "cache:" 前缀和不带前缀的键

- timestamp: 2026-06-04T00:00:02Z
  source: cache_monitor_analysis
  finding: |
    缓存监控 API (cache_service.go) 的 GetCacheList() 方法：
    - 使用 Keys(pattern) 查询 Redis 键
    - pattern "*" 会匹配所有键，包括 "xingran:*" 格式
    - normalizeCacheKeyForService() 移除 "xingran:" 前缀显示给用户
    - **问题**: 无法区分 "cache:user:id" (新规范) 和 "user:id" (旧规范) 的键
    - **结果**: 用户看到的缓存键列表混乱，无法理解命名规则

## Eliminated

- timestamp: 2026-06-04T00:00:00Z
  hypothesis: Redis 配置问题导致键丢失
  evidence: Redis 连接正常，Keys() 方法能返回所有键
  reasoning: 问题不是键丢失，而是命名规范不统一导致展示混乱

## Resolution

### root_cause
缓存键命名存在两套并行规范导致不一致：
1. **新规范** (system/cache_keys.go): 使用 "cache:" 作为第二层前缀，产生 "cache:user:id:uuid" 格式
2. **旧规范** (data_cache_service.go): 直接使用模块名，产生 "user:id:uuid" 格式
3. **实际存储**: 两套规范在 Redis 中都存在，如 "xingran:cache:user:id:uuid" 和 "xingran:user:id:uuid"
4. **监控混乱**: 缓存管理界面无法区分这两种格式，导致用户看到的缓存键列表不一致

### fix
**建议的修复方案** (需要用户确认):

1. **统一到新规范**: 
   - 将所有旧规范键迁移到新规范格式 (user:id → cache:user:id)
   - 更新 data_cache_service.go 中的键常量和构建函数
   - 提供迁移脚本清理旧键

2. **或者统一到旧规范**:
   - 移除 system/cache_keys.go 中的 "cache:" 前缀层
   - 更新所有使用新规范的地方
   - 保留简单的 "module:key" 格式

3. **改进监控界面**:
   - 在缓存管理界面添加分组/筛选功能
   - 区分显示不同前缀的键
   - 提供键命名规范说明

**推荐方案**: 统一到新规范 (方案1)，因为：
- system/cache_keys.go 提供了更完整的键管理框架
- CacheKeyManager 提供了更好的构建和解析功能
- 便于未来扩展和管理

### verification
验证步骤：
1. 检查所有使用缓存键的地方
2. 确认新规范的迁移范围
3. 测试缓存管理界面显示效果
4. 确认缓存功能正常工作

### files_changed
待定 (取决于选择的修复方案)
可能涉及的文件：
- internal/services/data_cache_service.go
- internal/services/system/cache_keys.go  
- internal/services/system/*_cache_impl.go
- internal/services/monitor/cache_service.go
- 迁移脚本 (如需要)

