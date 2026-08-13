---
phase: 60-security-hardening-and-enable-decision
plan: 01
subsystem: auth
tags: [apikey, multi-auth, rate-limit, middleware-chain, router-mount, sm2-sm3, strconv-itoa]

# Dependency graph
requires:
  - phase: 57-auth-chain-core-fix-regression-test
    provides: MultiAuth / RateLimitByScope 中间件自洽 + 7 context 键约束 (D-03 / D-04)
  - phase: 59-observability-usage-log-fix
    provides: UsageLogger detached-context + 真实 StatusCode/Duration/Success 写入 (D-01 / D-02)
provides:
  - MultiAuth + RateLimitByScope 真实挂载到 /system/apikeys/* 管理面 8 路由 (AUTH-03 启用)
  - 限流响应头 RFC 6585 合规数字字符串化 (QUAL-01 P2-a 修复)
  - AUTH-03 5 维度决策记录 notes (挂载范围 / 认证优先级 / IP 白名单 / JWT 回退 / InheritPerms scope-boundary)
  - InheritPerms Phase 60 scope-boundary 显式划定 (resource 维度留 Phase 61)
affects:
  - 61-resource-permission-matrix-and-rate-limit-tuning (Phase 60 AUTH-03=启用 触发 Phase 61 无条件执行)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "router 装配: 路由组现场构造 service (NewAPIKeyService / NewUsageLogger / NewRateLimiter),与 location-alias 范式一致,不扩 core.Core"
    - "internal/middleware 与 pkg/middleware 同 package 名的双 import 模式 (internalmw 别名)"
    - "限流响应头数字序列化: strconv.Itoa 取代 string(rune(int)) (RFC 6585 合规)"
    - "集成测试防 nil 限流器: services.NewRateLimiter() 必须传非 nil (Pitfall 4)"

key-files:
  created:
    - .planning/notes/260813-auth03-enable-decision.md
  modified:
    - internal/api/router.go
    - internal/middleware/apikey.go
    - internal/middleware/apikey_test.go
    - internal/middleware/apikey_integration_test.go

key-decisions:
  - "挂载顺序 RequirePermissions -> MultiAuth -> RateLimitByScope (D-01 锁定,权限校验先于认证)"
  - "X-API-Key 优先 + JWT 回退 (D-03 沿用 apikey.go:26-31 既有逻辑,router 层零 fallback 分支)"
  - "IP 白名单严格拒绝 (D-04 沿用 isIPAllowed,空白名单放行所有 IP)"
  - "Phase 60 不修改 InheritPerms 行为 (context 键保留 + resource 维度留 Phase 61 / AUTH-04)"
  - "限流头 strconv.Itoa 修复 (D-11),getScopeFromContext 多 scope 选择严格不修 (D-13 留 QUAL-03 / Phase 61)"

patterns-established:
  - "5 维度决策记录格式: 挂载范围 / 认证优先级 / IP 白名单 / JWT 回退 / InheritPerms scope-boundary"
  - "限流响应头回归锚: strconv.Itoa 数字字面量 + strconv.Atoi 反解析 + 防御性 != rune 字面量"

requirements-completed: [AUTH-03, QUAL-01]

# Metrics
duration: 15 min
completed: 2026-08-13
---

# Phase 60 Plan 01: MultiAuth 生产挂载 + 限流响应头 RFC 6585 修复 Summary

**X-API-Key 认证链在 `/system/apikeys/*` 管理面真实挂载启用 + RateLimitByScope 限流响应头 `strconv.Itoa` 编码修复,使 API Key 认证链在生产路径具备可观测、可限流的启用条件 (AUTH-03 = 启用,触发 Phase 61 无条件执行)**

## Performance

- **Duration:** 15 min
- **Started:** 2026-08-13T05:46:56Z
- **Completed:** 2026-08-13T06:01:31Z
- **Tasks:** 2 / 2 complete
- **Files modified:** 4

## Accomplishments

- **`/system/apikeys/*` 中间件链真实挂载**:`internal/api/router.go:241-262` apikeys 路由组按 D-01 锁定顺序装配 `RequirePermissions` → `MultiAuth` → `RateLimitByScope`,8 路由生效 (Create/List/GetByID/Update/Delete/ToggleStatus/ListUsageLogs/GetUsageSummary)
- **P2-a 限流响应头编码修复**:`internal/middleware/apikey.go:267-268` `string(rune(result.Limit))` → `strconv.Itoa(result.Limit)` (Limit=100 → "100" 而非 "d"),前端 / 第三方工具可按 RFC 6585 标准 `parseInt` 消费
- **5 维度 AUTH-03 决策记录**:`.planning/notes/260813-auth03-enable-decision.md` 含挂载范围 / 认证优先级 / IP 白名单 / JWT 回退+安全评估 / **作用域继承 (InheritPerms) 行为 — Phase 60 scope-boundary** 5 段,引用 D-01..D-04 决策 ID 与源码行号
- **InheritPerms scope-boundary 显式划定**:Phase 60 保留 7 context 键(含 `inherit_perms`)与 `default` 限流档短路,但**resource 维度细粒度校验明确留 Phase 61 / AUTH-04**——`RequireAPIKeyResourcePermission(resource, action)` 的 resource 参数本 phase 不接入
- **回归零破坏**:`TestMultiAuthIntegration` 三路径 + `TestMultiAuthUsageLogTiming/Failure` + `TestRateLimitResult` + `TestIsIPAllowed` 9 子测试全部 PASS

## Task Commits

Each task committed atomically:

1. **Task 1 (AUTH-03)**: `6324e45` (feat) — 挂载 MultiAuth + RateLimitByScope + 5 维度 AUTH-03 决策记录 notes
2. **Task 2 (QUAL-01)**: `6891936` (fix) — `strconv.Itoa` 修复 + TestRateLimitHeaderEncoding 单测 + TestRateLimitHeadersInResponse 集成测

## Files Created/Modified

- `internal/api/router.go` — apikeys 路由组新增 `internalmw.MultiAuth` + `internalmw.RateLimitByScope` 两段 middleware 装配 (D-01)
- `internal/middleware/apikey.go` — 引入 `strconv` import;限流响应头 `string(rune(int))` → `strconv.Itoa` (D-11);X-RateLimit-Reset 保持 time.RFC3339;getScopeFromContext 严格不动 (D-13)
- `internal/middleware/apikey_test.go` — 新增 `TestRateLimitHeaderEncoding` (2 子测试: 数字字符串化 + strconv.Atoi 反解析 + 防御性 ≠ "d")
- `internal/middleware/apikey_integration_test.go` — 新增 `TestRateLimitHeadersInResponse` (跨真实 gin.Engine + MultiAuth → RateLimitByScope 完整链路 + 防御性 ≠ "d"/"c")
- `.planning/notes/260813-auth03-enable-decision.md` — 5 维度决策记录 (挂载范围 / 认证优先级 / IP 白名单 / JWT 回退 / InheritPerms scope-boundary)

## Decisions Made

| 决策 | 理由 |
|------|------|
| 挂载粒度限定单路由组 (apikeys),非全局 authorized.Use | 任何回滚 = 删 2 行 `apikeys.Use(...)`,不触及其他 20+ 路由组 |
| 中间件代码零改动 (apikey.go 业务逻辑未动) | Phase 57 D-03 已验证 4 中间件自洽;Phase 60 只做"装配"不做"改造" |
| 路由组现场构造 service (NewAPIKeyService/NewUsageLogger/NewRateLimiter) | 与 `location-alias` / `department_router` 既有范式一致,不扩 `core.Core` 字段 |
| 内部 middleware import 取别名 `internalmw` | `pkg/middleware` 已用别名 `middleware`,同名 package 不可重复 import |
| 限流响应头 `strconv.Itoa` (而非 `fmt.Sprintf("%d", x)`) | 项目既有 `cache_config_service.go:138-141` 已用 `strconv.Itoa` 标准做法 |
| `X-RateLimit-Reset` 不动 (time.RFC3339) | D-11 明确不动;RFC3339 已是合规时间字符串 |
| `getScopeFromContext` (QUAL-03 范畴) 严格不修 | D-13 锁定留 Phase 61,避免与 Phase 61 资源权限设计决策冲突 |
| InheritPerms Phase 60 保留 context 键 + `default` 短路 + 不接入 resource | Phase 60 scope-boundary,显式划归 Phase 61 / AUTH-04 |

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- **Phase 61 / AUTH-04 (资源级权限矩阵) 已无条件执行**:AUTH-03=启用,不再 conditional。需要:
  1. `RequireAPIKeyResourcePermission(resource, action)` 的 `resource` 参数真实接入 + resource×action → permission 映射矩阵
  2. InheritPerms=true 时真正加载关联 User 角色权限集并参与校验 (当前仅置 bool context 键)
  3. `RequireAPIKeyResourcePermission` 在生产路由的挂载点决策
  4. `getScopeFromContext` 多 scope 选择策略 (QUAL-03,与 InheritPerms `default` 短路互锁)
- **Phase 60 Plan 02 (SEC-01 + SEC-02) 仍待执行**:SM3 单向哈希迁移 (`sys_api_keys.key` → `KeyHash`+`Salt`+`KeyPrefix`) + 冗余索引手动 SQL。本 Plan 01 完成后即可承接。
- **已知遗留**:API Key 认证路径当前仍位于 `authorized` JWT 组下,纯 API Key 无 JWT 的调用路径需把 apikeys 组从 `authorized` 提到 `system` 层级——超出 D-02「最小爆炸半径」约束,属 Phase 61 范畴。

---

*Phase: 60-security-hardening-and-enable-decision*
*Completed: 2026-08-13*
