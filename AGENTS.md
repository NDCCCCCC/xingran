# AGENTS.md - Coding Guidelines for XingRan-Next

## Build Commands

### Backend (Go)
```bash
# 构建产物一律输出到 out/（gitignored），禁止落在仓库根目录
mkdir -p out            # Windows: if not exist out mkdir out
# Build for Windows
go build -o out/xingran-backend.exe ./cmd/main.go

# Build for Linux (production)
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o out/xingran-backend-linux ./cmd/main.go

# 或直接用在用脚本（产物同样落 out/）
# scripts\build-linux.bat [root] / scripts\build_linux.sh / scripts\build-embedded.bat

# Run tests
go test ./...
go test ./internal/services/operations/

# Run specific test
go test -v -run TestFunctionName ./package/path/
cd internal/services/operations && go test -v -run TestBatchUpsert

# Run with coverage
go test -cover ./...
```

## 构建产物与目录约定（强制）

- **所有构建产物必须落在 `out/`**：本地脚本（`scripts/build-*.bat|sh`、`scripts/build/*.bat`、`scripts/agent/build.sh`）与 CI（`.github/workflows/deploy.yml` 的 `-o out/xingran-backend`）统一。`out/` 已在 `.gitignore` 中，禁止把 `.exe` / `-linux` / `-embedded` / 打包 `.tar.gz|.zip` 落在仓库根目录。
- **一次性运维/诊断 CLI 放 `scripts/ops/`**，不要放 `cmd/`。`cmd/` 只保留应用入口 `cmd/main.go`(+`main_test.go`) 与独立服务 `cmd/agent/`。`scripts/ops/` 下的 `package main` 仍由 `go build ./...` 覆盖。
- **临时/一次性脚本不得散落在 `scripts/` 顶层**：无现存引用的历史脚本归档到 `.planning/milestones/v1.53-archived-scripts/` 并写索引；在用脚本（`check-route-registry.mjs`、`scan-routes.cjs`、`generate_swagger.sh`、`migrate-sm4-key.go`、`operlog_e2e_verify.sh` 等）保留在 `scripts/` 顶层并在 `scripts/README.md` 登记。
- **禁止在根目录新增启动/构建包装脚本**（如 `run.bat`）；前后端由开发者各自启动。

## 测试目录与文件命名（强制）

- 统一使用 **`tests/`（复数）**，**禁止再建 `test/`（单数）**；`test/fixtures` 曾与 `tests/fixtures` 重复，已删除前者。
- Go 单元测试：就近放业务包内，命名 `*_test.go`；禁止把 `*_test.go` 放进 `tests/` 之外的自建目录充当源码。
- 分层目录（均位于 `tests/` 下，按用途二选一，不得混放同名文件）：
  - `tests/benchmark/`：`*_bench_test.go`（性能基准，如 `login_encryption_bench_test.go`）
  - `tests/e2e/`：`*.spec.ts`（前端 e2e）、`*_test.go` / `*.sh`（后端 e2e）
  - `tests/integration/`：`*_test.go`（集成测试）
  - `tests/fixtures/`：共享 fixture（`.go` / `.json`），**唯一权威副本**
  - `tests/scripts/`：测试辅助脚本（按子域 `agent/ crypto/ rpa/ vdi/` 分目录）
- 前端：组件/工具测试用 `*.test.ts(x)` 或 `src/**/__tests__/`，与被测模块就近；快照与截图证据放 `.planning/notes/v1.24-visual-evidence/`（唯一豁免的 png 目录）。
- 禁止 `test_*.py` / `test_*.go` 这类「以 test 前缀命名源码」的写法散落在 `scripts/` 顶层；测试脚本归 `tests/scripts/` 并加对应子域目录。

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
