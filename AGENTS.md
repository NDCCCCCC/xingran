# AGENTS.md - Coding Guidelines for XingRan-Next

## Build Commands

### Backend (Go)
```bash
# Build for Windows
go build -o xingran-backend.exe ./cmd/main.go

# Build for Linux (production)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o xingran-backend-linux ./cmd/main.go

# Run tests
go test ./...
go test ./internal/services/operations/

# Run specific test
go test -v -run TestFunctionName ./package/path/
cd internal/services/operations && go test -v -run TestBatchUpsert

# Run with coverage
go test -cover ./...
```

### Frontend (React)
```bash
cd xingran-react-frontend

# Install dependencies
npm install

# Development server
npm run dev

# Build for production
npm run build

# Lint code
npm run lint

# Preview production build
npm run preview
```

## Code Style Guidelines

### Go

**Imports:** Group by standard library, third-party, then internal packages:
```go
import (
    "context"
    "errors"
    
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)
```

**Naming:**
- Interfaces: `UserService`, `CacheProvider` (exported), `userService` (implementation)
- Structs: `UserHandler`, `BuildingService` (PascalCase for exported)
- Private implementations: `userService`, `buildingService` (camelCase)
- Methods: `Create`, `GetByID`, `validateOrg` (camelCase)
- Constants: `CacheKeyDictType` (PascalCase)
- Test files: `*_test.go` suffix

**Service Pattern:**
```go
// Interface definition
type UserService interface {
    Create(ctx context.Context, req *CreateRequest) error
}

// Private implementation
type userService struct {
    db *gorm.DB
}

// Constructor
func NewUserService(db *gorm.DB) UserService {
    return &userService{db: db}
}
```

**Error Handling:**
- Return errors, don't panic
- Use `response.Error(c, http.StatusInternalServerError, err.Error())` in handlers
- Context propagation: always pass `c.Request.Context()` to service methods

**Status Values:** `0 = enabled/normal/visible`, `1 = disabled/stopped/hidden`
Exception: Menu visibility uses `1 = visible, 0 = hidden`

### TypeScript/React

**Naming:**
- Components: PascalCase (`UserTable`, `BuildingForm`)
- Hooks: camelCase starting with `use` (`useAuth`, `useBuilding`)
- Types/Interfaces: PascalCase with descriptive names
- Files: PascalCase for components, camelCase for utilities

**API Calls:**
```typescript
// Use wrapped API functions, NOT raw axios
import { post } from '@/lib/api';
const result = await post('/system/users/list', params);
const users = result.data.list; // Direct access

// WRONG - don't use api instance directly
const response = await api.post('/system/users/list', params);
if (response.data.code === 0) { } // Manual check unnecessary
```

**State Management:** Use Zustand stores in `src/store/`

## Architecture Patterns

1. **Handler-Service Pattern:** Handlers depend on service interfaces, not implementations
2. **Context Propagation:** Always pass `ctx context.Context` through service layers
3. **Response Wrapper:** Use `response.Success()` and `response.Error()` in handlers
4. **Cache Keys:** Use helper functions like `GetDictDataByTypeKey()`, don't hardcode
5. **Pagination:** `current` starts from 1, response uses `list`, `total`, `current`, `pageSize`

## Critical Rules

- **Cache Prefix:** Redis uses `xingran:` prefix automatically. Strip it from user input before cache operations
- **UUID Validation:** `org_id` must be valid UUID format, not department names
- **Import Path:** Use full module path `github.com/xingran-next/xingran-go-backend/...`
- **No Raw JSON:** Always use response wrappers, never `c.JSON()` directly
