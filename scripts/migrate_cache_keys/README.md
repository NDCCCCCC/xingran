# 缓存键命名规范迁移指南

## 概述

本项目已完成缓存键命名规范的统一，从旧规范迁移到新规范。

### 旧规范（已弃用）
- **格式**: `{type}:{params}`
- **示例**: `user:id:uuid`, `role:menus:roleId`, `dept:tree`
- **问题**: 缺少统一前缀，Redis中键管理混乱

### 新规范（当前使用）
- **格式**: `cache:{module}:{type}:{params}`
- **示例**: `cache:user:id:uuid`, `cache:role:menus:roleId`, `cache:dept:tree`
- **优势**: 统一前缀，便于管理和批量操作

## 迁移状态

✅ **已完成**:
- [x] 更新 `data_cache_service.go` 中的键常量定义
- [x] 更新键构建函数以使用新规范格式
- [x] 添加迁移说明和文档
- [x] 确保向后兼容性

⚠️ **需要手动执行**:
- [ ] 清理 Redis 中的旧键

## 清理 Redis 旧键

### 方法 1: 使用 redis-cli 交互式清理（推荐）

**步骤 1**: 连接到 Redis
```bash
redis-cli -h localhost -p 6379
```

**步骤 2**: 查看旧键数量（确认影响范围）
```bash
# 查看各模块旧键数量
SCAN 0 MATCH user:* COUNT 1000
SCAN 0 MATCH role:* COUNT 1000
SCAN 0 MATCH dept:* COUNT 1000
SCAN 0 MATCH menu:* COUNT 1000
SCAN 0 MATCH dict:* COUNT 1000
SCAN 0 MATCH post:* COUNT 1000
```

**步骤 3**: 查看具体键名（抽样检查）
```bash
# 查看用户模块旧键
KEYS user:id:*

# 查看角色模块旧键
KEYS role:menus:*
```

**步骤 4**: 删除旧键（确认后执行）
```bash
# 删除用户模块旧键
SCAN 0 MATCH user:* COUNT 1000 | xargs -I {} DEL {}

# 删除角色模块旧键
SCAN 0 MATCH role:* COUNT 1000 | xargs -I {} DEL {}

# 删除部门模块旧键
SCAN 0 MATCH dept:* COUNT 1000 | xargs -I {} DEL {}

# 删除菜单模块旧键
SCAN 0 MATCH menu:* COUNT 1000 | xargs -I {} DEL {}

# 删除字典模块旧键
SCAN 0 MATCH dict:* COUNT 1000 | xargs -I {} DEL {}

# 删除岗位模块旧键
SCAN 0 MATCH post:* COUNT 1000 | xargs -I {} DEL {}
```

**注意**: 以上命令在 Linux/Mac 上有效。Windows 用户请使用方法 2。

### 方法 2: 使用 Python 脚本批量清理

**步骤 1**: 安装依赖
```bash
pip install redis
```

**步骤 2**: 创建清理脚本 `cleanup_old_cache_keys.py`

```python
import redis

# 连接 Redis
r = redis.Redis(host='localhost', port=6379, decode_responses=True)

# 定义旧键模式
old_patterns = [
    'user:*',
    'role:*',
    'dept:*',
    'menu:*',
    'dict:*',
    'post:*',
]

# 统计和删除旧键
for pattern in old_patterns:
    keys = []
    for key in r.scan_iter(match=pattern, count=1000):
        keys.append(key)
    
    if keys:
        print(f"找到 {len(keys)} 个匹配 '{pattern}' 的键")
        print(f"示例键: {keys[:5]}")  # 显示前5个键
        
        # 确认删除
        response = input(f"是否删除这 {len(keys)} 个键? (y/n): ")
        if response.lower() == 'y':
            deleted = r.delete(*keys)
            print(f"已删除 {deleted} 个键")
        else:
            print("跳过删除")
    else:
        print(f"未找到匹配 '{pattern}' 的键")

print("清理完成!")
```

**步骤 3**: 运行脚本
```bash
python cleanup_old_cache_keys.py
```

### 方法 3: 使用 Makefile（推荐）

本工具集提供了 Makefile 简化操作：

```bash
# 进入工具目录
cd scripts/migrate_cache_keys

# 1. 验证迁移结果（检查当前状态）
make verify

# 2. 预览将要删除的旧键（DRY RUN 模式）
make cleanup-dry

# 3. 确认后执行删除
make cleanup-force

# 4. 再次验证清理结果
make verify
```

### 方法 4: 直接运行 Go 工具

**验证工具**:
```bash
cd scripts/migrate_cache_keys/verify
go run verify_migration.go -host localhost -port 6379
```

**清理工具**:
```bash
cd scripts/migrate_cache_keys/cleanup
make run-dry    # 预览模式
make run-force  # 强制删除
```

## 验证迁移结果

### 验证新键已创建

**方法 1**: 在 redis-cli 中查看
```bash
# 查看新规范键
KEYS cache:user:*
KEYS cache:role:*
KEYS cache:dept:*
```

**方法 2**: 使用缓存监控界面
访问系统管理 -> 缓存监控，查看缓存列表，确认键名格式为 `cache:*`

### 验证功能正常

1. **登录功能**: 用户登录应正常工作，缓存用户信息
2. **权限加载**: 角色菜单权限应正确加载
3. **字典数据**: 系统字典数据应正常缓存
4. **部门树**: 部门树结构应正常显示

## 回滚方案

如果迁移后出现问题，执行以下回滚步骤：

1. **恢复旧键常量**（暂不执行，代码已更新）
   ```go
   // 将 data_cache_service.go 中的键常量恢复为旧格式
   CacheKeyUserByID = "user:id"  // 移除 cache: 前缀
   ```

2. **重启应用服务**
   ```bash
   # 停止应用
   pkill xingran-backend

   # 重新编译
   go build -o xingran-backend ./cmd/main.go

   # 启动应用
   ./xingran-backend
   ```

## 常见问题

### Q1: 迁移后缓存不生效怎么办？
**A**: 检查以下几点：
1. 确认 Redis 连接正常
2. 检查键名是否为新格式（`cache:*`）
3. 查看应用日志中的错误信息
4. 重启应用服务

### Q2: 是否需要停机维护？
**A**: 不需要停机。本次迁移是向后兼容的：
- 新键格式已生效
- 旧键会在过期后自动失效
- 清理旧键可以在运行时进行

### Q3: 迁移后性能会受影响吗？
**A**: 不会。键名格式变更不影响性能：
- Redis 键查找性能相同
- 缓存命中率不变
- 新键会自动创建，无需手动迁移数据

### Q4: 如何确认所有旧键已清理？
**A**: 使用以下命令检查：
```bash
# 在 redis-cli 中执行
SCAN 0 MATCH user:* COUNT 1000
SCAN 0 MATCH role:* COUNT 1000
# ... 其他模块
```

如果返回空列表，说明旧键已全部清理。

## 技术细节

### 键前缀处理

在 `monitor/cache_service.go` 中，有键前缀标准化函数：

```go
func normalizeCacheKeyForService(key string) string {
    if len(key) > 6 && key[:6] == "xingran:" {
        return key[6:]
    }
    return key
}
```

这个函数确保：
- Redis 实际存储的键: `xingran:cache:user:id:uuid`
- 业务层使用的键: `cache:user:id:uuid`
- 显示层显示的键: `cache:user:id:uuid`

### 缓存失效模式

使用新规范后，缓存失效模式也相应更新：

```go
// 旧规范（已弃用）
InvalidateCacheByPattern(ctx, cache, []string{"user:id:*"}, "USER")

// 新规范（当前使用）
InvalidateCacheByPattern(ctx, cache, []string{"cache:user:id:*"}, "USER")
```

## 相关文件

- `internal/services/data_cache_service.go` - 旧规范键定义（已更新）
- `internal/services/system/cache_keys.go` - 新规范键定义
- `internal/services/system/*_cache_impl.go` - 使用新规范的服务实现
- `internal/services/monitor/cache_service.go` - 缓存监控服务

## 联系方式

如有问题，请联系开发团队或提交 Issue。
