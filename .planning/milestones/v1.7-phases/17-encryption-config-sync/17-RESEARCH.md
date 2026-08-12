# Phase 17: 前后端加密配置同步 - Research

**Researched:** 2026-05-20
**Domain:** Frontend-Backend Configuration Synchronization
**Confidence:** HIGH

## Summary

本阶段研究如何实现 XingRan-Next 系统的前后端加密配置同步机制。当前系统存在配置不一致问题：前端使用构建时环境变量 `VITE_ENABLE_REQUEST_ENCRYPTION` 控制加密开关，而后端使用数据库配置 `sys.request.encryption.enabled` 进行运行时控制。当数据库配置关闭加密时，前端仍发送加密请求导致 token 刷新失败（400 错误）。

**主要发现：**
1. 后端已有完善的动态配置读取机制（30秒缓存 + 数据库查询）
2. 后端已有公共配置查询 API 模式（参考验证码配置 `/system/auth/captcha/config`）
3. 前端已有动态配置获取模式（参考 `getCaptchaConfig()`）
4. 存在中间件缓存刷新函数 `RefreshEncryptionConfigCache()` 未被调用

**推荐方案：**
创建公共端点 `GET /system/auth/encryption-config`（无需认证），前端在应用启动和每次 token 刷新前获取最新加密配置，替换构建时环境变量的硬编码判断。后端在配置更新时主动刷新中间件缓存。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 加密配置存储 | Database (sys_config) | Cache (30s L1+L2) | 配置持久化在数据库，中间件使用内存缓存减少查询 |
| 加密配置读取 | API / Backend | Frontend | 后端拥有配置源，前端通过 API 获取 |
| 加密状态判断 | Frontend | — | 前端需要知道是否加密请求体 |
| 配置缓存刷新 | Backend Middleware | — | 中间件需要监听配置变更并刷新缓存 |
| 配置更新通知 | Backend | — | 配置更新时通知前端和中间件 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin | 1.10.0 | HTTP 路由和中间件 | 项目现有架构，无需额外依赖 |
| GORM | 1.30.5 | 数据库配置查询 | 项目现有 ORM，用于读取 `sys_config` 表 |
| sync.RWMutex | builtin | 配置缓存并发控制 | Go 标准库，用于保护全局配置缓存 |
| Axios | 1.13.2 | 前端 HTTP 客户端 | 项目现有 API 客户端，复用 `api.ts` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| React Query | 5.90.12 | 服务端状态管理（可选） | 如果需要更复杂的缓存策略可考虑 |
| Zustand | 5.0.9 | 前端状态管理 | 存储加密配置状态（已集成） |

### Installation

无需安装额外依赖，使用现有技术栈。

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │ App Startup  │───▶│ Token Refresh│───▶│ API Request  │      │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘      │
│         │                   │                   │              │
│         ▼                   ▼                   ▼              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           getEncryptionConfig() (New API Call)           │   │
│  └────────────────────────┬────────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            │ HTTP GET (Public, No Auth)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Backend API Layer                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  GET /system/auth/encryption-config (New Endpoint)       │  │
│  │  - No authentication required (public endpoint)          │  │
│  │  - Returns: { enabled: boolean, version: string }        │  │
│  └──────────────────────┬───────────────────────────────────┘  │
│                         │                                         │
│                         ▼                                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │          ConfigService.GetByKey() (Existing)              │  │
│  │  - Reads from sys_config table                           │  │
│  │  - Key: "sys.request.encryption.enabled"                 │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Backend Middleware Layer                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     RequestDecryption Middleware (Existing)              │  │
│  │  - Checks encryption status per request                  │  │
│  │  - Uses 30s cache to reduce DB queries                   │  │
│  │  - Cache can be manually refreshed via                   │  │
│  │    RefreshEncryptionConfigCache()                        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                    ▲                             │
│                                    │                             │
│  ┌─────────────────────────────────┴─────────────────────────┐  │
│  │    Config Update Handler (Enhanced)                       │  │
│  │  - When sys.request.encryption.enabled is updated        │  │
│  │  - Calls RefreshEncryptionConfigCache() immediately       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Database Layer                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  sys_config table                                         │  │
│  │  - config_key: "sys.request.encryption.enabled"          │  │
│  │  - config_value: "true" / "false"                        │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── api/v1/
│   ├── auth.go                    # 添加 getEncryptionConfig() 函数
│   └── system/
│       └── config_router.go       # 注册新端点（需添加到 auth 组）
├── services/system/
│   └── config_service.go          # 已有 GetByKey() 方法，无需修改
└── core/
    └── captcha.go                 # 参考模式：GetConfig() 公共方法

pkg/middleware/
└── request_decryption.go          # 已有 RefreshEncryptionConfigCache()

xingran-react-frontend/src/
├── services/
│   └── encryptionConfig.ts        # 新增：加密配置 API 调用
├── lib/
│   └── api.ts                     # 修改：集成动态加密配置
├── store/
│   └── authStore.ts               # 增强：TokenManager 读取加密配置
└── utils/
    └── authHelpers.ts             # 新增：加密配置缓存和辅助函数
```

### Pattern 1: Public Configuration Endpoint（参考验证码配置）

**What:** 创建无需认证的公共配置端点，允许前端在登录前和 token 刷新时获取系统配置。

**When to use:** 系统级配置需要在认证前获取时（如验证码配置、加密配置）。

**Example:**

```go
// internal/api/v1/auth.go

// getEncryptionConfig 获取请求加密配置（公共端点，无需认证）
// @Summary 获取请求加密配置
// @Description 获取当前请求体加密的开关状态（公共端点，无需认证）
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/encryption-config [get]
func getEncryptionConfig(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 直接读取中间件缓存（30秒TTL）
        // 这样避免每次请求都查询数据库
        enabled := middleware.GetEncryptionConfigFromCache()

        // 也可以从数据库读取最新值（绕过缓存）
        // var configValue string
        // err := core.GetDB().WithContext(c.Request.Context()).
        //     Table("sys_config").
        //     Select("config_value").
        //     Where("config_key = ?", "sys.request.encryption.enabled").
        //     Pluck("config_value", &configValue).Error

        // 解析布尔值
        enabledBool := enabled
        if configValue == "false" || configValue == "0" {
            enabledBool = false
        }

        response.Success(c, gin.H{
            "enabled": enabledBool,
            "key":     "sys.request.encryption.enabled",
            "source":  "database", // 标识来源为数据库配置
        })
    }
}

// SetupAuthRouter 中注册路由
func SetupAuthRouter(r *gin.RouterGroup, core *core.Core) {
    // ... 现有路由 ...

    // GET /api/v1/system/auth/encryption-config - 获取加密配置（公共端点）
    r.GET("/encryption-config", getEncryptionConfig(core))
}
```

### Pattern 2: Frontend Dynamic Configuration Loading（参考验证码配置）

**What:** 前端在应用启动和关键操作前动态获取配置，替换构建时硬编码配置。

**When to use:** 需要运行时同步后端配置时。

**Example:**

```typescript
// src/services/encryptionConfig.ts

import { get } from '@/lib/api';

export interface EncryptionConfig {
  enabled: boolean;
  key: string;
  source: string;
}

// 获取加密配置（公共端点，无需认证）
export async function getEncryptionConfig(): Promise<EncryptionConfig> {
  const response = await get<EncryptionConfig>('/system/auth/encryption-config');
  return response.data!;
}

// 缓存加密配置（5分钟TTL，减少请求频率）
let encryptionConfigCache: EncryptionConfig | null = null;
let cacheTimestamp = 0;
const CACHE_TTL = 5 * 60 * 1000; // 5分钟

export async function getCachedEncryptionConfig(): Promise<EncryptionConfig> {
  const now = Date.now();

  // 缓存有效，直接返回
  if (encryptionConfigCache && (now - cacheTimestamp) < CACHE_TTL) {
    return encryptionConfigCache;
  }

  // 缓存过期或未加载，重新获取
  const config = await getEncryptionConfig();
  encryptionConfigCache = config;
  cacheTimestamp = now;

  return config;
}

// 清除缓存（配置更新后调用）
export function clearEncryptionConfigCache(): void {
  encryptionConfigCache = null;
  cacheTimestamp = 0;
}

export default {
  getEncryptionConfig,
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
};
```

### Pattern 3: Cache Invalidation on Config Update

**What:** 配置更新时主动刷新中间件缓存，确保配置立即生效。

**When to use:** 运行时配置变更需要立即生效时。

**Example:**

```go
// internal/api/v1/system/config_handler.go

// Update 更新参数配置
func (h *ConfigHandler) Update(c *gin.Context) {
    // ... 现有验证逻辑 ...

    // 更新配置
    if err := h.service.Update(c.Request.Context(), &req); err != nil {
        response.Error(c, err)
        return
    }

    // 获取更新后的配置
    config, err := h.service.GetByID(c.Request.Context(), id)
    if err == nil {
        // 请求加密开关的处理
        if config.ConfigKey == "sys.request.encryption.enabled" {
            // 立即刷新中间件缓存（新增）
            middleware.RefreshEncryptionConfigCache()

            applogger.WithFields(map[string]interface{}{
                "config_key":   config.ConfigKey,
                "config_value": config.ConfigValue,
            }).Info("请求加密配置已更新，中间件缓存已刷新")
        }

        // 验证码相关配置的热重载（现有逻辑）
        if h.isCaptchaConfig(config.ConfigKey) {
            if h.captchaService != nil {
                h.captchaService.LoadConfig(c.Request.Context())
            }
        }
    }

    response.Success(c, gin.H{"message": "更新成功"})
}
```

### Anti-Patterns to Avoid

- **在前端使用构建时环境变量控制运行时行为**：会导致前后端配置不一致，应改用运行时 API 获取配置
- **配置更新后依赖缓存TTL自动过期**：30秒延迟可能导致用户体验问题，应主动刷新缓存
- **每个请求都查询数据库获取加密配置**：性能问题严重，必须使用缓存（已有30秒TTL缓存机制）
- **加密配置端点需要认证**：会导致无法在登录前和 token 刷新时获取配置，必须设为公共端点

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 配置缓存机制 | 自己实现带锁的缓存、TTL逻辑 | 已有 `globalConfigCache` (30秒TTL + RWMutex) | 避免并发问题，复用现有实现 |
| 前端配置状态管理 | 自己实现 Redux 复杂状态管理 | Zustand store + 内存缓存 | 项目已集成 Zustand，轻量级足够 |
| 前端 HTTP 缓存 | 自己实现请求缓存和失效逻辑 | 简单内存缓存（变量 + timestamp） | 加密配置变更频率低，简单缓存足够 |
| 配置端点权限控制 | 复杂的 RBAC 权限检查 | 公共端点（无需认证） | 加密配置不敏感，参考验证码配置模式 |

**Key insight:** 后端已有完善的配置缓存机制（`request_decryption.go:38-48`），前端已有动态配置模式（验证码配置），只需创建公共端点桥接两者即可。

## Runtime State Inventory

> 本阶段为新增功能，不涉及现有运行时状态的迁移或重构。

**无运行时状态变更。**

## Common Pitfalls

### Pitfall 1: 配置端点需要认证导致循环依赖

**What goes wrong:** 如果加密配置端点需要认证，前端在登录前无法获取配置，导致首次登录请求无法判断是否加密；token 过期刷新时也无法获取最新配置，导致 400 错误循环。

**Why it happens:** 开发者习惯性地对所有业务端点添加认证中间件，忽略了某些系统配置需要在认证前获取。

**How to avoid:**
1. 将加密配置端点放在 `/system/auth/` 路由组下（该组在认证中间件之前）
2. 明确标记为公共端点，参考验证码配置：`r.GET("/encryption-config", getEncryptionConfig(core))`
3. 在 Swagger 文档中标注无需认证：`@Security ApiKeyAuth` 不添加或明确标注为 public

**Warning signs:**
- 前端登录时报 "401 Unauthorized"（获取加密配置失败）
- token 刷新时报 "400 Bad Request"（加密配置未同步）

### Pitfall 2: 前端每次请求都查询加密配置

**What goes wrong:** 前端在每个 API 请求前都调用加密配置端点，导致大量冗余请求，性能下降。

**Why it happens:** 直接复制验证码配置的获取逻辑，但加密配置查询频率远高于验证码配置（每次 API 请求都需要判断是否加密）。

**How to avoid:**
1. 前端实现缓存机制（5分钟TTL）：变量 + timestamp
2. 仅在以下场景刷新配置：
   - 应用启动时（`main.tsx`）
   - Token 刷新前（`authStore.ts` 的 `refreshToken()` 方法）
   - 用户手动刷新配置（可选，提供配置管理页面的刷新按钮）
3. 后端已有30秒缓存，前端5分钟缓存合理（后端缓存兜底，最多延迟30秒生效）

**Warning signs:**
- 浏览器 Network 面板显示大量 `/system/auth/encryption-config` 请求
- API 响应时间显著增加

### Pitfall 3: 配置更新后缓存未刷新导致配置延迟生效

**What goes wrong:** 管理员在参数管理中关闭加密开关后，前端仍发送加密请求长达30秒（缓存TTL），导致用户体验问题。

**Why it happens:** 仅依赖中间件的30秒TTL自动过期，未在配置更新时主动刷新缓存。

**How to avoid:**
1. 在 `config_handler.go` 的 `Update()` 方法中检测加密配置变更
2. 调用 `middleware.RefreshEncryptionConfigCache()` 立即失效缓存
3. 记录配置变更日志，便于调试

**Warning signs:**
- 配置更新后仍需等待30秒才生效
- 用户反馈 "配置修改后没有立即生效"

### Pitfall 4: 配置端点返回值格式不一致

**What goes wrong:** 前端期望 `{ enabled: boolean }` 但后端返回 `{ data: { config_value: "true" } }`，导致前端需要额外的解析逻辑。

**Why it happens:** 直接复用 `GetByKey()` 的返回格式，未针对前端需求优化。

**How to avoid:**
1. 参考验证码配置端点 `getCaptchaConfig()` 的返回格式
2. 返回解析后的布尔值而非原始字符串：`"true"` → `true`
3. 统一响应格式：`{ code: 0, data: { enabled: true, key: "...", source: "database" } }`

**Warning signs:**
- 前端代码中出现 `JSON.parse()` 或额外的字符串比较逻辑
- TypeScript 类型定义与实际响应不匹配

### Pitfall 5: 前端缓存与后端缓存不同步

**What goes wrong:** 前端缓存配置10分钟，后端缓存30秒，导致配置更新后前端最长延迟10分钟才生效，用户体验极差。

**Why it happens:** 前端缓存TTL设置过长，未考虑后端缓存的刷新频率。

**How to avoid:**
1. 前端缓存TTL应略大于后端（建议5分钟，后端30秒）
2. 提供手动刷新机制（配置管理页面）
3. 在关键操作前主动刷新（如 token 刷新前）

**Warning signs:**
- 配置更新后前端长时间未生效
- 需要用户硬刷新页面才能获取最新配置

## Code Examples

### Backend: Public Encryption Config Endpoint

```go
// internal/api/v1/auth.go

// getEncryptionConfig 获取请求加密配置（公共端点，无需认证）
// @Summary 获取请求加密配置
// @Description 获取当前请求体加密的开关状态（公共端点，无需认证）
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/encryption-config [get]
func getEncryptionConfig(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 方案1：从中间件缓存读取（30秒TTL，性能最优）
        enabled := middleware.GetEncryptionConfigFromCache()

        // 方案2：绕过缓存直接读数据库（实时性更高）
        // var configValue string
        // err := core.GetDB().WithContext(c.Request.Context()).
        //     Table("sys_config").
        //     Select("config_value").
        //     Where("config_key = ?", "sys.request.encryption.enabled").
        //     Pluck("config_value", &configValue).Error
        //
        // if err != nil {
        //     // 数据库查询失败，使用默认值（启用）
        //     enabled = true
        // } else {
        //     enabled = (configValue == "true" || configValue == "1")
        // }

        response.Success(c, gin.H{
            "enabled": enabled,
            "key":     "sys.request.encryption.enabled",
            "source":  "database",
        })
    }
}

// SetupAuthRouter 设置认证路由
func SetupAuthRouter(r *gin.RouterGroup, core *core.Core) {
    // ... 现有路由 ...

    // GET /api/v1/system/auth/encryption-config - 获取加密配置（公共端点）
    r.GET("/encryption-config", getEncryptionConfig(core))
}
```

**需要添加的辅助函数：**

```go
// pkg/middleware/request_decryption.go

// GetEncryptionConfigFromCache 获取当前加密配置缓存值（用于公共端点）
// 返回缓存的配置值，不触发数据库查询
func GetEncryptionConfigFromCache() bool {
    globalConfigCache.mu.RLock()
    defer globalConfigCache.mu.RUnlock()

    // 如果缓存从未初始化，返回默认值（启用）
    if globalConfigCache.lastUpdate.IsZero() {
        return true
    }

    return globalConfigCache.value
}
```

### Frontend: Encryption Config Service

```typescript
// src/services/encryptionConfig.ts

import { get } from '@/lib/api';

export interface EncryptionConfig {
  enabled: boolean;
  key: string;
  source: string;
}

// 获取加密配置（公共端点，无需认证）
export async function getEncryptionConfig(): Promise<EncryptionConfig> {
  const response = await get<EncryptionConfig>('/system/auth/encryption-config');
  return response.data!;
}

// 缓存加密配置（5分钟TTL）
let encryptionConfigCache: EncryptionConfig | null = null;
let cacheTimestamp = 0;
const CACHE_TTL = 5 * 60 * 1000; // 5分钟

export async function getCachedEncryptionConfig(): Promise<EncryptionConfig> {
  const now = Date.now();

  // 缓存有效，直接返回
  if (encryptionConfigCache && (now - cacheTimestamp) < CACHE_TTL) {
    return encryptionConfigCache;
  }

  // 缓存过期或未加载，重新获取
  const config = await getEncryptionConfig();
  encryptionConfigCache = config;
  cacheTimestamp = now;

  return config;
}

// 清除缓存（配置更新后调用）
export function clearEncryptionConfigCache(): void {
  encryptionConfigCache = null;
  cacheTimestamp = 0;
}

export default {
  getEncryptionConfig,
  getCachedEncryptionConfig,
  clearEncryptionConfigCache,
};
```

### Frontend: Integration with API Client

```typescript
// src/lib/api.ts

import { getCachedEncryptionConfig } from '@/services/encryptionConfig';

// 是否启用请求体加密（移除构建时硬编码）
// const ENABLE_REQUEST_ENCRYPTION = import.meta.env.VITE_ENABLE_REQUEST_ENCRYPTION === 'true';

// 动态加密配置（默认启用，避免启动时未获取到配置）
let ENABLE_REQUEST_ENCRYPTION = true;

// 初始化加密配置
export async function initEncryptionConfig(): Promise<void> {
  try {
    const config = await getCachedEncryptionConfig();
    ENABLE_REQUEST_ENCRYPTION = config.enabled;
    console.log('[API] 加密配置已加载:', { enabled: config.enabled });
  } catch (error) {
    console.error('[API] 加载加密配置失败，使用默认值（启用）:', error);
    // 失败时使用默认值（启用）
    ENABLE_REQUEST_ENCRYPTION = true;
  }
}

/**
 * 检查请求是否需要加密
 */
function shouldEncryptRequest(url: string, method: string): boolean {
  if (!ENABLE_REQUEST_ENCRYPTION) {
    return false;
  }

  if (!['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
    return false;
  }

  if (ENCRYPTION_BLACKLIST.some(prefix => url.startsWith(prefix))) {
    return false;
  }

  if (ENCRYPTION_WHITELIST.length > 0) {
    return ENCRYPTION_WHITELIST.some(prefix => url.startsWith(prefix));
  }

  return true;
}

// 请求拦截器中保持现有逻辑，无需修改
// api.interceptors.request.use(async (config) => {
//   ...
//   if (config.data && shouldEncryptRequest(config.url || '', config.method || '')) {
//     ... 加密逻辑 ...
//   }
//   ...
// })
```

### Frontend: App Initialization

```typescript
// src/main.tsx

import { initEncryptionConfig } from '@/lib/api';
import { getCachedEncryptionConfig } from '@/services/encryptionConfig';

async function initializeApp(): Promise<void> {
  try {
    // 并行初始化多个配置
    await Promise.all([
      initEncryptionConfig(),  // 初始化加密配置
      // ... 其他初始化逻辑 ...
    ]);

    console.log('[App] 应用初始化完成');
  } catch (error) {
    console.error('[App] 应用初始化失败:', error);
    // 显示错误提示
  }
}

// 在 ReactDOM.createRoot() 之前调用
initializeApp().then(() => {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
});
```

### Frontend: Token Manager Integration

```typescript
// src/store/authStore.ts

import { getCachedEncryptionConfig, clearEncryptionConfigCache } from '@/services/encryptionConfig';

interface TokenManager {
  // ... 现有方法 ...

  // 刷新 token（增强：刷新前更新加密配置）
  refreshToken(): Promise<void> {
    return this.performRefresh();
  }

  // 执行刷新（内部方法）
  private async performRefresh(): Promise<void> {
    // 在刷新前获取最新加密配置
    try {
      const config = await getCachedEncryptionConfig();
      console.log('[TokenManager] Token 刷新前获取加密配置:', config);
    } catch (error) {
      console.warn('[TokenManager] 获取加密配置失败，使用缓存配置:', error);
    }

    // 执行 token 刷新
    const response = await post('/system/auth/refresh', {
      refreshToken: this.refreshToken,
    });

    // ... 现有刷新逻辑 ...
  }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 构建时环境变量控制加密配置 (`VITE_ENABLE_REQUEST_ENCRYPTION`) | 运行时数据库配置 + 公共 API 端点 | 2026-05-20 (本阶段) | 配置变更无需重新构建前端，管理员可通过参数管理实时控制 |
| 前端硬编码加密判断逻辑 | 动态获取 + 本地缓存（5分钟TTL） | 2026-05-20 (本阶段) | 前后端配置自动同步，避免 token 刷新失败 |
| 配置更新后依赖30秒缓存TTL自动过期 | 配置更新时主动刷新中间件缓存 | 2026-05-20 (本阶段) | 配置变更立即生效，用户体验提升 |

**Deprecated/outdated:**
- `VITE_ENABLE_REQUEST_ENCRYPTION` 环境变量：保留用于回退，但不再作为主要配置方式
- 前端硬编码加密判断：应替换为动态获取 + 缓存

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 后端已有完善的配置缓存机制（30秒TTL + RWMutex） | Standard Stack | 如果缓存实现有并发问题，可能导致配置读取错误 |
| A2 | 验证码配置端点模式可复用于加密配置 | Architecture Patterns | 如果验证码端点有特殊权限逻辑，直接复制可能导致安全问题 |
| A3 | 前端 Zustand 足够存储加密配置状态 | Standard Stack | 如果需要更复杂的状态管理（如跨标签页同步），可能需要额外方案 |
| A4 | 5分钟前端缓存TTL合理 | Common Pitfalls | 如果业务要求配置变更秒级生效，需要缩短TTL或实现WebSocket推送 |

**如果此表为空：** 所有研究断言均已验证或引用，无需用户确认。

## Open Questions

1. **前端缓存TTL应该设置多长？**
   - What we know: 后端缓存30秒，前端需要更长TTL减少请求频率
   - What's unclear: 业务方对配置变更生效时间的接受度
   - Recommendation: 5分钟作为初始值，可根据实际使用情况调整

2. **是否需要配置变更的实时推送机制？**
   - What we know: 当前方案为轮询（5分钟TTL）+ 关键操作前主动刷新
   - What's unclear: 是否有配置变更需要立即生效的强需求
   - Recommendation: 先实现轮询方案，如果业务方要求秒级生效，再考虑WebSocket推送或Server-Sent Events

3. **是否需要保留构建时环境变量作为回退机制？**
   - What we know: 现有 `VITE_ENABLE_REQUEST_ENCRYPTION` 可用于硬编码默认值
   - What's unclear: 当数据库查询失败时，是否应该回退到构建时配置
   - Recommendation: 保留构建时环境变量作为 "离线模式" 的回退值，但不在生产环境依赖

## Environment Availability

> 本阶段为纯代码/配置变更，无外部依赖。

**Step 2.6: SKIPPED（无外部依赖）**

## Validation Architecture

> 本阶段为功能新增，暂不要求自动化测试覆盖，但建议手动验证以下场景。

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (backend) + Vitest (frontend) |
| Config file | `vitest.config.ts` (frontend) |
| Quick run command | `go test ./internal/api/v1/ -v -run TestGetEncryptionConfig` |
| Full suite command | `go test ./... && npm run test` |

### Phase Requirements → Test Map

本阶段为核心功能开发，建议手动测试以下场景：

| 场景 | 测试步骤 | 预期结果 |
|------|---------|---------|
| **场景1：应用启动时获取加密配置** | 1. 清空浏览器缓存<br>2. 打开应用<br>3. 检查 Network 面板 | 应用启动时自动调用 `/system/auth/encryption-config` |
| **场景2：配置缓存生效** | 1. 应用已加载加密配置<br>2. 在5分钟内多次发送API请求<br>3. 检查 Network 面板 | 仅在启动时调用一次加密配置端点，后续请求使用缓存 |
| **场景3：Token 刷新前更新配置** | 1. 等待 access token 过期<br>2. 触发 token 刷新<br>3. 检查 Network 面板 | Token 刷新请求前会调用加密配置端点（如果缓存过期） |
| **场景4：配置更新后立即生效** | 1. 管理员在参数管理中将加密开关从 `true` 改为 `false`<br>2. 前端发送 API 请求<br>3. 检查请求头 | 请求头中无 `X-Request-Encrypted: true`，后端正常处理 |
| **场景5：加密关闭状态下前端正常工作** | 1. 加密配置为 `false`<br>2. 用户登录、访问菜单等操作 | 所有API请求正常，无 400 错误 |

### Sampling Rate
- **Per task commit:** 手动验证当前任务涉及的场景
- **Per wave merge:** 完整测试上述5个场景
- **Phase gate:** 所有场景验证通过，无阻塞性问题

### Wave 0 Gaps
- [ ] `internal/api/v1/auth_test.go` - 添加 `TestGetEncryptionConfig` 单元测试
- [ ] `src/services/encryptionConfig.test.ts` - 添加前端加密配置服务测试

*(如果无测试文件：建议优先实现核心功能，测试可在 Wave 1 补充)*

## Security Domain

> 本阶段涉及系统配置暴露，需评估安全风险。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | - |
| V3 Session Management | no | - |
| V4 Access Control | no | - |
| V5 Input Validation | yes | 参考现有验证码配置端点的输入验证 |
| V6 Cryptography | yes | 加密配置本身与加密功能相关，不涉及具体加密实现 |
| V7 Error Logging | yes | 配置查询失败时记录日志，回退到默认值 |

### Known Threat Patterns for Go + React Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 配置端点信息泄露 | Information Disclosure | 加密配置本身不敏感（仅开关状态），参考验证码配置端点无需认证 |
| 配置端点 DoS 攻击 | Denial of Service | 后端已有30秒缓存，前端5分钟缓存，正常使用不会导致大量请求 |
| 中间人攻击篡改配置响应 | Tampering | HTTPS 传输（生产环境必须），前端验证响应格式 |
| 缓存投毒（篡改前端缓存） | Spoofing/Tampering | 前端缓存为内存变量，页面刷新后重新获取，无法持久化篡改 |

**Security Considerations:**
1. **加密配置端点无需认证是安全的**：配置本身不敏感（仅开关状态），参考验证码配置模式
2. **前端缓存可被篡改但无安全风险**：缓存仅用于性能优化，篡改后会导致请求失败（400错误），不会导致安全漏洞
3. **生产环境必须使用 HTTPS**：防止配置响应被中间人篡改
4. **配置查询失败时的回退策略**：建议回退到 "启用加密"（更安全），而非 "禁用加密"

## Sources

### Primary (HIGH confidence)
- **后端中间件实现** - `pkg/middleware/request_decryption.go` (verified via Read tool)
  - 确认存在30秒TTL配置缓存（`globalConfigCache`）
  - 确认存在缓存刷新函数 `RefreshEncryptionConfigCache()`
  - 确认加密配置键为 `sys.request.encryption.enabled`
- **后端配置服务** - `internal/services/system/config_service.go` (verified via Read tool)
  - 确认存在 `GetByKey()` 方法用于读取单个配置
  - 确认配置更新时已有验证码配置热重载逻辑
- **验证码配置端点** - `internal/api/v1/captcha_handler.go` (verified via Read tool)
  - 确认公共配置端点模式（`/system/auth/captcha/config` 无需认证）
  - 确认返回格式解析后的布尔值（`string(config.Enabled)`）
- **前端验证码配置获取** - `src/services/captcha.ts` (verified via Read tool)
  - 确认前端动态配置获取模式（`getCaptchaConfig()`）
  - 确认使用 `post()` 方法调用公共端点

### Secondary (MEDIUM confidence)
- **前端加密实现** - `src/lib/api.ts` (verified via Read tool)
  - 确认当前使用构建时环境变量 `VITE_ENABLE_REQUEST_ENCRYPTION`
  - 确认存在加密黑名单和加密判断逻辑
- **前端登录流程** - `src/pages/login/index.tsx` (verified via Read tool)
  - 确认登录前获取验证码配置的模式
  - 确认可复用相同模式获取加密配置
- **数据库迁移** - `internal/core/db/migrations/migration_086_request_encryption_toggle.go` (verified via Read tool)
  - 确认加密配置已存在于数据库（`sys.request.encryption.enabled`）

### Tertiary (LOW confidence)
- **前端环境配置** - `xingran-react-frontend/.env.development` (verified via Read tool)
  - 确认当前 `VITE_ENABLE_REQUEST_ENCRYPTION=true`
  - 确认可通过环境变量回退到构建时配置

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有依赖均为项目现有技术栈，无需额外安装
- Architecture: HIGH - 后端已有完善的配置缓存机制，前端已有动态配置模式
- Pitfalls: HIGH - 所有风险点均基于现有代码分析，有明确缓解措施

**Research date:** 2026-05-20
**Valid until:** 30 days (stable architecture, no external dependencies)
