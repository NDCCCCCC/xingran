# Phase 16: API 密钥管理功能 - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning
**Source:** Technical Specification (User Provided)

---

## Phase Boundary

实现完整的 API 密钥管理系统，支持密钥的 CRUD 操作、自动生成、权限控制、使用日志审计和前端管理页面。该系统将为外部应用提供安全的 API 访问能力，支持速率限制、IP 白名单、权限继承等企业级特性。

**核心交付物：**
1. 后端数据模型和服务层（APIKey、APIKeyUsageLog）
2. 密钥自动生成和验证机制
3. 多重认证中间件（JWT + API Key）
4. 速率限制器和 IP 白名单
5. 前端管理页面（列表、创建、编辑、删除、使用日志）
6. 使用日志和统计分析

---

## Implementation Decisions

### 数据模型设计

**APIKey 模型 (`internal/models/api_key.go`)**
- `id`: UUID 主键
- `name`: 密钥名称（必填）
- `key`: 密钥值（rec_ + 64位十六进制随机数，唯一索引）
- `user_id`: 所属用户 ID（外键 → sys_user）
- `expires_at`: 过期时间（可选，NULL 表示永不过期）
- `last_used_at`: 最后使用时间（自动更新）
- `is_active`: 是否启用（默认 true）
- `scopes`: 作用域 JSON 数组（read、write、admin）
- `ip_whitelist`: IP 白名单 JSON 数组（支持单 IP 或 CIDR）
- `description`: 描述信息（可选）
- `inherit_perms`: 是否继承用户角色权限（默认 false）
- `created_at`, `updated_at`, `deleted_at`: 标准时间戳

**APIKeyUsageLog 模型 (`internal/models/api_key_usage_log.go`)**
- `id`: 主键
- `api_key_id`: API 密钥 ID（外键）
- `user_id`: 用户 ID（外键）
- `method`: HTTP 方法
- `path`: 请求路径
- `status_code`: 响应状态码
- `client_ip`: 客户端 IP
- `user_agent`: User-Agent 字符串
- `duration`: 请求耗时（毫秒）
- `success`: 是否成功
- `created_at`: 创建时间

### 服务层功能

**核心服务方法 (`internal/services/apikey_service.go`)**
- `CreateAPIKey(ctx, req)`: 生成密钥、验证作用域、解析过期时间
- `ListAPIKeys(ctx, page, pageSize, keyword, status)`: 分页查询、关键词搜索、状态筛选
- `GetAPIKey(ctx, id)`: 获取详情（管理员可查看所有密钥）
- `UpdateAPIKey(ctx, id, req)`: 更新名称、状态、作用域、IP 白名单
- `DeleteAPIKey(ctx, id)`: 软删除
- `ToggleAPIKeyStatus(ctx, id)`: 启用/禁用切换
- `ValidateAPIKey(ctx, keyStr)`: 验证密钥有效性（供中间件使用）
- `ListUsageLogs(ctx, keyID, page, pageSize)`: 分页查询使用日志
- `GetUsageLogSummary(ctx, keyID)`: 聚合统计数据

### 路由配置

**API 路由 (`cmd/server/app.go`)**
```
GET    /api/v1/apikeys          — 列表查询
POST   /api/v1/apikeys          — 创建密钥
GET    /api/v1/apikeys/:id      — 获取详情
PUT    /api/v1/apikeys/:id      — 更新密钥
DELETE /api/v1/apikeys/:id      — 删除密钥
POST   /api/v1/apikeys/:id/toggle — 切换状态
GET    /api/v1/apikeys/:id/logs — 使用日志
GET    /api/v1/apikeys/:id/summary — 使用统计
```

### 中间件层

**多重认证中间件 (`internal/middleware/apikey.go`)**
- `MultiAuth()`: 检查 X-API-Key 请求头
  - 验证密钥格式（rec_ 前缀 + 64位 hex）
  - 检查过期时间和启用状态
  - 验证 IP 白名单（支持 CIDR）
  - 设置用户上下文
  - 异步记录使用日志

**作用域验证中间件**
- `RequireScope(requiredScope)`: 验证 API Key 作用域
  - inherit_perms=true: 检查用户角色权限
  - inherit_perms=false: 检查 API Key 自定义作用域

**资源权限验证中间件**
- `RequireAPIKeyResourcePermission(resource, action)`: 将资源操作映射到作用域
  - view → read
  - create/edit/delete → write

**速率限制中间件**
- `RateLimitByScope()`: 滑动窗口算法
  - read: 30/min, 500/hour, 5,000/day
  - write: 100/min, 1,500/hour, 15,000/day
  - admin: 200/min, 5,000/hour, 50,000/day
  - 继承权限: 120/min, 2,000/hour, 20,000/day

### 前端实现

**类型定义 (`src/types/apikey.ts`)**
```typescript
interface APIKey {
  id: number
  name: string
  key: string
  scopes: string[]
  ip_whitelist: string[]
  inherit_perms: boolean
  expires_at?: string
  last_used_at?: string
  is_active: boolean
  description: string
  created_at: string
}

interface CreateAPIKeyRequest {
  name: string
  description?: string
  scopes: string[]
  inherit_perms: boolean
  ip_whitelist?: string[]
  expires_at?: string
}
```

**API 调用 (`src/api/apikey.ts`)**
```typescript
export function listAPIKeys(params?: ListAPIKeysRequest)
export function createAPIKey(data: CreateAPIKeyRequest)
export function updateAPIKey(id: number, data: UpdateAPIKeyRequest)
export function deleteAPIKey(id: number)
export function toggleAPIKeyStatus(id: number)
export function listUsageLogs(keyID: number, params?: ListLogsRequest)
export function getUsageSummary(keyID: number)
```

**管理页面 (`src/pages/system/apikeys/index.tsx`)**
- 密钥列表（分页、搜索、筛选）
- 创建密钥（作用域、有效期、IP 白名单）
- 编辑密钥配置
- 启用/禁用切换
- 删除密钥
- 复制密钥（脱敏显示）
- 使用日志和统计

**安全设计：**
- 完整密钥仅创建时显示一次
- 列表仅显示前 12 位脱敏密钥
- 支持一键复制

### 速率限制器

**实现位置 (`internal/services/rate_limiter.go`)**
- 滑动窗口算法
- 内存级速率限制（分钟/小时/天三级）
- 自动清理过期窗口
- 根据作用域动态调整限制
- 返回剩余请求数和重置时间

### 安全特性

| 特性 | 实现方式 |
|------|----------|
| 密钥生成 | crypto/rand + rec_ 前缀 |
| 密钥存储 | GORM 自动加密（单次显示） |
| IP 限制 | 白名单 + CIDR 支持 |
| 过期检查 | 每次请求验证 |
| 权限隔离 | 用户只能查看自己的密钥（管理员除外） |
| 审计日志 | 所有操作记录到审计表 |
| 使用追踪 | 异步记录每次 API 调用 |
| 速率限制 | 滑动窗口算法 |
| 脱敏显示 | 密钥仅显示前 12 位 |

### 关键设计决策

1. **密钥格式**: rec_ 前缀便于识别，64 位 hex 保证安全性
2. **权限继承**: 支持两种模式，灵活应对不同场景
3. **异步日志**: 使用 goroutine 记录使用日志，不影响请求性能
4. **脱敏显示**: 完整密钥仅返回一次，防止泄露
5. **速率限制**: 根据作用域差异化配置，优化资源分配

---

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目架构
- `docs/项目概述和架构设计.md` — 系统整体架构和模块划分
- `docs/开发规范.md` — 代码规范和开发约定
- `docs/API响应规范.md` — API 响应格式规范
- `docs/安全和认证设计（国密）.md` — 安全认证机制（JWT、SM2/SM4）

### 后端模式
- `internal/models/base.go` — 模型基类和公共字段
- `internal/api/v1/system/user_handler.go` — Handler 模式参考
- `internal/services/system/user_service.go` — Service 接口模式参考
- `pkg/middleware/auth.go` — 现有认证中间件
- `pkg/permission/permissions.go` — 权限控制机制

### 前端模式
- `src/lib/api.ts` — API 调用封装
- `src/types/` — 类型定义目录
- `src/pages/system/` — 系统管理页面参考

---

## Specific Ideas

### 密钥生成算法
```go
// 使用 crypto/rand 生成 32 字节随机数，转换为 64 位 hex
key := fmt.Sprintf("rec_%x", randomBytes)
```

### 速率限制数据结构
```go
type rateLimitWindow struct {
    minute []time.Time
    hour   []time.Time
    day    []time.Time
    mu     sync.Mutex
}
```

### IP 白名单验证
```go
// 使用 net.ParseCIDR 和 net.Contains 支持 CIDR 表示法
// 例如: 192.168.1.100 或 10.0.0.0/24
```

### 前端脱敏显示
```typescript
// 仅显示前 12 位：rec_1a2b3c4d5e6f...
const maskedKey = key.slice(0, 12) + '...'
```

---

## Deferred Ideas

无 — 所有功能已在本次阶段规划中完成

---

*Phase: 16-api-key-mgt*
*Context gathered: 2026-05-18 via Technical Specification*
