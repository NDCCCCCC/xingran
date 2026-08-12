# Conventions

## Go Code Style

### Handler-Service Pattern (Mandatory)
Every module follows the three-file pattern:

**Handler** (`internal/api/v1/{module}/xxx_handler.go`):
```go
type XxxHandler struct {
    xxxService module.XxxService
    // other dependencies...
}

func NewXxxHandler(xxxSvc module.XxxService) *XxxHandler {
    return &XxxHandler{xxxService: xxxSvc}
}

func (h *XxxHandler) List(c *gin.Context) {
    // 1. Parse/bind request
    // 2. Call service
    // 3. Return response
}
```

**Service** (`internal/services/{module}/xxx_service.go`):
```go
type XxxService interface {
    CreateXxx(ctx context.Context, req *CreateXxxRequest) (*Xxx, error)
    // ...
}

type xxxServiceImpl struct {
    db    *gorm.DB
    cache CacheProvider
}

func NewXxxService(db *gorm.DB, cache CacheProvider) XxxService {
    return &xxxServiceImpl{db: db, cache: cache}
}
```

**Router** (`internal/api/v1/{module}/xxx_router.go`):
```go
func SetupXxxRouter(r *gin.RouterGroup, core *core.Core) {
    svc := module.NewXxxService(core.GetDB(), core.Cache)
    handler := module.NewXxxHandler(svc)
    r.POST("/list", handler.List)
}
```

### Response Wrapping
Always use `response.Success()` and `response.Error()`:
```go
response.Success(c, data)                          // 200, code=0
response.Error(c, http.StatusBadRequest, "msg")   // 400
response.PageSuccess(c, list, total, current, pageSize)
```

Never return raw `c.JSON()`.

### Error Handling
- Service layer returns `error` (not response codes)
- Handler layer translates errors to HTTP status codes
- Use `fmt.Errorf("context: %w", err)` for wrapping
- Use `pkg/errors` for typed errors where needed

### Context Propagation
Always pass `c.Request.Context()` from handler to service:
```go
user, err := h.userService.CreateUser(c.Request.Context(), &req)
```

### Request Binding
Use `ShouldBindJSON` for POST bodies:
```go
var req CreateRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.Error(c, http.StatusBadRequest, "请求参数错误")
    return
}
```

Use `ShouldBindQuery` for URL params:
```go
var req ListQuery
if err := c.ShouldBindQuery(&req); err != nil {
    response.Error(c, http.StatusBadRequest, "请求参数错误")
    return
}
```

### GORM Patterns
- Always use soft delete (GORM default with `deleted_at`)
- UUID primary keys via `gen_random_uuid()`
- Use GORM scopes for reusable query logic
- Avoid raw SQL; prefer GORM chainable API

### Cache Key Helpers
Define cache key functions in service files:
```go
const CacheKeyXxx = "xxx:all"
func GetXxxByIdKey(id string) string {
    return fmt.Sprintf("xxx:detail:%s", id)
}
```

Never hardcode cache key strings inline.

### Cache Prefix Handling
Redis prefix `xingran:` is added automatically by the cache layer.
- Store: `cache.Set(ctx, "user:1", value)` → Redis key `xingran:user:1`
- When user input includes `xingran:` prefix, strip it before cache operations:
```go
if strings.HasPrefix(key, "xingran:") {
    key = key[6:]
}
```

## Status Value Convention

**Universal:** `0 = enabled/normal, 1 = disabled/stopped`

**Exception:** Menu `visible` field: `1 = visible, 0 = hidden`

## API Convention

### Route Pattern
- `POST /list` — List with pagination
- `POST /` — Create
- `POST /:id` — Get by ID
- `POST /:id/update` — Update
- `POST /:id/delete` — Delete

### Response Format
```json
{
    "code": 0,
    "message": "success",
    "data": {},
    "timestamp": 1766380800,
    "request_id": "uuid"
}
```

### Pagination
Request: `{ current: 1, pageSize: 10 }`
Response: `{ list: [], total: 100, current: 1, pageSize: 10 }`

## Frontend Conventions

### API Calling
```typescript
// ✅ Use wrapped functions
import { post } from '@/lib/api';
const result = await post('/system/users/list', params);

// ✅ Use opsApi for operations module
import { buildingApi } from '@/lib/opsApi';
const result = await buildingApi.list(params);

// ❌ Never use raw axios
```

### Token Management
```typescript
import { getAccessToken, getAuthHeaders } from '@/utils/authHelpers';
```

### State Management (Zustand)
- 7 stores in `src/store/`
- `authStore` includes `TokenManager` for auto-refresh
- Use `useStore` hook for component access

### useEffect Dependencies
Objects/arrays in useEffect deps must be memoized:
```typescript
const params = useMemo(() => ({ page, size }), [page, size]);
useEffect(() => { fetchData(params); }, [params]);
```

### Import Organization
```
1. React / framework imports
2. Third-party libraries (antd, axios, etc.)
3. Internal components
4. Hooks
5. Stores
6. Types
7. Utils
8. Styles
```

## Database Conventions

- Table prefix: `sys_` for system tables
- Column names: `snake_case`
- Primary keys: UUID with `gen_random_uuid()`
- Timestamps: `created_at`, `updated_at`, `deleted_at`
- Soft delete: GORM default with `deleted_at` field
- Migrations: auto-run on startup, versioned in `internal/core/db/migrations/`

## Git Conventions

- Commit messages: conventional commits format (`fix:`, `feat:`, `chore:`, `docs:`)
- Build check before commit: `go build ./...`
- Test check before commit: `go test ./...`
- Always ask user before committing
