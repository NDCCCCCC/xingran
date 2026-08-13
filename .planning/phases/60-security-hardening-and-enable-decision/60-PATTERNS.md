# Phase 60: 安全加固与启用决策 - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 12 个改动点（5 新建 + 5 修改 + 2 扩展测试）
**Analogs found:** 11 / 12（1 项 KeyPrefix LIKE 无现成 analog，靠代码内 diff）

---

## File Classification

| 文件 | 类型 | 数据流 | 最近 Analog | 匹配质量 |
|------|------|--------|-------------|---------|
| `internal/api/router.go:238-248` | 修改（middleware 链装配） | request-response | `internal/api/router.go:155-161`（depts/users 的 `RequirePermissions` + `DataScopePermission` 串联模式） | exact（既有 router 链装配） |
| `internal/middleware/apikey.go:267-268` | 修改（限流响应头编码） | request-response | `internal/services/cache_config_service.go:138-141`（`strconv.Itoa(info.Default)`） | exact（既有 strconv.Itoa 标准做法） |
| `internal/middleware/apikey.go`（顶 import 块） | 修改（新增 `"strconv"` import） | import | `internal/services/workorder/assignment.go` 中已有 `strconv` import | exact |
| `internal/models/api_key.go` | 修改（`Key` 三字段替换） | schema | `internal/models/api_key.go` 现有 gorm tag 块（Name/UserID/ExpiresAt）+ `internal/models/base.go:11-19` BaseModel 字段 | exact |
| `internal/services/system/apikey_service.go` (`ValidateAPIKey` line 128-161) | 修改（明文 WHERE → SM3 比对） | CRUD | `internal/core/security/password.go:124-153` `VerifyPassword`（SM3 + salt + `subtle.ConstantTimeCompare`） | exact |
| `internal/services/system/apikey_service.go` (`CreateAPIKey` line 164-222) | 修改（明文存 → hash 存） | CRUD | `internal/core/security/password.go:99-120` `HashPassword`（rand+sm3 派生）+ `apikey_service.go:118-126` `generateKey`（crypto/rand 32 字节） | exact |
| `internal/services/system/apikey_service.go` (`ListAPIKeys` line 224-289) | 修改（移除 LEFT(key,12) / substr 双 dialect） | CRUD | `internal/services/system/apikey_service.go:239-247` 同函数（删除 dialect 分支即可） | exact（行内 diff） |
| `internal/services/system/apikey_service_test.go` | **扩展**（SEC-01 三函数测试覆盖） | test | 已存在（`setupTestDB` + `createTestAPIKey` + `TestValidateAPIKey` + `TestCreateAPIKey` + `TestListAPIKeys`），需按新 schema 改造 `CREATE TABLE sys_api_keys` DDL + 测试 | exact（既有同包 test 文件） |
| `internal/middleware/apikey_test.go` | **扩展**（`TestRateLimitHeaderEncoding`） | test | `TestIsValidKeyFormat`（同文件 line 10-40）单测模式：`t.Run` 子测试 + `assert.True/Equal/False` | exact |
| `internal/middleware/apikey_integration_test.go` | **扩展**（`TestRateLimitHeadersInResponse`） | test | 既有 `TestMultiAuthIntegration`（line 151-233）：`gin.SetMode` + `router.Use(MultiAuth(...))` + `httptest.NewRequest` + `w.Header().Get(...)` | exact |
| `internal/api/router_test.go` | **新建**（Wave 0 可选，验证 `/system/apikeys/*` 中间件链装配） | test | 仓库内**无现成 `*_router_test.go`**；最近 analog 是 `internal/middleware/apikey_integration_test.go:TestMultiAuthIntegration`（gin.Engine + httptest 模式） | partial（无 router-test 既有模板，需自创） |
| `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` | **新建**（手动运维 SQL） | config/manual | `internal/core/db/migrations/archive/020_drop_mac_unique_index.sql`（"DROP INDEX IF EXISTS" + 注释段格式） | exact |
| `.planning/notes/260813-auth03-enable-decision.md` | **新建**（决策记录） | doc | `.planning/notes/260615-oper-log-coverage-audit.md`（"段落 1/2/3" + 表格 + 链接先例） | exact |
| `.planning/notes/260813-sec01-hash-migration.md` | **新建**（决策记录） | doc | 同上 | exact |
| `.planning/notes/260813-sec02-redundant-index-removal.md` | **新建**（SEC-02 验证文档） | doc | 同上 | exact |

---

## Pattern Assignments

### `internal/api/router.go`（修改，D-01/D-02/D-03）

**改动点**：line 238-248 `apikeys` 路由组在 `RequirePermissions` 之后追加 `MultiAuth` + `RateLimitByScope`，形成 3 段 middleware 链。

**Analog**：`internal/api/router.go:155-161`（depts 用户的 `RequirePermissions` + `DataScopePermission` 串联）+ `apikey_router.go:9-26`（既有 `SetupAPIKeyRouter` 装配 DI）。

**Middleware 链装配模式**（router.go:155-164，depts 段，仅做长度参考）：
```go
depts := authorized.Group("/depts")
depts.Use(middleware.RequirePermissions([]string{
    string(permission.DeptList),
    string(permission.DeptAdd),
    string(permission.DeptEdit),
    string(permission.DeptView),
}, middleware.OpsSelectorReadPerms, core))
// D-01 改造: 追加 MultiAuth + RateLimitByScope
depts.Use(middleware.DataScopePermission(core))
```

**改造后目标形态**（router.go:238-248 替换块）：
```go
apikeys := authorized.Group("/apikeys")
apikeys.Use(middleware.RequirePermissions([]string{
    "system:apikey:list",
    "system:apikey:add",
    "system:apikey:edit",
    "system:apikey:delete",
}, core))
// D-01 改造: 追加 MultiAuth + RateLimitByScope
apikeys.Use(middleware.MultiAuth(
    systemServices.NewAPIKeyService(core.GetDB()),
    services.NewUsageLogger(core.GetDB()),
))
apikeys.Use(middleware.RateLimitByScope(services.NewRateLimiter()))
{
    systemV1.SetupAPIKeyRouter(apikeys, core)
}
```

**复用 checklist:**
- [ ] `import` `systemServices` 和 `services`（已在 line 21-22）
- [ ] `core.GetDB()` 既有 helper（`core.go:131`）
- [ ] `systemServices.NewAPIKeyService(db)` 与 `services.NewUsageLogger(db)` 现有 constructor
- [ ] `services.NewRateLimiter()` Phase 57 D-02 验证无参版（`apikey_integration_test.go:246` 实证）

---

### `internal/middleware/apikey.go:267-268`（修改，D-11/D-12）

**改动点**：`RateLimitByScope` 函数内 `string(rune(result.Limit))` + `string(rune(result.Remaining))` → `strconv.Itoa(result.Limit)` + `strconv.Itoa(result.Remaining)`；加 `"strconv"` import。

**Analog A — 限流头现状**（apikey.go:266-269 改造前）：
```go
// 设置速率限制响应头（RFC 6585）
c.Header("X-RateLimit-Limit", string(rune(result.Limit)))        // ← P2-a bug
c.Header("X-RateLimit-Remaining", string(rune(result.Remaining))) // ← P2-a bug
c.Header("X-RateLimit-Reset", result.ResetAt.Format(time.RFC3339))
```

**Analog B — 项目内 `strconv.Itoa` 标准做法**（`internal/services/cache_config_service.go:138-141`）：
```go
s.db.Model(&models.Config{}).Where("config_key = ?", config.ConfigKey).
    Update("config_value", strconv.Itoa(info.Default))
```

**改造后目标形态**：
```go
import (
    "strconv"  // 新增 (D-11)
    // ... 既有 imports
)

// D-11: 修复 P2-a string(rune(int)) → strconv.Itoa
c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
c.Header("X-RateLimit-Reset", result.ResetAt.Format(time.RFC3339))
```

**复用 checklist:**
- [ ] `strconv` import 加入 `apikey.go` block（line 3-13）
- [ ] `result.Limit` / `result.Remaining` 字段类型 `int`（`rate_limiter.go:39-41`），`Itoa` 直接接受
- [ ] 不动 line 269 (`time.RFC3339` 是字符串，不是整数)

---

### `internal/models/api_key.go`（修改，D-06）

**改动点**：移除 `Key string` 字段，新增 `KeyHash` + `Salt` + `KeyPrefix` 三列。

**Analog — 项目内 GORM 字段 tag 约定**（既有 api_key.go:8-23 + models/base.go:11-19）：
```go
type APIKey struct {
    BaseModel
    Name        string     `gorm:"size:100;not null" json:"name"`               // 业务字段模式
    Key         string     `gorm:"size:100;uniqueIndex;not null" json:"key"`   // 既有 Key 字段 (line 11, D-06 替换)
    UserID      *string    `gorm:"type:uuid" json:"userId,omitempty"`         // nullable FK 模式
    // ...
}
```

**BaseModel 字段集**（`base.go:11-19`）：
```go
type BaseModel struct {
    ID        string         `gorm:"type:uuid;primary_key" json:"id"`
    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
    CreatedBy string         `gorm:"size:64" json:"createdBy"`
    UpdatedBy string         `gorm:"size:64" json:"updatedBy"`
    Version   int            `json:"version"`
}
```

**改造后目标形态**（D-06）：
```go
type APIKey struct {
    BaseModel
    Name         string     `gorm:"size:100;not null" json:"name"`
    // REMOVED: Key 字段 (D-06 不再明文存储)
    KeyHash      string     `gorm:"size:64;uniqueIndex;not null" json:"-"`       // SM3 hex 64 字符
    Salt         string     `gorm:"size:32;not null" json:"-"`                   // 16 字节随机 hex
    KeyPrefix    string     `gorm:"size:12;index;not null" json:"keyPrefix"`     // 前 12 字符用于 List 搜索
    UserID       *string    `gorm:"type:uuid" json:"userId,omitempty"`
    ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
    LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
    IsActive     bool       `gorm:"default:true" json:"isActive"`
    Scopes       []string   `gorm:"type:jsonb;serializer:json" json:"scopes"`
    IPWhitelist  []string   `gorm:"type:jsonb;serializer:json" json:"ipWhitelist"`
    Description  *string    `gorm:"size:500" json:"description,omitempty"`
    InheritPerms bool       `gorm:"default:false" json:"inheritPerms"`

    User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

**复用 checklist:**
- [ ] `TableName()` 不变（line 26-28）
- [ ] `uniqueIndex` 命名约定：GORM 自动生成 `sys_api_keys_key_hash_key`
- [ ] `json:"-"` 隐藏 `KeyHash`/`Salt` 不暴露前端
- [ ] `json:"keyPrefix"` 暴露 `KeyPrefix` 用于 List UI
- [ ] `KeyPrefix` 用普通 `index` 而非 `uniqueIndex`（前缀 12 字符多 key 可重复）

---

### `internal/services/system/apikey_service.go` 三函数改造（D-07/D-08/D-09）

**Analog — SM3 哈希原语 + 盐生成 + 恒定时间比对**（`internal/core/security/password.go:79-82, 104-106, 152`）：
```go
// SM3 第一次迭代 (password.go:79-82)
h := sm3.New()
h.Write(blockData)
copy(block, h.Sum(nil))

// 盐生成 (password.go:104-106)
salt := make([]byte, pm.config.SaltLength)
if _, err := rand.Read(salt); err != nil {
    return "", fmt.Errorf("生成盐失败: %w", err)
}

// 恒定时间比对 (password.go:152)
return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
```

**改造 A — `ValidateAPIKey`（line 128-161，D-08）现状 → 目标**：

现状（line 137-140）：
```go
err := s.db.WithContext(ctx).
    Preload("User").
    Where("key = ? AND is_active = ?", keyStr, true).
    First(&apiKey).Error
```

目标：缩窄候选 → 恒定时间比对：
```go
// 1) KeyPrefix 缩窄候选
var candidates []models.APIKey
err := s.db.WithContext(ctx).
    Preload("User").
    Where("key_prefix = ? AND is_active = ?", keyStr[:12], true).
    Find(&candidates).Error

// 2) 恒定时间比对 SM3
for _, candidate := range candidates {
    h := sm3.New()
    h.Write([]byte(keyStr))
    h.Write([]byte(candidate.Salt))
    computedHash := hex.EncodeToString(h.Sum(nil))
    if subtle.ConstantTimeCompare([]byte(computedHash), []byte(candidate.KeyHash)) == 1 {
        // 验证过期...异步 last_used_at...
        return &candidate, nil
    }
}
return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥不存在或已禁用")
```

**改造 B — `CreateAPIKey`（line 164-222，D-09）目标**：

```go
// D-09 一次性明文返回流程: generateKey → SM3 hash → 存三列 → 返回明文
key, err := generateKey()  // 既有 helper (line 118-126)
if err != nil {
    return nil, apperrors.Wrap(err, apperrors.CodeServerError, "密钥生成失败")
}

saltBytes := make([]byte, 16)
if _, err := rand.Read(saltBytes); err != nil {
    return nil, apperrors.Wrap(err, apperrors.CodeServerError, "盐生成失败")
}
salt := hex.EncodeToString(saltBytes)

h := sm3.New()
h.Write([]byte(key))
h.Write([]byte(salt))
keyHash := hex.EncodeToString(h.Sum(nil))

apiKey := models.APIKey{
    Name:         req.Name,
    KeyHash:      keyHash,                          // D-06 替换 Key
    Salt:         salt,
    KeyPrefix:    key[:12],                         // D-06 新增
    UserID:       &userID,
    Scopes:       req.Scopes,
    IPWhitelist:  req.IPWhitelist,
    Description:  req.Description,
    InheritPerms: req.InheritPerms,
    ExpiresAt:    expiresAt,
    IsActive:     true,
}

if err := s.db.WithContext(ctx).Create(&apiKey).Error; err != nil {
    return nil, apperrors.DatabaseError(err)
}

return &key, nil  // 一次性明文返回
```

**改造 C — `ListAPIKeys`（line 224-289，D-07）现状 → 目标**：

现状（line 238-247，含 dialect 分支）：
```go
isSQLite := s.db.Dialector.Name() == "sqlite"
if params.Keyword != nil && *params.Keyword != "" {
    keyword := "%" + *params.Keyword + "%"
    if isSQLite {
        query = query.Where("name LIKE ? OR substr(key, 1, 12) LIKE ?", keyword, keyword)
    } else {
        query = query.Where("name LIKE ? OR LEFT(key, 12) LIKE ?", keyword, keyword)
    }
}
```

目标：删除 dialect 分支，改为 KeyPrefix LIKE：
```go
if params.Keyword != nil && *params.Keyword != "" {
    keyword := "%" + *params.Keyword + "%"
    query = query.Where("name LIKE ? OR key_prefix LIKE ?", keyword, keyword)
}
```

**复用 checklist:**
- [ ] 新增 `import "github.com/tjfoc/gmsm/sm3"` + `"crypto/subtle"` 到 apikey_service.go
- [ ] 既有 `"crypto/rand"` + `"encoding/hex"` 已 import（line 5-6）
- [ ] `apperrors.Wrap` / `apperrors.DatabaseError` / `apperrors.CodeParamError` 等错误模式不变
- [ ] `isValidKeyFormat` line 79-99 + `isKeyExpired` line 71-76 既有 helpers 全部复用

---

### `internal/services/system/apikey_service_test.go`（扩展/重写）

**关键事实**：此文件**已存在**，但其 `setupTestDB` DDL（line 82-103）+ `createTestAPIKey` helper（line 209-251）+ `TestValidateAPIKey`（line 401-503）+ `TestCreateAPIKey`（line 273-398）+ `TestListAPIKeys`（line 506-621）+ `TestGetAPIKey`（line 623-665）都依赖旧 `Key` 列。D-06 改 schema 后这些测试会大规模失败——**必须按新 schema 重写 DDL + helper + 子测试断言**。

**既有 Analog 形态**（已被实测，复用而非新建，**这是同包扩展任务**）：
```go
// setupTestDB (line 37-187) — per-test 共享内存 SQLite + 裸 CREATE TABLE DDL
db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_enable_boolean=true"), &gorm.Config{
    DisableForeignKeyConstraintWhenMigrating: true,
})
require.NoError(t, err)
err = db.Exec(`
    CREATE TABLE IF NOT EXISTS sys_api_keys (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        ...
        key TEXT NOT NULL UNIQUE,          // ← D-06 后改为 key_hash TEXT NOT NULL UNIQUE, salt TEXT NOT NULL, key_prefix TEXT NOT NULL
        user_id TEXT,
        ...
    )
`).Error
require.NoError(t, err)
```

**改造 checklist（按 D-06/D-07/D-08/D-09 增量）:**
- [ ] `setupTestDB` 的 `sys_api_keys` DDL：移除 `key TEXT NOT NULL UNIQUE`，新增 `key_hash TEXT NOT NULL UNIQUE` + `salt TEXT NOT NULL` + `key_prefix TEXT NOT NULL`
- [ ] `createTestAPIKey` helper：调用 `hashAPIKey(key, "deadbeef"+<32hex>)` 构造 `KeyHash`、固定 `Salt` 字符串、`KeyPrefix = key[:12]`
- [ ] `TestValidateAPIKey`：将 `apiKey.Key` 改为构造的明文 key，断言仍走 `service.ValidateAPIKey(ctx, 明文)` → 走 KeyPrefix 缩窄 + SM3 比对
- [ ] `TestCreateAPIKey`：断言 `*key` 仍为 `rec_+64hex` 形式；DB 行 `KeyHash` 长度=64、Salt 长度=32、KeyPrefix=明文前 12 字符
- [ ] `TestListAPIKeys` "关键词搜索" 子测试：构造 key 时让 Name/KeyPrefix 都含可搜子串，断言 LIKE 跨 dialect 命中
- [ ] `TestGetAPIKey` "密钥脱敏" 子测试（line 651-665）：原断言 `len(result.Key) == 68` 已无意义（Key 字段不存在）—— 改为断言 `len(result.KeyPrefix) == 12`

---

### `internal/middleware/apikey_test.go`（扩展，D-12 单测）

**新增函数**：`TestRateLimitHeaderEncoding`

**Analog — 同文件 `t.Run` 子测试风格**（`TestIsValidKeyFormat` line 10-40）：
```go
func TestIsValidKeyFormat(t *testing.T) {
    t.Run("有效密钥格式", func(t *testing.T) {
        validKey := "rec_0123456789abcdef..."
        assert.True(t, isValidKeyFormat(validKey))
    })
    t.Run("非十六进制字符", func(t *testing.T) {
        invalidHex := "rec_..."
        assert.False(t, isValidKeyFormat(invalidHex))
    })
}
```

**目标 `TestRateLimitHeaderEncoding`**：
```go
import "strconv"  // 文件顶 import 块补充

func TestRateLimitHeaderEncoding(t *testing.T) {
    t.Run("strconv.Itoa 数字字符串化", func(t *testing.T) {
        // 验证替换 string(rune(int)) 后输出是数字字面量
        assert.Equal(t, "100", strconv.Itoa(100))
        assert.Equal(t, "99", strconv.Itoa(99))
        assert.Equal(t, "0", strconv.Itoa(0))
        // 防御性断言: 原 string(rune(100)) = "d" 不再出现
        assert.NotEqual(t, "d", strconv.Itoa(100))
    })

    t.Run("RateLimitResult 字段反序列化", func(t *testing.T) {
        // 验证 header 字符串可被 strconv.Atoi 反解析
        if n, err := strconv.Atoi(strconv.Itoa(100)); err != nil || n != 100 {
            t.Errorf("header not parseable: %v", err)
        }
    })
}
```

**复用 checklist:**
- [ ] `assert.Equal` / `assert.NotEqual` 既有模式
- [ ] `"strconv"` import 加入测试包

---

### `internal/middleware/apikey_integration_test.go`（扩展，D-12 集成）

**新增函数**：`TestRateLimitHeadersInResponse`

**Analog — 既有 `TestMultiAuthIntegration` 三路径**（line 151-233）：
```go
gin.SetMode(gin.TestMode)

db := setupUsageLoggerTestDB(t)
realLogger := services.NewUsageLogger(db)
fakeSvc := &fakeAPIKeyService{
    validKey: &models.APIKey{
        BaseModel: models.BaseModel{ID: "ak-ratelimit"},
        Name:      "rl-key",
        Scopes:    []string{"read"},
        IsActive:  true,
    },
}
rl := services.NewRateLimiter()  // Pitfall 4 防御

router := gin.New()
router.Use(MultiAuth(fakeSvc, realLogger))
router.Use(RateLimitByScope(rl))
router.GET("/ping", func(c *gin.Context) {
    c.JSON(200, gin.H{"ok": true})
})

req := httptest.NewRequest("GET", "/ping", nil)
req.Header.Set("X-API-Key", "rec_"+hex64())
w := httptest.NewRecorder()
router.ServeHTTP(w, req)

assert.Equal(t, 200, w.Code)

// 核心断言: header 是数字字面量
limitHeader := w.Header().Get("X-RateLimit-Limit")
remainingHeader := w.Header().Get("X-RateLimit-Remaining")

// 验证可被 strconv.Atoi 反解析 (D-12 QUAL-01 验证锚)
n, err := strconv.Atoi(limitHeader)
assert.NoError(t, err, "X-RateLimit-Limit 必须是数字字符串")
assert.Greater(t, n, 0)

n2, err := strconv.Atoi(remainingHeader)
assert.NoError(t, err, "X-RateLimit-Remaining 必须是数字字符串")
assert.GreaterOrEqual(t, n2, 0)

// 防御性断言: 不再是单字符 ("d" = rune(100))
assert.NotEqual(t, "d", limitHeader)
```

**复用 checklist:**
- [ ] `setupUsageLoggerTestDB` (line 111-137) Phase 59 既有 helper —— QUAL-01 集成测试**不**用真实 DB，但仍 helper 复用保证 gin.Engine 装配一致性
- [ ] `fakeAPIKeyService` (line 33-68) 9-method 全实现
- [ ] `hex64()` (line 143-145) 64 位 hex 字符串 helper
- [ ] `NewRateLimiter()` 无参版 `services.NewRateLimiter()` 必须传非 nil（**Pitfall 4 防御**）

---

### `internal/api/router_test.go`（新建，Wave 0 可选）

**关键事实**：仓库**无现成 `*_router_test.go`**（已 Glob 验证 `internal/api/router*.go` 仅 `router.go`）。最近 analog 是 `apikey_integration_test.go:TestMultiAuthIntegration` 的 gin.Engine + httptest 模式。

**目标最小化骨架**（planner 自创）：
```go
package api

import (
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

// TestAPIKeysMiddlewareChain 验证 /system/apikeys/* 路由组装配
// 了 RequirePermissions → MultiAuth → RateLimitByScope 三段 middleware (AUTH-03 / D-01)。
func TestAPIKeysMiddlewareChain(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // 构造测试 engine，复制 router.go:238-248 的 middleware 装配
    router := gin.New()
    apikeys := router.Group("/system/apikeys")
    // middlewares: RequirePermissions → MultiAuth → RateLimitByScope
    // ...
    apikeys.POST("", func(c *gin.Context) {
        c.JSON(200, gin.H{"ok": true})
    })

    req := httptest.NewRequest("POST", "/system/apikeys", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 仅验证 middleware 链顺序不 panic + 注册成功
    assert.NotPanics(t, func() {
        router.ServeHTTP(w, req)
    })
}
```

**复用 checklist:**
- [ ] 部分：参考 `apikey_integration_test.go` 的 gin.Engine + httptest 模式
- [ ] 此 file 是 Wave 0 可选项，planner 可选定 deferred（如不为 SEC-02 阻塞可省）

---

### `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`（新建，D-10）

**Analog A — DROP INDEX 注释 + IF EXISTS 风格**（`internal/core/db/migrations/archive/020_drop_mac_unique_index.sql` 全文件）：
```sql
-- 删除MAC地址表的唯一索引
-- 迁移编号: 020
-- 描述: 删除sys_device_mac_address表的唯一索引，允许同一MAC地址在多次采集中存在

-- 删除唯一索引（如果存在）
DROP INDEX IF EXISTS idx_sys_device_mac_address_unique;

-- 说明：
-- 1. 历史数据中存在重复的MAC地址记录
-- 2. 同一设备接口可能在不同时间采集到相同的MAC地址
-- 3. 保留普通索引 idx_sys_device_mac_address_composite 以保证查询性能
```

**Analog B — 重复索引清理风格**（`internal/core/db/migrations/archive/legacy-2026-06-15/027_cleanup_duplicate_indexes.sql:13-31`，**P2-A4 source-tracking** 也建议在此查看）：
```sql
-- ============================================
-- 知识库分类表 - 清理重复的 category_name 唯一索引
-- ============================================
-- 删除旧的唯一约束（如果存在）
ALTER TABLE sys_knowledge_category DROP CONSTRAINT IF EXISTS uk_knowledge_category_name;
-- 删除旧的唯一索引（如果存在）
DROP INDEX IF EXISTS uk_knowledge_category_name;
```

**目标 SEC-02 SQL 文件**：
```sql
-- Phase 60 / SEC-02: 移除 migration_085 遗留的冗余索引 idx_api_keys_key
-- 原因: 与 sys_api_keys.key 列的 uniqueIndex 重复 (migration_085 创建后 GORM AutoMigrate 又生成 sys_api_keys_key_hash_key, 二者等价)
-- 风险: 无 — uniqueIndex 仍保留, 索引扫描性能不变
-- 回滚: CREATE INDEX idx_api_keys_key ON sys_api_keys(key);
-- 幂等: IF EXISTS 关键字保证重复跑安全
-- 验证查询: 见 .planning/notes/260813-sec02-redundant-index-removal.md

DROP INDEX IF EXISTS idx_api_keys_key;
```

**复用 checklist:**
- [ ] 单语句 `DROP INDEX IF EXISTS ...;`
- [ ] 文件顶 7-行注释块（原因/风险/回滚/幂等/验证引用）
- [ ] 不需要 Go migration runner 触发（D-10 显式偏离 migration_085 `//go:build archive_skip` 模式）

---

### `.planning/notes/260813-*.md`（新建，3 个决策/验证文档）

**Analog — 既有 notes 文件结构**（`.planning/notes/260615-oper-log-coverage-audit.md:1-39`）：
```markdown
# 操作日志全模块覆盖度审计（修订版）

**审计日期**: 2026-06-15
**修订**: v2 - 重大修正
**审计范围**: XingRan-Next Go 后端所有业务模块
**审计目的**: 为 Phase 34 "操作日志全模块集成" 提供基线数据

---

## 1. 现有操作日志基础设施

### 1.1 存储结构

| 组件 | 路径 | 说明 |
|------|------|------|
| 数据模型 | `internal/models/log.go:5-24` | `OperLog` struct，表名 `sys_oper_log` |
| ... |  |  |

---

## 2. ...
```

**目标 notes 文件通用骨架**：
```markdown
# Phase 60 / {AUTH-03 | SEC-01 | SEC-02}: {决策标题}

**记录日期**: 2026-08-13
**关联 ROADMAP**: AUTH-03 / SEC-01 / SEC-02
**决策上下文**: 详见 `.planning/phases/60-.../60-CONTEXT.md` D-{NN}

---

## 维度 1

{内容}

## 维度 2

{内容}
```

**3 个 notes 文件专属结构：**

| 文件 | 段落结构（VALIDATION.md SC#1-4 规定） |
|------|--------------------------------------|
| `260813-auth03-enable-decision.md` | 「挂载范围 / 认证优先级 / IP 白名单 / JWT 回退」4 维度（对应 D-01/D-02/D-03/D-04） |
| `260813-sec01-hash-migration.md` | 「存储方案 / Schema / List 搜索 / 迁移路径 / 创建流程」5 维度（对应 D-05/D-06/D-07/D-08/D-09） |
| `260813-sec02-redundant-index-removal.md` | 「为什么 / 怎么跑 SQL / 验证查询（PG `pg_indexes` + SQLite `sqlite_master` 双 fragment）/ AutoMigrate 行为说明」 |

**复用 checklist:**
- [ ] 文件名前缀 `260813-` 沿用 `260615-`/`260616-`/`260627-`/`260630-`/`260703-` 等既有 notes 命名
- [ ] frontmatter 单 `**记录日期**` 行即可（不强求 frontmatter YAML，因仓库 notes 文件风格混合）
- [ ] 段间空一行 + 粗体二级标题 `##`
- [ ] 跨文件链接用相对路径 `.planning/phases/60-.../60-CONTEXT.md`

---

## Shared Patterns

### SM3 哈希签名（hex 64 字符，无前缀）

**Source:** `internal/core/security/password.go:79-82` (`sm3.New()`)
**Apply to:** `internal/services/system/apikey_service.go` 新增 helper `hashAPIKey(key, salt) string`

```go
// 既有 sm3 用法 (password.go:79-82)
h := sm3.New()
h.Write(blockData)
copy(block, h.Sum(nil))
```

**包装 helper 模板**：
```go
func hashAPIKey(key, salt string) string {
    h := sm3.New()
    h.Write([]byte(key))
    h.Write([]byte(salt))
    return hex.EncodeToString(h.Sum(nil))
}
```

**为什么不复用 `HashPassword`?** 那个返回 `$sm3$iterations$salt$hash` 格式（含 600k PBKDF2 iterations）。API Key 是 `crypto/rand` 32 字节高熵，不需要拉伸（PBKDF2 过度设计）。Phase 60 选单次 SM3 仅 hex 64 字符，避免与 `HashPassword` 格式混淆（D-05+D-09 Claude's Discretion）。

---

### crypto/rand 16 字节 Salt 生成

**Source:** `internal/core/security/password.go:104-106`
**Apply to:** `apikey_service.go` `generateSalt()` helper

```go
// 既有 (password.go:104-106)
salt := make([]byte, pm.config.SaltLength)
if _, err := rand.Read(salt); err != nil {
    return "", fmt.Errorf("生成盐失败: %w", err)
}
```

**包装 helper 模板**：
```go
func generateSalt() (string, error) {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return "", fmt.Errorf("生成盐失败: %w", err)
    }
    return hex.EncodeToString(bytes), nil
}
```

---

### crypto/subtle.ConstantTimeCompare 恒定时间比对

**Source:** `internal/core/security/password.go:152`
**Apply to:** `apikey_service.go` `ValidateAPIKey` SM3 比对路径

```go
// 既有 (password.go:152)
return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
```

**Apply 形态**：
```go
// ValidateAPIKey 循环内
computedHash := hashAPIKey(keyStr, candidate.Salt)
if subtle.ConstantTimeCompare([]byte(computedHash), []byte(candidate.KeyHash)) == 1 {
    return &candidate, nil
}
```

---

### sqlite per-test 独立 DB helper（Phase 59 模式）

**Source:** `internal/middleware/apikey_integration_test.go:111-137` (`setupUsageLoggerTestDB`)
**Apply to:** `internal/services/system/apikey_service_test.go` 重写 / `apikey_integration_test.go` QUAL-01 集成测试

```go
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
    dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db", time.Now().UnixNano(), os.Getpid()))
    dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    require.NoError(t, err)
    err = db.Exec(`CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (...)`).Error
    require.NoError(t, err)
    return db
}
```

**Why this pattern:**
- per-test 文件 DB（`os.TempDir` + `UnixNano` + `pid` 唯一名），测试间零共享
- `busy_timeout=5000` 让写锁排队而非立即报错
- 不用 `t.TempDir`（fire-and-forget goroutine 仍写文件，cleanup 删占用文件 mark test failed）

**`apikey_service_test.go` 的 DDL 改造 checklist（SEC-01 落地）**：
- [ ] `setupTestDB` 的 `sys_api_keys` DDL：`key TEXT NOT NULL UNIQUE` → `key_hash TEXT NOT NULL UNIQUE` + `salt TEXT NOT NULL` + `key_prefix TEXT NOT NULL`
- [ ] `setupTestDB` 必须保留既有用法（"`file::memory:?cache=shared&_enable_boolean=true`"）—— **不**改用 file DB（既有用得好好的，Phase 60 不引入新变体）
- [ ] `createTestAPIKey` helper：直接构造 `Salt: "0123..."32 hex...`, `KeyHash: hashAPIKey(rec_+64hex, Salt)`，`KeyPrefix: key[:12]`

---

### gin.Engine + httptest 集成测试模式

**Source:** `internal/middleware/apikey_integration_test.go:167-186`
**Apply to:** `apikey_integration_test.go` QUAL-01 集成测试, 可选 `internal/api/router_test.go`

```go
gin.SetMode(gin.TestMode)
router := gin.New()
router.Use(MultiAuth(fakeSvc, realLogger))
router.GET("/ping", func(c *gin.Context) {
    c.JSON(200, gin.H{"ok": true})
})
req := httptest.NewRequest("GET", "/ping", nil)
req.Header.Set("X-API-Key", "rec_"+hex64())
w := httptest.NewRecorder()
router.ServeHTTP(w, req)
assert.Equal(t, 200, w.Code)
```

**Apply to QUAL-01 集成测试**：
```go
router.Use(RateLimitByScope(rl))  // ★ 新增挂载
limitHeader := w.Header().Get("X-RateLimit-Limit")
n, err := strconv.Atoi(limitHeader)
assert.NoError(t, err)
```

---

### response 包调用约定

**Source:** `pkg/response/*.go` + `internal/api/v1/system/apikey_handler.go`
**Apply to:** Phase 60 不修改 handler，无需新增 error wrapping；SEC-01 service 改造沿用既有 `apperrors.Wrap` / `apperrors.DatabaseError`

```go
// 既有 service error wrapping (apikey_service.go:131-133)
return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "无效的密钥格式")

// 既有 DB error wrap
return nil, apperrors.DatabaseError(err)
```

**SEC-01 沿用**: D-08 ValidateAPIKey 失败仍 `apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥不存在或已禁用")`（不变）。

---

### operlog.Record 写入约定（CLAUDE.md 强制）

**Source:** `internal/api/v1/system/apikey_handler.go:69-73` + `internal/utils/operlog/operlog.go:80-100`
**Apply to:** Phase 60 **不修改 handler**，但需确认改造后 handler 仍合规

```go
// 既有 (apikey_handler.go:71) — 创建密钥是敏感路径
operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeCreate)
response.Success(c, gin.H{"key": *key})
```

**Phase 60 后的合规验证点**：
- [ ] `operlog.RecordWithBody` 仍位于 `success path 末尾，response.Success 之前`
- [ ] 模块中文名仍是 "API密钥管理"
- [ ] `OperTypeCreate`/`OperTypeUpdate`/`OperTypeDelete`/`OperTypeStatus` 不变
- [ ] KeyPrefix **不入 operlog**（请求体本就无 KeyPrefix，operlog 读 `c.GetRawData()`，无 KeyPrefix 即无记录）—— Pitfall 7 描述的设计意图
- [ ] D-06 后请求体仍不含 `key` 字段（前端 request 只传 Name/Scopes/IPWhitelist 等），operlog 关键词脱敏无变化

---

## No Analog Found

| 文件 | 角色 | 数据流 | 原因 |
|------|------|--------|------|
| `internal/api/router_test.go` | test | request-response | 仓库无现成 `internal/api/*_router_test.go`（已 Glob 验证），可参考 `apikey_integration_test.go` 但需自创 router-test harness；属 Wave 0 可选项，可由 planner 决定 skip |
| `ListAPIKeys` 的 `KeyPrefix LIKE` 改造 | service change | CRUD | 无跨文件 analog——是 `apikey_service.go` 函数内的行内 diff（D-07 直接删除 dialect 分支即可，参照同函数 line 238-247 现状） |

**关键提示**：`internal/services/system/apikey_service_test.go` 是**扩展任务**，不是新建任务——文件已存在（D-06 schema 变更需重写 DDL + helper + 多个子测试断言），planner 应意识到这相当于"拆 + 改"而非"从零写"。

---

## Metadata

**Phase:** 60 - 安全加固与启用决策
**Mapped date:** 2026-08-13
**Phase directory:** `.planning/phases/60-security-hardening-and-enable-decision/`
**Files analyzed:** 12 个改动点
**Analogs found:** 11 / 12（1 项 No Analog：router-test 自创）
**Analog search scope:**
- `internal/api/` （router.go, v1/system/apikey_router.go）
- `internal/middleware/` （apikey.go + apikey_test.go + apikey_integration_test.go）
- `internal/models/` （api_key.go + base.go）
- `internal/services/system/` （apikey_service.go + apikey_service_test.go + user_service_test.go）
- `internal/services/` （rate_limiter.go + usage_logger.go + cache_config_service.go）
- `internal/core/security/` （password.go）
- `internal/api/v1/system/` （apikey_handler.go + helper.go）
- `internal/core/db/migrations/` （archive/020_drop_mac_unique_index.sql + legacy-2026-06-15/027_cleanup_duplicate_indexes.sql）
- `.planning/notes/` （260615-oper-log-coverage-audit.md）

**Files scanned:** ~50 个 source/test/SQL/notes（详见上）

**Decisions locked:** D-01..D-13（详见 `.planning/notes/260813-*` 三份待建文件）

**Claude's Discretion applied:**
- Salt 长度：**16 字节 / 32 hex 字符**（与 `DefaultPasswordConfig.SaltLength=16` 一致）
- SM3 哈希格式：**仅 hex 64 字符无前缀**（不复用 `$sm3$iterations$salt$hash` 格式）
- 验证查询：**PG `pg_indexes` + SQLite `sqlite_master` 双 fragment**
- 新 schema 列位置：**BaseModel 之后，业务字段 Name 紧邻（替换原 `Key` 字段位）**
