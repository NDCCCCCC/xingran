# Architecture

## Pattern: Layered Handler-Service with Core DI

The application follows a clean layered architecture with dependency injection via the `Core` singleton.

```
┌─────────────────────────────────────────────────────┐
│                    Router (gin)                       │
│              internal/api/router.go                   │
├─────────────────────────────────────────────────────┤
│  Middleware Chain                                     │
│  Auth → CORS → Logging → Encryption → Permission     │
│  pkg/middleware/                                      │
├─────────────────────────────────────────────────────┤
│  Handlers (HTTP Layer)                               │
│  internal/api/v1/{module}/                           │
│  - Validate request                                  │
│  - Call service                                      │
│  - Return response                                   │
├─────────────────────────────────────────────────────┤
│  Services (Business Logic)                           │
│  internal/services/{module}/                         │
│  - Interface definition                              │
│  - Private implementation                            │
│  - Cache integration                                 │
├─────────────────────────────────────────────────────┤
│  Models (Data Layer)                                 │
│  internal/models/                                    │
│  - GORM models with JSON tags                       │
│  - Base model with UUID, timestamps, soft delete    │
├─────────────────────────────────────────────────────┤
│  Core (DI Container)                                 │
│  internal/core/core.go                               │
│  - DB, Cache, JWT, Scheduler instances               │
│  - Passed to routers for DI                          │
├─────────────────────────────────────────────────────┤
│  Infrastructure                                      │
│  PostgreSQL + Redis + Config                         │
└─────────────────────────────────────────────────────┘
```

## Core Dependency Injection

The `Core` struct (`internal/core/core.go`) is the central DI container:

```go
type Core struct {
    DB          *gorm.DB
    Cache       cache.Cache       // pkg/cache/ interface
    PwdManager  PasswordManager
    JWTManager  *JWTManager
    Scheduler   *cron.Cron
    Config      *config.Config
}
```

All routers receive `*core.Core` and use it to instantiate services and handlers:
```go
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
    userService := system.NewUserService(core.GetDB(), core.Cache, core.PwdManager)
    userHandler := system.NewUserHandler(userSvc, core.PwdManager)
    // register routes...
}
```

## Data Flow

### Request Flow
1. Client sends HTTP request (optionally SM4-encrypted)
2. Gin router matches route → middleware chain executes
3. Auth middleware validates JWT token
4. Permission middleware checks RBAC access
5. Encryption middleware decrypts SM4 body if enabled
6. Handler parses request, calls service
7. Service executes business logic (DB + Cache)
8. Response wrapped in standard format: `{code, message, data, timestamp, request_id}`

### Cache Flow
1. Read: Check L1 (memory) → L2 (Redis) → DB → populate cache
2. Write: Update DB → invalidate L1 + L2 (async via writer pool)
3. L2 writer pool handles async Redis writes with retry

### Import Flow (Workstation/Building)
1. Upload Excel file → `excel_service.go` parses
2. `excel_config.go` maps columns to fields
3. `reference_resolver.go` resolves department names to UUIDs
4. `batch_upserter.go` batch insert/update with conflict handling
5. `cache_invalidator.go` clears related caches
6. Optional: geocoding for address → coordinates

## Key Abstractions

### Service Interface Pattern
Every service follows the interface pattern:
- Interface defined in service file (e.g., `UserService interface`)
- Private implementation struct (`userServiceImpl`)
- Constructor function (`NewUserService`)
- Dependencies injected via constructor

### CacheProvider Interface
```go
type CacheProvider interface {
    GetCached(ctx, key, dest) error
    SetCache(ctx, key, value, ttl) error
    DeleteCache(ctx, key) error
}
```
Dual implementation:
- Cached version: reads/writes Redis
- Non-cached version: pass-through to DB

### Dual Cache Architecture
- **Legacy**: Root-level `*_cache_service.go` files wrapping `DataCacheService`
- **New**: `internal/services/system/*_cache_impl.go` using `CacheProvider` interface

## Entry Points

- **HTTP**: `cmd/main.go` → `internal/api/router.go` → module routers
- **WebSocket**: `internal/websocket/` for real-time connections
- **Scheduler**: `internal/scheduler/` for cron-based tasks (AD sync, work orders)
- **Migrations**: `internal/core/db/migrations/` auto-run on startup

## Module Boundaries

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
