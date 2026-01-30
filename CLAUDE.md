# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RuoYi-Next is a modern enterprise permission management system based on the RuoYi framework. It uses a Go backend + React frontend architecture with national cryptography (SM2/SM3/SM4) for security.

**Tech Stack:**
- Backend: Go 1.24, Gin, GORM, PostgreSQL 18, Redis 7.4
- Frontend: React 19, TypeScript 5, Vite 7, Ant Design 6, Zustand 5
- Security: SM2/SM3/SM4 national cryptography algorithms

---

## Common Commands

### Backend (Go)

```bash
# Build for Windows
go build -o ruoyi-backend.exe ./cmd/main.go

# Build for Linux (production)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o ruoyi-backend-linux ./cmd/main.go

# Run tests
go test ./...
go test ./internal/api/v1/

# Run backend
.\ruoyi-backend.exe
```

### Frontend (React)

```bash
cd ruoyi-react-frontend

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
```

### Configuration

- Backend config: `configs/config.yaml`, `configs/config.dev.yaml`, `configs/config.prod.yaml`
- Frontend env: `ruoyi-react-frontend/.env.development`
- Environment variables: `.env` (DB_HOST, DB_PASSWORD, REDIS_URL, etc.)

**Important Config Values:**
- `server.mode`: "debug" (dev) or "release" (production)
- `database.type`: "postgres" or "sqlite"
- `cache.prefix`: "ruoyi" (used for all Redis keys, see Cache System section)
- `baidu.map_ak`: Baidu Maps API key for geocoding (read from env or config)
- `security.request_encryption.exclude_paths`: Endpoints that skip SM2+SM4 encryption

**Environment Variables:**
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=ruoyi
DB_PASSWORD=your_password
DB_NAME=ruoyi_next

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=

# Baidu Maps (for geocoding)
BAIDU_MAP_AK=your_api_key_here
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
├── components/      # Reusable components
├── store/           # Zustand state stores (7 stores)
├── hooks/           # Custom React hooks
├── services/        # API service layer
├── utils/           # Utilities (sm2.ts, sm4.ts, etc.)
├── types/           # TypeScript definitions
└── design-system/   # Design tokens and theme
```

**State Management (Zustand):**
- `authStore` - Authentication state
- `layoutStore` - Layout preferences
- `menuStore` - Menu navigation
- `noticeStore` - Notifications
- `settingsStore` - User settings
- `tabsStore` - Tab navigation
- `themeStore` - Theme (light/dark)

---

## Critical Development Rules

### Status Value Convention (IMPORTANT)

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

**Exception - Menu Visibility:** `1 = visible, 0 = hidden` (boolean semantics)

| Module | Field | 0 Value | 1 Value | Default |
|--------|-------|---------|---------|---------|
| User | status | Enabled | Disabled | 0 |
| Role | status | Normal | Stopped | 0 |
| Menu | status | Normal | Stopped | 0 |
| Menu | visible | Hidden | Visible | 1 |
| Dept | status | Normal | Stopped | 0 |
| Post | status | Normal | Stopped | 0 |

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

- **Enabled for sensitive endpoints** (configurable via `security.request_encryption.exclude_paths`)
- **SM4-CBC** encrypts request body
- **SM2** encrypts SM4 key
- **Anti-replay:** timestamp (300s window) + nonce

### LDAP/AD Connections

**CRITICAL:** Current code uses `InsecureSkipVerify: true` for LDAPS/StartTLS. This should be addressed for production.

### Authentication

- JWT dual-token mechanism
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
   - **Important**: Redis uses prefix `ruoyi:` for all keys
   - When calling cache methods, use keys WITHOUT prefix, the prefix is added automatically

2. **services/** - High-level caching
   - `DataCacheService` - Generic caching with JSON serialization
   - `CacheConfigService` - Dynamic cache TTL configuration
   - Module-specific cache services (dept, role, dict, menu, user, post)

**Cache key patterns:**
- Use helper functions like `GetDictDataByTypeKey(dictType)` instead of hardcoding
- Constants defined in root-level `*_cache_service.go` files

**CRITICAL: Cache Key Prefix Handling**
- Redis prefix is set to `ruoyi` in `internal/core/core.go:342`
- When storing: `cache.Set(ctx, "user:1", value)` → actual Redis key: `ruoyi:user:1`
- When user provides key with prefix in cache monitor operations, always strip it first:
  ```go
  if strings.HasPrefix(key, "ruoyi:") {
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
- Entry: `ruoyi-react-frontend/src/main.tsx`
- Router: `ruoyi-react-frontend/src/App.tsx`
- API wrapper: `ruoyi-react-frontend/src/lib/api.ts`
- Crypto utils: `ruoyi-react-frontend/src/utils/sm2.ts`, `sm4.ts`
- Stores: `ruoyi-react-frontend/src/store/`

**Documentation:**
- Architecture: `docs/项目概述和架构设计.md`
- Standards: `docs/开发规范.md`
- API: `docs/API响应规范.md`
- Security: `docs/安全和认证设计（国密）.md`

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
- **Import paths**: Use full module path `github.com/ruoyi-next/ruoyi-go-backend/...`
- **Context propagation**: Always pass `c.Request.Context()` to service methods
- **Response wrapper**: Use `response.Success()` and `response.Error()`, not raw JSON
- **Cache prefix confusion**: User input may include `ruoyi:` prefix from UI, always strip before cache operations
- **UUID validation**: When validating org_id, check format first to avoid PostgreSQL type errors
- **Frontend API calls**: Use wrapped functions in `@/lib/api.ts`, NOT raw axios

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

### Building

```bash
# Development build (Windows)
go build -o ruoyi-backend.exe ./cmd/main.go

# Production build (Linux)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o ruoyi-backend-linux ./cmd/main.go
```

### Database Migrations

Migrations are in `internal/core/db/migrations/`. They run automatically on application startup.

To run migrations manually:
```bash
# Check migration status
# Migrations are versioned and auto-applied via gorm-auto-migrate
# Check `internal/core/db/migrations/` for available migrations
```
