---
phase: 61-resource-permission-matrix-and-rate-limit-tuning
plan: 02
subsystem: auth
tags:
  - api-key
  - rate-limit
  - cache-config
  - select-scope
  - sys_config
  - gin

requires:
  - phase: 61-resource-permission-matrix-and-rate-limit-tuning
    plan: 01
    provides: MultiAuth 签名扩展 + InheritPerms 集成 + resource_action_map(Wave 1 基线)
  - phase: 60-security-hardening-and-enable-decision
    provides: MultiAuth/RateLimitByScope 生产挂载 + strconv.Itoa 限流响应头(QUAL-01)
  - phase: 59-observability-usage-log-fix
    provides: sqlite in-memory 测试模式(D-03)

provides:
  - internal/middleware/select_scope.go: 纯函数 SelectScope(scopes, inheritPerms, action) → (scope, allowed)
  - RateLimitByScope(rateLimiter, action) action-aware 多 scope 严格选择(fail-closed 403)
  - getRequiredScope 扩展 list→read
  - 12 个 rate_limit.* sys_config 配置键(CacheConfigService 共存,运维可调)
  - RateLimiter 配置化(RateLimitProvider 接口,移除硬编码 limits map)
  - 四层测试:9 SelectScope 纯函数 + 7 RateLimitByScope 中间件 + 9 RateLimiter 配置驱动 + 5 CacheConfigService rate_limit

affects:
  - internal/api/router.go apikeys 路由组 RateLimitByScope 调用形态(action + core.CacheConfigService)
  - 未来 per-API-Key 限流 override(FUTURE-APIKEY-04)可基于 RateLimitProvider 扩展

tech-stack:
  added: []
  patterns:
    - "纯函数 SelectScope 替代 context key 中转,单元测试直接断言返回值"
    - "action-aware 多 scope 选择: 精确匹配 requiredScope → admin 覆盖 → fail-closed 403(无 fallback)"
    - "RateLimitProvider 接口解耦 RateLimiter 与 CacheConfigService(duck typing)"
    - "rate_limit.* 与 cache.* 共存同一 CacheConfigService,独立 map 存储(次数 vs Duration)"
    - "reload 后新阈值仅对新请求生效,在途滑动窗口保留旧阈值(D-19)"

key-files:
  created:
    - internal/middleware/select_scope.go
    - internal/middleware/select_scope_test.go
    - internal/middleware/apikey_rate_limit_test.go
    - internal/services/cache_config_service_test.go
  modified:
    - internal/middleware/apikey.go (RateLimitByScope + getScopeFromContext + getRequiredScope)
    - internal/middleware/apikey_test.go (TestGetRequiredScope 新增 list→read 用例)
    - internal/middleware/apikey_integration_test.go (2 处 RateLimitByScope 签名 + NewRateLimiter(nil))
    - internal/services/rate_limiter.go (D-18 配置化改造)
    - internal/services/rate_limiter_test.go (449 行迁移到新签名 + 2 新用例)
    - internal/services/cache_config_service.go (12 rate_limit.* 键 + RateLimitProvider + GetRateLimit)
    - internal/api/router.go (RateLimitByScope action 参数 + core.CacheConfigService 字段)

key-decisions:
  - "SelectScope 为纯函数,继承 Phase 57 D-04 既有 7 context 键,不新增第 8 个中转键"
  - "scopes 不含 requiredScope 且无 admin → 403 fail-closed,不进入限流检查(D-12)"
  - "InheritPerms=true 短路走 default 限额,细粒度 permission code 不参与 action 匹配(D-13)"
  - "rate_limit.* 值语义为次数(int),独立 rateLimits map 存储,不混入 cache.* Duration map"
  - "NewRateLimiter(nil) 兜底 staticRateLimitProvider,默认值与既有硬编码一致(D-17)"

requirements-completed:
  - QUAL-03

# Metrics
duration: 31min
completed: 2026-08-13
---

# Phase 61 Plan 02: 限流生产调优(action-aware 多 scope + 配置化阈值) Summary

**QUAL-03 限流生产调优: RateLimitByScope 改造为 action-aware 多 scope 严格选择语义(scopes 不含 requiredScope 且无 admin → 403 fail-closed),scope 选择提取为纯函数 SelectScope 直接可单测;12 个 rate_limit.* 配置键走 sys_config 经 CacheConfigService 运维可调;RateLimiter 从硬编码改为 RateLimitProvider 配置驱动,reload 后新阈值仅对新请求生效。**

## Performance

- **Duration:** 31 min
- **Started:** 2026-08-13T09:11:25Z
- **Completed:** 2026-08-13T09:42:36Z
- **Tasks:** 3
- **Files modified:** 11 (4 created + 7 modified)

## Accomplishments

- 新建 `internal/middleware/select_scope.go`:纯函数 `SelectScope(scopes, inheritPerms, action) → (scope, allowed)`,5 路径语义(InheritPerms 短路 / 精确匹配 / admin 覆盖 / fail-closed / 未知 action 默认 read),替代原 `getScopeFromContext` 任意取 `scopes[0]` 的错误语义
- 改造 `RateLimitByScope(rateLimiter, action string)`:注册期 `requiredScope := getRequiredScope(action)` 闭包捕获;`!allowed` → 403「权限作用域不足」+ Warnf 审计日志 + abort,不进入限流检查(D-12 fail-closed)
- `getScopeFromContext` 改为薄壳包装 SelectScope,从既有 `inherit_perms` / `scopes` context 键读取(Phase 57 D-04 七键契约不破坏)
- `getRequiredScope` 扩展 `list → read`(D-11),view/create/edit/delete 映射不变
- `CacheConfigService` 新增 12 个 `rate_limit.{read|write|admin|default}.{per_minute|per_hour|per_day}` 配置键(D-16),默认值与既有硬编码一致(D-17),Min/Max 范围校验沿用既有 cache.* 模式(非法值自动修复回默认并回写 DB)
- 新增 `RateLimitProvider` 接口 + `GetRateLimit(key, defaultValue)` 方法,`*CacheConfigService` duck typing 自动实现(D-18 解耦)
- `RateLimiter` 配置化:移除硬编码 `limits map`,`Check` 内部走 `getLimit(scope)` 运行时从 provider 读;`NewRateLimiter(nil)` 兜底 `staticRateLimitProvider`(默认值一致 + Warnf)
- `router.go` 调用形态: `RateLimitByScope(services.NewRateLimiter(core.CacheConfigService), "list")`(字段访问,非 getter)
- 既有 `rate_limiter_test.go` 449 行全量迁移:7 个测试函数 `NewRateLimiter()` → `NewRateLimiter(newMockRateLimitProvider())`,`limiter.limits` 断言全部移除,新增 `TestRateLimiter_NilProviderFallback` + `TestRateLimiter_ConfigDrivenRead`

## Task Commits

1. **Task 1: RateLimitByScope action-aware + SelectScope 纯函数 + 中间件测试** - `7f13dc9` (feat)
2. **Task 2: CacheConfigService 12 个 rate_limit.* 配置键 + 单元测试** - `7a0b9fe` (feat)
3. **Task 3: RateLimiter 配置化 + 449 行测试迁移 + router.go 接入** - `8b4ad04` (feat)

## Files Created/Modified

- `internal/middleware/select_scope.go` - SelectScope 纯函数(D-12/D-13)
- `internal/middleware/select_scope_test.go` - 9 个纯函数单测(D-20)
- `internal/middleware/apikey_rate_limit_test.go` - 7 个 RateLimitByScope 中间件单测(D-20)
- `internal/middleware/apikey.go` - RateLimitByScope/getScopeFromContext/getRequiredScope 三函数改造
- `internal/middleware/apikey_test.go` - TestGetRequiredScope 新增 list→read 用例
- `internal/middleware/apikey_integration_test.go` - 2 处 RateLimitByScope 签名更新 + NewRateLimiter(nil) 适配
- `internal/services/cache_config_service.go` - 12 rate_limit.* 键 + RateLimitProvider + GetRateLimit + LoadConfigs 双查询
- `internal/services/cache_config_service_test.go` - 5 个 rate_limit.* 配置用例(默认值/reload/range 校验/cache 不回归/接口断言)
- `internal/services/rate_limiter.go` - D-18 配置化(RateLimitProvider + staticRateLimitProvider 兜底)
- `internal/services/rate_limiter_test.go` - 449 行迁移 + mockRateLimitProvider + 2 新用例
- `internal/api/router.go` - RateLimitByScope 新调用形态(action + core.CacheConfigService 字段)

## Decisions Made

- 与计划一致:SelectScope 纯函数 5 路径语义,fail-closed 无 fallback(D-12)
- 与计划一致:12 配置键默认值与既有硬编码一致(D-17),Range 校验 Min/Max 防误调(D-16)
- 与计划一致:复用 CacheConfigService 不新增独立 service(D-15)
- 与计划一致:router.go 用 `core.CacheConfigService` 字段访问(非 getter,Core struct 无该方法)
- 实践微调:RateLimitByScope 拒绝路径追加 `applogger.Warnf` 审计日志(action + requiredScope + path),承接威胁注册 T-61-10「拒绝路径可审计」

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] router.go RateLimitByScope 调用需在 Task 1 同步补 action 参数**
- **Found during:** Task 1 verify(`go build ./...`)
- **Issue:** 计划把 router.go 修改整体放在 Task 3,但 Task 1 改了 `RateLimitByScope` 签名后,唯一生产调用点 `router.go:260` 立即编译失败,Task 1 的 acceptance「go build ./... 退出码 0」无法满足
- **Fix:** Task 1 中先将调用点改为 `RateLimitByScope(services.NewRateLimiter(), "list")`(最小编译修复),Task 3 再替换为 `NewRateLimiter(core.CacheConfigService)` 最终形态
- **Files modified:** `internal/api/router.go`
- **Verification:** 两阶段 `go build ./...` 均 exit 0
- **Committed in:** `7f13dc9`(Task 1)/ `8b4ad04`(Task 3)

**2. [Rule 3 - Blocking] middleware 测试文件 NewRateLimiter() 无参调用编译失败**
- **Found during:** Task 3(`NewRateLimiter` 签名变更后)
- **Issue:** `apikey_integration_test.go`(2 处)与 `apikey_rate_limit_test.go`(1 处)的 `services.NewRateLimiter()` 无参调用在 Task 3 签名变更后编译失败;计划 Task 3 files 列表只列了 rate_limiter_test.go 与 router.go,未列这两个 middleware 测试文件
- **Fix:** 3 处调用改为 `services.NewRateLimiter(nil)`(触发 staticRateLimitProvider 兜底,默认值与既有硬编码一致,测试语义不变)
- **Files modified:** `internal/middleware/apikey_integration_test.go`, `internal/middleware/apikey_rate_limit_test.go`
- **Verification:** `go test ./internal/middleware/` 全部 PASS
- **Committed in:** `8b4ad04`(Task 3)

**3. [Rule 2 - Missing Critical] getConfigRemark 对 rate_limit.* 输出「分钟」语义错误**
- **Found during:** Task 2(12 键 DB 自动 INSERT 路径)
- **Issue:** `getConfigRemark` 硬编码「默认%d分钟,范围%d-%d分钟」,rate_limit.* 值语义为次数,自动写入 sys_config 的 remark 会误导运维
- **Fix:** `getConfigRemark` 对 `rate_limit.` 前缀输出「次」语义
- **Files modified:** `internal/services/cache_config_service.go`
- **Verification:** `TestCacheConfigService_RateLimitDefaults` 断言 12 条记录写入成功
- **Committed in:** `7a0b9fe`(Task 2)

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 missing critical)
**Impact on plan:** 全部为编译/正确性必需修复,无 scope creep。

## Issues Encountered

- **并发会话工作树污染(瞬时):** 执行期间另一会话正在重构 `internal/core/db/database.go`(AutoMigrate → MigrateModelList 提取,见 `.planning/debug/backend-hang-on-automigrate.md`),中途一次 `go build ./...` 命中其半成品状态(MigrateModelList undefined)失败;数秒后对方会话完成编辑,重新构建 exit 0。本 plan 未触碰、未提交对方任何文件(`internal/core/core.go` / `internal/core/db/database.go` / `pkg/logger/logger.go` / 前端文件 / `scripts/dbprobe*` 等均保持未暂存状态留待对方会话处理)。
- **慢速测试固有开销:** `TestRateLimiter_SlidingWindow/Cleanup/Reset` 含 6 处 `time.Sleep(61 * time.Second)`(既有设计,验证滑动窗口),全套 `TestRateLimiter_*` 运行 368s,为既有成本非本次引入。

## User Setup Required

None - 12 个 rate_limit.* 配置键由 `setDefaultsIfNeeded` 启动期自动写入 sys_config 表(IsSystem=1),运维可通过既有 `POST /monitor/cache/reload` 触发 ReloadConfig 刷新阈值。

## Verification Results

- `go build ./...` exit 0
- `go vet ./internal/middleware/ ./internal/services/ ./internal/api/` 无警告
- `go test -v -run TestSelectScope ./internal/middleware/` PASS(9 用例)
- `go test -v -run TestRateLimitByScope ./internal/middleware/` PASS(7 用例)
- `go test -v -run "TestCacheConfigService_RateLimit|TestCacheConfigService_ReloadRateLimit|TestCacheConfigService_RangeValidation|TestCacheConfigService_CacheUnaffected|TestRateLimitProviderInterface" ./internal/services/` PASS(5 用例)
- `go test -v -run "TestRateLimiter_" ./internal/services/` PASS(8 用例,368s 含滑动窗口 sleep)
- `go test -v -run TestNewRateLimiter ./internal/services/` PASS
- 回归锚:
  - `go test -v -run TestRateLimitHeaderEncoding ./internal/middleware/` PASS(Phase 60 QUAL-01 strconv.Itoa 不回归)
  - `go test -v -run TestRateLimitHeadersInResponse ./internal/middleware/` PASS(签名更新后)
  - `go test -v -run TestMultiAuthIntegration ./internal/middleware/` PASS(Phase 57)
  - `go test -v -run TestMultiAuthUsageLogTiming ./internal/middleware/` PASS(Phase 59)
  - `go test ./internal/middleware/ ./pkg/permission/` PASS(Plan 01 成果不回归)
- Grep 验收:
  - `grep -n "func SelectScope" internal/middleware/select_scope.go` → line 16 命中
  - `grep -n "requiredScope := getRequiredScope(action)" internal/middleware/apikey.go` → line 378 命中(闭包捕获)
  - `grep -c "rate_limit\." internal/services/cache_config_service.go` → 27 ≥ 25
  - `grep -n "NewRateLimiter()" internal/services/rate_limiter_test.go` → 0 匹配
  - `grep -n "limiter.limits" internal/services/rate_limiter_test.go` → 0 匹配
  - `grep -n "RateLimitByScope(services.NewRateLimiter" internal/api/router.go` → 含 `core.CacheConfigService` + `, "list"`
  - `grep -n "GetCacheConfigService" internal/api/router.go` → 0 匹配
  - `grep -rn "_internal_scope" internal/middleware/` → 0 匹配
  - `grep -rn "extractScope" internal/middleware/` → 0 匹配

## Known Stubs

None - 无占位/硬编码空值流入 UI;所有配置键均有真实默认值并落库。

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: ops_tunable_rate_limit | internal/services/cache_config_service.go | 限流阈值改为 sys_config 运维可调,IsSystem=1 + Min/Max 范围校验防误调(D-16),reload 需管理员显式触发(T-61-12 已 mitigate) |
| threat_flag: fail_closed_scope_deny | internal/middleware/apikey.go:RateLimitByScope | scopes 不含 requiredScope 且无 admin → 403;行为较 Phase 60「任意 scopes[0]」更严格,存量多 scope API Key 若缺 read 档会在 list 类请求被拒(T-61-08 设计内行为,admin 可重建 key 修复) |

## Next Phase Readiness

- Phase 61 两 plan 全部完成(AUTH-04 + QUAL-03),可进入 phase verification(/gsd:verify-phase 61)
- FUTURE-APIKEY-04 后续可基于 RateLimitProvider 接口扩展 per-API-Key 限流 override,无需再改 RateLimiter 内部

## Self-Check: PASSED

- [x] `internal/middleware/select_scope.go` exists
- [x] `internal/middleware/select_scope_test.go` exists
- [x] `internal/middleware/apikey_rate_limit_test.go` exists
- [x] `internal/services/cache_config_service_test.go` exists
- [x] Commit `7f13dc9` (Task 1) exists in history
- [x] Commit `7a0b9fe` (Task 2) exists in history
- [x] Commit `8b4ad04` (Task 3) exists in history
- [x] `go build ./...` exit 0
- [x] `go test ./internal/middleware/ ./pkg/permission/ ./internal/services/` PASS

---

*Phase: 61-resource-permission-matrix-and-rate-limit-tuning*
*Plan: 02*
*Completed: 2026-08-13*
