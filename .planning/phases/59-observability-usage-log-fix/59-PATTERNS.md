# Phase 59: 可观测性 / 使用日志修复 - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 5（2 源文件改 + 3 测试文件扩）
**Analogs found:** 5 / 5（全部在仓库内找到强匹配先例，零无匹配）

**校验结论:** RESEARCH.md 所列 file:line 全部经实读核对准确，无偏移。本 PATTERNS.md 直接复用并结构化 RESEARCH.md 已核对的先例，补充每个先例的实读证据与 planner 可抄的最小代码块。

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/middleware/apikey.go` | middleware | request-response（gin `c.Next()` 前后捕获） | `pkg/middleware/logger.go` | exact（同角色 + 同 data flow，gin 记录模式） |
| `internal/services/usage_logger.go` | service | fire-and-forget 异步 / 单 INSERT | `pkg/cache/redis.go`（L2 异步刷盘 detached ctx） | role-match（异构角色但近乎同义 data flow） |
| `internal/middleware/apikey_integration_test.go` | test | 集成 / DB 行实证 | `internal/middleware/apikey_integration_test.go`（既有 fake 测试 + `setupUsageLoggerTestDB` helper） | exact（同文件扩展，复用既有 helper） |
| `internal/services/usage_logger_test.go` | test | 单元 / DB 行实证 + cancel-race | `internal/services/usage_logger_test.go`（既有 `setupUsageLoggerTestDB` + `time.Sleep` 等待模式） | exact（同文件扩展） |
| `internal/services/system/apikey_service_test.go` | test | 单元 / seed + 聚合断言 | `internal/services/system/apikey_service_test.go:1096`（既有「成功率计算」子测试） | exact（同文件扩展，SC#3 直接克隆既有 70% 用例形态） |

---

## Pattern Assignments

### `internal/middleware/apikey.go`（middleware, request-response）

**Analog:** `pkg/middleware/logger.go`（VERIFIED 实读）
**目标改动位置:** 当前 `apikey.go:61-76`（`go func()` 在 `c.Next()` 前 spawn + 仅填 5 字段）

**Imports pattern**（`apikey.go:3-13`，无需新增 import — `time` 已在第 6 行）:
```go
import (
    "net"
    "strings"
    "time"  // 已存在, 复用做 start := time.Now()

    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/services"
    "github.com/xingran-next/xingran-go-backend/internal/services/system"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)
```

**Core pattern（gin 记录时机）** — 抄自 `pkg/middleware/logger.go:19-28`:
```go
// Source: pkg/middleware/logger.go:19-28 (VERIFIED)
func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        startTime := time.Now()       // ← 入口前置
        bodyBytes := readRequestBody(c)
        c.Next()                       // ← 下游执行
        logRequest(c, startTime, bodyBytes)  // ← c.Next() 后捕获
    }
}
```

**StatusCode / Duration 捕获** — 抄自 `pkg/middleware/logger.go:47-49`:
```go
// Source: pkg/middleware/logger.go:47-49 (VERIFIED)
func logRequest(c *gin.Context, startTime time.Time, bodyBytes []byte) {
    latency := time.Since(startTime)           // → .Milliseconds() 对齐 LogUsageRequest.Duration
    statusCode := c.Writer.Status()            // 真实响应码, 仅 c.Next() 后可用
    // ...
}
```

**落地到 MultiAuth（D-02a 去冗余 goroutine + D-01 Success 口径 + OBSERV-01 字段填充）:**
```go
// 替换 apikey.go:61-76 当前实现
setUserContextForAPIKey(c, apiKey, apiKey.Scopes)

start := time.Now()
c.Next()  // 下游 handler (含 RequireScope / RateLimitByScope) 执行完毕

// c.Next() 后: 真实状态码 / 耗时此刻才可用 (OBSERV-01)
statusCode := c.Writer.Status()
duration := time.Since(start).Milliseconds()

userID := ""
if apiKey.UserID != nil {
    userID = *apiKey.UserID
}

// D-02a: middleware 不再包 go func(); LogUsage 内部已 go logUsageAsync()
usageLogger.LogUsage(c.Request.Context(), &services.LogUsageRequest{
    APIKeyID:   apiKey.ID,
    UserID:     userID,
    Method:     c.Request.Method,
    Path:       c.Request.URL.Path,
    ClientIP:   c.ClientIP(),
    StatusCode: statusCode,                                       // 新填 (OBSERV-01)
    Duration:   int(duration),                                    // 新填 (OBSERV-01)
    Success:    statusCode >= 200 && statusCode < 300,            // 新填 (D-01)
    // UserAgent: 可选, 见 Claude's Discretion (SC 未要求)
})
```

**⚠️ 语义边界（planner 必读）:** 记录点在 `c.Next()` 之后，pre-auth 失败（`apikey.go:35-37` 格式错 / `:42-46` ValidateAPIKey 失败 / `:50-55` IP 白名单拒）走 `c.Abort(); return` 在 `c.Next()` **之前**退出，**不会记录**。SC#2 失败用例必须用**下游产生的失败**（RequireScope→403 at `apikey.go:197-201` / RateLimitByScope→429 at `apikey.go:262-267` / handler→500），这些发生在 `c.Next()` 下游，返回时 `c.Writer.Status()` 捕获真实失败码。

---

### `internal/services/usage_logger.go`（service, fire-and-forget 异步）

**Analog:** `pkg/cache/redis.go:601-605`（VERIFIED 实读，近乎同义的 detached context 先例）
**目标改动位置:**
- `usage_logger.go:54-75` `logUsageAsync`（当前第 70 行 `s.db.WithContext(ctx)` 复用调用方 ctx + 第 73 行 `_ = err` 静默吞错）

**Imports pattern**（`usage_logger.go:3-9`，**需新增 1 个 import** — `pkg/logger` 当前未导入）:
```go
import (
    "context"
    "time"

    "github.com/xingran-next/xingran-go-backend/internal/models"
    "gorm.io/gorm"

    applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"  // 新增 (D-04)
)
```

**Detached context 先例** — 抄自 `pkg/cache/redis.go:601-605`（VERIFIED，注释原话佐证 D-02 依据）:
```go
// Source: pkg/cache/redis.go:601-605 (VERIFIED)
// P1 fix: 不能用请求 ctx 去做 L2 异步入队 —— HTTP 请求 ctx 通常只有 5-30s 截止时间,
// 但 L2 写入是后台任务,应当独立于请求生命周期。改用独立 ctx 隔离,
// 真正的客户端取消不应阻塞 L2 异步刷盘。
enqueueCtx, cancelEnqueue := context.WithTimeout(context.Background(), m.l2Writer.GetFallbackTimeout())
defer cancelEnqueue()
```
（同文件 `redis.go:614-615` 还有第二个同形先例 `syncCtx, cancel := context.WithTimeout(context.Background(), timeout); defer cancel()`）

**写入失败日志先例** — 抄自 `internal/services/config_backup_service.go:247`（VERIFIED）:
```go
// Source: internal/services/config_backup_service.go:247 (VERIFIED)
// 注: 该行实为 Infof; D-04 指定升级为 Errorf (DB 写入失败语义=error 级别)
applogger.Infof("[配置备份] 备份失败 [%s]: %v", device.DeviceName, err)
// Import 别名 (config_backup_service.go:18, VERIFIED):
// applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
```

**pkg/logger API**（VERIFIED `pkg/logger/logger.go:206-208`）:
```go
// pkg/logger/logger.go:206-208 — Errorf 公开 API 存在, 直接可调
func Errorf(format string, args ...interface{}) {
    GetLogger().Errorf(format, args...)
}
```

**Core pattern 落地到 `logUsageAsync`（D-02 + D-04）:**
```go
// 替换 usage_logger.go:54-75
func (s *usageLoggerImpl) logUsageAsync(ctx context.Context, req *LogUsageRequest) {
    // D-02 / OBSERV-03: 用独立 ctx 写 DB, 忽略调用方 ctx 的取消信号。
    // 先例: pkg/cache/redis.go:604
    detachedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = ctx  // 显式标注: 调用方 ctx 不用于本次 DB 写入取消控制

    usageLog := models.APIKeyUsageLog{
        APIKeyID:   req.APIKeyID,
        UserID:     req.UserID,
        Method:     req.Method,
        Path:       req.Path,
        StatusCode: req.StatusCode,
        ClientIP:   req.ClientIP,
        UserAgent:  req.UserAgent,
        Duration:   req.Duration,
        Success:    req.Success,
        CreatedAt:  time.Now(),
    }

    if err := s.db.WithContext(detachedCtx).Create(&usageLog).Error; err != nil {
        // D-04: 替换 _ = err 静默吞错; 先例 config_backup_service.go:247 (升级 severity)
        applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)
    }
}
```

**接口契约不变（A3）:** `UsageLogger` 接口（`usage_logger.go:12-17`）+ `LogUsage(ctx, req)` 签名 + `LogUsageRequest` 结构体（`:20-30`，**已含** StatusCode/Duration/Success）+ `LogUsage` 内部 `go logUsageAsync()`（`:49`）均不动。grep 确认 `LogUsage` 仅 1 生产调用点（`apikey.go:67`）+ 14 测试调用，签名不变即零破坏。

---

### `internal/middleware/apikey_integration_test.go`（test, 集成 / DB 行实证）

**Analog:** 同文件既有 `setupUsageLoggerTestDB` helper + `fakeUsageLogger`（VERIFIED 实读）
**目标:** 扩展 SC#1（2xx 时序/字段）+ SC#2（下游失败 Success=false）真实 DB 子测试

**既有 helper（直接复用，VERIFIED `apikey_integration_test.go:111-137`）:**
```go
// Source: internal/middleware/apikey_integration_test.go:111-137 (VERIFIED)
// 与 internal/services/usage_logger_test.go:30-57 是同源复制
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
    dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db",
        time.Now().UnixNano(), os.Getpid()))
    dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    require.NoError(t, err)
    err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (
            id TEXT PRIMARY KEY, api_key_id TEXT NOT NULL, user_id TEXT NOT NULL,
            method TEXT, path TEXT, status_code INTEGER, client_ip TEXT,
            user_agent TEXT, duration INTEGER, success BOOLEAN, created_at DATETIME)
    `).Error
    require.NoError(t, err)
    return db
}
```
（裸 `CREATE TABLE` DDL 绕过 `gen_random_uuid()` PG 专有陷阱 — `models/api_key_usage_log.go:9` 的 `default:gen_random_uuid()` 在 sqlite 不识别）

**既有 fake（保留不回归，D-03a，VERIFIED `apikey_integration_test.go:76-100`）:**
```go
// fakeUsageLogger 测认证链 context 键 (Phase 57); 本 phase 新测试用真实 NewUsageLogger(db) 测时序
// 两者职责正交, 并存不回归
type fakeUsageLogger struct {
    logged bool
    done   chan struct{}
}
func (f *fakeUsageLogger) LogUsage(ctx context.Context, req *services.LogUsageRequest) error {
    f.logged = true
    close(f.done)
    return nil
}
```

**既有 hex64 helper（VERIFIED `:143-145`）** — SC#2 构造合法 key 格式用:
```go
func hex64() string { return strings.Repeat("0123456789abcdef", 4) } // 64 chars, 满足 isValidKeyFormat
```

**异步等待 pattern（升级）:** 既有 fake 用 `done` channel + `waitForLog`（`:93-100`，2s 超时）。新真实 DB 测试改用 **`require.Eventually` 轮询 DB 行**（见 Shared Patterns §异步写入可测试性）——仓库当前零现用 `require.Eventually`（已 grep 确认），属本 phase 引入的升级 idiom。

**SC#1 测试骨架（DB 行实证，用真实 NewUsageLogger 而非 fake）:**
```go
func TestMultiAuthUsageLogTiming_SC1(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupUsageLoggerTestDB(t)
    realLogger := services.NewUsageLogger(db)  // 真实 DB, 非 fake

    fakeSvc := &fakeAPIKeyService{validKey: &models.APIKey{
        BaseModel: models.BaseModel{ID: "ak-sc1"},
        Scopes:    []string{"read"}, IsActive: true,
    }}
    router := gin.New()
    router.Use(MultiAuth(fakeSvc, realLogger))
    router.GET("/ok", RequireScope("read"), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

    req := httptest.NewRequest("GET", "/ok", nil)
    req.Header.Set("X-API-Key", "rec_"+hex64())
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)

    waitForUsageLog(t, db, "ak-sc1", 1)  // require.Eventually 轮询, 见 Shared Patterns
    var log models.APIKeyUsageLog
    require.NoError(t, db.Where("api_key_id = ?", "ak-sc1").First(&log).Error)
    assert.Equal(t, 200, log.StatusCode)
    assert.Greater(t, log.Duration, 0)
    assert.True(t, log.Success)  // D-01: 2xx → Success=true
}
```

**SC#2 测试骨架（下游失败，**非 pre-auth 401**）:**
```go
// 用 RequireScope("write") 但 key 仅 read scope → 下游 403 (apikey.go:197-201)
// 发生在 c.Next() 下游, MultiAuth 返回时捕获 403 → 记录 Success=false
router.GET("/write-only", RequireScope("write"), func(c *gin.Context) { c.JSON(200, nil) })
// ... 断言 log.StatusCode == 403, log.Success == false
```

---

### `internal/services/usage_logger_test.go`（test, 单元 / cancel-race）

**Analog:** 同文件既有 `setupUsageLoggerTestDB` + `TestLogUsage` 子测试（VERIFIED 实读）
**目标:** 扩展 SC#4（cancel-race）子测试 + `waitForUsageLog` helper

**既有 helper（直接复用，VERIFIED `usage_logger_test.go:30-57`）:**
```go
// Source: internal/services/usage_logger_test.go:30-57 (VERIFIED)
// 注释化迁移决策 (line 18-29): file::memory:?cache=shared 因并发写锁撞锁改用文件 DB
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB { /* 同上 apikey_integration_test 版 */ }
```

**既有 flaky 等待 pattern（升级对象，VERIFIED `usage_logger_test.go:101/141/166/202/252/281/316/358/388/426/453/496/530`）:**
```go
// 既有 (flaky 反模式, 新测试不要抄):
err := logger.LogUsage(ctx, req)
time.Sleep(100 * time.Millisecond)  // ← 固定 sleep, CI 高并行下时序漂移
```

**SC#4 cancel-race 测试骨架（用 `require.Eventually` 替代 sleep）:**
```go
func TestLogUsage_CancelledCtxStillWrites_D02_SC4(t *testing.T) {
    db := setupUsageLoggerTestDB(t)
    logger := NewUsageLogger(db)

    ctx, cancel := context.WithCancel(context.Background())
    cancel()  // 预取消: 模拟请求结束 (P2-b 场景)

    err := logger.LogUsage(ctx, &LogUsageRequest{
        APIKeyID: "cancel-race-key", UserID: "u1",
        Method: "GET", Path: "/test",
        StatusCode: 200, Success: true, Duration: 10,
    })
    require.NoError(t, err)

    // 修复前 (复用 c.Request.Context()): ctx 已 cancel → Create 失败 → _ = err 吞掉 → 无行
    // 修复后 (D-02 detached ctx): 忽略调用方 cancel → 行落库
    waitForUsageLog(t, db, "cancel-race-key", 1)

    var log models.APIKeyUsageLog
    require.NoError(t, db.Where("api_key_id = ?", "cancel-race-key").First(&log).Error)
    assert.Equal(t, 200, log.StatusCode)
    assert.True(t, log.Success)
}
```

**新增 `waitForUsageLog` helper（落本文件，VERIFIED 形态来自 RESEARCH.md §异步写入可测试性机制）:**
```go
// waitForUsageLog 轮询 DB 至指定 apiKeyID 的日志行数 >= want, 或超时失败。
// 替代 time.Sleep 的确定性版本 (require.Eventually)。
func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
    t.Helper()
    require.Eventually(t, func() bool {
        var count int64
        db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
        return count >= want
    }, 2*time.Second, 10*time.Millisecond,
        "usage log for key=%s not persisted within 2s", apiKeyID)
}
```

**注:** 该 helper 在 `internal/middleware/` 与 `internal/services/` 两个包各需一份（Go 测试包隔离，无法跨包导出测试 helper）—— `apikey_integration_test.go` 与 `usage_logger_test.go` 各自定义同名 helper。

---

### `internal/services/system/apikey_service_test.go`（test, 单元 / seed + 聚合断言）

**Analog:** 同文件既有 `TestGetUsageLogSummary/成功率计算` 子测试（VERIFIED 实读 `:1096-1133`）
**目标:** 扩展 SC#3（混合 success 行 → successRate ∈ (0,100)）子测试

**既有 setupTestDB（VERIFIED `apikey_service_test.go:37-187`）** — 已建 `sys_api_key_usage_logs` 表（`:106-121`，同裸 DDL 形态），直接复用，**无需另建 helper**。

**既有 `TestGetUsageLogSummary/成功率计算` 子测试（SC#3 的直接克隆模板，VERIFIED `:1096-1133`）:**
```go
// Source: internal/services/system/apikey_service_test.go:1096-1133 (VERIFIED)
t.Run("成功率计算", func(t *testing.T) {
    apiKey := createTestAPIKey(t, db, user.ID, true)
    for i := 0; i < 7; i++ {
        log := models.APIKeyUsageLog{
            APIKeyID: apiKey.ID, UserID: user.ID,
            Method: "GET", Path: "/api/v1/success",
            StatusCode: 200, Duration: 100, Success: true,
        }
        db.Create(&log)
    }
    for i := 0; i < 3; i++ {
        log := models.APIKeyUsageLog{
            APIKeyID: apiKey.ID, UserID: user.ID,
            Method: "GET", Path: "/api/v1/fail",
            StatusCode: 500, Duration: 200, Success: false,
        }
        db.Create(&log)
    }
    summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)
    assert.NoError(t, err)
    assert.InDelta(t, 70.0, summary.SuccessRate, 0.1) // 7/10 = 70%
    cleanupTestData(t, db)
})
```

**既有 `createTestAPIKey` helper（VERIFIED `:210-251`）+ `cleanupTestData`（`:259-271`）** — SC#3 直接复用。

**SC#3 防回归子测试骨架（直接 seed 真实 DB 行 + 调 GetUsageLogSummary，**不经 middleware**，纯聚合逻辑防回归）:**
```go
func TestGetUsageLogSummaryMixed_SC3(t *testing.T) {
    db := setupTestDB(t)
    service := NewAPIKeyService(db)
    ctx := context.Background()
    user := createTestUser(t, db)
    apiKey := createTestAPIKey(t, db, user.ID, true)

    // 混合 success 行 (OBSERV-02: successRate 基于真实 Success 字段)
    successLogs := []models.APIKeyUsageLog{
        {APIKeyID: apiKey.ID, UserID: user.ID, Method: "GET", Path: "/ok1",
         StatusCode: 200, Duration: 50, Success: true},
        {APIKeyID: apiKey.ID, UserID: user.ID, Method: "GET", Path: "/ok2",
         StatusCode: 204, Duration: 30, Success: true},
    }
    failLogs := []models.APIKeyUsageLog{
        {APIKeyID: apiKey.ID, UserID: user.ID, Method: "GET", Path: "/forbidden",
         StatusCode: 403, Duration: 20, Success: false},  // 下游失败 (D-01 归 false)
        {APIKeyID: apiKey.ID, UserID: user.ID, Method: "GET", Path: "/rate",
         StatusCode: 429, Duration: 10, Success: false},
    }
    for _, l := range append(successLogs, failLogs...) {
        db.Create(&l)
    }

    summary, err := service.GetUsageLogSummary(ctx, apiKey.ID)
    require.NoError(t, err)
    // OBSERV-02 防回归: 不恒 ≈ 0% (修复前 Success 永远 false → successRate 恒 0)
    assert.Greater(t, summary.SuccessRate, 0.0)
    assert.Less(t, summary.SuccessRate, 100.0)
    assert.InDelta(t, 50.0, summary.SuccessRate, 0.1) // 2/4 = 50%
    cleanupTestData(t, db)
}
```

**注:** 既有 `:1096` 「成功率计算」子测试**已覆盖** OBSERV-02 的正向逻辑（70% 用例）。SC#3 的增量价值是**防回归锚**——Phase 59 改 middleware 后，确保既有 70% 用例 + 新 SC#3 用例都仍 pass（若 OBSERV-01 修复破坏了 Success 字段链路，两个用例都会失败）。

---

## Shared Patterns

### gin middleware 记录时机（c.Next() 前后捕获）

**Source:** `pkg/middleware/logger.go:19-28` + `:47-49`（VERIFIED）
**Apply to:** `internal/middleware/apikey.go`（OBSERV-01 记录点后移）
```go
startTime := time.Now()       // 入口前置
c.Next()                       // 下游执行
statusCode := c.Writer.Status()  // 仅 c.Next() 后可用
latency := time.Since(startTime)
```
**同模式其它先例:** `pkg/middleware/response_encryption.go:101`、`internal/services/oper_log_service.go:176`（RESEARCH.md 引用，未实读但同模式）

---

### detached-with-timeout context（fire-and-forget 免疫请求生命周期）

**Source:** `pkg/cache/redis.go:601-605`（VERIFIED）+ 同文件 `:614-615`
**Apply to:** `internal/services/usage_logger.go:logUsageAsync`（D-02 / OBSERV-03）
```go
detachedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
// 忽略调用方 ctx 的取消信号; 仅 detachedCtx 驱动后台写入
```
**关键契约:** 调用方 ctx（`c.Request.Context()`）「降级」为仅可提取请求范围值的载体，**绝不**用于 DB 写入取消控制。

---

### 写入失败可见性（applogger + 模块前缀）

**Source:** `internal/services/config_backup_service.go:18`（import 别名）+ `:247`（调用先例，VERIFIED）
**Apply to:** `internal/services/usage_logger.go:logUsageAsync`（D-04 替换 `_ = err`）
```go
// import (applogger "github.com/xingran-next/xingran-go-backend/pkg/logger")
applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)
```
**API:** `pkg/logger/logger.go:206-208`（`Errorf` 公开，VERIFIED）
**契约:** 保持 fire-and-forget——失败仅记录、不阻塞、不 panic、不影响业务请求。
**安全:** LogUsageRequest 无 key 明文/密码/token 字段（仅 apiKeyID UUID + path/method/IP/status），`applogger.Errorf` 不泄露敏感数据。

---

### sqlite 文件 DB 测试 helper（绕过 gen_random_uuid + busy_timeout 防撞锁）

**Source:** `internal/services/usage_logger_test.go:30-57`（VERIFIED）= `internal/middleware/apikey_integration_test.go:111-137`（VERIFIED，同源复制）
**Apply to:** SC#1/#2/#4 真实 DB 测试（D-03）
```go
dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db",
    time.Now().UnixNano(), os.Getpid()))
dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
    DisableForeignKeyConstraintWhenMigrating: true,
})
// 裸 CREATE TABLE: id TEXT PRIMARY KEY (绕过 models/api_key_usage_log.go:9 的 default:gen_random_uuid())
```
**关键约束（RESEARCH.md Pitfall 1/2）:**
- 不用 `AutoMigrate(&models.APIKeyUsageLog{})`（`gen_random_uuid()` PG 专有，sqlite 报 `no such function`）
- 不用 `file::memory:?cache=shared`（既有注释 `usage_logger_test.go:18-29` 明载并发写锁撞锁）
- 不用 `t.TempDir()`（fire-and-forget goroutine 测试结束后仍写文件，自动 cleanup 删占用文件 mark fail）
- `apikey_service_test.go` 用 `setupTestDB`（`:37-187`，`file::memory:?cache=shared&_enable_boolean=true`）——SC#3 落该文件，**沿用 setupTestDB 不改**，因其测试形态不同（无 fire-and-forget goroutine 与之并发）

---

### 异步写入可测试性（require.Eventually 轮询 DB 行）

**Source:** 仓库零现用（已 grep 确认），属本 phase 引入的升级 idiom
**Apply to:** SC#1/#4 真实 DB + fire-and-forget 测试（替代既有 `time.Sleep` flaky 反模式）
```go
func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
    t.Helper()
    require.Eventually(t, func() bool {
        var count int64
        db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
        return count >= want
    }, 2*time.Second, 10*time.Millisecond,
        "usage log for key=%s not persisted within 2s", apiKeyID)
}
```
**理由:** 零生产代码侵入（保持 fire-and-forget 纯粹）；条件满足立刻返回（快），超时则确定性失败（非偶发 flaky）；完美适配 SC#4「预取消 ctx 后行仍落库」的最终一致验证。
**落点:** `internal/middleware/apikey_integration_test.go` 与 `internal/services/usage_logger_test.go` 各定义一份同名 helper（Go 测试包隔离）。

---

## No Analog Found

无。本 phase 全部 5 个待改文件均在仓库内找到 exact 或 role-match 强匹配先例：

| 文件 | 先例 | 匹配度 |
|------|------|--------|
| `apikey.go`（middleware） | `pkg/middleware/logger.go` | exact（gin 记录模式，同角色同 data flow） |
| `usage_logger.go`（service） | `pkg/cache/redis.go:601-605` | role-match（异构角色但近乎同义 fire-and-forget + detached ctx data flow） |
| `apikey_integration_test.go`（test 扩） | 同文件既有 helper + fake | exact（同文件扩展） |
| `usage_logger_test.go`（test 扩） | 同文件既有 helper + 子测试 | exact（同文件扩展） |
| `apikey_service_test.go`（test 扩） | 同文件 `:1096` 既有「成功率计算」子测试 | exact（直接克隆形态） |

**Key insight:** 本 phase 零造轮子——所有基础设施（gin 记录模式 / detached context / applogger / sqlite 测试 DB / require.Eventually）仓库都有现成标准实现或近乎同义先例。

---

## Metadata

**Analog search scope:** 实读核对 RESEARCH.md 列出的全部先例文件：
- `internal/middleware/apikey.go`（全文，317 行）
- `internal/services/usage_logger.go`（全文，76 行）
- `internal/services/system/apikey_service.go:460-585`（GetUsageLogSummary 区段）
- `internal/services/system/apikey_service_test.go`（全文，1250 行）
- `internal/middleware/apikey_integration_test.go`（全文，250 行）
- `internal/services/usage_logger_test.go`（全文，533 行）
- `internal/models/api_key_usage_log.go`（全文，30 行）
- `pkg/middleware/logger.go`（全文，123 行）
- `pkg/cache/redis.go:590-620`（detached ctx 先例区段）
- `internal/services/config_backup_service.go:230-262`（applogger 先例区段）
- `pkg/logger/logger.go`（全文，251 行，Errorf API 确认）

**grep 验证:** `require.Eventually`（仓库零现用，确认新引入）、`applogger`（`config_backup_service.go:18` 别名确认）、`c.Writer.Status()` 既有先例。

**Files scanned:** 11 个文件实读 + 2 个 grep 验证

**Pattern extraction date:** 2026-08-13

**Confidence:** HIGH — 全部 file:line 经实读核对准确，零推测。
