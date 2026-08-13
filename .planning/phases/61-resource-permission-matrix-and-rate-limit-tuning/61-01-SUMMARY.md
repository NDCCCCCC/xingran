---
phase: 61-resource-permission-matrix-and-rate-limit-tuning
plan: 01
subsystem: auth
tags:
  - api-key
  - auth
  - permission
  - resource-permission-matrix
  - inherit-perms
  - gin
  - gorm
  - sqlite

requires:
  - phase: 60-security-hardening-and-enable-decision
    provides: MultiAuth production mount on /system/apikeys/* (AUTH-03=enabled), SM3 hash storage, rate limit header fix
  - phase: 59-observability-usage-log-fix
    provides: UsageLogger detached context + async log timing
  - phase: 57-auth-chain-core-fix-regression-test
    provides: 7 context keys + MultiAuth self-consistent integration tests

provides:
  - pkg/permission/resource_action_map.go: static (resource, action) → PermissionCode map for system:* resources
  - RequireAPIKeyResourcePermission helper with fail-closed lookup and union scope check
  - MultiAuth + setUserContextForAPIKey InheritPerms real-time User permission load + scope union
  - username/nickname context values read from apiKey.User association
  - 3-tier test coverage (unit + middleware unit + sqlite integration)

affects:
  - 61-resource-permission-matrix-and-rate-limit-tuning (Plan 02 - RateLimitByScope action-aware tuning)
  - Any future phase expanding resource_action_map beyond system:*

tech-stack:
  added: []
  patterns:
    - "Static compile-time resource permission map with explicit PermissionCode entries (no string concat)"
    - "Fail-closed lookup: unmapped resource/action → 403"
    - "Per-request DB query for InheritPerms User permission load (no cache)"
    - "Union of coarse API Key scopes + fine-grained User permission codes in c.scopes"
    - "No gomock: real sqlite DB + real permission.Service in integration tests"

key-files:
  created:
    - pkg/permission/resource_action_map.go
    - pkg/permission/resource_action_map_test.go
    - internal/middleware/apikey_resource_permission_test.go
    - internal/middleware/apikey_inherit_integration_test.go
  modified:
    - pkg/permission/config.go (added APIKey* PermissionCode constants)
    - internal/middleware/apikey.go (MultiAuth, setUserContextForAPIKey, RequireAPIKeyResourcePermission)
    - internal/api/router.go (updated MultiAuth call site)
    - internal/middleware/apikey_integration_test.go (added nil nil args to MultiAuth calls)

key-decisions:
  - "MultiAuth signature requires permission.Service + *gorm.DB so InheritPerms=true can load User permissions from real service"
  - "RequireAPIKeyResourcePermission accepts union of admin / fine-grained PermissionCode / coarse scope(read/write) to work for both InheritPerms and non-InheritPerms keys"
  - "Unmapped resources/actions return 403 '资源权限未定义' (fail-closed D-03)"
  - "User permission load failure returns 401 '用户权限加载失败' (fail-closed D-09)"
  - "No Redis cache for InheritPerms User permissions in this phase (D-07)"

patterns-established:
  - "resource_action_map entries must use explicit PermissionCode constants (no string concatenation)"
  - "New resources/actions must explicitly add map entries or requests fail-closed"
  - "InheritPerms=true path merges coarse scopes and fine-grained permission codes into a single c.scopes slice"

requirements-completed:
  - AUTH-04

# Metrics
duration: 20min
completed: 2026-08-13
---

# Phase 61 Plan 01: 资源权限矩阵与 InheritPerms 落地 Summary

**AUTH-04 资源权限矩阵真实生效: system:* 11 资源 × 59 action 静态映射接入 RequireAPIKeyResourcePermission,InheritPerms=true 实时加载 User 权限与 scopes 取并集,username/nickname 从 apiKey.User 读取,配套 3 层测试覆盖。**

## Performance

- **Duration:** 20 min
- **Started:** 2026-08-13T08:35:41Z
- **Completed:** 2026-08-13T08:55:14Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- 创建静态资源权限矩阵 `pkg/permission/resource_action_map.go`,覆盖 `system:*` 11 个资源(user/role/menu/dept/post/workstation/dict/config/captchaBackground/notice/apikey) 共 59 个 (resource, action) 组合,全部显式映射到 `PermissionCode` 常量
- 改造 `RequireAPIKeyResourcePermission` 从「忽略 resource」变为真实生效:查 map → 命中走 union scope 检查(admin / PermissionCode / coarse scope);未命中 → 403 fail-closed
- 改造 `MultiAuth` + `setUserContextForAPIKey`:`InheritPerms=true` 时调用 `permission.Service.GetUserPermissions` 实时加载 User 权限代码,与 API Key 自带 scopes 取并集写入 `c.Set("scopes", mergedScopes)`;加载失败 → 401 fail-closed
- 修正 `username`/`nickname` 语义:从 `apiKey.User.Username` / `apiKey.User.Nickname` 读取(ValidateAPIKey 已 `Preload("User")`);User 关联缺失时兜底 `apiKey.Name` + Warnf
- 更新 `internal/api/router.go` 中 `MultiAuth` 调用形态,传入 `permission.NewService()` 与 `core.GetDB()`
- 三层测试覆盖:`pkg/permission` map 单元测试(6 个) + `RequireAPIKeyResourcePermission` 中间件单元测试(8 个) + `InheritPerms` sqlite in-memory 集成测试(5 个),无 gomock

## Task Commits

1. **Task 1: 创建 pkg/permission/resource_action_map.go + 单元测试** - `c55a3c5` (feat)
2. **Task 2: 改造 MultiAuth + setUserContextForAPIKey + RequireAPIKeyResourcePermission + router.go 调用形态** - `cba12ce` (feat)
3. **Task 3: 单元测试 + 集成测试覆盖 D-20/D-21** - `1eae873` (test)

## Files Created/Modified

- `pkg/permission/resource_action_map.go` - 静态资源→权限映射 + `LookupResourceAction` + `MapKeys`
- `pkg/permission/resource_action_map_test.go` - map 单元测试(命中/未命中/MapKeys/范围断言)
- `pkg/permission/config.go` - 新增 `APIKeyList/View/Add/Edit/Remove` 5 个 PermissionCode 常量
- `internal/middleware/apikey.go` - MultiAuth 签名扩展 + InheritPerms 加载 + username/nickname 修正 + RequireAPIKeyResourcePermission 接入 map
- `internal/api/router.go` - MultiAuth 调用形态变更(`permission.NewService()`, `core.GetDB()`)
- `internal/middleware/apikey_integration_test.go` - 既有 6 处 `MultiAuth(...)` 调用补 `nil, nil` 参数(仅签名适配,无逻辑修改)
- `internal/middleware/apikey_resource_permission_test.go` - `RequireAPIKeyResourcePermission` 中间件单元测试 8 个
- `internal/middleware/apikey_inherit_integration_test.go` - `InheritPerms` sqlite in-memory 集成测试 5 个

## Decisions Made

- 与计划一致:静态 map 采用嵌套 `map[string]map[string]PermissionCode` + 显式 `PermissionCode(...)` 包装,既保证编译期常量又可 `grep -c PermissionCode` 验证条目数
- 与计划一致:`InheritPerms=true` 时走每请求一次 DB 查询,不引入缓存(D-07)
- 与计划一致:`InheritPerms=true` 且 `UserID==nil` 时 401 fail-closed(D-09)
- 与计划一致:`RequireAPIKeyResourcePermission` 仍为公共 helper,本 phase 不挂载到 `apikey_router.go`(D-05)
- 实践微调:`RequireAPIKeyResourcePermission` 的 scope 检查扩展为 union 语义,同时接受 admin / PermissionCode / coarse scope,确保非 InheritPerms API Key(coarse scopes) 与 InheritPerms API Key(fine-grained permission codes) 都能通过

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] MultiAuth signature需传入 *gorm.DB 才能调用 GetUserPermissions**
- **Found during:** Task 2 (MultiAuth 签名扩展)
- **Issue:** `permission.Service.GetUserPermissions` 第一参数是 `*gorm.DB` 而非 `context.Context`;原计划在 Task 2 描述中未显式说明需要传入 db,导致初次编译失败
- **Fix:** 扩展 `MultiAuth` 和 `setUserContextForAPIKey` 签名新增 `*gorm.DB`,并在 router.go 传入 `core.GetDB()`;集成测试中也相应传入测试 DB
- **Files modified:** `internal/middleware/apikey.go`, `internal/api/router.go`, `internal/middleware/apikey_integration_test.go`, `internal/middleware/apikey_inherit_integration_test.go`
- **Verification:** `go build ./...` 通过,`go test ./internal/middleware/...` 通过
- **Committed in:** `cba12ce` / `1eae873` (Task 2 + Task 3 commits)

**2. [Rule 2 - Missing Critical] RequireAPIKeyResourcePermission 未兼容 coarse scope 导致非 InheritPerms key 无法通过**
- **Found during:** Task 3 (中间件单元测试 `TestRequireAPIKeyResourcePermission_HitReadScope` 失败)
- **Issue:** 仅检查 `string(permCode)` 时,`scopes=["read"]` 无法匹配 `system:user:view`,破坏了非 InheritPerms API Key 的向后兼容(D-08)
- **Fix:** scope 检查改为 union 语义:admin 通配 / PermissionCode 直接匹配 / coarse scope(`getRequiredScope(action)`) 三选一通过
- **Files modified:** `internal/middleware/apikey.go`
- **Verification:** `TestRequireAPIKeyResourcePermission*` 8 个测试全部 PASS
- **Committed in:** `1eae873` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both fixes are necessary for correctness and backward compatibility. No scope creep.

## Issues Encountered

- `go test ./...` 全量运行发现若干既有失败(与本次修改无关):
  - `internal/api/v1/auth`:`TestADLoginWithOUProcessing`
  - `internal/api/v1`:`TestIntegration_ParseUserAgent`, `TestLoginWithInvalidEncryptedRequest`
  - `pkg/errors`:`TestWrap_NilError`
  - `tests/integration`:`TestPublicKeyEndpoint`, `TestResponseHeaders`, `TestRequestMethodValidation`
  - 这些包不依赖 `internal/middleware`、`pkg/permission` 或本次修改文件,判断为先前已存在的回归/ flaky,未纳入本 plan 修复范围,已在 SUMMARY.md 记录。

## User Setup Required

None - no external service configuration required.

## Verification Results

- `go build ./...` exit 0
- `go vet ./...` no warnings
- `go test -v -run "TestLookupResourceAction|TestMapKeys" ./pkg/permission/...` PASS (6 tests)
- `go test -v -run "TestRequireAPIKeyResourcePermission" ./internal/middleware/...` PASS (8 tests)
- `go test -v -run "TestMultiAuthInheritPerms" ./internal/middleware/...` PASS (5 tests)
- Regression anchors:
  - `go test -v -run TestRateLimitHeaderEncoding ./internal/middleware/...` PASS
  - `go test -v -run TestMultiAuthUsageLogTiming ./internal/middleware/...` PASS
  - `go test -v -run TestMultiAuthIntegration ./internal/middleware/...` PASS
- `go test ./internal/middleware/...` PASS
- `go test ./pkg/permission/...` PASS
- `grep -c "PermissionCode" pkg/permission/resource_action_map.go` = 67 (>= 50)
- `grep -E "monitor:|network:|tool:|operations:" pkg/permission/resource_action_map.go` 仅出现在注释(D-02 范围限定)

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new_permission_surface | pkg/permission/resource_action_map.go | 新增 `system:apikey:*` PermissionCode 常量并接入 auth helper,属于本 phase 授权面扩展,已在 map 中显式定义 |
| threat_flag: per-request_db_query | internal/middleware/apikey.go:setUserContextForAPIKey | `InheritPerms=true` 时每次请求查询 `sys_menu/role_menu/user_role`;D-07 决策接受该风险,由 `RateLimitByScope` 兜底(Plan 02 将进一步配置化限流) |

## Next Phase Readiness

- Plan 02 (QUAL-03) 可继续:RateLimitByScope 接收 action 参数、扩展 `getRequiredScope` (list→read)、提取 `SelectScope` 纯函数、`CacheConfigService` 新增 `rate_limit.*` 配置项、`RateLimiter` 配置化改造
- Plan 02 需要修改 `internal/middleware/apikey.go` 中的 `getRequiredScope`/`RateLimitByScope`/`getScopeFromContext` 及 `internal/services/rate_limiter.go`,不影响本次 Plan 01 成果

## Self-Check: PASSED

- [x] `pkg/permission/resource_action_map.go` exists
- [x] `pkg/permission/resource_action_map_test.go` exists
- [x] `internal/middleware/apikey_resource_permission_test.go` exists
- [x] `internal/middleware/apikey_inherit_integration_test.go` exists
- [x] Commit `c55a3c5` (Task 1) exists in history
- [x] Commit `cba12ce` (Task 2) exists in history
- [x] Commit `1eae873` (Task 3) exists in history
- [x] Commit `3a71c11` (SUMMARY) exists in history
- [x] Commit `5020ac5` (STATE/ROADMAP) exists in history
- [x] `go build ./...` exit 0
- [x] `go test ./pkg/permission/... ./internal/middleware/...` PASS

---
*Phase: 61-resource-permission-matrix-and-rate-limit-tuning*
*Plan: 01*
*Completed: 2026-08-13*
