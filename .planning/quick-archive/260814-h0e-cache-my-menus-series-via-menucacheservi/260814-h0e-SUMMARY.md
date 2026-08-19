---
phase: quick-260814-h0e
plan: 01
subsystem: backend/services/system/menu-cache
tags: [backend, cache, performance, menu]
requires:
  - DataCacheService.GetOrSet (L1 内存 + L2 Redis, 已有)
  - menuService.GetUserMenus/GetAllUserMenus/GetUserPermissions (原始实现, 未改)
  - InvalidateCacheByPattern (已有, pattern 多前缀失效)
provides:
  - menuCacheService.GetUserMenus/GetAllUserMenus/GetUserPermissions 的 GetOrSet 缓存覆盖
  - 3 个 user-scoped 缓存键常量 + helper (menu:user:{menus|all|perms}:{userID})
  - InvalidateMenuCache 覆盖 6 个 menu: 前缀 (3 既有 + 3 新 user-scoped)
affects:
  - POST /system/my-menus (首次慢 → L1 命中 ms 级)
  - POST /system/my-menus/all (同上)
  - GET  /system/my-permissions (同上)
  - 菜单写操作 (Create/Update/Delete/BatchDelete/UpdateStatus) — 已有的 InvalidateMenuCache 调用现会同步清 user-scoped 缓存
tech-stack:
  added: []
  patterns:
    - GetOrSet L1+L2 覆盖 (1:1 仿 GetTree 模式)
    - query 闭包显式包装 (interface{}, error) — Go 无返回类型协变
    - user-scoped 缓存键 (prefix:owner:key:{userID}) + 通配失效 (":*")
key-files:
  created: []
  modified:
    - internal/services/system/cache_keys.go
    - internal/services/system/menu_cache_impl.go
decisions:
  - D1: 缓存键置 menu: 命名空间 (menu:user:*), owner=menu 服务, 与 user 模块解耦
  - D2: TTL 复用 CacheConfigMenuTree (30min), 不引入新 CacheConfig 项, 缩小改动面
  - D3: helper 命名加 Menu 前缀 (GetMenuUserPermissionsKey) 以避免与既有 user 模块 GetUserPermissionsKey 冲突
  - D4: InvalidateMenuCache 新 key 用 ":*" 而非 "*" — 精确匹配 prefix:{userID}, 避免误伤
  - D5: T-quick-h0e-02 accept — 缓存 TTL 30min < JWT access_token TTL 2h, 不引入新特权陈旧窗口
metrics:
  duration: 4m
  completed: 2026-08-14T04:25:00Z
  tasks_completed: 2
  files_modified: 2
  files_created: 1 (SUMMARY.md only)
---

# quick-260814-h0e: Cache my-menus 系列 via menuCacheService

为 `menuCacheService` 补齐 `GetUserMenus` / `GetAllUserMenus` / `GetUserPermissions` 三个方法的 `GetOrSet` 缓存覆盖，消除 dev 库（远程 Supabase pooler）下 `/system/my-menus*` 三接口每次直接打 DB 的 24s+ 超时（→ context canceled → 500）。

## Problem / Root Cause

- `menu_router.go` 注入 `NewMenuServiceWithCache(...)` → 期望 `menuCacheService` 走缓存层
- 但 `menuCacheService` 只覆盖了 `GetTree/GetTreeWithCache/GetRouterDataWithCache` 三个方法
- `GetUserMenus` / `GetAllUserMenus` / `GetUserPermissions` fallback 到嵌入式 `*menuService` 的原始实现
- 原始实现每次全表查询（`appendAncestorMenuIDs` 的 `SELECT id,parent_id FROM sys_menu` + `SELECT * ... IN(239)`）→ dev DB 高延迟下三接口并发 → 30s 超时 → context canceled → 500

## Cache Key Design

### 常量 (cache_keys.go)

| 常量 | 值 | 含义 |
|------|----|----|
| `CacheKeyMenuUserMenus` | `menu:user:menus` | 用户菜单树前缀 |
| `CacheKeyMenuUserAllMenus` | `menu:user:all` | 用户全部菜单(含隐藏)前缀 |
| `CacheKeyMenuUserPermissions` | `menu:user:perms` | 用户权限标识前缀 |

### Helper (返回 `{prefix}:{userID}`)

- `GetMenuUserMenusKey(userID)` → `menu:user:menus:{userID}`
- `GetMenuUserAllMenusKey(userID)` → `menu:user:all:{userID}`
- `GetMenuUserPermissionsKey(userID)` → `menu:user:perms:{userID}`

**命名空间决策 (D1, D3):** 缓存的所有者和失效方是 menu 服务（`InvalidateMenuCache` 在 menu 写操作触发），故置于 `menu:` 命名空间而非 `user:`。helper 加 `Menu` 前缀（`GetMenuUserPermissionsKey`）以避免与既有 `GetUserPermissionsKey`（user 模块语义，未被缓存层使用，返回 `user:permissions:{id}`）冲突并明确所有权。

**按 userID 隔离:** 不同用户的菜单/权限互不串读（T-quick-h0e-01 mitigate）。L1 内存缓存 + L2 Redis 双层；L1 命中即返回，miss 时同步写 L1 + 异步写 L2。

## Invalidation Strategy

`InvalidateMenuCache` 的 `DeleteByPattern` pattern 列表从 3 条扩展到 6 条：

```go
[]string{
    CacheKeyMenuTree + "*",                // 既有 (向后兼容 menu:tree:all 后缀)
    CacheKeyMenuRouter + "*",              // 既有
    CacheKeyMenuAll + "*",                 // 既有
    CacheKeyMenuUserMenus + ":*",          // 新 (精确匹配 menu:user:menus:{userID})
    CacheKeyMenuUserAllMenus + ":*",
    CacheKeyMenuUserPermissions + ":*",
}
```

**失效触发点 (已存在, 未改):** `menuCacheService.Create/Update/Delete/BatchDelete/UpdateStatus` 均在成功路径末尾调用 `InvalidateMenuCache(ctx)`。本次只需把新 key 加进 pattern 列表，菜单任意写操作自动同步失效 user-scoped 缓存。

**新 key 用 `":*"` 而非 `"*"` (D4):** 既有 3 条用 `"*"`（无冒号）是为了兼容 `menu:tree:all` 这种后缀变体；新 key 都带 `:{userID}` 后缀，用 `":*"`（冒号星号）精确匹配 `prefix:{任意ID}`，避免误伤同名前缀的其他 key（T-quick-h0e-03 mitigate）。`InvalidateCacheByPattern` 失败会经 `logger.Warn` 记录，可追溯。

## TTL Decision (D2)

复用既有 `services.CacheConfigMenuTree`（动态配置，默认 30 分钟），不引入新 `CacheConfig*` 项：

```go
expiration := s.GetExpiration(services.CacheConfigMenuTree, 30*time.Minute)
```

**理由:**
1. **菜单数据稳定** — 不频繁变更，30min TTL 足够
2. **缩小改动面** — 不动 `cache_config_service.go`、不动配置 schema、不动 `GetCacheTTL` switch（PLAN action 显式要求"不要动 GetCacheTTL switch"）
3. **一致性** — 与既有 `GetTree/GetRouterDataWithCache` 用同一 TTL 配置项，运维侧一处调整全菜单缓存

## T-quick-h0e-02 Tradeoff: 角色撤销不触发菜单缓存失效 (accept)

**威胁:** 用户角色被撤销后，30min 内 `GetUserPermissions` 仍返回旧权限列表（特权陈旧窗口）。

**Accept 理由:**
1. **JWT access_token TTL = 7200s (2h)** — 权限变更本就需要等 token 刷新（前端 TokenManager）才在新 token 中体现；服务端 RBAC 中间件用的是 token claims，而非 `GetUserPermissions` 实时结果
2. **缓存 TTL 30min < 2h** — 缓存陈旧窗口被 token TTL 包含，**不引入新的权限陈旧窗口**
3. **首次冷启动已填缓存 → 后续 L1 命中** — 30min 后自然过期回源，无需手动失效
4. **如需更严格** — 后续可让 `role_service` 在用户角色变更时调用 `InvalidateMenuCache`（超出本 quick 范围，记入 deferred 分析）

`GetUserPermissions` 的 doc comment 中已显式记录此 tradeoff 与缓解路径。

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | cache_keys.go 加 3 常量 + 3 helper | `ba36b1b` | `internal/services/system/cache_keys.go` (+26 lines) |
| 2 | menu_cache_impl.go 加 3 GetOrSet 覆盖 + 扩展 InvalidateMenuCache 到 6 pattern | `f481c93` | `internal/services/system/menu_cache_impl.go` (+55 lines) |

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` (全量) | PASS (exit 0) |
| `go vet ./internal/services/system/` | PASS |
| `go test -run 'TestMenu\|TestCache\|TestValidateAPIKey' ./internal/services/system/` | PASS (1.792s) |
| `go test ./internal/utils/operlog/...` (CLAUDE.md 强制回归守护) | PASS |
| `go test ./internal/services/system/` (全包) | **FAIL** — pre-existing flake, see below |

### Pre-existing flake (out of scope, 已记入 deferred-items.md)

`TestValidateAPIKey/密钥已禁用` 在完整包测试运行中偶发失败（SQLite 内存锁竞争，
`database table is locked: sys_user`）。已验证：
- 单独运行 `go test -run TestValidateAPIKey` **PASS**
- 窄过滤 `go test -run 'TestMenu|TestCache|TestValidateAPIKey'` **PASS**
- 在 stash 掉 Task 2 改动后（仅 Task 1 提交）跑全包测试 **同样 FAIL**

故此 failure 与本 quick 任务无因果关系，属 SCOPE BOUNDARY 之外的既有技术债。
详见 `deferred-items.md`。

## Deviations from Plan

None - plan executed exactly as written. Task 1/2 完全按 PLAN.md action 块执行，
无 Rule 1-4 触发，无 auth gate。

## Self-Check: PASSED

- [FOUND] `internal/services/system/cache_keys.go` — modified (3 consts + 3 helpers)
- [FOUND] `internal/services/system/menu_cache_impl.go` — modified (3 overrides + extended invalidate)
- [FOUND] commit `ba36b1b` — Task 1
- [FOUND] commit `f481c93` — Task 2
- [VERIFIED] `menuService.GetUserMenus/GetAllUserMenus/GetUserPermissions` 原始实现未改（diff 仅触及 2 个目标文件）
- [VERIFIED] `MenuService` 接口签名、`menu_handler`、`menu_router`、`NewMenuServiceWithCache` 均未改
- [VERIFIED] 既有 `GetUserPermissionsKey` (user 模块) 未改

## Runtime Verification (deferred to dev env)

PLAN verification step 4 是可选的运行时验证（启动 backend + 真实请求 my-menus*），
本 quick 任务未执行（无运行中的 dev backend，且代码层验证已覆盖契约正确性）。
如运维侧需 E2E 确认，按 PLAN.md `<verification>` step 4 流程执行：
首次慢一次填缓存 → 第二次 ms 级返回 → 改菜单 → 重新回源 → 换用户独立回源。
