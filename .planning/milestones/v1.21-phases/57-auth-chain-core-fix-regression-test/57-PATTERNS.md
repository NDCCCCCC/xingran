# Phase 57: 认证链核心修复 + 回归测试 - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 2 (1 MODIFY + 1 CREATE)
**Analogs found:** 2 / 2 (both exact matches in codebase)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| MODIFY `internal/middleware/apikey.go` | middleware | request-response | `internal/middleware/apikey.go` (self — 内部函数签名修复) + `pkg/response/response.go` (错误响应基准) | exact (self-edit) |
| CREATE `internal/middleware/apikey_integration_test.go` | test | request-response | `internal/api/v1/auth_integration_test.go` (gin+httptest 装配) + `internal/services/usage_logger_test.go` (sqlite helper) + `internal/services/system/apikey_service_test.go:402` (testify 风格) | exact (multi-analog) |

---

## Pattern Assignments

### MODIFY `internal/middleware/apikey.go` (middleware, request-response)

**Analog:** 自身修复 + 与同文件其余 3 个中间件保持一致。CLAUDE.md "Scope Constrainment" 规则要求只改报告的问题——本 phase 只动 P0-2 (`setUserContextForAPIKey`) 与 P0-1 (`RequireAPIKeyResourcePermission`) 两处。

**当前 import 块** (`internal/middleware/apikey.go:1-12`) — 新增 `internal/models` import 时保持此结构：
```go
package middleware

import (
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)
```

**AUTH-01 病灶代码** (`internal/middleware/apikey.go:144-179`) — P0-2 根因，须替换：
```go
// setUserContextForAPIKey 设置API Key认证的用户上下文（私有函数）
// apiKey 参数的类型是 *models.APIKey，使用 interface{} 避免循环导入   ← 误判注释, 删除
func setUserContextForAPIKey(c *gin.Context, apiKey interface{}, scopes []string) {
	type apiKeyType struct {           // ← 局部 VALUE 类型, 删除
		ID           string
		Name         string
		UserID       *string
		InheritPerms bool
		User         *interface{}
	}
	if ak, ok := apiKey.(apiKeyType); ok {   // ← *models.APIKey 断言为值类型 apiKeyType → 恒 false
		// 这 7 行 c.Set 永远不执行 (P0-2 病灶)
		c.Set("user_id", userID); ...
	}
}
```

**AUTH-01 修复后写法 (D-05)** — 签名 `interface{}` → `*models.APIKey`，新增 `internal/models` import，删除局部 `apiKeyType`。**D-04 锁定零行为变更**：`username := ak.Name` 语义原样保留（Name 是密钥名而非用户名）：
```go
import "github.com/xingran-next/xingran-go-backend/internal/models"  // 新增 import

// setUserContextForAPIKey 设置API Key认证的用户上下文（私有函数）
func setUserContextForAPIKey(c *gin.Context, apiKey *models.APIKey, scopes []string) {
	userID := ""
	if apiKey.UserID != nil {
		userID = *apiKey.UserID
	}
	c.Set("user_id", userID)
	c.Set("username", apiKey.Name)   // D-04: 语义保留 (Name=密钥名, 非 username)
	c.Set("nickname", "")
	c.Set("api_key_id", apiKey.ID)
	c.Set("scopes", scopes)
	c.Set("auth_type", "api_key")
	if apiKey.InheritPerms && apiKey.User != nil {
		c.Set("inherit_perms", true)
	}
}
```

**目标类型形状** (`internal/models/api_key.go:8-23` + `internal/models/base.go:11-19`) — 决定字段访问路径，已直接确认：
```go
// APIKey struct shape (VERIFIED)
type APIKey struct {
	BaseModel                            // 内嵌 → apiKey.ID 实际来自 BaseModel.ID
	Name         string                  // apiKey.Name
	UserID       *string                 // apiKey.UserID (指针, 须 nil-check)
	IsActive     bool
	Scopes       []string                // 已由 GORM 反序列化为 []string
	IPWhitelist  []string
	InheritPerms bool
	User         *User                   // apiKey.User (指针)
}
// BaseModel.ID 是 uuid string (BeforeCreate 钩子空时自动生成)
```

**AUTH-02 病灶代码** (`internal/middleware/apikey.go:221-231`) — P0-1 反模式，须重写：
```go
// RequireAPIKeyResourcePermission API Key资源权限验证中间件
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 确定所需作用域
		requiredScope := getRequiredScope(action)
		// 调用作用域验证
		RequireScope(requiredScope)(c)   // ← 反模式: 内联调用, 靠内部 c.Next() 副作用推进
	}
}
```

**AUTH-02 修复后最小写法 (D-03，CONTEXT.md 已把具体写法留给 Claude 裁量)** — 注册期算 scope + 直接 `return RequireScope(...)`，由 gin 正常挂载：
```go
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	requiredScope := getRequiredScope(action)   // 纯函数, 注册期即可算 (apikey.go:235-249 确认不依赖 *gin.Context)
	return RequireScope(requiredScope)           // 直接返回中间件
}
```
**`getRequiredScope` 是纯函数证据** (`internal/middleware/apikey.go:235-249`)：仅查局部 `scopeMap`，无 `*gin.Context` 参数，可安全提到注册期。

**备选重写（显式内联，为 Phase 61 资源逻辑预留 hooks）** — 若想为 Phase 61 AUTH-04 留扩展点：
```go
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requiredScope := getRequiredScope(action)
		scopes, exists := c.Get("scopes")
		if !exists {
			response.Error(c, response.ErrForbidden, "缺少权限作用域"); c.Abort(); return
		}
		userScopes, ok := scopes.([]string)
		if !ok {
			response.Error(c, response.ErrForbidden, "权限作用域格式错误"); c.Abort(); return
		}
		hasScope := false
		for _, scope := range userScopes {
			if scope == "admin" || scope == requiredScope { hasScope = true; break }
		}
		if !hasScope {
			response.Error(c, response.ErrForbidden, "权限不足，需要作用域: "+requiredScope); c.Abort(); return
		}
		c.Next()
	}
}
```

**`RequireScope` 既有正确实现** (`internal/middleware/apikey.go:184-219`) — 备选重写参考此逻辑，确保 `response.Error` 调用一致：
```go
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopes, exists := c.Get("scopes")
		if !exists {
			response.Error(c, response.ErrForbidden, "缺少权限作用域")
			c.Abort(); return
		}
		userScopes, ok := scopes.([]string)
		if !ok {
			response.Error(c, response.ErrForbidden, "权限作用域格式错误")
			c.Abort(); return
		}
		hasScope := false
		for _, scope := range userScopes {
			if scope == "admin" || scope == requiredScope { hasScope = true; break }
		}
		if !hasScope {
			response.Error(c, response.ErrForbidden, "权限不足，需要作用域: "+requiredScope)
			c.Abort(); return
		}
		c.Next()
	}
}
```

**4 中间件签名（须保持不变，SC#2 审查项）**：
| 中间件 | 签名 | 来源 |
|--------|------|------|
| `MultiAuth` | `(apiKeyService system.APIKeyService, usageLogger services.UsageLogger) gin.HandlerFunc` | `apikey.go:22` |
| `RequireScope` | `(requiredScope string) gin.HandlerFunc` | `apikey.go:184` |
| `RequireAPIKeyResourcePermission` | `(resource string, action string) gin.HandlerFunc` | `apikey.go:223` |
| `RateLimitByScope` | `(rateLimiter *services.RateLimiter) gin.HandlerFunc` | `apikey.go:254` |

**错误响应约定** (`internal/middleware/apikey.go:34,42,51,189,198,212`) — 现有写法，修复时保持一致：
```go
response.Error(c, response.ErrUnauthorized, "密钥验证失败: "+err.Error())
c.Abort()
return
```

**调用链关键点** (`internal/middleware/apikey.go:40,58`) — P0-2 根因链证据，修复后这两行行为不变：
```go
apiKey, err := apiKeyService.ValidateAPIKey(c.Request.Context(), apiKeyStr)  // 返回 *models.APIKey (apikey_service.go:36,129)
// ...
setUserContextForAPIKey(c, apiKey, apiKey.Scopes)  // 传指针, 修复后签名匹配
```

**D-03 明确不在 Phase 57 修的两项（plan 须显式标注"延后"防 scope creep）**：
1. `resource` 参数被忽略（`apikey.go:223` 入参 `resource string` 从未使用）→ Phase 61 AUTH-04
2. `getScopeFromContext` 只取 `scopes[0]`（`apikey.go:310`）→ Phase 61 QUAL-03

---

### CREATE `internal/middleware/apikey_integration_test.go` (test, request-response)

**Analog 1 (gin+httptest 装配):** `internal/api/v1/auth_integration_test.go`
**Analog 2 (sqlite test-DB helper):** `internal/services/usage_logger_test.go:30-57` (`setupUsageLoggerTestDB`)
**Analog 3 (testify + 真实构造函数风格):** `internal/services/system/apikey_service_test.go:402` (`TestValidateAPIKey`)
**Analog 4 (既有同包纯函数测试基线，不回归):** `internal/middleware/apikey_test.go`

**package 与 import 块** — 同包（`package middleware`），与 `apikey_test.go:1` 一致：
```go
package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
)
```

**gin + httptest 装配骨架 (Analog 1)** — 复用 `auth_integration_test.go:21,154-186,419-454` 的三段式 pattern：
```go
// Source: internal/api/v1/auth_integration_test.go:20-46,154-186,419-454 (VERIFIED pattern)
func TestMultiAuthIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)   // ← auth_integration_test.go:21,155,240,322,356,420,459 既有写法

	t.Run("有效key+正确scope_通过并写入context", func(t *testing.T) {
		// --- arrange: fake + 真实 gin engine ---
		router := gin.New()                                    // ← auth_integration_test.go:158,215,243,268,437 既有
		router.Use(MultiAuth(fakeSvc, fakeLogger))            // 真实 MultiAuth
		router.GET("/ping", RequireScope("read"), func(c *gin.Context) { ... })

		// --- act: httptest 驱动 ---
		req := httptest.NewRequest("GET", "/ping", nil)        // ← auth_integration_test.go:32,54,183,221,251,276,378,442,498
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()                            // ← auth_integration_test.go:34,56,185,223,253,278,379,443,499
		router.ServeHTTP(w, req)                               // ← auth_integration_test.go:186,224,254,279,381,445,501

		// --- assert: 状态码 + context 键 ---
		assert.Equal(t, 200, w.Code)                          // ← auth_integration_test.go:188,226,255,280,382,446,501
	})
}
```

**错误码状态映射断言基准** — 直接复用 `auth_integration_test.go:419-454` 的 `TestIntegration_AuthError_HTTPStatusCodes` 风格断言：
```go
// Source: internal/api/v1/auth_integration_test.go:419-454 (VERIFIED)
// response.ErrUnauthorized → HTTPStatus=401 (pkg/response/response.go:40)
// response.ErrForbidden    → HTTPStatus=403 (pkg/response/response.go:41)
// middleware 的 response.Error(c, response.ErrUnauthorized, ...) 经 response.go:95 用 appErr.HTTPStatus 作 HTTP 状态码
// 故三条路径断言: 路径①=200, 路径②=403, 路径③=401
```

**sqlite test-DB helper (Analog 2)** — 复制 `usage_logger_test.go:30-57` 的 `setupUsageLoggerTestDB`（含 `busy_timeout=5000` 防写锁）。D-02 证据需要它为 `NewUsageLogger(db)` 提供不 nil 的 `*gorm.DB`：
```go
// Source: internal/services/usage_logger_test.go:30-57 (VERIFIED, copy verbatim)
// 关键设计点 (helper 注释原文):
//   - 每测试独立文件 DB (os.TempDir + 唯一名), 替代共享内存 DB 防 SQLite 写锁
//   - busy_timeout=5000 让写锁排队而非立即报错
//   - 不用 t.TempDir: LogUsage 的 fire-and-forget goroutine 测试结束后仍写文件, t.TempDir 自动 cleanup 会 mark test failed
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db", time.Now().UnixNano(), os.Getpid()))
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			method TEXT, path TEXT, status_code INTEGER, client_ip TEXT,
			user_agent TEXT, duration INTEGER, success BOOLEAN, created_at DATETIME
		)
	`).Error
	require.NoError(t, err)
	return db
}
```
**新增 import（除 Analog 1/3 已有外）**：`"fmt"`, `"os"`, `"path/filepath"`, `"time"`, `"gorm.io/driver/sqlite"`, `"gorm.io/gorm"`。

**D-02 构造函数证据测试 (Analog 3 风格)** — 复用 `apikey_service_test.go:402-405` 的 `setupTestDB(t)` + `NewXxx(db)` 写法：
```go
// Source: internal/services/system/apikey_service_test.go:402-405 (VERIFIED pattern)
//   db := setupTestDB(t)
//   service := NewAPIKeyService(db)
//   ctx := context.Background()
// Adapted for D-02:
func TestConstructorsCallable_D02(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := services.NewUsageLogger(db)     // 真实构造函数 (usage_logger.go:38 返回 UsageLogger interface)
	assert.NotNil(t, logger)
	_ = MultiAuth(&fakeAPIKeyService{}, logger)   // 编译通过即证明 MultiAuth 第2参接受 services.UsageLogger

	rl := services.NewRateLimiter()           // 真实构造函数 (rate_limiter.go:45 无参, 返回 *RateLimiter)
	assert.NotNil(t, rl)
	_ = RateLimitByScope(rl)                  // 编译通过即证明签名兼容
}
```

**fake `system.APIKeyService` 形状 (D-01)** — 须实现 `apikey_service.go:29-39` 全部 9 方法，仅 `ValidateAPIKey` 给真逻辑：
```go
// Source: internal/services/system/apikey_service.go:29-39 (interface shape VERIFIED, 9 methods)
type fakeAPIKeyService struct {
	validKey    *models.APIKey  // 预置: 有效 key 时返回
	validateErr error           // 预置: 无效 key 时返回 (触发 401 路径)
}
func (f *fakeAPIKeyService) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
	if f.validateErr != nil { return nil, f.validateErr }
	return f.validKey, nil
}
// 其余 8 方法签名 (apikey_service.go:30-35,37-38 确认):
func (f *fakeAPIKeyService) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) { panic("not used in MultiAuth path") }
func (f *fakeAPIKeyService) ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error) { panic("not used") }
func (f *fakeAPIKeyService) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) { panic("not used") }
func (f *fakeAPIKeyService) UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error { panic("not used") }
func (f *fakeAPIKeyService) DeleteAPIKey(ctx context.Context, id string) error { panic("not used") }
func (f *fakeAPIKeyService) ToggleAPIKeyStatus(ctx context.Context, id string) error { panic("not used") }
func (f *fakeAPIKeyService) ListUsageLogs(ctx context.Context, params ListUsageLogsParams) (*UsageLogsPageResult, error) { panic("not used") }
func (f *fakeAPIKeyService) GetUsageLogSummary(ctx context.Context, apiKeyID string) (*UsageSummary, error) { panic("not used") }
```
**注意（Pitfall 2）**：fake 须实现全部 9 方法否则编译失败。fake 引用了 `requests.CreateAPIKeyRequest` / `PageResult` / `ListUsageLogsParams` / `UsageLogsPageResult` / `UsageSummary` 等 system 包内类型——测试文件须 `import ".../internal/services/system"` 或在 fake 方法签名中用全限定类型 `system.PageResult` 等。鉴于 `MultiAuth(apiKeyService system.APIKeyService, ...)` 入参类型是 `system.APIKeyService`，**fake 应定义在同 package middleware 但用 system 包全限定类型引用**（避免名字冲突）。Planner 须确认 fake 的 8 个 panic 方法签名严格对齐 `apikey_service.go:29-39`。

**fake `services.UsageLogger` 形状 (D-01)** — 仅 1 方法 (`usage_logger.go:12-17`)：
```go
// Source: internal/services/usage_logger.go:12-17 (interface shape VERIFIED)
type fakeUsageLogger struct{ logged bool }
func (f *fakeUsageLogger) LogUsage(ctx context.Context, req *services.LogUsageRequest) error {
	f.logged = true
	return nil   // 记录被调用即可, 不验证落库
}
```

**X-API-Key 格式 helper (Pitfall 3 防御)** — `isValidKeyFormat` 要求 `rec_` + 64 位 hex = 68 字符（`apikey.go:86-106`），格式错会在 `ValidateAPIKey` 前被拦截走 401：
```go
// hex64 返回 64 位十六进制字符串 (满足 isValidKeyFormat: rec_ + 64 hex = 68 字符)
func hex64() string { return strings.Repeat("0123456789abcdef", 4) }  // 64 chars
```

**SC#1 context 键断言清单（4 键）** — 路径① handler 内断言（context 已被修复后的 `setUserContextForAPIKey` 写入）：
```go
assert.NotEmpty(t, c.GetString("user_id"))               // *UserID 解引用后非空
assert.Equal(t, "ak-test-id", c.GetString("api_key_id")) // BaseModel.ID (UUID)
assert.Equal(t, []string{"read"}, c.MustGet("scopes"))   // []string
assert.Equal(t, "api_key", c.GetString("auth_type"))     // 字面量
```

**三条路径断言清单（SC#3）**：

| 路径 | 预置 | 期望 `w.Code` | 对应 `response` 错误 |
|------|------|--------------|---------------------|
| ① 有效+正确 scope | fake 返回 `Scopes:["read"]`，路由挂 `RequireScope("read")` | **200** | — |
| ② 有效+缺失 scope | fake 返回 `Scopes:["read"]`，路由挂 `RequireScope("write")` | **403** | `ErrForbidden` |
| ③ 无效 key | fake 返回 `validateErr` | **401** | `ErrUnauthorized` |

**既有同包测试基线（不回归）** (`internal/middleware/apikey_test.go`) — 3 个纯函数测试 `TestIsValidKeyFormat` / `TestIsIPAllowed` / `TestGetRequiredScope`（`apikey_test.go:10,43,104`）不修改，新增集成测试与之并存于 `package middleware`。

---

## Shared Patterns

### 统一响应包装 (response.Error + AppError)

**Source:** `pkg/response/response.go:36-102`
**Apply to:** `apikey.go` 错误分支（保持现有写法不变）+ 测试断言状态码基准
```go
// pkg/response/response.go:40-41 (错误定义)
ErrUnauthorized = &AppError{Code: 401, Message: "未授权", HTTPStatus: 401}
ErrForbidden    = &AppError{Code: 403, Message: "禁止访问", HTTPStatus: 403}

// pkg/response/response.go:87-102 (Error 函数用 appErr.HTTPStatus 作 HTTP 状态码)
func Error(c *gin.Context, err interface{}, message ...string) {
	appErr := toAppError(err)
	if len(message) > 0 { appErr.Message = message[0] }
	c.JSON(appErr.HTTPStatus, Response{Code: appErr.Code, Message: appErr.Message, ...})
}

// middleware 调用约定 (apikey.go:34,42,51,189,198,212):
response.Error(c, response.ErrUnauthorized, "密钥验证失败: "+err.Error())
c.Abort()
return
```

### testify 断言风格 (TESTING.md "无 gomock")

**Source:** `internal/api/v1/auth_integration_test.go:12-13` + `internal/services/usage_logger_test.go:11-12` + `internal/services/system/apikey_service_test.go:402`
**Apply to:** 新建 `apikey_integration_test.go` 全部断言
```go
import (
	"github.com/stretchr/testify/assert"   // 非致命断言
	"github.com/stretchr/testify/require"  // 致命断言 (setup 失败用)
)
// assert.Equal / assert.NotNil / assert.NotEmpty / assert.True / assert.NoError
// require.NoError (仅在 setupUsageLoggerTestDB 等 setup 步骤用, 失败立即停止)
```
**无 gomock 约定**：fake 是手写 struct，不引入 mock 框架（`.planning/codebase/TESTING.md` 依据）。

### t.Run 子测试命名 (中文子测试名)

**Source:** `internal/services/usage_logger_test.go:74` (`t.Run("正常日志记录", ...)`) + `internal/services/system/apikey_service_test.go:408` (`t.Run("有效密钥", ...)`) + `internal/middleware/apikey_test.go:11` (`t.Run("有效密钥格式", ...)`)
**Apply to:** 三条路径的子测试名用中文：`"有效key+正确scope_通过并写入context"` / `"有效key+缺失scope_403"` / `"无效key_401"`。
```go
// 子测试风格 (既有约定, VERIFIED across apikey_test.go / usage_logger_test.go / apikey_service_test.go)
t.Run("有效密钥", func(t *testing.T) {
	assert.NoError(t, err)
	assert.NotNil(t, validated)
})
```

### gin.TestMode + httptest 三段式 (arrange-act-assert)

**Source:** `internal/api/v1/auth_integration_test.go:20-46,154-186`
**Apply to:** `apikey_integration_test.go` 所有 `TestMultiAuthIntegration` 子测试
```go
gin.SetMode(gin.TestMode)                 // 测试函数开头调一次
// arrange
router := gin.New()
router.Use(MultiAuth(fakeSvc, fakeLogger))
router.GET("/path", RequireScope("..."), handlerFunc)
// act
req := httptest.NewRequest("GET", "/path", nil)
req.Header.Set("X-API-Key", "rec_"+hex64())
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
// assert
assert.Equal(t, expectedCode, w.Code)
```

---

## No Analog Found

无。本 phase 所有 2 个文件均有 codebase 内精确或多个 analog。新建测试文件采用 multi-analog 组合（`auth_integration_test.go` 的 gin+httptest 装配 + `usage_logger_test.go` 的 sqlite helper + `apikey_service_test.go:402` 的 testify+真实构造函数风格），均已在 codebase 验证。

---

## Metadata

**Analog 搜索范围:**
- `internal/middleware/` (目标源 + 既有同包测试)
- `internal/api/v1/auth_integration_test.go` (gin+httptest 装配 analog)
- `internal/services/usage_logger_test.go` (sqlite helper analog)
- `internal/services/usage_logger.go` (UsageLogger 接口 + NewUsageLogger 签名)
- `internal/services/rate_limiter.go` (NewRateLimiter 签名 + RateLimiter 类型)
- `internal/services/system/apikey_service.go` (APIKeyService 接口 9 方法 + ValidateAPIKey 返回类型)
- `internal/services/system/apikey_service_test.go:402` (testify + 真实 DB 风格)
- `internal/models/api_key.go` + `internal/models/base.go` (APIKey struct + BaseModel.ID)
- `pkg/response/response.go` (错误码 HTTPStatus 映射基准)
- `internal/api/router.go` (grep 验证 4 中间件零挂载 = P0-1 死代码现状)

**Files scanned:** 9（含目标源、4 个 analog、2 个 model、1 个 response、1 个 router grep）

**关键 ground-truth（planner 须直接引用）:**
- `APIKeyService` 接口 **9 方法** (`apikey_service.go:29-39`) — fake 必须全实现
- `ValidateAPIKey` 返回 `*models.APIKey` (`apikey_service.go:36,129`) — P0-2 根因链源头
- `NewUsageLogger(db) UsageLogger` 返回 **interface** (`usage_logger.go:38`)
- `NewRateLimiter() *RateLimiter` **无参** 返回具体类型 (`rate_limiter.go:45`) — **勿与** `internal/services/operations/rate_limiter.go:28` 的同名带参构造混淆
- `response.ErrUnauthorized.HTTPStatus=401` / `response.ErrForbidden.HTTPStatus=403` (`response.go:40-41`)
- `router.go` 4 中间件零挂载（grep no matches）= P0-1 死代码现状（Phase 57 不挂载，挂载属 Phase 60）

**Pattern extraction date:** 2026-08-13

---

*Phase: 57 - 认证链核心修复 + 回归测试*
*Pattern map ready for planner consumption*
