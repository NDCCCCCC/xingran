---
task: unify-cache-key-naming
id: 260604-lr9
created: "2026-06-04T07:39:54Z"
status: pending
---

# 统一缓存键命名规范到新标准

## 目标
将所有旧规范缓存键迁移到新标准（CacheKeyManager），确保命名规范统一。

## 背景
当前系统存在两套缓存键命名规范：
- **新规范**：使用 `CacheKeyManager` 构建，产生 `cache:user:id:uuid` 格式
- **旧规范**：直接常量定义，产生 `user:id:uuid` 格式

这导致 Redis 中同时存在两种格式的键，缓存管理界面显示混乱。

## 实施计划

### 1. 分析现有缓存键使用情况
- 搜索所有使用旧规范键常量的位置
- 列出需要迁移的服务和模块

### 2. 更新 data_cache_service.go
- 替换旧规范键常量为新规范格式
- 更新键构建函数使用 `CacheKeyManager`

### 3. 更新使用旧规范的代码
- 更新所有调用旧规范键构建函数的地方
- 确保新规范键格式一致

### 4. 提供清理旧键的方案
- 编写迁移建议或脚本
- 说明如何清理 Redis 中的旧键

### 5. 验证
- 确认缓存功能正常工作
- 检查缓存管理界面显示统一

## 涉及文件
- `internal/services/data_cache_service.go` - 旧规范键定义
- `internal/services/system/cache_keys.go` - 新规范键定义
- `internal/services/system/*_cache_impl.go` - 使用新规范的服务实现
- `internal/services/monitor/cache_service.go` - 缓存监控服务

## 约束
- 保持缓存功能向后兼容
- 避免影响现有业务逻辑
- 确保缓存性能不受影响
