# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XingRan-Next is a modern enterprise permission management system based on the XingRan framework. It uses a Go backend + React frontend architecture with national cryptography (SM2/SM3/SM4) for security.

**Tech Stack:**
- Backend: Go 1.24, Gin, GORM, PostgreSQL 18, Redis 7.4
- Frontend: React 19.2, TypeScript 5.9, Vite 7.2, Ant Design 6.1, Zustand 5.0, Three.js (3D visualization)
- Security: SM2/SM3/SM4 national cryptography algorithms

**Important Files:**
- 详细架构文档: `docs/architecture/项目概述和架构设计.md`
- 开发规范: `docs/standards/开发规范.md`
- API响应规范: `docs/standards/API响应规范.md`
- 安全设计: `docs/architecture/安全和认证设计（国密）.md`
- 数据库设计: `docs/architecture/数据库设计.md`

---

## Common Commands

### Backend (Go)

```bash
# First-time setup: create config from dev template, then edit DB/Redis/SM4_KEY
cp configs/config.dev.yaml configs/config.yaml

# Run directly (no build) — server listens on :9000
go run ./cmd/main.go

# Build for Windows
go build -o xingran-backend.exe ./cmd/main.go

# Build for Linux (production)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o xingran-backend-linux ./cmd/main.go

# Run tests
go test ./...
go test ./internal/api/v1/

# Run backend
.\xingran-backend.exe
```

### Frontend (React) — requires Node.js 24+

```bash
cd xingran-react-frontend

# Install dependencies
npm install

# Development server (runs on http://localhost:4000)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Lint code
npm run lint
npm run lint:fix     # Auto-fix lint issues

# Type checking
npm run type-check
npm run type-check:strict

# Testing
npm run test
npm run test:ui      # Interactive test UI
npm run test:coverage
```

### Configuration

- Backend config: `configs/config.yaml`, `configs/config.dev.yaml`, `configs/config.prod.yaml`
- Frontend env: `xingran-react-frontend/.env.development`
- Environment variables: `.env` (DB_HOST, DB_PASSWORD, REDIS_URL, etc.)

### Swagger Documentation

```bash
# Generate Swagger docs
./scripts/build/generate_swagger.bat   # Windows
./scripts/build/generate_swagger.sh    # Linux/Mac

# Access Swagger UI after starting server
# http://localhost:9000/swagger/index.html
```

**Important Config Values:**
- `server.mode`: "debug" (dev) or "release" (production)
- `database.type`: "postgres" or "sqlite"
- `cache.prefix`: "xingran" (used for all Redis keys, see Cache System section)
- `baidu.map_ak`: Baidu Maps API key for geocoding (read from env or config)
- `security.request_encryption.exclude_paths`: Endpoints that skip SM2+SM4 encryption

**Environment Variables:**
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=xingran
DB_PASSWORD=your_password
DB_NAME=xingran_next

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=

# Baidu Maps (for geocoding)
BAIDU_MAP_AK=your_api_key_here

# National crypto — SM4 key (generate via: openssl rand -base64 16)
SM4_KEY=
```

---

## Architecture

### Backend Layer Structure

```
internal/
├── api/v1/          # HTTP handlers organized by module
│   ├── system/      # User, Role, Dept, Menu, Dict, Post, Config, Notice, etc.
│   ├── operations/  # Building, Floor, Workstation, ServerRoom, etc.
│   ├── scheduler/   # Job management (JobService, JobLogService)
│   ├── workorder/   # Work order management
│   ├── duty/        # Duty roster and scheduling
│   ├── network/     # Network device management
│   ├── knowledge/   # Knowledge base
│   └── monitor/     # Server monitoring, cache monitoring, logs
├── services/        # Business logic layer (modularized)
│   ├── system/      # System services with cache implementations
│   ├── operations/  # Operations management services
│   ├── scheduler/   # Job scheduling services
│   ├── workorder/   # Work order services
│   ├── monitor/     # Monitoring services
│   └── *_cache_service.go  # Legacy cache services (dept, role, dict, menu, user, post)
├── models/          # Data models (GORM)
├── core/            # Core modules (DB, Cache, JWT, SM4, Device, Scheduler)
├── config/          # Configuration management
├── device/          # Device management (Scrapli, TextFSM)
├── collectors/      # Data collectors
├── scheduler/       # Cron job scheduler (internal, not to confuse with api/v1/scheduler)
├── templates/       # Template parsing
├── utils/           # Utilities
└── websocket/       # WebSocket service

pkg/                 # Public packages (reusable)
├── cache/           # Redis + Memory cache interface
├── crypto/          # SM2/SM4 encryption
├── middleware/      # Auth, CORS, logging, encryption
├── permission/      # RBAC permission control
├── query/           # Query builders
└── response/        # Response wrappers
```

**Key Architecture Patterns:**

1. **Handler-Service Pattern (Completed Migration)**
   - All handlers use struct-based pattern with dependency injection
   - Services defined as interfaces with private implementations
   - Example:
     ```go
     // Handler in internal/api/v1/system/
     type UserHandler struct {
         userService system.UserService
         pwdManager  PasswordManager
     }

     // Service in internal/services/system/
     type UserService interface {
         CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
     }
     ```

2. **Dual Cache Architecture**
   - **Legacy**: Root-level `*_cache_service.go` files (dept, role, dict, menu, user, post)
     - Used by `core.Core` for backward compatibility
     - Wrap `DataCacheService` with business logic
   - **New**: `internal/services/system/*_cache_impl.go`
     - Use `CacheProvider` interface for decoupling
     - Support both cached and non-cached implementations

3. **Middleware Chain**: Auth → Permission → Encryption → Handler

4. **Dual Token System**: Access Token (short-lived) + Refresh Token (long-lived)

5. **Request Encryption**: SM2+SM4 hybrid encryption for sensitive endpoints

### Frontend Structure

```
src/
├── pages/           # Route pages (organized by module)
│   ├── system/      # User, Role, Dept, Menu, Dict, Post, Config, Notice
│   ├── operations/  # Building, Floor, Workstation, ServerRoom, DedicatedLine
│   ├── monitor/     # Server monitoring, cache monitoring
│   └── ...
├── components/      # Reusable components
│   ├── layout/      # Layout components (HybridLayout, Sidebar, Header)
│   ├── three/       # 3D visualization components (Three.js)
│   └── ...
├── store/           # Zustand state stores (7 stores)
├── hooks/           # Custom React hooks
├── lib/             # API clients
│   ├── api.ts       # Core API client with encryption and token refresh
│   └── opsApi.ts    # Operations module CRUD API factory
├── utils/           # Utilities (sm2.ts, sm4.ts, authHelpers.ts, etc.)
├── types/           # TypeScript definitions
└── design-system/   # Design tokens and theme
```

**State Management (Zustand):**
- `authStore` - Authentication state (includes TokenManager for auto-refresh)
- `layoutStore` - Layout preferences
- `menuStore` - Menu navigation
- `noticeStore` - Notifications
- `settingsStore` - User settings
- `tabsStore` - Tab navigation
- `themeStore` - Theme (light/dark)

**3D Visualization (Three.js):**
- Location: `src/components/three/`, floor plan editor components
- Dependencies: `@react-three/fiber`, `@react-three/drei`, `three`
- Usage: Building/floor 3D visualization, CAD-style floor plan editor

---

## Critical Development Rules

### 操作日志记录约定 (operlog convention) — 强制

**规则:** 所有业务写操作（POST 创建 / 更新 / 删除 / 状态变更 / 导入导出 / 同步 / 批量等）handler 必须在 success path 末尾、`response.Success(...)` **之前**调用 `operlog.Record(...)` 记录操作日志。该约定由 Phase 34（操作日志全模块集成）落地，用于满足审计与可追溯性要求（参考 `.planning/notes/260615-oper-log-coverage-audit.md`）。

**helper 包路径:** `github.com/xingran-next/xingran-go-backend/internal/utils/operlog`

**调用模式（非敏感端点）:**

```go
import "github.com/xingran-next/xingran-go-backend/internal/utils/operlog"

// success path 末尾，response.Success 之前
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeCreate)
response.Success(c, user)
```

**敏感端点（密码 / 密钥 / token / 凭据 / SMTP password / API headers）使用 `RecordWithBody`，自动读取并恢复请求体，对 `password / pwd / secret / token / key / salt / privateKey / oldPassword / macKey / sm4Key / sm2Key / adminPassword / clientSecret / accessKey / secretKey / private_key / publicKey` 等关键词的值脱敏为 `******`:**

```go
operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeCreate)
```

**OperType 业务类型常量集（共 25 个，值固定不可重排）:**

| 常量 | 值 | 语义 |
|------|----|----|
| `OperTypeOther` | 0 | 其他 |
| `OperTypeCreate` | 1 | 新增 |
| `OperTypeUpdate` | 2 | 修改 |
| `OperTypeDelete` | 3 | 删除 |
| `OperTypeGrant` | 4 | 授权 |
| `OperTypeExport` | 5 | 导出 |
| `OperTypeImport` | 6 | 导入 |
| `OperTypeForce` | 7 | 强退 |
| `OperTypeGenCode` | 8 | 生成代码 |
| `OperTypeClean` | 9 | 清空数据 |
| `OperTypeStatus` | 10 | 状态变更（启用/停用） |
| `OperTypeReset` | 11 | 密码/密钥重置 |
| `OperTypeEnable` | 12 | 启用 |
| `OperTypeDisable` | 13 | 停用 |
| `OperTypeSync` | 14 | LDAP/VDI/资产/外部同步 |
| `OperTypeMove` | 15 | 移动 OU/部门/虚拟机 |
| `OperTypeBatch` | 16 | 批量新增/删除 |
| `OperTypeUpload` | 17 | 文件上传 |
| `OperTypeDownload` | 18 | 文件/模板下载 |
| `OperTypeLogin` | 19 | 登录 |
| `OperTypeLogout` | 20 | 登出 |
| `OperTypeRegister` | 21 | Agent/虚拟机注册 |
| `OperTypeApprove` | 22 | 审批通过 |
| `OperTypeReject` | 23 | 审批驳回 |
| `OperTypeUnlock` | 24 | 账号解锁（管理员手动解锁被锁定的用户） |

**module 中文名称规范:** 用户管理 / 角色管理 / 部门管理 / 菜单管理 / 通知公告 / API密钥管理 / 参数管理 / 字典管理 / 岗位管理 / 楼宇管理 / 楼层管理 / 工位管理 / 机房管理 / 资产管理 / 专线管理 / 信息点 / 网络设备 / 设备凭据 / 命令模板 / 拓扑 / MAC地址 / 端口采集 / 虚拟机管理 / VDI服务器 / 工单管理 / 值班池 / 知识库文章 / 定时任务 / 缓存监控 / 操作日志 / Agent管理 / RPA任务 / 仪表盘 / 列配置 / 通知配置 / AD域控同步 等。

**参考实现:** `internal/api/v1/system/ad_domain_handler.go`（Phase 34 之前的唯一参考，现已成为全模块通用约定）。

**回归守护:** `internal/utils/operlog/regression_test.go` 通过 4 个测试锁定公共 API（常量值 / 常量数=25 / Record 5参+可变参 / 11 个强制敏感关键词），任何静默改动都会立即失败。

### Status Value Convention (IMPORTANT)

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

**Exception - Menu Visibility:** `1 = visible, 0 = hidden` (boolean semantics; see `models.VisibleShow` / `models.VisibleHidden`)

**Source of truth (do not hard-code values):**

1. **状态常量唯一真相源** — `internal/models/` (e.g. `internal/models/base.go`)。所有模块的 `status` / `visible` / 业务三态字段都以具名常量引用(如 `models.UserStatusEnabled = 0`、`models.VisibleShow = 1`、`models.WorkstationStatus*`)。
2. **常量值由回归测试锁定** — `internal/models/status_constants_test.go` AST 扫描所有 `models.XxxStatus*` / `Visible*` 字面量,任何静默改动(包括 0/1 调换、跨包同名异值)立即测试失败。修改值先改测试再改常量。
3. **运营可维护枚举(type / category / 业务选项)真相源** — `sys_dict_type` / `sys_dict_data`,通过字典管理页维护。Seed 见 `internal/core/db/migrations/migration_208_dict_seed.go`(sqlite / postgres 双分支均注册)。
4. **前端通用启停选项共享常量** — `xingran-react-frontend/src/constants/status.ts`(ENABLE_DISABLE / NORMAL_STOP / 三态工作组等),不再每页重复 `[{value: 0, label: '启用'}, ...]`。
5. **status 0/1 不入字典** — 通用规则的 0/1 是代码分支语义(管理员可配值会破坏 `if status == 0` 逻辑);type / category / 业务选项等可选项才走 sys_dict。**Menu visible 例外**仅保留文档提示,不破坏普适规则。

新增 status / visible 常量:先在 `internal/models/<file>.go` 命名常量 → 同步 `status_constants_test.go` 期望表 → 业务代码按常量引用。

### API Response Format

All API responses follow this structure:

```json
{
    "code": 0,           // 0 = success, other = error
    "message": "success",
    "data": {},          // Business data
    "timestamp": 1766380800,
    "request_id": "uuid-string"
}
```

**Success Code:** `0`
**Common Error Codes:** `400` (param error), `401` (unauthorized), `403` (forbidden), `1001` (invalid params), `1007` (invalid token)

### Frontend API Calling

**Use wrapped API functions, NOT raw axios:**

```typescript
// ✅ CORRECT - use wrapped functions
import { post } from '@/lib/api';
const result = await post('/system/users/list', params);
const users = result.data.list; // Direct access

// ❌ WRONG - don't use api instance directly
const response = await api.post('/system/users/list', params);
if (response.data.code === 0) { } // Manual check unnecessary
```

**For operations module (楼宇/楼层/工位/机房), use opsApi.ts:**

```typescript
// ✅ CORRECT - use opsApi for CRUD operations
import { buildingApi, floorApi, workstationApi, excelApi } from '@/lib/opsApi';

// List buildings with pagination
const result = await buildingApi.list({ current: 1, pageSize: 10, name: '测试' });
const buildings = result.data.list;

// Create/update/delete
await buildingApi.create({ name: 'New Building', ... });
await buildingApi.update(id, { name: 'Updated Name' });
await buildingApi.delete(id);

// Excel operations
await excelApi.downloadTemplate('building');
await excelApi.import('building', file);
await excelApi.export('building', { status: 0 });

// ❌ WRONG - don't use raw post for operations module
await post('/ops/building/list', params); // Use buildingApi.list instead
```

**Token management with authHelpers:**

```typescript
// ✅ CORRECT - use authHelpers for token access
import { getAccessToken } from '@/utils/authHelpers';
const token = await getAccessToken();

// For Excel import component
import { getAuthHeaders } from '@/utils/authHelpers';
const headers = await getAuthHeaders();
```

### Pagination Convention

**Request:**
```typescript
interface PageParams {
    current: number;    // Page number, starts from 1
    pageSize: number;
}
```

**Response:**
```typescript
interface PageData<T> {
    list: T[];
    total: number;
    current: number;
    pageSize: number;
}
```

### Database Naming

- Tables: `sys_` prefix, lowercase with underscores
- Fields: lowercase with underscores
- Primary Keys: UUID with `gen_random_uuid()` default
- Timestamps: `created_at`, `updated_at`, `deleted_at` (soft delete)

### Go Code Patterns

**Service Interface Pattern (Standard):**
```go
// internal/services/system/user_service.go
type UserService interface {
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    UpdateUser(ctx context.Context, id string, req *UpdateUserRequest) error
}

// Private implementation
type userServiceImpl struct {
    db         *gorm.DB
    cache      CacheProvider
    pwdManager PasswordManager
}

func NewUserService(db *gorm.DB, cache CacheProvider, pwdMgr PasswordManager) UserService {
    return &userServiceImpl{db: db, cache: cache, pwdManager: pwdMgr}
}
```

**Handler Pattern (Standard):**
```go
// internal/api/v1/system/user_handler.go
type UserHandler struct {
    userService system.UserService
    pwdManager  PasswordManager
}

func NewUserHandler(userSvc system.UserService, pwdMgr PasswordManager) *UserHandler {
    return &UserHandler{userService: userSvc, pwdManager: pwdMgr}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    var req system.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }

    user, err := h.userService.CreateUser(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    response.Success(c, user)
}
```

**Router Pattern:**
```go
// internal/api/v1/system/user_router.go
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
    userService := system.NewUserService(core.GetDB(), core.Cache, core.PwdManager)
    userHandler := system.NewUserHandler(userService, core.PwdManager)

    r.POST("/list", userHandler.List)
    r.POST("", userHandler.Create)
    r.POST("/:id", userHandler.GetByID)
    r.POST("/:id/update", userHandler.Update)
    r.POST("/:id/delete", userHandler.Delete)
}
```

**Cache Key Helper Functions (in services root):**
```go
// internal/services/cache_keys.go (or defined in cache_config_service.go)
const CacheKeyDictType = "dict:type:all"
const CacheKeyDictDataByType = "dict:data"

func GetDictDataByTypeKey(dictType string) string {
    return fmt.Sprintf("%s:%s", CacheKeyDictDataByType, dictType)
}
```

---

## Security Considerations

### Request Encryption (SM2+SM4)

- **Enabled for sensitive endpoints** (configurable via `security.request_encryption.exclude_paths`; login is NOT excluded)
- **SM4-CBC** encrypts request body
- **SM2** encrypts SM4 key
- **Anti-replay:** timestamp (300s window) + nonce
- Frontend toggle: `VITE_ENABLE_REQUEST_ENCRYPTION`

### Response Encryption (default disabled)

- `security.response_encryption.enabled` controls SM4-CBC response body encryption
- Dynamically toggled at runtime via param `sys.request.encryption.enabled` (not just the YAML flag)

### SM4 Key (`security.sm4_key`)

- Base64-encoded 16-byte key; encrypts device passwords, AD passwords, RPA credentials
- Generate a fresh key for production: `export SM4_KEY="$(openssl rand -base64 16)"`

### LDAP/AD Connections

**CRITICAL:** Current code uses `InsecureSkipVerify: true` for LDAPS/StartTLS. This should be addressed for production.

### Authentication

- JWT dual-token mechanism (config: `jwt.access_key_expire` 7200s, `jwt.refresh_key_expire` 604800s, `jwt.use_sm2`)
- SM3 for password hashing
- RBAC permission model
- All API endpoints require auth (except configured exclusions)

---

## Module-Specific Notes

### Network Device Management

- **Scrapli** for device connections (Python-based, invoked via subprocess)
- **TextFSM** templates for command parsing
- **Template cache** for performance
- **Port collection** with parser abstraction
- **Location**: `internal/device/`, `internal/services/portcollection/`

### AD Domain Management

- **LDAP client** with connection pooling
- **Sync tasks** via scheduler
- **Modular services**: config, group, user, OU, sync
- **Location**: `internal/services/addomain/`
- **Warning**: Uses `InsecureSkipVerify: true` for LDAPS/StartTLS

### AD Service Account Pool (Phase 36)

**解决 AD 单管理员账号被锁导致所有用户登录失败**（data 775 锁定事件）。

**核心组件**：

- **AccountPool** (`internal/services/addomain/account_pool.go`)
  - 多账号池（`sys_ad_service_accounts` 表）
  - 状态机：`0=可用 / 1=停用 / 2=熔断`
  - 自动熔断：连续 3 次失败 → 30 分钟熔断
  - 行锁 + 事务保证并发安全（`SELECT FOR UPDATE`）
  - `RecoverExpiredBreakers`：cron 每 5 分钟恢复过期熔断

- **FailoverClient** (`internal/services/addomain/failover_client.go`)
  - 顺序遍历 `ListAvailable` 快照
  - 任一账号成功即返回
  - `maxHops = min(DefaultMaxHops=10, len(available))`

- **LDAPClient** (`internal/services/addomain/ldap_client.go`)
  - 可变参数接收 `*ADServiceAccount`
  - `tryBindAttempts` 优先用 `c.account` 凭证（核心！）

- **API**: 8 个 POST 端点（`internal/api/v1/system/ad_account_handler.go`）
  - 权限粒度：list/stats → `list`；create/update/enable/disable/unlock → `edit`；delete → `delete`

- **前端**: `src/pages/ad-domain/accounts/index.tsx`（独立页面，统计卡片 + 列表 + Modal）

- **路由**: `POST /system/ad-config/accounts/{list,create,update,delete,enable,disable,unlock,stats}`

**关键设计决策**：

| 决策 | 说明 |
|------|------|
| 字段名 `password_ciphertext` | 含 `password` 关键词 → operlog 自动脱敏（OPERLOG-03 兼容） |
| 随机轮询 | FailoverClient 用 `ListAvailable` 快照顺序遍历（避免 random pick + 防重） |
| 无冷却期 | 失败账号不再被排除（避免池子饥饿） |
| ManualUnlock | service 层校验 `reason ≥10 字符` + `operator 非空`（安全 invariant） |
| Cron 注册 | `internal/scheduler/ad_sync_tasks.go` `StartADSyncScheduler` |
| 双读兼容期 | `sys_ad_config.admin_username/password` 字段保留并标 @Deprecated |

### Workorder Management

- **Assignment logic** with rotation
- **Periodic workorders** with Cron scheduling
- **Statistics** and rating system
- **Location**: `internal/services/workorder/`

### Cache System

**Two-tier architecture:**

1. **pkg/cache/** - Low-level cache interface
   - `Cache` interface with Get/Set/Delete methods
   - Redis implementation in `redis.go`
   - **Important**: Redis uses prefix `xingran:` for all keys
   - When calling cache methods, use keys WITHOUT prefix, the prefix is added automatically

2. **services/** - High-level caching
   - `DataCacheService` - Generic caching with JSON serialization
   - `CacheConfigService` - Dynamic cache TTL configuration
   - Module-specific cache services (dept, role, dict, menu, user, post)

**Cache key patterns:**
- Use helper functions like `GetDictDataByTypeKey(dictType)` instead of hardcoding
- Constants defined in root-level `*_cache_service.go` files

**CRITICAL: Cache Key Prefix Handling**
- Redis prefix is set to `xingran` in `internal/core/core.go:342`
- When storing: `cache.Set(ctx, "user:1", value)` → actual Redis key: `xingran:user:1`
- When user provides key with prefix in cache monitor operations, always strip it first:
  ```go
  if strings.HasPrefix(key, "xingran:") {
      key = key[6:]  // Remove prefix before operations
  }
  ```

### Operations Management (楼宇/楼层/工位管理)

- **Excel Import/Export**: Batch import with geocoding support
  - Location: `internal/services/operations/excel_service.go`
  - Config: `internal/services/operations/excel_config.go`
  - Supports: Building, Floor, Workstation, InfoPoint
  - Auto-features: Geocoding (address → coordinates), department matching, cache invalidation

- **Geocoding Service**: Baidu Maps integration for address resolution
  - Location: `internal/services/operations/geocoding_service.go`
  - API: `POST /ops/building/geocode`
  - Purpose: Solve CORS issues when calling Baidu Maps from frontend
  - In-memory caching: 30 minutes TTL
  - Concurrency limit: 5 parallel requests

- **Building Org ID Validation**:
  - **CRITICAL**: `org_id` must be a valid UUID, NOT a department name
  - Validation happens in `building_service.go` `validateOrg()` function
  - Pattern: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
  - Migration 084 fixes existing data with department names instead of UUIDs

### Scheduler System

- **Internal scheduler**: `internal/scheduler/` - Cron job execution engine
- **API module**: `internal/api/v1/scheduler/` - Job management endpoints
- **Service layer**: `internal/services/scheduler/` - Job business logic
- **DO NOT confuse**: `internal/scheduler/` (engine) vs `api/v1/scheduler/` (job CRUD UI)

---

## Key File Locations

**Backend Core:**
- Entry: `cmd/main.go`
- Main router: `internal/api/router.go`
- Core initialization: `internal/core/core.go`
- Models base: `internal/models/base.go`

**Security & Cache:**
- Request encryption: `pkg/crypto/request_encryption.go`
- Cache interface: `pkg/cache/redis.go`
- Data cache service: `internal/services/data_cache_service.go`
- Cache config: `internal/services/cache_config_service.go`

**Authentication & Permissions:**
- JWT manager: `internal/core/security/jwt.go`
- Password manager: `internal/core/security/password.go`
- Permission middleware: `pkg/middleware/permission.go`
- Permission definitions: `pkg/permission/permissions.go`

**Frontend:**
- Entry: `xingran-react-frontend/src/main.tsx`
- Router: `xingran-react-frontend/src/App.tsx`
- API wrapper: `xingran-react-frontend/src/lib/api.ts`
- Crypto utils: `xingran-react-frontend/src/utils/sm2.ts`, `sm4.ts`
- Stores: `xingran-react-frontend/src/store/`

**Documentation:**
- Architecture: `docs/architecture/项目概述和架构设计.md`
- Standards: `docs/standards/开发规范.md`
- API: `docs/standards/API响应规范.md`
- Security: `docs/architecture/安全和认证设计（国密）.md`

**Legacy Services (still used by core):**
- `internal/services/dept_service.go`
- `internal/services/role_cache_service.go`
- `internal/services/dict_cache_service.go`
- `internal/services/menu_cache_service.go`
- `internal/services/user_cache_service.go`
- `internal/services/post_cache_service.go`

---

## Development Workflow

### Adding a New Module

1. **Create Service Layer** (`internal/services/<module>/`)
   - Define interface: `type XxxService interface { ... }`
   - Create private implementation: `type xxxServiceImpl struct { ... }`
   - Add constructor: `func NewXxxService(...) XxxService`

2. **Create Handler** (`internal/api/v1/<module>/xxx_handler.go`)
   - Define struct with service dependencies
   - Add constructor: `func NewXxxHandler(...) *XxxHandler`
   - Implement handler methods with context propagation

3. **Create Router** (`internal/api/v1/<module>/xxx_router.go`)
   - Define `SetupXxxRouter(r *gin.RouterGroup, core *core.Core)`
   - Initialize service and handler
   - Register routes with middleware

4. **Register in Main Router** (`internal/api/router.go`)
   - Import new module's router
   - Call `SetupXxxRouter()` in appropriate group

### Working with Cache

- **New code**: Use `internal/services/system/` pattern with `CacheProvider` interface
- **Existing code**: May use legacy `*_cache_service.go` files in root
- **Cache keys**: Use helper functions, don't hardcode strings
- **Invalidation**: Call `Invalidate*Cache()` methods after mutations

### Common Gotchas

- **Scheduler confusion**: `internal/scheduler/` (engine) ≠ `api/v1/scheduler/` (job management)
- **Cache dual architecture**: Legacy root files vs new `system/` implementations
- **Import paths**: Use full module path `github.com/xingran-next/xingran-go-backend/...`
- **Context propagation**: Always pass `c.Request.Context()` to service methods
- **Response wrapper**: Use `response.Success()` and `response.Error()`, not raw JSON
- **Cache prefix confusion**: User input may include `xingran:` prefix from UI, always strip before cache operations
- **UUID validation**: When validating org_id, check format first to avoid PostgreSQL type errors
- **Frontend API calls**: Use wrapped functions in `@/lib/api.ts`, NOT raw axios
- **Operations module API**: Use `@/lib/opsApi.ts` for building/floor/workstation CRUD
- **Token refresh**: Frontend uses TokenManager in authStore for automatic token refresh
- **Request encryption**: Frontend encrypts POST/PUT/PATCH requests with SM2+SM4 (configurable via `VITE_ENABLE_REQUEST_ENCRYPTION`)
- **Temporary files**: Root-level `temp_*.go` and `test_*.go` files cause `main redeclared` errors - delete them before building

### Testing

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/services/operations/

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -v -run TestGeocode ./internal/services/operations/
```

### Running Single Test

```bash
# Navigate to the package directory first
cd internal/services/operations

# Run specific test function
go test -v -run TestBatchUpsert

# Run with verbose output
go test -v .
```

### Running Frontend Tests

```bash
cd xingran-react-frontend

# Run all tests
npm run test

# Run specific test file
npx vitest run src/utils/errorHandler.test.ts

# Run in watch mode
npm run test -- --watch
```

### Building

```bash
# Development build (Windows)
go build -o xingran-backend.exe ./cmd/main.go

# Production build (Linux)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o xingran-backend-linux ./cmd/main.go
```

### Database Migrations

Migrations are in `internal/core/db/migrations/`. They run automatically on application startup.

To run migrations manually:
```bash
# Check migration status
# Migrations are versioned and auto-applied via gorm-auto-migrate
# Check `internal/core/db/migrations/` for available migrations
```

### Compilation & Build Verification

**CRITICAL:** Always run `go build ./...` after modifying Go files to catch compilation errors immediately. Fix errors iteratively rather than attempting bulk fixes across multiple files.

```bash
# After any Go code changes, run full build check
go build ./...

# If errors occur, fix one file completely, then re-build
# Do NOT attempt to fix multiple files at once
```

This prevents cascading compilation errors and makes it easier to identify which changes actually matter.

### Temporary Files Cleanup

**IMPORTANT:** Temporary test files in the project root (e.g., `temp_*.go`, `test_import.go`) can cause compilation errors due to multiple `main` function declarations. These files should be deleted or moved outside the project:

```bash
# Check for problematic temporary files
ls temp_*.go test_*.go 2>/dev/null

# Remove them if they exist and are not needed
rm -f temp_*.go test_import.go
```

These files are typically created for debugging and should not be committed to version control.

---

## Frontend/React Best Practices

### useEffect Dependencies

**CRITICAL:** For TypeScript React components, ensure useEffect dependencies are stable (use useMemo/useCallback) to prevent infinite API request loops. Common pattern: objects/arrays in useEffect cause re-renders.

```typescript
// ❌ WRONG - unstable object reference causes infinite loop
useEffect(() => {
  fetchData({ page: 1, size: 10 });
}, [{ page: 1, size: 10 }]); // Object recreated on every render

// ✅ CORRECT - memoize the params object
const params = useMemo(() => ({ page: 1, size: 10 }), []);
useEffect(() => {
  fetchData(params);
}, [params]);

// ✅ CORRECT - use primitive values
useEffect(() => {
  fetchData({ page, pageSize });
}, [page, pageSize]); // Primitives are stable
```

**Before deploying React changes, check all useEffect hooks:**
1. Are objects/arrays in dependencies memoized?
2. Could any dependency change on every render?
3. Add temporary `console.log('effect triggered')` to verify effect doesn't run excessively.

---

## Debugging & Bug Fixing

### Scope Constrainment

**CRITICAL:** When fixing bugs, start with the specific reported issue before expanding scope. Do not proactively 'fix' unreported issues across other modules unless explicitly requested.

1. Describe exact symptom and expected behavior
2. Specify "fix only this issue, don't touch other files"
3. After verification, explicitly ask for refactor if needed
4. Review each file changed and ask "was this change necessary?"

This prevents over-engineering and reduces the risk of introducing new bugs while fixing unrelated issues.

### Autonomous Debugging Protocol

For systematic bug investigation and resolution, follow this autonomous workflow:

**Template prompt:**

```
Investigate and fix the bug described as [symptom/error message]. Follow this autonomous debugging protocol:
1) Use Grep to search codebase for related error patterns and log outputs
2) Identify the root cause by examining stack traces and code flow
3) Create a minimal reproduction case using the actual application (use chrome-devtools click tool if it's a UI bug)
4) Implement a targeted fix that addresses the root cause without touching unrelated code
5) Run existing tests to verify the fix doesn't break anything
6) If tests fail, iteratively refine the fix until all tests pass
7) Summarize the root cause, fix applied, and test results

Do not ask for confirmation during this process—proceed autonomously and report results when complete.
```

**Key principles:**
- Search for patterns first, don't guess
- Reproduce the issue before fixing
- Fix only the root cause, not symptoms
- Validate with tests, not assumptions
- Report comprehensive summary at the end

---

## Git Workflow

### Commit Process

**CRITICAL:** Before making git commits, ask for explicit user confirmation. Do not autonomously commit uncommitted changes.

1. **Before committing any changes:**
   - `go build ./...` (or `npm run build`)
   - `go test ./...` (or `npm test`)
   - Check for any console warnings
   - Only if all pass, then proceed with git operations

2. **Ask for confirmation:**
   - Show the diff of changes
   - Ask "Should I commit these changes?"
   - Wait for explicit user approval before running `git commit`

---

## Using Agents for Codebase Exploration

For complex multi-file changes, use the Task tool with specialized agents to systematically explore dependencies and related code before making changes.

**When to use agents:**
- Making schema changes that affect multiple files
- Refactoring across modules
- Finding all usages of a specific pattern
- Exploring unfamiliar code areas

**Example prompts:**

```
# Before modifying a model
"Use an agent to explore all files that import the Computer model before making schema changes"

# Before refactoring configuration
"Use an agent to find all places where Excel templates are configured"

# For API changes
"Use an agent to find all handlers and services that use the UserService interface"
```

This prevents unintended breakage and ensures comprehensive understanding of impact scope.

### Parallel Development Approach

For complex full-stack features, use a parallel development approach to work on multiple layers simultaneously:

**Template prompt:**

```
Implement the [feature name] using a parallel development approach. Break down the work into 4 independent tracks:
1) Backend: API endpoints and business logic
2) Database: migrations and models
3) Frontend: UI components and forms
4) Testing: unit and integration tests

Work on these tracks simultaneously, using Write to create stub interfaces for dependencies. Every 10 minutes, sync progress by checking if any track is blocked on another. When all tracks are complete, run a full integration test. Use Glob to find all related files first, then create a parallel execution plan in TodoWrite before starting.
```

**Benefits:**
- Reduces feature delivery time by working on independent tracks in parallel
- Identifies dependencies and blocking issues early through periodic syncs
- Ensures comprehensive coverage across all architectural layers

### Test-Driven Refactoring Pipeline

For code quality improvements while maintaining test coverage, use this autonomous refactoring workflow:

**Template prompt:**

```
I want you to autonomously refactor the [module/component] to improve code quality while maintaining 100% test coverage. Start by running the full test suite and capturing baseline results. Then:
1) Identify 3-5 refactoring opportunities through static analysis
2) Implement changes incrementally (one improvement at a time)
3) After each change, immediately run the relevant tests
4) If tests fail, analyze the failure and either fix the refactor or update the test if the behavior change is intentional
5) Continue until all improvements are complete

Use TodoWrite to track progress. Ask for my approval only before starting and for final review.
```

**Key principles:**
- Establish baseline test coverage before any changes
- Make incremental improvements, not bulk refactors
- Test after every single change
- Fail fast - fix issues immediately
- Continuous verification prevents regression debt

<!-- GSD:project-start source:PROJECT.md -->
## Project

**工位导入部门/用户关联功能**

为 XingRan-Next 运维管理系统的工位（Workstation）Excel 导入功能添加"所属部门"和"所属用户"两个可选字段。导入时通过部门名称匹配 `sys_dept`、通过用户名匹配 `sys_user`，建立工位与部门、用户的关联关系，并在前端展示。

**Core Value:** 工位导入时能自动关联部门和用户，避免手动逐条配置，提升批量管理效率。

### Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有工位数据和导入流程
- **UUID Foreign Keys**: `org_id` 和 `user_id` 必须是有效 UUID，参考 `building_service.go` 的 `validateOrg()` 模式
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装
<!-- GSD:project-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture (reference)

### Core Dependency Injection

```go
// Core struct holds all core dependencies
type Core struct {
    Config     *config.Config
    DB         *db.Database
    Cache      cache.Cache
    JWTManager *security.JWTManager
    PwdManager *security.PasswordManager
    // ... other services
}

// Router setup with Core DI
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
    userService := system.NewUserService(core.GetDB(), core.Cache, core.PwdManager)
    userHandler := system.NewUserHandler(userService, core.PwdManager)
    // ...
}
```

### Data Flow

```
# Request Flow
Client Request → Gin Router → Auth Middleware → Permission Middleware
→ Encryption Middleware → Handler → Service → Repository/Cache → Database
→ Response Wrapper → JSON Response

# Cache Flow
Service.GetOrSet() → CacheProvider.GetOrSet()
→ Check L1 (in-memory) → Hit? Return
→ Check L2 (Redis) → Hit? Return & populate L1
→ Execute query() → Store to L2 → Store to L1 → Return

# Import Flow (Workstation/Building)
Excel Upload → ExcelService.Parse() → Validate rows
→ ReferenceResolver.ResolveDept() → ReferenceResolver.ResolveUser()
→ GeocodingService.Geocode() (optional) → BatchUpsert()
→ CacheInvalidator.Invalidate() → Return ImportResult
```

### CacheProvider Interface

```go
type CacheProvider interface {
    // GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
    GetOrSet(ctx context.Context, key string, dest interface{},
        expiration time.Duration, query func() (interface{}, error)) error

    // Delete 删除缓存
    Delete(ctx context.Context, key string) error

    // DeleteByPattern 根据模式删除缓存
    DeleteByPattern(ctx context.Context, pattern string) error

    // MGet/MDelete 批量操作
    MGet(ctx context.Context, keys ...string) (map[string]string, error)
    MDelete(ctx context.Context, keys ...string) error
}
```
- Cached version: reads/writes Redis
- Non-cached version: pass-through to DB

### Module Boundaries

| Module | API Layer | Service Layer | Models |
|--------|-----------|---------------|--------|
| System | `api/v1/system/` | `services/system/` | `models/` |
| Operations | `api/v1/operations/` | `services/operations/` | `models/` |
| Scheduler | `api/v1/scheduler/` | `services/scheduler/` | `models/` |
| Work Order | `api/v1/workorder/` | `services/workorder/` | `models/` |
| Network | `api/v1/network/` | `services/network/` | `models/` |
| Monitor | `api/v1/monitor/` | `services/monitor/` | `models/` |
| Duty | `api/v1/duty/` | `services/duty/` | `models/` |
| Knowledge | `api/v1/knowledge/` | `services/knowledge/` | `models/` |
| AD Domain | — | `services/addomain/` | `models/` |
| Port Collection | — | `services/portcollection/` | — |
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
