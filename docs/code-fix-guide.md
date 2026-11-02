<!-- generated-by: gsd-doc-writer -->
# 代码修复详细指南

**文档版本**: 2.6
**基于审查报告**: `docs/code-review-report.md`
**生成日期**: 2026-01-31
**最后更新**: 2026-02-01

**重要变更**:
- ✅ 重构前端 API 错误处理（创建统一错误处理器）
- ✅ 添加前端测试框架（vitest + 测试用例）
- ✅ 添加后端测试（密码管理器测试）
- ✅ 创建共享 Handler 辅助函数
- ✅ 提取常量到 internal/constants（时间、缓存、分页、状态）
- ✅ 正确传递 Context（monitor 模块修复）
- ✅ N+1 查询审查（项目已有良好实践，文档化现有实践）
- ✅ 统一命名规范审查（项目已有统一规范，文档化规范）
- ✅ workorder/duty/network 模块 Handler 错误处理迁移完成（共 13 个文件）
- ✅ Context 传递优化（所有 Handler 已正确使用 c.Request.Context()）
- ✅ SQL 注入防护验证（白名单验证已实现）
- ✅ monitor 模块 Handler interface 模式迁移完成（4 个 handler）
- 📋 规划：其他模块 Handler 迁移、interface{} 优化

本文档提供了代码审查报告中所有问题的详细修复步骤、代码示例和验证方法。

> **⚠️ 注意**:
> 1. LDAP/AD 连接证书验证问题已确认忽略。该功能在内部网络环境中使用，LDAP 服务器使用自签名证书，当前配置已通过安全评估。
> 2. `BaiduMapCache` 已重命名为 `DualLevelCache`，使其成为通用缓存组件。
> 3. `cacheStats.ts` 已删除，统计功能已集成到 `DualLevelCache` 中。
> 4. **前端类型错误已全部修复**，`npm run build` 构建成功。
> 5. **操作日志中间件已实现**，Handler 只需调用 `SetOperLogInfo` 即可自动记录。
> 6. **登出功能已实现**，使用令牌黑名单机制使 JWT 失效。
> 7. **用户解锁功能已实现**，管理员可手动解锁被锁定的用户账号。
> 8. **缓存键处理重复已消除**，添加 `normalizeCacheKeyForService()` 辅助函数。

---

## ✅ 已完成修复清单

### 后端修复
- ✅ SQL 注入风险防护验证 - `workstation_service.go:107-110` (已实现白名单验证)
- ✅ 数据库查询错误处理 - `pkg/permission/service.go`
- ✅ JWT 初始化 panic 修复 - `jwt.go`, `core.go`, `main.go`
- ✅ 操作日志中间件实现 - `pkg/middleware/oper_log.go`
  - ✅ 完善 OperLogMiddleware 实现
  - ✅ 添加运维管理模块路径配置
  - ✅ 注册到各模块路由组
- ✅ 登出功能实现（令牌黑名单） - `internal/services/token_blacklist_service.go`
  - ✅ TokenBlacklistService 服务
  - ✅ JWTAuthWithBlacklist 中件
  - ✅ 登出 handler 实现
- ✅ 用户解锁功能实现 - `internal/api/v1/system/user_unlock_handler.go`
  - ✅ 清除登录失败记录
  - ✅ 清除锁定状态
- ✅ 缓存键处理重复消除 - `internal/services/monitor/cache_service.go`
  - ✅ 添加 `normalizeCacheKeyForService()` 辅助函数
  - ✅ 替换 6 处重复代码
- ✅ operations 模块 Handler 错误处理迁移
  - ✅ 迁移 10 个 handler 使用辅助函数
- ✅ workorder 模块 Handler 错误处理迁移
  - ✅ 迁移 1 个 handler 使用辅助函数
- ✅ duty 模块 Handler 错误处理迁移
  - ✅ 迁移 1 个 handler 使用辅助函数
- ✅ network 模块 Handler 错误处理迁移（完整）
  - ✅ 迁移 9 个 handler 使用辅助函数 (device, credential, template, command, backup, discovery, execution, port, mac)
- ✅ 创建共享 Handler 辅助函数 - `pkg/response/handler_helpers.go`
- ✅ 后端测试框架 - `internal/core/security/password_test.go`
- ✅ 提取常量 - `internal/constants/`
  - ✅ time.go - 时间相关常量
  - ✅ cache.go - 缓存键格式常量
  - ✅ pagination.go - 分页大小常量
  - ✅ status.go - 状态值常量
  - ✅ example_test.go - 使用示例
- ✅ 正确传递 Context - `internal/api/v1/monitor/`
  - ✅ 修改 `cache_service.go` 辅助函数签名，添加 context 参数
  - ✅ 修复 cache_handler.go 中 13 处 context.Background() 使用

### 前端修复
- ✅ BaiduMapCache 重命名为 DualLevelCache
- ✅ cacheStats.ts 删除
- ✅ 类型错误修复（已完成）
  - ✅ PageParams 重复导出问题修复
  - ✅ Workstation → WorkstationOps 类型统一
  - ✅ CAD 组件导出方式修复（命名导出）
  - ✅ Building 类型添加 longitude/latitude 属性
  - ✅ DRAG_THRESHOLD 常量添加
  - ✅ FloorData 类型添加 code 属性
  - ✅ 服务文件 PageParams 导入路径修复
- ✅ console.log 清理
  - ✅ 使用 vite-plugin-remove-console 自动清理
- ✅ 重构 API 错误处理
  - ✅ 扩展 `errorHandler.ts` 添加 HTTP 响应错误处理
  - ✅ 创建 `HttpErrorType` 枚举
  - ✅ 集成到 `api.ts` 响应拦截器
- ✅ 添加前端测试框架
  - ✅ 安装 vitest、@testing-library/react
  - ✅ 配置 vitest.config.ts
  - ✅ 创建测试用例（13 个测试全部通过）

---

## 目录

- [第一阶段：紧急安全修复](#第一阶段紧急安全修复)
- [第二阶段：功能完善](#第二阶段功能完善)
- [第三阶段：代码质量提升](#第三阶段代码质量提升)
- [第四阶段：持续改进](#第四阶段持续改进)
- [第五阶段：类型错误修复](#第五阶段类型错误修复)

---

## 第一阶段：紧急安全修复

### 修复 1: SQL 注入风险 ✅ **已验证防护已实现**

**文件**: `internal/services/operations/workstation_service.go:89`
**严重程度**: 🔴 Critical
**状态**: ✅ **已验证 - 白名单验证已实现**

#### 问题描述
工位查询中 `floorTable` 变量直接拼接 SQL 语句，缺少输入验证，存在 SQL 注入风险。

#### 已实现的防护
代码中已经实现了白名单验证函数（lines 19-28）：

```go
// validateTableName 验证表名是否在白名单中
func validateTableName(tableName string) bool {
    allowedTables := map[string]bool{
        "ops_floors":      true,
        "sys_floors":      true,
    }
    return allowedTables[tableName]
}
```

查询逻辑中也已包含验证调用（lines 107-110）：

```go
if floorCode := extractStringParam(params, "floorCode"); floorCode != "" {
    // 验证表名
    if !validateTableName(floorTable) {
        return nil, fmt.Errorf("invalid table name: %s", floorTable)
    }
    query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE "+floorTable+".id = sys_workstation.floor_id::uuid AND "+floorTable+".floor_no = ?)", floorCode)
}
```

---

### 修复 2: LDAP/AD 连接证书验证 ⚠️ **已忽略**

**文件**:
- `internal/services/ad_ldap_client.go:32`
- `internal/services/addomain/ldap_client.go:58`

**严重程度**: 🔴 Critical

**状态**: 🔕 **已标记忽略 - 用户确认**

**原因**: AD 域管理功能在内部网络环境中使用，LDAP 服务器使用自签名证书。当前配置已通过安全评估，符合内部使用场景。

#### 当前代码
```go
&ldap.DialTLSConfig{
    InsecureSkipVerify: true, // ❌ 生产环境应验证证书
}
```

#### 修复步骤（已忽略，仅供参考）

1. **添加配置项**

在 `configs/config.yaml` 中添加：

```yaml
ldap:
  skip_verify: false  # 默认启用证书验证
  ca_cert_path: /etc/ssl/certs/ca.pem
```

2. **修改客户端代码**

```go
skipVerify := config.GetBool("ldap.skip_verify", false)

config := &ldap.DialTLSConfig{
    InsecureSkipVerify: skipVerify,
}

// 如果提供了 CA 证书
if caCertPath := config.GetString("ldap.ca_cert_path"); caCertPath != "" {
    caCert, err := os.ReadFile(caCertPath)
    if err != nil {
        return nil, fmt.Errorf("读取CA证书失败: %w", err)
    }
    config.RootCAs = x509.NewCertPool()
    config.RootCAs.AppendCertsFromPEM(caCert)
}
```

---

### 修复 3: 前端 console.log 清理 ✅ **已完成**

**影响**: 92 个文件，226+ 处 console

#### 修复方案（已实施）

使用 `vite-plugin-remove-console` 插件，在生产构建时自动移除所有 console 语句：

**已修改文件**: `xingran-react-frontend/vite.config.ts`

```typescript
import removeConsole from 'vite-plugin-remove-console'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    removeConsole(),  // 生产构建时移除 console 语句
  ],
  // ...
})
```

#### 优点
- 无需手动修改 226+ 处 console 调用
- 不影响开发时调试
- 生产构建自动清理

---

### 修复 4: Go interface{} 类型断言 ✅ **已完成 system 模块迁移**

**影响**: Service 层接口（44+ 文件使用 `map[string]interface{}` 参数）

**当前模式**:
```go
// Service 接口使用通用参数
func (s *BuildingService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error)

// Handler 需要手动类型断言
if val, ok := params["name"].(string); ok {
    // ...
}
```

**建议模式**:
```go
// 定义专门的请求结构体
type BuildingListRequest struct {
    PaginationParams  // 嵌入分页参数
    StatusRequest     // 嵌入状态筛选
    Name       string `json:"name"`
    OrgID      string `json:"orgId"`
}

// Service 接口使用类型安全参数
func (s *BuildingService) List(ctx context.Context, req BuildingListRequest) (*PageResult, error)
```

**评估**:
- **优点**: 类型安全、IDE 自动补全、减少运行时错误
- **缺点**: 大规模重构、影响范围广、需要前后端协调
- **建议**: 作为长期改进项，新代码优先使用类型安全的请求结构体

**已完成工作**:
- ✅ 创建通用请求结构体 (`internal/api/v1/operations/requests/common.go`)
- ✅ **operations 模块完整迁移**（10 个模块）
- ✅ **system 模块完整迁移**（8 个模块：User, Role, Menu, Department, Post, Dict, Config, Notice）
- ✅ 创建类型安全的 Service 示例 (`building_service_typesafe.go`)
- ✅ 创建类型安全的 Handler 示例 (`building_handler_typesafe.go`)
- ✅ 创建迁移指南文档 (`docs/interface-type-safe-migration-guide.md`)
- ✅ 添加缺失的分页常量 (`internal/constants/pagination.go`)

**operations 模块迁移清单** (已完成):
| 模块 | Service | Handler | 请求结构体 | 状态 |
|------|---------|---------|-----------|------|
| Building | ✅ | ✅ | building_requests.go | ✅ |
| Floor | ✅ | ✅ | floor_requests.go | ✅ |
| Workstation | ✅ | ✅ | workstation_requests.go | ✅ |
| ServerRoom | ✅ | ✅ | server_room_requests.go | ✅ |
| RoomDevice | ✅ | ✅ | room_device_requests.go | ✅ |
| InfoPoint | ✅ | ✅ | infopoint_requests.go | ✅ |
| DedicatedLine | ✅ | ✅ | dedicated_line_requests.go | ✅ |
| FloorPlanText | ✅ | ✅ | floor_plan_text_requests.go | ✅ |
| Door | ✅ | ✅ | door_requests.go | ✅ |
| Wall | ✅ | ✅ | wall_requests.go | ✅ |

**system 模块迁移清单** (已完成):
| 模块 | Service | Handler | 请求结构体 | 状态 |
|------|---------|---------|-----------|------|
| User | ✅ | ✅ | user_requests.go | ✅ |
| Role | ✅ | ✅ | role_requests.go | ✅ |
| Menu | ✅ | ✅ | menu_requests.go | ✅ |
| Department | ✅ | ✅ | department_requests.go | ✅ |
| Post | ✅ | ✅ | post_requests.go | ✅ |
| Dict | ✅ | ✅ | dict_requests.go (DictType + DictData) | ✅ |
| Config | ✅ | ✅ | config_requests.go | ✅ |
| Notice | ✅ | ✅ | notice_requests.go | ✅ |

**已完成迁移模块**:
- ✅ operations 模块（已完成 10 个子模块）
- ✅ system 模块（已完成 8 个核心模块）

**文件位置**:
- `internal/api/v1/operations/requests/` - 请求结构体目录
- `internal/services/operations/*_service.go` - 已迁移为类型安全版本
- `internal/api/v1/operations/*_handler.go` - 已迁移为类型安全版本
- `docs/interface-type-safe-migration-guide.md` - 详细迁移指南

---

### 修复 5-7: 其他紧急修复 ✅ **已完成**

- ✅ 数据库查询错误处理 - `pkg/permission/service.go`
- ✅ JWT 初始化 panic 修复 - `jwt.go`, `core.go`, `main.go`
- ✅ cache_handler.go 拆分 - `cache_service.go`
- ✅ CADFloorPlanEditor 拆分（前端组件重构）

---

## 第二阶段：功能完善

### 修复 9: 实现登出功能（令牌黑名单）✅ **已完成**

#### 修复内容

**发现**: 登出 handler 在 `auth.go` 中原本只有 TODO，未实现令牌失效机制。

**修复内容**:

1. **创建 TokenBlacklistService** (`internal/services/token_blacklist_service.go`)
   - `AddToBlacklist()` - 将令牌加入黑名单
   - `IsBlacklisted()` - 检查令牌是否在黑名单中
   - 黑名单键格式: `token:blacklist:{token}`
   - TTL 与令牌过期时间一致

2. **创建 JWTAuthWithBlacklist 中间件** (`pkg/middleware/auth.go`)
   - 在验证令牌前检查黑名单
   - 将 token 和 claims 存储到上下文供登出使用

3. **完善登出 Handler** (`internal/api/v1/auth.go`)
   - 从上下文获取 token 和 claims
   - 调用 `TokenBlacklistService.AddToBlacklist` 将令牌加入黑名单
   - TTL 设置为令牌的过期时间

4. **更新路由配置** (`internal/api/router.go`)
   - 系统管理模块使用 `JWTAuthWithBlacklist`

5. **初始化服务** (`internal/core/core.go`)
   - 添加 `TokenBlacklistService` 字段
   - 在 New 函数中初始化服务

---

### 修复 10: 实现操作日志记录 ✅ **已完成**

#### 修复总结

**发现**: 项目已有操作日志的基础实现（数据模型、服务层、辅助函数），但中间件未完成。

**修复内容**:

1. **完善中间件实现** (`pkg/middleware/oper_log.go`)
   - 修改 `OperLogMiddleware` 签名，接收具体类型而非 `interface{}`
   - 实现 TODO 部分的日志记录逻辑
   - 添加 `getMethodDescription` 辅助函数

2. **更新路径配置**
   - 添加运维管理模块路径到 `LogPaths`: `/ops/building`, `/ops/floor`, `/ops/workstation` 等
   - 添加排除路径: `/tree` (树形结构查询)

3. **注册中间件到路由** (`internal/api/router.go`)
   - `/system` 路由组
   - `/ops` 路由组
   - `/network` 路由组
   - `/workorder` 路由组
   - `/knowledge` 路由组
   - `/duty` 路由组
   - `/ad-domain` 路由组

#### 使用方式

Handler 中只需设置操作日志信息，中间件会自动记录：

```go
import "github.com/xingran-next/xingran-go-backend/pkg/middleware"

func (h *BuildingHandler) Create(c *gin.Context) {
    // ... 业务逻辑 ...

    // 设置操作日志信息
    middleware.SetOperLogInfo(c, "楼宇管理", 1, "新增")
    // 中间件会自动记录，无需手动调用 recordOperLog
}
```

**优势**:
- Handler 代码简洁，不需要手动调用记录函数
- 自动记录耗时、状态、用户信息等
- 统一在中间件处理，易于维护

---

### 修复 11: 实现用户解锁功能 ✅ **已完成**

#### 修复内容

**发现**: 登录失败锁定机制已存在（在 `CaptchaService` 中），但缺少管理员手动解锁的 API。

**修复内容**:

1. **创建用户解锁 Handler** (`internal/api/v1/system/user_unlock_handler.go`)
   - API: `POST /system/user/unlock`
   - 清除登录失败记录 (`ClearLoginFailure`)
   - 清除锁定状态（删除 Redis 中的 `login:lock:{username}`）

2. **注册解锁路由** (`internal/api/router.go`)
   - 需要系统配置权限 (`permSystemConfig`)

#### 使用方式

```bash
# 解锁用户账号
curl -X POST http://localhost:8080/api/v1/system/user/unlock \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"username": "locked_user"}'
```

**响应**:
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "message": "用户账号已解锁"
    }
}
```

---

### 修复 12: 移除 cacheStats.ts 并重命名 BaiduMapCache 为通用缓存类 ✅ **已完成**

#### 修复步骤

**问题分析**:
- `cacheStats.ts` 已标记 @deprecated，但仍被 `opsApi.ts` 引用
- `BaiduMapCache` 名称暗示仅用于百度地图，实际上是通用的 Memory + localStorage 双层缓存
- 功能重复：BaiduMapCache 已内置统计功能，无需额外的 CacheStats

**第一步：重命名文件和类**

```bash
# 重命名文件
mv xingran-react-frontend/src/utils/baiduMapCache.ts \
   xingran-react-frontend/src/utils/dualLevelCache.ts
```

修改 `xingran-react-frontend/src/utils/dualLevelCache.ts`:

```typescript
/**
 * 双层缓存管理器
 * 实现 Memory + localStorage 双层缓存
 * 提供缓存统计和管理功能
 */

// 修改日志前缀
const LOG_PREFIX = '[DualLevelCache]';
const DEFAULT_STORAGE_PREFIX = 'dual_level_cache_';

// 修改类名
export class DualLevelCache<T> {  // 原 BaiduMapCache
  private memoryCache: MemoryCache<T>;
  private storageCache: StorageCache<T>;
  // ... 其余实现保持不变
}

// 修改导出函数
export const getDualLevelCache = <T>(): DualLevelCache<T> => {  // 原 getBaiduMapCache
  if (!cacheInstance) {
    cacheInstance = new DualLevelCache<T>();
  }
  return cacheInstance as DualLevelCache<T>;
};

export const clearDualLevelCache = (): void => {  // 原 clearBaiduMapCache
  if (cacheInstance) {
    cacheInstance.destroy();
    cacheInstance = null;
  }
};
```

**第二步：更新 opsApi.ts 引用**

修改 `xingran-react-frontend/src/lib/opsApi.ts`:

```typescript
// ❌ 删除
// import { cacheStats } from '@/utils/cacheStats';

// ✅ 修改导入
import { getDualLevelCache } from '@/utils/dualLevelCache';

// ✅ 更新地理编码函数（移除冗余的 cacheStats 调用）
export const getGeocode = async (address: string): Promise<GeocodeResult | null> => {
  const cache = getDualLevelCache<GeocodeResult>();
  const cacheKey = cache.generateKey({ address });

  // 尝试从缓存获取（BaiduMapCache 内部已自动统计）
  const cached = cache.get(cacheKey);
  if (cached) {
    // ❌ 删除: cacheStats.recordStorageHit();
    console.log('✅ 地理编码缓存命中:', address);
    return cached;
  }

  // 调用后端API
  // ❌ 删除: cacheStats.recordAPICall();
  console.log('🌐 调用后端地理编码API:', address);

  try {
    const response = await post<BaseResponse<GeocodeResult>>('/ops/building/geocode', { address });
    if (response.data?.lng && response.data?.lat) {
      // 设置缓存
      cache.set(cacheKey, response.data);
      return response.data;
    }
    return null;
  } catch (error) {
    console.error('❌ 地理编码失败:', error);
    return null;
  }
};

// ✅ 更新统计函数（仅使用 DualLevelCache 的统计）
export const getGeocodeStats = () => {
  const cache = getDualLevelCache<unknown>();
  const cacheInfo = cache.getStats();

  return {
    // 直接使用 DualLevelCache 的统计
    ...cacheInfo,
    // 兼容旧字段名（可选）
    apiCalls: cacheInfo.misses,  // misses = API 调用次数
  };
};

// ✅ 简化重置函数
export const resetGeocodeStats = () => {
  const cache = getDualLevelCache<unknown>();
  cache.resetStats();
  console.log('🔄 已重置地理编码统计');
};
```

**第三步：搜索并更新其他引用**

```bash
# 搜索所有 BaiduMapCache 引用
cd xingran-react-frontend
grep -r "BaiduMapCache\|baiduMapCache\|getBaiduMapCache" src/

# 搜索所有 cacheStats 引用
grep -r "cacheStats" src/
```

**第四步：删除旧文件**

```bash
# 删除已弃用的 cacheStats.ts
rm xingran-react-frontend/src/utils/cacheStats.ts
```

**第五步：验证**

```typescript
// 测试新的缓存类
import { getDualLevelCache } from '@/utils/dualLevelCache';

const cache = getDualLevelCache<MyDataType>();
cache.set('test', { data: 'value' });
const value = cache.get('test');
console.log('Stats:', cache.getStats());
```

#### 影响范围

- **文件修改**: 3 个文件
  - `src/utils/baiduMapCache.ts` → `src/utils/dualLevelCache.ts` (重命名)
  - `src/lib/opsApi.ts` (更新导入，删除 cacheStats 调用)
  - `src/utils/cacheStats.ts` (删除)

- **类名变更**:
  - `BaiduMapCache<T>` → `DualLevelCache<T>`
  - `getBaiduMapCache()` → `getDualLevelCache()`
  - `clearBaiduMapCache()` → `clearDualLevelCache()`

#### 优势

1. **语义清晰**: `DualLevelCache` 准确描述了 Memory + localStorage 双层架构
2. **通用可重用**: 可用于任何需要双层缓存的场景，不仅限于百度地图
3. **代码简化**: 移除冗余的 `cacheStats`，统计逻辑统一在 `DualLevelCache` 内部
4. **维护性提升**: 单一统计来源，避免不同步问题



---

## 第三阶段：代码质量提升

### 修复 13: 消除缓存键处理重复 ✅ **已完成**

#### 修复内容

**问题**: 缓存键前缀处理逻辑在多处重复（6处）

**修复**:
- 在 `cache_service.go` 添加 `normalizeCacheKeyForService()` 辅助函数
- 替换 `cache_handler.go` 中的所有重复代码
- 移除未使用的 `strings` 导入

#### 修改文件
- `internal/services/monitor/cache_service.go` - 添加辅助函数
- `internal/api/v1/monitor/cache_handler.go` - 使用辅助函数

---

### 修复 14: 消除 Handler 错误处理重复 ✅ **operations 模块已完成**

#### 现状

**已有辅助函数** (`internal/api/v1/operations/base_handler.go`):
- `handleJSONBinding()` - 统一处理 JSON 绑定
- `handleServiceError()` - 统一处理服务层错误

**已迁移的 Handler** (operations 模块):
- ✅ building_handler.go
- ✅ floor_handler.go
- ✅ workstation_handler.go
- ✅ server_room_handler.go
- ✅ room_device_handler.go
- ✅ infopoint_handler.go
- ✅ dedicated_line_handler.go
- ✅ floor_plan_text_handler.go
- ✅ door_handler.go (已部分迁移，已完善)
- ✅ wall_handler.go (已部分迁移，已完善)

**未迁移的 Handler** (其他模块):
- system 模块 (department_handler.go, post_handler.go, user_handler.go 等)
- workorder 模块 (workorder_handler.go)
- duty 模块 (duty_handler.go)
- network 模块 (device_handler.go, port_handler.go 等)

**说明**: 由于辅助函数位于 operations 包中，其他模块需要:
1. 将辅助函数移至共享位置 (pkg/response 或 pkg/middleware)
2. 或在各模块中复制辅助函数
3. 建议作为独立任务处理其他模块的迁移

---

### 修复 15: 重构前端 API 错误处理 ✅ **已完成**

#### 修复内容

扩展 `src/utils/errorHandler.ts`，添加 HTTP 响应级别的错误处理：

1. **添加错误类型枚举** `HttpErrorType`
   - 覆盖 400-503 常见 HTTP 状态码

2. **添加响应错误处理函数**
   - `handleHttpResponseError()` - 统一处理 HTTP 响应错误
   - `handleNetworkError()` - 处理网络超时和连接错误
   - `handleParseError()` - 处理响应解析错误

3. **集成到 api.ts**
   - 导入新的错误处理函数
   - 替换响应拦截器中的错误处理逻辑

#### 优势
- 错误处理逻辑集中管理
- 自动处理 401 登出
- SM2 解密失败自动清除公钥缓存
- 统一的错误消息提示

---

### 修复 16: 添加前端测试 ✅ **已完成**

#### 修复内容

1. **安装测试依赖**
   - vitest (测试框架)
   - @testing-library/react (React 测试)
   - @testing-library/jest-dom (DOM 断言)
   - @vitest/ui (测试 UI)
   - jsdom (DOM 环境)

2. **创建配置文件**
   - `vitest.config.ts` - Vitest 配置
   - `src/test/setup.ts` - 测试设置

3. **添加测试脚本**
   - `npm test` - 运行测试
   - `npm run test:ui` - UI 模式
   - `npm run test:coverage` - 覆盖率报告

4. **创建测试用例**
   - `src/utils/sm4.test.ts` - SM4 加密测试 (12 个测试)
   - `src/utils/errorHandler.test.ts` - 错误处理测试 (1 个测试)

#### 测试结果
```
Test Files: 2 passed
Tests: 13 passed
```

#### 优点
- 测试框架配置完成
- 为后续测试提供了基础
- 覆盖关键工具函数

---

### 修复 17: 添加后端测试 ✅ **已完成**

#### 修复内容

1. **创建密码管理器测试** `internal/core/security/password_test.go`
   - `TestPasswordManager_HashPassword` - 测试密码哈希
   - `TestPasswordManager_HashAndVerify` - 测试哈希和验证
   - `TestPasswordManager_VerifyInvalidFormat` - 测试无效格式处理
   - `TestPasswordManager_DifferentHashesForSamePassword` - 测试盐值随机性
   - `TestPasswordManager_CustomConfig` - 测试自定义配置
   - `TestPasswordManager_EmptyPassword` - 测试空密码处理

2. **测试结果**
```
PASS: TestPasswordManager_HashPassword
PASS: TestPasswordManager_HashAndVerify
PASS: TestPasswordManager_VerifyInvalidFormat
PASS: TestPasswordManager_DifferentHashesForSamePassword
PASS: TestPasswordManager_CustomConfig
PASS: TestPasswordManager_EmptyPassword
ok      github.com/xingran-next/xingran-go-backend/internal/core/security
```

3. **测试命令**
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/core/security/...

# 详细输出
go test -v ./internal/core/security/...
```

---

## 第四阶段：持续改进

### 修复 18: 其他模块 Handler 错误处理迁移 ✅ **workorder/duty/network/monitor 模块已完成**

#### 当前进度

**已完成**:
- ✅ operations 模块完全迁移（10 个 handler）
- ✅ system 模块完全迁移（8 个核心模块）
- ✅ 创建共享辅助函数 `pkg/response/handler_helpers.go`
- ✅ workorder 模块迁移（1 个 handler）
- ✅ duty 模块迁移（1 个 handler）
- ✅ network 模块迁移（9 个 handler）
- ✅ monitor 模块 interface 模式迁移（4 个 handler）

**待迁移模块**:
- system 模块剩余 handler（非核心，约 9 个）
- scheduler 模块（约 3 个）
- knowledge 模块（约 2-3 个）

#### 共享辅助函数

已创建 `pkg/response/handler_helpers.go`:
```go
func HandleJSONBinding(c *gin.Context, obj interface{}) bool
func HandleServiceError(c *gin.Context, err error, operation string) bool
func HandleIDParam(c *gin.Context) (string, bool)
func HandleGetByID(c *gin.Context, getter func(string) (interface{}, error), notFoundMessage string) bool
```

#### 实施建议

1. **短期**: 保持现状，operations 使用本地辅助函数，新代码使用共享函数
2. **中期**: 按需逐步迁移其他模块
3. **长期**: 统一所有模块使用共享辅助函数

#### 优先级

1. **高优先级**: system 模块（核心功能）
2. **中优先级**: workorder、duty、network 模块
3. **低优先级**: scheduler、knowledge、monitor 模块

**注意**: 这是一个大规模重构任务，建议分阶段逐步实施，避免影响现有稳定功能。

**system 模块** (17 个文件):
- department_handler.go - 部门管理
- post_handler.go - 岗位管理
- user_handler.go - 用户管理
- role_handler.go - 角色管理
- menu_handler.go - 菜单管理
- dict_handler.go - 字典管理
- config_handler.go - 配置管理
- notice_handler.go - 通知管理
- settings_handler.go - 设置管理
- file_handler.go - 文件管理
- dashboard_handler.go - 仪表板
- profile_handler.go - 个人资料
- notification_config_handler.go - 通知配置
- fix_menu_handler.go - 菜单修复
- ad_domain_handler.go - AD 域管理
- user_unlock_handler.go - 用户解锁
- notice_user_handler.go - 通知用户

**workorder 模块** (1 个文件):
- workorder_handler.go - 工单管理

**duty 模块** (1 个文件):
- duty_handler.go - 值班管理

**network 模块** (10 个文件) - ✅ 已完成迁移:
- device_handler.go - 设备管理 ✅
- port_handler.go - 端口管理 ✅
- backup_handler.go - 备份管理 ✅
- command_handler.go - 命令管理 ✅
- credential_handler.go - 凭证管理 ✅
- discovery_handler.go - 发现管理 ✅
- execution_handler.go - 执行管理 ✅
- mac_handler.go - MAC 地址管理 ✅
- template_handler.go - 模板管理 ✅

**scheduler 模块** (约 3 个文件):
- job_handler.go - 定时任务管理
- job_log_handler.go - 任务日志管理

**knowledge 模块** (约 2-3 个文件):
- article_handler.go - 文章管理
- category_handler.go - 分类管理

**monitor 模块** (约 3 个文件) - ✅ **已完成 Handler interface 模式迁移**:
- server_handler.go - 服务器监控 ✅
- oper_log_handler.go - 操作日志 ✅
- login_log_handler.go - 登录日志 ✅
- cache_handler.go - 缓存监控 ✅

#### 修复方案

**方案一：创建共享辅助函数** (推荐)

1. 在 `pkg/response` 或 `pkg/middleware` 创建共享辅助函数:

```go
// pkg/response/handler_helpers.go
package response

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// HandleJSONBinding 统一处理 JSON 绑定
func HandleJSONBinding(c *gin.Context, obj interface{}) bool {
    if err := c.ShouldBindJSON(obj); err != nil {
        Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
        return false
    }
    return true
}

// HandleServiceError 统一处理服务层错误
func HandleServiceError(c *gin.Context, err error, operation string) bool {
    if err != nil {
        Error(c, http.StatusInternalServerError, operation+"失败: "+err.Error())
        return false
    }
    return true
}
```

2. 更新各模块 handler 使用共享辅助函数

**方案二：创建模块特定的 base_handler**

在每个模块创建自己的 base_handler.go，包含该模块的辅助函数。

#### 迁移步骤（以方案一为例）

1. 创建 `pkg/response/handler_helpers.go`
2. 更新 operations 模块使用新的共享辅助函数（保持向后兼容）
3. 迁移 system 模块 handlers
4. 迁移 workorder、duty、network 模块
5. 迁移 scheduler、knowledge、monitor 模块
6. 删除 `internal/api/v1/operations/base_handler.go`（可选）

#### 优先级

1. **高优先级**: system 模块（核心功能）
2. **中优先级**: workorder、duty、network 模块
3. **低优先级**: scheduler、knowledge、monitor 模块

---

### 修复 20: 统一命名规范 ✅ **已审查 - 项目已有统一命名规范**

#### 审查结果

经过详细代码审查，发现项目已经遵循了良好的命名规范，无需大规模重构。

#### 现有命名规范

**Go 命名规范**（已遵循）:
```go
// ✅ 包名: lowercase
package models
package services
package handler

// ✅ 常量: PascalCase
const (
    UserStatusEnabled  UserStatus = 0
    UserStatusDisabled UserStatus = 1
)

// ✅ 导出变量/函数: PascalCase
type UserService interface {}
func NewUserService() {}
var DefaultCacheExpire = 5 * time.Minute

// ✅ 私有变量/函数: camelCase
type userServiceImpl struct {}
func (s *userServiceImpl) createUser() {}

// ✅ 结构体字段: PascalCase
type User struct {
    ID        string
    UserName  string
    DeptID    *string
    CreatedAt time.Time
}
```

**JSON 标签命名**（已遵循 camelCase）:
```go
type User struct {
    ID        string    `json:"id"`           // ✅ camelCase
    UserName  string    `json:"userName"`     // ✅ camelCase
    DeptID    *string   `json:"deptId"`       // ✅ camelCase
    CreatedAt time.Time `json:"createdAt"`    // ✅ camelCase
}
```

**数据库字段命名**（已遵循 snake_case）:
```go
type User struct {
    ID        string `gorm:"type:uuid;primary_key"`              // id
    CreatedAt time.Time `gorm:"autoCreateTime"`                  // created_at
    DeptID    *string  `gorm:"type:uuid;column:dept_id"`         // dept_id
}
```

**接口命名**（已遵循描述性名称）:
```go
// ✅ 使用描述性名称，不以 I 结尾
type UserService interface {}
type CacheService interface {}
type DeviceManager interface {}

// ✅ 接口方法清晰描述功能
type UserService interface {
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    UpdateUser(ctx context.Context, id string, req *UpdateUserRequest) error
}
```

#### Request/Response 结构体命名示例

```go
// ✅ Request 结构体 - PascalCase + Request 后缀
type CreateUserRequest struct {
    UserName  string `json:"userName" binding:"required"`
    Password  string `json:"password" binding:"required"`
    DeptID    *string `json:"deptId"`
}

type UpdateUserRequest struct {
    UserName string `json:"userName"`
    Status   *int   `json:"status"`
}

// ✅ Response 结构体 - PascalCase + Response/Result 后缀（可选）
type UserResponse struct {
    ID       string
    UserName string
}

// ✅ 分页参数 - 统一命名
type PageParams struct {
    Current  int `json:"current"`
    PageSize int `json:"pageSize"`
}

type PageData struct {
    List     interface{} `json:"list"`
    Total    int64       `json:"total"`
    Current  int         `json:"current"`
    PageSize int         `json:"pageSize"`
}
```

#### 已验证的统一规范

| 类别 | 规范 | 示例 |
|------|------|------|
| 包名 | lowercase | `models`, `services`, `handler` |
| 常量 | PascalCase | `UserStatusEnabled`, `DefaultPageSize` |
| 导出类型/函数 | PascalCase | `UserService`, `CreateUser` |
| 私有类型/函数 | camelCase | `userServiceImpl`, `parseUser` |
| 结构体字段 | PascalCase | `ID`, `UserName`, `DeptID` |
| JSON 标签 | camelCase | `"id"`, `"userName"`, `"deptId"` |
| 数据库字段 | snake_case | `id`, `user_name`, `dept_id` |
| 接口 | 描述性名称 | `UserService` (非 `IUserService`) |
| 请求结构 | XxxxRequest | `CreateUserRequest` |
| 响应结构 | XxxxResponse | `UserResponse` |

#### 命名规范优势

1. **类型安全**: PascalCase 的导出类型避免命名冲突
2. **JSON 兼容**: camelCase 与前端 JavaScript 自然对接
3. **数据库标准**: snake_case 符合 SQL 数据库惯例
4. **可读性**: 描述性接口名清晰表达用途

#### 注意事项

**✅ 遵循 Go 官方规范**:
- 包名应为小写单词，不使用下划线或驼峰
- 导出标识符使用 PascalCase
- 私有标识符使用 camelCase
- 接口名通常是描述性名称，不以 `I` 开头（避免 C# 风格）

**✅ GORM 标签规范**:
- 使用 `column` 指定数据库列名（snake_case）
- 使用 `json` 指定 JSON 字段名（camelCase）
- 主键使用 `primary_key`
- 外键使用 `foreignKey`

---

### 修复 21: 提取常量 ✅ **已完成**

#### 修复内容

创建了 `internal/constants` 目录，统一管理项目中的硬编码常量。

**新增文件**:
1. **`time.go`** - 时间相关常量
   - 缓存过期时间常量（默认 5 分钟、配置缓存 30 分钟等）
   - 网络超时时间常量（HTTP 30秒、AD 同步 30分钟、SNMP 5秒等）
   - 系统时间常量（一天、服务关闭超时、WebSocket 心跳间隔）
   - JWT Token 过期时间常量（访问令牌 2小时、刷新令牌 7天）

2. **`cache.go`** - 缓存相关常量
   - Redis 键前缀 `xingran:`
   - 缓存键格式常量（token 黑名单、登录失败记录、验证码等）

3. **`pagination.go`** - 分页相关常量
   - 默认分页大小 `10`
   - 最大分页大小 `100`
   - LDAP 分页大小常量

4. **`status.go`** - 状态值常量
   - 用户状态（启用/禁用）
   - 角色状态（正常/停用）
   - 菜单状态和可见性
   - 工位状态、通知发布状态
   - 定时任务状态、设备发现状态等
   - HTTP 状态码常量

5. **`example_test.go`** - 使用示例
   - 展示如何在服务中使用这些常量

#### 使用示例

```go
import "github.com/xingran-next/xingran-go-backend/internal/constants"

// 使用时间常量
cacheExpire := constants.DefaultCacheExpire  // 5 * time.Minute
timeout := constants.DefaultHTTPTimeout      // 30 * time.Second

// 使用缓存键格式
blacklistKey := fmt.Sprintf(constants.TokenBlacklistKeyFormat, token)

// 使用分页常量
pageSize := constants.DefaultPageSize  // 10

// 使用状态常量
userStatus := constants.UserStatusEnabled  // 0
```

#### 优势

- **集中管理**: 所有常量统一维护，修改时只需改一处
- **类型安全**: 编译时检查，避免拼写错误
- **语义清晰**: 常量名称清晰表达用途，提高代码可读性
- **易于维护**: 新开发者可以快速找到常量定义

---

### 修复 22: 优化 N+1 查询 ✅ **已审查 - 项目已有良好实践**

#### 审查结果

经过详细代码审查，发现项目已经很好地处理了 N+1 查询问题。大部分查询都使用了 GORM 的 `Preload` 预加载或批量查询（`WHERE IN`）来避免 N+1 问题。

#### 现有良好实践

**1. 使用 Preload 预加载关联数据**

`internal/services/workorder/base.go` - 工单列表查询:
```go
// ✅ 正确 - 使用 Preload 预加载所有关联数据
if err := query.
    Preload("Category").
    Preload("Submitter").
    Preload("Assignee").
    Preload("Dept").
    Order("created_at DESC").
    Limit(pageSize).
    Offset(offset).
    Find(&list).Error; err != nil {
    return nil, 0, fmt.Errorf("查询工单列表失败: %w", err)
}
```

`internal/services/system/user_service.go` - 用户查询:
```go
// ✅ 正确 - 预加载部门信息
if err := query.Preload("Dept").
    Where("deleted_at IS NULL").
    Count(&total).Error; err != nil {
    return nil, err
}
```

`internal/services/duty_pool_service.go` - 值班组查询:
```go
// ✅ 正确 - 预加载成员和部门信息
if err := s.db.WithContext(ctx).
    Preload("Members.User").
    Preload("Dept").
    Where("id = ?", pool.ID).
    First(pool).Error; err != nil {
    return err
}
```

`internal/services/notice_service.go` - 通知查询:
```go
// ✅ 正确 - 预加载通知渠道和目标
if err := query.
    Preload("Channels").
    Order("priority DESC, created_at DESC").
    Offset(offset).
    Limit(pageSize).
    Find(&notices).Error; err != nil {
    return nil, err
}
```

**2. 使用批量查询（WHERE IN）**

`internal/services/network_device_service.go` - 设备关联信息填充:
```go
// ✅ 正确 - 先收集所有需要查询的 ID，然后批量查询
deptMap := make(map[string]string)
if len(deptIDs) > 0 {
    var depts []models.Department
    s.db.WithContext(ctx).Where("id IN ?", deptIDs).Find(&depts)
    for _, dept := range depts {
        deptMap[dept.ID] = dept.DeptName
    }
}

credentialMap := make(map[string]string)
if len(credentialIDs) > 0 {
    var credentials []models.AuthCredential
    s.db.WithContext(ctx).Where("id IN ?", credentialIDs).Find(&credentials)
    for _, credential := range credentials {
        credentialMap[credential.ID] = credential.CredentialName
    }
}

// 在内存中填充关联信息
for i := range *devices {
    if (*devices)[i].DeptID != nil {
        if deptName, ok := deptMap[*(*devices)[i].DeptID]; ok {
            (*devices)[i].DeptName = &deptName
        }
    }
}
```

`internal/services/system/user_service.go` - 用户角色填充:
```go
// ✅ 正确 - 批量查询用户角色和角色信息
userIDs := make([]string, len(list))
for i, u := range list {
    userIDs[i] = u.ID
}

// 批量查询用户角色关系
var userRoles []models.UserRole
if len(userIDs) > 0 {
    s.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&userRoles)
}

// 批量查询角色
var allRoles []models.Role
if len(userRoles) > 0 {
    roleIDs := make([]string, 0)
    for _, ur := range userRoles {
        roleIDs = append(roleIDs, ur.RoleID)
    }
    s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&allRoles)
}
```

**3. 嵌套 Preload**

`internal/services/workorder/base.go` - 工单详情查询:
```go
// ✅ 正确 - 嵌套预加载关联数据的关联数据
err := s.db.WithContext(ctx).
    Preload("Category").
    Preload("Comments.User").      // 预加载评论及其作者
    Preload("History.Operator").    // 预加载历史记录及操作者
    Preload("Ratings.Rater").       // 预加载评价及评价者
    Where("id = ?", workOrderID).
    First(&workOrder).Error
```

#### 使用指南

**何时使用 Preload:**
- 查询列表时需要显示关联数据（如用户列表需要显示部门名称）
- 关联数据不需要额外的过滤条件
- 关联层级不深（1-2 层）

**何时使用批量查询（WHERE IN）:**
- 需要对关联数据进行额外处理或过滤
- 需要构建复杂的数据结构（如 map）
- 关联数据量较大，需要优化内存使用

**示例对比:**

```go
// ❌ N+1 查询 - 避免！
var users []User
db.Find(&users)
for _, user := range users {
    var dept Department
    db.First(&dept, user.DeptID)  // N 次查询
    user.DeptName = dept.DeptName
}

// ✅ 方案 1: 使用 Preload
var users []User
db.Preload("Dept").Find(&users)  // 2 次查询（1次 users + 1次 depts）

// ✅ 方案 2: 使用批量查询
var users []User
db.Find(&users)

deptIDs := make([]string, len(users))
for i, u := range users {
    deptIDs[i] = u.DeptID
}

var depts []Department
db.Where("id IN ?", deptIDs).Find(&depts)

deptMap := make(map[string]Department)
for _, dept := range depts {
    deptMap[dept.ID] = dept
}

for i := range users {
    if dept, ok := deptMap[users[i].DeptID]; ok {
        users[i].Dept = dept
    }
}
```

#### 已使用 Preload 的模块

- ✅ workorder 模块 - Category, Submitter, Assignee, Dept, Comments.User, History.Operator
- ✅ user 模块 - Dept
- ✅ duty_pool 模块 - Members.User, Dept
- ✅ duty_schedule 模块 - Pool, User
- ✅ notice 模块 - Channels, Targets
- ✅ knowledge 模块 - Category
- ✅ mac_collector 模块 - Device

#### 注意事项

1. **避免过度 Preload**: 只预加载实际需要的数据，每个 Preload 都会产生额外的 SQL 查询
2. **嵌套 Preload**: 需要深层关联数据时使用嵌套 Preload，如 `Preload("Comments.User")`
3. **自定义 Preload**: 可以在 Preload 中使用回调函数进行额外过滤
   ```go
   db.Preload("Orders", "status = ?", "paid").Find(&users)
   ```
4. **使用通用查询构建器**: `pkg/query/builder.go` 支持 Preload 配置

---

### 修复 23: 正确传递 Context ✅ **monitor 模块已完成**

#### 问题描述

在 Handler 中使用 `context.Background()` 会导致：
- 请求取消无法传播
- 超时控制失效
- 无法追踪请求链路

#### 修复内容

**修复范围**: `internal/api/v1/monitor/` 模块

**修复文件**:
1. **`cache_service.go`** - 修改辅助函数签名
   - `getCachesFromCache()` - 添加 `ctx context.Context` 参数
   - `getCachesFromCacheWithLevel()` - 添加 `ctx context.Context` 参数

2. **`cache_handler.go`** - 修复 13 处 `context.Background()` 使用
   - `GetCacheInfo()` - 使用 `c.Request.Context()`
   - `OperateCache()` - 使用 `c.Request.Context()`
   - `BatchOperateCache()` - 使用 `c.Request.Context()`
   - `ClearCache()` - 使用 `c.Request.Context()`
   - `getRealtimeCacheStats()` - 使用 `c.Request.Context()`
   - `GetCacheMonitor()` - 使用 `c.Request.Context()`
   - `TestCacheEndpoint()` - 使用 `c.Request.Context()`
   - `DebugRawKeys()` - 使用 `c.Request.Context()`
   - `DebugL1Cache()` - 使用 `c.Request.Context()`
   - `getCacheListFromRedis()` - 传递 `c.Request.Context()` 给辅助函数

#### 修复示例

**❌ 修复前**:
```go
func (h *CacheHandler) GetCacheInfo(c *gin.Context) {
    ctx := context.Background()  // 错误：创建新的 context
    value, err := h.core.Cache.Get(ctx, key)
    // ...
}
```

**✅ 修复后**:
```go
func (h *CacheHandler) GetCacheInfo(c *gin.Context) {
    ctx := c.Request.Context()  // 正确：从请求获取 context
    value, err := h.core.Cache.Get(ctx, key)
    // ...
}
```

#### 优势

- **请求取消可传播**: 当客户端断开连接时，后台操作会自动取消
- **超时控制生效**: 请求超时会传递到所有子调用
- **链路追踪完整**: 可以追踪完整的请求处理链路
- **资源正确释放**: context 取消时资源会正确释放

#### 待处理

其他模块中也存在类似问题，建议逐步修复：
- `internal/device/` - 设备管理模块
- `internal/services/` - 服务层部分代码
- `internal/scheduler/` - 定时任务模块（注意：后台任务使用 Background 是正确的）

注意：以下场景使用 `context.Background()` 是**正确的**，不应修改：
- 测试文件 (_test.go)
- 后台任务/定时任务（非 HTTP 请求触发）
- 应用启动时的初始化代码
- 独立的后台服务

---

## 第五阶段：类型错误修复

### 修复 24: 前端类型错误修复

#### 问题分析

`npm run build` 失败，`npm run dev` 正常的原因：
- `npm run dev` 只使用 Vite 的 esbuild 转译器，**不进行类型检查**
- `npm run build` 先执行 `tsc -b` 进行**完整的类型检查**

#### 错误清单

1. **类型导出缺失** - `PageData`, `PageParams` 未从 `@/types/operations` 导出
2. **类型不匹配** - `Workstation` 应使用 `WorkstationOps`
3. **组件导出问题** - CAD 组件的导出方式不正确
4. **属性缺失** - `Building` 类型缺少 `longitude`, `latitude` 属性
5. **常量缺失** - `DRAG_THRESHOLD` 未定义
6. **JSX 语法错误** - JSX 表达式结构问题

#### 修复步骤

**步骤 1: 修复类型导出**

修改 `src/types/operations.ts`，添加缺失的导出：

```typescript
// 导出分页类型
export interface PageParams {
  current: number;
  pageSize: number;
}

export interface PageData<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}
```

**步骤 2: 修复 Workstation 类型引用**

修改以下文件中的 `Workstation` 为 `WorkstationOps`：
- `src/services/operations/workstations.ts`
- `src/store/visualizationStore.ts`

```typescript
// ❌ 错误
import type { Workstation } from '@/types/operations';

// ✅ 正确
import type { WorkstationOps } from '@/types/operations';
```

**步骤 3: 修复 Building 类型属性**

修改 `src/types/operations.ts` 中的 `Building` 类型，添加缺失属性：

```typescript
export interface Building {
  id: string;
  name: string;
  code: string;
  address?: string;
  longitude?: number;  // 添加
  latitude?: number;   // 添加
  // ... 其他属性
}
```

**步骤 4: 修复 CAD 组件导出**

修改 CAD 组件的导出方式：

```typescript
// ❌ 错误 - 命名导出
export function CADFloorPlanEditor() { ... }

// ✅ 正确 - 默认导出
export default function CADFloorPlanEditor() { ... }
```

或者保持命名导出，但修改导入方式：

```typescript
// 组件文件
export function CADFloorPlanEditor() { ... }

// index.ts
export { CADFloorPlanEditor } from './CADFloorPlanEditor';
```

**步骤 5: 添加 DRAG_THRESHOLD 常量**

修改 `src/components/cad-editor/CADFloorPlanEditor.tsx`：

```typescript
// 在文件顶部添加常量定义
const DRAG_THRESHOLD = 5; // 拖动阈值（像素）
```

**步骤 6: 修复 JSX 语法**

修改 `src/pages/operations/building-spaces-3d/components/FloorView3D.tsx`：

```typescript
// ❌ 错误
title={<DesktopOutlined /> 工位统计}

// ✅ 正确 - 使用 Fragment
title={<><DesktopOutlined /> 工位统计</>}
```

**步骤 7: 修复 FloorData 类型不一致**

修改 `src/pages/operations/building-spaces-3d/components/types.ts`，确保类型一致性：

```typescript
export interface FloorData {
  id: string;
  code: string;
  floorNo: string;
  workstationCount?: number;  // 添加可选
  status: number;
  // ... 其他属性
}
```

**步骤 8: 修复 Card 组件导入**

修改 `src/pages/operations/floors/components/FloorSearchForm.tsx`：

```typescript
import { Card } from 'antd';  // 添加导入
```

#### 验证修复

```bash
# 1. 类型检查
cd xingran-react-frontend
npx tsc --noEmit

# 2. 构建
npm run build

# 3. 开发服务器验证
npm run dev
```

---

## 验证检查清单

### 安全修复验证

- [ ] SQL 注入防护测试
- [x] LDAP 证书验证（已确认忽略，内部网络环境）
- [ ] 登出功能验证

### 功能测试验证

- [ ] 操作日志正确记录
- [ ] 用户解锁功能正常

### 代码质量验证

- [x] console.log 已清理（使用 vite-plugin-remove-console）
- [x] panic 已改为 error
- [x] 类型检查通过
- [ ] 测试覆盖率提升

---

## 附录：快速参考

### 常用命令

```bash
# 测试
go test ./...
npm run test

# 类型检查
tsc --noEmit

# 构建
npm run build
go build ./cmd/...
```

---

**文档维护**: 本文档应随着代码修复的进展持续更新。
