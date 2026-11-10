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
- **临时/一次性脚本不得散落在 `scripts/` 顶层**：无现存引用的历史脚本归档到 `.planning/milestones/v1.53-archived-scripts/` 并写索引；在用脚本（`check-route-registry.mjs`、`scan-routes.cjs`、`generate_swagger.sh`、`migrate-sm4-key.go` 等）保留在 `scripts/` 顶层并在 `scripts/README.md` 登记。**操作日志 LIVE e2e 不在此列**：2026-11-09 已收敛到唯一权威 `tests/e2e/operlog_e2e_verify.sh`（见下方测试目录约定）。
- **禁止在根目录新增启动/构建包装脚本**（如 `run.bat`）；前后端由开发者各自启动。
- **Excel 导入/导出模板一律动态生成、禁止落盘**：后端由 `ExcelService.GenerateTemplate(entityType)` 依据 `internal/services/operations/excel_config.go` 的列定义在内存生成 xlsx、以 HTTP 流下载（`user_template_<ts>.xlsx` 等）；导出同理；前端如节假日模板用 `XLSX.writeFile` 在浏览器端生成。`.gitignore` 已忽略 `*.xlsx`，**仓库中不得提交或依赖任何 `.xlsx` 文件**；测试若需 xlsx 夹具用 `t.TempDir()` 自建或仅用 `xlsxMagic` 字节，不读取磁盘模板。
- **`data/` 不是仓库目录**：仅指运行时 SQLite/本地数据（已被 `.gitignore` 忽略），不得在仓库创建或提交 `data/`；文档里的示例路径用 `<sqlite-db-path>` 占位，不写具体已删文件名。

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
- **就近单测豁免（2026-11-09 quick 261109-wdu 登记）**：`internal/services/portwrite/` 包内的
  `physical_e2e_test.go` / `port_write_e2e_test.go` 虽名为 e2e，但依赖整套**同包私有 fixture**
  （`newTestDB` / `seedPortAndDevice` / `fileTransportExecutor` / `newTestService` 等，定义在同包其它
  `*_test.go`）与**未导出的生产类型**（`portWriteServiceImpl` / `portWriteExecutor` 等）。Go 的 `_test.go`
  符号无法被其它包 import，硬搬到 `tests/e2e/` 会编译失败；按「Go 单元测试就近放业务包内」判定为
  **合规例外，保留在业务包内、不迁移**。若将来要真迁移，须先把 fixture 抽成正式 `.go` 源码/
  `tests/fixtures/` 共享包（会改变生产包符号面，需单独立项评估），不得仅移文件。

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
