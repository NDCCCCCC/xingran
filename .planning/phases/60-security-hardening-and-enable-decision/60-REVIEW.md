---
phase: 60-security-hardening-and-enable-decision
reviewed: 2026-08-13T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - internal/api/router.go
  - internal/api/v1/system/apikey_handler.go
  - internal/middleware/apikey.go
  - internal/middleware/apikey_integration_test.go
  - internal/middleware/apikey_test.go
  - internal/models/api_key.go
  - internal/services/system/apikey_service.go
  - internal/services/system/apikey_service_test.go
  - docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql
findings:
  critical: 0
  warning: 6
  info: 0
  total: 6
status: issues_found
---

# Phase 60: Code Review Report

**Reviewed:** 2026-08-13
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

本次审查覆盖 Phase 60 两个核心变更：(60-01) 在 `/system/apikeys/*` 路由组挂载 `MultiAuth` + `RateLimitByScope` 中间件并修复限流响应头编码；(60-02) 将 API Key 存储从明文 `key` 列迁移为 `KeyHash/Salt/KeyPrefix` 三列，配合 `subtle.ConstantTimeCompare` 校验。

整体实现符合既定决策（`260813-auth03-enable-decision.md`、`260813-sec01-hash-migration.md`），SM3 哈希构造、KeyPrefix 缩窄查询、常量时间比对、新的参数化 Where 子句均正确。但仍存在 6 项质量/健壮性/安全边界问题，其中**中间件链顺序导致纯 API Key 调用无法真正到达管理面路由**是最关键的已知边界未闭合问题；其余包括异步更新缺乏错误处理、作用域参数未校验、旧明文列残留、响应字段掩码不一致、以及校验逻辑重复。

## Critical Issues

无。

## Warnings

### WR-01: 中间件链顺序使纯 API Key 请求无法命中 `/system/apikeys/*`

**File:** `internal/api/router.go:116, 242-258`
**Issue:**
`apikeys` 路由组嵌套在 `authorized` 组之下，继承的 `JWTAuthWithBlacklist`（line 116）会在子组中间件之前执行。`MultiAuth` 和 `RateLimitByScope` 注册在 `RequirePermissions` 之后。因此一个只带 `X-API-Key`、不带 JWT 的请求会先在 `JWTAuthWithBlacklist` 处因「缺少认证令牌」被 401 中止，`MultiAuth` 永远不会被执行。虽然决策记录明确将此列为 Phase 60 已知边界（Phase 61 处理），但代码上挂载了却不可用，导致 `MultiAuth`/`RateLimitByScope` 对纯 API Key 调用者形同死代码。

**Fix:**
若要在 Phase 60 真正启用纯 API Key 访问，需要重构路由层次：将 `/system/apikeys` 从 `authorized` 子组提出到 `system` 组，并引入一个组合认证中间件（先尝试 API Key，失败再尝试 JWT），或让该路由组跳过 JWTAuth、由 `MultiAuth` 负责在缺失 API Key 时调用 JWT 校验。示例方向：

```go
// 选项：在 system 组下新建 apikeys 组，避免继承 authorized 的 JWTAuth
apikeys := system.Group("/apikeys")
apikeys.Use(CombinedAuth(core.JWTManager, core.TokenBlacklistService, apiKeyService, usageLogger))
apikeys.Use(middleware.RequirePermissions([]string{
    "system:apikey:list", "system:apikey:add", "system:apikey:edit", "system:apikey:delete",
}, core))
apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter()))
```

### WR-02: ValidateAPIKey 异步更新 last_used_at 缺乏错误处理与 panic 恢复

**File:** `internal/services/system/apikey_service.go:186-189`
**Issue:**
校验通过后启动 goroutine 异步更新 `last_used_at`，但：
1. 未处理 `Update` 返回的错误，DB 不可用时更新失败完全静默；
2. goroutine 没有 `recover`，若 GORM 内部出现 panic 可能拖垮进程；
3. 更新不携带上下文，无法追踪或取消。

**Fix:**
使用分离上下文并记录错误，增加 recover：

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("更新 api_key last_used_at panic: %v", r)
        }
    }()
    detachedCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := s.db.WithContext(detachedCtx).Model(&apiKey).Update("last_used_at", time.Now()).Error; err != nil {
        applogger.Errorf("更新 api_key last_used_at 失败: %v", err)
    }
}()
```

### WR-03: ListAPIKeys 未校验 params.Scope 即拼接 JSONB 查询

**File:** `internal/services/system/apikey_service.go:295-302`
**Issue:**
`params.Scope` 在绑定到 `ListAPIKeysParams` 时没有任何 `binding` 校验或白名单检查，直接拼入 `fmt.Sprintf("[\"%s\"]", *params.Scope)` 并作为参数传入 `@>`。虽然 GORM 使用参数化查询，不存在 SQL 注入，但如果 `Scope` 包含引号、反斜杠等字符，会导致 PostgreSQL 收到非法 JSONB 字面量，引发数据库错误；同时该分支逻辑仅支持 `read/write/admin` 三个语义值，缺少输入校验会让非法值透传到存储层。

**Fix:**
在构建查询前校验 scope 取值：

```go
if params.Scope != nil && *params.Scope != "" {
    if !validScopes[*params.Scope] {
        return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "无效的作用域筛选")
    }
    // ... 原 dialect 分支
}
```

### WR-04: 退役的明文 `key` 列未删除，存在残留敏感数据风险

**File:** `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql:9`
**Issue:**
SQL 文件仅删除冗余索引 `idx_api_keys_key`，未删除退役的 `key` 列。虽然决策记录基于「当前无活跃 API Key」的前提选择不迁移，但如果历史行（测试数据、已禁用 key、残留数据）仍然存在，DB 中会继续保留明文凭证。索引清理并不消除数据泄露面。

**Fix:**
在同一运维 SQL 或后续运维窗口追加列删除（前提是已确认无业务依赖）：

```sql
ALTER TABLE sys_api_keys DROP COLUMN IF EXISTS key;
```

若无法立即删列，应至少添加注释标记 `@Deprecated` 并纳入数据清理计划。

### WR-05: GetByID 返回完整 APIKey（含 User 关联），与 List 的字段掩码不一致

**File:** `internal/api/v1/system/apikey_handler.go:127-143`
**Issue:**
`List` 通过 `maskAPIKeys` 显式构造响应字段，严格控制不暴露 `KeyHash`、`Salt` 等敏感字段；但 `GetByID` 直接返回 `apiKey` 对象，依赖模型 `json:"-"` 隐藏敏感列。由于服务层 `Preload("User")`，`GetByID` 还会把关联用户的 `email`、`phone`、`loginIp` 等敏感信息一并序列化。一旦模型标签被误改或新增敏感字段，`GetByID` 就会比 `List` 多暴露数据，且两者维护口径不一致。

**Fix:**
在 `GetByID` 中也复用统一的掩码逻辑，例如：

```go
masked := maskSingleAPIKey(apiKey)
response.Success(c, masked)
```

或至少对 `User` 字段做按需裁剪/不预加载，避免返回完整的用户对象。

### WR-06: `isValidKeyFormat` 在校验中间件和服务层重复实现

**File:** `internal/middleware/apikey.go:97-117` 与 `internal/services/system/apikey_service.go:81-101`
**Issue:**
两个包内分别存在一份几乎相同的 `isValidKeyFormat` 函数。未来若调整 key 格式（如长度、前缀、字符集），需要同时修改两处，容易出现不一致，导致中间件提前拦截的格式与服务层接受的格式不同。

**Fix:**
将格式校验逻辑提取到公共包（如 `pkg/crypto/apikey` 或 `internal/utils/apikey`），中间件和服务层共用同一函数：

```go
package apikeyutil

func IsValidKeyFormat(key string) bool { ... }
```

---

_Reviewed: 2026-08-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
