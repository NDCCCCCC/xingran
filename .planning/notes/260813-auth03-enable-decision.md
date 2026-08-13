# Phase 60 / AUTH-03: MultiAuth 生产挂载启用决策记录

**记录日期**: 2026-08-13
**关联 Requirement**: AUTH-03（MultiAuth 启用决策）
**关联 ROADMAP**: Phase 60「安全加固与启用决策」SC#1
**决策上下文**: `.planning/phases/60-security-hardening-and-enable-decision/60-CONTEXT.md` §decisions D-01 / D-02 / D-03 / D-04
**落地 commit 范围**: Phase 60 Plan 01 Task 1（`internal/api/router.go` apikeys 路由组）

> **结论先行**：**启用**。X-API-Key 认证链在 `/system/apikeys/*` 管理面路由组**全量挂载并立即生效**（D-01）。
> 该决策的直接后果：Phase 61（AUTH-04 资源级权限矩阵 + QUAL-03 限流生产调优）由 conditional 转为**无条件执行**。

---

## 挂载范围（维度 1 — D-01 全量挂载 + D-02 范围限定）

**决策**：仅 `/system/apikeys/*` 管理面路由组挂载 `MultiAuth` + `RateLimitByScope`；其余全部模块（运维 / 资产 / 网络 / 工单 / VDI / 值班 / 知识库 …）**继续纯 JWT 认证，零冲击**。

**落地位置**：`internal/api/router.go:241-262`（`apikeys := authorized.Group("/apikeys")` 路由组）

```go
// internal/api/router.go:241-262
apikeys := authorized.Group("/apikeys")
apikeys.Use(middleware.RequirePermissions([]string{ /* system:apikey:list|add|edit|delete */ }, core))
apikeys.Use(internalmw.MultiAuth(                                   // D-01 新增
    systemServices.NewAPIKeyService(core.GetDB()),
    services.NewUsageLogger(core.GetDB()),
))
apikeys.Use(internalmw.RateLimitByScope(services.NewRateLimiter())) // D-01 新增
{
    systemV1.SetupAPIKeyRouter(apikeys, core)
}
```

**生效端点清单（8 条，`internal/api/v1/system/apikey_router.go:18-25`）**：

| # | Method | Path | Handler | 说明 |
|---|--------|------|---------|------|
| 1 | POST | `/system/apikeys` | `Create` | 创建密钥（一次性返回明文） |
| 2 | POST | `/system/apikeys/list` | `List` | 分页列表 |
| 3 | POST | `/system/apikeys/:id` | `GetByID` | 单条详情 |
| 4 | POST | `/system/apikeys/:id/update` | `Update` | 更新 |
| 5 | POST | `/system/apikeys/:id/delete` | `Delete` | 删除 |
| 6 | POST | `/system/apikeys/:id/toggle` | `ToggleStatus` | 启停状态切换 |
| 7 | POST | `/system/apikeys/:id/logs` | `ListUsageLogs` | 使用日志分页 |
| 8 | GET | `/system/apikeys/:id/summary` | `GetUsageSummary` | 使用统计摘要 |

**为什么是"最小爆炸半径"**：

| 维度 | 决策 | 理由 |
|------|------|------|
| 挂载粒度 | 单一路由组（非全局 `authorized.Use`） | 任何回退只需删 2 行 `apikeys.Use(...)`，不触及其他 20+ 路由组 |
| 中间件代码 | **零改动**（`internal/middleware/apikey.go` 本 task 不动） | Phase 57 D-03 已验证 4 中间件自洽；Phase 60 只做"装配"不做"改造" |
| 依赖注入 | 路由组现场构造（`NewAPIKeyService` / `NewUsageLogger` / `NewRateLimiter`） | 与 `location-alias` / `department_router` 现场构造范式一致，不扩 `core.Core` 字段 |
| 可观测性前提 | 依赖 Phase 59 已修复的 UsageLogger（真实 StatusCode / Duration / Success） | 启用即产生**可信**使用日志，而非 successRate≈0% 的失真数据 |

---

## 认证优先级（维度 2 — D-03 X-API-Key 优先 + JWT 回退）

**决策**：`X-API-Key` 请求头存在 → `MultiAuth` 完成认证并写 context + 记使用日志；请求头缺失 → `MultiAuth` 直接 `c.Next()` 跳过，由上游 JWT 中间件（`authorized.Use(middleware.JWTAuthWithBlacklist(...))`，`router.go:112`）接管。**router 层不加任何 fallback 分支**——优先级完全由 `internal/middleware/apikey.go:26-31` 既有逻辑承担。

```go
// internal/middleware/apikey.go:26-31（既有逻辑，D-03 沿用未改）
apiKeyStr := extractAPIKey(c)
if apiKeyStr == "" {
    // 没有 API Key，跳过（允许 JWT 认证）
    c.Next()
    return
}
```

**优先级判定表**：

| 场景 | `X-API-Key` 头 | 认证承担者 | `auth_type` context 值 | 使用日志 | 结果 |
|------|----------------|-----------|------------------------|---------|------|
| 浏览器管理后台请求 | 缺失 | JWT（`JWTAuthWithBlacklist`，router.go:112） | `jwt`（由 JWT 中间件写） | 不记 | 200 / JWT 既有语义 |
| 第三方脚本携带密钥 | 存在且合法 | MultiAuth（apikey.go:23-87） | `api_key`（apikey.go:167） | **记**（apikey.go:76-85） | 200 |
| 密钥格式非法（≠ `rec_`+64hex） | 存在但格式错 | MultiAuth 早退 | — | 不记（`c.Next()` 前 abort） | 401「无效的密钥格式」 |
| 密钥不存在 / 已禁用 / 过期 | 存在但校验失败 | MultiAuth 早退 | — | 不记 | 401「密钥验证失败: …」 |
| 密钥合法但 IP 不在白名单 | 存在 | MultiAuth 早退 | — | 不记 | 403（见下节） |

**关键不变量**：

1. `MultiAuth` 挂在 JWT 中间件**之后**（JWT 在 `authorized` 组，MultiAuth 在其子组 `apikeys`）。因此**携带 X-API-Key 但不带 JWT 的请求，仍会先被 JWT 中间件拦截**——这是 Phase 60 的已知边界，纯 API Key 无 JWT 的调用路径属 Phase 61 范畴（需要把 apikeys 组从 `authorized` 提到 `system` 层级，改动面超出 D-02「最小爆炸半径」约束）。
2. `MultiAuth` 的日志记录点在 `c.Next()` **之后**（Phase 59 OBSERV-01 修复），因此下游 `RateLimitByScope` 的 429 也会被如实记录为 `Success=false`。
3. 7 个 gin context 键（`user_id` / `username` / `nickname` / `api_key_id` / `scopes` / `auth_type` / `inherit_perms`）保持 Phase 57 D-04 约束，本次挂载**不增不减**。

---

## IP 白名单（维度 3 — D-04 严格拒绝）

**决策**：保留 `internal/middleware/apikey.go` 既有 `isIPAllowed` 行为，**不改一行中间件代码**。语义为「非空白名单 = 严格拒绝，空白名单 = 放行所有 IP」。

**校验入口**（`internal/middleware/apikey.go:48-56`）：

```go
// 验证 IP 白名单（GORM已自动反序列化为[]string）
if len(apiKey.IPWhitelist) > 0 {
    clientIP := c.ClientIP()
    if !isIPAllowed(clientIP, apiKey.IPWhitelist) {
        response.Error(c, response.ErrForbidden, "客户端IP不在白名单中")
        c.Abort()
        return
    }
}
```

**匹配实现**（`internal/middleware/apikey.go:120-152` `isIPAllowed`）：

| 白名单条目形态 | 匹配方式 | 示例 | 命中判定 |
|----------------|---------|------|---------|
| 空数组 `[]` | 提前 `return true`（apikey.go:122-124） | `[]` | **全部 IP 放行**（配置默认） |
| 单个 IP | 字符串精确相等（apikey.go:144-147） | `192.168.1.1` | 客户端 IP 完全一致才放行 |
| CIDR | `net.ParseCIDR` + `ipNet.Contains`（apikey.go:134-142） | `10.0.0.0/24` | 落入网段即放行 |
| 非法条目 | `ParseCIDR` 出错 → `continue` 跳过（apikey.go:137-139） | `invalid-ip` | 不放行（fail-secure） |
| 客户端 IP 不可解析 | `net.ParseIP` 返回 nil → `return false`（apikey.go:127-130） | — | 拒绝（fail-secure） |

**拒绝响应**：`403` + `"客户端IP不在白名单中"`（`response.ErrForbidden`）。回归锚点见 `internal/middleware/apikey_test.go:43-101` `TestIsIPAllowed`（9 个子测试，含 CIDR / IPv6 / 非法 IP / 空白名单 5 类边界）。

**运维注意**：生效 IP 取 `c.ClientIP()`。若后端置于 Nginx / LB 之后，必须确保 `X-Forwarded-For` / `X-Real-IP` 被正确透传且 gin `TrustedProxies` 配置正确，否则白名单会命中网关 IP 而非真实客户端 IP。本 phase 不改 TrustedProxies 配置。

---

## JWT 回退 + 安全评估（维度 4 — D-01 衍生结论）

**中间件链最终顺序**（`internal/api/router.go:241-262`）：

```
JWTAuthWithBlacklist (router.go:112, authorized 组)
    → OperLogMiddleware (router.go:114, authorized 组)
        → RequirePermissions(["system:apikey:list|add|edit|delete"])   ← 既有，未改
            → MultiAuth(APIKeyService, UsageLogger)                    ← D-01 新增
                → RateLimitByScope(RateLimiter)                        ← D-01 新增
                    → SetupAPIKeyRouter 8 handlers
```

**安全评估表**：

| 关注点 | 评估 | 处置 |
|--------|------|------|
| `RequirePermissions` 在 `MultiAuth` **之前** | 权限校验先于 API Key 认证。无 `system:apikey:*` 权限的调用方在 MultiAuth 之前即被 403，不会触达密钥校验与限流逻辑 | **接受**（threat T-60-A7）：更严格的失败快路径，且与 D-01 锁定顺序一致 |
| JWT 通过 + 无 X-API-Key | MultiAuth 跳过，handler 内 `auth_type` 由 JWT 中间件决定，行为与 Phase 60 之前**完全一致** | **零回归**——这是最主要的现存调用路径（前端管理页） |
| JWT 通过 + 携带 X-API-Key | 两条认证都走，MultiAuth 用 API Key 的 context 覆盖 JWT 写入的 `user_id` / `username` / `auth_type` | **接受**：管理面语义上「密钥优先」符合 D-03；handler 侧按 `auth_type` 区分上下文 |
| 限流粒度 | `RateLimitByScope` 仅对 `auth_type == "api_key"` 生效（apikey.go:250-255），JWT 请求直接 `c.Next()` | **零冲击** JWT 用户 |
| 限流 scope 选择 | `getScopeFromContext` 只取 `scopes[0]`（apikey.go:302-303） | **不修**（D-13 严格划归 QUAL-03 / Phase 61），Phase 60 仅修限流响应头编码（QUAL-01） |
| 使用日志可信度 | 依赖 Phase 59 OBSERV-01/02/03 修复（真实 StatusCode / Duration / Success + detached context） | **前置条件已满足**，故 Phase 60 才允许启用 |
| 密钥明文存储 | `sys_api_keys.key` 当前明文（P2-c） | **同 phase 修复**：SEC-01 SM3 单向哈希（Plan 60-02，D-05..D-09） |

**Production 启用后果**：

1. 启用**立即生效**——无灰度开关、无 feature flag（D-01「不再 conditional」）。回滚 = 删除 `router.go` 两条 `apikeys.Use(...)`。
2. **触发 Phase 61 无条件执行**：AUTH-04（资源级权限矩阵 + `RequireAPIKeyResourcePermission` 的 `resource` 参数真实接入）+ QUAL-03（`getScopeFromContext` 多 scope 选择 + 限流阈值生产调优）。
3. 已知遗留（**显式不在 Phase 60 修**）：`username = apiKey.Name`（apikey.go:163，Name 是密钥名非用户名，Phase 57 D-04 语义保留）→ Phase 61 资源权限领域处理。

---

## 作用域继承 (InheritPerms) 行为 — Phase 60 scope-boundary（维度 5）

**决策：Phase 60 范围内不修改 `InheritPerms` 的任何行为。** 保持既有持久化 + 既有 context 写入，**resource 维度的细粒度校验明确留 Phase 61 / AUTH-04**。

**Phase 60 保留原样的三处既有实现**：

| # | 位置 | 现状 | Phase 60 处置 |
|---|------|------|--------------|
| 1 | `internal/models/api_key.go:19` | `InheritPerms bool \`gorm:"default:false" json:"inheritPerms"\`` | **不动**（GORM 持久化 + camelCase JSON 契约保持，Phase 58 D-03 命名约定） |
| 2 | `internal/middleware/apikey.go:169-173` | `if apiKey.InheritPerms && apiKey.User != nil { c.Set("inherit_perms", true) }` | **不动**（Phase 57 D-04 7 键约束之一，键必须继续写入） |
| 3 | `internal/middleware/apikey.go:286-289` | `getScopeFromContext` 见 `inherit_perms` → 返回 `"default"` scope | **不动**（D-13：多 scope 选择逻辑严格划归 QUAL-03 / Phase 61） |

**Phase 60 启用后 InheritPerms 的实际语义**：

| 能力 | Phase 60 状态 | 说明 |
|------|--------------|------|
| `inherit_perms` context 键写入 | ✅ 生效 | 密钥 `InheritPerms=true` 且关联 User 非空时写入，下游可读 |
| 限流 scope 归并为 `default` | ✅ 生效 | 继承权限的密钥走 `default` 限流档（`PerMinute:120/PerHour:2000/PerDay:20000`，rate_limiter.go:64） |
| `RequireScope(scope)` 层级校验 | ✅ 生效 | 基于 `scopes` 数组（admin > write > read），与 InheritPerms 正交 |
| **resource 维度细粒度校验** | ❌ **不生效** | `RequireAPIKeyResourcePermission(resource, action)`（apikey.go:221-224）当前**忽略 `resource` 参数**，仅按 action 映射 scope；且该中间件在 Phase 60 **未挂载任何生产路由** |
| InheritPerms → 加载关联 User 角色权限并校验 | ❌ **不生效** | 代码注释「User 角色会在需要时从数据库加载」（apikey.go:170-171）—— 该加载路径至今**没有实现** |

**明确留给 Phase 61 / AUTH-04 的工作**：

1. `RequireAPIKeyResourcePermission(resource, action)` 的 `resource` 参数真实接入 → resource×action → permission 映射矩阵。
2. `InheritPerms=true` 时真正加载关联 `User` 的角色权限集并参与校验（当前仅置一个 bool context 键）。
3. 该中间件在生产路由的挂载点决策（Phase 60 完全未挂载）。
4. `getScopeFromContext` 多 scope 选择策略（QUAL-03）——与 InheritPerms 的 `default` 短路互相影响，必须同期设计。

**Phase 60 的 scope-boundary 一句话表述**：**context 键完整保留（Phase 57 D-04 7 键约束不破），resource 维度不强制校验**。任何期待「API Key 带 InheritPerms 就自动获得关联用户全部资源权限」的行为，在 Phase 60 **不存在**——现阶段 `/system/apikeys/*` 的权限闸门仍是 `RequirePermissions(["system:apikey:*"])`（JWT 用户维度），而非 API Key 维度的资源矩阵。

---

**参考文件清单**：

- `internal/api/router.go:241-262` — AUTH-03 挂载点（本 phase 唯一生产代码改动）
- `internal/api/v1/system/apikey_router.go:18-25` — 生效的 8 条路由
- `internal/middleware/apikey.go:23-87` — `MultiAuth`（D-03 优先级 + D-04 IP 白名单调用点）
- `internal/middleware/apikey.go:120-152` — `isIPAllowed`（D-04 匹配实现）
- `internal/middleware/apikey.go:154-174` — `setUserContextForAPIKey`（7 context 键 + `inherit_perms`）
- `internal/middleware/apikey.go:221-224` — `RequireAPIKeyResourcePermission`（resource 参数留 Phase 61）
- `internal/middleware/apikey.go:247-281` — `RateLimitByScope`（QUAL-01 限流头修复同 plan Task 2）
- `internal/middleware/apikey.go:285-304` — `getScopeFromContext`（D-13 严格留 Phase 61）
- `internal/models/api_key.go:19` — `InheritPerms` 字段
- `internal/services/rate_limiter.go:44-53` — `NewRateLimiter()` 零参构造 + 三档 scope 限额
- `internal/services/usage_logger.go:38-41` — `NewUsageLogger(db)` 构造
- `.planning/phases/60-security-hardening-and-enable-decision/60-CONTEXT.md` — D-01..D-04 锁定决策原文
- `.planning/phases/57-auth-chain-core-fix-regression-test/57-CONTEXT.md` — D-03（4 中间件自洽）+ D-04（7 context 键约束）
- `.planning/phases/59-observability-usage-log-fix/59-CONTEXT.md` — D-01（Success 仅 2xx）+ D-02（detached context 属 UsageLogger 契约）
