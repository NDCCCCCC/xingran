# Phase 60: 安全加固与启用决策 - Research

**Researched:** 2026-08-13
**Domain:** Go 后端 / Gin 中间件 / GORM schema / 国密 SM3 哈希 / 限流响应头
**Confidence:** HIGH（所有锁定决策均有源码定位 + upstream phase 决策支撑，外加既有测试基建）

## Summary

Phase 60 本质是**决策落地 + 4 项硬化点**：(1) `MultiAuth` 生产挂载到 `/system/apikeys/*` 管理面触发 Phase 61；(2) API Key 改 SM3 单向哈希（`KeyHash` + `Salt` + `KeyPrefix` 三列、`ValidateAPIKey`/`CreateAPIKey`/`ListAPIKeys` 同步改写）；(3) `idx_api_keys_key` 冗余索引以手动 SQL+notes 形式交付（不写 Go migration）；(4) `RateLimitByScope` 限流响应头 `string(rune(int))` 改为 `strconv.Itoa` + 单测+集成测试。

**全部改造点都基于 Phase 57 已就绪的中间件代码 + Phase 59 detached context 使用日志**。变更面 = 4 个源文件 + 1 个新测试文件 + 1 个 SQL 文件 + 1 个 notes 文件。

**Primary recommendation:** 4 个改动面互不耦合，可分 4 个独立计划或 1 个 multi-task 计划执行。建议 1 plan 解决（4 tasks 线性依赖：AUTH-03 挂载 → QUAL-01 修复+测试 → SEC-01 schema + service 三改 → SEC-02 文档交付），所有 task 用既有 `_test.go` 扩展 + Phase 59 sqlite in-memory 模式复用。

## Architectural Responsibility Map

| 能力 | 主责层 | 协同层 | 理由 |
|------|--------|--------|------|
| AUTH-03 MultiAuth 路由挂载 | `internal/api/router.go`（routing tier） | `internal/api/v1/system/apikey_router.go`（handler 装配） | 仅一处 group middleware 追加（line 238-248），不改 handler/router 注册 |
| AUTH-03 IP 白名单 / 优先级 | `internal/middleware/apikey.go`（已就绪） | — | D-04 沿用 line 49-56 现有 `isIPAllowed` 逻辑，配置默认空白名单即「所有 IP 允许」 |
| SEC-01 schema 变更 | `internal/models/api_key.go` | `internal/services/system/apikey_service.go` | model 字段 + service `ValidateAPIKey`/`CreateAPIKey`/`ListAPIKeys` 三改，model 优先定义 → service 引用 |
| SEC-01 SM3 哈希原语 | `internal/core/security/password.go` (`sm3.New()`) | `internal/services/system/apikey_service.go` 新增 helper | 复用 password.go 的 `sm3.New()`，不另起 crypto 包 |
| SEC-02 冗余索引移除 | `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`（运维脚本） | `.planning/notes/260813-sec02-redundant-index-removal.md`（文档） | D-10 显式**不**写 Go migration，手动 SQL + notes 是 sole 交付物 |
| QUAL-01 限流头编码 | `internal/middleware/apikey.go` (line 267-268) | `internal/middleware/apikey_test.go`（单元） + `internal/middleware/apikey_integration_test.go`（集成） | 1 行改 + 2 个新测试 |
| 测试基建 | `internal/middleware/apikey_integration_test.go` (Phase 59 sqlite 模式) | `internal/services/usage_logger.go` 真实 logger | 复用 setupUsageLoggerTestDB + waitForUsageLog helper |

**单责分配原则:** 所有 4 项改动不跨模块、不影响其他中间件（除 AUTH-03 触发 Phase 61）。Tests 都集中在 `internal/middleware/` 包，不需新建跨包测试。

## User Constraints (from CONTEXT.md)

<user_constraints>
### Locked Decisions（严格不重新论证）

- **D-01** AUTH-03 = 启用全量挂载 — `internal/api/router.go:238-248` 的 `apikeys` 路由组按 `RequirePermissions` → `MultiAuth` → `RateLimitByScope` 顺序挂载。触发 Phase 61 立即执行。
- **D-02** 挂载范围 = 仅 `/system/apikeys/*` 管理面，不冲击其他 JWT 模块。
- **D-03** 认证优先级 = X-API-Key 优先 + JWT 回退（沿用 `apikey.go:27-31` 现有逻辑）。
- **D-04** IP 白名单 = 启用严格拒绝（沿用 `apikey.go:49-56` 现有 `isIPAllowed` 行为；配置默认空白名单即「所有 IP 允许」）。
- **D-05** SEC-01 存储方案 = SM3 单向哈希（不选 PBKDF2 / SM4 / 保留明文）。
- **D-06** Schema 变更 = `Key` 列移除 + `KeyHash` + `Salt` + `KeyPrefix` 三列新增。`KeyHash` `size:64;uniqueIndex;not null`，`Salt` `size:32;not null`，`KeyPrefix` `size:12;not null`。
- **D-07** List 搜索 = `Name LIKE ? OR KeyPrefix LIKE ?`（删除 `LEFT(key, 12) LIKE` / `substr(key, 1, 12) LIKE` SQLite 分支）。
- **D-08** 无迁移路径 = 直接切换（用户确认无活跃 API key，无需双读期 / 回填）。
- **D-09** CreateAPIKey 一次性返回明文：`generateKey()` → `SM3(key+salt)` → 存 `KeyHash`/`Salt`/`KeyPrefix` → 返回明文。
- **D-10** SEC-02 交付形式 = 手动 SQL + notes + 验证查询（**不**写 Go migration）。
- **D-11** QUAL-01 修复 = `strconv.Itoa` 替换 `string(rune(int))` + 同步加 `"strconv"` import。
- **D-12** QUAL-01 测试 = 单元测试 + 集成测试（复用 Phase 59 sqlite 模式）。
- **D-13** Phase 60 范围限定 = 仅 QUAL-01，不顺手修 `getScopeFromContext` 多 scope（QUAL-03 → Phase 61）。

### Claude's Discretion（research 给推荐）

| 项 | 推荐值 | 理由 |
|----|--------|------|
| SM3 哈希格式是否带版本前缀 | **仅 hex 无前缀**（64 字符） | 避免与 `HashPassword` 的 `$sm3$iterations$salt$hash` 格式混淆；API Key 单次哈希无 iterations，伪前缀反而误导 |
| Salt 长度 | **16 字节 / 32 hex 字符** | 与 `DefaultPasswordConfig.SaltLength` 一致；API Key 256-bit 熵已足够对抗字典攻击，盐只防 collision |
| 新 schema 列在 `models/api_key.go` 中的位置 | **BaseModel 之后，业务字段末尾**（紧跟 `InheritPerms` 之后） | 保持 业务字段分组（Name/Key 三列 → UserID → 时间 → 状态 → 作用域 → 白名单 → 描述 → 权限），`Key` 移除后 `KeyHash/Salt/KeyPrefix` 替换原位 |
| 验证查询 SQL 形式 | **PG `pg_indexes` + SQLite `sqlite_master` 双 fragment** | 单 SQL 跨 dialect 难写，分两段更易读；运维可直接 `psql` / `sqlite3` 跑 |

### Deferred Ideas（OUT OF SCOPE）

- 资源级权限矩阵 `RequireAPIKeyResourcePermission` → **Phase 61 / AUTH-04**（`D-13` 严格留 Phase 61）
- 限流生产调优 + `getScopeFromContext` 多 scope → **Phase 61 / QUAL-03**
- `username` 语义修正 → **Phase 61 资源权限领域**
- 密钥轮换/吊销、配额告警 → **FUTURE-APIKEY-03/04**（v2 Future，未升级）
- SEC-01 哈希后管理界面支持「查回明文」轮换 → **不**做（D-09 走「重新创建」路径），如需该能力需切到 SM4 对称加密（v2 Future 决策）
</user_constraints>

## Phase Requirements

<phase_requirements>
| ID | 描述 | 实现面（research 支撑） |
|----|------|------------------------|
| **AUTH-03** | MultiAuth 路由挂载生产启用 — `/system/apikeys/*` 全量挂载 + 决策记录 | `internal/api/router.go:238-248` 追加 2 个 middleware；决策记录在 `.planning/notes/260813-auth03-enable-decision.md`（新建） |
| **SEC-01** | API Key 改 SM3 单向哈希存储 — `KeyHash`+`Salt`+`KeyPrefix` 三列 + ValidateAPIKey 哈希比对 + CreateAPIKey 一次性明文 + ListAPIKeys KeyPrefix 搜索 | `internal/models/api_key.go` 字段重定义 + `internal/services/system/apikey_service.go` 3 函数主体重写 + 1 新 helper `hashAPIKey(key, salt) string` |
| **SEC-02** | 移除 `idx_api_keys_key` 冗余索引（migration_085 P3 收敛） | `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` + `.planning/notes/260813-sec02-redundant-index-removal.md`（**不**写 Go migration） |
| **QUAL-01** | `RateLimitByScope` 限流响应头 `string(rune(int))` → `strconv.Itoa` + 单测 + 集成测试 | `internal/middleware/apikey.go:267-268` 1 行 + `"strconv"` import + `internal/middleware/apikey_test.go` 新增 `TestRateLimitHeaderEncoding` + `internal/middleware/apikey_integration_test.go` 新增 `TestRateLimitHeadersInResponse` |
</phase_requirements>

## Standard Stack

### Core（全部复用，无新增依赖）

| 库 | 版本 | 用途 | 标准来源 |
|----|------|------|----------|
| `github.com/tjfoc/gmsm/sm3` | v1.4.1 (go.mod) | SM3 哈希原语 `sm3.New()` | `internal/core/security/password.go:14` (Phase 16 既有) |
| `gorm.io/gorm` | v1.30.5 | tag 迁移（`uniqueIndex` / `not null` / `size`） | `internal/models/api_key.go` |
| `github.com/gin-gonic/gin` | v1.10.0 | `c.Header()` 限流响应头 | `internal/middleware/apikey.go` 已有 |
| `crypto/rand` | stdlib | Salt 生成（16 字节 `rand.Read`） | `internal/core/security/password.go:104` 模式 |
| `encoding/hex` | stdlib | `hex.EncodeToString` SM3 输出 + Salt | `internal/services/system/apikey_service.go:6` 已有 |
| `strconv` | stdlib | `strconv.Itoa(result.Limit)` 替换 | 本 phase 新增 |
| `github.com/stretchr/testify` | v1.11.1 | `assert.Equal` / `require.NoError` | 现有测试基建 |

### Alternatives Considered

| 替代方案 | 不选原因 | Phase 60 选择 |
|----------|----------|---------------|
| SM3-PBKDF2（600k iterations） | API Key 256-bit 熵无字典攻击风险，PBKDF2 拉伸是过度设计 | SM3 单次哈希（PBKDF2 留密码场景） |
| SM4 对称加密 | 需 SM4_KEY 保护强度，DB+SM4_KEY 泄漏可直接还原 | SM3 单向哈希（不可逆） |
| argon2id | 跨生态一致性差（与国密栈不一致），且无字典攻击风险 | SM3（与密码哈希国密栈一致） |
| Go migration（migration_086） | D-10 手动 SQL 是显式偏离，决策快路径 + 运维可控 | 手动 SQL + notes |
| `FormatInt(int64, 10)` | 与 `strconv.Itoa` 等价，但需先转换类型 | `strconv.Itoa` (D-11 锁定) |

**Verdict:** 无任何新增依赖或包安装。所有改动在 go.mod 现有约束内。

## Architecture Patterns

### Pattern 1: MultiAuth 挂载顺序（Router Group → middleware chain）

**What:** `MultiAuth` + `RateLimitByScope` 追加到既有 `apikeys` 路由组，与 `RequirePermissions` 串联形成 3 段 middleware 链。

**When to use:** 任意启用 MultiAuth 的路由组必须遵循「鉴权 → 权限 → 限流」顺序。

**Example:**
```go
// internal/api/router.go:237-248 (D-01 改造后)
apikeys := authorized.Group("/apikeys")
apikeys.Use(middleware.RequirePermissions([]string{
    "system:apikey:list",
    "system:apikey:add",
    "system:apikey:edit",
    "system:apikey:delete",
}, core))
// D-01: 追加 MultiAuth + RateLimitByScope
apikeys.Use(middleware.MultiAuth(
    systemServices.NewAPIKeyService(core.GetDB()),
    services.NewUsageLogger(core.GetDB()),
))
apikeys.Use(middleware.RateLimitByScope(services.NewRateLimiter()))
{
    systemV1.SetupAPIKeyRouter(apikeys, core)
}
```

**关键点:**
- `MultiAuth` 接受 `system.APIKeyService` 接口 + `services.UsageLogger` 接口（D-03 锁定）
- `RateLimitByScope` 接受 `*services.RateLimiter`（D-12 验证可调用）
- 3 个 middleware 顺序：D-01 锁定的 `RequirePermissions` → `MultiAuth` → `RateLimitByScope`

### Pattern 2: SM3 hash + salt + prefix（不可逆哈希 + 查询友好）

**What:** API Key 存储分 3 列 —— `KeyHash`（SM3 输出） + `Salt`（16 字节随机） + `KeyPrefix`（明文前 12 字符）。ValidateAPIKey 接受明文 → 算 `SM3(input+salt)` → 与 `KeyHash` 恒定时间比对。

**When to use:** 任何需要"不可逆 + 仍可按前缀搜索"的高熵随机凭据。

**Example:**
```go
// internal/services/system/apikey_service.go (D-06 + D-09 改造后)

// hashAPIKey 计算 API Key 的 SM3 哈希值。
// SM3 单次哈希（无 PBKDF2 拉伸）：API Key 256-bit 熵无字典攻击风险。
// 输出 hex 字符串（64 字符），与 Salt（hex 32 字符）独立存储。
func hashAPIKey(key, salt string) string {
    h := sm3.New()
    h.Write([]byte(key))
    h.Write([]byte(salt))
    return hex.EncodeToString(h.Sum(nil))
}

// generateSalt 16 字节随机盐（hex 32 字符）
func generateSalt() (string, error) {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// CreateAPIKey 流程（D-09 一次性明文返回）
func (s *apiKeyServiceImpl) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) {
    // ... 验证用户 / 作用域 / 过期 同原代码 ...
    
    key, err := generateKey()  // 32 字节随机 → rec_ + 64 hex = 68 字符
    if err != nil {
        return nil, apperrors.Wrap(err, apperrors.CodeServerError, "密钥生成失败")
    }
    
    salt, err := generateSalt()
    if err != nil {
        return nil, apperrors.Wrap(err, apperrors.CodeServerError, "盐生成失败")
    }
    
    apiKey := models.APIKey{
        Name:         req.Name,
        KeyHash:      hashAPIKey(key, salt),
        Salt:         salt,
        KeyPrefix:    key[:12],  // 用于 List 搜索
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
}

// ValidateAPIKey 流程（D-08 哈希比对）
func (s *apiKeyServiceImpl) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
    if !isValidKeyFormat(keyStr) {
        return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "无效的密钥格式")
    }
    
    // 第一步: 用 KeyPrefix 缩小候选集（避免全表 hash）
    var candidates []models.APIKey
    err := s.db.WithContext(ctx).
        Where("key_prefix = ? AND is_active = ?", keyStr[:12], true).
        Find(&candidates).Error
    if err != nil {
        return nil, apperrors.DatabaseError(err)
    }
    
    // 第二步: 恒定时间比对每个候选的 SM3
    for _, candidate := range candidates {
        computedHash := hashAPIKey(keyStr, candidate.Salt)
        if subtle.ConstantTimeCompare([]byte(computedHash), []byte(candidate.KeyHash)) == 1 {
            // 验证过期
            if isKeyExpired(candidate.ExpiresAt) {
                return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥已过期")
            }
            // 异步更新最后使用时间
            go func() {
                s.db.Model(&candidate).Update("last_used_at", time.Now())
            }()
            return &candidate, nil
        }
    }
    
    return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥不存在或已禁用")
}
```

**关键点:**
- `KeyPrefix` 唯一索引（size:12）支持窄集合查询（避免 SHA-style 全表扫描）
- `KeyHash` 唯一索引确保 salt+hash 唯一性
- `subtle.ConstantTimeCompare` 防止时序攻击（**必加**）
- Salt 16 字节（hex 32 字符）足以平衡存储与安全

### Pattern 3: ListAPIKeys 关键词搜索（KeyPrefix LIKE，跨 dialect 一致）

**What:** 删除 `LEFT(key, 12) LIKE` / `substr(key, 1, 12) LIKE` 双 dialect 分支，改为 `Name LIKE ? OR KeyPrefix LIKE ?` —— 利用新建的 `KeyPrefix` 字段天然跨 dialect。

**When to use:** 任何需要在前缀字段上做模糊搜索、又想跨 PG/SQLite 的场景。

**Example:**
```go
// internal/services/system/apikey_service.go (D-07 改造后)
if params.Keyword != nil && *params.Keyword != "" {
    keyword := "%" + *params.Keyword + "%"
    // KeyPrefix 字段跨 dialect 一致，不再需要 isSQLite 分支
    query = query.Where("name LIKE ? OR key_prefix LIKE ?", keyword, keyword)
}
```

**关键点:**
- 减少 2 行 SQL dialect 分支
- `KeyPrefix` 大小 12 字符，索引仍可用（PG uniqueIndex / SQLite 自动索引）
- 搜索能力等价于原 `LEFT(key, 12) LIKE`（用户感知的"按前缀搜"）

### Pattern 4: strconv.Itoa 限流响应头（P2-a 修复）

**What:** `RateLimitByScope` 中 `c.Header("X-RateLimit-Limit", string(rune(result.Limit)))` 改为 `c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))`。

**When to use:** 任何 Go 中需要将整数序列化为 HTTP header 字符串的场景。

**Example:**
```go
// internal/middleware/apikey.go:267-268 (D-11 改造后)
import (
    // ... 既有 imports
    "strconv"
)

// 修复后
c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
c.Header("X-RateLimit-Reset", result.ResetAt.Format(time.RFC3339))
```

**关键点:**
- `string(rune(100))` = `"d"`（Unicode code point 100 是 'd'）—— 已知 bug
- `strconv.Itoa(100)` = `"100"` —— 正确
- `result.Limit` / `result.Remaining` 字段类型 `int`（rate_limiter.go:39-41），`Itoa` 直接接受

### Pattern 5: 手动 SQL + notes 交付（SEC-02 替代 Go migration）

**What:** 不写 Go migration，仅以 SQL 脚本 + notes 文档 + 验证查询三件套交付给运维。

**When to use:** Phase 60 决策快路径风格的"运维可控"场景。

**Example:**
```sql
-- docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql
-- Phase 60 / SEC-02: 移除 migration_085 遗留的冗余索引 idx_api_keys_key
-- 原因: 与 sys_api_keys.key 的 uniqueIndex 重复, 浪费写入+存储
-- 风险: 无 — uniqueIndex 仍保留, 索引扫描性能不变
-- 回滚: CREATE INDEX idx_api_keys_key ON sys_api_keys(key);

DROP INDEX IF EXISTS idx_api_keys_key;
```

```markdown
<!-- .planning/notes/260813-sec02-redundant-index-removal.md -->
# Phase 60 / SEC-02: 冗余索引 idx_api_keys_key 移除

## 为什么
详见 STATE.md P3 + migration_085 history。

## 怎么跑
1. `psql -h ... -d xingran_next -f docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`
2. 跑验证查询确认 schema 收敛（见下）

## 验证查询
PG:
```sql
SELECT indexname FROM pg_indexes
WHERE tablename = 'sys_api_keys'
  AND indexname IN ('idx_api_keys_key', 'sys_api_keys_key_hash_key');
-- 预期: 第一行 0 条, 第二行 1 条
```

SQLite:
```sql
SELECT name FROM sqlite_master
WHERE type = 'index' AND tbl_name = 'sys_api_keys';
-- 预期: 仅 sys_api_keys_key_hash_key (uniqueIndex), 无 idx_api_keys_key
```
```

**关键点:**
- Idempotent 脚本：`DROP INDEX IF EXISTS` 重复跑安全
- 双 dialect 验证查询：PG `pg_indexes` / SQLite `sqlite_master`
- 不写 Go migration 与 D-10 锁定一致

## Anti-Patterns to Avoid

| 反模式 | 为什么会出问题 | 正确做法 |
|--------|----------------|----------|
| 复用 `HashPassword` 格式 `$sm3$iterations$salt$hash` | API Key 单次哈希无 iterations，伪版本前缀误导；DB 字段冗余 | SM3 单次哈希 → 仅 hex 64 字符存 `KeyHash` 列 |
| 改 `Key` 字段名为 `key_hash` 保留列 | 字段语义模糊，新代码不知存明文还是哈希 | **移除** `Key` 字段 → 新增 `KeyHash` + `Salt` + `KeyPrefix` 三列 |
| 写 Go migration_086 移除索引 | 违反 D-10 锁定；migration 链增长 | 手动 SQL + notes 三件套 |
| 在 router 加 `auth=true` 时跳过 `MultiAuth` | 破坏 D-03 优先级一致性 | 始终按 `RequirePermissions` → `MultiAuth` → `RateLimitByScope` 顺序挂载 |
| 顺手修 `getScopeFromContext` 多 scope | 违反 D-13；与 Phase 61 资源权限设计冲突 | 严格留 Phase 61（QUAL-03 范畴） |
| `ValidateAPIKey` 用 `WHERE key = ?` 改为 `WHERE key_hash = ?` | 用户无法 prepend known prefix，需扫描全表 | 用 `KeyPrefix` 缩窄候选 → 恒定时间比对 `KeyHash` |
| `CreateAPIKey` 返回 `key` 字段给前端永久存储 | 哈希后无意义且误导 | 仅在 CreateAPIKey 一次性返回；List/GetByID 不返回 |
| `KeyPrefix` 用 `index` 而非 `uniqueIndex` | prefix 重复在多 key 间可能，不需唯一 | 不用 `uniqueIndex`，仅普通 `index` 提升 like 查询性能 |
| `c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))` | 与 `strconv.Itoa` 等价但更慢 | `strconv.Itoa` (D-11 锁定) |
| 测试用 `setupTestDB` 共享内存 SQLite | 并发测试写锁冲突 | 复用 Phase 59 `setupUsageLoggerTestDB` per-test 独立文件 DB（apikey_integration_test.go:111-137） |

## Don't Hand-Roll

| 问题 | 不要自造 | 用现成 | 理由 |
|------|----------|--------|------|
| SM3 哈希 | 自实现 SM3 算法 | `github.com/tjfoc/gmsm/sm3` `sm3.New()` | 已有 v1.4.1 国密库，内部 password.go 复用 |
| Salt 生成 | 自实现确定性 RNG | `crypto/rand.Read` (16 字节) | 密码学安全；现有 `password.go:104-106` 模式 |
| 恒定时间比对 | `==` 字符串比较 | `crypto/subtle.ConstantTimeCompare` | 防时序攻击；现有 `password.go:152` 模式 |
| 限流器 | 自实现滑动窗口 | `internal/services/rate_limiter.go` `NewRateLimiter()` | Phase 57 D-02 验证可调用 |
| 使用日志 | 自实现 async logger | `internal/services/usage_logger.go` `NewUsageLogger(db)` | Phase 59 detached context 已修复 |
| GORM 索引声明 | 自建 manual SQL 索引 | GORM tag `uniqueIndex;size:64;not null` | Phase 16 migration_085 既有 `uniqueIndex` 模式 |
| 限流响应头 | `string(rune(int))` | `strconv.Itoa` | stdlib 标准做法 |
| 测试 DB | 自建 sqlite helper | `setupUsageLoggerTestDB` (Phase 59) | 已有无跨测污染的 per-test 独立文件 DB 模式 |

**Key insight:** Phase 60 没有任何需要"自造"的子模块。每个改造点都有现成 Phase 16-59 的实现可复用。

## Common Pitfalls

### Pitfall 1: GORM tag `uniqueIndex` 重复声明

**What goes wrong:** `KeyHash` 字段写了 `gorm:"uniqueIndex"`，但 migration_085 在 `sys_api_keys` 表上已用 plain SQL 创建 `idx_api_keys_key` 唯一索引。重复索引会被 GORM AutoMigrate 跳过 warning，但 Phase 60 移除冗余索引前 schema 数据迁移期会"两个索引共存"。

**Why it happens:** Phase 60 仅改 `models/api_key.go` 字段定义，不主动清理 history 索引。

**How to avoid:** D-10 手动 SQL + notes **不依赖** schema 迁移，可在代码改造后**任何时刻**运行 `DROP INDEX IF EXISTS idx_api_keys_key`。

**Warning signs:** `pg_indexes` 显示 `sys_api_keys_key_hash_key` (AutoMigrate) + `idx_api_keys_key` (history) 同时存在。

**Planned mitigation:** SEC-02 notes 验证查询显式列出 **2 个**索引预期 (`sys_api_keys_key_hash_key` 留，`idx_api_keys_key` 删) —— 验收脚本幂等。

### Pitfall 2: SM3 哈希的 `crypto/rand` 不可用

**What goes wrong:** 测试环境无 `/dev/urandom` 或 `crypto/rand` 被 mock 时，`generateSalt()` 失败 → `CreateAPIKey` 失败。

**Why it happens:** Windows CI runner 偶发 `crypto/rand` 不可用（旧 Go 1.18 之前有过）。

**How to avoid:** Phase 60 测试不依赖 `generateSalt` 的真实输出（直接契约 `Salt` 字段 hex 32 字符），测试代码可手写 `Salt: "0123456789abcdef0123456789abcdef"`。

**Warning signs:** `go test ./internal/services/system/ -run TestCreateAPIKey` 报 `rand: not available` 或类似。

**Planned mitigation:** 测试层构造 `APIKey` 时直接填 Salt 字段，绕开 `generateSalt()` 路径，仅在 production 代码路径调用。

### Pitfall 3: 集成测试 fake UsageLogger 仍合理

**What goes wrong:** Phase 59 修复了 UsageLogger 真实链路（detached context），集成测试是否仍用 fake？

**Why it happens:** Phase 60 QUAL-01 集成测试只验证 `X-RateLimit-Limit` header 值，不需要 DB 实证 usage log。

**How to avoid:** AUTH-03 集成测试沿用 Phase 57 fake UsageLogger（无 DB 依赖，更快）；QUAL-01 集成测试**不需要** UsageLogger，可直接构造 fake UsageLogger（实现 `LogUsage` 接口即可）。

**Warning signs:** 测试断言 header 时若注册真实 UsageLogger，会因 DB schema 缺失 `sys_api_key_usage_logs` 表失败。

**Planned mitigation:** 2 套 fake：
- `fakeUsageLogger` (Phase 57 现有) — 单方法 `LogUsage` no-op
- 不需要真实 DB；可以用 `&fakeUsageLogger{logged: true, done: make(chan struct{})}` 复用

### Pitfall 4: RateLimitByScope 中 `rateLimiter` nil 时崩溃

**What goes wrong:** 集成测试 `MultiAuth` + `RateLimitByScope` 串联时，若 `rateLimiter` 是 nil，调用 `rl.Check(...)` 直接 panic。

**Why it happens:** Phase 60 新增挂载需要 `NewRateLimiter()` 实例化，但单测可能漏写。

**How to avoid:** 集成测试显式 `rl := services.NewRateLimiter()` 注入 `RateLimitByScope(rl)`。

**Warning signs:** `go test ./internal/middleware/` 时 `nil pointer dereference`。

**Planned mitigation:** TestFixture helper 集中创建 `services.NewRateLimiter()`，文档化所有 MultiAuth/RateLimitByScope 测试必须传非 nil 限流器。

### Pitfall 5: `idx_api_keys_key` 索引在 GORM AutoMigrate 时被重新创建

**What goes wrong:** 运维跑 SQL 删除 `idx_api_keys_key` 后，下次 `go run ./cmd/main.go` 启动时 GORM AutoMigrate 自动检测 schema 变化，若 `models.APIKey` 字段 tag 仍隐含该索引名（如 `Key` 字段历史拥有），会重新创建。

**Why it happens:** D-06 改 `Key` 字段为 `KeyHash`，GORM tag 从 `uniqueIndex` 改成 `KeyHash` 列的 `uniqueIndex`，索引名变为 `sys_api_keys_key_hash_key`（GORM 自动生成）—— 不再是 `idx_api_keys_key`。但 Phase 60 改造属"field 重命名"，GORM 视为 drop+add，未必能 in-place 重建。

**How to avoid:** 文档化：迁移期生产 DB 跑完 SQL 后，**重启后 AutoMigrate 会自动创建 `sys_api_keys_key_hash_key`**（正确），不会触碰已删除的 `idx_api_keys_key`。

**Warning signs:** 启动日志出现 `CREATE INDEX sys_api_keys_key_hash_key` 信息（这是正常的）。

**Planned mitigation:** notes 文档注释 AutoMigrate 行为，运维理解 schema 收敛是迁移期正常现象。

### Pitfall 6: Phase 60 改动影响 Phase 57 集成测试

**What goes wrong:** Phase 57 已写好的 `TestMultiAuthIntegration` 用 fake APIKeyService（line 33-68），fake 不通过 `ValidateAPIKey` 真实 DB 路径，所以 Phase 60 改 `ValidateAPIKey` 不影响 Phase 57 测试。

**Why it happens:** Phase 60 改造 `ValidateAPIKey` 是 service 层实现，fake 仍走 service 接口 mock 路径。

**How to avoid:** Phase 57 fake 继续可用；Phase 60 新增独立单测覆盖 `ValidateAPIKey` 真实链路的 SM3 比对。

**Warning signs:** `go test ./internal/middleware/` 全跑时 Phase 57 测试失败。

**Planned mitigation:** Phase 60 新增 `TestValidateAPIKeySM3Hash` 放在 `internal/services/system/apikey_service_test.go`（新建），不复用 fake。

### Pitfall 7: operlog 记录 `key` 字段脱敏失效

**What goes wrong:** SEC-01 改 `Key` → `KeyHash`/`Salt`/`KeyPrefix` 后，Phase 34 operlog 关键词脱敏列表（`key`/`secret`/`token`/`salt`）仍生效；但 `KeyPrefix` 字段值（前 12 字符）可能被识别为 `key` 关键词脱敏为 `******`。

**Why it happens:** operlog 脱敏基于 `c.GetRawData()` 读请求体 JSON，CreateAPIKey request 不含 `KeyPrefix`（仅 `Name`/`Scopes`/`IPWhitelist`/`ExpiresAt`/`Description`/`InheritPerms`），**不会**误脱敏。

**How to avoid:** 验证 operlog 记录创建 API Key 时，请求体不回显 `KeyPrefix`（因前端 request 不传）—— 实际无风险。

**Warning signs:** 看似 operlog 漏记录某些字段，**实际不漏**，因 KeyPrefix 是后端生成字段。

**Planned mitigation:** 无需 mitigation。PLAN.md 显式标注「`KeyPrefix` 不入 operlog」是设计意图。

## Code Examples

### Example 1: SEC-01 Model 字段定义（D-06 改造后）

```go
// internal/models/api_key.go (D-06 改造后)
type APIKey struct {
    BaseModel
    Name         string     `gorm:"size:100;not null" json:"name"`
    // REMOVED: Key 字段 (D-06: 不再明文存储)
    KeyHash      string     `gorm:"size:64;uniqueIndex;not null" json:"-"`         // SM3 hex
    Salt         string     `gorm:"size:32;not null" json:"-"`                      // 16 字节随机 hex
    KeyPrefix    string     `gorm:"size:12;index;not null" json:"keyPrefix"`        // 前 12 字符用于 List 搜索
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

**Notes:**
- `KeyHash` `json:"-"` 不暴露给前端（重要！）
- `Salt` `json:"-"` 不暴露给前端
- `KeyPrefix` `json:"keyPrefix"` 暴露（仅 List 搜索回显用）
- `KeyPrefix` 用普通 `index` 而非 `uniqueIndex`（首 12 字符可能重复）

### Example 2: QUAL-01 限流头修复 + 单测

```go
// internal/middleware/apikey.go:267-268 (D-11 改造后)
import (
    "strconv"  // ← D-11 新增
)

c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
```

```go
// internal/middleware/apikey_test.go (D-12 新增)
func TestRateLimitHeaderEncoding(t *testing.T) {
    t.Run("Limit=100→'100' 而非 'd'", func(t *testing.T) {
        // 验证 strconv.Itoa 替换 string(rune(int)) 后输出是数字字面量
        assert.Equal(t, "100", strconv.Itoa(100))
        assert.Equal(t, "99", strconv.Itoa(99))
        assert.Equal(t, "0", strconv.Itoa(0))
        // 防御性断言: 原 string(rune(100)) = "d" 不再出现
        assert.NotEqual(t, "d", strconv.Itoa(100))
    })

    t.Run("RateLimitResult 反序列化", func(t *testing.T) {
        result := struct {
            Limit     int
            Remaining int
        }{Limit: 100, Remaining: 99}
        // 验证 header 字符串可被 strconv.Atoi 反解析
        if n, err := strconv.Atoi(strconv.Itoa(result.Limit)); err != nil || n != 100 {
            t.Errorf("header not parseable: %v", err)
        }
    })
}
```

### Example 3: QUAL-01 集成测试

```go
// internal/middleware/apikey_integration_test.go (D-12 新增)
func TestRateLimitHeadersInResponse(t *testing.T) {
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
    rl := services.NewRateLimiter()  // ★ 必传非 nil, Pitfall 4

    router := gin.New()
    router.Use(MultiAuth(fakeSvc, realLogger))
    router.Use(RateLimitByScope(rl))  // ★ 新增挂载
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
    
    // 验证可被 strconv.Atoi 反解析
    n, err := strconv.Atoi(limitHeader)
    assert.NoError(t, err, "X-RateLimit-Limit 必须是数字字符串")
    assert.Greater(t, n, 0)
    
    n2, err := strconv.Atoi(remainingHeader)
    assert.NoError(t, err, "X-RateLimit-Remaining 必须是数字字符串")
    assert.GreaterOrEqual(t, n2, 0)
    
    // 防御性断言: 不再是单字符 ("d" = rune(100))
    assert.NotEqual(t, "d", limitHeader)
}
```

### Example 4: SEC-02 SQL 脚本

```sql
-- docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql
-- Phase 60 / SEC-02: 移除 migration_085 遗留的冗余索引 idx_api_keys_key
-- 原因: 与 sys_api_keys.key 列的 uniqueIndex 重复, 写放大 + 存储浪费
-- 风险: 无 — uniqueIndex 仍保留, 索引扫描性能不变
-- 回滚: CREATE INDEX idx_api_keys_key ON sys_api_keys(key);
-- 幂等: IF EXISTS 关键字保证重复跑安全

DROP INDEX IF EXISTS idx_api_keys_key;
```

### Example 5: AUTH-03 挂载（router.go 改造后）

```go
// internal/api/router.go:237-248 (D-01 改造后)
apikeys := authorized.Group("/apikeys")
apikeys.Use(middleware.RequirePermissions([]string{
    "system:apikey:list",
    "system:apikey:add",
    "system:apikey:edit",
    "system:apikey:delete",
}, core))
// D-01: 追加 MultiAuth + RateLimitByScope
apikeys.Use(middleware.MultiAuth(
    systemServices.NewAPIKeyService(core.GetDB()),
    services.NewUsageLogger(core.GetDB()),
))
apikeys.Use(middleware.RateLimitByScope(services.NewRateLimiter()))
{
    systemV1.SetupAPIKeyRouter(apikeys, core)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| API Key `Key` 字段明文存储 | `KeyHash`+`Salt`+`KeyPrefix` 三列 SM3 哈希 | Phase 60 (2026-08-13) | DB 泄漏后无法还原明文 |
| `LEFT(key, 12) LIKE` 双 dialect 分支 | `KeyPrefix LIKE` 跨 dialect 一致 | Phase 60 | 减少 SQL 复杂度 |
| 限流响应头 `string(rune(int))` | `strconv.Itoa` | Phase 60 | Limit=100 → "100" 而非 "d" |
| MultiAuth 死代码（未挂载） | `/system/apikeys/*` 管理面全量挂载 | Phase 60 | X-API-Key 认证真正生效 |
| `idx_api_keys_key` 冗余索引 | 手动 SQL 移除 | Phase 60 | 写放大消除 |
| migration_085 `idx_api_keys_key` 创建 | `//go:build archive_skip` 不再执行 | Phase 60 | 历史索引靠运维手动清理 |

**Deprecated/outdated:**
- `Key` 字段（D-06 移除）: 不再明文存储
- `string(rune(int))` 编码模式（P2-a）: 标准做法是 `strconv.Itoa`
- `LEFT(key, 12) LIKE` (D-07): 改用 `KeyPrefix LIKE`

## Assumptions Log

> 全部沿用 Phase 57 + 59 上游决策，详见 `.planning/phases/57-.../57-CONTEXT.md` 与 `.planning/phases/59-.../59-CONTEXT.md`。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `services.NewUsageLogger(db)` Phase 60 仍可用（Phase 59 D-02 验证） | AUTH-03 / Phase 59 integration | Phase 59 集成测试已验证，低风险 |
| A2 | `services.NewRateLimiter()` 无参版可用（Phase 57 D-02 验证） | AUTH-03 / QUAL-01 integration | Phase 57 集成测试已验证，低风险 |
| A3 | `models.APIKey` BaseModel 有 `gorm.DeletedAt` 支持软删除（base.go:15） | SEC-01 schema 不影响删除 | 低风险 |
| A4 | `Core` 字段含 `DB` / `Cache` / `JWTManager` / `PwdManager` / `OperLogService` (core.go:131-136) | AUTH-03 挂载需 `core.GetDB()` | 字段已验证，低风险 |
| A5 | `gmsm/sm3` v1.4.1 库 `sm3.New()` + `h.Sum(nil)` 返回 32 字节（password.go:79-82 既有用法） | SEC-01 SM3 哈希 | 沿用既有模式，低风险 |
| A6 | `crypto/rand.Read` 在 Windows CI runner 可用（password.go:104 既有用法） | SEC-01 Salt 生成 | 沿用既有模式，低风险 |
| A7 | `pg_indexes` 是 PG 内置视图（标准 PG 9.5+） | SEC-02 验证查询 | 标准 SQL，无风险 |
| A8 | `sqlite_master` 是 SQLite 系统表（标准 SQLite 3.x） | SEC-02 验证查询 | 标准 SQL，无风险 |
| A9 | `response.ErrForbidden.HTTPStatus == 401`（response.go:40-41） Phase 57 测试已验证 | QUAL-01 集成测试断言 | 现有测试逻辑无需修改 |
| A10 | `gin.Header()` 接受任意字符串值 | QUAL-01 修复 | 既有 c.Header("X-RateLimit-Reset", ...) 用法，低风险 |

**Assumptions table empty after A1-A10:** All LOW risk, recycled from upstream phases.

## Open Questions

1. **`KeyPrefix` 在响应中是否暴露？**
   - What we know: CONTEXT.md D-06+D-07 提到 `KeyPrefix` 用于 List 搜索，未明确 JSON 行为
   - What's unclear: 前端是否需要 `keyPrefix` 字段做 UI 展示
   - Recommendation: **暴露** `keyPrefix` 字段（仅前 12 字符 + `rec_` 前缀），前端可显示给用户「密钥前缀」做 UI 提示，类似 AWS/GitHub API Key 卡片

2. **operlog 创建日志是否记录 `KeyPrefix`？**
   - What we know: 请求体不含 `KeyPrefix`（后端生成），operlog 读 `c.GetRawData()`
   - What's unclear: 是否需在 `APIKey` 表的 `KeyPrefix` 字段写入后，由 service 层显式记录
   - Recommendation: 不记录（请求体本就无 KeyPrefix，operlog 不可见）

3. **Phase 60 完成后是否需要 Phase 61 立即启动？**
   - What we know: D-01 触发 Phase 61 立即执行
   - What's unclear: GSD workflow 是否自动串接 Phase 60 → 61
   - Recommendation: Phase 60 plan 完成后由 operator 手动 `/gsd:plan-phase 61`

4. **SEC-02 验证查询是否提供 SQL 函数封装？**
   - What we know: notes 手动列出 PG/SQLite 两段
   - What's unclear: 是否提供 Go introspection helper
   - Recommendation: **不**做（避免 Go migration 化），notes 是 sole 交付物

5. **`MultiAuth` 注册的 `apiKeyService` 是否缓存 instance？**
   - What we know: `systemServices.NewAPIKeyService(core.GetDB())` 每次调用新建实例
   - What's unclear: 是否有必要在 Core 持有单例
   - Recommendation: **不**改（现有架构 `apikeyHandler.NewAPIKeyHandler(apiKeyService).WithCore(coreCore)` 每次新建 router 调用已存在），middleware 闭包捕获 `apiKeyService` 不影响性能

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24 | 所有 Go 代码 | ✓ | go1.24.5 windows/amd64 | — |
| `gmsm/sm3` v1.4.1 | SEC-01 哈希 | ✓ | go.mod | — |
| `gin` v1.10.0 | 所有 middleware | ✓ | go.mod | — |
| `gorm` v1.30.5 | Schema 迁移 | ✓ | go.mod | — |
| `gorm.io/driver/sqlite` v1.5.6 | 集成测试 | ✓ | go.mod (replace 1.5.4) | — |
| `crypto/rand` (stdlib) | Salt 生成 | ✓ | Go 1.24 | — |
| `sqlite3` CLI | SEC-02 验证查询 | ✓ | /d/Program Files/Sqlite3/sqlite3 | — |
| `go test ./internal/...` | 所有测试 | ✓ | go1.24.5 | — |
| `psql` (PG client) | SEC-02 验证查询 | ✗ | — | notes 提供 SQL 即可，运维按需安装 |
| WSL / Linux runtime | 跨 dialect 测试 | ✗ | — | 单一 Windows CI 即可，PG 验证靠运维 |

**Missing dependencies with no fallback:**
- 无（所有 Phase 60 必需依赖均可用）

**Missing dependencies with fallback:**
- psql: SEC-02 验证查询仅在生产 DB 跑，Phase 60 CI 不强制

## Validation Architecture

> Phase 60 nyquist_validation enabled (config.json `workflow.nyquist_validation: true`)，必须输出此节。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `testing` (Go stdlib) + `github.com/stretchr/testify` v1.11.1 |
| Config file | 无独立 vitest/jest 配置（Go 默认）；`apikey_test.go` 与 `apikey_integration_test.go` 已在 middleware 包 |
| Quick run command | `go test -v -run "TestRateLimit|TestMultiAuth|TestValidateAPIKey" ./internal/middleware/ ./internal/services/system/` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| **AUTH-03** | MultiAuth 挂载到 `/system/apikeys/*` 路由组 | 集成 | `go test -v -run TestMultiAuthProductionMount ./internal/api/...` | ❌ Wave 0 |
| **AUTH-03** | MultiAuth 优先级 + JWT 回退 | 集成 | Phase 57 `TestMultiAuthIntegration` 三路径全跑 | ✅ Phase 57 |
| **SEC-01** | `CreateAPIKey` 生成 SM3 哈希 + Salt + KeyPrefix | 单元 | `go test -v -run TestCreateAPIKeySM3Hash ./internal/services/system/` | ❌ Wave 0 |
| **SEC-01** | `ValidateAPIKey` 哈希比对 + 恒定时间 | 单元 | `go test -v -run TestValidateAPIKeySM3Hash ./internal/services/system/` | ❌ Wave 0 |
| **SEC-01** | `ListAPIKeys` 关键词搜索 `KeyPrefix LIKE` | 单元 | `go test -v -run TestListAPIKeysKeyPrefixLike ./internal/services/system/` | ❌ Wave 0 |
| **SEC-02** | idx_api_keys_key 移除 | 手动 | SQL 跑 + notes 验证查询 | N/A (manual) |
| **QUAL-01** | 限流头 strconv.Itoa 编码 | 单元 | `go test -v -run TestRateLimitHeaderEncoding ./internal/middleware/` | ❌ Wave 0 |
| **QUAL-01** | 限流头集成测试 | 集成 | `go test -v -run TestRateLimitHeadersInResponse ./internal/middleware/` | ❌ Wave 0 |

### Success Criteria → 验证机制（断言形式）

| SC | 验证方法 | 断言形式 |
|----|----------|----------|
| **SC#1** AUTH-03 决策记录 | `.planning/notes/260813-auth03-enable-decision.md` 存在 + 含 4 维度 | Notes 含「InheritPerms / IP 白名单 / 优先级 / JWT 回退」4 段 |
| **SC#2** SEC-01 决策记录 | `.planning/notes/260813-sec01-hash-migration.md` 存在 + 含 5 维度 | Notes 含「存储方案 / Schema / List 搜索 / 迁移路径 / 创建流程」5 段 |
| **SC#3** SEC-02 索引收敛 | `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 存在 + notes 验证查询可执行 | `ls docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` 退出码 0 |
| **SC#4** QUAL-01 限流头 | 集成测试断言 + curl 可解析 | `strconv.Atoi(limitHeader)` 返回整数，err=nil |

### Hermetic 性保证

- **集成测试:** 复用 Phase 59 `setupUsageLoggerTestDB` (per-test 独立文件 DB + busy_timeout=5000)
- **单元测试:** 无外部依赖（手写 `models.APIKey{}` 字面量）
- **SEC-02 SQL:** 单独手动文件，不进 Go binary

### Sampling Rate

- **Per task commit:** `go test ./internal/middleware/ ./internal/services/system/`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/services/system/apikey_service_test.go` — 新建文件，覆盖 SEC-01 三函数（CreateAPIKey / ValidateAPIKey / ListAPIKeys）
- [ ] `internal/middleware/apikey_test.go` — 扩展 `TestRateLimitHeaderEncoding` (D-12 QUAL-01 单测)
- [ ] `internal/middleware/apikey_integration_test.go` — 扩展 `TestRateLimitHeadersInResponse` (D-12 QUAL-01 集成)
- [ ] `internal/api/router_test.go` — 新建（可选），验证 `/system/apikeys/*` middleware 链装配（用 `gin.Engine` 注册 + 触发）

*(Wave 0 4 项，预计 plan 1 起步完成)*

## Security Domain

> Phase 60 security 关联度高，本节必填。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | `MultiAuth` (X-API-Key) + JWT 双认证通路；IP 白名单严格拒绝 |
| V3 Session Management | partial | API Key 无 session 概念；与 JWT 并存 |
| V4 Access Control | yes | `RequirePermissions` (Permission) + `RequireScope` (Scope) 双层 |
| V5 Input Validation | yes | `isValidKeyFormat` / `validateScopes` / `isKeyExpired` （既有） |
| V6 Cryptography | **YES (核心)** | SM3 单向哈希 + `crypto/rand` Salt + `crypto/subtle.ConstantTimeCompare` |
| V7 Error Handling | yes | operlog 敏感字段脱敏（既有） |
| V9 Logging | yes | `UsageLogger` 真实记录（Phase 59 修复） |
| V10 SSRF | N/A | — |
| V14 Configuration | yes | `IPWhitelist` / `ExpiresAt` 配置层 |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| API Key DB 泄漏 → 明文还原 | Information Disclosure | **SEC-01 SM3 哈希**（不可逆）+ Salt 抗 collision |
| 时序攻击枚举 API Key | Information Disclosure | `crypto/subtle.ConstantTimeCompare` |
| IP 白名单绕过 | Spoofing | `apikey.go:49-56` `isIPAllowed` CIDR 验证 |
| 限流头被错误解析（如 `d` 字面） | Tampering | **QUAL-01 `strconv.Itoa`** 修复 |
| 重复索引影响 DB 写入性能 | Denial of Service | **SEC-02 索引移除** |
| 暴力枚举 API Key | Information Disclosure | `isValidKeyFormat` 长度 68+hex 校验 |
| Salt 不足导致 collision | Information Disclosure | 16 字节 = 128 bits（密码学安全下限） |
| Request body 泄漏 key 明文到 operlog | Information Disclosure | Phase 34 operlog 关键词脱敏（既有） |

### ASVS V6 Cryptography 强制要求

- **SM3 用法:** 单次哈希（不回环），输入 `key+salt` 拼接 32 字节输出 hex 64 字符
- **Salt:** `crypto/rand.Read` 16 字节（128 bits），密码学安全
- **比对:** `subtle.ConstantTimeCompare` 防止时序攻击
- **存储:** `KeyHash` 独立于 `Salt`，DB 泄漏后无法还原明文
- **不退化:** 严禁将 `KeyHash` 降级为明文（无回滚路径）

## Sources

### Primary (HIGH confidence)

- `internal/middleware/apikey.go` — 现状 MultiAuth / RateLimitByScope / isIPAllowed / getScopeFromContext 全部 line 锚定
- `internal/services/system/apikey_service.go` — ValidateAPIKey (line 128-161) / CreateAPIKey (line 164-222) / ListAPIKeys (line 224-289) 改造点
- `internal/models/api_key.go` — APIKey struct 与 gorm tag 字段
- `internal/core/security/password.go` — sm3.New() 既有用法（line 79）
- `internal/middleware/apikey_integration_test.go` — Phase 57/59 集成测试基建（per-test 独立 DB + fake UsageLogger）
- `.planning/phases/57-.../57-CONTEXT.md` — D-02 4 中间件签名自洽 / D-04 7 context 键
- `.planning/phases/59-.../59-CONTEXT.md` — D-02 detached context / D-01 Success = 仅 2xx
- `.planning/REQUIREMENTS.md` — REQUIREMENTS §AUTH-03 / SEC-01 / SEC-02 / QUAL-01 定义
- `.planning/ROADMAP.md` — Phase 60 Success Criteria SC#1-4 锚定

### Secondary (MEDIUM confidence)

- gmsm/sm3 v1.4.1 godoc — 沿用既有 `password.go` 模式无可证伪点
- GORM v1.30.5 tag 文档 — `uniqueIndex` / `size` / `not null` 既有 project 用法
- gin v1.10.0 `c.Header()` API — 既有 project 用法

### Tertiary (LOW confidence)

- Context7 MCP unavailable — 包名/版本通过 go.mod + 既有 import 验证，无 Context7 文档

## Metadata

**Confidence breakdown:**
- Standard Stack: **HIGH** — 全部沿用既有 import，0 新依赖
- Architecture: **HIGH** — 4 改动点 line 锚定，跨 phase 决策一致
- Pitfalls: **HIGH** — 7 个 Pitfall 全部基于既有 Phase 57/59 经验 + 源码 line 验证
- Security: **HIGH** — SM3/Salt/ConstantTimeCompare 均有密码学标准 + 既有密码用法 reference

**Research date:** 2026-08-13
**Valid until:** 2026-09-13 (30 days, stable — no external library churn risk)
